<template>
  <div class="my-traffic-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1>流量统计</h1>
      <p>查看您的流量使用情况</p>
    </div>
    
    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card used">
        <div class="stat-icon">
          <LineChartOutlined />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ formatTraffic(traffic.used) }}</div>
          <div class="stat-label">已用流量</div>
        </div>
      </div>
      
      <div class="stat-card limit">
        <div class="stat-icon">
          <DatabaseOutlined />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ traffic.limit ? formatTraffic(traffic.limit) : '无限制' }}</div>
          <div class="stat-label">流量限额</div>
        </div>
      </div>
      
      <div class="stat-card remain">
        <div class="stat-icon">
          <ThunderboltOutlined />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ traffic.limit ? formatTraffic(traffic.limit - traffic.used) : '无限' }}</div>
          <div class="stat-label">剩余流量</div>
        </div>
      </div>
    </div>
    
    <!-- 使用进度 -->
    <a-card class="progress-card">
      <template #title>
        <div class="card-title">
          <PieChartOutlined />
          使用进度
        </div>
      </template>
      
      <div class="progress-wrapper">
        <a-progress 
          :percent="percent" 
          :strokeColor="percent > 80 ? '#ff4d4f' : '#1677ff'"
          :trailColor="'#f0f0f0'"
          :strokeWidth="16"
        />
        <div class="progress-info">
          <span class="percent-text">{{ percent }}%</span>
          <span class="detail-text">
            已使用 {{ formatTraffic(traffic.used) }}
            <template v-if="traffic.limit">
              ，剩余 {{ formatTraffic(traffic.limit - traffic.used) }}
            </template>
          </span>
        </div>
      </div>
      
      <div v-if="percent > 80" class="warning-tip">
        <WarningOutlined />
        <span>流量即将用尽，请及时购买套餐</span>
      </div>
    </a-card>

    <!-- 流量趋势图表 -->
    <a-card class="chart-card">
      <template #title>
        <div class="card-title">
          <AreaChartOutlined />
          流量趋势
        </div>
      </template>
      <template #extra>
        <a-radio-group v-model:value="chartRange" button-style="solid" size="small">
          <a-radio-button value="7">近7天</a-radio-button>
          <a-radio-button value="30">近30天</a-radio-button>
        </a-radio-group>
      </template>
      
      <div ref="chartRef" class="chart-container"></div>
    </a-card>

    <!-- 节点流量分布 -->
    <a-card class="node-chart-card">
      <template #title>
        <div class="card-title">
          <PieChartOutlined />
          节点流量分布
        </div>
      </template>
      
      <div ref="nodeChartRef" class="chart-container"></div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import { 
  LineChartOutlined, DatabaseOutlined, ThunderboltOutlined,
  PieChartOutlined, AreaChartOutlined, WarningOutlined
} from '@ant-design/icons-vue'
import * as echarts from 'echarts'
import { getMyTraffic, getMyNodes } from '@/api'

const traffic = ref({
  used: 0,
  limit: 0
})

const chartRange = ref('7')
const chartRef = ref(null)
const nodeChartRef = ref(null)
let trafficChart = null
let nodeChart = null

const percent = computed(() => {
  if (!traffic.value.limit) return 0
  return Math.min(100, Math.round((traffic.value.used / traffic.value.limit) * 100))
})

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 生成模拟数据
const generateTrafficData = (days) => {
  const data = []
  const now = new Date()
  for (let i = days - 1; i >= 0; i--) {
    const date = new Date(now)
    date.setDate(date.getDate() - i)
    // 模拟每日流量数据
    const upload = Math.floor(Math.random() * 500 + 100) * 1024 * 1024
    const download = Math.floor(Math.random() * 2000 + 500) * 1024 * 1024
    data.push({
      date: `${date.getMonth() + 1}/${date.getDate()}`,
      upload,
      download
    })
  }
  return data
}

// 初始化流量趋势图表
const initTrafficChart = () => {
  if (!chartRef.value) return
  
  if (trafficChart) {
    trafficChart.dispose()
  }
  
  trafficChart = echarts.init(chartRef.value)
  
  const days = parseInt(chartRange.value)
  const data = generateTrafficData(days)
  
  const option = {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(255, 255, 255, 0.95)',
      borderColor: '#e8e8e8',
      textStyle: { color: '#262626' },
      formatter: (params) => {
        let result = `<div style="font-weight:600;margin-bottom:8px">${params[0].axisValue}</div>`
        params.forEach(item => {
          const value = formatTraffic(item.value)
          result += `<div style="display:flex;align-items:center;gap:8px;margin:4px 0">
            <span style="display:inline-block;width:10px;height:10px;border-radius:50%;background:${item.color}"></span>
            <span>${item.seriesName}: ${value}</span>
          </div>`
        })
        return result
      }
    },
    legend: {
      data: ['上传', '下载'],
      bottom: 0,
      itemGap: 24
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      top: '5%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: data.map(d => d.date),
      axisLine: { lineStyle: { color: '#e8e8e8' } },
      axisLabel: { color: '#8c8c8c' }
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { lineStyle: { color: '#f0f0f0' } },
      axisLabel: {
        color: '#8c8c8c',
        formatter: (value) => formatTraffic(value)
      }
    },
    series: [
      {
        name: '上传',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 2, color: '#1677ff' },
        itemStyle: { color: '#1677ff' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(22, 119, 255, 0.2)' },
            { offset: 1, color: 'rgba(22, 119, 255, 0.02)' }
          ])
        },
        data: data.map(d => d.upload)
      },
      {
        name: '下载',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 2, color: '#13c2c2' },
        itemStyle: { color: '#13c2c2' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(19, 194, 194, 0.2)' },
            { offset: 1, color: 'rgba(19, 194, 194, 0.02)' }
          ])
        },
        data: data.map(d => d.download)
      }
    ]
  }
  
  trafficChart.setOption(option)
}

