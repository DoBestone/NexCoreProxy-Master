import request from './request'

// 认证
export const login = (data) => request.post('/login', data)
export const logout = () => request.post('/logout')
export const getUserInfo = () => request.get('/userinfo')

// ========== 公开接口 ==========

// 套餐列表（公开）
export const getPackages = () => request.get('/packages')

// ========== 用户端接口 ==========

// 我的节点
export const getMyNodes = () => request.get('/my/nodes')
export const getMyTraffic = () => request.get('/my/traffic')
export const getMySubscribe = () => request.get('/my/subscribe')

// 我的订单
export const getMyOrders = () => request.get('/my/orders')
export const createOrder = (data) => request.post('/orders', data)

// 我的工单
export const getMyTickets = () => request.get('/my/tickets')
export const createTicket = (data) => request.post('/tickets', data)
export const getTicketDetail = (id) => request.get(`/tickets/${id}`)

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
export const restartNodeXray = (id) => request.post(`/nodes/${id}/restart`)

// 套餐管理
export const addPackage = (data) => request.post('/packages', data)
export const updatePackage = (id, data) => request.put(`/packages/${id}`, data)
export const deletePackage = (id) => request.delete(`/packages/${id}`)

// 订单管理
export const getAllOrders = () => request.get('/orders')
export const updateOrderStatus = (id, status) => request.put(`/orders/${id}/status`, { status })

// 工单管理
export const getAllTickets = () => request.get('/tickets')
export const replyTicket = (id, content) => request.post(`/tickets/${id}/reply`, { content })
export const closeTicket = (id) => request.put(`/tickets/${id}/close`)

// 入站模板
export const getTemplates = () => request.get('/templates')
export const addTemplate = (data) => request.post('/templates', data)
export const deleteTemplate = (id) => request.delete(`/templates/${id}`)

// 统计
export const getStatsOverview = () => request.get('/stats/overview')