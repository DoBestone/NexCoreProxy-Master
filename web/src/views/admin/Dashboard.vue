<template>
  <div class="dashboard">
    <!-- Hero -->
    <section class="hero">
      <div class="hero-main">
        <div>
          <span class="hero-eyebrow">概览</span>
          <h1 class="hero-title">{{ greet }}{{ username }}</h1>
          <p class="hero-sub">
            <strong class="mono">{{ stats.totalNodes || 0 }}</strong> 节点 ·
            <strong class="mono">{{ stats.onlineNodes || 0 }}</strong> 在线 ·
            今日 <strong class="mono">{{ formatTraffic((stats.todayUpload||0) + (stats.todayDownload||0)) }}</strong> ·
            7 天 <strong class="mono">{{ formatTraffic((stats.weekUpload||0) + (stats.weekDownload||0)) }}</strong>
          </p>
        </div>
        <div class="hero-actions">
          <a-button size="large" @click="refreshStats" :loading="loading">
            <ReloadOutlined /> 刷新
          </a-button>
          <a-button type="primary" size="large" @click="$router.push('/admin/nodes')">
            <CloudServerOutlined /> 节点管理
          </a-button>
        </div>
      </div>
    </section>

    <!-- 指标卡 -->
    <section class="metrics">
      <div class="metric metric-primary">
        <div class="metric-head">
          <span class="metric-label">节点总数</span>
          <span class="metric-icon"><CloudServerOutlined /></span>
        </div>
        <div class="metric-value mono">{{ stats.totalNodes || 0 }}</div>
        <div class="metric-foot muted">包含离线与未知</div>
      </div>

      <div class="metric">
        <div class="metric-head">
          <span class="metric-label">在线节点</span>
          <span class="metric-icon icon-ok"><CheckCircleOutlined /></span>
        </div>
        <div class="metric-value mono">{{ stats.onlineNodes || 0 }}</div>
        <div class="metric-foot">
          <span class="dot dot-ok"></span>
          在线率 {{ onlineRate }}%
        </div>
      </div>

      <div class="metric">
        <div class="metric-head">
          <span class="metric-label">离线节点</span>
          <span class="metric-icon icon-danger"><CloseCircleOutlined /></span>
        </div>
        <div class="metric-value mono">{{ stats.offlineNodes || 0 }}</div>
        <div class="metric-foot muted">
          <a class="link" @click="$router.push('/admin/nodes')">查看详情 →</a>
        </div>
      </div>

      <div class="metric">
        <div class="metric-head">
          <span class="metric-label">7 天流量</span>
          <span class="metric-icon"><LineChartOutlined /></span>
        </div>
        <div class="metric-value mono">
          {{ formatTraffic((stats.weekUpload||0) + (stats.weekDownload||0)) }}
        </div>
        <div class="metric-foot muted">
          ↑ <span class="mono">{{ formatTraffic(stats.weekUpload) }}</span> ·
          ↓ <span class="mono">{{ formatTraffic(stats.weekDownload) }}</span>
        </div>
      </div>
    </section>

    <!-- 二级资源指标 -->
    <section class="sub-metrics">
      <div class="sub-card">
        <span class="sub-k">Backend</span>
        <span class="sub-v mono">{{ stats.backendNodes || 0 }}</span>
      </div>
      <div class="sub-card">
        <span class="sub-k">Relay</span>
        <span class="sub-v mono">{{ stats.relayNodes || 0 }}</span>
      </div>
      <div class="sub-card">
        <span class="sub-k">已部署 agent</span>
        <span class="sub-v mono">{{ stats.installedNodes || 0 }}</span>
      </div>
      <div class="sub-card">
        <span class="sub-k">在线设备</span>
        <span class="sub-v mono">{{ stats.onlineDevices || 0 }}</span>
      </div>
      <div class="sub-card">
        <span class="sub-k">入站</span>
        <span class="sub-v mono">{{ stats.inboundCount || 0 }}</span>
      </div>
      <div class="sub-card">
        <span class="sub-k">中转</span>
        <span class="sub-v mono">{{ stats.relayCount || 0 }} <span class="sub-aux">/ {{ stats.bindingCount || 0 }} 绑定</span></span>
      </div>
      <div class="sub-card">
        <span class="sub-k">证书</span>
        <span class="sub-v mono">{{ stats.certCount || 0 }}</span>
      </div>
      <div class="sub-card">
        <span class="sub-k">付费用户</span>
        <span class="sub-v mono">{{ stats.paidUsers || 0 }} <span class="sub-aux">/ {{ stats.activeUsers || 0 }}</span></span>
      </div>
    </section>

    <!-- 7 天流量趋势 mini-chart（纯 div 柱图，避免引入 echarts） -->
    <section class="trend-panel" v-if="trendDays.length">
      <div class="trend-head">
        <span class="trend-title">最近 7 天流量</span>
        <span class="trend-sub mono">峰值 {{ formatTraffic(trendMax) }}</span>
      </div>
      <div class="trend-bars">
        <div v-for="d in trendDays" :key="d.day" class="trend-day">
          <div class="trend-stack" :title="`${d.day}: ↑${formatTraffic(d.upload)}  ↓${formatTraffic(d.download)}`">
            <div class="trend-bar trend-down" :style="{ height: trendH(d.download) }"></div>
            <div class="trend-bar trend-up"   :style="{ height: trendH(d.upload) }"></div>
          </div>
          <div class="trend-day-label mono">{{ d.day.slice(5) }}</div>
        </div>
      </div>
    </section>

    <!-- 节点状态 -->
    <section class="panel">
      <div class="panel-head">
        <div class="panel-head-left">
          <h2 class="panel-title">节点状态</h2>
          <span class="panel-count mono">共 {{ nodes.length }} 个</span>
        </div>
        <div class="panel-head-right">
          <a-button size="small" @click="refreshStats" :loading="loading">
            <ReloadOutlined /> 刷新
          </a-button>
        </div>
      </div>

      <a-table
        class="nodes-table"
        :columns="columns"
        :dataSource="nodes"
        :loading="loading"
        rowKey="id"
        :pagination="{ pageSize: 10, showSizeChanger: true }"
        :scroll="{ x: 900 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <div class="cell-node">
              <span class="cell-node-name">{{ record.name }}</span>
              <span class="cell-node-ip mono">{{ record.ip }}</span>
            </div>
          </template>

          <template v-if="column.key === 'status'">
            <span :class="['pill', 'pill-' + statusCls(record.status)]">
              <span class="pill-dot"></span>
              {{ statusLabel(record.status) }}
            </span>
          </template>

          <template v-if="column.key === 'resource'">
            <div class="bars">
              <div class="bar-row">
                <span class="bar-k">CPU</span>
                <div class="bar"><div class="bar-fill" :style="{ width: (record.cpu || 0) + '%' }"></div></div>
                <span class="bar-v mono">{{ (record.cpu || 0).toFixed(1) }}%</span>
              </div>
              <div class="bar-row">
                <span class="bar-k">MEM</span>
                <div class="bar"><div class="bar-fill is-mem" :style="{ width: (record.mem || 0) + '%' }"></div></div>
                <span class="bar-v mono">{{ (record.mem || 0).toFixed(1) }}%</span>
              </div>
            </div>
          </template>

          <template v-if="column.key === 'traffic'">
            <div class="traffic">
              <span class="traffic-line">
                <ArrowUpOutlined class="up" />
                <span class="mono">{{ formatTraffic(record.uploadTotal) }}</span>
              </span>
              <span class="traffic-line">
                <ArrowDownOutlined class="down" />
                <span class="mono">{{ formatTraffic(record.downloadTotal) }}</span>
              </span>
            </div>
          </template>

          <template v-if="column.key === 'xrayVersion'">
            <span class="mono cell-ver">{{ record.xrayVersion || '—' }}</span>
          </template>

          <template v-if="column.key === 'action'">
            <div class="row-actions">
              <button class="row-btn" @click="viewNode(record)">详情</button>
              <button class="row-btn" @click="syncNodeStatus(record.id)">同步</button>
            </div>
          </template>
        </template>
      </a-table>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
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

