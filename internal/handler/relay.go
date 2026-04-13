package handler

import (
	"log"
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// GetRelayRules 获取所有中转规则
func (h *Handler) GetRelayRules(c *gin.Context) {
	rules, err := h.node.GetRelayRules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取中转规则失败"})
		return
	}

	result := make([]gin.H, 0, len(rules))
	for _, r := range rules {
		result = append(result, gin.H{
			"id":               r.ID,
			"relayNodeId":      r.RelayNodeID,
			"backendNodeId":    r.BackendNodeID,
			"relayNodeName":    r.RelayNode.Name,
			"backendNodeName":  r.BackendNode.Name,
			"relayInboundPort": r.RelayInboundPort,
			"relayInboundTag":  r.RelayInboundTag,
			"relayOutboundTag": r.RelayOutboundTag,
			"protocol":         r.Protocol,
			"enable":           r.Enable,
			"remark":           r.Remark,
			"syncStatus":       r.SyncStatus,
			"syncError":        r.SyncError,
			"createdAt":        r.CreatedAt,
			"updatedAt":        r.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": result})
}

// CreateRelayRule 创建中转规则
func (h *Handler) CreateRelayRule(c *gin.Context) {
	var req struct {
		RelayNodeID      uint   `json:"relayNodeId" binding:"required"`
		BackendNodeID    uint   `json:"backendNodeId" binding:"required"`
		Protocol         string `json:"protocol" binding:"required"`
		RelayInboundPort int    `json:"relayInboundPort"`
		Remark           string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	// 校验备注长度
	if len(req.Remark) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "备注长度不能超过255"})
		return
	}

	// 校验中转端口
	if req.RelayInboundPort != 0 && (req.RelayInboundPort < 1 || req.RelayInboundPort > 65535) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "中转端口范围无效 (1-65535)"})
		return
	}

	// 校验协议
	validProtocols := map[string]bool{"vmess": true, "vless": true, "trojan": true, "shadowsocks": true}
	if !validProtocols[req.Protocol] {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "不支持的协议"})
		return
	}

	// 校验中转节点类型
	var relayNode model.Node
	if err := model.GetDB().First(&relayNode, req.RelayNodeID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "中转节点不存在"})
		return
	}
	if relayNode.Type != "relay" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "所选节点不是中转类型"})
		return
	}

	// 校验落地节点类型
	var backendNode model.Node
	if err := model.GetDB().First(&backendNode, req.BackendNodeID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "落地节点不存在"})
		return
	}
	if backendNode.Type != "backend" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "所选节点不是落地类型"})
		return
	}

	rule := model.RelayRule{
		RelayNodeID:      req.RelayNodeID,
		BackendNodeID:    req.BackendNodeID,
		Protocol:         req.Protocol,
		RelayInboundPort: req.RelayInboundPort,
		Remark:           req.Remark,
		Enable:           true,
		SyncStatus:       "pending",
	}

	if err := h.node.CreateRelayRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "创建中转规则失败"})
		return
	}

	// 自动同步
	go func() {
		_ = h.node.SyncRelayRule(rule.ID)
	}()

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{"id": rule.ID}})
}

// DeleteRelayRule 删除中转规则
func (h *Handler) DeleteRelayRule(c *gin.Context) {
	id := parseUint(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "无效的ID"})
		return
	}

	if err := h.node.DeleteRelayRule(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "删除中转规则失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SyncRelayRule 手动同步中转规则
func (h *Handler) SyncRelayRule(c *gin.Context) {
	id := parseUint(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "无效的ID"})
		return
	}

	if err := h.node.SyncRelayRule(id); err != nil {
		log.Printf("同步中转规则失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "同步失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "同步成功"})
}
