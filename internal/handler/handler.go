package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"nexcoreproxy-master/internal/model"
	"nexcoreproxy-master/internal/service"
)

// Handler API处理器
type Handler struct {
	node  *service.NodeService
	user  *service.UserService
	email *service.EmailService
	agent *service.AgentManager
}

// NewHandler 创建处理器
func NewHandler(services *service.Services) *Handler {
	return &Handler{
		node:  services.Node,
		user:  services.User,
		email: services.Email,
		agent: services.Agent,
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// 静态文件
	r.Static("/assets", "./web/dist/assets")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	api := r.Group("/api")
	{
		// 认证
		api.POST("/login", h.Login)
		api.POST("/register", h.Register)  // 用户注册
		api.POST("/logout", h.Logout)
		api.GET("/userinfo", h.GetUserInfo)

		// 公开接口
		api.GET("/packages", h.GetPackages)       // 套餐列表（公开）
		api.GET("/announcements", h.GetAnnouncements) // 公告列表（公开）
		api.GET("/turnstile-config", h.GetTurnstileConfig) // Turnstile配置（公开）

		// 订阅链接（公开，通过token验证）
		api.GET("/sub/:token", h.GetSubscription)

		// Agent WebSocket 连接
		api.GET("/agent/ws", h.AgentWebSocket)

		// x-ui 面板反向代理 (通过密钥访问)
		panel := api.Group("/panel/:agentKey")
		{
			panel.GET("/*path", h.PanelProxy)
			panel.POST("/*path", h.PanelProxy)
			panel.PUT("/*path", h.PanelProxy)
			panel.DELETE("/*path", h.PanelProxy)
			panel.PATCH("/*path", h.PanelProxy)
		}

		// 需要认证的路由
		auth := api.Group("")
		auth.Use(h.AuthMiddleware())
		{
			// 用户信息
			auth.GET("/user/info", h.GetUserInfo)
			auth.PUT("/user/password", h.UpdatePassword)

			// 管理员路由
			admin := auth.Group("")
			admin.Use(h.AdminMiddleware())
			{
				// 用户管理
				admin.GET("/users", h.GetUsers)
				admin.POST("/users", h.AddUser)
				admin.PUT("/users/:id", h.UpdateUser)
				admin.DELETE("/users/:id", h.DeleteUser)

				// 节点管理
				admin.GET("/nodes", h.GetNodes)
				admin.POST("/nodes", h.AddNode)
				admin.GET("/nodes/:id", h.GetNode)
				admin.PUT("/nodes/:id", h.UpdateNode)
				admin.DELETE("/nodes/:id", h.DeleteNode)
				admin.POST("/nodes/:id/test", h.TestNode)
				admin.POST("/nodes/:id/sync", h.SyncNode)
				admin.POST("/nodes/:id/install", h.InstallNode)
				admin.POST("/nodes/:id/restart", h.RestartNodeXray)
				admin.POST("/nodes/:id/reset-credentials", h.ResetNodeCredentials)
				admin.POST("/nodes/:id/check-update", h.CheckNodeUpdate)
				admin.POST("/nodes/:id/update-agent", h.UpdateNodeAgent)
				admin.GET("/nodes/:id/inbounds", h.GetNodeInbounds)
				admin.POST("/nodes/:id/inbounds", h.AddNodeInbound)
				admin.DELETE("/nodes/:id/inbounds/:inboundId", h.DeleteNodeInbound)
				admin.POST("/nodes/:id/inbounds/:inboundId/toggle", h.ToggleNodeInbound)
				admin.POST("/nodes/:id/ssh-status", h.SSHNodeStatus)
				admin.POST("/nodes/:id/ssh-restart-xray", h.SSHRestartXray)
				admin.GET("/nodes/:id/api-token", h.GetNodeAPIToken)
				admin.POST("/nodes/:id/api-token", h.GenNodeAPIToken)

				// 套餐管理
				admin.POST("/packages", h.AddPackage)
				admin.PUT("/packages/:id", h.UpdatePackage)
				admin.DELETE("/packages/:id", h.DeletePackage)

				// 订单管理
				admin.GET("/orders", h.GetAllOrders)
				admin.PUT("/orders/:id/status", h.UpdateOrderStatus)

				// 工单管理
				admin.GET("/tickets", h.GetAllTickets)
				admin.POST("/tickets/:id/reply", h.ReplyTicket)
				admin.PUT("/tickets/:id/close", h.CloseTicket)

				// 入站模板
				admin.GET("/templates", h.GetTemplates)
				admin.POST("/templates", h.AddTemplate)
				admin.DELETE("/templates/:id", h.DeleteTemplate)

				// 公告管理
				admin.GET("/admin/announcements", h.GetAdminAnnouncements)
				admin.POST("/admin/announcements", h.AddAnnouncement)
				admin.PUT("/admin/announcements/:id", h.UpdateAnnouncement)
				admin.DELETE("/admin/announcements/:id", h.DeleteAnnouncement)

				// 邮件配置
				admin.GET("/admin/email-config", h.GetEmailConfig)
				admin.PUT("/admin/email-config", h.UpdateEmailConfig)
				admin.POST("/admin/email-test", h.TestEmail)

				// 统计
				admin.GET("/stats/overview", h.GetStatsOverview)
			}

			// 用户端接口
			// 我的节点
			auth.GET("/my/nodes", h.GetMyNodes)
			auth.GET("/my/traffic", h.GetMyTraffic)
			auth.GET("/my/subscribe", h.GetMySubscribe)

			// 订单
			auth.GET("/my/orders", h.GetMyOrders)
			auth.POST("/orders", h.CreateOrder)

			// 工单
			auth.GET("/my/tickets", h.GetMyTickets)
			auth.POST("/tickets", h.CreateTicket)
			auth.GET("/tickets/:id", h.GetTicketDetail)
			auth.POST("/my/tickets/:id/reply", h.UserReplyTicket)
		}
	}
}

// AuthMiddleware 认证中间件
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			if cookie, err := c.Cookie("token"); err == nil {
				token = cookie
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "未登录"})
			c.Abort()
			return
		}

		// 解析JWT Token
		claims, err := h.user.ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "Token无效或已过期"})
			c.Abort()
			return
		}

		// 将用户信息存入context
		c.Set("userId", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AdminMiddleware 管理员中间件
func (h *Handler) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "msg": "无权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// getCurrentUserID 从context获取当前用户ID
func (h *Handler) getCurrentUserID(c *gin.Context) uint {
	if userID, exists := c.Get("userId"); exists {
		return userID.(uint)
	}
	return 0
}

// Login 登录
func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		TurnstileToken string `json:"turnstileToken"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	// 验证 Turnstile
	secretKey := os.Getenv("TURNSTILE_SECRET_KEY")
	if secretKey != "" && req.TurnstileToken != "" {
		if !h.verifyTurnstile(req.TurnstileToken, secretKey, c.ClientIP()) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "人机验证失败，请重试"})
			return
		}
	}

	user, err := h.user.Authenticate(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": err.Error()})
		return
	}

	// 生成JWT Token
	token, err := h.user.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "生成Token失败"})
		return
	}

	// 设置 cookie
	c.SetCookie("token", token, 3600*24, "/", "", false, true)

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

// Logout 登出
func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetUserInfo 获取用户信息
func (h *Handler) GetUserInfo(c *gin.Context) {
	// TODO: 从 token 获取用户信息
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj": gin.H{
			"username": "admin",
			"role":     "admin",
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

	// TODO: 实现密码修改

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "密码修改成功"})
}

// GetNodes 获取节点列表
func (h *Handler) GetNodes(c *gin.Context) {
	nodes, err := h.node.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取节点列表失败"})
		return
	}

	// 返回完整密码，方便确认
	// 密码已加密存储，管理端可见

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": nodes})
}

// AddNode 添加节点
func (h *Handler) AddNode(c *gin.Context) {
	var node model.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	if node.Port == 0 {
		node.Port = 54321
	}
	if node.SSHPort == 0 {
		node.SSHPort = 22
	}

	if err := h.node.Create(&node); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "添加节点失败"})
		return
	}

	// 返回完整节点信息
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": node})
}

// GetNode 获取节点详情
func (h *Handler) GetNode(c *gin.Context) {
	id := c.Param("id")
	var node model.Node
	if err := model.GetDB().First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "节点不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": node})
}

// UpdateNode 更新节点
func (h *Handler) UpdateNode(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string `json:"name"`
		IP          string `json:"ip"`
		Port        int    `json:"port"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		SSHPort     int    `json:"sshPort"`
		SSHUser     string `json:"sshUser"`
		SSHPassword string `json:"sshPassword"`
		Remark      string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	nodeID := parseUint(id)

	// 获取现有节点
	var existingNode model.Node
	if err := model.GetDB().First(&existingNode, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "节点不存在"})
		return
	}

	// 构建更新数据
	updates := map[string]interface{}{
		"name":      req.Name,
		"ip":        req.IP,
		"port":      req.Port,
		"username":  req.Username,
		"ssh_port":  req.SSHPort,
		"ssh_user":  req.SSHUser,
		"remark":    req.Remark,
	}

	// 只有提供了密码才更新
	if req.Password != "" {
		updates["password"] = req.Password
	}
	if req.SSHPassword != "" {
		updates["ssh_password"] = req.SSHPassword
	}

	if err := model.GetDB().Model(&model.Node{}).Where("id = ?", nodeID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "更新节点失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteNode 删除节点
func (h *Handler) DeleteNode(c *gin.Context) {
	id := c.Param("id")

	if err := h.node.Delete(parseUint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "删除节点失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TestNode 测试节点连接
func (h *Handler) TestNode(c *gin.Context) {
	id := parseUint(c.Param("id"))

	if err := h.node.TestConnection(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "连接失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "连接成功"})
}

// SyncNode 同步节点状态
func (h *Handler) SyncNode(c *gin.Context) {
	id := parseUint(c.Param("id"))

	status, err := h.node.SyncStatus(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "同步失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": status})
}

// GetNodeInbounds 获取节点入站列表
func (h *Handler) GetNodeInbounds(c *gin.Context) {
	id := parseUint(c.Param("id"))

	inbounds, err := h.node.GetInbounds(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "获取入站列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": inbounds})
}

// AddNodeInbound 添加节点入站
func (h *Handler) AddNodeInbound(c *gin.Context) {
	id := parseUint(c.Param("id"))

	var inbound map[string]interface{}
	if err := c.ShouldBindJSON(&inbound); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	if err := h.node.AddInbound(id, inbound); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "添加入站失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteNodeInbound 删除节点入站
func (h *Handler) DeleteNodeInbound(c *gin.Context) {
	id := parseUint(c.Param("id"))
	inboundId := parseInt(c.Param("inboundId"))

	if err := h.node.DeleteInbound(id, inboundId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "删除入站失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ToggleNodeInbound 启用/禁用入站
func (h *Handler) ToggleNodeInbound(c *gin.Context) {
	id := parseUint(c.Param("id"))
	inboundId := parseInt(c.Param("inboundId"))

	var req struct {
		Enable bool `json:"enable"`
	}
	c.ShouldBindJSON(&req)

	if err := h.node.SSHEnableInbound(id, inboundId, req.Enable); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "操作失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SSHNodeStatus 通过SSH获取节点状态
func (h *Handler) SSHNodeStatus(c *gin.Context) {
	id := parseUint(c.Param("id"))

	result, err := h.node.SSHGetStatus(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": result})
}

// SSHRestartXray 通过SSH重启Xray
func (h *Handler) SSHRestartXray(c *gin.Context) {
	id := parseUint(c.Param("id"))

	if err := h.node.SSHRestartXray(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "重启失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetNodeAPIToken 获取节点API Token
func (h *Handler) GetNodeAPIToken(c *gin.Context) {
	id := parseUint(c.Param("id"))

	token, err := h.node.GetAPIToken(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": map[string]string{"token": token}})
}

// GenNodeAPIToken 生成新的API Token
func (h *Handler) GenNodeAPIToken(c *gin.Context) {
	id := parseUint(c.Param("id"))

	token, err := h.node.GenAPIToken(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": map[string]string{"token": token}})
}

// RestartNodeXray 重启节点 Xray
func (h *Handler) RestartNodeXray(c *gin.Context) {
	id := parseUint(c.Param("id"))

	if err := h.node.RestartXray(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "重启失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetTemplates 获取入站模板列表
func (h *Handler) GetTemplates(c *gin.Context) {
	var templates []model.InboundTemplate
	model.GetDB().Find(&templates)
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": templates})
}

// AddTemplate 添加入站模板
func (h *Handler) AddTemplate(c *gin.Context) {
	var template model.InboundTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
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
	id := c.Param("id")
	if err := model.GetDB().Delete(&model.InboundTemplate{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "删除模板失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetStatsOverview 获取统计概览
func (h *Handler) GetStatsOverview(c *gin.Context) {
	var nodes []model.Node
	model.GetDB().Find(&nodes)

	online := 0
	offline := 0
	var totalUpload, totalDownload int64

	for _, node := range nodes {
		if node.Status == "online" {
			online++
		} else {
			offline++
		}
		totalUpload += node.UploadTotal
		totalDownload += node.DownloadTotal
	}

	// 用户统计
	var totalUsers, activeUsers int64
	model.GetDB().Model(&model.User{}).Count(&totalUsers)
	model.GetDB().Model(&model.User{}).Where("enable = ?", true).Count(&activeUsers)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj": gin.H{
			"totalNodes":    len(nodes),
			"onlineNodes":   online,
			"offlineNodes":  offline,
			"totalUpload":   totalUpload,
			"totalDownload": totalDownload,
			"totalUsers":    totalUsers,
			"activeUsers":   activeUsers,
		},
	})
}

// ========== 用户管理 ==========

// GetUsers 获取用户列表
func (h *Handler) GetUsers(c *gin.Context) {
	var users []model.User
	model.GetDB().Find(&users)

	// 清空密码字段（bcrypt哈希，不需要返回）
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": users})
}

// AddUser 添加用户
func (h *Handler) AddUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		Balance  float64 `json:"balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	if req.Password == "" {
		req.Password = "123456" // 默认密码
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
		Username string  `json:"username"`
		Password string  `json:"password"`
		Email    string  `json:"email"`
		Role     string  `json:"role"`
		Balance  float64 `json:"balance"`
		TrafficLimit int64 `json:"trafficLimit"`
		Enable   bool    `json:"enable"`
		Remark   string  `json:"remark"`
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
		"username": req.Username,
		"email":    req.Email,
		"role":     req.Role,
		"balance":  req.Balance,
		"traffic_limit": req.TrafficLimit,
		"enable":   req.Enable,
		"remark":   req.Remark,
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

// ========== 用户端接口 ==========

// GetMyNodes 获取当前用户的节点
func (h *Handler) GetMyNodes(c *gin.Context) {
	userID := h.getCurrentUserID(c)

	var userNodes []model.UserNode
	model.GetDB().Where("user_id = ? AND enable = ?", userID, true).Preload("Node").Find(&userNodes)

	var result []gin.H
	for _, un := range userNodes {
		if un.Node.ID > 0 {
			result = append(result, gin.H{
				"id":       un.Node.ID,
				"name":     un.Node.Name,
				"ip":       un.Node.IP,
				"status":   un.Node.Status,
				"protocol": "multi",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": result})
}

// GetMyTraffic 获取当前用户的流量统计
func (h *Handler) GetMyTraffic(c *gin.Context) {
	userID := h.getCurrentUserID(c)

	var user model.User
	if err := model.GetDB().First(&user, userID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj": gin.H{
			"used":  user.TrafficUsed,
			"limit": user.TrafficLimit,
		},
	})
}

// ========== 套餐管理 ==========

// GetPackages 获取套餐列表
func (h *Handler) GetPackages(c *gin.Context) {
	var packages []model.Package
	model.GetDB().Where("enable = ?", true).Order("sort asc, price asc").Find(&packages)
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": packages})
}

// AddPackage 添加套餐
func (h *Handler) AddPackage(c *gin.Context) {
	var pkg model.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}
	if err := model.GetDB().Create(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "添加套餐失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": pkg})
}

// UpdatePackage 更新套餐
func (h *Handler) UpdatePackage(c *gin.Context) {
	id := c.Param("id")
	var pkg model.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}
	pkg.ID = parseUint(id)
	if err := model.GetDB().Save(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "更新套餐失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeletePackage 删除套餐
func (h *Handler) DeletePackage(c *gin.Context) {
	id := c.Param("id")
	if err := model.GetDB().Delete(&model.Package{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "删除套餐失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ========== 订单管理 ==========

// GetMyOrders 获取我的订单
func (h *Handler) GetMyOrders(c *gin.Context) {
	// TODO: 从 token 获取用户ID
	var orders []model.Order
	model.GetDB().Order("created_at desc").Limit(20).Find(&orders)
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": orders})
}

// GetAllOrders 获取所有订单（管理员）
func (h *Handler) GetAllOrders(c *gin.Context) {
	var orders []model.Order
	model.GetDB().Preload("User").Order("created_at desc").Find(&orders)
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": orders})
}

// CreateOrder 创建订单
func (h *Handler) CreateOrder(c *gin.Context) {
	userID := h.getCurrentUserID(c)

	var req struct {
		PackageID uint   `json:"packageId"`
		PayMethod string `json:"payMethod"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	// 获取套餐信息
	var pkg model.Package
	if err := model.GetDB().First(&pkg, req.PackageID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "套餐不存在"})
		return
	}

	// 如果是余额支付，直接扣除并分配节点
	if req.PayMethod == "balance" {
		if err := h.user.PurchasePackage(userID, req.PackageID); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
			return
		}
	}

	// 生成订单号
	orderNo := fmt.Sprintf("NCP%d%d", time.Now().UnixNano()/1000000, rand.Intn(1000))

	order := &model.Order{
		OrderNo:     orderNo,
		UserID:      userID,
		PackageID:   pkg.ID,
		PackageName: pkg.Name,
		Amount:      pkg.Price,
		Status:      "paid",
		PayMethod:   req.PayMethod,
		PaidAt:      &[]time.Time{time.Now()}[0],
	}

	if err := model.GetDB().Create(order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "创建订单失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": order, "msg": "购买成功"})
}

// UpdateOrderStatus 更新订单状态（管理员）
func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status": req.Status,
	}
	if req.Status == "paid" {
		updates["paid_at"] = &now
	}

	if err := model.GetDB().Model(&model.Order{}).Where("id = ?", parseUint(id)).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ========== 工单管理 ==========

// GetMyTickets 获取我的工单
func (h *Handler) GetMyTickets(c *gin.Context) {
	var tickets []model.Ticket
	model.GetDB().Order("created_at desc").Find(&tickets)
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": tickets})
}

// GetAllTickets 获取所有工单（管理员）
func (h *Handler) GetAllTickets(c *gin.Context) {
	var tickets []model.Ticket
	model.GetDB().Preload("User").Order("created_at desc").Find(&tickets)
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": tickets})
}

// CreateTicket 创建工单
func (h *Handler) CreateTicket(c *gin.Context) {
	var ticket model.Ticket
	if err := c.ShouldBindJSON(&ticket); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}
	ticket.Status = "open"
	if err := model.GetDB().Create(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "创建工单失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": ticket})
}

// GetTicketDetail 获取工单详情
func (h *Handler) GetTicketDetail(c *gin.Context) {
	id := c.Param("id")
	var ticket model.Ticket
	if err := model.GetDB().Preload("Replies").First(&ticket, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "工单不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": ticket})
}

// ReplyTicket 回复工单
func (h *Handler) ReplyTicket(c *gin.Context) {
	id := c.Param("id")
	var reply model.TicketReply
	if err := c.ShouldBindJSON(&reply); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}
	reply.TicketID = parseUint(id)
	if err := model.GetDB().Create(&reply).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "回复失败"})
		return
	}
	// 更新工单时间
	model.GetDB().Model(&model.Ticket{}).Where("id = ?", reply.TicketID).Update("updated_at", time.Now())
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
	userID := c.GetUint("userID")
	ticketID := c.Param("id")

	var req struct {
		Content string `json:"content" binding:"required"`
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
		model.GetDB().Model(&ticket).Update("status", "open")
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "回复成功"})
}

