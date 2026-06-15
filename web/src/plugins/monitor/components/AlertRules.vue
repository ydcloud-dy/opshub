<template>
  <div class="monitor-rules-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Warning /></el-icon>
        </div>
        <div>
          <h2 class="page-title">告警规则</h2>
          <p class="page-subtitle">按规则组组织查询条件，并将事件推送到指定故障中心</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="openImportDialog()">
          <el-icon><Upload /></el-icon>
          导入规则
        </el-button>
        <el-button @click="handleExport(false)">
          <el-icon><Download /></el-icon>
          导出
        </el-button>
        <el-button type="primary" @click="handleAdd">
          <el-icon><Plus /></el-icon>
          新增规则
        </el-button>
        <el-button @click="loadAll">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon stat-icon-primary">
          <el-icon><Document /></el-icon>
        </div>
        <div>
          <div class="stat-label">规则总数</div>
          <div class="stat-value">{{ stats.totalRules || ruleData.length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div>
          <div class="stat-label">启用规则</div>
          <div class="stat-value">{{ stats.enabledRules || enabledCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <el-icon><Bell /></el-icon>
        </div>
        <div>
          <div class="stat-label">告警中</div>
          <div class="stat-value">{{ stats.firingRules || firingCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><Timer /></el-icon>
        </div>
        <div>
          <div class="stat-label">今日事件</div>
          <div class="stat-value">{{ stats.todayEvents || 0 }}</div>
        </div>
      </div>
    </div>

    <div class="rule-workspace">
      <aside class="group-sidebar">
        <div class="sidebar-head">
          <span>规则组</span>
          <el-button link type="primary" @click="openGroupCreate">
            <el-icon><Plus /></el-icon>
          </el-button>
        </div>
        <div
          v-for="group in groupFilters"
          :key="group.id || 0"
          role="button"
          tabindex="0"
          class="group-item"
          :class="{ active: selectedGroupId === (group.id || 0) }"
          @click="selectedGroupId = group.id || 0"
          @keydown.enter="selectedGroupId = group.id || 0"
        >
          <span class="group-name">
            <el-icon><Folder /></el-icon>
            <span>{{ group.name }}</span>
          </span>
          <span class="group-tail">
            <em>{{ getGroupCount(group.id || 0) }}</em>
            <el-dropdown
              v-if="Number(group.id) > 0"
              trigger="click"
              placement="bottom-end"
              popper-class="rule-group-dropdown"
              @click.stop
              @command="command => handleRuleGroupCommand(String(command), group)"
            >
              <el-button class="group-more" text :icon="MoreFilled" aria-label="规则组操作" @click.stop />
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">
                    <el-icon><Edit /></el-icon>
                    编辑规则组
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" divided class="danger-item">
                    <el-icon><Delete /></el-icon>
                    删除规则组
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </span>
        </div>
      </aside>

      <main class="rules-main">
        <div class="search-bar">
          <div class="search-inputs">
            <el-input v-model="searchForm.keyword" placeholder="搜索规则名、查询语句、告警详情..." clearable class="search-input">
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
            <el-select v-model="searchForm.dataSourceType" placeholder="数据源类型" clearable class="search-input small">
              <el-option v-for="item in sourceTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-select v-model="searchForm.faultCenterId" placeholder="故障中心" clearable class="search-input">
              <el-option v-for="item in faultCenters" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
            <el-select v-model="searchForm.state" placeholder="状态" clearable class="search-input small">
              <el-option label="正常" value="inactive" />
              <el-option label="预告警" value="pending" />
              <el-option label="告警中" value="firing" />
              <el-option label="评估失败" value="error" />
            </el-select>
          </div>
          <el-button @click="handleReset">
            <el-icon><RefreshLeft /></el-icon>
            重置
          </el-button>
        </div>

        <div class="table-wrapper rule-table-wrapper">
          <div v-if="selectedRuleIds.length" class="batch-toolbar">
            <div class="batch-summary">
              <strong>{{ selectedRuleIds.length }}</strong>
              <span>条规则已选择</span>
            </div>
            <div class="batch-actions">
              <el-button @click="openBatchUpdate">
                <el-icon><Edit /></el-icon>
                批量更新
              </el-button>
              <el-button @click="handleExport(true)">
                <el-icon><Download /></el-icon>
                导出选中
              </el-button>
              <el-button type="danger" plain @click="handleBatchDelete">
                <el-icon><Delete /></el-icon>
                批量删除
              </el-button>
            </div>
          </div>
          <el-table
            :data="pagedRules"
            v-loading="loading"
            class="modern-table"
            :header-cell-style="tableHeaderStyle"
            height="540"
            empty-text="暂无告警规则"
            row-key="id"
            @selection-change="handleSelectionChange"
          >
            <el-table-column type="selection" width="46" fixed="left" />
            <el-table-column label="规则名称" prop="name" min-width="190" show-overflow-tooltip />
            <el-table-column label="规则组" width="140" show-overflow-tooltip>
              <template #default="{ row }">{{ getGroupName(row.ruleGroupId) }}</template>
            </el-table-column>
            <el-table-column label="故障中心" width="160" show-overflow-tooltip>
              <template #default="{ row }">{{ getFaultCenterName(row.faultCenterId) }}</template>
            </el-table-column>
            <el-table-column label="数据源" min-width="190" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="source-cell">
                  <el-tag :type="getTypeTag(row.dataSourceType)" effect="light">{{ getTypeName(row.dataSourceType) }}</el-tag>
                  <span>{{ getDataSourceName(row.dataSourceId) }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="查询语句" prop="query" min-width="280" show-overflow-tooltip />
            <el-table-column label="等级条件" min-width="210">
              <template #default="{ row }">
                <div class="condition-tags">
                  <el-tag v-for="condition in getSeverityRules(row)" :key="`${condition.severity}-${condition.threshold}`" :type="getSeverityTag(condition.severity)" effect="light" size="small">
                    {{ getSeverityName(condition.severity) }} {{ getConditionSymbol(condition.condition) }} {{ formatNumber(condition.threshold) }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="最近值" width="110" align="right">
              <template #default="{ row }">{{ formatNumber(row.lastValue) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="getStateTag(row.lastState)" effect="light">{{ getStateName(row.lastState) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="最后评估" width="170">
              <template #default="{ row }">{{ formatDateTime(row.lastEvalAt) }}</template>
            </el-table-column>
            <el-table-column label="启用" width="86" align="center">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" @change="handleEnabledChange(row)" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="164" fixed="right" align="center">
              <template #default="{ row }">
                <div class="action-buttons">
                  <el-tooltip content="立即评估" placement="top">
                    <el-button link class="action-btn action-check" :loading="row.evaluating" @click="handleEvaluate(row)">
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
          <div class="table-pagination">
            <span>已筛选 {{ filteredRules.length }} 条规则</span>
            <el-pagination
              v-model:current-page="rulePagination.page"
              v-model:page-size="rulePagination.pageSize"
              :page-sizes="[10, 20, 50, 100]"
              :total="filteredRules.length"
              layout="total, sizes, prev, pager, next, jumper"
              background
              @size-change="handleRulePageSizeChange"
              @current-change="handleRulePageChange"
            />
          </div>
        </div>
      </main>
    </div>

    <div class="events-section">
      <div class="section-header">
        <div>
          <h3 class="section-title">近期告警事件</h3>
          <p class="section-subtitle">规则触发、恢复、评估失败都会进入事件流，并按故障中心聚合</p>
        </div>
        <el-button @click="loadEvents">
          <el-icon><Refresh /></el-icon>
          刷新事件
        </el-button>
      </div>
      <el-table :data="eventData" v-loading="eventsLoading" class="modern-table" :header-cell-style="tableHeaderStyle">
        <el-table-column label="规则" prop="ruleName" min-width="180" show-overflow-tooltip />
        <el-table-column label="故障中心" width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ getFaultCenterName(row.faultCenterId) }}</template>
        </el-table-column>
        <el-table-column label="等级" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="getSeverityTag(row.severity)" effect="light">{{ getSeverityName(row.severity) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getStateTag(row.state)" effect="light">{{ getStateName(row.state) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="消息" prop="message" min-width="320" show-overflow-tooltip />
        <el-table-column label="时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.lastEvalAt) }}</template>
        </el-table-column>
      </el-table>
    </div>

    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="90%" class="rule-drawer" :close-on-click-modal="false" @close="handleDrawerClose">
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="rule-drawer-form">
        <div class="rule-drawer-shell">
          <aside class="rule-drawer-nav">
            <a href="#rule-basic">基础配置</a>
            <a href="#rule-source">数据源类型</a>
            <a href="#rule-query">查询与条件</a>
            <a href="#rule-event">评估与事件属性</a>
          </aside>
          <div class="rule-drawer-content">
        <div id="rule-basic" class="drawer-section">
          <div class="drawer-section-title">基础配置</div>
          <div class="form-grid">
            <el-form-item label="规则名称" prop="name">
              <el-input v-model="form.name" placeholder="如：Nginx 连接数过高" />
            </el-form-item>
            <el-form-item label="规则组" prop="ruleGroupId">
              <el-select v-model="form.ruleGroupId" placeholder="请选择规则组" style="width: 100%">
                <el-option v-for="group in ruleGroups" :key="group.id" :label="group.name" :value="group.id" />
              </el-select>
            </el-form-item>
          </div>
          <el-form-item label="故障中心" prop="faultCenterId">
            <el-select v-model="form.faultCenterId" placeholder="事件推送到 WatchAlert 风格故障中心" style="width: 100%">
              <el-option v-for="item in faultCenters" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
        </div>

        <div id="rule-source" class="drawer-section">
          <div class="drawer-section-title">数据源类型</div>
          <div class="source-type-grid">
            <div
              v-for="item in sourceTypeOptions"
              :key="item.value"
              role="button"
              tabindex="0"
              class="source-type-card"
              :class="{ active: form.dataSourceType === item.value }"
              @click="selectSourceType(item.value)"
              @keydown.enter="selectSourceType(item.value)"
            >
              <span class="source-type-logo" :class="`logo-${item.value}`">
                <svg v-if="item.value === 'prometheus'" viewBox="0 0 48 48" aria-hidden="true">
                  <circle cx="24" cy="24" r="20" fill="#e6522c" />
                  <path fill="#fff" d="M24 10c2.7 3.2 4.2 6.5 4.2 10.1 0 2.1-.6 4-1.7 5.7 2.7-1.5 4.4-3.6 5.4-6.3 2.2 4.4 1.6 10.2-2.4 13.7h3.7v3.4H14.8v-3.4h3.7c-4-3.5-4.6-9.3-2.4-13.7 1 2.7 2.7 4.8 5.4 6.3-1.1-1.7-1.7-3.6-1.7-5.7C19.8 16.5 21.3 13.2 24 10Z" />
                  <path fill="#e6522c" d="M24 22.2c1.5 2 2.4 3.9 2.4 5.8a2.4 2.4 0 1 1-4.8 0c0-1.9.9-3.8 2.4-5.8Z" />
                </svg>
                <svg v-else-if="item.value === 'loki'" viewBox="0 0 48 48" aria-hidden="true">
                  <rect x="10" y="8" width="8" height="32" rx="3" fill="#f7c948" />
                  <path d="M18 38c9-2 14-8 17-17" fill="none" stroke="#0f766e" stroke-width="6" stroke-linecap="round" />
                  <path d="M24 14l10-5" fill="none" stroke="#ef4444" stroke-width="4" stroke-linecap="round" />
                </svg>
                <svg v-else-if="item.value === 'victoriametrics'" viewBox="0 0 48 48" aria-hidden="true">
                  <rect x="9" y="10" width="6" height="28" fill="#111827" rx="2" />
                  <rect x="19" y="10" width="6" height="28" fill="#111827" rx="2" />
                  <rect x="29" y="10" width="6" height="28" fill="#111827" rx="2" />
                  <rect x="38" y="25" width="4" height="13" fill="#f59e0b" rx="1" />
                  <rect x="9" y="34" width="26" height="4" fill="#ef4444" />
                </svg>
                <svg v-else-if="item.value === 'elasticsearch'" viewBox="0 0 48 48" aria-hidden="true">
                  <path fill="#fbbf24" d="M11 19a13 13 0 0 1 25-5H17a6 6 0 0 0-6 5Z" />
                  <path fill="#14b8a6" d="M36 34a13 13 0 0 1-25-5h19a6 6 0 0 0 6 5Z" />
                  <path fill="#111827" d="M11 21h26a4 4 0 0 1 0 8H11a14 14 0 0 1 0-8Z" />
                </svg>
              </span>
              <span class="source-type-copy">
                <strong>{{ item.label }}</strong>
                <em>{{ item.desc }}</em>
              </span>
              <el-icon class="source-type-check"><CircleCheck /></el-icon>
            </div>
          </div>
          <el-form-item label="数据源" prop="dataSourceIds" class="source-select">
            <el-select v-model="form.dataSourceIds" multiple filterable placeholder="可选择多个同类型数据源，当前评估优先使用第一个" style="width: 100%" @change="handleSourceChange">
              <el-option v-for="item in filteredDataSourcesForForm" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
        </div>

        <div id="rule-query" class="drawer-section">
          <div class="drawer-section-title">查询与条件</div>
          <el-form-item v-if="form.dataSourceType === 'elasticsearch'" label="索引">
            <div class="es-index-select-row">
              <el-select
                v-model="form.index"
                filterable
                allow-create
                default-first-option
                clearable
                remote
                reserve-keyword
                :remote-method="loadElasticsearchIndices"
                :loading="esIndexLoading"
                placeholder="选择索引 / 别名，也可输入 logs-*"
                @visible-change="handleEsIndexDropdownVisible"
              >
                <el-option
                  v-for="item in esIndexOptions"
                  :key="`${item.type}-${item.name}`"
                  :label="item.name"
                  :value="item.name"
                >
                  <div class="es-index-option">
                    <span>{{ item.name }}</span>
                    <em>{{ item.type === 'alias' ? 'alias' : item.status || 'index' }}{{ item.docsCount ? ` · ${item.docsCount} docs` : '' }}</em>
                  </div>
                </el-option>
              </el-select>
              <el-button :loading="esIndexLoading" @click="loadElasticsearchIndices()">
                <el-icon><Refresh /></el-icon>
                刷新索引
              </el-button>
            </div>
          </el-form-item>
          <el-form-item v-else label="查询模式">
            <el-segmented v-model="form.queryMode" :options="queryModeOptions" />
          </el-form-item>
          <el-form-item :label="form.dataSourceType === 'elasticsearch' ? '查询 DSL' : '查询语句'" prop="query">
            <div class="query-editor">
              <div v-if="form.dataSourceType === 'elasticsearch'" class="es-dsl-helper">
                <div class="es-dsl-helper-head">
                  <span>常用 ES 告警模板</span>
                  <em>ES 告警值会优先取 aggregations 里的第一个数值，没有聚合时取 hits.total.value</em>
                </div>
                <div class="es-dsl-template-row">
                  <el-tooltip
                    v-for="item in elasticsearchDslTemplates"
                    :key="item.key"
                    :content="item.description"
                    placement="top"
                    effect="light"
                  >
                    <el-button @click="applyElasticsearchDslTemplate(item)">
                      {{ item.name }}
                    </el-button>
                  </el-tooltip>
                </div>
              </div>
              <el-input
                v-model="form.query"
                type="textarea"
                :rows="6"
                :placeholder="getQueryPlaceholder(form.dataSourceType)"
                @input="handleQueryInput"
                @focus="handleQueryInput"
              />
              <div class="query-tools">
                <div class="query-source-hint">
                  <el-tag v-if="getFormPrimarySource()" :type="getTypeTag(getFormPrimarySource()?.type)" effect="light">
                    {{ getTypeName(getFormPrimarySource()?.type) }}
                  </el-tag>
                  <span>{{ getFormPrimarySource()?.name || '请选择数据源后预览' }}</span>
                </div>
                <el-button class="preview-query-btn" :loading="previewLoading" @click="handlePreviewQuery">
                  <el-icon><DataAnalysis /></el-icon>
                  数据预览
                </el-button>
              </div>
              <div v-if="suggestionVisible" class="suggestion-popover">
                <div v-for="item in suggestions" :key="`${item.kind}-${item.value}`" role="button" tabindex="0" class="suggestion-item" @mousedown.prevent="insertSuggestion(item)" @keydown.enter="insertSuggestion(item)">
                  <span>
                    <strong>{{ item.value }}</strong>
                    <em>{{ item.kind }} · {{ item.type || '-' }}</em>
                  </span>
                  <small>{{ item.help || '无说明' }}</small>
                </div>
              </div>
            </div>
          </el-form-item>

          <div class="condition-editor">
            <div class="condition-head">
              <span>等级条件</span>
              <el-button link type="primary" @click="addCondition">
                <el-icon><Plus /></el-icon>
                添加条件
              </el-button>
            </div>
            <div class="condition-grid-head">
              <span>等级</span>
              <span>判断</span>
              <span>阈值</span>
              <span>持续时间</span>
              <span>操作</span>
            </div>
            <div v-for="(condition, index) in form.severityRules" :key="index" class="condition-row">
              <el-select v-model="condition.severity" class="condition-severity">
                <el-option label="P0" value="p0" />
                <el-option label="P1" value="p1" />
                <el-option label="P2" value="p2" />
              </el-select>
              <el-select v-model="condition.condition" class="condition-op">
                <el-option label=">" value="gt" />
                <el-option label=">=" value="gte" />
                <el-option label="<" value="lt" />
                <el-option label="<=" value="lte" />
                <el-option label="=" value="eq" />
                <el-option label="!=" value="neq" />
              </el-select>
              <el-input-number v-model="condition.threshold" :precision="2" :step="1" class="condition-number" />
              <div class="condition-duration">
                <el-input-number v-model="condition.forSeconds" :min="0" :max="86400" class="condition-number" />
                <span class="condition-unit">秒</span>
              </div>
              <div class="condition-actions">
                <el-button link class="action-btn action-delete" :disabled="form.severityRules.length === 1" @click="removeCondition(index)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
            </div>
          </div>
        </div>

        <div id="rule-event" class="drawer-section">
          <div class="drawer-section-title">评估与事件属性</div>
          <div class="form-grid">
            <el-form-item label="评估间隔">
              <div class="inline-number-field">
                <el-input-number v-model="form.evaluateInterval" :min="10" :max="86400" />
                <span class="form-suffix">秒</span>
              </div>
            </el-form-item>
            <el-form-item label="兜底重复通知">
              <div class="inline-number-field">
                <el-input-number v-model="form.repeatInterval" :min="60" :max="86400" />
                <span class="form-suffix">秒</span>
              </div>
            </el-form-item>
          </div>
          <div class="notice-target-tip">
            <el-icon><Bell /></el-icon>
            <div>
              <strong>通知目标由故障中心管理</strong>
              <span>规则只负责评估和产生事件，重复通知优先使用故障中心 P0/P1/P2 配置；这里仅在没有故障中心策略时兜底。</span>
            </div>
          </div>
          <div class="form-grid">
            <el-form-item label="恢复通知">
              <el-switch v-model="form.notifyRecovery" />
            </el-form-item>
            <el-form-item label="启用状态">
              <el-switch v-model="form.enabled" />
            </el-form-item>
          </div>
          <div class="event-field-grid">
            <div class="kv-editor">
              <div class="kv-head">
                <span>标签 <small>用于分组、搜索、路由和静默匹配</small></span>
                <el-button link type="primary" @click="addKeyValueRow(form.labels)">
                  <el-icon><Plus /></el-icon>
                  添加标签
                </el-button>
              </div>
              <div v-for="(item, index) in form.labels" :key="`label-${index}`" class="kv-row">
                <el-input v-model="item.key" placeholder="team" />
                <el-input v-model="item.value" placeholder="ops" />
                <el-button link class="action-btn action-delete" @click="removeKeyValueRow(form.labels, index)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
              <el-empty v-if="!form.labels.length" :image-size="48" description="暂无标签" />
            </div>

            <div class="detail-template-editor">
              <div class="kv-head">
                <span class="required-title">告警详情 <small>触发后写入故障中心事件详情，并作为通知内容里的告警事件</small></span>
              </div>
              <el-form-item prop="detailTemplate" label-width="0" class="detail-template-form-item">
                <el-input
                  v-model="form.detailTemplate"
                  type="textarea"
                  :rows="5"
                  maxlength="2000"
                  show-word-limit
                  placeholder="例如：节点：${labels.instance}，CPU 使用率过高，当前：${value}%，阈值：${threshold}%，请及时处理。"
                />
              </el-form-item>
              <div class="template-token-row">
                <el-tooltip
                  v-for="item in availableDetailTemplateTokens"
                  :key="item.token"
                  :content="item.description"
                  placement="top"
                  effect="light"
                >
                  <span class="template-token-pill">
                    <code>{{ item.token }}</code>
                    <em>{{ item.label }}</em>
                  </span>
                </el-tooltip>
              </div>
              <div class="detail-preview-box">
                <div class="detail-preview-head">
                  <span>告警详情预览</span>
                  <el-tag size="small" :type="detailPreviewSourceType" effect="light">{{ detailPreviewSourceText }}</el-tag>
                </div>
                <div class="detail-preview-body">
                  <template v-for="(block, index) in detailPreviewBlocks" :key="`${block.type}-${index}`">
                    <pre v-if="block.type === 'code'" class="detail-preview-code">{{ block.content }}</pre>
                    <pre v-else class="detail-preview-text">{{ block.content }}</pre>
                  </template>
                </div>
              </div>
            </div>
          </div>

          <div class="callback-editor">
            <div class="kv-head">
              <span>回调查询 <small>告警触发后会执行并进入通知上下文，也会在故障中心详情中展示；可用于 CPU 曲线、关联日志等排障数据</small></span>
              <el-button link type="primary" @click="addCallbackQuery">
                <el-icon><Plus /></el-icon>
                添加查询
              </el-button>
            </div>
            <div v-for="(item, index) in form.callbackQueries" :key="`callback-${index}`" class="callback-row">
              <el-input v-model="item.key" placeholder="查询名称，如 CPU 曲线" />
              <el-select v-model="item.dataSourceId" clearable filterable placeholder="默认规则数据源">
                <el-option v-for="source in dataSources" :key="source.id" :label="source.name" :value="source.id" />
              </el-select>
              <el-select v-model="item.queryMode">
                <el-option label="即时查询" value="instant" />
                <el-option label="范围查询" value="range" />
              </el-select>
              <div class="callback-range-field">
                <el-input-number v-model="item.rangeSeconds" :min="60" :max="86400" :step="60" controls-position="right" />
                <span>秒</span>
              </div>
              <el-input v-model="item.query" class="callback-query-input" placeholder='例如：100 - avg(rate(node_cpu_seconds_total{mode="idle",instance="${labels.instance}"}[5m])) * 100' />
              <el-button link class="action-btn action-delete" @click="removeCallbackQuery(index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            <el-empty v-if="!form.callbackQueries.length" :image-size="48" description="暂无回调查询" />
          </div>
        </div>
          </div>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-drawer>

    <el-dialog v-model="groupDialogVisible" :title="groupDialogTitle" width="520px" class="group-dialog" @closed="resetGroupForm">
      <el-form :model="groupForm" label-width="96px">
        <el-form-item label="名称">
          <el-input v-model="groupForm.name" placeholder="请输入规则组名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="groupForm.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitGroup">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="batchDialogVisible" title="批量更新规则" width="680px" class="rule-dialog batch-dialog" :close-on-click-modal="false">
      <div class="batch-dialog-tip">
        已选择 {{ selectedRuleIds.length }} 条规则。勾选需要变更的字段，未勾选的配置会保持原样。
      </div>
      <el-form label-width="104px" class="batch-form">
        <div class="batch-field">
          <el-checkbox v-model="batchForm.updateRuleGroup">规则组</el-checkbox>
          <el-select v-model="batchForm.ruleGroupId" :disabled="!batchForm.updateRuleGroup" placeholder="选择新的规则组">
            <el-option v-for="group in ruleGroups" :key="group.id" :label="group.name" :value="group.id" />
          </el-select>
        </div>
        <div class="batch-field">
          <el-checkbox v-model="batchForm.updateDataSource">数据源</el-checkbox>
          <el-select v-model="batchForm.dataSourceId" :disabled="!batchForm.updateDataSource" filterable placeholder="选择新的数据源">
            <el-option v-for="source in dataSources" :key="source.id" :label="`${source.name} · ${getTypeName(source.type)}`" :value="source.id" />
          </el-select>
        </div>
        <div class="batch-field">
          <el-checkbox v-model="batchForm.updateFaultCenter">故障中心</el-checkbox>
          <el-select v-model="batchForm.faultCenterId" :disabled="!batchForm.updateFaultCenter" placeholder="选择新的故障中心">
            <el-option v-for="center in faultCenters" :key="center.id" :label="center.name" :value="center.id" />
          </el-select>
        </div>
        <div class="batch-field">
          <el-checkbox v-model="batchForm.updateEnabled">规则状态</el-checkbox>
          <el-select v-model="batchForm.enabled" :disabled="!batchForm.updateEnabled" placeholder="选择状态">
            <el-option label="启用" :value="true" />
            <el-option label="关闭" :value="false" />
          </el-select>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="batchDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="batchSubmitting" @click="submitBatchUpdate">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="importDialogVisible" :title="importDialogTitle" width="860px" class="rule-dialog import-dialog" :close-on-click-modal="false">
      <div class="import-mode-tabs">
        <button type="button" :class="{ active: importForm.mode === 'json' }" @click="setImportMode('json')">OpsHub / WatchAlert JSON</button>
        <button type="button" :class="{ active: importForm.mode === 'prometheusRule' }" @click="setImportMode('prometheusRule')">PrometheusRule YAML</button>
      </div>
      <div class="import-tip">
        {{ importTipText }}
      </div>
      <el-form label-width="108px" class="import-form">
        <div class="form-grid">
          <el-form-item label="数据源" required>
            <el-select v-model="importForm.dataSourceId" filterable placeholder="选择导入后关联的数据源" style="width: 100%">
              <el-option v-for="source in currentImportDataSources" :key="source.id" :label="`${source.name} · ${getTypeName(source.type)}`" :value="source.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="默认等级">
            <el-select v-model="importForm.defaultSeverity" style="width: 100%">
              <el-option label="P0" value="p0" />
              <el-option label="P1" value="p1" />
              <el-option label="P2" value="p2" />
            </el-select>
          </el-form-item>
          <el-form-item label="规则组">
            <el-select v-model="importForm.ruleGroupId" clearable placeholder="不选则使用默认规则组" style="width: 100%">
              <el-option v-for="group in ruleGroups" :key="group.id" :label="group.name" :value="group.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="故障中心">
            <el-select v-model="importForm.faultCenterId" clearable placeholder="不选则使用默认故障中心" style="width: 100%">
              <el-option v-for="center in faultCenters" :key="center.id" :label="center.name" :value="center.id" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item :label="importContentLabel" required>
          <el-input
            v-model="importForm.content"
            type="textarea"
            :rows="14"
            spellcheck="false"
            :placeholder="importPlaceholder"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="importDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="importSubmitting" @click="submitAlertRuleImport">导入</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="evalDialogVisible" title="评估结果" width="1180px" class="rule-dialog eval-dialog">
      <div v-if="evalResult" class="eval-result-panel">
        <section class="eval-summary-card">
          <div class="eval-summary-main">
            <el-tag :type="getStateTag(evalResult.state)" effect="dark">{{ getStateName(evalResult.state) }}</el-tag>
            <h3>{{ evalResult.ruleName || '-' }}</h3>
            <p>{{ evalResult.message || '本次评估已完成' }}</p>
          </div>
          <div class="eval-summary-value">
            <span>当前值</span>
            <strong>{{ formatNumber(evalResult.value) }}</strong>
          </div>
        </section>

        <div class="eval-meta-grid">
          <div>
            <span>数据源</span>
            <strong>{{ evalResult.dataSourceName || '-' }}</strong>
            <small>{{ getTypeName(evalResult.dataSourceType) }}</small>
          </div>
          <div>
            <span>评估时间</span>
            <strong>{{ formatDateTime(evalResult.evaluatedAt) }}</strong>
            <small>HTTP {{ evalResult.statusCode || '-' }}</small>
          </div>
          <div>
            <span>状态变化</span>
            <strong>{{ getStateName(evalResult.previousState) }} → {{ getStateName(evalResult.state) }}</strong>
            <small>{{ evalResult.matched ? '命中告警条件' : '未命中告警条件' }}</small>
          </div>
          <div>
            <span>判断条件</span>
            <strong>{{ getSeverityName(evalResult.severity) }} {{ getConditionSymbol(evalResult.condition) }} {{ formatNumber(evalResult.threshold) }}</strong>
            <small>持续 {{ formatEvalForSeconds(evalResult.forSeconds) }}</small>
          </div>
        </div>

        <div class="eval-samples-head">
          <div>
            <strong>样本明细</strong>
            <span>{{ evalSamples.length }} 条结果，{{ evalMatchedCount }} 条命中</span>
          </div>
        </div>
        <div v-if="evalSamples.length" class="eval-sample-list">
          <article
            v-for="(row, index) in evalSamples"
            :key="row.fingerprint || `${row.value}-${index}`"
            class="eval-sample-card"
            :class="{ 'is-matched': row.matched }"
          >
            <div class="eval-sample-main">
              <div class="eval-sample-title">
                <span class="sample-index">#{{ index + 1 }}</span>
                <el-tag :type="getStateTag(row.state)" effect="light">{{ getStateName(row.state) }}</el-tag>
                <el-tag :type="getSeverityTag(row.severity)" effect="plain">{{ getSeverityName(row.severity) }}</el-tag>
                <strong>{{ formatNumber(row.value) }}</strong>
              </div>
              <div class="eval-sample-condition">
                <span>条件</span>
                <b>{{ getConditionSymbol(row.condition) }} {{ formatNumber(row.threshold) }}</b>
                <em>{{ formatEvalForSeconds(row.forSeconds) }}</em>
              </div>
            </div>
            <div class="eval-sample-labels">
              <span class="sample-section-title">标签</span>
              <div v-if="getEvalLabelEntries(row.labels).length" class="eval-labels">
                <span
                  v-for="item in getEvalLabelEntries(row.labels)"
                  :key="`${row.fingerprint || index}-${item.key}`"
                  class="eval-label-chip"
                  :title="`${item.key}=${item.value}`"
                >
                  <b>{{ item.key }}</b>
                  <em>{{ item.value }}</em>
                </span>
              </div>
              <span v-else class="eval-empty-text">无标签</span>
            </div>
            <div class="eval-sample-message">
              <span class="sample-section-title">说明</span>
              <p>{{ row.message || '-' }}</p>
            </div>
            <div v-if="row.matchedLogs?.length" class="eval-sample-logs">
              <span class="sample-section-title">命中日志</span>
              <pre>{{ formatMatchedLogs(row.matchedLogs, row.matchedLogCount) }}</pre>
            </div>
          </article>
        </div>
        <el-empty v-else :image-size="72" description="本次评估没有返回可展示样本" />
      </div>
      <template #footer>
        <el-button @click="evalDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="previewDialogVisible"
      width="980px"
      class="rule-dialog data-preview-dialog"
      :close-on-click-modal="false"
      @closed="disposePreviewChart"
    >
      <template #header>
        <div class="preview-dialog-head">
          <div class="preview-title-icon">
            <el-icon><DataAnalysis /></el-icon>
          </div>
          <div>
            <h3>{{ getTypeName(previewSource?.type) }} 数据预览</h3>
            <p>{{ previewSource?.name || '-' }} · {{ previewQueryText || '-' }}</p>
          </div>
        </div>
      </template>

      <div class="preview-meta">
        <div>
          <span>数据源</span>
          <strong>{{ previewSource?.name || '-' }}</strong>
        </div>
        <div>
          <span>查询模式</span>
          <strong>{{ previewQueryModeText }}</strong>
        </div>
        <div>
          <span>响应耗时</span>
          <strong>{{ previewResult?.duration ?? '-' }}ms</strong>
        </div>
        <div>
          <span>返回条数</span>
          <strong>{{ previewItems.length }}</strong>
        </div>
      </div>

      <el-alert v-if="previewError" class="preview-alert" type="error" :closable="false" :title="previewError" />

      <el-tabs v-model="previewTab" class="preview-tabs" @tab-change="handlePreviewTabChange">
        <el-tab-pane name="card">
          <template #label>
            <span class="tab-label"><el-icon><Grid /></el-icon>Card</span>
          </template>
          <div v-loading="previewLoading" class="preview-tab-panel">
            <div v-if="previewItems.length" class="preview-card-list">
              <div v-for="(item, index) in previewItems" :key="`${item.title}-${index}`" class="preview-card">
                <div class="preview-card-left">
                  <div class="preview-card-title">
                    <el-icon><Histogram /></el-icon>
                    <strong>{{ formatMetricLabelSet(item.labels, item.title) }}</strong>
                  </div>
                  <div class="preview-labels">
                    <span v-for="label in item.labels" :key="`${item.title}-${label.key}-${label.value}`">
                      <b>{{ label.key }}:</b> {{ label.value }}
                    </span>
                    <em v-if="!item.labels.length">暂无标签</em>
                  </div>
                </div>
                <div class="preview-card-value">
                  <span>{{ item.kind === 'log' ? '日志' : item.kind === 'document' ? '文档' : '数值' }}</span>
                  <strong :title="item.valueText">{{ item.valueText }}</strong>
                  <small>{{ item.timestamp || previewFetchedAtText }}</small>
                </div>
              </div>
            </div>
            <el-empty v-else :image-size="86" description="暂无预览数据" />
          </div>
        </el-tab-pane>
        <el-tab-pane name="graph">
          <template #label>
            <span class="tab-label"><el-icon><TrendCharts /></el-icon>Graph</span>
          </template>
          <div v-loading="previewLoading" class="preview-tab-panel">
            <div v-if="hasPreviewChartData" ref="previewChartRef" class="preview-chart"></div>
            <el-empty v-else :image-size="86" description="当前结果没有可绘制的数值数据" />
            <div v-if="previewGraphItems.length" class="preview-series-table">
              <div class="preview-series-head">
                <span>标签</span>
                <span>数值</span>
              </div>
              <div v-for="(item, index) in previewGraphItems" :key="`${item.title}-${index}`" class="preview-series-row">
                <div class="preview-series-labels">
                  <span v-for="label in item.labels" :key="`${item.title}-${label.key}-${label.value}`">
                    <b>{{ label.key }}:</b> {{ label.value }}
                  </span>
                  <em v-if="!item.labels.length">{{ item.title }}</em>
                </div>
                <div class="preview-series-value">
                  <strong>{{ item.valueText }}</strong>
                  <small>{{ item.timestamp || previewFetchedAtText }}</small>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <template #footer>
        <el-button @click="previewDialogVisible = false">关闭</el-button>
        <el-button type="primary" :loading="previewLoading" @click="handlePreviewQuery">重新查询</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import * as echarts from 'echarts'
import {
  Bell,
	CircleCheck,
	DataAnalysis,
	Delete,
	Document,
	Download,
	Edit,
  Folder,
  Grid,
  Histogram,
  MoreFilled,
  Plus,
  Refresh,
  RefreshLeft,
  Search,
  Timer,
  TrendCharts,
  Upload,
  View,
  Warning
} from '@element-plus/icons-vue'
import {
  batchDeleteMonitorAlertRules,
  batchUpdateMonitorAlertRules,
  createMonitorAlertRule,
  createMonitorRuleGroup,
  deleteMonitorAlertRule,
  deleteMonitorRuleGroup,
  evaluateMonitorAlertRule,
  exportMonitorAlertRules,
  getMonitorAlertEventStats,
  getMonitorAlertEvents,
  getMonitorAlertRules,
  getMonitorDataSourceIndices,
  getMonitorDataSourceSuggestions,
  getMonitorDataSources,
  getMonitorFaultCenters,
  getMonitorRuleGroups,
  importMonitorAlertRules,
  importPrometheusRuleYaml,
  queryMonitorDataSource,
  updateMonitorAlertRule,
  updateMonitorRuleGroup,
  type DataSourceType,
  type DataSourceQueryRequest,
  type MonitorAlertEvent,
  type MonitorAlertEventStats,
  type MonitorAlertRule,
  type MonitorDataSourceIndex,
  type MonitorDataSource,
  type MonitorFaultCenter,
  type MonitorQuerySuggestion,
  type MonitorRuleGroup
} from '@/api/monitor-datasource'

type ConditionOperator = 'gt' | 'gte' | 'lt' | 'lte' | 'eq' | 'neq'
type SeverityValue = 'p0' | 'p1' | 'p2'
type ImportMode = 'json' | 'prometheusRule'
type RuleRow = MonitorAlertRule & { evaluating?: boolean }

interface SeverityRule {
  severity: SeverityValue
  condition: ConditionOperator
  threshold: number
  forSeconds: number
}

interface KeyValueRow {
  key: string
  value: string
}

interface CallbackQueryRow {
  key: string
  query: string
  dataSourceId?: number
  queryMode: 'instant' | 'range'
  rangeSeconds: number
}

interface RuleForm {
  id?: number
  name: string
  ruleGroupId?: number
  faultCenterId?: number
  dataSourceType: DataSourceType
  dataSourceIds: number[]
  query: string
  queryMode: 'instant' | 'range'
  index: string
  severityRules: SeverityRule[]
  evaluateInterval: number
  enabled: boolean
  channelIds: number[]
  notifyRecovery: boolean
  repeatInterval: number
  labels: KeyValueRow[]
  annotations: KeyValueRow[]
  detailTemplate: string
  callbackQueries: CallbackQueryRow[]
}

interface PreviewLabel {
  key: string
  value: string
}

interface PreviewPoint {
  time: string
  tooltipTime?: string
  axisTime?: string
  timestamp?: number
  value: number
}

interface PreviewItem {
  title: string
  valueText: string
  numericValue?: number
  timestamp?: string
  labels: PreviewLabel[]
  points: PreviewPoint[]
  kind: 'metric' | 'log' | 'document'
}

interface QueryPreviewResult {
  statusCode?: number
  duration?: number
  result?: any
}

interface LokiMatchedLogPreview {
  query: string
  logs: MatchedLogEntry[]
  count: number
}

interface EvaluationSampleResult {
  fingerprint?: string
  matched: boolean
  state: string
  value: number
  severity: string
  condition: ConditionOperator
  threshold: number
  forSeconds: number
  labels?: Record<string, string>
  matchedLogs?: MatchedLogEntry[]
  matchedLogCount?: number
  matchedLogQuery?: string
  message: string
}

interface MatchedLogEntry {
  timestamp?: string
  line: string
  labels?: Record<string, string>
}

interface DetailTemplateToken {
  token: string
  label: string
  description: string
  sourceTypes?: DataSourceType[]
}

interface DetailPreviewContext {
  source: 'evaluation' | 'preview' | 'empty'
  labels: Record<string, string>
  value: string
  condition: string
  threshold: string
  severity: string
  state: string
  fingerprint: string
  matchedLogQuery: string
  matchedLogCount: string
  matchedLogs: string
  message: string
  dataSourceName: string
  dataSourceType: string
}

interface DetailPreviewBlock {
  type: 'text' | 'code'
  content: string
}

interface ElasticsearchDslTemplate {
  key: string
  name: string
  description: string
  build: () => string
}

interface EvaluationResult {
  ruleId: number
  ruleName: string
  dataSourceName: string
  dataSourceType: DataSourceType
  severity: string
  state: string
  previousState: string
  matched: boolean
  value: number
  condition: ConditionOperator
  threshold: number
  forSeconds: number
  labels?: Record<string, string>
  fingerprint?: string
  message: string
  statusCode: number
  evaluatedAt: string
  samples?: EvaluationSampleResult[]
}

const tableHeaderStyle = { background: '#fafbfc', color: '#606266', fontWeight: '600' }
const baseDetailTemplateTokens: DetailTemplateToken[] = [
  {
    token: '${value}',
    label: '当前值',
    description: '当前查询返回并参与判断的数值；需要评估或数据预览后才会有真实值。'
  },
  {
    token: '${condition}',
    label: '判断符',
    description: '规则判断条件，会渲染成 >、>=、<、<=、== 或 !=。'
  },
  {
    token: '${threshold}',
    label: '阈值',
    description: '触发告警的阈值，和判断符一起组成触发条件。'
  },
  {
    token: '${severity}',
    label: '告警等级',
    description: '当前触发的告警等级，只会是 P0、P1 或 P2。'
  },
  {
    token: '${fingerprint}',
    label: '告警指纹',
    description: '事件去重和聚合使用的唯一指纹，同一规则同一标签集会生成相同指纹。'
  }
]

const lokiDetailTemplateTokens: DetailTemplateToken[] = [
  {
    token: '${matched_log_query}',
    label: '日志查询',
    description: 'Loki 日志告警中用于反查原始命中日志的查询语句。聚合语句会自动提取原始日志查询。',
    sourceTypes: ['loki']
  },
  {
    token: '${matched_log_count}',
    label: '日志条数',
    description: 'Loki 本次命中的日志条数；如果只展示部分日志，这里仍表示总命中数。',
    sourceTypes: ['loki']
  },
  {
    token: '${matched_logs}',
    label: '命中日志',
    description: 'Loki 最近命中的日志正文，适合普通文本展示；通知里会尽量自动转成代码块。',
    sourceTypes: ['loki']
  },
  {
    token: '${matched_logs_block}',
    label: '日志代码块',
    description: 'Loki 最近命中日志的 Markdown 代码块版本，推荐用于飞书、钉钉、企微通知。',
    sourceTypes: ['loki']
  }
]

const commonLabelTokenDescriptions: Record<string, string> = {
  instance: '当前命中样本的 instance 标签，常用于主机、服务实例或采集目标。',
  job: '当前命中样本的 job 标签，常用于 Prometheus 采集任务或服务分组。',
  pod: '当前命中样本的 pod 标签，Loki / Kubernetes 日志告警常用。',
  namespace: '当前命中样本的 namespace 标签，常用于 Kubernetes 命名空间。',
  container: '当前命中样本的 container 标签，常用于容器维度定位。',
  app: '当前命中样本的 app 标签，常用于应用或服务名称。',
  service: '当前命中样本的 service 标签，常用于服务维度定位。',
  endpoint: '当前命中样本的 endpoint 标签，常用于接口或探测目标。',
  host: '当前命中样本的 host 标签，常用于主机名或日志来源。',
  level: '当前命中样本的 level 标签，常用于日志级别。'
}

const fallbackLabelKeysBySource: Record<string, string[]> = {
  prometheus: ['instance', 'job'],
  victoriametrics: ['instance', 'job'],
  loki: ['pod', 'app', 'namespace', 'container', 'instance'],
  elasticsearch: ['index', 'service', 'host', 'level']
}

const logDataSourceTypes: DataSourceType[] = ['loki']
const lokiMatchedLogsPendingText = '请先点击“数据预览”，系统会用上面的日志查询去 Loki 拉取最近命中日志。'
const lokiMatchedLogsEmptyText = '当前查询窗口内未查询到命中日志；请确认 Loki 中对应时间范围有匹配日志，或放宽 LogQL / 时间范围。'
const maxLokiPreviewLookbackSeconds = 7 * 24 * 60 * 60

const createLabelToken = (key: string): DetailTemplateToken => ({
  token: `\${labels.${key}}`,
  label: `${key} 标签`,
  description: commonLabelTokenDescriptions[key] || `当前命中样本的 ${key} 标签；只有查询结果或数据预览里存在该标签时才会渲染。`
})
const loading = ref(false)
const eventsLoading = ref(false)
const submitting = ref(false)
const previewLoading = ref(false)
const drawerVisible = ref(false)
const groupDialogVisible = ref(false)
const evalDialogVisible = ref(false)
const previewDialogVisible = ref(false)
const batchDialogVisible = ref(false)
const importDialogVisible = ref(false)
const drawerTitle = ref('新增规则')
const selectedGroupId = ref(0)
const groupEditingId = ref<number>()
const formRef = ref<FormInstance>()
const evalResult = ref<EvaluationResult | null>(null)
const suggestions = ref<MonitorQuerySuggestion[]>([])
const suggestionVisible = ref(false)
const esIndexOptions = ref<MonitorDataSourceIndex[]>([])
const esIndexLoading = ref(false)
const previewTab = ref<'card' | 'graph'>('card')
const previewSource = ref<MonitorDataSource>()
const previewResult = ref<QueryPreviewResult>()
const previewGraphResult = ref<QueryPreviewResult>()
const previewMatchedLog = ref<LokiMatchedLogPreview | null>(null)
const previewMatchedLogSignature = ref('')
const previewQueryText = ref('')
const previewQueryMode = ref<'instant' | 'range' | 'dsl'>('instant')
const previewFetchedAt = ref<Date>()
const previewError = ref('')
const previewSignature = ref('')
const evalResultSignature = ref('')
const previewChartRef = ref<HTMLElement>()
let suggestionTimer: ReturnType<typeof setTimeout> | undefined
let previewChart: echarts.ECharts | null = null

const ruleData = ref<RuleRow[]>([])
const selectedRuleRows = ref<RuleRow[]>([])
const eventData = ref<MonitorAlertEvent[]>([])
const dataSources = ref<MonitorDataSource[]>([])
const faultCenters = ref<MonitorFaultCenter[]>([])
const ruleGroups = ref<MonitorRuleGroup[]>([])
const stats = ref<MonitorAlertEventStats>({
  totalRules: 0,
  enabledRules: 0,
  firingRules: 0,
  pendingRules: 0,
  todayEvents: 0,
  unresolvedEvents: 0
})

const searchForm = reactive({
  keyword: '',
  dataSourceType: '' as DataSourceType | '',
  faultCenterId: undefined as number | undefined,
  state: ''
})

const rulePagination = reactive({
  page: 1,
  pageSize: 10
})

const groupForm = reactive({
  name: '',
  description: ''
})

const batchSubmitting = ref(false)
const importSubmitting = ref(false)
const batchForm = reactive({
  updateRuleGroup: false,
  ruleGroupId: undefined as number | undefined,
  updateDataSource: false,
  dataSourceId: undefined as number | undefined,
  updateFaultCenter: false,
  faultCenterId: undefined as number | undefined,
  updateEnabled: false,
  enabled: true
})

const importForm = reactive({
  mode: 'json' as ImportMode,
  dataSourceId: undefined as number | undefined,
  ruleGroupId: undefined as number | undefined,
  faultCenterId: undefined as number | undefined,
  defaultSeverity: 'p1' as SeverityValue,
  content: ''
})

const form = reactive<RuleForm>({
  name: '',
  ruleGroupId: undefined,
  faultCenterId: undefined,
  dataSourceType: 'prometheus',
  dataSourceIds: [],
  query: '',
  queryMode: 'instant',
  index: '',
  severityRules: [{ severity: 'p1', condition: 'gt', threshold: 0, forSeconds: 60 }],
  evaluateInterval: 60,
  enabled: true,
  channelIds: [],
  notifyRecovery: true,
  repeatInterval: 3600,
  labels: [],
  annotations: [],
  detailTemplate: '',
  callbackQueries: []
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  ruleGroupId: [{ required: true, message: '请选择规则组', trigger: 'change' }],
  faultCenterId: [{ required: true, message: '请选择故障中心', trigger: 'change' }],
  dataSourceIds: [{ required: true, message: '请选择数据源', trigger: 'change' }],
  query: [{ required: true, message: '请输入查询语句', trigger: 'blur' }],
  detailTemplate: [{ required: true, message: '请输入告警详情', trigger: 'blur' }]
}

const queryModeOptions = [
  { label: '即时查询', value: 'instant' },
  { label: '范围查询', value: 'range' }
]

const sourceTypeOptions: Array<{ label: string; value: DataSourceType; desc: string }> = [
  { label: 'Prometheus', value: 'prometheus', desc: '指标查询 PromQL' },
  { label: 'VictoriaMetrics', value: 'victoriametrics', desc: '兼容 PromQL' },
  { label: 'Loki', value: 'loki', desc: '日志查询 LogQL' },
  { label: 'Elasticsearch', value: 'elasticsearch', desc: '日志 DSL / 聚合' }
]

const elasticsearchDslTemplates: ElasticsearchDslTemplate[] = [
  {
    key: 'keyword-count',
    name: '关键字命中次数',
    description: '统计包含 ERROR / Exception 的文档数量；需要限制时间时请改成索引真实的时间字段。',
    build: () => buildElasticsearchKeywordCountDsl()
  },
  {
    key: 'http-5xx-count',
    name: 'HTTP 5xx 次数',
    description: '统计 status 在 500-599 的文档数量，适合接口错误告警。',
    build: () => buildElasticsearchStatusCountDsl()
  },
  {
    key: 'unique-users',
    name: '唯一用户数',
    description: '按 user_id.keyword 统计唯一用户数，可用于异常访问量或登录用户数告警。',
    build: () => buildElasticsearchCardinalityDsl()
  },
  {
    key: 'latest-docs',
    name: '最近日志样例',
    description: '返回日志样例；默认不依赖 @timestamp，避免索引没有时间字段时报错。',
    build: () => buildElasticsearchLatestDocsDsl()
  }
]

const enabledCount = computed(() => ruleData.value.filter(item => item.enabled).length)
const firingCount = computed(() => ruleData.value.filter(item => item.lastState === 'firing').length)
const selectedRuleIds = computed(() => selectedRuleRows.value.map(item => Number(item.id)).filter(Boolean))
const groupFilters = computed(() => [{ id: 0, name: '全部规则', description: '', sort: 0 }, ...ruleGroups.value])
const filteredDataSourcesForForm = computed(() => dataSources.value.filter(item => item.type === form.dataSourceType && item.enabled))
const promCompatibleSources = computed(() => dataSources.value.filter(item => item.enabled && ['prometheus', 'victoriametrics'].includes(item.type)))
const importDataSources = computed(() => dataSources.value.filter(item => item.enabled))
const currentImportDataSources = computed(() => importForm.mode === 'prometheusRule' ? promCompatibleSources.value : importDataSources.value)
const importDialogTitle = computed(() => importForm.mode === 'prometheusRule' ? '导入 PrometheusRule' : '导入 OpsHub / WatchAlert JSON')
const importContentLabel = computed(() => importForm.mode === 'prometheusRule' ? 'YAML' : 'JSON')
const groupDialogTitle = computed(() => groupEditingId.value ? '编辑规则组' : '新增规则组')
const importTipText = computed(() => importForm.mode === 'prometheusRule'
  ? '导入 PrometheusRule 常用 rules: 根结构，会使用下方选择的数据源、规则组和故障中心。仅支持 Prometheus / VictoriaMetrics 数据源。'
  : '导入 OpsHub 导出的 JSON，同时兼容 WatchAlert 导出的规则 JSON。跨环境导入时会优先使用下方选择的数据源、规则组和故障中心。'
)
const jsonImportPlaceholder = `[
  {
    "tenantId": "default",
    "ruleId": "rule-demo",
    "ruleGroupId": "rg-demo",
    "datasourceType": "Prometheus",
    "datasourceId": ["ds-demo"],
    "ruleName": "CPU 使用率过高",
    "evalInterval": 60,
    "repeatNoticeInterval": 3600,
    "prometheusConfig": {
      "promQL": "(1 - avg by(instance) (rate(node_cpu_seconds_total{mode=\\"idle\\"}[5m]))) * 100",
      "annotations": "CPU 使用率过高，当前值 \${value}%",
      "rules": [{ "forDuration": 60, "severity": "P1", "expr": ">80" }]
    }
  }
]`
const prometheusRuleImportPlaceholder = `# 示例：
rules:
- alert: Exporter Component is Down
  expr: up == 0
  for: 2m
  labels:
    severity: serious
  annotations:
    summary: 节点 Exporter Component is Down
    description: 节点 Exporter Component is Down`
const importPlaceholder = computed(() => importForm.mode === 'prometheusRule' ? prometheusRuleImportPlaceholder : jsonImportPlaceholder)
const previewItems = computed(() => normalizePreviewItems(previewResult.value?.result, previewSource.value?.type))
const previewGraphItems = computed(() => normalizePreviewItems((previewGraphResult.value || previewResult.value)?.result, previewSource.value?.type))
const hasPreviewChartData = computed(() => previewGraphItems.value.some(item => item.points.length > 0 || typeof item.numericValue === 'number'))
const previewFetchedAtText = computed(() => previewFetchedAt.value ? formatDateTime(previewFetchedAt.value.toISOString()) : '-')
const evalSamples = computed(() => evalResult.value?.samples || [])
const evalMatchedCount = computed(() => evalSamples.value.filter(item => item.matched).length)
const detailPreviewContext = computed(() => buildDetailPreviewContext())
const shouldShowLogTokens = computed(() => logDataSourceTypes.includes(form.dataSourceType))
const availableDetailTemplateTokens = computed<DetailTemplateToken[]>(() => {
  const tokens: DetailTemplateToken[] = []
  const labelKeys = new Set<string>()
  const fallbackKeys = fallbackLabelKeysBySource[form.dataSourceType] || ['instance']
  fallbackKeys.forEach(key => labelKeys.add(key))
  Object.keys(detailPreviewContext.value.labels).forEach(key => labelKeys.add(key))
  form.labels.forEach(item => {
    const key = item.key.trim()
    if (key) labelKeys.add(key)
  })
  Array.from(labelKeys)
    .filter(Boolean)
    .slice(0, 8)
    .forEach(key => tokens.push(createLabelToken(key)))
  tokens.push(...baseDetailTemplateTokens)
  if (shouldShowLogTokens.value) {
    tokens.push(...lokiDetailTemplateTokens)
  }
  return tokens
})
const detailPreviewText = computed(() => renderDetailTemplatePreview(form.detailTemplate, detailPreviewContext.value))
const detailPreviewBlocks = computed(() => splitDetailPreviewBlocks(detailPreviewText.value))
const detailPreviewSourceText = computed(() => {
  const map: Record<DetailPreviewContext['source'], string> = {
    evaluation: '使用评估结果',
    preview: '使用数据预览',
    empty: '等待数据预览'
  }
  return map[detailPreviewContext.value.source]
})
const detailPreviewSourceType = computed(() => {
  const map: Record<DetailPreviewContext['source'], string> = {
    evaluation: 'success',
    preview: 'primary',
    empty: 'info'
  }
  return map[detailPreviewContext.value.source]
})
const previewQueryModeText = computed(() => {
  if (previewSource.value?.type === 'elasticsearch') return 'DSL'
  return previewQueryMode.value === 'range' ? '范围查询' : '即时查询'
})

const filteredRules = computed(() => {
  const keyword = searchForm.keyword.trim().toLowerCase()
  return ruleData.value.filter(item => {
    const matchesGroup = selectedGroupId.value === 0 || item.ruleGroupId === selectedGroupId.value
    const matchesKeyword = !keyword ||
      item.name.toLowerCase().includes(keyword) ||
      item.query.toLowerCase().includes(keyword) ||
      (item.detailTemplate || '').toLowerCase().includes(keyword) ||
      (item.annotations || '').toLowerCase().includes(keyword)
    const matchesType = !searchForm.dataSourceType || item.dataSourceType === searchForm.dataSourceType
    const matchesFaultCenter = !searchForm.faultCenterId || item.faultCenterId === searchForm.faultCenterId
    const matchesState = !searchForm.state || item.lastState === searchForm.state
    return matchesGroup && matchesKeyword && matchesType && matchesFaultCenter && matchesState
  })
})

const pagedRules = computed(() => {
  const start = (rulePagination.page - 1) * rulePagination.pageSize
  return filteredRules.value.slice(start, start + rulePagination.pageSize)
})

watch(
  () => [selectedGroupId.value, searchForm.keyword, searchForm.dataSourceType, searchForm.faultCenterId, searchForm.state],
  () => {
    rulePagination.page = 1
  }
)

watch(filteredRules, () => {
  const maxPage = Math.max(1, Math.ceil(filteredRules.value.length / rulePagination.pageSize))
  if (rulePagination.page > maxPage) rulePagination.page = maxPage
})

watch(
  () => [form.dataSourceType, form.dataSourceIds.join(','), form.query, form.queryMode, form.index],
  () => {
    const signature = buildDetailPreviewSignature()
    if (previewSignature.value && previewSignature.value !== signature) {
      previewResult.value = undefined
      previewGraphResult.value = undefined
      previewMatchedLog.value = null
      previewMatchedLogSignature.value = ''
      previewSource.value = undefined
      previewQueryText.value = ''
      previewFetchedAt.value = undefined
      previewError.value = ''
      previewSignature.value = ''
      disposePreviewChart()
    }
    if (evalResultSignature.value && evalResultSignature.value !== signature) {
      evalResult.value = null
      evalResultSignature.value = ''
    }
  }
)

const loadRules = async () => {
  loading.value = true
  try {
    ruleData.value = await getMonitorAlertRules() || []
  } finally {
    loading.value = false
  }
}

const loadEvents = async () => {
  eventsLoading.value = true
  try {
    const result = await getMonitorAlertEvents({ page: 1, pageSize: 8 })
    eventData.value = result?.list || []
  } finally {
    eventsLoading.value = false
  }
}

const loadStats = async () => {
  try {
    const data = await getMonitorAlertEventStats()
    if (data) stats.value = data
  } catch {
    // Keep local counters if stats endpoint is unavailable.
  }
}

const loadMeta = async () => {
  const [sources, centers, groups] = await Promise.all([
    getMonitorDataSources(),
    getMonitorFaultCenters(),
    getMonitorRuleGroups()
  ])
  dataSources.value = sources || []
  faultCenters.value = centers || []
  ruleGroups.value = groups || []
  if (!selectedGroupId.value && ruleGroups.value[0]?.id) {
    selectedGroupId.value = 0
  }
}

const loadAll = async () => {
  await Promise.all([loadMeta(), loadRules(), loadEvents(), loadStats()])
}

const handleReset = () => {
  searchForm.keyword = ''
  searchForm.dataSourceType = ''
  searchForm.faultCenterId = undefined
  searchForm.state = ''
  selectedGroupId.value = 0
}

const handleRulePageSizeChange = (size: number) => {
  rulePagination.pageSize = size
  rulePagination.page = 1
}

const handleRulePageChange = (page: number) => {
  rulePagination.page = page
}

const handleSelectionChange = (rows: RuleRow[]) => {
  selectedRuleRows.value = rows
}

const resetBatchForm = () => {
  batchForm.updateRuleGroup = false
  batchForm.ruleGroupId = undefined
  batchForm.updateDataSource = false
  batchForm.dataSourceId = undefined
  batchForm.updateFaultCenter = false
  batchForm.faultCenterId = undefined
  batchForm.updateEnabled = false
  batchForm.enabled = true
}

const openBatchUpdate = () => {
  if (!selectedRuleIds.value.length) {
    ElMessage.warning('请先选择告警规则')
    return
  }
  resetBatchForm()
  batchDialogVisible.value = true
}

const submitBatchUpdate = async () => {
  if (!selectedRuleIds.value.length) {
    ElMessage.warning('请先选择告警规则')
    return
  }
  const payload: any = { ids: selectedRuleIds.value }
  if (batchForm.updateRuleGroup) {
    if (!batchForm.ruleGroupId) {
      ElMessage.warning('请选择规则组')
      return
    }
    payload.ruleGroupId = batchForm.ruleGroupId
  }
  if (batchForm.updateDataSource) {
    if (!batchForm.dataSourceId) {
      ElMessage.warning('请选择数据源')
      return
    }
    payload.dataSourceId = batchForm.dataSourceId
  }
  if (batchForm.updateFaultCenter) {
    if (!batchForm.faultCenterId) {
      ElMessage.warning('请选择故障中心')
      return
    }
    payload.faultCenterId = batchForm.faultCenterId
  }
  if (batchForm.updateEnabled) {
    payload.enabled = batchForm.enabled
  }
  if (Object.keys(payload).length <= 1) {
    ElMessage.warning('请至少选择一个需要更新的字段')
    return
  }
  batchSubmitting.value = true
  try {
    await batchUpdateMonitorAlertRules(payload)
    ElMessage.success('批量更新成功')
    batchDialogVisible.value = false
    selectedRuleRows.value = []
    await Promise.all([loadRules(), loadEvents(), loadStats(), loadMeta()])
  } finally {
    batchSubmitting.value = false
  }
}

const handleBatchDelete = async () => {
  if (!selectedRuleIds.value.length) {
    ElMessage.warning('请先选择告警规则')
    return
  }
  await ElMessageBox.confirm(`确定删除选中的 ${selectedRuleIds.value.length} 条告警规则吗？相关活跃告警会自动恢复。`, '批量删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await batchDeleteMonitorAlertRules(selectedRuleIds.value)
  ElMessage.success('批量删除成功')
  selectedRuleRows.value = []
  await Promise.all([loadRules(), loadEvents(), loadStats(), loadMeta()])
}

const setImportMode = (mode: ImportMode) => {
  importForm.mode = mode
  const candidates = mode === 'prometheusRule' ? promCompatibleSources.value : importDataSources.value
  if (!candidates.some(item => item.id === importForm.dataSourceId)) {
    importForm.dataSourceId = candidates[0]?.id
  }
}

const openImportDialog = (mode: ImportMode = 'json') => {
  importForm.mode = mode
  const candidates = mode === 'prometheusRule' ? promCompatibleSources.value : importDataSources.value
  importForm.dataSourceId = candidates[0]?.id
  importForm.ruleGroupId = ruleGroups.value[0]?.id
  importForm.faultCenterId = faultCenters.value[0]?.id
  importForm.defaultSeverity = 'p1'
  importForm.content = ''
  importDialogVisible.value = true
}

const submitAlertRuleImport = async () => {
  if (!importForm.dataSourceId) {
    ElMessage.warning('请选择数据源')
    return
  }
  if (!importForm.content.trim()) {
    ElMessage.warning(importForm.mode === 'prometheusRule' ? '请粘贴 PrometheusRule YAML' : '请粘贴 OpsHub / WatchAlert JSON')
    return
  }
  importSubmitting.value = true
  try {
    const payload = {
      dataSourceId: Number(importForm.dataSourceId),
      ruleGroupId: importForm.ruleGroupId,
      faultCenterId: importForm.faultCenterId,
      defaultSeverity: importForm.defaultSeverity
    }
    const content = importForm.content.trim()
    const result = importForm.mode === 'prometheusRule'
      ? await importPrometheusRuleYaml({ ...payload, yaml: content })
      : await importMonitorAlertRules({ ...payload, content })
    const imported = Number(result?.imported || 0)
    const skipped = Array.isArray(result?.skipped) ? result.skipped.length : 0
    ElMessage.success(`导入成功：${imported} 条${skipped ? `，跳过 ${skipped} 条` : ''}`)
    importDialogVisible.value = false
    await Promise.all([loadRules(), loadEvents(), loadStats(), loadMeta()])
  } finally {
    importSubmitting.value = false
  }
}

const handleExport = async (selectedOnly: boolean) => {
  const ids = selectedOnly
    ? selectedRuleIds.value
    : filteredRules.value.map(item => Number(item.id)).filter(Boolean)
  if (selectedOnly && !ids.length) {
    ElMessage.warning('请先选择告警规则')
    return
  }
  const blob = await exportMonitorAlertRules(ids) as Blob
  downloadBlob(blob, `rules_export_${formatExportTimestamp()}.json`)
  ElMessage.success('导出成功')
}

const handleAdd = () => {
  resetForm()
  drawerTitle.value = '新增规则'
  drawerVisible.value = true
}

const handleEdit = (row: MonitorAlertRule) => {
  resetForm()
  drawerTitle.value = '编辑规则'
  const ids = parseIdArray(row.dataSourceIds)
  Object.assign(form, {
    id: row.id,
    name: row.name,
    ruleGroupId: row.ruleGroupId || ruleGroups.value[0]?.id,
    faultCenterId: row.faultCenterId || faultCenters.value[0]?.id,
    dataSourceType: (row.dataSourceType || 'prometheus') as DataSourceType,
    dataSourceIds: ids.length > 0 ? ids : row.dataSourceId ? [row.dataSourceId] : [],
    query: row.query,
    queryMode: row.queryMode || 'instant',
    index: row.index || '',
    severityRules: getSeverityRules(row),
    evaluateInterval: row.evaluateInterval || 60,
    enabled: row.enabled,
    channelIds: parseIdArray(row.channelIds),
    notifyRecovery: row.notifyRecovery ?? true,
    repeatInterval: row.repeatInterval || 3600,
    labels: parseKeyValueRows(row.labels),
    annotations: parseKeyValueRows(row.annotations),
    detailTemplate: resolveRuleDetailTemplate(row),
    callbackQueries: parseCallbackQueryRows(row.callbackQueries)
  })
  drawerVisible.value = true
}

const handleDrawerClose = () => {
  suggestionVisible.value = false
  resetForm()
}

const selectSourceType = (type: DataSourceType) => {
  if (form.dataSourceType === type) return
  form.dataSourceType = type
  form.dataSourceIds = []
  form.index = ''
  form.queryMode = type === 'elasticsearch' ? 'instant' : 'instant'
  form.query = getQueryPlaceholder(type)
  suggestions.value = []
  suggestionVisible.value = false
  esIndexOptions.value = []
}

const handleSourceChange = async () => {
  const source = getFormPrimarySource()
  if (!source) return
  form.query = form.query || getQueryPlaceholder(source.type)
  if (source.type === 'elasticsearch') {
    form.queryMode = 'instant'
    await loadElasticsearchIndices()
  }
}

const handleEsIndexDropdownVisible = (visible: boolean) => {
  if (visible && !esIndexOptions.value.length) {
    loadElasticsearchIndices()
  }
}

const loadElasticsearchIndices = async (keyword = '') => {
  const source = getFormPrimarySource()
  if (!source?.id || source.type !== 'elasticsearch') {
    esIndexOptions.value = []
    return
  }
  esIndexLoading.value = true
  try {
    esIndexOptions.value = await getMonitorDataSourceIndices(source.id, {
      keyword: keyword || undefined,
      limit: 300
    }) || []
  } finally {
    esIndexLoading.value = false
  }
}

const handleQueryInput = () => {
  if (suggestionTimer) clearTimeout(suggestionTimer)
  suggestionTimer = setTimeout(loadSuggestions, 220)
}

const loadSuggestions = async () => {
  const source = getFormPrimarySource()
  if (!source?.id) {
    suggestionVisible.value = false
    return
  }
  const keyword = extractQueryKeyword(form.query)
  if (!keyword) {
    suggestionVisible.value = false
    return
  }
  try {
    suggestions.value = await getMonitorDataSourceSuggestions(source.id, {
      keyword,
      index: form.index,
      limit: 20
    }) || []
    suggestionVisible.value = suggestions.value.length > 0
  } catch {
    suggestionVisible.value = false
  }
}

const insertSuggestion = (item: MonitorQuerySuggestion) => {
  const keyword = extractQueryKeyword(form.query)
  if (keyword) {
    form.query = form.query.slice(0, form.query.length - keyword.length) + item.insertText
  } else {
    form.query = form.query ? `${form.query} ${item.insertText}` : item.insertText
  }
  suggestionVisible.value = false
}

const applyElasticsearchDslTemplate = (item: ElasticsearchDslTemplate) => {
  form.query = item.build()
  ElMessage.success(`已填入「${item.name}」模板`)
}

const handlePreviewQuery = async () => {
  const source = getFormPrimarySource()
  if (!source?.id) {
    ElMessage.warning('请先选择数据源')
    return
  }
  if (!form.query.trim() && source.type !== 'elasticsearch') {
    ElMessage.warning('请输入查询语句')
    return
  }

  const payload = buildPreviewQueryPayload(source)
  const requestSignature = buildDetailPreviewSignature()
  previewSource.value = source
  previewQueryText.value = payload.query || '(默认查询)'
  previewQueryMode.value = source.type === 'elasticsearch' ? 'dsl' : payload.queryMode || 'instant'
  previewTab.value = 'card'
  previewResult.value = undefined
  previewGraphResult.value = undefined
  previewMatchedLog.value = null
  previewMatchedLogSignature.value = ''
  previewFetchedAt.value = undefined
  previewError.value = ''
  previewSignature.value = requestSignature
  previewDialogVisible.value = true
  previewLoading.value = true
  disposePreviewChart()

  try {
    const graphPayload = buildPreviewGraphQueryPayload(source, payload)
    const matchedLogPayload = buildLokiMatchedLogPreviewPayload(source, payload)
    const matchedLogPromise = matchedLogPayload
      ? queryMonitorDataSource(source.id, matchedLogPayload)
        .then(result => buildLokiMatchedLogPreview(matchedLogPayload.query, result?.result))
        .catch(() => ({ query: matchedLogPayload.query, logs: [], count: 0 }))
      : Promise.resolve(undefined)
    if (graphPayload) {
      const [instantResult, graphResult, matchedLogs] = await Promise.all([
        queryMonitorDataSource(source.id, payload),
        queryMonitorDataSource(source.id, graphPayload),
        matchedLogPromise
      ])
      if (requestSignature !== buildDetailPreviewSignature()) return
      previewResult.value = instantResult || {}
      previewGraphResult.value = graphResult || previewResult.value
      applyLokiMatchedLogPreview(matchedLogs, requestSignature)
    } else {
      const [result, matchedLogs] = await Promise.all([
        queryMonitorDataSource(source.id, payload),
        matchedLogPromise
      ])
      if (requestSignature !== buildDetailPreviewSignature()) return
      previewResult.value = result || {}
      previewGraphResult.value = previewResult.value
      applyLokiMatchedLogPreview(matchedLogs, requestSignature)
    }
    previewFetchedAt.value = new Date()
    await nextTick()
    if (previewTab.value === 'graph') {
      renderPreviewChart()
    }
  } catch (error: any) {
    previewError.value = getPreviewErrorMessage(error)
  } finally {
    previewLoading.value = false
  }
}

const getPreviewErrorMessage = (error: any) => {
  return error?.response?.data?.message ||
    error?.response?.data?.error ||
    error?.message ||
    '查询失败，请检查数据源连接和查询语句'
}

const buildPreviewQueryPayload = (source: MonitorDataSource): DataSourceQueryRequest => {
  const payload: DataSourceQueryRequest = {
    query: form.query.trim(),
    queryMode: source.type === 'elasticsearch' ? 'instant' : form.queryMode,
    index: form.index.trim()
  }
  if (source.type !== 'elasticsearch' && form.queryMode === 'range') {
    const end = new Date()
    const start = new Date(end.getTime() - 30 * 60 * 1000)
    payload.start = String(Math.floor(start.getTime() / 1000))
    payload.end = String(Math.floor(end.getTime() / 1000))
    payload.step = '60s'
  }
  return payload
}

const buildLokiMatchedLogPreviewPayload = (source: MonitorDataSource, payload: DataSourceQueryRequest): DataSourceQueryRequest | undefined => {
  if (source.type !== 'loki') return undefined
  const query = deriveLokiPreviewMatchedLogQuery(form.query)
  if (!query) return undefined
  const end = new Date()
  const lookbackSeconds = getLokiPreviewLookbackSeconds(form.query)
  const start = new Date(end.getTime() - lookbackSeconds * 1000)
  return {
    ...payload,
    query,
    queryMode: 'range',
    start: String(Math.floor(start.getTime() / 1000)),
    end: String(Math.floor(end.getTime() / 1000)),
    limit: 20
  }
}

const buildPreviewGraphQueryPayload = (source: MonitorDataSource, payload: DataSourceQueryRequest): DataSourceQueryRequest | undefined => {
  if (!isPromSeriesSource(source.type) || !payload.query?.trim()) return undefined
  if (payload.queryMode === 'range' && payload.start && payload.end) return undefined
  const end = new Date()
  const start = new Date(end.getTime() - 5 * 60 * 1000)
  return {
    ...payload,
    queryMode: 'range',
    start: String(Math.floor(start.getTime() / 1000)),
    end: String(Math.floor(end.getTime() / 1000)),
    step: '15s'
  }
}

const handlePreviewTabChange = async (name: string | number) => {
  if (name === 'graph') {
    await renderPreviewChart()
  }
}

const renderPreviewChart = async () => {
  await nextTick()
  if (!previewChartRef.value || !hasPreviewChartData.value) return
  disposePreviewChart()
  previewChart = echarts.init(previewChartRef.value)

  const lineItems = previewGraphItems.value.filter(item => item.points.length > 1).slice(0, 200)
  if (lineItems.length) {
    previewChart.setOption({
      color: previewChartPalette,
      tooltip: {
        trigger: 'axis',
        triggerOn: 'mousemove|click',
        renderMode: 'html',
        enterable: true,
        confine: false,
        appendToBody: true,
        className: 'watchalert-preview-tooltip',
        backgroundColor: '#1f1f1f',
        borderWidth: 0,
        padding: [12, 14],
        textStyle: {
          color: '#fff',
          fontSize: 13
        },
        extraCssText: 'z-index:30000;border-radius:2px;box-shadow:0 8px 24px rgba(0,0,0,.28);max-height:380px;overflow-y:auto;',
        axisPointer: {
          type: 'line',
          lineStyle: {
            color: '#1890ff',
            width: 1,
            type: 'solid'
          }
        },
        position: (pos: number[], _params: any, _dom: HTMLElement, _rect: any, size: any) => {
          const gap = 14
          const viewWidth = size?.viewSize?.[0] || window.innerWidth
          const viewHeight = size?.viewSize?.[1] || window.innerHeight
          const boxWidth = size?.contentSize?.[0] || 320
          const boxHeight = size?.contentSize?.[1] || 180
          const x = Math.min(pos[0] + gap, Math.max(gap, viewWidth - boxWidth - gap))
          const y = Math.min(pos[1] + gap, Math.max(gap, viewHeight - boxHeight - gap))
          return [x, y]
        },
        formatter: (params: any) => formatPreviewChartTooltip(params)
      },
      legend: {
        show: lineItems.length <= 12,
        type: 'scroll',
        top: 0,
        left: 8,
        right: 8,
        itemWidth: 18,
        itemHeight: 3,
        textStyle: { color: '#4b5563', fontSize: 12 }
      },
      grid: { left: 48, right: 20, top: lineItems.length <= 12 ? 48 : 22, bottom: 36 },
      xAxis: {
        type: 'time',
        boundaryGap: false,
        axisLabel: {
          color: '#6b7280',
          fontSize: 12,
          formatter: (value: number) => formatPreviewAxisTime(value)
        },
        axisLine: { lineStyle: { color: '#d9d9d9' } },
        axisTick: { show: false },
        splitLine: { show: false }
      },
      yAxis: {
        type: 'value',
        scale: true,
        axisLabel: { color: '#6b7280', fontSize: 12 },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { lineStyle: { color: '#f0f0f0' } }
      },
      series: lineItems.map(item => ({
        name: item.title,
        type: 'line',
        smooth: false,
        showSymbol: false,
        emphasis: { focus: 'series' },
        connectNulls: true,
        lineStyle: { width: lineItems.length > 80 ? 1 : 1.6 },
        data: item.points.map(point => {
          const x = point.timestamp ?? parsePreviewTimestamp(point.time)
          if (typeof x !== 'number') return undefined
          return {
            value: [x, point.value],
            time: point.time,
            tooltipTime: point.tooltipTime,
            axisTime: point.axisTime,
            title: item.title,
            labels: item.labels
          }
        }).filter(Boolean)
      }))
    })
    return
  }

  const barItems = previewGraphItems.value.filter(item => typeof item.numericValue === 'number').slice(0, 60)
  previewChart.setOption({
    color: ['#2563eb'],
    tooltip: { trigger: 'axis' },
    grid: { left: 48, right: 18, top: 26, bottom: 68 },
    xAxis: {
      type: 'category',
      data: barItems.map((item, index) => item.title || `Metric #${index + 1}`),
      axisLabel: { color: '#667085', interval: 0, rotate: 24 },
      axisLine: { lineStyle: { color: '#d8dee9' } }
    },
    yAxis: { type: 'value', axisLabel: { color: '#667085' }, splitLine: { lineStyle: { color: '#edf1f7' } } },
    series: [{
      name: '数值',
      type: 'bar',
      barMaxWidth: 34,
      data: barItems.map(item => item.numericValue)
    }]
  })
}

const previewChartPalette = [
  '#2563eb', '#16a34a', '#d97706', '#dc2626', '#7c3aed', '#0891b2', '#db2777', '#4b5563',
  '#22c55e', '#f97316', '#06b6d4', '#8b5cf6', '#ef4444', '#84cc16', '#0ea5e9', '#f59e0b',
  '#14b8a6', '#a855f7', '#e11d48', '#64748b'
]

const formatPreviewChartTooltip = (params: any) => {
  const items = (Array.isArray(params) ? params : [params])
    .filter(item => item?.data !== null && item?.data !== undefined)
  if (!items.length) return ''
  const firstData = items[0]?.data && typeof items[0].data === 'object' ? items[0].data : {}
  const firstValue = Array.isArray(firstData.value) ? firstData.value : items[0]?.value
  const time = firstData.tooltipTime || formatPreviewTooltipTime(Array.isArray(firstValue) ? firstValue[0] : items[0]?.axisValue)
  const rows = items.map(item => {
    const data = item?.data && typeof item.data === 'object' ? item.data : {}
    const labels = Array.isArray(data.labels) ? data.labels as PreviewLabel[] : []
    const labelText = formatMetricLabelSet(labels, data.title || item.seriesName || 'series')
    const rawValue = Array.isArray(data.value) ? data.value[1] : data.value ?? item.value
    return `<div style="display:flex;align-items:center;gap:7px;min-height:22px;white-space:nowrap;">
      <span style="display:inline-block;width:9px;height:9px;border-radius:50%;background:${escapeHTML(String(item.color || '#1890ff'))};"></span>
      <span style="color:#f5f5f5;">${escapeHTML(labelText)}</span>
      <span style="color:#f5f5f5;">: ${escapeHTML(formatPreviewValue(rawValue))}</span>
    </div>`
  }).join('')
  return `<div style="min-width:260px;max-width:560px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
    <div style="margin-bottom:9px;color:#fff;font-size:14px;font-weight:500;">时间: ${escapeHTML(String(time))}</div>
    <div style="display:grid;gap:3px;">${rows}</div>
  </div>`
}

const disposePreviewChart = () => {
  if (!previewChart) return
  previewChart.dispose()
  previewChart = null
}

const extractQueryKeyword = (query: string) => {
  const match = query.match(/[a-zA-Z_:][a-zA-Z0-9_:.-]*$/)
  return match?.[0] || ''
}

const addCondition = () => {
  form.severityRules.push({ severity: 'p2', condition: 'gt', threshold: 0, forSeconds: 60 })
}

const removeCondition = (index: number) => {
  if (form.severityRules.length <= 1) return
  form.severityRules.splice(index, 1)
}

const addKeyValueRow = (rows: KeyValueRow[]) => {
  rows.push({ key: '', value: '' })
}

const removeKeyValueRow = (rows: KeyValueRow[], index: number) => {
  rows.splice(index, 1)
}

const addCallbackQuery = () => {
  form.callbackQueries.push({ key: '', query: '', dataSourceId: undefined, queryMode: 'range', rangeSeconds: 1800 })
}

const removeCallbackQuery = (index: number) => {
  form.callbackQueries.splice(index, 1)
}

const parseKeyValueRows = (raw?: string): KeyValueRow[] => {
  const parsed = safeParse<any>(raw, undefined)
  if (!parsed) return []
  if (Array.isArray(parsed)) {
    return parsed.map(item => ({
      key: String(item?.key ?? item?.label ?? '').trim(),
      value: String(item?.value ?? '').trim()
    })).filter(item => item.key || item.value)
  }
  if (typeof parsed === 'object') {
    return Object.entries(parsed).map(([key, value]) => ({
      key,
      value: String(value ?? '')
    }))
  }
  return []
}

const resolveRuleDetailTemplate = (row: MonitorAlertRule) => {
  const detail = String(row.detailTemplate || '').trim()
  if (detail) return detail
  const annotations = parseKeyValueRows(row.annotations)
  for (const key of ['description', 'detail', 'summary', 'message']) {
    const matched = annotations.find(item => item.key.trim().toLowerCase() === key)
    if (matched?.value.trim()) return matched.value.trim()
  }
  return ''
}

const stringifyKeyValueRows = (rows: KeyValueRow[]) => {
  const result = rows.reduce<Record<string, string>>((acc, item) => {
    const key = item.key.trim()
    if (!key) return acc
    acc[key] = item.value.trim()
    return acc
  }, {})
  return Object.keys(result).length ? JSON.stringify(result) : ''
}

const parseCallbackQueryRows = (raw?: string): CallbackQueryRow[] => {
  const parsed = safeParse<any>(raw, undefined)
  if (!parsed) return []
  if (Array.isArray(parsed)) {
    return parsed.map(item => ({
      key: String(item?.key ?? '').trim(),
      query: String(item?.query ?? item?.value ?? '').trim(),
      dataSourceId: item?.dataSourceId ? Number(item.dataSourceId) : undefined,
      queryMode: item?.queryMode === 'range' ? 'range' : 'instant',
      rangeSeconds: Number(item?.rangeSeconds || 1800)
    })).filter(item => item.key || item.query)
  }
  if (typeof parsed === 'object') {
    return Object.entries(parsed).map(([key, value]) => ({
      key,
      query: String(value ?? ''),
      dataSourceId: undefined,
      queryMode: 'instant' as const,
      rangeSeconds: 1800
    }))
  }
  return []
}

const stringifyCallbackQueryRows = (rows: CallbackQueryRow[]) => {
  const result = rows
    .map(item => ({
      key: item.key.trim(),
      query: item.query.trim(),
      value: item.query.trim(),
      dataSourceId: item.dataSourceId || undefined,
      queryMode: item.queryMode || 'instant',
      rangeSeconds: Number(item.rangeSeconds || 1800)
    }))
    .filter(item => item.key || item.query)
  return result.length ? JSON.stringify(result) : ''
}

const buildPayload = (): MonitorAlertRule => {
  const primary = form.severityRules[0] || { severity: 'p1', condition: 'gt', threshold: 0, forSeconds: 60 }
  return {
    id: form.id,
    name: form.name,
    ruleGroupId: form.ruleGroupId,
    faultCenterId: form.faultCenterId,
    dataSourceId: form.dataSourceIds[0] || 0,
    dataSourceIds: JSON.stringify(form.dataSourceIds),
    dataSourceType: form.dataSourceType,
    query: form.query,
    queryMode: form.queryMode,
    index: form.index,
    condition: primary.condition,
    threshold: primary.threshold,
    severityRules: JSON.stringify(form.severityRules),
    forSeconds: primary.forSeconds,
    evaluateInterval: form.evaluateInterval,
    severity: primary.severity,
    enabled: form.enabled,
    channelIds: JSON.stringify(form.channelIds),
    notifyRecovery: form.notifyRecovery,
    repeatInterval: form.repeatInterval,
    labels: stringifyKeyValueRows(form.labels),
    annotations: '',
    detailTemplate: form.detailTemplate.trim(),
    callbackQueries: stringifyCallbackQueryRows(form.callbackQueries),
    effectiveTime: JSON.stringify({ week: [], startTime: '', endTime: '' })
  }
}

const handleSubmit = async () => {
  await formRef.value?.validate()
  if (!form.severityRules.length) {
    ElMessage.warning('请至少配置一个等级条件')
    return
  }
  submitting.value = true
  try {
    if (form.id) {
      await updateMonitorAlertRule(form.id, buildPayload())
      ElMessage.success('更新成功')
    } else {
      await createMonitorAlertRule(buildPayload())
      ElMessage.success('创建成功')
    }
    drawerVisible.value = false
    await Promise.all([loadRules(), loadEvents(), loadStats(), loadMeta()])
  } finally {
    submitting.value = false
  }
}

const handleEnabledChange = async (row: RuleRow) => {
  if (!row.id) return
  try {
    const payload = { ...row }
    delete payload.evaluating
    await updateMonitorAlertRule(row.id, payload)
    ElMessage.success('状态已更新')
    await loadStats()
  } catch {
    row.enabled = !row.enabled
  }
}

const handleEvaluate = async (row: RuleRow) => {
  if (!row.id) return
  row.evaluating = true
  try {
    const result = await evaluateMonitorAlertRule(row.id)
    evalResult.value = normalizeEvaluationResult(result)
    evalResultSignature.value = buildRulePreviewSignature(row)
    evalDialogVisible.value = true
    await Promise.all([loadRules(), loadEvents(), loadStats(), loadMeta()])
  } finally {
    row.evaluating = false
  }
}

const normalizeEvaluationResult = (result: any): EvaluationResult => ({
  ruleId: Number(result?.ruleId || 0),
  ruleName: String(result?.ruleName || ''),
  dataSourceName: String(result?.dataSourceName || ''),
  dataSourceType: String(result?.dataSourceType || 'prometheus') as DataSourceType,
  severity: String(result?.severity || ''),
  state: String(result?.state || ''),
  previousState: String(result?.previousState || ''),
  matched: Boolean(result?.matched),
  value: Number(result?.value || 0),
  condition: String(result?.condition || 'gt') as ConditionOperator,
  threshold: Number(result?.threshold || 0),
  forSeconds: Number(result?.forSeconds || 0),
  labels: result?.labels || {},
  fingerprint: String(result?.fingerprint || ''),
  message: String(result?.message || ''),
  statusCode: Number(result?.statusCode || 0),
  evaluatedAt: String(result?.evaluatedAt || ''),
  samples: Array.isArray(result?.samples)
    ? result.samples.map((item: any) => ({
      fingerprint: String(item?.fingerprint || ''),
      matched: Boolean(item?.matched),
      state: String(item?.state || ''),
      value: Number(item?.value || 0),
      severity: String(item?.severity || ''),
      condition: String(item?.condition || 'gt') as ConditionOperator,
      threshold: Number(item?.threshold || 0),
      forSeconds: Number(item?.forSeconds || 0),
      labels: item?.labels || {},
      matchedLogs: parseMatchedLogs(item?.matchedLogs),
      matchedLogCount: Number(item?.matchedLogCount || 0),
      matchedLogQuery: String(item?.matchedLogQuery || ''),
      message: String(item?.message || '')
    }))
    : []
})

const parseMatchedLogs = (value: any): MatchedLogEntry[] => {
  if (!Array.isArray(value)) return []
  return value.map((item: any) => ({
    timestamp: String(item?.timestamp || ''),
    line: String(item?.line || ''),
    labels: item?.labels || {}
  })).filter(item => item.line.trim())
}

const formatMatchedLogs = (logs: MatchedLogEntry[] = [], total = 0) => {
  const lines = logs.map(item => {
    const line = item.timestamp ? `[${item.timestamp}] ${item.line}` : item.line
    return line.trim()
  }).filter(Boolean)
  if (total > logs.length) {
    lines.push(`... 还有 ${total - logs.length} 行命中日志未展示`)
  }
  return lines.join('\n')
}

const handleDelete = async (row: MonitorAlertRule) => {
  if (!row.id) return
  await ElMessageBox.confirm(`确定删除规则「${row.name}」吗？`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await deleteMonitorAlertRule(row.id)
  ElMessage.success('删除成功')
  await Promise.all([loadRules(), loadEvents(), loadStats(), loadMeta()])
}

const resetGroupForm = () => {
  groupEditingId.value = undefined
  groupForm.name = ''
  groupForm.description = ''
}

const openGroupCreate = () => {
  resetGroupForm()
  groupDialogVisible.value = true
}

const openGroupEdit = (group: MonitorRuleGroup) => {
  if (!group.id) return
  groupEditingId.value = group.id
  groupForm.name = group.name || ''
  groupForm.description = group.description || ''
  groupDialogVisible.value = true
}

const handleRuleGroupCommand = (command: string, group: MonitorRuleGroup) => {
  if (command === 'edit') {
    openGroupEdit(group)
    return
  }
  if (command === 'delete') {
    void handleDeleteRuleGroup(group)
  }
}

const submitGroup = async () => {
  if (!groupForm.name.trim()) {
    ElMessage.warning('请输入规则组名称')
    return
  }
  if (groupEditingId.value) {
    const current = ruleGroups.value.find(item => item.id === groupEditingId.value)
    await updateMonitorRuleGroup(groupEditingId.value, {
      name: groupForm.name.trim(),
      description: groupForm.description.trim(),
      sort: current?.sort
    })
    ElMessage.success('更新成功')
  } else {
    await createMonitorRuleGroup({ name: groupForm.name.trim(), description: groupForm.description.trim(), sort: ruleGroups.value.length + 1 })
    ElMessage.success('创建成功')
  }
  groupDialogVisible.value = false
  await loadMeta()
}

const handleDeleteRuleGroup = async (group: MonitorRuleGroup) => {
  if (!group.id) return
  const count = getGroupCount(group.id)
  const message = count > 0
    ? `规则组「${group.name}」下还有 ${count} 条告警规则，删除前请确认这些规则是否已经迁移。确定继续删除吗？`
    : `确定删除规则组「${group.name}」吗？`
  await ElMessageBox.confirm(message, '删除规则组', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    confirmButtonClass: 'el-button--danger'
  })
  await deleteMonitorRuleGroup(group.id)
  if (selectedGroupId.value === group.id) {
    selectedGroupId.value = 0
  }
  ElMessage.success('删除成功')
  await Promise.all([loadMeta(), loadRules(), loadStats()])
}

const resetForm = () => {
  form.id = undefined
  form.name = ''
  form.ruleGroupId = ruleGroups.value[0]?.id
  form.faultCenterId = faultCenters.value[0]?.id
  form.dataSourceType = 'prometheus'
  form.dataSourceIds = []
  form.query = ''
  form.queryMode = 'instant'
  form.index = ''
  form.severityRules = [{ severity: 'p1', condition: 'gt', threshold: 0, forSeconds: 60 }]
  form.evaluateInterval = 60
  form.enabled = true
  form.channelIds = []
  form.notifyRecovery = true
  form.repeatInterval = 3600
  form.labels = []
  form.annotations = []
  form.detailTemplate = ''
  form.callbackQueries = []
  clearRuntimePreviewContext()
  formRef.value?.clearValidate()
}

const getFormPrimarySource = () => dataSources.value.find(item => item.id === form.dataSourceIds[0])

const buildDetailPreviewSignature = () => {
  const source = getFormPrimarySource()
  return [
    source?.id || 0,
    source?.type || form.dataSourceType || '',
    form.queryMode || 'instant',
    form.index.trim(),
    form.query.trim()
  ].join('|')
}

const buildRulePreviewSignature = (row: MonitorAlertRule) => {
  const ids = parseIdArray(row.dataSourceIds)
  const sourceId = ids[0] || row.dataSourceId || 0
  return [
    sourceId,
    row.dataSourceType || '',
    row.queryMode || 'instant',
    row.index || '',
    row.query || ''
  ].map(item => String(item).trim()).join('|')
}

const clearRuntimePreviewContext = () => {
  evalResult.value = null
  evalResultSignature.value = ''
  previewResult.value = undefined
  previewGraphResult.value = undefined
  previewMatchedLog.value = null
  previewMatchedLogSignature.value = ''
  previewSource.value = undefined
  previewQueryText.value = ''
  previewFetchedAt.value = undefined
  previewError.value = ''
  previewSignature.value = ''
  disposePreviewChart()
}

const getGroupCount = (id: number) => id === 0 ? ruleData.value.length : ruleData.value.filter(item => item.ruleGroupId === id).length
const getGroupName = (id?: number) => ruleGroups.value.find(item => item.id === id)?.name || '默认规则组'
const getFaultCenterName = (id?: number) => faultCenters.value.find(item => item.id === id)?.name || '默认故障中心'
const getDataSourceName = (id?: number) => dataSources.value.find(item => item.id === id)?.name || '-'

const getSeverityRules = (row: MonitorAlertRule): SeverityRule[] => {
  const parsed = safeParse<SeverityRule[]>(row.severityRules, [])
  if (Array.isArray(parsed) && parsed.length > 0) {
    return parsed.map(item => ({
      ...item,
      severity: normalizeSeverityForForm(item.severity)
    }))
  }
  return [{
    severity: normalizeSeverityForForm(row.severity),
    condition: row.condition || 'gt',
    threshold: row.threshold ?? 0,
    forSeconds: row.forSeconds || 60
  }]
}

const normalizeSeverityForForm = (severity?: string): SeverityValue => {
  const value = String(severity || '').toLowerCase()
  if (value === 'critical' || value === 'p0') return 'p0'
  if (value === 'warning' || value === 'p1') return 'p1'
  if (value === 'info' || value === 'p2') return 'p2'
  return 'p1'
}

const parseIdArray = (raw?: string) => {
  const parsed = safeParse<number[]>(raw, [])
  if (Array.isArray(parsed)) return parsed.map(Number).filter(Boolean)
  return []
}

const safeParse = <T,>(raw: string | undefined, fallback: T): T => {
  if (!raw) return fallback
  try {
    return JSON.parse(raw)
  } catch {
    return fallback
  }
}

const buildDetailPreviewContext = (): DetailPreviewContext => {
  const primary = form.severityRules[0] || { severity: 'p1', condition: 'gt', threshold: 0, forSeconds: 60 }
  const source = getFormPrimarySource()
  const currentSignature = buildDetailPreviewSignature()
  const derivedMatchedLogQuery = form.dataSourceType === 'loki' ? deriveLokiPreviewMatchedLogQuery(form.query) : ''
  const base: DetailPreviewContext = {
    source: 'empty',
    labels: {},
    value: '',
    condition: getConditionSymbol(primary.condition),
    threshold: formatNumber(primary.threshold),
    severity: getSeverityName(primary.severity),
    state: '告警中',
    fingerprint: '',
    matchedLogQuery: derivedMatchedLogQuery,
    matchedLogCount: '',
    matchedLogs: derivedMatchedLogQuery ? lokiMatchedLogsPendingText : '',
    message: '',
    dataSourceName: source?.name || '-',
    dataSourceType: getTypeName(form.dataSourceType)
  }

  const evaluation = evalResult.value &&
    evalResultSignature.value === currentSignature &&
    (!form.id || evalResult.value.ruleId === form.id)
    ? evalResult.value
    : undefined
  const evalSample = evaluation?.samples?.find(item => item.matched) || evaluation?.samples?.[0]
  if (evaluation && evalSample) {
    const logs = formatMatchedLogs(evalSample.matchedLogs || [], Number(evalSample.matchedLogCount || 0))
    return withFormLabels({
      ...base,
      source: 'evaluation',
      labels: { ...base.labels, ...(evaluation.labels || {}), ...(evalSample.labels || {}) },
      value: formatNumber(evalSample.value),
      condition: getConditionSymbol(evalSample.condition),
      threshold: formatNumber(evalSample.threshold),
      severity: getSeverityName(evalSample.severity),
      state: getStateName(evalSample.state || evaluation.state),
      fingerprint: evalSample.fingerprint || evaluation.fingerprint || '',
      matchedLogQuery: form.dataSourceType === 'loki' ? (evalSample.matchedLogQuery || base.matchedLogQuery) : '',
      matchedLogCount: form.dataSourceType === 'loki' ? String(evalSample.matchedLogCount || evalSample.matchedLogs?.length || '') : '',
      matchedLogs: form.dataSourceType === 'loki' ? logs : '',
      message: evalSample.message || evaluation.message || base.message,
      dataSourceName: evaluation.dataSourceName || base.dataSourceName,
      dataSourceType: getTypeName(evaluation.dataSourceType || form.dataSourceType)
    })
  }

  const previewItem = currentDetailPreviewItem()
  if (previewItem) {
    const previewLabels = labelsFromPreviewItem(previewItem)
    const lokiLogPreview = currentLokiMatchedLogPreview()
    const lokiMatchedLogs = lokiLogPreview
      ? formatMatchedLogs(lokiLogPreview.logs, lokiLogPreview.count) || lokiMatchedLogsEmptyText
      : previewItem.kind === 'log'
        ? String(previewItem.valueText || '')
        : ''
    const lokiMatchedLogCount = lokiLogPreview
      ? String(lokiLogPreview.count)
      : previewItem.kind === 'log'
        ? String(Math.max(previewItem.points.length, 1))
        : ''
    const numericValue = typeof previewItem.numericValue === 'number'
      ? previewItem.numericValue
      : previewItem.kind === 'log'
        ? previewItem.points.length
        : Number(primary.threshold || 0) + 1
    return withFormLabels({
      ...base,
      source: 'preview',
      labels: { ...base.labels, ...previewLabels },
      value: formatNumber(numericValue),
      condition: getConditionSymbol(primary.condition),
      threshold: formatNumber(primary.threshold),
      severity: getSeverityName(primary.severity),
      fingerprint: `preview-${previewItem.title || form.name || 'rule'}`,
      matchedLogQuery: form.dataSourceType === 'loki' ? (lokiLogPreview?.query || base.matchedLogQuery) : '',
      matchedLogCount: form.dataSourceType === 'loki' ? lokiMatchedLogCount : '',
      matchedLogs: form.dataSourceType === 'loki' ? lokiMatchedLogs : '',
      dataSourceName: previewSource.value?.name || source?.name || base.dataSourceName,
      dataSourceType: getTypeName(previewSource.value?.type || form.dataSourceType)
    })
  }

  return withFormLabels(base)
}

const currentDetailPreviewItem = () => {
  const source = getFormPrimarySource()
  if (!source || previewSource.value?.id !== source.id || !previewItems.value.length) return undefined
  if (previewSignature.value !== buildDetailPreviewSignature()) return undefined
  return previewItems.value[0]
}

const currentLokiMatchedLogPreview = () => {
  if (form.dataSourceType !== 'loki') return undefined
  if (!previewMatchedLog.value) return undefined
  if (previewMatchedLogSignature.value !== buildDetailPreviewSignature()) return undefined
  return previewMatchedLog.value
}

const withFormLabels = (context: DetailPreviewContext): DetailPreviewContext => {
  const labels = { ...context.labels }
  form.labels.forEach(item => {
    const key = item.key.trim()
    if (!key) return
    labels[key] = renderSimpleDetailTemplateValue(item.value, context, labels)
  })
  return { ...context, labels }
}

const labelsFromPreviewItem = (item: PreviewItem) => item.labels.reduce<Record<string, string>>((acc, label) => {
  if (label.key) acc[label.key] = label.value
  return acc
}, {})

const renderDetailTemplatePreview = (template: string, context: DetailPreviewContext) => {
  const text = template.trim() || '请先填写告警详情模板'
  return renderSimpleDetailTemplateValue(text, context, context.labels)
}

const renderSimpleDetailTemplateValue = (value: string, context: DetailPreviewContext, labels: Record<string, string>) => {
  const matchedLogsBlock = context.matchedLogs ? `\`\`\`text\n${sanitizePreviewCodeBlock(context.matchedLogs)}\n\`\`\`` : ''
  const replacements: Record<string, string> = {
    '{{ruleName}}': form.name || '告警规则',
    '{{rule_name}}': form.name || '告警规则',
    '${rule_name}': form.name || '告警规则',
    '{{severity}}': context.severity,
    '${severity}': context.severity,
    '{{state}}': context.state,
    '${state}': context.state,
    '{{value}}': context.value,
    '${value}': context.value,
    '{{condition}}': context.condition,
    '${condition}': context.condition,
    '{{threshold}}': context.threshold,
    '${threshold}': context.threshold,
    '{{dataSourceName}}': context.dataSourceName,
    '${data_source_name}': context.dataSourceName,
    '{{dataSourceType}}': context.dataSourceType,
    '${data_source_type}': context.dataSourceType,
    '{{message}}': context.message,
    '${message}': context.message,
    '{{fingerprint}}': context.fingerprint,
    '${fingerprint}': context.fingerprint,
    '{{matchedLogs}}': context.matchedLogs,
    '{{matched_logs}}': context.matchedLogs,
    '${matched_logs}': context.matchedLogs,
    '{{matchedLogsBlock}}': matchedLogsBlock,
    '{{matched_logs_block}}': matchedLogsBlock,
    '${matched_logs_block}': matchedLogsBlock,
    '{{matchedLogCount}}': context.matchedLogCount,
    '{{matched_log_count}}': context.matchedLogCount,
    '${matched_log_count}': context.matchedLogCount,
    '{{matchedLogQuery}}': context.matchedLogQuery,
    '{{matched_log_query}}': context.matchedLogQuery,
    '${matched_log_query}': context.matchedLogQuery
  }
  let rendered = value
  Object.entries(replacements).forEach(([key, replacement]) => {
    rendered = rendered.replaceAll(key, replacement)
  })
  return rendered.replace(/\$\{labels\.([A-Za-z0-9_.:-]+)\}|{{\s*labels\.([A-Za-z0-9_.:-]+)\s*}}/g, (_match, keyA, keyB) => {
    const key = keyA || keyB
    return labels[key] || ''
  })
}

const splitDetailPreviewBlocks = (text: string): DetailPreviewBlock[] => {
  const blocks: DetailPreviewBlock[] = []
  const pattern = /```(?:[A-Za-z0-9_-]+)?\n?([\s\S]*?)```/g
  let lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = pattern.exec(text)) !== null) {
    const before = text.slice(lastIndex, match.index).trimEnd()
    if (before.trim()) blocks.push({ type: 'text', content: before })
    if (match[1].trim()) blocks.push({ type: 'code', content: match[1].trim() })
    lastIndex = pattern.lastIndex
  }
  const rest = text.slice(lastIndex).trimEnd()
  if (rest.trim()) blocks.push({ type: 'text', content: rest })
  return blocks.length ? blocks : [{ type: 'text', content: '暂无可预览内容' }]
}

const sanitizePreviewCodeBlock = (value: string) => value.replaceAll('```', "'''")

const deriveLokiPreviewMatchedLogQuery = (query: string) => {
  const text = query.trim()
  if (!text) return ''
  if (text.startsWith('{')) return stripLokiPreviewRangeSelector(text)
  for (const fn of ['count_over_time', 'rate', 'bytes_over_time', 'bytes_rate', 'absent_over_time']) {
    const arg = lokiPreviewFunctionArgument(text, fn)
    if (arg && arg.trim().startsWith('{')) return stripLokiPreviewRangeSelector(arg.trim())
  }
  return ''
}

const lokiPreviewFunctionArgument = (query: string, fn: string) => {
  const lower = query.toLowerCase()
  const fnIndex = lower.indexOf(fn.toLowerCase())
  if (fnIndex < 0) return ''
  const open = query.indexOf('(', fnIndex + fn.length)
  if (open < 0) return ''
  let depth = 0
  for (let i = open; i < query.length; i += 1) {
    const char = query[i]
    if (char === '(') depth += 1
    if (char === ')') depth -= 1
    if (depth === 0) return query.slice(open + 1, i)
  }
  return ''
}

const stripLokiPreviewRangeSelector = (query: string) => query.replace(/\s*\[[^\]]+\]\s*$/, '').trim()

const getLokiPreviewLookbackSeconds = (query: string) => {
  const rangeSeconds = extractLastLokiPreviewRangeSeconds(query)
  const forSeconds = Math.max(...form.severityRules.map(item => Number(item.forSeconds) || 0), 0)
  const intervalSeconds = Number(form.evaluateInterval) || 0
  const value = Math.max(rangeSeconds, forSeconds, intervalSeconds, 300)
  return Math.min(Math.max(value, 60), maxLokiPreviewLookbackSeconds)
}

const extractLastLokiPreviewRangeSeconds = (query: string) => {
  const matches = Array.from(query.matchAll(/\[([0-9.]+(?:ms|s|m|h|d|w|y)(?:\s*[0-9.]+(?:ms|s|m|h|d|w|y))*)\]/gi))
  const last = matches[matches.length - 1]?.[1]
  return last ? parseDurationToSeconds(last) : 0
}

const parseDurationToSeconds = (value: string) => {
  let total = 0
  const pattern = /([0-9.]+)\s*(ms|s|m|h|d|w|y)/gi
  let match: RegExpExecArray | null
  while ((match = pattern.exec(value)) !== null) {
    const amount = Number(match[1])
    if (!Number.isFinite(amount)) continue
    const unit = match[2].toLowerCase()
    const multiplier: Record<string, number> = {
      ms: 0.001,
      s: 1,
      m: 60,
      h: 3600,
      d: 86400,
      w: 604800,
      y: 31536000
    }
    total += amount * (multiplier[unit] || 0)
  }
  return Math.round(total)
}

const buildLokiMatchedLogPreview = (query: string, raw: any): LokiMatchedLogPreview => {
  const { logs, count } = extractLokiPreviewLogs(raw, 20)
  return { query, logs, count }
}

const applyLokiMatchedLogPreview = (matchedLogs: LokiMatchedLogPreview | undefined, signature: string) => {
  if (!matchedLogs) return
  previewMatchedLog.value = matchedLogs
  previewMatchedLogSignature.value = signature
}

const extractLokiPreviewLogs = (raw: any, limit = 20) => {
  const data = raw?.data || raw
  const result = data?.result ?? raw?.result
  const logs: MatchedLogEntry[] = []
  let count = 0
  if (!Array.isArray(result)) return { logs, count }

  result.forEach((stream: any) => {
    const labels = stream?.stream || stream?.metric || {}
    const values = Array.isArray(stream?.values)
      ? stream.values
      : Array.isArray(stream?.value)
        ? [stream.value]
        : []
    values.forEach((pair: any[]) => {
      const line = String(pair?.[1] || '').trim()
      if (!line) return
      count += 1
      if (logs.length >= limit) return
      logs.push({
        timestamp: formatPreviewTime(pair?.[0]),
        line,
        labels
      })
    })
  })

  return { logs, count }
}

const normalizePreviewItems = (raw: any, type?: string): PreviewItem[] => {
  if (!raw) return []
  if (type === 'elasticsearch') return normalizeElasticsearchPreview(raw)

  const data = raw?.data || raw
  const resultType = data?.resultType || raw?.data?.resultType
  const result = data?.result ?? raw?.result

  if (resultType === 'scalar' && Array.isArray(result)) {
    const numericValue = toNumber(result[1])
    const point = buildPreviewPoint(result)
    return [{
      title: 'Scalar',
      valueText: formatPreviewValue(result[1]),
      numericValue,
      timestamp: formatPreviewTime(result[0]),
      labels: [],
      points: point ? [point] : [],
      kind: 'metric'
    }]
  }

  if (!Array.isArray(result)) return []
  if (resultType === 'streams' || result.some((item: any) => item?.stream && Array.isArray(item?.values))) {
    return normalizeLokiStreams(result)
  }

  return result.map((item: any, index: number) => normalizeMetricPreview(item, index)).filter(Boolean)
}

const normalizeMetricPreview = (item: any, index: number): PreviewItem => {
  const metric = item?.metric || item?.labels || {}
  const values = Array.isArray(item?.values) ? item.values : Array.isArray(item?.value) ? [item.value] : []
  const last = values[values.length - 1] || []
  const numericValue = toNumber(last[1])
  const labels = labelsFromRecord(metric, ['__name__'])
  return {
    title: formatMetricSeriesTitle(metric, index),
    valueText: formatPreviewValue(last[1]),
    numericValue,
    timestamp: formatPreviewTime(last[0]),
    labels,
    points: values
      .map((pair: any[]) => buildPreviewPoint(pair))
      .filter((point: PreviewPoint | undefined): point is PreviewPoint => Boolean(point)),
    kind: 'metric'
  }
}

const normalizeLokiStreams = (result: any[]): PreviewItem[] => result.map((item, index) => {
  const values = Array.isArray(item?.values) ? item.values : []
  const last = values[values.length - 1] || []
  return {
    title: `Log Stream #${index + 1}`,
    valueText: values.length ? clipText(String(last[1] || ''), 96) : '0 条日志',
    timestamp: formatPreviewTime(last[0]),
    labels: labelsFromRecord(item?.stream || {}),
    points: values
      .map((pair: any[]) => buildPreviewPoint([pair?.[0], 1]))
      .filter((point: PreviewPoint | undefined): point is PreviewPoint => Boolean(point)),
    kind: 'log'
  }
})

const normalizeElasticsearchPreview = (raw: any): PreviewItem[] => {
  const items: PreviewItem[] = []
  collectAggregationPreviewItems(raw?.aggregations, items)

  const hits = raw?.hits?.hits
  if (Array.isArray(hits)) {
    hits.slice(0, 20).forEach((hit: any, index: number) => {
      const source = hit?._source || {}
      const message = source.message || source.msg || source.log || JSON.stringify(source)
      const numericValue = toNumber(hit?._score)
      items.push({
        title: hit?._id ? `Document ${hit._id}` : `Document #${index + 1}`,
        valueText: clipText(String(message || '-'), 110),
        numericValue,
        timestamp: formatPreviewTime(source['@timestamp'] || source.timestamp || source.time),
        labels: [
          ...labelsFromRecord({ index: hit?._index, score: hit?._score }, []),
          ...labelsFromRecord(source, ['message', 'msg', 'log']).slice(0, 4)
        ],
        points: typeof numericValue === 'number' ? [{ time: `Doc #${index + 1}`, value: numericValue }] : [],
        kind: 'document'
      })
    })
  }

  return items
}

const collectAggregationPreviewItems = (value: any, items: PreviewItem[], path: string[] = []) => {
  if (!value || typeof value !== 'object') return
  Object.entries(value).forEach(([key, child]) => {
    const nextPath = [...path, key]
    if (child && typeof child === 'object' && 'value' in child) {
      const numericValue = toNumber((child as any).value)
      items.push({
        title: nextPath.join(' / '),
        valueText: formatPreviewValue((child as any).value),
        numericValue,
        labels: [],
        points: typeof numericValue === 'number' ? [{ time: nextPath.join(' / '), value: numericValue }] : [],
        kind: 'metric'
      })
      return
    }
    if (child && typeof child === 'object' && Array.isArray((child as any).buckets)) {
      ;(child as any).buckets.slice(0, 20).forEach((bucket: any) => {
        const numericValue = toNumber(bucket.doc_count)
        items.push({
          title: `${nextPath.join(' / ')}: ${bucket.key_as_string || bucket.key}`,
          valueText: formatPreviewValue(bucket.doc_count),
          numericValue,
          labels: labelsFromRecord({ key: bucket.key_as_string || bucket.key }),
          points: typeof numericValue === 'number' ? [{ time: String(bucket.key_as_string || bucket.key), value: numericValue }] : [],
          kind: 'metric'
        })
        collectAggregationPreviewItems(bucket, items, nextPath)
      })
      return
    }
    collectAggregationPreviewItems(child, items, nextPath)
  })
}

const labelsFromRecord = (record: Record<string, any> | undefined, omit: string[] = []): PreviewLabel[] => {
  if (!record || typeof record !== 'object') return []
  return Object.entries(record)
    .filter(([key, value]) => !omit.includes(key) && value !== undefined && value !== null && String(value) !== '')
    .slice(0, 10)
    .map(([key, value]) => ({ key, value: clipText(String(value), 80) }))
}

const formatMetricSeriesTitle = (metric: Record<string, any> | undefined, index: number) => {
  if (!metric || typeof metric !== 'object') return `Series #${index + 1}`
  const metricName = String(metric.__name__ || '').trim()
  const preferredKeys = ['ecs_cname', 'instance', 'mountpoint', 'device', 'job', 'namespace', 'pod', 'container']
  const parts = preferredKeys
    .map(key => {
      const value = metric[key]
      if (value === undefined || value === null || String(value).trim() === '') return ''
      return `${key}=${String(value).trim()}`
    })
    .filter(Boolean)
  if (parts.length) {
    return metricName ? `${metricName}{${parts.slice(0, 3).join(', ')}}` : parts.slice(0, 3).join(' / ')
  }
  const fallbackParts = Object.entries(metric)
    .filter(([key, value]) => key !== '__name__' && value !== undefined && value !== null && String(value).trim() !== '')
    .slice(0, 3)
    .map(([key, value]) => `${key}=${String(value).trim()}`)
  if (fallbackParts.length) {
    return metricName ? `${metricName}{${fallbackParts.join(', ')}}` : fallbackParts.join(' / ')
  }
  return metricName || `Series #${index + 1}`
}

const formatMetricLabelSet = (labels: PreviewLabel[], fallback = 'series') => {
  if (!labels.length) return String(fallback || 'series')
  const preferred = ['ecs_cname', 'instance', 'mountpoint', 'device', 'job', 'namespace', 'pod', 'container']
  const picked = preferred
    .map(key => labels.find(label => label.key === key))
    .filter((label): label is PreviewLabel => Boolean(label))
  const finalLabels = picked.length ? picked.slice(0, 3) : labels.slice(0, 3)
  return `{${finalLabels.map(label => `${label.key}=${label.value}`).join(',')}}`
}

const toNumber = (value: unknown) => {
  const numberValue = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(numberValue) ? numberValue : undefined
}

const buildPreviewPoint = (pair: any[]): PreviewPoint | undefined => {
  const value = toNumber(pair?.[1])
  if (typeof value !== 'number') return undefined
  const timestamp = parsePreviewTimestamp(pair?.[0])
  return {
    time: formatPreviewTime(pair?.[0]),
    tooltipTime: formatPreviewTooltipTime(timestamp ?? pair?.[0]),
    axisTime: formatPreviewAxisTime(timestamp ?? pair?.[0]),
    timestamp,
    value
  }
}

const formatPreviewValue = (value: unknown) => {
  const numberValue = toNumber(value)
  if (typeof numberValue === 'number') return formatNumber(numberValue)
  return clipText(String(value ?? '-'), 120)
}

const parsePreviewTimestamp = (value: unknown) => {
  if (value === undefined || value === null || value === '') return undefined
  if (typeof value === 'number' || /^\d+(\.\d+)?$/.test(String(value))) {
    const numeric = Number(value)
    const milliseconds = numeric > 1e15 ? numeric / 1e6 : numeric > 1e12 ? numeric : numeric * 1000
    return Number.isFinite(milliseconds) ? milliseconds : undefined
  }
  const parsed = new Date(String(value))
  if (!Number.isNaN(parsed.getTime())) return parsed.getTime()
  return undefined
}

const formatPreviewTime = (value: unknown) => {
  const timestamp = parsePreviewTimestamp(value)
  if (typeof timestamp === 'number') return formatDateTime(new Date(timestamp).toISOString())
  if (value === undefined || value === null || value === '') return '-'
  return String(value)
}

const formatPreviewTooltipTime = (value: unknown) => {
  const timestamp = parsePreviewTimestamp(value)
  if (typeof timestamp !== 'number') return String(value ?? '-')
  const d = new Date(timestamp)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const formatPreviewAxisTime = (value: unknown) => {
  const timestamp = parsePreviewTimestamp(value)
  if (typeof timestamp !== 'number') return String(value ?? '-')
  const d = new Date(timestamp)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const clipText = (value: string, max = 80) => value.length > max ? `${value.slice(0, max)}...` : value

const escapeHTML = (value: string) => value
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')
  .replace(/'/g, '&#39;')

const getTypeName = (type?: string) => {
  const map: Record<string, string> = {
    prometheus: 'Prometheus',
    victoriametrics: 'VictoriaMetrics',
    loki: 'Loki',
    elasticsearch: 'Elasticsearch'
  }
  return type ? map[type] || type : '-'
}

const isPromSeriesSource = (type?: string) => type === 'prometheus' || type === 'victoriametrics'

const getTypeTag = (type?: string) => {
  const map: Record<string, string> = {
    prometheus: 'success',
    victoriametrics: 'primary',
    loki: 'warning',
    elasticsearch: 'danger'
  }
  return type ? map[type] || 'info' : 'info'
}

const getConditionSymbol = (condition?: string) => {
  const map: Record<string, string> = { gt: '>', gte: '>=', lt: '<', lte: '<=', eq: '=', neq: '!=' }
  return condition ? map[condition] || condition : '-'
}

const getStateName = (state?: string) => {
  const map: Record<string, string> = {
    inactive: '正常',
    pending: '预告警',
    firing: '告警中',
    processing: '处理中',
    silenced: '静默中',
    recovering: '待恢复',
    recovered: '已恢复',
    error: '失败'
  }
  return state ? map[state] || state : '未评估'
}

const getStateTag = (state?: string) => {
  const map: Record<string, string> = {
    inactive: 'success',
    pending: 'warning',
    firing: 'danger',
    processing: 'primary',
    silenced: 'info',
    recovering: 'warning',
    recovered: 'success',
    error: 'danger'
  }
  return state ? map[state] || 'info' : 'info'
}

const getSeverityName = (severity?: string) => {
  const map: Record<string, string> = { p0: 'P0', critical: 'P0', p1: 'P1', warning: 'P1', p2: 'P2', info: 'P2' }
  return severity ? map[severity] || severity : '-'
}

const getSeverityTag = (severity?: string) => {
  const map: Record<string, string> = { p0: 'danger', p1: 'warning', p2: 'info', critical: 'danger', warning: 'warning', info: 'info' }
  return severity ? map[severity] || 'info' : 'info'
}

const formatEvalForSeconds = (seconds?: number) => {
  const value = Math.round(Number(seconds) || 0)
  if (value <= 0) return '立即触发'
  if (value < 60) return `${value}秒`
  const minutes = Math.floor(value / 60)
  const secs = value % 60
  if (minutes < 60) return secs ? `${minutes}分${secs}秒` : `${minutes}分钟`
  const hours = Math.floor(minutes / 60)
  const remainMinutes = minutes % 60
  return remainMinutes ? `${hours}小时${remainMinutes}分钟` : `${hours}小时`
}

const getEvalLabelEntries = (labels?: Record<string, string>) => {
  return Object.entries(labels || {})
    .filter(([key, value]) => key && String(value).trim())
    .slice(0, 8)
    .map(([key, value]) => ({ key, value }))
}

const formatNumber = (value?: number) => {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-'
  return Number.isInteger(value) ? String(value) : value.toFixed(2).replace(/\.?0+$/, '')
}

const formatDateTime = (date?: string) => {
  if (!date) return '-'
  const d = new Date(date)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const formatExportTimestamp = () => {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
}

const downloadBlob = (blob: Blob, fileName: string) => {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

const stringifyDsl = (value: Record<string, any>) => JSON.stringify(value, null, 2)

const buildElasticsearchKeywordCountDsl = () => stringifyDsl({
  size: 0,
  track_total_hits: true,
  query: {
    query_string: {
      query: '"ERROR" OR "Exception"'
    }
  }
})

const buildElasticsearchStatusCountDsl = () => stringifyDsl({
  size: 0,
  track_total_hits: true,
  query: {
    bool: {
      filter: [
        {
          range: {
            status: {
              gte: 500,
              lt: 600
            }
          }
        }
      ]
    }
  }
})

const buildElasticsearchCardinalityDsl = () => stringifyDsl({
  size: 0,
  track_total_hits: true,
  query: {
    match_all: {}
  },
  aggs: {
    unique_users: {
      cardinality: {
        field: 'user_id.keyword'
      }
    }
  }
})

const buildElasticsearchLatestDocsDsl = () => stringifyDsl({
  size: 5,
  track_total_hits: true,
  query: {
    match_all: {}
  }
})

const getQueryPlaceholder = (type?: DataSourceType) => {
  if (type === 'loki') return 'sum(count_over_time({job=~".+"}[5m]))'
  if (type === 'elasticsearch') return buildElasticsearchKeywordCountDsl()
  return 'up == 0'
}

onMounted(loadAll)
onBeforeUnmount(() => {
  if (suggestionTimer) clearTimeout(suggestionTimer)
  disposePreviewChart()
})
</script>

<style scoped>
.monitor-rules-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.page-header,
.stats-cards,
.rule-workspace,
.events-section,
.search-bar,
.table-wrapper {
  background: #fff;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 14px 16px;
}

.page-title-group,
.header-actions,
.search-inputs,
.source-cell,
.action-buttons,
.section-header,
.condition-tags,
.condition-head {
  display: flex;
  align-items: center;
}

.page-title-group {
  gap: 14px;
}

.page-title-icon {
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1px solid #edf1f7;
  border-radius: 7px;
  background: #f8fafc;
  color: #111827;
  font-size: 18px;
}

.page-title,
.section-title {
  margin: 0;
  color: #111827;
  font-weight: 750;
  line-height: 1.3;
}

.page-title {
  font-size: 20px;
}

.section-title {
  font-size: 16px;
}

.page-subtitle,
.section-subtitle {
  margin: 4px 0 0;
  color: #667085;
  font-size: 13px;
}

.header-actions,
.search-inputs,
.action-buttons,
.source-cell,
.condition-tags {
  gap: 10px;
}

.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0;
  overflow: hidden;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 74px;
  padding: 13px 16px;
  border-right: 1px solid #edf1f7;
}

.stat-card:last-child {
  border-right: 0;
}

.stat-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 7px;
  font-size: 18px;
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
  font-size: 24px;
  font-weight: 750;
  line-height: 1;
}

.rule-workspace {
  display: grid;
  grid-template-columns: 196px minmax(0, 1fr);
  overflow: hidden;
}

.group-sidebar {
  padding: 10px 8px;
  border-right: 1px solid #edf1f7;
  background: #fbfcfe;
}

.sidebar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  color: #111827;
  font-size: 13px;
  font-weight: 750;
}

.group-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  min-height: 38px;
  padding: 0 8px 0 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: #344054;
  font-size: 13px;
  cursor: pointer;
  text-align: left;
  outline: none;
  transition: background-color .16s ease, border-color .16s ease, color .16s ease, box-shadow .16s ease;
}

.group-item.active,
.group-item:hover,
.group-item:focus-visible {
  border-color: #d6e4ff;
  background: #f5f8ff;
  color: #1d4ed8;
}

.group-item.active {
  box-shadow: 0 1px 2px rgba(16, 24, 40, .05);
}

.group-name {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  flex: 1;
}

.group-name > span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-item em {
  flex-shrink: 0;
  min-width: 26px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #edf1f7;
  border-radius: 999px;
  background: #fff;
  color: #667085;
  font-style: normal;
  font-size: 12px;
  font-weight: 650;
}

.group-tail {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  flex-shrink: 0;
  min-width: 56px;
  margin-left: 8px;
}

.group-more {
  width: 24px;
  height: 24px;
  padding: 0;
  border-radius: 6px;
  color: #98a2b3;
  opacity: 0;
  pointer-events: none;
  transform: translateX(2px);
  transition: opacity .16s ease, transform .16s ease, background-color .16s ease, color .16s ease;
}

.group-item:hover .group-more,
.group-item.active .group-more,
.group-item:focus-within .group-more {
  opacity: 1;
  pointer-events: auto;
  transform: translateX(0);
}

.group-more:hover,
.group-more:focus {
  background: #eaf2ff;
  color: #2563eb;
}

.group-item.active em,
.group-item:hover em {
  border-color: #bfdbfe;
  color: #1d4ed8;
}

.rules-main {
  min-width: 0;
  padding: 12px;
}

.search-bar {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 10px;
  padding: 10px;
  border-radius: 7px;
}

.search-inputs {
  flex: 1;
  flex-wrap: wrap;
}

.search-input {
  width: 260px;
}

.search-input.small {
  width: 150px;
}

.table-wrapper,
.events-section {
  overflow: hidden;
}

.rule-table-wrapper {
  display: flex;
  flex-direction: column;
  min-height: 604px;
}

.batch-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-bottom: 1px solid #edf1f7;
  background: #fafbfc;
}

.batch-summary {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #606266;
  font-size: 13px;
}

.batch-summary strong {
  color: #111827;
  font-size: 16px;
}

.batch-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.batch-dialog-tip {
  margin-bottom: 14px;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fafafa;
  color: #606266;
  font-size: 13px;
}

.import-tip {
  margin-bottom: 14px;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fafafa;
  color: #606266;
  font-size: 13px;
  line-height: 1.6;
}

.import-mode-tabs {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 14px;
  padding: 4px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background: #f6f7f9;
}

.import-mode-tabs button {
  height: 38px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #303133;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.import-mode-tabs button.active {
  background: #1677ff;
  color: #fff;
  box-shadow: 0 4px 10px rgba(22, 119, 255, .22);
}

.import-tip code {
  padding: 1px 5px;
  border-radius: 4px;
  background: #fff;
  color: #111827;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.import-form :deep(.el-textarea__inner) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  line-height: 1.55;
}

.batch-form {
  display: grid;
  gap: 12px;
}

.batch-field {
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
}

.batch-field .el-select {
  width: 100%;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body td) {
  padding: 8px 0;
}

.modern-table :deep(.el-table__header th) {
  padding: 8px 0;
}

.table-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-top: 1px solid #edf1f7;
  background: #fbfcfe;
}

.table-pagination > span {
  color: #667085;
  font-size: 12px;
}

.source-cell {
  min-width: 0;
}

.source-cell span:last-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.condition-tags {
  flex-wrap: wrap;
  gap: 5px;
}

.action-buttons {
  justify-content: center;
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

.action-edit:hover {
  background: #eff6ff;
  color: #2563eb;
}

.action-delete:hover {
  background: #fff1f2;
  color: #e11d48;
}

.section-header {
  justify-content: space-between;
  gap: 16px;
  padding: 12px 14px;
  border-bottom: 1px solid #edf1f7;
}

.rule-drawer :deep(.el-drawer__header) {
  margin-bottom: 0;
  padding: 16px 22px;
  border-bottom: 1px solid #edf0f5;
  color: #111827;
  font-size: 18px;
  font-weight: 760;
}

.rule-drawer :deep(.el-drawer__body) {
  padding: 0;
  background: #f6f7f9;
}

.rule-drawer :deep(.el-drawer__footer) {
  padding: 12px 22px;
  border-top: 1px solid #edf0f5;
  background: #fff;
}

.rule-drawer-form {
  min-height: 100%;
}

.rule-drawer-shell {
  display: grid;
  grid-template-columns: 176px minmax(0, 1fr);
  gap: 18px;
  width: min(100%, 1320px);
  margin: 0 auto;
  padding: 18px 22px 28px;
}

.rule-drawer-nav {
  position: sticky;
  top: 18px;
  align-self: start;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.rule-drawer-nav a {
  display: flex;
  align-items: center;
  min-height: 34px;
  padding: 0 10px;
  border-radius: 6px;
  color: #344054;
  font-size: 13px;
  font-weight: 650;
  text-decoration: none;
}

.rule-drawer-nav a:hover,
.rule-drawer-nav a:focus-visible {
  background: #f3f6fb;
  color: #111827;
  outline: none;
}

.rule-drawer-content {
  display: grid;
  gap: 14px;
  min-width: 0;
}

.drawer-section {
  padding: 18px 18px 20px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  scroll-margin-top: 18px;
}

.drawer-section-title {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 18px;
  color: #111827;
  font-size: 16px;
  font-weight: 750;
  line-height: 1.35;
}

.drawer-section-title::before {
  content: '';
  width: 4px;
  height: 18px;
  border-radius: 999px;
  background: #111827;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 14px;
  row-gap: 2px;
}

.rule-drawer-form :deep(.el-form-item__label) {
  margin-bottom: 6px;
  color: #344054;
  font-size: 13px;
  font-weight: 650;
  line-height: 1.35;
}

.rule-drawer-form :deep(.el-input__wrapper),
.rule-drawer-form :deep(.el-select__wrapper),
.rule-drawer-form :deep(.el-textarea__inner),
.rule-drawer-form :deep(.el-input-number .el-input__wrapper) {
  border-radius: 7px;
  box-shadow: 0 0 0 1px #d8dee9 inset;
}

.rule-drawer-form :deep(.el-input__wrapper:hover),
.rule-drawer-form :deep(.el-select__wrapper:hover),
.rule-drawer-form :deep(.el-textarea__inner:hover) {
  box-shadow: 0 0 0 1px #b8c4d6 inset;
}

.form-suffix {
  flex: 0 0 auto;
  color: #667085;
  font-size: 13px;
  white-space: nowrap;
}

.inline-number-field {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-width: 0;
}

.inline-number-field :deep(.el-input-number) {
  flex: 1 1 auto;
  width: auto;
  min-width: 0;
}

.source-type-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(130px, 1fr));
  gap: 12px;
}

.source-type-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  position: relative;
  min-height: 118px;
  padding: 14px 12px;
  border: 1px solid #dedede;
  border-radius: 8px;
  background: #fff;
  color: #344054;
  cursor: pointer;
  text-align: center;
  outline: none;
  transition: border-color .16s ease, background-color .16s ease, box-shadow .16s ease;
}

.source-type-card.active,
.source-type-card:hover,
.source-type-card:focus-visible {
  border-color: #1677ff;
  background: #f7fbff;
  box-shadow: 0 0 0 1px rgba(22, 119, 255, .12);
}

.source-type-logo {
  width: 54px;
  height: 54px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.source-type-logo svg {
  width: 100%;
  height: 100%;
  display: block;
}

.source-type-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.source-type-card strong {
  color: #111827;
  font-size: 14px;
  font-weight: 650;
  line-height: 1.2;
}

.source-type-card em {
  color: #667085;
  font-style: normal;
  font-size: 12px;
  line-height: 1.35;
}

.source-type-check {
  position: absolute;
  top: 10px;
  right: 10px;
  display: none;
  color: #2563eb;
}

.source-type-card.active .source-type-check {
  display: inline-flex;
}

.source-select {
  margin-top: 16px;
}

.es-index-select-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  width: 100%;
}

.es-index-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  min-width: 0;
}

.es-index-option span {
  min-width: 0;
  overflow: hidden;
  color: #111827;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.es-index-option em {
  flex-shrink: 0;
  color: #98a2b3;
  font-size: 12px;
  font-style: normal;
}

.query-editor {
  position: relative;
  width: 100%;
}

.es-dsl-helper {
  margin-bottom: 10px;
  padding: 10px 12px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fbfcff;
}

.es-dsl-helper-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 9px;
}

.es-dsl-helper-head span {
  color: #111827;
  font-size: 13px;
  font-weight: 760;
}

.es-dsl-helper-head em {
  color: #667085;
  font-size: 12px;
  font-style: normal;
}

.es-dsl-template-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.es-dsl-template-row :deep(.el-button) {
  margin-left: 0;
}

.query-tools {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 10px;
}

.preview-query-btn {
  min-width: 126px;
  min-height: 36px;
  border-color: #c7d7fe;
  background: #fff !important;
  color: #1d4ed8 !important;
  font-weight: 650;
}

.preview-query-btn:hover,
.preview-query-btn:focus {
  border-color: #2563eb;
  background: #eff6ff !important;
  color: #1d4ed8 !important;
}

.preview-query-btn :deep(span),
.preview-query-btn :deep(.el-icon) {
  color: inherit;
}

.query-source-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: #667085;
  font-size: 13px;
}

.query-source-hint span:last-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.suggestion-popover {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 6px);
  z-index: 20;
  max-height: 260px;
  overflow: auto;
  border: 1px solid #d8dee9;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 12px 30px rgba(15, 23, 42, .12);
}

.suggestion-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  width: 100%;
  min-height: 48px;
  padding: 9px 12px;
  border: 0;
  border-bottom: 1px solid #f2f4f7;
  background: #fff;
  color: #344054;
  cursor: pointer;
  text-align: left;
  outline: none;
}

