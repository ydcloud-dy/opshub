<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑 Service' : '创建 Service'"
    width="900px"
    class="network-resource-dialog"
    :close-on-click-modal="false"
    :lock-scroll="false"
    @close="handleClose"
  >
    <!-- 基本信息区域 -->
    <div class="basic-info-section">
      <el-form ref="formRef" :model="formData" :rules="rules" label-width="100px">
        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="名称" prop="name">
              <el-input v-model="formData.name" placeholder="Service 名称" :disabled="isEdit" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="命名空间" prop="namespace">
              <el-select v-model="formData.namespace" placeholder="选择命名空间" :disabled="isEdit" style="width: 100%">
                <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="类型" prop="type">
              <el-select v-model="formData.type" placeholder="选择类型" style="width: 100%">
                <el-option label="ClusterIP" value="ClusterIP" />
                <el-option label="NodePort" value="NodePort" />
                <el-option label="LoadBalancer" value="LoadBalancer" />
                <el-option label="ExternalName" value="ExternalName" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </div>

    <!-- Tab 导航 -->
    <el-tabs v-model="activeTab" class="service-tabs">
      <!-- 端口配置 -->
      <el-tab-pane label="端口" name="ports">
        <div class="tab-content">
          <div class="ports-config">
            <div v-for="(port, index) in formData.ports" :key="index" class="port-item">
              <div class="port-header">
                <span>端口 {{ index + 1 }}</span>
                <el-button type="danger" link @click="removePort(index)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
              <el-row :gutter="12">
                <el-col :span="6">
                  <div class="field-group">
                    <label>名称</label>
                    <el-input v-model="port.name" placeholder="可选" size="small" />
                  </div>
                </el-col>
                <el-col :span="6">
                  <div class="field-group">
                    <label>协议</label>
                    <el-select v-model="port.protocol" size="small" style="width: 100%">
                      <el-option label="TCP" value="TCP" />
                      <el-option label="UDP" value="UDP" />
                      <el-option label="SCTP" value="SCTP" />
                    </el-select>
                  </div>
                </el-col>
                <el-col :span="6">
                  <div class="field-group">
                    <label>端口</label>
                    <el-input-number v-model="port.port" :min="1" :max="65535" size="small" style="width: 100%" />
                  </div>
                </el-col>
                <el-col :span="6">
                  <div class="field-group">
                    <label>目标端口</label>
                    <el-input-number v-model="port.targetPort" :min="1" :max="65535" size="small" style="width: 100%" />
                  </div>
                </el-col>
              </el-row>
              <el-row :gutter="12" v-if="formData.type === 'NodePort'">
                <el-col :span="6">
                  <div class="field-group">
                    <label>NodePort</label>
                    <el-input-number v-model="port.nodePort" :min="30000" :max="32767" size="small" style="width: 100%" />
                  </div>
                </el-col>
              </el-row>
            </div>
            <el-button type="primary" link @click="addPort" style="margin-top: 8px">
              <el-icon><Plus /></el-icon> 添加端口
            </el-button>
          </div>
        </div>
      </el-tab-pane>

      <!-- 选择器配置 -->
      <el-tab-pane label="选择器" name="selector">
        <div class="tab-content">
          <div class="selector-config">
            <div class="config-header-with-desc">
              <div class="header-text">
                <div class="title">Pod 标签选择器</div>
                <div class="description">配置用于选择 Pod 的标签</div>
              </div>
              <el-button type="primary" link @click="addSelector">
                <el-icon><Plus /></el-icon> 添加
              </el-button>
            </div>
            <div v-if="selectorList.length === 0" class="empty-state">
              <el-icon class="empty-icon"><Connection /></el-icon>
              <p>暂无选择器配置，点击上方"添加"按钮添加</p>
            </div>
            <div v-else>
              <div v-for="(item, index) in selectorList" :key="index" class="selector-item">
                <div class="selector-row">
                  <div class="field-group">
                    <label>键</label>
                    <el-input v-model="item.key" placeholder="例如: app" size="small" />
                  </div>
                  <div class="equal-sign">=</div>
                  <div class="field-group">
                    <label>值</label>
                    <el-input v-model="item.value" placeholder="例如: nginx" size="small" />
                  </div>
                  <div class="action-area">
                    <el-button type="danger" link @click="removeSelector(index)">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- Session Affinity -->
      <el-tab-pane label="Session Affinity" name="sessionAffinity">
        <div class="tab-content">
          <div class="affinity-config-new">
            <!-- 类型选择 -->
            <div class="affinity-type-cards">
              <div
                :class="['affinity-type-card', { 'is-selected': formData.sessionAffinity === 'None' }]"
                @click="formData.sessionAffinity = 'None'"
              >
                <div class="card-icon">
                  <el-icon><SwitchButton /></el-icon>
                </div>
                <div class="card-content">
                  <div class="card-title">None</div>
                  <div class="card-desc">不保持会话亲和性，来自同一客户端的请求可能被分发到不同的 Pod</div>
                </div>
                <div class="card-check">
                  <el-icon v-if="formData.sessionAffinity === 'None'"><CircleCheckFilled /></el-icon>
                </div>
              </div>

              <div
                :class="['affinity-type-card', { 'is-selected': formData.sessionAffinity === 'ClientIP' }]"
                @click="formData.sessionAffinity = 'ClientIP'"
              >
                <div class="card-icon">
                  <el-icon><Position /></el-icon>
                </div>
                <div class="card-content">
                  <div class="card-title">ClientIP</div>
                  <div class="card-desc">基于客户端 IP 的会话亲和性，同一客户端的请求将被发送到同一个 Pod</div>
                </div>
                <div class="card-check">
                  <el-icon v-if="formData.sessionAffinity === 'ClientIP'"><CircleCheckFilled /></el-icon>
                </div>
              </div>
            </div>

            <!-- 超时配置 -->
            <div v-if="formData.sessionAffinity === 'ClientIP'" class="affinity-timeout-config">
              <div class="timeout-config-card">
                <div class="config-card-header">
                  <div class="header-left">
                    <div class="header-icon">
                      <el-icon><Clock /></el-icon>
                    </div>
                    <div class="header-text">
                      <div class="header-title">会话保持时间</div>
                      <div class="header-desc">同一客户端的请求将被持续发送到同一个 Pod 的时间</div>
                    </div>
                  </div>
                </div>
                <div class="config-card-body">
                  <div class="timeout-input-wrapper">
                    <el-input-number
                      v-model="formData.sessionAffinityConfig.clientIP.timeoutSeconds"
                      :min="1"
                      :max="86400"
                      controls-position="right"
                      size="large"
                      class="timeout-input"
                    />
                    <div class="time-conversions">
                      <div class="conversion-item">
                        <span class="conversion-label">=</span>
                        <span class="conversion-value">{{ formatTimeout(formData.sessionAffinityConfig.clientIP.timeoutSeconds) }}</span>
                      </div>
                    </div>
                  </div>
                  <div class="timeout-info-box">
                    <el-icon class="info-icon"><InfoFilled /></el-icon>
                    <div class="info-text">
                      <div class="info-main">默认 10800 秒（3 小时），最大 86400 秒（24 小时）</div>
                      <div class="info-sub">建议根据业务需求调整时间，过短可能导致频繁切换 Pod，过长可能影响负载均衡效果</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- IP 地址配置 -->
      <el-tab-pane label="IP地址" name="ipAddresses">
        <div class="tab-content">
          <div class="ip-config">
            <!-- Cluster IP -->
            <div class="ip-section">
              <div class="ip-section-header">
                <div class="header-icon">
                  <el-icon><Link /></el-icon>
                </div>
                <div class="header-text">
                  <div class="title">Cluster IP</div>
                  <div class="description">集群内部访问的 IP 地址</div>
                </div>
              </div>
              <div class="ip-input-group">
                <el-input
                  v-model="formData.clusterIP"
                  placeholder="留空自动分配，或输入 'None' 创建无头服务"
                  size="large"
                >
                  <template #prefix>
                    <el-icon><Connection /></el-icon>
                  </template>
                </el-input>
              </div>
              <div class="ip-hint">
                <el-icon><InfoFilled /></el-icon>
                <span>留空则由系统自动分配 ClusterIP；输入 'None' 可创建无头服务（Headless Service）</span>
              </div>
            </div>

            <!-- External IP -->
            <div class="ip-section" style="margin-top: 30px">
              <div class="ip-section-header">
                <div class="header-icon">
                  <el-icon><Position /></el-icon>
                </div>
                <div class="header-text">
                  <div class="title">外部 IP</div>
                  <div class="description">配置外部可访问的 IP 地址</div>
                </div>
                <el-button type="primary" link @click="addExternalIP">
                  <el-icon><Plus /></el-icon> 添加
                </el-button>
              </div>
              <div v-if="formData.externalIPs.length === 0" class="empty-state">
                <el-icon class="empty-icon"><Position /></el-icon>
                <p>暂无外部 IP 配置，点击上方"添加"按钮添加</p>
              </div>
              <div v-else class="external-ip-list">
                <div v-for="(ip, index) in formData.externalIPs" :key="index" class="external-ip-item">
                  <div class="ip-input-wrapper">
                    <el-input
                      v-model="formData.externalIPs[index]"
                      placeholder="例如: 192.168.1.100"
                      size="small"
                    >
                      <template #prefix>
                        <el-icon><Position /></el-icon>
                      </template>
                    </el-input>
                  </div>
                  <div class="ip-actions">
                    <el-button type="danger" link @click="removeExternalIP(index)">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- 标签/注解 -->
      <el-tab-pane label="标签/注解" name="labelsAnnotations">
        <div class="tab-content">
          <!-- 标签 -->
          <div class="labels-config">
            <div class="config-header-with-desc">
              <div class="header-text">
                <div class="title">标签 (Labels)</div>
                <div class="description">用于标识和选择组织的键值对</div>
              </div>
              <el-button type="primary" link @click="addLabel">
                <el-icon><Plus /></el-icon> 添加
              </el-button>
            </div>
            <div v-if="labelsList.length === 0" class="empty-state">
              <el-icon class="empty-icon"><PriceTag /></el-icon>
              <p>暂无标签配置，点击上方"添加"按钮添加</p>
            </div>
            <div v-else class="kv-list">
              <div v-for="(item, index) in labelsList" :key="index" class="kv-item">
                <div class="kv-row">
                  <div class="kv-fields">
                    <div class="field-group">
                      <label>键</label>
                      <el-input v-model="item.key" placeholder="键名" size="small" />
                    </div>
                    <div class="equal-sign">=</div>
                    <div class="field-group">
                      <label>值</label>
                      <el-input v-model="item.value" placeholder="键值" size="small" />
                    </div>
                  </div>
                  <div class="kv-actions">
                    <el-button type="danger" link @click="removeLabel(index)">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 注解 -->
          <div class="annotations-config" style="margin-top: 40px">
            <div class="config-header-with-desc">
              <div class="header-text">
                <div class="title">注解 (Annotations)</div>
                <div class="description">用于存储任意非标识性数据的键值对</div>
              </div>
              <el-button type="primary" link @click="addAnnotation">
                <el-icon><Plus /></el-icon> 添加
              </el-button>
            </div>
            <div v-if="annotationsList.length === 0" class="empty-state">
              <el-icon class="empty-icon"><Document /></el-icon>
              <p>暂无注解配置，点击上方"添加"按钮添加</p>
            </div>
            <div v-else class="kv-list">
              <div v-for="(item, index) in annotationsList" :key="index" class="kv-item">
                <div class="kv-row">
                  <div class="kv-fields">
                    <div class="field-group">
                      <label>键</label>
                      <el-input v-model="item.key" placeholder="键名" size="small" />
                    </div>
                    <div class="equal-sign">=</div>
                    <div class="field-group">
                      <label>值</label>
                      <el-input v-model="item.value" placeholder="键值" size="small" />
                    </div>
                  </div>
                  <div class="kv-actions">
                    <el-button type="danger" link @click="removeAnnotation(index)">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">保存</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete, Plus, Connection, Clock, InfoFilled, Position, PriceTag, Document, Link, SwitchButton, CircleCheckFilled } from '@element-plus/icons-vue'
