<template>
  <div class="certs-page">
    <!-- ACME 账户配置卡片（紧凑） -->
    <div class="acme-panel">
      <div class="panel-head">
        <div class="title">ACME 账户</div>
        <div class="status">
          <span class="dot-line">
            <span :class="['dot', acme.configured ? 'on' : 'off']" />
            {{ acme.configured ? '已配置 Cloudflare Token' : '尚未配置' }}
          </span>
          <span v-if="acme.hasRegistered" class="ok">已注册</span>
        </div>
      </div>
      <a-form :model="acmeForm" layout="inline" class="acme-form" @finish="onSaveAcme">
        <a-form-item label="账户邮箱">
          <a-input v-model:value="acmeForm.email" placeholder="admin@yourdomain.com" style="width: 240px" />
        </a-form-item>
        <a-form-item label="Cloudflare API Token">
          <a-input-password v-model:value="acmeForm.cloudflareToken" placeholder="只用于 DNS-01 验证" style="width: 320px" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" html-type="submit" :loading="savingAcme">保存</a-button>
        </a-form-item>
      </a-form>
      <p class="tip">Token 权限只需 Zone:DNS:Edit 即可，不要给 Account 级权限。</p>
    </div>

    <!-- 证书签发工具条 + 列表 -->
    <div class="page-toolbar">
      <div class="issue-inline">
        <a-input v-model:value="newDomain" placeholder="输入域名签发，如 node1.example.com" style="width: 320px" class="mono-input" />
        <a-button type="primary" :loading="issuing" @click="onIssue">
          <template #icon><ThunderboltOutlined /></template>
          立即签发
        </a-button>
      </div>
      <a-button @click="fetchAll">
        <template #icon><ReloadOutlined /></template>
        刷新
      </a-button>
    </div>

    <a-table
      class="cert-table"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="{ pageSize: 20, showSizeChanger: false }"
      row-key="id"
      size="small"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'domain'">
          <span class="mono dom">{{ record.domain }}</span>
        </template>
        <template v-else-if="column.key === 'status'">
          <span :class="['st-tag', record.status]">{{ statusText(record.status) }}</span>
        </template>
        <template v-else-if="column.key === 'issuedAt'">
          <span class="mono dim">{{ fmt(record.issuedAt) }}</span>
        </template>
        <template v-else-if="column.key === 'expiresAt'">
          <span class="mono">{{ fmt(record.expiresAt) }}</span>
          <span :class="['days', expiryClass(record.expiresAt)]"> · {{ daysToExpiry(record.expiresAt) }}d</span>
        </template>
        <template v-else-if="column.key === 'actions'">
          <a class="link" @click="onIssueAgain(record.domain)">续签</a>
          <a-popconfirm title="确定删除？吊销不会自动通知 ACME。" @confirm="onDelete(record.id)">
            <a class="link danger">删除</a>
          </a-popconfirm>
        </template>
      </template>
    </a-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { ThunderboltOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import {
  listCerts, issueCert, deleteCert,
  getAcmeSettings, updateAcmeSettings,
} from '@/api'

const loading = ref(false)
const issuing = ref(false)
const savingAcme = ref(false)
const rows = ref([])
const newDomain = ref('')
const acme = ref({ email: '', configured: false, hasRegistered: false })
const acmeForm = ref({ email: '', cloudflareToken: '' })

const columns = [
  { title: '域名', key: 'domain' },
  { title: '状态', key: 'status', width: 120 },
  { title: '签发时间', key: 'issuedAt', width: 180 },
  { title: '到期', key: 'expiresAt', width: 220 },
  { title: '操作', key: 'actions', width: 140 },
]

const statusText = (s) => ({ issued: '已签发', failed: '失败', pending: '签发中' }[s] || s)
const fmt = (ts) => ts ? new Date(ts).toLocaleString('zh-CN', { hour12: false }) : '—'
const daysToExpiry = (ts) => {
  if (!ts) return 0
  return Math.max(0, Math.round((new Date(ts) - new Date()) / 86400000))
}
const expiryClass = (ts) => {
  const d = daysToExpiry(ts)
  if (d <= 7)  return 'critical'
  if (d <= 30) return 'warn'
  return 'ok'
}

async function fetchAll() {
  loading.value = true
  try {
    const [c, a] = await Promise.all([listCerts(), getAcmeSettings()])
    rows.value = c.obj || []
    acme.value = a.obj || acme.value
    if (!acmeForm.value.email) acmeForm.value.email = acme.value.email || ''
  } catch {
    message.error('加载失败')
  } finally {
    loading.value = false
  }
}

async function onSaveAcme() {
  if (!acmeForm.value.email) return message.warning('请填账户邮箱')
  savingAcme.value = true
  try {
    await updateAcmeSettings(acmeForm.value)
    acmeForm.value.cloudflareToken = '' // 不回显
    message.success('已保存')
    fetchAll()
  } catch (e) {
    message.error(e?.msg || '保存失败')
  } finally {
    savingAcme.value = false
  }
}

async function onIssue() {
  const d = newDomain.value.trim()
  if (!d) return message.warning('请输入域名')
  issuing.value = true
  try {
    await issueCert(d)
    message.success('已签发')
    newDomain.value = ''
    fetchAll()
  } catch (e) {
    message.error(e?.msg || '签发失败，请检查 Cloudflare Token / 域名归属')
  } finally {
    issuing.value = false
  }
}
async function onIssueAgain(d) {
  issuing.value = true
  try { await issueCert(d); message.success('续签完成'); fetchAll() }
  catch (e) { message.error(e?.msg || '续签失败') }
  finally { issuing.value = false }
}
async function onDelete(id) {
  try { await deleteCert(id); message.success('已删除'); fetchAll() }
  catch { message.error('删除失败') }
}

onMounted(fetchAll)
</script>

<style scoped>
.certs-page { --font-mono: ui-monospace, 'JetBrains Mono', 'SF Mono', monospace; }

/* === ACME 账户卡片 === */
.acme-panel {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 16px 18px;
  margin-bottom: 16px;
}
.panel-head {
  display: flex; align-items: center; justify-content: space-between;
  padding-bottom: 12px; margin-bottom: 12px;
  border-bottom: 1px dashed #e2e8f0;
}
.panel-head .title { font-size: 14px; font-weight: 600; color: #1e293b; }
.panel-head .status { display: inline-flex; align-items: center; gap: 12px; font-size: 12px; }
.panel-head .ok { color: #16a34a; font-weight: 500; }

.acme-form { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.acme-form :deep(.ant-form-item) { margin-bottom: 0; margin-right: 0; }
.acme-form :deep(.ant-form-item-label > label) { font-size: 12px; color: #64748b; height: 26px; }

.tip { margin: 10px 0 0; color: #94a3b8; font-size: 11.5px; }

/* === 工具条 === */
.page-toolbar {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 12px; gap: 12px;
}
.issue-inline { display: flex; gap: 8px; align-items: center; }

/* === 表格紧凑 === */
.cert-table :deep(.ant-table-thead > tr > th) {
  padding: 10px 14px; font-size: 12px; font-weight: 600;
  background: #f8fafc; color: #1e293b;
}
.cert-table :deep(.ant-table-tbody > tr > td) {
  padding: 10px 14px; font-size: 13px; line-height: 1.5;
}
.cert-table :deep(.ant-table-tbody > tr:hover > td) { background: #f1f5f9; }

.dom { color: #1e293b; font-weight: 500; }
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.mono.dim { color: #94a3b8; }
.mono-input :deep(input) { font-family: var(--font-mono); font-size: 12.5px; }

.st-tag {
  display: inline-block; padding: 2px 8px; border-radius: 6px;
  font-size: 11.5px; font-weight: 600;
}
.st-tag.issued  { background: #dcfce7; color: #15803d; }
.st-tag.failed  { background: #fee2e2; color: #b91c1c; }
.st-tag.pending { background: #e0e7ff; color: #4338ca; }

.dot-line { display: inline-flex; align-items: center; gap: 6px; color: #475569; font-size: 12px; }
.dot { width: 6px; height: 6px; border-radius: 50%; background: #cbd5e1; }
.dot.on  { background: #16a34a; }
.dot.off { background: #cbd5e1; }

.days { font-size: 11.5px; }
.days.ok       { color: #16a34a; }
.days.warn     { color: #d97706; }
.days.critical { color: #dc2626; font-weight: 600; }

.link { font-size: 12.5px; color: #2563eb; margin-right: 12px; cursor: pointer; }
.link:last-child { margin-right: 0; }
.link.danger { color: #dc2626; }
.link:hover { text-decoration: underline; }
</style>
