<template>
  <div class="settings-page">
    <div class="settings-grid">
      <!-- 左侧：账户概览 -->
      <section class="panel panel-profile">
        <div class="profile-top">
          <div class="profile-avatar">{{ avatarLetter }}</div>
          <div class="profile-id">
            <h3 class="profile-name">{{ userInfo.username || '—' }}</h3>
            <span class="profile-mail mono">{{ userInfo.email || '未设置邮箱' }}</span>
          </div>
        </div>

        <div class="profile-stats">
          <div class="stat">
            <span class="stat-label">账户余额</span>
            <span class="stat-value stat-emphasis mono">
              <span class="stat-currency">¥</span>{{ (userInfo.balance || 0).toFixed(2) }}
            </span>
          </div>
          <div class="stat">
            <span class="stat-label">注册时间</span>
            <span class="stat-value mono">{{ formatDate(userInfo.createdAt) }}</span>
          </div>
        </div>

        <dl class="info-list">
          <div class="info-item">
            <dt>流量限额</dt>
            <dd class="mono">{{ formatTraffic(userInfo.trafficLimit) || '无限制' }}</dd>
          </div>
          <div class="info-item">
            <dt>已用流量</dt>
            <dd class="mono">{{ formatTraffic(userInfo.trafficUsed) }}</dd>
          </div>
          <div class="info-item">
            <dt>到期时间</dt>
            <dd :class="['mono', { danger: isExpired }]">
              {{ userInfo.expireAt ? formatDate(userInfo.expireAt) : '永久有效' }}
            </dd>
          </div>
        </dl>
      </section>

      <!-- 右侧：修改密码 -->
      <section class="panel panel-password">
        <div class="panel-head">
          <h3 class="panel-title">
            <LockOutlined /> 修改密码
          </h3>
        </div>

        <a-form :model="passwordForm" layout="vertical" class="password-form">
          <a-form-item label="当前密码" required>
            <a-input-password
              v-model:value="passwordForm.oldPassword"
              placeholder="输入当前密码"
              autocomplete="current-password"
            />
          </a-form-item>
          <a-form-item label="新密码" required>
            <a-input-password
              v-model:value="passwordForm.newPassword"
              placeholder="至少 6 位"
              autocomplete="new-password"
            />
          </a-form-item>
          <a-form-item label="确认新密码" required>
            <a-input-password
              v-model:value="passwordForm.confirmPassword"
              placeholder="再次输入新密码"
              autocomplete="new-password"
            />
          </a-form-item>

          <div class="password-tips">
            <span class="dot" :class="passwordForm.newPassword.length >= 6 ? 'dot-ok' : 'dot-muted'"></span>
            至少 6 位
            <span class="sep">·</span>
            <span class="dot" :class="passwordsMatch ? 'dot-ok' : 'dot-muted'"></span>
            两次输入一致
          </div>

          <a-button
            type="primary"
            block
            size="large"
            :loading="changing"
            :disabled="!canSubmit"
            @click="changePassword"
          >
            <SaveOutlined /> 保存修改
          </a-button>
        </a-form>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { LockOutlined, SaveOutlined } from '@ant-design/icons-vue'
import { getUserInfo, updatePassword } from '@/api'

const changing = ref(false)
const userInfo = ref({})
const passwordForm = ref({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const avatarLetter = computed(() => (userInfo.value.username || 'U').charAt(0).toUpperCase())

const isExpired = computed(() => {
  if (!userInfo.value.expireAt) return false
  return new Date(userInfo.value.expireAt) < new Date()
})

const passwordsMatch = computed(() =>
  passwordForm.value.newPassword.length >= 6 &&
  passwordForm.value.newPassword === passwordForm.value.confirmPassword
)

const canSubmit = computed(() =>
  passwordForm.value.oldPassword.length > 0 &&
  passwordForm.value.newPassword.length >= 6 &&
  passwordsMatch.value
)

const formatTraffic = (bytes) => {
  if (!bytes) return ''
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let n = bytes
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++ }
  return `${n.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

const formatDate = (v) => {
  if (!v) return '—'
  const d = new Date(v)
  if (isNaN(d)) return '—'
  const pad = (x) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
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
    message.warning('密码长度至少 6 位')
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

onMounted(fetchUserInfo)
</script>

<style scoped>
.settings-page { animation: rise .4s ease-out both; }

.settings-grid {
  display: grid;
  grid-template-columns: 380px 1fr;
  gap: 20px;
  align-items: start;
}

.panel {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 12px;
  padding: 16px 18px;
}

.panel-head { margin-bottom: 18px; }
.panel-title {
  font-family: var(--font-display);
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: -0.01em;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.panel-title .anticon { color: #3b82f6; font-size: 16px; }

/* ============================================================
   账户概览
   ============================================================ */
.profile-top {
  display: flex;
  align-items: center;
  gap: 14px;
  padding-bottom: 18px;
  border-bottom: 1px solid #f1f5f9;
  margin-bottom: 16px;
}

.profile-avatar {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-display);
  font-size: 20px;
  font-weight: 700;
  box-shadow: 0 6px 16px rgba(59,130,246,.24);
  flex-shrink: 0;
}

.profile-id { display: flex; flex-direction: column; gap: 3px; min-width: 0; }

.profile-name {
  font-family: var(--font-display);
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.01em;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-mail {
  font-size: 12.5px;
  color: #64748b;
  font-variant-numeric: tabular-nums;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-stats {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 16px;
}

.stat {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  background: #f8fafc;
  border: 1px solid #eef1f6;
  border-radius: 10px;
}

.stat-label { font-size: 11.5px; color: #94a3b8; letter-spacing: .02em; }

.stat-value {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
  font-variant-numeric: tabular-nums;
}

.stat-emphasis {
  font-family: var(--font-display);
  font-size: 19px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
}

.stat-currency {
  font-size: 13px;
  font-weight: 600;
  color: #64748b;
  margin-right: 1px;
}

.info-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 11px 0;
  border-bottom: 1px dashed #eef1f6;
  font-size: 13px;
}

.info-item:last-child { border-bottom: none; }

.info-item dt { color: #64748b; margin: 0; }
.info-item dd { margin: 0; color: #1e293b; font-weight: 500; font-variant-numeric: tabular-nums; }
.info-item dd.danger { color: #dc2626; }

/* ============================================================
   密码表单
   ============================================================ */
.password-form :deep(.ant-form-item) { margin-bottom: 16px; }
.password-form :deep(.ant-form-item-label > label) {
  font-size: 12.5px;
  color: #475569;
  font-weight: 500;
}

.password-tips {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 12px;
  color: #64748b;
  margin: -8px 0 16px;
}

.dot { width: 6px; height: 6px; border-radius: 50%; display: inline-block; }
.dot-ok { background: #16a34a; }
.dot-muted { background: #cbd5e1; }

.sep { color: #cbd5e1; padding: 0 4px; }

/* ============================================================
   响应式
   ============================================================ */
@media (max-width: 992px) {
  .settings-grid { grid-template-columns: 1fr; }
}

@media (max-width: 576px) {
  .panel { padding: 18px; }
  .profile-stats { grid-template-columns: 1fr; }
}

@keyframes rise {
  from { opacity: 0; transform: translateY(6px); }
  to   { opacity: 1; transform: none; }
}
</style>