const greet = computed(() => {
  const h = new Date().getHours()
  if (h < 6)  return '夜深了，'
  if (h < 12) return '上午好，'
  if (h < 14) return '中午好，'
  if (h < 18) return '下午好，'
  return '晚上好，'
})

const onlineRate = computed(() => {
  if (!stats.value.totalNodes) return 0
  return Math.round((stats.value.onlineNodes / stats.value.totalNodes) * 100)
})

// 7 天趋势：补齐空缺日期，按时间正序
const trendDays = computed(() => {
  const raw = stats.value.trafficTrend || []
  const map = new Map(raw.map(r => [r.Day || r.day, r]))
  const out = []
  for (let i = 6; i >= 0; i--) {
    const d = new Date()
    d.setDate(d.getDate() - i)
    const key = d.toISOString().slice(0, 10)
    const r = map.get(key)
    out.push({
      day: key,
      upload: r ? Number(r.Upload || r.upload || 0) : 0,
      download: r ? Number(r.Download || r.download || 0) : 0,
    })
  }
  return out
})
const trendMax = computed(() => {
  let m = 0
  for (const d of trendDays.value) {
    const v = (d.upload || 0) + (d.download || 0)
    if (v > m) m = v
  }
  return m
})
const trendH = (v) => {
  if (!trendMax.value) return '0%'
  return `${Math.max(2, Math.round((v / trendMax.value) * 100))}%`
}

