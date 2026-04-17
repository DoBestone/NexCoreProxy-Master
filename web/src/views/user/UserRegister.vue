<template>
  <div class="register-page">
    <!-- 背景 -->
    <div class="bg-grid"></div>
    <div class="bg-orb bg-orb-1"></div>
    <div class="bg-orb bg-orb-2"></div>

    <div class="shell">
      <!-- 左：品牌 + 欢迎文案 -->
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
          <span class="brand-eyebrow">JOIN US</span>
          <h2 class="brand-title">开通账号，<br/>几秒钟就能连上全球节点。</h2>
          <p class="brand-sub">
            注册即可获得试用流量，支持多协议订阅与按量计费，
            随时随地切换节点，一个账号搞定所有代理需求。
          </p>
        </div>

        <ul class="brand-steps">
          <li>
            <span class="step-idx mono">01</span>
            <div class="step-body">
              <span class="step-title">创建账号</span>
              <span class="step-sub">用户名 + 密码，邮箱可留空</span>
            </div>
          </li>
          <li>
            <span class="step-idx mono">02</span>
            <div class="step-body">
              <span class="step-title">选购套餐</span>
              <span class="step-sub">按流量或按时长灵活计费</span>
            </div>
          </li>
          <li>
            <span class="step-idx mono">03</span>
            <div class="step-body">
              <span class="step-title">导入订阅</span>
              <span class="step-sub">复制链接到客户端即可使用</span>
            </div>
          </li>
        </ul>
      </aside>

      <!-- 右：注册表单 -->
      <section class="register-card">
        <div class="register-head">
          <div class="brand-logo brand-logo-small hide-desktop">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2"/>
              <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2"/>
              <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <h1 class="register-title">创建账号</h1>
          <p class="register-sub">填写信息，几秒即可完成注册</p>
        </div>

        <a-form
          :model="form"
          @finish="handleRegister"
          layout="vertical"
          class="register-form"
        >
          <a-form-item
            name="username"
            label="用户名"
            :rules="[
              { required: true, message: '请输入用户名' },
              { min: 3, message: '用户名至少 3 个字符' }
            ]"
          >
            <a-input
              v-model:value="form.username"
              placeholder="3 位以上字母 / 数字"
              size="large"
              autocomplete="username"
            >
              <template #prefix><UserOutlined class="input-icon" /></template>
            </a-input>
          </a-form-item>

          <a-form-item
            name="email"
            label="邮箱（可选）"
            :rules="[{ type: 'email', message: '请输入有效邮箱' }]"
          >
            <a-input
              v-model:value="form.email"
              placeholder="用于接收通知，可留空"
              size="large"
              autocomplete="email"
            >
              <template #prefix><MailOutlined class="input-icon" /></template>
            </a-input>
          </a-form-item>

          <a-form-item
            name="password"
            label="密码"
            :rules="[
              { required: true, message: '请输入密码' },
              { min: 6, message: '密码至少 6 位' }
            ]"
          >
            <a-input-password
              v-model:value="form.password"
              placeholder="至少 6 位"
              size="large"
              autocomplete="new-password"
            >
              <template #prefix><LockOutlined class="input-icon" /></template>
            </a-input-password>
          </a-form-item>

          <a-form-item
            name="confirmPassword"
            label="确认密码"
            :rules="[
              { required: true, message: '请再次输入密码' },
              { validator: validateConfirmPassword }
            ]"
          >
            <a-input-password
              v-model:value="form.confirmPassword"
              placeholder="再次输入密码"
              size="large"
              autocomplete="new-password"
            >
              <template #prefix><LockOutlined class="input-icon" /></template>
            </a-input-password>
          </a-form-item>

          <!-- 实时校验指示 -->
          <div class="req-checklist">
            <span class="req" :class="{ ok: usernameOk }">
              <span class="dot"></span>用户名 ≥ 3 位
            </span>
            <span class="req" :class="{ ok: passwordOk }">
              <span class="dot"></span>密码 ≥ 6 位
            </span>
            <span class="req" :class="{ ok: passwordMatch }">
              <span class="dot"></span>两次输入一致
            </span>
          </div>

          <a-form-item name="inviteCode" label="邀请码（可选）">
            <a-input
              v-model:value="form.inviteCode"
              placeholder="有邀请码可获赠流量"
              size="large"
            >
              <template #prefix><GiftOutlined class="input-icon" /></template>
            </a-input>
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
            class="register-btn"
          >
            注册账号
          </a-button>
        </a-form>

        <div class="register-foot">
          <span>已有账号？</span>
          <a @click="$router.push('/user/login')">立即登录 →</a>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  UserOutlined,
  LockOutlined,
  MailOutlined,
  GiftOutlined
} from '@ant-design/icons-vue'
import { register, getTurnstileConfig } from '@/api'

