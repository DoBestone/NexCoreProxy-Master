<template>
  <div class="buy-page">
    <!-- 顶部说明条 -->
    <section class="intro">
      <div class="intro-main">
        <h2 class="intro-title">选择适合你的套餐</h2>
        <p class="intro-sub">随时续费、流量到期提醒，支持账户余额一键支付</p>
      </div>
      <div class="intro-badge">
        <span class="dot dot-ok"></span>
        <span class="mono">{{ packages.length }}</span> 个可选方案
      </div>
    </section>

    <!-- 加载 / 空态 -->
    <div v-if="loading" class="state">
      <a-spin />
    </div>

    <div v-else-if="packages.length === 0" class="state state-empty">
      <AppstoreOutlined class="state-icon" />
      <p class="state-title">暂无可用套餐</p>
      <p class="state-sub">管理员尚未配置公开套餐</p>
    </div>

    <!-- 套餐网格 -->
    <section v-else class="pkg-grid">
      <article
        v-for="(pkg, idx) in packages"
        :key="pkg.id"
        class="pkg-card"
        :class="{ 'is-featured': pkg.featured }"
        :style="{ animationDelay: (idx * 60) + 'ms' }"
      >
        <div v-if="pkg.featured" class="pkg-ribbon">推荐</div>

        <div class="pkg-top">
          <span class="pkg-proto" :class="(pkg.protocol || 'all').toLowerCase()">
            {{ (pkg.protocol || 'ALL').toUpperCase() }}
          </span>
          <h3 class="pkg-name">{{ pkg.name }}</h3>
          <p v-if="pkg.description" class="pkg-desc">{{ pkg.description }}</p>
        </div>

        <div class="pkg-price">
          <span class="currency">¥</span>
          <span class="amount mono">{{ pkg.price }}</span>
          <span class="per" v-if="pkg.duration">/ {{ pkg.duration }} 天</span>
        </div>

        <ul class="pkg-features">
          <li>
            <DatabaseOutlined class="f-icon" />
            <span class="f-label">流量</span>
            <span class="f-value mono">{{ pkg.traffic ? formatTraffic(pkg.traffic) : '无限制' }}</span>
          </li>
          <li>
            <ClockCircleOutlined class="f-icon" />
            <span class="f-label">有效期</span>
            <span class="f-value mono">{{ pkg.duration ? pkg.duration + ' 天' : '永久' }}</span>
          </li>
          <li>
            <CloudServerOutlined class="f-icon" />
            <span class="f-label">节点数</span>
            <span class="f-value mono">{{ pkg.nodes || '不限' }}</span>
          </li>
        </ul>

        <button
          class="pkg-btn"
          :class="{ 'is-featured': pkg.featured }"
          @click="openBuy(pkg)"
        >立即购买</button>
      </article>
    </section>

    <!-- 购买确认弹窗 -->
    <a-modal
      v-model:open="buyVisible"
      :footer="null"
      :width="460"
      :closable="true"
      :maskClosable="!buying"
      centered
      class="buy-modal"
    >
      <div class="modal-head">
        <h3 class="modal-title">确认订单</h3>
        <p class="modal-sub">请核对套餐信息后完成支付</p>
      </div>

      <div class="summary">
        <div class="summary-row">
          <span class="summary-key">套餐</span>
          <span class="summary-val">{{ selectedPackage?.name }}</span>
        </div>
        <div class="summary-row">
          <span class="summary-key">流量</span>
          <span class="summary-val mono">
            {{ selectedPackage?.traffic ? formatTraffic(selectedPackage.traffic) : '无限制' }}
          </span>
        </div>
        <div class="summary-row">
          <span class="summary-key">有效期</span>
          <span class="summary-val mono">
            {{ selectedPackage?.duration ? selectedPackage.duration + ' 天' : '永久' }}
          </span>
        </div>
        <div class="summary-row summary-row-total">
          <span class="summary-key">应付</span>
          <span class="summary-total mono">¥{{ selectedPackage?.price }}</span>
        </div>
      </div>

      <div class="pay-title">支付方式</div>
      <div class="pay-options">
        <label
          class="pay-option"
          :class="{ 'is-active': payMethod === 'balance' }"
          @click="payMethod = 'balance'"
        >
          <WalletOutlined class="pay-icon" />
          <div class="pay-body">
            <span class="pay-name">账户余额</span>
            <span class="pay-sub">即时到账</span>
          </div>
          <span class="pay-check"></span>
        </label>
      </div>

      <div class="modal-foot">
        <a-button @click="buyVisible = false" :disabled="buying">取消</a-button>
        <a-button type="primary" :loading="buying" @click="confirmBuy">
          确认支付 ¥{{ selectedPackage?.price }}
        </a-button>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, onDeactivated } from 'vue'
