<template>
  <div class="fault-detail-page">
    <section class="fault-identity" v-loading="loading">
      <div class="breadcrumb-line">
        <el-button link class="back-icon" @click="router.push('/monitor/fault-centers')">
          <el-icon><Back /></el-icon>
        </el-button>
        <el-icon class="home-icon"><House /></el-icon>
        <span>/ 故障中心 /</span>
        <strong>详情</strong>
      </div>

      <div class="identity-grid">
        <div class="identity-item">
          <span>ID：</span>
          <strong>{{ centerCode }}</strong>
        </div>
        <div class="identity-item identity-center">
          <span>名称：</span>
          <div v-if="editingBasicField === 'name'" class="inline-edit-field">
            <el-input v-model="basicFieldValue" />
            <el-button link class="inline-save" title="保存" @click="saveInlineBasicField">
              <el-icon><Check /></el-icon>
            </el-button>
            <el-button link class="inline-save" title="取消" @click="cancelInlineBasicField">
              <el-icon><Close /></el-icon>
            </el-button>
          </div>
          <template v-else>
            <strong>{{ center?.name || '-' }}</strong>
            <el-button link class="inline-edit" title="编辑名称" @click="beginInlineBasicEdit('name')">
              <el-icon><Edit /></el-icon>
            </el-button>
          </template>
        </div>
        <div class="identity-item identity-right">
          <span>描述：</span>
          <div v-if="editingBasicField === 'description'" class="inline-edit-field">
            <el-input v-model="basicFieldValue" />
            <el-button link class="inline-save" title="保存" @click="saveInlineBasicField">
              <el-icon><Check /></el-icon>
            </el-button>
            <el-button link class="inline-save" title="取消" @click="cancelInlineBasicField">
              <el-icon><Close /></el-icon>
            </el-button>
          </div>
          <template v-else>
            <strong>{{ center?.description || '-' }}</strong>
            <el-button link class="inline-edit" title="编辑描述" @click="beginInlineBasicEdit('description')">
              <el-icon><Edit /></el-icon>
            </el-button>
          </template>
        </div>
      </div>

      <div class="mini-stats">
        <div class="mini-stat">
          <span>预告警</span>
          <strong>{{ center?.currentPreAlertNumber || 0 }}</strong>
        </div>
        <div class="mini-stat danger">
          <span>告警中</span>
          <strong>{{ center?.currentAlertNumber || 0 }}</strong>
        </div>
        <div class="mini-stat success">
          <span>已恢复</span>
          <strong>{{ center?.currentRecoverNumber || 0 }}</strong>
        </div>
        <div class="mini-stat">
          <span>恢复等待</span>
          <strong>{{ center?.recoverWaitSeconds || 30 }}s</strong>
        </div>
        <div class="identity-actions">
          <el-button @click="loadAll">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
          <el-button type="danger" plain @click="deleteCenter">
            <el-icon><Delete /></el-icon>
            删除
          </el-button>
        </div>
      </div>
    </section>

    <section class="slo-card-grid">
      <article class="watch-chart-card">
        <div class="chart-card-head">
          <div>
            <h3>平均修复时间 (MTTR)</h3>
            <span>Mean Time To Repair</span>
            <p>7日平均: {{ formatDuration(averageSlo('mttr')) }}</p>
          </div>
          <el-icon class="metric-icon repair"><Operation /></el-icon>
        </div>
        <div ref="mttrChartRef" class="single-slo-chart"></div>
      </article>

      <article class="watch-chart-card">
        <div class="chart-card-head">
          <div>
            <h3>平均响应时间 (MTTA)</h3>
            <span>Mean Time To Acknowledge</span>
            <p>7日平均: {{ formatDuration(averageSlo('mtta')) }}</p>
          </div>
          <el-icon class="metric-icon response"><Clock /></el-icon>
        </div>
        <div ref="mttaChartRef" class="single-slo-chart"></div>
      </article>
    </section>

    <el-tabs
      v-model="activeTab"
      class="watch-tabs"
      @pointerdown.capture="captureDetailScroll"
      @mousedown.capture="captureDetailScroll"
      @touchstart.capture="captureDetailScroll"
      @tab-change="handleTabChange"
      @tab-click="restoreDetailScroll"
    >
      <el-tab-pane label="活跃告警" name="active">
        <section class="event-board">
          <div class="watch-filter-bar">
            <el-input
              v-model="eventKeyword"
              placeholder="输入搜索关键字"
              clearable
              class="keyword-input"
              @clear="refreshEventList"
              @keyup.enter="refreshEventList"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
            <el-select v-model="activeFilters.dataSourceType" clearable placeholder="数据源类型" @change="refreshActiveEvents">
              <el-option v-for="item in dataSourceOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-select v-model="activeFilters.severity" clearable placeholder="告警等级" @change="refreshActiveEvents">
              <el-option v-for="item in severityOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-select v-model="activeFilters.state" clearable placeholder="事件状态" @change="refreshActiveEvents">
              <el-option v-for="item in eventStateOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-button @click="loadEvents">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
            <el-button @click="openExportDialog('active')">
              <el-icon><Download /></el-icon>
              导出
            </el-button>
            <el-dropdown trigger="click" :disabled="!selectedActiveEventIds.length" @command="handleActiveBatchCommand">
              <el-button class="batch-action" :disabled="!selectedActiveEventIds.length">
                批量操作
                <el-icon><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="ack">批量认领</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>批量删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>

          <el-table
            :data="activeEvents"
            v-loading="eventsLoading"
            class="watch-table"
            row-key="id"
            :header-cell-style="tableHeaderStyle"
            @selection-change="handleActiveSelectionChange"
          >
            <el-table-column type="selection" width="54" />
            <el-table-column label="事件信息" min-width="430">
              <template #default="{ row }">
                <div class="event-info-cell" :class="row.state">
                  <span class="event-rail"></span>
                  <span class="event-flame">{{ getSeverityMark(row.severity) }}</span>
                  <div class="event-copy">
	                    <button class="event-title" type="button" @click="openEventDetail(row)">{{ row.ruleName }}</button>
                    <p>{{ formatFriendlyEventMessage(row.message) }}</p>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="触发时间" width="220">
              <template #default="{ row }">
                <span class="time-text">{{ formatDateTime(row.startedAt) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="持续时长" width="250" sortable>
              <template #default="{ row }">
                <div class="duration-cell">
                  <div class="duration-text">
                    <el-icon><Clock /></el-icon>
                    <strong>{{ formatDuration(getEventDurationSeconds(row)) }}</strong>
                  </div>
                  <div class="duration-bars">
                    <span
                      v-for="index in 10"
                      :key="index"
                      :class="{ active: index <= durationBlockCount(row) }"
                    ></span>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="事件状态" width="140">
              <template #default="{ row }">
                <el-tag class="state-tag" :type="getStateTag(row.state)" effect="plain">{{ getStateName(row.state) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="认领人" width="150">
              <template #default="{ row }">
                <el-tag v-if="row.acknowledged" type="success" effect="plain">{{ row.acknowledgedBy || '已认领' }}</el-tag>
                <el-tag v-else effect="plain">未认领</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="92" fixed="right">
              <template #default="{ row }">
                <el-dropdown trigger="click" @command="handleEventCommand($event, row)">
                  <el-button link class="more-btn">
                    <el-icon><MoreFilled /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item v-if="!row.acknowledged" command="ack">认领它</el-dropdown-item>
                      <el-dropdown-item command="silence">静默它</el-dropdown-item>
                      <el-dropdown-item divided command="delete">删除它</el-dropdown-item>
                      <el-dropdown-item disabled>AI 分析</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </template>
            </el-table-column>
          </el-table>

          <div class="table-footer">
            <span>共 {{ activePager.total }} 条</span>
            <el-pagination
              v-model:current-page="activePager.page"
              v-model:page-size="activePager.pageSize"
              layout="sizes, prev, pager, next, jumper"
              :total="activePager.total"
              :page-sizes="[10, 20, 50, 100]"
              @current-change="loadActiveEvents"
              @size-change="handleActiveSizeChange"
            />
          </div>
        </section>
      </el-tab-pane>

      <el-tab-pane label="历史告警" name="history">
        <section class="event-board">
          <div class="watch-filter-bar">
            <el-input
              v-model="eventKeyword"
              placeholder="输入搜索关键字"
              clearable
              class="keyword-input"
              @clear="refreshEventList"
              @keyup.enter="refreshEventList"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
            <el-select v-model="historyFilters.dataSourceType" clearable placeholder="数据源类型" @change="refreshHistoryEvents">
              <el-option v-for="item in dataSourceOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-select v-model="historyFilters.severity" clearable placeholder="告警等级" @change="refreshHistoryEvents">
              <el-option v-for="item in severityOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-date-picker
              v-model="historyDateRange"
              type="daterange"
              range-separator="→"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              value-format="YYYY-MM-DD"
              class="date-range"
              @change="refreshHistoryEvents"
            />
            <el-button @click="loadHistoryEvents">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
            <el-button @click="openExportDialog('history')">
              <el-icon><Download /></el-icon>
              导出
            </el-button>
          </div>

          <el-table
            :data="historyEvents"
            v-loading="eventsLoading"
            class="watch-table"
            row-key="id"
            :header-cell-style="tableHeaderStyle"
          >
            <el-table-column label="事件信息" min-width="450">
              <template #default="{ row }">
                <div class="event-info-cell recovered">
                  <span class="event-rail"></span>
                  <span class="event-flame">{{ getSeverityMark(row.severity) }}</span>
                  <div class="event-copy">
	                    <button class="event-title" type="button" @click="openEventDetail(row)">{{ row.ruleName }}</button>
                    <p>{{ formatFriendlyEventMessage(row.message) }}</p>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="告警时间" width="240">
              <template #default="{ row }">
                <div class="time-stack">
                  <strong>{{ formatDateTime(row.startedAt) }}</strong>
                  <span>{{ formatDateTime(row.endedAt) }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="持续时长" width="250" sortable>
              <template #default="{ row }">
                <div class="duration-cell">
                  <div class="duration-text">
                    <el-icon><Clock /></el-icon>
                    <strong>{{ formatDuration(getEventDurationSeconds(row)) }}</strong>
                  </div>
                  <div class="duration-bars">
                    <span
                      v-for="index in 10"
                      :key="index"
                      :class="{ active: index <= durationBlockCount(row) }"
                    ></span>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="事件状态" width="150">
              <template #default="{ row }">
                <el-tag class="state-tag" :type="getStateTag(row.state)" effect="plain">{{ getStateName(row.state) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="处理人" width="150">
              <template #default="{ row }">
                <el-tag effect="plain">{{ row.acknowledgedBy || '自动恢复' }}</el-tag>
              </template>
            </el-table-column>
          </el-table>

          <div class="table-footer">
            <span>共 {{ historyPager.total }} 条</span>
            <el-pagination
              v-model:current-page="historyPager.page"
              v-model:page-size="historyPager.pageSize"
              layout="sizes, prev, pager, next, jumper"
              :total="historyPager.total"
              :page-sizes="[10, 20, 50, 100]"
              @current-change="loadHistoryEvents"
              @size-change="handleHistorySizeChange"
            />
          </div>
        </section>
      </el-tab-pane>

      <el-tab-pane label="降噪配置" name="silence">
        <section class="config-section">
          <div class="aggregation-row">
            <div class="section-title">
              <el-icon><Connection /></el-icon>
              <strong>告警聚合</strong>
            </div>
            <el-radio-group v-model="configForm.aggregationType" @change="saveConfig">
              <el-radio label="Rule">相同规则聚合</el-radio>
              <el-radio label="None">不聚合</el-radio>
            </el-radio-group>
          </div>

          <div class="rule-list-head watch">
            <div>
              <div class="section-title">
                <el-icon><Timer /></el-icon>
                <strong>静默规则</strong>
              </div>
              <div class="silence-filter">
                <el-radio-group v-model="silenceStatusFilter" class="watch-radio-buttons">
                  <el-radio-button v-for="item in silenceStatusOptions" :key="item.value" :label="item.value">
                    {{ item.label }}
                  </el-radio-button>
                </el-radio-group>
                <el-input v-model="silenceKeyword" placeholder="搜索规则名称" clearable>
                  <template #prefix><el-icon><Search /></el-icon></template>
                </el-input>
              </div>
            </div>
            <el-button type="primary" class="dark-action" @click="addSilenceRule">
              <el-icon><Plus /></el-icon>
              添加新规则
            </el-button>
          </div>

          <el-table :data="filteredSilenceRules" class="watch-table config-table" :header-cell-style="tableHeaderStyle">
            <el-table-column label="规则名称" min-width="180">
              <template #default="{ row }">
                <button class="silence-name-link" type="button" @click="editSilenceRule(row)">{{ row.name || '-' }}</button>
              </template>
            </el-table-column>
            <el-table-column label="标签" min-width="300">
              <template #default="{ row }">
                <div class="silence-tags">
                  <el-tag
                    v-for="tag in splitMatcherTags(row.matcher)"
                    :key="`${row.key}-${tag}`"
                    type="primary"
                    effect="light"
                    size="small"
                  >
                    {{ tag }}
                  </el-tag>
                  <span v-if="!splitMatcherTags(row.matcher).length">-</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="时间范围" min-width="230">
              <template #default="{ row }">
                <span class="time-text small">{{ row.effectiveTime || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作人" width="150">
              <template #default>
                <div class="operator-cell">
                  <strong>admin</strong>
                  <span>{{ currentTimeText }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'" effect="light" class="pill-tag">
                  {{ row.enabled ? '启用' : '失效' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="92">
              <template #default="{ $index }">
                <el-button link class="danger-link" @click="removeSilenceRule($index)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </section>
      </el-tab-pane>

      <el-tab-pane label="通知配置" name="notify">
        <section class="config-section">
          <div class="section-heading">
            <div>
              <div class="section-title">
                <el-icon><Bell /></el-icon>
                <strong>通知配置</strong>
              </div>
              <p>告警产生和恢复时使用的默认通知对象、路由和重复通知间隔</p>
            </div>
          </div>

          <div class="notify-grid">
            <article class="setting-card compact">
              <header>
                <div class="section-title small">
                  <el-icon><Setting /></el-icon>
                  <strong>基本配置</strong>
                </div>
                <div class="card-edit-actions">
                  <template v-if="notifyBasicEditing">
                    <el-button class="edit-save-btn" :loading="saving" @click="saveNotifyBasicEdit">
                      <el-icon><Check /></el-icon>
                      保存
                    </el-button>
                    <el-button class="edit-cancel-btn" @click="cancelNotifyBasicEdit">
                      <el-icon><Close /></el-icon>
                      取消
                    </el-button>
                  </template>
                  <el-button v-else link class="inline-edit-btn" @click="beginNotifyBasicEdit">
                    <el-icon><Edit /></el-icon>
                    编辑
                  </el-button>
                </div>
              </header>
              <div class="setting-form">
                <label>
                  <span>默认通知对象</span>
                  <el-select
                    v-model="configForm.noticeObjectIds"
                    multiple
                    clearable
                    filterable
                    placeholder="选择通知对象"
                    :disabled="!notifyBasicEditing"
                  >
                    <el-option v-for="item in enabledNoticeObjects" :key="item.id" :label="item.name" :value="item.id" />
                  </el-select>
                </label>
                <label>
                  <span>重复通知间隔</span>
                  <div class="repeat-inputs">
                    <el-input-number v-model="configForm.repeat.p0" :min="1" :max="10080" controls-position="right" :disabled="!notifyBasicEditing" />
                    <em>P0 / 分钟</em>
                  </div>
                  <div class="repeat-inputs">
                    <el-input-number v-model="configForm.repeat.p1" :min="1" :max="10080" controls-position="right" :disabled="!notifyBasicEditing" />
                    <em>P1 / 分钟</em>
                  </div>
                  <div class="repeat-inputs">
                    <el-input-number v-model="configForm.repeat.p2" :min="1" :max="10080" controls-position="right" :disabled="!notifyBasicEditing" />
                    <em>P2 / 分钟</em>
                  </div>
                </label>
                <label>
                  <span>恢复等待时间</span>
                  <div class="suffix-input">
                    <el-input-number v-model="configForm.recoverWaitSeconds" :min="1" :max="86400" controls-position="right" :disabled="!notifyBasicEditing" />
                    <em>秒</em>
                  </div>
                </label>
                <label class="switch-line">
                  <span>启用恢复通知</span>
                  <el-switch v-model="configForm.recoverNotify" :disabled="!notifyBasicEditing" />
                </label>
              </div>
            </article>

            <article class="setting-card route-card">
              <header>
                <div class="section-title small">
                  <el-icon><PriceTag /></el-icon>
                  <strong>告警路由</strong>
                </div>
                <div class="card-edit-actions">
                  <template v-if="notifyRoutesEditing">
                    <el-button class="edit-save-btn" :loading="saving" @click="saveNotifyRoutesEdit">
                      <el-icon><Check /></el-icon>
                      保存
                    </el-button>
                    <el-button class="edit-cancel-btn" @click="cancelNotifyRoutesEdit">
                      <el-icon><Close /></el-icon>
                      取消
                    </el-button>
                  </template>
                  <el-button v-else link class="inline-edit-btn" @click="beginNotifyRoutesEdit">
                    <el-icon><Edit /></el-icon>
                    编辑
                  </el-button>
                </div>
              </header>
              <div v-if="!configForm.noticeRoutes.length && !notifyRoutesEditing" class="empty-route">
                <div class="empty-box"></div>
                <span>暂无告警路由配置</span>
              </div>
              <div v-else class="route-list">
                <div
                  v-for="(routeItem, index) in configForm.noticeRoutes"
                  :key="routeItem.key"
                  class="route-rule-card"
                  :class="{ 'is-readonly': !notifyRoutesEditing }"
                >
                  <div v-if="!notifyRoutesEditing && routeItem.labels.length" class="route-label-tags">
                    <span class="route-meta-title">匹配条件</span>
                    <el-tag
                      v-for="label in routeItem.labels.filter(item => item.key)"
                      :key="`${routeItem.key}-${label.key}-${label.value}`"
                      type="primary"
                      effect="plain"
                      size="small"
                    >
                      {{ label.key }} {{ label.operator }} {{ label.value }}
                    </el-tag>
                  </div>

                  <div v-if="notifyRoutesEditing" class="route-condition-list">
                    <div v-for="(label, labelIndex) in routeItem.labels" :key="`${routeItem.key}-${labelIndex}`" class="route-condition-row">
                      <el-input v-model="label.key" placeholder="标签 Key" />
                      <el-select v-model="label.operator" class="operator-select">
                        <el-option label="==" value="==" />
                        <el-option label="=~" value="=~" />
                        <el-option label="!=" value="!=" />
                        <el-option label="!~" value="!~" />
                      </el-select>
                      <el-input v-model="label.value" placeholder="标签 Value" />
                      <el-button
                        v-if="routeItem.labels.length > 1"
                        link
                        class="danger-link route-remove-btn"
                        @click="removeNoticeRouteLabel(index, labelIndex)"
                      >
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </div>
                    <button class="condition-add-tile" type="button" @click="addNoticeRouteLabel(index)">
                      <el-icon><Plus /></el-icon>
                      <span>添加条件</span>
                    </button>
                  </div>

                  <div class="route-notice-target">
                    <div class="route-target-label">
                      <el-icon><Setting /></el-icon>
                      <strong>通知对象</strong>
                    </div>
                    <el-select v-model="routeItem.noticeObjectIds" multiple clearable filterable placeholder="选择通知对象" :disabled="!notifyRoutesEditing">
                      <el-option v-for="item in enabledNoticeObjects" :key="item.id" :label="item.name" :value="item.id" />
                    </el-select>
                    <el-button v-if="notifyRoutesEditing" link class="danger-link route-delete-btn" @click="removeNoticeRoute(index)">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                </div>
                <button v-if="notifyRoutesEditing" class="route-add-tile" type="button" @click="addNoticeRoute">
                  <el-icon><Plus /></el-icon>
                  <span>添加路由规则</span>
                </button>
              </div>
            </article>
          </div>

          <div class="notice-preview" v-if="selectedNoticeObjects.length">
            <div v-for="object in selectedNoticeObjects" :key="object.id" class="notice-object-card">
              <div>
                <strong>{{ object.name }}</strong>
                <span>{{ object.dutyTableName || '未绑定值班表' }}</span>
              </div>
              <div class="user-pills">
                <em v-if="!object.currentDutyUsers?.length">今日暂无值班</em>
                <em v-for="user in object.currentDutyUsers || []" :key="`${object.id}-${user.username}`">{{ user.realName || user.username }}</em>
              </div>
            </div>
          </div>
        </section>
      </el-tab-pane>

      <el-tab-pane label="告警升级" name="upgrade">
        <section class="config-section">
          <div class="section-heading upgrade-heading">
            <div>
              <div class="section-title">
                <el-icon><Warning /></el-icon>
                <strong>告警升级</strong>
              </div>
              <div class="upgrade-switches">
                <el-switch v-model="configForm.upgradeEnabled" :disabled="!upgradeEditing" />
                <el-checkbox-group v-model="configForm.upgradableSeverities" :disabled="!upgradeEditing">
                  <el-checkbox label="p0">P0</el-checkbox>
                  <el-checkbox label="p1">P1</el-checkbox>
                  <el-checkbox label="p2">P2</el-checkbox>
                </el-checkbox-group>
              </div>
            </div>
            <div class="card-edit-actions">
              <template v-if="upgradeEditing">
                <el-button class="edit-save-btn" :loading="saving" @click="saveUpgradeEdit">
                  <el-icon><Check /></el-icon>
                  保存
                </el-button>
                <el-button class="edit-cancel-btn" @click="cancelUpgradeEdit">
                  <el-icon><Close /></el-icon>
                  取消
                </el-button>
              </template>
              <el-button v-else link class="inline-edit-btn" @click="beginUpgradeEdit">
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
            </div>
          </div>

          <article class="setting-card upgrade-card">
            <label>
              <span>认领超时时间</span>
              <div class="suffix-input">
                <el-input-number v-model="configForm.upgrade.timeout" :min="1" :max="1440" controls-position="right" :disabled="!upgradeEditing" />
                <em>分钟</em>
              </div>
            </label>
            <label>
              <span>重复通知间隔</span>
              <div class="suffix-input">
                <el-input-number v-model="configForm.upgrade.repeatInterval" :min="1" :max="1440" controls-position="right" :disabled="!upgradeEditing" />
                <em>分钟</em>
              </div>
            </label>
            <label>
              <span>升级到通知对象</span>
              <el-select v-model="configForm.upgrade.noticeObjectIds" multiple clearable filterable placeholder="选择通知对象" :disabled="!upgradeEditing">
                <el-option v-for="item in enabledNoticeObjects" :key="item.id" :label="item.name" :value="item.id" />
              </el-select>
            </label>
          </article>
        </section>
      </el-tab-pane>
    </el-tabs>

    <el-dialog
      v-model="basicInfoDialogVisible"
      title="编辑基本信息"
      width="460px"
      class="basic-info-dialog"
      destroy-on-close
    >
      <el-form class="basic-info-form" label-position="top" @submit.prevent>
        <el-form-item label="名称" required>
          <el-input
            v-model.trim="basicInfoForm.name"
            maxlength="64"
            show-word-limit
            placeholder="请输入故障中心名称"
          />
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="basicInfoForm.description"
            type="textarea"
            :rows="3"
            maxlength="200"
            show-word-limit
            placeholder="请输入故障中心描述"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="basicInfoDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="basicInfoSaving" @click="saveBasicInfo">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="exportDialogVisible"
      title="导出告警事件"
      width="540px"
      class="export-event-dialog"
      destroy-on-close
    >
      <el-form label-width="94px" class="export-event-form" @submit.prevent>
        <el-form-item label="导出范围">
          <el-radio-group v-model="exportForm.scope">
            <el-radio-button label="active">活跃告警</el-radio-button>
            <el-radio-button label="history">历史告警</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="exportForm.scope === 'history'" label="时间范围">
          <el-date-picker
            v-model="exportForm.dateRange"
            type="daterange"
            range-separator="→"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
            clearable
            class="export-date-range"
          />
        </el-form-item>
        <el-form-item label="每页条数">
          <el-select v-model="exportForm.pageSize" class="export-select">
            <el-option :value="100" label="100 条/页" />
            <el-option :value="200" label="200 条/页" />
            <el-option :value="500" label="500 条/页" />
            <el-option :value="1000" label="1000 条/页" />
          </el-select>
        </el-form-item>
        <el-form-item label="最多导出">
          <el-input-number v-model="exportForm.maxRows" :min="100" :max="50000" :step="1000" controls-position="right" />
        </el-form-item>
        <el-alert
          :closable="false"
          type="info"
          show-icon
          title="导出会按页拉取数据，并沿用当前关键字、数据源、等级等筛选条件。"
        />
      </el-form>
      <template #footer>
        <el-button @click="exportDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="exporting" @click="confirmExportEvents">开始导出</el-button>
      </template>
    </el-dialog>

	    <el-drawer
	      v-model="silenceDrawerVisible"
	      :title="silenceDrawerTitle"
	      size="520px"
      class="watch-drawer"
      :close-on-click-modal="false"
    >
      <el-form label-position="top" class="silence-drawer-form" @submit.prevent>
        <el-form-item label="规则名称" required>
          <el-input v-model.trim="silenceForm.name" placeholder="请输入静默规则名称" />
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="silenceForm.matcher" placeholder="fingerprint=xxx 或 severity=P1" />
        </el-form-item>
        <el-form-item label="时间范围">
          <el-input v-model="silenceForm.effectiveTime" placeholder="00:00-23:59" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="silenceForm.enabled" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="silenceDrawerVisible = false">取消</el-button>
        <el-button class="dark-action" :loading="saving" @click="submitSilenceRule">保存</el-button>
	      </template>
	    </el-drawer>

	    <el-drawer
	      v-model="eventDetailVisible"
	      title="事件详情"
	      size="58%"
	      class="watch-drawer event-detail-drawer"
	    >
	      <div v-if="selectedEvent" class="event-detail-body">
	        <div class="event-detail-metrics">
	          <div class="event-detail-metric">
	            <span>当前状态</span>
	            <el-tag :type="getStateTag(selectedEvent.state)" effect="plain">{{ getStateName(selectedEvent.state) }}</el-tag>
	          </div>
	          <div class="event-detail-metric">
	            <span>告警等级</span>
	            <el-tag :type="getSeverityTag(selectedEvent.severity)" effect="plain">{{ getSeverityName(selectedEvent.severity) }}</el-tag>
	          </div>
	          <div class="event-detail-metric">
	            <span>当前值</span>
	            <strong>{{ formatEventValue(selectedEvent.value) }}</strong>
	          </div>
	          <div class="event-detail-metric">
	            <span>持续时间</span>
	            <strong>{{ formatDuration(getEventDurationSeconds(selectedEvent)) }}</strong>
	          </div>
	        </div>

	        <section class="event-detail-table-card">
	          <div class="event-detail-table">
	            <div class="event-detail-row">
	              <div class="event-detail-label">规则名称</div>
	              <div class="event-detail-value">
	                <button class="event-rule-link" type="button" @click="goToEventRule(selectedEvent)">
	                  {{ selectedEvent.ruleName || '-' }}
	                </button>
	              </div>
	            </div>
	            <div class="event-detail-row">
	              <div class="event-detail-label">告警指纹</div>
	              <div class="event-detail-value mono">{{ selectedEvent.fingerprint || '-' }}</div>
	            </div>
	            <div class="event-detail-row">
	              <div class="event-detail-label">数据源</div>
	              <div class="event-detail-value">{{ selectedEvent.dataSourceName || '-' }}</div>
	            </div>
	            <div class="event-detail-row">
	              <div class="event-detail-label">事件状态</div>
	              <div class="event-detail-value">
	                <el-tag :type="getStateTag(selectedEvent.state)" effect="plain">{{ getStateName(selectedEvent.state) }}</el-tag>
	              </div>
	            </div>
	            <div class="event-detail-row">
	              <div class="event-detail-label">事件标签</div>
	              <div class="event-detail-value">
	                <div class="event-detail-tags">
	                  <el-tag v-for="item in parseEventMap(selectedEvent.labels)" :key="`label-${item.key}`" effect="plain">
	                    {{ item.key }}: {{ item.value }}
	                  </el-tag>
	                  <span v-if="!parseEventMap(selectedEvent.labels).length" class="empty-detail-text">暂无标签</span>
	                </div>
	              </div>
	            </div>
	            <div class="event-detail-row">
	              <div class="event-detail-label">触发条件</div>
	              <div class="event-detail-value condition-text">{{ buildEventConditionText(selectedEvent) }}</div>
	            </div>
	            <div class="event-detail-row">
	              <div class="event-detail-label">触发时间</div>
	              <div class="event-detail-value">{{ formatDateTime(selectedEvent.startedAt) }}</div>
	            </div>
	            <div class="event-detail-row">
	              <div class="event-detail-label">触发时值</div>
	              <div class="event-detail-value">{{ formatEventValue(selectedEvent.value) }}</div>
	            </div>
	            <div class="event-detail-row">
	              <div class="event-detail-label">认领人</div>
	              <div class="event-detail-value">
	                <el-tag effect="plain">{{ selectedEvent.acknowledgedBy || '未认领' }}</el-tag>
	              </div>
	            </div>
	            <div class="event-detail-row event-detail-row-large">
	              <div class="event-detail-label">事件详情</div>
	              <div class="event-detail-value">
	                <pre class="event-detail-message">{{ getEventDetailText(selectedEvent) }}</pre>
	              </div>
	            </div>
	            <div class="event-detail-row">
	              <div class="event-detail-label">通知记录</div>
	              <div class="event-detail-value">
	                <el-tag :type="selectedEvent.notifyStatus === 'success' ? 'success' : selectedEvent.notifyStatus === 'failed' ? 'danger' : 'info'" effect="plain">
	                  {{ selectedEvent.notifyStatus || 'none' }}
	                </el-tag>
	                <span v-if="selectedEvent.notifyError" class="event-notify-error">{{ selectedEvent.notifyError }}</span>
	              </div>
	            </div>
	          </div>
	        </section>

	        <section class="event-detail-section">
	          <div class="detail-section-head">
	            <div class="detail-section-title">回调查询</div>
	            <el-button size="small" :loading="callbackLoading" @click="loadEventCallbacks(selectedEvent)">刷新</el-button>
	          </div>
	          <div v-loading="callbackLoading" class="callback-list">
	            <article v-for="item in callbackResults" :key="`${item.name}-${item.query}`" class="callback-card">
	              <header>
	                <div>
	                  <strong>{{ item.name }}</strong>
	                  <span>{{ item.dataSourceName || item.dataSourceType || '-' }} / {{ item.queryMode || 'instant' }}</span>
	                </div>
	                <el-tag :type="item.error ? 'danger' : 'success'" effect="plain">
	                  {{ item.error ? '失败' : `${item.duration || 0}ms` }}
	                </el-tag>
	              </header>
	              <div class="callback-query-code">
	                <span>渲染语句</span>
	                <code>{{ item.renderedQuery || item.query }}</code>
	              </div>
	              <small v-if="item.renderedQuery && item.renderedQuery !== item.query" class="callback-original-query">
	                原始语句：{{ item.query }}
	              </small>
	              <div v-if="item.error" class="callback-error">{{ item.error }}</div>
	              <CallbackMetricChart v-else :item="item" :event="selectedEvent" />
	              <el-collapse v-if="!item.error" class="callback-raw-collapse">
	                <el-collapse-item title="原始返回" :name="`${item.key || item.name}-raw`">
	                  <pre>{{ formatCallbackResult(item.result) }}</pre>
	                </el-collapse-item>
	              </el-collapse>
	            </article>
	            <span v-if="!callbackLoading && !callbackResults.length" class="empty-detail-text">当前规则未配置回调查询</span>
	          </div>
	        </section>
	      </div>
	    </el-drawer>
	  </div>
	</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowDown,
  Back,
  Bell,
  Check,
  Clock,
  Close,
  Connection,
  Delete,
  Download,
  Edit,
  House,
  Operation,
  Plus,
  PriceTag,
  Refresh,
  Search,
  Setting,
  Timer,
  Warning
} from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import {
  acknowledgeMonitorAlertEvent,
  batchAcknowledgeMonitorAlertEvents,
  batchDeleteMonitorAlertEvents,
	  deleteMonitorFaultCenter,
	  deleteMonitorAlertEvent,
	  getMonitorAlertEvent,
	  getMonitorAlertEventCallbacks,
	  getMonitorAlertEvents,
	  getMonitorFaultCenter,
  getMonitorFaultCenterSLO,
  getMonitorNoticeObjects,
  silenceMonitorAlertEvent,
	  updateMonitorFaultCenter,
	  type MonitorAlertCallbackResult,
	  type MonitorAlertEvent,
  type MonitorFaultCenter,
  type MonitorNoticeObject
} from '@/api/monitor-datasource'
import CallbackMetricChart from './CallbackMetricChart.vue'

interface SLOPoint {
  date: string
  mtta: number
  mttr: number
  mtbf?: number
}

interface SilenceRule {
  key: string
  name: string
  matcher: string
  effectiveTime: string
  enabled: boolean
}

interface NoticeRoute {
  key: string
  name: string
  matcher: string
  labels: NoticeRouteLabel[]
  noticeObjectIds: number[]
  enabled: boolean
}

interface NoticeRouteLabel {
  key: string
  operator: string
  value: string
}

interface NotifyBasicSnapshot {
  noticeObjectIds: number[]
  repeat: {
    p0: number
    p1: number
    p2: number
  }
  recoverNotify: boolean
  recoverWaitSeconds: number
}

interface UpgradeSnapshot {
  upgradeEnabled: boolean
  upgradableSeverities: string[]
  upgrade: {
    timeout: number
    repeatInterval: number
    noticeObjectIds: number[]
  }
}

type EventScope = 'active' | 'history'
type DetailTab = EventScope | 'silence' | 'notify' | 'upgrade'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const eventsLoading = ref(false)
const saving = ref(false)
const exporting = ref(false)
const basicInfoDialogVisible = ref(false)
const exportDialogVisible = ref(false)
const eventDetailVisible = ref(false)
const basicInfoSaving = ref(false)
const editingBasicField = ref<'name' | 'description' | ''>('')
const basicFieldValue = ref('')
const notifyBasicEditing = ref(false)
const notifyRoutesEditing = ref(false)
const upgradeEditing = ref(false)
const activeTab = ref<DetailTab>('active')
const eventKeyword = ref('')
const historyDateRange = ref<string[]>([])
const silenceKeyword = ref('')
const silenceStatusFilter = ref('enabled')
const silenceDrawerVisible = ref(false)
const silenceEditingKey = ref('')
const callbackLoading = ref(false)
const center = ref<MonitorFaultCenter>()
const noticeObjects = ref<MonitorNoticeObject[]>([])
const activeEvents = ref<MonitorAlertEvent[]>([])
const historyEvents = ref<MonitorAlertEvent[]>([])
const selectedActiveEventRows = ref<MonitorAlertEvent[]>([])
const selectedEvent = ref<MonitorAlertEvent>()
const callbackResults = ref<MonitorAlertCallbackResult[]>([])
const sloData = ref<SLOPoint[]>([])
const currentTime = ref(Date.now())
const mttrChartRef = ref<HTMLElement>()
const mttaChartRef = ref<HTMLElement>()
let mttrChart: echarts.ECharts | null = null
let mttaChart: echarts.ECharts | null = null
let durationTimer: number | undefined
let detailScrollState = {
  windowTop: 0,
  mainTop: 0
}
let notifyBasicSnapshot: NotifyBasicSnapshot | null = null
let notifyRoutesSnapshot: NoticeRoute[] | null = null
let upgradeSnapshot: UpgradeSnapshot | null = null

const activeFilters = reactive({
  dataSourceType: '',
  severity: '',
  state: ''
})

const historyFilters = reactive({
  dataSourceType: '',
  severity: ''
})

const activePager = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const historyPager = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const exportForm = reactive({
  scope: 'history' as EventScope,
  dateRange: [] as string[],
  pageSize: 200,
  maxRows: 5000
})

const configForm = reactive({
  noticeObjectIds: [] as number[],
  noticeRoutes: [] as NoticeRoute[],
  aggregationType: 'Rule',
  silenceEnabled: false,
  silenceRules: [] as SilenceRule[],
  recoverNotify: true,
  recoverWaitSeconds: 30,
  repeat: {
    p0: 30,
    p1: 60,
    p2: 120
  },
  upgradeEnabled: false,
  upgradableSeverities: ['p0', 'p1'] as string[],
  upgrade: {
    timeout: 30,
    repeatInterval: 60,
    noticeObjectIds: [] as number[]
  }
})

const basicInfoForm = reactive({
  name: '',
  description: ''
})

const silenceForm = reactive<SilenceRule>({
  key: '',
  name: '',
  matcher: '',
  effectiveTime: '00:00-23:59',
  enabled: true
})

const tableHeaderStyle = { background: '#fafafa', color: '#101828', fontWeight: '700' }
const centerId = computed(() => Number(route.params.id))
const centerCode = computed(() => `fc-${String(center.value?.id || centerId.value || 0).padStart(10, '0')}`)
const currentTimeText = computed(() => formatDateTime(new Date().toISOString()))
const enabledNoticeObjects = computed(() => noticeObjects.value.filter(item => item.enabled !== false && item.id))
const selectedNoticeObjects = computed(() => noticeObjects.value.filter(item => item.id && configForm.noticeObjectIds.includes(item.id)))
const selectedActiveEventIds = computed(() => selectedActiveEventRows.value.map(item => item.id).filter(Boolean))
const silenceDrawerTitle = computed(() => silenceEditingKey.value ? '编辑静默规则' : '创建静默规则')
const filteredSilenceRules = computed(() => {
  const keyword = silenceKeyword.value.trim().toLowerCase()
  return configForm.silenceRules.filter(rule => {
    const statusMatched =
      silenceStatusFilter.value === 'all' ||
      (silenceStatusFilter.value === 'pending' && false) ||
      (silenceStatusFilter.value === 'enabled' && rule.enabled) ||
      (silenceStatusFilter.value === 'disabled' && !rule.enabled)
    const keywordMatched =
      !keyword ||
      rule.name.toLowerCase().includes(keyword) ||
      rule.matcher.toLowerCase().includes(keyword)
    return statusMatched && keywordMatched
  })
})

const dataSourceOptions = [
  { label: 'Prometheus', value: 'prometheus' },
  { label: 'VictoriaMetrics', value: 'victoriametrics' },
  { label: 'Loki', value: 'loki' },
  { label: 'Elasticsearch', value: 'elasticsearch' }
]

const severityOptions = [
  { label: 'P0', value: 'p0' },
  { label: 'P1', value: 'p1' },
  { label: 'P2', value: 'p2' }
]

const eventStateOptions = [
  { label: '预告警', value: 'pending' },
  { label: '告警中', value: 'firing' },
  { label: '处理中', value: 'processing' },
  { label: '静默中', value: 'silenced' },
  { label: '待恢复', value: 'recovering' }
]

const silenceStatusOptions = [
  { label: '全部', value: 'all' },
  { label: '未生效', value: 'pending' },
  { label: '生效中', value: 'enabled' },
  { label: '已失效', value: 'disabled' }
]

const loadCenter = async () => {
  if (!centerId.value) return
  loading.value = true
  try {
    center.value = await getMonitorFaultCenter(centerId.value)
    fillConfig(center.value)
    resetConfigEditStates()
  } finally {
    loading.value = false
  }
}

const buildEventParams = (
  scope: EventScope,
  overrides: { page?: number; pageSize?: number; dateRange?: string[] } = {}
) => {
  const pager = scope === 'active' ? activePager : historyPager
  const filters = scope === 'active' ? activeFilters : historyFilters
  const params: Record<string, unknown> = {
    page: overrides.page ?? pager.page,
    pageSize: overrides.pageSize ?? pager.pageSize,
    faultCenterId: centerId.value,
    scope,
    keyword: eventKeyword.value.trim(),
    dataSourceType: filters.dataSourceType || undefined,
    severity: filters.severity || undefined
  }
  if (scope === 'active') {
    params.state = activeFilters.state || undefined
  } else {
    const dateRange = overrides.dateRange || historyDateRange.value
    if (dateRange?.length === 2) {
      params.startDate = dateRange[0]
      params.endDate = dateRange[1]
    }
  }
  return params
}

const routeQueryText = (value: unknown) => {
  if (Array.isArray(value)) return String(value[0] ?? '')
  return String(value ?? '')
}

const applyRouteQueryState = () => {
  const tab = routeQueryText(route.query.tab)
  const hasRouteQuery = Object.prototype.hasOwnProperty.call(route.query, 'query')
  if (['active', 'history', 'silence', 'notify', 'upgrade'].includes(tab)) {
    activeTab.value = tab as DetailTab
  }
  if (hasRouteQuery) {
    eventKeyword.value = routeQueryText(route.query.query).trim()
  } else if (!Object.keys(route.query).length) {
    eventKeyword.value = ''
  }
  if (activeTab.value === 'active') {
    activePager.page = 1
    if (hasRouteQuery) {
      activeFilters.dataSourceType = ''
      activeFilters.severity = ''
      activeFilters.state = ''
    }
  } else if (activeTab.value === 'history') {
    historyPager.page = 1
    if (hasRouteQuery) {
      historyFilters.dataSourceType = ''
      historyFilters.severity = ''
    }
  }
}

const loadActiveEvents = async () => {
  if (!centerId.value) return
  eventsLoading.value = true
  try {
    const active = await getMonitorAlertEvents(buildEventParams('active'))
    activeEvents.value = active?.list || []
    activePager.total = active?.total || 0
  } finally {
    eventsLoading.value = false
  }
}

const loadHistoryEvents = async () => {
  if (!centerId.value) return
  eventsLoading.value = true
  try {
    const history = await getMonitorAlertEvents(buildEventParams('history'))
    historyEvents.value = history?.list || []
    historyPager.total = history?.total || 0
  } finally {
    eventsLoading.value = false
  }
}

const loadEvents = async () => {
  eventsLoading.value = true
  try {
    await Promise.all([loadActiveEvents(), loadHistoryEvents()])
  } finally {
    eventsLoading.value = false
  }
}

const openEventDetail = async (row: MonitorAlertEvent) => {
  selectedEvent.value = row
  callbackResults.value = []
  eventDetailVisible.value = true
  await loadEventCallbacks(row)
}

const loadEventCallbacks = async (row: MonitorAlertEvent) => {
  if (!row?.id) return
  callbackLoading.value = true
  try {
    callbackResults.value = await getMonitorAlertEventCallbacks(row.id) || []
  } finally {
    callbackLoading.value = false
  }
}

const openEventFromRoute = async () => {
  const eventId = Number(route.query.eventId)
  if (!eventId) return
  let event = [...activeEvents.value, ...historyEvents.value].find(item => item.id === eventId)
  if (!event) {
    event = await getMonitorAlertEvent(eventId)
  }
  if (!event) return
  activeTab.value = event.endedAt || event.state === 'recovered' ? 'history' : 'active'
  await nextTick()
  await openEventDetail(event)
}

const loadSLO = async () => {
  if (!centerId.value) return
  sloData.value = await getMonitorFaultCenterSLO(centerId.value) || []
  await nextTick()
  renderSloCharts()
}

const loadNoticeObjects = async () => {
  noticeObjects.value = await getMonitorNoticeObjects() || []
}

const loadAll = async () => {
  applyRouteQueryState()
  await Promise.all([loadNoticeObjects(), loadCenter(), loadEvents(), loadSLO()])
  await openEventFromRoute()
}

const refreshEventList = () => {
  if (activeTab.value === 'history') {
    refreshHistoryEvents()
  } else {
    refreshActiveEvents()
  }
}

const refreshActiveEvents = () => {
  activePager.page = 1
  loadActiveEvents()
}

const refreshHistoryEvents = () => {
  historyPager.page = 1
  loadHistoryEvents()
}

const handleActiveSizeChange = () => {
  activePager.page = 1
  loadActiveEvents()
}

const handleActiveSelectionChange = (rows: MonitorAlertEvent[]) => {
  selectedActiveEventRows.value = rows
}

const handleActiveBatchCommand = async (command: string | number | object) => {
  if (!selectedActiveEventIds.value.length) {
    ElMessage.warning('请先选择活跃告警')
    return
  }
  if (command === 'ack') {
    await batchAckActiveEvents()
  } else if (command === 'delete') {
    await batchDeleteActiveEvents()
  }
}

const batchAckActiveEvents = async () => {
  const username = localStorage.getItem('username') || 'admin'
  await batchAcknowledgeMonitorAlertEvents(selectedActiveEventIds.value, { username })
  ElMessage.success(`已认领 ${selectedActiveEventIds.value.length} 条告警`)
  selectedActiveEventRows.value = []
  await Promise.all([loadActiveEvents(), loadHistoryEvents(), loadCenter()])
}

const batchDeleteActiveEvents = async () => {
  await ElMessageBox.confirm(`确定删除选中的 ${selectedActiveEventIds.value.length} 条活跃告警吗？`, '批量删除告警事件', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await batchDeleteMonitorAlertEvents(selectedActiveEventIds.value)
  ElMessage.success('批量删除成功')
  selectedActiveEventRows.value = []
  await Promise.all([loadActiveEvents(), loadHistoryEvents(), loadCenter()])
}

const handleHistorySizeChange = () => {
  historyPager.page = 1
  loadHistoryEvents()
}

const getMainScrollElement = () => document.querySelector('.el-main') as HTMLElement | null

const captureDetailScroll = () => {
  detailScrollState = {
    windowTop: window.scrollY || document.documentElement.scrollTop || 0,
    mainTop: getMainScrollElement()?.scrollTop || 0
  }
}

const restoreDetailScroll = () => {
  const state = { ...detailScrollState }
  const restore = () => {
    const main = getMainScrollElement()
    if (main) main.scrollTop = state.mainTop
    window.scrollTo({ top: state.windowTop, left: 0 })
  }
  nextTick(() => {
    restore()
    window.requestAnimationFrame(restore)
    window.requestAnimationFrame(() => window.requestAnimationFrame(restore))
    ;[80, 180, 320].forEach(delay => window.setTimeout(restore, delay))
  })
}

const handleTabChange = () => {
  restoreDetailScroll()
  nextTick(renderSloCharts)
}

const fillConfig = (data?: MonitorFaultCenter) => {
  if (!data) return
  const repeat = safeParse<Record<string, number>>(data.repeatNoticeInterval, {})
  const upgrade = safeParse<{ timeout?: number; repeatInterval?: number; noticeObjectIds?: number[] }>(data.upgradeStrategy, {})
  configForm.noticeObjectIds = parseIdArray(data.noticeObjectIds)
  configForm.noticeRoutes = parseNoticeRoutes(data.noticeRoutes)
  configForm.aggregationType = data.aggregationType || 'Rule'
  configForm.silenceEnabled = data.silenceEnabled || false
  configForm.silenceRules = parseSilenceRules(data.silenceRules)
  configForm.recoverNotify = data.recoverNotify ?? true
  configForm.recoverWaitSeconds = data.recoverWaitSeconds || 30
  configForm.repeat = {
    p0: repeat.p0 || 30,
    p1: repeat.p1 || 60,
    p2: repeat.p2 || 120
  }
  configForm.upgradeEnabled = data.upgradeEnabled || false
  configForm.upgradableSeverities = safeParse<string[]>(data.upgradableSeverities, ['p0', 'p1'])
  configForm.upgrade = {
    timeout: upgrade.timeout || 30,
    repeatInterval: upgrade.repeatInterval || 60,
    noticeObjectIds: Array.isArray(upgrade.noticeObjectIds) ? upgrade.noticeObjectIds.map(Number).filter(Boolean) : []
  }
}

const saveConfig = async () => {
  if (!center.value?.id) return
  saving.value = true
  try {
    await updateMonitorFaultCenter(center.value.id, {
      ...center.value,
      noticeObjectIds: JSON.stringify(configForm.noticeObjectIds),
      noticeChannelIds: '[]',
      noticeRoutes: JSON.stringify(configForm.noticeRoutes.map(normalizeNoticeRouteForSave)),
      repeatNoticeInterval: JSON.stringify({
        p0: configForm.repeat.p0,
        p1: configForm.repeat.p1,
        p2: configForm.repeat.p2
      }),
      recoverNotify: configForm.recoverNotify,
      aggregationType: configForm.aggregationType,
      silenceEnabled: configForm.silenceEnabled,
      silenceRules: JSON.stringify(configForm.silenceRules.map(({ key, ...rule }) => rule)),
      recoverWaitSeconds: configForm.recoverWaitSeconds,
      upgradeEnabled: configForm.upgradeEnabled,
      upgradableSeverities: JSON.stringify(configForm.upgradableSeverities),
      upgradeStrategy: JSON.stringify({
        enabled: configForm.upgradeEnabled,
        timeout: configForm.upgrade.timeout,
        repeatInterval: configForm.upgrade.repeatInterval,
        noticeObjectIds: configForm.upgrade.noticeObjectIds
      })
	    })
	    ElMessage.success('保存成功')
	    await Promise.all([loadCenter(), loadActiveEvents(), loadHistoryEvents()])
	  } finally {
    saving.value = false
  }
}

const cloneNoticeRoutes = (routes: NoticeRoute[]) => routes.map(routeItem => ({
  key: routeItem.key,
  name: routeItem.name,
  matcher: routeItem.matcher,
  labels: routeItem.labels.map(label => ({ ...label })),
  noticeObjectIds: [...routeItem.noticeObjectIds],
  enabled: routeItem.enabled
}))

const getNotifyBasicSnapshot = (): NotifyBasicSnapshot => ({
  noticeObjectIds: [...configForm.noticeObjectIds],
  repeat: { ...configForm.repeat },
  recoverNotify: configForm.recoverNotify,
  recoverWaitSeconds: configForm.recoverWaitSeconds
})

const restoreNotifyBasic = (snapshot: NotifyBasicSnapshot) => {
  configForm.noticeObjectIds = [...snapshot.noticeObjectIds]
  configForm.repeat = { ...snapshot.repeat }
  configForm.recoverNotify = snapshot.recoverNotify
  configForm.recoverWaitSeconds = snapshot.recoverWaitSeconds
}

const getUpgradeSnapshot = (): UpgradeSnapshot => ({
  upgradeEnabled: configForm.upgradeEnabled,
  upgradableSeverities: [...configForm.upgradableSeverities],
  upgrade: {
    timeout: configForm.upgrade.timeout,
    repeatInterval: configForm.upgrade.repeatInterval,
    noticeObjectIds: [...configForm.upgrade.noticeObjectIds]
  }
})

const restoreUpgrade = (snapshot: UpgradeSnapshot) => {
  configForm.upgradeEnabled = snapshot.upgradeEnabled
  configForm.upgradableSeverities = [...snapshot.upgradableSeverities]
  configForm.upgrade = {
    timeout: snapshot.upgrade.timeout,
    repeatInterval: snapshot.upgrade.repeatInterval,
    noticeObjectIds: [...snapshot.upgrade.noticeObjectIds]
  }
}

const resetConfigEditStates = () => {
  notifyBasicEditing.value = false
  notifyRoutesEditing.value = false
  upgradeEditing.value = false
  notifyBasicSnapshot = null
  notifyRoutesSnapshot = null
  upgradeSnapshot = null
}

const beginNotifyBasicEdit = () => {
  notifyBasicSnapshot = getNotifyBasicSnapshot()
  notifyBasicEditing.value = true
}

const cancelNotifyBasicEdit = () => {
  if (notifyBasicSnapshot) {
    restoreNotifyBasic(notifyBasicSnapshot)
  }
  notifyBasicEditing.value = false
  notifyBasicSnapshot = null
}

const saveNotifyBasicEdit = async () => {
  await saveConfig()
  notifyBasicEditing.value = false
  notifyBasicSnapshot = null
}

const beginNotifyRoutesEdit = () => {
  notifyRoutesSnapshot = cloneNoticeRoutes(configForm.noticeRoutes)
  notifyRoutesEditing.value = true
}

const cancelNotifyRoutesEdit = () => {
  if (notifyRoutesSnapshot) {
    configForm.noticeRoutes = cloneNoticeRoutes(notifyRoutesSnapshot)
  }
  notifyRoutesEditing.value = false
  notifyRoutesSnapshot = null
}

const saveNotifyRoutesEdit = async () => {
  await saveConfig()
  notifyRoutesEditing.value = false
  notifyRoutesSnapshot = null
}

const beginUpgradeEdit = () => {
  upgradeSnapshot = getUpgradeSnapshot()
  upgradeEditing.value = true
}

const cancelUpgradeEdit = () => {
  if (upgradeSnapshot) {
    restoreUpgrade(upgradeSnapshot)
  }
  upgradeEditing.value = false
  upgradeSnapshot = null
}

const saveUpgradeEdit = async () => {
  await saveConfig()
  upgradeEditing.value = false
  upgradeSnapshot = null
}

const openBasicInfoDialog = () => {
  if (!center.value) return
  basicInfoForm.name = center.value.name || ''
  basicInfoForm.description = center.value.description || ''
  basicInfoDialogVisible.value = true
}

const beginInlineBasicEdit = (field: 'name' | 'description') => {
  if (!center.value) return
  editingBasicField.value = field
  basicFieldValue.value = String(center.value[field] || '')
}

const cancelInlineBasicField = () => {
  editingBasicField.value = ''
  basicFieldValue.value = ''
}

const saveInlineBasicField = async () => {
  if (!center.value?.id || !editingBasicField.value) return
  const field = editingBasicField.value
  const value = basicFieldValue.value.trim()
  if (field === 'name' && !value) {
    ElMessage.warning('请输入故障中心名称')
    return
  }
  basicInfoSaving.value = true
  try {
    await updateMonitorFaultCenter(center.value.id, {
      ...center.value,
      [field]: value
    })
    center.value = {
      ...center.value,
      [field]: value
    }
    cancelInlineBasicField()
    ElMessage.success('保存成功')
  } finally {
    basicInfoSaving.value = false
  }
}

const saveBasicInfo = async () => {
  if (!center.value?.id) return
  const name = basicInfoForm.name.trim()
  if (!name) {
    ElMessage.warning('请输入故障中心名称')
    return
  }
  basicInfoSaving.value = true
  try {
    await updateMonitorFaultCenter(center.value.id, {
      ...center.value,
      name,
      description: basicInfoForm.description.trim()
    })
    center.value = {
      ...center.value,
      name,
      description: basicInfoForm.description.trim()
    }
    basicInfoDialogVisible.value = false
    ElMessage.success('基本信息已保存')
  } finally {
    basicInfoSaving.value = false
  }
}

const ackEvent = async (row: MonitorAlertEvent) => {
  await acknowledgeMonitorAlertEvent(row.id, { username: localStorage.getItem('username') || 'admin' })
  ElMessage.success('已认领')
  await loadEvents()
}

const deleteEvent = async (row: MonitorAlertEvent) => {
  await ElMessageBox.confirm(`确认删除告警事件「${row.ruleName}」吗？`, '删除告警事件', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await deleteMonitorAlertEvent(row.id)
  ElMessage.success('已删除')
  await loadEvents()
  await loadCenter()
}

const deleteCenter = async () => {
  if (!center.value?.id) return
  await ElMessageBox.confirm(`确定删除故障中心「${center.value.name}」吗？`, '删除故障中心', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await deleteMonitorFaultCenter(center.value.id)
  ElMessage.success('故障中心已删除')
  router.push('/monitor/fault-centers')
}

const handleEventCommand = async (command: string | number | object, row: MonitorAlertEvent) => {
  if (command === 'ack') {
    await ackEvent(row)
  } else if (command === 'silence') {
    await quickSilenceEvent(row)
  } else if (command === 'delete') {
    await deleteEvent(row)
  }
}

const quickSilenceEvent = async (row: MonitorAlertEvent) => {
  const username = localStorage.getItem('username') || 'admin'
  await silenceMonitorAlertEvent(row.id, { username })
  await Promise.all([loadActiveEvents(), loadCenter()])
  configForm.silenceEnabled = true
  configForm.silenceRules.unshift({
    key: `${Date.now()}-${Math.random()}`,
    name: row.ruleName || '临时静默',
    matcher: row.fingerprint ? `fingerprint=${row.fingerprint}` : `rule_id=${row.ruleId}`,
    effectiveTime: '00:00-23:59',
    enabled: true
  })
  row.state = 'silenced'
  row.acknowledged = true
  row.acknowledgedBy = username
  await saveConfig()
  activeTab.value = 'silence'
  ElMessage.success('已静默事件，静默规则已保存')
}

const addSilenceRule = () => {
  silenceEditingKey.value = ''
  Object.assign(silenceForm, {
    key: `${Date.now()}-${Math.random()}`,
    name: '',
    matcher: '',
    effectiveTime: '00:00-23:59',
    enabled: true
  })
  silenceDrawerVisible.value = true
}

const editSilenceRule = (rule: SilenceRule) => {
  silenceEditingKey.value = rule.key
  Object.assign(silenceForm, { ...rule })
  silenceDrawerVisible.value = true
}

const submitSilenceRule = async () => {
  const name = silenceForm.name.trim()
  if (!name) {
    ElMessage.warning('请输入静默规则名称')
    return
  }
  const payload = {
    key: silenceForm.key || `${Date.now()}-${Math.random()}`,
    name,
    matcher: silenceForm.matcher.trim(),
    effectiveTime: silenceForm.effectiveTime || '00:00-23:59',
    enabled: silenceForm.enabled
  }
  const index = configForm.silenceRules.findIndex(rule => rule.key === silenceEditingKey.value)
  configForm.silenceEnabled = true
  if (index >= 0) {
    configForm.silenceRules.splice(index, 1, payload)
  } else {
    configForm.silenceRules.unshift(payload)
  }
  await saveConfig()
  silenceDrawerVisible.value = false
  silenceEditingKey.value = ''
}

const removeSilenceRule = async (index: number) => {
  const target = filteredSilenceRules.value[index]
  if (!target) return
  await ElMessageBox.confirm(`确定删除静默规则「${target.name || '-'}」吗？`, '删除静默规则', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  const originalIndex = configForm.silenceRules.findIndex(rule => rule.key === target?.key)
  if (originalIndex >= 0) {
    configForm.silenceRules.splice(originalIndex, 1)
    await saveConfig()
  }
}

const splitMatcherTags = (matcher?: string) => String(matcher || '')
  .split(/[,;\n]/)
  .map(item => item.trim())
  .filter(Boolean)

const addNoticeRoute = () => {
  configForm.noticeRoutes.push({
    key: `${Date.now()}-${Math.random()}`,
    name: '',
    matcher: '',
    labels: [{ key: '', operator: '==', value: '' }],
    noticeObjectIds: [],
    enabled: true
  })
}

const addNoticeRouteLabel = (routeIndex: number) => {
  configForm.noticeRoutes[routeIndex]?.labels.push({ key: '', operator: '==', value: '' })
}

const removeNoticeRouteLabel = (routeIndex: number, labelIndex: number) => {
  const labels = configForm.noticeRoutes[routeIndex]?.labels
  if (labels && labels.length > 1) {
    labels.splice(labelIndex, 1)
  }
}

const removeNoticeRoute = (index: number) => {
  configForm.noticeRoutes.splice(index, 1)
}

const averageSlo = (key: keyof Pick<SLOPoint, 'mtta' | 'mttr' | 'mtbf'>) => {
  const values = sloData.value.map(item => Number(item[key]) || 0).filter(value => value > 0)
  if (!values.length) return 0
  return values.reduce((sum, value) => sum + value, 0) / values.length
}

const parseSilenceRules = (raw?: string): SilenceRule[] => {
  const rules = safeParse<Array<Partial<SilenceRule>>>(raw, [])
  if (!Array.isArray(rules)) return []
  return rules.map(rule => ({
    key: `${Date.now()}-${Math.random()}`,
    name: rule.name || '',
    matcher: rule.matcher || '',
    effectiveTime: rule.effectiveTime || '00:00-23:59',
    enabled: rule.enabled ?? true
  }))
}

const parseNoticeRoutes = (raw?: string): NoticeRoute[] => {
  const routes = safeParse<Array<Partial<NoticeRoute>>>(raw, [])
  if (!Array.isArray(routes)) return []
  return routes.map(routeItem => ({
    key: `${Date.now()}-${Math.random()}`,
    name: routeItem.name || '',
    matcher: routeItem.matcher || '',
    labels: normalizeRouteLabels(routeItem),
    noticeObjectIds: Array.isArray(routeItem.noticeObjectIds) ? routeItem.noticeObjectIds.map(Number).filter(Boolean) : [],
    enabled: routeItem.enabled ?? true
  }))
}

const normalizeRouteLabels = (routeItem: Partial<NoticeRoute>): NoticeRouteLabel[] => {
  if (Array.isArray(routeItem.labels) && routeItem.labels.length) {
    return routeItem.labels.map(label => ({
      key: label.key || '',
      operator: label.operator || '==',
      value: label.value || ''
    }))
  }
  const matcher = String(routeItem.matcher || '').trim()
  if (!matcher) return [{ key: '', operator: '==', value: '' }]
  const labels = matcher
    .split(/[,;\n]/)
    .map(item => item.trim())
    .filter(Boolean)
    .map(item => {
      const operator = ['=~', '!~', '!=', '=='].find(op => item.includes(op)) || (item.includes(':') ? ':' : '=')
      const [key, ...rest] = item.split(operator)
      return {
        key: key?.trim() || '',
        operator: operator === ':' || operator === '=' ? '==' : operator,
        value: rest.join(operator).trim().replace(/^['"]|['"]$/g, '')
      }
    })
  return labels.length ? labels : [{ key: '', operator: '==', value: '' }]
}

const normalizeNoticeRouteForSave = (routeItem: NoticeRoute) => {
  const labels = routeItem.labels
    .map(label => ({
      key: label.key.trim(),
      operator: label.operator || '==',
      value: label.value.trim()
    }))
    .filter(label => label.key && label.value)
  const matcher = labels
    .map(label => `${label.key}${label.operator === '==' ? '=' : label.operator}${label.value}`)
    .join(',')
  return {
    name: routeItem.name || matcher || '告警路由',
    matcher,
    labels,
    noticeObjectIds: routeItem.noticeObjectIds,
    enabled: routeItem.enabled
  }
}

const formatDuration = (seconds?: number) => {
  const value = Math.round(Number(seconds) || 0)
  if (value <= 0) return '0 秒'
  const days = Math.floor(value / 86400)
  const hours = Math.floor((value % 86400) / 3600)
  const minutes = Math.floor((value % 3600) / 60)
  const secs = value % 60
  if (days > 0) return `${days}天 ${hours}小时 ${minutes}分 ${secs}秒`
  if (hours > 0) return `${hours}小时 ${minutes}分 ${secs}秒`
  if (minutes > 0) return `${minutes}分钟 ${secs}秒`
  return `${secs}秒`
}

const formatShortDuration = (seconds?: number) => {
  const value = Math.round(Number(seconds) || 0)
  if (value >= 86400) return `${Math.round(value / 86400)}d`
  if (value >= 3600) return `${Math.round(value / 3600)}h`
  if (value >= 60) return `${Math.round(value / 60)}m`
  return `${value}s`
}

const getEventDurationSeconds = (row: MonitorAlertEvent) => {
	const started = new Date(row.startedAt).getTime()
	if (!started || Number.isNaN(started)) return 0
	const ended = row.endedAt ? new Date(row.endedAt).getTime() : currentTime.value
	if (!ended || Number.isNaN(ended) || ended < started) return 0
	return Math.round((ended - started) / 1000)
}

const durationBlockCount = (row: MonitorAlertEvent) => {
  const seconds = getEventDurationSeconds(row)
  if (seconds <= 0) return 0
  if (seconds < 1800) return Math.max(1, Math.ceil(seconds / 600))
  if (seconds < 7200) return Math.min(5, Math.ceil(seconds / 1200) + 1)
  if (seconds < 86400) return Math.min(7, Math.ceil(seconds / 14400) + 2)
  return Math.min(10, Math.ceil(seconds / 86400) + 5)
}

const renderMetricChart = (chartRef: HTMLElement | undefined, chart: echarts.ECharts | null, key: 'mttr' | 'mtta') => {
  if (!chartRef) return null
  const instance = chart || echarts.init(chartRef)
  const dates = sloData.value.map(item => item.date)
  const values = sloData.value.map(item => Math.round(item[key] || 0))
  instance.setOption({
    color: ['#111827'],
    grid: { left: 42, right: 12, top: 20, bottom: 30, containLabel: false },
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#ffffff',
      borderColor: '#d8dee9',
      borderWidth: 1,
      textStyle: { color: '#344054', fontSize: 12 },
      extraCssText: 'box-shadow:0 10px 28px rgba(15,23,42,.12);border-radius:8px;',
      valueFormatter: (value: number) => formatDuration(value)
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: dates,
      axisLine: { lineStyle: { color: '#d8dee9' } },
      axisTick: { show: false },
      axisLabel: { color: '#667085', fontSize: 12 }
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      axisLabel: { color: '#667085', fontSize: 12, formatter: (value: number) => formatShortDuration(value) },
      splitLine: { lineStyle: { color: '#cfd4dc', type: 'dashed' } }
    },
    series: [
      {
        name: key.toUpperCase(),
        type: 'line',
        data: values,
        smooth: true,
        symbolSize: 5,
        lineStyle: { width: 2 },
        itemStyle: { color: '#111827' },
        areaStyle: { color: 'rgba(17, 24, 39, 0.03)' }
      }
    ]
  })
  instance.resize()
  return instance
}

const renderSloCharts = () => {
  mttrChart = renderMetricChart(mttrChartRef.value, mttrChart, 'mttr')
  mttaChart = renderMetricChart(mttaChartRef.value, mttaChart, 'mtta')
}

const handleChartResize = () => {
  mttrChart?.resize()
  mttaChart?.resize()
}

const parseIdArray = (raw?: string) => {
  const parsed = safeParse<number[]>(raw, [])
  return Array.isArray(parsed) ? parsed.map(Number).filter(Boolean) : []
}

const safeParse = <T,>(raw: string | undefined, fallback: T): T => {
  if (!raw) return fallback
  try {
    return JSON.parse(raw)
  } catch {
    return fallback
  }
}

const parseEventMap = (raw?: string) => {
  const parsed = safeParse<Record<string, unknown>>(raw, {})
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return []
  }
  return Object.entries(parsed).map(([key, value]) => ({
    key,
    value: String(value ?? '')
  }))
}

const formatCallbackResult = (value: unknown) => {
  if (value === undefined || value === null) {
    return '无返回数据'
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

const getSeverityName = (severity?: string) => {
  const map: Record<string, string> = { p0: 'P0', critical: 'P0', p1: 'P1', warning: 'P1', p2: 'P2', info: 'P2' }
  return severity ? map[severity] || severity : '-'
}

const getSeverityMark = (severity?: string) => {
  if (severity === 'p0' || severity === 'critical') return 'P0'
  if (severity === 'p1' || severity === 'warning') return 'P1'
  if (severity === 'p2' || severity === 'info') return 'P2'
  return getSeverityName(severity).slice(0, 2)
}

const getSeverityTag = (severity?: string) => {
  const level = getSeverityName(severity)
  if (level === 'P0') return 'danger'
  if (level === 'P1') return 'warning'
  if (level === 'P2') return 'info'
  return 'info'
}

const getStateName = (state?: string) => {
  const map: Record<string, string> = {
    inactive: '正常',
    pending: '预告警',
    firing: '告警中',
    error: '告警中',
    processing: '处理中',
    silenced: '静默中',
    recovering: '待恢复',
    recovered: '已恢复'
  }
  return state ? map[state] || state : '-'
}

const getStateTag = (state?: string) => {
  const map: Record<string, string> = {
    inactive: 'success',
    pending: 'warning',
    firing: 'danger',
    error: 'danger',
    processing: 'primary',
    silenced: 'info',
    recovering: 'warning',
    recovered: 'success'
  }
  return state ? map[state] || 'info' : 'info'
}

const formatDateTime = (date?: string) => {
  if (!date) return '-'
  const d = new Date(date)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const formatEventValue = (value?: number) => {
  const number = Number(value)
  if (!Number.isFinite(number)) return '-'
  return Number.isInteger(number) ? String(number) : String(Number(number.toFixed(6)))
}

const conditionText = (condition?: string) => {
  const map: Record<string, string> = {
    gt: '>',
    gte: '>=',
    lt: '<',
    lte: '<=',
    eq: '=',
    neq: '!='
  }
  return condition ? map[condition] || condition : '-'
}

const buildEventConditionText = (event: MonitorAlertEvent) => {
  return `${formatEventValue(event.value)} ${conditionText(event.condition)} ${formatEventValue(event.threshold)}`
}

const getEventDetailText = (event: MonitorAlertEvent) => {
  const annotations = parseEventMap(event.annotations)
  const detail = annotations.find(item => ['description', 'summary', 'detail', 'message'].includes(item.key.toLowerCase()))
  return formatFriendlyEventMessage(detail?.value || event.message || '-')
}

const formatFriendlyEventMessage = (message?: string) => {
  const text = String(message || '').trim()
  if (!text) return '-'
  const lower = text.toLowerCase()
  if (lower.includes('duplicate time series') || lower.includes('many-to-many matching')) {
    return '查询语句存在重复时间序列，数据源无法完成标签匹配，请检查 PromQL 的 on()/group_left()/聚合维度。'
  }
  if (lower.includes('cannot execute') || lower.includes('cannot evaluate')) {
    return '数据源无法执行当前查询语句，请检查查询条件和指标标签是否匹配。'
  }
  if (lower.includes('parse error') || lower.includes('bad_data') || lower.includes('invalid parameter')) {
    return '查询语句解析失败，请检查 PromQL / LogQL / DSL 语法。'
  }
  return text
}

const goToEventRule = (event: MonitorAlertEvent) => {
  if (!event?.ruleId) return
  router.push({ path: '/monitor/rules', query: { ruleId: String(event.ruleId) } })
}

const openExportDialog = (scope: EventScope) => {
  exportForm.scope = scope
  exportForm.dateRange = scope === 'history' ? [...historyDateRange.value] : []
  exportForm.pageSize = 200
  exportForm.maxRows = 5000
  exportDialogVisible.value = true
}

const confirmExportEvents = async () => {
  exporting.value = true
  try {
    const rows: MonitorAlertEvent[] = []
    let page = 1
    let total = 0
    while (rows.length < exportForm.maxRows) {
      const response = await getMonitorAlertEvents(buildEventParams(exportForm.scope, {
        page,
        pageSize: exportForm.pageSize,
        dateRange: exportForm.scope === 'history' ? exportForm.dateRange : []
      }))
      const list = response?.list || []
      total = response?.total || total
      rows.push(...list)
      if (!list.length || list.length < exportForm.pageSize || rows.length >= total) break
      page += 1
    }
    downloadEventsCsv(rows.slice(0, exportForm.maxRows), exportForm.scope, total)
  } finally {
    exporting.value = false
  }
}

const downloadEventsCsv = (rows: MonitorAlertEvent[], scope: EventScope, total = rows.length) => {
  if (!rows.length) {
    ElMessage.warning('暂无可导出的告警事件')
    return
  }
  const headers = ['事件名称', '消息', '等级', '状态', '开始时间', '结束时间', '持续时长', '认领人']
  const csvRows = rows.map(row => [
    row.ruleName,
    row.message,
    getSeverityName(row.severity),
    getStateName(row.state),
    formatDateTime(row.startedAt),
    formatDateTime(row.endedAt),
    formatDuration(getEventDurationSeconds(row)),
    row.acknowledgedBy || ''
  ])
  const csv = [headers, ...csvRows].map(items => items.map(escapeCsv).join(',')).join('\n')
  const blob = new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `${center.value?.name || 'fault-center'}-${scope}-events.csv`
  link.click()
  URL.revokeObjectURL(url)
  exportDialogVisible.value = false
  if (total > rows.length) {
    ElMessage.warning(`已导出前 ${rows.length} 条，仍有 ${total - rows.length} 条未导出，可调大最多导出条数`)
  } else {
    ElMessage.success(`已导出 ${rows.length} 条告警事件`)
  }
}

const escapeCsv = (value: unknown) => {
  const text = String(value ?? '')
  if (/[",\n]/.test(text)) {
    return `"${text.replace(/"/g, '""')}"`
  }
  return text
}

watch(() => [route.params.id, route.query.tab, route.query.query, route.query.eventId], loadAll)
watch(sloData, () => nextTick(renderSloCharts), { deep: true })
onMounted(() => {
  loadAll()
  durationTimer = window.setInterval(() => {
    currentTime.value = Date.now()
  }, 1000)
  window.addEventListener('resize', handleChartResize)
})
onBeforeUnmount(() => {
  if (durationTimer) {
    window.clearInterval(durationTimer)
    durationTimer = undefined
  }
  window.removeEventListener('resize', handleChartResize)
  mttrChart?.dispose()
  mttaChart?.dispose()
  mttrChart = null
  mttaChart = null
})
</script>

<style scoped>
.fault-detail-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  color: #1f2937;
}

.fault-identity,
.watch-chart-card,
.event-board,
.config-section,
.setting-card {
  background: #fff;
  border: 1px solid #dedede;
  border-radius: 8px;
}

.fault-identity {
  padding: 18px 22px 14px;
}

.breadcrumb-line {
  display: flex;
  align-items: center;
  gap: 9px;
  color: #8a8f98;
  font-size: 14px;
}

.breadcrumb-line strong {
  color: #111827;
  font-weight: 700;
}

.back-icon {
  width: 28px;
  min-height: 28px;
  padding: 0;
  color: #8a8f98;
}

.home-icon {
  color: #9aa1aa;
}

.identity-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 16px;
  margin-top: 22px;
}

.identity-item {
  display: flex;
  align-items: center;
  min-width: 0;
  color: #8a8f98;
  font-size: 15px;
}

.identity-item strong {
  min-width: 0;
  overflow: hidden;
  color: #111827;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.identity-center {
  justify-content: center;
}

.identity-right {
  justify-content: flex-end;
}

.inline-edit {
  width: 24px;
  min-height: 24px;
  margin-left: 6px;
  padding: 0;
  color: #344054;
  border-radius: 6px;
}

.inline-edit:hover {
  color: #1677ff;
  background: #eef5ff;
}

.mini-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(130px, 1fr)) auto;
  gap: 12px;
  align-items: center;
  margin-top: 18px;
}

.mini-stat {
  min-height: 62px;
  padding: 10px 12px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fafafa;
}

.mini-stat span,
.chart-card-head span,
.chart-card-head p,
.section-heading p {
  color: #667085;
}

.mini-stat span {
  display: block;
  font-size: 12px;
}

.mini-stat strong {
  display: block;
  margin-top: 5px;
  color: #111827;
  font-size: 24px;
  font-weight: 800;
  line-height: 1;
}

.mini-stat.danger strong {
  color: #f04438;
}

.mini-stat.success strong {
  color: #52c41a;
}

.identity-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.basic-info-form {
  padding-top: 4px;
}

.slo-card-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.watch-chart-card {
  min-height: 302px;
  padding: 22px 24px 14px;
}

.chart-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.chart-card-head h3 {
  margin: 0;
  color: #111827;
  font-size: 16px;
  font-weight: 800;
}

.chart-card-head span {
  display: block;
  margin-top: 7px;
  font-size: 13px;
  letter-spacing: 0;
}

.chart-card-head p {
  margin: 20px 0 0;
  font-size: 13px;
}

.metric-icon {
  margin-top: 10px;
  font-size: 20px;
}

.metric-icon.repair {
  color: #faad14;
}

.metric-icon.response {
  color: #667085;
}

.single-slo-chart {
  width: 100%;
  height: 184px;
  margin-top: 8px;
}

.watch-tabs {
  margin-top: 2px;
}

.watch-tabs :deep(.el-tabs__header) {
  margin: 0 0 14px;
}

.watch-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 0;
}

.watch-tabs :deep(.el-tabs__item) {
  height: 40px;
  padding: 0 18px;
  color: #111827;
  font-size: 16px;
  font-weight: 700;
}

.watch-tabs :deep(.el-tabs__item.is-active) {
  color: #1677ff;
  background: #e8f3ff;
  box-shadow: inset 0 -3px 0 #1677ff;
}

.watch-tabs :deep(.el-tabs__active-bar) {
  display: none;
}

.event-board {
  padding: 0;
  overflow: hidden;
}

.watch-filter-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px;
  border-bottom: 1px solid #eef0f3;
}

.watch-filter-bar :deep(.el-select) {
  width: 180px;
}

.keyword-input {
  width: 250px;
}

.date-range {
  width: 360px;
}

.batch-action {
  margin-left: auto;
  min-width: 118px;
  justify-content: center;
  gap: 6px;
  border-color: #d8dee8;
  color: #111827;
  font-weight: 700;
  background: #fff;
}

.batch-action :deep(.el-icon) {
  margin-left: 2px;
  font-size: 14px;
}

.batch-action:not(.is-disabled):hover,
.batch-action:not(.is-disabled):focus {
  border-color: #111827;
  color: #111827;
  background: #f8fafc;
}

.watch-table {
  width: 100%;
}

.watch-table :deep(.el-table__header th) {
  height: 56px;
  border-bottom: 1px solid #eceff3;
}

.watch-table :deep(.el-table__row) {
  height: 74px;
}

.watch-table :deep(.el-table__cell) {
  border-bottom: 1px solid #edf0f2;
}

.watch-table :deep(.el-table__body-wrapper) {
  min-height: 360px;
}

.event-info-cell {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
  padding-left: 12px;
}

.event-rail {
  position: absolute;
  left: -12px;
  top: -18px;
  bottom: -18px;
  width: 5px;
  background: #b7e8ff;
}

.event-info-cell.firing .event-rail {
  background: #ffa500;
}

.event-info-cell.error .event-rail {
  background: #ff4d4f;
}

.event-info-cell.recovered .event-rail {
  background: #91d5ff;
}

.event-flame {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #ff5b35;
  color: #fff;
  font-size: 10px;
  font-weight: 800;
  line-height: 1;
}

.event-copy {
  min-width: 0;
}

.event-title {
  max-width: 100%;
  margin: 0;
  padding: 0;
  overflow: hidden;
  border: 0;
  background: transparent;
  color: #1677ff;
  font-size: 13px;
  font-weight: 700;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.event-copy p {
  display: -webkit-box;
  margin: 5px 0 0;
  overflow: hidden;
  color: #8a8f98;
  font-size: 13px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 1;
}

.time-text {
  color: #111827;
  font-size: 16px;
  font-weight: 500;
}

.time-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.time-stack strong {
  color: #111827;
  font-size: 15px;
  font-weight: 600;
}

.time-stack span {
  color: #111827;
  font-size: 15px;
}

.duration-cell {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.duration-text {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #667085;
  font-size: 13px;
}

.duration-text strong {
  color: #111827;
  font-size: 14px;
  font-weight: 800;
}

.duration-bars {
  display: flex;
  align-items: center;
  gap: 4px;
}

.duration-bars span {
  width: 11px;
  height: 10px;
  border-radius: 3px;
  background: #f3f4f6;
}

.duration-bars span.active {
  background: #ffeb00;
}

.duration-bars span.active:nth-child(n + 6) {
  background: #ff9500;
}

.duration-bars span.active:nth-child(n + 8) {
  background: #ff1f1f;
}

.state-tag {
  font-weight: 700;
}

.more-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 6px;
  color: #111827;
}

.more-btn:hover,
.more-btn:focus {
  border-color: #1677ff;
  background: #e8f3ff;
  color: #1677ff;
}

.table-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 16px 14px;
  border-top: 1px solid #eef0f3;
}

.table-footer > span {
  color: #667085;
  font-size: 13px;
}

.config-section {
  padding: 22px 24px;
}

.aggregation-row,
.section-heading,
.rule-list-head.watch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
}

.aggregation-row {
  justify-content: flex-start;
  margin-bottom: 34px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #111827;
  font-size: 18px;
}

.section-title.small {
  font-size: 15px;
}

.section-title .el-icon {
  color: #1677ff;
}

.silence-filter {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 14px;
}

.silence-filter .el-input {
  width: 360px;
}

.dark-action {
  border-color: #1677ff;
  background: #1677ff;
  color: #fff;
}

.dark-action:hover,
.dark-action:focus {
  border-color: #0958d9;
  background: #0958d9;
  color: #fff;
}

.config-table {
  margin-top: 18px;
  border: 1px solid #edf0f2;
  border-radius: 8px;
  overflow: hidden;
}

.operator-cell {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.operator-cell strong {
  color: #111827;
  font-size: 15px;
}

.operator-cell span {
  color: #98a2b3;
  font-size: 12px;
}

.danger-link {
  color: #ff4d4f;
}

.section-heading {
  margin-bottom: 22px;
}

.section-heading p {
  margin: 7px 0 0;
  font-size: 14px;
}

.notify-grid {
  display: grid;
  grid-template-columns: minmax(320px, 0.8fr) minmax(0, 1.8fr);
  gap: 22px;
}

.setting-card {
  min-height: 260px;
  padding: 24px 28px;
}

.setting-card header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 22px;
  padding-bottom: 18px;
  border-bottom: 1px solid #edf0f2;
}

.card-edit-actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  flex: 0 0 auto;
  gap: 8px;
}

.inline-edit-btn {
  color: #111827;
  font-size: 14px;
  font-weight: 700;
}

.inline-edit-btn:hover,
.inline-edit-btn:focus {
  color: #1677ff;
}

.edit-save-btn {
  min-height: 32px;
  padding: 0 12px;
  border-color: #050505;
  background: #050505;
  color: #fff;
  font-weight: 700;
}

.edit-save-btn:hover,
.edit-save-btn:focus {
  border-color: #222;
  background: #222;
  color: #fff;
}

.edit-cancel-btn {
  min-height: 32px;
  padding: 0 12px;
  border-color: #d0d5dd;
  background: #fff;
  color: #111827;
  font-weight: 650;
}

.edit-cancel-btn:hover,
.edit-cancel-btn:focus {
  border-color: #98a2b3;
  background: #f8fafc;
  color: #111827;
}

.setting-form,
.upgrade-card {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.setting-form label,
.upgrade-card label {
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.setting-form label > span,
.upgrade-card label > span {
  color: #111827;
  font-size: 14px;
  font-weight: 700;
}

.switch-line {
  flex-direction: row !important;
  align-items: center;
  justify-content: space-between;
}

.repeat-inputs,
.suffix-input {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 88px;
}

.repeat-inputs + .repeat-inputs {
  margin-top: 8px;
}

.repeat-inputs em,
.suffix-input em {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #dcdfe6;
  border-left: 0;
  border-radius: 0 4px 4px 0;
  background: #f7f8fa;
  color: #8a8f98;
  font-size: 13px;
  font-style: normal;
}

.repeat-inputs :deep(.el-input__wrapper),
.suffix-input :deep(.el-input__wrapper) {
  border-radius: 4px 0 0 4px;
}

.repeat-inputs :deep(.el-input-number),
.suffix-input :deep(.el-input-number) {
  width: 100%;
}

.route-card {
  min-height: 310px;
}

.empty-route {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  color: #98a2b3;
  font-size: 14px;
}

.empty-box {
  width: 58px;
  height: 42px;
  margin-bottom: 12px;
  border: 1px solid #d8dee9;
  border-radius: 8px 8px 12px 12px;
  background: linear-gradient(#fff, #f7f8fa);
  box-shadow: 0 12px 24px rgba(15, 23, 42, .06);
}

.route-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.route-item {
  display: grid;
  grid-template-columns: 180px minmax(200px, 1fr) minmax(220px, 1fr) 70px 40px;
  gap: 10px;
  align-items: center;
}

.route-item.is-readonly {
  grid-template-columns: 180px minmax(200px, 1fr) minmax(220px, 1fr) 70px;
}

.route-add-tile {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 72px;
  border: 1px dashed #cfd8e3;
  border-radius: 8px;
  background: #fff;
  color: #667085;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  transition: border-color .16s ease, background .16s ease, color .16s ease;
}

.route-add-tile:hover,
.route-add-tile:focus {
  border-color: #1677ff;
  background: #f5f9ff;
  color: #1677ff;
}

.notice-preview {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 18px;
}

.notice-object-card {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  min-height: 82px;
  padding: 13px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fff;
}

.notice-object-card strong,
.notice-object-card span {
  display: block;
}

.notice-object-card strong {
  color: #111827;
  font-size: 14px;
}

.notice-object-card span {
  margin-top: 5px;
  color: #667085;
  font-size: 12px;
}

.user-pills {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 5px;
  min-width: 120px;
}

.user-pills em {
  padding: 2px 7px;
  border: 1px solid #dbe8ff;
  border-radius: 999px;
  background: #eef4ff;
  color: #1d4ed8;
  font-size: 12px;
  font-style: normal;
}

.upgrade-heading {
  align-items: flex-start;
}

.upgrade-switches {
  display: flex;
  align-items: center;
  gap: 18px;
  margin-top: 12px;
}

.upgrade-card {
  min-height: 220px;
}

.fault-detail-page {
  gap: 14px;
  background: #fff;
  color: #111827;
  font-size: 14px;
}

.fault-identity {
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
}

.breadcrumb-line {
  gap: 8px;
  color: #8c8c8c;
  font-size: 14px;
  line-height: 32px;
}

.breadcrumb-line strong {
  color: #111827;
  font-weight: 600;
}

.identity-grid {
  margin-top: 14px;
}

.identity-item {
  color: #8c8c8c;
  font-size: 14px;
}

.identity-item strong {
  color: #111827;
  font-size: 14px;
  font-weight: 600;
}

.inline-edit {
  width: 26px;
  min-height: 26px;
  color: #111827;
}

.inline-edit:hover {
  background: transparent;
  color: #1677ff;
}

.mini-stats {
  display: none;
}

.slo-card-grid {
  gap: 16px;
}

.watch-chart-card {
  height: 250px;
  min-height: 250px;
  padding: 20px 20px 12px;
  border-color: #d9d9d9;
  border-radius: 12px;
  box-shadow: none;
}

.chart-card-head {
  margin-bottom: 12px;
}

.chart-card-head h3 {
  font-size: 13px;
  font-weight: 600;
}

.chart-card-head span {
  margin-top: 2px;
  color: #6b7280;
  font-size: 11px;
}

.chart-card-head p {
  margin: 16px 0 0;
  color: #6b7280;
  font-size: 12px;
}

.single-slo-chart {
  height: 140px;
  margin-top: 0;
}

.watch-tabs {
  margin-top: 0;
}

.watch-tabs :deep(.el-tabs__header) {
  margin: 0 0 14px;
}

.watch-tabs :deep(.el-tabs__item) {
  height: 42px;
  padding: 0 18px;
  color: #111827;
  font-size: 14px;
  font-weight: 600;
}

.watch-tabs :deep(.el-tabs__item.is-active) {
  color: #1677ff;
  background: transparent;
  box-shadow: inset 0 -2px 0 #1677ff;
}

.event-board {
  border-color: #f0f0f0;
  border-radius: 8px;
  background: #fff;
}

.watch-filter-bar {
  gap: 10px;
  padding: 16px;
  border-bottom-color: #f0f0f0;
}

.watch-filter-bar :deep(.el-input__wrapper),
.watch-filter-bar :deep(.el-select__wrapper),
.watch-filter-bar :deep(.el-date-editor) {
  min-height: 34px;
  border-radius: 6px;
  box-shadow: 0 0 0 1px #d9d9d9 inset;
}

.watch-filter-bar :deep(.el-select) {
  width: 180px;
}

.keyword-input {
  width: 240px;
}

.date-range {
  width: 360px;
}

.watch-filter-bar .el-button {
  min-height: 34px;
  padding: 0 14px;
  border-color: #d9d9d9;
  border-radius: 6px;
  color: #111827;
  font-size: 14px;
}

.watch-table :deep(.el-table__header th) {
  height: 52px;
  background: #fafafa !important;
  color: #111827;
  font-size: 14px;
  font-weight: 600;
}

.watch-table :deep(.el-table__row) {
  height: 76px;
}

.watch-table :deep(.el-table__cell) {
  border-bottom-color: #f0f0f0;
}

.watch-table :deep(.cell) {
  padding: 0 12px;
}

.watch-table :deep(.el-table__body-wrapper) {
  min-height: 280px;
}

.event-title {
  color: #1677ff;
  font-size: 14px;
  font-weight: 600;
}

.event-copy p {
  color: #8c8c8c;
  font-size: 13px;
}

.event-flame {
  width: 20px;
  height: 20px;
  background: #ff4d4f;
  font-size: 10px;
}

.time-text,
.time-stack strong,
.time-stack span {
  color: #111827;
  font-size: 14px;
  font-weight: 500;
}

.duration-text strong {
  font-size: 14px;
  font-weight: 700;
}

.duration-bars {
  gap: 4px;
}

.duration-bars span {
  width: 9px;
  height: 8px;
  border-radius: 2px;
  background: #f5f5f5;
}

.table-footer {
  min-height: 56px;
  padding: 10px 16px;
  border-top-color: #f0f0f0;
}

.config-section {
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
}

.section-heading {
  margin-bottom: 24px;
}

.section-title {
  gap: 12px;
  font-size: 16px;
}

.section-title.small {
  gap: 8px;
  font-size: 14px;
}

.section-heading p {
  margin: 6px 0 0;
  color: #8c8c8c;
  font-size: 14px;
}

.aggregation-row {
  align-items: flex-start;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 26px;
}

.rule-list-head.watch {
  align-items: flex-end;
  margin-bottom: 16px;
}

.silence-filter {
  gap: 10px;
  margin-top: 12px;
}

.silence-filter .el-input {
  width: 300px;
}

.config-table {
  margin-top: 0;
  border-color: #f0f0f0;
  border-radius: 8px;
}

.dark-action {
  min-height: 34px;
  border-color: #000;
  border-radius: 6px;
  background: #000;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
}

.dark-action:hover,
.dark-action:focus {
  border-color: #000;
  background: #000;
  color: #fff;
}

.notify-grid {
  grid-template-columns: minmax(360px, 7fr) minmax(0, 17fr);
  gap: 24px;
}

.setting-card {
  min-height: 0;
  padding: 24px;
  border-color: #f0f0f0;
  border-radius: 12px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, .03), 0 1px 6px -1px rgba(0, 0, 0, .02);
}

.setting-card header {
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom-color: #f0f0f0;
}

.inline-edit-btn {
  min-height: 24px;
  padding: 0;
  color: #111827;
  font-size: 14px;
  font-weight: 600;
}

.edit-save-btn,
.edit-cancel-btn {
  min-height: 28px;
  padding: 0 10px;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
}

.edit-save-btn {
  border-color: #000;
  background: #000;
}

.edit-save-btn:hover,
.edit-save-btn:focus {
  border-color: #000;
  background: #000;
}

.edit-cancel-btn {
  border-color: #d9d9d9;
}

.setting-form,
.upgrade-card {
  gap: 16px;
}

.setting-form label,
.upgrade-card label {
  gap: 8px;
}

.setting-form label > span,
.upgrade-card label > span {
  color: #111827;
  font-size: 14px;
  font-weight: 600;
}

.repeat-inputs,
.suffix-input {
  grid-template-columns: minmax(0, 1fr) 88px;
}

.repeat-inputs :deep(.el-input-number),
.suffix-input :deep(.el-input-number),
.setting-form :deep(.el-select),
.upgrade-card :deep(.el-select) {
  width: 100%;
}

.setting-card :deep(.el-input__wrapper),
.setting-card :deep(.el-select__wrapper),
.setting-card :deep(.el-input-number),
.setting-card :deep(.el-input-number .el-input__wrapper) {
  min-height: 34px;
  border-radius: 6px;
}

.setting-card :deep(.is-disabled .el-input__wrapper),
.setting-card :deep(.is-disabled.el-select__wrapper),
.setting-card :deep(.el-input__wrapper.is-disabled) {
  background: #f5f7fb;
}

.repeat-inputs em,
.suffix-input em {
  border-color: #d9d9d9;
  background: #fafafa;
  color: #8c8c8c;
  font-size: 13px;
}

.route-card {
  min-height: 300px;
}

.empty-route {
  min-height: 190px;
  color: #bfbfbf;
}

.empty-box {
  width: 44px;
  height: 34px;
  border-color: #d9d9d9;
  background: #fff;
  box-shadow: none;
}

.route-list {
  gap: 16px;
}

.route-rule-card {
  padding: 16px;
  border: 1px solid #e8e8e8;
  border-radius: 12px;
  background: #fff;
}

.route-label-tags {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 16px;
}

.route-meta-title {
  width: 100%;
  color: #8c8c8c;
  font-size: 12px;
}

.route-condition-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 16px;
}

.route-condition-row {
  display: grid;
  grid-template-columns: minmax(150px, 1fr) 96px minmax(170px, 1fr) 24px;
  gap: 8px;
  align-items: center;
}

.operator-select {
  width: 96px;
}

.condition-add-tile,
.route-add-tile {
  width: 100%;
  min-height: 36px;
  border: 1px dashed #d9d9d9;
  border-radius: 8px;
  background: #fafafa;
  color: #8c8c8c;
  font-size: 14px;
  font-weight: 400;
}

.route-add-tile {
  min-height: 68px;
  border-width: 2px;
  border-radius: 12px;
}

.condition-add-tile:hover,
.condition-add-tile:focus,
.route-add-tile:hover,
.route-add-tile:focus {
  border-color: #1677ff;
  background: #f0f5ff;
  color: #1677ff;
}

.route-notice-target {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 24px;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px dashed #e8e8e8;
}

.route-target-label {
  display: flex;
  align-items: center;
  grid-column: 1 / -1;
  gap: 6px;
  color: #111827;
  font-size: 13px;
}

.route-target-label .el-icon {
  color: #1677ff;
}

.route-delete-btn,
.route-remove-btn {
  width: 24px;
  min-height: 24px;
  padding: 0;
}

.notice-preview {
  grid-template-columns: minmax(320px, 7fr) minmax(0, 17fr);
  margin-top: 24px;
}

.notice-object-card {
  min-height: 72px;
  padding: 16px 20px;
  border-color: #e6f0ff;
  border-radius: 8px;
}

.upgrade-heading {
  align-items: flex-start;
  margin-bottom: 24px;
}

.upgrade-heading .inline-edit-btn {
  min-height: 34px;
  padding: 0 14px;
  border: 1px solid #000;
  border-radius: 6px;
  background: #000;
  color: #fff;
  font-weight: 500;
}

.upgrade-heading .inline-edit-btn:hover,
.upgrade-heading .inline-edit-btn:focus {
  border-color: #000;
  background: #000;
  color: #fff;
}

.upgrade-switches {
  gap: 16px;
  margin-top: 12px;
}

.upgrade-card {
  min-height: 0;
}

.inline-edit-field {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 260px;
  margin-top: -5px;
}

.inline-edit-field :deep(.el-input) {
  width: 200px;
}

.inline-edit-field :deep(.el-input__wrapper) {
  min-height: 32px;
  border-radius: 6px;
  box-shadow: 0 0 0 1px #d9d9d9 inset;
}

.inline-save {
  width: 24px;
  min-height: 24px;
  padding: 0;
  color: #111827;
}

.inline-save:hover,
.inline-save:focus {
  background: transparent;
  color: #1677ff;
}

.identity-grid {
  align-items: center;
}

.metric-icon {
  color: transparent !important;
}

.section-title .el-icon {
  color: #111827;
}

.section-title.small .el-icon,
.route-target-label .el-icon {
  color: #1677ff;
}

.watch-radio-buttons :deep(.el-radio-button__inner) {
  height: 32px;
  padding: 0 15px;
  border-color: #d9d9d9;
  background: #fff;
  color: #111827;
  font-size: 14px;
  line-height: 30px;
  box-shadow: none;
}

.watch-radio-buttons :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  border-color: #1677ff;
  background: #fff;
  color: #1677ff;
  box-shadow: -1px 0 0 0 #1677ff;
}

.silence-name-link {
  max-width: 100%;
  padding: 0;
  overflow: hidden;
  border: 0;
  background: transparent;
  color: #1677ff;
  font-size: 14px;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.silence-tags {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px;
  min-width: 0;
}

.time-text.small {
  font-size: 12px;
  color: #999;
  white-space: nowrap;
}

.pill-tag {
  border-radius: 12px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 500;
}

.export-event-dialog :deep(.el-dialog__header) {
  margin-right: 0;
  padding: 18px 20px 14px;
  border-bottom: 1px solid #f0f0f0;
}

.export-event-dialog :deep(.el-dialog__body) {
  padding: 18px 20px;
}

.export-event-form {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.export-date-range,
.export-select {
  width: 100%;
}

.export-event-form :deep(.el-input-number) {
  width: 180px;
}

.watch-drawer :deep(.el-drawer__header) {
  margin-bottom: 0;
  padding: 18px 20px;
  border-bottom: 1px solid #f0f0f0;
  color: #111827;
  font-size: 16px;
  font-weight: 600;
}

.watch-drawer :deep(.el-drawer__body) {
  padding: 20px;
}

.watch-drawer :deep(.el-drawer__footer) {
  padding: 14px 20px;
  border-top: 1px solid #f0f0f0;
}

.event-detail-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.event-detail-metrics,
.event-detail-table-card,
.event-detail-section {
  border: 1px solid #e5e9f2;
  border-radius: 8px;
  background: #fff;
}

.event-detail-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  overflow: hidden;
}

.event-detail-metric {
  min-width: 0;
  padding: 12px 14px;
  border-right: 1px solid #edf1f7;
}

.event-detail-metric:last-child {
  border-right: 0;
}

.event-detail-metric span,
.detail-section-title {
  display: block;
  color: #667085;
  font-size: 12px;
  font-weight: 650;
}

.event-detail-metric strong {
  display: block;
  margin-top: 6px;
  color: #101828;
  font-size: 14px;
  font-weight: 700;
}

.event-detail-metric :deep(.el-tag) {
  margin-top: 6px;
}

.event-detail-table-card {
  overflow: hidden;
}

.event-detail-table {
  display: flex;
  flex-direction: column;
}

.event-detail-row {
  display: grid;
  grid-template-columns: 132px minmax(0, 1fr);
  min-height: 52px;
  border-bottom: 1px solid #edf1f7;
}

.event-detail-row:last-child {
  border-bottom: 0;
}

.event-detail-label {
  display: flex;
  align-items: center;
  padding: 13px 16px;
  border-right: 1px solid #edf1f7;
  background: #fbfcfe;
  color: #667085;
  font-size: 13px;
  font-weight: 700;
}

.event-detail-value {
  min-width: 0;
  padding: 13px 16px;
  color: #101828;
  font-size: 13px;
  line-height: 1.65;
  overflow-wrap: anywhere;
}

.event-detail-value.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.event-detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.event-detail-tags :deep(.el-tag) {
  height: 26px;
  border-color: #93c5fd;
  background: #eff6ff;
  color: #2563eb;
  font-weight: 600;
}

.event-rule-link {
  border: 0;
  padding: 0;
  background: transparent;
  color: #1677ff;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

.event-rule-link:hover {
  color: #0958d9;
  text-decoration: underline;
}

.event-notify-error {
  display: block;
  margin-top: 6px;
  color: #b42318;
  font-size: 12px;
  line-height: 1.5;
}

.condition-text {
  white-space: pre-wrap;
}

.event-detail-message {
  min-height: 280px;
  margin: 0;
  padding: 12px 14px;
  border: 1px solid #d8e2f0;
  border-radius: 6px;
  background: #fff;
  color: #1f2937;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.event-detail-section {
  padding: 14px 16px;
}

.detail-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.callback-list {
  min-height: 80px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.callback-card {
  padding: 12px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fbfcfe;
}

.callback-card header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.callback-card strong,
.callback-card span {
  display: block;
}

.callback-card strong {
  color: #101828;
  font-size: 13px;
  font-weight: 700;
}

.callback-card span {
  margin-top: 3px;
  color: #667085;
  font-size: 12px;
}

.callback-query-code {
  padding: 9px 10px;
  border: 1px solid #e8edf5;
  border-radius: 8px;
  background: #ffffff;
}

.callback-query-code span {
  margin: 0 0 5px;
  color: #667085;
  font-size: 12px;
  font-weight: 600;
}

.callback-query-code code {
  display: block;
  color: #1f2937;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.callback-original-query {
  display: block;
  margin-top: 6px;
  color: #98a2b3;
  font-size: 12px;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.callback-error {
  margin-top: 10px;
  padding: 9px 10px;
  border: 1px solid #fecaca;
  border-radius: 8px;
  background: #fef2f2;
  color: #b42318;
  font-size: 12px;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.callback-raw-collapse {
  margin-top: 10px;
  border: 1px solid #edf1f7;
  border-radius: 8px;
  background: #fff;
}

.callback-raw-collapse :deep(.el-collapse-item__header) {
  height: 34px;
  padding: 0 10px;
  border: 0;
  color: #667085;
  font-size: 12px;
  font-weight: 600;
}

.callback-raw-collapse :deep(.el-collapse-item__wrap) {
  border: 0;
}

.callback-raw-collapse :deep(.el-collapse-item__content) {
  padding: 0 10px 10px;
}

.callback-card pre {
  max-height: 260px;
  overflow: auto;
}

.empty-detail-text {
  display: inline-flex;
  margin-top: 8px;
  color: #98a2b3;
  font-size: 13px;
}

.silence-drawer-form :deep(.el-form-item__label) {
  color: #111827;
  font-size: 14px;
  font-weight: 500;
}

.silence-drawer-form :deep(.el-input__wrapper) {
  min-height: 34px;
  border-radius: 6px;
  box-shadow: 0 0 0 1px #d9d9d9 inset;
}

@media (max-width: 1180px) {
  .mini-stats,
  .identity-grid,
  .slo-card-grid,
  .notify-grid,
  .notice-preview {
    grid-template-columns: 1fr;
  }

  .identity-center,
  .identity-right {
    justify-content: flex-start;
  }

  .identity-actions {
    justify-content: flex-start;
  }

  .watch-filter-bar,
  .aggregation-row,
  .section-heading,
  .rule-list-head.watch {
    align-items: stretch;
    flex-direction: column;
  }

  .batch-action {
    margin-left: 0;
  }

  .watch-filter-bar :deep(.el-select),
  .keyword-input,
  .date-range,
  .silence-filter .el-input {
    width: 100%;
  }

  .silence-filter {
    align-items: stretch;
    flex-direction: column;
  }

  .route-item {
    grid-template-columns: 1fr;
  }

  .event-detail-drawer {
    width: 88% !important;
  }

  .event-detail-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .event-detail-metric:nth-child(2n) {
    border-right: 0;
  }
}

@media (max-width: 640px) {
  .fault-identity,
  .watch-chart-card,
  .config-section {
    padding: 16px;
  }

  .watch-tabs :deep(.el-tabs__item) {
    padding: 0 10px;
    font-size: 14px;
  }

  .table-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .event-detail-drawer {
    width: 100% !important;
  }

  .event-detail-metrics {
    grid-template-columns: 1fr;
  }

  .event-detail-metric {
    border-right: 0;
    border-bottom: 1px solid #edf1f7;
  }

  .event-detail-metric:last-child {
    border-bottom: 0;
  }

  .event-detail-row {
    grid-template-columns: 1fr;
  }

  .event-detail-label {
    border-right: 0;
    border-bottom: 1px solid #edf1f7;
  }
}
</style>