// InstallNode SSH安装节点
func (h *Handler) InstallNode(c *gin.Context) {
	id := c.Param("id")

	node, err := h.node.GetByID(parseUint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "节点不存在"})
		return
	}

	if err := h.node.Install(parseUint(id)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "安装失败: " + err.Error()})
		return
	}

	// 重新获取更新后的节点信息
	node, _ = h.node.GetByID(parseUint(id))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     "安装成功",
		"obj": gin.H{
			"ip":       node.IP,
			"port":     node.Port,
			"username": node.Username,
			"password": node.Password,
		},
	})
}

// GetSubscription 获取订阅链接（公开，通过token验证）
func (h *Handler) GetSubscription(c *gin.Context) {
	token := c.Param("token")

	// 解析token获取用户ID
	claims, err := h.user.ParseToken(token)
	if err != nil {
		c.String(http.StatusNotFound, "无效的订阅链接")
		return
	}

	sub, err := h.node.GenerateSubscription(claims.UserID)
	if err != nil {
		c.String(http.StatusInternalServerError, "生成订阅失败")
		return
	}

	c.String(http.StatusOK, sub)
}

// GetMySubscribe 获取我的订阅链接
func (h *Handler) GetMySubscribe(c *gin.Context) {
	userID := h.getCurrentUserID(c)

	// 生成JWT Token作为订阅token
	var user model.User
	if err := model.GetDB().First(&user, userID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "用户不存在"})
		return
	}

	token, err := h.user.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "生成订阅失败"})
		return
	}

	// 生成订阅内容
	sub, err := h.node.GenerateSubscription(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "生成订阅失败"})
		return
	}

	// 返回订阅链接URL
	baseURL := "http://" + c.Request.Host
	subURL := fmt.Sprintf("%s/api/sub/%s", baseURL, token)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj": gin.H{
			"url":     subURL,
			"content": sub,
		},
	})
}