.suggestion-item:hover,
.suggestion-item:focus-visible {
  background: #f8fafc;
}

.suggestion-item span {
  min-width: 0;
}

.suggestion-item strong,
.suggestion-item em,
.suggestion-item small {
  display: block;
}

.suggestion-item strong {
  color: #111827;
  font-size: 13px;
}

.suggestion-item em,
.suggestion-item small {
  color: #667085;
  font-size: 12px;
  font-style: normal;
}

.suggestion-item small {
  max-width: 48%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: right;
}

.condition-editor {
  border: 1px solid #dedede;
  border-radius: 7px;
  overflow: hidden;
  background: #fff;
}

.condition-head {
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid #f0f0f0;
  background: #fff;
  color: #111827;
  font-size: 14px;
  font-weight: 650;
}

.condition-grid-head,
.condition-row {
  display: grid;
  grid-template-columns: 150px 130px minmax(180px, 1fr) minmax(220px, 1fr) 54px;
  align-items: center;
  gap: 12px;
}

.condition-grid-head {
  min-height: 36px;
  padding: 0 12px;
  border-bottom: 1px solid #f2f4f7;
  background: #fff;
  color: #667085;
  font-size: 12px;
  font-weight: 650;
}

.condition-row {
  padding: 12px 14px;
  border-bottom: 1px solid #f5f5f5;
}

