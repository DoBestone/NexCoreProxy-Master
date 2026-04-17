<template>
  <div class="login-page">
    <!-- 公告条 -->
    <div v-if="announcements.length > 0" class="announce-bar">
      <div class="announce-inner">
        <a-carousel autoplay :dots="false" class="announce-carousel">
          <div
            v-for="item in announcements"
            :key="item.id"
            class="announce-slide"
            :class="item.type"
          >
            <span class="announce-ico">
              <InfoCircleOutlined v-if="item.type === 'info'" />
              <WarningOutlined v-else-if="item.type === 'warning'" />
              <CheckCircleOutlined v-else />
            </span>
            <span class="announce-text">{{ item.title }}</span>
          </div>
        </a-carousel>
      </div>
    </div>

    <!-- 背景装饰 -->
    <div class="bg-grid"></div>
    <div class="bg-orb bg-orb-1"></div>
    <div class="bg-orb bg-orb-2"></div>

    <!-- 主内容 -->
    <div class="shell">
      <!-- 左：品牌 -->
      <aside class="brand hide-mobile">
        <div class="brand-top">
          <div class="brand-logo">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <span class="brand-name">NexCore</span>
        </div>

        <div class="brand-copy">
          <span class="brand-eyebrow">USER CENTER</span>
          <h2 class="brand-title">安全、稳定、<br/>随时随地连接。</h2>
          <p class="brand-sub">
            多协议订阅 · 按量计费 · 全球节点组合，
            一个账号管理你的全部代理资源。
          </p>
        </div>

        <ul class="brand-highlights">
          <li><span class="hl-dot"></span><span>多协议 / 多节点统一订阅</span></li>
          <li><span class="hl-dot"></span><span>流量用量实时可视化</span></li>
          <li><span class="hl-dot"></span><span>客服工单 24 小时内响应</span></li>
        </ul>
      </aside>

      <!-- 右：登录表单 -->
      <section class="login-card">
        <div class="login-head">
          <div class="brand-logo brand-logo-small hide-desktop">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2"/>
              <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2"/>
              <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <h1 class="login-title">登录账户</h1>
          <p class="login-sub">管理订阅、流量与订单</p>
        </div>

        <a-form
          :model="form"
          @finish="handleLogin"
          layout="vertical"
          class="login-form"
        >
          <a-form-item
            name="username"
            label="用户名"
            :rules="[{ required: true, message: '请输入用户名' }]"
          >
            <a-input
              v-model:value="form.username"
              placeholder="输入用户名"
              size="large"
              autocomplete="username"
            >
              <template #prefix><UserOutlined class="input-icon" /></template>
            </a-input>
          </a-form-item>

          <a-form-item
            name="password"
            label="密码"
            :rules="[{ required: true, message: '请输入密码' }]"
          >
            <a-input-password
              v-model:value="form.password"
              placeholder="输入密码"
              size="large"
              autocomplete="current-password"
            >
              <template #prefix><LockOutlined class="input-icon" /></template>
            </a-input-password>
          </a-form-item>

          <a-form-item v-if="turnstileSiteKey" class="turnstile-item">
            <div ref="turnstileRef" class="turnstile-container"></div>
          </a-form-item>

          <a-button
            type="primary"
            html-type="submit"
            :loading="loading"
            block
            size="large"
            class="login-btn"
          >
            登 录
          </a-button>
        </a-form>

        <div class="login-foot">
          <span>还没有账号？</span>
          <a @click="$router.push('/user/register')">立即注册 →</a>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  UserOutlined,
  LockOutlined,
  InfoCircleOutlined,
  WarningOutlined,
  CheckCircleOutlined
} from '@ant-design/icons-vue'
import { login, getAnnouncements } from '@/api'
import request from '@/api/request'

const router = useRouter()
const loading = ref(false)
const announcements = ref([])
const turnstileSiteKey = ref('')
const turnstileRef = ref(null)
const turnstileWidgetId = ref(null)
const turnstileToken = ref('')

const form = ref({ username: '', password: '' })

