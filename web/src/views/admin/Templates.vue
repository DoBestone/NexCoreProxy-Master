<template>
  <div class="templates-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">
          <FileTextOutlined class="title-icon" />
          服务模板
        </h1>
        <p class="page-desc">管理入站模板，快速部署节点配置</p>
      </div>
      <a-button type="primary" size="large" @click="showAddModal">
        <template #icon><PlusOutlined /></template>
        添加模板
      </a-button>
    </div>
    
    <!-- 模板列表 -->
    <div class="templates-grid">
      <div 
        v-for="template in templates" 
        :key="template.id" 
        class="template-card"
      >
        <div class="template-header">
          <span :class="['protocol-badge', template.protocol]">
            {{ template.protocol?.toUpperCase() }}
          </span>
          <span class="template-port">:{{ template.port }}</span>
        </div>
        
        <div class="template-name">{{ template.name }}</div>
        
        <div v-if="template.remark" class="template-remark">
          {{ template.remark }}
        </div>
        
        <div class="template-actions">
          <a-button type="primary" size="small" @click="applyTemplate(template)">
            应用到节点
          </a-button>
          <div class="action-btns">
            <button class="action-btn" @click="editTemplate(template)">
              <EditOutlined />
            </button>
            <a-popconfirm title="确定删除此模板?" @confirm="deleteTemplateRecord(template.id)">
              <button class="action-btn danger">
                <DeleteOutlined />
              </button>
            </a-popconfirm>
          </div>
        </div>
      </div>
      
      <!-- 空状态 -->
      <div v-if="!loading && templates.length === 0" class="empty-card" @click="showAddModal">
        <PlusOutlined class="add-icon" />
        <span>添加第一个模板</span>
      </div>
    </div>
    
    <!-- 加载状态 -->
    <div v-if="loading" class="loading-container">
      <a-spin size="large" />
    </div>

    <!-- 添加模板弹窗 -->
    <a-modal 
      v-model:open="modalVisible" 
      :title="editingTemplate ? '编辑模板' : '添加模板'" 
      @ok="handleSubmit" 
      width="700px"
      :confirmLoading="submitting"
    >
      <a-form :model="form" layout="vertical" class="template-form">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="模板名称" required>
              <a-input v-model:value="form.name" placeholder="如: VMess-WS-TLS" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="协议" required>
              <a-select v-model:value="form.protocol" style="width: 100%">
                <a-select-option value="vmess">VMess</a-select-option>
                <a-select-option value="vless">VLESS</a-select-option>
                <a-select-option value="trojan">Trojan</a-select-option>
                <a-select-option value="shadowsocks">Shadowsocks</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="端口">
              <a-input-number v-model:value="form.port" :min="1" :max="65535" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-form-item label="配置 JSON">
          <a-textarea v-model:value="form.settings" :rows="6" placeholder="入站配置 JSON" class="code-textarea" />
        </a-form-item>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="传输层配置">
              <a-textarea v-model:value="form.stream" :rows="4" placeholder="传输层配置 JSON（可选）" class="code-textarea" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="安全传输配置">
              <a-textarea v-model:value="form.tls" :rows="4" placeholder="安全传输配置 JSON（可选）" class="code-textarea" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-form-item label="备注">
          <a-input v-model:value="form.remark" placeholder="模板说明" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 应用模板弹窗 -->
    <a-modal 
      v-model:open="applyVisible" 
      title="应用模板到节点" 
      @ok="handleApply"
      :confirmLoading="applying"
    >
      <div class="apply-info">
        <div class="info-row">
          <span class="label">模板</span>
          <span class="value">{{ currentTemplate?.name }}</span>
        </div>
        <div class="info-row">
          <span class="label">协议</span>
          <span class="value">{{ currentTemplate?.protocol?.toUpperCase() }}</span>
        </div>
      </div>
      
      <a-form layout="vertical" style="margin-top: 16px">
        <a-form-item label="选择目标节点" required>
          <a-select v-model:value="selectedNodeId" style="width: 100%" placeholder="选择要应用的节点">
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
import { ref, onMounted, onDeactivated } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, FileTextOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { getTemplates, addTemplate, deleteTemplate, getNodes, addNodeInbound } from '@/api'

const loading = ref(false)
const templates = ref([])
const nodes = ref([])
const modalVisible = ref(false)
const applyVisible = ref(false)

onDeactivated(() => { modalVisible.value = false; applyVisible.value = false })
const editingTemplate = ref(null)
const currentTemplate = ref(null)
const selectedNodeId = ref(null)
const submitting = ref(false)
const applying = ref(false)

const form = ref({
  name: '',
  protocol: 'vmess',
  port: 443,
  settings: '',
  stream: '',
  tls: '',
  remark: ''
})

const safeJsonParse = (str) => {
  try { return JSON.parse(str || '{}') }
  catch { return {} }
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
  submitting.value = true
  try {
    // 编辑模式：先删旧的再创建新的
    if (editingTemplate.value?.id) {
      await deleteTemplate(editingTemplate.value.id)
    }
    await addTemplate(form.value)
    message.success('保存成功')
    modalVisible.value = false
    fetchData()
  } catch (e) {
    message.error('保存失败')
  } finally {
    submitting.value = false
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
  applying.value = true
  try {
    const inbound = {
      remark: currentTemplate.value.name,
      protocol: currentTemplate.value.protocol,
      port: currentTemplate.value.port,
      settings: safeJsonParse(currentTemplate.value.settings),
      streamSettings: safeJsonParse(currentTemplate.value.stream)
    }
    await addNodeInbound(selectedNodeId.value, inbound)
    message.success('应用成功')
    applyVisible.value = false
  } catch (e) {
    message.error('应用失败')
  } finally {
    applying.value = false
  }
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.templates-page {
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

/* 模板网格 */
.templates-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

/* 模板卡片 */
.template-card {
  background: white;
  border-radius: 14px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  border: 1px solid rgba(0, 0, 0, 0.04);
  transition: all 0.2s ease;
}

.template-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.06);
}

.template-header {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 12px;
}

.protocol-badge {
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 700;
}

.protocol-badge.vmess { background: #eff6ff; color: #3b82f6; }
.protocol-badge.vless { background: #f0fdf4; color: #16a34a; }
.protocol-badge.trojan { background: #fffbeb; color: #b45309; }
.protocol-badge.shadowsocks { background: #e6fffb; color: #08979c; }

.template-port {
  font-size: 14px;
  color: #64748b;
  font-family: 'SF Mono', Monaco, monospace;
}

.template-name {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 8px;
}

.template-remark {
  font-size: 13px;
  color: #64748b;
  margin-bottom: 16px;
}

.template-actions {
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
  min-height: 200px;
}

.empty-card:hover {
  border-color: #3b82f6;
  background: #eff6ff;
}

.empty-card .add-icon {
  font-size: 32px;
  color: #94a3b8;
}

/* 加载状态 */
.loading-container {
  display: flex;
  justify-content: center;
  padding: 48px;
}

/* 表单样式 */
.template-form :deep(.ant-form-item-label > label) {
  font-weight: 500;
}

.code-textarea {
  font-family: 'SF Mono', Monaco, 'Courier New', monospace;
  font-size: 12px;
}

/* 应用弹窗 */
.apply-info {
  background: #f8fafc;
  border-radius: 8px;
  padding: 16px;
}

.info-row {
  display: flex;
  gap: 16px;
}

.info-row .label {
  color: #64748b;
  width: 50px;
}

.info-row .value {
  font-weight: 500;
}

/* 响应式 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
  }
  
  .templates-grid {
    grid-template-columns: 1fr;
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>