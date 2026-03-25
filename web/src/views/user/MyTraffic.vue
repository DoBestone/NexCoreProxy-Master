<template>
  <div class="my-traffic">
    <a-row :gutter="16" class="stats-row">
      <a-col :xs="24" :sm="12">
        <a-card class="stat-card">
          <a-statistic title="已用流量" :value="formatTraffic(traffic.used)" :value-style="{ color: '#1890ff' }">
            <template #prefix><LineChartOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12">
        <a-card class="stat-card">
          <a-statistic title="流量限额" :value="traffic.limit ? formatTraffic(traffic.limit) : '无限制'">
            <template #prefix><DashboardOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <a-card title="流量使用" style="margin-top: 16px">
      <a-progress 
        :percent="traffic.percent" 
        :strokeColor="traffic.percent > 80 ? '#f5222d' : '#1890ff'"
        :format="() => traffic.percent + '%'"
      />
      <p style="margin-top: 16px; color: #666">
        {{ traffic.limit ? `已使用 ${formatTraffic(traffic.used)}，剩余 ${formatTraffic(traffic.limit - traffic.used)}` : '流量无限制' }}
      </p>
    </a-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { LineChartOutlined, DashboardOutlined } from '@ant-design/icons-vue'
import { getMyTraffic } from '@/api'

const traffic = ref({
  used: 0,
  limit: 0,
  percent: computed(() => {
    if (!traffic.value.limit) return 0
    return Math.min(100, Math.round((traffic.value.used / traffic.value.limit) * 100))
  })
})

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

onMounted(async () => {
  try {
    const res = await getMyTraffic()
    if (res.obj) {
      traffic.value.used = res.obj.used || 0
      traffic.value.limit = res.obj.limit || 0
    }
  } catch (e) {
    message.error('获取流量信息失败')
  }
})
</script>

<style scoped>
.my-traffic {
  max-width: 800px;
  margin: 0 auto;
}

.stats-row {
  margin-bottom: 16px;
}

.stat-card {
  border-radius: 8px;
}
</style>