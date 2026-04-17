<template>
  <div class="my-orders-page">
    <section class="panel">
      <div class="panel-head">
        <div class="panel-head-left">
          <h2 class="panel-title">订单记录</h2>
          <span class="panel-count mono">共 {{ orders.length }} 条</span>
        </div>
        <div class="panel-head-right">
          <div class="seg">
            <button
              v-for="opt in filterOptions"
              :key="opt.key"
              class="seg-btn"
              :class="{ 'is-active': filter === opt.key }"
              @click="filter = opt.key"
            >{{ opt.label }}</button>
          </div>
          <a-button type="primary" @click="$router.push('/user/buy')">
            <ShoppingCartOutlined /> 购买套餐
          </a-button>
        </div>
      </div>

      <!-- 桌面：表格 -->
      <div class="orders-table hide-mobile">
        <a-table
          :columns="columns"
          :dataSource="filteredOrders"
          :loading="loading"
          rowKey="id"
          :pagination="{ pageSize: 10, showSizeChanger: false }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'orderNo'">
              <span class="cell-order-no mono">{{ record.orderNo }}</span>
            </template>
            <template v-if="column.key === 'package'">
              <div class="cell-package">
                <span class="cell-package-name">{{ record.packageName || '—' }}</span>
                <span v-if="record.protocol" class="cell-package-proto mono">
                  {{ (record.protocol || '').toUpperCase() }}
                </span>
              </div>
            </template>
            <template v-if="column.key === 'amount'">
              <span class="cell-amount mono">¥{{ Number(record.amount || 0).toFixed(2) }}</span>
            </template>
            <template v-if="column.key === 'status'">
              <span :class="['pill', 'pill-' + statusClass(record.status)]">
                <span class="pill-dot"></span>
                {{ statusLabel(record.status) }}
              </span>
            </template>
            <template v-if="column.key === 'createdAt'">
              <span class="cell-time mono">{{ formatDateTime(record.createdAt) }}</span>
            </template>
          </template>
        </a-table>
      </div>

      <!-- 移动：卡片 -->
      <div class="orders-cards hide-desktop">
        <div v-if="loading" class="state">加载中…</div>
        <div v-else-if="filteredOrders.length === 0" class="state state-empty">
          <ShoppingOutlined class="state-icon" />
          <p class="state-title">暂无订单记录</p>
          <a-button type="primary" @click="$router.push('/user/buy')">
            去购买套餐
          </a-button>
        </div>
        <article
          v-else
          v-for="o in filteredOrders"
          :key="o.id"
          class="order-card"
        >
          <div class="order-card-head">
            <span class="cell-package-name">{{ o.packageName || '—' }}</span>
            <span :class="['pill', 'pill-' + statusClass(o.status)]">
              <span class="pill-dot"></span>
              {{ statusLabel(o.status) }}
            </span>
          </div>
          <div class="order-card-row">
            <span class="k">订单号</span>
            <span class="v mono">{{ o.orderNo }}</span>
          </div>
          <div class="order-card-row">
            <span class="k">金额</span>
            <span class="v cell-amount mono">¥{{ Number(o.amount || 0).toFixed(2) }}</span>
          </div>
          <div class="order-card-row">
            <span class="k">时间</span>
            <span class="v mono">{{ formatDateTime(o.createdAt) }}</span>
          </div>
        </article>
      </div>

      <!-- 桌面空态（表格内部空态兜底） -->
      <div v-if="!loading && orders.length === 0" class="state state-empty hide-mobile">
        <ShoppingOutlined class="state-icon" />
        <p class="state-title">暂无订单记录</p>
        <a-button type="primary" @click="$router.push('/user/buy')">
          去购买套餐
        </a-button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { ShoppingOutlined, ShoppingCartOutlined } from '@ant-design/icons-vue'
import { getMyOrders } from '@/api'

const loading = ref(false)
const orders = ref([])
const filter = ref('all')

const filterOptions = [
  { key: 'all',       label: '全部' },
  { key: 'paid',      label: '已支付' },
  { key: 'pending',   label: '待支付' },
  { key: 'cancelled', label: '已取消' }
]