import { getServiceYAML, updateServiceYAML, createService, type ServiceInfo } from '@/api/kubernetes'

interface PortConfig {
  name?: string
  protocol: string
  port: number
  targetPort: number
  nodePort?: number
}

interface KeyValueItem {
  key: string
  value: string
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['success'])

const visible = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const formRef = ref()
const namespaces = ref<any[]>([])
const originalData = ref<any>(null)
const activeTab = ref('ports')

const formData = ref({
  name: '',
  namespace: '',
  type: 'ClusterIP' as any,
  clusterIP: '',
  externalIPs: [] as string[],
  ports: [] as PortConfig[],
  selector: {} as Record<string, string>,
  sessionAffinity: 'None',
  sessionAffinityConfig: {
    clientIP: {
      timeoutSeconds: 10800
    }
  },
  labels: {} as Record<string, string>,
  annotations: {} as Record<string, string>
})

const selectorList = ref<KeyValueItem[]>([])
const labelsList = ref<KeyValueItem[]>([])
const annotationsList = ref<KeyValueItem[]>([])

const rules = {
  name: [{ required: true, message: '请输入 Service 名称', trigger: 'blur' }],
  namespace: [{ required: true, message: '请选择命名空间', trigger: 'change' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }]
}

// 打开对话框（编辑模式）
const openEdit = async (service: ServiceInfo, nsList: any[]) => {
  namespaces.value = nsList
  isEdit.value = true
  visible.value = true
  activeTab.value = 'ports'

  try {
    const response = await getServiceYAML(props.clusterId!, service.namespace, service.name)
    originalData.value = response.items || response

    const spec = originalData.value.spec || {}
    const metadata = originalData.value.metadata || {}

    formData.value = {
      name: metadata.name || '',
      namespace: metadata.namespace || '',
      type: spec.type || 'ClusterIP',
      clusterIP: spec.clusterIP || '',
      externalIPs: spec.externalIPs || [],
      ports: (spec.ports || []).map((p: any) => ({
        name: p.name || '',
        protocol: p.protocol || 'TCP',
        port: p.port,
        targetPort: p.targetPort,
        nodePort: p.nodePort
      })),
      selector: spec.selector || {},
      sessionAffinity: spec.sessionAffinity || 'None',
      sessionAffinityConfig: spec.sessionAffinityConfig || {
        clientIP: {
          timeoutSeconds: 10800
        }
      },
      labels: metadata.labels || {},
      annotations: metadata.annotations || {}
    }

    // 同步到列表
    syncSelectorFromForm()
    syncLabelsFromForm()
    syncAnnotationsFromForm()

    if (formData.value.ports.length === 0) {
      addPort()
    }
  } catch (error) {
    ElMessage.error('获取 Service 详情失败')
  }
}

