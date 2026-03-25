<template>
  <div class="orders-page">
    <a-card title="订单管理">
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
          <template v-if="column.key === 'action'">
            <a-space v-if="record.status === 'pending'">
              <a-button type="link" size="small" @click="markPaid(record)">确认支付</a-button>
              <a-button type="link" size="small" danger @click="cancelOrder(record)">取消</a-button>
            </a-space>
            <span v-else style="color: #999">-</span>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getAllOrders, updateOrderStatus } from '@/api'

const loading = ref(false)
const orders = ref([])

const columns = [
  { title: '订单号', dataIndex: 'orderNo', key: 'orderNo' },
  { title: '用户', dataIndex: 'userId', key: 'userId' },
  { title: '套餐', dataIndex: 'packageName', key: 'packageName' },
  { title: '金额', key: 'amount' },
  { title: '支付方式', dataIndex: 'payMethod', key: 'payMethod' },
  { title: '状态', key: 'status' },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
  { title: '操作', key: 'action', width: 150 }
]

const getStatusColor = (status) => {
  const colors = { pending: 'orange', paid: 'green', cancelled: 'default', refunded: 'red' }
  return colors[status] || 'default'
}

const getStatusText = (status) => {
  const texts = { pending: '待支付', paid: '已支付', cancelled: '已取消', refunded: '已退款' }
  return texts[status] || status
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