<template>
  <div class="my-tickets">
    <a-card title="我的工单">
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <template #icon><PlusOutlined /></template>
          提交工单
        </a-button>
      </template>
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
            <a-button type="link" size="small" @click="viewTicket(record)">查看</a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建工单弹窗 -->
    <a-modal v-model:open="createVisible" title="提交工单" @ok="createTicketSubmit" :confirmLoading="creating">
      <a-form :model="form" :label-col="{ span: 4 }">
        <a-form-item label="主题" required>
          <a-input v-model:value="form.subject" placeholder="请输入工单主题" />
        </a-form-item>
        <a-form-item label="优先级">
          <a-radio-group v-model:value="form.priority">
            <a-radio :value="0">普通</a-radio>
            <a-radio :value="1">紧急</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="内容" required>
          <a-textarea v-model:value="form.content" :rows="4" placeholder="请描述您遇到的问题" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 工单详情弹窗 -->
    <a-modal v-model:open="detailVisible" :title="currentTicket?.subject" width="700px" :footer="null">
      <a-descriptions :column="1" bordered size="small">
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
      <a-divider>回复记录</a-divider>
      <a-empty v-if="!currentTicket?.replies?.length" description="暂无回复" />
      <div v-else class="replies">
        <div v-for="reply in currentTicket?.replies" :key="reply.id" class="reply-item">
          <div class="reply-header">
            <span :class="reply.userId ? 'user' : 'admin'">
              {{ reply.userId ? '我' : '客服' }}
            </span>
            <span class="time">{{ formatDate(reply.createdAt) }}</span>
          </div>
          <div class="reply-content">{{ reply.content }}</div>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { getMyTickets, createTicket, getTicketDetail } from '@/api'

const loading = ref(false)
const tickets = ref([])
const createVisible = ref(false)
const detailVisible = ref(false)
const creating = ref(false)
const currentTicket = ref(null)

const form = ref({
  subject: '',
  content: '',
  priority: 0
})

const columns = [
  { title: '主题', dataIndex: 'subject', key: 'subject' },
  { title: '优先级', key: 'priority', width: 80 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
  { title: '操作', key: 'action', width: 80 }
]

const formatDate = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

const fetchTickets = async () => {
  loading.value = true
  try {
    const res = await getMyTickets()
    tickets.value = res.obj || []
  } catch (e) {
    message.error('获取工单失败')
  } finally {
    loading.value = false
  }
}

const showCreateModal = () => {
  form.value = { subject: '', content: '', priority: 0 }
  createVisible.value = true
}

const createTicketSubmit = async () => {
  if (!form.value.subject || !form.value.content) {
    message.warning('请填写完整')
    return
  }
  creating.value = true
  try {
    await createTicket(form.value)
    message.success('工单提交成功')
    createVisible.value = false
    fetchTickets()
  } catch (e) {
    message.error('提交失败')
  } finally {
    creating.value = false
  }
}

const viewTicket = async (ticket) => {
  try {
    const res = await getTicketDetail(ticket.id)
    currentTicket.value = res.obj
    detailVisible.value = true
  } catch (e) {
    message.error('获取详情失败')
  }
}

onMounted(() => {
  fetchTickets()
})
</script>

<style scoped>
.my-tickets {
  max-width: 1000px;
  margin: 0 auto;
}

.reply-item {
  margin-bottom: 16px;
  padding: 12px;
  background: #fafafa;
  border-radius: 8px;
}

.reply-header {
  margin-bottom: 8px;
}

.reply-header .user {
  color: #1890ff;
  font-weight: bold;
}

.reply-header .admin {
  color: #52c41a;
  font-weight: bold;
}

.reply-header .time {
  float: right;
  color: #999;
  font-size: 12px;
}

.reply-content {
  color: #333;
}
</style>