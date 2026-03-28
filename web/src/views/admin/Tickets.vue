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
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { MessageOutlined, EyeOutlined, CommentOutlined, CheckOutlined } from '@ant-design/icons-vue'
import { getAllTickets, replyTicket as replyTicketApi, closeTicket as closeTicketApi } from '@/api'

const loading = ref(false)
const tickets = ref([])
const detailVisible = ref(false)
const replyVisible = ref(false)
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
  color: #1677ff;
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
  background: #fff2f0;
  color: #ff4d4f;
}

/* 用户单元格 */
.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  background: linear-gradient(135deg, #13c2c2 0%, #36cfc9 100%);
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
  background: #f5f5f5;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #595959;
  transition: all 0.15s ease;
}

.action-btn:hover {
  background: #e6f4ff;
  color: #1677ff;
}

.action-btn.primary:hover {
  background: #e6f4ff;
  color: #1677ff;
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
  color: #8c8c8c;
}

.meta-item .value {
  font-weight: 500;
  color: #262626;
}

.priority-badge {
  display: inline-flex;
  padding: 4px 12px;
  border-radius: 20px;
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

.ticket-content {
  background: white;
  border: 1px solid #f0f0f0;
  border-radius: 10px;
  overflow: hidden;
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
  color: #8c8c8c;
}

.ticket-name {
  font-weight: 500;
  color: #262626;
}

.reply-input {
  border-radius: 10px;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>