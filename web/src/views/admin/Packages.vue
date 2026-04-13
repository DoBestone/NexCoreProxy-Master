<template>
  <div class="packages-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">
          <AppstoreOutlined class="title-icon" />
          套餐管理
        </h1>
        <p class="page-desc">配置销售套餐，设置流量、时长和价格</p>
      </div>
      <a-button type="primary" size="large" @click="showAddModal">
        <template #icon><PlusOutlined /></template>
        添加套餐
      </a-button>
    </div>
    
    <!-- 套餐列表 -->
    <div class="packages-grid">
      <div 
        v-for="pkg in packages" 
        :key="pkg.id" 
        class="package-card"
        :class="{ disabled: !pkg.enable }"
      >
        <div class="package-header">
          <span class="package-name">{{ pkg.name }}</span>
          <span :class="['protocol-badge', pkg.protocol]">{{ pkg.protocol?.toUpperCase() || 'ALL' }}</span>
        </div>
        
        <div class="package-price">
          <span class="price-symbol">$</span>
          <span class="price-value">{{ pkg.price }}</span>
        </div>
        
        <div class="package-features">
          <div class="feature-item">
            <DatabaseOutlined class="feature-icon" />
            <span>流量: {{ pkg.traffic ? formatTraffic(pkg.traffic) : '无限制' }}</span>
          </div>
          <div class="feature-item">
            <ClockCircleOutlined class="feature-icon" />
            <span>有效期: {{ pkg.duration ? pkg.duration + '天' : '永久' }}</span>
          </div>
          <div class="feature-item">
            <CloudServerOutlined class="feature-icon" />
            <span>节点数: {{ pkg.nodes || '不限' }}</span>
          </div>
        </div>
        
        <div class="package-actions">
          <a-switch 
            v-model:checked="pkg.enable" 
            @change="toggleEnable(pkg)"
            size="small"
          />
          <div class="action-btns">
            <button class="action-btn" @click="editPackage(pkg)">
              <EditOutlined />
            </button>
            <a-popconfirm title="确定删除此套餐?" @confirm="deletePackageRecord(pkg.id)">
              <button class="action-btn danger">
                <DeleteOutlined />
              </button>
            </a-popconfirm>
          </div>
        </div>
      </div>
      
      <!-- 空状态 -->
      <div v-if="!loading && packages.length === 0" class="empty-card" @click="showAddModal">
        <PlusOutlined class="add-icon" />
        <span>添加第一个套餐</span>
      </div>
    </div>
    
    <!-- 加载状态 -->
    <div v-if="loading" class="loading-container">
      <a-spin size="large" />
    </div>

    <!-- 添加/编辑套餐弹窗 -->
    <a-modal 
      v-model:open="modalVisible" 
      :title="editingPackage ? '编辑套餐' : '添加套餐'" 
      @ok="handleSubmit" 
      :confirmLoading="submitting"
      :width="520"
    >
      <a-form :model="form" layout="vertical" class="package-form">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="套餐名称" required>
              <a-input v-model:value="form.name" placeholder="如: 基础版" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="协议类型">
              <a-select v-model:value="form.protocol" style="width: 100%">
                <a-select-option value="all">全部</a-select-option>
                <a-select-option value="vmess">VMess</a-select-option>
                <a-select-option value="vless">VLESS</a-select-option>
                <a-select-option value="trojan">Trojan</a-select-option>
                <a-select-option value="shadowsocks">Shadowsocks</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="数据量">
              <a-input-number v-model:value="form.trafficGB" :min="0" style="width: 100%" addon-after="GB" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="有效期">
              <a-input-number v-model:value="form.duration" :min="0" style="width: 100%" addon-after="天" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="价格" required>
              <a-input-number v-model:value="form.price" :min="0" :precision="2" style="width: 100%" addon-before="$" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="服务数量">
              <a-input-number v-model:value="form.nodes" :min="0" style="width: 100%" />
              <span class="form-hint">0 表示不限制</span>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="排序">
              <a-input-number v-model:value="form.sort" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-form-item label="备注">
          <a-textarea v-model:value="form.remark" :rows="2" placeholder="套餐说明" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, onDeactivated } from 'vue'
import { message } from 'ant-design-vue'
import { 
  PlusOutlined, AppstoreOutlined, EditOutlined, DeleteOutlined,
  DatabaseOutlined, ClockCircleOutlined, CloudServerOutlined
} from '@ant-design/icons-vue'
import { getAllPackages, addPackage, updatePackage, deletePackage } from '@/api'

const loading = ref(false)
const packages = ref([])
const modalVisible = ref(false)
const editingPackage = ref(null)

onDeactivated(() => { modalVisible.value = false })
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
    const res = await getAllPackages()
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

<style scoped>
.packages-page {
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
  color: #1e293b;
  margin: 0;
}

.title-icon {
  color: #3b82f6;
  font-size: 24px;
}

.page-desc {
  color: #64748b;
  font-size: 14px;
  margin-top: 4px;
}

/* 套餐网格 */
.packages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
}

/* 套餐卡片 */
.package-card {
  background: white;
  border-radius: 14px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  border: 1px solid rgba(0, 0, 0, 0.04);
  transition: all 0.2s ease;
}

.package-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.06);
}

.package-card.disabled {
  opacity: 0.6;
}

.package-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.package-name {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
}

.protocol-badge {
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
  background: #f1f5f9;
  color: #64748b;
}

.protocol-badge.all { background: #eff6ff; color: #3b82f6; }
.protocol-badge.vmess { background: #f9f0ff; color: #7c3aed; }
.protocol-badge.vless { background: #fffbeb; color: #b45309; }
.protocol-badge.trojan { background: #f0fdf4; color: #15803d; }
.protocol-badge.shadowsocks { background: #e6fffb; color: #08979c; }

.package-price {
  margin-bottom: 20px;
}

.price-symbol {
  font-size: 18px;
  color: #3b82f6;
  font-weight: 500;
}

.price-value {
  font-size: 36px;
  font-weight: 700;
  color: #3b82f6;
}

.package-features {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 20px;
  padding-top: 16px;
  border-top: 1px solid #e2e8f0;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #475569;
}

.feature-icon {
  color: #64748b;
}

.package-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 1px solid #e2e8f0;
}

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
  background: #f1f5f9;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #475569;
  transition: all 0.15s ease;
}

.action-btn:hover {
  background: #eff6ff;
  color: #3b82f6;
}

.action-btn.danger:hover {
  background: #fef2f2;
  color: #dc2626;
}

/* 空状态卡片 */
.empty-card {
  background: white;
  border-radius: 14px;
  padding: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  border: 2px dashed #cbd5e1;
  cursor: pointer;
  transition: all 0.2s ease;
  min-height: 240px;
}

.empty-card:hover {
  border-color: #3b82f6;
  background: #eff6ff;
}

.empty-card .add-icon {
  font-size: 32px;
  color: #94a3b8;
}

.empty-card span {
  color: #64748b;
}

/* 加载状态 */
.loading-container {
  display: flex;
  justify-content: center;
  padding: 48px;
}

/* 表单提示 */
.form-hint {
  font-size: 12px;
  color: #64748b;
}

/* 响应式 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
  }
  
  .packages-grid {
    grid-template-columns: 1fr;
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>