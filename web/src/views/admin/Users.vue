<template>
  <div class="users-page">
    <a-card title="用户管理">
      <template #extra>
        <a-button type="primary" @click="showAddModal">
          <template #icon><PlusOutlined /></template>
          添加用户
        </a-button>
      </template>
      <a-table :columns="columns" :dataSource="users" :loading="loading" rowKey="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'role'">
            <a-tag :color="record.role === 'admin' ? 'red' : 'blue'">
              {{ record.role === 'admin' ? '管理员' : '用户' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'enable'">
            <a-switch v-model:checked="record.enable" @change="toggleEnable(record)" />
          </template>
          <template v-if="column.key === 'balance'">
            ¥{{ record.balance?.toFixed(2) || '0.00' }}
          </template>
          <template v-if="column.key === 'traffic'">
            {{ formatTraffic(record.trafficUsed) }} / {{ formatTraffic(record.trafficLimit) || '无限' }}
          </template>
          <template v-if="column.key === 'expireAt'">
            {{ record.expireAt ? formatDate(record.expireAt) : '永久' }}
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="editUser(record)">编辑</a-button>
              <a-button type="link" size="small" @click="showRechargeModal(record)">充值</a-button>
              <a-popconfirm title="确定删除?" @confirm="deleteUserRecord(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 添加/编辑用户弹窗 -->
    <a-modal v-model:open="modalVisible" :title="editingUser ? '编辑用户' : '添加用户'" @ok="handleSubmit" :confirmLoading="submitting">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="用户名" required>
          <a-input v-model:value="form.username" placeholder="用户名" :disabled="!!editingUser" />
        </a-form-item>
        <a-form-item label="密码" :required="!editingUser">
          <a-input-password v-model:value="form.password" :placeholder="editingUser ? '留空不修改' : '密码'" />
        </a-form-item>
        <a-form-item label="邮箱">
          <a-input v-model:value="form.email" placeholder="邮箱" />
        </a-form-item>
        <a-form-item label="角色">
          <a-select v-model:value="form.role" style="width: 100%">
            <a-select-option value="user">用户</a-select-option>
            <a-select-option value="admin">管理员</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="数据限额">
          <a-input-number v-model:value="form.trafficLimit" :min="0" style="width: 100%" addon-after="GB" />
        </a-form-item>
        <a-form-item label="备注">
          <a-textarea v-model:value="form.remark" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 充值弹窗 -->
    <a-modal v-model:open="rechargeVisible" title="用户充值" @ok="handleRecharge" :confirmLoading="recharging">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="当前余额">
          ¥{{ currentUser?.balance?.toFixed(2) || '0.00' }}
        </a-form-item>
        <a-form-item label="充值金额">
          <a-input-number v-model:value="rechargeAmount" :min="0" style="width: 200px" addon-before="¥" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { getUsers, addUser, updateUser, deleteUser } from '@/api'

const loading = ref(false)
const users = ref([])
const modalVisible = ref(false)
const rechargeVisible = ref(false)
const editingUser = ref(null)
const submitting = ref(false)
const recharging = ref(false)
const rechargeAmount = ref(0)
const currentUser = ref(null)

const form = ref({
  username: '',
  password: '',
  email: '',
  role: 'user',
  trafficLimit: 0,
  remark: ''
})

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '邮箱', dataIndex: 'email', key: 'email' },
  { title: '角色', key: 'role', width: 80 },
  { title: '余额', key: 'balance', width: 100 },
  { title: '数据量', key: 'traffic' },
  { title: '到期时间', key: 'expireAt' },
  { title: '启用', key: 'enable', width: 80 },
  { title: '操作', key: 'action', width: 150 }
]

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatDate = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleDateString('zh-CN')
}

const fetchUsers = async () => {
  loading.value = true
  try {
    const res = await getUsers()
    users.value = res.obj || []
  } catch (e) {
    message.error('获取用户列表失败')
  } finally {
    loading.value = false
  }
}

const showAddModal = () => {
  editingUser.value = null
  form.value = { username: '', password: '', email: '', role: 'user', trafficLimit: 0, remark: '' }
  modalVisible.value = true
}

const editUser = (user) => {
  editingUser.value = user
  form.value = { ...user, password: '' }
  modalVisible.value = true
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    if (editingUser.value) {
      await updateUser(editingUser.value.id, form.value)
      message.success('更新成功')
    } else {
      await addUser(form.value)
      message.success('添加成功')
    }
    modalVisible.value = false
    fetchUsers()
  } catch (e) {
    message.error('操作失败')
  } finally {
    submitting.value = false
  }
}

const deleteUserRecord = async (id) => {
  try {
    await deleteUser(id)
    message.success('删除成功')
    fetchUsers()
  } catch (e) {
    message.error('删除失败')
  }
}

const toggleEnable = async (user) => {
  try {
    await updateUser(user.id, { enable: user.enable })
    message.success('状态已更新')
  } catch (e) {
    message.error('更新失败')
  }
}

const showRechargeModal = (user) => {
  currentUser.value = user
  rechargeAmount.value = 0
  rechargeVisible.value = true
}

const handleRecharge = async () => {
  recharging.value = true
  try {
    const newBalance = (currentUser.value.balance || 0) + rechargeAmount.value
    await updateUser(currentUser.value.id, { balance: newBalance })
    message.success('充值成功')
    rechargeVisible.value = false
    fetchUsers()
  } catch (e) {
    message.error('充值失败')
  } finally {
    recharging.value = false
  }
}

onMounted(() => {
  fetchUsers()
})
</script>