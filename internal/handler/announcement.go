package handler

import (
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// GetAnnouncements 获取公告列表（公开）
func (h *Handler) GetAnnouncements(c *gin.Context) {
	var announcements []model.Announcement
	db := model.GetDB()

	query := db.Where("enable = ?", true).Order("pinned DESC, created_at DESC")
	if err := query.Find(&announcements).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "获取公告失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": announcements})
}

// GetAdminAnnouncements 管理员获取所有公告
func (h *Handler) GetAdminAnnouncements(c *gin.Context) {
	var announcements []model.Announcement
	db := model.GetDB()

	if err := db.Order("pinned DESC, created_at DESC").Find(&announcements).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "获取公告失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": announcements})
}

// AddAnnouncement 添加公告
func (h *Handler) AddAnnouncement(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required,max=200"`
		Content string `json:"content" binding:"required,max=10000"`
		Type    string `json:"type"`
		Pinned  bool   `json:"pinned"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	if req.Type == "" {
		req.Type = "info"
	}

	announcement := model.Announcement{
		Title:   req.Title,
		Content: req.Content,
		Type:    req.Type,
		Pinned:  req.Pinned,
		Enable:  true,
	}

	if err := model.GetDB().Create(&announcement).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "添加失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": announcement})
}

// UpdateAnnouncement 更新公告
func (h *Handler) UpdateAnnouncement(c *gin.Context) {
	id := parseUint(c.Param("id"))

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Type    string `json:"type"`
		Pinned  bool   `json:"pinned"`
		Enable  bool   `json:"enable"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	db := model.GetDB()
	updates := map[string]interface{}{}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	updates["pinned"] = req.Pinned
	updates["enable"] = req.Enable

	if err := db.Model(&model.Announcement{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "更新成功"})
}

// DeleteAnnouncement 删除公告
func (h *Handler) DeleteAnnouncement(c *gin.Context) {
	id := parseUint(c.Param("id"))

	if err := model.GetDB().Delete(&model.Announcement{}, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "删除成功"})
}
