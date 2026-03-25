<template>
  <div class="nodes-page">
    <a-card title="节点管理">
      <template #extra>
        <a-button type="primary" @click="showAddModal">
          <template #icon><PlusOutlined /></template>
          添加节点
        </a-button>
      </template>
      <a-table :columns="columns" :dataSource="nodes" :loading="loading" rowKey="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-if="column.key === 'enable'">
            <a-switch v-model:checked="record.enable" @change="toggleEnable(record)" />
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-tooltip title="测试连接">
                <a-button type="link" size="small" @click="testConnection(record)">
                  <ApiOutlined />
                </a-button>
              </a-tooltip>
              <a-tooltip title="SSH安装">
                <a-button type="link" size="small" @click="doInstallNode(record)" :loading="record.installing">
                  <CloudUploadOutlined />
                </a-button>
              </a-tooltip>
              <a-tooltip title="同步状态">
                <a-button type="link" size="small" @click="syncStatus(record)">
                  <SyncOutlined />
                </a-button>
              </a-tooltip>
              <a-tooltip title="查看入站">
                <a-button type="link" size="small" @click="showInbounds(record)">
                  <UnorderedListOutlined />
                </a-button>
              </a-tooltip>
              <a-tooltip title="编辑">
                <a-button type="link" size="small" @click="editNode(record)">
                  <EditOutlined />
                </a-button>
              </a-tooltip>
              <a-popconfirm title="确定删除?" @confirm="deleteNodeRecord(record.id)">
                <a-button type="link" size="small" danger>
                  <DeleteOutlined />
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 添加/编辑节点弹窗 -->
    <a-modal v-model:open="modalVisible" :title="editingNode ? '编辑节点' : '添加节点'" @ok="handleSubmit" :confirmLoading="submitting">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="节点名称" required>
          <a-input v-model:value="form.name" placeholder="如: 香港节点1" />
        </a-form-item>
        <a-form-item label="IP地址" required>
          <a-input v-model:value="form.ip" placeholder="服务器IP" />
        </a-form-item>
        <a-form-item label="面板端口">
          <a-input-number v-model:value="form.port" :min="1" :max="65535" style="width: 100%" />
        </a-form-item>
        <a-form-item label="用户名" required>
          <a-input v-model:value="form.username" placeholder="面板登录用户名" />
        </a-form-item>
        <a-form-item label="密码" required>
          <a-input-password v-model:value="form.password" placeholder="面板登录密码" />
        </a-form-item>
        <a-divider>SSH 配置（可选）</a-divider>
        <a-form-item label="SSH端口">
          <a-input-number v-model:value="form.sshPort" :min="1" :max="65535" style="width: 100%" />
        </a-form-item>
        <a-form-item label="SSH用户">
          <a-input v-model:value="form.sshUser" placeholder="root" />
        </a-form-item>
        <a-form-item label="SSH密码">
          <a-input-password v-model:value="form.sshPassword" placeholder="SSH密码" />
        </a-form-item>
        <a-form-item label="备注">
          <a-textarea v-model:value="form.remark" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 入站列表弹窗 -->
    <a-modal v-model:open="inboundsVisible" :title="currentNode?.name + ' - 入站列表'" width="800px" :footer="null">
      <a-table :columns="inboundColumns" :dataSource="inbounds" :loading="inboundsLoading" rowKey="id" size="small">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'enable'">
            <a-tag :color="record.enable ? 'green' : 'red'">
              {{ record.enable ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'traffic'">
            <div>↑ {{ formatTraffic(record.up) }} / ↓ {{ formatTraffic(record.down) }}</div>
            <div>限额: {{ formatTraffic(record.total) || '无限制' }}</div>
          </template>
        </template>
      </a-table>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  PlusOutlined,
  ApiOutlined,
  SyncOutlined,
  UnorderedListOutlined,
  EditOutlined,
  DeleteOutlined,
  CloudUploadOutlined
} from '@ant-design/icons-vue'
import { getNodes, addNode, updateNode, deleteNode, testNode, syncNode, installNode as installNodeApi, getNodeInbounds } from '@/api'

const loading = ref(false)
const nodes = ref([])
const modalVisible = ref(false)
const inboundsVisible = ref(false)
const editingNode = ref(null)
const submitting = ref(false)
const inboundsLoading = ref(false)
const currentNode = ref(null)
const inbounds = ref([])

const form = ref({
  name: '',
  ip: '',
  port: 54321,
  username: '',
  password: '',
  sshPort: 22,
  sshUser: 'root',
  sshPassword: '',
  remark: ''
})

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: 'IP', dataIndex: 'ip', key: 'ip' },
  { title: '端口', dataIndex: 'port', key: 'port' },
  { title: '状态', key: 'status' },
  { title: '启用', key: 'enable' },
  { title: 'Xray版本', dataIndex: 'xrayVersion', key: 'xrayVersion' },
  { title: '操作', key: 'action', width: 180 }
]