.notice-target-tip {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 14px;
  padding: 10px 12px;
  border: 1px solid #f0f0f0;
  border-radius: 7px;
  background: #fafafa;
  color: #111827;
}

.notice-target-tip strong,
.notice-target-tip span {
  display: block;
}

.notice-target-tip strong {
  color: #111827;
  font-size: 13px;
}

.notice-target-tip span {
  margin-top: 3px;
  color: #667085;
  font-size: 12px;
  line-height: 1.5;
}

.condition-row:last-child {
  border-bottom: 0;
}

.condition-severity {
  width: 100%;
}

.condition-op {
  width: 100%;
}

.condition-number {
  flex: 1 1 auto;
  width: auto;
  min-width: 0;
}

.condition-duration {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-width: 0;
}

.condition-unit {
  flex-shrink: 0;
  color: #667085;
  font-size: 13px;
}

.condition-actions {
  display: flex;
  justify-content: center;
}

.event-field-grid {
  display: grid;
  grid-template-columns: minmax(280px, .8fr) minmax(0, 1.2fr);
  gap: 14px;
  margin-top: 4px;
}

.kv-editor,
.detail-template-editor,
.callback-editor {
  padding: 14px;
  border: 1px solid #dedede;
  border-radius: 7px;
  background: #fff;
}

