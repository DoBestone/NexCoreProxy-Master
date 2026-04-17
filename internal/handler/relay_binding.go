package handler

import (
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// ListRelayBindings GET /api/relay-bindings
func (h *Handler) ListRelayBindings(c *gin.Context) {
	rows, err := h.relayBinding.List()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": rows})
}

// CreateRelayBinding POST /api/relay-bindings
func (h *Handler) CreateRelayBinding(c *gin.Context) {
	var b model.RelayBinding
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	if err := h.relayBinding.Create(&b); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": b})
}

// UpdateRelayBinding PUT /api/relay-bindings/:id
func (h *Handler) UpdateRelayBinding(c *gin.Context) {
	id := parseUint(c.Param("id"))
	b, err := h.relayBinding.Get(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "binding 不存在"})
		return
	}
	if err := c.ShouldBindJSON(b); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	b.ID = id
	if err := h.relayBinding.Update(b); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": b})
}

// DeleteRelayBinding DELETE /api/relay-bindings/:id
func (h *Handler) DeleteRelayBinding(c *gin.Context) {
	if err := h.relayBinding.Delete(parseUint(c.Param("id"))); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ResyncRelayBinding POST /api/relay-bindings/:id/resync
func (h *Handler) ResyncRelayBinding(c *gin.Context) {
	if err := h.relayBinding.Resync(parseUint(c.Param("id"))); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
