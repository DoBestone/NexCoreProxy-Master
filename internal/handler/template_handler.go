package handler

import (
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// GetTemplates 获取入站模板列表
func (h *Handler) GetTemplates(c *gin.Context) {
	var templates []model.InboundTemplate
	if err := model.GetDB().Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取模板列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": templates})
}

// AddTemplate 添加入站模板
func (h *Handler) AddTemplate(c *gin.Context) {
	var template model.InboundTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}
	if len(template.Name) == 0 || len(template.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "模板名称长度需为1-100"})
		return
	}
	if len(template.Protocol) == 0 || len(template.Protocol) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "协议名称无效"})
		return
	}
	if len(template.Remark) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "备注长度不能超过255"})
		return
	}

	if err := model.GetDB().Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "添加模板失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": template})
}

// DeleteTemplate 删除入站模板
func (h *Handler) DeleteTemplate(c *gin.Context) {
	id := parseUint(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "无效的ID"})
		return
	}
	if err := model.GetDB().Delete(&model.InboundTemplate{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "删除模板失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