// 打开对话框（创建模式）
const openCreate = (nsList: any[]) => {
  namespaces.value = nsList
  isEdit.value = false
  visible.value = true
  activeTab.value = 'ports'

  formData.value = {
    name: '',
    namespace: '',
    type: 'ClusterIP',
    clusterIP: '',
    externalIPs: [],
    ports: [{
      name: '',
      protocol: 'TCP',
      port: 80,
      targetPort: 80
    }],
    selector: {},
    sessionAffinity: 'None',
    sessionAffinityConfig: {
      clientIP: {
        timeoutSeconds: 10800
      }
    },
    labels: {},
    annotations: {}
  }

  selectorList.value = []
  labelsList.value = []
  annotationsList.value = []
}

// 同步方法
const syncSelectorFromForm = () => {
  selectorList.value = Object.entries(formData.value.selector).map(([key, value]) => ({ key, value }))
}

const syncSelectorToList = () => {
  formData.value.selector = selectorList.value.reduce((acc, { key, value }) => {
    if (key && value) {
      acc[key] = value
    }
    return acc
  }, {} as Record<string, string>)
}

const syncLabelsFromForm = () => {
  labelsList.value = Object.entries(formData.value.labels).map(([key, value]) => ({ key, value }))
}

const syncLabelsToList = () => {
  formData.value.labels = labelsList.value.reduce((acc, { key, value }) => {
    if (key && value) {
      acc[key] = value
    }
    return acc
  }, {} as Record<string, string>)
}

