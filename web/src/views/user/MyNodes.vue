<template>
  <div class="my-nodes-page">
    <!-- 订阅卡 -->
    <section class="sub-card">
      <div class="sub-accent"></div>
      <div class="sub-head">
        <div class="sub-icon"><LinkOutlined /></div>
        <div class="sub-title-wrap">
          <h3 class="sub-title">订阅链接</h3>
          <p class="sub-desc">导入到客户端即可使用所有可用节点</p>
        </div>
      </div>

      <div class="sub-rows">
        <div class="sub-row">
          <div class="sub-row-label">
            <span class="sub-tag">通用</span>
            <span>V2Ray / Xray / Shadowrocket</span>
          </div>
          <div class="sub-url">
            <code class="mono">{{ subscribeUrl || '— 加载中 —' }}</code>
            <button class="sub-copy" :disabled="!subscribeUrl" @click="copy(subscribeUrl, '通用订阅')">
              <CopyOutlined />
            </button>
          </div>
        </div>

        <div class="sub-row">
          <div class="sub-row-label">
            <span class="sub-tag sub-tag-alt">Clash</span>
            <span>Clash / ClashX / Stash</span>
          </div>
          <div class="sub-url">
            <code class="mono">{{ clashUrl || '— 加载中 —' }}</code>
            <button class="sub-copy" :disabled="!clashUrl" @click="copy(clashUrl, 'Clash 订阅')">
              <CopyOutlined />
            </button>
          </div>
        </div>
      </div>

      <div class="sub-actions">
        <a-button type="primary" @click="copy(subscribeUrl, '通用订阅')">
          <CopyOutlined /> 复制通用订阅
        </a-button>
        <a-button @click="copy(clashUrl, 'Clash 订阅')">
          <CopyOutlined /> 复制 Clash
        </a-button>
        <a-button :loading="refreshing" @click="refreshSubscribe">
          <ReloadOutlined /> 刷新订阅
        </a-button>
      </div>
    </section>

    <!-- 节点列表 -->
    <section class="nodes-section">
      <div class="section-head">
        <div class="section-head-left">
          <h2 class="section-title">我的节点</h2>
          <span class="section-count">
            <span class="dot dot-ok"></span>
            {{ onlineCount }} 在线 · 共 {{ nodes.length }} 节点
          </span>
        </div>
        <div class="section-head-right">
          <div class="seg">
            <button
              v-for="opt in filterOptions"
              :key="opt.key"
              class="seg-btn"
              :class="{ 'is-active': filter === opt.key }"
              @click="filter = opt.key"
            >{{ opt.label }}</button>
          </div>
        </div>
      </div>

      <div v-if="loading" class="state state-loading">加载节点中…</div>

      <div v-else-if="nodes.length === 0" class="state state-empty">
        <CloudServerOutlined class="state-icon" />
        <p class="state-title">暂无分配节点</p>
        <p class="state-sub">购买套餐后会自动分配可用节点</p>
        <a-button type="primary" @click="$router.push('/user/buy')">
          <ShoppingCartOutlined /> 去购买套餐
        </a-button>
      </div>

      <div v-else-if="filteredNodes.length === 0" class="state state-empty">
        <p class="state-title">当前筛选下无节点</p>
      </div>

      <div v-else class="nodes-grid">
        <article
          v-for="(node, idx) in filteredNodes"
          :key="node.id"
          class="node-card"
          :style="{ animationDelay: (idx * 40) + 'ms' }"
        >
          <header class="node-head">
            <div class="node-name-wrap">
              <span class="node-name">{{ node.name }}</span>
              <span class="node-region mono" v-if="node.region">{{ node.region }}</span>
            </div>
            <span :class="['node-status', node.status]">
              <span class="dot"></span>
              {{ node.status === 'online' ? '在线' : '离线' }}
            </span>
          </header>

          <dl class="node-fields">
            <div class="field">
              <dt>协议</dt>
              <dd><span class="pill-proto">{{ (node.protocol || '—').toUpperCase() }}</span></dd>
            </div>
            <div class="field">
              <dt>地址</dt>
              <dd class="mono">{{ node.ip || '—' }}</dd>
            </div>
          </dl>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  CloudServerOutlined,
  CopyOutlined,
  ReloadOutlined,
  LinkOutlined,
  ShoppingCartOutlined
} from '@ant-design/icons-vue'
import { getMyNodes, getMySubscribe } from '@/api'

const nodes = ref([])
const subscribeUrl = ref('')
const clashUrl = ref('')
const loading = ref(false)
const refreshing = ref(false)
const filter = ref('all')

const filterOptions = [
  { key: 'all',     label: '全部' },
  { key: 'online',  label: '在线' },
  { key: 'offline', label: '离线' }
]

const onlineCount = computed(() => nodes.value.filter(n => n.status === 'online').length)

const filteredNodes = computed(() => {
  if (filter.value === 'all') return nodes.value
  return nodes.value.filter(n => n.status === filter.value)
})

const copy = (text, label) => {
  if (!text) return
  navigator.clipboard.writeText(text)
  message.success(`${label}已复制`)
}

const refreshSubscribe = async () => {
  refreshing.value = true
  try {
    const res = await getMySubscribe()
    if (res.success && res.obj) {
      subscribeUrl.value = res.obj.url || ''
      clashUrl.value = res.obj.clashUrl || (res.obj.url ? res.obj.url + '?flag=clash' : '')
      message.success('订阅已刷新')
    }
  } catch (e) {
    message.error('刷新订阅失败')
  } finally {
    refreshing.value = false
  }
}

