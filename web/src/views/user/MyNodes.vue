<template>
  <div class="my-nodes">
    <!-- 订阅链接卡片 -->
    <a-card title="订阅链接" style="margin-bottom: 16px">
      <a-space direction="vertical" style="width: 100%">
        <a-typography-text copyable>{{ subscribeUrl || '加载中...' }}</a-typography-text>
        <a-space>
          <a-button type="primary" @click="copySubscribe">
            <template #icon><CopyOutlined /></template>
            复制订阅链接
          </a-button>
          <a-button @click="refreshSubscribe">
            <template #icon><ReloadOutlined /></template>
            刷新订阅
          </a-button>
        </a-space>
        <a-alert type="info" show-icon>
          <template #message>将订阅链接导入到客户端（如 v2rayN、Clash 等）即可使用所有节点</template>
        </a-alert>
      </a-space>
    </a-card>

    <a-card title="我的节点">
      <a-empty v-if="nodes.length === 0" description="暂无分配节点">
        <template #image>
          <CloudServerOutlined style="font-size: 48px; color: #ccc" />
        </template>
      </a-empty>
      <a-row v-else :gutter="16">
        <a-col :xs="24" :sm="12" :lg="8" v-for="node in nodes" :key="node.id">
          <a-card class="node-card" hoverable>
            <template #title>
              <span>{{ node.name }}</span>
              <a-tag :color="node.status === 'online' ? 'green' : 'red'" style="margin-left: 8px">
                {{ node.status === 'online' ? '在线' : '离线' }}
              </a-tag>
            </template>
            <template #extra>
              <a-button type="link" size="small" @click="copyLink(node)">
                <CopyOutlined /> 复制链接
              </a-button>
            </template>
            <p><strong>协议:</strong> {{ node.protocol }}</p>
            <p><strong>端口:</strong> {{ node.port }}</p>
            <p><strong>流量:</strong> {{ formatTraffic(node.up + node.down) }} / {{ formatTraffic(node.total) || '无限制' }}</p>
          </a-card>
        </a-col>
      </a-row>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { CloudServerOutlined, CopyOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { getMyNodes, getMySubscribe } from '@/api'

const nodes = ref([])
const subscribeUrl = ref('')

const formatTraffic = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const copyLink = (node) => {
  if (node.link) {
    navigator.clipboard.writeText(node.link)
    message.success('链接已复制')
  } else {
    message.warning('暂无可用链接')
  }
}

const copySubscribe = () => {
  if (subscribeUrl.value) {
    navigator.clipboard.writeText(subscribeUrl.value)
    message.success('订阅链接已复制')
  }
}

const refreshSubscribe = async () => {
  try {
    const res = await getMySubscribe()
    if (res.success && res.obj) {
      subscribeUrl.value = res.obj.url
      message.success('订阅已刷新')
    }
  } catch (e) {
    message.error('刷新订阅失败')
  }
}

onMounted(async () => {
  try {
    const res = await getMyNodes()
    nodes.value = res.obj || []
  } catch (e) {
    message.error('获取节点失败')
  }

  // 获取订阅链接
  refreshSubscribe()
})
</script>

<style scoped>
.my-nodes {
  max-width: 1200px;
  margin: 0 auto;
}

.node-card {
  margin-bottom: 16px;
  border-radius: 8px;
}
</style>