<template>
  <div class="templates-page">
    <a-card title="入站模板">
      <template #extra>
        <a-button type="primary" @click="showAddModal">
          <template #icon><PlusOutlined /></template>
          添加模板
        </a-button>
      </template>
      <a-table :columns="columns" :dataSource="templates" :loading="loading" rowKey="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'protocol'">
            <a-tag :color="getProtocolColor(record.protocol)">{{ record.protocol }}</a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="applyTemplate(record)">应用</a-button>
              <a-button type="link" size="small" @click="editTemplate(record)">编辑</a-button>
              <a-popconfirm title="确定删除?" @confirm="deleteTemplateRecord(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 添加模板弹窗 -->
    <a-modal v-model:open="modalVisible" :title="editingTemplate ? '编辑模板' : '添加模板'" @ok="handleSubmit" width="700px">
      <a-form :model="form" :label-col="{ span: 4 }" :wrapper-col="{ span: 18 }">
        <a-form-item label="模板名称" required>
          <a-input v-model:value="form.name" placeholder="如: VMess-WS-TLS" />
        </a-form-item>
        <a-form-item label="协议" required>
          <a-select v-model:value="form.protocol" style="width: 100%">
            <a-select-option value="vmess">VMess</a-select-option>
            <a-select-option value="vless">VLESS</a-select-option>
            <a-select-option value="trojan">Trojan</a-select-option>
            <a-select-option value="shadowsocks">Shadowsocks</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="端口">
          <a-input-number v-model:value="form.port" :min="1" :max="65535" style="width: 100%" />
        </a-form-item>
        <a-form-item label="配置JSON">
          <a-textarea v-model:value="form.settings" :rows="6" placeholder="入站配置 JSON" />
        </a-form-item>
        <a-form-item label="传输层">
          <a-textarea v-model:value="form.stream" :rows="4" placeholder="传输层配置 JSON（可选）" />
        </a-form-item>
        <a-form-item label="TLS配置">
          <a-textarea v-model:value="form.tls" :rows="4" placeholder="TLS 配置 JSON（可选）" />
        </a-form-item>
        <a-form-item label="备注">
          <a-input v-model:value="form.remark" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 应用模板弹窗 -->
    <a-modal v-model:open="applyVisible" title="应用模板到节点" @ok="handleApply">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="选择节点">
          <a-select v-model:value="selectedNodeId" style="width: 100%" placeholder="选择节点">
            <a-select-option v-for="node in nodes" :key="node.id" :value="node.id">
              {{ node.name }} ({{ node.ip }})
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { getTemplates, addTemplate, deleteTemplate, getNodes, addNodeInbound } from '@/api'

const loading = ref(false)
const templates = ref([])
const nodes = ref([])
const modalVisible = ref(false)
const applyVisible = ref(false)
const editingTemplate = ref(null)
const currentTemplate = ref(null)
const selectedNodeId = ref(null)

const form = ref({
  name: '',
  protocol: 'vmess',
  port: 443,
  settings: '',
  stream: '',
  tls: '',
  remark: ''
})

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '协议', key: 'protocol' },
  { title: '端口', dataIndex: 'port', key: 'port' },
  { title: '备注', dataIndex: 'remark', key: 'remark' },
  { title: '操作', key: 'action', width: 180 }
]

const getProtocolColor = (protocol) => {
  const colors = {
    vmess: 'blue',
    vless: 'green',
    trojan: 'orange',
    shadowsocks: 'purple'
  }
  return colors[protocol] || 'default'
}

const fetchData = async () => {
  loading.value = true
  try {
    const [templatesRes, nodesRes] = await Promise.all([
      getTemplates(),
      getNodes()
    ])
    templates.value = templatesRes.obj || []
    nodes.value = nodesRes.obj || []
  } catch (e) {
    message.error('获取数据失败')
  } finally {
    loading.value = false
  }
}

const showAddModal = () => {
  editingTemplate.value = null
  form.value = {
    name: '',
    protocol: 'vmess',
    port: 443,
    settings: '',
    stream: '',
    tls: '',
    remark: ''
  }
  modalVisible.value = true
}

const editTemplate = (template) => {
  editingTemplate.value = template
  form.value = { ...template }
  modalVisible.value = true
}

const handleSubmit = async () => {
  try {
    await addTemplate(form.value)
    message.success('保存成功')
    modalVisible.value = false
    fetchData()
  } catch (e) {
    message.error('保存失败')
  }
}

const deleteTemplateRecord = async (id) => {
  try {
    await deleteTemplate(id)
    message.success('删除成功')
    fetchData()
  } catch (e) {
    message.error('删除失败')
  }
}

const applyTemplate = (template) => {
  currentTemplate.value = template
  selectedNodeId.value = null
  applyVisible.value = true
}

const handleApply = async () => {
  if (!selectedNodeId.value) {
    message.warning('请选择节点')
    return
  }
  try {
    const inbound = {
      remark: currentTemplate.value.name,
      protocol: currentTemplate.value.protocol,
      port: currentTemplate.value.port,
      settings: JSON.parse(currentTemplate.value.settings || '{}'),
      streamSettings: JSON.parse(currentTemplate.value.stream || '{}')
    }
    await addNodeInbound(selectedNodeId.value, inbound)
    message.success('应用成功')
    applyVisible.value = false
  } catch (e) {
    message.error('应用失败')
  }
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.templates-page {
  /* styles */
}
</style>