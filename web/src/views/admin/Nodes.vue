<template>
  <div class="nodes-page">
    <div class="page-toolbar">
      <a-button @click="refreshAllNodes" :loading="refreshing">
        <ReloadOutlined /> 刷新状态
      </a-button>
      <a-button type="primary" @click="showAddModal">
        <PlusOutlined /> 添加节点
      </a-button>
    </div>

    <!-- 节点统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card total">
        <div class="stat-value">{{ nodes.length }}</div>
        <div class="stat-label">节点总数</div>
      </div>
      <div class="stat-card online">
        <div class="stat-value">{{ onlineCount }}</div>
        <div class="stat-label">在线节点</div>
      </div>
      <div class="stat-card offline">
        <div class="stat-value">{{ offlineCount }}</div>
        <div class="stat-label">离线节点</div>
      </div>
      <div class="stat-card traffic">
        <div class="stat-value">{{ formatTraffic(totalTraffic) }}</div>
        <div class="stat-label">总流量</div>
      </div>
    </div>
    
    <!-- 节点列表卡片 -->
    <a-card class="nodes-card" title="节点列表">
      <template #extra>
        <a-switch v-model:checked="autoRefresh" checked-children="自动刷新" un-checked-children="手动" />
      </template>
      
      <div class="nodes-grid">
        <div v-for="node in nodes" :key="node.id" class="node-card" :class="node.status">
          <!-- 节点头部 -->
          <div class="node-header">
            <div class="node-info">
              <span class="node-name">{{ node.name }}</span>
              <a-tag v-if="node.type === 'relay'" color="orange" size="small">中转</a-tag>
              <a-tag v-else-if="node.type === 'backend'" color="purple" size="small">落地</a-tag>
              <span class="node-ip">{{ node.ip }}:{{ node.port }}</span>
            </div>
            <span :class="['status-badge', node.status]">
              <span class="status-dot"></span>
              {{ getStatusText(node.status) }}
            </span>
          </div>
          
          <!-- 延迟显示 -->
          <div class="node-latency" v-if="node.latency !== undefined">
            <ThunderboltOutlined />
            <span :class="['latency-value', getLatencyClass(node.latency)]">
              {{ node.latency }}ms
            </span>
          </div>
          
          <!-- 资源使用 -->
          <div class="node-resources">
            <div class="resource-item">
              <span class="resource-label">CPU</span>
              <div class="resource-bar">
                <div class="resource-fill" :style="{ width: `${node.cpu || 0}%` }" :class="getResourceClass(node.cpu)"></div>
              </div>
              <span class="resource-value">{{ (node.cpu || 0).toFixed(1) }}%</span>
            </div>
            <div class="resource-item">
              <span class="resource-label">内存</span>
              <div class="resource-bar memory">
                <div class="resource-fill" :style="{ width: `${node.memory || 0}%` }" :class="getResourceClass(node.memory)"></div>
              </div>
              <span class="resource-value">{{ (node.memory || 0).toFixed(1) }}%</span>
            </div>
          </div>
          
          <!-- 流量统计 -->
          <div class="node-traffic">
            <div class="traffic-item">
              <ArrowUpOutlined /> {{ formatTraffic(node.uploadTotal) }}
            </div>
            <div class="traffic-item">
              <ArrowDownOutlined /> {{ formatTraffic(node.downloadTotal) }}
            </div>
          </div>
          
          <!-- 内核版本 -->
          <div class="node-version" v-if="node.xrayVersion">
            <TagOutlined /> 内核 {{ node.xrayVersion }}
          </div>

          <!-- 节点密钥和面板访问 -->
          <div class="node-key-panel" v-if="node.agentKey">
            <div class="key-info">
              <span class="key-label">节点密钥:</span>
              <code class="key-value">{{ node.agentKey }}</code>
              <button class="copy-key-btn" @click="copyAgentKey(node.agentKey)">
                <CopyOutlined />
              </button>
            </div>
            <div class="panel-info">
              <span class="panel-label">面板:</span>
              <code class="panel-value">{{ panelBaseUrl }}/{{ node.agentKey }}/</code>
              <button class="copy-key-btn" @click="copyPanelLink(node)">
                <CopyOutlined />
              </button>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="node-actions">
            <a-tooltip title="测速">
              <button class="action-btn" @click="pingNode(node)" :disabled="node.pinging">
                <ThunderboltOutlined :class="{ 'spin': node.pinging }" />
              </button>
            </a-tooltip>
            <a-tooltip title="同步状态">
              <button class="action-btn" @click="syncStatus(node)" :disabled="node.syncing">
                <SyncOutlined :class="{ 'spin': node.syncing }" />
              </button>
            </a-tooltip>
            <a-tooltip title="远程安装">
              <button class="action-btn primary" @click="doInstallNode(node)" :disabled="node.installing">
                <CloudUploadOutlined :class="{ 'spin': node.installing }" />
              </button>
            </a-tooltip>
            <a-tooltip title="重启服务">
              <button class="action-btn" @click="restartXray(node)">
                <ReloadOutlined />
              </button>
            </a-tooltip>
            <a-tooltip title="重置密码">
              <button class="action-btn" @click="showResetPasswordModal(node)">
                <KeyOutlined />
              </button>
            </a-tooltip>
            <a-tooltip title="检查更新">
              <button class="action-btn" @click="checkUpdate(node)">
                <CloudSyncOutlined />
              </button>
            </a-tooltip>
            <a-tooltip title="编辑">
              <button class="action-btn" @click="editNode(node)">
                <EditOutlined />
              </button>
            </a-tooltip>
            <a-popconfirm title="确定删除此节点?" @confirm="deleteNodeRecord(node.id)">
              <button class="action-btn danger">
                <DeleteOutlined />
              </button>
            </a-popconfirm>
          </div>
          
          <!-- 最后同步时间 -->
          <div class="node-sync-time" v-if="node.lastSyncAt">
            最后同步: {{ formatTime(node.lastSyncAt) }}
          </div>
        </div>
      </div>
      
      <!-- 空状态 -->
      <div v-if="!loading && nodes.length === 0" class="empty-state">
        <CloudServerOutlined class="empty-icon" />
        <p>暂无节点</p>
        <a-button type="primary" @click="showAddModal">添加第一个节点</a-button>
      </div>
    </a-card>

    <!-- 添加/编辑节点弹窗 -->
    <a-modal 
      v-model:open="modalVisible" 
      :title="editingNode ? '编辑节点' : '添加节点'" 
      @ok="handleSubmit" 
      :confirmLoading="submitting"
      :width="520"
    >
      <a-form :model="form" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="节点名称" required>
              <a-input v-model:value="form.name" placeholder="如: 香港节点1" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="IP地址" required>
              <a-input v-model:value="form.ip" placeholder="服务器IP" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="节点类型">
          <a-select v-model:value="form.type" style="width: 100%">
            <a-select-option value="standalone">独立节点</a-select-option>
            <a-select-option value="relay">中转节点</a-select-option>
            <a-select-option value="backend">落地节点</a-select-option>
          </a-select>
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="面板端口">
              <a-input-number v-model:value="form.port" :min="1" :max="65535" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="面板用户名">
              <a-input v-model:value="form.username" placeholder="自动安装后生成" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="面板密码">
          <a-input-password v-model:value="form.password" placeholder="自动安装后生成" />
        </a-form-item>
        <a-row :gutter="16" v-if="editingNode">
          <a-col :span="16">
            <a-form-item label="面板令牌">
              <a-input-group compact>
                <a-input v-model:value="form.apiToken" style="width: calc(100% - 60px)" readonly />
                <a-button @click="copyToClipboard(form.apiToken, '面板令牌')"><CopyOutlined /></a-button>
              </a-input-group>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="面板端口">
              <a-input-number v-model:value="form.apiPort" :min="1" :max="65535" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-divider>远程连接配置（节点安装需要）</a-divider>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="远程端口">
              <a-input-number v-model:value="form.sshPort" :min="1" :max="65535" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="远程用户">
              <a-input v-model:value="form.sshUser" placeholder="默认 root" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="远程密码">
              <a-input-password v-model:value="form.sshPassword" placeholder="远程密码" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="备注">
          <a-textarea v-model:value="form.remark" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 重置密码弹窗 -->
    <a-modal 
      v-model:open="resetPasswordVisible" 
      title="重置面板密码" 
      @ok="handleResetPassword"
      :width="400"
    >
      <a-form layout="vertical">
        <a-form-item label="用户名" required>
          <a-input v-model:value="resetPasswordForm.username" placeholder="面板用户名" />
        </a-form-item>
        <a-form-item label="密码" required>
          <a-input-password v-model:value="resetPasswordForm.password" placeholder="新密码" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, onDeactivated, watch } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { h } from 'vue'
