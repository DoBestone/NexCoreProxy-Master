package handler

import (
	"encoding/json"
	"io"
	"log"
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
	if msg := validatePasswordStrength(req.Password); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": msg})
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}

	if err := h.user.Create(req.Username, req.Password, req.Email, req.Role); err != nil {
		log.Printf("添加用户失败 [username=%s]: %v", req.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "添加用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateUser 更新用户
func (h *Handler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	userID := parseUint(id)

	// 获取现有用户
	var existingUser model.User
	if err := model.GetDB().First(&existingUser, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "用户不存在"})
		return
	}

	// 解析原始 JSON，仅更新请求中明确提供的字段
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	var rawJSON map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &rawJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	// 字段映射：JSON key → DB column
	fieldMap := map[string]string{
		"username":     "username",
		"email":        "email",
		"role":         "role",
		"balance":      "balance",
		"trafficLimit": "traffic_limit",
		"enable":       "enable",
		"remark":       "remark",
	}

	updates := map[string]interface{}{}
	for jsonKey, dbCol := range fieldMap {
		if raw, ok := rawJSON[jsonKey]; ok {
			var val interface{}
			if err := json.Unmarshal(raw, &val); err != nil {
				continue
			}
			updates[dbCol] = val
		}
	}

	// 密码单独处理：仅在提供且非空时才更新
	if raw, ok := rawJSON["password"]; ok {
		var pwd string
		if err := json.Unmarshal(raw, &pwd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "密码格式错误"})
			return
		}
		if pwd != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "密码加密失败"})
				return
			}
			updates["password"] = string(hashedPassword)
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
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
