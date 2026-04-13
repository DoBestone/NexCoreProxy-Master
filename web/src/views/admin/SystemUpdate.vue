<template>
  <div class="system-update-page">
    <div class="page-header">
      <h1 class="page-title">
        <CloudSyncOutlined class="title-icon" />
        系统更新
      </h1>
      <p class="page-desc">在线检测更新、查看更新日志、配置更新代理</p>
    </div>

    <!-- 版本信息栏 -->
    <div class="version-bar">
      <div class="version-info">
        <span class="version-label">当前版本</span>
        <span class="version-val">{{ currentVersion || '-' }}</span>
      </div>
      <div class="version-info" v-if="latestVersion">
        <span class="version-label">最新版本</span>
        <span class="version-val">
          {{ latestVersion }}
          <a-tag v-if="updateStatus === 'available'" color="red" style="margin-left:6px">有新版本</a-tag>
          <a-tag v-else-if="updateStatus === 'latest'" color="green" style="margin-left:6px">已是最新</a-tag>
        </span>
      </div>
      <div style="margin-left:auto;display:flex;gap:8px;">
        <a-button :loading="checking" @click="checkUpdate" size="small" type="primary" ghost>
          <template #icon><ReloadOutlined /></template>
          {{ checking ? '检测中...' : '检测更新' }}
        </a-button>
        <a-button v-if="updateStatus === 'available'" type="primary" size="small" :loading="updating" @click="doUpdate">
          立即更新
        </a-button>
      </div>
    </div>

    <!-- 错误提示 -->
    <a-alert v-if="updateStatus === 'error'" type="error" :message="updateError || '检测更新失败'" show-icon style="margin-bottom:16px" />

    <!-- Tabs -->
    <a-card class="tabs-card">
      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="proxy" tab="代理配置">
          <div class="tab-inner">
            <div class="tab-header">
              <span></span>
              <a-space>
                <a-switch v-model:checked="configForm.enabled" checked-children="启用" un-checked-children="停用" />
                <a-button :loading="configSaving" @click="saveProxyConfig" type="primary" size="small">保存配置</a-button>
              </a-space>
            </div>
            <a-form :model="configForm" layout="vertical">
              <a-row :gutter="20">
                <a-col :span="12">
                  <a-form-item label="代理地址">
                    <a-input v-model:value="configForm.proxy_url" placeholder="https://license.nexcores.net" />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="仓库令牌">
                    <a-input-password v-model:value="configForm.repo_token" placeholder="nxr_xxxxxxxxxx" />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="仓库拥有者">
                    <a-input v-model:value="configForm.owner" placeholder="仓库拥有者" />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="仓库名称">
                    <a-input v-model:value="configForm.repo" placeholder="仓库名称" />
                  </a-form-item>
                </a-col>
              </a-row>
            </a-form>
          </div>
        </a-tab-pane>

        <a-tab-pane key="changelog" tab="更新日志">
          <div class="tab-inner">
            <div class="tab-header">
              <span></span>
              <a-button :loading="changelogLoading" @click="loadChangelog" size="small">
                <template #icon><ReloadOutlined /></template>
                刷新
              </a-button>
            </div>
            <div v-if="changelog.length" class="changelog-list">
              <div v-for="release in changelog" :key="release.tag_name" class="changelog-item" :class="{ current: release.tag_name === currentVersion }">
                <div class="changelog-item-header" @click="toggleExpand(release.tag_name)">
                  <div class="changelog-item-left">
                    <RightOutlined class="expand-arrow" :class="{ rotated: expandedTags.has(release.tag_name) }" />
                    <a-tag :color="release.tag_name === currentVersion ? 'blue' : 'default'" size="small">{{ release.tag_name }}</a-tag>
                    <a-tag v-if="release.tag_name === currentVersion" color="green" size="small">当前版本</a-tag>
                    <span class="changelog-summary">{{ extractSummary(release) }}</span>
                  </div>
                  <span class="changelog-date">{{ formatDate(release.published_at) }}</span>
                </div>
                <div v-show="expandedTags.has(release.tag_name)" class="changelog-body-wrap">
                  <div v-if="release.body" class="changelog-body" v-html="renderMarkdown(release.body)"></div>
                  <div v-else class="changelog-body empty">暂无更新说明</div>
                </div>
              </div>
            </div>
            <a-empty v-else description="点击刷新加载更新日志" />
          </div>
        </a-tab-pane>
      </a-tabs>
    </a-card>

    <!-- 更新进度弹窗 -->
    <a-modal v-model:open="showProgressDialog" :closable="updateProgress !== 'updating'" :maskClosable="false" :footer="null" width="420px" title="系统更新" centered>
      <!-- 更新中 -->
      <div v-if="updateProgress === 'updating'" class="update-progress-content">
        <div class="spinner-wrap"><div class="update-spinner"></div></div>
        <h3 class="progress-title">正在更新...</h3>
        <p class="progress-hint">请勿关闭页面</p>
        <div class="update-steps">
          <div :class="['step', { done: updateStep >= 1 }]">
            <CheckCircleFilled v-if="updateStep >= 1" style="color:#16a34a" />
            <LoadingOutlined v-else style="color:#94a3b8" />
            1. 下载更新
          </div>
          <div :class="['step', { done: updateStep >= 2 }]">
            <CheckCircleFilled v-if="updateStep >= 2" style="color:#16a34a" />
            <LoadingOutlined v-else style="color:#94a3b8" />
            2. 替换文件
          </div>
          <div :class="['step', { done: updateStep >= 3 }]">
            <CheckCircleFilled v-if="updateStep >= 3" style="color:#16a34a" />
            <LoadingOutlined v-else style="color:#94a3b8" />
            3. 重启服务
          </div>
        </div>
      </div>
      <!-- 更新成功 -->
      <div v-else-if="updateProgress === 'done'" class="update-progress-content">
        <CheckCircleFilled style="font-size:56px;color:#16a34a" />
        <h3 class="progress-title">更新完成！</h3>
        <p class="progress-hint">页面将在 3 秒后自动刷新</p>
      </div>
      <!-- 更新失败 -->
      <div v-else-if="updateProgress === 'error'" class="update-progress-content">
        <CloseCircleFilled style="font-size:56px;color:#dc2626" />
        <h3 class="progress-title">{{ updateErrorMsg }}</h3>
        <p class="progress-hint">{{ updateErrorHint }}</p>
        <div v-if="updateErrorCmd" class="update-cmd-box"><code>{{ updateErrorCmd }}</code></div>
        <a-button @click="showProgressDialog = false" style="margin-top:16px">关闭</a-button>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, onDeactivated } from 'vue'
