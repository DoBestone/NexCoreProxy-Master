<template>
  <div class="buy-package">
    <a-card title="套餐选购">
      <a-row :gutter="16">
        <a-col :xs="24" :sm="12" :lg="8" v-for="pkg in packages" :key="pkg.id">
          <a-card class="package-card" hoverable>
            <template #cover>
              <div class="package-header" :style="{ background: getPackageColor(pkg.protocol) }">
                <h3>{{ pkg.name }}</h3>
                <div class="price">¥{{ pkg.price }}</div>
              </div>
            </template>
            <a-card-meta>
              <template #description>
                <div class="package-info">
                  <p><strong>数据量:</strong> {{ pkg.traffic ? formatTraffic(pkg.traffic) : '无限制' }}</p>
                  <p><strong>有效期:</strong> {{ pkg.duration ? pkg.duration + '天' : '永久' }}</p>
                  <p><strong>服务数量:</strong> {{ pkg.nodes || '不限' }}</p>
                  <p class="remark">{{ pkg.remark }}</p>
                </div>
                <a-button type="primary" block @click="buyPackage(pkg)" style="margin-top: 16px">
                  立即购买
                </a-button>
              </template>
            </a-card-meta>
          </a-card>
        </a-col>
      </a-row>
      <a-empty v-if="packages.length === 0" description="暂无可用套餐" />
    </a-card>

    <!-- 购买确认弹窗 -->
    <a-modal v-model:open="buyVisible" title="确认购买" @ok="confirmBuy" :confirmLoading="buying">
      <a-descriptions :column="1" bordered>
        <a-descriptions-item label="套餐名称">{{ selectedPackage?.name }}</a-descriptions-item>
        <a-descriptions-item label="价格">¥{{ selectedPackage?.price }}</a-descriptions-item>
        <a-descriptions-item label="数据量">{{ selectedPackage?.traffic ? formatTraffic(selectedPackage.traffic) : '无限制' }}</a-descriptions-item>
        <a-descriptions-item label="有效期">{{ selectedPackage?.duration ? selectedPackage.duration + '天' : '永久' }}</a-descriptions-item>
      </a-descriptions>
      <a-form style="margin-top: 16px">
        <a-form-item label="支付方式">
          <a-radio-group v-model:value="payMethod">
            <a-radio value="balance">余额支付</a-radio>
            <a-radio value="alipay">支付宝</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getPackages, createOrder } from '@/api'

const packages = ref([])
const buyVisible = ref(false)
const selectedPackage = ref(null)
const payMethod = ref('balance')
const buying = ref(false)

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const getPackageColor = (protocol) => {
  const colors = {
    vmess: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    vless: 'linear-gradient(135deg, #11998e 0%, #38ef7d 100%)',
    trojan: 'linear-gradient(135deg, #fc4a1a 0%, #f7b733 100%)',
    shadowsocks: 'linear-gradient(135deg, #ee0979 0%, #ff6a00 100%)'
  }
  return colors[protocol] || 'linear-gradient(135deg, #1890ff 0%, #36cfc9 100%)'
}

const fetchPackages = async () => {
  try {
    const res = await getPackages()
    packages.value = res.obj || []
  } catch (e) {
    message.error('获取套餐列表失败')
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
      message.success('订单创建成功')
      buyVisible.value = false
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
.buy-package {
  max-width: 1200px;
  margin: 0 auto;
}

.package-card {
  margin-bottom: 16px;
  border-radius: 8px;
  overflow: hidden;
}

.package-header {
  padding: 24px;
  text-align: center;
  color: #fff;
}

.package-header h3 {
  color: #fff;
  margin-bottom: 8px;
}

.price {
  font-size: 28px;
  font-weight: bold;
}

.package-info p {
  margin-bottom: 8px;
}

.remark {
  color: #999;
  font-size: 12px;
}
</style>