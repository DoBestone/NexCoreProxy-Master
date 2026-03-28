<template>
  <div class="orders-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">
          <ShoppingOutlined class="title-icon" />
          订单管理
        </h1>
        <p class="page-desc">查看和管理所有订单</p>
      </div>
    </div>
    
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
              <span class="package-duration">{{ record.duration }}天</span>
            </div>
          </template>
          <template v-if="column.key === 'amount'">
            <span class="amount">¥{{ record.amount }}</span>
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

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
  gap: 16px;
}

.page-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 22px;
  font-weight: 700;
  color: #262626;
  margin: 0;
}

.title-icon {
  color: #1677ff;
  font-size: 24px;
}

.page-desc {
  color: #8c8c8c;
  font-size: 14px;
  margin-top: 4px;
}

.orders-card {
  border-radius: 14px;
}

.order-no {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 13px;
  color: #595959;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  background: linear-gradient(135deg, #1677ff 0%, #4096ff 100%);
}

.package-cell {
  display: flex;
  flex-direction: column;
}

.package-name {
  font-weight: 500;
  color: #262626;
}

.package-duration {
  font-size: 12px;
  color: #8c8c8c;
}

.amount {
  font-weight: 600;
  color: #ff4d4f;
}

.time-text {
  font-size: 13px;
  color: #8c8c8c;
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
  background: #fff7e6;
  color: #d46b08;
}

.status-badge.paid {
  background: #f6ffed;
  color: #52c41a;
}

.status-badge.cancelled {
  background: #f5f5f5;
  color: #8c8c8c;
}

.status-badge.refunded {
  background: #fff2f0;
  color: #ff4d4f;
}

.text-muted {
  color: #bfbfbf;
}

.action-btns {
  display: flex;
  gap: 8px;
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>