const columns = [
  { title: '节点',      key: 'name' },
  { title: '状态',      key: 'status',       width: 110 },
  { title: '资源使用',  key: 'resource',     width: 260 },
  { title: '流量统计',  key: 'traffic',      width: 160 },
  { title: '内核版本',  key: 'xrayVersion',  width: 110 },
  { title: '操作',      key: 'action',       width: 140, fixed: 'right' }
]

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let n = bytes
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++ }
  return `${n.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

const statusLabel = (s) => ({ online: '在线', offline: '离线' }[s] || '未知')
const statusCls   = (s) => ({ online: 'ok',   offline: 'danger' }[s] || 'muted')

const fetchData = async () => {
  loading.value = true
  try {
    const [nodesRes, statsRes] = await Promise.all([getNodes(), getStatsOverview()])
    nodes.value = nodesRes.obj || []
    stats.value = statsRes.obj || stats.value
  } catch (e) {
    message.error('获取数据失败')
  } finally {
    loading.value = false
  }
}

const refreshStats = () => fetchData()
const viewNode = (node) => router.push(`/admin/nodes?id=${node.id}`)
const syncNodeStatus = async (id) => {
  try {
    await syncNode(id)
    message.success('同步成功')
    fetchData()
  } catch (e) {
    message.error('同步失败')
  }
}

onMounted(fetchData)
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* ============================================================
   Hero
   ============================================================ */
.hero {
  position: relative;
  background:
    radial-gradient(120% 140% at 100% 0%, #dbeafe 0%, transparent 55%),
    radial-gradient(80% 120% at 0% 100%, #eef4ff 0%, transparent 60%),
    #ffffff;
  border: 1px solid #e6ecf4;
  border-radius: 14px;
  padding: 18px 22px;
  animation: rise .4s ease-out both;
  overflow: hidden;
}

.hero-main {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  flex-wrap: wrap;
  gap: 18px;
}

.hero-eyebrow {
  display: inline-block;
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: #64748b;
  letter-spacing: .14em;
  text-transform: uppercase;
  margin-bottom: 6px;
}

.hero-title {
  font-family: var(--font-display);
  margin: 0 0 8px;
  font-size: 26px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
  line-height: 1.15;
}

.hero-sub {
  margin: 0;
  font-size: 13px;
  color: #475569;
}

.hero-sub strong {
  color: #1e293b;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.hero-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

/* ============================================================
   指标卡
   ============================================================ */
.metrics {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
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

.metric:nth-child(1) { animation-delay: 40ms; }
.metric:nth-child(2) { animation-delay: 80ms; }
.metric:nth-child(3) { animation-delay: 120ms; }
.metric:nth-child(4) { animation-delay: 160ms; }

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

.metric-primary .metric-icon { background: #eff6ff; color: #3b82f6; }
.icon-ok     { background: #ecfdf5 !important; color: #047857 !important; }
.icon-danger { background: #fef2f2 !important; color: #b91c1c !important; }

.metric-value {
  font-family: var(--font-display);
  font-size: 26px;
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

.link { color: #3b82f6; cursor: pointer; font-weight: 500; }
.link:hover { color: #2563eb; }

/* ============================================================
   二级资源指标 / 7 天趋势
   ============================================================ */
.sub-metrics {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 8px;
  animation: rise .4s ease-out both;
  animation-delay: 200ms;
}
.sub-card {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 10px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  transition: border-color .15s, box-shadow .15s;
}
.sub-card:hover { border-color: #c7d8f2; }
.sub-k { font-size: 11px; color: #94a3b8; letter-spacing: .02em; }
.sub-v { font-size: 18px; font-weight: 700; color: #0f172a; line-height: 1.1; font-variant-numeric: tabular-nums; }
.sub-aux { font-size: 11px; font-weight: 500; color: #94a3b8; }

@media (max-width: 1200px) {
  .sub-metrics { grid-template-columns: repeat(4, 1fr); }
}
@media (max-width: 576px) {
  .sub-metrics { grid-template-columns: repeat(2, 1fr); }
}

/* 7 天趋势 */
.trend-panel {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 12px;
  padding: 14px 16px;
  animation: rise .4s ease-out both;
  animation-delay: 240ms;
}
.trend-head {
  display: flex; justify-content: space-between; align-items: baseline;
  margin-bottom: 12px;
}
.trend-title {
  font-family: var(--font-display);
  font-size: 13px; font-weight: 600; color: #0f172a;
}
.trend-sub { font-size: 11px; color: #94a3b8; }
.trend-bars {
  display: flex; align-items: flex-end; justify-content: space-between;
  gap: 8px; height: 80px;
}
.trend-day { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 6px; }
.trend-stack {
  width: 100%; max-width: 36px;
  height: 60px; display: flex; flex-direction: column-reverse; gap: 1px;
  cursor: default;
}
.trend-bar { width: 100%; border-radius: 2px 2px 0 0; transition: height .3s ease; }
.trend-up   { background: linear-gradient(180deg, #60a5fa, #3b82f6); }
.trend-down { background: linear-gradient(180deg, #93c5fd, #60a5fa); opacity: .6; }
.trend-day-label { font-size: 10.5px; color: #94a3b8; }

/* ============================================================
   面板 / 表格
   ============================================================ */
.panel {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 12px;
  padding: 14px 16px;
  animation: rise .4s ease-out both;
  animation-delay: 200ms;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding-bottom: 14px;
  border-bottom: 1px solid #f1f5f9;
  margin-bottom: 12px;
}

.panel-head-left { display: flex; align-items: baseline; gap: 12px; flex-wrap: wrap; }
.panel-head-right { display: flex; align-items: center; gap: 8px; }

.panel-title {
  font-family: var(--font-display);
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: -0.01em;
}

.panel-count {
  font-size: 12px;
  color: #64748b;
  font-variant-numeric: tabular-nums;
}

/* 表格细节 */
.nodes-table :deep(.ant-table-thead > tr > th) {
  background: #f8fafc !important;
  color: #64748b !important;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: .02em;
  border-bottom: 1px solid #eef1f6 !important;
  padding: 9px 13px !important;
}

.nodes-table :deep(.ant-table-tbody > tr > td) {
  padding: 10px 13px !important;
  border-bottom: 1px solid #f1f5f9 !important;
  font-size: 13px;
}

.nodes-table :deep(.ant-table-tbody > tr:hover > td) {
  background: #f8fbff !important;
}

.cell-node {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cell-node-name {
  font-weight: 600;
  color: #0f172a;
}

.cell-node-ip {
  font-size: 12px;
  color: #94a3b8;
  font-variant-numeric: tabular-nums;
}

.cell-ver {
  font-size: 12px;
  color: #475569;
}

/* 状态 */
.pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 10px;
  border-radius: 99px;
  font-size: 11.5px;
  font-weight: 500;
}

.pill-dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }

.pill-ok     { background: #ecfdf5; color: #047857; }
.pill-danger { background: #fef2f2; color: #b91c1c; }
.pill-muted  { background: #f1f5f9; color: #64748b; }

/* 资源条 */
.bars {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.bar-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.bar-k {
  font-family: var(--font-mono);
  font-size: 10.5px;
  font-weight: 600;
  color: #64748b;
  width: 30px;
  letter-spacing: .04em;
}

.bar {
  flex: 1;
  height: 5px;
  background: #e2e8f0;
  border-radius: 3px;
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #60a5fa);
  border-radius: 3px;
  transition: width .3s ease;
}

.bar-fill.is-mem {
  background: linear-gradient(90deg, #1e40af, #3b82f6);
}

.bar-v {
  font-size: 11.5px;
  color: #475569;
  width: 44px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

/* 流量 */
.traffic {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12.5px;
}

.traffic-line {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #475569;
}

.traffic-line .up   { color: #16a34a; }
.traffic-line .down { color: #2563eb; }

/* 操作 */
.row-actions {
  display: flex;
  gap: 4px;
}

.row-btn {
  padding: 4px 10px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px;
  color: #2563eb;
  font-size: 12.5px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color .15s, border-color .15s;
}

.row-btn:hover {
  background: #eff6ff;
  border-color: #dbeafe;
}

/* ============================================================
   响应式
   ============================================================ */
@media (max-width: 1200px) {
  .metrics { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 768px) {
  .hero { padding: 22px 20px; }
  .hero-title { font-size: 22px; }
  .hero-main { flex-direction: column; align-items: flex-start; }
  .hero-actions { width: 100%; }
  .hero-actions .ant-btn { flex: 1; }
}

@media (max-width: 576px) {
  .metrics { grid-template-columns: 1fr; }
  .metric { padding: 16px; }
}

@keyframes rise {
  from { opacity: 0; transform: translateY(6px); }
  to   { opacity: 1; transform: none; }
}
</style>