import {
  CloudServerOutlined, PlusOutlined, ReloadOutlined, SyncOutlined,
  EditOutlined, DeleteOutlined, CloudUploadOutlined, ThunderboltOutlined,
  ArrowUpOutlined, ArrowDownOutlined, TagOutlined, CopyOutlined, KeyOutlined, CloudSyncOutlined
} from '@ant-design/icons-vue'
import { getNodes, addNode, updateNode, deleteNode, syncNode, installNode, testNode, restartNodeXray, resetNodeCredentials, checkNodeUpdate, updateNodeAgent } from '@/api'

const loading = ref(false)
const refreshing = ref(false)
const nodes = ref([])
const modalVisible = ref(false)
const editingNode = ref(null)
const submitting = ref(false)
const autoRefresh = ref(false)
const panelBaseUrl = `${window.location.origin}/api/panel`
let refreshTimer = null

const form = ref({
  name: '', ip: '', port: 54321, username: '', password: '',
  sshPort: 22, sshUser: 'root', sshPassword: '', type: 'standalone', remark: ''
})

// 统计数据
const onlineCount = computed(() => nodes.value.filter(n => n.status === 'online').length)
const offlineCount = computed(() => nodes.value.filter(n => n.status !== 'online').length)
const totalTraffic = computed(() => nodes.value.reduce((sum, n) => sum + (n.uploadTotal || 0) + (n.downloadTotal || 0), 0))

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