.callback-editor {
  margin-top: 12px;
}

.detail-template-editor {
  min-width: 0;
}

.kv-head,
.kv-row {
  display: flex;
  align-items: center;
}

.kv-head {
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
  color: #111827;
  font-size: 13px;
  font-weight: 750;
}

.kv-head small {
  display: block;
  margin-top: 3px;
  color: #667085;
  font-size: 12px;
  font-weight: 400;
  line-height: 1.4;
}

.required-title::before {
  content: '*';
  margin-right: 4px;
  color: #f56c6c;
  font-weight: 700;
}

.detail-template-form-item {
  margin-bottom: 10px;
}

.detail-template-form-item :deep(.el-form-item__content) {
  width: 100%;
}

.detail-template-editor :deep(.el-textarea__inner) {
  min-height: 118px !important;
  resize: vertical;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  line-height: 1.55;
}

.template-token-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.template-token-pill {
  display: inline-flex;
  align-items: flex-start;
  gap: 7px;
  max-width: 100%;
  min-width: 0;
  padding: 5px 8px;
  border: 1px solid #dbe8ff;
  border-radius: 6px;
  background: #f8fbff;
  color: #1f2937;
  cursor: help;
  line-height: 1.2;
}

.template-token-pill code {
  color: #1d4ed8;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  line-height: 1.35;
  overflow-wrap: anywhere;
  white-space: normal;
}

