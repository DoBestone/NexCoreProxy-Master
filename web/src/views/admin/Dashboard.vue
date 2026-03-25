<template>
  <div class="dashboard">
    <!-- 统计卡片 -->
    <a-row :gutter="16" class="stats-row">
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card class="stat-card">
          <a-statistic title="节点总数" :value="stats.totalNodes">
            <template #prefix><CloudServerOutlined style="color: #1890ff" /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card class="stat-card">
          <a-statistic title="在线节点" :value="stats.onlineNodes" :value-style="{ color: '#52c41a' }">
            <template #prefix><CheckCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card class="stat-card">
          <a-statistic title="离线节点" :value="stats.offlineNodes" :value-style="{ color: '#f5222d' }">
            <template #prefix><CloseCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card class="stat-card">
          <a-statistic title="总流量" :value="formatTraffic(stats.totalUpload + stats.totalDownload)">
            <template #prefix><LineChartOutlined style="color: #722ed1" /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 节点列表 -->
    <a-card title="节点状态" class="nodes-card">
      <template #extra>
        <a-button type="primary" @click="refreshStats">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </template>
      <a-table :columns="columns" :dataSource="nodes" :loading="loading" rowKey="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-if="column.key === 'resource'">
            <div class="resource-bar">
              <span>CPU: {{ record.cpu?.toFixed(1) }}%</span>
              <a-progress :percent="record.cpu || 0" :showInfo="false" size="small" />
            </div>
            <div class="resource-bar">
              <span>内存: {{ record.mem?.toFixed(1) }}%</span>
              <a-progress :percent="record.mem || 0" :showInfo="false" size="small" status="active" />
            </div>
          </template>
          <template v-if="column.key === 'traffic'">
            <div>↑ {{ formatTraffic(record.uploadTotal) }}</div>
            <div>↓ {{ formatTraffic(record.downloadTotal) }}</div>
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
  ReloadOutlined
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

const columns = [
  { title: '节点名称', dataIndex: 'name', key: 'name' },
  { title: 'IP', dataIndex: 'ip', key: 'ip' },
  { title: '状态', key: 'status' },
  { title: '资源使用', key: 'resource', width: 200 },
  { title: '流量', key: 'traffic' },
  { title: '版本', dataIndex: 'xrayVersion', key: 'xrayVersion' },
  { title: '操作', key: 'action', width: 120 }
]

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const getStatusColor = (status) => {
  if (status === 'online') return 'green'
  if (status === 'offline') return 'red'
  return 'default'
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
  router.push(`/nodes?id=${node.id}`)
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
  /* styles */
}

.stats-row {
  margin-bottom: 24px;
}

.stat-card {
  border-radius: 8px;
}

.nodes-card {
  border-radius: 8px;
}

.resource-bar {
  margin-bottom: 4px;
}

.resource-bar span {
  font-size: 12px;
  color: #666;
}
</style>