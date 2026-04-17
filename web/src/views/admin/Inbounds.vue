<template>
  <div class="inbounds-page">
    <!-- 顶部工具条：右对齐操作 + 节点筛选 -->
    <div class="page-toolbar">
      <a-select
        v-model:value="filterNodeId"
        placeholder="全部节点"
        style="width: 200px"
        allowClear
        @change="fetchInbounds"
      >
        <a-select-option v-for="n in nodes" :key="n.id" :value="n.id">
          {{ n.name }} · {{ n.ip }}
        </a-select-option>
      </a-select>

      <div class="actions">
        <a-dropdown :disabled="!filterNodeId">
          <a-button>
            一键预设
            <DownOutlined />
          </a-button>
          <template #overlay>
            <a-menu @click="onProvision">
              <a-menu-item key="minimal">最小集 (VLESS-Reality)</a-menu-item>
              <a-menu-item key="standard">标准集 (Reality+Trojan+SS)</a-menu-item>
              <a-menu-item key="full">完整集 (含 Hy2/TUIC/VMess)</a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          新增入站
        </a-button>
      </div>
    </div>

    <!-- 数据表 -->
    <a-table
      class="inbound-table"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="{ pageSize: 20, showSizeChanger: false }"
      row-key="id"
      size="small"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'node'">
          <span class="node-name">{{ nodeName(record.nodeId) }}</span>
        </template>
        <template v-else-if="column.key === 'protocol'">
          <span :class="['proto-pill', record.protocol]">{{ record.protocol?.toUpperCase() }}</span>
        </template>
        <template v-else-if="column.key === 'port'">
          <span class="mono">{{ record.port }}</span>
          <span v-if="record.portRange" class="mono dim"> · {{ record.portRange }}</span>
        </template>
        <template v-else-if="column.key === 'security'">
          <span :class="['sec-tag', record.security || 'none']">{{ (record.security || 'none').toUpperCase() }}</span>
          <span v-if="record.network" class="mono dim"> / {{ record.network }}</span>
        </template>
        <template v-else-if="column.key === 'cert'">
          <span v-if="record.certDomain" class="mono">{{ record.certDomain }}</span>
          <span v-else class="dim">—</span>
        </template>
        <template v-else-if="column.key === 'enable'">
          <span class="dot-line">
            <span :class="['dot', record.enable ? 'on' : 'off']" />
            {{ record.enable ? '启用' : '禁用' }}
          </span>
        </template>
        <template v-else-if="column.key === 'actions'">
          <a class="link" @click="openEdit(record)">编辑</a>
          <a class="link" @click="onToggle(record)">{{ record.enable ? '禁用' : '启用' }}</a>
          <a-popconfirm title="确定删除？" @confirm="onDelete(record.id)">
            <a class="link danger">删除</a>
          </a-popconfirm>
        </template>
      </template>
    </a-table>

    <!-- 创建/编辑表单 -->
    <a-modal
      v-model:open="modalOpen"
      :title="form.id ? '编辑入站' : '新增入站'"
      width="720px"
      :confirmLoading="submitting"
      @ok="onSubmit"
    >
      <a-form :model="form" layout="vertical" class="inbound-form">
        <a-alert
          v-if="!form.id"
          message="极简模式：只需选协议 + 节点，其他全部自动。Reality 密钥、SS PSK、WS path 都由系统生成。"
          type="info"
          show-icon
          class="thin-alert"
          style="margin-bottom: 12px"
        />

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="节点" required>
              <a-select v-model:value="form.nodeId" placeholder="选择节点">
                <a-select-option v-for="n in nodes" :key="n.id" :value="n.id">{{ n.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="协议" required>
              <a-select v-model:value="form.protocol">
                <a-select-option value="vless">VLESS + Reality (推荐)</a-select-option>
                <a-select-option value="vmess">VMess + TLS</a-select-option>
                <a-select-option value="trojan">Trojan + TLS</a-select-option>
                <a-select-option value="ss">Shadowsocks-2022</a-select-option>
                <a-select-option value="hysteria2">Hysteria2 (UDP)</a-select-option>
                <a-select-option value="tuic">TUIC v5 (UDP)</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item>
          <a-checkbox v-model:checked="showAdvanced">显示高级选项</a-checkbox>
        </a-form-item>

        <template v-if="showAdvanced">
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item label="端口 (留空按协议默认)">
                <a-input-number v-model:value="form.port" :min="0" :max="65535" style="width: 100%" placeholder="auto" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="名称 (留空自动)">
                <a-input v-model:value="form.name" placeholder="auto" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="Tag (留空自动)">
                <a-input v-model:value="form.tag" placeholder="auto" class="mono-input" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="6">
              <a-form-item label="网络">
                <a-select v-model:value="form.network" allowClear placeholder="auto">
                  <a-select-option value="tcp">tcp</a-select-option>
                  <a-select-option value="ws">ws</a-select-option>
                  <a-select-option value="grpc">grpc</a-select-option>
                  <a-select-option value="h2">h2</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item label="安全层">
                <a-select v-model:value="form.security" allowClear placeholder="auto">
                  <a-select-option value="none">none</a-select-option>
                  <a-select-option value="tls">tls</a-select-option>
                  <a-select-option value="reality">reality</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="端口跳跃 (仅 Hy2)">
                <a-input v-model:value="form.portRange" placeholder="20000-30000" class="mono-input" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16" v-if="form.security === 'reality'">
            <a-col :span="12">
              <a-form-item label="伪装 SNI (留空随机)">
                <a-input v-model:value="form.realitySni" placeholder="auto" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="Dest (留空随机)">
                <a-input v-model:value="form.realityDest" placeholder="auto" class="mono-input" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-form-item v-if="form.security === 'tls'" label="TLS 证书域名 (ACME 自动签发)">
            <a-input v-model:value="form.certDomain" placeholder="node1.example.com" class="mono-input" />
          </a-form-item>
          <a-form-item label="settings JSON (可选)">
            <a-textarea v-model:value="form.settingsJson" :rows="3" class="mono-input" placeholder="auto" />
          </a-form-item>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="streamSettings JSON (可选)">
                <a-textarea v-model:value="form.streamJson" :rows="3" class="mono-input" placeholder="auto" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="tls/reality JSON (可选)">
                <a-textarea v-model:value="form.tlsJson" :rows="3" class="mono-input" placeholder="auto" />
              </a-form-item>
            </a-col>
          </a-row>
        </template>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, DownOutlined } from '@ant-design/icons-vue'
import {
  listInbounds, createInbound, updateInbound, deleteInbound, toggleInbound,
  getNodes, provisionNode,
} from '@/api'

const loading = ref(false)
const submitting = ref(false)
const rows = ref([])
const nodes = ref([])
const filterNodeId = ref(null)
const modalOpen = ref(false)
const showAdvanced = ref(false)
const form = ref(emptyForm())

const columns = [
  { title: '节点', key: 'node', width: 160 },
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '协议', key: 'protocol', width: 100 },
  { title: '端口', key: 'port', width: 130 },
  { title: '安全/网络', key: 'security', width: 150 },
  { title: '证书域名', key: 'cert', width: 180 },
  { title: '状态', key: 'enable', width: 90 },
  { title: '操作', key: 'actions', width: 160 },
]

