<template>
  <div class="login-page">
    <!-- 背景装饰 -->
    <div class="bg-decoration">
      <div class="bg-circle bg-circle-1"></div>
      <div class="bg-circle bg-circle-2"></div>
      <div class="bg-circle bg-circle-3"></div>
    </div>
    
    <!-- 公告区域 -->
    <div v-if="announcements.length > 0" class="announcements-bar">
      <div class="announcements-container">
        <a-carousel autoplay :dots="false" class="announcement-carousel">
          <div v-for="item in announcements" :key="item.id" class="announcement-item" :class="item.type">
            <span class="announcement-icon">
              <InfoCircleOutlined v-if="item.type === 'info'" />
              <WarningOutlined v-else-if="item.type === 'warning'" />
              <CheckCircleOutlined v-else />
            </span>
            <span class="announcement-title">{{ item.title }}</span>
          </div>
        </a-carousel>
      </div>
    </div>
    
    <!-- 登录卡片 -->
    <div class="login-wrapper">
      <div class="login-card">
        <!-- Logo 和标题 -->
        <div class="login-header">
          <div class="logo-icon">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <h1 class="login-title">NexCore 代理主机</h1>
          <p class="login-subtitle">多节点网络代理管理平台</p>
        </div>
        
        <!-- 登录表单 -->
        <a-form 
          :model="form" 
          @finish="handleLogin"
          layout="vertical"
          class="login-form"
        >
          <a-form-item 
            name="username" 
            :rules="[{ required: true, message: '请输入用户名' }]"
          >
            <a-input 
              v-model:value="form.username" 
              placeholder="请输入用户名" 
              size="large"
            >
              <template #prefix>
                <UserOutlined class="input-icon" />
              </template>
            </a-input>
          </a-form-item>
          
          <a-form-item 
            name="password" 
            :rules="[{ required: true, message: '请输入密码' }]"
          >
            <a-input-password 
              v-model:value="form.password" 
              placeholder="请输入密码" 
              size="large"
            >
              <template #prefix>
                <LockOutlined class="input-icon" />
              </template>
            </a-input-password>
          </a-form-item>
          
          <!-- Turnstile 人机验证 -->
          <a-form-item v-if="turnstileSiteKey" class="turnstile-item">
            <div ref="turnstileRef" class="turnstile-container"></div>
          </a-form-item>
          
          <a-form-item class="login-actions">
            <a-button 
              type="primary" 
              html-type="submit" 
              :loading="loading" 
              block
              size="large"
            >
              登录系统
            </a-button>
          </a-form-item>
        </a-form>
        
        <!-- 注册链接 -->
        <div class="register-link">
          <span>管理员登录入口</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { UserOutlined, LockOutlined, InfoCircleOutlined, WarningOutlined, CheckCircleOutlined } from '@ant-design/icons-vue'
import { login, getAnnouncements } from '@/api'
import request from '@/api/request'

const router = useRouter()
const loading = ref(false)
const announcements = ref([])
const turnstileSiteKey = ref('')
const turnstileRef = ref(null)
const turnstileWidgetId = ref(null)
const turnstileToken = ref('')

const form = ref({
  username: '',
  password: ''
})

// 加载 Turnstile 脚本
const loadTurnstile = () => {
  return new Promise((resolve) => {
    if (window.turnstile) {
      resolve()
      return
    }
    const script = document.createElement('script')
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
    script.async = true
    script.defer = true
    script.onload = resolve
    document.head.appendChild(script)
  })
}

// 渲染 Turnstile 组件
const renderTurnstile = async () => {
  if (!turnstileSiteKey.value || !turnstileRef.value) return
  
  await loadTurnstile()
  await nextTick()
  
  if (window.turnstile && turnstileRef.value) {
    // 清除旧的 widget
    if (turnstileWidgetId.value) {
      window.turnstile.remove(turnstileWidgetId.value)
    }
    
    // 渲染新的 widget
    turnstileWidgetId.value = window.turnstile.render(turnstileRef.value, {
      sitekey: turnstileSiteKey.value,
      theme: 'light',
      callback: (token) => {
        turnstileToken.value = token
      },
      'expired-callback': () => {
        turnstileToken.value = ''
      }
    })
  }
}

// 获取 Turnstile 配置
const fetchTurnstileConfig = async () => {
  try {
    const res = await request.get('/turnstile-config')
    if (res.success && res.obj?.siteKey) {
      turnstileSiteKey.value = res.obj.siteKey
      await nextTick()
      renderTurnstile()
    }
  } catch (e) {
    // 忽略错误，Turnstile 未配置时不显示
  }
}

const handleLogin = async () => {
  // 如果启用了 Turnstile，检查是否已验证
  if (turnstileSiteKey.value && !turnstileToken.value) {
    message.warning('请完成人机验证')
    return
  }
  
  loading.value = true
  try {
    const res = await login({
      username: form.value.username,
      password: form.value.password,
      turnstileToken: turnstileToken.value
    })
    if (res.success && res.obj?.token) {
      // 管理端使用独立的 token 存储
      localStorage.setItem('admin_token', res.obj.token)
      localStorage.setItem('admin_username', res.obj.username || form.value.username)
      message.success('登录成功')
      
      // 管理端登录只跳转到管理端
      router.push('/admin/dashboard')
    } else {
      message.error(res.msg || '登录失败')
      // 重置 Turnstile
      if (window.turnstile && turnstileWidgetId.value) {
        window.turnstile.reset(turnstileWidgetId.value)
        turnstileToken.value = ''
      }
    }
  } catch (e) {
    message.error('登录失败')
    if (window.turnstile && turnstileWidgetId.value) {
      window.turnstile.reset(turnstileWidgetId.value)
      turnstileToken.value = ''
    }
  } finally {
    loading.value = false
  }
}

