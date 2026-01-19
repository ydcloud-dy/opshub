<template>
  <div class="cloud-accounts-page">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Cloudy /></el-icon>
        </div>
        <div>
          <h2 class="page-title">云账号管理</h2>
          <p class="page-subtitle">管理云平台账号，用于导入云主机</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="handleAdd" class="black-button">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增账号
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.keyword"
          placeholder="搜索账号名称..."
          clearable
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.provider"
          placeholder="云厂商"
          clearable
          class="search-input"
        >
          <el-option label="全部" value="" />
          <el-option label="阿里云" value="aliyun" />
          <el-option label="腾讯云" value="tencent" />
          <el-option label="AWS" value="aws" />
          <el-option label="华为云" value="huawei" />
        </el-select>

        <el-select
          v-model="searchForm.status"
          placeholder="状态"
          clearable
          class="search-input"
        >
          <el-option label="全部" value="" />
          <el-option label="启用" :value="1" />
          <el-option label="禁用" :value="0" />
        </el-select>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 账号列表 -->
    <div class="table-wrapper">
      <el-table :data="filteredAccountList" v-loading="loading" stripe class="modern-table" :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }">
        <el-table-column label="账号名称" prop="name" min-width="150" />
        <el-table-column label="云厂商" align="center" width="120">
          <template #default="{ row }">
            <el-tag :type="getProviderType(row.provider)">
              {{ row.providerText }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="区域" prop="region" min-width="120">
          <template #default="{ row }">
            <span v-if="row.region">{{ row.region }}</span>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="Access Key" min-width="200">
          <template #default="{ row }">
            <span class="access-key">************</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" align="center" width="80">
          <template #default="{ row }">
            <el-switch
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              @change="handleStatusChange(row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createTime" width="180" />
        <el-table-column label="操作" width="120" align="center" fixed="right">
          <template #default="{ row }">
            <el-button
              link
              :type="row.status === 1 ? 'primary' : 'info'"
              :disabled="row.status === 0"
              @click="handleImportHost(row)"
              title="导入主机"
            >
              <el-icon><Upload /></el-icon>
            </el-button>
            <el-button link type="primary" @click="handleEdit(row)" title="编辑">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)" title="删除">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑云账号' : '新增云账号'"
      width="70%"
      :style="{ maxWidth: '800px' }"
      @close="handleDialogClose"
      class="account-dialog"
    >
      <div class="dialog-content">
        <el-form :model="form" :rules="rules" ref="formRef" label-width="120px" class="account-form">
          <!-- 云厂商选择 -->
          <el-form-item label="云厂商" prop="provider" required>
            <div class="provider-options-inline">
              <div
                v-for="provider in providers"
                :key="provider.value"
                :class="['provider-option', { active: form.provider === provider.value }]"
                @click="form.provider = provider.value"
              >
                <span class="provider-short">{{ provider.short }}</span>
                <span class="provider-name">{{ provider.label }}</span>
              </div>
            </div>
          </el-form-item>

          <el-form-item label="账号名称" prop="name">
            <el-input v-model="form.name" placeholder="如：生产环境阿里云账号" />
          </el-form-item>

          <template v-if="!isEdit">
            <el-form-item label="Access Key" prop="accessKey">
              <el-input v-model="form.accessKey" placeholder="请输入 Access Key ID" />
            </el-form-item>

            <el-form-item label="Secret Key" prop="secretKey">
              <el-input v-model="form.secretKey" type="password" show-password placeholder="请输入 Access Key Secret" />
            </el-form-item>
          </template>

          <el-alert v-else type="info" :closable="false" style="margin-bottom: 20px;">
            <template #title>
              <span style="font-size: 13px;">如需修改 Access Key 或 Secret Key，请删除后重新创建账号</span>
            </template>
          </el-alert>

          <el-form-item label="默认区域">
            <el-select v-model="form.region" placeholder="选择默认区域" filterable style="width: 100%">
              <el-option v-for="region in currentRegions" :key="region.value" :label="region.label" :value="region.value" />
            </el-select>
          </el-form-item>

          <el-form-item label="备注">
            <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选，填写备注信息" />
          </el-form-item>

          <el-form-item label="状态">
            <el-radio-group v-model="form.status">
              <el-radio :value="1">启用</el-radio>
              <el-radio :value="0">禁用</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit" :loading="submitting">
            {{ isEdit ? '保存修改' : '创建账号' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 导入云主机对话框 -->
    <el-dialog
      v-model="importDialogVisible"
      title="导入云主机"
      width="70%"
      :style="{ maxWidth: '1200px' }"
      @close="handleImportDialogClose"
      class="import-dialog"
    >
      <el-form :model="importForm" label-width="100px" class="import-form">
        <div class="form-row">
          <el-form-item label="云账号" class="form-item-full">
            <el-select v-model="importForm.accountId" placeholder="请选择云账号" style="width: 100%" @change="handleAccountChange">
              <el-option v-for="acc in enabledAccountList" :key="acc.id" :label="acc.name" :value="acc.id">
                <span>{{ acc.name }}</span>
                <el-tag :type="getProviderType(acc.provider)" size="small" style="margin-left: 8px;">
                  {{ acc.providerText }}
                </el-tag>
              </el-option>
            </el-select>
          </el-form-item>
        </div>
        <div class="form-row">
          <el-form-item label="区域" class="form-item-half">
            <el-select v-model="importForm.region" placeholder="请选择区域" style="width: 100%" filterable>
              <el-option v-for="region in regions" :key="region.value" :label="region.label" :value="region.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="所属分组" class="form-item-half">
            <el-tree-select
              v-model="importForm.groupId"
              :data="groupTreeOptions"
              :props="{ value: 'id', label: 'name', children: 'children' }"
              clearable
              check-strictly
              placeholder="请选择分组"
              style="width: 100%"
            />
          </el-form-item>
        </div>
      </el-form>

      <div v-loading="loadingInstances" class="instances-container">
        <el-alert v-if="!selectedAccount" title="请先选择云账号" type="info" :closable="false" />
        <el-alert v-else-if="!importForm.region" title="请选择区域" type="info" :closable="false" />
        <div v-else-if="cloudHosts.length === 0" class="empty-instances">
          <el-empty description="该区域下没有可导入的云主机" />
        </div>
        <div v-else class="instances-list">
          <div class="instances-header">
            <div class="instances-info">
              <span class="instances-count">找到 <strong>{{ cloudHosts.length }}</strong> 台云主机</span>
              <span class="instances-region">当前区域: {{ importForm.region }}</span>
            </div>
            <el-checkbox v-model="selectAll" @change="handleSelectAll" size="large">
              <span class="select-all-text">全选</span>
            </el-checkbox>
          </div>
          <el-table
            ref="cloudHostsTableRef"
            :data="cloudHosts"
            @selection-change="handleSelectionChange"
            :max-height="400"
            class="cloud-hosts-table"
            :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
            stripe
          >
            <el-table-column type="selection" width="50" align="center" />
            <el-table-column label="实例名称" prop="name" min-width="160" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="instance-name">
                  <el-icon class="instance-icon"><Monitor /></el-icon>
                  <span>{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="实例ID" prop="instanceId" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                <span class="instance-id">{{ row.instanceId }}</span>
              </template>
            </el-table-column>
            <el-table-column label="IP地址" min-width="150">
              <template #default="{ row }">
                <div class="ip-list">
                  <div v-if="row.publicIp" class="ip-item public-ip">
                    <el-tag size="small" type="success">公</el-tag>
                    <span>{{ row.publicIp }}</span>
                  </div>
                  <div v-if="row.privateIp" class="ip-item private-ip">
                    <el-tag size="small" type="info">私</el-tag>
                    <span>{{ row.privateIp }}</span>
                  </div>
                  <span v-if="!row.publicIp && !row.privateIp" class="text-muted">-</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="操作系统" prop="os" min-width="120" show-overflow-tooltip />
            <el-table-column label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="getStatusType(row.status)" size="small">
                  {{ getStatusText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <template #footer>
        <el-button @click="importDialogVisible = false" size="large">取消</el-button>
        <el-button type="primary" @click="handleConfirmImport" :loading="importing" :disabled="selectedInstances.length === 0" size="large">
          <el-icon v-if="!importing"><Upload /></el-icon>
          <span>导入 {{ selectedInstances.length > 0 ? `(${selectedInstances.length})` : '' }}</span>
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Edit,
  Delete,
  Upload,
  Cloudy,
  Search,
  RefreshLeft,
  Monitor
} from '@element-plus/icons-vue'
import {
  getCloudAccounts,
  createCloudAccount,
  updateCloudAccount,
  deleteCloudAccount,
  importFromCloud,
  getCloudInstances,
  getCloudRegions
} from '@/api/host'
import { getGroupTree } from '@/api/assetGroup'

const loading = ref(false)
const accountList = ref<any[]>([])

// 搜索表单
const searchForm = reactive({
  keyword: '',
  provider: '',
  status: ''
})

// 过滤后的账号列表
const filteredAccountList = computed(() => {
  return accountList.value.filter((account: any) => {
    const matchKeyword = !searchForm.keyword || account.name?.toLowerCase().includes(searchForm.keyword.toLowerCase())
    const matchProvider = !searchForm.provider || account.provider === searchForm.provider
    const matchStatus = searchForm.status === '' || account.status === parseInt(searchForm.status)
    return matchKeyword && matchProvider && matchStatus
  })
})

// 重置搜索
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.provider = ''
  searchForm.status = ''
}

// 对话框相关
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref()

// 云厂商选项
const providers = [
  { value: 'aliyun', label: '阿里云', short: '阿里' },
  { value: 'tencent', label: '腾讯云', short: '腾讯' },
  { value: 'aws', label: 'AWS', short: 'AWS' },
  { value: 'huawei', label: '华为云', short: '华为' }
]

// 当前厂商的区域列表（新增/编辑对话框用）
const currentRegions = ref<any[]>([])

const form = reactive({
  id: 0,
  name: '',
  provider: 'aliyun',
  accessKey: '',
  secretKey: '',
  region: '',
  description: '',
  status: 1
})

// 动态验证规则
const rules = computed(() => {
  const baseRules: any = {
    name: [{ required: true, message: '请输入账号名称', trigger: 'blur' }],
    provider: [{ required: true, message: '请选择云厂商', trigger: 'change' }]
  }

  // 只有新增时才验证 Access Key 和 Secret Key
  if (!isEdit.value) {
    baseRules.accessKey = [{ required: true, message: '请输入Access Key', trigger: 'blur' }]
    baseRules.secretKey = [{ required: true, message: '请输入Secret Key', trigger: 'blur' }]
  }

  return baseRules
})

// 获取启用的云账号列表
const enabledAccountList = computed(() => {
  return accountList.value.filter((a: any) => a.status === 1)
})

// 导入相关
const importDialogVisible = ref(false)
const importing = ref(false)
const loadingInstances = ref(false)
const selectedAccount = ref<any>(null)
const cloudHosts = ref<any[]>([])
const selectedInstances = ref<string[]>([])
const selectAll = ref(false)
const groupTreeOptions = ref<any[]>([])
const cloudHostsTableRef = ref()

const importForm = reactive({
  accountId: null as number | null,
  region: '',
  groupId: null as number | null
})

const regions = ref<any[]>([])

// 获取云厂商类型
const getProviderType = (provider: string) => {
  const typeMap: Record<string, string> = {
    aliyun: 'warning',
    tencent: 'info',
    aws: 'success',
    huawei: 'primary'
  }
  return typeMap[provider] || ''
}

// 掩码Access Key
const maskAccessKey = (key: string) => {
  if (!key || key.length <= 8) return key
  return key.substring(0, 4) + '****' + key.substring(key.length - 4)
}

// 加载账号列表
const loadAccountList = async () => {
  loading.value = true
  try {
    const res = await getCloudAccounts()
    // getCloudAccounts 返回的是数组，不是 { list: [] }
    accountList.value = Array.isArray(res) ? res : []
  } catch (error) {
    console.error('加载云账号列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载分组树
const loadGroupTree = async () => {
  try {
    const res = await getGroupTree()
    groupTreeOptions.value = res || []
  } catch (error) {
    console.error('加载分组树失败:', error)
  }
}

// 新增
const handleAdd = () => {
  Object.assign(form, {
    id: 0,
    name: '',
    provider: 'aliyun',
    accessKey: '',
    secretKey: '',
    region: '',
    description: '',
    status: 1
  })
  isEdit.value = false
  // 初始化区域列表
  currentRegions.value = getLocalRegions('aliyun')
  // 清除之前的验证
  nextTick(() => {
    formRef.value?.clearValidate()
  })
  dialogVisible.value = true
}

// 编辑
const handleEdit = async (row: any) => {
  isEdit.value = true // 先设置编辑状态

  Object.assign(form, {
    id: row.id,
    name: row.name,
    provider: row.provider,
    accessKey: '',
    secretKey: '',
    region: row.region || '',
    description: row.description || '',
    status: row.status
  })

  // 加载该账号的区域列表
  try {
    const res = await getCloudRegions(row.id)
    currentRegions.value = Array.isArray(res) ? res : []
  } catch (error) {
    console.error('加载区域列表失败:', error)
    currentRegions.value = getLocalRegions(row.provider)
  }

  // 清除之前的验证
  nextTick(() => {
    formRef.value?.clearValidate()
  })
  dialogVisible.value = true
}

// 删除
const handleDelete = (row: any) => {
  ElMessageBox.confirm(`确定要删除云账号"${row.name}"吗？`, '提示', {
    type: 'warning'
  }).then(async () => {
    try {
      await deleteCloudAccount(row.id)
      ElMessage.success('删除成功')
      loadAccountList()
    } catch (error: any) {
      ElMessage.error(error.message || '删除失败')
    }
  })
}

// 状态切换
const handleStatusChange = async (row: any) => {
  try {
    await updateCloudAccount(row.id, {
      id: row.id,
      name: row.name,
      provider: row.provider,
      region: row.region || '',
      description: row.description || '',
      status: row.status
    })
    ElMessage.success('状态更新成功')
  } catch (error: any) {
    // 恢复原状态
    row.status = row.status === 1 ? 0 : 1
    ElMessage.error(error.message || '状态更新失败')
  }
}

// 提交表单
const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await updateCloudAccount(form.id, form)
      ElMessage.success('更新成功')
    } else {
      await createCloudAccount(form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadAccountList()
  } catch (error: any) {
    ElMessage.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// 对话框关闭
const handleDialogClose = () => {
  formRef.value?.resetFields()
}

// 导入主机
const handleImportHost = (row: any) => {
  if (row.status === 0) {
    ElMessage.warning('该账号已禁用，无法导入主机')
    return
  }
  selectedAccount.value = row
  importForm.accountId = row.id
  importForm.region = row.region || ''
  importForm.groupId = null
  handleAccountChange()
  importDialogVisible.value = true
}

// 账号变化
const handleAccountChange = async () => {
  const account = accountList.value.find((a: any) => a.id === importForm.accountId)
  if (!account) return

  selectedAccount.value = account

  // 清空主机列表和区域
  cloudHosts.value = []
  selectedInstances.value = []
  selectAll.value = false
  regions.value = []
  importForm.region = ''

  // 从云API获取区域列表
  try {
    const res = await getCloudRegions(account.id)
    regions.value = Array.isArray(res) ? res : []

    // 设置默认区域（如果账号有默认区域且该区域在列表中）
    if (account.region) {
      const hasDefaultRegion = regions.value.some((r: any) => r.value === account.region)
      if (hasDefaultRegion) {
        importForm.region = account.region
      }
    }
  } catch (error: any) {
    console.error('加载区域列表失败:', error)
    ElMessage.error(error.message || '加载区域列表失败')
  }
}

// 加载云主机实例列表
const loadCloudInstances = async () => {
  if (!importForm.accountId || !importForm.region) return

  loadingInstances.value = true
  try {
    const res = await getCloudInstances(importForm.accountId, importForm.region)
    cloudHosts.value = Array.isArray(res) ? res : []
  } catch (error: any) {
    console.error('加载云主机列表失败:', error)
    ElMessage.error(error.message || '加载云主机列表失败')
    cloudHosts.value = []
  } finally {
    loadingInstances.value = false
  }
}

// 监听区域变化，自动加载实例列表
watch(() => importForm.region, () => {
  if (importForm.region) {
    loadCloudInstances()
  }
})

// 监听表单中的云厂商变化，更新区域列表（新增/编辑对话框用）
watch(() => form.provider, async (newProvider) => {
  if (!newProvider) return
  form.region = '' // 清空已选择的区域

  // 如果是编辑模式且有账号ID，调用云API获取区域
  if (isEdit.value && form.id > 0) {
    try {
      const res = await getCloudRegions(form.id)
      currentRegions.value = Array.isArray(res) ? res : []
    } catch (error) {
      console.error('加载区域列表失败:', error)
      currentRegions.value = getLocalRegions(newProvider)
    }
  } else {
    // 新增模式：使用本地常用区域列表
    currentRegions.value = getLocalRegions(newProvider)
  }
})

// 本地常用区域列表（新增账号时使用）
const getLocalRegions = (provider: string): any[] => {
  const localMap: Record<string, any[]> = {
    aliyun: [
      { value: 'cn-hangzhou', label: '华东1 (杭州)' },
      { value: 'cn-shanghai', label: '华东2 (上海)' },
      { value: 'cn-beijing', label: '华北2 (北京)' },
      { value: 'cn-shenzhen', label: '华南1 (深圳)' },
      { value: 'cn-guangzhou', label: '华南2 (广州)' },
      { value: 'cn-chengdu', label: '西南1 (成都)' }
    ],
    tencent: [
      { value: 'ap-guangzhou', label: '华南地区 (广州)' },
      { value: 'ap-shanghai', label: '华东地区 (上海)' },
      { value: 'ap-beijing', label: '华北地区 (北京)' },
      { value: 'ap-chengdu', label: '西南地区 (成都)' },
      { value: 'ap-chongqing', label: '西南地区 (重庆)' }
    ],
    aws: [
      { value: 'us-east-1', label: 'US East (N. Virginia)' },
      { value: 'us-west-2', label: 'US West (Oregon)' },
      { value: 'ap-southeast-1', label: 'Asia Pacific (Singapore)' }
    ],
    huawei: [
      { value: 'cn-south-1', label: '华南-广州' },
      { value: 'cn-east-3', label: '华东-上海' },
      { value: 'cn-north-1', label: '华北-北京' },
      { value: 'cn-southwest-2', label: '西南-贵阳' }
    ]
  }
  return localMap[provider] || []
}

// 全选
const handleSelectAll = (checked: boolean) => {
  if (cloudHostsTableRef.value) {
    cloudHosts.value.forEach((row: any) => {
      cloudHostsTableRef.value.toggleRowSelection(row, checked)
    })
  }
}

// 获取状态类型
const getStatusType = (status: string) => {
  const typeMap: Record<string, string> = {
    'Running': 'success',
    'Starting': 'warning',
    'Stopping': 'warning',
    'Stopped': 'info',
    'Deleted': 'danger'
  }
  return typeMap[status] || 'info'
}

// 获取状态文本
const getStatusText = (status: string) => {
  const textMap: Record<string, string> = {
    'Running': '运行中',
    'Starting': '启动中',
    'Stopping': '停止中',
    'Stopped': '已停止',
    'Deleted': '已删除'
  }
  return textMap[status] || status
}

// 选择变化
const handleSelectionChange = (selection: any[]) => {
  selectedInstances.value = selection.map((s: any) => s.instanceId)
}

// 确认导入
const handleConfirmImport = async () => {
  if (!importForm.groupId) {
    ElMessage.warning('请选择所属分组')
    return
  }

  importing.value = true
  try {
    await importFromCloud({
      accountId: importForm.accountId,
      region: importForm.region,
      groupId: importForm.groupId,
      instanceIds: selectedInstances.value
    })
    ElMessage.success('导入成功')
    importDialogVisible.value = false
  } catch (error: any) {
    ElMessage.error(error.message || '导入失败')
  } finally {
    importing.value = false
  }
}

// 导入对话框关闭
const handleImportDialogClose = () => {
  cloudHosts.value = []
  selectedInstances.value = []
  selectAll.value = false
}

onMounted(() => {
  loadAccountList()
  loadGroupTree()
})
</script>

<style scoped>
.cloud-accounts-page {
  padding: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 搜索栏 */
.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.search-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 280px;
}

.search-actions {
  display: flex;
  gap: 10px;
}

.reset-btn {
  background: #f5f7fa;
  border-color: #dcdfe6;
  color: #606266;
}

.reset-btn:hover {
  background: #e6e8eb;
  border-color: #c0c4cc;
}

.search-bar :deep(.el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;
  background-color: #fff;
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

.search-icon {
  color: #d4af37;
}

/* 表格容器 */
.table-wrapper {
  flex: 1;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 12px 12px;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.access-key {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #606266;
}

.text-muted {
  color: #c0c4cc;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
}

/* 对话框样式 */
:deep(.account-dialog) {
  border-radius: 12px;
}

:deep(.account-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.account-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.account-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

.dialog-content {
  padding: 0;
}

/* 云厂商选择器 - 内联样式 */
.provider-options-inline {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.provider-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border: 2px solid #e4e7ed;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  background: #fafbfc;
  min-width: 100px;
  justify-content: center;
}

.provider-option:hover {
  border-color: #d4af37;
  background: #fffaf0;
}

.provider-option.active {
  border-color: #d4af37;
  background: linear-gradient(135deg, #fffaf0 0%, #fef5e7 100%);
}

.provider-short {
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 700;
  color: #fff;
}

.provider-option:nth-child(1) .provider-short {
  background: linear-gradient(135deg, #ff6a00 0%, #ff9500 100%);
}

.provider-option:nth-child(2) .provider-short {
  background: linear-gradient(135deg, #00a4ff 0%, #00c6ff 100%);
}

.provider-option:nth-child(3) .provider-short {
  background: linear-gradient(135deg, #ff9900 0%, #ffb84d 100%);
}

.provider-option:nth-child(4) .provider-short {
  background: linear-gradient(135deg, #ce0e2d 0%, #e63946 100%);
}

.provider-name {
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.provider-option.active .provider-name {
  color: #d4af37;
  font-weight: 600;
}

/* 表单样式 */
.account-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

.account-form :deep(.el-form-item__label) {
  font-weight: 500;
  color: #606266;
  width: 120px !important;
  white-space: nowrap;
}

.account-form :deep(.el-form-item__content) {
  flex: 1;
}

.account-form :deep(.el-input),
.account-form :deep(.el-select),
.account-form :deep(.el-textarea) {
  width: 100%;
}

.account-form :deep(.el-input__wrapper),
.account-form :deep(.el-textarea__inner) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  transition: all 0.3s ease;
}

.account-form :deep(.el-input__wrapper:hover),
.account-form :deep(.el-textarea__inner:hover) {
  border-color: #d4af37;
}

.account-form :deep(.el-input__wrapper.is-focus),
.account-form :deep(.el-textarea__inner:focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.2);
}

/* 对话框底部 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.empty-instances {
  padding: 40px 0;
}

/* 导入对话框样式 */
:deep(.import-dialog) {
  border-radius: 12px;
}

:deep(.import-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.import-dialog .el-dialog__body) {
  padding: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

:deep(.import-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

.import-form {
  margin-bottom: 16px;
}

.form-row {
  display: flex;
  gap: 16px;
}

.form-item-full {
  flex: 1;
}

.form-item-half {
  width: 50%;
}

.instances-container {
  min-height: 200px;
}

.instances-list {
  background: #fafbfc;
  border-radius: 12px;
  padding: 16px;
}

.instances-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
}

.instances-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.instances-count {
  font-size: 15px;
  color: #303133;
}

.instances-count strong {
  color: #d4af37;
  font-size: 18px;
}

.instances-region {
  font-size: 12px;
  color: #909399;
}

.select-all-text {
  font-size: 14px;
  font-weight: 500;
}

.cloud-hosts-table {
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
}

.cloud-hosts-table :deep(.el-table__header-wrapper) {
  border-radius: 8px 8px 0 0;
}

.cloud-hosts-table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 8px 8px;
}

.cloud-hosts-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.cloud-hosts-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.instance-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.instance-icon {
  color: #d4af37;
  font-size: 16px;
}

.instance-id {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  color: #606266;
  background: #f5f7fa;
  padding: 2px 8px;
  border-radius: 4px;
}

.ip-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ip-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.ip-item span {
  font-family: 'Monaco', 'Menlo', monospace;
}

.public-ip span {
  color: #67c23a;
}

.private-ip span {
  color: #909399;
}
</style>
