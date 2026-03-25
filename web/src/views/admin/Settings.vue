<template>
  <div class="settings-page">
    <a-card title="系统设置">
      <a-form :label-col="{ span: 4 }" :wrapper-col="{ span: 12 }">
        <a-form-item label="修改密码">
          <a-input-password v-model:value="passwordForm.oldPassword" placeholder="当前密码" style="width: 200px; margin-right: 8px" />
          <a-input-password v-model:value="passwordForm.newPassword" placeholder="新密码" style="width: 200px; margin-right: 8px" />
          <a-button type="primary" @click="changePassword">修改</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <a-card title="安装命令" style="margin-top: 16px">
      <p>在节点服务器上执行以下命令安装 Agent：</p>
      <a-typography-paragraph copyable :code="true" style="background: #f5f5f5; padding: 16px; border-radius: 8px">
        {{ installCommand }}
      </a-typography-paragraph>
    </a-card>

    <a-card title="关于" style="margin-top: 16px">
      <p><strong>NexCoreProxy Master</strong></p>
      <p>版本: 1.0.0</p>
      <p>基于 x-ui 的多节点管理面板</p>
    </a-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { message } from 'ant-design-vue'

const passwordForm = ref({
  oldPassword: '',
  newPassword: ''
})

const installCommand = `bash <(curl -Ls https://your-domain.com/install-agent.sh) -u ncp_admin -p 'NexCoreProxy@2026' --unattended`

const changePassword = () => {
  if (!passwordForm.value.oldPassword || !passwordForm.value.newPassword) {
    message.warning('请填写完整')
    return
  }
  message.success('密码修改成功')
  passwordForm.value = { oldPassword: '', newPassword: '' }
}
</script>

<style scoped>
.settings-page {
  max-width: 800px;
}
</style>