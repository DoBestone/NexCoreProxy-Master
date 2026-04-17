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

    <!-- 背景 -->
    <div class="bg-grid"></div>
    <div class="bg-orb bg-orb-1"></div>
    <div class="bg-orb bg-orb-2"></div>

    <div class="shell">
      <!-- 左：品牌 + 指示 -->
      <aside class="brand hide-mobile">
        <div class="brand-top">
          <div class="brand-logo">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <div class="brand-id">
            <span class="brand-name">NexCore</span>
            <span class="brand-sub">Admin Console</span>
          </div>
        </div>

        <div class="brand-copy">
          <span class="brand-eyebrow">
            <SafetyOutlined /> ADMIN · RESTRICTED
          </span>
          <h2 class="brand-title">仅限授权人员访问。</h2>
          <p class="brand-body">
            控制台用于节点部署、用户管理、套餐与订单配置。
            所有操作会被记录并审计。
          </p>
        </div>

        <div class="brand-meta">
          <div class="meta-row">
            <span class="meta-k">当前时间</span>
            <span class="meta-v mono">{{ clock }}</span>
          </div>
          <div class="meta-row">
            <span class="meta-k">会话环境</span>
            <span class="meta-v mono">PROD · TLS</span>
          </div>
        </div>
      </aside>

      <!-- 右：表单 -->
      <section class="login-card">
        <div class="login-head">
          <div class="brand-logo brand-logo-small hide-desktop">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2"/>
              <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2"/>
              <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>

          <h1 class="login-title">管理员登录</h1>
          <p class="login-sub">
            <span class="sub-tag">
              <SafetyOutlined /> ADMIN
            </span>
            限管理员使用此入口
          </p>
        </div>

        <a-form
          :model="form"
          @finish="handleLogin"
          layout="vertical"
          class="login-form"
        >
          <a-form-item
            name="username"
            label="管理员账号"
            :rules="[{ required: true, message: '请输入用户名' }]"
          >
            <a-input
              v-model:value="form.username"
              placeholder="输入管理员账号"
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
            安全登录
          </a-button>
        </a-form>

        <div class="login-foot">
          <span class="foot-dot"></span>
          管理员登录入口 · 非管理员请返回
          <a @click="$router.push('/user/login')">用户登录 →</a>
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
  SafetyOutlined,
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
const clock = ref('')
let clockTimer = null

const form = ref({ username: '', password: '' })

const tickClock = () => {
  const d = new Date()
  const pad = (x) => String(x).padStart(2, '0')
  clock.value = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

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
  } catch (e) {}
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
      localStorage.setItem('admin_token', res.obj.token)
      localStorage.setItem('admin_username', res.obj.username || form.value.username)
      message.success('登录成功')
      router.push('/admin/dashboard')
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
  tickClock()
  clockTimer = setInterval(tickClock, 1000)
})

