<template>
  <div class="nginx-config-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Setting /></el-icon>
        </div>
        <div>
          <h2 class="page-title">功能配置</h2>
          <p class="page-subtitle">管理 Nginx 数据源配置</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增数据源
        </el-button>
        <el-button @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 筛选条件 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-select v-model="filterForm.type" placeholder="数据源类型" clearable class="search-input">
          <el-option label="主机Nginx" value="host" />
          <el-option label="K8s Ingress" value="k8s_ingress" />
        </el-select>
        <el-select v-model="filterForm.status" placeholder="状态" clearable class="search-input-xs">
          <el-option label="启用" :value="1" />
          <el-option label="禁用" :value="0" />
        </el-select>
      </div>
      <div class="search-actions">
        <el-button class="black-button" @click="loadData">查询</el-button>
        <el-button class="reset-btn" @click="handleReset">重置</el-button>
      </div>
    </div>

    <!-- 表格 -->
    <div class="table-wrapper">
      <el-table :data="tableData" v-loading="loading" class="modern-table">
        <el-table-column label="名称" prop="name" min-width="150" />
        <el-table-column label="类型" prop="type" width="120" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.type === 'host'" type="primary" effect="dark">主机Nginx</el-tag>
            <el-tag v-else type="success" effect="dark">K8s Ingress</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="描述" prop="description" min-width="200" show-overflow-tooltip />
        <el-table-column label="状态" prop="status" width="100" align="center">
          <template #default="{ row }">
            <el-switch
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              @change="handleStatusChange(row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="采集间隔" prop="collectInterval" width="100" align="center">
          <template #default="{ row }">
            {{ row.collectInterval }}秒
          </template>
        </el-table-column>
        <el-table-column label="数据保留" prop="retentionDays" width="100" align="center">
          <template #default="{ row }">
            {{ row.retentionDays }}天
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" width="180">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm
              title="确定要删除该数据源吗？"
              @confirm="handleDelete(row)"
            >
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadData"
          @current-change="loadData"
        />
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑数据源' : '新增数据源'"
      width="600px"
      destroy-on-close
      class="config-dialog"
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="100px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入数据源名称" />
        </el-form-item>

        <el-form-item label="类型" prop="type">
          <el-select v-model="formData.type" placeholder="请选择数据源类型" style="width: 100%">
            <el-option label="主机Nginx" value="host" />
            <el-option label="K8s Ingress" value="k8s_ingress" />
          </el-select>
        </el-form-item>

        <el-form-item label="描述" prop="description">
          <el-input
            v-model="formData.description"
            type="textarea"
            :rows="2"
            placeholder="请输入描述"
          />
        </el-form-item>

        <!-- 主机类型配置 -->
        <template v-if="formData.type === 'host'">
          <el-form-item label="关联主机" prop="hostId">
            <el-select v-model="formData.hostId" placeholder="请选择主机" style="width: 100%" filterable>
              <el-option
                v-for="host in hosts"
                :key="host.ID || host.id"
                :label="`${host.name} (${host.ip})`"
                :value="host.ID || host.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="日志路径" prop="logPath">
            <el-input v-model="formData.logPath" placeholder="/var/log/nginx/access.log" />
          </el-form-item>

          <el-form-item label="日志格式" prop="logFormat">
            <el-select v-model="formData.logFormat" placeholder="请选择日志格式" style="width: 100%">
              <el-option label="combined" value="combined" />
              <el-option label="main" value="main" />
              <el-option label="json" value="json" />
            </el-select>
          </el-form-item>
        </template>

        <!-- K8s Ingress 类型配置 -->
        <template v-if="formData.type === 'k8s_ingress'">
          <el-form-item label="关联集群" prop="clusterId">
            <el-select v-model="formData.clusterId" placeholder="请选择集群" style="width: 100%" filterable>
              <el-option
                v-for="cluster in clusters"
                :key="cluster.ID || cluster.id"
                :label="cluster.name"
                :value="cluster.ID || cluster.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="命名空间" prop="namespace">
            <el-input v-model="formData.namespace" placeholder="ingress-nginx" />
          </el-form-item>

          <el-form-item label="Ingress名称" prop="ingressName">
            <el-input v-model="formData.ingressName" placeholder="nginx-ingress-controller (可选)" />
          </el-form-item>

          <el-form-item label="Pod选择器">
            <el-input
              v-model="formData.k8sPodSelector"
              placeholder="app.kubernetes.io/name=ingress-nginx,app.kubernetes.io/component=controller"
            />
            <div class="form-tip">用于选择 Ingress Controller Pod 的标签选择器</div>
          </el-form-item>

          <el-form-item label="容器名称">
            <el-input v-model="formData.k8sContainerName" placeholder="controller" />
            <div class="form-tip">Ingress Controller 容器的名称，默认为 controller</div>
          </el-form-item>

          <el-form-item label="日志格式" prop="logFormat">
            <el-select v-model="formData.logFormat" placeholder="请选择日志格式" style="width: 100%">
              <el-option label="combined" value="combined" />
              <el-option label="json" value="json" />
            </el-select>
          </el-form-item>
        </template>

        <el-divider content-position="left">通用配置</el-divider>

        <el-form-item label="采集间隔" prop="collectInterval">
          <el-input-number
            v-model="formData.collectInterval"
            :min="10"
            :max="3600"
            :step="10"
          />
          <span style="margin-left: 8px; color: #909399">秒</span>
        </el-form-item>

        <el-form-item label="数据保留" prop="retentionDays">
          <el-input-number
            v-model="formData.retentionDays"
            :min="1"
            :max="365"
          />
          <span style="margin-left: 8px; color: #909399">天</span>
        </el-form-item>

        <el-form-item label="状态" prop="status">
          <el-switch
            v-model="formData.status"
            :active-value="1"
            :inactive-value="0"
            active-text="启用"
            inactive-text="禁用"
          />
        </el-form-item>

        <el-divider content-position="left">高级选项</el-divider>

        <el-form-item label="地理位置解析">
          <el-switch
            v-model="formData.geoEnabled"
            active-text="启用"
            inactive-text="禁用"
          />
          <div class="form-tip">启用后将解析访问者 IP 的地理位置信息</div>
        </el-form-item>

        <el-form-item label="会话跟踪">
          <el-switch
            v-model="formData.sessionEnabled"
            active-text="启用"
            inactive-text="禁用"
          />
          <div class="form-tip">启用后将跟踪用户会话，用于更精确的 UV 统计</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button class="black-button" :loading="submitLoading" @click="handleSubmit">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Setting, Plus, Refresh } from '@element-plus/icons-vue'