const syncAnnotationsFromForm = () => {
  annotationsList.value = Object.entries(formData.value.annotations).map(([key, value]) => ({ key, value }))
}

const syncAnnotationsToList = () => {
  formData.value.annotations = annotationsList.value.reduce((acc, { key, value }) => {
    if (key) {
      acc[key] = value || ''
    }
    return acc
  }, {} as Record<string, string>)
}

// 端口操作
const addPort = () => {
  formData.value.ports.push({
    name: '',
    protocol: 'TCP',
    port: 80,
    targetPort: 80
  })
}

const removePort = (index: number) => {
  formData.value.ports.splice(index, 1)
}

// 选择器操作
const addSelector = () => {
  selectorList.value.push({ key: '', value: '' })
}

const removeSelector = (index: number) => {
  selectorList.value.splice(index, 1)
}

// 外部 IP 操作
const addExternalIP = () => {
  formData.value.externalIPs.push('')
}

const removeExternalIP = (index: number) => {
  formData.value.externalIPs.splice(index, 1)
}

// 标签操作
const addLabel = () => {
  labelsList.value.push({ key: '', value: '' })
}

const removeLabel = (index: number) => {
  labelsList.value.splice(index, 1)
}

// 注解操作
const addAnnotation = () => {
  annotationsList.value.push({ key: '', value: '' })
}

const removeAnnotation = (index: number) => {
  annotationsList.value.splice(index, 1)
}

// 格式化超时时间显示
const formatTimeout = (seconds: number) => {
  if (seconds < 60) {
    return `${seconds} 秒`
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60)
    return `${minutes} 分钟`
  } else if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600)
    return `${hours} 小时`
  } else {
    const days = Math.floor(seconds / 86400)
    return `${days} 天`
  }
}

// 构建保存的数据
const buildSaveData = () => {
  // 同步所有列表到对象
  syncSelectorToList()
  syncLabelsToList()
  syncAnnotationsToList()

  // 构建端口数组
  const ports = formData.value.ports.map(p => {
    const portObj: any = {
      protocol: p.protocol,
      port: p.port,
      targetPort: p.targetPort
    }
    if (p.name) portObj.name = p.name
    if (formData.value.type === 'NodePort' && p.nodePort) {
      portObj.nodePort = p.nodePort
    }
    return portObj
  })

  // 构建 Service 对象
  const serviceData: any = {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name: formData.value.name,
      namespace: formData.value.namespace
    },
    spec: {
      type: formData.value.type,
      selector: formData.value.selector,
      ports: ports,
      sessionAffinity: formData.value.sessionAffinity
    }
  }

  // 添加 SessionAffinityConfig
  if (formData.value.sessionAffinity === 'ClientIP') {
    serviceData.spec.sessionAffinityConfig = formData.value.sessionAffinityConfig
  }

  // 如果是编辑模式，需要包含完整的 spec
  if (isEdit.value && originalData.value) {
    const originalSpec = originalData.value.spec || {}

    // 保留 metadata
    serviceData.metadata = {
      ...originalData.value.metadata,
      name: formData.value.name,
      namespace: formData.value.namespace
    }

    // 添加 labels 和 annotations
    if (Object.keys(formData.value.labels).length > 0) {
      serviceData.metadata.labels = formData.value.labels
    }
    if (Object.keys(formData.value.annotations).length > 0) {
      serviceData.metadata.annotations = formData.value.annotations
    }

    // 保留 ClusterIP
    if (originalSpec.clusterIP) {
      serviceData.spec.clusterIP = originalSpec.clusterIP
    }
    if (formData.value.clusterIP) {
      serviceData.spec.clusterIP = formData.value.clusterIP
    }

    // 保留其他必要的 spec 字段
    if (originalSpec.clusterIPs) {
      serviceData.spec.clusterIPs = originalSpec.clusterIPs
    }
    if (originalSpec.ipFamilies) {
      serviceData.spec.ipFamilies = originalSpec.ipFamilies
    }
    if (originalSpec.ipFamilyPolicy) {
      serviceData.spec.ipFamilyPolicy = originalSpec.ipFamilyPolicy
    }
    if (originalSpec.internalTrafficPolicy) {
      serviceData.spec.internalTrafficPolicy = originalSpec.internalTrafficPolicy
    }
    if (originalSpec.externalTrafficPolicy) {
      serviceData.spec.externalTrafficPolicy = originalSpec.externalTrafficPolicy
    }
    if (originalSpec.healthCheckNodePort) {
      serviceData.spec.healthCheckNodePort = originalSpec.healthCheckNodePort
    }
  } else {
    // 创建模式
    if (formData.value.clusterIP) {
      serviceData.spec.clusterIP = formData.value.clusterIP
    }
    if (Object.keys(formData.value.labels).length > 0) {
      serviceData.metadata.labels = formData.value.labels
    }
    if (Object.keys(formData.value.annotations).length > 0) {
      serviceData.metadata.annotations = formData.value.annotations
    }
  }

  // 添加外部 IP
  const validExternalIPs = formData.value.externalIPs.filter(ip => ip)
  if (validExternalIPs.length > 0) {
    serviceData.spec.externalIPs = validExternalIPs
  }

  return serviceData
}