// 辅助函数
func parseUint(s string) uint {
	var n uint
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + uint(c-'0')
		}
	}
	return n
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// Register 用户注册
func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Username       string `json:"username" binding:"required,min=3,max=50"`
		Password       string `json:"password" binding:"required,min=6"`
		Email          string `json:"email"`
		InviteCode     string `json:"inviteCode"`
		TurnstileToken string `json:"turnstileToken"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	// 验证 Turnstile
	secretKey := os.Getenv("TURNSTILE_SECRET_KEY")
	if secretKey != "" && req.TurnstileToken != "" {
		if !h.verifyTurnstile(req.TurnstileToken, secretKey, c.ClientIP()) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "人机验证失败，请重试"})
			return
		}
	}

	db := model.GetDB()

	// 检查用户名是否存在
	var existUser model.User
	if db.Where("username = ?", req.Username).First(&existUser).Error == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "用户名已存在"})
		return
	}

	// 检查邮箱是否已使用
	if req.Email != "" {
		if db.Where("email = ?", req.Email).First(&existUser).Error == nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": "邮箱已被使用"})
			return
		}
	}

	// 密码加密
	hashedPassword, err := h.user.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "密码加密失败"})
		return
	}

	// 创建用户
	user := model.User{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
		Role:     "user",
		Enable:   true,
	}

	// 处理邀请码
	if req.InviteCode != "" {
		var inviter model.User
		if db.Where("invite_code = ?", req.InviteCode).First(&inviter).Error == nil {
			user.InvitedBy = inviter.ID
			// 给邀请人奖励余额
			db.Model(&inviter).Update("balance", gorm.Expr("balance + ?", 5.0))
		}
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "注册失败"})
		return
	}

	// 生成邀请码
	inviteCode := fmt.Sprintf("NC%06d", user.ID)
	db.Model(&user).Update("invite_code", inviteCode)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     "注册成功",
		"obj": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

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
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
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

// GetEmailConfig 获取邮件配置
func (h *Handler) GetEmailConfig(c *gin.Context) {
	var config model.EmailConfig
	db := model.GetDB()
	
	if err := db.First(&config).Error; err != nil {
		// 返回空配置
		c.JSON(http.StatusOK, gin.H{"success": true, "obj": nil})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": config})
}

// UpdateEmailConfig 更新邮件配置
func (h *Handler) UpdateEmailConfig(c *gin.Context) {
	var req struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
		FromName string `json:"fromName"`
		UseTLS   bool   `json:"useTLS"`
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
			Host:     req.Host,
			Port:     req.Port,
			Username: req.Username,
			Password: req.Password,
			From:     req.From,
			FromName: req.FromName,
			UseTLS:   req.UseTLS,
			Enable:   req.Enable,
		}
		if err := db.Create(&config).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": "保存失败"})
			return
		}
	} else {
		// 更新配置
		updates := map[string]interface{}{
			"host":      req.Host,
			"port":      req.Port,
			"username":  req.Username,
			"from":      req.From,
			"from_name": req.FromName,
			"use_tls":   req.UseTLS,
			"enable":    req.Enable,
		}
		if req.Password != "" {
			updates["password"] = req.Password
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
	
	// 初始化邮件服务并发送测试邮件
	h.email.LoadConfig(config.Host, config.Port, config.Username, config.Password, config.From, config.FromName, config.UseTLS)
	
	if err := h.email.Send(req.Email, "NexCore 测试邮件", "<h1>测试成功</h1><p>这是一封测试邮件，邮件服务配置正常。</p>"); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "发送失败: " + err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "测试邮件已发送"})
}

// GetTurnstileConfig 获取 Turnstile 配置
func (h *Handler) GetTurnstileConfig(c *gin.Context) {
	siteKey := os.Getenv("TURNSTILE_SITE_KEY")
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{"siteKey": siteKey}})
}

// PanelProxy x-ui 面板反向代理
func (h *Handler) PanelProxy(c *gin.Context) {
	agentKey := c.Param("agentKey")
	
	// 根据 agentKey 查找节点
	var node model.Node
	if err := model.GetDB().Where("agent_key = ? AND enable = ?", agentKey, true).First(&node).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "节点不存在或已禁用"})
		return
	}
	
	// 获取请求路径
	path := c.Param("path")
	if path == "" {
		path = "/"
	}
	
	// 构建目标 URL
	targetURL := fmt.Sprintf("http://%s:%d%s", node.IP, node.Port, path)
	
	// 添加查询参数
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}
	
	// 创建反向代理
	target, err := url.Parse(targetURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "解析目标地址失败"})
		return
	}
	
	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// 自定义 Director 修改请求
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = fmt.Sprintf("%s:%d", node.IP, node.Port)
		req.Header.Set("X-Forwarded-Host", c.Request.Host)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Real-IP", c.ClientIP())
	}
	
	// 错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"success":false,"msg":"节点连接失败"}`))
	}
	
	// 执行代理
	proxy.ServeHTTP(c.Writer, c.Request)
}

