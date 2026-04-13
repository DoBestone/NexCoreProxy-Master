<template>
  <div class="email-settings-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1 class="page-title">
        <MailOutlined class="title-icon" />
        邮件配置
      </h1>
      <p class="page-desc">配置 SMTP Lite API 邮件服务，用于发送通知邮件</p>
    </div>

    <!-- 配置卡片 -->
    <a-card class="config-card">
      <a-form :model="form" layout="vertical" class="email-form">
        <a-form-item label="接口地址" required>
          <a-input v-model:value="form.apiUrl" placeholder="https://smtp-lite.nexcores.net" />
        </a-form-item>

        <a-form-item label="接口密钥" required>
          <a-input-password v-model:value="form.apiKey" placeholder="smtp_xxxxxxxxxxxx" />
        </a-form-item>

        <a-form-item label="发件人名称">
          <a-input v-model:value="form.fromName" placeholder="NexCore代理主机" />
        </a-form-item>

        <a-form-item>
          <a-checkbox v-model:checked="form.enable">启用邮件服务</a-checkbox>
        </a-form-item>

        <a-form-item>
          <a-space>
            <a-button type="primary" @click="saveConfig" :loading="saving">
              <SaveOutlined /> 保存配置
            </a-button>
            <a-button @click="showTestModal" :disabled="!form.enable">
              <SendOutlined /> 发送测试邮件
            </a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 使用说明 -->
    <a-card class="help-card">
      <template #title>
        <InfoCircleOutlined /> 配置说明
      </template>
      <div class="help-content">
        <h4>邮件发送接口</h4>
        <p>系统使用 SMTP Lite API 统一发送邮件，自动轮询可用 SMTP 账号，无需自行配置 SMTP 服务器。</p>

        <h4 style="margin-top: 16px;">配置步骤</h4>
        <ol style="color: #475569; line-height: 2;">
          <li>填写接口地址（默认：https://smtp-lite.nexcores.net）</li>
          <li>填写接口密钥（在 SMTP Lite 管理后台创建）</li>
          <li>设置发件人显示名称</li>
          <li>勾选"启用邮件服务"</li>
          <li>点击"发送测试邮件"验证配置</li>
        </ol>

        <a-alert
          type="info"
          show-icon
          style="margin-top: 16px"
        >
          <template #message>
            接口密钥仅在创建时完整展示一次，请妥善保存。
          </template>
        </a-alert>
      </div>
    </a-card>

    <!-- 测试邮件弹窗 -->
    <a-modal
      v-model:open="testVisible"
      title="发送测试邮件"
      @ok="sendTestEmail"
      :confirmLoading="testing"
    >
      <a-form layout="vertical">
        <a-form-item label="收件人邮箱" required>
          <a-input v-model:value="testEmailAddress" placeholder="请输入收件人邮箱地址" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, onDeactivated } from 'vue'
import { message } from 'ant-design-vue'
import { MailOutlined, SaveOutlined, SendOutlined, InfoCircleOutlined } from '@ant-design/icons-vue'
import { getEmailConfig, updateEmailConfig, testEmail as sendTestEmailApi } from '@/api'

const saving = ref(false)
const testing = ref(false)
const testVisible = ref(false)

onDeactivated(() => { testVisible.value = false })
const testEmailAddress = ref('')

const form = ref({
  apiUrl: 'https://smtp-lite.nexcores.net',
  apiKey: '',
  fromName: 'NexCore代理主机',
  enable: false
})

const fetchConfig = async () => {
  try {
    const res = await getEmailConfig()
    if (res.success && res.obj) {
      form.value = { ...form.value, ...res.obj }
    }
  } catch (e) {
    // 忽略
  }
}

const saveConfig = async () => {
  if (!form.value.apiUrl) {
    message.warning('请填写接口地址')
    return
  }

  saving.value = true
  try {
    await updateEmailConfig(form.value)
    message.success('保存成功')
  } catch (e) {
    message.error('保存失败')
  } finally {
    saving.value = false
  }
}

const showTestModal = () => {
  testEmailAddress.value = ''
  testVisible.value = true
}

const sendTestEmail = async () => {
  if (!testEmailAddress.value) {
    message.warning('请输入收件人邮箱')
    return
  }

  testing.value = true
  try {
    await sendTestEmailApi(testEmailAddress.value)
    message.success('测试邮件已发送，请检查收件箱')
    testVisible.value = false
  } catch (e) {
    message.error('发送失败')
  } finally {
    testing.value = false
  }
}

onMounted(() => {
  fetchConfig()
})
</script>

<style scoped>
.email-settings-page {
  max-width: 800px;
  animation: fadeIn 0.3s ease;
}

.page-header {
  margin-bottom: 24px;
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

.config-card, .help-card {
  border-radius: 14px;
  margin-bottom: 24px;
}

.email-form :deep(.ant-form-item-label > label) {
  font-weight: 500;
}

.help-content h4 {
  color: #1e293b;
  margin-bottom: 8px;
}

.help-content p {
  color: #475569;
  font-size: 14px;
  line-height: 1.6;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
