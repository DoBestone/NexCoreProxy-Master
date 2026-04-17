import request from './request'

// 认证
export const login = (data) => request.post('/login', data)
export const register = (data) => request.post('/register', data)
export const logout = () => request.post('/logout')
export const getUserInfo = () => request.get('/user/info')
export const updatePassword = (data) => request.put('/user/password', data)

// ========== 公开接口 ==========

// 套餐列表（公开）
export const getPackages = () => request.get('/packages')

// 公告列表（公开）
export const getAnnouncements = () => request.get('/announcements')

// Turnstile配置（公开）
export const getTurnstileConfig = () => request.get('/turnstile-config')

// ========== 用户端接口 ==========

// 我的节点
export const getMyNodes = () => request.get('/my/nodes')
export const getMyTraffic = () => request.get('/my/traffic')
export const getMyTrafficTrend = (days = 7) => request.get('/my/traffic/trend', { params: { days } })
export const getMySubscribe = () => request.get('/my/subscribe')

// 我的订单
export const getMyOrders = () => request.get('/my/orders')
export const createOrder = (data) => request.post('/orders', data)

// ========== 管理员接口 ==========

// 用户管理
export const getUsers = () => request.get('/users')
export const addUser = (data) => request.post('/users', data)
export const updateUser = (id, data) => request.put(`/users/${id}`, data)
export const deleteUser = (id) => request.delete(`/users/${id}`)

// 节点管理
export const getNodes = () => request.get('/nodes')
export const addNode = (data) => request.post('/nodes', data)
export const updateNode = (id, data) => request.put(`/nodes/${id}`, data)
export const deleteNode = (id) => request.delete(`/nodes/${id}`)
export const testNode = (id) => request.post(`/nodes/${id}/test`)
export const syncNode = (id) => request.post(`/nodes/${id}/sync`)
export const installNode = (id) => request.post(`/nodes/${id}/install`)
export const getNodeInbounds = (id) => request.get(`/nodes/${id}/inbounds`)
export const addNodeInbound = (id, data) => request.post(`/nodes/${id}/inbounds`, data)
export const deleteNodeInbound = (nodeId, inboundId) => request.delete(`/nodes/${nodeId}/inbounds/${inboundId}`)
export const toggleNodeInbound = (nodeId, inboundId, enable) => request.post(`/nodes/${nodeId}/inbounds/${inboundId}/toggle`, { enable })
export const restartNodeXray = (id) => request.post(`/nodes/${id}/restart`)
export const sshNodeStatus = (id) => request.post(`/nodes/${id}/ssh-status`)
export const getNodeAPIToken = (id) => request.get(`/nodes/${id}/api-token`)
export const genNodeAPIToken = (id) => request.post(`/nodes/${id}/api-token`)
export const resetNodeCredentials = (id, data) => request.post(`/nodes/${id}/reset-credentials`, data)
export const checkNodeUpdate = (id) => request.post(`/nodes/${id}/check-update`)
export const updateNodeAgent = (id) => request.post(`/nodes/${id}/update-agent`)

// 中转规则
export const getRelayRules = () => request.get('/relay-rules')
export const createRelayRule = (data) => request.post('/relay-rules', data)
export const deleteRelayRule = (id) => request.delete(`/relay-rules/${id}`)
export const syncRelayRule = (id) => request.post(`/relay-rules/${id}/sync`)

// 套餐管理
export const getAllPackages = () => request.get('/admin/packages')
export const addPackage = (data) => request.post('/packages', data)
export const updatePackage = (id, data) => request.put(`/packages/${id}`, data)
export const deletePackage = (id) => request.delete(`/packages/${id}`)

// 订单管理
export const getAllOrders = () => request.get('/orders')
export const updateOrderStatus = (id, status) => request.put(`/orders/${id}/status`, { status })

// 入站模板
export const getTemplates = () => request.get('/templates')
export const addTemplate = (data) => request.post('/templates', data)
export const deleteTemplate = (id) => request.delete(`/templates/${id}`)

// 公告管理
export const getAdminAnnouncements = () => request.get('/admin/announcements')
export const addAnnouncement = (data) => request.post('/admin/announcements', data)
export const updateAnnouncement = (id, data) => request.put(`/admin/announcements/${id}`, data)
export const deleteAnnouncement = (id) => request.delete(`/admin/announcements/${id}`)

// 邮件配置
export const getEmailConfig = () => request.get('/admin/email-config')
export const updateEmailConfig = (data) => request.put('/admin/email-config', data)
export const testEmail = (email) => request.post('/admin/email-test', { email })

// 统计
export const getStatsOverview = () => request.get('/stats/overview')

// ========== 自研 agent 架构（v1） ==========

// Inbound 管理
export const listInbounds = (nodeId) => request.get('/inbounds', { params: nodeId ? { nodeId } : {} })
export const createInbound = (data) => request.post('/inbounds', data)
export const updateInbound = (id, data) => request.put(`/inbounds/${id}`, data)
export const deleteInbound = (id) => request.delete(`/inbounds/${id}`)
export const toggleInbound = (id, enable) => request.post(`/inbounds/${id}/toggle`, { enable })
export const provisionNode = (id, set = 'standard') => request.post(`/nodes/${id}/provision`, { set })
export const installNodeAgent = (id) => request.post(`/nodes/${id}/install-agent`)

// Package ↔ Inbound 关联
export const getPackageInbounds = (id) => request.get(`/packages/${id}/inbounds`)
export const setPackageInbounds = (id, inboundIds) => request.put(`/packages/${id}/inbounds`, { inboundIds })

// RelayBinding 管理
export const listRelayBindings = () => request.get('/relay-bindings')
export const createRelayBinding = (data) => request.post('/relay-bindings', data)
export const updateRelayBinding = (id, data) => request.put(`/relay-bindings/${id}`, data)
export const deleteRelayBinding = (id) => request.delete(`/relay-bindings/${id}`)
export const resyncRelayBinding = (id) => request.post(`/relay-bindings/${id}/resync`)

// ACME 证书
export const listCerts = () => request.get('/certs')
export const issueCert = (domain) => request.post('/certs/issue', { domain })
export const deleteCert = (id) => request.delete(`/certs/${id}`)
export const getAcmeSettings = () => request.get('/acme/settings')
export const updateAcmeSettings = (data) => request.put('/acme/settings', data)

// 系统更新
export const systemUpdate = {
  version: () => request.get('/system/version'),
  updateCheck: () => request.get('/system/update-check'),
  changelog: () => request.get('/system/changelog'),
  updatePrepare: () => request.post('/system/update-prepare'),
  update: (data) => request.post('/system/update', data),
  getConfig: () => request.get('/system/proxy-config'),
  saveConfig: (data) => request.put('/system/proxy-config', data),
}