package handler

import (
	"log"
	"net/http"
	"time"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// GetMyTickets 获取我的工单
func (h *Handler) GetMyTickets(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	var tickets []model.Ticket
	if err := model.GetDB().Where("user_id = ?", userID).Order("created_at desc").Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取工单失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": tickets})
}

// GetAllTickets 获取所有工单（管理员）
func (h *Handler) GetAllTickets(c *gin.Context) {
	var tickets []model.Ticket
	if err := model.GetDB().Preload("User").Order("created_at desc").Limit(500).Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取工单失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": tickets})
}

// CreateTicket 创建工单
func (h *Handler) CreateTicket(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	var req struct {
		Subject  string `json:"subject" binding:"required,max=200"`
		Content  string `json:"content" binding:"required,max=5000"`
		Priority int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}
	ticket := model.Ticket{
		UserID:   userID,
		Subject:  req.Subject,
		Content:  req.Content,
		Priority: req.Priority,
		Status:   "open",
	}
	if err := model.GetDB().Create(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "创建工单失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": ticket})
}

// GetTicketDetail 获取工单详情
func (h *Handler) GetTicketDetail(c *gin.Context) {
	id := parseUint(c.Param("id"))
	userID := h.getCurrentUserID(c)
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)

	var ticket model.Ticket
	if err := model.GetDB().First(&ticket, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "工单不存在"})
		return
	}

	// 非管理员只能查看自己的工单
	if role != "admin" && ticket.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "msg": "无权查看"})
		return
	}

	// 加载回复
	var replies []model.TicketReply
	model.GetDB().Where("ticket_id = ?", ticket.ID).Order("created_at asc").Find(&replies)

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{
		"ticket":  ticket,
		"replies": replies,
	}})
}

// ReplyTicket 回复工单（管理员）
func (h *Handler) ReplyTicket(c *gin.Context) {
	id := parseUint(c.Param("id"))

	// 验证工单存在
	var ticket model.Ticket
	if err := model.GetDB().First(&ticket, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "工单不存在"})
		return
	}

	// 已关闭工单不允许管理员回复
	if ticket.Status == "closed" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "工单已关闭，无法回复"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required,max=5000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}
	reply := model.TicketReply{
		TicketID: id,
		UserID:   0, // 0 = 管理员回复
		Content:  req.Content,
	}
	if err := model.GetDB().Create(&reply).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "回复失败"})
		return
	}
	if err := model.GetDB().Model(&model.Ticket{}).Where("id = ?", reply.TicketID).Update("updated_at", time.Now()).Error; err != nil {
		log.Printf("更新工单时间失败: %v", err)
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// CloseTicket 关闭工单
func (h *Handler) CloseTicket(c *gin.Context) {
	id := c.Param("id")
	if err := model.GetDB().Model(&model.Ticket{}).Where("id = ?", parseUint(id)).Update("status", "closed").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "关闭失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UserReplyTicket 用户回复工单
func (h *Handler) UserReplyTicket(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	ticketID := c.Param("id")

	var req struct {
		Content string `json:"content" binding:"required,max=5000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	// 检查工单是否属于当前用户
	var ticket model.Ticket
	if err := model.GetDB().First(&ticket, parseUint(ticketID)).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "工单不存在"})
		return
	}
	if ticket.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "msg": "无权操作"})
		return
	}

	// 创建回复
	reply := model.TicketReply{
		TicketID: ticket.ID,
		UserID:   userID,
		Content:  req.Content,
	}
	if err := model.GetDB().Create(&reply).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "回复失败"})
		return
	}

	// 如果工单已关闭，重新打开
	if ticket.Status == "closed" {
		if err := model.GetDB().Model(&ticket).Update("status", "open").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "重新打开工单失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "回复成功"})
}
