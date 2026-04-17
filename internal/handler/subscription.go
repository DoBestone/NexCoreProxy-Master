package handler

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"nexcoreproxy-master/internal/model"
	"nexcoreproxy-master/internal/service"

	"github.com/gin-gonic/gin"
)

// GetSubscription 获取订阅链接（公开，通过持久订阅令牌验证）
//
// 格式选择优先级：?type=clash|singbox|v2rayn > UA 嗅探 > 默认 v2rayN base64
// 兼容旧调用：?flag=clash 仍然有效
func (h *Handler) GetSubscription(c *gin.Context) {
	token := c.Param("token")

	var user model.User
	if err := model.GetDB().Where("subscribe_token = ? AND enable = ?", token, true).First(&user).Error; err != nil {
		c.String(http.StatusNotFound, "无效的订阅链接")
		return
	}

	nodes, err := h.sub.GenerateForUser(user.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "生成订阅失败: "+err.Error())
		return
	}

	forceType := c.Query("type")
	if forceType == "" && c.Query("flag") == "clash" {
		forceType = "clash"
	}
	format := service.DetectFormat(c.GetHeader("User-Agent"), forceType)
	body, contentType := service.Render(format, nodes)

	// 流量信息塞进 Subscription-Userinfo 头（v2rayN/Clash 等会显示）
	if user.TrafficLimit > 0 || user.ExpireAt != nil {
		usage := fmt.Sprintf("upload=0; download=%d; total=%d", user.TrafficUsed, user.TrafficLimit)
		if user.ExpireAt != nil {
			usage += fmt.Sprintf("; expire=%d", user.ExpireAt.Unix())
		}
		c.Header("Subscription-Userinfo", usage)
	}
	c.Header("Profile-Update-Interval", "12")
	c.Data(http.StatusOK, contentType, []byte(body))
}

// GetMySubscribe 获取我的订阅链接（使用持久订阅令牌，URL 不变）
func (h *Handler) GetMySubscribe(c *gin.Context) {
	userID := h.getCurrentUserID(c)

	var user model.User
	if err := model.GetDB().First(&user, userID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "用户不存在"})
		return
	}

	// 历史账号未持有令牌时懒加载生成一次
	if user.SubscribeToken == "" {
		tok, err := h.user.ResetSubscribeToken(user.ID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": "生成订阅令牌失败"})
			return
		}
		user.SubscribeToken = tok
	}

	nodes, err := h.sub.GenerateForUser(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "生成订阅失败: " + err.Error()})
		return
	}
	body, _ := service.Render(service.FormatV2rayN, nodes)

	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := scheme + "://" + c.Request.Host
	subURL := fmt.Sprintf("%s/api/sub/%s", baseURL, user.SubscribeToken)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj": gin.H{
			"url":        subURL,
			"clashUrl":   subURL + "?type=clash",
			"singboxUrl": subURL + "?type=singbox",
			"content":    body,
			"nodeCount":  len(nodes),
		},
	})
}

// ResetMySubscribe 重置当前用户的订阅令牌，旧 URL 立即失效
func (h *Handler) ResetMySubscribe(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	tok, err := h.user.ResetSubscribeToken(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "重置订阅令牌失败"})
		return
	}

	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := scheme + "://" + c.Request.Host
	subURL := fmt.Sprintf("%s/api/sub/%s", baseURL, tok)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj": gin.H{
			"url":      subURL,
			"clashUrl": subURL + "?flag=clash",
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