.template-token-pill em {
  flex-shrink: 0;
  padding-left: 7px;
  border-left: 1px solid #dbe8ff;
  color: #667085;
  font-size: 12px;
  font-style: normal;
  white-space: nowrap;
}

.detail-preview-box {
  margin-top: 12px;
  overflow: hidden;
  border: 1px solid #e5e7eb;
  border-radius: 7px;
  background: #fbfcff;
}

.detail-preview-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 11px;
  border-bottom: 1px solid #edf1f7;
  background: #fff;
}

.detail-preview-head span:first-child {
  color: #111827;
  font-size: 13px;
  font-weight: 700;
}

.detail-preview-body {
  display: grid;
  gap: 8px;
  max-height: 280px;
  overflow: auto;
  padding: 11px;
}

.detail-preview-text,
.detail-preview-code {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  line-height: 1.65;
}

.detail-preview-text {
  color: #1f2937;
}

.detail-preview-code {
  padding: 10px 11px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: #111827;
  color: #e5e7eb;
}

.kv-row {
  gap: 8px;
}

.kv-row + .kv-row,
.callback-row + .callback-row {
  margin-top: 8px;
}

.kv-row .el-input:first-child {
  flex: 0 0 38%;
}

.kv-row .el-input:nth-child(2) {
  min-width: 0;
  flex: 1;
}