import {
  getNginxSources,
  createNginxSource,
  updateNginxSource,
  deleteNginxSource,
  type NginxSource,
} from '@/api/nginx'
import { getHostList } from '@/api/host'
import { getClusterList } from '@/api/kubernetes'

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const tableData = ref<NginxSource[]>([])
const hosts = ref<any[]>([])
const clusters = ref<any[]>([])
const formRef = ref<FormInstance>()

const filterForm = ref({
  type: '',
  status: undefined as number | undefined,
})

const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0,
})

const defaultFormData: NginxSource = {
  name: '',
  type: 'host',
  description: '',
  status: 1,
  hostId: undefined,
  logPath: '/var/log/nginx/access.log',
  logFormat: 'combined',
  clusterId: undefined,
  namespace: 'ingress-nginx',
  ingressName: '',
  k8sPodSelector: '',
  k8sContainerName: 'controller',
  geoEnabled: true,
  sessionEnabled: false,
  collectInterval: 60,
  retentionDays: 30,
}

const formData = ref<NginxSource>({ ...defaultFormData })

const formRules: FormRules = {
  name: [
    { required: true, message: '请输入数据源名称', trigger: 'blur' },
  ],
  type: [
    { required: true, message: '请选择数据源类型', trigger: 'change' },
  ],
}

