<template>
  <div class="spec-content-wrapper">
    <div class="spec-content-header">
      <h3>网络配置</h3>
      <p>配置 Pod 的网络设置</p>
    </div>
    <div class="spec-content">
      <div class="network-config-form">
        <div class="config-form-section">
          <div class="form-section-title">
            <span class="section-icon">🌐</span>
            <span class="section-text">主机网络</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">使用主机网络</label>
            <el-switch
              v-model="formData.hostNetwork"
              active-text="开启"
              inactive-text="关闭"
            />
            <span class="form-hint">Pod 将使用主机的网络命名空间</span>
          </div>
        </div>

        <div class="config-form-section">
          <div class="form-section-title">
            <span class="section-icon">🔧</span>
            <span class="section-text">DNS 策略</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">DNS 策略</label>
            <el-select v-model="formData.dnsPolicy" placeholder="选择 DNS 策略" style="width: 300px;">
              <el-option label="ClusterFirst" value="ClusterFirst" />
              <el-option label="Default" value="Default" />
              <el-option label="ClusterFirstWithHostNet" value="ClusterFirstWithHostNet" />
              <el-option label="None" value="None" />
            </el-select>
          </div>
        </div>

        <div class="config-form-section">
          <div class="form-section-title">
            <span class="section-icon">🖥️</span>
            <span class="section-text">主机名设置</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">主机名</label>
            <el-input v-model="formData.hostname" placeholder="可选，指定 Pod 的主机名" style="width: 300px;" />
            <span class="form-hint">不指定则默认为 pod 名</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">子域名</label>
            <el-input v-model="formData.subdomain" placeholder="可选，指定 Pod 的子域名" style="width: 300px;" />
            <span class="form-hint">完整主机名为：hostname.subdomain</span>
          </div>
        </div>

        <div class="config-form-section">
          <div class="form-section-title">
            <span class="section-icon">📡</span>
            <span class="section-text">DNS 配置</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">服务器地址</label>
            <div class="dns-inputs-wrapper">
              <div v-for="(ns, index) in formData.dnsConfig.nameservers" :key="'ns-'+index" class="dns-input-item">
                <el-input v-model="formData.dnsConfig.nameservers[index]" placeholder="如: 8.8.8.8" size="small" style="width: 200px;" />
                <el-button type="danger" link @click="emit('removeDNSNameserver', index)" :icon="Delete" size="small">删除</el-button>
              </div>
              <el-button type="primary" link @click="emit('addDNSNameserver')" :icon="Plus" size="small">添加服务器</el-button>
            </div>
            <span class="form-hint">DNS 服务器 IP 地址列表</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">搜索域</label>
            <div class="dns-inputs-wrapper">
              <div v-for="(search, index) in formData.dnsConfig.searches" :key="'search-'+index" class="dns-input-item">
                <el-input v-model="formData.dnsConfig.searches[index]" placeholder="如: default.svc.cluster.local" size="small" style="width: 250px;" />
                <el-button type="danger" link @click="emit('removeDNSSearch', index)" :icon="Delete" size="small">删除</el-button>
              </div>
              <el-button type="primary" link @click="emit('addDNSSearch')" :icon="Plus" size="small">添加搜索域</el-button>
            </div>
            <span class="form-hint">DNS 搜索域列表</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">DNS 选项</label>
            <div class="dns-options-wrapper">
              <div v-for="(opt, index) in formData.dnsConfig.options" :key="'opt-'+index" class="dns-option-item">
                <el-input v-model="opt.name" placeholder="选项名，如: ndots" size="small" style="width: 150px;" />
                <span class="option-separator">:</span>
                <el-input v-model="opt.value" placeholder="值，如: 5" size="small" style="width: 120px;" />
                <el-button type="danger" link @click="emit('removeDNSOption', index)" :icon="Delete" size="small">删除</el-button>
              </div>
              <el-button type="primary" link @click="emit('addDNSOption')" :icon="Plus" size="small">添加选项</el-button>
            </div>
            <span class="form-hint">自定义 DNS 选项</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Delete } from '@element-plus/icons-vue'

interface DNSConfig {
  nameservers: string[]
  searches: string[]
  options: { name: string; value: string }[]
}

interface FormData {
  hostNetwork: boolean
  dnsPolicy: string
  hostname: string
  subdomain: string
  dnsConfig: DNSConfig
}

const props = defineProps<{
  formData: FormData
}>()

const emit = defineEmits<{
  addDNSNameserver: []
  removeDNSNameserver: [index: number]
  addDNSSearch: []
  removeDNSSearch: [index: number]
  addDNSOption: []
  removeDNSOption: [index: number]
}>()
</script>

<style scoped>
.spec-content-wrapper {
  padding: 24px 32px;
}

.spec-content-header {
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 2px solid #f0f0f0;
}

.spec-content-header h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.spec-content-header p {
  margin: 0;
  font-size: 13px;
  color: #909399;
}

.spec-content {
  background: #fff;
}

.network-config-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.config-form-section {
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.form-section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.section-icon {
  font-size: 18px;
}

.section-text {
  font-size: 15px;
  font-weight: 600;
}

.form-item-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.form-item-row:last-child {
  border-bottom: none;
}

.form-label {
  width: 120px;
  font-size: 14px;
  font-weight: 500;
  color: #606266;
  flex-shrink: 0;
}

.form-hint {
  font-size: 12px;
  color: #909399;
  margin-left: 8px;
}

.dns-inputs-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}

.dns-input-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dns-options-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}

.dns-option-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.option-separator {
  color: #909399;
  font-size: 14px;
  font-weight: 500;
}
</style>