const columns = [
  { title: '订单号',   key: 'orderNo', width: 180 },
  { title: '套餐',     key: 'package' },
  { title: '金额',     key: 'amount',  width: 110, align: 'right' },
  { title: '状态',     key: 'status',  width: 110 },
  { title: '创建时间', key: 'createdAt', width: 170 }
]

const filteredOrders = computed(() => {
  if (filter.value === 'all') return orders.value
  return orders.value.filter(o => o.status === filter.value)
})

const statusLabel = (s) => ({
  pending:   '待支付',
  paid:      '已支付',
  cancelled: '已取消',
  refunded:  '已退款'
}[s] || s || '—')

const statusClass = (s) => ({
  pending:   'warn',
  paid:      'ok',
  cancelled: 'muted',
  refunded:  'danger'
}[s] || 'muted')

const formatDateTime = (v) => {
  if (!v) return '—'
  const d = new Date(v)
  if (isNaN(d)) return '—'
  const pad = (x) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const fetchOrders = async () => {
  loading.value = true
  try {
    const res = await getMyOrders()
    orders.value = res.obj || []
  } catch (e) {
    message.error('获取订单失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchOrders)
</script>

<style scoped>
.my-orders-page { animation: rise .4s ease-out both; }

.panel {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 12px;
  padding: 14px 16px;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding-bottom: 14px;
  border-bottom: 1px solid #f1f5f9;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.panel-head-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.panel-head-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

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
  transition: color .15s, background-color .15s;
}

.seg-btn:hover { color: #1e293b; }

.seg-btn.is-active {
  background: #fff;
  color: #2563eb;
  box-shadow: 0 1px 2px rgba(15,23,42,.08);
}

/* 单元格 */
.cell-order-no {
  font-size: 12.5px;
  color: #475569;
  font-variant-numeric: tabular-nums;
}

.cell-package {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cell-package-name {
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
}

.cell-package-proto {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 4px;
  background: #eff6ff;
  color: #2563eb;
  font-size: 10.5px;
  font-weight: 600;
}

.cell-amount {
  font-size: 13px;
  font-weight: 600;
  color: #0f172a;
  font-variant-numeric: tabular-nums;
}

.cell-time {
  font-size: 12px;
  color: #64748b;
  font-variant-numeric: tabular-nums;
}

/* 状态胶囊 */
.pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 10px;
  border-radius: 99px;
  font-size: 11.5px;
  font-weight: 500;
}

.pill-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}

.pill-ok    { background: #ecfdf5; color: #047857; }
.pill-warn  { background: #fffbeb; color: #b45309; }
.pill-muted { background: #f1f5f9; color: #64748b; }
.pill-danger{ background: #fef2f2; color: #b91c1c; }

/* 表格样式覆写 */
.orders-table :deep(.ant-table-thead > tr > th) {
  background: #f8fafc !important;
  color: #64748b !important;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: .02em;
  border-bottom: 1px solid #eef1f6 !important;
  padding: 9px 13px !important;
}

.orders-table :deep(.ant-table-tbody > tr > td) {
  padding: 10px 13px !important;
  border-bottom: 1px solid #f1f5f9 !important;
  font-size: 13px;
}

.orders-table :deep(.ant-table-tbody > tr:hover > td) {
  background: #f8fbff !important;
}

/* 移动卡片 */
.orders-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.order-card {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 12px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.order-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.order-card-row {
  display: flex;
  justify-content: space-between;
  font-size: 12.5px;
}

.order-card-row .k { color: #94a3b8; }
.order-card-row .v { color: #1e293b; font-weight: 500; }

/* 状态区 */
.state {
  padding: 56px 20px;
  text-align: center;
  color: #94a3b8;
  font-size: 13px;
}

.state-icon { font-size: 42px; color: #cbd5e1; margin-bottom: 12px; }
.state-title {
  margin: 4px 0 16px;
  font-size: 14px;
  font-weight: 600;
  color: #475569;
}

/* 响应式 */
.hide-mobile { display: block; }
.hide-desktop { display: none; }

@media (max-width: 768px) {
  .hide-mobile { display: none !important; }
  .hide-desktop { display: block !important; }
  .panel-head { flex-direction: column; align-items: flex-start; }
  .panel-head-right { width: 100%; justify-content: space-between; }
}

@keyframes rise {
  from { opacity: 0; transform: translateY(6px); }
  to   { opacity: 1; transform: none; }
}
</style>