import { message } from 'ant-design-vue'
import {
  CloudSyncOutlined, ReloadOutlined, RightOutlined,
  CheckCircleFilled, CloseCircleFilled, LoadingOutlined
} from '@ant-design/icons-vue'
import { systemUpdate } from '@/api'

const activeTab = ref('proxy')
const currentVersion = ref('')
const latestVersion = ref('')
const updateStatus = ref('')
const updateError = ref('')
const checking = ref(false)
const updating = ref(false)

const configForm = ref({ proxy_url: '', repo_token: '', owner: '', repo: '', enabled: false })
const configSaving = ref(false)

const changelog = ref([])
const changelogLoading = ref(false)
const expandedTags = ref(new Set())

const showProgressDialog = ref(false)

onDeactivated(() => { showProgressDialog.value = false })
const updateProgress = ref('')
const updateStep = ref(0)
const updateErrorMsg = ref('')
const updateErrorHint = ref('')
const updateErrorCmd = ref('')

const toggleExpand = (tag) => {
  const s = new Set(expandedTags.value)
  s.has(tag) ? s.delete(tag) : s.add(tag)
  expandedTags.value = s
}

const extractSummary = (release) => {
  if (release.name && release.name !== release.tag_name) return release.name.slice(0, 80)
  if (!release.body) return ''
  for (const l of release.body.split('\n')) {
    const t = l.trim()
    if (t.startsWith('* ') || t.startsWith('- ')) {
      const text = t.replace(/^[*\-]\s*/, '').replace(/\s+by\s+@\S+\s+in\s+\S+$/i, '').trim()
      if (text && !/^(Full Changelog|http)/i.test(text)) return text.slice(0, 80)
    }
  }
  return ''
}

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

