package handler

import (
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// GetNodes 获取节点列表
func (h *Handler) GetNodes(c *gin.Context) {
	nodes, err := h.node.GetAllAdmin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取节点列表失败"})
		return
	}

	// 管理端返回节点信息，敏感字段脱敏
	var result []gin.H
	for _, n := range nodes {
		result = append(result, gin.H{
			"id": n.ID, "name": n.Name, "ip": n.IP, "port": n.Port,
			"username": n.Username, "password": maskPassword(n.Password),
			"sshPort": n.SSHPort, "sshUser": n.SSHUser, "sshPassword": maskPassword(n.SSHPassword),
			"agentKey": n.AgentKey, "apiToken": n.APIToken, "apiPort": n.APIPort,
			"masterUrl": n.MasterURL, "enable": n.Enable, "remark": n.Remark,
			"status": n.Status, "xrayVersion": n.XrayVersion,
			"cpu": n.CPU, "memory": n.Memory, "disk": n.Disk, "uptime": n.Uptime,
			"uploadTotal": n.UploadTotal, "downloadTotal": n.DownloadTotal,
			"lastSyncAt": n.LastSyncAt, "connected": n.Connected,
			"createdAt": n.CreatedAt, "updatedAt": n.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": result})
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

	// 敏感字段脱敏
	node.Password = maskPassword(node.Password)
	node.SSHPassword = maskPassword(node.SSHPassword)
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
		"name":     req.Name,
		"ip":       req.IP,
		"port":     req.Port,
		"username": req.Username,
		"ssh_port": req.SSHPort,
		"ssh_user": req.SSHUser,
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

// InstallNode 安装节点
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