// 保存
const handleSave = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  // 验证端口配置
  if (formData.value.ports.length === 0) {
    ElMessage.error('请至少配置一个端口')
    return
  }

  // 验证选择器 - 先同步数据
  syncSelectorToList()
  const hasValidSelector = Object.values(formData.value.selector).some(v => v)
  if (!hasValidSelector && formData.value.type !== 'ExternalName') {
    ElMessage.error('请配置至少一个选择器')
    return
  }

  saving.value = true
  try {
    const serviceData = buildSaveData()

    if (isEdit.value) {
      await updateServiceYAML(
        props.clusterId!,
        formData.value.namespace,
        formData.value.name,
        serviceData
      )
      ElMessage.success('更新成功')
    } else {
      await createService(props.clusterId!, formData.value.namespace, {
        name: formData.value.name,
        type: formData.value.type,
        clusterIP: formData.value.clusterIP || undefined,
        ports: formData.value.ports.map(p => ({
          name: p.name,
          protocol: p.protocol,
          port: p.port,
          targetPort: p.targetPort.toString(),
          nodePort: p.nodePort
        })) as any,
        selector: formData.value.selector,
        sessionAffinity: formData.value.sessionAffinity
      })
      ElMessage.success('创建成功')
    }

    emit('success')
    handleClose()
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

// 关闭对话框
const handleClose = () => {
  visible.value = false
  formRef.value?.resetFields()
  originalData.value = null
  activeTab.value = 'ports'
  selectorList.value = []
  labelsList.value = []
  annotationsList.value = []
}

defineExpose({
  openEdit,
  openCreate
})
</script>

<style scoped>
/* 基本信息区域 */
.basic-info-section {
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #dcdfe6;
}

/* Tab 样式 */
.service-tabs {
  margin-top: 10px;
}

.service-tabs :deep(.el-tabs__header) {
  margin-bottom: 20px;
}

.service-tabs :deep(.el-tabs__item) {
  color: #606266;
  font-weight: 500;
}

.service-tabs :deep(.el-tabs__item.is-active) {
  color: #d4af37;
}

.service-tabs :deep(.el-tabs__active-bar) {
  background-color: #d4af37;
}

.tab-content {
  min-height: 300px;
}

/* 配置区域通用样式 */
.port-item,
.selector-config,
.affinity-config,
.ip-config,
.labels-config,
.annotations-config {
  width: 100%;
}

.port-item {
  margin-bottom: 20px;
  padding: 20px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background-color: #fafafa;
}

.port-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  font-weight: 500;
  color: #d4af37;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-group label {
  font-size: 13px;
  color: #606266;
  font-weight: 500;
}

.hint {
  font-size: 12px;
  color: #909399;
  margin-left: 8px;
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  font-weight: 500;
  color: #303133;
}

.config-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.config-row span {
  color: #909399;
  font-weight: 500;
}

/* 输入框样式 */
:deep(.el-input__wrapper) {
  background-color: #fff;
  border-color: #dcdfe6;
  box-shadow: none;
}

:deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 0 0 1px #d4af37;
}

:deep(.el-input__inner) {
  color: #606266;
}

:deep(.el-select .el-input__wrapper) {
  background-color: #fff;
}

:deep(.el-select .el-input__inner) {
  color: #606266;
}

:deep(.el-input-number .el-input__wrapper) {
  background-color: #fff;
}

:deep(.el-input-number .el-input__inner) {
  color: #606266;
}

:deep(.el-input-number__decrease),
:deep(.el-input-number__increase) {
  background-color: #f5f7fa;
  border-color: #dcdfe6;
  color: #606266;
}

:deep(.el-input-number__decrease:hover),
:deep(.el-input-number__increase:hover) {
  color: #d4af37;
}

/* 单选框样式 */
:deep(.el-radio__label) {
  color: #606266;
}

:deep(.el-radio.is-checked .el-radio__inner) {
  border-color: #d4af37;
  background: #d4af37;
}

