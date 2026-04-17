<template>
  <div class="settings-page">
    <!-- 设置卡片 -->
    <div class="settings-grid">
      <!-- 密码修改 -->
      <a-card class="settings-card">
        <template #title>
          <div class="card-title">
            <LockOutlined />
            密码修改
          </div>
        </template>
        
        <a-form layout="vertical" class="password-form">
          <a-form-item label="当前密码">
            <a-input-password 
              v-model:value="passwordForm.oldPassword" 
              placeholder="请输入当前密码" 
            />
          </a-form-item>
          <a-form-item label="新密码">
            <a-input-password 
              v-model:value="passwordForm.newPassword" 
              placeholder="请输入新密码" 
            />
          </a-form-item>
          <a-form-item>
            <a-button type="primary" @click="changePassword">
              修改密码
            </a-button>
          </a-form-item>
        </a-form>
      </a-card>
      
      <!-- 安装命令 -->
      <a-card class="settings-card">
        <template #title>
          <div class="card-title">
            <CodeOutlined />
            节点安装命令
          </div>
        </template>
        
        <p class="card-desc">在节点服务器上执行以下命令安装代理程序：</p>
        
        <div class="code-block">
          <pre>{{ installCommand }}</pre>
          <button class="copy-btn" @click="copyCommand">
            <CopyOutlined />
          </button>
        </div>
        
        <div class="tips">
          <InfoCircleOutlined />
          <span>安装完成后，节点将自动连接到主控并上线</span>
        </div>
      </a-card>
      
      <!-- 关于 -->
      <a-card class="settings-card">
        <template #title>
          <div class="card-title">
            <InfoCircleOutlined />
            关于系统
          </div>
        </template>
        
        <div class="about-content">
          <div class="about-logo">
            <div class="logo-icon">
              <svg viewBox="0 0 24 24" fill="none">
                <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2"/>
                <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2"/>
                <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2"/>
              </svg>
            </div>
            <div class="logo-text">NexCore</div>
          </div>
          
          <div class="about-info">
            <div class="info-item">
              <span class="label">版本</span>
              <span class="value">v1.0.0</span>
            </div>
            <div class="info-item">
              <span class="label">框架</span>
              <span class="value">Go + Gin + Vue 3</span>
            </div>
            <div class="info-item">
              <span class="label">说明</span>
              <span class="value">基于 x-ui 的多节点网络代理管理系统</span>
            </div>
          </div>
        </div>
        
        <div class="about-links">
          <a href="https://github.com/DoBestone/NexCoreProxy-Master" target="_blank" class="link-btn">
            <GithubOutlined />
            GitHub
          </a>
        </div>
      </a-card>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import {
  SettingOutlined, LockOutlined, CodeOutlined,
  InfoCircleOutlined, CopyOutlined, GithubOutlined
} from '@ant-design/icons-vue'
import { updatePassword } from '@/api'

const passwordForm = ref({
  oldPassword: '',
  newPassword: ''
})

const installCommand = `bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/NexCoreProxy-Agent/main/install.sh) -u admin -pass YourPassword`

const changingPassword = ref(false)
const changePassword = async () => {
  if (!passwordForm.value.oldPassword || !passwordForm.value.newPassword) {
    message.warning('请填写完整')
    return
  }
  if (passwordForm.value.newPassword.length < 6) {
    message.warning('新密码至少6位')
    return
  }
  changingPassword.value = true
  try {
    await updatePassword(passwordForm.value)
    message.success('密码修改成功')
    passwordForm.value = { oldPassword: '', newPassword: '' }
  } catch (e) {
    // error already shown by request interceptor
  } finally {
    changingPassword.value = false
  }
}

const copyCommand = () => {
  navigator.clipboard.writeText(installCommand)
  message.success('已复制到剪贴板')
}
</script>

<style scoped>
.settings-page {
  animation: fadeIn 0.3s ease;
  max-width: 900px;
}

.settings-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.settings-card {
  border-radius: 14px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.card-title .anticon {
  color: #3b82f6;
}

.card-desc {
  color: #475569;
  margin-bottom: 16px;
}

/* 密码表单 */
.password-form {
  max-width: 320px;
}

/* 代码块 */
.code-block {
  position: relative;
  background: #1e1e1e;
  border-radius: 10px;
  padding: 16px;
  overflow-x: auto;
}

.code-block pre {
  margin: 0;
  color: #d4d4d4;
  font-family: 'SF Mono', Monaco, 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}

.copy-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.1);
  border: none;
  border-radius: 6px;
  color: #d4d4d4;
  cursor: pointer;
  transition: all 0.15s ease;
}

.copy-btn:hover {
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

.tips {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 12px 16px;
  background: #eff6ff;
  border-radius: 8px;
  font-size: 13px;
  color: #3b82f6;
}

/* 关于内容 */
.about-content {
  margin-bottom: 20px;
}

.about-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #e2e8f0;
}

.logo-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  border-radius: 12px;
  color: white;
}

.logo-icon svg {
  width: 28px;
  height: 28px;
}

.logo-text {
  font-size: 20px;
  font-weight: 700;
  color: #1e293b;
}

.about-info {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-item {
  display: flex;
  gap: 16px;
}

.info-item .label {
  color: #64748b;
  width: 60px;
  flex-shrink: 0;
}

.info-item .value {
  color: #1e293b;
}

.about-links {
  display: flex;
  gap: 12px;
}

.link-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: #f1f5f9;
  border-radius: 8px;
  color: #475569;
  text-decoration: none;
  transition: all 0.15s ease;
}

.link-btn:hover {
  background: #eff6ff;
  color: #3b82f6;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>