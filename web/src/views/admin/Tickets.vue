<template>
  <div class="tickets-page">
    <a-card title="工单管理">
      <a-table :columns="columns" :dataSource="tickets" :loading="loading" rowKey="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="record.status === 'open' ? 'blue' : 'default'">
              {{ record.status === 'open' ? '处理中' : '已关闭' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'priority'">
            <a-tag :color="record.priority > 0 ? 'red' : 'default'">
              {{ record.priority > 0 ? '紧急' : '普通' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="viewTicket(record)">查看</a-button>
              <a-button type="link" size="small" @click="replyTicket(record)">回复</a-button>
              <a-button type="link" size="small" v-if="record.status === 'open'" @click="closeTicketRecord(record)">关闭</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 工单详情弹窗 -->
    <a-modal v-model:open="detailVisible" :title="currentTicket?.subject" width="700px" :footer="null">
      <a-descriptions :column="1" bordered size="small">
        <a-descriptions-item label="用户ID">{{ currentTicket?.userId }}</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-tag :color="currentTicket?.status === 'open' ? 'blue' : 'default'">
            {{ currentTicket?.status === 'open' ? '处理中' : '已关闭' }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="优先级">
          {{ currentTicket?.priority > 0 ? '紧急' : '普通' }}
        </a-descriptions-item>
      </a-descriptions>
      <div class="ticket-content" style="margin-top: 16px; padding: 16px; background: #f5f5f5; border-radius: 8px">
        {{ currentTicket?.content }}
      </div>
    </a-modal>

    <!-- 回复弹窗 -->
    <a-modal v-model:open="replyVisible" title="回复工单" @ok="submitReply" :confirmLoading="replying">
      <a-textarea v-model:value="replyContent" :rows="4" placeholder="请输入回复内容" />
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getAllTickets, replyTicket as replyTicketApi, closeTicket as closeTicketApi } from '@/api'

const loading = ref(false)
const tickets = ref([])
const detailVisible = ref(false)
const replyVisible = ref(false)
const currentTicket = ref(null)
const replyContent = ref('')
const replying = ref(false)

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '主题', dataIndex: 'subject', key: 'subject' },
  { title: '用户', dataIndex: 'userId', key: 'userId', width: 80 },
  { title: '优先级', key: 'priority', width: 80 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
  { title: '操作', key: 'action', width: 150 }
]

const fetchTickets = async () => {
  loading.value = true
  try {
    const res = await getAllTickets()
    tickets.value = res.obj || []
  } catch (e) {
    message.error('获取工单失败')
  } finally {
    loading.value = false
  }
}

const viewTicket = (ticket) => {
  currentTicket.value = ticket
  detailVisible.value = true
}

const replyTicket = (ticket) => {
  currentTicket.value = ticket
  replyContent.value = ''
  replyVisible.value = true
}

const submitReply = async () => {
  if (!replyContent.value.trim()) {
    message.warning('请输入回复内容')
    return
  }
  replying.value = true
  try {
    await replyTicketApi(currentTicket.value.id, replyContent.value)
    message.success('回复成功')
    replyVisible.value = false
    fetchTickets()
  } catch (e) {
    message.error('回复失败')
  } finally {
    replying.value = false
  }
}

const closeTicketRecord = async (ticket) => {
  try {
    await closeTicketApi(ticket.id)
    message.success('工单已关闭')
    fetchTickets()
  } catch (e) {
    message.error('操作失败')
  }
}

onMounted(() => {
  fetchTickets()
})
</script>