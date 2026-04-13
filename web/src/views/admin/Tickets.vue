<template>
  <div class="tickets-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">
          <MessageOutlined class="title-icon" />
          工单管理
        </h1>
        <p class="page-desc">处理用户提交的工单和问题反馈</p>
      </div>
    </div>
    
    <!-- 工单列表卡片 -->
    <a-card class="tickets-card">
      <a-table 
        :columns="columns" 
        :dataSource="tickets" 
        :loading="loading" 
        rowKey="id"
        :pagination="{ pageSize: 10, showSizeChanger: true }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'subject'">
            <div class="ticket-subject" @click="viewTicket(record)">
              <span class="subject-text">{{ record.subject }}</span>
              <span v-if="record.priority > 0" class="urgent-badge">紧急</span>
            </div>
          </template>
          <template v-if="column.key === 'user'">
            <div class="user-cell">
              <a-avatar :size="28" class="user-avatar">
                {{ record.username?.charAt(0)?.toUpperCase() || 'U' }}
              </a-avatar>
              <span>{{ record.username || `用户${record.userId}` }}</span>
            </div>
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
            <div class="action-btns">
              <button class="action-btn" @click="viewTicket(record)">
                <EyeOutlined />
              </button>
              <button class="action-btn primary" @click="replyTicket(record)">
                <CommentOutlined />
              </button>
              <button 
                v-if="record.status === 'open'" 
                class="action-btn" 
                @click="closeTicketRecord(record)"
              >
                <CheckOutlined />
              </button>
            </div>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 工单详情弹窗 -->
    <a-modal 
      v-model:open="detailVisible" 
      :title="currentTicket?.subject" 
      width="700px" 
      :footer="null"
      class="ticket-modal"
    >
      <div class="ticket-meta">
        <div class="meta-item">
          <span class="label">提交用户</span>
          <span class="value">{{ currentTicket?.username || `用户${currentTicket?.userId}` }}</span>
        </div>
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
    </a-modal>

    <!-- 回复弹窗 -->
    <a-modal 
      v-model:open="replyVisible" 
      title="回复工单" 
      @ok="submitReply" 
      :confirmLoading="replying"
      :width="500"
    >
      <div class="reply-info">
        <span class="ticket-label">工单:</span>
        <span class="ticket-name">{{ currentTicket?.subject }}</span>
      </div>
      <a-textarea 
        v-model:value="replyContent" 
        :rows="4" 
        placeholder="请输入回复内容..." 
        class="reply-input"
      />
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, onDeactivated } from 'vue'
import { message } from 'ant-design-vue'
import { MessageOutlined, EyeOutlined, CommentOutlined, CheckOutlined } from '@ant-design/icons-vue'
import { getAllTickets, replyTicket as replyTicketApi, closeTicket as closeTicketApi, getTicketDetail } from '@/api'

const loading = ref(false)
const tickets = ref([])
const detailVisible = ref(false)
const replyVisible = ref(false)

onDeactivated(() => { detailVisible.value = false; replyVisible.value = false })
const currentTicket = ref(null)
const replyContent = ref('')
const replying = ref(false)

const columns = [
  { title: '主题', key: 'subject' },
  { title: '用户', key: 'user', width: 120 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', key: 'createdAt', width: 160 },
  { title: '操作', key: 'action', width: 130, fixed: 'right' }
]

const formatDateTime = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

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

const viewTicket = async (ticket) => {
  currentTicket.value = ticket
  detailVisible.value = true
  try {
    const res = await getTicketDetail(ticket.id)
    currentTicket.value = res.obj.ticket
    currentTicket.value.replies = res.obj.replies || []
  } catch (e) {
    // 加载失败时保留列表数据
  }
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

<style scoped>
.tickets-page {
  animation: fadeIn 0.3s ease;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 22px;
  font-weight: 700;
  color: #1e293b;
  margin: 0;
}

.title-icon {
  color: #3b82f6;
  font-size: 24px;
}

.page-desc {
  color: #64748b;
  font-size: 14px;
  margin-top: 4px;
}

.tickets-card {
  border-radius: 14px;
}

/* 工单主题 */
.ticket-subject {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.ticket-subject:hover .subject-text {
  color: #3b82f6;
}

.subject-text {
  font-weight: 500;
  transition: color 0.15s;
}

.urgent-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  background: #fef2f2;
  color: #dc2626;
}

/* 用户单元格 */
.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  background: linear-gradient(135deg, #0891b2 0%, #36cfc9 100%);
}

/* 状态徽章 */
.status-badge {
  display: inline-flex;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.status-badge.open {
  background: #eff6ff;
  color: #3b82f6;
}

.status-badge.closed {
  background: #f1f5f9;
  color: #64748b;
}

.time-text {
  font-size: 13px;
  color: #64748b;
}

/* 操作按钮 */
.action-btns {
  display: flex;
  gap: 4px;
}

.action-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f1f5f9;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #475569;
  transition: all 0.15s ease;
}

.action-btn:hover {
  background: #eff6ff;
  color: #3b82f6;
}

.action-btn.primary:hover {
  background: #eff6ff;
  color: #3b82f6;
}

/* 工单详情弹窗 */
.ticket-meta {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  padding: 20px;
  background: #f8fafc;
  border-radius: 10px;
  margin-bottom: 20px;
}

.meta-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.meta-item .label {
  font-size: 12px;
  color: #64748b;
}

.meta-item .value {
  font-weight: 500;
  color: #1e293b;
}

.priority-badge {
  display: inline-flex;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.priority-badge.urgent {
  background: #fef2f2;
  color: #dc2626;
}

.priority-badge.normal {
  background: #f1f5f9;
  color: #64748b;
}

.ticket-content {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  overflow: hidden;
}

.content-label {
  padding: 12px 16px;
  background: #f8fafc;
  font-size: 13px;
  font-weight: 600;
  color: #475569;
  border-bottom: 1px solid #e2e8f0;
}

.content-body {
  padding: 16px;
  font-size: 14px;
  line-height: 1.6;
  color: #1e293b;
  white-space: pre-wrap;
}

/* 回复弹窗 */
.reply-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #f8fafc;
  border-radius: 8px;
}

.ticket-label {
  color: #64748b;
}

.ticket-name {
  font-weight: 500;
  color: #1e293b;
}

.reply-input {
  border-radius: 10px;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>