const renderMarkdown = (text) => {
  if (!text) return ''
  return text
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    .replace(/^### (.+)$/gm, '<h4>$1</h4>')
    .replace(/^## (.+)$/gm, '<h3>$1</h3>')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`(.+?)`/g, '<code>$1</code>')
    .replace(/^- (.+)$/gm, '<li>$1</li>')
    .replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>')
    .replace(/\n{2,}/g, '<br><br>')
    .replace(/\n/g, '<br>')
}

const loadVersion = async () => {
  try {
    const res = await systemUpdate.version()
    currentVersion.value = res.obj?.version || ''
  } catch {}
}

const loadProxyConfig = async () => {
  try {
    const res = await systemUpdate.getConfig()
    if (res.obj) {
      const d = res.obj
      configForm.value = { proxy_url: d.proxy_url || '', repo_token: d.repo_token || '', owner: d.owner || '', repo: d.repo || '', enabled: !!d.enabled }
    }
  } catch {}
}

const saveProxyConfig = async () => {
  configSaving.value = true
  try {
    await systemUpdate.saveConfig(configForm.value)
    message.success('代理配置已保存')
    loadProxyConfig()
  } catch { message.error('保存配置失败') }
  finally { configSaving.value = false }
}

const checkUpdate = async () => {
  checking.value = true
  updateStatus.value = ''
  updateError.value = ''
  try {
    const res = await systemUpdate.updateCheck()
    const d = res.obj
    if (!d) { updateStatus.value = 'error'; return }
    currentVersion.value = d.current || currentVersion.value
    latestVersion.value = d.latest || ''
    if (d.error) {
      updateStatus.value = 'error'
      updateError.value = d.error
    } else {
      updateStatus.value = d.has_update ? 'available' : 'latest'
      loadChangelog()
    }
  } catch { updateStatus.value = 'error' }
  finally { checking.value = false }
}

const loadChangelog = async () => {
  changelogLoading.value = true
  try {
    const res = await systemUpdate.changelog()
    changelog.value = res.obj?.releases || []
  } catch {}
  finally { changelogLoading.value = false }
}

const showUpdateError = (msg, hint, cmd) => {
  updateErrorMsg.value = msg
  updateErrorHint.value = hint
  updateErrorCmd.value = cmd
  updateProgress.value = 'error'
}

const doUpdate = async () => {
  updating.value = true
  showProgressDialog.value = true
  updateProgress.value = 'updating'
  updateStep.value = 0

  try {
    updateStep.value = 1
    let confirmToken = ''
    try {
      const prepareRes = await systemUpdate.updatePrepare()
      confirmToken = prepareRes.obj?.confirm_token || ''
    } catch {
      showUpdateError('更新失败', '请在服务器上使用命令行更新：', 'bash update.sh --force')
      return
    }

    try {
      await systemUpdate.update({ confirm_token: confirmToken })
    } catch {
      showUpdateError('更新启动失败', '请在服务器上使用命令行更新：', 'bash update.sh --force')
      return
    }
    updateStep.value = 2

    const target = latestVersion.value
    const startTime = Date.now()
    const poll = async () => {
      if (Date.now() - startTime > 180000) {
        showUpdateError('更新超时', '请检查服务器日志，或使用命令行更新：', 'bash update.sh --force')
        return
      }
      try {
        const res = await systemUpdate.version()
        if (res.obj?.version === target) {
          updateStep.value = 3
          updateProgress.value = 'done'
          setTimeout(() => window.location.reload(), 3000)
          return
        }
      } catch {}
      setTimeout(poll, 2000)
    }
    poll()
  } catch {
    showUpdateError('更新失败', '请在服务器上使用命令行更新：', 'bash update.sh --force')
  } finally { updating.value = false }
}

onMounted(() => { loadVersion(); loadProxyConfig(); checkUpdate() })
</script>

<style scoped>
.system-update-page { max-width: 900px; animation: fadeIn 0.3s ease; }
.page-header { margin-bottom: 20px; }
.page-title { display: flex; align-items: center; gap: 10px; font-size: 22px; font-weight: 700; color: #1e293b; margin: 0; }
.title-icon { color: #3b82f6; font-size: 24px; }
.page-desc { color: #64748b; font-size: 14px; margin-top: 4px; }

.version-bar { display: flex; align-items: center; gap: 24px; background: #fff; border: 1px solid #e2e8f0; border-radius: 12px; padding: 12px 20px; margin-bottom: 16px; }
.version-info { display: flex; align-items: center; gap: 8px; }
.version-label { font-size: 12px; color: #64748b; }
.version-val { font-size: 14px; font-weight: 600; color: #1e293b; display: flex; align-items: center; }

.tabs-card { border-radius: 14px; }
.tab-inner { padding: 4px 0; }
.tab-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }

.changelog-list { }
.changelog-item { border-bottom: 1px solid #f1f5f9; }
.changelog-item:last-child { border-bottom: none; }
.changelog-item.current { background: #eff6ff; }
.changelog-item-header { display: flex; justify-content: space-between; align-items: center; padding: 10px 0; cursor: pointer; user-select: none; }
.changelog-item-header:hover { opacity: 0.8; }
.changelog-item-left { display: flex; align-items: center; gap: 8px; }
.expand-arrow { transition: transform 0.2s; color: #94a3b8; font-size: 10px; }
.expand-arrow.rotated { transform: rotate(90deg); }
.changelog-summary { font-size: 12px; color: #64748b; max-width: 340px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.changelog-date { font-size: 12px; color: #64748b; }
.changelog-body-wrap { padding: 0 0 12px 24px; }
.changelog-body { font-size: 13px; line-height: 1.7; color: #475569; }
.changelog-body.empty { color: #94a3b8; }
.changelog-body :deep(h3), .changelog-body :deep(h4) { margin: 8px 0 4px; font-size: 13px; font-weight: 600; color: #1e293b; }
.changelog-body :deep(ul) { margin: 4px 0; padding-left: 18px; }
.changelog-body :deep(code) { background: #f1f5f9; padding: 1px 4px; border-radius: 3px; font-size: 12px; }

.update-progress-content { display: flex; flex-direction: column; align-items: center; text-align: center; padding: 20px 0; }
.spinner-wrap { width: 64px; height: 64px; display: flex; align-items: center; justify-content: center; }
.update-spinner { width: 48px; height: 48px; border: 3px solid #e2e8f0; border-top-color: #3b82f6; border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.progress-title { margin: 16px 0 4px; font-size: 17px; font-weight: 600; color: #1e293b; }
.progress-hint { color: #64748b; font-size: 13px; margin: 0; }
.update-steps { margin-top: 20px; display: flex; flex-direction: column; gap: 12px; text-align: left; width: 100%; max-width: 200px; }
.step { display: flex; align-items: center; gap: 8px; font-size: 14px; color: #94a3b8; transition: color 0.3s; }
.step.done { color: #16a34a; font-weight: 500; }
.update-cmd-box { margin-top: 12px; padding: 10px 16px; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; width: 100%; }
.update-cmd-box code { font-size: 13px; color: #1e293b; font-family: 'SF Mono', Monaco, monospace; }

@keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
@media (max-width: 768px) { .version-bar { flex-wrap: wrap; gap: 12px; } .changelog-summary { display: none; } }
</style>
