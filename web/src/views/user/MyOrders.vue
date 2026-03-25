<template>
  <div class="my-orders">
    <a-card title="我的订单">
      <a-table :columns="columns" :dataSource="orders" :loading="loading" rowKey="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-if="column.key === 'amount'">
            ¥{{ record.amount }}
          </template>
          <template v-if="column.key === 'createdAt'">
            {{ formatDate(record.createdAt) }}
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getMyOrders } from '@/api'

const loading = ref(false)
const orders = ref([])

const columns = [
  { title: '订单号', dataIndex: 'orderNo', key: 'orderNo' },
  { title: '套餐', dataIndex: 'packageName', key: 'packageName' },
  { title: '金额', key: 'amount' },
  { title: '状态', key: 'status' },
  { title: '创建时间', key: 'createdAt' }
]

const getStatusColor = (status) => {
  const colors = {
    pending: 'orange',
    paid: 'green',
    cancelled: 'default',
    refunded: 'red'
  }
  return colors[status] || 'default'
}

const getStatusText = (status) => {
  const texts = {
    pending: '待支付',
    paid: '已支付',
    cancelled: '已取消',
    refunded: '已退款'
  }
  return texts[status] || status
}

const formatDate = (date) => {
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
.my-orders {
  max-width: 1000px;
  margin: 0 auto;
}
</style>