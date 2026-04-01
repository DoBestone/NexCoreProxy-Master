<template>
  <div class="buy-package-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1>选择套餐</h1>
      <p>选择适合您的套餐，享受高速稳定的代理服务</p>
    </div>
    
    <!-- 套餐网格 -->
    <div class="packages-grid">
      <div 
        v-for="pkg in packages" 
        :key="pkg.id" 
        class="package-card"
        :class="{ featured: pkg.featured }"
      >
        <div class="package-protocol" :class="pkg.protocol">
          {{ pkg.protocol?.toUpperCase() || 'ALL' }}
        </div>
        
        <div class="package-name">{{ pkg.name }}</div>
        
        <div class="package-price">
          <span class="currency">$</span>
          <span class="amount">{{ pkg.price }}</span>
        </div>
        
        <div class="package-features">
          <div class="feature">
            <DatabaseOutlined />
            <span>流量 {{ pkg.traffic ? formatTraffic(pkg.traffic) : '无限制' }}</span>
          </div>
          <div class="feature">
            <ClockCircleOutlined />
            <span>有效期 {{ pkg.duration ? pkg.duration + '天' : '永久' }}</span>
          </div>
          <div class="feature">
            <CloudServerOutlined />
            <span>{{ pkg.nodes || '不限' }} 个节点</span>
          </div>
        </div>
        
        <button class="buy-btn" @click="buyPackage(pkg)">
          立即购买
        </button>
      </div>
    </div>
    
    <!-- 空状态 -->
    <div v-if="!loading && packages.length === 0" class="empty-state">
      <AppstoreOutlined class="empty-icon" />
      <p>暂无可用套餐</p>
    </div>
    
    <!-- 加载状态 -->
    <div v-if="loading" class="loading-state">
      <a-spin size="large" />
    </div>

    <!-- 购买确认弹窗 -->
    <a-modal 
      v-model:open="buyVisible" 
      title="确认购买" 
      @ok="confirmBuy" 
      :confirmLoading="buying"
      :width="480"
      class="buy-modal"
    >
      <div class="order-summary">
        <div class="summary-item">
          <span class="label">套餐名称</span>
          <span class="value">{{ selectedPackage?.name }}</span>
        </div>
        <div class="summary-item">
          <span class="label">价格</span>
          <span class="value price">${{ selectedPackage?.price }}</span>
        </div>
        <div class="summary-item">
          <span class="label">流量</span>
          <span class="value">{{ selectedPackage?.traffic ? formatTraffic(selectedPackage.traffic) : '无限制' }}</span>
        </div>
        <div class="summary-item">
          <span class="label">有效期</span>
          <span class="value">{{ selectedPackage?.duration ? selectedPackage.duration + '天' : '永久' }}</span>
        </div>
      </div>
      
      <div class="payment-method">
        <div class="method-label">支付方式</div>
        <div class="method-options">
          <div 
            :class="['method-option', { active: payMethod === 'balance' }]"
            @click="payMethod = 'balance'"
          >
            <WalletOutlined />
            <span>余额支付</span>
          </div>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { 
  DatabaseOutlined, ClockCircleOutlined, CloudServerOutlined,
  AppstoreOutlined, WalletOutlined
} from '@ant-design/icons-vue'
import { getPackages, createOrder } from '@/api'

const packages = ref([])
const buyVisible = ref(false)
const selectedPackage = ref(null)
const payMethod = ref('balance')
const buying = ref(false)
const loading = ref(false)

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
    packages.value = (res.obj || []).filter(p => p.enable)
  } catch (e) {
    message.error('获取套餐列表失败')
  } finally {
    loading.value = false
  }
}

const buyPackage = (pkg) => {
  selectedPackage.value = pkg
  buyVisible.value = true
}

const confirmBuy = async () => {
  if (!selectedPackage.value) return
  
  buying.value = true
  try {
    const res = await createOrder({
      packageId: selectedPackage.value.id,
      payMethod: payMethod.value
    })
    if (res.success) {
      message.success('购买成功')
      buyVisible.value = false
    } else {
      message.error(res.msg || '购买失败')
    }
  } catch (e) {
    message.error('创建订单失败')
  } finally {
    buying.value = false
  }
}

