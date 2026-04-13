<template>
  <div class="dashboard">
    <!-- 欢迎横幅 -->
    <div class="welcome-banner">
      <div class="banner-content">
        <div class="banner-text">
          <h1>欢迎回来，{{ username }}</h1>
          <p>NexCore 代理节点管理控制台，一切尽在掌控</p>
        </div>
        <div class="banner-illustration">
          <div class="illustration-circle circle-1"></div>
          <div class="illustration-circle circle-2"></div>
          <div class="illustration-circle circle-3"></div>
        </div>
      </div>
    </div>
    
    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card stat-card-blue">
        <div class="stat-icon">
          <CloudServerOutlined />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.totalNodes }}</div>
          <div class="stat-label">节点总数</div>
        </div>
        <div class="stat-trend up">
          <ArrowUpOutlined />
        </div>
      </div>
      
      <div class="stat-card stat-card-green">
        <div class="stat-icon">
          <CheckCircleOutlined />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.onlineNodes }}</div>
          <div class="stat-label">在线节点</div>
        </div>
      </div>
      
      <div class="stat-card stat-card-red">
        <div class="stat-icon">
          <CloseCircleOutlined />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.offlineNodes }}</div>
          <div class="stat-label">离线节点</div>
        </div>
      </div>
      
      <div class="stat-card stat-card-purple">
        <div class="stat-icon">
          <LineChartOutlined />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ formatTraffic(stats.totalUpload + stats.totalDownload) }}</div>
          <div class="stat-label">总流量</div>
        </div>
      </div>
    </div>
    
    <!-- 节点状态表格 -->
    <a-card class="nodes-card">
      <template #title>
        <div class="card-title">
          <span class="title-icon"><CloudServerOutlined /></span>
          节点状态
        </div>
      </template>
      <template #extra>
        <a-button type="primary" @click="refreshStats" :loading="loading">
          <template #icon><ReloadOutlined /></template>
          刷新数据
        </a-button>
      </template>
      
      <a-table 
        :columns="columns" 
        :dataSource="nodes" 
        :loading="loading" 
        rowKey="id"
        :pagination="{ pageSize: 10, showSizeChanger: true }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <div class="node-name">
              <span class="name-text">{{ record.name }}</span>
              <span class="ip-text">{{ record.ip }}</span>
            </div>
          </template>
          <template v-if="column.key === 'status'">
            <span :class="['status-badge', `status-${record.status}`]">
              {{ getStatusText(record.status) }}
            </span>
          </template>
          <template v-if="column.key === 'resource'">
            <div class="resource-bars">
              <div class="resource-item">
                <span class="resource-label">CPU</span>
                <div class="resource-bar">
                  <div class="resource-fill" :style="{ width: `${record.cpu || 0}%` }"></div>
                </div>
                <span class="resource-value">{{ record.cpu?.toFixed(1) }}%</span>
              </div>
              <div class="resource-item">
                <span class="resource-label">内存</span>
                <div class="resource-bar">
                  <div class="resource-fill memory" :style="{ width: `${record.mem || 0}%` }"></div>
                </div>
                <span class="resource-value">{{ record.mem?.toFixed(1) }}%</span>
              </div>
            </div>
          </template>
          <template v-if="column.key === 'traffic'">
            <div class="traffic-data">
              <div class="traffic-item upload">
                <ArrowUpOutlined />
                {{ formatTraffic(record.uploadTotal) }}
              </div>
              <div class="traffic-item download">
                <ArrowDownOutlined />
                {{ formatTraffic(record.downloadTotal) }}
              </div>
            </div>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="viewNode(record)">详情</a-button>
              <a-button type="link" size="small" @click="syncNodeStatus(record.id)">同步</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  CloudServerOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LineChartOutlined,
  ReloadOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined
} from '@ant-design/icons-vue'
import { getNodes, getStatsOverview, syncNode } from '@/api'

const router = useRouter()
const loading = ref(false)
const nodes = ref([])
const stats = ref({
  totalNodes: 0,
  onlineNodes: 0,
  offlineNodes: 0,
  totalUpload: 0,
  totalDownload: 0
})

const username = ref(localStorage.getItem('admin_username') || 'Admin')

const columns = [
  { title: '节点', key: 'name' },
  { title: '状态', key: 'status', width: 100 },
  { title: '资源使用', key: 'resource', width: 220 },
  { title: '流量统计', key: 'traffic', width: 140 },
  { title: '内核版本', dataIndex: 'xrayVersion', key: 'xrayVersion', width: 100 },
  { title: '操作', key: 'action', width: 120, fixed: 'right' }
]

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const getStatusText = (status) => {
  if (status === 'online') return '在线'
  if (status === 'offline') return '离线'
  return '未知'
}

