<template>
  <div class="template-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Document /></el-icon>
        </div>
        <div>
          <h2 class="page-title">模板管理</h2>
          <p class="page-subtitle">创建和管理执行模板，支持模板复用和参数化</p>
        </div>
      </div>
    </div>

    <!-- 搜索区域 -->
    <div class="search-card">
      <div class="search-row">
        <div class="search-item">
          <span class="search-label">模板类型:</span>
          <el-select v-model="searchForm.type" placeholder="请选择" clearable style="width: 200px;">
            <el-option label="系统信息" value="system" />
            <el-option label="部署" value="deploy" />
            <el-option label="监控" value="monitor" />
            <el-option label="备份" value="backup" />
          </el-select>
        </div>
        <div class="search-item">
          <span class="search-label">模板名称:</span>
          <el-input v-model="searchForm.name" placeholder="请输入" clearable style="width: 300px;" />
        </div>
      </div>
    </div>

    <!-- 模板列表 -->
    <div class="template-list-card">
      <div class="list-header">
        <span class="header-title">模板列表</span>
        <div class="header-actions">
          <el-button type="primary" @click="handleCreate" class="black-button">
            <el-icon style="margin-right: 6px;"><Plus /></el-icon>
            新建
          </el-button>
          <el-button :icon="Refresh" @click="loadTemplates" />
          <el-button :icon="Setting" />
          <el-button :icon="FullScreen" />
        </div>
      </div>
      <el-table :data="templates" v-loading="loading">
        <el-table-column type="index" label="序号" width="80" align="center" />
        <el-table-column label="模板名称" prop="name" min-width="200" />
        <el-table-column label="模板类型" prop="type" width="120" align="center">
          <template #default="{ row }">
            <el-tag size="small">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="模板内容" prop="content" min-width="300" show-overflow-tooltip />
        <el-table-column label="描述信息" prop="description" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="150" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" link @click="handleEdit(row)">编辑</el-button>
            <el-button type="danger" size="small" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadTemplates"
          @current-change="loadTemplates"
        />
      </div>
    </div>

    <!-- 新建/编辑模板对话框 -->
    <el-dialog
      v-model="showTemplateDialog"
      :title="isEdit ? '编辑模板' : '新建模板'"
      width="900px"
      destroy-on-close
    >
      <el-form :model="templateForm" label-width="100px">
        <el-form-item label="模板类型" required>
          <el-select v-model="templateForm.type" placeholder="请选择模板类型" style="width: 100%;">
            <el-option label="系统信息" value="system" />
            <el-option label="部署" value="deploy" />
            <el-option label="监控" value="monitor" />
            <el-option label="备份" value="backup" />
          </el-select>
          <el-link type="primary" style="margin-left: 8px;">添加类型</el-link>
        </el-form-item>

        <el-form-item label="模板名称" required>
          <el-input v-model="templateForm.name" placeholder="请输入模板名称" />
        </el-form-item>

        <el-form-item label="脚本语言" required>
          <el-radio-group v-model="templateForm.scriptType">
            <el-radio-button label="Shell">Shell</el-radio-button>
            <el-radio-button label="Python">Python</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="模板内容" required>
          <el-input
            v-model="templateForm.content"
            type="textarea"
            :rows="10"
            placeholder="请输入脚本内容..."
            style="font-family: 'Courier New', monospace;"
          />
        </el-form-item>

        <el-form-item label="参数化">
          <el-link type="primary" @click="showParamDialog = true">添加参数</el-link>
          <div v-if="templateForm.parameters.length > 0" class="parameters-list">
            <el-tag
              v-for="(param, index) in templateForm.parameters"
              :key="index"
              closable
              @close="removeParameter(index)"
              style="margin: 8px 8px 0 0;"
            >
              {{ param.name }} ({{ param.varName }})
            </el-tag>
          </div>
        </el-form-item>

        <el-form-item label="备注信息">
          <el-input
            v-model="templateForm.remark"
            type="textarea"
            :rows="3"
            placeholder="请输入模板备注信息"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTemplateDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSaveTemplate">确定</el-button>
      </template>
    </el-dialog>

    <!-- 编辑参数对话框 -->
    <el-dialog
      v-model="showParamDialog"
      title="编辑参数"
      width="600px"
      destroy-on-close
    >
      <el-form :model="paramForm" label-width="100px">
        <el-form-item label="参数名" required>
          <el-input v-model="paramForm.name" placeholder="请输入参数名称">
            <template #suffix>
              <el-icon><QuestionFilled /></el-icon>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="变量名" required>
          <el-input v-model="paramForm.varName" placeholder="请输入变量名">
            <template #suffix>
              <el-icon><QuestionFilled /></el-icon>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="参数类型" required>
          <el-radio-group v-model="paramForm.type">
            <el-radio-button label="text">文本框</el-radio-button>
            <el-radio-button label="password">密码框</el-radio-button>
            <el-radio-button label="select">下拉选择</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="必填">
          <el-switch v-model="paramForm.required" inactive-text="否" />
        </el-form-item>

        <el-form-item label="默认值">
          <el-input v-model="paramForm.defaultValue" placeholder="请输入" />
        </el-form-item>

        <el-form-item label="提示信息">
          <el-input
            v-model="paramForm.helpText"
            type="textarea"
            :rows="2"
            placeholder="请输入该参数的帮助提示信息"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showParamDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSaveParameter">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Refresh,
  Setting,
  FullScreen,
  Document,
  QuestionFilled
} from '@element-plus/icons-vue'

