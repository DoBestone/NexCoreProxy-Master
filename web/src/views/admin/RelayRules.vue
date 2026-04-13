<template>
  <div class="relay-rules-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">
          <SwapOutlined class="title-icon" />
          中转规则
        </h1>
        <p class="page-desc">管理中转节点与落地节点的转发规则</p>
      </div>
      <a-button type="primary" size="large" @click="showAddModal">
        <template #icon><PlusOutlined /></template>
        添加规则
      </a-button>
    </div>

    <!-- 规则列表 -->
    <a-spin :spinning="loading">
      <div v-if="rules.length === 0 && !loading" class="empty-state">
        <SwapOutlined style="font-size: 48px; color: var(--gray-300)" />
        <p>暂无中转规则，点击上方按钮添加</p>
      </div>
      <div class="rules-grid" v-else>
        <div v-for="rule in rules" :key="rule.id" class="rule-card">
          <div class="rule-header">
            <a-tag :color="protocolColor(rule.protocol)">{{ rule.protocol?.toUpperCase() }}</a-tag>
            <span :class="['sync-badge', rule.syncStatus]">
              <span class="sync-dot"></span>
              {{ syncStatusText(rule.syncStatus) }}
            </span>
          </div>

          <div class="rule-route">
            <div class="route-node relay">
              <span class="route-label">中转</span>
              <span class="route-name">{{ rule.relayNodeName }}</span>
            </div>
            <div class="route-arrow">
              <SwapRightOutlined />
            </div>
            <div class="route-node backend">
              <span class="route-label">落地</span>
              <span class="route-name">{{ rule.backendNodeName }}</span>
            </div>
          </div>

          <div class="rule-info">
            <span v-if="rule.relayInboundPort">端口: {{ rule.relayInboundPort }}</span>
            <span v-if="rule.remark">{{ rule.remark }}</span>
          </div>

          <div v-if="rule.syncStatus === 'error' && rule.syncError" class="rule-error">
            {{ rule.syncError }}
          </div>

          <div class="rule-actions">
            <a-button size="small" @click="syncRule(rule.id)" :loading="syncingId === rule.id">
              <template #icon><SyncOutlined /></template>
              同步
            </a-button>
            <a-popconfirm title="确定删除此规则？" @confirm="deleteRule(rule.id)">
              <a-button size="small" danger>
                <template #icon><DeleteOutlined /></template>
                删除
              </a-button>
            </a-popconfirm>
          </div>
        </div>
      </div>
    </a-spin>

    <!-- 添加规则弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      title="添加中转规则"
      @ok="handleSubmit"
      :confirmLoading="submitting"
    >
      <a-form :model="form" layout="vertical">
        <a-form-item label="中转节点" required>
          <a-select v-model:value="form.relayNodeId" placeholder="选择中转节点" show-search option-filter-prop="label">
            <a-select-option v-for="n in relayNodes" :key="n.id" :value="n.id" :label="n.name">
              {{ n.name }} ({{ n.ip }})
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="落地节点" required>
          <a-select v-model:value="form.backendNodeId" placeholder="选择落地节点" show-search option-filter-prop="label">
            <a-select-option v-for="n in backendNodes" :key="n.id" :value="n.id" :label="n.name">
              {{ n.name }} ({{ n.ip }})
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="协议" required>
              <a-select v-model:value="form.protocol" placeholder="选择协议">
                <a-select-option value="vmess">VMess</a-select-option>
                <a-select-option value="vless">VLESS</a-select-option>
                <a-select-option value="trojan">Trojan</a-select-option>
                <a-select-option value="shadowsocks">Shadowsocks</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="中转端口">
              <a-input-number v-model:value="form.relayInboundPort" :min="1" :max="65535" placeholder="自动分配" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="备注">
          <a-input v-model:value="form.remark" placeholder="可选" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { SwapOutlined, SwapRightOutlined, PlusOutlined, SyncOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { getNodes, getRelayRules, createRelayRule, deleteRelayRule, syncRelayRule } from '@/api'

const rules = ref([])
const nodes = ref([])
const loading = ref(false)
const modalVisible = ref(false)
const submitting = ref(false)
const syncingId = ref(null)

const form = ref({
  relayNodeId: undefined,
  backendNodeId: undefined,
  protocol: 'vless',
  relayInboundPort: undefined,
  remark: ''
})

const relayNodes = computed(() => nodes.value.filter(n => n.type === 'relay'))
const backendNodes = computed(() => nodes.value.filter(n => n.type === 'backend'))

const protocolColor = (p) => {
  const map = { vmess: 'blue', vless: 'cyan', trojan: 'green', shadowsocks: 'orange' }
  return map[p] || 'default'
}

const syncStatusText = (s) => {
  const map = { synced: '已同步', pending: '待同步', error: '同步失败' }
  return map[s] || s
}

const fetchData = async () => {
  loading.value = true
  try {
    const [rulesRes, nodesRes] = await Promise.all([getRelayRules(), getNodes()])
    rules.value = rulesRes.data?.obj || []
    nodes.value = nodesRes.data?.obj || []
  } catch (e) {
    message.error('加载数据失败')
  } finally {
    loading.value = false
  }
}

const showAddModal = () => {
  form.value = { relayNodeId: undefined, backendNodeId: undefined, protocol: 'vless', relayInboundPort: undefined, remark: '' }
  modalVisible.value = true
}

const handleSubmit = async () => {
  if (!form.value.relayNodeId || !form.value.backendNodeId || !form.value.protocol) {
    message.warning('请填写必填项')
    return
  }
  submitting.value = true
  try {
    await createRelayRule(form.value)
    message.success('创建成功，正在同步...')
    modalVisible.value = false
    fetchData()
  } catch (e) {
    message.error('创建失败')
  } finally {
    submitting.value = false
  }
}

const syncRule = async (id) => {
  syncingId.value = id
  try {
    const res = await syncRelayRule(id)
    if (res.data?.success) {
      message.success('同步成功')
    } else {
      message.error(res.data?.msg || '同步失败')
    }
    fetchData()
  } catch (e) {
    message.error('同步失败')
  } finally {
    syncingId.value = null
  }
}

const deleteRule = async (id) => {
  try {
    await deleteRelayRule(id)
    message.success('删除成功')
    fetchData()
  } catch (e) {
    message.error('删除失败')
  }
}

onMounted(fetchData)
</script>

<style scoped>
.relay-rules-page {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--gray-800);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.title-icon {
  color: var(--primary-500);
}

.page-desc {
  color: var(--gray-500);
  margin: 4px 0 0;
  font-size: 14px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: var(--gray-400);
}

.rules-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
}