const getStatusText = (status) => {
  const map = { online: '在线', offline: '离线', unknown: '未知' }
  return map[status] || '未知'
}

const getLatencyClass = (latency) => {
  if (latency < 100) return 'fast'
  if (latency < 300) return 'medium'
  return 'slow'
}

const getResourceClass = (value) => {
  if (value > 80) return 'danger'
  if (value > 60) return 'warning'
  return 'normal'
}

const fetchNodes = async () => {
  loading.value = true
  try {
    const res = await getNodes()
    nodes.value = (res.obj || []).map(n => ({ ...n, pinging: false, syncing: false }))
  } catch (e) {
    message.error('获取节点列表失败')
  } finally {
    loading.value = false
  }
}

const refreshAllNodes = async () => {
  refreshing.value = true
  await fetchNodes()
  refreshing.value = false
}

// Ping测速
const pingNode = async (node) => {
  node.pinging = true
  const startTime = Date.now()
  try {
    const res = await testNode(node.id)
    node.latency = Date.now() - startTime
    if (res.success) {
      message.success(`${node.name} 延迟: ${node.latency}ms`)
    } else {
      node.latency = -1
      message.error('连接失败')
    }
  } catch (e) {
    node.latency = -1
    message.error('测速失败')
  } finally {
    node.pinging = false
  }
}

// 同步状态
const syncStatus = async (node) => {
  node.syncing = true
  try {
    const res = await syncNode(node.id)
    if (res.success) {
      message.success('同步成功')
      await fetchNodes()
    }
  } catch (e) {
    message.error('同步失败')
  } finally {
    node.syncing = false
  }
}

// 安装节点
const doInstallNode = async (node) => {
  node.installing = true
  try {
    const res = await installNode(node.id)
    if (res.success) {
      message.success('安装成功')
      if (res.obj) {
        Modal.success({
          title: '节点安装成功',
          content: h('div', {}, [
            h('p', {}, `IP: ${res.obj.ip}`),
            h('p', {}, `端口: ${res.obj.port}`),
            h('p', {}, `用户名: ${res.obj.username}`),
            h('p', {}, `密码: ${res.obj.password}`),
          ]),
        })
      }
      fetchNodes()
    } else {
      message.error(res.msg || '安装失败')
    }
  } catch (e) {
    message.error('安装失败')
  } finally {
    node.installing = false
  }
}

// 重启Xray
const restartXray = async (node) => {
  try {
    await restartNodeXray(node.id)
    message.success('重启命令已发送')
  } catch (e) {
    message.error('重启失败')
  }
}

onDeactivated(() => { modalVisible.value = false; resetPasswordVisible.value = false })

