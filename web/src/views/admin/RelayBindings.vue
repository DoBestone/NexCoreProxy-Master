<template>
  <div class="rb-page">
    <div class="page-toolbar">
      <span class="hint">中转节点 ⇄ Backend 节点的整体绑定，自动展开为每条 Inbound 的 Relay。</span>
      <a-button type="primary" @click="openCreate">
        <template #icon><PlusOutlined /></template>
        新增绑定
      </a-button>
    </div>

    <a-table
      class="rb-table"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="{ pageSize: 20, showSizeChanger: false }"
      row-key="id"
      size="small"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'pair'">
          <span class="pair">
            <span class="node-name">{{ nodeName(record.relayNodeId) }}</span>
            <ArrowRightOutlined class="arrow" />
            <span class="node-name">{{ nodeName(record.backendNodeId) }}</span>
          </span>
        </template>
        <template v-else-if="column.key === 'mode'">
          <span :class="['mode-tag', record.mode]">{{ record.mode === 'wrap' ? 'WRAP 套壳' : '透传' }}</span>
        </template>
        <template v-else-if="column.key === 'port'">
          <span class="mono">{{ portStrategyText(record) }}</span>
        </template>
        <template v-else-if="column.key === 'sync'">
          <span class="dot-line">
            <span :class="['dot', record.autoSync ? 'on' : 'off']" />
            {{ record.autoSync ? '自动同步' : '关闭' }}
          </span>
        </template>
        <template v-else-if="column.key === 'enable'">
          <span class="dot-line">
            <span :class="['dot', record.enable ? 'on' : 'off']" />
            {{ record.enable ? '启用' : '禁用' }}
          </span>
        </template>
        <template v-else-if="column.key === 'actions'">
          <a class="link" @click="onResync(record.id)">立即同步</a>
          <a class="link" @click="openEdit(record)">编辑</a>
          <a-popconfirm title="删除会清掉该绑定生成的 Relay" @confirm="onDelete(record.id)">
            <a class="link danger">删除</a>
          </a-popconfirm>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="modalOpen"
      :title="form.id ? '编辑绑定' : '新增绑定'"
      width="640px"
      :confirmLoading="submitting"
      @ok="onSubmit"
    >
      <a-form :model="form" layout="vertical" class="rb-form">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="Relay 中转节点" required>
              <a-select v-model:value="form.relayNodeId" placeholder="选择中转节点">
                <a-select-option v-for="n in nodes" :key="n.id" :value="n.id">
                  {{ n.name }} · {{ n.ip }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="Backend 落地节点" required>
              <a-select v-model:value="form.backendNodeId" placeholder="选择落地节点">
                <a-select-option v-for="n in nodes" :key="n.id" :value="n.id">
                  {{ n.name }} · {{ n.ip }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="模式">
              <a-select v-model:value="form.mode">
                <a-select-option value="transparent">透传 (推荐)</a-select-option>
                <a-select-option value="wrap">协议套壳 (wrap)</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="端口策略">
              <a-select v-model:value="form.portStrategy">
                <a-select-option value="same">相同端口</a-select-option>
                <a-select-option value="offset">偏移</a-select-option>
                <a-select-option value="pool">池中分配</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item v-if="form.portStrategy === 'offset'" label="偏移量">
              <a-input-number v-model:value="form.portOffset" style="width: 100%" />
            </a-form-item>
            <a-form-item v-else-if="form.portStrategy === 'pool'" label="池范围">
              <a-input-group compact>
                <a-input-number v-model:value="form.portPoolStart" :min="1" :max="65535" placeholder="start" style="width: 50%" />
                <a-input-number v-model:value="form.portPoolEnd" :min="1" :max="65535" placeholder="end" style="width: 50%" />
              </a-input-group>
            </a-form-item>
          </a-col>
        </a-row>

        <template v-if="form.mode === 'wrap'">
          <a-divider orientation="left" plain class="thin-divider">套壳设置</a-divider>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="套壳协议">
                <a-select v-model:value="form.wrapProtocol">
                  <a-select-option value="vless">VLESS</a-select-option>
                  <a-select-option value="trojan">Trojan</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="套壳安全">
                <a-select v-model:value="form.wrapSecurity">
                  <a-select-option value="reality">Reality</a-select-option>
                  <a-select-option value="tls">TLS</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-form-item>
            <a-checkbox v-model:checked="form.autoGenReality">每条 Relay 自动生成独立 Reality 密钥</a-checkbox>
          </a-form-item>
        </template>

        <a-form-item>
          <a-checkbox v-model:checked="form.autoSync">Backend 入站变化时自动同步</a-checkbox>
          <a-checkbox v-model:checked="form.enable" style="margin-left: 16px">启用</a-checkbox>
        </a-form-item>

        <a-form-item label="备注">
          <a-input v-model:value="form.remark" placeholder="可选" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, ArrowRightOutlined } from '@ant-design/icons-vue'
import {
  listRelayBindings, createRelayBinding, updateRelayBinding, deleteRelayBinding, resyncRelayBinding,
  getNodes,
} from '@/api'

const loading = ref(false)
const submitting = ref(false)
const rows = ref([])
const nodes = ref([])
const modalOpen = ref(false)
const form = ref(emptyForm())

const columns = [
  { title: '中转 → 落地', key: 'pair', width: 280 },
  { title: '模式', key: 'mode', width: 120 },
  { title: '端口策略', key: 'port', width: 160 },
  { title: '同步', key: 'sync', width: 110 },
  { title: '状态', key: 'enable', width: 90 },
  { title: '操作', key: 'actions', width: 180 },
]

function emptyForm() {
  return {
    id: 0,
    relayNodeId: null, backendNodeId: null,
    mode: 'transparent',
    portStrategy: 'same', portOffset: 0, portPoolStart: 30000, portPoolEnd: 40000,
    wrapProtocol: 'vless', wrapSecurity: 'reality', autoGenReality: true,
    autoSync: true, enable: true,
    remark: '',
  }
}

const nodeName = (id) => nodes.value.find(n => n.id === id)?.name || `#${id}`
function portStrategyText(r) {
  if (r.portStrategy === 'offset') return `+${r.portOffset || 0}`
  if (r.portStrategy === 'pool')   return `${r.portPoolStart}-${r.portPoolEnd}`
  return '同 backend'
}

async function fetchAll() {
  loading.value = true
  try {
    const [bs, ns] = await Promise.all([listRelayBindings(), getNodes()])
    rows.value = bs.obj || []
    nodes.value = ns.obj || []
  } catch {
    message.error('加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() { form.value = emptyForm(); modalOpen.value = true }
function openEdit(r)  { form.value = { ...r };   modalOpen.value = true }

async function onSubmit() {
  if (!form.value.relayNodeId || !form.value.backendNodeId) {
    return message.warning('请选择 relay / backend 节点')
  }
  if (form.value.relayNodeId === form.value.backendNodeId) {
    return message.warning('两端节点不能相同')
  }
  submitting.value = true
  try {
    if (form.value.id) await updateRelayBinding(form.value.id, form.value)
    else               await createRelayBinding(form.value)
    message.success('已保存并同步')
    modalOpen.value = false
    fetchAll()
  } catch (e) {
    message.error(e?.msg || '保存失败')
  } finally {
    submitting.value = false
  }
}

async function onResync(id) {
  try { await resyncRelayBinding(id); message.success('已重新同步') }
  catch { message.error('同步失败') }
}
async function onDelete(id) {
  try { await deleteRelayBinding(id); message.success('已删除'); fetchAll() }
  catch { message.error('删除失败') }
}

onMounted(fetchAll)
</script>

<style scoped>
.rb-page { --font-mono: ui-monospace, 'JetBrains Mono', 'SF Mono', monospace; }

.page-toolbar {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 12px; gap: 12px;
}
.hint { color: #64748b; font-size: 12.5px; }

.rb-table :deep(.ant-table-thead > tr > th) {
  padding: 10px 14px; font-size: 12px; font-weight: 600;
  background: #f8fafc; color: #1e293b;
}
.rb-table :deep(.ant-table-tbody > tr > td) {
  padding: 10px 14px; font-size: 13px; line-height: 1.5;
}
.rb-table :deep(.ant-table-tbody > tr:hover > td) { background: #f1f5f9; }

.pair { display: inline-flex; align-items: center; gap: 8px; }
.pair .arrow { color: #94a3b8; font-size: 12px; }
.node-name { font-weight: 500; color: #1e293b; }

.mode-tag {
  display: inline-block; padding: 2px 8px; border-radius: 6px;
  font-size: 11.5px; font-weight: 600; letter-spacing: .03em;
  background: #eff6ff; color: #2563eb;
}
.mode-tag.wrap { background: #fef3c7; color: #b45309; }

.dot-line { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; color: #475569; }
.dot { width: 6px; height: 6px; border-radius: 50%; background: #cbd5e1; }
.dot.on  { background: #16a34a; }
.dot.off { background: #cbd5e1; }
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }

.link { font-size: 12.5px; color: #2563eb; margin-right: 12px; cursor: pointer; }
.link:last-child { margin-right: 0; }
.link.danger { color: #dc2626; }
.link:hover { text-decoration: underline; }

.rb-form :deep(.ant-form-item) { margin-bottom: 14px; }
.rb-form :deep(.ant-form-item-label > label) {
  font-size: 12px; font-weight: 500; color: #64748b; height: 26px;
}
.thin-divider { margin: 14px 0 10px; font-size: 12px; color: #64748b; }
</style>