// 默认大部分字段留空，让后端 autofill 生成随机值
function emptyForm() {
  return {
    id: 0, nodeId: null, name: '', tag: '',
    protocol: 'vless', port: 0, portRange: '',
    network: '', security: '',
    realitySni: '', realityDest: '',
    certDomain: '',
    settingsJson: '', streamJson: '', tlsJson: '',
  }
}

const nodeName = (id) => nodes.value.find(n => n.id === id)?.name || `#${id}`

async function fetchInbounds() {
  loading.value = true
  try {
    const r = await listInbounds(filterNodeId.value)
    rows.value = r.obj || []
  } catch {
    message.error('加载入站列表失败')
  } finally {
    loading.value = false
  }
}

async function fetchNodes() {
  const r = await getNodes()
  nodes.value = (r.obj || []).filter(n => n.role !== 'relay')
}

function openCreate() {
  form.value = emptyForm()
  if (filterNodeId.value) form.value.nodeId = filterNodeId.value
  showAdvanced.value = false
  modalOpen.value = true
}

function openEdit(row) {
  form.value = { ...row }
  showAdvanced.value = true // 编辑时直接展开
  modalOpen.value = true
}

async function onSubmit() {
  submitting.value = true
  try {
    if (form.value.id) {
      await updateInbound(form.value.id, form.value)
    } else {
      await createInbound(form.value)
    }
    message.success('保存成功')
    modalOpen.value = false
    fetchInbounds()
  } catch (e) {
    message.error(e?.msg || '保存失败')
  } finally {
    submitting.value = false
  }
}