const fetchData = async () => {
  loading.value = true
  try {
    const [nodesRes, statsRes] = await Promise.all([
      getNodes(),
      getStatsOverview()
    ])
    nodes.value = nodesRes.obj || []
    stats.value = statsRes.obj || stats.value
  } catch (e) {
    message.error('获取数据失败')
  } finally {
    loading.value = false
  }
}

const refreshStats = () => {
  fetchData()
}

const viewNode = (node) => {
  router.push(`/admin/nodes?id=${node.id}`)
}

const syncNodeStatus = async (id) => {
  try {
    await syncNode(id)
    message.success('同步成功')
    fetchData()
  } catch (e) {
    message.error('同步失败')
  }
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.dashboard {
  animation: fadeIn 0.3s ease;
}

/* 欢迎横幅 */
.welcome-banner {
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 50%, #0891b2 100%);
  border-radius: 16px;
  padding: 32px;
  margin-bottom: 24px;
  position: relative;
  overflow: hidden;
}

.banner-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  position: relative;
  z-index: 1;
}

.banner-text h1 {
  color: white;
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 8px;
}

.banner-text p {
  color: rgba(255, 255, 255, 0.85);
  font-size: 14px;
  margin: 0;
}

.banner-illustration {
  position: absolute;
  right: 40px;
  top: 50%;
  transform: translateY(-50%);
}

.illustration-circle {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
}

.circle-1 {
  width: 120px;
  height: 120px;
  right: 0;
  top: -60px;
}

.circle-2 {
  width: 80px;
  height: 80px;
  right: 80px;
  top: 20px;
}

.circle-3 {
  width: 60px;
  height: 60px;
  right: 40px;
  bottom: -30px;
}

/* 统计卡片网格 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}

@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 576px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}

.stat-card {
  background: white;
  border-radius: 14px;
  padding: 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  border: 1px solid rgba(0, 0, 0, 0.04);
  transition: all 0.2s ease;
  position: relative;
  overflow: hidden;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.06);
}

.stat-icon {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
}

.stat-card-blue .stat-icon {
  background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
  color: #3b82f6;
}

.stat-card-green .stat-icon {
  background: linear-gradient(135deg, #f0fdf4 0%, #bbf7d0 100%);
  color: #16a34a;
}

.stat-card-red .stat-icon {
  background: linear-gradient(135deg, #fef2f2 0%, #fecaca 100%);
  color: #dc2626;
}

.stat-card-purple .stat-icon {
  background: linear-gradient(135deg, #f9f0ff 0%, #efdbff 100%);
  color: #7c3aed;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #1e293b;
  line-height: 1.2;
}

.stat-label {
  font-size: 13px;
  color: #64748b;
  margin-top: 4px;
}

.stat-trend {
  position: absolute;
  right: 16px;
  top: 16px;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 12px;
}

.stat-trend.up {
  background: #f0fdf4;
  color: #16a34a;
}

/* 节点卡片 */
.nodes-card {
  border-radius: 14px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.title-icon {
  color: #3b82f6;
}

/* 节点名称 */
.node-name {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.name-text {
  font-weight: 600;
  color: #1e293b;
}

.ip-text {
  font-size: 12px;
  color: #64748b;
}

/* 状态徽章 */
.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.status-online {
  background: #f0fdf4;
  color: #16a34a;
}

.status-offline {
  background: #fef2f2;
  color: #dc2626;
}

.status-unknown {
  background: #f1f5f9;
  color: #64748b;
}

/* 资源使用条 */
.resource-bars {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.resource-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-label {
  font-size: 12px;
  color: #64748b;
  width: 28px;
  flex-shrink: 0;
}

.resource-bar {
  flex: 1;
  height: 6px;
  background: #e2e8f0;
  border-radius: 3px;
  overflow: hidden;
}

.resource-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6 0%, #60a5fa 100%);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.resource-fill.memory {
  background: linear-gradient(90deg, #0891b2 0%, #36cfc9 100%);
}

.resource-value {
  font-size: 12px;
  color: #475569;
  width: 42px;
  text-align: right;
  flex-shrink: 0;
}

/* 流量数据 */
.traffic-data {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.traffic-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #475569;
}

.traffic-item.upload {
  color: #3b82f6;
}

.traffic-item.download {
  color: #0891b2;
}

/* 动画 */
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 768px) {
  .welcome-banner {
    padding: 24px;
  }
  
  .banner-text h1 {
    font-size: 20px;
  }
  
  .banner-illustration {
    display: none;
  }
  
  .stat-card {
    padding: 20px;
  }
  
  .stat-value {
    font-size: 24px;
  }
}
</style>