const loadTurnstile = () => new Promise((resolve) => {
  if (window.turnstile) return resolve()
  const s = document.createElement('script')
  s.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
  s.async = true
  s.defer = true
  s.onload = resolve
  document.head.appendChild(s)
})

const renderTurnstile = async () => {
  if (!turnstileSiteKey.value || !turnstileRef.value) return
  await loadTurnstile()
  await nextTick()
  if (window.turnstile && turnstileRef.value) {
    if (turnstileWidgetId.value) window.turnstile.remove(turnstileWidgetId.value)
    turnstileWidgetId.value = window.turnstile.render(turnstileRef.value, {
      sitekey: turnstileSiteKey.value,
      theme: 'light',
      callback: (token) => { turnstileToken.value = token },
      'expired-callback': () => { turnstileToken.value = '' }
    })
  }
}

const fetchTurnstileConfig = async () => {
  try {
    const res = await request.get('/turnstile-config')
    if (res.success && res.obj?.siteKey) {
      turnstileSiteKey.value = res.obj.siteKey
      await nextTick()
      renderTurnstile()
    }
  } catch (e) { /* 未配置则不显示 */ }
}

const handleLogin = async () => {
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
      localStorage.setItem('user_token', res.obj.token)
      localStorage.setItem('user_username', res.obj.username || form.value.username)
      message.success('登录成功')
      router.push('/user/dashboard')
    } else {
      message.error(res.msg || '登录失败')
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

const fetchAnnouncements = async () => {
  try {
    const res = await getAnnouncements()
    if (res.success) announcements.value = res.obj || []
  } catch (e) {}
}

onMounted(() => {
  fetchAnnouncements()
  fetchTurnstileConfig()
})

onUnmounted(() => {
  if (window.turnstile && turnstileWidgetId.value) {
    window.turnstile.remove(turnstileWidgetId.value)
  }
})
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f6f8fb;
  padding: 20px;
}

