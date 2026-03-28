<template>
  <div class="my-tickets-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h1>我的工单</h1>
        <p>提交问题反馈，获取技术支持</p>
      </div>
      <a-button type="primary" size="large" @click="showCreateModal">
        <PlusOutlined /> 提交工单
      </a-button>
    </div>
    
    <!-- 工单列表卡片 -->
    <a-card class="tickets-card">
      <a-table 
        :columns="columns" 
        :dataSource="tickets" 
        :loading="loading" 
        rowKey="id"
        :pagination="{ pageSize: 10 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'subject'">
            <span class="subject-text" @click="viewTicket(record)">{{ record.subject }}</span>
          </template>
          <template v-if="column.key === 'priority'">
            <span :class="['priority-badge', record.priority > 0 ? 'urgent' : 'normal']">
              {{ record.priority > 0 ? '紧急' : '普通' }}
            </span>
          </template>
          <template v-if="column.key === 'status'">
            <span :class="['status-badge', record.status]">
              {{ record.status === 'open' ? '处理中' : '已关闭' }}
            </span>
          </template>
          <template v-if="column.key === 'createdAt'">
            <span class="time-text">{{ formatDateTime(record.createdAt) }}</span>
          </template>
          <template v-if="column.key === 'action'">
            <button class="view-btn" @click="viewTicket(record)">
              <EyeOutlined /> 查看
            </button>
          </template>
        </template>
      </a-table>
      
      <!-- 空状态 -->
      <div v-if="!loading && tickets.length === 0" class="empty-state">
        <MessageOutlined class="empty-icon" />
        <p>暂无工单记录</p>
        <a-button type="primary" @click="showCreateModal">
          提交工单
        </a-button>
      </div>
    </a-card>

    <!-- 创建工单弹窗 -->
    <a-modal 
      v-model:open="createVisible" 
      title="提交工单" 
      @ok="createTicketSubmit" 
      :confirmLoading="creating"
      :width="520"
    >
      <a-form :model="form" layout="vertical" class="ticket-form">
        <a-form-item label="主题" required>
          <a-input v-model:value="form.subject" placeholder="请输入工单主题" />
        </a-form-item>
        <a-form-item label="优先级">
          <a-radio-group v-model:value="form.priority" class="priority-radio">
            <a-radio :value="0">普通</a-radio>
            <a-radio :value="1">紧急</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="内容" required>
          <a-textarea v-model:value="form.content" :rows="5" placeholder="请详细描述您遇到的问题..." />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 工单详情弹窗 -->
    <a-modal 
      v-model:open="detailVisible" 
      :title="currentTicket?.subject" 
      width="700px" 
      :footer="null"
      class="detail-modal"
    >
      <div class="ticket-meta">
        <div class="meta-item">
          <span class="label">状态</span>
          <span :class="['status-badge', currentTicket?.status]">
            {{ currentTicket?.status === 'open' ? '处理中' : '已关闭' }}
          </span>
        </div>
        <div class="meta-item">
          <span class="label">优先级</span>
          <span :class="['priority-badge', currentTicket?.priority > 0 ? 'urgent' : 'normal']">
            {{ currentTicket?.priority > 0 ? '紧急' : '普通' }}
          </span>
        </div>
        <div class="meta-item">
          <span class="label">提交时间</span>
          <span class="value">{{ formatDateTime(currentTicket?.createdAt) }}</span>
        </div>
      </div>
      
      <div class="ticket-content">
        <div class="content-label">工单内容</div>
        <div class="content-body">{{ currentTicket?.content }}</div>
      </div>
      
      <div class="replies-section">
        <div class="section-title">回复记录</div>
        <div v-if="!currentTicket?.replies?.length" class="no-replies">
          暂无回复
        </div>
        <div v-else class="replies-list">
          <div v-for="reply in currentTicket?.replies" :key="reply.id" class="reply-item">
            <div class="reply-header">
              <span :class="['author', reply.userId ? 'user' : 'admin']">
                {{ reply.userId ? '我' : '客服' }}
              </span>
              <span class="time">{{ formatDateTime(reply.createdAt) }}</span>
            </div>
            <div class="reply-content">{{ reply.content }}</div>
          </div>
        </div>
      </div>
      
      <!-- 回复输入区 -->
      <div class="reply-input-section">
        <a-textarea 
          v-model:value="replyContent" 
          :rows="3" 
          placeholder="输入回复内容..." 
          class="reply-textarea"
        />
        <div class="reply-actions">
          <a-button @click="detailVisible = false">关闭</a-button>
          <a-button type="primary" @click="submitReply" :loading="replying" :disabled="!replyContent.trim()">
            <SendOutlined /> 发送回复
          </a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, MessageOutlined, EyeOutlined, SendOutlined } from '@ant-design/icons-vue'
