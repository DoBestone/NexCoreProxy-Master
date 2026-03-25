<template>
  <div class="packages-page">
    <a-card title="套餐管理">
      <template #extra>
        <a-button type="primary" @click="showAddModal">
          <template #icon><PlusOutlined /></template>
          添加套餐
        </a-button>
      </template>
      <a-table :columns="columns" :dataSource="packages" :loading="loading" rowKey="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'traffic'">
            {{ record.traffic ? formatTraffic(record.traffic) : '无限制' }}
          </template>
          <template v-if="column.key === 'duration'">
            {{ record.duration ? record.duration + '天' : '永久' }}
          </template>
          <template v-if="column.key === 'price'">
            ¥{{ record.price }}
          </template>
          <template v-if="column.key === 'enable'">
            <a-switch v-model:checked="record.enable" @change="toggleEnable(record)" />
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="editPackage(record)">编辑</a-button>
              <a-popconfirm title="确定删除?" @confirm="deletePackageRecord(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 添加/编辑套餐弹窗 -->
    <a-modal v-model:open="modalVisible" :title="editingPackage ? '编辑套餐' : '添加套餐'" @ok="handleSubmit" :confirmLoading="submitting">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="套餐名称" required>
          <a-input v-model:value="form.name" placeholder="如: 基础版" />
        </a-form-item>
        <a-form-item label="协议类型">
          <a-select v-model:value="form.protocol" style="width: 100%">
            <a-select-option value="all">全部</a-select-option>
            <a-select-option value="vmess">VMess</a-select-option>
            <a-select-option value="vless">VLESS</a-select-option>
            <a-select-option value="trojan">Trojan</a-select-option>
            <a-select-option value="shadowsocks">Shadowsocks</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="数据量">
          <a-input-number v-model:value="form.trafficGB" :min="0" style="width: 100%" addon-after="GB" />
          <span style="color: #999; font-size: 12px">0 表示无限制</span>
        </a-form-item>
        <a-form-item label="有效期">
          <a-input-number v-model:value="form.duration" :min="0" style="width: 100%" addon-after="天" />
          <span style="color: #999; font-size: 12px">0 表示永久</span>
        </a-form-item>
        <a-form-item label="价格" required>
          <a-input-number v-model:value="form.price" :min="0" :precision="2" style="width: 100%" addon-before="¥" />
        </a-form-item>
        <a-form-item label="服务数量">
          <a-input-number v-model:value="form.nodes" :min="0" style="width: 100%" />
          <span style="color: #999; font-size: 12px">0 表示不限制</span>
        </a-form-item>
        <a-form-item label="排序">
          <a-input-number v-model:value="form.sort" :min="0" style="width: 100%" />
        </a-form-item>
        <a-form-item label="备注">
          <a-textarea v-model:value="form.remark" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { getPackages, addPackage, updatePackage, deletePackage } from '@/api'

const loading = ref(false)
const packages = ref([])
const modalVisible = ref(false)
const editingPackage = ref(null)
const submitting = ref(false)

const form = ref({
  name: '',
  protocol: 'all',
  trafficGB: 0,
  duration: 30,
  price: 0,
  nodes: 0,
  sort: 0,
  remark: ''
})

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '协议', dataIndex: 'protocol', key: 'protocol' },
  { title: '数据量', key: 'traffic' },
  { title: '有效期', key: 'duration' },
  { title: '价格', key: 'price' },
  { title: '服务数', dataIndex: 'nodes', key: 'nodes' },
  { title: '启用', key: 'enable' },
  { title: '操作', key: 'action', width: 120 }
]

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const fetchPackages = async () => {
  loading.value = true
  try {
    const res = await getPackages()
    packages.value = res.obj || []
  } catch (e) {
    message.error('获取套餐列表失败')
  } finally {
    loading.value = false
  }
}

const showAddModal = () => {
  editingPackage.value = null
  form.value = { name: '', protocol: 'all', trafficGB: 0, duration: 30, price: 0, nodes: 0, sort: 0, remark: '' }
  modalVisible.value = true
}

const editPackage = (pkg) => {
  editingPackage.value = pkg
  form.value = {
    ...pkg,
    trafficGB: pkg.traffic ? pkg.traffic / (1024 * 1024 * 1024) : 0
  }
  modalVisible.value = true
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    const data = {
      ...form.value,
      traffic: form.value.trafficGB * 1024 * 1024 * 1024
    }
    if (editingPackage.value) {
      await updatePackage(editingPackage.value.id, data)
      message.success('更新成功')
    } else {
      await addPackage(data)
      message.success('添加成功')
    }
    modalVisible.value = false
    fetchPackages()
  } catch (e) {
    message.error('操作失败')
  } finally {
    submitting.value = false
  }
}

const deletePackageRecord = async (id) => {
  try {
    await deletePackage(id)
    message.success('删除成功')
    fetchPackages()
  } catch (e) {
    message.error('删除失败')
  }
}

const toggleEnable = async (pkg) => {
  try {
    await updatePackage(pkg.id, { enable: pkg.enable })
    message.success('状态已更新')
  } catch (e) {
    message.error('更新失败')
  }
}

onMounted(() => {
  fetchPackages()
})
</script>