/* 背景：网格 + 两团柔和光斑 */
.bg-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(#e2e8f0 1px, transparent 1px),
    linear-gradient(90deg, #e2e8f0 1px, transparent 1px);
  background-size: 48px 48px;
  mask-image: radial-gradient(ellipse at center, #000 10%, transparent 70%);
  -webkit-mask-image: radial-gradient(ellipse at center, #000 10%, transparent 70%);
  opacity: .35;
  pointer-events: none;
}

.bg-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(60px);
  pointer-events: none;
}

.bg-orb-1 {
  width: 520px; height: 520px;
  background: radial-gradient(circle, rgba(59,130,246,.32), transparent 65%);
  top: -140px; right: -160px;
}

.bg-orb-2 {
  width: 420px; height: 420px;
  background: radial-gradient(circle, rgba(14,165,233,.22), transparent 65%);
  bottom: -140px; left: -140px;
}

/* 公告条 */
.announce-bar {
  position: fixed;
  top: 0; left: 0; right: 0;
  z-index: 100;
  background: rgba(255,255,255,.88);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid #eef1f6;
}

.announce-inner {
  max-width: 900px;
  margin: 0 auto;
  padding: 9px 20px;
}

.announce-carousel { height: 24px; line-height: 24px; }

.announce-slide {
  display: flex !important;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 500;
}

.announce-slide.info    { color: #2563eb; }
.announce-slide.warning { color: #b45309; }
.announce-slide.success { color: #047857; }

.announce-ico { font-size: 14px; }

/* Shell 双栏 */
.shell {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 960px;
  display: grid;
  grid-template-columns: 1fr 400px;
  gap: 32px;
  align-items: stretch;
  animation: rise .5s ease-out both;
}

/* 品牌区 */
.brand {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 8px 4px;
}

.brand-top {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-logo {
  width: 40px;
  height: 40px;
  border-radius: 11px;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 20px rgba(59,130,246,.28);
}

.brand-logo svg { width: 22px; height: 22px; }

.brand-logo-small {
  width: 46px;
  height: 46px;
  margin-bottom: 14px;
}

.brand-logo-small svg { width: 24px; height: 24px; }

.brand-name {
  font-family: var(--font-display);
  font-size: 20px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
}

.brand-copy {
  margin: 24px 0 20px;
}

.brand-eyebrow {
  display: inline-block;
  margin-bottom: 14px;
  padding: 3px 10px;
  font-size: 10.5px;
  font-weight: 600;
  letter-spacing: .14em;
  color: #2563eb;
  background: #eff6ff;
  border-radius: 99px;
  font-family: var(--font-mono);
}

.brand-title {
  font-family: var(--font-display);
  margin: 0 0 10px;
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.025em;
  line-height: 1.15;
}

.brand-sub {
  margin: 0;
  font-size: 14px;
  color: #475569;
  line-height: 1.6;
  max-width: 320px;
}

.brand-highlights {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.brand-highlights li {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: #334155;
}

.hl-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: linear-gradient(135deg, #3b82f6, #60a5fa);
  flex-shrink: 0;
}

/* 登录卡 */
.login-card {
  background: rgba(255,255,255,.94);
  backdrop-filter: blur(14px);
  border: 1px solid rgba(226, 232, 240, .8);
  border-radius: 16px;
  padding: 24px 24px 20px;
  box-shadow:
    0 20px 40px -22px rgba(15,23,42,.14),
    0 1px 2px rgba(15,23,42,.03);
}

.login-head {
  margin-bottom: 16px;
}

.login-title {
  font-family: var(--font-display);
  margin: 0 0 4px;
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
}

.login-sub {
  margin: 0;
  font-size: 13px;
  color: #64748b;
}

.login-form :deep(.ant-form-item) { margin-bottom: 12px !important; }

.login-form :deep(.ant-form-item-label) {
  padding-bottom: 2px !important;
}

.login-form :deep(.ant-form-item-label > label) {
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
  height: 22px !important;
}

.login-form :deep(.ant-input-affix-wrapper),
.login-form :deep(.ant-input-affix-wrapper-lg) {
  padding: 7px 12px !important;
  border-radius: 10px !important;
  border: 1px solid #e2e8f0;
  background: #fff;
  transition: border-color .15s, box-shadow .15s;
}

.login-form :deep(.ant-input) {
  font-size: 13.5px !important;
}

.login-form :deep(.ant-input-affix-wrapper:hover) {
  border-color: #cbd5e1;
}

.login-form :deep(.ant-input-affix-wrapper-focused) {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59,130,246,.12);
}

.input-icon {
  color: #94a3b8;
  font-size: 15px;
  margin-right: 4px;
}

.turnstile-item { margin-bottom: 12px; }
.turnstile-container { display: flex; justify-content: center; }

.login-btn {
  height: 40px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: .04em;
  background: #3b82f6 !important;
  border: none;
  box-shadow: 0 6px 16px rgba(59,130,246,.24);
  transition: background-color .15s, box-shadow .15s;
}

.login-btn:hover {
  background: #2563eb !important;
  box-shadow: 0 10px 24px rgba(59,130,246,.32);
}

.login-foot {
  margin-top: 14px;
  text-align: center;
  font-size: 12.5px;
  color: #64748b;
  padding-top: 12px;
  border-top: 1px dashed #eef1f6;
}

.login-foot a {
  margin-left: 6px;
  color: #2563eb;
  cursor: pointer;
  font-weight: 500;
}

.login-foot a:hover { color: #1d4ed8; }

/* 响应式 */
.hide-mobile { display: flex; }
.hide-desktop { display: none; }

@media (max-width: 860px) {
  .shell {
    grid-template-columns: 1fr;
    max-width: 440px;
    gap: 0;
  }
  .hide-mobile { display: none !important; }
  .hide-desktop { display: flex; }
  .login-card { padding: 32px 26px; }
}

@media (max-width: 576px) {
  .login-page { padding: 12px; }
  .login-card { padding: 28px 22px; border-radius: 16px; }
  .login-title { font-size: 20px; }
}

@keyframes rise {
  from { opacity: 0; transform: translateY(10px); }
  to   { opacity: 1; transform: none; }
}
</style>
