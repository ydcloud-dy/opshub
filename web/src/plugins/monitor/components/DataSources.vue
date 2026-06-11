<template>
  <div class="monitor-datasources-container">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Connection /></el-icon>
        </div>
        <div>
          <h2 class="page-title">数据源管理</h2>
          <p class="page-subtitle">接入 Prometheus、VictoriaMetrics、Loki、Elasticsearch 等观测数据源</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="handleAdd">
          <el-icon><Plus /></el-icon>
          新增数据源
        </el-button>
        <el-button @click="loadData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.keyword"
          placeholder="搜索名称或地址..."
          clearable
          class="search-input"
          @keyup.enter="loadData"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
        <el-select
          v-model="searchForm.type"
          placeholder="数据源类型"
          clearable
          class="search-input"
          @change="loadData"
        >
          <el-option label="Prometheus" value="prometheus" />
          <el-option label="VictoriaMetrics" value="victoriametrics" />
          <el-option label="Loki" value="loki" />
          <el-option label="Elasticsearch" value="elasticsearch" />
        </el-select>
      </div>
      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon stat-icon-primary">
          <el-icon><Document /></el-icon>
        </div>
        <div>
          <div class="stat-label">数据源总数</div>
          <div class="stat-value">{{ tableData.length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div>
          <div class="stat-label">连接正常</div>
          <div class="stat-value">{{ normalCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><Warning /></el-icon>
        </div>
        <div>
          <div class="stat-label">未测试</div>
          <div class="stat-value">{{ unknownCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <el-icon><CircleClose /></el-icon>
        </div>
        <div>
          <div class="stat-label">异常</div>
          <div class="stat-value">{{ abnormalCount }}</div>
        </div>
      </div>
    </div>

    <div class="table-wrapper">
      <el-table
        :data="tableData"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="名称" prop="name" min-width="180" show-overflow-tooltip />

        <el-table-column label="类型" width="150" align="center">
          <template #default="{ row }">
            <el-tag :type="getTypeTag(row.type)" effect="light">{{ getTypeName(row.type) }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="地址" prop="url" min-width="280" show-overflow-tooltip />

        <el-table-column label="认证" width="110" align="center">
          <template #default="{ row }">
            <el-tag type="info" effect="light">{{ getAuthName(row.authType) }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="远程写入" width="120" align="center">
          <template #default="{ row }">
            <el-tag
              v-if="isPromCompatible(row.type) && row.remoteWriteEnabled"
              type="success"
              effect="light"
            >
              已开启
            </el-tag>
            <el-tag v-else-if="isPromCompatible(row.type)" type="info" effect="light">未开启</el-tag>
            <span v-else class="muted-text">-</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'normal'" type="success" effect="light">正常</el-tag>
            <el-tag v-else-if="row.status === 'abnormal'" type="danger" effect="light">异常</el-tag>
            <el-tag v-else type="info" effect="light">未测试</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="最后测试" width="190">
          <template #default="{ row }">
            <span class="time-text">{{ formatDateTime(row.lastTestAt) || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="启用" width="90" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="handleEnabledChange(row)" />
          </template>
        </el-table-column>

        <el-table-column label="操作" width="220" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="测试连接" placement="top">
                <el-button link class="action-btn action-check" @click="handleTest(row)" :loading="row.testing">
                  <el-icon><CircleCheck /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="查询测试" placement="top">
                <el-button link class="action-btn action-query" @click="handleOpenQuery(row)">
                  <el-icon><View /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn action-delete" @click="handleDelete(row)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-drawer
      v-model="dialogVisible"
      size="980px"
      class="datasource-wizard-drawer"
      :close-on-click-modal="false"
      :show-close="false"
      @close="handleDialogClose"
    >
      <template #header="{ close, titleId, titleClass }">
        <div class="wizard-drawer-head">
          <el-button text class="drawer-close-btn" @click="close">
            <el-icon><Close /></el-icon>
          </el-button>
          <h3 :id="titleId" :class="titleClass">{{ dialogTitle }}</h3>
        </div>
      </template>

      <div class="wizard-steps">
        <div class="wizard-step" :class="{ active: wizardStep === 1, done: wizardStep > 1 }">
          <span>{{ wizardStep > 1 ? '✓' : '1' }}</span>
          <strong>选择数据源</strong>
        </div>
        <div class="wizard-line"></div>
        <div class="wizard-step" :class="{ active: wizardStep === 2 }">
          <span>2</span>
          <strong>配置数据源</strong>
        </div>
      </div>

      <section v-if="wizardStep === 1" class="source-picker-panel">
        <button
          v-for="item in dataSourceTypeOptions"
          :key="item.value"
          type="button"
          class="source-type-card"
          :class="{ active: form.type === item.value }"
          @click="selectDataSourceType(item.value)"
        >
          <span class="source-type-icon" :class="`source-${item.value}`">
            <el-icon v-if="item.value === 'elasticsearch'"><Search /></el-icon>
            <el-icon v-else-if="item.value === 'loki'"><Document /></el-icon>
            <el-icon v-else><Connection /></el-icon>
          </span>
          <span class="source-type-copy">
            <strong>{{ item.label }}</strong>
            <em>{{ item.desc }}</em>
          </span>
        </button>
      </section>

      <section v-else class="source-config-panel">
        <div class="selected-source-strip">
          <span class="source-type-icon compact" :class="`source-${form.type}`">
            <el-icon v-if="form.type === 'elasticsearch'"><Search /></el-icon>
            <el-icon v-else-if="form.type === 'loki'"><Document /></el-icon>
            <el-icon v-else><Connection /></el-icon>
          </span>
          <div>
            <strong>{{ getTypeName(form.type) }}</strong>
            <p>{{ getTypeDesc(form.type) }}</p>
          </div>
          <el-button v-if="!form.id" text type="primary" @click="wizardStep = 1">重新选择</el-button>
        </div>

        <el-form :model="form" :rules="rules" ref="formRef" label-position="top" class="datasource-wizard-form">
          <section class="wizard-form-section">
            <div class="section-divider"><span>基础信息</span></div>
            <div class="wizard-form-grid">
              <el-form-item label="数据源名称" prop="name">
                <el-input v-model="form.name" placeholder="请输入数据源名称" />
              </el-form-item>
              <el-form-item label="访问地址" prop="url">
                <el-input v-model="form.url" :placeholder="getUrlPlaceholder(form.type)" />
              </el-form-item>
              <el-form-item label="超时时间" prop="timeout">
                <el-input-number v-model="form.timeout" :min="1" :max="120" controls-position="right" style="width: 100%" />
                <span class="form-suffix">秒</span>
              </el-form-item>
              <el-form-item label="启用状态">
                <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
              </el-form-item>
            </div>
          </section>

          <section class="wizard-form-section">
            <div class="section-divider"><span>HTTP</span></div>
            <div class="wizard-form-grid">
              <el-form-item label="TLS 校验">
                <div class="inline-setting">
                  <el-switch v-model="form.skipTlsVerify" active-text="跳过" inactive-text="校验" />
                  <span>内部自签证书或 HTTP 被网关重定向到 HTTPS 时可开启</span>
                </div>
              </el-form-item>
              <el-form-item label="认证方式">
                <el-segmented v-model="form.authType" :options="authOptions" />
              </el-form-item>
              <template v-if="form.authType === 'basic'">
                <el-form-item label="用户名" required>
                  <el-input v-model="form.username" placeholder="请输入用户名" />
                </el-form-item>
                <el-form-item label="密码" required>
                  <el-input v-model="form.password" type="password" placeholder="请输入密码" show-password />
                </el-form-item>
              </template>
              <el-form-item v-if="form.authType === 'bearer'" label="Token" required class="full-row">
                <el-input v-model="form.token" type="textarea" :rows="3" placeholder="请输入 Bearer Token" />
              </el-form-item>
            </div>

            <div class="headers-editor">
              <div class="mini-section-head">
                <span>请求头</span>
                <el-button link type="primary" @click="addHeaderRow(headerRows)">
                  <el-icon><Plus /></el-icon>
                  添加请求头
                </el-button>
              </div>
              <div v-if="headerRows.length" class="header-row-list">
                <div v-for="(header, index) in headerRows" :key="header.key" class="header-row">
                  <el-input v-model="header.name" placeholder="Header 名称" />
                  <el-input v-model="header.value" placeholder="Header 值" />
                  <el-button text class="action-btn action-delete" @click="removeHeaderRow(headerRows, index)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
              </div>
              <div v-else class="empty-dashed">默认不添加自定义 Header</div>
            </div>
          </section>

          <section v-if="isPromCompatible(form.type)" class="wizard-form-section">
            <div class="section-divider"><span>远程写入</span></div>
            <el-alert
              class="remote-write-tip"
              type="info"
              :closable="false"
              show-icon
              title="用于拨测任务写入 opshub_probe_* 指标。Prometheus 需要配置 --web.enable-remote-write-receiver 参数。"
            />
            <div class="wizard-form-grid">
              <el-form-item label="启用写入">
                <el-switch v-model="form.remoteWriteEnabled" active-text="启用" inactive-text="禁用" @change="handleRemoteWriteToggle" />
              </el-form-item>
              <template v-if="form.remoteWriteEnabled">
                <el-form-item label="写入地址" prop="remoteWriteUrl" required class="full-row">
                  <div class="remote-write-url-row">
                    <el-input
                      v-model="form.remoteWriteUrl"
                      placeholder="例如：http://localhost:9090/api/v1/write"
                      @keyup.enter="handleRemoteWriteTest"
                    />
                    <el-button
                      type="primary"
                      class="remote-write-test-btn"
                      :loading="remoteWriteTesting"
                      @click="handleRemoteWriteTest"
                    >
                      <el-icon><CircleCheck /></el-icon>
                      连接测试
                    </el-button>
                  </div>
                </el-form-item>
                <el-form-item label="写入认证方式">
                  <el-segmented v-model="form.remoteWriteAuthType" :options="authOptions" />
                </el-form-item>
                <el-form-item label="写入 TLS">
                  <el-switch v-model="form.remoteWriteSkipTlsVerify" active-text="跳过" inactive-text="校验" />
                </el-form-item>
                <template v-if="form.remoteWriteAuthType === 'basic'">
                  <el-form-item label="写入用户名" required>
                    <el-input v-model="form.remoteWriteUsername" placeholder="请输入用户名" />
                  </el-form-item>
                  <el-form-item label="写入密码" required>
                    <el-input v-model="form.remoteWritePassword" type="password" placeholder="请输入密码" show-password />
                  </el-form-item>
                </template>
                <el-form-item v-if="form.remoteWriteAuthType === 'bearer'" label="写入 Token" required class="full-row">
                  <el-input v-model="form.remoteWriteToken" type="textarea" :rows="3" placeholder="请输入 Bearer Token" />
                </el-form-item>
              </template>
            </div>
            <div v-if="form.remoteWriteEnabled" class="headers-editor">
              <div class="mini-section-head">
                <span>写入请求头</span>
                <el-button link type="primary" @click="addHeaderRow(remoteHeaderRows)">
                  <el-icon><Plus /></el-icon>
                  添加请求头
                </el-button>
              </div>
              <div v-if="remoteHeaderRows.length" class="header-row-list">
                <div v-for="(header, index) in remoteHeaderRows" :key="header.key" class="header-row">
                  <el-input v-model="header.name" placeholder="Header 名称" />
                  <el-input v-model="header.value" placeholder="Header 值" />
                  <el-button text class="action-btn action-delete" @click="removeHeaderRow(remoteHeaderRows, index)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
              </div>
              <div v-else class="empty-dashed">默认不添加写入 Header</div>
            </div>
          </section>

          <section class="wizard-form-section">
            <div class="section-divider"><span>其他</span></div>
            <el-form-item label="描述">
              <el-input v-model="form.description" type="textarea" :rows="3" placeholder="请输入备注" />
            </el-form-item>
          </section>
        </el-form>
      </section>

      <template #footer>
        <div class="wizard-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <div v-if="wizardStep === 2" class="wizard-footer-actions">
            <el-button v-if="!form.id" @click="wizardStep = 1">上一步</el-button>
            <el-button @click="handleCurrentTest" :loading="currentTesting">
              <el-icon><CircleCheck /></el-icon>
              连接测试
            </el-button>
            <el-button type="primary" @click="handleSubmit" :loading="submitting">提交</el-button>
          </div>
        </div>
      </template>
    </el-drawer>

    <el-dialog
      v-model="queryDialogVisible"
      title="查询测试"
      width="1040px"
      class="datasource-dialog query-dialog"
      :close-on-click-modal="false"
    >
      <div v-if="currentSource" class="query-source">
        <el-tag :type="getTypeTag(currentSource.type)" effect="light">{{ getTypeName(currentSource.type) }}</el-tag>
        <span>{{ currentSource.name }}</span>
      </div>

      <el-form :model="queryForm" label-width="100px">
        <el-form-item label="查询模式" v-if="currentSource?.type !== 'elasticsearch'">
          <el-segmented v-model="queryForm.queryMode" :options="queryModeOptions" />
        </el-form-item>
        <el-form-item label="索引" v-if="currentSource?.type === 'elasticsearch'">
          <div class="query-index-select-row">
            <el-select
              v-model="queryForm.index"
              filterable
              allow-create
              default-first-option
              clearable
              remote
              reserve-keyword
              :remote-method="loadQueryIndices"
              :loading="queryIndexLoading"
              placeholder="选择索引 / 别名，也可输入 logs-*"
              @visible-change="handleQueryIndexDropdownVisible"
            >
              <el-option
                v-for="item in queryIndexOptions"
                :key="`${item.type}-${item.name}`"
                :label="item.name"
                :value="item.name"
              >
                <div class="query-index-option">
                  <span>{{ item.name }}</span>
                  <em>{{ item.type === 'alias' ? 'alias' : item.status || 'index' }}{{ item.docsCount ? ` · ${item.docsCount} docs` : '' }}</em>
                </div>
              </el-option>
            </el-select>
            <el-button :loading="queryIndexLoading" @click="loadQueryIndices()">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </el-form-item>
        <el-form-item :label="currentSource?.type === 'elasticsearch' ? '查询 DSL' : '查询语句'">
          <el-input
            v-model="queryForm.query"
            type="textarea"
            :rows="5"
            :placeholder="getQueryPlaceholder(currentSource?.type)"
          />
        </el-form-item>
      </el-form>

      <div class="query-actions">
        <el-button type="primary" @click="handleQuery" :loading="querying">
          <el-icon><View /></el-icon>
          执行查询
        </el-button>
      </div>

      <div v-if="queryRawResult" class="query-result-panel">
        <div class="query-summary-grid">
          <div class="query-summary-item success">
            <span>查询状态</span>
            <strong>{{ queryStatusText }}</strong>
          </div>
          <div class="query-summary-item">
            <span>响应耗时</span>
            <strong>{{ queryRawResult.duration ?? '-' }}ms</strong>
          </div>
          <div class="query-summary-item">
            <span>结果类型</span>
            <strong>{{ queryResultTypeText }}</strong>
          </div>
          <div class="query-summary-item">
            <span>展示条数</span>
            <strong>{{ queryPreviewRows.length }}</strong>
          </div>
        </div>

        <div class="query-preview-section">
          <div class="query-preview-head">
            <div>
              <strong>查询结果</strong>
              <span>{{ queryPreviewHint }}</span>
            </div>
          </div>

          <div v-if="queryPreviewRows.length && queryResultKind === 'log'" class="query-log-list">
            <article v-for="(row, index) in queryPreviewRows" :key="`${row.title}-${row.timestamp}-${index}`" class="query-log-card">
              <div class="query-log-head">
                <strong>{{ row.title }}</strong>
                <span>{{ row.timestamp || '-' }}</span>
              </div>
              <div v-if="row.labels.length" class="query-labels">
                <span v-for="label in row.labels" :key="`${row.title}-${label.key}-${label.value}`">
                  <b>{{ label.key }}</b>
                  <em>{{ label.value }}</em>
                </span>
              </div>
              <pre>{{ row.message || row.value }}</pre>
            </article>
          </div>

          <el-table
            v-else-if="queryPreviewRows.length"
            :data="queryPreviewRows"
            class="query-preview-table"
            max-height="360"
            :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
          >
            <el-table-column label="序列 / 文档" min-width="240" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="query-row-title">
                  <strong>{{ row.title }}</strong>
                  <small>{{ row.timestamp || '-' }}</small>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="标签" min-width="360">
              <template #default="{ row }">
                <div v-if="row.labels.length" class="query-labels">
                  <span v-for="label in row.labels" :key="`${row.title}-${label.key}-${label.value}`">
                    <b>{{ label.key }}</b>
                    <em>{{ label.value }}</em>
                  </span>
                </div>
                <span v-else class="muted-text">无标签</span>
              </template>
            </el-table-column>
            <el-table-column label="当前值 / 摘要" min-width="180" align="right" show-overflow-tooltip>
              <template #default="{ row }">
                <strong class="query-value-text" :title="row.value">{{ row.value }}</strong>
              </template>
            </el-table-column>
          </el-table>

          <el-empty v-else :image-size="72" description="查询成功，但当前响应中没有可展示的数据" />
        </div>

        <el-collapse class="query-raw-collapse">
          <el-collapse-item title="原始响应 JSON" name="raw">
            <pre class="query-result">{{ queryResult }}</pre>
          </el-collapse-item>
        </el-collapse>
      </div>

      <el-empty v-else class="query-empty" :image-size="72" description="输入查询语句后点击执行查询，结果会在这里展示" />

      <template #footer>
        <el-button @click="queryDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import {
  CircleCheck,
  CircleClose,
  Close,
  Connection,
  Delete,
  Document,
  Edit,
  Plus,
  Refresh,
  RefreshLeft,
  Search,
  View,
  Warning
} from '@element-plus/icons-vue'
import {
  createMonitorDataSource,
  deleteMonitorDataSource,
  getMonitorDataSourceIndices,
  getMonitorDataSources,
  queryMonitorDataSource,
  testMonitorDataSource,
  testMonitorDataSourceRemoteWriteConfig,
  updateMonitorDataSource,
  type DataSourceAuthType,
  type DataSourceQueryRequest,
  type DataSourceType,
  type MonitorDataSourceIndex,
  type MonitorDataSource
} from '@/api/monitor-datasource'

interface DataSourceForm {
  id?: number
  name: string
  type: DataSourceType
  url: string
  authType: DataSourceAuthType
  username: string
  password: string
  token: string
  headers: string
  timeout: number
  skipTlsVerify: boolean
  enabled: boolean
  remoteWriteEnabled: boolean
  remoteWriteUrl: string
  remoteWriteAuthType: DataSourceAuthType
  remoteWriteUsername: string
  remoteWritePassword: string
  remoteWriteToken: string
  remoteWriteHeaders: string
  remoteWriteSkipTlsVerify: boolean
  description: string
}

interface QueryTestResult {
  duration?: number
  statusCode?: number
  result?: any
}

interface QueryPreviewLabel {
  key: string
  value: string
}

interface QueryPreviewRow {
  title: string
  value: string
  timestamp: string
  labels: QueryPreviewLabel[]
  kind: 'metric' | 'log' | 'document'
  message?: string
}

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const queryDialogVisible = ref(false)
const querying = ref(false)
const queryIndexLoading = ref(false)
const remoteWriteTesting = ref(false)
const currentTesting = ref(false)
const wizardStep = ref<1 | 2>(1)
const dialogTitle = ref('新增数据源')
const formRef = ref<FormInstance>()
const tableData = ref<(MonitorDataSource & { testing?: boolean })[]>([])
const currentSource = ref<MonitorDataSource | null>(null)
const queryRawResult = ref<QueryTestResult | null>(null)
const queryIndexOptions = ref<MonitorDataSourceIndex[]>([])

interface HeaderRow {
  key: string
  name: string
  value: string
}

const headerRows = ref<HeaderRow[]>([])
const remoteHeaderRows = ref<HeaderRow[]>([])

const searchForm = reactive({
  keyword: '',
  type: ''
})

const form = reactive<DataSourceForm>({
  name: '',
  type: 'prometheus',
  url: '',
  authType: 'none',
  username: '',
  password: '',
  token: '',
  headers: '',
  timeout: 10,
  skipTlsVerify: false,
  enabled: true,
  remoteWriteEnabled: false,
  remoteWriteUrl: '',
  remoteWriteAuthType: 'none',
  remoteWriteUsername: '',
  remoteWritePassword: '',
  remoteWriteToken: '',
  remoteWriteHeaders: '',
  remoteWriteSkipTlsVerify: false,
  description: ''
})

const queryForm = reactive({
  queryMode: 'instant' as 'instant' | 'range',
  query: '',
  index: ''
})

const authOptions = [
  { label: '无认证', value: 'none' },
  { label: 'Basic', value: 'basic' },
  { label: 'Bearer', value: 'bearer' }
]

const queryModeOptions = [
  { label: '即时查询', value: 'instant' },
  { label: '范围查询', value: 'range' }
]

const dataSourceTypeOptions: Array<{ label: string; value: DataSourceType; desc: string }> = [
  { label: 'Prometheus', value: 'prometheus', desc: 'PromQL 指标查询和远程写入' },
  { label: 'VictoriaMetrics', value: 'victoriametrics', desc: '兼容 Prometheus 的高性能时序库' },
  { label: 'Loki', value: 'loki', desc: 'LogQL 日志聚合与检索' },
  { label: 'Elasticsearch', value: 'elasticsearch', desc: '日志 DSL 查询和搜索分析' }
]

function isPromCompatible(type?: string) {
  return type === 'prometheus' || type === 'victoriametrics'
}

const rules: FormRules = {
  name: [{ required: true, message: '请输入数据源名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择数据源类型', trigger: 'change' }],
  url: [{ required: true, message: '请输入访问地址', trigger: 'blur' }],
  authType: [{ required: true, message: '请选择认证方式', trigger: 'change' }],
  remoteWriteUrl: [
    {
      validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
        if (isPromCompatible(form.type) && form.remoteWriteEnabled && !String(value || '').trim()) {
          callback(new Error('启用远程写入后必须填写远程写入地址'))
          return
        }
        callback()
      },
      trigger: ['blur', 'change']
    }
  ]
}

const normalCount = computed(() => tableData.value.filter(item => item.status === 'normal').length)
const abnormalCount = computed(() => tableData.value.filter(item => item.status === 'abnormal').length)
const unknownCount = computed(() => tableData.value.filter(item => !item.status || item.status === 'unknown').length)
const queryResult = computed(() => queryRawResult.value ? JSON.stringify(queryRawResult.value, null, 2) : '')
const queryPreviewRows = computed(() => normalizeQueryPreviewRows(queryRawResult.value?.result, currentSource.value?.type))
const queryResultKind = computed(() => {
  if (queryPreviewRows.value.some(item => item.kind === 'log')) return 'log'
  if (queryPreviewRows.value.some(item => item.kind === 'document')) return 'document'
  return 'metric'
})
const queryResultTypeText = computed(() => getQueryResultTypeText(queryRawResult.value?.result, currentSource.value?.type))
const queryStatusText = computed(() => {
  const raw = queryRawResult.value?.result
  const status = raw?.status || raw?.data?.status
  if (status) return status === 'success' ? '成功' : String(status)
  return queryRawResult.value ? '成功' : '-'
})
const queryPreviewHint = computed(() => {
  if (queryResultKind.value === 'log') return '展示最近返回的日志行，完整响应可在下方展开查看'
  if (queryResultKind.value === 'document') return '展示命中的文档摘要，完整 _source 可在原始响应中查看'
  return '展示每个时间序列的最新值和标签'
})

const getTypeName = (type?: string) => {
  const map: Record<string, string> = {
    prometheus: 'Prometheus',
    victoriametrics: 'VictoriaMetrics',
    loki: 'Loki',
    elasticsearch: 'Elasticsearch'
  }
  return type ? map[type] || type : '-'
}

const getTypeTag = (type?: string) => {
  const map: Record<string, string> = {
    prometheus: 'success',
    victoriametrics: 'primary',
    loki: 'warning',
    elasticsearch: 'danger'
  }
  return type ? map[type] || 'info' : 'info'
}

const getAuthName = (type?: string) => {
  const map: Record<string, string> = {
    none: '无认证',
    basic: 'Basic',
    bearer: 'Bearer'
  }
  return type ? map[type] || type : '无认证'
}

const getTypeDesc = (type?: string) => {
  return dataSourceTypeOptions.find(item => item.value === type)?.desc || '外部观测数据源'
}

const getUrlPlaceholder = (type?: string) => {
  if (type === 'loki') return 'http://127.0.0.1:3100'
  if (type === 'elasticsearch') return 'http://127.0.0.1:9200'
  if (type === 'victoriametrics') return 'http://127.0.0.1:8428'
  return 'http://127.0.0.1:9090'
}

const formatDateTime = (date?: string) => {
  if (!date) return ''
  const d = new Date(date)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const getQueryPlaceholder = (type?: string) => {
  if (type === 'loki') return '{job!=""}'
  if (type === 'elasticsearch') return JSON.stringify({
    size: 0,
    track_total_hits: true,
    query: {
      match_all: {}
    }
  }, null, 2)
  return 'up'
}

const handleTypeChange = () => {
  if (!isPromCompatible(form.type)) {
    form.remoteWriteEnabled = false
    remoteHeaderRows.value = []
  }
  formRef.value?.clearValidate('remoteWriteUrl')
}

const selectDataSourceType = (type: DataSourceType) => {
  form.type = type
  handleTypeChange()
  wizardStep.value = 2
}

const handleRemoteWriteToggle = () => {
  if (!form.remoteWriteEnabled) {
    formRef.value?.clearValidate('remoteWriteUrl')
    return
  }
  formRef.value?.validateField('remoteWriteUrl').catch(() => undefined)
}

const loadData = async () => {
  loading.value = true
  try {
    const data = await getMonitorDataSources({
      keyword: searchForm.keyword || undefined,
      type: searchForm.type || undefined
    })
    tableData.value = data || []
  } finally {
    loading.value = false
  }
}

const handleReset = () => {
  searchForm.keyword = ''
  searchForm.type = ''
  loadData()
}

const resetForm = () => {
  form.id = undefined
  form.name = ''
  form.type = 'prometheus'
  form.url = ''
  form.authType = 'none'
  form.username = ''
  form.password = ''
  form.token = ''
  form.headers = ''
  form.timeout = 10
  form.skipTlsVerify = false
  form.enabled = true
  form.remoteWriteEnabled = false
  form.remoteWriteUrl = ''
  form.remoteWriteAuthType = 'none'
  form.remoteWriteUsername = ''
  form.remoteWritePassword = ''
  form.remoteWriteToken = ''
  form.remoteWriteHeaders = ''
  form.remoteWriteSkipTlsVerify = false
  form.description = ''
  headerRows.value = []
  remoteHeaderRows.value = []
  wizardStep.value = 1
  formRef.value?.clearValidate()
}

const handleAdd = () => {
  resetForm()
  dialogTitle.value = '创建数据源'
  dialogVisible.value = true
}

const handleEdit = (row: MonitorDataSource) => {
  resetForm()
  dialogTitle.value = '编辑数据源'
  Object.assign(form, {
    id: row.id,
    name: row.name,
    type: row.type,
    url: row.url,
    authType: row.authType || 'none',
    username: row.username || '',
    password: row.password || '',
    token: row.token || '',
    headers: row.headers || '',
    timeout: row.timeout || 10,
    skipTlsVerify: row.skipTlsVerify || false,
    enabled: row.enabled,
    remoteWriteEnabled: row.remoteWriteEnabled || false,
    remoteWriteUrl: row.remoteWriteUrl || '',
    remoteWriteAuthType: row.remoteWriteAuthType || 'none',
    remoteWriteUsername: row.remoteWriteUsername || '',
    remoteWritePassword: row.remoteWritePassword || '',
    remoteWriteToken: row.remoteWriteToken || '',
    remoteWriteHeaders: row.remoteWriteHeaders || '',
    remoteWriteSkipTlsVerify: row.remoteWriteSkipTlsVerify || false,
    description: row.description || ''
  })
  headerRows.value = parseHeaderRows(row.headers)
  remoteHeaderRows.value = parseHeaderRows(row.remoteWriteHeaders)
  wizardStep.value = 2
  dialogVisible.value = true
}

const handleDialogClose = () => {
  resetForm()
}

const buildPayload = (): MonitorDataSource => {
  const promCompatible = isPromCompatible(form.type)
  return {
    id: form.id,
    name: form.name,
    type: form.type,
    url: form.url,
    authType: form.authType,
    username: form.username,
    password: form.password,
    token: form.token,
    headers: buildHeaderJSON(headerRows.value),
    timeout: form.timeout,
    skipTlsVerify: form.skipTlsVerify,
    enabled: form.enabled,
    remoteWriteEnabled: promCompatible ? form.remoteWriteEnabled : false,
    remoteWriteUrl: promCompatible ? form.remoteWriteUrl : '',
    remoteWriteAuthType: promCompatible ? form.remoteWriteAuthType : 'none',
    remoteWriteUsername: promCompatible ? form.remoteWriteUsername : '',
    remoteWritePassword: promCompatible ? form.remoteWritePassword : '',
    remoteWriteToken: promCompatible ? form.remoteWriteToken : '',
    remoteWriteHeaders: promCompatible ? buildHeaderJSON(remoteHeaderRows.value) : '',
    remoteWriteSkipTlsVerify: promCompatible ? form.remoteWriteSkipTlsVerify || form.skipTlsVerify : false,
    description: form.description
  }
}

const createHeaderRow = (name = '', value = ''): HeaderRow => ({
  key: `${Date.now()}-${Math.random()}`,
  name,
  value
})

const addHeaderRow = (rows: HeaderRow[]) => {
  rows.push(createHeaderRow())
}

const removeHeaderRow = (rows: HeaderRow[], index: number) => {
  rows.splice(index, 1)
}

const parseHeaderRows = (raw?: string): HeaderRow[] => {
  if (!raw?.trim()) return []
  try {
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return []
    return Object.entries(parsed).map(([name, value]) => createHeaderRow(name, String(value ?? '')))
  } catch {
    return []
  }
}

const buildHeaderJSON = (rows: HeaderRow[]) => {
  const result = rows.reduce<Record<string, string>>((acc, item) => {
    const name = item.name.trim()
    const value = item.value.trim()
    if (name) acc[name] = value
    return acc
  }, {})
  return Object.keys(result).length ? JSON.stringify(result) : ''
}

const handleRemoteWriteTest = async () => {
  if (!isPromCompatible(form.type)) {
    ElMessage.warning('只有 Prometheus 或 VictoriaMetrics 支持远程写入测试')
    return
  }
  if (!form.remoteWriteEnabled) {
    ElMessage.warning('请先启用远程写入')
    return
  }
  if (!form.remoteWriteUrl.trim()) {
    ElMessage.warning('启用远程写入后必须填写远程写入地址')
    formRef.value?.validateField('remoteWriteUrl').catch(() => undefined)
    return
  }
  if (!validateRemoteWriteAuthConfig()) {
    return
  }

  remoteWriteTesting.value = true
  try {
    const result = await testMonitorDataSourceRemoteWriteConfig(buildPayload())
    if (result?.ok) {
      ElMessage.success(`远程写入连接正常，耗时 ${result.duration}ms`)
    } else {
      ElMessage.warning(result?.error || '远程写入连接异常')
    }
  } finally {
    remoteWriteTesting.value = false
  }
}

const handleSubmit = async () => {
  await formRef.value?.validate()
  if (!validateDataSourceAuthConfig()) {
    return
  }
  if (isPromCompatible(form.type) && form.remoteWriteEnabled && !form.remoteWriteUrl.trim()) {
    ElMessage.warning('启用远程写入后必须填写远程写入地址')
    formRef.value?.validateField('remoteWriteUrl').catch(() => undefined)
    return
  }
  if (isPromCompatible(form.type) && form.remoteWriteEnabled && !validateRemoteWriteAuthConfig()) {
    return
  }
  submitting.value = true
  try {
    if (form.id) {
      await updateMonitorDataSource(form.id, buildPayload())
      ElMessage.success('更新成功')
    } else {
      await createMonitorDataSource(buildPayload())
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadData()
  } finally {
    submitting.value = false
  }
}

const handleCurrentTest = async () => {
  if (!form.id) {
    ElMessage.warning('新增数据源请先提交保存，保存后可进行连接测试')
    return
  }
  currentTesting.value = true
  try {
    const result = await testMonitorDataSource(form.id)
    if (result?.ok) {
      ElMessage.success(`连接正常，耗时 ${result.duration}ms`)
    } else {
      ElMessage.warning(result?.error || '连接异常')
    }
    loadData()
  } finally {
    currentTesting.value = false
  }
}

const validateDataSourceAuthConfig = () => {
  if (form.authType === 'basic' && (!form.username.trim() || !form.password.trim())) {
    ElMessage.warning('Basic 认证必须填写用户名和密码')
    return false
  }
  if (form.authType === 'bearer' && !form.token.trim()) {
    ElMessage.warning('Bearer 认证必须填写 Token')
    return false
  }
  return true
}

const validateRemoteWriteAuthConfig = () => {
  if (form.remoteWriteAuthType === 'basic' && (!form.remoteWriteUsername.trim() || !form.remoteWritePassword.trim())) {
    ElMessage.warning('远程写入 Basic 认证必须填写用户名和密码')
    return false
  }
  if (form.remoteWriteAuthType === 'bearer' && !form.remoteWriteToken.trim()) {
    ElMessage.warning('远程写入 Bearer 认证必须填写 Token')
    return false
  }
  return true
}

const handleEnabledChange = async (row: MonitorDataSource) => {
  if (!row.id) return
  try {
    await updateMonitorDataSource(row.id, row)
    ElMessage.success('状态已更新')
  } catch {
    row.enabled = !row.enabled
  }
}

const handleTest = async (row: MonitorDataSource & { testing?: boolean }) => {
  if (!row.id) return
  row.testing = true
  try {
    const result = await testMonitorDataSource(row.id)
    if (result?.ok) {
      ElMessage.success(`连接正常，耗时 ${result.duration}ms`)
    } else {
      ElMessage.warning(result?.error || '连接异常')
    }
    loadData()
  } finally {
    row.testing = false
  }
}

const handleOpenQuery = (row: MonitorDataSource) => {
  currentSource.value = row
  queryForm.queryMode = 'instant'
  queryForm.query = getQueryPlaceholder(row.type)
  queryForm.index = ''
  queryRawResult.value = null
  queryIndexOptions.value = []
  queryDialogVisible.value = true
  if (row.type === 'elasticsearch') {
    loadQueryIndices()
  }
}

const handleQueryIndexDropdownVisible = (visible: boolean) => {
  if (visible && !queryIndexOptions.value.length) {
    loadQueryIndices()
  }
}

const loadQueryIndices = async (keyword = '') => {
  if (!currentSource.value?.id || currentSource.value.type !== 'elasticsearch') {
    queryIndexOptions.value = []
    return
  }
  queryIndexLoading.value = true
  try {
    queryIndexOptions.value = await getMonitorDataSourceIndices(currentSource.value.id, {
      keyword: keyword || undefined,
      limit: 300
    }) || []
  } finally {
    queryIndexLoading.value = false
  }
}

const handleQuery = async () => {
  if (!currentSource.value?.id) return
  querying.value = true
  try {
    const data = await queryMonitorDataSource(currentSource.value.id, buildQueryPayload())
    queryRawResult.value = data || null
  } finally {
    querying.value = false
  }
}

const buildQueryPayload = (): DataSourceQueryRequest => {
  const payload: DataSourceQueryRequest = {
    queryMode: queryForm.queryMode,
    query: queryForm.query,
    index: queryForm.index
  }
  if (currentSource.value?.type !== 'elasticsearch' && queryForm.queryMode === 'range') {
    const end = new Date()
    const start = new Date(end.getTime() - 30 * 60 * 1000)
    payload.start = String(Math.floor(start.getTime() / 1000))
    payload.end = String(Math.floor(end.getTime() / 1000))
    payload.step = '60s'
    payload.limit = 100
  }
  return payload
}

const normalizeQueryPreviewRows = (raw: any, type?: string): QueryPreviewRow[] => {
  if (!raw) return []
  if (type === 'elasticsearch') return normalizeElasticsearchRows(raw)

  const data = raw?.data || raw
  const resultType = data?.resultType || raw?.data?.resultType
  const result = data?.result ?? raw?.result

  if (resultType === 'scalar' && Array.isArray(result)) {
    return [{
      title: 'Scalar',
      value: formatQueryValue(result[1]),
      timestamp: formatQueryTime(result[0]),
      labels: [],
      kind: 'metric'
    }]
  }

  if (!Array.isArray(result)) return []
  if (resultType === 'streams' || result.some((item: any) => item?.stream && Array.isArray(item?.values))) {
    return normalizeLokiRows(result)
  }
  return result.map((item: any, index: number) => normalizeMetricRow(item, index)).filter(Boolean)
}

const normalizeMetricRow = (item: any, index: number): QueryPreviewRow => {
  const metric = item?.metric || item?.labels || {}
  const values = Array.isArray(item?.values) ? item.values : Array.isArray(item?.value) ? [item.value] : []
  const last = values[values.length - 1] || []
  return {
    title: formatMetricTitle(metric, index),
    value: formatQueryValue(last[1]),
    timestamp: formatQueryTime(last[0]),
    labels: labelsFromRecord(metric, ['__name__']),
    kind: 'metric'
  }
}

const normalizeLokiRows = (streams: any[]): QueryPreviewRow[] => {
  const rows: QueryPreviewRow[] = []
  streams.forEach((stream: any, streamIndex: number) => {
    const labels = labelsFromRecord(stream?.stream || {})
    const values = Array.isArray(stream?.values) ? stream.values : []
    values.slice(0, 20).forEach((pair: any[], index: number) => {
      const line = String(pair?.[1] || '').trim()
      if (!line) return
      rows.push({
        title: labels.length ? formatLabelSet(labels, `Log Stream #${streamIndex + 1}`) : `Log Stream #${streamIndex + 1}.${index + 1}`,
        value: clipText(line, 120),
        message: line,
        timestamp: formatQueryTime(pair?.[0]),
        labels,
        kind: 'log'
      })
    })
  })
  return rows.slice(0, 100)
}

const normalizeElasticsearchRows = (raw: any): QueryPreviewRow[] => {
  const rows: QueryPreviewRow[] = []
  collectAggregationRows(raw?.aggregations || raw?.data?.aggregations, rows)
  const hits = raw?.hits?.hits || raw?.data?.hits?.hits
  if (Array.isArray(hits)) {
    hits.slice(0, 100).forEach((hit: any, index: number) => {
      const source = hit?._source || {}
      const message = source.message || source.msg || source.log || JSON.stringify(source)
      rows.push({
        title: hit?._id ? `Document ${hit._id}` : `Document #${index + 1}`,
        value: clipText(String(message || '-'), 140),
        message: String(message || '-'),
        timestamp: formatQueryTime(source['@timestamp'] || source.timestamp || source.time),
        labels: labelsFromRecord({ index: hit?._index, score: hit?._score }, []),
        kind: 'document'
      })
    })
  }
  return rows
}

const collectAggregationRows = (value: any, rows: QueryPreviewRow[], path: string[] = []) => {
  if (!value || typeof value !== 'object') return
  Object.entries(value).forEach(([key, child]) => {
    const nextPath = [...path, key]
    if (child && typeof child === 'object' && 'value' in child) {
      rows.push({
        title: nextPath.join(' / '),
        value: formatQueryValue((child as any).value),
        timestamp: '-',
        labels: [],
        kind: 'metric'
      })
      return
    }
    if (child && typeof child === 'object' && Array.isArray((child as any).buckets)) {
      ;(child as any).buckets.slice(0, 50).forEach((bucket: any) => {
        rows.push({
          title: `${nextPath.join(' / ')}: ${bucket.key_as_string || bucket.key}`,
          value: formatQueryValue(bucket.doc_count),
          timestamp: '-',
          labels: labelsFromRecord({ key: bucket.key_as_string || bucket.key }),
          kind: 'metric'
        })
        collectAggregationRows(bucket, rows, nextPath)
      })
      return
    }
    collectAggregationRows(child, rows, nextPath)
  })
}

const getQueryResultTypeText = (raw: any, type?: string) => {
  if (!raw) return '-'
  if (type === 'elasticsearch') return 'Elasticsearch 文档 / 聚合'
  const data = raw?.data || raw
  const resultType = data?.resultType || raw?.data?.resultType
  const map: Record<string, string> = {
    vector: 'Vector 即时向量',
    matrix: 'Matrix 范围矩阵',
    scalar: 'Scalar 标量',
    streams: 'Loki 日志流'
  }
  return resultType ? map[resultType] || resultType : '查询响应'
}

const labelsFromRecord = (record: Record<string, any> | undefined, omit: string[] = []): QueryPreviewLabel[] => {
  if (!record || typeof record !== 'object') return []
  return Object.entries(record)
    .filter(([key, value]) => !omit.includes(key) && value !== undefined && value !== null && String(value) !== '')
    .slice(0, 12)
    .map(([key, value]) => ({ key, value: clipText(String(value), 96) }))
}

const formatMetricTitle = (metric: Record<string, any> | undefined, index: number) => {
  if (!metric || typeof metric !== 'object') return `Series #${index + 1}`
  const metricName = String(metric.__name__ || '').trim()
  const labels = labelsFromRecord(metric, ['__name__'])
  const labelSet = formatLabelSet(labels, `Series #${index + 1}`)
  return metricName ? `${metricName} ${labelSet}` : labelSet
}

const formatLabelSet = (labels: QueryPreviewLabel[], fallback = '-') => {
  if (!labels.length) return fallback
  const preferred = ['instance', 'job', 'pod', 'namespace', 'container', 'cluster', 'app']
  const picked = preferred
    .map(key => labels.find(label => label.key === key))
    .filter((label): label is QueryPreviewLabel => Boolean(label))
  const finalLabels = picked.length ? picked.slice(0, 3) : labels.slice(0, 3)
  return `{${finalLabels.map(label => `${label.key}=${label.value}`).join(', ')}}`
}

const formatQueryValue = (value: unknown) => {
  const numberValue = typeof value === 'number' ? value : Number(value)
  if (Number.isFinite(numberValue)) {
    return Number.isInteger(numberValue) ? String(numberValue) : numberValue.toFixed(4).replace(/\.?0+$/, '')
  }
  return clipText(String(value ?? '-'), 140)
}

const formatQueryTime = (value: unknown) => {
  if (value === undefined || value === null || value === '') return '-'
  if (typeof value === 'number' || /^\d+(\.\d+)?$/.test(String(value))) {
    const numeric = Number(value)
    const milliseconds = numeric > 1e15 ? numeric / 1e6 : numeric > 1e12 ? numeric : numeric * 1000
    if (Number.isFinite(milliseconds)) return formatDateTime(new Date(milliseconds).toISOString()) || '-'
  }
  const parsed = new Date(String(value))
  if (!Number.isNaN(parsed.getTime())) return formatDateTime(parsed.toISOString()) || '-'
  return String(value)
}

const clipText = (value: string, max = 80) => value.length > max ? `${value.slice(0, max)}...` : value

const handleDelete = async (row: MonitorDataSource) => {
  if (!row.id) return
  await ElMessageBox.confirm(`确定删除数据源「${row.name}」吗？`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await deleteMonitorDataSource(row.id)
  ElMessage.success('删除成功')
  loadData()
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.monitor-datasources-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0;
  background: transparent;
}

.page-header,
.search-bar,
.stat-card,
.table-wrapper {
  background: #fff;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  box-shadow: none;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px;
}

.page-title-group {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-title-icon {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 8px;
  border: 1px solid #edf1f7;
  background: #f8fafc;
  color: #111827;
  font-size: 22px;
}

.page-title {
  margin: 0;
  color: #111827;
  font-size: 22px;
  font-weight: 750;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0;
  color: #667085;
  font-size: 13px;
}

.header-actions,
.search-inputs,
.search-actions,
.action-buttons,
.query-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.search-bar {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
}

.search-inputs {
  flex: 1;
}

.search-input {
  width: 280px;
}

.search-icon {
  color: #98a2b3;
}

.reset-btn {
  background: #fff;
  border-color: #d8dee9;
  color: #344054;
}

.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 92px;
  padding: 16px;
}

.stat-icon {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 8px;
  font-size: 21px;
}

.stat-icon-primary {
  background: #eef4ff;
  border: 1px solid #dbe8ff;
  color: #2563eb;
}

.stat-icon-success {
  background: #ecfdf3;
  border: 1px solid #bbf7d0;
  color: #16a34a;
}

.stat-icon-warning {
  background: #fffbeb;
  border: 1px solid #fde68a;
  color: #d97706;
}

.stat-icon-danger {
  background: #fff1f2;
  border: 1px solid #fecdd3;
  color: #e11d48;
}

.stat-label {
  color: #667085;
  font-size: 13px;
}

.stat-value {
  margin-top: 4px;
  color: #111827;
  font-size: 28px;
  font-weight: 750;
  line-height: 1;
}

.table-wrapper {
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.time-text {
  white-space: nowrap;
}

.muted-text {
  color: #98a2b3;
}

.wizard-drawer-head {
  display: flex;
  align-items: center;
  gap: 12px;
}

.wizard-drawer-head h3 {
  margin: 0;
  color: #111827;
  font-size: 20px;
  font-weight: 760;
}

.drawer-close-btn {
  width: 34px;
  height: 34px;
  color: #667085;
}

.wizard-steps {
  display: grid;
  grid-template-columns: auto minmax(120px, 1fr) auto;
  align-items: center;
  gap: 18px;
  margin: 4px 0 28px;
  padding: 10px 0 4px;
}

.wizard-step {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #98a2b3;
}

.wizard-step span {
  width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: #f2f4f7;
  color: #667085;
  font-weight: 760;
}

.wizard-step strong {
  color: inherit;
  font-size: 18px;
  font-weight: 760;
}

.wizard-step.active,
.wizard-step.done {
  color: #111827;
}

.wizard-step.active span {
  background: #1677ff;
  color: #fff;
}

.wizard-step.done span {
  background: #e8f3ff;
  color: #1677ff;
}

.wizard-line {
  height: 2px;
  background: #1677ff;
}

.source-picker-panel {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  padding: 8px 4px 24px;
}

.source-type-card {
  min-height: 132px;
  display: flex;
  align-items: center;
  gap: 22px;
  padding: 24px 28px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
  color: #111827;
  text-align: left;
  cursor: pointer;
  transition: border-color .18s ease, box-shadow .18s ease, transform .18s ease;
}

.source-type-card:hover,
.source-type-card.active {
  border-color: #1677ff;
  box-shadow: 0 10px 28px rgba(22, 119, 255, .12);
  transform: translateY(-1px);
}

.source-type-icon {
  width: 54px;
  height: 54px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 12px;
  background: #eef4ff;
  color: #1677ff;
  font-size: 28px;
}

.source-type-icon.compact {
  width: 42px;
  height: 42px;
  border-radius: 10px;
  font-size: 22px;
}

.source-victoriametrics {
  background: #f0fdf4;
  color: #16a34a;
}

.source-loki {
  background: #fffbeb;
  color: #d97706;
}

.source-elasticsearch {
  background: #fff1f2;
  color: #e11d48;
}

.source-type-copy {
  min-width: 0;
  display: grid;
  gap: 8px;
}

.source-type-copy strong {
  font-size: 22px;
  font-weight: 780;
  color: #111827;
}

.source-type-copy em {
  color: #667085;
  font-size: 15px;
  font-style: normal;
  line-height: 1.55;
}

.source-config-panel {
  padding-bottom: 18px;
}

.selected-source-strip {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  padding: 14px 16px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fbfcfe;
}

.selected-source-strip div {
  flex: 1;
  min-width: 0;
}

.selected-source-strip strong {
  color: #111827;
  font-size: 15px;
  font-weight: 760;
}

.selected-source-strip p {
  margin: 2px 0 0;
  color: #667085;
  font-size: 13px;
}

.datasource-wizard-form {
  display: grid;
  gap: 18px;
}

.wizard-form-section {
  padding: 16px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fff;
}

.section-divider {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 16px;
  color: #111827;
  font-size: 16px;
  font-weight: 780;
}

.section-divider::after {
  content: '';
  height: 1px;
  flex: 1;
  background: #edf1f7;
}

.wizard-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 18px;
}

.wizard-form-grid .full-row {
  grid-column: 1 / -1;
}

.inline-setting {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  min-height: 32px;
}

.inline-setting span {
  color: #98a2b3;
  font-size: 12px;
}

.headers-editor {
  margin-top: 12px;
}

.mini-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  color: #344054;
  font-size: 14px;
  font-weight: 700;
}

.header-row-list {
  display: grid;
  gap: 10px;
}

.header-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.6fr) 34px;
  gap: 10px;
  align-items: center;
}

.empty-dashed {
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed #d0d5dd;
  border-radius: 8px;
  color: #98a2b3;
  background: #fcfcfd;
  font-size: 13px;
}

.remote-write-section {
  margin: 4px 0 18px;
  padding: 14px 14px 2px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fbfcfe;
}

.remote-write-tip {
  margin-bottom: 14px;
}

.remote-write-url-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  width: 100%;
}

.remote-write-test-btn {
  min-width: 112px;
  border-color: #111827 !important;
  background: #111827 !important;
  color: #fff !important;
}

.remote-write-test-btn:hover,
.remote-write-test-btn:focus {
  border-color: #303133 !important;
  background: #303133 !important;
  color: #fff !important;
}

.remote-write-test-btn :deep(.el-icon) {
  color: inherit;
}

.drawer-section-title {
  margin-bottom: 10px;
  color: #111827;
  font-size: 15px;
  font-weight: 750;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 7px;
  color: #667085;
}

.action-check:hover {
  background: #ecfdf3;
  color: #16a34a;
}

.action-query:hover,
.action-edit:hover {
  background: #eff6ff;
  color: #2563eb;
}

.action-delete:hover {
  background: #fff1f2;
  color: #e11d48;
}

.form-suffix {
  margin-left: 8px;
  color: #667085;
  font-size: 13px;
}

.form-hint {
  margin-left: 10px;
  color: #98a2b3;
  font-size: 12px;
}

.query-source {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 18px;
  color: #344054;
  font-weight: 600;
}

.query-index-select-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  width: 100%;
}

.query-index-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  min-width: 0;
}

.query-index-option span {
  min-width: 0;
  overflow: hidden;
  color: #111827;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.query-index-option em {
  flex-shrink: 0;
  color: #98a2b3;
  font-size: 12px;
  font-style: normal;
}

.query-actions {
  justify-content: flex-end;
  margin-bottom: 12px;
}

.query-empty {
  border: 1px dashed #d8dee9;
  border-radius: 8px;
  background: #fbfcff;
}

.query-result-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.query-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.query-summary-item {
  min-height: 72px;
  padding: 12px 14px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
}

.query-summary-item.success {
  background: #f6fffb;
  border-color: #cdeedd;
}

.query-summary-item span {
  display: block;
  color: #667085;
  font-size: 12px;
}

.query-summary-item strong {
  display: block;
  margin-top: 8px;
  overflow: hidden;
  color: #111827;
  font-size: 18px;
  font-weight: 760;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.query-preview-section {
  overflow: hidden;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
}

.query-preview-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 48px;
  padding: 10px 14px;
  border-bottom: 1px solid #edf1f7;
  background: #fafbfc;
}

.query-preview-head strong {
  display: block;
  color: #111827;
  font-size: 14px;
  font-weight: 760;
}

.query-preview-head span {
  display: block;
  margin-top: 3px;
  color: #667085;
  font-size: 12px;
}

.query-preview-table {
  width: 100%;
}

.query-row-title {
  min-width: 0;
}

.query-row-title strong {
  display: block;
  overflow: hidden;
  color: #111827;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.query-row-title small {
  display: block;
  margin-top: 3px;
  color: #98a2b3;
  font-size: 12px;
}

.query-labels {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 6px;
}

.query-labels span {
  display: inline-flex;
  align-items: flex-start;
  max-width: 100%;
  overflow: hidden;
  border: 1px solid #dbe8ff;
  border-radius: 6px;
  background: #f8fbff;
  color: #344054;
  font-size: 12px;
  line-height: 1.35;
}

.query-labels b {
  flex-shrink: 0;
  padding: 4px 6px;
  border-right: 1px solid #dbe8ff;
  background: #eef4ff;
  color: #1d4ed8;
  font-weight: 700;
}

.query-labels em {
  min-width: 0;
  padding: 4px 6px;
  overflow-wrap: anywhere;
  color: #344054;
  font-style: normal;
}

.query-value-text {
  color: #111827;
  font-weight: 760;
}

.query-log-list {
  display: grid;
  gap: 10px;
  max-height: 420px;
  overflow: auto;
  padding: 12px;
}

.query-log-card {
  padding: 12px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fbfcff;
}

.query-log-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.query-log-head strong {
  min-width: 0;
  overflow: hidden;
  color: #111827;
  font-size: 13px;
  font-weight: 760;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.query-log-head span {
  flex-shrink: 0;
  color: #667085;
  font-size: 12px;
}

.query-log-card pre {
  margin: 10px 0 0;
  padding: 10px 12px;
  overflow: auto;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: #111827;
  color: #e5e7eb;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.query-raw-collapse {
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
}

.query-raw-collapse :deep(.el-collapse-item__header) {
  height: 44px;
  padding: 0 14px;
  border-bottom-color: #edf1f7;
  color: #344054;
  font-weight: 700;
}

.query-raw-collapse :deep(.el-collapse-item__content) {
  padding: 0;
}

.query-result {
  max-height: 360px;
  margin: 0;
  padding: 14px;
  overflow: auto;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #0f172a;
  color: #e5e7eb;
  font-size: 12px;
  line-height: 1.6;
}

.wizard-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.wizard-footer-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

:deep(.datasource-wizard-drawer .el-drawer__header) {
  margin-bottom: 0;
  padding: 18px 24px;
  border-bottom: 1px solid #edf1f7;
}

:deep(.datasource-wizard-drawer .el-drawer__body) {
  padding: 20px 24px 96px;
  background: #fff;
}

:deep(.datasource-wizard-drawer .el-drawer__footer) {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  padding: 14px 24px;
  border-top: 1px solid #edf1f7;
  background: #fff;
}

:deep(.datasource-dialog .el-dialog__header) {
  border-bottom: 1px solid #edf1f7;
  background: #fbfcfe;
}

:deep(.datasource-dialog .el-dialog__footer) {
  border-top: 1px solid #edf1f7;
  background: #fbfcfe;
}

@media (max-width: 900px) {
  .page-header,
  .search-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .stats-cards {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .remote-write-url-row {
    grid-template-columns: 1fr;
  }

  .source-picker-panel,
  .wizard-form-grid {
    grid-template-columns: 1fr;
  }

  .header-row {
    grid-template-columns: 1fr;
  }
}
</style>
