<template>
  <div class="dashboard-page">
    <!-- 欢迎条 -->
    <section class="hero">
      <div class="hero-main">
        <div class="hero-greet">
          <span class="hero-eyebrow">{{ greet }}</span>
          <h1 class="hero-title">{{ userInfo.username || '用户' }}</h1>
          <p class="hero-sub">
            当前订阅 <strong>{{ planName }}</strong>
            ·
            <span :class="['hero-tag', expireState.cls]">{{ expireState.label }}</span>
          </p>
        </div>
        <div class="hero-cta">
          <a-button type="primary" size="large" @click="$router.push('/user/buy')">
            <ShoppingCartOutlined /> 续费 / 升级
          </a-button>
          <a-button size="large" @click="$router.push('/user/nodes')">
            <LinkOutlined /> 订阅链接
          </a-button>
        </div>
      </div>

      <!-- 公告 -->
      <div v-if="latestAnnouncement" class="hero-announce">
        <NotificationOutlined />
        <span class="announce-label">公告</span>
        <span class="announce-text">{{ latestAnnouncement.title }}</span>
      </div>
    </section>

    <!-- 核心指标 -->
    <section class="metrics">
      <div class="metric metric-primary">
        <div class="metric-head">
          <span class="metric-label">本月流量</span>
          <span class="metric-icon"><LineChartOutlined /></span>
        </div>
        <div class="metric-value">
          <span class="mono">{{ formatTraffic(traffic.used) }}</span>
          <span class="metric-of">/ {{ traffic.limit ? formatTraffic(traffic.limit) : '无限制' }}</span>
        </div>
        <div class="metric-bar">
          <div class="metric-bar-fill" :style="{ width: trafficPercent + '%' }"></div>
        </div>
        <div class="metric-foot">
          <span :class="['dot', trafficPercent > 80 ? 'dot-warn' : 'dot-ok']"></span>
          <span>已用 {{ trafficPercent }}%</span>
        </div>
      </div>

      <div class="metric">
        <div class="metric-head">
          <span class="metric-label">订阅到期</span>
          <span class="metric-icon"><ClockCircleOutlined /></span>
        </div>
        <div class="metric-value">
          <span class="mono">{{ daysRemaining === null ? '∞' : daysRemaining }}</span>
          <span class="metric-of">{{ daysRemaining === null ? '永久' : '天' }}</span>
        </div>
        <div class="metric-foot muted">
          {{ userInfo.expireAt ? formatDate(userInfo.expireAt) : '无到期限制' }}
        </div>
      </div>

      <div class="metric">
        <div class="metric-head">
          <span class="metric-label">可用节点</span>
          <span class="metric-icon"><CloudServerOutlined /></span>
        </div>
        <div class="metric-value">
          <span class="mono">{{ nodeOnline }}</span>
          <span class="metric-of">/ {{ nodeTotal }} 在线</span>
        </div>
        <div class="metric-foot muted">
          <a class="link" @click="$router.push('/user/nodes')">查看节点 →</a>
        </div>
      </div>

      <div class="metric">
        <div class="metric-head">
          <span class="metric-label">账户余额</span>
          <span class="metric-icon"><WalletOutlined /></span>
        </div>
        <div class="metric-value">
          <span class="metric-currency">¥</span>
          <span class="mono">{{ (userInfo.balance || 0).toFixed(2) }}</span>
        </div>
        <div class="metric-foot muted">
          <a class="link" @click="$router.push('/user/orders')">订单记录 →</a>
        </div>
      </div>
    </section>

    <!-- 底部双栏 -->
    <section class="grid-2">
      <!-- 快捷操作 -->
      <div class="panel">
        <div class="panel-head">
          <h3 class="panel-title">快捷操作</h3>
        </div>
        <div class="actions">
          <a class="action" @click="$router.push('/user/nodes')">
            <span class="action-icon"><LinkOutlined /></span>
            <span class="action-body">
              <span class="action-title">获取订阅</span>
              <span class="action-sub">复制链接或刷新</span>
            </span>
          </a>
          <a class="action" @click="$router.push('/user/traffic')">
            <span class="action-icon"><LineChartOutlined /></span>
            <span class="action-body">
              <span class="action-title">查看流量</span>
              <span class="action-sub">30 天用量趋势</span>
            </span>
          </a>
          <a class="action" @click="$router.push('/user/buy')">
            <span class="action-icon"><ShoppingCartOutlined /></span>
            <span class="action-body">
              <span class="action-title">购买套餐</span>
              <span class="action-sub">选择流量方案</span>
            </span>
          </a>
          <a class="action" @click="$router.push('/user/orders')">
            <span class="action-icon"><ShoppingOutlined /></span>
            <span class="action-body">
              <span class="action-title">订单记录</span>
              <span class="action-sub">查询历史订单</span>
            </span>
          </a>
        </div>
      </div>

      <!-- 最近订单 -->
      <div class="panel">
        <div class="panel-head">
          <h3 class="panel-title">最近订单</h3>
          <a class="panel-link" @click="$router.push('/user/orders')">全部 →</a>
        </div>
        <div v-if="recentOrders.length === 0" class="panel-empty">
          暂无订单
        </div>
        <ul v-else class="order-list">
          <li v-for="o in recentOrders" :key="o.id" class="order-item">
            <div class="order-main">
              <span class="order-title">{{ o.packageName || '套餐' }}</span>
              <span class="order-time">{{ formatDate(o.createdAt || o.createTime) }}</span>
            </div>
            <div class="order-side">
              <span class="order-amount mono">¥{{ Number(o.amount || 0).toFixed(2) }}</span>
              <span :class="['order-status', statusClass(o.status)]">
                <span class="dot"></span>{{ statusLabel(o.status) }}
              </span>
            </div>
          </li>
        </ul>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import {
  ShoppingCartOutlined,
  LinkOutlined,
  LineChartOutlined,
  ClockCircleOutlined,
  CloudServerOutlined,
  WalletOutlined,
  ShoppingOutlined,
  NotificationOutlined
} from '@ant-design/icons-vue'
import {
  getUserInfo,
  getMyTraffic,
  getMyNodes,
  getMyOrders,
  getAnnouncements
} from '@/api'

