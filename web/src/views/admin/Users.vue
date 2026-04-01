<template>
  <div class="users-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">
          <TeamOutlined class="title-icon" />
          用户管理
        </h1>
        <p class="page-desc">管理系统用户，设置权限和余额</p>
      </div>
      <a-button type="primary" size="large" @click="showAddModal">
        <template #icon><PlusOutlined /></template>
        添加用户
      </a-button>
    </div>
    
    <!-- 用户列表卡片 -->
    <a-card class="users-card">
      <a-table 
        :columns="columns" 
        :dataSource="users" 
        :loading="loading" 
        rowKey="id"
        :pagination="{ pageSize: 10, showSizeChanger: true }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'user'">
            <div class="user-info">
              <a-avatar :size="36" class="user-avatar">
                {{ record.username?.charAt(0)?.toUpperCase() }}
              </a-avatar>
              <div class="user-detail">
                <span class="user-name">{{ record.username }}</span>
                <span class="user-email">{{ record.email || '-' }}</span>
              </div>
            </div>
          </template>
          <template v-if="column.key === 'role'">
            <span :class="['role-badge', record.role]">
              {{ record.role === 'admin' ? '管理员' : '用户' }}
            </span>
          </template>
          <template v-if="column.key === 'balance'">
            <span class="balance">${{ record.balance?.toFixed(2) || '0.00' }}</span>
          </template>
          <template v-if="column.key === 'traffic'">
            <div class="traffic-cell">
              <span class="used">{{ formatTraffic(record.trafficUsed) }}</span>
              <span class="divider">/</span>
              <span class="limit">{{ formatTraffic(record.trafficLimit) || '无限' }}</span>
            </div>
          </template>
          <template v-if="column.key === 'expireAt'">
            <span :class="['expire-date', { expired: isExpired(record.expireAt) }]">
              {{ record.expireAt ? formatDate(record.expireAt) : '永久' }}
            </span>
          </template>
          <template v-if="column.key === 'enable'">
            <a-switch 
              v-model:checked="record.enable" 
              @change="toggleEnable(record)"
              size="small"
            />
          </template>
          <template v-if="column.key === 'action'">
            <div class="action-btns">
              <a-tooltip title="编辑">
                <button class="action-btn" @click="editUser(record)">
                  <EditOutlined />
                </button>
              </a-tooltip>
              <a-tooltip title="充值">
                <button class="action-btn" @click="showRechargeModal(record)">
                  <DollarOutlined />
                </button>
              </a-tooltip>
              <a-popconfirm title="确定删除此用户?" @confirm="deleteUserRecord(record.id)">
                <button class="action-btn danger">
                  <DeleteOutlined />
                </button>
              </a-popconfirm>
            </div>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 添加/编辑用户弹窗 -->
    <a-modal 
      v-model:open="modalVisible" 
      :title="editingUser ? '编辑用户' : '添加用户'" 
      @ok="handleSubmit" 
      :confirmLoading="submitting"
      :width="480"
    >
      <a-form :model="form" layout="vertical" class="user-form">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="用户名" required>
              <a-input v-model:value="form.username" placeholder="用户名" :disabled="!!editingUser" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="密码" :required="!editingUser">
              <a-input-password v-model:value="form.password" :placeholder="editingUser ? '留空不修改' : '密码'" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="邮箱">
              <a-input v-model:value="form.email" placeholder="邮箱地址" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="角色">
              <a-select v-model:value="form.role" style="width: 100%">
                <a-select-option value="user">普通用户</a-select-option>
                <a-select-option value="admin">管理员</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="数据限额">
          <a-input-number v-model:value="form.trafficLimit" :min="0" style="width: 100%" addon-after="GB" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="余额">
              <a-input-number v-model:value="form.balance" :min="0" :precision="2" style="width: 100%" addon-before="$" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="状态">
              <a-switch v-model:checked="form.enable" checked-children="启用" un-checked-children="禁用" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="备注">
          <a-textarea v-model:value="form.remark" :rows="2" placeholder="备注信息" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 充值弹窗 -->
    <a-modal 
      v-model:open="rechargeVisible" 
      title="用户充值" 
      @ok="handleRecharge" 
      :confirmLoading="recharging"
      :width="400"
    >
      <div class="recharge-info">
        <div class="recharge-row">
          <span class="label">用户</span>
          <span class="value">{{ currentUser?.username }}</span>
        </div>
        <div class="recharge-row">
          <span class="label">当前余额</span>
          <span class="value balance">${{ currentUser?.balance?.toFixed(2) || '0.00' }}</span>
        </div>
        <div class="recharge-row">
          <span class="label">充值金额</span>
          <a-input-number 
            v-model:value="rechargeAmount" 
            :min="0" 
            :precision="2"
            style="width: 150px" 
            addon-before="$" 
          />
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, TeamOutlined, EditOutlined, DeleteOutlined, DollarOutlined } from '@ant-design/icons-vue'
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
  balance: 0,
  trafficLimit: 0,
  enable: true,
  remark: ''
})