.rule-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid var(--gray-200);
  transition: box-shadow 0.2s;
}

.rule-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.rule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.sync-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  padding: 2px 10px;
  border-radius: 10px;
}

.sync-badge.synced {
  background: #f0fdf4;
  color: #16a34a;
}

.sync-badge.pending {
  background: #fffbeb;
  color: #d97706;
}

.sync-badge.error {
  background: #fef2f2;
  color: #dc2626;
}

.sync-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.rule-route {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.route-node {
  flex: 1;
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--gray-50);
}

.route-node.relay {
  border-left: 3px solid #f59e0b;
}

.route-node.backend {
  border-left: 3px solid #8b5cf6;
}

.route-label {
  display: block;
  font-size: 11px;
  color: var(--gray-400);
  margin-bottom: 2px;
}

.route-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--gray-700);
}

.route-arrow {
  color: var(--gray-400);
  font-size: 20px;
}

.rule-info {
  display: flex;
  gap: 12px;
  font-size: 13px;
  color: var(--gray-500);
  margin-bottom: 12px;
}

.rule-error {
  background: #fef2f2;
  color: #dc2626;
  font-size: 12px;
  padding: 8px 12px;
  border-radius: 6px;
  margin-bottom: 12px;
  word-break: break-all;
}

.rule-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
