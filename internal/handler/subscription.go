package handler

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

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
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := scheme + "://" + c.Request.Host
	subURL := fmt.Sprintf("%s/api/sub/%s", baseURL, token)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj": gin.H{
			"url":     subURL,
			"content": sub,
		},
	})
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

// GetMyNodes 获取当前用户的节点
func (h *Handler) GetMyNodes(c *gin.Context) {
	userID := h.getCurrentUserID(c)

	var userNodes []model.UserNode
	if err := model.GetDB().Where("user_id = ? AND enable = ?", userID, true).Preload("Node").Find(&userNodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取节点失败"})
		return
	}

	result := make([]gin.H, 0) // 确保返回 [] 而非 null
	for _, un := range userNodes {
		if un.Node.ID > 0 {
			result = append(result, gin.H{
				"id":       un.Node.ID,
				"name":     un.Node.Name,
				"ip":       un.Node.IP,
				"type":     un.Node.Type,
				"status":   un.Node.Status,
				"protocol": "multi",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": result})
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

	// SSRF 防护：解析目标地址并验证非内部地址
	nodeIP := net.ParseIP(node.IP)
	resolvedHost := node.IP // 使用解析后的 IP 构建 URL，防止 DNS rebinding
	if nodeIP == nil {
		// 域名情况：先 DNS 解析
		addrs, err := net.LookupHost(node.IP)
		if err != nil || len(addrs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "无法解析节点地址"})
			return
		}
		nodeIP = net.ParseIP(addrs[0])
		resolvedHost = addrs[0] // 使用解析后的 IP，防止二次解析到不同地址
	}
	if nodeIP != nil {
		if nodeIP.IsLoopback() || nodeIP.IsUnspecified() || nodeIP.IsLinkLocalUnicast() ||
			nodeIP.IsLinkLocalMulticast() || nodeIP.IsMulticast() || nodeIP.IsPrivate() ||
			nodeIP.Equal(net.ParseIP("169.254.169.254")) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "msg": "目标地址不允许"})
			return
		}
	}

	// 获取请求路径
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	// 构建目标 URL（使用解析后的 IP 防止 DNS rebinding 攻击）
	targetURL := fmt.Sprintf("http://%s:%d%s", resolvedHost, node.Port, path)

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
		req.Host = fmt.Sprintf("%s:%d", resolvedHost, node.Port)
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

// AgentWebSocket Agent WebSocket 连接（预留）
func (h *Handler) AgentWebSocket(c *gin.Context) {
	// TODO: 实现 Agent 反向连接 WebSocket
	c.JSON(http.StatusOK, gin.H{"success": false, "msg": "功能开发中"})
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
