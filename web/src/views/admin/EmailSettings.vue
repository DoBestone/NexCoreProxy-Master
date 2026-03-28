<template>
  <div class="email-settings-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1 class="page-title">
        <MailOutlined class="title-icon" />
        邮件配置
      </h1>
      <p class="page-desc">配置SMTP邮件服务，用于发送通知邮件</p>
    </div>
    
    <!-- 配置卡片 -->
    <a-card class="config-card">
      <a-form :model="form" layout="vertical" class="email-form">
        <a-row :gutter="24">
          <a-col :span="12">
            <a-form-item label="SMTP服务器" required>
              <a-input v-model:value="form.host" placeholder="smtp.example.com" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="端口">
              <a-input-number v-model:value="form.port" :min="1" :max="65535" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="使用TLS">
              <a-switch v-model:checked="form.useTLS" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="24">
          <a-col :span="12">
            <a-form-item label="用户名">
              <a-input v-model:value="form.username" placeholder="SMTP用户名" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="密码">
              <a-input-password v-model:value="form.password" placeholder="SMTP密码" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="24">
          <a-col :span="12">
            <a-form-item label="发件人邮箱">
              <a-input v-model:value="form.from" placeholder="noreply@example.com" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="发件人名称">
              <a-input v-model:value="form.fromName" placeholder="NexCore代理主机" />
            </a-form-item>
          </a-col>
        </a-row>
        
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
        <h4>常见SMTP配置</h4>
        <table class="smtp-table">
          <thead>
            <tr>
              <th>服务商</th>
              <th>服务器</th>
              <th>端口</th>
              <th>TLS</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>QQ邮箱</td>
              <td>smtp.qq.com</td>
              <td>587</td>
              <td>是</td>
            </tr>
            <tr>
              <td>163邮箱</td>
              <td>smtp.163.com</td>
              <td>465</td>
              <td>是</td>
            </tr>
            <tr>
              <td>阿里企业邮箱</td>
              <td>smtp.mxhichina.com</td>
              <td>465</td>
              <td>是</td>
            </tr>
            <tr>
              <td>Gmail</td>
              <td>smtp.gmail.com</td>
              <td>587</td>
              <td>是</td>
            </tr>
          </tbody>
        </table>
        
        <a-alert 
          type="info" 
          show-icon 
          style="margin-top: 16px"
        >
          <template #message>
            <strong>注意：</strong>部分邮箱服务商需要开启SMTP服务并获取授权码，而非使用登录密码。
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
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { MailOutlined, SaveOutlined, SendOutlined, InfoCircleOutlined } from '@ant-design/icons-vue'
import { getEmailConfig, updateEmailConfig, testEmail as sendTestEmailApi } from '@/api'

const saving = ref(false)
const testing = ref(false)
const testVisible = ref(false)
const testEmailAddress = ref('')

const form = ref({
  host: '',
  port: 587,
  username: '',
  password: '',
  from: '',
  fromName: 'NexCore代理主机',
  useTLS: true,
  enable: false
})

const fetchConfig = async () => {
  try {
    const res = await getEmailConfig()
    if (res.success && res.obj) {
      form.value = { ...res.obj }
    }
  } catch (e) {
    // 忽略
  }
}

const saveConfig = async () => {
  if (!form.value.host) {
    message.warning('请填写SMTP服务器地址')
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

.config-card, .help-card {
  border-radius: 14px;
  margin-bottom: 24px;
}

.email-form :deep(.ant-form-item-label > label) {
  font-weight: 500;
}

/* SMTP配置表格 */
.smtp-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.smtp-table th,
.smtp-table td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 1px solid #f0f0f0;
}

.smtp-table th {
  background: #f8fafc;
  font-weight: 600;
  color: #595959;
}

.smtp-table td {
  color: #262626;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>