.callback-row {
  display: grid;
  grid-template-columns: minmax(150px, .8fr) minmax(160px, .9fr) 120px 150px minmax(320px, 1.8fr) 32px;
  gap: 8px;
  align-items: center;
}

.callback-query-input {
  min-width: 0;
}

.callback-range-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 28px;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.callback-range-field span {
  color: #667085;
  font-size: 13px;
}

.callback-row :deep(.el-input-number) {
  width: 100%;
}

.kv-editor :deep(.el-empty),
.callback-editor :deep(.el-empty) {
  padding: 8px 0;
}

.eval-result-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.eval-summary-card {
  display: flex;
  align-items: stretch;
  justify-content: space-between;
  gap: 18px;
  padding: 18px;
  border: 1px solid #e5e9f2;
  border-radius: 10px;
  background: linear-gradient(180deg, #fff, #fafbfc);
}

.eval-summary-main {
  min-width: 0;
}

.eval-summary-main h3 {
  margin: 12px 0 8px;
  color: #111827;
  font-size: 20px;
  line-height: 1.35;
}

.eval-summary-main p {
  margin: 0;
  color: #667085;
  font-size: 14px;
  line-height: 1.7;
}

.eval-summary-value {
  min-width: 150px;
  padding: 14px 16px;
  border: 1px solid #e8edf5;
  border-radius: 10px;
  background: #fff;
  text-align: right;
}

.eval-summary-value span,
.eval-meta-grid span,
.eval-samples-head span {
  display: block;
  color: #667085;
  font-size: 13px;
}

.eval-summary-value strong {
  display: block;
  margin-top: 8px;
  color: #111827;
  font-size: 28px;
  line-height: 1;
}

.eval-meta-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.eval-meta-grid > div {
  min-width: 0;
  padding: 14px;
  border: 1px solid #e8edf5;
  border-radius: 10px;
  background: #fff;
}

.eval-meta-grid strong {
  display: block;
  margin-top: 8px;
  overflow: hidden;
  color: #111827;
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.eval-meta-grid small {
  display: block;
  margin-top: 6px;
  color: #98a2b3;
  font-size: 12px;
}

.eval-samples-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 2px;
}

.eval-samples-head strong {
  display: block;
  margin-bottom: 4px;
  color: #111827;
  font-size: 16px;
}

.eval-sample-table {
  border: 1px solid #edf0f5;
  border-radius: 10px;
  overflow: hidden;
}

.eval-sample-list {
  display: grid;
  max-height: 440px;
  padding-right: 4px;
  overflow: auto;
  gap: 10px;
}

.eval-sample-card {
  display: grid;
  grid-template-columns: minmax(250px, 0.75fr) minmax(360px, 1.45fr) minmax(240px, 0.8fr);
  gap: 14px;
  padding: 14px;
  border: 1px solid #e8edf5;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 8px 20px rgba(16, 24, 40, 0.04);
}

.eval-sample-card.is-matched {
  border-color: #ffd2d2;
  background: #fffafa;
}

.eval-sample-main,
.eval-sample-labels,
.eval-sample-message,
.eval-sample-logs {
  min-width: 0;
}

.eval-sample-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.eval-sample-title strong {
  color: #111827;
  font-size: 18px;
  font-weight: 760;
  line-height: 1.2;
}

.sample-index {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  background: #f2f4f7;
  color: #667085;
  font-size: 12px;
  font-weight: 700;
}

.eval-sample-condition {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;
  margin-top: 12px;
  color: #667085;
  font-size: 13px;
}

.eval-sample-condition b {
  color: #344054;
  font-size: 14px;
}

.eval-sample-condition em {
  font-style: normal;
  color: #98a2b3;
}

.sample-section-title {
  display: block;
  margin-bottom: 8px;
  color: #667085;
  font-size: 12px;
  font-weight: 700;
}

.eval-value-text {
  color: #111827;
  font-size: 15px;
}

.eval-labels {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 8px;
}

.eval-label-chip {
  display: inline-grid;
  grid-template-columns: auto minmax(80px, auto);
  max-width: 100%;
  overflow: hidden;
  border: 1px solid #d8e0ee;
  border-radius: 7px;
  background: #fbfcff;
  color: #344054;
  font-size: 12px;
  line-height: 1.4;
}

.eval-label-chip b {
  min-width: 0;
  padding: 5px 7px;
  overflow-wrap: anywhere;
  border-right: 1px solid #d8e0ee;
  background: #f3f6fb;
  color: #475467;
  font-weight: 700;
}

.eval-label-chip em {
  min-width: 0;
  padding: 5px 7px;
  overflow-wrap: anywhere;
  color: #111827;
  font-style: normal;
  font-weight: 600;
}

.eval-message {
  color: #475467;
  line-height: 1.6;
}

.eval-sample-message p {
  margin: 0;
  color: #475467;
  font-size: 13px;
  line-height: 1.65;
  overflow-wrap: anywhere;
}

.eval-sample-logs {
  grid-column: 1 / -1;
}

.eval-sample-logs pre {
  max-height: 180px;
  margin: 0;
  padding: 12px;
  overflow: auto;
  border: 1px solid #fde2e2;
  border-radius: 8px;
  background: #fff7f7;
  color: #7f1d1d;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.eval-empty-text {
  color: #98a2b3;
  font-size: 13px;
}

@media (max-width: 1180px) {
  .eval-sample-card {
    grid-template-columns: 1fr;
  }
}

.preview-dialog-head {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.preview-title-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  width: 38px;
  height: 38px;
  border: 1px solid #dbe8ff;
  border-radius: 8px;
  background: #eef4ff;
  color: #2563eb;
  font-size: 18px;
}

.preview-dialog-head h3 {
  margin: 0;
  color: #111827;
  font-size: 18px;
  font-weight: 760;
}

.preview-dialog-head p {
  max-width: 720px;
  margin: 4px 0 0;
  overflow: hidden;
  color: #667085;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-meta {
  display: grid;
  grid-template-columns: 1.3fr 1fr 1fr 1fr;
  gap: 10px;
  margin-bottom: 14px;
}

.preview-meta > div {
  min-width: 0;
  padding: 12px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fbfcfe;
}

.preview-meta span,
.preview-card-value span,
.preview-card-value small {
  display: block;
  color: #667085;
  font-size: 12px;
}

.preview-meta strong {
  display: block;
  min-width: 0;
  margin-top: 5px;
  overflow: hidden;
  color: #111827;
  font-size: 16px;
  font-weight: 750;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-alert {
  margin-bottom: 12px;
}

.preview-tabs :deep(.el-tabs__header) {
  margin-bottom: 12px;
}

.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.preview-tab-panel {
  min-height: 420px;
}

.preview-card-list {
  display: grid;
  gap: 12px;
  max-height: 520px;
  overflow: auto;
  padding-right: 4px;
}

.preview-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 160px;
  gap: 18px;
  min-height: 126px;
  padding: 16px;
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 20px rgba(15, 23, 42, .04);
}

.preview-card-left {
  min-width: 0;
}

.preview-card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding-bottom: 12px;
  border-bottom: 1px solid #edf1f7;
  color: #2563eb;
}

