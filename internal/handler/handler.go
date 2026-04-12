package handler

import (
	"net/http"
	"strconv"
	"strings"

	"nexcoreproxy-master/internal/service"

	"github.com/gin-gonic/gin"
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
		api.POST("/register", h.Register) // 用户注册
		api.POST("/logout", h.Logout)

		// 公开接口
		api.GET("/packages", h.GetPackages)                // 套餐列表（公开）
		api.GET("/announcements", h.GetAnnouncements)      // 公告列表（公开）
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

				// 系统更新
				admin.GET("/system/version", h.SystemVersion)
				admin.GET("/system/update-check", h.SystemUpdateCheck)
				admin.GET("/system/changelog", h.SystemChangelog)
				admin.POST("/system/update-prepare", h.SystemUpdatePrepare)
				admin.POST("/system/update", h.SystemUpdate)
				admin.GET("/system/proxy-config", h.SystemProxyConfig)
				admin.PUT("/system/proxy-config", h.SystemProxySaveConfig)
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
		role, exists := c.Get("role")
		roleStr, ok := role.(string)
		if !exists || !ok || roleStr != "admin" {
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

// 辅助函数
func parseUint(s string) uint {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// maskPassword 对密码进行脱敏处理
func maskPassword(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 3 {
		return strings.Repeat("*", len(s))
	}
	return s[:1] + strings.Repeat("*", len(s)-2) + s[len(s)-1:]
}
