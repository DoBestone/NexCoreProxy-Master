<template>
  <div class="announcements-page">
    <div class="page-toolbar">
      <a-button type="primary" @click="showAddModal">
        <template #icon><PlusOutlined /></template>
        发布公告
      </a-button>
    </div>

    <!-- 公告列表 -->
    <div class="announcements-list">
      <div 
        v-for="announcement in announcements" 
        :key="announcement.id" 
        class="announcement-card"
        :class="announcement.type"
      >
        <div class="announcement-header">
          <div class="announcement-info">
            <span class="announcement-title">
              <span v-if="announcement.pinned" class="pinned-badge">置顶</span>
              {{ announcement.title }}
            </span>
            <span class="announcement-time">{{ formatDateTime(announcement.createdAt) }}</span>
          </div>
          <div class="announcement-status">
            <a-switch 
              v-model:checked="announcement.enable" 
              @change="toggleEnable(announcement)"
              size="small"
            />
          </div>
        </div>
        
        <div class="announcement-content">{{ announcement.content }}</div>
        
        <div class="announcement-actions">
          <a-button size="small" @click="editAnnouncement(announcement)">
            <EditOutlined /> 编辑
          </a-button>
          <a-popconfirm title="确定删除此公告?" @confirm="deleteAnnouncementRecord(announcement.id)">
            <a-button size="small" danger>
              <DeleteOutlined /> 删除
            </a-button>
          </a-popconfirm>
        </div>
      </div>
      
      <!-- 空状态 -->
      <div v-if="!loading && announcements.length === 0" class="empty-state">
        <NotificationOutlined class="empty-icon" />
        <p>暂无公告</p>
        <a-button type="primary" @click="showAddModal">
          发布第一条公告
        </a-button>
      </div>
    </div>
    
    <!-- 加载状态 -->
    <div v-if="loading" class="loading-container">
      <a-spin size="large" />
    </div>

    <!-- 添加/编辑公告弹窗 -->
    <a-modal 
      v-model:open="modalVisible" 
      :title="editingAnnouncement ? '编辑公告' : '发布公告'" 
      @ok="handleSubmit" 
      :confirmLoading="submitting"
      :width="600"
    >
      <a-form :model="form" layout="vertical" class="announcement-form">
        <a-form-item label="标题" required>
          <a-input v-model:value="form.title" placeholder="请输入公告标题" />
        </a-form-item>
        
        <a-form-item label="类型">
          <a-radio-group v-model:value="form.type">
            <a-radio value="info">
              <span class="type-info">ℹ️ 信息</span>
            </a-radio>
            <a-radio value="warning">
              <span class="type-warning">⚠️ 警告</span>
            </a-radio>
            <a-radio value="success">
              <span class="type-success">✅ 成功</span>
            </a-radio>
          </a-radio-group>
        </a-form-item>
        
        <a-form-item label="内容" required>
          <a-textarea v-model:value="form.content" :rows="5" placeholder="请输入公告内容" />
        </a-form-item>
        
        <a-form-item>
          <a-checkbox v-model:checked="form.pinned">置顶显示</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, onDeactivated } from 'vue'
import { message } from 'ant-design-vue'
import { 
  NotificationOutlined, PlusOutlined, EditOutlined, DeleteOutlined 
} from '@ant-design/icons-vue'
import request from '@/api/request'

const loading = ref(false)
const announcements = ref([])
const modalVisible = ref(false)

onDeactivated(() => { modalVisible.value = false })
const editingAnnouncement = ref(null)
const submitting = ref(false)

const form = ref({
  title: '',
  content: '',
  type: 'info',
  pinned: false
})

const formatDateTime = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

const fetchAnnouncements = async () => {
  loading.value = true
  try {
    const res = await request.get('/admin/announcements')
    announcements.value = res.obj || []
  } catch (e) {
    message.error('获取公告失败')
  } finally {
    loading.value = false
  }
}

const showAddModal = () => {
  editingAnnouncement.value = null
  form.value = { title: '', content: '', type: 'info', pinned: false }
  modalVisible.value = true
}

const editAnnouncement = (announcement) => {
  editingAnnouncement.value = announcement
  form.value = { ...announcement }
  modalVisible.value = true
}

const handleSubmit = async () => {
  if (!form.value.title || !form.value.content) {
    message.warning('请填写完整')
    return
  }
  
  submitting.value = true
  try {
    if (editingAnnouncement.value) {
      await request.put(`/admin/announcements/${editingAnnouncement.value.id}`, form.value)
      message.success('更新成功')
    } else {
      await request.post('/admin/announcements', form.value)
      message.success('发布成功')
    }
    modalVisible.value = false
    fetchAnnouncements()
  } catch (e) {
    message.error('操作失败')
  } finally {
    submitting.value = false
  }
}

const deleteAnnouncementRecord = async (id) => {
  try {
    await request.delete(`/admin/announcements/${id}`)
    message.success('删除成功')
    fetchAnnouncements()
  } catch (e) {
    message.error('删除失败')
  }
}

const toggleEnable = async (announcement) => {
  try {
    await request.put(`/admin/announcements/${announcement.id}`, { enable: announcement.enable })
    message.success('状态已更新')
  } catch (e) {
    message.error('更新失败')
  }
}

onMounted(() => {
  fetchAnnouncements()
})
</script>

<style scoped>
.announcements-page {
  animation: fadeIn 0.3s ease;
}

.page-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 14px;
}

/* 公告列表 */
.announcements-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 公告卡片 */
.announcement-card {
  background: white;
  border-radius: 14px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  border: 1px solid rgba(0, 0, 0, 0.04);
  border-left: 4px solid #3b82f6;
  transition: all 0.2s ease;
}

.announcement-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
}

.announcement-card.warning {
  border-left-color: #d97706;
}

.announcement-card.success {
  border-left-color: #16a34a;
}

.announcement-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.announcement-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.announcement-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  display: flex;
  align-items: center;
  gap: 8px;
}

.pinned-badge {
  background: #dc2626;
  color: white;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.announcement-time {
  font-size: 13px;
  color: #64748b;
}

.announcement-content {
  font-size: 14px;
  color: #475569;
  line-height: 1.6;
  margin-bottom: 16px;
  white-space: pre-wrap;
}

.announcement-actions {
  display: flex;
  gap: 8px;
  padding-top: 16px;
  border-top: 1px solid #e2e8f0;
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 64px 24px;
  background: white;
  border-radius: 14px;
}

.empty-icon {
  font-size: 48px;
  color: #cbd5e1;
  margin-bottom: 16px;
}

.empty-state p {
  color: #64748b;
  margin-bottom: 20px;
}

/* 加载状态 */
.loading-container {
  display: flex;
  justify-content: center;
  padding: 48px;
}

/* 表单样式 */
.announcement-form :deep(.ant-form-item-label > label) {
  font-weight: 500;
}

.type-info { color: #3b82f6; }
.type-warning { color: #d97706; }
.type-success { color: #16a34a; }

/* 响应式 */
@media (max-width: 768px) {
  .page-toolbar .ant-btn { width: 100%; }
  
  .announcement-header {
    flex-direction: column;
    gap: 12px;
  }
  
  .announcement-actions {
    flex-wrap: wrap;
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>