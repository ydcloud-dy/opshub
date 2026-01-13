<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑 Ingress' : '创建 Ingress'"
    width="1000px"
    :close-on-click-modal="false"
    :lock-scroll="false"
    @close="handleClose"
  >
    <!-- 基本信息区域 -->
    <div class="basic-info-section">
      <el-form ref="formRef" :model="formData" :rules="rules" label-width="100px">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="名称" prop="name">
              <el-input v-model="formData.name" placeholder="Ingress 名称" :disabled="isEdit" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="命名空间" prop="namespace">
              <el-select v-model="formData.namespace" placeholder="选择命名空间" :disabled="isEdit" style="width: 100%">
                <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </div>

    <!-- Tab 导航 -->
    <el-tabs v-model="activeTab" class="ingress-tabs">
      <!-- 规则配置 -->
      <el-tab-pane label="规则" name="rules">
        <div class="tab-content">
          <div class="rules-config">
            <div v-for="(rule, ruleIndex) in formData.rules" :key="ruleIndex" class="rule-item">
              <div class="rule-header">
                <div class="rule-title">
                  <el-icon class="title-icon"><Document /></el-icon>
                  <span>规则 {{ ruleIndex + 1 }}</span>
                </div>
                <el-button type="danger" link @click="removeRule(ruleIndex)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>

              <!-- 主机名 -->
              <div class="rule-host-section">
                <div class="field-group">
                  <label>主机名</label>
                  <el-input v-model="rule.host" placeholder="例如: example.com" size="small" />
                </div>
              </div>

              <!-- 路径配置 -->
              <div class="rule-paths-section">
                <div class="paths-header">
                  <div class="paths-title">路径配置</div>
                  <el-button type="primary" link @click="addPath(ruleIndex)">
                    <el-icon><Plus /></el-icon> 添加路径
                  </el-button>
                </div>
                <div v-if="rule.paths.length === 0" class="empty-state">
                  <el-icon class="empty-icon"><Link /></el-icon>
                  <p>暂无路径配置，点击上方"添加路径"按钮添加</p>
                </div>
                <div v-else class="paths-list">
                  <div v-for="(path, pathIndex) in rule.paths" :key="pathIndex" class="path-card">
                    <div class="path-card-header">
                      <span>路径 {{ pathIndex + 1 }}</span>
                      <el-button type="danger" link @click="removePath(ruleIndex, pathIndex)">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </div>
                    <div class="path-card-body">
                      <el-row :gutter="12">
                        <el-col :span="6">
                          <div class="field-group">
                            <label>路径</label>
                            <el-input v-model="path.path" placeholder="例如: /api" size="small" />
                          </div>
                        </el-col>
                        <el-col :span="6">
                          <div class="field-group">
                            <label>匹配类型</label>
                            <el-select v-model="path.pathType" size="small" style="width: 100%">
                              <el-option label="Prefix" value="Prefix" />
                              <el-option label="Exact" value="Exact" />
                              <el-option label="ImplementationSpecific" value="ImplementationSpecific" />
                            </el-select>
                          </div>
                        </el-col>
                        <el-col :span="6">
                          <div class="field-group">
                            <label>Service 名称</label>
                            <el-select v-model="path.service" placeholder="选择服务" size="small" style="width: 100%" filterable>
                              <el-option v-for="svc in servicesList" :key="svc.name" :label="svc.name" :value="svc.name" />
                            </el-select>
                          </div>
                        </el-col>
                        <el-col :span="6">
                          <div class="field-group">
                            <label>端口</label>
                            <el-input-number v-model="path.port" :min="1" :max="65535" size="small" style="width: 100%" />
                          </div>
                        </el-col>
                      </el-row>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <el-button type="primary" link @click="addRule" class="add-rule-btn">
              <el-icon><Plus /></el-icon> 添加规则
            </el-button>
          </div>
        </div>
      </el-tab-pane>

      <!-- 证书配置 -->
      <el-tab-pane label="证书" name="tls">
        <div class="tab-content">
          <div class="tls-config">
            <div v-if="formData.tls.length === 0" class="empty-state">
              <el-icon class="empty-icon"><Lock /></el-icon>
              <p>暂无证书配置，点击下方"添加证书"按钮添加</p>
            </div>
            <div v-else>
              <div v-for="(tls, tlsIndex) in formData.tls" :key="tlsIndex" class="tls-item">
                <div class="tls-header">
                  <div class="tls-title">
                    <el-icon class="title-icon"><Lock /></el-icon>
                    <span>证书 {{ tlsIndex + 1 }}</span>
                  </div>
                  <el-button type="danger" link @click="removeTLS(tlsIndex)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>

                <!-- 主机名列表 -->
                <div class="tls-hosts-section">
                  <div class="field-group">
                    <label>主机名</label>
                    <div class="tls-hosts-list">
                      <el-tag
                        v-for="(host, hostIdx) in tls.hosts"
                        :key="hostIdx"
                        closable
                        @close="removeTLSHost(tlsIndex, hostIdx)"
                        class="host-tag"
                      >
                        {{ host }}
                      </el-tag>
                      <el-input
                        v-model="newTLSHost[tlsIndex]"
                        placeholder="输入主机名后回车添加"
                        @keyup.enter="addTLSHost(tlsIndex)"
                        size="small"
                        class="host-input"
                      />
                    </div>
                  </div>
                </div>

                <!-- Secret 名称 -->
                <div class="tls-secret-section">
                  <div class="field-group">
                    <label>Secret 名称</label>
                    <el-select
                      v-model="tls.secretName"
                      placeholder="选择 Secret"
                      size="small"
                      style="width: 100%"
                      filterable
                    >
                      <el-option
                        v-for="secret in secretsList"
                        :key="secret.name"
                        :label="secret.name"
                        :value="secret.name"
                      >
                        <div style="display: flex; justify-content: space-between; align-items: center;">
                          <span>{{ secret.name }}</span>
                          <el-tag v-if="secret.type" size="small" style="margin-left: 8px;">{{ secret.type }}</el-tag>
                        </div>
                      </el-option>
                    </el-select>
                  </div>
                </div>
              </div>
            </div>
            <el-button type="primary" link @click="addTLS" class="add-tls-btn">
              <el-icon><Plus /></el-icon> 添加证书
            </el-button>
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
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete, Plus, Document, Lock, Link, PriceTag } from '@element-plus/icons-vue'
import { getIngressYAML, updateIngressYAML, createIngress, getSecrets, getServices, type IngressInfo } from '@/api/kubernetes'

