<template>
  <div class="settings-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1>
        <SettingOutlined class="title-icon" />
        账户设置
      </h1>
      <p>管理您的账户信息和安全设置</p>
    </div>
    
    <!-- 修改密码卡片 -->
    <a-card class="settings-card" title="修改密码">
      <a-form :model="passwordForm" layout="vertical" class="password-form">
        <a-row :gutter="24">
          <a-col :span="12">
            <a-form-item label="当前密码" required>
              <a-input-password 
                v-model:value="passwordForm.oldPassword" 
                placeholder="请输入当前密码"
              />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="新密码" required>
              <a-input-password 
                v-model:value="passwordForm.newPassword" 
                placeholder="请输入新密码"
              />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="24">
          <a-col :span="12">
            <a-form-item label="确认新密码" required>
              <a-input-password 
                v-model:value="passwordForm.confirmPassword" 
                placeholder="请再次输入新密码"
              />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label=" ">
              <a-button type="primary" @click="changePassword" :loading="changing">
                <SaveOutlined /> 保存修改
              </a-button>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-card>
    
    <!-- 账户信息卡片 -->
    <a-card class="settings-card" title="账户信息">
      <div class="info-list">
        <div class="info-item">
          <span class="info-label">用户名</span>
          <span class="info-value">{{ userInfo.username }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">邮箱</span>
          <span class="info-value">{{ userInfo.email || '未设置' }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">余额</span>
          <span class="info-value balance">${{ userInfo.balance?.toFixed(2) || '0.00' }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">流量限额</span>
          <span class="info-value">{{ formatTraffic(userInfo.trafficLimit) || '无限制' }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">已用流量</span>
          <span class="info-value">{{ formatTraffic(userInfo.trafficUsed) }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">到期时间</span>
          <span class="info-value" :class="{ expired: isExpired }">
            {{ userInfo.expireAt ? formatDate(userInfo.expireAt) : '永久有效' }}
          </span>
        </div>
        <div class="info-item">
          <span class="info-label">注册时间</span>
          <span class="info-value">{{ formatDate(userInfo.createdAt) }}</span>
        </div>
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import { SettingOutlined, SaveOutlined } from '@ant-design/icons-vue'
import { getUserInfo, updatePassword } from '@/api'

const changing = ref(false)
const userInfo = ref({})
const passwordForm = ref({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const isExpired = computed(() => {
  if (!userInfo.value.expireAt) return false
  return new Date(userInfo.value.expireAt) < new Date()
})

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatDate = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleDateString('zh-CN')
}

const fetchUserInfo = async () => {
  try {
    const res = await getUserInfo()
    userInfo.value = res.obj || {}
  } catch (e) {
    message.error('获取用户信息失败')
  }
}

const changePassword = async () => {
  const { oldPassword, newPassword, confirmPassword } = passwordForm.value
  
  if (!oldPassword || !newPassword || !confirmPassword) {
    message.warning('请填写完整')
    return
  }
  
  if (newPassword !== confirmPassword) {
    message.warning('两次输入的密码不一致')
    return
  }
  
  if (newPassword.length < 6) {
    message.warning('密码长度至少6位')
    return
  }
  
  changing.value = true
  try {
    await updatePassword({ oldPassword, newPassword })
    message.success('密码修改成功')
    passwordForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
  } catch (e) {
    message.error(e.response?.data?.msg || '修改失败')
  } finally {
    changing.value = false
  }
}

onMounted(() => {
  fetchUserInfo()
})
</script>

<style scoped>
.settings-page {
  animation: fadeIn 0.3s ease;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h1 {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 22px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 8px;
}

.title-icon {
  color: #3b82f6;
  font-size: 24px;
}

.page-header p {
  color: #64748b;
  font-size: 14px;
  margin: 0;
}

.settings-card {
  border-radius: 14px;
  margin-bottom: 24px;
}

.settings-card :deep(.ant-card-head-title) {
  font-weight: 600;
}

.password-form {
  max-width: 600px;
}

.info-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f8fafc;
  border-radius: 10px;
}

.info-label {
  color: #64748b;
  font-size: 14px;
}

.info-value {
  font-weight: 500;
  color: #1e293b;
}

.info-value.balance {
  color: #dc2626;
  font-size: 16px;
}

.info-value.expired {
  color: #dc2626;
}

@media (max-width: 768px) {
  .password-form :deep(.ant-col) {
    width: 100%;
    flex: none;
    max-width: 100%;
  }
  
  .info-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>