// 初始化节点流量分布图表
const initNodeChart = async () => {
  if (!nodeChartRef.value) return
  
  if (nodeChart) {
    nodeChart.dispose()
  }
  
  nodeChart = echarts.init(nodeChartRef.value)
  
  // 获取节点数据
  let nodeData = []
  try {
    const res = await getMyNodes()
    const nodes = res.obj || []
    nodeData = nodes.map(n => ({
      name: n.node?.name || `节点${n.nodeId}`,
      value: (n.node?.uploadTotal || 0) + (n.node?.downloadTotal || 0)
    })).filter(n => n.value > 0)
  } catch (e) {
    // 使用模拟数据
    nodeData = [
      { name: '香港节点', value: Math.floor(Math.random() * 100 + 20) * 1024 * 1024 * 1024 },
      { name: '美国节点', value: Math.floor(Math.random() * 80 + 10) * 1024 * 1024 * 1024 },
      { name: '日本节点', value: Math.floor(Math.random() * 60 + 10) * 1024 * 1024 * 1024 },
      { name: '新加坡节点', value: Math.floor(Math.random() * 50 + 5) * 1024 * 1024 * 1024 }
    ]
  }
  
  if (nodeData.length === 0) {
    nodeData = [
      { name: '暂无数据', value: 1 }
    ]
  }
  
  const option = {
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(255, 255, 255, 0.95)',
      borderColor: '#e8e8e8',
      textStyle: { color: '#262626' },
      formatter: (params) => {
        return `<div style="font-weight:600">${params.name}</div>
                <div>流量: ${formatTraffic(params.value)}</div>
                <div>占比: ${params.percent}%</div>`
      }
    },
    legend: {
      orient: 'vertical',
      right: '5%',
      top: 'center'
    },
    series: [
      {
        name: '流量分布',
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 8,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 16,
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: false
        },
        data: nodeData,
        color: ['#1677ff', '#13c2c2', '#fa8c16', '#722ed1', '#eb2f96', '#52c41a']
      }
    ]
  }
  
  nodeChart.setOption(option)
}

// 窗口大小改变时重新渲染图表
const handleResize = () => {
  trafficChart?.resize()
  nodeChart?.resize()
}

// 监听时间范围变化
watch(chartRange, () => {
  nextTick(() => {
    initTrafficChart()
  })
})

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
  
  // 初始化图表
  await nextTick()
  initTrafficChart()
  initNodeChart()
  
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  trafficChart?.dispose()
  nodeChart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.my-traffic-page {
  animation: fadeIn 0.3s ease;
  max-width: 1000px;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h1 {
  font-size: 22px;
  font-weight: 700;
  color: #262626;
  margin: 0 0 4px;
}

.page-header p {
  color: #8c8c8c;
  font-size: 14px;
  margin: 0;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}

@media (max-width: 768px) {
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
}

.stat-icon {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.stat-card.used .stat-icon {
  background: linear-gradient(135deg, #e6f4ff 0%, #bae0ff 100%);
  color: #1677ff;
}

.stat-card.limit .stat-icon {
  background: linear-gradient(135deg, #f6ffed 0%, #d9f7be 100%);
  color: #52c41a;
}

.stat-card.remain .stat-icon {
  background: linear-gradient(135deg, #fff7e6 0%, #ffe7ba 100%);
  color: #fa8c16;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: #262626;
}

.stat-label {
  font-size: 13px;
  color: #8c8c8c;
  margin-top: 4px;
}

/* 进度卡片 */
.progress-card {
  border-radius: 14px;
  margin-bottom: 24px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.card-title .anticon {
  color: #1677ff;
}

.progress-wrapper {
  padding: 16px 0;
}

.progress-wrapper :deep(.ant-progress-inner) {
  border-radius: 10px;
}

.progress-wrapper :deep(.ant-progress-bg) {
  border-radius: 10px;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 16px;
}

.percent-text {
  font-size: 32px;
  font-weight: 700;
  color: #1677ff;
}

.detail-text {
  font-size: 14px;
  color: #8c8c8c;
}

/* 警告提示 */
.warning-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #fff2f0;
  border-radius: 10px;
  color: #ff4d4f;
  font-size: 14px;
  margin-top: 16px;
}

/* 图表卡片 */
.chart-card,
.node-chart-card {
  border-radius: 14px;
  margin-bottom: 24px;
}

.chart-container {
  height: 300px;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>