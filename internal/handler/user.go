package handler

import (
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// GetUsers 获取用户列表
func (h *Handler) GetUsers(c *gin.Context) {
	var users []model.User
	model.GetDB().Limit(500).Find(&users)

	// 清空密码字段（bcrypt哈希，不需要返回）
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": users})
}

// AddUser 添加用户
func (h *Handler) AddUser(c *gin.Context) {
	var req struct {
		Username string  `json:"username"`
		Password string  `json:"password"`
		Email    string  `json:"email"`
		Role     string  `json:"role"`
		Balance  float64 `json:"balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	if req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "密码不能为空"})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "密码长度不能少于6位"})
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}

	if err := h.user.Create(req.Username, req.Password, req.Email, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "添加用户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateUser 更新用户
func (h *Handler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Username     string  `json:"username"`
		Password     string  `json:"password"`
		Email        string  `json:"email"`
		Role         string  `json:"role"`
		Balance      float64 `json:"balance"`
		TrafficLimit int64   `json:"trafficLimit"`
		Enable       bool    `json:"enable"`
		Remark       string  `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	userID := parseUint(id)

	// 获取现有用户
	var existingUser model.User
	if err := model.GetDB().First(&existingUser, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "用户不存在"})
		return
	}

	// 构建更新数据
	updates := map[string]interface{}{
		"username":      req.Username,
		"email":         req.Email,
		"role":          req.Role,
		"balance":       req.Balance,
		"traffic_limit": req.TrafficLimit,
		"enable":        req.Enable,
		"remark":        req.Remark,
	}

	// 只有提供了密码才更新
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "密码加密失败"})
			return
		}
		updates["password"] = string(hashedPassword)
	}

	if err := model.GetDB().Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "更新用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteUser 删除用户
func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.user.Delete(parseUint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "删除用户失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
