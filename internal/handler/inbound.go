package handler

import (
	"net/http"

	"nexcoreproxy-master/internal/model"
	"nexcoreproxy-master/internal/service"

	"github.com/gin-gonic/gin"
)

// ListInbounds GET /api/inbounds?nodeId=X
func (h *Handler) ListInbounds(c *gin.Context) {
	nodeID := parseUint(c.Query("nodeId"))
	rows, err := h.inbound.List(nodeID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": rows})
}

// CreateInbound POST /api/inbounds
func (h *Handler) CreateInbound(c *gin.Context) {
	var in model.Inbound
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	if err := h.inbound.Create(&in); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": in})
}

// UpdateInbound PUT /api/inbounds/:id
func (h *Handler) UpdateInbound(c *gin.Context) {
	id := parseUint(c.Param("id"))
	in, err := h.inbound.Get(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "inbound 不存在"})
		return
	}
	if err := c.ShouldBindJSON(in); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	in.ID = id
	if err := h.inbound.Update(in); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": in})
}

// DeleteInbound DELETE /api/inbounds/:id
func (h *Handler) DeleteInbound(c *gin.Context) {
	if err := h.inbound.Delete(parseUint(c.Param("id"))); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ToggleInbound POST /api/inbounds/:id/toggle
func (h *Handler) ToggleInbound(c *gin.Context) {
	var req struct {
		Enable bool `json:"enable"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.inbound.Toggle(parseUint(c.Param("id")), req.Enable); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ProvisionNode POST /api/nodes/:id/provision  body: {set: "minimal"|"standard"|"full"}
//
// 给某 backend 节点一键写入预设入站集（reality 密钥/SS PSK/WS path 等自动生成）。
// 已存在的同名 inbound 跳过，幂等。
func (h *Handler) ProvisionNode(c *gin.Context) {
	var req struct {
		Set string `json:"set"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Set == "" {
		req.Set = "standard"
	}
	created, err := h.provisioner.Provision(parseUint(c.Param("id")), service.InboundSet(req.Set))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": created, "count": len(created)})
}