// 重置密码弹窗
const resetPasswordVisible = ref(false)
const resetPasswordNode = ref(null)
const resetPasswordForm = ref({ username: '', password: '' })

const showResetPasswordModal = (node) => {
  resetPasswordNode.value = node
  resetPasswordForm.value = { username: node.username || '', password: '' }
  resetPasswordVisible.value = true
}

const handleResetPassword = async () => {
  if (!resetPasswordForm.value.username || !resetPasswordForm.value.password) {
    message.warning('请填写用户名和密码')
    return
  }
  try {
    await resetNodeCredentials(resetPasswordNode.value.id, resetPasswordForm.value)
    message.success('密码重置成功')
    resetPasswordVisible.value = false
    fetchNodes()
  } catch (e) {
    message.error('重置失败')
  }
}

// 检查更新
const checkUpdate = async (node) => {
  try {
    const res = await checkNodeUpdate(node.id)
    if (res.success && res.obj) {
      const { currentVersion, latestVersion, needUpdate, status } = res.obj
      Modal.confirm({
        title: '3X-UI 面板更新',
        content: `服务状态: ${status === 'active' ? '运行中' : '未运行'}\n当前版本: ${currentVersion}\n最新版本: ${latestVersion}\n\n注意: 更新会重启面板服务，确定继续?`,
        okText: '更新',
        cancelText: '取消',
        onOk: () => doUpdateAgent(node)
      })
    }
  } catch (e) {
    message.error('检查更新失败')
  }
}

// 更新Agent
const doUpdateAgent = async (node) => {
  node.updating = true
  try {
    await updateNodeAgent(node.id)
    message.success('更新成功')
  } catch (e) {
    message.error('更新失败')
  } finally {
    node.updating = false
  }
}

const showAddModal = () => {
  editingNode.value = null
  form.value = { name: '', ip: '', port: 54321, username: '', password: '', sshPort: 22, sshUser: 'root', sshPassword: '', type: 'standalone', remark: '' }
  modalVisible.value = true
}

const editNode = (node) => {
  editingNode.value = node
  form.value = { ...node }
  modalVisible.value = true
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    if (editingNode.value) {
      await updateNode(editingNode.value.id, form.value)
      message.success('更新成功')
    } else {
      await addNode(form.value)
      message.success('添加成功')
    }
    modalVisible.value = false
    fetchNodes()
  } catch (e) {
    message.error('操作失败')
  } finally {
    submitting.value = false
  }
}

const deleteNodeRecord = async (id) => {
  try {
    await deleteNode(id)
    message.success('删除成功')
    fetchNodes()
  } catch (e) {
    message.error('删除失败')
  }
}

// 复制面板访问链接
const copyPanelLink = (node) => {
  const link = `${panelBaseUrl}/${node.agentKey}/`
  navigator.clipboard.writeText(link).then(() => {
    message.success('面板链接已复制')
  }).catch(() => {
    message.error('复制失败')
  })
}

// 复制AgentKey
const copyAgentKey = (agentKey) => {
  navigator.clipboard.writeText(agentKey).then(() => {
    message.success('节点密钥已复制')
  }).catch(() => {
    message.error('复制失败')
  })
}

// 复制到剪贴板
const copyToClipboard = (text, label = '内容') => {
  if (!text) {
    message.warning('无内容可复制')
    return
  }
  navigator.clipboard.writeText(text).then(() => {
    message.success(`${label}已复制`)
  }).catch(() => {
    message.error('复制失败')
  })
}

// 监听自动刷新开关变化
watch(autoRefresh, (val) => {
  if (val) {
    fetchNodes() // 打开时立即刷新一次
  }
})

// 自动刷新
const startAutoRefresh = () => {
  if (refreshTimer) clearInterval(refreshTimer)
  refreshTimer = setInterval(() => {
    if (autoRefresh.value) {
      fetchNodes()
    }
  }, 30000)
}