const router = useRouter()
const loading = ref(false)
const turnstileSiteKey = ref('')
const turnstileRef = ref(null)
const turnstileWidgetId = ref(null)
const turnstileToken = ref('')

const form = ref({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  inviteCode: ''
})

const usernameOk = computed(() => form.value.username.length >= 3)
const passwordOk = computed(() => form.value.password.length >= 6)
const passwordMatch = computed(() =>
  passwordOk.value && form.value.password === form.value.confirmPassword
)

const validateConfirmPassword = (rule, value) => {
  if (value !== form.value.password) {
    return Promise.reject('两次密码输入不一致')
  }
  return Promise.resolve()
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
    const res = await getTurnstileConfig()
    if (res.success && res.obj?.siteKey) {
      turnstileSiteKey.value = res.obj.siteKey
      await nextTick()
      renderTurnstile()
    }
  } catch (e) {}
}

const handleRegister = async () => {
  if (turnstileSiteKey.value && !turnstileToken.value) {
    message.warning('请完成人机验证')
    return
  }

  loading.value = true
  try {
    const res = await register({
      username: form.value.username,
      email: form.value.email,
      password: form.value.password,
      inviteCode: form.value.inviteCode,
      turnstileToken: turnstileToken.value
    })
    if (res.success) {
      message.success('注册成功，请登录')
      router.push('/user/login')
    } else {
      message.error(res.msg || '注册失败')
      if (window.turnstile && turnstileWidgetId.value) {
        window.turnstile.reset(turnstileWidgetId.value)
        turnstileToken.value = ''
      }
    }
  } catch (e) {
    message.error('注册失败')
    if (window.turnstile && turnstileWidgetId.value) {
      window.turnstile.reset(turnstileWidgetId.value)
      turnstileToken.value = ''
    }
  } finally {
    loading.value = false
  }
}

onMounted(fetchTurnstileConfig)
onUnmounted(() => {
  if (window.turnstile && turnstileWidgetId.value) {
    window.turnstile.remove(turnstileWidgetId.value)
  }
})
</script>

<style scoped>
.register-page {
  min-height: 100vh;
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f6f8fb;
  padding: 28px 20px;
}

/* 背景 */
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