// GenerateAgentKey 生成节点密钥
func (h *Handler) generateAgentKey() string {
	return service.GenerateAgentKey()
}

// AgentWebSocket Agent WebSocket 连接（预留）
func (h *Handler) AgentWebSocket(c *gin.Context) {
	// TODO: 实现 Agent 反向连接 WebSocket
	c.JSON(http.StatusOK, gin.H{"success": false, "msg": "功能开发中"})
}

// ========== 节点 SSH 管理 ==========

// ResetNodeCredentials 通过SSH重置节点面板账号密码
func (h *Handler) ResetNodeCredentials(c *gin.Context) {
	id := parseUint(c.Param("id"))

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "用户名和密码不能为空"})
		return
	}

	node, err := h.node.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "节点不存在"})
		return
	}

	if node.SSHPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "节点未配置SSH密码"})
		return
	}

	// SSH连接执行重置
	err = h.node.ResetCredentials(id, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "重置成功"})
}

// CheckNodeUpdate 检查Agent更新
func (h *Handler) CheckNodeUpdate(c *gin.Context) {
	id := parseUint(c.Param("id"))

	result, err := h.node.CheckUpdate(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": result})
}

// UpdateNodeAgent 更新Agent
func (h *Handler) UpdateNodeAgent(c *gin.Context) {
	id := parseUint(c.Param("id"))

	output, err := h.node.UpdateAgent(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error(), "output": output})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "更新成功", "output": output})
}
