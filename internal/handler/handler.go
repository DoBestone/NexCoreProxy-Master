package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"nexcoreproxy-master/internal/model"
	"nexcoreproxy-master/internal/service"
)

// Handler API处理器
type Handler struct {
	node *service.NodeService
	user *service.UserService
}

// NewHandler 创建处理器
func NewHandler(services *service.Services) *Handler {
	return &Handler{
		node: services.Node,
		user: services.User,
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
		api.POST("/logout", h.Logout)
		api.GET("/userinfo", h.GetUserInfo)

		// 公开接口
		api.GET("/packages", h.GetPackages) // 套餐列表（公开）

		// 订阅链接（公开，通过token验证）
		api.GET("/sub/:token", h.GetSubscription)

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
				admin.GET("/nodes/:id/inbounds", h.GetNodeInbounds)
				admin.POST("/nodes/:id/inbounds", h.AddNodeInbound)
				admin.DELETE("/nodes/:id/inbounds/:inboundId", h.DeleteNodeInbound)

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
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
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
	var node model.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	node.ID = parseUint(id)

	if err := h.node.Update(&node); err != nil {
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
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": users})
}

// AddUser 添加用户
func (h *Handler) AddUser(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	if err := h.user.Create(user.Username, user.Password, user.Email, user.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "添加用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateUser 更新用户
func (h *Handler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	user.ID = parseUint(id)
	if err := h.user.Update(&user); err != nil {
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

// InstallNode SSH安装节点
func (h *Handler) InstallNode(c *gin.Context) {
	id := c.Param("id")

	if err := h.node.Install(parseUint(id)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "安装失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "安装成功"})
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