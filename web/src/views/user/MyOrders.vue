<template>
  <div class="my-orders-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1>我的订单</h1>
      <p>查看您的购买记录</p>
    </div>
    
    <!-- 订单列表卡片 -->
    <a-card class="orders-card">
      <a-table 
        :columns="columns" 
        :dataSource="orders" 
        :loading="loading" 
        rowKey="id"
        :pagination="{ pageSize: 10 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'orderNo'">
            <span class="order-no">{{ record.orderNo }}</span>
          </template>
          <template v-if="column.key === 'package'">
            <span class="package-name">{{ record.packageName }}</span>
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
        </template>
      </a-table>
      
      <!-- 空状态 -->
      <div v-if="!loading && orders.length === 0" class="empty-state">
        <ShoppingOutlined class="empty-icon" />
        <p>暂无订单记录</p>
        <a-button type="primary" @click="$router.push('/user/buy')">
          去购买套餐
        </a-button>
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { ShoppingOutlined } from '@ant-design/icons-vue'
import { getMyOrders } from '@/api'

const loading = ref(false)
const orders = ref([])

const columns = [
  { title: '订单号', key: 'orderNo' },
  { title: '套餐', key: 'package' },
  { title: '金额', key: 'amount', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', key: 'createdAt', width: 180 }
]

const getStatusText = (status) => {
  const texts = {
    pending: '待支付',
    paid: '已支付',
    cancelled: '已取消',
    refunded: '已退款'
  }
  return texts[status] || status
}

const formatDateTime = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
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

onMounted(() => {
  fetchOrders()
})
</script>

<style scoped>
.my-orders-page {
  animation: fadeIn 0.3s ease;
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

.orders-card {
  border-radius: 14px;
}

.order-no {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 13px;
  color: #595959;
}

.package-name {
  font-weight: 500;
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

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 48px 24px;
}

.empty-icon {
  font-size: 48px;
  color: #d9d9d9;
  margin-bottom: 16px;
}

.empty-state p {
  color: #8c8c8c;
  margin-bottom: 20px;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>