const userInfo = ref({})
const traffic = ref({ used: 0, limit: 0 })
const nodes = ref([])
const orders = ref([])
const announcements = ref([])

const greet = computed(() => {
  const h = new Date().getHours()
  if (h < 6)  return '夜深了，'
  if (h < 12) return '上午好，'
  if (h < 14) return '中午好，'
  if (h < 18) return '下午好，'
  return '晚上好，'
})

const planName = computed(() => userInfo.value.packageName || '基础套餐')

const daysRemaining = computed(() => {
  if (!userInfo.value.expireAt) return null
  const diff = new Date(userInfo.value.expireAt) - new Date()
  return Math.max(0, Math.ceil(diff / 86400000))
})

const expireState = computed(() => {
  const d = daysRemaining.value
  if (d === null) return { label: '永久有效', cls: 'ok' }
  if (d <= 0)     return { label: '已过期', cls: 'danger' }
  if (d <= 7)     return { label: `剩余 ${d} 天`, cls: 'warn' }
  return { label: `剩余 ${d} 天`, cls: 'ok' }
})

const trafficPercent = computed(() => {
  if (!traffic.value.limit) return 0
  return Math.min(100, Math.round((traffic.value.used / traffic.value.limit) * 100))
})

const nodeOnline = computed(() => nodes.value.filter(n => n.status === 'online').length)
const nodeTotal = computed(() => nodes.value.length)

const recentOrders = computed(() => orders.value.slice(0, 5))