const columns = [
  { title: '用户信息', key: 'user' },
  { title: '角色', key: 'role', width: 100 },
  { title: '余额', key: 'balance', width: 100 },
  { title: '流量使用', key: 'traffic' },
  { title: '到期时间', key: 'expireAt' },
  { title: '启用', key: 'enable', width: 80 },
  { title: '操作', key: 'action', width: 130, fixed: 'right' }
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

const isExpired = (date) => {
  if (!date) return false
  return new Date(date) < new Date()
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
  form.value = { username: '', password: '', email: '', role: 'user', balance: 0, trafficLimit: 0, enable: true, remark: '' }
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

<style scoped>
.users-page {
  animation: fadeIn 0.3s ease;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
  gap: 16px;
}

.page-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 22px;
  font-weight: 700;
  color: #262626;
  margin: 0;
}

.title-icon {
  color: #1677ff;
  font-size: 24px;
}

.page-desc {
  color: #8c8c8c;
  font-size: 14px;
  margin-top: 4px;
}

.users-card {
  border-radius: 14px;
}

/* 用户信息 */
.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  background: linear-gradient(135deg, #1677ff 0%, #4096ff 100%);
  flex-shrink: 0;
}

.user-detail {
  display: flex;
  flex-direction: column;
}

.user-name {
  font-weight: 600;
  color: #262626;
}

.user-email {
  font-size: 12px;
  color: #8c8c8c;
}

/* 角色徽章 */
.role-badge {
  display: inline-flex;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.role-badge.admin {
  background: #fff2f0;
  color: #ff4d4f;
}

.role-badge.user {
  background: #e6f4ff;
  color: #1677ff;
}

/* 余额 */
.balance {
  font-weight: 600;
  color: #1677ff;
}

/* 流量 */
.traffic-cell {
  display: flex;
  gap: 4px;
  font-size: 13px;
}

.traffic-cell .used {
  color: #1677ff;
  font-weight: 500;
}

.traffic-cell .divider {
  color: #d9d9d9;
}

.traffic-cell .limit {
  color: #8c8c8c;
}

/* 到期时间 */
.expire-date {
  font-size: 13px;
}

.expire-date.expired {
  color: #ff4d4f;
}

/* 操作按钮 */
.action-btns {
  display: flex;
  gap: 4px;
}

.action-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f5f5;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #595959;
  transition: all 0.15s ease;
}

.action-btn:hover {
  background: #e6f4ff;
  color: #1677ff;
}

.action-btn.danger:hover {
  background: #fff2f0;
  color: #ff4d4f;
}

/* 充值弹窗 */
.recharge-info {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.recharge-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.recharge-row .label {
  color: #8c8c8c;
}

.recharge-row .value {
  font-weight: 500;
}

.recharge-row .value.balance {
  font-size: 18px;
  color: #1677ff;
}

/* 响应式 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
  }
  
  .page-header .ant-btn {
    width: 100%;
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>