const fetchNodes = async () => {
  loading.value = true
  try {
    const res = await getMyNodes()
    nodes.value = res.obj || []
  } catch (e) {
    message.error('获取节点失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchNodes()
  refreshSubscribe()
})
</script>

<style scoped>
.my-nodes-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* ============================================================
   订阅卡
   ============================================================ */
.sub-card {
  position: relative;
  background: linear-gradient(160deg, #f0f6ff 0%, #ffffff 65%);
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  padding: 16px 18px;
  overflow: hidden;
  animation: rise .4s ease-out both;
}

.sub-accent {
  position: absolute;
  top: 0; left: 0; right: 0;
  height: 3px;
  background: linear-gradient(90deg, #3b82f6 0%, #60a5fa 50%, #3b82f6 100%);
}

.sub-head {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 20px;
}

.sub-icon {
  width: 42px;
  height: 42px;
  border-radius: 11px;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  box-shadow: 0 4px 12px rgba(59,130,246,.24);
  flex-shrink: 0;
}

.sub-title {
  font-family: var(--font-display);
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.01em;
  line-height: 1.2;
}

.sub-desc {
  margin: 4px 0 0;
  color: #64748b;
  font-size: 12.5px;
  line-height: 1.4;
}

.sub-rows {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 16px;
}

.sub-row-label {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 11.5px;
  color: #64748b;
  margin-bottom: 6px;
}

.sub-tag {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 4px;
  background: #eff6ff;
  color: #2563eb;
  font-size: 10.5px;
  font-weight: 600;
  letter-spacing: .02em;
}

.sub-tag-alt {
  background: #f0fdf4;
  color: #047857;
}

.sub-url {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 10px 10px 10px 14px;
  transition: border-color .15s;
}

.sub-url:hover { border-color: #cbd5e1; }

.sub-url code {
  flex: 1;
  font-size: 12.5px;
  color: #1e293b;
  background: transparent;
  word-break: break-all;
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
}

.sub-copy {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  background: #f8fafc;
  color: #64748b;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: color .15s, border-color .15s, background-color .15s;
}

.sub-copy:hover:not(:disabled) {
  color: #2563eb;
  border-color: #dbeafe;
  background: #fff;
}

.sub-copy:disabled { opacity: .5; cursor: not-allowed; }

.sub-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

/* ============================================================
   节点区
   ============================================================ */
.nodes-section {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 12px;
  padding: 14px 16px 18px;
  animation: rise .4s ease-out both;
  animation-delay: 80ms;
}

.section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding-bottom: 14px;
  border-bottom: 1px solid #f1f5f9;
  margin-bottom: 18px;
}

.section-head-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
}

.section-title {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  color: #0f172a;
  margin: 0;
  letter-spacing: -0.01em;
}

.section-count {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #64748b;
}

/* 段控件 */
.seg {
  display: inline-flex;
  padding: 3px;
  background: #f1f5f9;
  border-radius: 9px;
}

.seg-btn {
  height: 28px;
  padding: 0 12px;
  border: none;
  background: transparent;
  color: #64748b;
  font-size: 12.5px;
  font-weight: 500;
  border-radius: 7px;
  cursor: pointer;
  transition: background-color .15s, color .15s;
}

.seg-btn:hover { color: #1e293b; }

.seg-btn.is-active {
  background: #fff;
  color: #2563eb;
  box-shadow: 0 1px 2px rgba(15,23,42,.08);
}

/* 状态区 */
.state {
  padding: 56px 20px;
  text-align: center;
  color: #94a3b8;
}

.state-loading { font-size: 13px; }

.state-icon {
  font-size: 42px;
  color: #cbd5e1;
  margin-bottom: 14px;
}

.state-title {
  margin: 0 0 6px;
  font-size: 14px;
  font-weight: 600;
  color: #475569;
}

.state-sub {
  margin: 0 0 20px;
  font-size: 12.5px;
  color: #94a3b8;
}

/* 节点网格 */
.nodes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 14px;
}

.node-card {
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 12px;
  padding: 16px;
  transition: border-color .15s, box-shadow .15s;
  animation: rise .35s ease-out both;
}

.node-card:hover {
  border-color: #c7d8f2;
  box-shadow: 0 4px 14px rgba(59,130,246,.08);
}

.node-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 12px;
}

.node-name-wrap {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.node-name {
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-region {
  font-size: 11px;
  color: #94a3b8;
}

.node-status {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 9px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  flex-shrink: 0;
}

.node-status.online  { background: #ecfdf5; color: #047857; }
.node-status.offline { background: #fef2f2; color: #b91c1c; }

.node-status .dot {
  width: 5px; height: 5px; border-radius: 50%;
  background: currentColor;
}

.dot { width: 6px; height: 6px; border-radius: 50%; display: inline-block; }
.dot-ok { background: #16a34a; }

.node-fields {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12.5px;
}

.field dt { color: #94a3b8; margin: 0; }
.field dd { margin: 0; color: #1e293b; max-width: 65%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.pill-proto {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 5px;
  background: #eff6ff;
  color: #2563eb;
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: .02em;
}

/* ============================================================
   响应式
   ============================================================ */
@media (max-width: 768px) {
  .section-head { flex-direction: column; align-items: flex-start; }
}

@media (max-width: 576px) {
  .sub-card { padding: 20px 16px; }
  .sub-head { gap: 10px; }
  .sub-actions .ant-btn { flex: 1; }
  .nodes-grid { grid-template-columns: 1fr; }
}

@keyframes rise {
  from { opacity: 0; transform: translateY(6px); }
  to   { opacity: 1; transform: none; }
}
</style>