const inboundColumns = [
  { title: '备注', dataIndex: 'remark', key: 'remark' },
  { title: '协议', dataIndex: 'protocol', key: 'protocol' },
  { title: '端口', dataIndex: 'port', key: 'port' },
  { title: '状态', key: 'enable' },
  { title: '流量', key: 'traffic' }
]

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const getStatusColor = (status) => {
  if (status === 'online') return 'green'
  if (status === 'offline') return 'red'
  return 'default'
}

const getStatusText = (status) => {
  if (status === 'online') return '在线'
  if (status === 'offline') return '离线'
  return '未知'
}

const fetchNodes = async () => {
  loading.value = true
  try {
    const res = await getNodes()
    nodes.value = res.obj || []
  } catch (e) {
    message.error('获取节点列表失败')
  } finally {
    loading.value = false
  }
}

const showAddModal = () => {
  editingNode.value = null
  form.value = {
    name: '',
    ip: '',
    port: 54321,
    username: '',
    password: '',
    sshPort: 22,
    sshUser: 'root',
    sshPassword: '',
    remark: ''
  }
  modalVisible.value = true
}

const editNode = (node) => {
  editingNode.value = node
  form.value = { ...node }
  modalVisible.value = true
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    if (editingNode.value) {
      await updateNode(editingNode.value.id, form.value)
      message.success('更新成功')
    } else {
      await addNode(form.value)
      message.success('添加成功')
    }
    modalVisible.value = false
    fetchNodes()
  } catch (e) {
    message.error('操作失败')
  } finally {
    submitting.value = false
  }
}

const deleteNodeRecord = async (id) => {
  try {
    await deleteNode(id)
    message.success('删除成功')
    fetchNodes()
  } catch (e) {
    message.error('删除失败')
  }
}

const testConnection = async (node) => {
  try {
    const res = await testNode(node.id)
    if (res.success) {
      message.success('连接成功')
    } else {
      message.error(res.msg || '连接失败')
    }
  } catch (e) {
    message.error('连接失败')
  }
}

const syncStatus = async (node) => {
  try {
    await syncNode(node.id)
    message.success('同步成功')
    fetchNodes()
  } catch (e) {
    message.error('同步失败')
  }
}

const doInstallNode = async (node) => {
  node.installing = true
  try {
    const res = await installNodeApi(node.id)
    if (res.success) {
      message.success('安装成功')
      fetchNodes()
    } else {
      message.error(res.msg || '安装失败')
    }
  } catch (e) {
    message.error('安装失败')
  } finally {
    node.installing = false
  }
}

const showInbounds = async (node) => {
  currentNode.value = node
  inboundsVisible.value = true
  inboundsLoading.value = true
  try {
    const res = await getNodeInbounds(node.id)
    inbounds.value = res.obj || []
  } catch (e) {
    message.error('获取入站列表失败')
  } finally {
    inboundsLoading.value = false
  }
}

const toggleEnable = async (node) => {
  try {
    await updateNode(node.id, { enable: node.enable })
    message.success('状态已更新')
  } catch (e) {
    message.error('更新失败')
  }
}

onMounted(() => {
  fetchNodes()
})
</script>

<style scoped>
.nodes-page {
  /* styles */
}
</style>