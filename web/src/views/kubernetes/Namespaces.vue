<template>
  <div class="namespaces-container">
    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon stat-icon-blue">
          <el-icon><FolderOpened /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">命名空间总数</div>
          <div class="stat-value">{{ namespaceList.length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-green">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">正常状态</div>
          <div class="stat-value">{{ activeNamespaceCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-orange">
          <el-icon><PriceTag /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">标签总数</div>
          <div class="stat-value">{{ totalLabels }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-purple">
          <el-icon><Clock /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">平均运行时间</div>
          <div class="stat-value">{{ averageAge }}</div>
        </div>
      </div>
    </div>

    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><FolderOpened /></el-icon>
        </div>
        <div>
          <h2 class="page-title">命名空间</h2>
          <p class="page-subtitle">管理 Kubernetes 命名空间，实现资源隔离和分组</p>
        </div>
      </div>
      <div class="header-actions">
        <el-select
          v-model="selectedClusterId"
          placeholder="选择集群"
          class="cluster-select"
          @change="handleClusterChange"
        >
          <template #prefix>
            <el-icon class="search-icon"><Platform /></el-icon>
          </template>
          <el-option
            v-for="cluster in clusterList"
            :key="cluster.id"
            :label="cluster.alias || cluster.name"
            :value="cluster.id"
          />
        </el-select>
        <el-button class="black-button" @click="loadNamespaces">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button class="black-button" @click="handleCreateNamespace">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新建命名空间
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchName"
          placeholder="搜索命名空间名称..."
          clearable
          @clear="handleSearch"
          @keyup.enter="handleSearch"
          @input="handleSearch"
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchStatus"
          placeholder="状态"
          clearable
          @change="handleSearch"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><CircleCheck /></el-icon>
          </template>
          <el-option label="正常" value="Active" />
          <el-option label="终止中" value="Terminating" />
        </el-select>
      </div>
    </div>

    <!-- 命名空间列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedNamespaceList"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        :row-style="{ height: '56px' }"
        :cell-style="{ padding: '8px 0' }"
      >
      <el-table-column label="命名空间" min-width="220" fixed="left">
        <template #header>
          <span class="header-with-icon">
            <el-icon class="header-icon header-icon-blue"><FolderOpened /></el-icon>
            命名空间
          </span>
        </template>
        <template #default="{ row }">
          <div class="namespace-name-cell">
            <div class="namespace-icon-wrapper">
              <el-icon class="namespace-icon" :size="18"><FolderOpened /></el-icon>
            </div>
            <div class="namespace-name-content">
              <div class="namespace-name">{{ row.name }}</div>
              <div class="namespace-status">{{ row.status || 'Active' }}</div>
            </div>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="状态" width="120" align="center">
        <template #default="{ row }">
          <el-tag
            :type="getStatusType(row.status)"
            effect="dark"
            size="large"
            class="status-tag"
          >
            {{ getStatusText(row.status) }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="标签" width="120" align="center">
        <template #default="{ row }">
          <div class="label-cell" @click="showLabels(row)">
            <div class="label-badge-wrapper">
              <span class="label-count">{{ Object.keys(row.labels || {}).length }}</span>
              <el-icon class="label-icon"><PriceTag /></el-icon>
            </div>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="运行时间" width="160">
        <template #default="{ row }">
          <div class="age-cell">
            <el-icon class="age-icon"><Clock /></el-icon>
            <span>{{ row.age || '-' }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="操作" width="80" fixed="right" align="center">
        <template #default="{ row }">
          <el-dropdown trigger="click" @command="(command: string) => handleActionCommand(command, row)">
            <el-button link class="action-btn">
              <el-icon :size="18"><Edit /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu class="action-dropdown-menu">
                <el-dropdown-item command="edit">
                  <el-icon><Edit /></el-icon>
                  <span>修改</span>
                </el-dropdown-item>
                <el-dropdown-item command="yaml">
                  <el-icon><Document /></el-icon>
                  <span>YAML</span>
                </el-dropdown-item>
                <el-dropdown-item command="delete" divided class="danger-item">
                  <el-icon><Delete /></el-icon>
                  <span>删除</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredNamespaceList.length"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 新建命名空间弹窗 -->
    <el-dialog
      v-model="createDialogVisible"
      title="新建命名空间"
      width="600px"
      class="create-dialog"
      :lock-scroll="false"
    >
      <el-form :model="createForm" :rules="createRules" ref="createFormRef" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input
            v-model="createForm.name"
            placeholder="请输入命名空间名称"
            maxlength="63"
            show-word-limit
          >
            <template #append>
              <el-tooltip content="命名空间名称只能包含小写字母、数字和连字符，且必须以字母或数字开头和结尾" placement="top">
                <el-icon><InfoFilled /></el-icon>
              </el-tooltip>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleCreateSubmit" :loading="createLoading" class="black-button">
            创建
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 标签编辑弹窗 -->
    <el-dialog
      v-model="editLabelDialogVisible"
      :title="labelEditMode ? '编辑命名空间标签' : '命名空间标签'"
      width="850px"
      class="label-dialog"
      :close-on-click-modal="!labelEditMode"
      :lock-scroll="false"
    >
      <div class="label-dialog-content">
        <!-- 编辑模式 -->
        <div v-if="labelEditMode" class="label-edit-container">
          <div class="label-edit-header">
            <div class="label-edit-info">
              <el-icon class="info-icon"><PriceTag /></el-icon>
              <span>编辑 {{ selectedNamespace?.name }} 的标签</span>
            </div>
            <div class="label-edit-count">
              共 {{ editLabelList.length }} 个标签
            </div>
          </div>

          <div class="label-edit-list">
            <div v-for="(label, index) in editLabelList" :key="index" class="label-edit-row">
              <div class="label-row-number">{{ index + 1 }}</div>
              <div class="label-row-content">
                <div class="label-input-group">
                  <div class="label-input-wrapper">
                    <span class="label-input-label">Key</span>
                    <el-input
                      v-model="label.key"
                      placeholder="如: app"
                      size="default"
                      class="label-edit-input"
                    />
                  </div>
                  <span class="label-separator">=</span>
                  <div class="label-input-wrapper">
                    <span class="label-input-label">Value</span>
                    <el-input
                      v-model="label.value"
                      placeholder="可为空"
                      size="default"
                      class="label-edit-input"
                    />
                  </div>
                </div>
              </div>
              <el-button
                type="danger"
                :icon="Delete"
                size="default"
                @click="removeEditLabel(index)"
                class="remove-btn"
                circle
              />
            </div>
            <div v-if="editLabelList.length === 0" class="empty-labels">
              <el-icon class="empty-icon"><PriceTag /></el-icon>
              <p>暂无标签</p>
              <span>点击下方按钮添加新标签</span>
            </div>
          </div>

          <el-button
            type="primary"
            :icon="Plus"
            @click="addEditLabel"
            class="add-label-btn"
            plain
          >
            添加标签
          </el-button>
        </div>

        <!-- 查看模式 -->
        <el-table v-else :data="labelList" class="label-table" max-height="500">
          <el-table-column prop="key" label="Key" min-width="200">
            <template #default="{ row }">
              <div class="label-key-wrapper" @click="copyToClipboard(row.key, 'Key')">
                <span class="label-key-text">{{ row.key }}</span>
                <el-icon class="copy-icon"><CopyDocument /></el-icon>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="value" label="Value" min-width="200">
            <template #default="{ row }">
              <span class="label-value">{{ row.value || '-' }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <template v-if="labelEditMode">
            <el-button @click="cancelLabelEdit" size="large">取消</el-button>
            <el-button type="primary" @click="saveLabels" :loading="labelSaving" size="large" class="save-btn">
              <el-icon v-if="!labelSaving"><Check /></el-icon>
              {{ labelSaving ? '保存中...' : '保存更改' }}
            </el-button>
          </template>
          <template v-else>
            <el-button @click="editLabelDialogVisible = false" size="large">关闭</el-button>
            <el-button type="primary" @click="startLabelEdit" :icon="Edit" size="large" class="edit-btn">
              编辑标签
            </el-button>
          </template>
        </div>
      </template>
    </el-dialog>

    <!-- YAML 编辑弹窗 -->
    <el-dialog
      v-model="yamlDialogVisible"
      :title="`命名空间 YAML - ${selectedNamespace?.name || ''}`"
      width="900px"
      class="yaml-dialog"
      :lock-scroll="false"
    >
      <div class="yaml-dialog-content">
        <div class="yaml-editor-wrapper">
          <div class="yaml-line-numbers">
            <div v-for="line in yamlLineCount" :key="line" class="line-number">{{ line }}</div>
          </div>
          <textarea
            v-model="yamlContent"
            class="yaml-textarea"
            placeholder="YAML 内容"
            spellcheck="false"
            @input="handleYamlInput"
            @scroll="handleYamlScroll"
            ref="yamlTextarea"
            readonly
          ></textarea>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="yamlDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import axios from 'axios'
import {
  Search,
  PriceTag,
  FolderOpened,
  Platform,
  CircleCheck,
  Clock,
  Refresh,
  CopyDocument,
  Edit,
  Document,
  Delete,
  Plus,
  Check,
  InfoFilled
} from '@element-plus/icons-vue'
import { getClusterList, type Cluster, getNamespaces, type NamespaceInfo } from '@/api/kubernetes'

const loading = ref(false)
const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const namespaceList = ref<NamespaceInfo[]>([])

// 搜索条件
const searchName = ref('')
const searchStatus = ref('')

// 分页状态
const currentPage = ref(1)
const pageSize = ref(10)
const paginationStorageKey = ref('namespaces_pagination')

// 新建命名空间
const createDialogVisible = ref(false)
const createLoading = ref(false)
const createFormRef = ref<FormInstance>()
const createForm = ref({
  name: ''
})
const createRules: FormRules = {
  name: [
    { required: true, message: '请输入命名空间名称', trigger: 'blur' },
    {
      pattern: /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/,
      message: '命名空间名称只能包含小写字母、数字和连字符，且必须以字母或数字开头和结尾',
      trigger: 'blur'
    },
    {
      max: 63,
      message: '命名空间名称最多63个字符',
      trigger: 'blur'
    }
  ]
}

// 标签弹窗
const editLabelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])
const labelEditMode = ref(false)
const labelSaving = ref(false)
const editLabelList = ref<{ key: string; value: string }[]>([])
const labelOriginalYaml = ref('')

// YAML 编辑弹窗
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedNamespace = ref<NamespaceInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 过滤后的命名空间列表
const filteredNamespaceList = computed(() => {
  let result = namespaceList.value

  if (searchName.value) {
    result = result.filter(ns =>
      ns.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (searchStatus.value) {
    result = result.filter(ns => ns.status === searchStatus.value)
  }

  return result
})

// 分页后的命名空间列表
const paginatedNamespaceList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredNamespaceList.value.slice(start, end)
})

// 统计数据
const activeNamespaceCount = computed(() => {
  return namespaceList.value.filter(ns => !ns.status || ns.status === 'Active').length
})

const totalLabels = computed(() => {
  return namespaceList.value.reduce((sum, ns) => {
    return sum + Object.keys(ns.labels || {}).length
  }, 0)
})

const averageAge = computed(() => {
  if (namespaceList.value.length === 0) return '-'
  // 简单显示，实际可能需要更复杂的计算
  return namespaceList.value[0]?.age || '-'
})

// 获取状态类型
const getStatusType = (status: string | undefined) => {
  if (!status || status === 'Active') return 'success'
  if (status === 'Terminating') return 'warning'
  return 'info'
}

// 获取状态文本
const getStatusText = (status: string | undefined) => {
  if (!status || status === 'Active') return '正常'
  if (status === 'Terminating') return '终止中'
  return status
}

// 显示标签弹窗
const showLabels = (row: NamespaceInfo) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelEditMode.value = false
  selectedNamespace.value = row
  editLabelDialogVisible.value = true
}

// 复制到剪贴板
const copyToClipboard = async (text: string, type: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(`${type} 已复制到剪贴板`)
  } catch (error) {
    // 降级方案：使用传统方法
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    try {
      document.execCommand('copy')
      ElMessage.success(`${type} 已复制到剪贴板`)
    } catch (err) {
      ElMessage.error('复制失败')
    }
    document.body.removeChild(textarea)
  }
}

// 开始编辑标签
const startLabelEdit = async () => {
  if (!selectedNamespace.value) return

  try {
    const token = localStorage.getItem('token')
    const namespaceName = selectedNamespace.value.name

    // 获取命名空间当前 YAML
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/namespaces/${namespaceName}/yaml`,
      {
        params: { clusterId: selectedClusterId.value },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    labelOriginalYaml.value = response.data.data?.yaml || ''

    // 复制当前标签到编辑列表
    editLabelList.value = labelList.value.map(label => ({
      key: label.key,
      value: label.value
    }))

    labelEditMode.value = true
  } catch (error) {
    ElMessage.error('获取命名空间信息失败')
  }
}

// 取消编辑标签
const cancelLabelEdit = () => {
  labelEditMode.value = false
  editLabelList.value = []
}

// 添加编辑标签
const addEditLabel = () => {
  editLabelList.value.push({ key: '', value: '' })
}

// 删除编辑标签
const removeEditLabel = (index: number) => {
  editLabelList.value.splice(index, 1)
}

// 保存标签
const saveLabels = async () => {
  if (!selectedNamespace.value) return

  // 验证标签
  const validLabels = editLabelList.value.filter(label => label.key.trim() !== '')
  if (validLabels.some(label => !label.key)) {
    ElMessage.warning('标签键不能为空')
    return
  }

  // 检查是否有重复的键
  const keys = validLabels.map(l => l.key)
  const uniqueKeys = new Set(keys)
  if (keys.length !== uniqueKeys.size) {
    ElMessage.warning('存在重复的标签键，请检查')
    return
  }

  labelSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const namespaceName = selectedNamespace.value.name

    // 将 YAML 按行分割处理
    const lines = labelOriginalYaml.value.split('\n')
    let updatedLines: string[] = []
    let i = 0
    let inLabels = false
    let labelsIndent = ''

    while (i < lines.length) {
      const line = lines[i]

      // 检测 metadata: 开始
      if (/^metadata:\s*$/.test(line)) {
        updatedLines.push(line)
        i++

        // 处理 metadata 下的内容
        while (i < lines.length) {
          const metaLine = lines[i]

          // 检测 labels: 开始
          if (/^(\s+)labels:\s*$/.test(metaLine)) {
            inLabels = true
            labelsIndent = metaLine.match(/^(\s+)labels:\s*$/)?.[1] || ''
            updatedLines.push(metaLine)
            i++
            break
          }

          // 如果遇到其他字段（name, namespace等），保留它
          if (!/^\s/.test(metaLine) || /^(\s+)(name|namespace|uid|resourceVersion|generation|creationTimestamp|managedFields):\s*$/.test(metaLine)) {
            updatedLines.push(metaLine)
            i++

            // 如果遇到 labels，处理它
            if (/^(\s+)labels:\s*$/.test(metaLine)) {
              inLabels = true
              labelsIndent = metaLine.match(/^(\s+)labels:\s*$/)?.[1] || ''
              break
            }
          } else {
            // 其他字段（annotations等）
            updatedLines.push(metaLine)
            i++
          }
        }

        // 如果有标签，添加新内容
        if (validLabels.length > 0) {
          for (const label of validLabels) {
            if (label.value) {
              updatedLines.push(`${labelsIndent}  ${label.key}: ${label.value}`)
            } else {
              updatedLines.push(`${labelsIndent}  ${label.key}: ""`)
            }
          }
        }

        // 跳过原有的标签内容（如果有的话）
        while (i < lines.length && /^\s{2,}/.test(lines[i])) {
          // 如果遇到新的顶级节点，停止
          if (!/^(\s+)/.test(lines[i])) {
            break
          }
          const indent = lines[i].match(/^(\s+)/)?.[1]?.length || 0
          if (indent <= labelsIndent.length) {
            break
          }
          i++
        }

        continue
      }

      // 如果不在 labels 处理中，继续添加行
      if (!inLabels) {
        updatedLines.push(line)
      }

      i++
    }

    const updatedYaml = updatedLines.join('\n')


    // 调用 API 保存
    await axios.put(
      `/api/v1/plugins/kubernetes/resources/namespaces/${namespaceName}/yaml`,
      {
        clusterId: selectedClusterId.value,
        yaml: updatedYaml
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('标签保存成功')
    labelEditMode.value = false
    // 刷新命名空间列表
    await loadNamespaces()
    // 更新当前显示的标签列表
    const updatedNamespace = namespaceList.value.find(n => n.name === namespaceName)
    if (updatedNamespace) {
      selectedNamespace.value = updatedNamespace
      labelList.value = validLabels
    }
  } catch (error: any) {
    ElMessage.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    labelSaving.value = false
  }
}

// 保存分页状态到 localStorage
const savePaginationState = () => {
  try {
    localStorage.setItem(paginationStorageKey.value, JSON.stringify({
      currentPage: currentPage.value,
      pageSize: pageSize.value
    }))
  } catch (error) {
  }
}

// 从 localStorage 恢复分页状态
const restorePaginationState = () => {
  try {
    const saved = localStorage.getItem(paginationStorageKey.value)
    if (saved) {
      const state = JSON.parse(saved)
      currentPage.value = state.currentPage || 1
      pageSize.value = state.pageSize || 10
    }
  } catch (error) {
    currentPage.value = 1
    pageSize.value = 10
  }
}

// 处理页码变化
const handlePageChange = (page: number) => {
  currentPage.value = page
  savePaginationState()
}

// 处理每页数量变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  const maxPage = Math.ceil(filteredNamespaceList.value.length / size)
  if (currentPage.value > maxPage) {
    currentPage.value = maxPage || 1
  }
  savePaginationState()
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('namespaces_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      await loadNamespaces()
    }
  } catch (error) {
    ElMessage.error('获取集群列表失败')
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('namespaces_selected_cluster_id', selectedClusterId.value.toString())
  }
  currentPage.value = 1
  await loadNamespaces()
}

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!selectedClusterId.value) return

  loading.value = true
  try {
    const data = await getNamespaces(selectedClusterId.value)
    namespaceList.value = data || []
    restorePaginationState()
  } catch (error) {
    namespaceList.value = []
    ElMessage.error('获取命名空间列表失败')
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
  savePaginationState()
}

// 新建命名空间
const handleCreateNamespace = () => {
  createForm.value.name = ''
  createDialogVisible.value = true
}

// 提交新建
const handleCreateSubmit = async () => {
  if (!createFormRef.value) return

  await createFormRef.value.validate(async (valid) => {
    if (valid) {
      createLoading.value = true
      try {
        const token = localStorage.getItem('token')

        // 构建命名空间的 YAML
        const yaml = `apiVersion: v1
kind: Namespace
metadata:
  name: ${createForm.value.name}
`

        await axios.post(
          `/api/v1/plugins/kubernetes/resources/namespaces`,
          { yaml },
          {
            params: { clusterId: selectedClusterId.value },
            headers: { Authorization: `Bearer ${token}` }
          }
        )

        ElMessage.success('命名空间创建成功')
        createDialogVisible.value = false
        await loadNamespaces()
      } catch (error: any) {
        ElMessage.error(`创建失败: ${error.response?.data?.message || error.message}`)
      } finally {
        createLoading.value = false
      }
    }
  })
}

// 处理下拉菜单命令
const handleActionCommand = (command: string, row: NamespaceInfo) => {
  selectedNamespace.value = row

  switch (command) {
    case 'edit':
      handleEditLabels()
      break
    case 'yaml':
      handleShowYAML()
      break
    case 'delete':
      handleDelete()
      break
  }
}

// 编辑标签
const handleEditLabels = () => {
  if (!selectedNamespace.value) return
  const labels = selectedNamespace.value.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelEditMode.value = false
  editLabelDialogVisible.value = true
}

// 显示 YAML 编辑器
const handleShowYAML = async () => {
  if (!selectedNamespace.value) return

  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const namespaceName = selectedNamespace.value.name

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/namespaces/${namespaceName}/yaml`,
      {
        params: { clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    yamlContent.value = response.data.data?.yaml || ''
    yamlDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  }
}

// YAML编辑器输入处理
const handleYamlInput = () => {
  // 只读模式，不需要处理输入
}

// YAML编辑器滚动处理（同步行号滚动）
const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

// 删除命名空间
const handleDelete = async () => {
  if (!selectedNamespace.value) return

  try {
    await ElMessageBox.confirm(
      `确定要删除命名空间 ${selectedNamespace.value.name} 吗？此操作将删除该命名空间下的所有资源，且不可恢复！`,
      '删除命名空间确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/namespaces/${selectedNamespace.value.name}?clusterId=${selectedClusterId.value}`,
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('命名空间删除成功')
    await loadNamespaces()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.namespaces-container {
  padding: 0;
  background-color: transparent;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 12px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  transition: all 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
}

.stat-icon {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  flex-shrink: 0;
}

.stat-icon-blue {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-green {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-orange {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-purple {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #d4af37;
  line-height: 1;
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

.cluster-select {
  width: 280px;
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

/* 搜索栏 */
.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
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

.filter-select {
  width: 150px;
}

.search-icon {
  color: #d4af37;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

/* 搜索框样式优化 */
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

.search-bar :deep(.el-select .el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

.cluster-select :deep(.el-input__wrapper) {
  border-radius: 8px;
}

/* 表头图标 */
.header-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.header-icon {
  font-size: 16px;
}

.header-icon-blue {
  color: #d4af37;
}

/* 现代表格 */
.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 12px 12px;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
  height: 56px !important;
}

.modern-table :deep(.el-table__row td) {
  height: 56px !important;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.modern-table :deep(.el-tag) {
  border-radius: 6px;
  padding: 4px 10px;
  font-weight: 500;
}

/* 命名空间名称单元格 */
.namespace-name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.namespace-icon-wrapper {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #d4af37;
  flex-shrink: 0;
}

.namespace-icon {
  color: #d4af37;
}

.namespace-name-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.namespace-name {
  font-weight: 600;
  color: #303133;
}

.namespace-name:hover {
  color: #409eff;
}

.namespace-status {
  font-size: 12px;
  color: #909399;
}

/* 时间单元格 */
.age-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #606266;
}

.age-icon {
  color: #d4af37;
}

/* 标签单元格 */
.label-cell {
  display: flex;
  justify-content: center;
  align-items: center;
  cursor: pointer;
  padding: 5px 0;
}

.label-badge-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.label-icon {
  color: #d4af37;
  font-size: 20px;
  transition: all 0.3s;
}

.label-count {
  position: absolute;
  top: -6px;
  right: -6px;
  background-color: #d4af37;
  color: #000;
  font-size: 10px;
  font-weight: 600;
  min-width: 16px;
  height: 16px;
  line-height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  text-align: center;
  border: 1px solid #d4af37;
  z-index: 1;
}

.label-cell:hover .label-icon {
  color: #bfa13f;
  transform: scale(1.1);
}

.label-cell:hover .label-count {
  background-color: #bfa13f;
  border-color: #bfa13f;
}

/* 操作按钮 */
.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #d4af37;
}

.action-btn:hover {
  color: #bfa13f;
}

/* 下拉菜单样式 */
.action-dropdown-menu {
  min-width: 140px;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  font-size: 13px;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item .el-icon) {
  color: #d4af37;
  font-size: 16px;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item) {
  color: #f56c6c;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item .el-icon) {
  color: #f56c6c;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item:hover) {
  background-color: #f5f5f5;
  color: #d4af37;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item:hover .el-icon) {
  color: #d4af37;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item:hover) {
  background-color: #fef0f0;
  color: #f56c6c;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item:hover .el-icon) {
  color: #f56c6c;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

/* 状态标签 */
.status-tag {
  border-radius: 8px;
  padding: 6px 14px;
  font-weight: 500;
}

/* 标签编辑模式 */
.label-edit-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.label-edit-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  border-radius: 8px;
  border: 1px solid #d4af37;
}

.label-edit-info {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 15px;
  color: #d4af37;
  font-weight: 500;
}

.label-edit-info .info-icon {
  font-size: 18px;
}

.label-edit-count {
  font-size: 14px;
  color: #d4af37;
  padding: 6px 14px;
  background: rgba(212, 175, 55, 0.15);
  border-radius: 20px;
  font-weight: 500;
}

.label-edit-list {
  max-height: 420px;
  overflow-y: auto;
  padding: 12px;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  border-radius: 8px;
  border: 1px solid #e0e0e0;
}

.label-edit-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  padding: 16px;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #e0e0e0;
  transition: all 0.3s;
}

.label-edit-row:hover {
  border-color: #d4af37;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.15);
  transform: translateY(-2px);
}

.label-row-number {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border-radius: 6px;
  font-weight: 600;
  font-size: 14px;
}

.label-row-content {
  flex: 1;
  min-width: 0;
}

.label-input-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.label-input-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.label-input-label {
  font-size: 12px;
  color: #909399;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.label-edit-input {
  width: 100%;
}

.label-edit-input :deep(.el-input__wrapper) {
  background: #fafbfc;
  border: 1px solid #d0d0d0;
  border-radius: 6px;
  transition: all 0.3s;
}

.label-edit-input :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  background: #fff;
}

.label-edit-input :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  background: #fff;
  box-shadow: 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.label-edit-input :deep(.el-input__inner) {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
}

.label-separator {
  color: #909399;
  font-weight: 600;
  font-size: 18px;
  flex-shrink: 0;
}

.remove-btn {
  flex-shrink: 0;
}

.remove-btn:hover {
  transform: scale(1.1);
}

.empty-labels {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.empty-labels .empty-icon {
  font-size: 48px;
  color: #d4af37;
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-labels p {
  font-size: 16px;
  color: #606266;
  margin: 0 0 8px 0;
}

.empty-labels span {
  font-size: 14px;
  color: #909399;
}

.add-label-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  border: 2px dashed #d4af37;
  border-radius: 8px;
  transition: all 0.3s;
}

.add-label-btn:hover {
  border-style: solid;
  border-color: #bfa13f;
  background: rgba(212, 175, 55, 0.05);
  transform: translateY(-2px);
}

/* 标签弹窗 */
.label-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border-radius: 12px 12px 0 0;
  padding: 20px 28px;
  border-bottom: 2px solid #d4af37;
}

.label-dialog :deep(.el-dialog__title) {
  color: #d4af37;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.label-dialog :deep(.el-dialog__body) {
  padding: 24px 28px;
}

.label-dialog :deep(.el-dialog__footer) {
  padding: 16px 28px;
  background: #fafbfc;
  border-top: 1px solid #e0e0e0;
}

.label-table {
  width: 100%;
}

.label-table :deep(.el-table__cell) {
  padding: 8px 0;
}

.label-key-wrapper {
  display: inline-flex !important;
  align-items: center !important;
  gap: 6px !important;
  padding: 5px 12px !important;
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%) !important;
  color: #d4af37 !important;
  border: 1px solid #d4af37 !important;
  border-radius: 6px !important;
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 12px !important;
  font-weight: 500 !important;
  cursor: pointer !important;
  transition: all 0.3s !important;
  user-select: none;
}

.label-key-wrapper:hover {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%) !important;
  border-color: #bfa13f !important;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3) !important;
  transform: translateY(-1px);
}

.label-key-wrapper:active {
  transform: translateY(0);
}

.label-key-text {
  flex: 1;
  word-break: break-all;
  line-height: 1.4;
  white-space: pre-wrap;
}

.copy-icon {
  font-size: 14px;
  flex-shrink: 0;
  opacity: 0.7;
  transition: opacity 0.3s;
}

.label-key-wrapper:hover .copy-icon {
  opacity: 1;
}

.label-value {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #606266;
  word-break: break-all;
  white-space: pre-wrap;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.dialog-footer .edit-btn {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  border-color: #d4af37;
  color: #d4af37;
}

.dialog-footer .edit-btn:hover {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  border-color: #bfa13f;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.3);
}

.dialog-footer .save-btn {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  border-color: #d4af37;
  color: #d4af37;
  min-width: 120px;
}

.dialog-footer .save-btn:hover {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  border-color: #bfa13f;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.3);
}

/* YAML 编辑弹窗 */
.yaml-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.yaml-dialog :deep(.el-dialog__title) {
  color: #d4af37;
  font-size: 16px;
  font-weight: 600;
}

.yaml-dialog :deep(.el-dialog__body) {
  padding: 24px;
  background-color: #1a1a1a;
}

.yaml-dialog-content {
  padding: 0;
}

.yaml-editor-wrapper {
  display: flex;
  border: 1px solid #d4af37;
  border-radius: 6px;
  overflow: hidden;
  background-color: #000000;
}

.yaml-line-numbers {
  background-color: #0d0d0d;
  color: #666;
  padding: 16px 8px;
  text-align: right;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  user-select: none;
  overflow: hidden;
  min-width: 40px;
  border-right: 1px solid #333;
}

.line-number {
  height: 20.8px;
  line-height: 1.6;
}

.yaml-textarea {
  flex: 1;
  background-color: #000000;
  color: #d4af37;
  border: none;
  outline: none;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
  min-height: 400px;
}

.yaml-textarea::placeholder {
  color: #555;
}

.yaml-textarea:focus {
  outline: none;
}

/* 响应式设计 */
@media (max-width: 1400px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }

  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .header-actions {
    width: 100%;
    flex-direction: column;
  }

  .cluster-select {
    width: 100%;
  }
}

/* 修复弹窗打开时页面抖动问题 */
.namespaces-container :deep(.el-overlay) {
  overflow: hidden;
}

/* 命名空间标签最终覆盖：和节点标签保持一致，去掉黑金 chip */
.label-icon,
.action-btn,
.action-dropdown-menu :deep(.el-dropdown-menu__item .el-icon),
.action-dropdown-menu :deep(.el-dropdown-menu__item:hover),
.action-dropdown-menu :deep(.el-dropdown-menu__item:hover .el-icon) {
  color: #2563eb;
}

.label-count {
  background: #eff6ff;
  color: #2563eb;
  border-color: #bfdbfe;
}

.label-cell:hover .label-icon {
  color: #1d4ed8;
}

.label-cell:hover .label-count {
  background: #dbeafe;
  border-color: #93c5fd;
}

.label-dialog :deep(.el-dialog) {
  border-radius: 18px;
  overflow: hidden;
}

.label-dialog :deep(.el-dialog__header) {
  background: #ffffff;
  color: #111827;
  border-bottom: 1px solid #eef2f7;
}

.label-dialog :deep(.el-dialog__title) {
  color: #111827;
  font-size: 20px;
  font-weight: 800;
}

.label-dialog :deep(.el-dialog__body) {
  background: #fbfcfe;
}

.label-edit-header {
  background: #eff6ff;
  border-color: #dbeafe;
}

.label-edit-info,
.label-edit-count {
  color: #2563eb;
}

.label-edit-count {
  background: rgba(37, 99, 235, 0.1);
}

.label-row-number {
  background: #eff6ff;
  color: #2563eb;
}

.label-edit-row:hover {
  border-color: #bfdbfe;
  box-shadow: 0 8px 20px rgba(37, 99, 235, 0.12);
}

.label-edit-input :deep(.el-input__wrapper:hover),
.label-edit-input :deep(.el-input__wrapper.is-focus) {
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
}

.empty-labels .empty-icon {
  color: #2563eb;
}

.add-label-btn {
  background: #f8fbff !important;
  border: 2px dashed #bfdbfe !important;
  color: #2563eb !important;
  box-shadow: none !important;
}

.add-label-btn:hover {
  background: #eff6ff !important;
  border-color: #93c5fd !important;
  color: #1d4ed8 !important;
  box-shadow: 0 8px 18px rgba(37, 99, 235, 0.1) !important;
  transform: translateY(-1px);
}

.add-label-btn:focus,
.add-label-btn:active {
  background: #eff6ff !important;
  border-color: #93c5fd !important;
  color: #1d4ed8 !important;
}

.label-key-wrapper {
  display: inline-flex !important;
  align-items: center !important;
  gap: 6px !important;
  padding: 5px 12px !important;
  background: #eff6ff !important;
  color: #2563eb !important;
  border: 1px solid #bfdbfe !important;
  border-radius: 999px !important;
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 12px !important;
  font-weight: 700 !important;
  cursor: pointer !important;
  transition: all 0.3s !important;
  user-select: none;
}

.label-key-wrapper:hover {
  background: #dbeafe !important;
  border-color: #93c5fd !important;
  box-shadow: 0 6px 16px rgba(37, 99, 235, 0.14) !important;
  transform: translateY(-1px);
}

.dialog-footer .edit-btn,
.dialog-footer .save-btn {
  background: #111827 !important;
  border-color: #111827 !important;
  color: #ffffff !important;
  box-shadow: 0 8px 18px rgba(17, 24, 39, 0.16);
}

.dialog-footer .edit-btn:hover,
.dialog-footer .save-btn:hover {
  background: #1f2937 !important;
  border-color: #1f2937 !important;
  color: #ffffff !important;
  box-shadow: 0 10px 22px rgba(17, 24, 39, 0.2);
}
</style>