/* Shell */
.shell {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 1000px;
  display: grid;
  grid-template-columns: 1fr 420px;
  gap: 36px;
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

.brand-copy { margin: 24px 0 20px; }

.brand-eyebrow {
  display: inline-block;
  margin-bottom: 14px;
  padding: 3px 10px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  font-weight: 600;
  letter-spacing: .14em;
  color: #2563eb;
  background: #eff6ff;
  border-radius: 99px;
}

.brand-title {
  font-family: var(--font-display);
  margin: 0 0 10px;
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.025em;
  line-height: 1.18;
}

.brand-sub {
  margin: 0;
  font-size: 14px;
  color: #475569;
  line-height: 1.65;
  max-width: 360px;
}

/* 步骤列表 */
.brand-steps {
  list-style: none;
  margin: 0;
  padding: 12px 0 0;
  border-top: 1px dashed #e2e8f0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.brand-steps li {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.step-idx {
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  border-radius: 9px;
  background: #eff6ff;
  color: #2563eb;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 12.5px;
  font-variant-numeric: tabular-nums;
  letter-spacing: .02em;
}

.step-body { display: flex; flex-direction: column; gap: 2px; padding-top: 4px; }

.step-title {
  font-size: 13.5px;
  font-weight: 600;
  color: #1e293b;
}

.step-sub {
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.5;
}

/* 注册卡 */
.register-card {
  background: rgba(255,255,255,.94);
  backdrop-filter: blur(14px);
  border: 1px solid rgba(226, 232, 240, .8);
  border-radius: 16px;
  padding: 22px 22px 20px;
  box-shadow:
    0 20px 40px -22px rgba(15,23,42,.14),
    0 1px 2px rgba(15,23,42,.03);
}

.register-head { margin-bottom: 16px; }

.register-title {
  font-family: var(--font-display);
  margin: 0 0 4px;
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
}

.register-sub {
  margin: 0;
  font-size: 13px;
  color: #64748b;
}

.register-form :deep(.ant-form-item) { margin-bottom: 10px !important; }
.register-form :deep(.ant-form-item-label) {
  padding-bottom: 2px !important;
}

.register-form :deep(.ant-form-item-label > label) {
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
  height: 22px !important;
}

.register-form :deep(.ant-input-affix-wrapper),
.register-form :deep(.ant-input-affix-wrapper-lg) {
  padding: 7px 12px !important;
  border-radius: 10px !important;
  border: 1px solid #e2e8f0;
  background: #fff;
  transition: border-color .15s, box-shadow .15s;
}

.register-form :deep(.ant-input) {
  font-size: 13.5px !important;
}

.register-form :deep(.ant-input-affix-wrapper:hover) { border-color: #cbd5e1; }

.register-form :deep(.ant-input-affix-wrapper-focused) {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59,130,246,.12);
}

.input-icon {
  color: #94a3b8;
  font-size: 15px;
  margin-right: 4px;
}

/* 实时校验 */
.req-checklist {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  padding: 6px 10px;
  margin: -2px 0 10px;
  background: #f8fafc;
  border-radius: 8px;
  font-size: 11.5px;
}

.req {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #94a3b8;
  transition: color .15s;
}

.req .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #cbd5e1;
  transition: background-color .15s;
}

.req.ok { color: #047857; }
.req.ok .dot { background: #16a34a; }

.turnstile-item { margin-bottom: 12px; }
.turnstile-container { display: flex; justify-content: center; }

.register-btn {
  height: 40px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: .04em;
  background: #3b82f6 !important;
  border: none;
  box-shadow: 0 6px 16px rgba(59,130,246,.24);
  transition: background-color .15s, box-shadow .15s;
  margin-top: 2px;
}

.register-btn:hover {
  background: #2563eb !important;
  box-shadow: 0 10px 24px rgba(59,130,246,.32);
}

.register-foot {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px dashed #eef1f6;
  text-align: center;
  font-size: 12.5px;
  color: #64748b;
}

.register-foot a {
  margin-left: 6px;
  color: #2563eb;
  cursor: pointer;
  font-weight: 500;
}

.register-foot a:hover { color: #1d4ed8; }

/* 响应式 */
.hide-mobile { display: flex; }
.hide-desktop { display: none; }

@media (max-width: 900px) {
  .shell {
    grid-template-columns: 1fr;
    max-width: 460px;
    gap: 0;
  }
  .hide-mobile { display: none !important; }
  .hide-desktop { display: flex; }
  .register-card { padding: 28px 24px 24px; }
}

@media (max-width: 576px) {
  .register-page { padding: 12px; }
  .register-card { padding: 26px 20px 22px; border-radius: 16px; }
  .register-title { font-size: 20px; }
}

@keyframes rise {
  from { opacity: 0; transform: translateY(10px); }
  to   { opacity: 1; transform: none; }
}
</style>