// 搜索表单
const searchForm = ref({
  type: '',
  name: '',
})

// 分页
const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 1,
})

// 加载状态
const loading = ref(false)

// 模板列表
const templates = ref([
  {
    id: 1,
    name: '获取内存使用情况',
    type: '系统信息',
    content: 'free -m',
    description: '',
  },
])

// 模板对话框
const showTemplateDialog = ref(false)
const isEdit = ref(false)
const templateForm = ref({
  id: 0,
  type: '',
  name: '',
  scriptType: 'Shell',
  content: '',
  parameters: [] as any[],
  remark: '',
})

// 参数对话框
const showParamDialog = ref(false)
const paramForm = ref({
  name: '',
  varName: '',
  type: 'text',
  required: false,
  defaultValue: '',
  helpText: '',
})

// 加载模板列表
const loadTemplates = async () => {
  loading.value = true
  try {
    // TODO: 调用API加载模板列表
    await new Promise((resolve) => setTimeout(resolve, 500))
  } catch (error) {
    ElMessage.error('加载模板列表失败')
  } finally {
    loading.value = false
  }
}

// 新建模板
const handleCreate = () => {
  isEdit.value = false
  templateForm.value = {
    id: 0,
    type: '',
    name: '',
    scriptType: 'Shell',
    content: '',
    parameters: [],
    remark: '',
  }
  showTemplateDialog.value = true
}

// 编辑模板
const handleEdit = (row: any) => {
  isEdit.value = true
  templateForm.value = { ...row }
  if (!templateForm.value.parameters) {
    templateForm.value.parameters = []
  }
  showTemplateDialog.value = true
}

// 删除模板
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定要删除模板 "${row.name}" 吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    ElMessage.success('删除成功')
    loadTemplates()
  } catch {
    // 用户取消
  }
}

// 保存模板
const handleSaveTemplate = async () => {
  if (!templateForm.value.type) {
    ElMessage.warning('请选择模板类型')
    return
  }
  if (!templateForm.value.name) {
    ElMessage.warning('请输入模板名称')
    return
  }
  if (!templateForm.value.content) {
    ElMessage.warning('请输入模板内容')
    return
  }

  try {
    // TODO: 调用API保存模板
    ElMessage.success(isEdit.value ? '编辑成功' : '创建成功')
    showTemplateDialog.value = false
    loadTemplates()
  } catch (error: any) {
    ElMessage.error(error.message || '保存失败')
  }
}

// 移除参数
const removeParameter = (index: number) => {
  templateForm.value.parameters.splice(index, 1)
}

// 保存参数
const handleSaveParameter = () => {
  if (!paramForm.value.name) {
    ElMessage.warning('请输入参数名')
    return
  }
  if (!paramForm.value.varName) {
    ElMessage.warning('请输入变量名')
    return
  }

  templateForm.value.parameters.push({ ...paramForm.value })
  showParamDialog.value = false
  paramForm.value = {
    name: '',
    varName: '',
    type: 'text',
    required: false,
    defaultValue: '',
    helpText: '',
  }
  ElMessage.success('参数添加成功')
}

onMounted(() => {
  loadTemplates()
})
</script>

<style scoped lang="scss">
.template-container {
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  background-color: transparent;
}

.page-header {
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
  border-radius: 10px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 28px;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 14px;
  color: #909399;
  line-height: 20px;
}

.search-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  padding: 20px;
}

.search-row {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}

.search-item {
  display: flex;
  align-items: center;
  gap: 8px;

  .search-label {
    font-size: 14px;
    color: #606266;
    white-space: nowrap;
  }
}

.template-list-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  flex: 1;
  display: flex;
  flex-direction: column;
}

.list-header {
  padding: 16px 20px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  justify-content: space-between;

  .header-title {
    font-size: 16px;
    font-weight: 600;
    color: #303133;
  }

  .header-actions {
    display: flex;
    gap: 8px;
  }
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;

  &:hover {
    background-color: #1a1a1a !important;
  }
}

.pagination {
  padding: 16px 20px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid #e4e7ed;
}

.parameters-list {
  margin-top: 8px;
}
</style>