const latestAnnouncement = computed(() => announcements.value[0] || null)

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let n = bytes
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++ }
  return `${n.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

const formatDate = (v) => {
  if (!v) return '—'
  const d = new Date(v)
  if (isNaN(d)) return '—'
  const pad = (x) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

const statusLabel = (s) => ({
  paid: '已支付',
  pending: '待支付',
  cancelled: '已取消',
  refunded: '已退款'
}[s] || s || '—')

const statusClass = (s) => ({
  paid: 'ok',
  pending: 'warn',
  cancelled: 'muted',
  refunded: 'muted'
}[s] || 'muted')

async function loadAll() {
  try {
    const [u, t, n, o, a] = await Promise.allSettled([
      getUserInfo(),
      getMyTraffic(),
      getMyNodes(),
      getMyOrders(),
      getAnnouncements()
    ])
    if (u.status === 'fulfilled') userInfo.value = u.value?.obj || {}
    if (t.status === 'fulfilled') {
      traffic.value = {
        used:  t.value?.obj?.used  || 0,
        limit: t.value?.obj?.limit || 0
      }
    }
    if (n.status === 'fulfilled') nodes.value = n.value?.obj || []
    if (o.status === 'fulfilled') orders.value = o.value?.obj || []
    if (a.status === 'fulfilled') announcements.value = a.value?.obj || []
  } catch (e) {
    // 静默失败：各卡片自带占位
  }
}

onMounted(loadAll)
</script>

<style scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* ============================================================
   Hero
   ============================================================ */
.hero {
  position: relative;
  border-radius: 14px;
  overflow: hidden;
  background:
    radial-gradient(120% 140% at 100% 0%, #dbeafe 0%, transparent 55%),
    radial-gradient(80% 120% at 0% 100%, #eef4ff 0%, transparent 60%),
    #ffffff;
  border: 1px solid #e6ecf4;
  padding: 18px 22px;
  animation: rise .4s ease-out both;
}

.hero-main {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  justify-content: space-between;
  align-items: flex-end;
}

.hero-eyebrow {
  font-size: 12px;
  color: #64748b;
  letter-spacing: .02em;
}

.hero-title {
  font-family: var(--font-display);
  font-size: 26px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
  margin: 4px 0 6px;
  line-height: 1.15;
}

.hero-sub {
  font-size: 13px;
  color: #475569;
  margin: 0;
  line-height: 1.5;
}

.hero-sub strong {
  color: #1e293b;
  font-weight: 600;
}

.hero-tag {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 99px;
  font-size: 11.5px;
  font-weight: 500;
  letter-spacing: .01em;
  margin-left: 2px;
}

.hero-tag.ok     { background: #ecfdf5; color: #047857; }
.hero-tag.warn   { background: #fffbeb; color: #b45309; }
.hero-tag.danger { background: #fef2f2; color: #b91c1c; }

.hero-cta {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.hero-announce {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 20px;
  padding: 10px 14px;
  border-radius: 10px;
  background: rgba(255,255,255,.6);
  border: 1px solid #e2e8f0;
  font-size: 13px;
  color: #475569;
  cursor: pointer;
  transition: background-color .15s, border-color .15s;
}

.hero-announce:hover { background: #fff; border-color: #dbeafe; }
.hero-announce .anticon { color: #3b82f6; }
.announce-label {
  font-size: 11px;
  color: #3b82f6;
  font-weight: 600;
  padding: 1px 7px;
  border-radius: 4px;
  background: #eff6ff;
}
.announce-text {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #1e293b;
}
.announce-caret { color: #94a3b8; font-size: 11px; }

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

.metric-primary .metric-icon {
  background: #eff6ff;
  color: #3b82f6;
}

.metric-value {
  display: flex;
  align-items: baseline;
  gap: 6px;
  font-family: var(--font-display);
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.metric-value .mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.metric-of {
  font-family: var(--font-body);
  font-size: 12px;
  font-weight: 500;
  color: #94a3b8;
  letter-spacing: 0;
}
.metric-currency { font-size: 15px; color: #64748b; font-weight: 600; }

.metric-bar {
  margin-top: 14px;
  height: 6px;
  background: #e2e8f0;
  border-radius: 3px;
  overflow: hidden;
}

.metric-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #60a5fa);
  border-radius: 3px;
  transition: width .4s ease;
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

.metric-foot .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  display: inline-block;
}
.metric-foot .dot-ok { background: #16a34a; }
.metric-foot .dot-warn { background: #d97706; }

.link {
  color: #3b82f6;
  cursor: pointer;
  font-weight: 500;
}
.link:hover { color: #2563eb; }

/* ============================================================
   双栏
   ============================================================ */
.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.panel {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 12px;
  padding: 14px 16px;
}

.panel-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 14px;
}

.panel-title {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
  letter-spacing: -0.01em;
}

.panel-link {
  font-size: 12px;
  color: #3b82f6;
  cursor: pointer;
}
.panel-link:hover { color: #2563eb; }

.panel-empty {
  padding: 32px 0;
  text-align: center;
  color: #94a3b8;
  font-size: 13px;
}

/* 快捷操作网格 */
.actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.action {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border-radius: 10px;
  background: #f8fafc;
  border: 1px solid transparent;
  cursor: pointer;
  transition: background-color .15s, border-color .15s;
}

.action:hover {
  background: #fff;
  border-color: #dbeafe;
}

.action-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: #eff6ff;
  color: #3b82f6;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
}

.action-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.action-title {
  font-size: 13.5px;
  font-weight: 600;
  color: #1e293b;
}

.action-sub {
  font-size: 11.5px;
  color: #94a3b8;
}

/* 订单列表 */
.order-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
}

.order-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px dashed #eef1f6;
}

.order-item:last-child { border-bottom: none; }

.order-main {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.order-title {
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
}

.order-time {
  font-size: 11.5px;
  color: #94a3b8;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.order-side {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-shrink: 0;
}

.order-amount {
  font-size: 13px;
  font-weight: 600;
  color: #0f172a;
  font-variant-numeric: tabular-nums;
}

.order-status {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11.5px;
  padding: 2px 9px;
  border-radius: 99px;
  background: #f1f5f9;
  color: #64748b;
}

.order-status .dot {
  width: 5px; height: 5px; border-radius: 50%;
  background: currentColor;
}

.order-status.ok   { background: #ecfdf5; color: #047857; }
.order-status.warn { background: #fffbeb; color: #b45309; }
.order-status.muted { background: #f1f5f9; color: #64748b; }

/* ============================================================
   响应式
   ============================================================ */
@media (max-width: 1200px) {
  .metrics { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 768px) {
  .hero { padding: 22px 20px; }
  .hero-title { font-size: 22px; }
  .hero-main { flex-direction: column; align-items: stretch; }
  .hero-cta .ant-btn { flex: 1; }
  .grid-2 { grid-template-columns: 1fr; }
  .actions { grid-template-columns: 1fr; }
}

@media (max-width: 576px) {
  .metrics { grid-template-columns: 1fr; }
  .metric { padding: 16px; }
  .metric-value { font-size: 22px; }
}

@keyframes rise {
  from { opacity: 0; transform: translateY(6px); }
  to   { opacity: 1; transform: none; }
}
</style>
