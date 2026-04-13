package handler

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// loginRateLimiter 登录限速器 (IP → 失败次数 + 最后失败时间)
var loginRateLimiter = struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempt
}{
	attempts: make(map[string]*loginAttempt),
}

func init() {
	// 后台协程：每10分钟清理过期的限速记录，防止内存泄露
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			loginRateLimiter.mu.Lock()
			for ip, attempt := range loginRateLimiter.attempts {
				if time.Since(attempt.lastFail) > loginLockDuration {
					delete(loginRateLimiter.attempts, ip)
				}
			}
			loginRateLimiter.mu.Unlock()
		}
	}()
}

type loginAttempt struct {
	count    int
	lastFail time.Time
}

const (
	maxLoginAttempts  = 5               // 最大连续失败次数
	loginLockDuration = 15 * time.Minute // 锁定时长
)

// checkLoginRate 检查是否超过登录速率限制
func checkLoginRate(ip string) bool {
	loginRateLimiter.mu.Lock()
	defer loginRateLimiter.mu.Unlock()

	attempt, exists := loginRateLimiter.attempts[ip]
	if !exists {
		return true
	}

	// 锁定期过期，重置
	if time.Since(attempt.lastFail) > loginLockDuration {
		delete(loginRateLimiter.attempts, ip)
		return true
	}

	return attempt.count < maxLoginAttempts
}

// recordLoginFailure 记录登录失败
func recordLoginFailure(ip string) {
	loginRateLimiter.mu.Lock()
	defer loginRateLimiter.mu.Unlock()

	attempt, exists := loginRateLimiter.attempts[ip]
	if !exists {
		loginRateLimiter.attempts[ip] = &loginAttempt{count: 1, lastFail: time.Now()}
		return
	}
	attempt.count++
	attempt.lastFail = time.Now()
}

// clearLoginFailure 清除登录失败记录
func clearLoginFailure(ip string) {
	loginRateLimiter.mu.Lock()
	defer loginRateLimiter.mu.Unlock()
	delete(loginRateLimiter.attempts, ip)
}

// usernameRegex 用户名格式：字母数字下划线短横线
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`)

// validatePasswordStrength 密码强度校验：长度8-128，至少包含字母和数字
func validatePasswordStrength(password string) string {
	if len(password) < 8 || len(password) > 128 {
		return "密码长度需为8-128位"
	}
	hasLetter := false
	hasDigit := false
	for _, ch := range password {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			hasLetter = true
		}
		if ch >= '0' && ch <= '9' {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return "密码需同时包含字母和数字"
	}
	return ""
}

// Login 登录
func (h *Handler) Login(c *gin.Context) {
	// 登录限速检查
	clientIP := c.ClientIP()
	if !checkLoginRate(clientIP) {
		c.JSON(http.StatusTooManyRequests, gin.H{"success": false, "msg": "登录尝试过于频繁，请15分钟后再试"})
		return
	}

	var req struct {
		Username       string `json:"username"`
		Password       string `json:"password"`
		TurnstileToken string `json:"turnstileToken"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	// 验证 Turnstile
	secretKey := os.Getenv("TURNSTILE_SECRET_KEY")
	if secretKey != "" {
		if req.TurnstileToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "请完成人机验证"})
			return
		}
		if !h.verifyTurnstile(req.TurnstileToken, secretKey, c.ClientIP()) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "人机验证失败，请重试"})
			return
		}
	}

	user, err := h.user.Authenticate(req.Username, req.Password)
	if err != nil {
		recordLoginFailure(clientIP)
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "用户名或密码错误"})
		return
	}

	// 登录成功，清除失败记录
	clearLoginFailure(clientIP)

	// 生成JWT Token
	token, err := h.user.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "生成Token失败"})
		return
	}

	// 设置 cookie (Secure=true 用于 HTTPS, HttpOnly=true 防 XSS, SameSite=Lax 防 CSRF)
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("token", token, 3600*24, "/", "", isSecure, true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
			"token":    token,
		},
	})
}

