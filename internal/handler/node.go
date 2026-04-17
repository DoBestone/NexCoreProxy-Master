package handler

import (
	"log"
	"net"
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// sanitizeNode 从节点数据中移除敏感字段
func sanitizeNode(node model.Node) gin.H {
	return gin.H{
		"id": node.ID, "name": node.Name, "ip": node.IP,
		"port": node.Port, "username": node.Username,
		"sshPort": node.SSHPort, "sshUser": node.SSHUser,
		"apiPort": node.APIPort,
		"type": node.Type, "enable": node.Enable, "remark": node.Remark,
		"status": node.Status, "xrayVersion": node.XrayVersion,
		"cpu": node.CPU, "memory": node.Memory, "disk": node.Disk,
		"uptime": node.Uptime, "uploadTotal": node.UploadTotal,
		"downloadTotal": node.DownloadTotal, "lastSyncAt": node.LastSyncAt,
		"connected": node.Connected,
		"createdAt": node.CreatedAt, "updatedAt": node.UpdatedAt,
	}
}

// sanitizeNodes 批量脱敏
func sanitizeNodes(nodes []model.Node) []gin.H {
	result := make([]gin.H, len(nodes))
	for i, n := range nodes {
		result[i] = sanitizeNode(n)
	}
	return result
}

// GetNodes 获取节点列表
func (h *Handler) GetNodes(c *gin.Context) {
	nodes, err := h.node.GetAllAdmin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取节点列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": sanitizeNodes(nodes)})
}

// AddNode 添加节点
func (h *Handler) AddNode(c *gin.Context) {
	var node model.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	// 名称和 IP 校验
	if len(node.Name) == 0 || len(node.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "节点名称长度需为1-100"})
		return
	}
	if node.IP == "" || (net.ParseIP(node.IP) == nil && len(node.IP) > 255) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "节点地址无效"})
		return
	}

	if node.Port == 0 {
		node.Port = 54321
	}
	if node.SSHPort == 0 {
		node.SSHPort = 22
	}
	if node.Port < 1 || node.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "端口范围无效 (1-65535)"})
		return
	}
	if node.SSHPort < 1 || node.SSHPort > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "远程端口范围无效 (1-65535)"})
		return
	}

	if err := h.node.Create(&node); err != nil {
		log.Printf("添加节点失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "添加节点失败"})
		return
	}

	// 返回脱敏后的节点信息
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": sanitizeNode(node)})
}

// GetNode 获取节点详情
func (h *Handler) GetNode(c *gin.Context) {
	id := parseUint(c.Param("id"))
	var node model.Node
	if err := model.GetDB().First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "节点不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": sanitizeNode(node)})
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
		Type        string `json:"type"`
		Remark      string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	nodeID := parseUint(id)

	// 输入校验
	if len(req.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "节点名称长度不能超过100"})
		return
	}
	if req.IP != "" && net.ParseIP(req.IP) == nil && len(req.IP) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "节点地址无效"})
		return
	}
	if req.Port != 0 && (req.Port < 1 || req.Port > 65535) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "端口范围无效 (1-65535)"})
		return
	}
	if req.SSHPort != 0 && (req.SSHPort < 1 || req.SSHPort > 65535) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "SSH端口范围无效 (1-65535)"})
		return
	}
	if len(req.Username) > 100 || len(req.SSHUser) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "用户名长度不能超过100"})
		return
	}
	if len(req.Remark) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "备注长度不能超过255"})
		return
	}

	// 获取现有节点
	var existingNode model.Node
	if err := model.GetDB().First(&existingNode, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "节点不存在"})
		return
	}

	// 校验节点类型
	validTypes := map[string]bool{"standalone": true, "relay": true, "backend": true, "": true}
	if !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "无效的节点类型"})
		return
	}

	// 构建更新数据
	updates := map[string]interface{}{
		"name":     req.Name,
		"ip":       req.IP,
		"port":     req.Port,
		"username": req.Username,
		"ssh_port": req.SSHPort,
		"ssh_user": req.SSHUser,
		"type":     req.Type,
		"remark":   req.Remark,
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
		log.Printf("测试节点连接失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "连接失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "连接成功"})
}