import { message } from 'ant-design-vue'
import {
  DatabaseOutlined,
  ClockCircleOutlined,
  CloudServerOutlined,
  AppstoreOutlined,
  WalletOutlined
} from '@ant-design/icons-vue'
import { getPackages, createOrder } from '@/api'

const packages = ref([])
const loading = ref(false)
const buyVisible = ref(false)
const selectedPackage = ref(null)
const payMethod = ref('balance')
const buying = ref(false)

onDeactivated(() => { buyVisible.value = false })

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let n = bytes
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++ }
  return `${n.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
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

const openBuy = (pkg) => {
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

onMounted(fetchPackages)
</script>

<style scoped>
.buy-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* ============================================================
   顶部说明
   ============================================================ */
.intro {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 14px 18px;
  background: linear-gradient(160deg, #f0f6ff 0%, #ffffff 60%);
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  animation: rise .4s ease-out both;
}

.intro-title {
  font-family: var(--font-display);
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.01em;
}

.intro-sub {
  margin: 0;
  font-size: 12.5px;
  color: #64748b;
}

.intro-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: #fff;
  border: 1px solid #dbeafe;
  border-radius: 99px;
  font-size: 12px;
  color: #475569;
}

.intro-badge .mono {
  color: #2563eb;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

/* ============================================================
   套餐卡
   ============================================================ */
.pkg-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}

.pkg-card {
  position: relative;
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 16px;
  padding: 18px 18px;
  display: flex;
  flex-direction: column;
  transition: border-color .15s, box-shadow .15s, transform .15s;
  animation: rise .4s ease-out both;
}

.pkg-card:hover {
  border-color: #c7d8f2;
  box-shadow: 0 8px 24px rgba(59,130,246,.1);
  transform: translateY(-2px);
}

.pkg-card.is-featured {
  border-color: #3b82f6;
  box-shadow: 0 6px 20px rgba(59,130,246,.12);
}

.pkg-ribbon {
  position: absolute;
  top: 12px;
  right: 12px;
  padding: 3px 9px;
  background: #3b82f6;
  color: #fff;
  font-size: 10.5px;
  font-weight: 600;
  border-radius: 5px;
  letter-spacing: .04em;
}

.pkg-top {
  margin-bottom: 18px;
}

.pkg-proto {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 5px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  font-weight: 700;
  letter-spacing: .04em;
  background: #f1f5f9;
  color: #64748b;
  margin-bottom: 10px;
}

.pkg-proto.vmess       { background: #eff6ff; color: #2563eb; }
.pkg-proto.vless       { background: #eef2ff; color: #4338ca; }
.pkg-proto.trojan      { background: #f5f3ff; color: #6d28d9; }
.pkg-proto.shadowsocks { background: #ecfdf5; color: #047857; }

.pkg-name {
  font-family: var(--font-display);
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.01em;
  line-height: 1.3;
}

.pkg-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.5;
}

/* 价格 */
.pkg-price {
  display: flex;
  align-items: baseline;
  gap: 3px;
  margin-bottom: 18px;
  line-height: 1;
}

.pkg-price .currency {
  font-size: 14px;
  color: #64748b;
  font-weight: 600;
}

.pkg-price .amount {
  font-family: var(--font-mono);
  font-size: 38px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.pkg-price .per {
  margin-left: 4px;
  font-size: 12px;
  color: #94a3b8;
}

/* 功能列表 */
.pkg-features {
  list-style: none;
  margin: 0 0 20px;
  padding: 14px 0;
  border-top: 1px dashed #eef1f6;
  border-bottom: 1px dashed #eef1f6;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.pkg-features li {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}

.f-icon {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: #f1f5f9;
  color: #64748b;
  border-radius: 7px;
  font-size: 13px;
  flex-shrink: 0;
}

.f-label {
  color: #64748b;
  flex: 1;
}

.f-value {
  color: #0f172a;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

/* 按钮 */
.pkg-btn {
  width: 100%;
  height: 42px;
  border: 1px solid #e2e8f0;
  background: #fff;
  color: #2563eb;
  font-weight: 600;
  font-size: 13.5px;
  border-radius: 10px;
  cursor: pointer;
  transition: background-color .15s, color .15s, border-color .15s;
  margin-top: auto;
}

.pkg-btn:hover {
  border-color: #3b82f6;
  background: #eff6ff;
}

.pkg-btn.is-featured {
  background: #3b82f6;
  border-color: #3b82f6;
  color: #fff;
}

.pkg-btn.is-featured:hover {
  background: #2563eb;
  border-color: #2563eb;
}

/* 状态 */
.state {
  padding: 72px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: #fff;
  border: 1px solid #e6ecf4;
  border-radius: 14px;
}

.state-empty .state-icon { font-size: 42px; color: #cbd5e1; }
.state-title { margin: 4px 0 2px; font-size: 14px; font-weight: 600; color: #475569; }
.state-sub { margin: 0; font-size: 12.5px; color: #94a3b8; }

.dot { width: 6px; height: 6px; border-radius: 50%; display: inline-block; }
.dot-ok { background: #16a34a; }

/* ============================================================
   弹窗
   ============================================================ */
.modal-head { margin-bottom: 16px; }
.modal-title {
  font-family: var(--font-display);
  margin: 0 0 4px;
  font-size: 17px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.01em;
}
.modal-sub { margin: 0; font-size: 12px; color: #94a3b8; }

.summary {
  background: #f8fafc;
  border: 1px solid #eef1f6;
  border-radius: 10px;
  padding: 14px 16px;
  margin-bottom: 18px;
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.summary-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
}

.summary-key { color: #64748b; }
.summary-val { color: #1e293b; font-weight: 500; }

.summary-row-total {
  padding-top: 9px;
  border-top: 1px dashed #dbe2ec;
  margin-top: 2px;
}

.summary-total {
  font-family: var(--font-mono);
  font-size: 20px;
  font-weight: 700;
  color: #dc2626;
  font-variant-numeric: tabular-nums;
}

.pay-title {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 10px;
  letter-spacing: .02em;
}

.pay-options {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.pay-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  cursor: pointer;
  transition: border-color .15s, background-color .15s;
}

.pay-option:hover { border-color: #cbd5e1; }

.pay-option.is-active {
  border-color: #3b82f6;
  background: #f0f6ff;
}

.pay-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
  background: #eff6ff;
  color: #3b82f6;
  font-size: 16px;
  flex-shrink: 0;
}

.pay-body { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.pay-name { font-size: 13.5px; font-weight: 600; color: #1e293b; }
.pay-sub  { font-size: 11.5px; color: #94a3b8; }

.pay-check {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid #cbd5e1;
  transition: border-color .15s, background-color .15s;
  flex-shrink: 0;
  position: relative;
}

.pay-option.is-active .pay-check {
  border-color: #3b82f6;
  background: #3b82f6;
  box-shadow: inset 0 0 0 3px #fff;
}

.modal-foot {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
}

/* ============================================================
   响应式
   ============================================================ */
@media (max-width: 576px) {
  .intro { flex-direction: column; align-items: flex-start; padding: 18px 16px; }
  .pkg-card { padding: 20px 18px; }
  .pkg-price .amount { font-size: 32px; }
  .modal-foot { flex-direction: column-reverse; }
  .modal-foot .ant-btn { width: 100%; }
}

@keyframes rise {
  from { opacity: 0; transform: translateY(6px); }
  to   { opacity: 1; transform: none; }
}
</style>