onUnmounted(() => {
  if (window.turnstile && turnstileWidgetId.value) {
    window.turnstile.remove(turnstileWidgetId.value)
  }
  if (clockTimer) clearInterval(clockTimer)
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
  background: #0f172a;
  background-image:
    radial-gradient(ellipse at 20% 0%, rgba(59,130,246,.28), transparent 55%),
    radial-gradient(ellipse at 80% 100%, rgba(14,165,233,.18), transparent 55%),
    linear-gradient(180deg, #0f172a 0%, #0a1120 100%);
  padding: 20px;
  color: #cbd5e1;
}

/* 网格背景 */
.bg-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(148,163,184,.08) 1px, transparent 1px),
    linear-gradient(90deg, rgba(148,163,184,.08) 1px, transparent 1px);
  background-size: 52px 52px;
  mask-image: radial-gradient(ellipse at center, #000 20%, transparent 75%);
  -webkit-mask-image: radial-gradient(ellipse at center, #000 20%, transparent 75%);
  pointer-events: none;
}

.bg-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  pointer-events: none;
}

.bg-orb-1 {
  width: 520px; height: 520px;
  background: radial-gradient(circle, rgba(59,130,246,.35), transparent 65%);
  top: -180px; right: -180px;
}

.bg-orb-2 {
  width: 420px; height: 420px;
  background: radial-gradient(circle, rgba(14,165,233,.25), transparent 65%);
  bottom: -160px; left: -160px;
}

/* 公告条 */
.announce-bar {
  position: fixed;
  top: 0; left: 0; right: 0;
  z-index: 100;
  background: rgba(15,23,42,.7);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid rgba(148,163,184,.15);
}

.announce-inner { max-width: 900px; margin: 0 auto; padding: 9px 20px; }
.announce-carousel { height: 24px; line-height: 24px; }

.announce-slide {
  display: flex !important;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 500;
}

.announce-slide.info    { color: #93c5fd; }
.announce-slide.warning { color: #fbbf24; }
.announce-slide.success { color: #6ee7b7; }

.announce-ico { font-size: 14px; }

/* Shell */
.shell {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 960px;
  display: grid;
  grid-template-columns: 1fr 400px;
  gap: 40px;
  align-items: stretch;
  animation: rise .5s ease-out both;
}

/* 品牌 */
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
  width: 42px;
  height: 42px;
  border-radius: 11px;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 20px rgba(59,130,246,.35);
  flex-shrink: 0;
}

.brand-logo svg { width: 22px; height: 22px; }

.brand-logo-small {
  width: 46px;
  height: 46px;
  margin-bottom: 14px;
  border-radius: 12px;
}

.brand-logo-small svg { width: 24px; height: 24px; }

.brand-id { display: flex; flex-direction: column; gap: 2px; }

.brand-name {
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 700;
  color: #f8fafc;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.brand-sub {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: #64748b;
  letter-spacing: .16em;
  text-transform: uppercase;
}

.brand-copy { margin: 24px 0 20px; }

.brand-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 18px;
  padding: 4px 10px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  font-weight: 600;
  letter-spacing: .14em;
  color: #fca5a5;
  background: rgba(220,38,38,.12);
  border: 1px solid rgba(220,38,38,.25);
  border-radius: 99px;
}

.brand-eyebrow .anticon { font-size: 11px; }

.brand-title {
  font-family: var(--font-display);
  margin: 0 0 10px;
  font-size: 28px;
  font-weight: 700;
  color: #f8fafc;
  letter-spacing: -0.025em;
  line-height: 1.15;
}

.brand-body {
  margin: 0;
  font-size: 14px;
  color: #94a3b8;
  line-height: 1.65;
  max-width: 360px;
}

.brand-meta {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 0 0;
  border-top: 1px solid rgba(148,163,184,.12);
}

.meta-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12.5px;
}

.meta-k { color: #64748b; letter-spacing: .02em; }

.meta-v {
  color: #cbd5e1;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  letter-spacing: .02em;
}

/* 登录卡 */
.login-card {
  background: rgba(255,255,255,.97);
  border: 1px solid rgba(226, 232, 240, .8);
  border-radius: 16px;
  padding: 24px 24px 20px;
  box-shadow:
    0 32px 64px -28px rgba(0,0,0,.45),
    0 1px 3px rgba(0,0,0,.06);
}

.login-head { margin-bottom: 16px; }

.login-title {
  font-family: var(--font-display);
  margin: 0 0 8px;
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
}

.login-sub {
  margin: 0;
  font-size: 12.5px;
  color: #64748b;
  display: flex;
  align-items: center;
  gap: 8px;
}

.sub-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 5px;
  background: #fef2f2;
  color: #b91c1c;
  font-family: var(--font-mono);
  font-size: 10.5px;
  font-weight: 700;
  letter-spacing: .04em;
}

.sub-tag .anticon { font-size: 10px; }

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
  letter-spacing: .08em;
  background: #0f172a !important;
  border: none;
  box-shadow: 0 8px 18px rgba(15,23,42,.22);
  transition: background-color .15s, box-shadow .15s;
}

.login-btn:hover {
  background: #1e293b !important;
  box-shadow: 0 12px 28px rgba(15,23,42,.28);
}

.login-foot {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px dashed #eef1f6;
  font-size: 12px;
  color: #94a3b8;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.foot-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: #16a34a;
  box-shadow: 0 0 8px rgba(22,163,74,.6);
}

.login-foot a {
  color: #2563eb;
  cursor: pointer;
  font-weight: 500;
  margin-left: auto;
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
  .login-foot { flex-direction: column; align-items: flex-start; }
  .login-foot a { margin-left: 0; }
}

@keyframes rise {
  from { opacity: 0; transform: translateY(10px); }
  to   { opacity: 1; transform: none; }
}
</style>
