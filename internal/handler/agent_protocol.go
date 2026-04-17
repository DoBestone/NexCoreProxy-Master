package handler

import (
	"net/http"
	"strings"
	"time"

	"nexcoreproxy-master/internal/model"
	"nexcoreproxy-master/internal/service"

	"github.com/gin-gonic/gin"
)

// agentAuthMiddleware ncp-agent 鉴权
//
// agent 在每次请求里带 Authorization: Bearer <node.AgentKey>，
// 服务端反查 Node 并把 *model.Node 注入 context。
func (h *Handler) agentAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "missing token"})
			c.Abort()
			return
		}
		node, err := service.LookupNodeByToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": err.Error()})
			c.Abort()
			return
		}
		c.Set("agentNode", node)
		c.Next()
	}
}

// GetAgentConfig GET /api/v1/server/config?etag=xxx
//
// agent 周期性调用。Master 比对 etag，未变化返回 304；变化则返回完整 AgentConfig。
func (h *Handler) GetAgentConfig(c *gin.Context) {
	node := c.MustGet("agentNode").(*model.Node)

	currentEtag := model.GetEtag(node.ID)
	clientEtag := c.Query("etag")
	if currentEtag != "" && clientEtag == currentEtag {
		c.Status(http.StatusNotModified)
		return
	}

	cfg, err := h.agentCfg.Build(node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": err.Error()})
		return
	}

	// 顺手记录最后拉配置时间作为心跳
	now := time.Now()
	model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).
		Updates(map[string]any{"last_sync_at": now, "status": "online"})

	c.JSON(http.StatusOK, cfg)
}

// PushPayload 是 POST /api/v1/server/push 的请求体（Step 3 实现细节）
type PushPayload struct {
	EtagApplied string                  `json:"etagApplied"`
	Stats       map[string]TrafficDelta `json:"stats"`
	Online      map[string][]string     `json:"online"` // email -> []ip
	System      *SystemSnapshot         `json:"system"`
}

type TrafficDelta struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

type SystemSnapshot struct {
	CPU          float64 `json:"cpu"`
	Mem          float64 `json:"mem"`
	Load         float64 `json:"load"`
	XrayUptime   uint64  `json:"xrayUptime"`
	XrayVersion  string  `json:"xrayVersion"`
	AgentVersion string  `json:"agentVersion"`
}

// PostAgentPush POST /api/v1/server/push
//
// 流量入账 + 在线 IP 刷新 + 计算并返回 kicks 名单。kicks 里的 email 表示
// 该用户在 xray runtime 里已无授权（超额/过期/禁用），agent 应立即从所有
// inbound 删除该 client 并 reload xray。
func (h *Handler) PostAgentPush(c *gin.Context) {
	node := c.MustGet("agentNode").(*model.Node)
	var payload PushPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": err.Error()})
		return
	}

	updates := map[string]any{
		"last_sync_at": time.Now(),
		"status":       "online",
	}
	if payload.System != nil {
		updates["cpu"] = payload.System.CPU
		updates["memory"] = payload.System.Mem
		updates["uptime"] = payload.System.XrayUptime
		updates["xray_version"] = payload.System.XrayVersion
		updates["agent_version"] = payload.System.AgentVersion
	}
	model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(updates)

	// 流量入账（service 层只认 service 包内类型，做一次桥接）
	if len(payload.Stats) > 0 {
		dto := make(map[string]service.TrafficDeltaDTO, len(payload.Stats))
		for k, v := range payload.Stats {
			dto[k] = service.TrafficDeltaDTO{Up: v.Up, Down: v.Down}
		}
		if err := h.agentPush.IngestStats(node.ID, dto); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": err.Error()})
			return
		}
	}
	if len(payload.Online) > 0 {
		_ = h.agentPush.IngestOnline(node.ID, payload.Online)
	}

	kicks, err := h.agentPush.CalcKicks(node.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": err.Error()})
		return
	}
	if kicks == nil {
		kicks = []string{}
	}
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"kicks":       kicks,
		"currentEtag": model.GetEtag(node.ID), // agent 可比对决定要不要立即重拉 config
	})
}