onMounted(() => {
  fetchNodes()
  startAutoRefresh()
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<style scoped>
.nodes-page { animation: fadeIn 0.3s ease; }

.page-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-bottom: 14px;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  text-align: center;
  box-shadow: 0 2px 8px rgba(0,0,0,0.04);
}

.stat-value { font-size: 28px; font-weight: 700; }
.stat-label { font-size: 13px; color: #64748b; margin-top: 4px; }
.stat-card.total .stat-value { color: #3b82f6; }
.stat-card.online .stat-value { color: #16a34a; }
.stat-card.offline .stat-value { color: #dc2626; }
.stat-card.traffic .stat-value { color: #7c3aed; }

@media (max-width: 992px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 576px) {
  .stats-grid { grid-template-columns: 1fr; }
  .page-toolbar .ant-btn { flex: 1; }
}

/* 节点卡片网格 */
.nodes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.node-card {
  background: white;
  border-radius: 14px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.04);
  border: 2px solid transparent;
  transition: all 0.2s;
}

.node-card.online { border-color: #86efac; }
.node-card.offline { border-color: #fecaca; opacity: 0.8; }

.node-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.node-info { display: flex; flex-direction: column; gap: 2px; }
.node-name { font-size: 16px; font-weight: 600; color: #1e293b; }
.node-ip { font-size: 12px; color: #64748b; }

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.status-dot { width: 6px; height: 6px; border-radius: 50%; }
.status-badge.online { background: #f0fdf4; color: #16a34a; }
.status-badge.online .status-dot { background: #16a34a; }
.status-badge.offline { background: #fef2f2; color: #dc2626; }
.status-badge.offline .status-dot { background: #dc2626; }
.status-badge.unknown { background: #f1f5f9; color: #64748b; }
.status-badge.unknown .status-dot { background: #64748b; }

/* 延迟显示 */
.node-latency {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  font-size: 14px;
}

.latency-value { font-weight: 600; }
.latency-value.fast { color: #16a34a; }
.latency-value.medium { color: #f59e0b; }
.latency-value.slow { color: #dc2626; }

/* 资源使用 */
.node-resources {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
  padding: 12px;
  background: #f8fafc;
  border-radius: 10px;
}

.resource-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-label { font-size: 12px; color: #64748b; width: 28px; }
.resource-bar {
  flex: 1;
  height: 6px;
  background: #e8e8e8;
  border-radius: 3px;
  overflow: hidden;
}

.resource-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.3s;
  background: linear-gradient(90deg, #3b82f6, #60a5fa);
}

.resource-fill.memory { background: linear-gradient(90deg, #0891b2, #36cfc9); }
.resource-fill.danger { background: linear-gradient(90deg, #dc2626, #ff7875); }
.resource-fill.warning { background: linear-gradient(90deg, #f59e0b, #ffc53d); }

.resource-value { font-size: 12px; color: #475569; width: 50px; text-align: right; }

/* 流量统计 */
.node-traffic {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
}

.traffic-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #475569;
}

.traffic-item:first-child { color: #3b82f6; }
.traffic-item:last-child { color: #0891b2; }

/* 版本信息 */
.node-version {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 12px;
}

/* AgentKey 和面板访问 */
.node-key-panel {
  margin-bottom: 12px;
  padding: 10px 12px;
  background: #f0f5ff;
  border-radius: 8px;
  border: 1px solid #d6e4ff;
}

.key-info, .panel-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.panel-info {
  margin-bottom: 0;
}

.key-label, .panel-label {
  font-size: 12px;
  color: #3b82f6;
  font-weight: 500;
  width: 70px;
  flex-shrink: 0;
}

.key-value, .panel-value {
  flex: 1;
  font-size: 11px;
  color: #475569;
  background: transparent;
  padding: 0;
  word-break: break-all;
  font-family: 'Monaco', 'Menlo', monospace;
}

.copy-key-btn {
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #3b82f6;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  color: white;
  transition: all 0.15s;
  flex-shrink: 0;
}

.copy-key-btn:hover {
  background: #2563eb;
}

/* 操作按钮 */
.node-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.action-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f1f5f9;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #475569;
  transition: all 0.15s;
}

.action-btn:hover { background: #eff6ff; color: #3b82f6; }
.action-btn.primary { background: #3b82f6; color: white; }
.action-btn.primary:hover { background: #2563eb; }
.action-btn.danger:hover { background: #fef2f2; color: #dc2626; }
.action-btn:disabled { opacity: 0.5; cursor: not-allowed; }

/* 同步时间 */
.node-sync-time {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 12px;
  text-align: right;
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 48px;
}

.empty-icon { font-size: 48px; color: #cbd5e1; margin-bottom: 16px; }
.empty-state p { color: #64748b; margin-bottom: 16px; }

.spin { animation: spin 1s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
@keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
</style>