async function onToggle(row) {
  try {
    await toggleInbound(row.id, !row.enable)
    fetchInbounds()
  } catch {
    message.error('操作失败')
  }
}

async function onDelete(id) {
  try {
    await deleteInbound(id)
    message.success('已删除')
    fetchInbounds()
  } catch {
    message.error('删除失败')
  }
}

async function onProvision({ key }) {
  if (!filterNodeId.value) return
  try {
    const r = await provisionNode(filterNodeId.value, key)
    message.success(`已写入 ${r.count || 0} 个预设入站`)
    fetchInbounds()
  } catch (e) {
    message.error(e?.msg || '预设失败')
  }
}

onMounted(async () => {
  await fetchNodes()
  await fetchInbounds()
})
</script>

<style scoped>
:root, .inbounds-page {
  --font-mono: ui-monospace, 'JetBrains Mono', 'SF Mono', 'Cascadia Code', monospace;
}

.inbounds-page { padding: 0; }

.page-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  gap: 12px;
}
.page-toolbar .actions { display: flex; gap: 8px; }

/* 表格紧凑化 */
.inbound-table :deep(.ant-table-thead > tr > th) {
  padding: 10px 14px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: .02em;
  background: #f8fafc;
  color: #1e293b;
}
.inbound-table :deep(.ant-table-tbody > tr > td) {
  padding: 10px 14px;
  font-size: 13px;
  line-height: 1.5;
}
.inbound-table :deep(.ant-table-tbody > tr:hover > td) { background: #f1f5f9; }

/* 协议色块 —— 主色权重，避免彩虹 */
.proto-pill {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: .03em;
  background: #eff6ff;
  color: #2563eb;
}
.proto-pill.trojan { background: #f1f5f9; color: #475569; }
.proto-pill.ss     { background: #ecfeff; color: #0e7490; }
.proto-pill.vmess  { background: #faf5ff; color: #6d28d9; }
.proto-pill.hysteria2,
.proto-pill.tuic   { background: #fef3c7; color: #b45309; }

/* 安全/网络小标签 */
.sec-tag {
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: .03em;
  color: #475569;
}
.sec-tag.reality { color: #2563eb; }
.sec-tag.tls     { color: #16a34a; }
.sec-tag.none    { color: #94a3b8; }

/* 状态点 */
.dot-line { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; color: #475569; }
.dot { width: 6px; height: 6px; border-radius: 50%; background: #94a3b8; }
.dot.on  { background: #16a34a; }
.dot.off { background: #cbd5e1; }

/* 数字/Tag/端口用 mono */
.mono       { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.mono.dim   { color: #94a3b8; }
.dim        { color: #94a3b8; }
.node-name  { font-weight: 500; color: #1e293b; }

/* 操作链接（小号） */
.link {
  font-size: 12.5px;
  color: #2563eb;
  margin-right: 12px;
  cursor: pointer;
}
.link:last-child { margin-right: 0; }
.link.danger { color: #dc2626; }
.link:hover { text-decoration: underline; }

/* 表单紧凑 */
.inbound-form :deep(.ant-form-item) { margin-bottom: 14px; }
.inbound-form :deep(.ant-form-item-label > label) {
  font-size: 12px;
  font-weight: 500;
  color: #64748b;
  height: 26px;
}
.mono-input :deep(input),
.mono-input :deep(textarea),
.mono-input { font-family: var(--font-mono); font-size: 12.5px; }

.thin-divider { margin: 14px 0 10px; font-size: 12px; color: #64748b; }
.thin-alert { padding: 6px 10px; font-size: 12px; }
</style>