import { getMyTickets, createTicket, getTicketDetail, replyMyTicket } from '@/api'

const loading = ref(false)
const tickets = ref([])
const createVisible = ref(false)
const detailVisible = ref(false)
const creating = ref(false)
const replying = ref(false)
const currentTicket = ref(null)
const replyContent = ref('')

const form = ref({
  subject: '',
  content: '',
  priority: 0
})

const columns = [
  { title: '主题', key: 'subject' },
  { title: '优先级', key: 'priority', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', key: 'createdAt', width: 180 },
  { title: '操作', key: 'action', width: 100 }
]

const formatDateTime = (date) => {
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
    replyContent.value = ''
    detailVisible.value = true
  } catch (e) {
    message.error('获取详情失败')
  }
}

const submitReply = async () => {
  if (!replyContent.value.trim()) {
    message.warning('请输入回复内容')
    return
  }
  replying.value = true
  try {
    await replyMyTicket(currentTicket.value.id, replyContent.value)
    message.success('回复成功')
    replyContent.value = ''
    // 刷新工单详情
    const res = await getTicketDetail(currentTicket.value.id)
    currentTicket.value = res.obj
  } catch (e) {
    message.error('回复失败')
  } finally {
    replying.value = false
  }
}

onMounted(() => {
  fetchTickets()
})
</script>

<style scoped>
.my-tickets-page {
  animation: fadeIn 0.3s ease;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
  gap: 16px;
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

.tickets-card {
  border-radius: 14px;
}

.subject-text {
  color: #1677ff;
  cursor: pointer;
  font-weight: 500;
}

.subject-text:hover {
  text-decoration: underline;
}

/* 优先级徽章 */
.priority-badge {
  display: inline-flex;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.priority-badge.urgent {
  background: #fff2f0;
  color: #ff4d4f;
}

.priority-badge.normal {
  background: #f5f5f5;
  color: #8c8c8c;
}

/* 状态徽章 */
.status-badge {
  display: inline-flex;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.status-badge.open {
  background: #e6f4ff;
  color: #1677ff;
}

.status-badge.closed {
  background: #f5f5f5;
  color: #8c8c8c;
}

.time-text {
  font-size: 13px;
  color: #8c8c8c;
}

.view-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: #f5f5f5;
  border: none;
  border-radius: 6px;
  color: #595959;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.15s ease;
}

.view-btn:hover {
  background: #e6f4ff;
  color: #1677ff;
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

/* 工单表单 */
.ticket-form :deep(.ant-form-item-label > label) {
  font-weight: 500;
}

.priority-radio :deep(.ant-radio-wrapper) {
  margin-right: 24px;
}

/* 工单详情 */
.ticket-meta {
  display: flex;
  gap: 24px;
  padding: 16px;
  background: #f8fafc;
  border-radius: 10px;
  margin-bottom: 20px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-item .label {
  color: #8c8c8c;
  font-size: 13px;
}

.meta-item .value {
  font-weight: 500;
}

.ticket-content {
  background: white;
  border: 1px solid #f0f0f0;
  border-radius: 10px;
  overflow: hidden;
  margin-bottom: 20px;
}

.content-label {
  padding: 12px 16px;
  background: #f8fafc;
  font-size: 13px;
  font-weight: 600;
  color: #595959;
  border-bottom: 1px solid #f0f0f0;
}

.content-body {
  padding: 16px;
  font-size: 14px;
  line-height: 1.6;
  color: #262626;
  white-space: pre-wrap;
}

.replies-section {
  border-top: 1px solid #f0f0f0;
  padding-top: 20px;
  margin-bottom: 20px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: #262626;
  margin-bottom: 16px;
}

.no-replies {
  text-align: center;
  padding: 24px;
  color: #8c8c8c;
  background: #f8fafc;
  border-radius: 10px;
}

.replies-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.reply-item {
  padding: 16px;
  background: #f8fafc;
  border-radius: 10px;
}

.reply-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.reply-header .author {
  font-weight: 600;
  font-size: 13px;
}

.reply-header .author.user {
  color: #1677ff;
}

.reply-header .author.admin {
  color: #52c41a;
}

.reply-header .time {
  font-size: 12px;
  color: #8c8c8c;
}

.reply-content {
  font-size: 14px;
  color: #262626;
  line-height: 1.6;
}

/* 回复输入区 */
.reply-input-section {
  border-top: 1px solid #f0f0f0;
  padding-top: 20px;
}

.reply-textarea {
  margin-bottom: 12px;
  border-radius: 10px;
}

.reply-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 响应式 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
  }
  
  .page-header .ant-btn {
    width: 100%;
  }
  
  .ticket-meta {
    flex-wrap: wrap;
    gap: 12px;
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>