// verifyTurnstile 验证 Cloudflare Turnstile
func (h *Handler) verifyTurnstile(token, secretKey, clientIP string) bool {
	apiURL := "https://challenges.cloudflare.com/turnstile/v0/siteverify"

	data := url.Values{}
	data.Set("secret", secretKey)
	data.Set("response", token)
	data.Set("remoteip", clientIP)

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}

	return result.Success
}

// Register 用户注册
func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Username       string `json:"username" binding:"required,min=3,max=50"`
		Password       string `json:"password" binding:"required,min=6,max=128"`
		Email          string `json:"email" binding:"omitempty,email"`
		InviteCode     string `json:"inviteCode"`
		TurnstileToken string `json:"turnstileToken"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	// 用户名格式校验：仅允许字母、数字、下划线、短横线
	if !usernameRegex.MatchString(req.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "用户名只能包含字母、数字、下划线和短横线"})
		return
	}

	// 密码强度校验
	if msg := validatePasswordStrength(req.Password); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": msg})
		return
	}

	// 验证 Turnstile
	secretKey := os.Getenv("TURNSTILE_SECRET_KEY")
	if secretKey != "" {
		if req.TurnstileToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "请完成人机验证"})
			return
		}
		if !h.verifyTurnstile(req.TurnstileToken, secretKey, c.ClientIP()) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "人机验证失败，请重试"})
			return
		}
	}

	db := model.GetDB()

	// 检查用户名是否存在
	var existUser model.User
	err := db.Where("username = ?", req.Username).First(&existUser).Error
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "用户名已存在"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "系统错误"})
		return
	}

	// 检查邮箱是否已使用
	if req.Email != "" {
		err = db.Where("email = ?", req.Email).First(&existUser).Error
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": "邮箱已被使用"})
			return
		} else if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "系统错误"})
			return
		}
	}

	// 密码加密
	hashedPassword, err := h.user.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "密码加密失败"})
		return
	}

	// 创建用户（使用事务保护邀请奖励）
	user := model.User{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
		Role:     "user",
		Enable:   true,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		// 处理邀请码
		if req.InviteCode != "" {
			var inviter model.User
			if tx.Where("invite_code = ?", req.InviteCode).First(&inviter).Error == nil {
				user.InvitedBy = inviter.ID
				// 给邀请人奖励余额
				if err := tx.Model(&inviter).Update("balance", gorm.Expr("balance + ?", 5.0)).Error; err != nil {
					return err
				}
			}
		}

		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// 生成随机邀请码
		inviteBytes := make([]byte, 4)
		if _, err := crand.Read(inviteBytes); err != nil {
			return fmt.Errorf("生成邀请码失败: %v", err)
		}
		inviteCode := fmt.Sprintf("NC%x", inviteBytes)
		return tx.Model(&user).Update("invite_code", inviteCode).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "注册失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     "注册成功",
		"obj": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// Logout 登出
func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetUserInfo 获取用户信息
func (h *Handler) GetUserInfo(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "未登录"})
		return
	}

	var user model.User
	if err := model.GetDB().First(&user, userID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
			"balance":  user.Balance,
			"enable":   user.Enable,
		},
	})
}

// UpdatePassword 修改密码
func (h *Handler) UpdatePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	if msg := validatePasswordStrength(req.NewPassword); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": msg})
		return
	}

	userID := h.getCurrentUserID(c)
	var user model.User
	if err := model.GetDB().First(&user, userID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "用户不存在"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "旧密码错误"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "密码加密失败"})
		return
	}

	if err := model.GetDB().Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "修改失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "密码修改成功"})
}

// GetTurnstileConfig 获取 Turnstile 配置
func (h *Handler) GetTurnstileConfig(c *gin.Context) {
	siteKey := os.Getenv("TURNSTILE_SITE_KEY")
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{"siteKey": siteKey}})
}