onMounted(() => {
  fetchPackages()
})
</script>

<style scoped>
.buy-package-page {
  animation: fadeIn 0.3s ease;
}

.page-header {
  text-align: center;
  margin-bottom: 32px;
}

.page-header h1 {
  font-size: 28px;
  font-weight: 700;
  color: #262626;
  margin: 0 0 8px;
}

.page-header p {
  color: #8c8c8c;
  font-size: 15px;
  margin: 0;
}

/* 套餐网格 */
.packages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 24px;
}

/* 套餐卡片 */
.package-card {
  background: white;
  border-radius: 16px;
  padding: 28px 24px;
  text-align: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  border: 2px solid transparent;
  transition: all 0.2s ease;
  position: relative;
}

.package-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.08);
  border-color: #1677ff;
}

.package-card.featured {
  border-color: #1677ff;
}

.package-protocol {
  display: inline-block;
  padding: 6px 14px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
  margin-bottom: 16px;
}

.package-protocol.vmess { background: #e6f4ff; color: #1677ff; }
.package-protocol.vless { background: #f6ffed; color: #52c41a; }
.package-protocol.trojan { background: #fff7e6; color: #d46b08; }
.package-protocol.shadowsocks { background: #e6fffb; color: #08979c; }
.package-protocol.all, .package-protocol.undefined { background: #f5f5f5; color: #8c8c8c; }

.package-name {
  font-size: 20px;
  font-weight: 700;
  color: #262626;
  margin-bottom: 16px;
}

.package-price {
  margin-bottom: 24px;
}

.package-price .currency {
  font-size: 18px;
  color: #1677ff;
  font-weight: 500;
}

.package-price .amount {
  font-size: 42px;
  font-weight: 700;
  color: #1677ff;
  line-height: 1;
}

.package-features {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 24px;
  padding: 20px 0;
  border-top: 1px solid #f0f0f0;
  border-bottom: 1px solid #f0f0f0;
}

.feature {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 14px;
  color: #595959;
}

.feature .anticon {
  color: #8c8c8c;
}

.buy-btn {
  width: 100%;
  padding: 14px;
  background: linear-gradient(135deg, #1677ff 0%, #4096ff 100%);
  border: none;
  border-radius: 10px;
  color: white;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.buy-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(22, 119, 255, 0.35);
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 64px 24px;
  background: white;
  border-radius: 16px;
}

.empty-icon {
  font-size: 48px;
  color: #d9d9d9;
  margin-bottom: 16px;
}

.empty-state p {
  color: #8c8c8c;
}

/* 加载状态 */
.loading-state {
  display: flex;
  justify-content: center;
  padding: 64px;
}

/* 购买弹窗 */
.order-summary {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 24px;
  padding: 20px;
  background: #f8fafc;
  border-radius: 12px;
}

.summary-item {
  display: flex;
  justify-content: space-between;
}

.summary-item .label {
  color: #8c8c8c;
}

.summary-item .value {
  font-weight: 500;
  color: #262626;
}

.summary-item .value.price {
  font-size: 18px;
  color: #ff4d4f;
  font-weight: 700;
}

.payment-method {
  margin-top: 8px;
}

.method-label {
  font-size: 13px;
  color: #8c8c8c;
  margin-bottom: 12px;
}

.method-options {
  display: flex;
  gap: 12px;
}

.method-option {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px;
  background: #f8fafc;
  border: 2px solid transparent;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.method-option:hover {
  border-color: #d9d9d9;
}

.method-option.active {
  border-color: #1677ff;
  background: #e6f4ff;
}

.method-option .anticon {
  font-size: 24px;
  color: #1677ff;
}

/* 响应式 */
@media (max-width: 768px) {
  .packages-grid {
    grid-template-columns: 1fr;
  }
  
  .page-header h1 {
    font-size: 24px;
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>