.preview-card-title strong {
  min-width: 0;
  overflow: hidden;
  color: #111827;
  font-size: 15px;
  font-weight: 760;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-labels {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;
  margin-top: 14px;
}

.preview-labels span {
  max-width: 100%;
  padding: 4px 8px;
  border: 1px solid #dbe8ff;
  border-radius: 6px;
  background: #f8fbff;
  color: #344054;
  font-size: 12px;
  line-height: 1.4;
  overflow-wrap: anywhere;
  white-space: normal;
}

.preview-labels b {
  color: #111827;
}

.preview-labels em {
  color: #98a2b3;
  font-size: 12px;
  font-style: normal;
}

.preview-card-value {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  justify-content: center;
  min-width: 0;
  text-align: right;
}

.preview-card-value strong {
  display: block;
  width: 100%;
  margin: 8px 0;
  overflow: hidden;
  color: #2563eb;
  font-size: 30px;
  font-weight: 800;
  line-height: 1.1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-card-value small {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-chart {
  width: 100%;
  height: 456px;
  border: 1px solid #e8e8e8;
  border-radius: 4px;
  background: #fff;
}

.preview-series-table {
  margin-top: 12px;
  max-height: 360px;
  overflow: hidden;
  overflow-y: auto;
  border: 1px solid #e8e8e8;
  border-radius: 4px;
  background: #fff;
}

.preview-series-head,
.preview-series-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 260px;
  align-items: center;
  gap: 16px;
}

.preview-series-head {
  position: sticky;
  top: 0;
  z-index: 1;
  min-height: 40px;
  padding: 0 14px;
  border-bottom: 1px solid #e8e8e8;
  background: #fafafa;
  color: #262626;
  font-size: 13px;
  font-weight: 600;
}

.preview-series-row {
  min-height: 44px;
  padding: 8px 14px;
  border-bottom: 1px solid #f0f0f0;
}

.preview-series-row:last-child {
  border-bottom: 0;
}

.preview-series-labels {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;
  min-width: 0;
}

.preview-series-labels span {
  display: inline-flex;
  align-items: flex-start;
  max-width: 100%;
  padding: 3px 8px;
  border: 1px solid #d9ecff;
  border-radius: 4px;
  background: #f5faff;
  color: #262626;
  font-size: 12px;
  line-height: 1.25;
  overflow-wrap: anywhere;
  white-space: normal;
}

.preview-series-labels b {
  flex-shrink: 0;
  margin-right: 4px;
  color: #111827;
}

.preview-series-labels em {
  color: #667085;
  font-size: 13px;
  font-style: normal;
}

.preview-series-value {
  display: flex;
  align-items: baseline;
  justify-content: flex-start;
  gap: 10px;
  min-width: 0;
  color: #667085;
}

.preview-series-value strong {
  color: #262626;
  font-size: 15px;
  font-weight: 600;
}

.preview-series-value small {
  color: #98a2b3;
  font-size: 12px;
}

.preview-series-more {
  padding: 10px 14px;
  border-top: 1px solid #f0f0f0;
  background: #fafafa;
  color: #667085;
  font-size: 12px;
}

:global(.watchalert-preview-tooltip) {
  pointer-events: auto;
  line-height: 1.45;
}

:global(.rule-group-dropdown) {
  min-width: 136px;
}

:global(.rule-group-dropdown .el-dropdown-menu) {
  padding: 6px;
}

:global(.rule-group-dropdown .el-dropdown-menu__item) {
  height: 32px;
  border-radius: 6px;
  color: #344054;
  font-size: 13px;
}

:global(.rule-group-dropdown .el-dropdown-menu__item .el-icon) {
  margin-right: 6px;
  color: #667085;
}

:global(.rule-group-dropdown .el-dropdown-menu__item:hover) {
  background: #f0f5ff;
  color: #1d4ed8;
}

:global(.rule-group-dropdown .el-dropdown-menu__item:hover .el-icon) {
  color: #1d4ed8;
}

:global(.rule-group-dropdown .danger-item) {
  color: #cf1322;
}

:global(.rule-group-dropdown .danger-item .el-icon) {
  color: #cf1322;
}

:global(.rule-group-dropdown .danger-item:hover) {
  background: #fff1f0;
  color: #cf1322;
}

:deep(.rule-drawer .el-drawer__header),
:deep(.rule-dialog .el-dialog__header),
:deep(.group-dialog .el-dialog__header) {
  margin-bottom: 0;
  padding-bottom: 14px;
  border-bottom: 1px solid #edf1f7;
  background: #fbfcfe;
}

:global(.rule-drawer .el-drawer__header) {
  margin-bottom: 0;
  padding: 18px 28px 16px;
  border-bottom: 1px solid #ededed;
  background: #fff;
}

:global(.rule-drawer .el-drawer__body) {
  padding: 0 28px 18px;
  background: #fff;
}

:global(.rule-drawer .el-drawer__footer) {
  padding: 12px 28px;
}

:deep(.rule-drawer .el-drawer__footer),
:deep(.rule-dialog .el-dialog__footer),
:deep(.group-dialog .el-dialog__footer) {
  border-top: 1px solid #ededed;
  background: #fff;
}

:global(.rule-drawer .el-drawer__title) {
  color: #111827;
  font-size: 20px;
  font-weight: 760;
}

.rule-drawer-form :deep(.el-form-item__label) {
  min-height: 20px;
  padding-bottom: 6px;
  color: #344054;
  font-size: 13px;
  font-weight: 650;
}

.rule-drawer-form :deep(.el-input__wrapper),
.rule-drawer-form :deep(.el-textarea__inner),
.rule-drawer-form :deep(.el-select__wrapper) {
  border-radius: 7px;
  box-shadow: 0 0 0 1px #d8dee9 inset;
}

.rule-drawer-form :deep(.el-input__wrapper:hover),
.rule-drawer-form :deep(.el-textarea__inner:hover),
.rule-drawer-form :deep(.el-select__wrapper:hover) {
  box-shadow: 0 0 0 1px #b8c4d6 inset;
}

.rule-drawer-form :deep(.el-input__wrapper.is-focus),
.rule-drawer-form :deep(.el-select__wrapper.is-focused) {
  box-shadow: 0 0 0 1px #1677ff inset;
}

.monitor-rules-page {
  gap: 12px;
  background: #fff;
}

.monitor-rules-page .stats-cards {
  display: none;
}

.monitor-rules-page .page-header {
  min-height: 42px;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
}

.monitor-rules-page .page-title-group {
  gap: 0;
}

.monitor-rules-page .page-title-icon,
.monitor-rules-page .page-subtitle {
  display: none;
}

.monitor-rules-page .page-title {
  font-size: 14px;
  font-weight: 500;
}

.monitor-rules-page .header-actions .el-button,
.monitor-rules-page .search-bar .el-button {
  min-height: 34px;
  border-radius: 6px;
}

.monitor-rules-page .header-actions .el-button--primary {
  border-color: #000;
  background: #000;
}

.monitor-rules-page .rule-workspace {
  grid-template-columns: 210px minmax(0, 1fr);
  border-color: #f0f0f0;
  border-radius: 8px;
}

.monitor-rules-page .group-sidebar {
  padding: 12px;
  border-right-color: #f0f0f0;
  background: #fff;
}

.monitor-rules-page .sidebar-head {
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 600;
}

.monitor-rules-page .group-item {
  min-height: 38px;
  border-radius: 8px;
  font-size: 13px;
}

.monitor-rules-page .group-tail {
  min-width: 58px;
}

.monitor-rules-page .group-item.active,
.monitor-rules-page .group-item:hover,
.monitor-rules-page .group-item:focus-visible {
  border-color: #d6e4ff;
  background: #f5f8ff;
  color: #1d4ed8;
}

.monitor-rules-page .rules-main {
  padding: 12px;
}

.monitor-rules-page .search-bar {
  margin-bottom: 12px;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: #fff;
}

.monitor-rules-page .table-wrapper,
.monitor-rules-page .events-section {
  border-color: #f0f0f0;
  border-radius: 8px;
}

.monitor-rules-page .modern-table :deep(.el-table__header th) {
  background: #fafafa !important;
  color: #111827;
  font-size: 14px;
  font-weight: 600;
}

.monitor-rules-page .modern-table :deep(.el-table__body td) {
  padding: 7px 0;
  font-size: 13px;
}

@media (max-width: 1180px) {
  .rule-workspace {
    grid-template-columns: 1fr;
  }

  .group-sidebar {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    border-right: 0;
    border-bottom: 1px solid #edf1f7;
  }

  .sidebar-head {
    min-width: 90px;
    margin-bottom: 0;
  }

  .group-item {
    width: auto;
    min-width: 140px;
  }

  .source-type-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .rule-drawer-shell {
    grid-template-columns: 1fr;
  }

  .rule-drawer-nav {
    position: static;
    flex-direction: row;
    overflow-x: auto;
  }

  .rule-drawer-nav a {
    flex: 0 0 auto;
  }
}

@media (max-width: 760px) {
  .page-header,
  .section-header,
  .search-bar,
  .table-pagination {
    flex-direction: column;
    align-items: stretch;
  }

  .stats-cards,
  .form-grid,
  .source-type-grid,
  .rule-drawer-shell,
  .event-field-grid,
  .preview-meta,
  .preview-card {
    grid-template-columns: 1fr;
  }

  .rule-drawer-shell {
    padding: 14px 12px 22px;
  }

  .query-tools {
    align-items: stretch;
    flex-direction: column;
  }

  .preview-card-value {
    align-items: flex-start;
    text-align: left;
  }

  .search-input,
  .search-input.small {
    width: 100%;
  }

  .batch-toolbar,
  .batch-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .batch-field {
    grid-template-columns: 1fr;
  }

  .condition-grid-head {
    display: none;
  }

  .condition-row {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .condition-actions {
    justify-content: flex-start;
  }

  .callback-row {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .preview-series-head,
  .preview-series-row {
    grid-template-columns: 1fr;
    align-items: stretch;
    gap: 8px;
  }
}
</style>
