<template>
  <div class="orders-page">
    <!-- 订单列表卡片 -->
    <a-card class="orders-card">
      <a-table 
        :columns="columns" 
        :dataSource="orders" 
        :loading="loading" 
        rowKey="id"
        :pagination="{ pageSize: 10, showSizeChanger: true }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'orderNo'">
            <span class="order-no">{{ record.orderNo }}</span>
          </template>
          <template v-if="column.key === 'user'">
            <div class="user-cell">
              <a-avatar :size="28" class="user-avatar">
                {{ record.username?.charAt(0)?.toUpperCase() }}
              </a-avatar>
              <span>{{ record.username || `用户${record.userId}` }}</span>
            </div>
          </template>
          <template v-if="column.key === 'package'">
            <div class="package-cell">
              <span class="package-name">{{ record.packageName }}</span>
              <span class="package-duration">{{ record.duration ? record.duration + '天' : '永久' }}</span>
            </div>
          </template>
          <template v-if="column.key === 'amount'">
            <span class="amount">${{ record.amount }}</span>
          </template>
          <template v-if="column.key === 'status'">
            <span :class="['status-badge', record.status]">
              {{ getStatusText(record.status) }}
            </span>
          </template>
          <template v-if="column.key === 'createdAt'">
            <span class="time-text">{{ formatDateTime(record.createdAt) }}</span>
          </template>
          <template v-if="column.key === 'action'">
            <div v-if="record.status === 'pending'" class="action-btns">
              <a-button type="primary" size="small" @click="markPaid(record)">
                确认支付
              </a-button>
              <a-button size="small" @click="cancelOrder(record)">
                取消
              </a-button>
            </div>
            <span v-else class="text-muted">-</span>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { ShoppingOutlined } from '@ant-design/icons-vue'
import { getAllOrders, updateOrderStatus } from '@/api'

const loading = ref(false)
const orders = ref([])

const columns = [
  { title: '订单号', key: 'orderNo', width: 180 },
  { title: '用户', key: 'user' },
  { title: '套餐', key: 'package' },
  { title: '金额', key: 'amount', width: 100 },
  { title: '支付方式', dataIndex: 'payMethod', key: 'payMethod', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', key: 'createdAt', width: 160 },
  { title: '操作', key: 'action', width: 150, fixed: 'right' }
]

const getStatusText = (status) => {
  const texts = { pending: '待支付', paid: '已支付', cancelled: '已取消', refunded: '已退款' }
  return texts[status] || status
}

const formatDateTime = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

const fetchOrders = async () => {
  loading.value = true
  try {
    const res = await getAllOrders()
    orders.value = res.obj || []
  } catch (e) {
    message.error('获取订单失败')
  } finally {
    loading.value = false
  }
}

const markPaid = async (order) => {
  try {
    await updateOrderStatus(order.id, 'paid')
    message.success('已确认支付')
    fetchOrders()
  } catch (e) {
    message.error('操作失败')
  }
}

const cancelOrder = async (order) => {
  try {
    await updateOrderStatus(order.id, 'cancelled')
    message.success('订单已取消')
    fetchOrders()
  } catch (e) {
    message.error('操作失败')
  }
}

onMounted(() => {
  fetchOrders()
})
</script>

<style scoped>
.orders-page {
  animation: fadeIn 0.3s ease;
}

.orders-card {
  border-radius: 14px;
}

.order-no {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 13px;
  color: #475569;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
}

.package-cell {
  display: flex;
  flex-direction: column;
}

.package-name {
  font-weight: 500;
  color: #1e293b;
}

.package-duration {
  font-size: 12px;
  color: #64748b;
}

.amount {
  font-weight: 600;
  color: #dc2626;
}

.time-text {
  font-size: 13px;
  color: #64748b;
}

/* 状态徽章 */
.status-badge {
  display: inline-flex;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.status-badge.pending {
  background: #fffbeb;
  color: #b45309;
}

.status-badge.paid {
  background: #f0fdf4;
  color: #16a34a;
}

.status-badge.cancelled {
  background: #f1f5f9;
  color: #64748b;
}

.status-badge.refunded {
  background: #fef2f2;
  color: #dc2626;
}

.text-muted {
  color: #94a3b8;
}

.action-btns {
  display: flex;
  gap: 8px;
}

@media (max-width: 768px) {
  .action-btns { gap: 4px; }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>