:deep(.el-radio__input.is-checked .el-radio__inner) {
  border-color: #d4af37;
  background: #d4af37;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 按钮样式 */
:deep(.el-button--primary) {
  background-color: #d4af37;
  border-color: #d4af37;
  color: #000000;
  font-weight: 500;
}

:deep(.el-button--primary:hover) {
  background-color: #bfa13f;
  border-color: #bfa13f;
  color: #000000;
}

:deep(.el-button--default) {
  background-color: #fff;
  border-color: #dcdfe6;
  color: #606266;
}

:deep(.el-button--default:hover) {
  border-color: #d4af37;
  color: #d4af37;
  background-color: #fff;
}

/* Link 按钮样式 */
:deep(.el-button.is-link) {
  color: #409eff;
  font-weight: 500;
}

:deep(.el-button.is-link:hover) {
  color: #66b1ff;
}

/* Primary Link 按钮样式（添加按钮）- 金色文字 */
:deep(.el-button--primary.is-link) {
  color: #d4af37;
  font-weight: 500;
  background-color: transparent;
}

:deep(.el-button--primary.is-link:hover) {
  color: #bfa13f;
  background-color: transparent;
}

:deep(.el-button.is-link.is-danger) {
  color: #f56c6c;
}

:deep(.el-button.is-link.is-danger:hover) {
  color: #f78989;
}

/* Dialog 对话框背景 */
:deep(.el-dialog) {
  background-color: #fff;
}

:deep(.el-dialog__header) {
  border-bottom: 1px solid #d4af37;
  padding: 20px;
}

:deep(.el-dialog__title) {
  color: #d4af37;
  font-weight: 500;
}

:deep(.el-dialog__headerbtn .el-dialog__close) {
  color: #d4af37;
}

:deep(.el-dialog__headerbtn .el-dialog__close:hover) {
  color: #bfa13f;
}

:deep(.el-dialog__body) {
  padding: 20px;
  color: #606266;
}

:deep(.el-dialog__footer) {
  border-top: 1px solid #dcdfe6;
  padding: 15px 20px;
}

/* Form label */
:deep(.el-form-item__label) {
  color: #606266;
  font-weight: 500;
}

/* 配置头部带描述 */
.config-header-with-desc {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.header-text {
  flex: 1;
}

.header-text .title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.header-text .description {
  font-size: 13px;
  color: #909399;
  font-weight: 400;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  background-color: #fafafa;
  border-radius: 8px;
  border: 1px dashed #dcdfe6;
}

.empty-state .empty-icon {
  font-size: 48px;
  color: #c0c4cc;
  margin-bottom: 12px;
}

.empty-state p {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

/* 选择器配置 */
.selector-item {
  margin-bottom: 12px;
  padding: 16px;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  transition: all 0.3s;
}

.selector-item:hover {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.1);
}

.selector-row {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.equal-sign {
  color: #909399;
  font-weight: 600;
  font-size: 16px;
  padding-top: 26px;
  min-width: 20px;
  text-align: center;
}

.action-area {
  padding-top: 26px;
}

/* Session Affinity 配置 - 新设计 */
.affinity-config-new {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 30px;
}

.affinity-type-cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.affinity-type-card {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 24px;
  background-color: #fff;
  border: 2px solid #e4e7ed;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.affinity-type-card:hover {
  border-color: #d4af37;
  box-shadow: 0 4px 20px rgba(212, 175, 55, 0.15);
  transform: translateY(-2px);
}

.affinity-type-card.is-selected {
  border-color: #d4af37;
  background-color: #fff;
  box-shadow: 0 4px 20px rgba(212, 175, 55, 0.2);
}

.affinity-type-card .card-icon {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #fef9e7;
  border-radius: 12px;
  flex-shrink: 0;
  color: #d4af37;
  font-size: 28px;
  border: 2px solid #d4af37;
}

.affinity-type-card.is-selected .card-icon {
  background: #d4af37;
  color: #fff;
}

.affinity-type-card .card-content {
  flex: 1;
}

.affinity-type-card .card-title {
  font-size: 18px;
  font-weight: 700;
  color: #303133;
  margin-bottom: 8px;
  letter-spacing: 0.5px;
}

.affinity-type-card .card-desc {
  font-size: 13px;
  color: #909399;
  line-height: 1.6;
  font-weight: 400;
}

.affinity-type-card .card-check {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #d4af37;
  border-radius: 50%;
  color: #fff;
  font-size: 20px;
  flex-shrink: 0;
}

/* 超时配置卡片 */
.affinity-timeout-config {
  margin-top: 10px;
}

.timeout-config-card {
  padding: 24px;
  background-color: #fff;
  border: 2px solid #d4af37;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(212, 175, 55, 0.15);
}

.timeout-config-card .config-card-header {
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 2px dashed #d4af37;
}

.timeout-config-card .header-left {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.timeout-config-card .header-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #d4af37;
  border-radius: 10px;
  color: #fff;
  font-size: 24px;
  flex-shrink: 0;
}

.timeout-config-card .header-text {
  flex: 1;
}

.timeout-config-card .header-title {
  font-size: 17px;
  font-weight: 700;
  color: #303133;
  margin-bottom: 6px;
  letter-spacing: 0.5px;
}

.timeout-config-card .header-desc {
  font-size: 13px;
  color: #909399;
  line-height: 1.6;
  font-weight: 400;
}

.timeout-config-card .config-card-body {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.timeout-config-card .timeout-input-wrapper {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 20px;
  background-color: #fff;
  border-radius: 10px;
  border: 1px solid #d4af37;
}

.timeout-config-card .timeout-input {
  flex-shrink: 0;
}

.timeout-config-card .timeout-input :deep(.el-input__wrapper) {
  width: 200px;
  padding: 8px 16px;
  background-color: #fff;
  border: 2px solid #d4af37;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.timeout-config-card .timeout-input :deep(.el-input__inner) {
  font-size: 18px;
  font-weight: 700;
  color: #d4af37;
  text-align: center;
}

.timeout-config-card .time-conversions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}

.timeout-config-card .conversion-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  background-color: #fef9e7;
  border-radius: 6px;
  border: 1px solid #d4af37;
}

.timeout-config-card .conversion-label {
  font-size: 14px;
  font-weight: 600;
  color: #d4af37;
  min-width: 24px;
}

.timeout-config-card .conversion-value {
  font-size: 15px;
  font-weight: 700;
  color: #303133;
}

.timeout-config-card .timeout-info-box {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  background-color: #fef9e7;
  border-radius: 10px;
  border: 1px dashed #d4af37;
}

.timeout-config-card .info-icon {
  font-size: 24px;
  color: #d4af37;
  flex-shrink: 0;
  margin-top: 2px;
}

.timeout-config-card .info-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.timeout-config-card .info-main {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.timeout-config-card .info-sub {
  font-size: 13px;
  font-weight: 400;
  color: #909399;
  line-height: 1.5;
}

/* IP 地址配置 */
.ip-config {
  width: 100%;
}

.ip-section {
  padding: 20px;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
}

.ip-section-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.ip-section-header .header-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #fef9e7;
  border-radius: 8px;
  color: #d4af37;
  font-size: 20px;
  flex-shrink: 0;
}

.ip-section-header .header-text {
  flex: 1;
}

.ip-section-header .header-text .title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.ip-section-header .header-text .description {
  font-size: 13px;
  color: #909399;
  font-weight: 400;
}

.ip-input-group {
  margin-bottom: 12px;
}

.ip-input-group :deep(.el-input__wrapper) {
  padding-left: 12px;
}

.ip-input-group :deep(.el-input__prefix) {
  color: #909399;
}

.ip-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #909399;
  padding: 8px 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.ip-hint .el-icon {
  font-size: 14px;
  color: #d4af37;
}

.ip-hint span {
  flex: 1;
}

/* 外部 IP 列表 */
.external-ip-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.external-ip-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.ip-input-wrapper {
  flex: 1;
}

.ip-actions {
  display: flex;
  gap: 8px;
}

/* 键值对列表 */
.kv-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.kv-item {
  padding: 16px;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  transition: all 0.3s;
}

.kv-item:hover {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.1);
}

.kv-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.kv-fields {
  display: flex;
  align-items: center;
  flex: 1;
  gap: 16px;
}

.kv-fields .field-group {
  flex: 1;
}

.kv-actions {
  display: flex;
  align-items: center;
  padding-top: 26px;
}

/* 端口配置优化 */
.ports-config {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 标签和注解之间的分隔 */
.labels-config,
.annotations-config {
  width: 100%;
}

/* 网络资源编辑弹窗统一视觉：去掉旧金色，保持和 OpsHub 其他模块一致 */
:deep(.network-resource-dialog.el-dialog),
:deep(.network-resource-dialog .el-dialog) {
  --net-dialog-ink: #111827;
  --net-dialog-muted: #6b7280;
  --net-dialog-border: #e5e7eb;
  --net-dialog-soft: #f8fafc;
  --net-dialog-panel: #ffffff;
  border-radius: 18px;
  overflow: hidden;
  background: var(--net-dialog-panel);
  box-shadow: 0 24px 70px rgba(15, 23, 42, 0.2);
}

:deep(.network-resource-dialog .el-dialog__header),
:deep(.network-resource-dialog.el-dialog .el-dialog__header) {
  padding: 20px 24px;
  border-bottom: 1px solid var(--net-dialog-border);
  background: #ffffff;
}

:deep(.network-resource-dialog .el-dialog__title),
:deep(.network-resource-dialog.el-dialog .el-dialog__title) {
  color: var(--net-dialog-ink);
  font-size: 19px;
  font-weight: 800;
  letter-spacing: -0.01em;
}

:deep(.network-resource-dialog .el-dialog__headerbtn .el-dialog__close),
:deep(.network-resource-dialog.el-dialog .el-dialog__headerbtn .el-dialog__close) {
  color: #9ca3af;
}

:deep(.network-resource-dialog .el-dialog__headerbtn:hover .el-dialog__close),
:deep(.network-resource-dialog.el-dialog .el-dialog__headerbtn:hover .el-dialog__close) {
  color: var(--net-dialog-ink);
}

:deep(.network-resource-dialog .el-dialog__body),
:deep(.network-resource-dialog.el-dialog .el-dialog__body) {
  padding: 20px 24px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
  color: var(--net-dialog-ink);
}

:deep(.network-resource-dialog .el-dialog__footer),
:deep(.network-resource-dialog.el-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid var(--net-dialog-border);
  background: #ffffff;
}

.basic-info-section {
  margin-bottom: 18px;
  padding: 18px 18px 2px;
  border: 1px solid var(--net-dialog-border);
  border-radius: 14px;
  background: #ffffff;
}

.service-tabs {
  margin-top: 0;
}

.service-tabs :deep(.el-tabs__header) {
  margin: 0 0 16px;
  padding: 0 2px;
  border-bottom: 1px solid var(--net-dialog-border);
}

.service-tabs :deep(.el-tabs__item) {
  height: 42px;
  color: var(--net-dialog-muted);
  font-size: 14px;
  font-weight: 700;
}

.service-tabs :deep(.el-tabs__item:hover),
.service-tabs :deep(.el-tabs__item.is-active) {
  color: var(--net-dialog-ink);
}

.service-tabs :deep(.el-tabs__active-bar) {
  height: 3px;
  border-radius: 999px 999px 0 0;
  background: var(--net-dialog-ink);
}

.tab-content {
  min-height: 320px;
}

.port-item,
.selector-config,
.ip-section,
.labels-config,
.annotations-config,
.timeout-config-card,
.affinity-type-card,
.selector-item,
.kv-item {
  border-color: var(--net-dialog-border);
  background: #ffffff;
  box-shadow: 0 8px 22px rgba(15, 23, 42, 0.04);
}

.port-header,
.paths-title,
.config-header,
.header-text .title,
.ip-section-header .header-text .title,
.timeout-config-card .header-title,
.affinity-type-card .card-title {
  color: var(--net-dialog-ink);
  font-weight: 800;
}

.field-group label,
:deep(.el-form-item__label) {
  color: #374151;
  font-weight: 700;
}

:deep(.el-input__wrapper),
:deep(.el-select .el-input__wrapper),
:deep(.el-input-number .el-input__wrapper) {
  border: 1px solid #d8dee8;
  border-radius: 10px;
  background: #ffffff;
  box-shadow: none;
}

:deep(.el-input__wrapper:hover),
:deep(.el-select .el-input__wrapper:hover),
:deep(.el-input-number .el-input__wrapper:hover) {
  border-color: #9ca3af;
  box-shadow: none;
}

:deep(.el-input__wrapper.is-focus),
:deep(.el-select .el-input__wrapper.is-focus),
:deep(.el-input-number .el-input__wrapper.is-focus) {
  border-color: #111827;
  box-shadow: 0 0 0 2px rgba(17, 24, 39, 0.08);
}

:deep(.el-input-number__decrease),
:deep(.el-input-number__increase) {
  border-color: #d8dee8;
  background: #f9fafb;
  color: #6b7280;
}

:deep(.el-input-number__decrease:hover),
:deep(.el-input-number__increase:hover),
.ip-input-group :deep(.el-input__prefix),
.ip-hint .el-icon {
  color: #111827;
}

:deep(.el-button--primary:not(.is-link)) {
  height: 36px;
  padding: 0 18px;
  border: 1px solid #111827;
  border-radius: 10px;
  background: #111827;
  color: #ffffff;
  font-weight: 800;
  box-shadow: none;
}

:deep(.el-button--primary:not(.is-link):hover) {
  border-color: #374151;
  background: #374151;
  color: #ffffff;
}

:deep(.el-button--default) {
  height: 36px;
  padding: 0 18px;
  border: 1px solid #d8dee8;
  border-radius: 10px;
  background: #ffffff;
  color: #4b5563;
  font-weight: 700;
}

:deep(.el-button--default:hover) {
  border-color: #9ca3af;
  background: #f9fafb;
  color: #111827;
}

:deep(.el-button--primary.is-link) {
  height: 32px;
  padding: 0 12px;
  border: 1px solid #d1d5db;
  border-radius: 9px;
  background: #ffffff;
  color: #111827;
  font-weight: 800;
}

:deep(.el-button--primary.is-link:hover) {
  border-color: #9ca3af;
  background: #f3f4f6;
  color: #111827;
}

:deep(.el-button.is-link.is-danger),
:deep(.el-button--danger.is-link) {
  width: 32px;
  min-width: 32px;
  height: 32px;
  padding: 0;
  border: 1px solid #fecdd3;
  border-radius: 9px;
  background: #fff1f2;
  color: #dc2626;
}

:deep(.el-button.is-link.is-danger:hover),
:deep(.el-button--danger.is-link:hover) {
  border-color: #fca5a5;
  background: #fee2e2;
  color: #b91c1c;
}

:deep(.el-radio.is-checked .el-radio__inner),
:deep(.el-radio__input.is-checked .el-radio__inner) {
  border-color: #111827;
  background: #111827;
}

.affinity-type-card {
  border: 1px solid var(--net-dialog-border);
  border-radius: 16px;
}

.affinity-type-card:hover {
  border-color: #9ca3af;
  box-shadow: 0 14px 28px rgba(15, 23, 42, 0.08);
  transform: none;
}

.affinity-type-card.is-selected {
  border-color: #111827;
  background: #ffffff;
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.1);
}

.affinity-type-card .card-icon,
.ip-section-header .header-icon,
.timeout-config-card .header-icon {
  border: 1px solid #e5e7eb;
  background: #f3f4f6;
  color: #111827;
}

.affinity-type-card.is-selected .card-icon,
.affinity-type-card .card-check {
  background: #111827;
  color: #ffffff;
}

.timeout-config-card {
  border: 1px solid var(--net-dialog-border);
  border-radius: 16px;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.05);
}

.timeout-config-card .config-card-header {
  border-bottom: 1px dashed #d1d5db;
}

.timeout-config-card .timeout-input-wrapper,
.timeout-config-card .conversion-item,
.timeout-config-card .timeout-info-box {
  border-color: var(--net-dialog-border);
  background: #f9fafb;
}

.timeout-config-card .timeout-input :deep(.el-input__wrapper) {
  border-color: #d8dee8;
}

.timeout-config-card .timeout-input :deep(.el-input__inner),
.timeout-config-card .conversion-label,
.timeout-config-card .info-icon {
  color: #111827;
}

.selector-item:hover,
.kv-item:hover {
  border-color: #9ca3af;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.06);
}

.empty-state {
  border-color: #d8dee8;
  background: #f9fafb;
}

.dialog-footer {
  gap: 10px;
}
</style>