// SyncNode 同步节点状态
func (h *Handler) SyncNode(c *gin.Context) {
	id := parseUint(c.Param("id"))

	status, err := h.node.SyncStatus(id)
	if err != nil {
		log.Printf("同步节点状态失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "同步失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": status})
}

// GetNodeInbounds 获取节点入站列表
func (h *Handler) GetNodeInbounds(c *gin.Context) {
	id := parseUint(c.Param("id"))

	inbounds, err := h.node.GetInbounds(id)
	if err != nil {
		log.Printf("获取节点入站列表失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "获取入站列表失败"})
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
		log.Printf("添加节点入站失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "添加入站失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteNodeInbound 删除节点入站
func (h *Handler) DeleteNodeInbound(c *gin.Context) {
	id := parseUint(c.Param("id"))
	inboundId := parseInt(c.Param("inboundId"))

	if err := h.node.DeleteInbound(id, inboundId); err != nil {
		log.Printf("删除节点入站失败 [id=%d, inbound=%d]: %v", id, inboundId, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "删除入站失败"})
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	if err := h.node.SSHEnableInbound(id, inboundId, req.Enable); err != nil {
		log.Printf("切换入站状态失败 [id=%d, inbound=%d]: %v", id, inboundId, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "操作失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SSHNodeStatus 通过SSH获取节点状态
func (h *Handler) SSHNodeStatus(c *gin.Context) {
	id := parseUint(c.Param("id"))

	result, err := h.node.SSHGetStatus(id)
	if err != nil {
		log.Printf("SSH获取节点状态失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "获取状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": result})
}

// SSHRestartXray 通过SSH重启Xray
func (h *Handler) SSHRestartXray(c *gin.Context) {
	id := parseUint(c.Param("id"))

	if err := h.node.SSHRestartXray(id); err != nil {
		log.Printf("SSH重启Xray失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "重启失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetNodeAPIToken 获取节点API Token
func (h *Handler) GetNodeAPIToken(c *gin.Context) {
	id := parseUint(c.Param("id"))

	token, err := h.node.GetAPIToken(id)
	if err != nil {
		log.Printf("获取节点API Token失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "获取Token失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": map[string]string{"token": token}})
}

// GenNodeAPIToken 生成新的API Token
func (h *Handler) GenNodeAPIToken(c *gin.Context) {
	id := parseUint(c.Param("id"))

	token, err := h.node.GenAPIToken(id)
	if err != nil {
		log.Printf("生成节点API Token失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "生成Token失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": map[string]string{"token": token}})
}

// RestartNodeXray 重启节点 Xray
func (h *Handler) RestartNodeXray(c *gin.Context) {
	id := parseUint(c.Param("id"))

	if err := h.node.RestartXray(id); err != nil {
		log.Printf("重启节点Xray失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "重启失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// InstallNode 安装节点
func (h *Handler) InstallNode(c *gin.Context) {
	id := c.Param("id")

	if _, err := h.node.GetByID(parseUint(id)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "节点不存在"})
		return
	}

	if err := h.node.Install(parseUint(id)); err != nil {
		log.Printf("安装节点失败 [id=%s]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "安装失败"})
		return
	}

	// 重新获取更新后的节点信息
	updatedNode, err := h.node.GetByID(parseUint(id))
	if err != nil || updatedNode == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "msg": "安装成功"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     "安装成功",
		"obj":     sanitizeNode(*updatedNode),
	})
}

// InstallNodeAgent 用 SSH 部署 ncp-agent + xray（自研 agent 架构）
//
// 与 InstallNode（旧版 3x-ui）并存。新建节点推荐走这个路径。
func (h *Handler) InstallNodeAgent(c *gin.Context) {
	id := parseUint(c.Param("id"))
	if _, err := h.node.GetByID(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "节点不存在"})
		return
	}
	if err := h.node.InstallAgent(id); err != nil {
		log.Printf("部署 ncp-agent 失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	updated, err := h.node.GetByID(id)
	if err != nil || updated == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "msg": "部署成功"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "部署成功", "obj": sanitizeNode(*updated)})
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
		log.Printf("重置节点凭证失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "重置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "重置成功"})
}

// CheckNodeUpdate 检查Agent更新
func (h *Handler) CheckNodeUpdate(c *gin.Context) {
	id := parseUint(c.Param("id"))

	result, err := h.node.CheckUpdate(id)
	if err != nil {
		log.Printf("检查节点更新失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "检查更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": result})
}

// UpdateNodeAgent 更新Agent
func (h *Handler) UpdateNodeAgent(c *gin.Context) {
	id := parseUint(c.Param("id"))

	output, err := h.node.UpdateAgent(id)
	if err != nil {
		log.Printf("更新节点Agent失败 [id=%d]: %v", id, err)
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "更新成功", "output": output})
}