interface PathConfig {
  path: string
  pathType: string
  service: string
  port: number
}

interface RuleConfig {
  host: string
  paths: PathConfig[]
}

interface TLSConfig {
  hosts: string[]
  secretName: string
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
const activeTab = ref('rules')
const newTLSHost = ref<string[]>([])
const secretsList = ref<any[]>([])
const servicesList = ref<any[]>([])

const formData = ref({
  name: '',
  namespace: '',
  rules: [] as RuleConfig[],
  tls: [] as TLSConfig[],
  labels: {} as Record<string, string>,
  annotations: {} as Record<string, string>
})

const labelsList = ref<KeyValueItem[]>([])
const annotationsList = ref<KeyValueItem[]>([])

const rules = {
  name: [{ required: true, message: '请输入 Ingress 名称', trigger: 'blur' }],
  namespace: [{ required: true, message: '请选择命名空间', trigger: 'change' }]
}

// 打开对话框（编辑模式）
const openEdit = async (ingress: IngressInfo, nsList: any[]) => {
  namespaces.value = nsList
  isEdit.value = true
  visible.value = true
  activeTab.value = 'rules'

  try {
    const response = await getIngressYAML(props.clusterId!, ingress.namespace, ingress.name)
    originalData.value = response.items || response

    const spec = originalData.value.spec || {}
    const metadata = originalData.value.metadata || {}

    // 解析规则
    const rules: RuleConfig[] = []
    if (spec.rules && spec.rules.length > 0) {
      spec.rules.forEach((rule: any) => {
        const ruleConfig: RuleConfig = {
          host: rule.host || '',
          paths: []
        }

        if (rule.http && rule.http.paths) {
          rule.http.paths.forEach((p: any) => {
            ruleConfig.paths.push({
              path: p.path || '/',
              pathType: p.pathType || 'Prefix',
              service: p.backend?.service?.name || '',
              port: p.backend?.service?.port?.number || 80
            })
          })
        }

        rules.push(ruleConfig)
      })
    }

    // 解析 TLS
    const tls: TLSConfig[] = []
    if (spec.tls) {
      spec.tls.forEach((t: any) => {
        tls.push({
          hosts: t.hosts || [],
          secretName: t.secretName || ''
        })
      })
    }

    formData.value = {
      name: metadata.name || '',
      namespace: metadata.namespace || '',
      rules,
      tls,
      labels: metadata.labels || {},
      annotations: metadata.annotations || {}
    }

    // 同步到列表
    syncLabelsFromForm()
    syncAnnotationsFromForm()

    // 初始化 TLS 主机输入
    newTLSHost.value = formData.value.tls.map(() => '')

    // 加载 Secret 列表
    await loadSecrets()

    // 加载 Service 列表
    await loadServices()

    if (formData.value.rules.length === 0) {
      addRule()
    }
  } catch (error) {
    console.error('获取 Ingress 详情失败:', error)
    ElMessage.error('获取 Ingress 详情失败')
  }
}

// 打开对话框（创建模式）
const openCreate = (nsList: any[]) => {
  namespaces.value = nsList
  isEdit.value = false
  visible.value = true
  activeTab.value = 'rules'

  formData.value = {
    name: '',
    namespace: '',
    rules: [],
    tls: [],
    labels: {},
    annotations: {}
  }

  labelsList.value = []
  annotationsList.value = []
  newTLSHost.value = []
  secretsList.value = []

  // 添加默认规则
  addRule()
}

// 同步方法
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

// 规则操作
const addRule = () => {
  formData.value.rules.push({
    host: '',
    paths: [{
      path: '/',
      pathType: 'Prefix',
      service: '',
      port: 80
    }]
  })
}

const removeRule = (index: number) => {
  formData.value.rules.splice(index, 1)
}

const addPath = (ruleIndex: number) => {
  const rule = formData.value.rules[ruleIndex]
  if (rule) {
    rule.paths.push({
      path: '/',
      pathType: 'Prefix',
      service: '',
      port: 80
    })
  }
}

const removePath = (ruleIndex: number, pathIndex: number) => {
  const rule = formData.value.rules[ruleIndex]
  if (rule) {
    rule.paths.splice(pathIndex, 1)
  }
}

// TLS 操作
const addTLS = () => {
  formData.value.tls.push({
    hosts: [],
    secretName: ''
  })
  newTLSHost.value.push('')
}

const removeTLS = (index: number) => {
  formData.value.tls.splice(index, 1)
  newTLSHost.value.splice(index, 1)
}

const addTLSHost = (tlsIndex: number) => {
  const host = newTLSHost.value[tlsIndex]
  if (host) {
    const tls = formData.value.tls[tlsIndex]
    if (tls) {
      tls.hosts.push(host)
      newTLSHost.value[tlsIndex] = ''
    }
  }
}

const removeTLSHost = (tlsIndex: number, hostIndex: number) => {
  const tls = formData.value.tls[tlsIndex]
  if (tls) {
    tls.hosts.splice(hostIndex, 1)
  }
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

// 加载 Secret 列表
const loadSecrets = async () => {
  if (!props.clusterId || !formData.value.namespace) return
  try {
    const data = await getSecrets(props.clusterId, formData.value.namespace)
    secretsList.value = (data || []).filter((s: any) =>
      s.type === 'kubernetes.io/tls' || s.type === 'cert-manager.io/v1alpha1'
    )
  } catch (error) {
    console.error('获取 Secret 列表失败:', error)
  }
}

// 加载 Service 列表
const loadServices = async () => {
  if (!props.clusterId || !formData.value.namespace) return
  try {
    const data = await getServices(props.clusterId, formData.value.namespace)
    servicesList.value = data || []
  } catch (error) {
    console.error('获取 Service 列表失败:', error)
  }
}

// 监听命名空间变化，自动加载 Secret 列表和 Service 列表
watch(() => formData.value.namespace, () => {
  if (formData.value.namespace) {
    loadSecrets()
    loadServices()
  }
})

// 构建保存的数据
const buildSaveData = () => {
  // 同步标签和注解
  syncLabelsToList()
  syncAnnotationsToList()

  // 构建 rules
  const rules = formData.value.rules
    .filter(rule => rule.paths.length > 0)
    .map(rule => {
      const ruleData: any = {}

      if (rule.host) {
        ruleData.host = rule.host
      }

      ruleData.http = {
        paths: rule.paths.map(path => ({
          path: path.path,
          pathType: path.pathType,
          backend: {
            service: {
              name: path.service,
              port: {
                number: path.port
              }
            }
          }
        }))
      }

      return ruleData
    })

  // 构建 Ingress 对象
  const ingressData: any = {
    apiVersion: 'networking.k8s.io/v1',
    kind: 'Ingress',
    metadata: {
      name: formData.value.name,
      namespace: formData.value.namespace
    },
    spec: {
      rules
    }
  }

  // 添加 TLS（显式处理空数组的情况）
  // 只保留有 secretName 的 TLS 配置，hosts 可选
  const validTLS = formData.value.tls.filter(t => t.secretName)
  // 如果编辑模式下原本有 TLS 配置但被删除了，需要显式设置 tls 为空数组
  const originalHasTLS = isEdit.value && originalData.value?.spec?.tls && originalData.value.spec.tls.length > 0
  if (originalHasTLS && formData.value.tls.length === 0) {
    ingressData.spec.tls = []
  } else if (validTLS.length > 0) {
    ingressData.spec.tls = validTLS.map(t => ({
      hosts: t.hosts.length > 0 ? t.hosts : undefined,
      secretName: t.secretName
    }))
  }

  // 如果是编辑模式，保留原有的 metadata 字段
  if (isEdit.value && originalData.value) {
    ingressData.metadata = {
      ...originalData.value.metadata,
      name: formData.value.name,
      namespace: formData.value.namespace
    }

    // 添加 labels 和 annotations
    if (Object.keys(formData.value.labels).length > 0) {
      ingressData.metadata.labels = formData.value.labels
    }
    if (Object.keys(formData.value.annotations).length > 0) {
      ingressData.metadata.annotations = formData.value.annotations
    }
  } else {
    // 创建模式
    if (Object.keys(formData.value.labels).length > 0) {
      ingressData.metadata.labels = formData.value.labels
    }
    if (Object.keys(formData.value.annotations).length > 0) {
      ingressData.metadata.annotations = formData.value.annotations
    }
  }

  return ingressData
}

// 保存
const handleSave = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  // 验证配置
  if (formData.value.rules.length === 0) {
    ElMessage.error('请至少配置一个规则')
    return
  }

  const hasValidPath = formData.value.rules.some(rule =>
    rule.paths.some(path => path.service && path.port > 0)
  )

  if (!hasValidPath) {
    ElMessage.error('请至少配置一个有效的路径（包含服务名称和端口）')
    return
  }

  // 验证 TLS 配置：如果配置了证书，secretName 必须填写
  const invalidTLS = formData.value.tls.filter(t => !t.secretName && t.hosts.length === 0)
  if (invalidTLS.length > 0) {
    ElMessage.error('请填写 TLS 证书的 Secret 名称或删除无效的证书配置')
    return
  }

  // 只保留有 secretName 的 TLS 配置
  const tlsWithSecret = formData.value.tls.filter(t => t.secretName)
  if (tlsWithSecret.length !== formData.value.tls.length) {
    ElMessage.warning('已自动过滤未填写 Secret 名称的证书配置')
    formData.value.tls = tlsWithSecret
  }

  saving.value = true
  try {
    const ingressData = buildSaveData()

    if (isEdit.value) {
      await updateIngressYAML(
        props.clusterId!,
        formData.value.namespace,
        formData.value.name,
        ingressData
      )
      ElMessage.success('更新成功')
    } else {
      // 构建创建请求数据
      const createData = {
        name: formData.value.name,
        rules: formData.value.rules
          .filter(rule => rule.paths.length > 0)
          .map(rule => ({
            host: rule.host || undefined,
            paths: rule.paths
              .filter(p => p.service)
              .map(p => ({
                path: p.path,
                pathType: p.pathType,
                service: p.service,
                port: p.port
              }))
          })),
        tls: formData.value.tls
          .filter(t => t.secretName)
          .map(t => ({
            hosts: t.hosts.length > 0 ? t.hosts : undefined,
            secretName: t.secretName
          }))
      }

      await createIngress(props.clusterId!, formData.value.namespace, createData)
      ElMessage.success('创建成功')
    }

    emit('success')
    handleClose()
  } catch (error) {
    console.error('保存失败:', error)
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
  activeTab.value = 'rules'
  labelsList.value = []
  annotationsList.value = []
  newTLSHost.value = []
  secretsList.value = []
  servicesList.value = []
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
.ingress-tabs {
  margin-top: 10px;
}

.ingress-tabs :deep(.el-tabs__header) {
  margin-bottom: 20px;
}

.ingress-tabs :deep(.el-tabs__item) {
  color: #606266;
  font-weight: 500;
}

.ingress-tabs :deep(.el-tabs__item.is-active) {
  color: #d4af37;
}

.ingress-tabs :deep(.el-tabs__active-bar) {
  background-color: #d4af37;
}

.tab-content {
  min-height: 400px;
}

/* 规则配置 */
.rules-config {
  width: 100%;
}

.rule-item {
  margin-bottom: 24px;
  padding: 20px;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
}

.rule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
}

.rule-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.rule-title .title-icon {
  color: #d4af37;
  font-size: 18px;
}

.rule-host-section {
  margin-bottom: 16px;
}

.rule-paths-section {
  padding-top: 16px;
  border-top: 1px dashed #e4e7ed;
}

.paths-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.paths-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.paths-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.path-card {
  padding: 16px;
  background-color: #fef9e7;
  border: 1px solid #d4af37;
  border-radius: 6px;
}

.path-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.path-card-body {
  margin-top: 12px;
}

/* TLS 配置 */
.tls-config {
  width: 100%;
}

.tls-item {
  margin-bottom: 20px;
  padding: 20px;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
}

.tls-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
}

.tls-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.tls-title .title-icon {
  color: #d4af37;
  font-size: 18px;
}

.tls-hosts-section {
  margin-bottom: 16px;
}

.tls-hosts-list {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.host-tag {
  background-color: #fef9e7;
  border-color: #d4af37;
  color: #303133;
}

.host-input {
  width: 200px;
}

.tls-secret-section {
  margin-top: 16px;
}

/* 通用样式 */
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

.equal-sign {
  color: #909399;
  font-weight: 600;
  font-size: 16px;
  padding-top: 26px;
  min-width: 20px;
  text-align: center;
}

.kv-actions {
  display: flex;
  align-items: center;
  padding-top: 26px;
}

/* 按钮样式 */
.add-rule-btn,
.add-tls-btn {
  margin-top: 16px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
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
</style>