// 格式化时间
const formatTime = (timeStr: string) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
    }
    if (filterForm.value.type) {
      params.type = filterForm.value.type
    }
    if (filterForm.value.status !== undefined) {
      params.status = filterForm.value.status
    }

    const res = await getNginxSources(params)
    // request.ts 拦截器已解包响应，直接返回 data
    tableData.value = res.list || res || []
    pagination.value.total = res.total || 0
  } catch (error) {
    console.error('获取数据源列表失败:', error)
    ElMessage.error('获取数据源列表失败')
  } finally {
    loading.value = false
  }
}

// 重置筛选
const handleReset = () => {
  filterForm.value = {
    type: '',
    status: undefined,
  }
  pagination.value.page = 1
  loadData()
}

// 新增
const handleAdd = () => {
  isEdit.value = false
  formData.value = { ...defaultFormData }
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: NginxSource) => {
  isEdit.value = true
  formData.value = { ...row }
  dialogVisible.value = true
}

// 状态变更
const handleStatusChange = async (row: NginxSource) => {
  try {
    await updateNginxSource(row.id!, row)
    ElMessage.success('状态更新成功')
  } catch (error) {
    console.error('状态更新失败:', error)
    ElMessage.error('状态更新失败')
    row.status = row.status === 1 ? 0 : 1
  }
}

// 删除
const handleDelete = async (row: NginxSource) => {
  try {
    await deleteNginxSource(row.id!)
    ElMessage.success('删除成功')
    loadData()
  } catch (error) {
    console.error('删除失败:', error)
    ElMessage.error('删除失败')
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    submitLoading.value = true

    if (isEdit.value) {
      await updateNginxSource(formData.value.id!, formData.value)
      ElMessage.success('更新成功')
    } else {
      await createNginxSource(formData.value)
      ElMessage.success('创建成功')
    }

    dialogVisible.value = false
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('提交失败:', error)
      ElMessage.error(error.response?.data?.message || '提交失败')
    }
  } finally {
    submitLoading.value = false
  }
}

onMounted(() => {
  loadData()
  loadHosts()
  loadClusters()
})

// 加载主机列表
const loadHosts = async () => {
  try {
    const res = await getHostList({ page: 1, pageSize: 1000 })
    // request.ts 拦截器已解包，res 直接是 data 内容
    hosts.value = res.list || res || []
  } catch (error) {
    console.error('获取主机列表失败:', error)
  }
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const res = await getClusterList()
    // request.ts 拦截器已解包，res 直接是 data 内容
    clusters.value = res.list || res || []
  } catch (error) {
    console.error('获取集群列表失败:', error)
  }
}
</script>

<style scoped>
.nginx-config-container {
  padding: 0;
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
  width: 150px;
}

.search-input-xs {
  width: 120px;
}

.search-actions {
  display: flex;
  gap: 10px;
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
  border-color: #333333 !important;
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
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

/* 表格 */
.table-wrapper {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.pagination-wrapper {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

/* 对话框样式 */
.config-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid #f0f0f0;
  padding: 16px 20px;
  margin-right: 0;
}

.config-dialog :deep(.el-dialog__title) {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.config-dialog :deep(.el-dialog__body) {
  padding: 20px;
}

.config-dialog :deep(.el-dialog__footer) {
  border-top: 1px solid #f0f0f0;
  padding: 12px 20px;
}

.config-dialog :deep(.el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

.config-dialog :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.config-dialog :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

.config-dialog :deep(.el-textarea__inner:hover) {
  border-color: #d4af37;
}

.config-dialog :deep(.el-textarea__inner:focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

.config-dialog :deep(.el-select .el-input__wrapper:hover) {
  border-color: #d4af37;
}

.config-dialog :deep(.el-select .el-input.is-focus .el-input__wrapper) {
  border-color: #d4af37 !important;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25) !important;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.5;
}

.config-dialog :deep(.el-divider__text) {
  font-size: 13px;
  font-weight: 500;
  color: #606266;
}
</style>
