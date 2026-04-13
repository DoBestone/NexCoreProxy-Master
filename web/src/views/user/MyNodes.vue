<template>
  <div class="my-nodes-page">
    <!-- 订阅链接卡片 -->
    <div class="subscribe-card">
      <div class="subscribe-header">
        <div class="subscribe-icon">
          <LinkOutlined />
        </div>
        <div class="subscribe-info">
          <h3>订阅链接</h3>
          <p>导入到客户端即可使用所有节点</p>
        </div>
      </div>
      
      <div class="subscribe-url">
        <code>{{ subscribeUrl || '加载中...' }}</code>
        <button class="copy-btn" @click="copySubscribe">
          <CopyOutlined />
        </button>
      </div>
      
      <div class="subscribe-actions">
        <a-button type="primary" @click="copySubscribe">
          <CopyOutlined /> 复制订阅链接
        </a-button>
        <a-button @click="refreshSubscribe" :loading="refreshing">
          <ReloadOutlined /> 刷新订阅
        </a-button>
      </div>
    </div>
    
    <!-- 节点列表 -->
    <div class="nodes-section">
      <div class="section-header">
        <h2>我的节点</h2>
        <span class="node-count">{{ nodes.length }} 个节点</span>
      </div>
      
      <!-- 空状态 -->
      <div v-if="!loading && nodes.length === 0" class="empty-state">
        <CloudServerOutlined class="empty-icon" />
        <p>暂无分配节点</p>
        <a-button type="primary" @click="$router.push('/user/buy')">
          去购买套餐
        </a-button>
      </div>
      
      <!-- 节点网格 -->
      <div v-else class="nodes-grid">
        <div v-for="node in nodes" :key="node.id" class="node-card">
          <div class="node-header">
            <div class="node-name">{{ node.name }}</div>
            <span :class="['status-badge', node.status]">
              {{ node.status === 'online' ? '在线' : '离线' }}
            </span>
          </div>
          
          <div class="node-details">
            <div class="detail-row">
              <span class="label">协议</span>
              <span class="value protocol">{{ node.protocol?.toUpperCase() }}</span>
            </div>
            <div class="detail-row">
              <span class="label">地址</span>
              <span class="value">{{ node.ip }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { CloudServerOutlined, CopyOutlined, ReloadOutlined, LinkOutlined } from '@ant-design/icons-vue'
import { getMyNodes, getMySubscribe } from '@/api'

const nodes = ref([])
const subscribeUrl = ref('')
const loading = ref(false)
const refreshing = ref(false)

const copySubscribe = () => {
  if (subscribeUrl.value) {
    navigator.clipboard.writeText(subscribeUrl.value)
    message.success('订阅链接已复制')
  }
}

const refreshSubscribe = async () => {
  refreshing.value = true
  try {
    const res = await getMySubscribe()
    if (res.success && res.obj) {
      subscribeUrl.value = res.obj.url
      message.success('订阅已刷新')
    }
  } catch (e) {
    message.error('刷新订阅失败')
  } finally {
    refreshing.value = false
  }
}

const fetchNodes = async () => {
  loading.value = true
  try {
    const res = await getMyNodes()
    nodes.value = res.obj || []
  } catch (e) {
    message.error('获取节点失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchNodes()
  refreshSubscribe()
})
</script>

<style scoped>
.my-nodes-page {
  animation: fadeIn 0.3s ease;
}

/* 订阅卡片 */
.subscribe-card {
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  border-radius: 16px;
  padding: 24px;
  margin-bottom: 24px;
  color: white;
}

.subscribe-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

.subscribe-icon {
  width: 48px;
  height: 48px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.subscribe-info h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.subscribe-info p {
  margin: 4px 0 0;
  opacity: 0.85;
  font-size: 13px;
}

.subscribe-url {
  display: flex;
  align-items: center;
  gap: 8px;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 10px;
  padding: 12px 16px;
  margin-bottom: 16px;
}

.subscribe-url code {
  flex: 1;
  font-size: 13px;
  word-break: break-all;
  color: white;
  background: transparent;
}

.subscribe-url .copy-btn {
  width: 36px;
  height: 36px;
  background: rgba(255, 255, 255, 0.2);
  border: none;
  border-radius: 8px;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.subscribe-url .copy-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

.subscribe-actions {
  display: flex;
  gap: 12px;
}

.subscribe-actions .ant-btn {
  border-radius: 10px;
  border: none;
}

.subscribe-actions .ant-btn-primary {
  background: white;
  color: #3b82f6;
  font-weight: 600;
}

.subscribe-actions .ant-btn-primary:hover {
  background: #eff6ff;
}

.subscribe-actions .ant-btn:not(.ant-btn-primary) {
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

/* 节点区域 */
.nodes-section {
  background: white;
  border-radius: 16px;
  padding: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #1e293b;
}

.node-count {
  font-size: 13px;
  color: #64748b;
  background: #f1f5f9;
  padding: 4px 12px;
  border-radius: 20px;
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 48px 24px;
}

.empty-icon {
  font-size: 48px;
  color: #cbd5e1;
  margin-bottom: 16px;
}

.empty-state p {
  color: #64748b;
  margin-bottom: 20px;
}

/* 节点网格 */
.nodes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.node-card {
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #e2e8f0;
  transition: all 0.2s ease;
}

.node-card:hover {
  border-color: #3b82f6;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.1);
}

.node-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.node-name {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
}

.status-badge {
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.status-badge.online {
  background: #f0fdf4;
  color: #16a34a;
}

.status-badge.offline {
  background: #fef2f2;
  color: #dc2626;
}

.node-details {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
}

.detail-row .label {
  color: #64748b;
}

.detail-row .value {
  color: #1e293b;
  font-weight: 500;
}

.detail-row .value.protocol {
  color: #3b82f6;
}

/* 响应式 */
@media (max-width: 576px) {
  .subscribe-card {
    padding: 20px;
  }
  
  .subscribe-actions {
    flex-direction: column;
  }
  
  .subscribe-actions .ant-btn {
    width: 100%;
  }
  
  .nodes-grid {
    grid-template-columns: 1fr;
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>