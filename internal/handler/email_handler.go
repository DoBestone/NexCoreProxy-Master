package handler

import (
	"net/http"
	"strings"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// GetEmailConfig 获取邮件配置
func (h *Handler) GetEmailConfig(c *gin.Context) {
	var config model.EmailConfig
	db := model.GetDB()

	if err := db.First(&config).Error; err != nil {
		// 返回空配置
		c.JSON(http.StatusOK, gin.H{"success": true, "obj": nil})
		return
	}

	// 脱敏 API Key
	maskedKey := config.APIKey
	if len(maskedKey) > 10 {
		maskedKey = maskedKey[:10] + "****"
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{
		"id":       config.ID,
		"apiUrl":   config.APIURL,
		"apiKey":   maskedKey,
		"fromName": config.FromName,
		"enable":   config.Enable,
	}})
}

// UpdateEmailConfig 更新邮件配置
func (h *Handler) UpdateEmailConfig(c *gin.Context) {
	var req struct {
		APIURL   string `json:"apiUrl"`
		APIKey   string `json:"apiKey"`
		FromName string `json:"fromName"`
		Enable   bool   `json:"enable"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	db := model.GetDB()
	var config model.EmailConfig

	if err := db.First(&config).Error; err != nil {
		// 创建新配置
		config = model.EmailConfig{
			APIURL:   req.APIURL,
			APIKey:   req.APIKey,
			FromName: req.FromName,
			Enable:   req.Enable,
		}
		if err := db.Create(&config).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": "保存失败"})
			return
		}
	} else {
		// 更新配置
		updates := map[string]interface{}{
			"api_url":   req.APIURL,
			"from_name": req.FromName,
			"enable":    req.Enable,
		}
		// 只有提供了新的 API Key 才更新（不含脱敏标记）
		if req.APIKey != "" && !strings.Contains(req.APIKey, "****") {
			updates["api_key"] = req.APIKey
		}
		if err := db.Model(&config).Updates(updates).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": "更新失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "保存成功"})
}

// TestEmail 测试邮件发送
func (h *Handler) TestEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "请输入邮箱地址"})
		return
	}

	// 加载邮件配置
	var config model.EmailConfig
	if err := model.GetDB().Where("enable = ?", true).First(&config).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "邮件服务未配置或未启用"})
		return
	}

	// 加载配置到邮件服务
	h.email.LoadConfig(config.APIURL, config.APIKey, config.FromName)

	if err := h.email.Send(req.Email, "NexCore 测试邮件", "<h1>测试成功</h1><p>这是一封测试邮件，SMTP Lite API 配置正常。</p>"); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "发送失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "测试邮件已发送"})
}