// GetMyTrafficTrend 当前用户最近 N 天每日流量
//
// 数据源 user_traffic 表（agent push 入账写入）。N 限定 1-90，默认 7。
func (h *Handler) GetMyTrafficTrend(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	days := parseInt(c.DefaultQuery("days", "7"))
	if days <= 0 || days > 90 {
		days = 7
	}

	type bucket struct {
		Day      string
		Upload   int64
		Download int64
	}
	var rows []bucket
	model.GetDB().Raw(`
		SELECT DATE_FORMAT(bucket_hour, '%Y-%m-%d') AS day,
		       SUM(upload) AS upload, SUM(download) AS download
		FROM user_traffic
		WHERE user_id = ? AND bucket_hour >= DATE_SUB(CURRENT_DATE(), INTERVAL ? DAY)
		GROUP BY DATE_FORMAT(bucket_hour, '%Y-%m-%d')
		ORDER BY day
	`, userID, days-1).Scan(&rows)

	// 补齐空缺日期
	have := make(map[string]bucket, len(rows))
	for _, r := range rows {
		have[r.Day] = r
	}
	out := make([]gin.H, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		b := have[d]
		out = append(out, gin.H{
			"day": d, "upload": b.Upload, "download": b.Download,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": out})
}

// GetMyNodes 用户视角的节点列表（基于授权 Inbound 反查 Node，按节点去重）
func (h *Handler) GetMyNodes(c *gin.Context) {
	userID := h.getCurrentUserID(c)

	type row struct {
		NodeID   uint
		Name     string
		IP       string
		Type     string
		Region   string
		Status   string
		Protocols string
	}
	var rows []row
	err := model.GetDB().Raw(`
		SELECT n.id AS node_id, n.name, n.ip, n.type, n.region, n.status,
		       GROUP_CONCAT(DISTINCT i.protocol) AS protocols
		FROM nodes n
		JOIN inbounds i ON i.node_id = n.id AND i.enable = true
		JOIN package_inbounds pi ON pi.inbound_id = i.id
		JOIN orders o ON o.package_id = pi.package_id
		WHERE o.user_id = ? AND o.status = 'paid' AND n.enable = true
		GROUP BY n.id
		ORDER BY n.region, n.name
	`, userID).Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取节点失败"})
		return
	}
	result := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		result = append(result, gin.H{
			"id": r.NodeID, "name": r.Name, "ip": r.IP,
			"type": r.Type, "region": r.Region, "status": r.Status,
			"protocols": r.Protocols,
		})
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

// GetStatsOverview 获取统计概览（基于自研 agent 架构的新数据源）
//
// 返回：
//   - 节点：总数 / 在线 / backend vs relay 分布 / 已部署 ncp-agent 数
//   - 用户：总数 / 启用 / 含活跃订阅
//   - 流量：今日 (按 user_traffic 表 SUM) / 7 天趋势数组
//   - 在线设备：node_online_ips 当前条数（去重 user）
//   - 入站 / 中转 / 证书：基础数量
func (h *Handler) GetStatsOverview(c *gin.Context) {
	db := model.GetDB()

	// 节点维度
	var nodes []model.Node
	db.Find(&nodes)
	online, offline, backends, relays, installed := 0, 0, 0, 0, 0
	for _, n := range nodes {
		if n.Status == "online" {
			online++
		} else {
			offline++
		}
		role := n.Role
		if role == "" {
			role = "backend"
		}
		if role == "relay" {
			relays++
		} else {
			backends++
		}
		if n.Installed {
			installed++
		}
	}

	// 用户维度
	var totalUsers, activeUsers, paidUsers int64
	db.Model(&model.User{}).Count(&totalUsers)
	db.Model(&model.User{}).Where("enable = ?", true).Count(&activeUsers)
	db.Raw(`SELECT COUNT(DISTINCT user_id) FROM orders WHERE status = 'paid'`).Scan(&paidUsers)

	// 资源维度
	var inboundCount, relayCount, bindingCount, certCount int64
	db.Model(&model.Inbound{}).Where("enable = ?", true).Count(&inboundCount)
	db.Model(&model.Relay{}).Where("enable = ?", true).Count(&relayCount)
	db.Model(&model.RelayBinding{}).Where("enable = ?", true).Count(&bindingCount)
	db.Model(&model.Certificate{}).Where("status = ?", "issued").Count(&certCount)

	// 流量：按小时桶聚合，最近 7 天
	type bucket struct {
		Day      string
		Upload   int64
		Download int64
	}
	var buckets []bucket
	db.Raw(`
		SELECT DATE_FORMAT(bucket_hour, '%Y-%m-%d') AS day,
		       SUM(upload) AS upload, SUM(download) AS download
		FROM user_traffic
		WHERE bucket_hour >= DATE_SUB(CURRENT_DATE(), INTERVAL 6 DAY)
		GROUP BY DATE_FORMAT(bucket_hour, '%Y-%m-%d')
		ORDER BY day
	`).Scan(&buckets)

	// 当日合计 + 7 天合计
	var todayUp, todayDown, weekUp, weekDown int64
	today := time.Now().Format("2006-01-02")
	for _, b := range buckets {
		weekUp += b.Upload
		weekDown += b.Download
		if b.Day == today {
			todayUp = b.Upload
			todayDown = b.Download
		}
	}

	// 在线设备数（按 user 去重）
	var onlineDevices int64
	db.Raw(`SELECT COUNT(DISTINCT user_id) FROM node_online_ips
	        WHERE last_seen > DATE_SUB(NOW(), INTERVAL 5 MINUTE)`).Scan(&onlineDevices)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj": gin.H{
			"totalNodes":    len(nodes),
			"onlineNodes":   online,
			"offlineNodes":  offline,
			"backendNodes":  backends,
			"relayNodes":    relays,
			"installedNodes": installed,

			"totalUsers":   totalUsers,
			"activeUsers":  activeUsers,
			"paidUsers":    paidUsers,
			"onlineDevices": onlineDevices,

			"inboundCount":  inboundCount,
			"relayCount":    relayCount,
			"bindingCount":  bindingCount,
			"certCount":     certCount,

			"todayUpload":   todayUp,
			"todayDownload": todayDown,
			"weekUpload":    weekUp,
			"weekDownload":  weekDown,
			"trafficTrend":  buckets,

			// 兼容旧前端字段
			"totalUpload":   weekUp,
			"totalDownload": weekDown,
		},
	})
}