// 获取公告
const fetchAnnouncements = async () => {
  try {
    const res = await getAnnouncements()
    if (res.success) {
      announcements.value = res.obj || []
    }
  } catch (e) {
    // 忽略错误
  }
}

onMounted(() => {
  fetchAnnouncements()
  fetchTurnstileConfig()
})

onUnmounted(() => {
  // 清理 Turnstile
  if (window.turnstile && turnstileWidgetId.value) {
    window.turnstile.remove(turnstileWidgetId.value)
  }
})
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, #f8fafc 0%, #e6f4ff 50%, #f0f9ff 100%);
}

/* 背景装饰圆圈 */
.bg-decoration {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
}

.bg-circle {
  position: absolute;
  border-radius: 50%;
  opacity: 0.4;
}

.bg-circle-1 {
  width: 600px;
  height: 600px;
  background: linear-gradient(135deg, rgba(22, 119, 255, 0.08) 0%, rgba(19, 194, 194, 0.05) 100%);
  top: -200px;
  right: -100px;
}

.bg-circle-2 {
  width: 400px;
  height: 400px;
  background: linear-gradient(135deg, rgba(19, 194, 194, 0.08) 0%, rgba(22, 119, 255, 0.05) 100%);
  bottom: -100px;
  left: -100px;
}

.bg-circle-3 {
  width: 300px;
  height: 300px;
  background: rgba(22, 119, 255, 0.03);
  top: 50%;
  left: 10%;
  transform: translateY(-50%);
}

/* 公告栏 */
.announcements-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.announcements-container {
  max-width: 800px;
  margin: 0 auto;
  padding: 10px 20px;
}

.announcement-carousel {
  height: 28px;
  line-height: 28px;
}

.announcement-item {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 14px;
}

.announcement-item.info { color: #1677ff; }
.announcement-item.warning { color: #fa8c16; }
.announcement-item.success { color: #52c41a; }
.announcement-icon { font-size: 16px; }
.announcement-title { font-weight: 500; }

/* 登录容器 */
.login-wrapper {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 420px;
  padding: 20px;
  animation: slideUp 0.5s ease-out;
}

/* 登录卡片 */
.login-card {
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(20px);
  border-radius: 20px;
  padding: 40px 36px;
  box-shadow: 
    0 4px 24px rgba(0, 0, 0, 0.04),
    0 1px 2px rgba(0, 0, 0, 0.02),
    0 0 0 1px rgba(255, 255, 255, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.6);
}

@media (max-width: 576px) {
  .login-card {
    padding: 32px 24px;
    margin: 16px;
    border-radius: 16px;
  }
}

/* 头部 */
.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo-icon {
  width: 56px;
  height: 56px;
  margin: 0 auto 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1677ff 0%, #4096ff 100%);
  border-radius: 16px;
  color: white;
  box-shadow: 0 8px 24px rgba(22, 119, 255, 0.25);
}

.logo-icon svg {
  width: 28px;
  height: 28px;
}

.login-title {
  font-size: 24px;
  font-weight: 700;
  color: #1f1f1f;
  margin-bottom: 6px;
  letter-spacing: -0.5px;
}

.login-subtitle {
  font-size: 14px;
  color: #8c8c8c;
  font-weight: 400;
}

/* 表单 */
.login-form :deep(.ant-form-item) {
  margin-bottom: 20px;
}

.login-form :deep(.ant-form-item:last-child) {
  margin-bottom: 0;
}

.login-form :deep(.ant-input-affix-wrapper) {
  padding: 10px 14px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid #e8e8e8;
  transition: all 0.2s ease;
}

.login-form :deep(.ant-input-affix-wrapper:hover) {
  border-color: #4096ff;
}

.login-form :deep(.ant-input-affix-wrapper-focused) {
  border-color: #1677ff;
  box-shadow: 0 0 0 3px rgba(22, 119, 255, 0.08);
}

.login-form :deep(.ant-input) {
  background: transparent;
}

.input-icon {
  color: #bfbfbf;
  font-size: 16px;
}

/* Turnstile */
.turnstile-item {
  margin-bottom: 16px;
}

.turnstile-container {
  display: flex;
  justify-content: center;
}

/* 登录按钮 */
.login-actions :deep(.ant-btn) {
  height: 48px;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  background: linear-gradient(135deg, #1677ff 0%, #4096ff 100%);
  border: none;
  box-shadow: 0 4px 14px rgba(22, 119, 255, 0.3);
  transition: all 0.2s ease;
}

.login-actions :deep(.ant-btn:hover) {
  background: linear-gradient(135deg, #0958d9 0%, #1677ff 100%);
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(22, 119, 255, 0.35);
}

/* 注册链接 */
.register-link {
  text-align: center;
  padding-top: 16px;
  font-size: 14px;
  color: #8c8c8c;
}

.register-link a {
  color: #1677ff;
  cursor: pointer;
  margin-left: 4px;
}

.register-link a:hover {
  text-decoration: underline;
}

/* 动画 */
@keyframes slideUp {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>