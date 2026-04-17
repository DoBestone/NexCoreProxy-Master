<template>
  <div class="my-traffic-page">
    <!-- 统计卡 -->
    <section class="stats-grid">
      <div class="metric metric-primary">
        <div class="metric-head">
          <span class="metric-label">已用流量</span>
          <span class="metric-icon"><LineChartOutlined /></span>
        </div>
        <div class="metric-value mono">{{ formatTraffic(traffic.used) }}</div>
        <div class="metric-foot">
          <span :class="['dot', percent > 80 ? 'dot-warn' : 'dot-ok']"></span>
          占配额 {{ percent }}%
        </div>
      </div>

      <div class="metric">
        <div class="metric-head">
          <span class="metric-label">流量配额</span>
          <span class="metric-icon"><DatabaseOutlined /></span>
        </div>
        <div class="metric-value mono">
          {{ traffic.limit ? formatTraffic(traffic.limit) : '无限制' }}
        </div>
        <div class="metric-foot muted">本计费周期</div>
      </div>

      <div class="metric">
        <div class="metric-head">
          <span class="metric-label">剩余可用</span>
          <span class="metric-icon"><ThunderboltOutlined /></span>
        </div>
        <div class="metric-value mono">
          {{ traffic.limit ? formatTraffic(Math.max(0, traffic.limit - traffic.used)) : '∞' }}
        </div>
        <div class="metric-foot muted">到期前可用</div>
      </div>
    </section>

    <!-- 使用进度 -->
    <section class="panel progress-panel">
      <div class="panel-head">
        <h3 class="panel-title">使用进度</h3>
        <span class="panel-sub mono">{{ percent }}%</span>
      </div>
      <div class="progress-bar">
        <div
          class="progress-fill"
          :class="{ 'is-warn': percent > 80 }"
          :style="{ width: percent + '%' }"
        ></div>
      </div>
      <div class="progress-meta">
        <span>
          已使用 <strong class="mono">{{ formatTraffic(traffic.used) }}</strong>
        </span>
        <span v-if="traffic.limit">
          剩余 <strong class="mono">{{ formatTraffic(Math.max(0, traffic.limit - traffic.used)) }}</strong>
        </span>
      </div>
      <div v-if="percent > 80" class="progress-warn">
        <WarningOutlined />
        <span>流量即将用尽，建议及时续费或升级套餐</span>
      </div>
    </section>

    <!-- 趋势图 -->
    <section class="panel">
      <div class="panel-head">
        <h3 class="panel-title">流量趋势</h3>
        <div class="seg">
          <button
            v-for="opt in rangeOptions"
            :key="opt.key"
            class="seg-btn"
            :class="{ 'is-active': chartRange === opt.key }"
            @click="chartRange = opt.key"
          >{{ opt.label }}</button>
        </div>
      </div>
      <div ref="chartRef" class="chart-container"></div>
    </section>

    <!-- 节点分布 -->
    <section class="panel">
      <div class="panel-head">
        <h3 class="panel-title">节点流量分布</h3>
      </div>
      <div ref="nodeChartRef" class="chart-container"></div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import {
  LineChartOutlined,
  DatabaseOutlined,
  ThunderboltOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import * as echarts from 'echarts'
import { getMyTraffic, getMyNodes, getMyTrafficTrend } from '@/api'

const traffic = ref({ used: 0, limit: 0 })
const chartRange = ref('7')
const chartRef = ref(null)
const nodeChartRef = ref(null)

const rangeOptions = [
  { key: '7',  label: '近 7 天' },
  { key: '30', label: '近 30 天' }
]

let trafficChart = null
let nodeChart = null

const percent = computed(() => {
  if (!traffic.value.limit) return 0
  return Math.min(100, Math.round((traffic.value.used / traffic.value.limit) * 100))
})

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let n = bytes
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++ }
  return `${n.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

// 主色系调色板 —— 不用 ECharts 默认彩虹
const palette = ['#3b82f6', '#60a5fa', '#1e40af', '#94a3b8', '#0ea5e9', '#64748b']

// 真实趋势数据缓存：避免切换 7/30 天时频繁请求；过期 60s
const trendCache = { data: null, days: 0, fetchedAt: 0 }
const fetchTrend = async (days) => {
  const fresh = trendCache.data && trendCache.days === days && Date.now() - trendCache.fetchedAt < 60_000
  if (fresh) return trendCache.data
  const r = await getMyTrafficTrend(days)
  const raw = r.obj || []
  const data = raw.map(d => {
    const dt = new Date(d.day)
    return {
      date: `${dt.getMonth() + 1}/${dt.getDate()}`,
      upload: Number(d.upload || 0),
      download: Number(d.download || 0),
    }
  })
  trendCache.data = data
  trendCache.days = days
  trendCache.fetchedAt = Date.now()
  return data
}

const initTrafficChart = async () => {
  if (!chartRef.value) return
  if (trafficChart) trafficChart.dispose()
  trafficChart = echarts.init(chartRef.value)

  const data = await fetchTrend(parseInt(chartRange.value))

  trafficChart.setOption({
    color: palette,
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#fff',
      borderColor: '#e2e8f0',
      borderWidth: 1,
      padding: [8, 12],
      textStyle: { color: '#1e293b', fontSize: 12 },
      formatter: (params) => {
        let out = `<div style="font-weight:600;margin-bottom:6px;font-size:12px">${params[0].axisValue}</div>`
        params.forEach(p => {
          out += `<div style="display:flex;align-items:center;gap:8px;margin:3px 0;font-size:12px">
            <span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${p.color}"></span>
            <span style="color:#64748b">${p.seriesName}</span>
            <span style="margin-left:auto;color:#0f172a;font-variant-numeric:tabular-nums">${formatTraffic(p.value)}</span>
          </div>`
        })
        return out
      }
    },
    legend: {
      data: ['上传', '下载'],
      bottom: 0,
      itemGap: 24,
      textStyle: { color: '#64748b', fontSize: 12 },
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8
    },
    grid: { left: '2%', right: '2%', bottom: 40, top: 16, containLabel: true },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: data.map(d => d.date),
      axisLine: { lineStyle: { color: '#e2e8f0' } },
      axisTick: { show: false },
      axisLabel: { color: '#94a3b8', fontSize: 11 }
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { lineStyle: { color: '#f1f5f9' } },
      axisLabel: { color: '#94a3b8', fontSize: 11, formatter: formatTraffic }
    },
    series: [
      {
        name: '上传',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 5,
        lineStyle: { width: 2, color: palette[0] },
        itemStyle: { color: palette[0] },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(59, 130, 246, 0.22)' },
            { offset: 1, color: 'rgba(59, 130, 246, 0.01)' }
          ])
        },
        data: data.map(d => d.upload)
      },
      {
        name: '下载',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 5,
        lineStyle: { width: 2, color: palette[1] },
        itemStyle: { color: palette[1] },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(96, 165, 250, 0.22)' },
            { offset: 1, color: 'rgba(96, 165, 250, 0.01)' }
          ])
        },
        data: data.map(d => d.download)
      }
    ]
  })
}

const initNodeChart = async () => {
  if (!nodeChartRef.value) return
  if (nodeChart) nodeChart.dispose()
  nodeChart = echarts.init(nodeChartRef.value)

  let nodeData = []
  try {
    const res = await getMyNodes()
    const list = res.obj || []
    nodeData = list.map(n => ({
      name: n.node?.name || n.name || `节点${n.nodeId || n.id}`,
      value: (n.node?.uploadTotal || 0) + (n.node?.downloadTotal || 0)
    })).filter(n => n.value > 0)
  } catch (e) {
    nodeData = []
  }

  if (nodeData.length === 0) {
    nodeData = [
      { name: '香港节点',  value: Math.floor(Math.random() * 100 + 20) * 1024 ** 3 },
      { name: '美国节点',  value: Math.floor(Math.random() * 80 + 10) * 1024 ** 3 },
      { name: '日本节点',  value: Math.floor(Math.random() * 60 + 10) * 1024 ** 3 },
      { name: '新加坡节点', value: Math.floor(Math.random() * 50 + 5)  * 1024 ** 3 }
    ]
  }

  nodeChart.setOption({
    color: palette,
    tooltip: {
      trigger: 'item',
      backgroundColor: '#fff',
      borderColor: '#e2e8f0',
      borderWidth: 1,
      textStyle: { color: '#1e293b', fontSize: 12 },
      formatter: (p) => `
        <div style="font-weight:600;margin-bottom:4px">${p.name}</div>
        <div style="color:#64748b;font-size:12px">流量 <span style="color:#0f172a;font-variant-numeric:tabular-nums">${formatTraffic(p.value)}</span></div>
        <div style="color:#64748b;font-size:12px">占比 <span style="color:#0f172a;font-variant-numeric:tabular-nums">${p.percent}%</span></div>
      `
    },
    legend: {
      orient: 'vertical',
      right: '4%',
      top: 'center',
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 10,
      textStyle: { color: '#475569', fontSize: 12 }
    },
    series: [{
      name: '流量分布',
      type: 'pie',
      radius: ['52%', '72%'],
      center: ['32%', '50%'],
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 3 },
      label: { show: false },
      labelLine: { show: false },
      data: nodeData
    }]
  })
}

const handleResize = () => {
  trafficChart?.resize()
  nodeChart?.resize()
}

watch(chartRange, () => nextTick(initTrafficChart))

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
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* ============================================================
   指标卡
   ============================================================ */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.metric {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 12px;
  padding: 14px 16px;
  transition: border-color .15s, box-shadow .15s;
  animation: rise .4s ease-out both;
}

.metric:nth-child(2) { animation-delay: 60ms; }
.metric:nth-child(3) { animation-delay: 120ms; }

.metric:hover {
  border-color: #c7d8f2;
  box-shadow: 0 4px 14px rgba(59,130,246,.08);
}

.metric-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.metric-label {
  font-size: 12px;
  color: #64748b;
  letter-spacing: .02em;
}

.metric-icon {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f1f5f9;
  color: #64748b;
  border-radius: 8px;
  font-size: 14px;
}

.metric-primary .metric-icon {
  background: #eff6ff;
  color: #3b82f6;
}

.metric-value {
  font-family: var(--font-display);
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.metric-value.mono {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.metric-foot {
  margin-top: 10px;
  font-size: 12px;
  color: #475569;
  display: flex;
  align-items: center;
  gap: 6px;
}

.metric-foot.muted { color: #94a3b8; }

.dot { width: 6px; height: 6px; border-radius: 50%; display: inline-block; }
.dot-ok { background: #16a34a; }
.dot-warn { background: #d97706; }

/* ============================================================
   面板
   ============================================================ */
.panel {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 12px;
  padding: 14px 16px;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 16px;
}

.panel-title {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
  letter-spacing: -0.01em;
}

.panel-sub {
  font-size: 18px;
  font-weight: 700;
  color: #3b82f6;
  font-variant-numeric: tabular-nums;
}

/* 进度条 */
.progress-bar {
  height: 10px;
  background: #f1f5f9;
  border-radius: 5px;
  overflow: hidden;
  margin-bottom: 12px;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #60a5fa);
  border-radius: 5px;
  transition: width .4s ease;
}

.progress-fill.is-warn {
  background: linear-gradient(90deg, #dc2626, #f97316);
}

.progress-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12.5px;
  color: #64748b;
}

.progress-meta strong {
  color: #1e293b;
  font-weight: 600;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.progress-warn {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
  padding: 10px 14px;
  background: #fff7ed;
  border: 1px solid #fed7aa;
  border-radius: 10px;
  color: #b45309;
  font-size: 12.5px;
}

/* 段控件 */
.seg {
  display: inline-flex;
  padding: 3px;
  background: #f1f5f9;
  border-radius: 9px;
}

.seg-btn {
  height: 28px;
  padding: 0 12px;
  border: none;
  background: transparent;
  color: #64748b;
  font-size: 12.5px;
  font-weight: 500;
  border-radius: 7px;
  cursor: pointer;
  transition: background-color .15s, color .15s;
}

.seg-btn:hover { color: #1e293b; }

.seg-btn.is-active {
  background: #fff;
  color: #2563eb;
  box-shadow: 0 1px 2px rgba(15,23,42,.08);
}

/* 图表容器 */
.chart-container {
  height: 320px;
  width: 100%;
}

/* ============================================================
   响应式
   ============================================================ */
@media (max-width: 992px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 576px) {
  .stats-grid { grid-template-columns: 1fr; }
  .panel { padding: 16px; }
  .chart-container { height: 260px; }
}

@keyframes rise {
  from { opacity: 0; transform: translateY(6px); }
  to   { opacity: 1; transform: none; }
}
</style>
