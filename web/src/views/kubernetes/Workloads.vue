<template>
  <div class="workloads-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Tools /></el-icon>
        </div>
        <div>
          <h2 class="page-title">工作负载</h2>
          <p class="page-subtitle">管理 Kubernetes 工作负载资源</p>
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
        <el-button class="black-button" @click="loadWorkloads">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 工作负载类型标签 -->
    <div class="workload-types-bar">
      <div
        v-for="type in workloadTypes"
        :key="type.value"
        :class="['type-tab', { active: selectedType === type.value }]"
        @click="handleTypeChange(type.value)"
      >
        <el-icon class="type-icon">
          <component :is="type.icon" />
        </el-icon>
        <span class="type-label">{{ type.label }}</span>
        <span class="type-count" v-if="type.count !== undefined">({{ type.count }})</span>
      </div>
    </div>

    <!-- 操作栏 -->
    <div class="action-bar">
      <div class="search-section">
        <el-input
          v-model="searchName"
          placeholder="搜索工作负载名称..."
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
          v-model="selectedNamespace"
          placeholder="所有命名空间"
          clearable
          filterable
          @change="handleSearch"
          class="namespace-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><FolderOpened /></el-icon>
          </template>
          <el-option
            v-for="ns in namespaceList"
            :key="ns.name"
            :label="ns.name"
            :value="ns.name"
          />
        </el-select>
      </div>

      <div class="action-buttons">
        <el-button type="primary" @click="handleAddWorkloadYAML" class="add-button">
          <el-icon><Document /></el-icon>
          YAML创建
        </el-button>

        <el-button
          v-if="selectedType !== 'Pod'"
          type="success"
          @click="handleAddWorkloadForm"
          class="add-button-form"
        >
          <el-icon><Edit /></el-icon>
          表单创建
        </el-button>
      </div>
    </div>

    <!-- 批量操作栏 -->
    <div v-if="selectedWorkloads.length > 0" class="batch-action-bar">
      <div class="batch-action-left">
        <span class="selected-count">已选择 {{ selectedWorkloads.length }} 项</span>
      </div>
      <div class="batch-action-right">
        <el-button
          v-if="selectedType === 'Deployment' || selectedType === 'StatefulSet' || selectedType === 'DaemonSet'"
          @click="handleBatchRestart"
          :loading="batchActionLoading"
          class="batch-btn"
        >
          <el-icon><Refresh /></el-icon>
          批量重启
        </el-button>
        <el-button
          v-if="selectedType === 'Deployment' || selectedType === 'StatefulSet'"
          @click="handleBatchPause"
          :loading="batchActionLoading"
          type="warning"
          class="batch-btn"
        >
          <el-icon><VideoPause /></el-icon>
          批量停止
        </el-button>
        <el-button
          v-if="selectedType === 'Deployment' || selectedType === 'StatefulSet'"
          @click="handleBatchResume"
          :loading="batchActionLoading"
          type="success"
          class="batch-btn"
        >
          <el-icon><VideoPlay /></el-icon>
          批量恢复
        </el-button>
        <el-button
          @click="handleBatchDelete"
          :loading="batchActionLoading"
          type="danger"
          class="batch-btn"
        >
          <el-icon><Delete /></el-icon>
          批量删除
        </el-button>
        <el-button @click="clearSelection" class="batch-btn">
          取消选择
        </el-button>
      </div>
    </div>

    <!-- 工作负载列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedWorkloadList"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        :row-style="{ height: '56px' }"
        :cell-style="{ padding: '8px 0' }"
        @selection-change="handleSelectionChange"
      >
        <!-- 选择列 -->
        <el-table-column type="selection" width="55" fixed="left" />
        <!-- 名称列（所有类型通用，但显示内容不同） -->
        <el-table-column label="名称" min-width="200" fixed="left">
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Tools /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="workload-name-cell">
              <div class="workload-type-icon-box">
                <el-icon class="workload-type-icon-gold" :size="18">
                  <Tools />
                </el-icon>
              </div>
              <div class="workload-name-content">
                <el-tooltip :content="row.name" placement="top" effect="light">
                  <div class="workload-name clickable" @click="row.type === 'Pod' ? handlePodWorkloadClick(row) : handleShowDetail(row)">{{ row.name }}</div>
                </el-tooltip>
                <!-- Pod类型显示容器，其他类型显示命名空间 -->
                <div v-if="selectedType === 'Pod'" class="workload-namespace">{{ row.containers || '-' }}</div>
                <div v-else class="workload-namespace">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <!-- Pod 专用列 -->
        <template v-if="selectedType === 'Pod'">
          <!-- CPU/内存 列 -->
          <el-table-column label="CPU/内存" min-width="150">
            <template #default="{ row }">
              <div class="resource-cell">
                <div v-if="row.cpu || row.memory" class="resource-item">
                  <span v-if="row.cpu" class="resource-value">{{ row.cpu }}</span>
                  <span v-if="row.cpu && row.memory" class="resource-separator"> / </span>
                  <span v-if="row.memory" class="resource-value">{{ row.memory }}</span>
                </div>
                <div v-else class="resource-empty">-</div>
              </div>
            </template>
          </el-table-column>

          <!-- 状态列 -->
          <el-table-column label="状态" min-width="150" align="center">
            <template #default="{ row }">
              <div :class="['status-badge', `status-${row.podStatus?.toLowerCase()}`]">
                {{ row.podStatus || '-' }}
              </div>
            </template>
          </el-table-column>

          <!-- 重启次数 -->
          <el-table-column label="重启次数" width="100" align="center">
            <template #default="{ row }">
              <span>{{ row.restartCount ?? '-' }}</span>
            </template>
          </el-table-column>

          <!-- 命名空间 -->
          <el-table-column label="命名空间" width="150">
            <template #default="{ row }">
              <span>{{ row.namespace }}</span>
            </template>
          </el-table-column>

          <!-- Pod IP -->
          <el-table-column label="PodIP" width="140" align="center">
            <template #default="{ row }">
              <span class="pod-ip">{{ row.podIP || '-' }}</span>
            </template>
          </el-table-column>

          <!-- 调度节点 -->
          <el-table-column label="调度节点" min-width="150">
            <template #default="{ row }">
              <span>{{ row.node || '-' }}</span>
            </template>
          </el-table-column>
        </template>

        <!-- DaemonSet 专用列 -->
        <template v-if="selectedType === 'DaemonSet'">
          <!-- 准备就绪 -->
          <el-table-column label="准备就绪" width="120" align="center">
            <template #default="{ row }">
              <div class="pod-count-cell">
                <span class="pod-count">{{ row.readyPods || 0 }}/{{ row.desiredPods || 0 }}</span>
              </div>
            </template>
          </el-table-column>

          <!-- 当前调度 -->
          <el-table-column label="当前调度" width="100" align="center">
            <template #default="{ row }">
              <span>{{ row.currentScheduled ?? '-' }}</span>
            </template>
          </el-table-column>

          <!-- 期望调度 -->
          <el-table-column label="期望调度" width="100" align="center">
            <template #default="{ row }">
              <span>{{ row.desiredScheduled ?? '-' }}</span>
            </template>
          </el-table-column>
        </template>

        <!-- Deployment 和 StatefulSet 通用列 -->
        <template v-if="selectedType === 'Deployment' || selectedType === 'StatefulSet'">
          <!-- 标签 -->
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

          <!-- 容器组 -->
          <el-table-column label="容器组" width="150" align="center">
            <template #default="{ row }">
              <div class="pod-count-cell">
                <span class="pod-count">{{ row.readyPods || 0 }}/{{ row.desiredPods || 0 }}</span>
                <span class="pod-label">Pods</span>
              </div>
            </template>
          </el-table-column>

          <!-- Requests/Limits -->
          <el-table-column label="Requests/Limits" min-width="200">
            <template #default="{ row }">
              <div class="resource-cell">
                <div v-if="row.requests?.cpu || row.limits?.cpu" class="resource-item">
                  <span class="resource-label">CPU:</span>
                  <span v-if="row.requests?.cpu" class="resource-value requests-value">{{ row.requests.cpu }}</span>
                  <span v-if="row.requests?.cpu && row.limits?.cpu" class="resource-separator">/</span>
                  <span v-if="row.limits?.cpu" class="resource-value limits-value">{{ row.limits.cpu }}</span>
                </div>
                <div v-if="row.requests?.memory || row.limits?.memory" class="resource-item">
                  <span class="resource-label">Mem:</span>
                  <span v-if="row.requests?.memory" class="resource-value requests-value">{{ row.requests.memory }}</span>
                  <span v-if="row.requests?.memory && row.limits?.memory" class="resource-separator">/</span>
                  <span v-if="row.limits?.memory" class="resource-value limits-value">{{ row.limits.memory }}</span>
                </div>
                <div v-if="!row.requests?.cpu && !row.requests?.memory && !row.limits?.cpu && !row.limits?.memory" class="resource-empty">-</div>
              </div>
            </template>
          </el-table-column>

          <!-- 镜像 -->
          <el-table-column label="镜像" min-width="300">
            <template #default="{ row }">
              <div class="image-cell">
                <el-tooltip
                  v-if="row.images && row.images.length > 0"
                  placement="top"
                  effect="light"
                >
                  <template #content>
                    <div class="image-tooltip-content">
                      <div v-for="(image, index) in row.images" :key="index" class="image-tooltip-item">
                        {{ image }}
                      </div>
                    </div>
                  </template>
                  <div class="image-list">
                    <span v-for="(image, index) in getDisplayImages(row.images)" :key="index" class="image-item">
                      {{ image }}
                    </span>
                    <span v-if="row.images.length > 2" class="image-more">
                      +{{ row.images.length - 2 }}
                    </span>
                  </div>
                </el-tooltip>
                <span v-else class="image-empty">-</span>
              </div>
            </template>
          </el-table-column>
        </template>

        <!-- Job 专用列 -->
        <template v-if="selectedType === 'Job'">
          <!-- 状态 -->
          <el-table-column label="状态" min-width="150" align="center">
            <template #default="{ row }">
              <div :class="['status-badge', `status-${row.status?.toLowerCase()}`]">
                {{ row.status || '-' }}
              </div>
            </template>
          </el-table-column>

          <!-- 耗时 -->
          <el-table-column label="耗时" width="150">
            <template #default="{ row }">
              <span>{{ row.duration || '-' }}</span>
            </template>
          </el-table-column>
        </template>

        <!-- CronJob 专用列 -->
        <template v-if="selectedType === 'CronJob'">
          <!-- 调度 -->
          <el-table-column label="调度" width="150">
            <template #default="{ row }">
              <span class="schedule-text">{{ row.schedule || '-' }}</span>
            </template>
          </el-table-column>

          <!-- 最后的调度时间 -->
          <el-table-column label="最后的调度时间" width="180">
            <template #default="{ row }">
              <span>{{ row.lastScheduleTime || '-' }}</span>
            </template>
          </el-table-column>

          <!-- 暂停 -->
          <el-table-column label="暂停" width="80" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.suspended" type="info" size="small">是</el-tag>
              <el-tag v-else type="success" size="small">否</el-tag>
            </template>
          </el-table-column>
        </template>

        <!-- 存活时间（除 Pod 外的所有类型） -->
        <el-table-column v-if="selectedType !== 'Pod'" label="存活时间" width="150">
          <template #default="{ row }">
            <div class="age-cell">
              <el-icon class="age-icon"><Clock /></el-icon>
              <span>{{ formatAge(row.createdAt) }}</span>
            </div>
          </template>
        </el-table-column>

        <!-- Pod 的存活时间列 -->
        <el-table-column v-if="selectedType === 'Pod'" label="存活时间" width="150">
          <template #default="{ row }">
            <div class="age-cell">
              <el-icon class="age-icon"><Clock /></el-icon>
              <span>{{ formatAge(row.createdAt) }}</span>
            </div>
          </template>
        </el-table-column>

        <!-- 操作列 -->
        <el-table-column label="操作" width="180" fixed="right" align="center">
          <template #default="{ row }">
            <!-- Pod 类型工作负载的特殊菜单 -->
            <template v-if="selectedType === 'Pod'">
              <el-popover
                placement="bottom"
                :width="220"
                trigger="click"
                @before-enter="fetchPodDetailsForMenu(row.name, row.namespace)"
              >
                <template #reference>
                  <el-button link class="action-btn">
                    <el-icon :size="18"><Edit /></el-icon>
                  </el-button>
                </template>
                <div v-loading="podMenuLoading" class="pod-action-menu">
                  <template v-if="podMenuData && podMenuData.spec?.containers">
                    <!-- 容器选项 -->
                    <div v-for="container in podMenuData.spec.containers" :key="container.name" class="container-actions">
                      <div class="container-name">{{ container.name }}</div>
                      <div class="container-menu-items">
                        <div class="menu-item" @click="handleOpenFileBrowser(row.name, container.name, row.namespace)">
                          <el-icon><FolderOpened /></el-icon>
                          <span>文件浏览</span>
                        </div>
                        <div class="menu-item" @click="handleOpenTerminal(row.name, container.name, row.namespace)">
                          <el-icon><Monitor /></el-icon>
                          <span>终端</span>
                        </div>
                        <div class="menu-item" @click="handleOpenLogs(row.name, container.name, row.namespace)">
                          <el-icon><Document /></el-icon>
                          <span>日志</span>
                        </div>
                      </div>
                    </div>
                    <!-- 分割线 -->
                    <el-divider style="margin: 8px 0" />
                    <!-- 删除 Pod -->
                    <div class="menu-item danger" @click="handleDeletePod(row.name, row.namespace)">
                      <el-icon><Delete /></el-icon>
                      <span>删除 Pod</span>
                    </div>
                  </template>
                  <div v-else-if="!podMenuLoading" class="menu-error">
                    加载失败
                  </div>
                </div>
              </el-popover>
            </template>
            <!-- 非Pod 类型工作负载的标准操作 -->
            <template v-else>
              <div class="action-buttons">
                <!-- YAML 按钮 -->
                <el-button link class="action-btn" @click="handleWorkloadYAML(row)" title="YAML">
                  <el-icon :size="16"><Document /></el-icon>
                </el-button>
                <!-- 编辑按钮 -->
                <el-button link class="action-btn" @click="handleWorkloadEdit(row)" title="编辑">
                  <el-icon :size="16"><Edit /></el-icon>
                </el-button>
                <!-- 删除按钮 -->
                <el-button link class="action-btn danger" @click="handleWorkloadDelete(row)" title="删除">
                  <el-icon :size="16"><Delete /></el-icon>
                </el-button>
              </div>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredWorkloadList.length"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 标签弹窗 -->
    <el-dialog
      v-model="labelDialogVisible"
      title="工作负载标签"
      width="700px"
      class="label-dialog"
    >
      <div class="label-dialog-content">
        <el-table :data="labelList" class="label-table" max-height="500">
          <el-table-column prop="key" label="Key" min-width="280">
            <template #default="{ row }">
              <div class="label-key-wrapper" @click="copyToClipboard(row.key, 'Key')">
                <span class="label-key-text">{{ row.key }}</span>
                <el-icon class="copy-icon"><CopyDocument /></el-icon>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="value" label="Value" min-width="350">
            <template #default="{ row }">
              <span class="label-value">{{ row.value }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="labelDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- YAML 编辑弹窗 -->
    <el-dialog
      v-model="yamlDialogVisible"
      :title="`工作负载 YAML - ${selectedWorkload?.name || ''}`"
      width="900px"
      class="yaml-dialog"
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
          ></textarea>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="yamlDialogVisible = false">取消</el-button>
          <el-button type="primary" class="black-button" @click="handleSaveYAML" :loading="yamlSaving">
            保存
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 工作负载详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="`${detailData?.type || ''} - ${detailData?.name || ''}`"
      width="1200px"
      class="detail-dialog"
    >
      <div v-if="detailData" class="detail-wrapper">
        <!-- 基本信息区域 -->
        <div class="detail-hero">
          <div class="detail-hero-main">
            <div class="detail-kind-chip">{{ detailData?.type || 'Workload' }}</div>
            <h2 class="detail-title">{{ detailData.workload?.metadata?.name || detailData.name }}</h2>
            <div class="detail-meta">
              <span>
                <el-icon><FolderOpened /></el-icon>
                {{ detailData.workload?.metadata?.namespace || detailData.namespace }}
              </span>
              <span>
                <el-icon><Clock /></el-icon>
                {{ formatAgeShort(detailData.workload?.metadata?.creationTimestamp) }}
              </span>
              <span>
                <el-icon><Box /></el-icon>
                {{ getContainerImageList(detailData.workload).length || 0 }} 个镜像
              </span>
            </div>
          </div>

          <div class="detail-summary-grid">
            <div class="detail-summary-card">
              <span class="summary-label">期望副本</span>
              <strong>{{ detailData.workload?.spec?.replicas ?? detailData.pods?.length ?? 0 }}</strong>
            </div>
            <div class="detail-summary-card">
              <span class="summary-label">可用副本</span>
              <strong>{{ detailData.workload?.status?.availableReplicas ?? detailData.workload?.status?.readyReplicas ?? 0 }}</strong>
            </div>
            <div class="detail-summary-card">
              <span class="summary-label">关联 Pod</span>
              <strong>{{ detailData.pods?.length || 0 }}</strong>
            </div>
          </div>
        </div>

        <div class="detail-meta-section">
          <div v-if="getContainerImageList(detailData.workload).length > 0" class="detail-meta-card images-card">
            <div class="meta-card-title">镜像</div>
            <div class="images-list">
              <div v-for="(image, idx) in getContainerImageList(detailData.workload)" :key="idx" class="image-tag">
                {{ image }}
              </div>
            </div>
          </div>

          <div v-if="detailData.workload?.metadata?.labels && Object.keys(detailData.workload.metadata.labels).length > 0" class="detail-meta-card">
            <div class="meta-card-title">标签</div>
            <div class="labels-list">
              <el-tag
                v-for="(value, key) in detailData.workload.metadata.labels"
                :key="key"
                size="small"
                class="label-tag"
                type="info"
              >
                {{ key }}: {{ value }}
              </el-tag>
            </div>
          </div>

          <div v-if="detailData.workload?.metadata?.annotations && Object.keys(detailData.workload.metadata.annotations).length > 0" class="detail-meta-card">
            <div class="meta-card-title">注解</div>
            <el-tooltip :content="getAnnotationsTooltip(detailData.workload.metadata.annotations)" placement="top" effect="light" :show-after="500">
              <span class="annotations-text">{{ getAnnotationsText(detailData.workload.metadata.annotations) }}</span>
            </el-tooltip>
          </div>
        </div>

        <!-- 标签页区域 -->
        <el-tabs v-model="activeDetailTab" type="border-card" class="detail-tabs">
          <el-tab-pane label="容器组" name="pods">
            <div class="tab-content">
              <el-table :data="detailData.pods" size="default" class="pods-table">
                <el-table-column prop="metadata.name" label="名称" min-width="220" show-overflow-tooltip>
                  <template #default="{ row }">
                    <div class="pod-name-cell" @click="showPodDetail(row)" style="cursor: pointer;">
                      <el-icon class="pod-icon"><Box /></el-icon>
                      <span class="pod-name">{{ row.metadata?.name }}</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="状态" width="90" align="center">
                  <template #default="{ row }">
                    <el-tag :type="getPodStatusType(row.status?.phase)" size="small" effect="plain">
                      {{ getPodStatusText(row.status?.phase) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="CPU" width="120" align="center">
                  <template #default="{ row }">
                    <span class="resource-value">{{ getPodCPU(row) }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="内存" width="120" align="center">
                  <template #default="{ row }">
                    <span class="resource-value">{{ getPodMemory(row) }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="重启" width="80" align="center">
                  <template #default="{ row }">
                    <span :class="{'restart-high': getRestartCount(row) > 5}">{{ getRestartCount(row) }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="spec.nodeName" label="节点" min-width="140" show-overflow-tooltip />
                <el-table-column label="操作" width="70" align="center" fixed="right">
                  <template #default="{ row }">
                    <el-dropdown trigger="click" @command="(cmd) => handlePodAction(cmd, row)">
                      <el-button type="primary" size="small" circle :icon="MoreFilled" />
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item v-for="container in row.spec?.containers || []" :key="container.name" disabled>
                            <div class="container-group-header">{{ container.name }}</div>
                          </el-dropdown-item>
                          <el-dropdown-item v-for="container in row.spec?.containers || []" :key="`terminal-${container.name}`" :command="{ action: 'terminal', container: container.name, pod: row.metadata?.name }">
                            <el-icon><Monitor /></el-icon>
                            <span>{{ container.name }} 终端</span>
                          </el-dropdown-item>
                          <el-dropdown-item v-for="container in row.spec?.containers || []" :key="`logs-${container.name}`" :command="{ action: 'logs', container: container.name, pod: row.metadata?.name }">
                            <el-icon><Document /></el-icon>
                            <span>{{ container.name }} 日志</span>
                          </el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </el-tab-pane>

          <el-tab-pane label="服务" name="services">
            <div class="tab-content">
              <el-table :data="detailData.services" class="detail-table services-table" v-if="detailData.services && detailData.services.length > 0">
                <el-table-column label="名称" min-width="220" show-overflow-tooltip>
                  <template #default="{ row }">
                    <div class="service-name-wrapper">
                      <el-icon class="service-icon"><Connection /></el-icon>
                      <span class="service-name-text">{{ row.metadata?.name || '-' }}</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="类型" width="110" align="center">
                  <template #default="{ row }">
                    <el-tag :type="getServiceTypeColor(row.spec?.type)" size="small" effect="plain">
                      {{ row.spec?.type || '-' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="集群IP" width="130" align="center">
                  <template #default="{ row }">
                    <div class="ip-cell">
                      <span v-if="row.spec?.clusterIP" class="ip-text">{{ row.spec.clusterIP }}</span>
                      <span v-else class="empty-text">None</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="外部IP" width="130" align="center">
                  <template #default="{ row }">
                    <div class="ip-cell">
                      <span v-if="row.spec?.externalIPs && row.spec.externalIPs.length > 0" class="ip-text external-ip">
                        {{ row.spec.externalIPs[0] }}
                        <el-tooltip v-if="row.spec.externalIPs.length > 1" :content="row.spec.externalIPs.join(', ')" placement="top">
                          <span class="more-badge">+{{ row.spec.externalIPs.length - 1 }}</span>
                        </el-tooltip>
                      </span>
                      <span v-else-if="row.status?.loadBalancer?.ingress && row.status.loadBalancer.ingress.length > 0" class="ip-text external-ip">
                        {{ row.status.loadBalancer.ingress[0].ip || row.status.loadBalancer.ingress[0].hostname }}
                      </span>
                      <span v-else class="empty-text">-</span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="端口" min-width="320">
                  <template #default="{ row }">
                    <div v-if="row.spec?.ports?.length > 0" class="ports-combined">
                      <div v-for="(port, idx) in row.spec.ports" :key="idx" class="port-row">
                        <div class="port-info">
                          <el-tag size="small" :type="port.protocol === 'TCP' ? '' : 'warning'" effect="plain">
                            {{ port.protocol || 'TCP' }}
                          </el-tag>
                          <span class="port-number">{{ port.port }}</span>
                          <el-icon class="port-arrow"><Right /></el-icon>
                          <span class="target-port">{{ port.targetPort || port.port }}</span>
                          <span v-if="row.spec?.type === 'NodePort' && port.nodePort" class="nodeport-badge">
                            NodePort: {{ port.nodePort }}
                          </span>
                        </div>
                        <div v-if="port.name" class="port-name">{{ port.name }}</div>
                      </div>
                    </div>
                    <span v-else class="empty-text">-</span>
                  </template>
                </el-table-column>
                <el-table-column label="存活时间" width="100" align="center">
                  <template #default="{ row }">
                    <span class="age-text">{{ calculateAge(row.metadata?.creationTimestamp) }}</span>
                  </template>
                </el-table-column>
              </el-table>
              <el-empty v-else description="暂无服务" :image-size="120" />
            </div>
          </el-tab-pane>

          <el-tab-pane label="路由" name="ingresses">
            <div class="tab-content">
              <div v-if="detailData.ingresses && detailData.ingresses.length > 0" class="ingress-content">
                <!-- 域名列表 -->
                <div class="ingress-hosts-section">
                  <div class="section-title">
                    <el-icon><Link /></el-icon>
                    <span>域名列表</span>
                  </div>
                  <div class="hosts-list">
                    <div v-for="ingress in ingressHosts" :key="ingress.host" class="host-item">
                      <div class="host-content">
                        <el-icon class="host-icon"><Platform /></el-icon>
                        <el-tooltip :content="ingress.host" placement="top">
                          <span class="host-text">{{ ingress.host }}</span>
                        </el-tooltip>
                      </div>
                      <div class="host-ingress-names">
                        <span v-for="name in ingress.names" :key="name" class="ingress-name-tag">{{ name }}</span>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 路由规则表格 -->
                <div class="ingress-rules-section">
                  <div class="section-title">
                    <el-icon><Guide /></el-icon>
                    <span>路由规则</span>
                  </div>
                  <el-table :data="ingressRules" class="ingress-rules-table">
                    <el-table-column label="名称" min-width="180">
                      <template #default="{ row }">
                        <div class="rule-name-cell">
                          <el-icon class="rule-icon"><Document /></el-icon>
                          <span class="rule-name-text">{{ row.ingressName }}</span>
                        </div>
                      </template>
                    </el-table-column>
                    <el-table-column label="域名" min-width="200" show-overflow-tooltip>
                      <template #default="{ row }">
                        <span class="host-text-cell">{{ row.host || '-' }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column label="路径" min-width="180">
                      <template #default="{ row }">
                        <el-tooltip :content="`${row.pathType || 'Prefix'}: ${row.path || '/'}`" placement="top">
                          <span class="path-text-simple">{{ row.path || '/' }}</span>
                        </el-tooltip>
                      </template>
                    </el-table-column>
                    <el-table-column label="服务" min-width="150">
                      <template #default="{ row }">
                        <span class="service-name-cell">{{ row.serviceName || '-' }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column label="端口" width="100" align="center">
                      <template #default="{ row }">
                        <span v-if="row.servicePort" class="port-number-cell">{{ row.servicePort }}</span>
                        <span v-else class="empty-text">-</span>
                      </template>
                    </el-table-column>
                  </el-table>
                </div>
              </div>
              <el-empty v-else description="暂无路由" :image-size="120" />
            </div>
          </el-tab-pane>

          <el-tab-pane label="运行时信息" name="runtime">
            <div class="tab-content">
              <div v-if="detailData.workload" class="runtime-content">
                <el-table :data="getRuntimeInfo()" class="runtime-table" border>
                  <el-table-column label="类别" width="150">
                    <template #default="{ row }">
                      <div class="runtime-category">
                        <el-icon :class="`category-icon ${row.iconClass}`"><component :is="row.icon" /></el-icon>
                        <span class="category-text">{{ row.category }}</span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="状态" min-width="150" align="center">
                    <template #default="{ row }">
                      <div class="status-cell">
                        <el-icon :class="`status-indicator status-${row.statusType} ${row.isLoading ? 'is-loading' : ''}`">
                          <component :is="row.statusIcon" />
                        </el-icon>
                        <span :class="`status-text status-${row.statusType}`">{{ row.status }}</span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="消息" min-width="350">
                    <template #default="{ row }">
                      <div class="message-cell">
                        <span class="message-text">{{ row.message }}</span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="最后更新时间" width="160" align="center">
                    <template #default="{ row }">
                      <span class="time-text">{{ row.lastUpdate }}</span>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
              <el-empty v-else description="暂无运行时信息" :image-size="120" />
            </div>
          </el-tab-pane>

          <el-tab-pane label="暂停" name="paused">
            <div class="tab-content">
              <div class="paused-content">
                <div class="paused-header">
                  <div class="paused-icon-wrapper">
                    <el-icon class="paused-icon" :class="{ 'is-paused': isWorkloadPaused }">
                      <VideoPause v-if="isWorkloadPaused" />
                      <VideoPlay v-else />
                    </el-icon>
                  </div>
                  <div class="paused-title">
                    <h3>工作负载暂停状态</h3>
                    <p class="paused-status-text" :class="{ 'paused': isWorkloadPaused }">
                      {{ isWorkloadPaused ? '当前已暂停' : '当前运行中' }}
                    </p>
                  </div>
                </div>

                <div class="paused-control">
                  <div class="paused-switch-wrapper">
                    <span class="switch-label">暂停状态</span>
                    <el-switch
                      v-model="isWorkloadPaused"
                      size="large"
                      :loading="pauseLoading"
                      active-text="已暂停"
                      inactive-text="运行中"
                      @change="handlePauseChange"
                      style="--el-switch-on-color: #f56c6c; --el-switch-off-color: #67c23a;"
                    />
                  </div>
                  <div class="paused-description">
                    <el-alert
                      :title="isWorkloadPaused ? '暂停状态下，新的 Pod 副本不会被创建，但现有的 Pod 不会被删除。' : '正常运行状态下，控制器会根据指定的副本数创建和管理 Pod。'"
                      :type="isWorkloadPaused ? 'warning' : 'success'"
                      :closable="false"
                      show-icon
                    />
                  </div>
                </div>

                <div class="paused-info">
                  <el-descriptions :column="2" border>
                    <el-descriptions-item label="工作负载类型">
                      {{ workloadType }}
                    </el-descriptions-item>
                    <el-descriptions-item label="命名空间">
                      {{ detailData.workload?.metadata?.namespace || '-' }}
                    </el-descriptions-item>
                    <el-descriptions-item label="当前副本数">
                      {{ detailData.workload?.spec?.replicas || 0 }}
                    </el-descriptions-item>
                    <el-descriptions-item label="可用副本数">
                      {{ detailData.workload?.status?.availableReplicas || 0 }}
                    </el-descriptions-item>
                  </el-descriptions>
                </div>
              </div>
            </div>
          </el-tab-pane>

          <el-tab-pane label="历史版本" name="revisions">
            <div class="tab-content">
              <div v-if="sortedReplicaSets.length > 0" class="revisions-content">
                <el-table :data="sortedReplicaSets" class="revisions-table" stripe>
                  <el-table-column label="版本" width="140" align="center">
                    <template #default="{ row }">
                      <div class="revision-cell">
                        <div class="revision-number-wrapper">
                          <span class="revision-icon">#</span>
                          <span class="revision-number">{{ getReplicaSetRevision(row) }}</span>
                        </div>
                        <el-tag v-if="isCurrentReplicaSet(row)" size="small" type="success" class="current-tag">
                          <el-icon><CircleCheck /></el-icon>
                          当前
                        </el-tag>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="镜像" min-width="350">
                    <template #default="{ row }">
                      <div class="images-column-enhanced">
                        <div v-for="(image, idx) in getReplicaSetImages(row)" :key="idx" class="image-card">
                          <div class="image-icon">
                            <el-icon><Box /></el-icon>
                          </div>
                          <div class="image-info">
                            <div class="image-name">{{ image }}</div>
                          </div>
                        </div>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="副本信息" width="160" align="center">
                    <template #default="{ row }">
                      <div class="replicas-info">
                        <div class="replica-item">
                          <span class="replica-label">期望</span>
                          <span class="replica-value">{{ row.spec?.replicas || 0 }}</span>
                        </div>
                        <div class="replica-divider"></div>
                        <div class="replica-item">
                          <span class="replica-label">就绪</span>
                          <span class="replica-value ready">{{ row.status?.availableReplicas || 0 }}</span>
                        </div>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="创建时间" width="180">
                    <template #default="{ row }">
                      <div class="time-cell">
                        <el-icon class="time-icon"><Clock /></el-icon>
                        <span class="time-text">{{ formatAgeShort(row.metadata?.creationTimestamp) }}</span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="状态" width="120" align="center">
                    <template #default="{ row }">
                      <div class="status-cell-enhanced">
                        <el-icon :class="`status-dot status-${getReplicaSetStatusType(row)}`">
                          <component :is="getStatusDotIcon(getReplicaSetStatusType(row))" />
                        </el-icon>
                        <span :class="`status-text-enhanced status-${getReplicaSetStatusType(row)}`">
                          {{ getReplicaSetStatusText(row) }}
                        </span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="200" align="center" fixed="right">
                    <template #default="{ row }">
                      <div class="action-buttons">
                        <el-button
                          type="primary"
                          size="small"
                          plain
                          @click="handleViewReplicaSetYAML(row)"
                          class="action-btn view-btn"
                        >
                          <el-icon><Document /></el-icon>
                          <span>详情</span>
                        </el-button>
                        <el-button
                          v-if="!isCurrentReplicaSet(row)"
                          type="warning"
                          size="small"
                          plain
                          @click="handleRollback(row)"
                          class="action-btn rollback-btn"
                        >
                          <el-icon><RefreshLeft /></el-icon>
                          <span>回滚</span>
                        </el-button>
                      </div>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
              <el-empty v-else description="暂无历史版本" :image-size="120" />
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="detailDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 工作负载编辑对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      :title="isCreateMode ? '创建工作负载' : '编辑工作负载'"
      width="90%"
      :close-on-click-modal="false"
      class="workload-edit-dialog"
      @close="isCreateMode = false"
    >
      <div class="workload-edit-content" v-if="editWorkloadData">
        <!-- 左侧：基础信息 -->
        <div class="edit-sidebar">
          <BasicInfo
            :formData="editWorkloadData"
            :isCreateMode="isCreateMode"
            :namespaceList="namespaceList"
            @add-label="handleAddLabel"
            @remove-label="handleRemoveLabel"
            @add-annotation="handleAddAnnotation"
            @remove-annotation="handleRemoveAnnotation"
          />
        </div>

        <!-- 右侧：详细配置 -->
        <div class="edit-main">
          <el-tabs v-model="activeEditTab" type="border-card">
            <el-tab-pane label="容器配置" name="containers">
              <div class="tab-content">
                <ContainerConfig
                  :containers="editWorkloadData.containers || []"
                  :initContainers="editWorkloadData.initContainers || []"
                  :volumes="editWorkloadData.volumes || []"
                  @updateContainers="updateContainers"
                  @updateInitContainers="updateInitContainers"
                />
              </div>
            </el-tab-pane>
            <el-tab-pane label="存储" name="storage">
              <div class="tab-content">
                <VolumeConfig
                  :volumes="editWorkloadData.volumes || []"
                  :configMaps="configMaps"
                  :secrets="secrets"
                  :pvcs="pvcs"
                  @addVolume="handleAddVolume"
                  @removeVolume="handleRemoveVolume"
                  @update="handleUpdateVolumes"
                  @refreshConfigMaps="loadConfigMaps"
                  @refreshSecrets="loadSecrets"
                  @refreshPVCs="loadPVCs"
                />
              </div>
            </el-tab-pane>
            <el-tab-pane label="扩容配置" name="scaling">
              <div class="tab-content scaling-tab-content">
                <ScalingConfig
                  :workloadType="editWorkloadData.type"
                  :formData="editWorkloadData"
                  :scalingStrategy="scalingStrategyData"
                  :jobConfig="jobConfig"
                  :cronJobConfig="cronJobConfig"
                  @update:formData="handleUpdateFormData"
                  @update:scalingStrategy="handleUpdateScalingStrategy"
                  @update:jobConfig="updateJobConfig"
                  @update:cronJobConfig="updateCronJobConfig"
                />
              </div>
            </el-tab-pane>
            <el-tab-pane label="节点调度" name="scheduling">
              <div class="tab-content scheduling-tab-content">
                <!-- 调度类型 -->
                <div class="info-panel">
                  <div class="panel-header">
                    <span class="panel-icon">🎯</span>
                    <span class="panel-title">调度类型</span>
                  </div>
                  <div class="panel-content">
                    <NodeSelector
                      :formData="editWorkloadData"
                      :nodeList="nodeList"
                      :commonNodeLabels="[]"
                      @addMatchRule="handleAddMatchRule"
                      @removeMatchRule="handleRemoveMatchRule"
                      @update="handleUpdateScheduling"
                    />
                  </div>
                </div>

                <!-- 亲和性配置 -->
                <div class="info-panel">
                  <div class="panel-header">
                    <span class="panel-icon">🔗</span>
                    <span class="panel-title">亲和性配置</span>
                  </div>
                  <div class="panel-content">
                    <Affinity
                      :affinityRules="affinityRules"
                      :editingAffinityRule="editingAffinityRule"
                      :namespaceList="namespaceList"
                      @startAddAffinity="handleStartAddAffinity"
                      @cancelAffinityEdit="handleCancelAffinityEdit"
                      @saveAffinityRule="handleSaveAffinityRule"
                      @addMatchExpression="handleAddMatchExpression"
                      @removeMatchExpression="handleRemoveMatchExpression"
                      @addMatchLabel="handleAddMatchLabel"
                      @removeMatchLabel="handleRemoveMatchLabel"
                      @removeAffinityRule="handleRemoveAffinityRule"
                    />
                  </div>
                </div>

                <!-- 容忍度配置 -->
                <div class="info-panel">
                  <div class="panel-header">
                    <span class="panel-icon">✅</span>
                    <span class="panel-title">容忍度配置</span>
                  </div>
                  <div class="panel-content">
                    <Tolerations
                      :tolerations="editWorkloadData.tolerations || []"
                      @addToleration="handleAddToleration"
                      @removeToleration="handleRemoveToleration"
                    />
                  </div>
                </div>
              </div>
            </el-tab-pane>
            <el-tab-pane label="网络" name="network">
              <div class="tab-content">
                <Network
                  :formData="editWorkloadData"
                  @addDNSNameserver="handleAddDNSNameserver"
                  @removeDNSNameserver="handleRemoveDNSNameserver"
                  @addDNSSearch="handleAddDNSSearch"
                  @removeDNSSearch="handleRemoveDNSSearch"
                  @addDNSOption="handleAddDNSOption"
                  @removeDNSOption="handleRemoveDNSOption"
                />
              </div>
            </el-tab-pane>
            <el-tab-pane label="其他" name="others">
              <div class="tab-content">
                <Others :formData="editWorkloadData" />
              </div>
            </el-tab-pane>
          </el-tabs>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="editDialogVisible = false">取消</el-button>
          <el-button type="primary" class="black-button" @click="handleSaveEdit" :loading="editSaving">
            保存
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 终端对话框 -->
    <el-dialog
      v-model="terminalDialogVisible"
      :title="`终端 - Pod: ${terminalData.pod} | 容器: ${terminalData.container}`"
      width="90%"
      :close-on-click-modal="false"
      class="terminal-dialog"
      @close="handleCloseTerminal"
      @opened="handleDialogOpened"
    >
      <div class="terminal-container">
        <div v-if="!terminalConnected" class="terminal-loading-overlay">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>正在连接终端...</span>
        </div>
        <div class="terminal-wrapper" ref="terminalWrapper"></div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="terminalDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 日志对话框 -->
    <el-dialog
      v-model="logsDialogVisible"
      :title="`日志 - Pod: ${logsData.pod} | 容器: ${logsData.container}`"
      width="90%"
      :close-on-click-modal="false"
      class="logs-dialog"
      @opened="handleLogsDialogOpened"
    >
      <div class="logs-toolbar">
        <el-button size="small" @click="handleRefreshLogs" :loading="logsLoading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button size="small" @click="handleDownloadLogs">
          <el-icon><Download /></el-icon>
          下载
        </el-button>
        <el-button size="small" @click="logsAutoScroll = !logsAutoScroll" :type="logsAutoScroll ? 'primary' : 'default'">
          <el-icon><Bottom /></el-icon>
          {{ logsAutoScroll ? '自动滚动' : '停止滚动' }}
        </el-button>
        <el-select v-model="logsTailLines" size="small" style="width: 120px; margin-left: 10px;">
          <el-option label="最近100行" :value="100" />
          <el-option label="最近500行" :value="500" />
          <el-option label="最近1000行" :value="1000" />
          <el-option label="全部" :value="0" />
        </el-select>
      </div>
      <div class="logs-wrapper" ref="logsWrapper">
        <pre v-if="logsContent" class="logs-content">{{ logsContent }}</pre>
        <el-empty v-else-if="!logsLoading" description="暂无日志" />
        <div v-if="logsLoading" class="logs-loading">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>正在加载日志...</span>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="logsDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- ReplicaSet YAML 对话框 -->
    <el-dialog
      v-model="replicaSetYamlDialogVisible"
      :title="`ReplicaSet YAML - ${replicaSetYamlData.name}`"
      width="900px"
      :close-on-click-modal="false"
      class="yaml-dialog"
    >
      <div class="yaml-dialog-content">
        <div class="yaml-editor-wrapper">
          <div class="yaml-line-numbers">
            <div v-for="line in replicaSetYamlLineCount" :key="line" class="line-number">{{ line }}</div>
          </div>
          <textarea
            v-model="replicaSetYamlContent"
            class="yaml-textarea"
            placeholder="YAML 内容"
            spellcheck="false"
            readonly
          ></textarea>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="handleCopyReplicaSetYAML">
            <el-icon><CopyDocument /></el-icon>
            复制
          </el-button>
          <el-button type="primary" @click="replicaSetYamlDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 创建工作负载弹窗 -->
    <el-dialog
      v-model="createWorkloadDialogVisible"
      :title="`YAML创建${selectedType || ''}`"
      width="800px"
      :close-on-click-modal="false"
      class="create-workload-dialog"
    >
      <div class="yaml-create-mode">
        <div class="yaml-editor-container">
          <div class="yaml-editor-wrapper">
            <div class="yaml-line-numbers">
              <div v-for="line in createYamlLineCount" :key="line" class="line-number">{{ line }}</div>
            </div>
            <textarea
              v-model="createYamlContent"
              class="yaml-textarea"
              placeholder="请输入或修改 YAML 内容..."
              spellcheck="false"
            ></textarea>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createWorkloadDialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :loading="createYamlLoading"
            @click="handleCreateFromYaml"
          >
            创建
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Pod 详情对话框 -->
    <PodDetail
      v-model:visible="podDetailVisible"
      :cluster-id="selectedClusterId"
      :namespace="selectedPodNamespace"
      :pod-name="selectedPodName"
    />

    <!-- File Browser 对话框 -->
    <FileBrowser
      v-model:visible="fileBrowserVisible"
      :cluster-id="selectedClusterId"
      :namespace="selectedFileBrowserNamespace"
      :pod-name="selectedFileBrowserPod"
      :container-name="selectedFileBrowserContainer"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, nextTick, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox, ElLoading } from 'element-plus'
import axios from 'axios'
import * as yaml from 'js-yaml'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import {
  Search,
  Tools,
  Grid,
  Platform,
  FolderOpened,
  PriceTag,
  Clock,
  Refresh,
  Edit,
  View,
  Document,
  Monitor,
  RefreshRight,
  RefreshLeft,
  Rank,
  Delete,
  CopyDocument,
  Box,
  ArrowDown,
  MoreFilled,
  Loading,
  Download,
  Bottom,
  Connection,
  Right,
  Link,
  Guide,
  CircleCheck,
  CircleClose,
  Warning,
  VideoPause,
  VideoPlay,
  Plus
} from '@element-plus/icons-vue'
import { getClusterList, updateWorkload, getConfigMaps, getSecrets, getPersistentVolumeClaims, type Cluster } from '@/api/kubernetes'
// 导入工作负载编辑组件
import BasicInfo from './workload-components/BasicInfo.vue'
import ContainerConfig from './workload-components/ContainerConfig.vue'
import ScalingConfig from './workload-components/ScalingConfig.vue'
import NodeSelector from './workload-components/spec/NodeSelector.vue'
import Affinity from './workload-components/spec/Affinity.vue'
import Tolerations from './workload-components/spec/Tolerations.vue'
import Network from './workload-components/spec/Network.vue'
import Others from './workload-components/spec/Others.vue'
import VolumeConfig from './workload-components/VolumeConfig.vue'
import PodDetail from './PodDetail.vue'
import FileBrowser from './FileBrowser.vue'

// 工作负载接口定义
interface Workload {
  name: string
  namespace: string
  type: string
  labels?: Record<string, string>
  readyPods?: number
  desiredPods?: number
  requests?: { cpu: string; memory: string }
  limits?: { cpu: string; memory: string }
  images?: string[]
  createdAt?: string
  updatedAt?: string
  // DaemonSet 专用字段
  currentScheduled?: number
  desiredScheduled?: number
  // Job 专用字段
  status?: string
  duration?: string
  completionTime?: string
  // CronJob 专用字段
  schedule?: string
  lastScheduleTime?: string
  suspended?: boolean
  // Pod 专用字段
  containers?: string
  cpu?: string
  memory?: string
  podStatus?: string
  restartCount?: number
  podIP?: string
  node?: string
}

interface Namespace {
  name: string
}

const loading = ref(false)
const clusterList = ref<Cluster[]>([])
const namespaceList = ref<Namespace[]>([])
const selectedClusterId = ref<number>()
const selectedNamespace = ref<string>('')

// 计算属性：当前选中的集群对象
const selectedCluster = computed(() => {
  return clusterList.value.find(c => c.id === selectedClusterId.value)
})
const selectedType = ref<string>('Deployment') // 默认选择 Deployment
const workloadList = ref<Workload[]>([])

// 工作负载类型配置
const workloadTypes = ref([
  { label: 'Deployment', value: 'Deployment', icon: 'Box', count: 0 },
  { label: 'StatefulSet', value: 'StatefulSet', icon: 'Rank', count: 0 },
  { label: 'DaemonSet', value: 'DaemonSet', icon: 'Connection', count: 0 },
  { label: 'Job', value: 'Job', icon: 'Guide', count: 0 },
  { label: 'CronJob', value: 'CronJob', icon: 'Clock', count: 0 },
  { label: 'Pod', value: 'Pod', icon: 'Box', count: 0 }
])

// 搜索条件
const searchName = ref('')

// 分页状态
const currentPage = ref(1)
const pageSize = ref(10)

// 标签弹窗
const labelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])

// YAML 编辑弹窗
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const yamlSaving = ref(false)
const selectedWorkload = ref<Workload | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 工作负载详情弹窗
const detailDialogVisible = ref(false)
const detailData = ref<any>(null)
const activeDetailTab = ref('pods')

// Pod 详情弹窗
const podDetailVisible = ref(false)
const selectedPodName = ref('')
const selectedPodNamespace = ref('')

// File Browser 对话框
const fileBrowserVisible = ref(false)
const selectedFileBrowserPod = ref('')
const selectedFileBrowserNamespace = ref('')
const selectedFileBrowserContainer = ref('')

// Pod 操作菜单数据
const podMenuData = ref<any>(null)
const podMenuLoading = ref(false)

// 工作负载编辑弹窗
const editDialogVisible = ref(false)
const editSaving = ref(false)
const editWorkloadData = ref<any>(null)
const activeEditTab = ref('containers')
const isCreateMode = ref(false) // 区分创建模式还是编辑模式

// 存储资源配置
const configMaps = ref<{ name: string }[]>([])
const secrets = ref<{ name: string }[]>([])
const pvcs = ref<{ name: string }[]>([])

// 终端弹窗
const terminalDialogVisible = ref(false)
const terminalConnected = ref(false)
const terminalData = ref({
  pod: '',
  container: '',
  namespace: ''
})
const terminalWrapper = ref<HTMLDivElement | null>(null)
let terminalWebSocket: WebSocket | null = null
let terminal: any = null

// 日志弹窗
const logsDialogVisible = ref(false)
const logsContent = ref('')
const logsLoading = ref(false)
const logsData = ref({
  pod: '',
  container: '',
  namespace: ''
})
const logsWrapper = ref<HTMLDivElement | null>(null)
const logsAutoScroll = ref(true)
const logsTailLines = ref(500)
let logsRefreshTimer: number | null = null

// 暂停状态
const isWorkloadPaused = ref(false)
const pauseLoading = ref(false)

// ReplicaSet YAML 弹窗
const replicaSetYamlDialogVisible = ref(false)
const replicaSetYamlContent = ref('')
const replicaSetYamlData = ref({
  name: '',
  namespace: ''
})

// 创建工作负载弹窗
const createWorkloadDialogVisible = ref(false)
const selectedWorkloadType = ref('Deployment')
const createYamlContent = ref('')
const createYamlLoading = ref(false)

// 批量操作
const selectedWorkloads = ref<Workload[]>([])
const batchActionLoading = ref(false)

// 工作负载类型模板
const workloadTemplates: Record<string, string> = {
  Deployment: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.21.0
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi`,

  StatefulSet: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: nginx-statefulset
  namespace: default
spec:
  serviceName: nginx-headless
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.21.0
        ports:
        - containerPort: 80
        volumeMounts:
        - name: www
          mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
  - metadata:
      name: www
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi`,

  DaemonSet: `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd-daemonset
  namespace: default
spec:
  selector:
    matchLabels:
      app: fluentd
  template:
    metadata:
      labels:
        app: fluentd
    spec:
      containers:
      - name: fluentd
        image: fluentd:v1.14.0
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi`,

  Job: `apiVersion: batch/v1
kind: Job
metadata:
  name: pi-job
  namespace: default
spec:
  template:
    spec:
      containers:
      - name: pi
        image: perl:5.34.0
        command: ["perl", "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never
  backoffLimit: 4`,

  CronJob: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: hello-cronjob
  namespace: default
spec:
  schedule: "*/1 * * * *"
  concurrencyPolicy: Allow
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: hello
            image: busybox:1.36
            imagePullPolicy: IfNotPresent
            command:
            - /bin/sh
            - -c
            - date; echo Hello from the Kubernetes cluster
          restartPolicy: OnFailure`,

  Pod: `apiVersion: v1
kind: Pod
metadata:
  name: debug
  namespace: default
spec:
  containers:
    - name: debug
      image: nicolaka/netshoot
      command:
        - /bin/sh
      args:
        - '-c'
        - sleep 100000
      resources:
        limits:
          cpu: 100m
          memory: 128Mi
        requests:
          cpu: 50m
          memory: 64Mi`
}

// 亲和性规则
const affinityRules = ref<any[]>([])
const editingAffinityRule = ref<any>(null)

// 节点列表
const nodeList = ref<{ name: string }[]>([])

// 扩缩容策略
const scalingStrategyData = ref<any>({
  strategyType: 'RollingUpdate',
  maxSurge: '25%',
  maxUnavailable: '25%',
  minReadySeconds: 0,
  progressDeadlineSeconds: 600,
})

// CronJob 配置
const cronJobConfig = ref<any>({
  schedule: '0 * * * *',
  concurrencyPolicy: 'Allow',
  timeZone: '',
  successfulJobsHistoryLimit: 3,
  failedJobsHistoryLimit: 1,
  startingDeadlineSeconds: null,
  suspend: false,
})

// Job 配置
const jobConfig = ref<any>({
  completions: 1,
  parallelism: 1,
  backoffLimit: 6,
  activeDeadlineSeconds: null,
})

// 过滤后的工作负载列表
const filteredWorkloadList = computed(() => {
  let result = workloadList.value

  if (searchName.value) {
    result = result.filter(workload =>
      workload.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  return result
})

// 分页后的工作负载列表
const paginatedWorkloadList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredWorkloadList.value.slice(start, end)
})

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 计算ReplicaSet YAML行数
const replicaSetYamlLineCount = computed(() => {
  if (!replicaSetYamlContent.value) return 1
  return replicaSetYamlContent.value.split('\n').length
})

// 获取类型图标
const getTypeIcon = (type: string) => {
  return Tools
}

// 格式化资源显示
const formatResource = (resource: { cpu: string; memory: string }) => {
  const parts: string[] = []
  if (resource.cpu) parts.push(`cpu: ${resource.cpu}`)
  if (resource.memory) parts.push(`mem: ${resource.memory}`)
  return parts.join(' | ')
}

// 格式化存活时间
const formatAge = (createdAt: string | undefined): string => {
  if (!createdAt) return '-'

  const created = new Date(createdAt)
  const now = new Date()
  const diffMs = now.getTime() - created.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays < 1) {
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
    if (diffHours < 1) {
      const diffMinutes = Math.floor(diffMs / (1000 * 60))
      return diffMinutes < 1 ? '刚刚' : `${diffMinutes}分钟前`
    }
    return `${diffHours}小时前`
  }

  if (diffDays < 7) {
    return `${diffDays}天前`
  }

  const diffWeeks = Math.floor(diffDays / 7)
  if (diffWeeks < 4) {
    return `${diffWeeks}周前`
  }

  const diffMonths = Math.floor(diffDays / 30)
  if (diffMonths < 12) {
    return `${diffMonths}个月前`
  }

  const diffYears = Math.floor(diffDays / 365)
  return `${diffYears}年前`
}

// 获取显示的镜像（最多显示2个）
const getDisplayImages = (images?: string[]) => {
  if (!images || images.length === 0) return []
  return images.slice(0, 2).map(img => {
    // 只保留镜像名和tag，去掉registry部分
    const parts = img.split('/')
    const nameAndTag = parts[parts.length - 1]
    // 如果tag太长，截断显示
    if (nameAndTag.length > 50) {
      return nameAndTag.substring(0, 50) + '...'
    }
    return nameAndTag
  })
}

// 显示标签弹窗
const showLabels = (row: Workload) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelDialogVisible.value = true
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

// 处理页码变化
const handlePageChange = (page: number) => {
  currentPage.value = page
}

// 处理每页数量变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  const maxPage = Math.ceil(filteredWorkloadList.value.length / size)
  if (currentPage.value > maxPage) {
    currentPage.value = maxPage || 1
  }
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('workloads_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      await loadNamespaces()
      await loadWorkloads()
    }
  } catch (error) {
    ElMessage.error('获取集群列表失败')
  }
}

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!selectedClusterId.value) return

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/namespaces`,
      {
        params: { clusterId: selectedClusterId.value },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    namespaceList.value = response.data.data || []
  } catch (error) {
    namespaceList.value = []
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('workloads_selected_cluster_id', selectedClusterId.value.toString())
  }
  selectedNamespace.value = ''
  currentPage.value = 1
  await loadNamespaces()
  await loadWorkloads()
}

// 切换工作负载类型
const handleTypeChange = (type: string) => {
  selectedType.value = type
  currentPage.value = 1
  loadWorkloads()
}

// 添加标签
const handleAddLabel = () => {
  if (!editWorkloadData.value) return
  editWorkloadData.value.labels.push({ key: '', value: '' })
}

// 删除标签
const handleRemoveLabel = (index: number) => {
  if (!editWorkloadData.value) return
  editWorkloadData.value.labels.splice(index, 1)
}

// 添加注解
const handleAddAnnotation = () => {
  if (!editWorkloadData.value) return
  editWorkloadData.value.annotations.push({ key: '', value: '' })
}

// 删除注解
const handleRemoveAnnotation = (index: number) => {
  if (!editWorkloadData.value) return
  editWorkloadData.value.annotations.splice(index, 1)
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
  loadWorkloads()
}

// YAML创建工作负载
const handleAddWorkloadYAML = () => {

  if (!selectedClusterId.value && clusterList.value.length > 0) {
    // 如果没有选择集群但有集群列表，自动选择第一个
    selectedClusterId.value = clusterList.value[0].id
  }

  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择集群')
    return
  }

  // 使用当前选择的工作负载类型
  const workloadType = selectedType.value || 'Deployment'

  // 重置状态
  selectedWorkloadType.value = workloadType
  createYamlContent.value = workloadTemplates[workloadType] || workloadTemplates.Deployment
  createWorkloadDialogVisible.value = true
}

// 表单创建工作负载
const handleAddWorkloadForm = async () => {

  if (!selectedClusterId.value && clusterList.value.length > 0) {
    // 如果没有选择集群但有集群列表，自动选择第一个
    selectedClusterId.value = clusterList.value[0].id
  }

  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择集群')
    return
  }

  // 使用当前选择的工作负载类型
  const workloadType = selectedType.value || 'Deployment'

  // 初始化空的表单数据
  isCreateMode.value = true

  // 初始化扩缩容策略数据
  scalingStrategyData.value = {
    strategyType: 'RollingUpdate',
    maxSurge: '25%',
    maxUnavailable: '25%',
    minReadySeconds: 0,
    progressDeadlineSeconds: 600
  }

  // 初始化 CronJob 配置（仅当类型为 CronJob 时使用）
  cronJobConfig.value = {
    schedule: '0 * * * *',
    concurrencyPolicy: 'Allow',
    timeZone: '',
    successfulJobsHistoryLimit: 3,
    failedJobsHistoryLimit: 1,
    startingDeadlineSeconds: null,
    suspend: false,
  }

  // 初始化 Job 配置（仅当类型为 Job 或 CronJob 时使用）
  jobConfig.value = {
    completions: 1,
    parallelism: 1,
    backoffLimit: 6,
    activeDeadlineSeconds: null,
  }

  // 初始化亲和性规则为空
  affinityRules.value = []

  // 初始化工作负载数据
  editWorkloadData.value = {
    name: '',
    namespace: selectedNamespace.value || 'default',
    type: workloadType,
    labels: [{ key: 'app', value: '' }],
    annotations: [],
    replicas: 1,
    containers: [],
    initContainers: [],
    volumes: [],
    nodeSelector: {},
    affinity: {},
    tolerations: [],
    strategy: {
      type: 'RollingUpdate',
      rollingUpdate: {
        maxUnavailable: '25%',
        maxSurge: '25%'
      }
    },
    hostNetwork: undefined,
    dnsPolicy: 'ClusterFirst',
    hostname: undefined,
    subdomain: undefined,
    dnsConfig: undefined,
    terminationGracePeriodSeconds: 30,
    serviceAccountName: 'default',
    restartPolicy: (workloadType === 'Job' || workloadType === 'CronJob') ? 'OnFailure' : 'Always'
  }

  // 加载节点列表
  await loadNodes()

  activeEditTab.value = 'containers'
  editDialogVisible.value = true
}

// 创建工作负载（YAML方式）
const handleCreateFromYaml = async () => {
  if (!createYamlContent.value.trim()) {
    ElMessage.warning('请输入YAML内容')
    return
  }

  createYamlLoading.value = true
  try {
    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/workloads/create`,
      {
        clusterId: selectedClusterId.value,
        yaml: createYamlContent.value
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )
    ElMessage.success('创建成功')
    createWorkloadDialogVisible.value = false
    loadWorkloads()
  } catch (error: any) {
    const errorMsg = error.response?.data?.message || '创建工作负载失败'
    ElMessage.error(errorMsg)
  } finally {
    createYamlLoading.value = false
  }
}

// 计算YAML行数
const createYamlLineCount = computed(() => {
  return createYamlContent.value.split('\n').length
})

// 加载工作负载列表
const loadWorkloads = async () => {
  if (!selectedClusterId.value) return

  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const params: any = { clusterId: selectedClusterId.value }
    // 不传 type 参数，获取所有类型的工作负载
    if (selectedNamespace.value) params.namespace = selectedNamespace.value

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/workloads`,
      {
        params,
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    const allWorkloads = response.data.data || []

    // 根据选中的类型过滤
    if (selectedType.value) {
      workloadList.value = allWorkloads.filter((w: Workload) => w.type === selectedType.value)
    } else {
      workloadList.value = allWorkloads
    }

    // 更新每个类型的数量
    updateWorkloadTypeCounts(allWorkloads)
  } catch (error) {
    workloadList.value = []
    ElMessage.error('获取工作负载列表失败')
  } finally {
    loading.value = false
  }
}

// 更新工作负载类型的数量统计
const updateWorkloadTypeCounts = (allWorkloads: Workload[]) => {
  const typeCounts: Record<string, number> = {
    'Deployment': 0,
    'StatefulSet': 0,
    'DaemonSet': 0,
    'Job': 0,
    'CronJob': 0,
    'Pod': 0
  }

  allWorkloads.forEach((w: Workload) => {
    if (typeCounts[w.type] !== undefined) {
      typeCounts[w.type]++
    }
  })

  workloadTypes.value.forEach(type => {
    type.count = typeCounts[type.value] || 0
  })
}

// 处理下拉菜单命令
const handleActionCommand = async (command: string | any, row: Workload) => {
  selectedWorkload.value = row

  // 处理 Pod 特定的命令（对象格式）
  if (typeof command === 'object' && command !== null) {
    const { action, container, pod } = command
    if (action === 'file-browser') {
      handleOpenFileBrowser(pod, container, row.namespace)
    } else if (action === 'terminal') {
      handleOpenTerminal(pod, container, row.namespace)
    } else if (action === 'logs') {
      handleOpenLogs(pod, container, row.namespace)
    } else if (action === 'delete-pod') {
      handleDeletePod(pod, row.namespace)
    }
    return
  }

  // 处理字符串命令
  switch (command) {
    case 'edit':
      // 如果是 Pod 类型，先获取 Pod 详情
      if (row.type === 'Pod') {
        await fetchPodDetailsForMenu(row.name, row.namespace)
      } else {
        handleShowEditDialog()
      }
      break
    case 'yaml':
      handleShowYAML()
      break
    case 'pods':
      ElMessage.info('Pods 列表功能开发中...')
      break
    case 'restart':
      handleRestart()
      break
    case 'scale':
      handleScale()
      break
    case 'delete':
      handleDelete()
      break
  }
}

// 获取 Pod 详情用于操作菜单
const fetchPodDetailsForMenu = async (podName: string, namespace: string) => {
  podMenuLoading.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/pods/${namespace}/${podName}`, {
      params: { clusterId: selectedClusterId.value },
      headers: { Authorization: `Bearer ${token}` }
    })
    // 后端现在返回标准格式 {code: 0, message: "success", data: pod}
    podMenuData.value = response.data.data
  } catch (error: any) {
    ElMessage.error('获取 Pod 详情失败: ' + (error.response?.data?.message || error.message))
    podMenuData.value = null
  } finally {
    podMenuLoading.value = false
  }
}

// 删除 Pod
const handleDeletePod = async (podName: string, namespace: string) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 Pod "${podName}" 吗？此操作不可撤销！`,
      '删除确认',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const loadingInstance = ElLoading.service({
      lock: true,
      text: '正在删除 Pod...',
      background: 'rgba(0, 0, 0, 0.7)'
    })

    try {
      const token = localStorage.getItem('token')
      await axios.delete(`/api/v1/plugins/kubernetes/resources/workloads/${namespace}/${podName}`, {
        params: {
          clusterId: selectedClusterId.value,
          type: 'Pod'
        },
        headers: { Authorization: `Bearer ${token}` }
      })

      loadingInstance.setText('删除成功，正在刷新...')
      await new Promise(resolve => setTimeout(resolve, 500))
      await loadWorkloads()

      loadingInstance.close()
      ElMessage.success('Pod 删除成功')
    } catch (err) {
      loadingInstance.close()
      throw err
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除 Pod 失败')
    }
  }
}

// 工作负载 YAML 操作（用于非Pod类型）
const handleWorkloadYAML = (row: Workload) => {
  selectedWorkload.value = row
  handleShowYAML()
}

// 工作负载编辑操作（用于非Pod类型）
const handleWorkloadEdit = (row: Workload) => {
  selectedWorkload.value = row
  handleShowEditDialog()
}

// 工作负载删除操作（用于非Pod类型）
const handleWorkloadDelete = (row: Workload) => {
  selectedWorkload.value = row
  handleDelete()
}

// 批量操作相关函数
const handleSelectionChange = (selection: Workload[]) => {
  selectedWorkloads.value = selection
}

const clearSelection = () => {
  selectedWorkloads.value = []
}

const handleBatchDelete = async () => {
  if (selectedWorkloads.value.length === 0) {
    ElMessage.warning('请先选择要删除的工作负载')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedWorkloads.value.length} 个工作负载吗？`,
      '批量删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    batchActionLoading.value = true
    const token = localStorage.getItem('token')

    const response = await axios.post(
      '/api/v1/plugins/kubernetes/resources/workloads/batch/delete',
      {
        clusterId: selectedClusterId.value,
        workloads: selectedWorkloads.value.map(w => ({
          namespace: w.namespace,
          name: w.name,
          type: w.type || selectedType.value
        }))
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    if (response.data.code === 0) {
      const results = response.data.data.results || []
      const successCount = results.filter(r => r.success).length
      const failureCount = results.filter(r => !r.success).length

      if (failureCount > 0) {
        const failures = results.filter(r => !r.success)
        const failureMsg = failures.map(f => `${f.name}: ${f.message}`).join('; ')
        ElMessage.warning(`批量删除完成：成功 ${successCount} 个，失败 ${failureCount} 个。${failureMsg}`)
      } else {
        ElMessage.success(`成功删除 ${successCount} 个工作负载`)
      }

      clearSelection()
      await loadWorkloads()
    } else {
      ElMessage.error(response.data.message || '批量删除失败')
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败')
    }
  } finally {
    batchActionLoading.value = false
  }
}

const handleBatchRestart = async () => {
  if (selectedWorkloads.value.length === 0) {
    ElMessage.warning('请先选择要重启的工作负载')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要重启选中的 ${selectedWorkloads.value.length} 个工作负载吗？`,
      '批量重启确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    batchActionLoading.value = true
    const token = localStorage.getItem('token')

    const workloadData = selectedWorkloads.value.map(w => ({
      namespace: w.namespace,
      name: w.name,
      type: w.type || selectedType.value
    }))


    const response = await axios.post(
      '/api/v1/plugins/kubernetes/resources/workloads/batch/restart',
      {
        clusterId: selectedClusterId.value,
        workloads: workloadData
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )


    if (response.data.code === 0) {
      const results = response.data.data.results || []
      const successCount = results.filter(r => r.success).length
      const failureCount = results.filter(r => !r.success).length


      if (failureCount > 0) {
        // 显示失败详情
        const failures = results.filter(r => !r.success)
        const failureMsg = failures.map(f => `${f.name}: ${f.message}`).join('; ')
        ElMessage.warning(`批量重启完成：成功 ${successCount} 个，失败 ${failureCount} 个。${failureMsg}`)
      } else {
        ElMessage.success(`成功重启 ${successCount} 个工作负载`)
      }

      clearSelection()
      await loadWorkloads()
    } else {
      ElMessage.error(response.data.message || '批量重启失败')
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '批量重启失败')
    }
  } finally {
    batchActionLoading.value = false
  }
}

const handleBatchPause = async () => {
  if (selectedWorkloads.value.length === 0) {
    ElMessage.warning('请先选择要停止的工作负载')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要停止选中的 ${selectedWorkloads.value.length} 个工作负载吗？`,
      '批量停止确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    batchActionLoading.value = true
    const token = localStorage.getItem('token')

    const response = await axios.post(
      '/api/v1/plugins/kubernetes/resources/workloads/batch/pause',
      {
        clusterId: selectedClusterId.value,
        workloads: selectedWorkloads.value.map(w => ({
          namespace: w.namespace,
          name: w.name,
          type: w.type || selectedType.value
        }))
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    if (response.data.code === 0) {
      const results = response.data.data.results || []
      const successCount = results.filter(r => r.success).length
      const failureCount = results.filter(r => !r.success).length

      if (failureCount > 0) {
        const failures = results.filter(r => !r.success)
        const failureMsg = failures.map(f => `${f.name}: ${f.message}`).join('; ')
        ElMessage.warning(`批量停止完成：成功 ${successCount} 个，失败 ${failureCount} 个。${failureMsg}`)
      } else {
        ElMessage.success(`成功停止 ${successCount} 个工作负载`)
      }

      clearSelection()
      await loadWorkloads()
    } else {
      ElMessage.error(response.data.message || '批量停止失败')
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('批量停止失败')
    }
  } finally {
    batchActionLoading.value = false
  }
}

const handleBatchResume = async () => {
  if (selectedWorkloads.value.length === 0) {
    ElMessage.warning('请先选择要恢复的工作负载')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要恢复选中的 ${selectedWorkloads.value.length} 个工作负载吗？`,
      '批量恢复确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    batchActionLoading.value = true
    const token = localStorage.getItem('token')

    const response = await axios.post(
      '/api/v1/plugins/kubernetes/resources/workloads/batch/resume',
      {
        clusterId: selectedClusterId.value,
        workloads: selectedWorkloads.value.map(w => ({
          namespace: w.namespace,
          name: w.name,
          type: w.type || selectedType.value
        }))
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    if (response.data.code === 0) {
      const results = response.data.data.results || []
      const successCount = results.filter(r => r.success).length
      const failureCount = results.filter(r => !r.success).length

      if (failureCount > 0) {
        const failures = results.filter(r => !r.success)
        const failureMsg = failures.map(f => `${f.name}: ${f.message}`).join('; ')
        ElMessage.warning(`批量恢复完成：成功 ${successCount} 个，失败 ${failureCount} 个。${failureMsg}`)
      } else {
        ElMessage.success(`成功恢复 ${successCount} 个工作负载`)
      }

      clearSelection()
      await loadWorkloads()
    } else {
      ElMessage.error(response.data.message || '批量恢复失败')
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('批量恢复失败')
    }
  } finally {
    batchActionLoading.value = false
  }
}

// 加载节点列表
const loadNodes = async () => {
  if (!selectedClusterId.value) {
    nodeList.value = []
    return
  }

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      '/api/v1/plugins/kubernetes/resources/nodes',
      {
        params: { clusterId: selectedClusterId.value },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    nodeList.value = response.data.data || []
  } catch (error: any) {
    nodeList.value = []
  }
}

// 添加匹配规则
const handleAddMatchRule = () => {
  if (!editWorkloadData.value) return
  if (!editWorkloadData.value.matchRules) {
    editWorkloadData.value.matchRules = []
  }
  // 自动切换到"调度规则匹配"类型
  editWorkloadData.value.schedulingType = 'match'
  editWorkloadData.value.matchRules.push({
    key: '',
    operator: 'In',
    value: ''
  })
}

// 删除匹配规则
const handleRemoveMatchRule = (index: number) => {
  if (!editWorkloadData.value || !editWorkloadData.value.matchRules) return
  editWorkloadData.value.matchRules.splice(index, 1)

  // 如果没有规则了，自动切换到"任意可用节点"
  if (editWorkloadData.value.matchRules.length === 0) {
    editWorkloadData.value.schedulingType = 'any'
  }
}

// 更新调度配置
const handleUpdateScheduling = (data: { schedulingType: string; specifiedNode: string }) => {
  if (!editWorkloadData.value) {
    return
  }


  // 使用 Object.assign 确保响应式更新
  Object.assign(editWorkloadData.value, {
    schedulingType: data.schedulingType,
    specifiedNode: data.specifiedNode
  })

}

// 更新表单数据
const handleUpdateFormData = (data: any) => {
  if (editWorkloadData.value) {
    Object.assign(editWorkloadData.value, data)
  }
}

// 更新扩缩容策略
const handleUpdateScalingStrategy = (data: any) => {
  if (editWorkloadData.value) {
    editWorkloadData.value.strategyType = data.strategyType
    editWorkloadData.value.maxSurge = data.maxSurge
    editWorkloadData.value.maxUnavailable = data.maxUnavailable
    editWorkloadData.value.minReadySeconds = data.minReadySeconds
    editWorkloadData.value.progressDeadlineSeconds = data.progressDeadlineSeconds
    editWorkloadData.value.revisionHistoryLimit = data.revisionHistoryLimit
    editWorkloadData.value.timeoutSeconds = data.timeoutSeconds
  }
  scalingStrategyData.value = { ...data }
}

// 更新 CronJob 配置
const updateCronJobConfig = (data: any) => {
  cronJobConfig.value = { ...data }
}

// 更新 Job 配置
const updateJobConfig = (data: any) => {
  jobConfig.value = { ...data }
}

// 显示 Pod 详情
const showPodDetail = (pod: any) => {
  selectedPodName.value = pod.metadata?.name || ''
  selectedPodNamespace.value = pod.metadata?.namespace || detailData.value.namespace || ''
  podDetailVisible.value = true
}

// 处理工作负载列表中点击 Pod 类型项目
const handlePodWorkloadClick = async (workload: Workload) => {
  // 构造一个类似 Pod 对象的结构
  const pod = {
    metadata: {
      name: workload.name,
      namespace: workload.namespace
    }
  }
  showPodDetail(pod)
}

// 显示工作负载详情
const handleShowDetail = async (workload: Workload) => {
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value

    // 并行获取所有数据
    const [workloadRes, replicaSetsRes, podsRes, servicesRes, ingressesRes] = await Promise.all([
      axios.get(`/api/v1/plugins/kubernetes/resources/workloads/${workload.namespace}/${workload.name}`, {
        params: { clusterId, type: workload.type },
        headers: { Authorization: `Bearer ${token}` }
      }),
      axios.get(`/api/v1/plugins/kubernetes/resources/workloads/${workload.namespace}/${workload.name}/replicasets`, {
        params: { clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }),
      axios.get(`/api/v1/plugins/kubernetes/resources/workloads/${workload.namespace}/${workload.name}/pods`, {
        params: { clusterId, type: workload.type },
        headers: { Authorization: `Bearer ${token}` }
      }),
      axios.get(`/api/v1/plugins/kubernetes/resources/workloads/${workload.namespace}/${workload.name}/services`, {
        params: { clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }),
      axios.get(`/api/v1/plugins/kubernetes/resources/workloads/${workload.namespace}/${workload.name}/ingresses`, {
        params: { clusterId },
        headers: { Authorization: `Bearer ${token}` }
      })
    ])

    // 获取 Pods metrics
    try {
      const metricsRes = await axios.get(`/api/v1/plugins/kubernetes/resources/pods/metrics`, {
        params: { clusterId, namespace: workload.namespace },
        headers: { Authorization: `Bearer ${token}` }
      })
      podMetricsData.value = metricsRes.data.data.metrics || {}
    } catch (metricsError) {
      podMetricsData.value = {}
    }

    // 提取工作负载对象
    const workloadObj = workloadRes.data.data.items?.[0]

    // 整理详情数据
    detailData.value = {
      name: workload.name,
      namespace: workload.namespace,
      type: workload.type,
      workload: workloadObj,
      replicaSets: replicaSetsRes.data.data.items || [],
      pods: podsRes.data.data.items || [],
      services: servicesRes.data.data.items || [],
      ingresses: ingressesRes.data.data.items || []
    }


    // 更新暂停状态
    isWorkloadPaused.value = !!workloadObj.spec?.paused

    // 如果是 CronJob，加载 CronJob 配置
    if (workload.type === 'CronJob' && workloadObj.spec) {
      cronJobConfig.value = {
        schedule: workloadObj.spec.schedule || '0 * * * *',
        concurrencyPolicy: workloadObj.spec.concurrencyPolicy || 'Allow',
        timeZone: workloadObj.spec.timeZone || '',
        successfulJobsHistoryLimit: workloadObj.spec.successfulJobsHistoryLimit || 3,
        failedJobsHistoryLimit: workloadObj.spec.failedJobsHistoryLimit || 1,
        startingDeadlineSeconds: workloadObj.spec.startingDeadlineSeconds || null,
        suspend: workloadObj.spec.suspend || false,
      }

      // 加载 CronJob 的 Job 配置
      const jobSpec = workloadObj.spec.jobTemplate?.spec
      if (jobSpec) {
        jobConfig.value = {
          completions: jobSpec.completions || 1,
          parallelism: jobSpec.parallelism || 1,
          backoffLimit: jobSpec.backoffLimit || 6,
          activeDeadlineSeconds: jobSpec.activeDeadlineSeconds || null,
        }
      }
    }

    // 如果是 Job，加载 Job 配置
    if (workload.type === 'Job' && workloadObj.spec) {
      jobConfig.value = {
        completions: workloadObj.spec.completions || 1,
        parallelism: workloadObj.spec.parallelism || 1,
        backoffLimit: workloadObj.spec.backoffLimit || 6,
        activeDeadlineSeconds: workloadObj.spec.activeDeadlineSeconds || null,
      }
    }

    activeDetailTab.value = 'pods'
    detailDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error('获取工作负载详情失败')
  }
}

// 格式化年龄显示（短格式）
const formatAgeShort = (timestamp: string) => {
  if (!timestamp) return '-'
  const now = new Date()
  const created = new Date(timestamp)
  const diff = now.getTime() - created.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))

  if (days > 0) {
    return `${days}d${hours}h`
  } else if (hours > 0) {
    return `${hours}h${minutes}m`
  } else {
    return `${minutes}m`
  }
}

// 获取Pod的就绪容器数
const getReadyContainers = (pod: any) => {
  if (!pod.status?.containerStatuses) return '0'
  const ready = pod.status.containerStatuses.filter((cs: any) => cs.ready).length
  return ready
}

// 获取Pod的重启次数
const getRestartCount = (pod: any) => {
  if (!pod.status?.containerStatuses) return 0
  return pod.status.containerStatuses.reduce((sum: number, cs: any) => sum + (cs.restartCount || 0), 0)
}

// 获取Pod状态对应的标签类型
const getPodStatusType = (status: string) => {
  const statusMap: Record<string, string> = {
    'Running': 'success',
    'Pending': 'warning',
    'Failed': 'danger',
    'Succeeded': 'info',
    'Unknown': 'info'
  }
  return statusMap[status] || 'info'
}

// 清理状态文本，去除多余的标点符号
const getPodStatusText = (status: string | undefined) => {
  if (!status) return '-'
  // 去除所有结尾的标点符号（包括中文和英文）
  let cleaned = status.trim()
  // 持续去除结尾的标点符号，直到没有为止
  while (cleaned && /[.,，。、;；:：！!？?]/.test(cleaned.slice(-1))) {
    cleaned = cleaned.slice(0, -1)
  }
  return cleaned || '-'
}

// 计算资源年龄
const calculateAge = (creationTimestamp: string | undefined) => {
  if (!creationTimestamp) return '-'
  const now = new Date()
  const created = new Date(creationTimestamp)
  const diff = now.getTime() - created.getTime()

  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) {
    return `${days}天`
  } else if (hours > 0) {
    return `${hours}小时`
  } else if (minutes > 0) {
    return `${minutes}分钟`
  } else {
    return `${seconds}秒`
  }
}

// 获取Service类型颜色
const getServiceTypeColor = (type: string | undefined) => {
  const colorMap: Record<string, string> = {
    'ClusterIP': 'info',
    'NodePort': 'success',
    'LoadBalancer': 'warning',
    'ExternalName': 'danger'
  }
  return colorMap[type || ''] || 'info'
}

// 获取Ingress域名列表（computed）
const ingressHosts = computed(() => {
  if (!detailData.value?.ingresses || !Array.isArray(detailData.value.ingresses)) {
    return []
  }

  const hostMap: Record<string, string[]> = {}

  detailData.value.ingresses.forEach((ingress: any) => {
    if (ingress.spec?.rules) {
      ingress.spec.rules.forEach((rule: any) => {
        if (rule.host) {
          if (!hostMap[rule.host]) {
            hostMap[rule.host] = []
          }
          hostMap[rule.host].push(ingress.metadata?.name || '')
        }
      })
    }
  })

  return Object.keys(hostMap).map(host => ({
    host,
    names: hostMap[host]
  }))
})

// 获取Ingress路由规则列表（computed）
const ingressRules = computed(() => {
  if (!detailData.value?.ingresses || !Array.isArray(detailData.value.ingresses)) {
    return []
  }

  const rules: any[] = []

  detailData.value.ingresses.forEach((ingress: any) => {
    const ingressName = ingress.metadata?.name || ''

    if (ingress.spec?.rules) {
      ingress.spec.rules.forEach((rule: any) => {
        const host = rule.host || '-'

        if (rule.http?.paths) {
          rule.http.paths.forEach((path: any) => {
            rules.push({
              ingressName,
              host,
              path: path.path || '/',
              pathType: path.pathType || 'Prefix',
              serviceName: path.backend?.service?.name || '-',
              servicePort: path.backend?.service?.port?.number || path.backend?.service?.port?.name || '-'
            })
          })
        }
      })
    }
  })

  return rules
})

// 排序后的 ReplicaSet 列表（computed）
const sortedReplicaSets = computed(() => {
  if (!detailData.value?.replicaSets || !Array.isArray(detailData.value.replicaSets)) {
    return []
  }

  // 复制数组并排序
  return [...detailData.value.replicaSets].sort((a: any, b: any) => {
    const revisionA = getReplicaSetRevision(a)
    const revisionB = getReplicaSetRevision(b)

    // 如果都是数字，按数字降序排序（最新版本在前）
    const numA = parseInt(revisionA)
    const numB = parseInt(revisionB)

    if (!isNaN(numA) && !isNaN(numB)) {
      return numB - numA
    }

    // 如果不是数字，按字符串降序排序
    return revisionB.localeCompare(revisionA)
  })
})

// 工作负载类型（computed）
const workloadType = computed(() => {
  if (!detailData.value?.type) return '-'
  const typeMap: Record<string, string> = {
    'Deployment': 'Deployment',
    'StatefulSet': 'StatefulSet',
    'DaemonSet': 'DaemonSet',
    'ReplicaSet': 'ReplicaSet'
  }
  return typeMap[detailData.value.type] || detailData.value.type
})

// 处理暂停状态变化
const handlePauseChange = async (value: boolean) => {
  if (!detailData.value?.workload) return

  pauseLoading.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const namespace = detailData.value.namespace
    const name = detailData.value.name
    const type = detailData.value.type

    // 调用后端API更新暂停状态
    await axios.post(
      `/api/v1/plugins/kubernetes/workloads/pause`,
      {
        clusterId,
        namespace,
        name,
        type,
        paused: value
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    ElMessage.success(value ? '工作负载已暂停' : '工作负载已恢复运行')

    // 保存当前标签页
    const currentTab = activeDetailTab.value

    // 刷新详情
    await handleShowDetail({
      namespace,
      name,
      type
    } as Workload)

    // 恢复标签页
    activeDetailTab.value = currentTab
  } catch (error: any) {
    ElMessage.error('更新暂停状态失败: ' + (error.response?.data?.message || error.message))
    // 恢复开关状态
    isWorkloadPaused.value = !value
  } finally {
    pauseLoading.value = false
  }
}

// 获取 ReplicaSet 版本号
const getReplicaSetRevision = (replicaSet: any) => {
  const annotations = replicaSet.metadata?.annotations || {}
  const revision = annotations['deployment.kubernetes.io/revision']
  return revision || '-'
}

// 获取 ReplicaSet 镜像列表
const getReplicaSetImages = (replicaSet: any) => {
  const containers = replicaSet.spec?.template?.spec?.containers || []
  return containers.map((c: any) => {
    const image = c.image || ''
    // 只保留镜像名和tag，去掉registry部分
    const parts = image.split('/')
    const nameAndTag = parts[parts.length - 1]
    return nameAndTag
  })
}

// 获取 ReplicaSet 状态类型
const getReplicaSetStatusType = (replicaSet: any) => {
  const replicas = replicaSet.spec?.replicas || 0
  const availableReplicas = replicaSet.status?.availableReplicas || 0

  if (replicas === 0) return 'info'
  if (availableReplicas === replicas) return 'success'
  if (availableReplicas > 0) return 'warning'
  return 'danger'
}

// 获取 ReplicaSet 状态文本
const getReplicaSetStatusText = (replicaSet: any) => {
  const replicas = replicaSet.spec?.replicas || 0
  const availableReplicas = replicaSet.status?.availableReplicas || 0

  if (replicas === 0) return '已停止'
  if (availableReplicas === replicas) return '运行中'
  if (availableReplicas > 0) return `${availableReplicas}/${replicas} 就绪`
  return '未就绪'
}

// 判断是否为当前版本的 ReplicaSet
const isCurrentReplicaSet = (replicaSet: any) => {
  if (!detailData.value?.workload) return false
  const workload = detailData.value.workload

  // 对于 Deployment，检查当前 ReplicaSet 是否匹配
  if (workload.status?.currentReplicas) {
    // 通过 annotations 中的 revision 判断
    const currentRevision = workload.metadata?.annotations?.['deployment.kubernetes.io/revision']
    const replicaSetRevision = replicaSet.metadata?.annotations?.['deployment.kubernetes.io/revision']
    return currentRevision === replicaSetRevision
  }

  return false
}

// 获取状态点图标
const getStatusDotIcon = (statusType: string) => {
  const iconMap: Record<string, any> = {
    'success': CircleCheck,
    'warning': Warning,
    'danger': CircleClose,
    'info': CircleCheck,
    'primary': CircleCheck
  }
  return iconMap[statusType] || CircleCheck
}

// 查看 ReplicaSet YAML
const handleViewReplicaSetYAML = async (replicaSet: any) => {
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const namespace = replicaSet.metadata?.namespace
    const name = replicaSet.metadata?.name

    // 直接将 ReplicaSet 对象转换为 YAML
    replicaSetYamlContent.value = yaml.dump(replicaSet, {
      lineWidth: -1,
      noRefs: true,
      sortKeys: false
    })

    replicaSetYamlData.value = {
      name,
      namespace
    }
    replicaSetYamlDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error('获取 ReplicaSet YAML 失败')
  }
}

// 复制 ReplicaSet YAML
const handleCopyReplicaSetYAML = async () => {
  try {
    await navigator.clipboard.writeText(replicaSetYamlContent.value)
    ElMessage.success('YAML 已复制到剪贴板')
  } catch (error: any) {
    ElMessage.error('复制失败')
  }
}

// 回滚到指定版本
const handleRollback = async (replicaSet: any) => {
  try {
    await ElMessageBox.confirm(
      `确定要回滚到版本 #${getReplicaSetRevision(replicaSet)} 吗？此操作将创建一个新的 ReplicaSet 并更新工作负载。`,
      '回滚确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const namespace = detailData.value.namespace
    const name = detailData.value.name
    const type = detailData.value.type

    // 调用后端回滚API
    await axios.post(
      `/api/v1/plugins/kubernetes/workloads/rollback`,
      {
        clusterId,
        namespace,
        name,
        type,
        revision: getReplicaSetRevision(replicaSet)
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    ElMessage.success('回滚成功')

    // 保存当前标签页
    const currentTab = activeDetailTab.value

    // 刷新详情
    await handleShowDetail({
      namespace,
      name,
      type
    } as Workload)

    // 恢复标签页
    activeDetailTab.value = currentTab
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('回滚失败: ' + (error.response?.data?.message || error.message))
    }
  }
}

// 获取运行时信息
const getRuntimeInfo = () => {
  if (!detailData.value?.workload || !detailData.value?.pods) {
    return []
  }

  const workload = detailData.value.workload
  const pods = detailData.value.pods
  const now = new Date()

  const info: any[] = []

  // Pod 状态
  const runningPods = pods.filter((p: any) => p.status?.phase === 'Running')
  const pendingPods = pods.filter((p: any) => p.status?.phase === 'Pending')
  const failedPods = pods.filter((p: any) => !['Running', 'Succeeded'].includes(p.status?.phase))

  info.push({
    category: 'Pod 状态',
    icon: 'Box',
    iconClass: 'icon-pod',
    status: runningPods.length === pods.length ? '正常' : '异常',
    statusIcon: runningPods.length === pods.length ? 'CircleCheck' : 'Warning',
    statusType: runningPods.length === pods.length ? 'success' : 'warning',
    isLoading: false,
    message: `总计 ${pods.length} 个 Pod：运行中 ${runningPods.length} 个，等待中 ${pendingPods.length} 个，失败 ${failedPods.length} 个`,
    lastUpdate: calculateAge(workload.metadata?.creationTimestamp)
  })

  // 副本状态
  const specReplicas = workload.spec?.replicas || 0
  const statusReplicas = workload.status?.replicas || 0
  const updatedReplicas = workload.status?.updatedReplicas || 0
  const availableReplicas = workload.status?.availableReplicas || 0
  const readyReplicas = workload.status?.readyReplicas || 0

  const replicasHealthy = specReplicas === availableReplicas && specReplicas === readyReplicas

  info.push({
    category: '副本状态',
    icon: 'CopyDocument',
    iconClass: 'icon-replica',
    status: replicasHealthy ? '正常' : '更新中',
    statusIcon: replicasHealthy ? 'CircleCheck' : 'Loading',
    statusType: replicasHealthy ? 'success' : 'primary',
    isLoading: !replicasHealthy,
    message: `期望 ${specReplicas} 个，当前 ${statusReplicas} 个，可用 ${availableReplicas} 个，就绪 ${readyReplicas} 个，已更新 ${updatedReplicas} 个`,
    lastUpdate: calculateAge(workload.status?.conditions?.find((c: any) => c.type === 'Progressing')?.lastTransitionTime)
  })

  // 更新状态
  const progressingCondition = workload.status?.conditions?.find((c: any) => c.type === 'Progressing')
  const availableCondition = workload.status?.conditions?.find((c: any) => c.type === 'Available')

  info.push({
    category: '更新状态',
    icon: 'Refresh',
    iconClass: 'icon-update',
    status: progressingCondition?.status === 'True' ? '进行中' : '已完成',
    statusIcon: progressingCondition?.status === 'True' ? 'Loading' : 'CircleCheck',
    statusType: progressingCondition?.status === 'True' ? 'primary' : 'success',
    isLoading: progressingCondition?.status === 'True',
    message: progressingCondition?.message || '副本集更新正常',
    lastUpdate: calculateAge(progressingCondition?.lastTransitionTime)
  })

  // 可用性状态
  info.push({
    category: '可用性',
    icon: 'CircleCheck',
    iconClass: 'icon-available',
    status: availableCondition?.status === 'True' ? '可用' : '不可用',
    statusIcon: availableCondition?.status === 'True' ? 'CircleCheck' : 'CircleClose',
    statusType: availableCondition?.status === 'True' ? 'success' : 'danger',
    isLoading: false,
    message: availableCondition?.message || '工作负载可用性检查',
    lastUpdate: calculateAge(availableCondition?.lastTransitionTime)
  })

  // 暂停状态
  const isPaused = workload.spec?.paused

  if (isPaused !== undefined) {
    info.push({
      category: '暂停状态',
      icon: 'VideoPause',
      iconClass: 'icon-paused',
      status: isPaused ? '已暂停' : '运行中',
      statusIcon: isPaused ? 'VideoPause' : 'VideoPlay',
      statusType: isPaused ? 'info' : 'success',
      isLoading: false,
      message: isPaused ? '工作负载更新已暂停，不会创建新的副本' : '工作负载正常运行，会自动更新副本',
      lastUpdate: '-'
    })
  }

  // 碰撞状态
  const collisionCount = workload.status?.collisionCount || 0

  if (collisionCount > 0) {
    info.push({
      category: '冲突计数',
      icon: 'Warning',
      iconClass: 'icon-collision',
      status: '有冲突',
      statusIcon: 'Warning',
      statusType: 'warning',
      isLoading: false,
      message: `检测到 ${collisionCount} 次更新冲突，可能有并发更新问题`,
      lastUpdate: calculateAge(workload.metadata?.creationTimestamp)
    })
  }

  // 观察者状态
  if (workload.status?.observedGeneration) {
    const observedGeneration = workload.status.observedGeneration
    const generation = workload.metadata?.generation || 0

    info.push({
      category: '观察者',
      icon: 'View',
      iconClass: 'icon-observer',
      status: observedGeneration === generation ? '同步' : '滞后',
      statusIcon: observedGeneration === generation ? 'CircleCheck' : 'Clock',
      statusType: observedGeneration === generation ? 'success' : 'warning',
      isLoading: false,
      message: `当前代数 ${generation}，已观察到代数 ${observedGeneration}${observedGeneration < generation ? '，控制器正在处理最新配置' : ''}`,
      lastUpdate: calculateAge(workload.metadata?.creationTimestamp)
    })
  }

  return info
}

// 获取容器镜像列表（返回数组）
const getContainerImageList = (workload: any) => {
  if (!workload?.spec?.template?.spec?.containers) return []
  return workload.spec.template.spec.containers.map((c: any) => c.image).filter((img: string) => img)
}

// 获取容器镜像列表（逗号分隔）
const getContainerImages = (workload: any) => {
  const images = getContainerImageList(workload)
  return images.length > 0 ? images.join(', ') : '-'
}

// 获取 Pod CPU 使用率
// Pod metrics 数据
const podMetricsData = ref<Record<string, { cpu: number, memory: number, cpuStr: string, memoryStr: string }>>({})

// 获取 Pod CPU 使用量（从 metrics 数据）
const getPodCPU = (pod: any) => {
  const podName = pod.metadata?.name
  const metrics = podMetricsData.value[podName]

  if (metrics && metrics.cpu > 0) {
    return metrics.cpuStr
  }

  // 如果没有 metrics，显示 requests 值
  const cpuRequests = pod.spec?.containers?.reduce((sum: number, c: any) => {
    const cpu = c.resources?.requests?.cpu
    if (cpu) {
      if (cpu.endsWith('m')) {
        return sum + parseInt(cpu)
      }
      return sum + parseInt(cpu) * 1000
    }
    return sum
  }, 0) || 0

  if (cpuRequests > 0) {
    if (cpuRequests >= 1000) {
      return `${(cpuRequests / 1000).toFixed(1)} Core (req)`
    }
    return `${cpuRequests}m (req)`
  }
  return '-'
}

// 获取 Pod 内存使用量（从 metrics 数据）
const getPodMemory = (pod: any) => {
  const podName = pod.metadata?.name
  const metrics = podMetricsData.value[podName]

  if (metrics && metrics.memory > 0) {
    return metrics.memoryStr
  }

  // 如果没有 metrics，显示 requests 值
  const memoryRequests = pod.spec?.containers?.reduce((sum: number, c: any) => {
    const mem = c.resources?.requests?.memory
    if (mem) {
      if (mem.endsWith('Mi')) {
        return sum + parseInt(mem)
      }
      if (mem.endsWith('Gi')) {
        return sum + parseInt(mem) * 1024
      }
    }
    return sum
  }, 0) || 0

  if (memoryRequests > 0) {
    if (memoryRequests >= 1024) {
      return `${(memoryRequests / 1024).toFixed(1)} Gi (req)`
    }
    return `${memoryRequests} Mi (req)`
  }
  return '-'
}

// 处理 Pod 操作
const handlePodAction = (command: any, pod: any) => {
  const { action, container, pod: podName } = command
  const namespace = pod.metadata?.namespace

  if (action === 'terminal') {
    handleOpenTerminal(podName, container, namespace)
  } else if (action === 'logs') {
    handleOpenLogs(podName, container, namespace)
  }
}

// 打开终端
const handleOpenTerminal = async (podName: string, containerName: string, namespace: string) => {
  terminalData.value = {
    pod: podName,
    container: containerName,
    namespace
  }
  terminalConnected.value = false
  terminalDialogVisible.value = true
  // 不在这里初始化终端，而是在对话框完全打开后通过 @opened 事件初始化
}

// 对话框完全打开后的回调
const handleDialogOpened = async () => {
  await nextTick()
  await initTerminal()
}

// 初始化终端
const initTerminal = async () => {

  // 等待 DOM 元素准备好，最多重试 10 次
  let retries = 0
  while (!terminalWrapper.value && retries < 10) {
    await new Promise(resolve => setTimeout(resolve, 100))
    retries++
  }

  if (!terminalWrapper.value) {
    return
  }


  // 清空容器
  terminalWrapper.value.innerHTML = ''

  // 创建终端实例
  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79',
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#11a8cd',
      white: '#e5e5e5',
      brightBlack: '#666666',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#f5f543',
      brightBlue: '#3b8eea',
      brightMagenta: '#d670d6',
      brightCyan: '#29b8db',
      brightWhite: '#ffffff'
    }
  })

  // 加载插件
  const fitAddon = new FitAddon()
  const webLinksAddon = new WebLinksAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(webLinksAddon)

  // 打开终端
  terminal.open(terminalWrapper.value)
  fitAddon.fit()

  // 欢迎信息
  terminal.writeln('\x1b[1;32m正在连接到容器...\x1b[0m')

  // 获取token
  const token = localStorage.getItem('token')
  const clusterId = selectedClusterId.value

  // 构建WebSocket URL - 在开发环境直接连接后端，生产环境使用当前域名
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.hostname
  // 开发环境直接连接9876端口，生产环境使用当前端口
  const isDev = import.meta.env.DEV
  const port = isDev ? '9876' : (window.location.port || (window.location.protocol === 'https:' ? '443' : '9876'))
  const wsUrl = `${protocol}//${host}:${port}/api/v1/plugins/kubernetes/shell/pods?` +
    `clusterId=${clusterId}&` +
    `namespace=${terminalData.value.namespace}&` +
    `podName=${terminalData.value.pod}&` +
    `container=${terminalData.value.container}&` +
    `token=${token}`


  try {
    // 建立WebSocket连接
    terminalWebSocket = new WebSocket(wsUrl)

    terminalWebSocket.onopen = () => {
      terminalConnected.value = true
      terminal.clear()
      terminal.writeln('\x1b[1;32m✓ 已连接到容器 ' + terminalData.value.container + '\x1b[0m')
      terminal.writeln('')
    }

    terminalWebSocket.onmessage = (event) => {
      terminal.write(event.data)
    }

    terminalWebSocket.onerror = (error) => {
      terminal.writeln('\x1b[1;31m✗ 连接错误\x1b[0m')
      terminal.writeln('请检查:')
      terminal.writeln('1. 集群连接是否正常')
      terminal.writeln('2. Pod是否正在运行')
      terminal.writeln('3. 浏览器控制台是否有错误信息')
    }

    terminalWebSocket.onclose = (event) => {
      terminalConnected.value = false
      // 安全检查：terminal 可能已经被销毁
      if (terminal) {
        try {
          terminal.writeln('\x1b[1;33m连接已关闭\x1b[0m')
        } catch (e) {
        }
      }
    }

    // 处理用户输入
    terminal.onData((data: string) => {
      if (terminalWebSocket && terminalWebSocket.readyState === WebSocket.OPEN) {
        terminalWebSocket.send(data)
      }
    })

    // 处理窗口大小变化
    terminal.onResize(({ cols, rows }) => {
      if (terminalWebSocket && terminalWebSocket.readyState === WebSocket.OPEN) {
        terminalWebSocket.send(JSON.stringify({ type: 'resize', cols, rows }))
      }
    })

  } catch (error: any) {
    terminal.writeln('\x1b[1;31m✗ 连接失败: ' + error.message + '\x1b[0m')
  }
}

// 关闭终端
const handleCloseTerminal = () => {
  if (terminalWebSocket) {
    terminalWebSocket.close()
    terminalWebSocket = null
  }
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  terminalConnected.value = false
}

// 打开日志
const handleOpenLogs = async (podName: string, containerName: string, namespace: string) => {
  logsData.value = {
    pod: podName,
    container: containerName,
    namespace
  }
  logsContent.value = ''
  logsDialogVisible.value = true
  // 不在这里加载日志，等待对话框打开后再加载
}

// 日志对话框打开后的事件处理
const handleLogsDialogOpened = async () => {
  await handleLoadLogs()

  // 启动自动刷新定时器（每3秒刷新一次）
  if (logsRefreshTimer) clearInterval(logsRefreshTimer)
  logsRefreshTimer = window.setInterval(() => {
    handleLoadLogs()
  }, 3000)
}

// 停止日志自动刷新
const stopLogsAutoRefresh = () => {
  if (logsRefreshTimer) {
    clearInterval(logsRefreshTimer)
    logsRefreshTimer = null
  }
}

// 打开文件浏览器
const handleOpenFileBrowser = (podName: string, containerName: string, namespace: string) => {
  if (!selectedClusterId.value) {
    ElMessage.error('请先选择集群')
    return
  }
  selectedFileBrowserPod.value = podName
  selectedFileBrowserNamespace.value = namespace
  selectedFileBrowserContainer.value = containerName
  fileBrowserVisible.value = true
}

// 加载日志
const handleLoadLogs = async () => {
  logsLoading.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const { pod, container, namespace } = logsData.value

    const response = await axios.get('/api/v1/plugins/kubernetes/resources/pods/logs', {
      params: {
        clusterId,
        namespace,
        podName: pod,
        container,
        tailLines: logsTailLines.value
      },
      headers: { Authorization: `Bearer ${token}` }
    })

    logsContent.value = response.data.data?.logs || ''

    // 自动滚动到底部 - 使用 setTimeout 确保 DOM 完全渲染
    if (logsAutoScroll.value) {
      setTimeout(() => {
        if (logsWrapper.value) {
          logsWrapper.value.scrollTop = logsWrapper.value.scrollHeight
        } else {
        }
      }, 100)
    }
  } catch (error: any) {
    ElMessage.error(`获取日志失败: ${error.response?.data?.message || error.message}`)
  } finally {
    logsLoading.value = false
  }
}

// 刷新日志
const handleRefreshLogs = () => {
  handleLoadLogs()
}

// 下载日志
const handleDownloadLogs = () => {
  const { pod, container } = logsData.value
  const blob = new Blob([logsContent.value], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `${pod}-${container}-${new Date().getTime()}.log`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
  ElMessage.success('日志下载成功')
}

// 监听日志内容变化，自动滚动到底部
watch(logsContent, () => {
  if (logsAutoScroll.value && logsWrapper.value) {
    // 使用 setTimeout 确保 DOM 完全渲染
    setTimeout(() => {
      if (logsWrapper.value) {
        logsWrapper.value.scrollTop = logsWrapper.value.scrollHeight
      }
    }, 100)
  }
})

// 监听对话框关闭，停止自动刷新
watch(logsDialogVisible, (newVal) => {
  if (!newVal) {
    stopLogsAutoRefresh()
  }
})

// 监听编辑对话框打开和命名空间变化，加载存储资源
watch(
  () => [editDialogVisible.value, editWorkloadData.value?.namespace],
  ([visible, namespace]) => {
    if (visible && namespace) {
      loadConfigMaps()
      loadSecrets()
      loadPVCs()
    }
  },
  { deep: true }
)

// 获取注解提示内容
const getAnnotationsTooltip = (annotations: Record<string, string>) => {
  return Object.entries(annotations).map(([k, v]) => `${k}: ${v}`).join('\n')
}

// 获取注解文本（只显示一行）
const getAnnotationsText = (annotations: Record<string, string>) => {
  const text = Object.entries(annotations).map(([k, v]) => `${k}: ${v}`).join(', ')
  if (text.length > 80) {
    return text.substring(0, 77) + '...'
  }
  return text
}

// 显示 YAML 编辑器
const handleShowYAML = async () => {
  if (!selectedWorkload.value) return

  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const name = selectedWorkload.value.name
    const namespace = selectedWorkload.value.namespace
    const type = selectedWorkload.value.type

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/workloads/${namespace}/${name}/yaml`,
      {
        params: { clusterId, type },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    // 后端返回的是 JSON 对象，需要转换为 YAML 字符串
    const jsonData = response.data.data?.items
    if (jsonData) {
      yamlContent.value = yaml.dump(jsonData, {
        indent: 2,
        lineWidth: -1,
        noRefs: true
      })
    } else {
      yamlContent.value = ''
    }

    yamlDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  } finally {
    yamlSaving.value = false
  }
}

// 保存 YAML
const handleSaveYAML = async () => {
  if (!selectedWorkload.value) return

  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const name = selectedWorkload.value.name
    const namespace = selectedWorkload.value.namespace
    const type = selectedWorkload.value.type

    await axios.put(
      `/api/v1/plugins/kubernetes/resources/workloads/${namespace}/${name}/yaml`,
      {
        clusterId,
        type,
        yaml: yamlContent.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    await loadWorkloads()
  } catch (error) {
    ElMessage.error('保存 YAML 失败')
  } finally {
    yamlSaving.value = false
  }
}

// YAML编辑器输入处理
const handleYamlInput = () => {
  // 输入时自动调整滚动
}

// YAML编辑器滚动处理（同步行号滚动）
const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

// 重启工作负载
const handleRestart = async () => {
  if (!selectedWorkload.value) return

  try {
    await ElMessageBox.confirm(
      `确定要重启工作负载 ${selectedWorkload.value.name} 吗？`,
      '重启确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'black-button'
      }
    )

    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/workloads/${selectedWorkload.value.namespace}/${selectedWorkload.value.name}/restart`,
      {
        clusterId: selectedClusterId.value,
        type: selectedWorkload.value.type
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('工作负载重启成功')
    await loadWorkloads()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`重启失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 扩缩容工作负载
const handleScale = async () => {
  if (!selectedWorkload.value) return

  try {
    const { value } = await ElMessageBox.prompt(
      `请输入 ${selectedWorkload.value.name} 的副本数：`,
      '扩缩容',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputValue: selectedWorkload.value.desiredPods?.toString() || '1',
        confirmButtonClass: 'black-button'
      }
    )

    const replicas = parseInt(value)
    if (isNaN(replicas) || replicas < 0) {
      ElMessage.error('请输入有效的副本数')
      return
    }

    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/workloads/${selectedWorkload.value.namespace}/${selectedWorkload.value.name}/scale`,
      {
        clusterId: selectedClusterId.value,
        type: selectedWorkload.value.type,
        replicas
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('扩缩容成功')
    await loadWorkloads()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`扩缩容失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 显示编辑对话框
const handleShowEditDialog = async () => {
  if (!selectedWorkload.value) return

  editSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const workloadType = selectedWorkload.value.type
    const name = selectedWorkload.value.name
    const namespace = selectedWorkload.value.namespace

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/workloads/${namespace}/${name}/yaml`,
      {
        params: { clusterId, type: workloadType },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    // 获取返回的 JSON 数据
    const workloadData = response.data.data?.items
    if (workloadData) {

      // CronJob 的数据路径不同，需要特殊处理
      const isCronJob = workloadType === 'CronJob'
      const templateSpec = isCronJob
        ? workloadData.spec?.jobTemplate?.spec?.template?.spec
        : workloadData.spec?.template?.spec


      // 转换 nodeSelector 为 matchRules 格式
      const nodeSelector = templateSpec?.nodeSelector || {}

      const matchRules = Object.entries(nodeSelector).map(([key, value]) => {
        // 如果值是布尔值 true，则是 Exists 操作符
        if (value === true) {
          return {
            key,
            operator: 'Exists',
            value: ''
          }
        }
        // 否则是 In 操作符
        return {
          key,
          operator: 'In',
          value: String(value)
        }
      })


      // 解析 DNS 配置 - 只有当后端有配置时才设置
      let parsedDnsConfig = undefined
      if (templateSpec?.dnsConfig) {
        parsedDnsConfig = {
          nameservers: templateSpec.dnsConfig.nameservers || [],
          searches: templateSpec.dnsConfig.searches || [],
          options: (templateSpec.dnsConfig.options || []).map((opt: any) => ({
            name: opt.name || '',
            value: opt.value || ''
          }))
        }
      }

      // 转换数据格式以适应组件
      const calculatedSchedulingType = templateSpec?.nodeName ? 'specified' :
                                        (Object.keys(nodeSelector).length > 0 ? 'match' : 'any')


      editWorkloadData.value = {
        name: workloadData.metadata?.name || name,
        namespace: workloadData.metadata?.namespace || namespace,
        type: workloadData.kind || workloadType,
        replicas: workloadData.spec?.replicas || 0,
        labels: objectToKeyValueArray(workloadData.metadata?.labels || {}),
        annotations: objectToKeyValueArray(workloadData.metadata?.annotations || {}),
        nodeSelector: nodeSelector,
        nodeName: templateSpec?.nodeName || '',
        specifiedNode: templateSpec?.nodeName || '',
        schedulingType: calculatedSchedulingType,
        matchRules: matchRules,
        affinity: templateSpec?.affinity || {},
        tolerations: templateSpec?.tolerations || [],
        containers: parseContainers(templateSpec?.containers || []),
        initContainers: parseContainers(templateSpec?.initContainers || []),
        volumes: parseVolumesFromKubernetes(templateSpec?.volumes || []),
        hostNetwork: templateSpec?.hostNetwork,
        dnsPolicy: templateSpec?.dnsPolicy || 'ClusterFirst',
        hostname: templateSpec?.hostname,
        subdomain: templateSpec?.subdomain,
        dnsConfig: parsedDnsConfig,
        priorityClassName: templateSpec?.priorityClassName,
        terminationGracePeriodSeconds: templateSpec?.terminationGracePeriodSeconds || 30,
        // activeDeadlineSeconds 对于 Job/CronJob 应该从 jobTemplate.spec 读取，而不是从 template.spec
        activeDeadlineSeconds: isCronJob
          ? (workloadData.spec?.jobTemplate?.spec?.activeDeadlineSeconds || null)
          : (workloadType === 'Job' ? (workloadData.spec?.activeDeadlineSeconds || null) : templateSpec?.activeDeadlineSeconds),
        serviceAccountName: templateSpec?.serviceAccountName || 'default',
        // 根据工作负载类型设置正确的重启策略默认值
        restartPolicy: templateSpec?.restartPolicy ||
          ((workloadType === 'Job' || workloadType === 'CronJob') ? 'OnFailure' : 'Always')
      }

      // 解析亲和性规则
      affinityRules.value = parseAffinityRules(templateSpec?.affinity || {})
      editingAffinityRule.value = null

      // 解析扩缩容策略
      const strategy = workloadData.spec?.strategy || {}
      const rollingParams = strategy.rollingUpdate || {}
      scalingStrategyData.value = {
        strategyType: strategy.type || 'RollingUpdate',
        maxSurge: rollingParams.maxSurge !== undefined ? rollingParams.maxSurge : '25%',
        maxUnavailable: rollingParams.maxUnavailable !== undefined ? rollingParams.maxUnavailable : '25%',
        minReadySeconds: workloadData.spec?.minReadySeconds ?? 0,
        progressDeadlineSeconds: workloadData.spec?.progressDeadlineSeconds ?? 600,
        revisionHistoryLimit: workloadData.spec?.revisionHistoryLimit ?? 10,
        timeoutSeconds: 600
      }

      // 解析 Job 配置（Job 类型）
      if (workloadType === 'Job' && workloadData.spec) {
        jobConfig.value = {
          completions: workloadData.spec.completions || 1,
          parallelism: workloadData.spec.parallelism || 1,
          backoffLimit: workloadData.spec.backoffLimit || 6,
          activeDeadlineSeconds: workloadData.spec.activeDeadlineSeconds || null,
        }
      }

      // 解析 CronJob 配置（CronJob 类型）
      if (workloadType === 'CronJob' && workloadData.spec) {
        cronJobConfig.value = {
          schedule: workloadData.spec.schedule || '0 * * * *',
          concurrencyPolicy: workloadData.spec.concurrencyPolicy || 'Allow',
          timeZone: workloadData.spec.timeZone || '',
          successfulJobsHistoryLimit: workloadData.spec.successfulJobsHistoryLimit || 3,
          failedJobsHistoryLimit: workloadData.spec.failedJobsHistoryLimit || 1,
          startingDeadlineSeconds: workloadData.spec.startingDeadlineSeconds || null,
          suspend: workloadData.spec.suspend || false,
        }

        // 解析 CronJob 的 Job 配置
        const jobSpec = workloadData.spec.jobTemplate?.spec
        if (jobSpec) {
          jobConfig.value = {
            completions: jobSpec.completions || 1,
            parallelism: jobSpec.parallelism || 1,
            backoffLimit: jobSpec.backoffLimit || 6,
            activeDeadlineSeconds: jobSpec.activeDeadlineSeconds || null,
          }
        }
      }

      // 加载节点列表
      await loadNodes()

      isCreateMode.value = false
      editDialogVisible.value = true
    } else {
      ElMessage.warning('未获取到工作负载数据')
    }
  } catch (error: any) {
    ElMessage.error(`获取工作负载详情失败: ${error.response?.data?.message || error.message}`)
  } finally {
    editSaving.value = false
  }
}

// 将对象转换为键值对数组
const objectToKeyValueArray = (obj: Record<string, any>): { key: string; value: string }[] => {
  return Object.entries(obj).map(([key, value]) => ({
    key,
    value: String(value)
  }))
}

// 解析 Kubernetes Volumes 数据
const parseVolumesFromKubernetes = (volumes: any[]): any[] => {
  if (!volumes || !Array.isArray(volumes)) return []

  return volumes.map(volume => {
    const base = { name: volume.name }

    if (volume.emptyDir) {
      return {
        ...base,
        type: 'emptyDir',
        medium: volume.emptyDir.medium || '',
        sizeLimit: volume.emptyDir.sizeLimit || ''
      }
    }
    if (volume.hostPath) {
      return {
        ...base,
        type: 'hostPath',
        hostPath: {
          path: volume.hostPath.path || '',
          type: volume.hostPath.type || ''
        }
      }
    }
    if (volume.nfs) {
      return {
        ...base,
        type: 'nfs',
        nfs: {
          server: volume.nfs.server || '',
          path: volume.nfs.path || '',
          readOnly: volume.nfs.readOnly || false
        }
      }
    }
    if (volume.persistentVolumeClaim) {
      return {
        ...base,
        type: 'persistentVolumeClaim',
        persistentVolumeClaim: {
          claimName: volume.persistentVolumeClaim.claimName || '',
          readOnly: volume.persistentVolumeClaim.readOnly || false
        }
      }
    }
    if (volume.configMap) {
      return {
        ...base,
        type: 'configMap',
        configMap: {
          name: volume.configMap.name || '',
          defaultMode: volume.configMap.defaultMode,
          items: volume.configMap.items || []
        }
      }
    }
    if (volume.secret) {
      return {
        ...base,
        type: 'secret',
        secret: {
          secretName: volume.secret.secretName || '',
          defaultMode: volume.secret.defaultMode,
          items: volume.secret.items || []
        }
      }
    }

    return { ...base, type: 'unknown' }
  })
}

// 解析亲和性规则
const parseAffinityRules = (affinity: any): any[] => {
  const rules: any[] = []

  if (!affinity) return rules

  // Node Affinity
  if (affinity.nodeAffinity) {
    const nodeAff = affinity.nodeAffinity
    // Required
    if (nodeAff.requiredDuringSchedulingIgnoredDuringExecution) {
      const matchExpressions = nodeAff.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms?.[0]?.matchExpressions || []
      rules.push({
        type: 'nodeAffinity',
        priority: 'Required',
        weight: undefined,
        matchExpressions: matchExpressions.map((exp: any) => ({
          key: exp.key,
          operator: exp.operator,
          valueStr: exp.values?.join(',') || ''
        })),
        matchLabels: []
      })
    }
    // Preferred
    if (nodeAff.preferredDuringSchedulingIgnoredDuringExecution) {
      nodeAff.preferredDuringSchedulingIgnoredDuringExecution.forEach((pref: any) => {
        const matchExpressions = pref.preference.matchExpressions || []
        rules.push({
          type: 'nodeAffinity',
          priority: 'Preferred',
          weight: pref.weight,
          matchExpressions: matchExpressions.map((exp: any) => ({
            key: exp.key,
            operator: exp.operator,
            valueStr: exp.values?.join(',') || ''
          })),
          matchLabels: []
        })
      })
    }
  }

  // Pod Affinity
  if (affinity.podAffinity) {
    const podAff = affinity.podAffinity
    // Required
    if (podAff.requiredDuringSchedulingIgnoredDuringExecution) {
      podAff.requiredDuringSchedulingIgnoredDuringExecution.forEach((rule: any) => {
        rules.push({
          type: 'podAffinity',
          priority: 'Required',
          namespaces: rule.labelSelector?.matchLabels ? Object.keys(rule.labelSelector.matchLabels) : [],
          matchExpressions: rule.labelSelector?.matchExpressions?.map((exp: any) => ({
            key: exp.key,
            operator: exp.operator,
            valueStr: exp.values?.join(',') || ''
          })) || [],
          matchLabels: rule.labelSelector?.matchLabels ? Object.entries(rule.labelSelector.matchLabels).map(([k, v]) => ({ key: k, value: v })) : [],
          weight: undefined
        })
      })
    }
    // Preferred
    if (podAff.preferredDuringSchedulingIgnoredDuringExecution) {
      podAff.preferredDuringSchedulingIgnoredDuringExecution.forEach((pref: any) => {
        rules.push({
          type: 'podAffinity',
          priority: 'Preferred',
          weight: pref.weight,
          namespaces: pref.podAffinityTerm?.labelSelector?.matchLabels ? Object.keys(pref.podAffinityTerm.labelSelector.matchLabels) : [],
          matchExpressions: pref.podAffinityTerm?.labelSelector?.matchExpressions?.map((exp: any) => ({
            key: exp.key,
            operator: exp.operator,
            valueStr: exp.values?.join(',') || ''
          })) || [],
          matchLabels: pref.podAffinityTerm?.labelSelector?.matchLabels ? Object.entries(pref.podAffinityTerm.labelSelector.matchLabels).map(([k, v]) => ({ key: k, value: v })) : [],
        })
      })
    }
  }

  // Pod Anti-Affinity
  if (affinity.podAntiAffinity) {
    const podAntiAff = affinity.podAntiAffinity
    // Required
    if (podAntiAff.requiredDuringSchedulingIgnoredDuringExecution) {
      podAntiAff.requiredDuringSchedulingIgnoredDuringExecution.forEach((rule: any) => {
        rules.push({
          type: 'podAntiAffinity',
          priority: 'Required',
          namespaces: rule.labelSelector?.matchLabels ? Object.keys(rule.labelSelector.matchLabels) : [],
          matchExpressions: rule.labelSelector?.matchExpressions?.map((exp: any) => ({
            key: exp.key,
            operator: exp.operator,
            valueStr: exp.values?.join(',') || ''
          })) || [],
          matchLabels: rule.labelSelector?.matchLabels ? Object.entries(rule.labelSelector.matchLabels).map(([k, v]) => ({ key: k, value: v })) : [],
          weight: undefined
        })
      })
    }
    // Preferred
    if (podAntiAff.preferredDuringSchedulingIgnoredDuringExecution) {
      podAntiAff.preferredDuringSchedulingIgnoredDuringExecution.forEach((pref: any) => {
        rules.push({
          type: 'podAntiAffinity',
          priority: 'Preferred',
          weight: pref.weight,
          namespaces: pref.podAffinityTerm?.labelSelector?.matchLabels ? Object.keys(pref.podAffinityTerm.labelSelector.matchLabels) : [],
          matchExpressions: pref.podAffinityTerm?.labelSelector?.matchExpressions?.map((exp: any) => ({
            key: exp.key,
            operator: exp.operator,
            valueStr: exp.values?.join(',') || ''
          })) || [],
          matchLabels: pref.podAffinityTerm?.labelSelector?.matchLabels ? Object.entries(pref.podAffinityTerm.labelSelector.matchLabels).map(([k, v]) => ({ key: k, value: v })) : [],
        })
      })
    }
  }

  return rules
}

// 添加亲和性规则
const handleStartAddAffinity = (type: 'pod' | 'node') => {
  const isPod = type === 'pod'
  editingAffinityRule.value = {
    type: isPod ? 'podAffinity' : 'nodeAffinity',
    namespaces: [],
    topologyKey: isPod ? 'kubernetes.io/hostname' : undefined,
    priority: 'Required',
    weight: 50,
    matchExpressions: [],
    matchLabels: []
  }

  // 滚动到配置区域
  nextTick(() => {
    const configContainer = document.querySelector('.affinity-config-container')
    if (configContainer) {
      configContainer.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  })
}

// 取消编辑亲和性
const handleCancelAffinityEdit = () => {
  editingAffinityRule.value = null
}

// 保存亲和性规则
const handleSaveAffinityRule = () => {
  if (!editingAffinityRule.value) return

  // 验证 Pod 亲和性的拓扑键
  if (editingAffinityRule.value.type.includes('pod') && !editingAffinityRule.value.topologyKey) {
    ElMessage.warning('Pod 亲和性必须指定拓扑键')
    return
  }

  // 验证必填字段
  if (editingAffinityRule.value.matchExpressions.length === 0 &&
      editingAffinityRule.value.matchLabels.length === 0) {
    ElMessage.warning('请至少添加一个匹配表达式或标签')
    return
  }

  affinityRules.value.push({ ...editingAffinityRule.value })
  editingAffinityRule.value = null
  ElMessage.success('亲和性规则添加成功')
}

// 添加匹配表达式
const handleAddMatchExpression = () => {
  if (!editingAffinityRule.value) return
  editingAffinityRule.value.matchExpressions.push({
    key: '',
    operator: 'In',
    valueStr: ''
  })
}

// 删除匹配表达式
const handleRemoveMatchExpression = (index: number) => {
  if (!editingAffinityRule.value) return
  editingAffinityRule.value.matchExpressions.splice(index, 1)
}

// 添加匹配标签
const handleAddMatchLabel = () => {
  if (!editingAffinityRule.value) return
  editingAffinityRule.value.matchLabels.push({
    key: '',
    value: ''
  })
}

// 删除匹配标签
const handleRemoveMatchLabel = (index: number) => {
  if (!editingAffinityRule.value) return
  editingAffinityRule.value.matchLabels.splice(index, 1)
}

// 删除亲和性规则
const handleRemoveAffinityRule = (index: number) => {
  affinityRules.value.splice(index, 1)
  ElMessage.success('亲和性规则删除成功')
}

// 添加容忍度
const handleAddToleration = () => {
  if (!editWorkloadData.value) return
  if (!editWorkloadData.value.tolerations) {
    editWorkloadData.value.tolerations = []
  }
  editWorkloadData.value.tolerations.push({
    key: '',
    operator: 'Equal',
    value: '',
    effect: 'NoSchedule',
    tolerationSeconds: ''
  })
}

// 删除容忍度
const handleRemoveToleration = (index: number) => {
  if (!editWorkloadData.value?.tolerations) return
  editWorkloadData.value.tolerations.splice(index, 1)
}

// DNS 配置处理方法
const handleAddDNSNameserver = () => {
  if (!editWorkloadData.value) return
  if (!editWorkloadData.value.dnsConfig) {
    editWorkloadData.value.dnsConfig = { nameservers: [], searches: [], options: [] }
  }
  editWorkloadData.value.dnsConfig.nameservers.push('')
}

const handleRemoveDNSNameserver = (index: number) => {
  if (!editWorkloadData.value?.dnsConfig?.nameservers) return
  editWorkloadData.value.dnsConfig.nameservers.splice(index, 1)
}

const handleAddDNSSearch = () => {
  if (!editWorkloadData.value) return
  if (!editWorkloadData.value.dnsConfig) {
    editWorkloadData.value.dnsConfig = { nameservers: [], searches: [], options: [] }
  }
  editWorkloadData.value.dnsConfig.searches.push('')
}

const handleRemoveDNSSearch = (index: number) => {
  if (!editWorkloadData.value?.dnsConfig?.searches) return
  editWorkloadData.value.dnsConfig.searches.splice(index, 1)
}

const handleAddDNSOption = () => {
  if (!editWorkloadData.value) return
  if (!editWorkloadData.value.dnsConfig) {
    editWorkloadData.value.dnsConfig = { nameservers: [], searches: [], options: [] }
  }
  editWorkloadData.value.dnsConfig.options.push({ name: '', value: '' })
}

const handleRemoveDNSOption = (index: number) => {
  if (!editWorkloadData.value?.dnsConfig?.options) return
  editWorkloadData.value.dnsConfig.options.splice(index, 1)
}

// 将前端数据转换为 Kubernetes YAML 格式
const convertToKubernetesYaml = (data: any, cluster: string, namespace: string): string => {
  const kindMap: Record<string, string> = {
    'Deployment': 'Deployment',
    'StatefulSet': 'StatefulSet',
    'DaemonSet': 'DaemonSet',
    'Job': 'Job',
    'CronJob': 'CronJob'
  }

  const kind = kindMap[data.type] || data.type
  const apiVersion = data.type === 'CronJob' ? 'batch/v1' : 'apps/v1'

  // 构建 labels
  const labels: Record<string, string> = {}
  if (data.labels) {
    data.labels.forEach((l: any) => {
      if (l.key) labels[l.key] = l.value
    })
  }

  // 构建 annotations
  const annotations: Record<string, string> = {}
  if (data.annotations) {
    data.annotations.forEach((a: any) => {
      if (a.key) annotations[a.key] = a.value
    })
  }

  // 构建 affinity
  const affinity = buildAffinityFromRules(affinityRules.value)

  // 构建 tolerations
  const tolerations = (data.tolerations || []).map((t: any) => {
    const toleration: any = {
      key: t.key,
      operator: t.operator,
      effect: t.effect
    }
    if (t.operator === 'Equal' && t.value) {
      toleration.value = t.value
    }
    if (t.effect === 'NoExecute' && t.tolerationSeconds) {
      toleration.tolerationSeconds = parseInt(t.tolerationSeconds)
    }
    return toleration
  })

  // 构建 volumes
  const volumes = (data.volumes || []).map((v: any) => {
    const volume: any = { name: v.name }
    if (v.type === 'emptyDir') {
      volume.emptyDir = {}
      if (v.medium) volume.emptyDir.medium = v.medium
      if (v.sizeLimit) volume.emptyDir.sizeLimit = v.sizeLimit
    } else if (v.type === 'hostPath' && v.hostPath) {
      volume.hostPath = {
        path: v.hostPath.path,
        type: v.hostPath.type || ''
      }
    } else if (v.type === 'nfs' && v.nfs) {
      volume.nfs = {
        server: v.nfs.server,
        path: v.nfs.path,
        readOnly: v.nfs.readOnly || false
      }
    } else if (v.type === 'configMap' && v.configMap) {
      const configMap: any = { name: v.configMap.name }
      if (v.configMap.defaultMode) configMap.defaultMode = v.configMap.defaultMode
      if (v.configMap.items && v.configMap.items.length > 0) {
        configMap.items = v.configMap.items
      }
      volume.configMap = configMap
    } else if (v.type === 'secret' && v.secret) {
      const secret: any = { secretName: v.secret.secretName }
      if (v.secret.defaultMode) secret.defaultMode = v.secret.defaultMode
      if (v.secret.items && v.secret.items.length > 0) {
        secret.items = v.secret.items
      }
      volume.secret = secret
    } else if (v.type === 'persistentVolumeClaim' && v.persistentVolumeClaim) {
      volume.persistentVolumeClaim = {
        claimName: v.persistentVolumeClaim.claimName,
        readOnly: v.persistentVolumeClaim.readOnly || false
      }
    }
    return volume
  })

  // 构建 containers
  const containers = (data.containers || []).map((c: any) => buildContainer(c, volumes))

  // 构建 initContainers
  const initContainers = (data.initContainers || []).map((c: any) => buildContainer(c, volumes))

  // 构建 pod template spec
  // 根据工作负载类型设置正确的 restartPolicy

  let restartPolicy = 'Always'  // 默认值
  if (data.type === 'Job' || data.type === 'CronJob') {
    // 如果用户明确设置了值，使用用户的值；否则使用默认值 OnFailure
    restartPolicy = (data.restartPolicy && data.restartPolicy !== '') ? data.restartPolicy : 'OnFailure'
  } else if (data.type === 'Pod') {
    restartPolicy = (data.restartPolicy && data.restartPolicy !== '') ? data.restartPolicy : 'Always'
  }
  // Deployment/StatefulSet/DaemonSet 使用 Always


  const podSpec: any = {
    containers,
    restartPolicy,
    dnsPolicy: data.dnsPolicy || 'ClusterFirst',
    serviceAccountName: data.serviceAccountName || 'default',
    terminationGracePeriodSeconds: data.terminationGracePeriodSeconds || 30
  }

  // 添加可选的 Pod 级别字段
  // 注意：对于编辑模式，需要明确发送这些字段来覆盖旧值，即使值是 "假" 值
  // 使用 !== undefined 而不是直接判断真值，以确保 false 和空字符串也能被发送
  if (data.hostNetwork !== undefined) {
    podSpec.hostNetwork = data.hostNetwork
  }
  if (data.hostname !== undefined) {
    // 空字符串需要转换为 null 来删除字段
    podSpec.hostname = data.hostname || null
  }
  if (data.subdomain !== undefined) {
    // 空字符串需要转换为 null 来删除字段
    podSpec.subdomain = data.subdomain || null
  }
  if (data.automountServiceAccountToken !== undefined) {
    podSpec.automountServiceAccountToken = data.automountServiceAccountToken
  }
  if (data.priorityClassName !== undefined) {
    // 空字符串需要转换为 null 来删除字段
    const value = data.priorityClassName || null
    podSpec.priorityClassName = value
  }

  // DNS 配置 - 明确处理删除情况
  // 如果 dnsConfig 存在，检查是否有内容
  if (data.dnsConfig !== undefined) {
    const hasContent = (data.dnsConfig.nameservers?.length > 0 || data.dnsConfig.searches?.length > 0 || data.dnsConfig.options?.length > 0)
    if (hasContent) {
      // 有内容，设置完整的 dnsConfig
      podSpec.dnsConfig = {
        nameservers: data.dnsConfig.nameservers,
        searches: data.dnsConfig.searches,
        options: data.dnsConfig.options
      }
    } else {
      // 没有内容，明确设置为 null 来删除配置
      // 注意：需要检查是否是编辑模式（有原始资源）
      // 对于 StrategicMergePatch，设置为 null 会删除字段
      podSpec.dnsConfig = null
    }
  }

  if (initContainers.length > 0) {
    podSpec.initContainers = initContainers
  }

  if (volumes.length > 0) {
    podSpec.volumes = volumes
  }

  if (affinity && Object.keys(affinity).length > 0) {
    podSpec.affinity = affinity
  }

  // 总是设置 tolerations，包括空数组，以确保删除旧的容忍度
  podSpec.tolerations = tolerations

  // 明确删除 Pod 级别的 securityContext（包括 sysctls 等可能导致问题的配置）
  // 通过设置为 null 来确保删除旧配置
  podSpec.securityContext = null

  // 处理调度类型 - 关键：先完全删除调度相关字段，然后根据类型重新设置
  delete podSpec.nodeName
  delete podSpec.nodeSelector


  if (data.schedulingType === 'specified' && data.specifiedNode) {
    // 指定节点 - 明确设置 nodeName
    podSpec.nodeName = data.specifiedNode
  } else if (data.schedulingType === 'match') {
    // 调度规则匹配 - 构建 nodeSelector
    const nodeSelector: Record<string, any> = {}
    if (data.matchRules && data.matchRules.length > 0) {
      data.matchRules.forEach((rule: any) => {
        if (rule.key) {
          if (rule.operator === 'In' || rule.operator === 'NotIn') {
            if (rule.value) {
              const values = rule.value.split(',').map((v: string) => v.trim()).filter((v: string) => v)
              if (values.length > 0) {
                nodeSelector[rule.key] = values.length === 1 ? values[0] : values
              }
            }
          } else if (rule.operator === 'Exists') {
            nodeSelector[rule.key] = true
          }
        }
      })
    }

    if (Object.keys(nodeSelector).length > 0) {
      podSpec.nodeSelector = nodeSelector
    } else {
    }
  } else {
    // 任意可用节点 - 明确设置为 null 以删除 Kubernetes 中的字段
    podSpec.nodeName = null
    podSpec.nodeSelector = null
  }

  // 构建 Pod template
  const podTemplate = {
    metadata: {
      labels
    },
    spec: podSpec
  }


  // 构建 metadata
  const metadata: any = {
    name: data.name,
    namespace,
    labels
  }

  if (Object.keys(annotations).length > 0) {
    metadata.annotations = annotations
  }

  // 根据类型构建不同的 spec
  let spec: any = {}

  if (data.type === 'Deployment' || data.type === 'StatefulSet') {
    // Deployment 或 StatefulSet spec
    spec = {
      replicas: data.replicas || 1,
      selector: {
        matchLabels: { app: labels.app || data.name }
      },
      template: podTemplate
    }

    // 添加扩缩容策略
    if (data.strategyType) {
      const strategy: any = {
        type: data.strategyType
      }

      if (data.strategyType === 'RollingUpdate') {
        strategy.rollingUpdate = {}
        if (data.maxSurge) strategy.rollingUpdate.maxSurge = data.maxSurge
        if (data.maxUnavailable) strategy.rollingUpdate.maxUnavailable = data.maxUnavailable
      }

      spec.strategy = strategy
    }

    if (data.minReadySeconds) {
      spec.minReadySeconds = data.minReadySeconds
    }

    if (data.progressDeadlineSeconds) {
      spec.progressDeadlineSeconds = data.progressDeadlineSeconds
    }

    if (data.revisionHistoryLimit) {
      spec.revisionHistoryLimit = data.revisionHistoryLimit
    }

    // StatefulSet 没有特殊的spec字段，serviceAccountName 在 podSpec 中
  } else if (data.type === 'DaemonSet') {
    // DaemonSet spec
    spec = {
      selector: {
        matchLabels: { app: labels.app || data.name }
      },
      template: podTemplate
    }
  } else if (data.type === 'Job') {
    // Job spec
    spec = {
      template: podTemplate
    }

    // 添加 Job 配置
    if (jobConfig.value.completions !== undefined) {
      spec.completions = jobConfig.value.completions
    }
    if (jobConfig.value.parallelism !== undefined) {
      spec.parallelism = jobConfig.value.parallelism
    }
    if (jobConfig.value.backoffLimit !== undefined && jobConfig.value.backoffLimit !== null) {
      spec.backoffLimit = jobConfig.value.backoffLimit
    }
    if (jobConfig.value.activeDeadlineSeconds !== undefined && jobConfig.value.activeDeadlineSeconds !== null) {
      spec.activeDeadlineSeconds = jobConfig.value.activeDeadlineSeconds
    }

    // Job 默认不自动清理
    spec.ttlSecondsAfterFinished = null
  } else if (data.type === 'CronJob') {
    // CronJob spec
    const jobSpec: any = {
      template: podTemplate
    }

    // 添加 Job 配置到 jobTemplate
    if (jobConfig.value.completions !== undefined) {
      jobSpec.completions = jobConfig.value.completions
    }
    if (jobConfig.value.parallelism !== undefined) {
      jobSpec.parallelism = jobConfig.value.parallelism
    }
    if (jobConfig.value.backoffLimit !== undefined && jobConfig.value.backoffLimit !== null) {
      jobSpec.backoffLimit = jobConfig.value.backoffLimit
    }
    if (jobConfig.value.activeDeadlineSeconds !== undefined && jobConfig.value.activeDeadlineSeconds !== null) {
      jobSpec.activeDeadlineSeconds = jobConfig.value.activeDeadlineSeconds
    }

    spec = {
      schedule: cronJobConfig.value.schedule,
      concurrencyPolicy: cronJobConfig.value.concurrencyPolicy,
      successfulJobsHistoryLimit: cronJobConfig.value.successfulJobsHistoryLimit,
      failedJobsHistoryLimit: cronJobConfig.value.failedJobsHistoryLimit,
      jobTemplate: {
        spec: jobSpec
      }
    }

    if (cronJobConfig.value.timeZone !== undefined && cronJobConfig.value.timeZone !== '') {
      spec.timeZone = cronJobConfig.value.timeZone
    }
    if (cronJobConfig.value.startingDeadlineSeconds !== undefined && cronJobConfig.value.startingDeadlineSeconds !== null) {
      spec.startingDeadlineSeconds = cronJobConfig.value.startingDeadlineSeconds
    }
    if (cronJobConfig.value.suspend !== undefined) {
      spec.suspend = cronJobConfig.value.suspend
    }
  } else if (data.type === 'Pod') {
    // Pod 直接使用 podTemplate 的 spec
    spec = podSpec
  }

  // 构建完整的资源对象
  const resource: any = {
    apiVersion,
    kind,
    metadata,
    spec
  }

  // 转换为 YAML 字符串
  const yamlStr = yaml.dump(resource, { indent: 2, lineWidth: -1 })

  return yamlStr
}

// 构建容器对象
const buildContainer = (container: any, volumes: any[]): any => {
  const c: any = {
    name: container.name,
    image: container.image,
    imagePullPolicy: container.imagePullPolicy || 'IfNotPresent'
  }

  // command 和 args
  if (container.command && container.command.length > 0) {
    c.command = container.command
  }
  if (container.args && container.args.length > 0) {
    c.args = container.args
  }

  // workingDir
  if (container.workingDir) {
    c.workingDir = container.workingDir
  }

  // ports
  if (container.ports && container.ports.length > 0) {
    c.ports = container.ports.map((p: any) => {
      const port: any = {
        containerPort: p.containerPort,
        protocol: p.protocol || 'TCP'
      }
      if (p.name) port.name = p.name
      if (p.hostPort) port.hostPort = p.hostPort
      if (p.hostIP) port.hostIP = p.hostIP
      return port
    })
  }

  // env
  if (container.env && container.env.length > 0) {
    c.env = container.env.map((e: any) => {
      const env: any = { name: e.name }
      if (e.valueFrom === 'configmap') {
        env.valueFrom = {
          configMapKeyRef: {
            name: e.configmapName,
            key: e.key
          }
        }
      } else if (e.valueFrom === 'secret') {
        env.valueFrom = {
          secretKeyRef: {
            name: e.secretName,
            key: e.key
          }
        }
      } else if (e.valueFrom === 'field') {
        env.valueFrom = {
          fieldRef: {
            fieldPath: e.fieldPath
          }
        }
      } else if (e.valueFrom === 'resource') {
        env.valueFrom = {
          resourceFieldRef: {
            container: container.name,
            resource: e.resourceField,
            divisor: e.divisor || '1'
          }
        }
      } else {
        env.value = e.value
      }
      return env
    })
  }

  // resources
  if (container.resources) {
    const resources: any = {}
    if (container.resources.requests && (container.resources.requests.cpu || container.resources.requests.memory)) {
      resources.requests = {}
      if (container.resources.requests.cpu) resources.requests.cpu = container.resources.requests.cpu
      if (container.resources.requests.memory) resources.requests.memory = container.resources.requests.memory
    }
    if (container.resources.limits && (container.resources.limits.cpu || container.resources.limits.memory)) {
      resources.limits = {}
      if (container.resources.limits.cpu) resources.limits.cpu = container.resources.limits.cpu
      if (container.resources.limits.memory) resources.limits.memory = container.resources.limits.memory
    }
    if (Object.keys(resources).length > 0) {
      c.resources = resources
    }
  }

  // volumeMounts
  if (container.volumeMounts && container.volumeMounts.length > 0) {
    c.volumeMounts = container.volumeMounts.map((vm: any) => {
      const mount: any = {
        name: vm.name,
        mountPath: vm.mountPath
      }
      if (vm.subPath) mount.subPath = vm.subPath
      if (vm.readOnly) mount.readOnly = true
      return mount
    })
  }

  // lifecycle (postStart, preStop)
  if (container.postStart || container.preStop) {
    c.lifecycle = {}
    if (container.postStart) {
      c.lifecycle.postStart = {
        exec: {
          command: container.postStart
        }
      }
    }
    if (container.preStop) {
      c.lifecycle.preStop = {
        exec: {
          command: container.preStop
        }
      }
    }
  }

  // probes
  if (container.livenessProbe) {
    c.livenessProbe = buildProbe(container.livenessProbe)
  }
  if (container.readinessProbe) {
    c.readinessProbe = buildProbe(container.readinessProbe)
  }
  if (container.startupProbe) {
    c.startupProbe = buildProbe(container.startupProbe)
  }

  return c
}

// 构建 probe 对象
const buildProbe = (probe: any): any => {
  if (!probe || !probe.enabled) return null

  const p: any = {
    initialDelaySeconds: probe.initialDelaySeconds || 0,
    timeoutSeconds: probe.timeoutSeconds || 3,
    periodSeconds: probe.periodSeconds || 10,
    successThreshold: probe.successThreshold || 1,
    failureThreshold: probe.failureThreshold || 3
  }

  // 根据类型构建探针
  if (probe.type === 'httpGet') {
    p.httpGet = {
      path: probe.path || '/',
      port: probe.port || 80,
      scheme: probe.scheme || 'HTTP'
    }
    if (probe.httpHeaders && probe.httpHeaders.length > 0) {
      p.httpGet.httpHeaders = probe.httpHeaders
    }
  } else if (probe.type === 'tcpSocket') {
    p.tcpSocket = {
      port: probe.port || 80
    }
  } else if (probe.type === 'exec') {
    if (probe.command && probe.command.length > 0) {
      p.exec = {
        command: probe.command
      }
    }
  } else if (probe.type === 'grpc') {
    p.grpc = {
      port: probe.port || 80,
      service: probe.service || null
    }
  }

  return p
}

// 从亲和性规则构建 Kubernetes affinity 对象
const buildAffinityFromRules = (rules: any[]): any => {

  const affinity: any = {}

  for (const rule of rules) {
    if (rule.type === 'nodeAffinity') {
      if (!affinity.nodeAffinity) {
        affinity.nodeAffinity = {}
      }
      if (rule.priority === 'Required') {
        if (!affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution) {
          affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution = {
            nodeSelectorTerms: []
          }
        }
        const term = buildNodeSelectorTerm(rule)
        affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.push(term)
      } else {
        if (!affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution) {
          affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution.push({
          weight: rule.weight || 50,
          preference: buildNodeSelectorTerm(rule)
        })
      }
    } else if (rule.type === 'nodeAntiAffinity') {
      if (!affinity.nodeAffinity) {
        affinity.nodeAffinity = {}
      }
      if (rule.priority === 'Required') {
        if (!affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution) {
          affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution = {
            nodeSelectorTerms: []
          }
        }
        const term = buildNodeSelectorTerm(rule)
        affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.push(term)
      } else {
        if (!affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution) {
          affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution.push({
          weight: rule.weight || 50,
          preference: buildNodeSelectorTerm(rule)
        })
      }
    } else if (rule.type === 'podAffinity') {
      if (!affinity.podAffinity) {
        affinity.podAffinity = {}
      }
      const podAffinityTerm = buildPodAffinityTerm(rule)
      if (!podAffinityTerm) {
        continue
      }
      if (rule.priority === 'Required') {
        if (!affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution) {
          affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution.push(podAffinityTerm)
      } else {
        if (!affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution) {
          affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution.push({
          weight: rule.weight || 50,
          podAffinityTerm
        })
      }
    } else if (rule.type === 'podAntiAffinity') {
      if (!affinity.podAntiAffinity) {
        affinity.podAntiAffinity = {}
      }
      const podAffinityTerm = buildPodAffinityTerm(rule)
      if (!podAffinityTerm) {
        continue
      }
      if (rule.priority === 'Required') {
        if (!affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution) {
          affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution.push(podAffinityTerm)
      } else {
        if (!affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution) {
          affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.push({
          weight: rule.weight || 50,
          podAffinityTerm
        })
      }
    }
  }

  // 清理空对象
  if (affinity.nodeAffinity && Object.keys(affinity.nodeAffinity).length === 0) {
    delete affinity.nodeAffinity
  }

  if (affinity.podAffinity && Object.keys(affinity.podAffinity).length === 0) {
    delete affinity.podAffinity
  }

  if (affinity.podAntiAffinity && Object.keys(affinity.podAntiAffinity).length === 0) {
    delete affinity.podAntiAffinity
  }


  if (Object.keys(affinity).length === 0) return undefined
  return affinity
}

// 构建节点选择器条件
const buildNodeSelectorTerm = (rule: any): any => {
  const matchExpressions = (rule.matchExpressions || []).map((exp: any) => {
    const expression: any = {
      key: exp.key,
      operator: exp.operator
    }
    if (exp.operator !== 'Exists' && exp.operator !== 'DoesNotExist') {
      expression.values = exp.valueStr ? exp.valueStr.split(',').filter((v: string) => v) : []
    }
    return expression
  })

  // 添加 matchLabels
  const matchLabels: Record<string, string> = {}
  if (rule.matchLabels) {
    rule.matchLabels.forEach((l: any) => {
      if (l.key && l.value) matchLabels[l.key] = l.value
    })
  }

  const term: any = {}

  // 只有在有内容时才添加 matchExpressions
  if (matchExpressions.length > 0) {
    term.matchExpressions = matchExpressions
  }

  // 只有在有内容时才添加 matchLabels
  if (Object.keys(matchLabels).length > 0) {
    term.matchLabels = matchLabels
  }


  return term
}

// 构建 Pod 亲和性条件
const buildPodAffinityTerm = (rule: any): any => {

  const matchExpressions = (rule.matchExpressions || []).map((exp: any) => {
    const expression: any = {
      key: exp.key,
      operator: exp.operator
    }
    if (exp.operator !== 'Exists' && exp.operator !== 'DoesNotExist') {
      expression.values = exp.valueStr ? exp.valueStr.split(',').filter((v: string) => v) : []
    }
    return expression
  })

  // 添加 matchLabels
  const matchLabels: Record<string, string> = {}
  if (rule.matchLabels) {
    rule.matchLabels.forEach((l: any) => {
      if (l.key && l.value) matchLabels[l.key] = l.value
    })
  }

  const labelSelector: any = {}

  // 只有在有内容时才添加 matchExpressions
  if (matchExpressions.length > 0) {
    labelSelector.matchExpressions = matchExpressions
  }

  // 只有在有内容时才添加 matchLabels
  if (Object.keys(matchLabels).length > 0) {
    labelSelector.matchLabels = matchLabels
  }

  // 如果 labelSelector 为空，返回 null 以表示无效配置
  if (Object.keys(labelSelector).length === 0) {
    return null
  }

  const podAffinityTerm: any = {
    labelSelector,
    topologyKey: rule.topologyKey || 'kubernetes.io/hostname'
  }


  return podAffinityTerm
}

// 保存编辑
const handleSaveEdit = async () => {
  if (!editWorkloadData.value) return

  // 创建模式下不需要selectedWorkload
  if (!isCreateMode.value && !selectedWorkload.value) return

  editSaving.value = true

  try {
    const clusterName = selectedCluster.value?.name || ''
    const yaml = convertToKubernetesYaml(
      editWorkloadData.value,
      clusterName,
      editWorkloadData.value.namespace || 'default'
    )

    if (isCreateMode.value) {
      // 创建模式：调用创建API
      const token = localStorage.getItem('token')
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/workloads/create`,
        {
          clusterId: selectedClusterId.value,
          yaml: yaml  // 发送YAML字符串，不是JSON对象
        },
        { headers: { Authorization: `Bearer ${token}` } }
      )
      ElMessage.success('工作负载创建成功')
      isCreateMode.value = false
    } else {
      // 编辑模式：调用更新API
      await updateWorkload({
        cluster: clusterName,
        namespace: editWorkloadData.value.namespace || 'default',
        type: editWorkloadData.value.type,
        name: editWorkloadData.value.name,
        yaml
      })
      ElMessage.success('工作负载更新成功')
    }

    editDialogVisible.value = false

    // 重新加载列表
    await loadWorkloads()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || (isCreateMode.value ? '创建工作负载失败' : '更新工作负载失败'))
  } finally {
    editSaving.value = false
  }
}

// 解析容器数据
const parseContainers = (containers: any[]): any[] => {
  if (!containers || !Array.isArray(containers)) return []

  return containers.map(container => {
    // 解析环境变量
    let envs: any[] = []
    if (container.env) {
      for (const e of container.env) {
        if (e.valueFrom?.configMapKeyRef) {
          // ConfigMap 引用
          envs.push({
            name: e.name,
            configmapName: e.valueFrom.configMapKeyRef.name,
            key: e.valueFrom.configMapKeyRef.key,
            valueFrom: {
              type: 'configmap',
              configMapName: e.valueFrom.configMapKeyRef.name,
              key: e.valueFrom.configMapKeyRef.key
            }
          })
        } else if (e.valueFrom?.secretKeyRef) {
          // Secret 引用
          envs.push({
            name: e.name,
            secretName: e.valueFrom.secretKeyRef.name,
            key: e.valueFrom.secretKeyRef.key,
            valueFrom: {
              type: 'secret',
              secretName: e.valueFrom.secretKeyRef.name,
              key: e.valueFrom.secretKeyRef.key
            }
          })
        } else if (e.valueFrom?.fieldRef) {
          // Pod 字段引用
          envs.push({
            name: e.name,
            value: e.value || '',
            valueFrom: {
              type: 'fieldRef',
              fieldPath: e.valueFrom.fieldRef.fieldPath
            }
          })
        } else if (e.valueFrom?.resourceFieldRef) {
          // 资源字段引用
          envs.push({
            name: e.name,
            value: e.value || '',
            valueFrom: {
              type: 'resourceFieldRef',
              resource: e.valueFrom.resourceFieldRef.resource,
              containerName: e.valueFrom.resourceFieldRef.containerName,
              divisor: e.valueFrom.resourceFieldRef.divisor
            }
          })
        } else {
          // 普通变量
          envs.push({
            name: e.name,
            value: e.value || ''
          })
        }
      }
    }

    return {
      name: container.name || '',
      image: container.image || '',
      imagePullPolicy: container.imagePullPolicy || 'IfNotPresent',
      workingDir: container.workingDir || '',
      command: container.command || [],
      args: container.args || [],
      env: envs,
      resources: {
        requests: {
          cpu: container.resources?.requests?.cpu || '',
          memory: container.resources?.requests?.memory || ''
        },
        limits: {
          cpu: container.resources?.limits?.cpu || '',
          memory: container.resources?.limits?.memory || ''
        }
      },
      ports: (container.ports || []).map((p: any) => ({
        name: p.name || '',
        containerPort: p.containerPort || 0,
        protocol: p.protocol || 'TCP',
        hostPort: p.hostPort,
        hostIP: p.hostIP || ''
      })),
      volumeMounts: (container.volumeMounts || []).map((vm: any) => ({
        name: vm.name || '',
        mountPath: vm.mountPath || '',
        subPath: vm.subPath || '',
        readOnly: vm.readOnly || false
      })),

      // 解析探针配置
      livenessProbe: parseProbe(container.livenessProbe),
      readinessProbe: parseProbe(container.readinessProbe),
      startupProbe: parseProbe(container.startupProbe),

      stdin: container.stdin || false,
      tty: container.tty || false,
      activeTab: 'basic'
    }
  })
}

// 解析探针配置
const parseProbe = (probe: any): any => {
  if (!probe) return null

  const result: any = {
    enabled: true,
    type: 'httpGet',
    initialDelaySeconds: probe.initialDelaySeconds || 0,
    timeoutSeconds: probe.timeoutSeconds || 3,
    periodSeconds: probe.periodSeconds || 10,
    successThreshold: probe.successThreshold || 1,
    failureThreshold: probe.failureThreshold || 3
  }

  // 确定探针类型
  if (probe.httpGet) {
    result.type = 'httpGet'
    result.path = probe.httpGet.path || '/'
    result.port = probe.httpGet.port || 80
    result.scheme = probe.httpGet.scheme || 'HTTP'
    if (probe.httpGet.httpHeaders) {
      result.httpHeaders = probe.httpGet.httpHeaders
    }
  } else if (probe.tcpSocket) {
    result.type = 'tcpSocket'
    result.port = probe.tcpSocket.port || 80
  } else if (probe.exec) {
    result.type = 'exec'
    result.command = probe.exec.command || []
  }

  return result
}

// 更新容器列表
const updateContainers = (containers: any[]) => {
  if (editWorkloadData.value) {
    editWorkloadData.value.containers = containers
  }
}

// 更新初始化容器列表
const updateInitContainers = (initContainers: any[]) => {
  if (editWorkloadData.value) {
    editWorkloadData.value.initContainers = initContainers
  }
}

// 添加数据卷
const handleAddVolume = () => {
  if (!editWorkloadData.value) return
  if (!editWorkloadData.value.volumes) {
    editWorkloadData.value.volumes = []
  }
  editWorkloadData.value.volumes.push({
    name: '',
    type: 'emptyDir',
    medium: '',
    sizeLimit: ''
  })
}

// 删除数据卷
const handleRemoveVolume = (index: number) => {
  if (!editWorkloadData.value?.volumes) return
  editWorkloadData.value.volumes.splice(index, 1)
}

// 更新数据卷
const handleUpdateVolumes = (volumes: any[]) => {
  if (editWorkloadData.value) {
    editWorkloadData.value.volumes = volumes
  }
}

// 加载 ConfigMap 列表
const loadConfigMaps = async () => {
  if (!selectedClusterId.value || !editWorkloadData.value?.namespace) return

  try {
    const data = await getConfigMaps(selectedClusterId.value, editWorkloadData.value.namespace)
    configMaps.value = data || []
  } catch (error) {
  }
}

// 加载 Secret 列表
const loadSecrets = async () => {
  if (!selectedClusterId.value || !editWorkloadData.value?.namespace) return

  try {
    const data = await getSecrets(selectedClusterId.value, editWorkloadData.value.namespace)
    secrets.value = data || []
  } catch (error) {
  }
}

// 加载 PVC 列表
const loadPVCs = async () => {
  if (!selectedClusterId.value || !editWorkloadData.value?.namespace) return

  try {
    const data = await getPersistentVolumeClaims(selectedClusterId.value, editWorkloadData.value.namespace)
    pvcs.value = data || []
  } catch (error) {
  }
}

// 删除工作负载
const handleDelete = async () => {
  if (!selectedWorkload.value) return

  try {
    await ElMessageBox.confirm(
      `确定要删除工作负载 ${selectedWorkload.value.name} 吗？此操作不可恢复！`,
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error',
        confirmButtonClass: 'black-button'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/workloads/${selectedWorkload.value.namespace}/${selectedWorkload.value.name}`,
      {
        params: {
          clusterId: selectedClusterId.value,
          type: selectedWorkload.value.type
        },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('删除成功')
    await loadWorkloads()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 组件卸载时清理资源
onUnmounted(() => {
  if (terminalWebSocket) {
    terminalWebSocket.close()
    terminalWebSocket = null
  }
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  // 停止日志自动刷新
  stopLogsAutoRefresh()
})

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.workloads-container {
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
  background: #000;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
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

.search-icon {
  color: #d4af37;
}

.black-button {
  background: #000 !important;
  color: #fff !important;
  border: none !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 600;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.black-button:hover {
  background: #333 !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
}

/* 上下文选择栏 */
.context-bar {
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.context-selectors {
  display: flex;
  gap: 24px;
  align-items: center;
}

.context-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.context-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 500;
  color: #606266;
  white-space: nowrap;
}

.context-select {
  min-width: 200px;
}

/* 工作负载类型标签栏 */
.workload-types-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  flex-wrap: wrap;
}

.type-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: #1a1a1a;
  color: #fff;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  user-select: none;
  border: 2px solid #1a1a1a;
}

.type-tab:hover {
  background: #333;
  border-color: #333;
  transform: translateY(-1px);
}

.type-tab.active {
  background: #d4af37;
  color: #1a1a1a;
  border-color: #d4af37;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.4);
  font-weight: 600;
}

.type-tab.active .type-icon {
  color: #1a1a1a;
}

.type-icon {
  font-size: 18px;
  color: #fff;
}

.type-tab.active .type-icon {
  color: #1a1a1a;
}

.type-label {
  font-size: 14px;
  font-weight: 500;
}

.type-count {
  font-size: 12px;
  opacity: 0.8;
  margin-left: 2px;
}

/* 操作栏 */
.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.search-section {
  display: flex;
  gap: 12px;
  align-items: center;
  flex: 1;
}

.search-input {
  width: 280px;
}

.namespace-select {
  width: 200px;
}

.action-section {
  display: flex;
  gap: 12px;
  align-items: center;
}

.action-buttons {
  display: flex;
  gap: 12px;
}

/* 批量操作栏 */
.batch-action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  margin-bottom: 16px;
  background: #ffffff;
  border-radius: 12px;
  border: 1px solid #e0e0e0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  animation: slideDown 0.3s ease-out;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.batch-action-left {
  display: flex;
  align-items: center;
}

.selected-count {
  font-size: 15px;
  color: #1a1a1a;
  font-weight: 700;
  padding: 8px 16px;
  background: #ffffff;
  border: 2px solid #d4af37;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.2);
}

.selected-count::before {
  content: '';
  width: 10px;
  height: 10px;
  background: #d4af37;
  border-radius: 50%;
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
    box-shadow: 0 0 0 0 rgba(212, 175, 55, 0.7);
  }
  50% {
    opacity: 0.8;
    transform: scale(1.05);
    box-shadow: 0 0 0 6px rgba(212, 175, 55, 0);
  }
}

.batch-action-right {
  display: flex;
  gap: 10px;
  align-items: center;
}

.batch-btn {
  padding: 10px 18px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  border: 2px solid #1a1a1a;
  background: #1a1a1a;
  color: #ffffff;
}

.batch-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.25);
  background: #333333;
  border-color: #333333;
  color: #ffffff;
}

/* 统一所有按钮类型为黑底白字 */
.batch-btn.el-button--danger,
.batch-btn.el-button--warning,
.batch-btn.el-button--success,
.batch-btn.el-button--info,
.batch-btn.el-button--primary {
  background: #1a1a1a;
  border-color: #1a1a1a;
  color: #ffffff;
}

.batch-btn.el-button--danger:hover,
.batch-btn.el-button--warning:hover,
.batch-btn.el-button--success:hover,
.batch-btn.el-button--info:hover,
.batch-btn.el-button--primary:hover {
  background: #333333;
  border-color: #333333;
  color: #ffffff;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.25);
}

/* 选择列样式优化 */
.modern-table :deep(.el-table__header .el-table-column--selection .cell) {
  display: flex;
  justify-content: center;
  align-items: center;
}

.modern-table :deep(.el-table__body .el-table-column--selection .cell) {
  display: flex;
  justify-content: center;
  align-items: center;
}

/* 自定义复选框样式 */
.modern-table :deep(.el-checkbox__inner) {
  border-radius: 4px;
  border: 2px solid #d4af37;
  background: transparent;
  width: 18px;
  height: 18px;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.modern-table :deep(.el-checkbox__inner:hover) {
  border-color: #c9a227;
  box-shadow: 0 0 0 2px rgba(212, 175, 55, 0.15);
}

.modern-table :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background: #d4af37;
  border-color: #d4af37;
}

.modern-table :deep(.el-checkbox__input.is-indeterminate .el-checkbox__inner) {
  background: #d4af37;
  border-color: #d4af37;
}

.modern-table :deep(.el-checkbox__input.is-checked .el-checkbox__inner::after) {
  border-color: #ffffff;
  border-width: 2px;
}

.modern-table :deep(.el-checkbox__input.is-indeterminate .el-checkbox__inner::before) {
  background-color: #ffffff;
}

/* 选中行的样式 */
.modern-table :deep(.el-table__body tr.current-row) {
  background: rgba(24, 144, 255, 0.05) !important;
}

.modern-table :deep(.el-table__body tr:hover td) {
  background: rgba(24, 144, 255, 0.03) !important;
}

/* 创建按钮样式 */
.add-button {
  background: #1a1a1a !important;
  color: #fff !important;
  border: none !important;
  font-weight: 500;
  padding: 10px 20px;
  border-radius: 8px;
  transition: all 0.3s ease;
}

.add-button:hover {
  background: #333 !important;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.add-button:active {
  transform: translateY(0);
}

.add-button-form {
  background: #1a1a1a !important;
  color: #fff !important;
  border: none !important;
  font-weight: 500;
  padding: 10px 20px;
  border-radius: 8px;
  transition: all 0.3s ease;
}

.add-button-form:hover {
  background: #333 !important;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.add-button-form:active {
  transform: translateY(0);
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

.filter-select,
.cluster-select {
  width: 180px;
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

/* 工作负载名称单元格 */
.workload-name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.workload-icon-wrapper {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 2px 6px rgba(212, 175, 55, 0.25);
}

.workload-icon {
  color: #1a1a1a;
  font-size: 18px;
}

.workload-name-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.workload-name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.workload-type-icon-box {
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

.workload-type-icon-gold {
  color: #d4af37;
}

.workload-name {
  font-weight: 500;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 400px;
}

.golden-text {
  color: #d4af37 !important;
}

.clickable {
  cursor: pointer;
  transition: opacity 0.2s;
}

.clickable:hover {
  opacity: 0.7;
}

.workload-namespace {
  font-size: 12px;
  color: #909399;
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

/* Pod 数量 */
.pod-count-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.pod-count {
  font-size: 18px;
  font-weight: 600;
  color: #d4af37;
}

.pod-label {
  font-size: 11px;
  color: #909399;
}

/* 资源单元格 */
.resource-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.resource-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}

.resource-label {
  color: #909399;
  font-weight: 500;
  min-width: 45px;
}

.resource-value {
  color: #303133;
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 500;
}

.requests-value {
  color: #67c23a;
}

.limits-value {
  color: #e6a23c;
}

.resource-separator {
  color: #dcdfe6;
  margin: 0 4px;
}

.resource-empty {
  color: #909399;
}

/* 状态标签 */
.status-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.status-running {
  background: #f0f9ff;
  color: #1890ff;
}

.status-succeeded {
  background: #f6ffed;
  color: #52c41a;
}

.status-failed {
  background: #fff1f0;
  color: #ff4d4f;
}

.status-pending {
  background: #fffbe6;
  color: #faad14;
}

.status-unknown {
  background: #f5f5f5;
  color: #8c8c8c;
}

/* 其他错误状态的默认样式 */
.status-badge[class*="status-"]:not(.status-running):not(.status-succeeded):not(.status-failed):not(.status-pending):not(.status-unknown) {
  background: #fff1f0;
  color: #ff4d4f;
}

/* 常见错误状态 */
.status-imagepullbackoff,
.status-errimagepull,
.status-crashloopbackoff,
.status-oomkilled,
.status-error,
.status-containercannotrun,
.status-invalidimagename {
  background: #fff1f0;
  color: #ff4d4f;
}

/* Pod IP */
.pod-ip {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #606266;
}

/* 调度时间文本 */
.schedule-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #303133;
}

/* 镜像单元格 */
.image-cell {
  display: flex;
  align-items: center;
}

.image-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.image-item {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 11px;
  color: #606266;
  background: #f5f7fa;
  padding: 2px 8px;
  border-radius: 4px;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.image-more {
  font-size: 11px;
  color: #909399;
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  cursor: pointer;
}

.image-empty {
  color: #909399;
  font-size: 13px;
}

/* 镜像提示框样式 */
.image-tooltip-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-width: 500px;
}

.image-tooltip-item {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #303133;
  line-height: 1.5;
  word-break: break-all;
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

/* 操作按钮 */
.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 13px;
  color: #d4af37;
  padding: 4px;
}

.action-btn:hover {
  color: #bfa13f;
}

.action-btn.danger {
  color: #f56c6c;
}

.action-btn.danger:hover {
  color: #f78989;
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

/* 标签弹窗 */
.label-dialog :deep(.el-dialog__header) {
  background: #d4af37;
  color: #1a1a1a;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.label-dialog :deep(.el-dialog__title) {
  color: #1a1a1a;
  font-size: 16px;
  font-weight: 600;
}

.label-dialog-content {
  padding: 8px 0;
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
  background: rgba(212, 175, 55, 0.1) !important;
  color: #d4af37 !important;
  border: 1px solid #d4af37 !important;
  border-radius: 6px !important;
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 12px !important;
  font-weight: 600 !important;
  cursor: pointer !important;
  transition: all 0.3s !important;
  user-select: none;
}

.label-key-wrapper:hover {
  background: rgba(212, 175, 55, 0.2) !important;
  border-color: #c9a227 !important;
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
  color: #666;
  word-break: break-all;
  white-space: pre-wrap;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* YAML 编辑弹窗 */
.yaml-dialog :deep(.el-dialog__header) {
  background: #d4af37;
  color: #1a1a1a;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.yaml-dialog :deep(.el-dialog__title) {
  color: #1a1a1a;
  font-size: 16px;
  font-weight: 600;
}

.yaml-dialog :deep(.el-dialog__body) {
  padding: 24px;
  background-color: #ffffff;
}

.yaml-dialog-content {
  padding: 0;
}

/* 详情对话框样式 */
.detail-wrapper {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.basic-info-section {
  padding: 24px;
  background: linear-gradient(135deg, #f5f7fa 0%, #ffffff 100%);
  border-radius: 12px;
  border: 1px solid #e4e7ed;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.info-row {
  display: flex;
  gap: 32px;
  margin-bottom: 20px;
  align-items: flex-start;
}

.info-row:last-child {
  margin-bottom: 0;
}

.basic-info-section .info-item {
  flex: 1;
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  gap: 12px;
}

.basic-info-section .info-item.full-width {
  flex: 1;
}

.basic-info-section .info-label {
  min-width: 80px;
  font-size: 14px;
  color: #606266;
  font-weight: 600;
  white-space: nowrap;
  padding-top: 2px;
}

.basic-info-section .info-value {
  font-size: 14px;
  color: #303133;
  flex: 1;
  line-height: 1.6;
}

/* 镜像列表样式 */
.images-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}

.image-tag {
  padding: 8px 16px;
  background: linear-gradient(135deg, #e8f4fd 0%, #f5f9ff 100%);
  border: 1px solid #b3d8ff;
  border-radius: 6px;
  font-size: 13px;
  color: #409eff;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  word-break: break-all;
  transition: all 0.3s ease;
}

.image-tag:hover {
  background: linear-gradient(135deg, #d9ecff 0%, #e8f4fd 100%);
  border-color: #409eff;
  box-shadow: 0 2px 6px rgba(64, 158, 255, 0.2);
}

/* 标签列表样式 */
.labels-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  flex: 1;
}

.labels-list .label-tag {
  margin: 0;
  padding: 6px 14px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  background: linear-gradient(135deg, #f0f2f5 0%, #ffffff 100%);
  border: 1px solid #dcdfe6;
  color: #606266;
  transition: all 0.3s ease;
}

.labels-list .label-tag:hover {
  background: linear-gradient(135deg, #e8f4fd 0%, #f5f9ff 100%);
  border-color: #409eff;
  color: #409eff;
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(64, 158, 255, 0.2);
}

/* 注解样式 */
.annotations-text {
  max-width: 100%;
  padding: 6px 12px;
  background: #fafafa;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  font-size: 13px;
  color: #606266;
  line-height: 1.6;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: help;
  transition: all 0.3s ease;
  display: inline-block;
}

.annotations-text:hover {
  background: #f0f2f5;
  border-color: #c0c4cc;
}

.basic-info-section .truncate-text {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-tabs {
  margin-top: 0;
}

.tab-content {
  padding: 16px;
}

/* Pods 表格样式 */
.pods-table {
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.pods-table :deep(.el-table__body-wrapper) {
  max-height: 400px;
  overflow-y: auto;
}

.pods-table :deep(.el-table__header) {
  background: linear-gradient(180deg, #f5f7fa 0%, #eef1f6 100%);
}

.pods-table :deep(.el-table__header th) {
  background: transparent !important;
  color: #1f2329;
  font-weight: 700;
  font-size: 13px;
  border-bottom: 2px solid #e8eaed;
}

.pods-table :deep(.el-table__body tr) {
  transition: all 0.3s ease;
}

.pods-table :deep(.el-table__body tr:hover) {
  background: linear-gradient(135deg, #f0f9ff 0%, #ffffff 100%) !important;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.1);
}

.pods-table :deep(.el-table__body tr td) {
  border-bottom: 1px solid #f0f2f5;
}

.pod-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 6px;
  transition: all 0.3s ease;
}

.pod-name-cell:hover {
  background: #ecf5ff;
}

.pod-name-cell:hover .pod-name {
  color: #409eff;
}

.pod-icon {
  color: #409eff;
  font-size: 16px;
}

.pod-name {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
}

.resource-value {
  font-size: 13px;
  color: #606266;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
}

/* 端口列表样式 */
.ports-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.port-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.port-text {
  font-size: 13px;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-weight: 500;
}

.name-cell {
  display: flex;
  align-items: center;
}

/* 服务表格样式 */
.services-table {
  font-size: 13px;
  border-radius: 8px;
  overflow: hidden;
}

.services-table :deep(.el-table__header) {
  background: linear-gradient(180deg, #f5f7fa 0%, #eef1f6 100%);
}

.services-table :deep(.el-table__header th) {
  background: transparent !important;
  color: #1f2329;
  font-weight: 700;
  font-size: 13px;
  border-bottom: 2px solid #e8eaed;
}

.services-table :deep(.el-table__body tr) {
  transition: all 0.2s ease;
}

.services-table :deep(.el-table__body tr:hover) {
  background: linear-gradient(90deg, #f5f7ff 0%, #ffffff 100%) !important;
}

.service-name-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-icon {
  color: #409eff;
  font-size: 18px;
  flex-shrink: 0;
}

.service-name-text {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.ip-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.ip-text {
  font-size: 13px;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 4px;
}

.ip-text.external-ip {
  color: #67c23a;
  font-weight: 600;
}

.more-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  font-size: 11px;
  line-height: 18px;
  color: #fff;
  background-color: #909399;
  border-radius: 9px;
  margin-left: 4px;
}

.empty-text {
  font-size: 13px;
  color: #909399;
}

.ports-combined {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.port-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.port-info {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.port-number {
  font-size: 14px;
  font-weight: 600;
  color: #409eff;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
}

.target-port {
  font-size: 13px;
  font-weight: 500;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
}

.port-arrow {
  color: #909399;
  font-size: 12px;
}

.nodeport-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 500;
  color: #e6a23c;
  background: linear-gradient(135deg, #fdf6ec 0%, #faecd8 100%);
  border: 1px solid #f5dab1;
  border-radius: 4px;
}

.port-name {
  font-size: 11px;
  color: #909399;
  font-style: italic;
  margin-left: 4px;
}

.age-text {
  font-size: 13px;
  color: #606266;
}

/* Ingress 样式 */
.ingress-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ingress-hosts-section,
.ingress-rules-section {
  background: #ffffff;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #ebeef5;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 2px solid #f0f2f5;
  position: relative;
}

.section-title::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  width: 50px;
  height: 2px;
  background: linear-gradient(90deg, #409eff 0%, #66b1ff 100%);
  border-radius: 2px;
}

.section-title .el-icon {
  color: #409eff;
  font-size: 18px;
}

.hosts-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 16px;
}

.host-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  background: linear-gradient(135deg, #f0f7ff 0%, #e8f4ff 100%);
  border-radius: 8px;
  border: 1px solid #d4e7ff;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.host-item::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 4px;
  height: 100%;
  background: linear-gradient(180deg, #409eff 0%, #66b1ff 100%);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.host-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.15);
  border-color: #409eff;
}

.host-item:hover::before {
  opacity: 1;
}

.host-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.host-icon {
  color: #67c23a;
  font-size: 18px;
  flex-shrink: 0;
}

.host-text {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  color: #1f2329;
  font-weight: 600;
  letter-spacing: 0.3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: help;
}

.host-ingress-names {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.ingress-name-tag {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 500;
  color: #409eff;
  background: #ffffff;
  border: 1px solid #b3d8ff;
  border-radius: 4px;
  box-shadow: 0 1px 3px rgba(64, 158, 255, 0.1);
  transition: all 0.2s ease;
}

.ingress-name-tag:hover {
  background: #ecf5ff;
  border-color: #409eff;
  transform: scale(1.05);
}

.ingress-rules-table {
  margin-top: 0;
  border-radius: 8px;
  overflow: hidden;
}

.ingress-rules-table :deep(.el-table__header) {
  background: linear-gradient(180deg, #f5f7fa 0%, #eef1f6 100%);
}

.ingress-rules-table :deep(.el-table__header th) {
  background: transparent !important;
  color: #1f2329;
  font-weight: 700;
  font-size: 13px;
  border-bottom: 2px solid #e8eaed;
}

.ingress-rules-table :deep(.el-table__body tr) {
  transition: all 0.2s ease;
}

.ingress-rules-table :deep(.el-table__body tr:hover) {
  background: linear-gradient(90deg, #f5f7ff 0%, #ffffff 100%) !important;
  transform: scale(1.005);
}

.rule-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.rule-icon {
  color: #409eff;
  font-size: 16px;
}

.rule-name-text {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.host-text-cell {
  font-size: 13px;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-weight: 500;
  background: linear-gradient(135deg, #f0f7ff 0%, #e8f4ff 100%);
  padding: 4px 10px;
  border-radius: 4px;
  display: inline-block;
  border: 1px solid #d4e7ff;
}

.path-text-simple {
  font-size: 13px;
  font-weight: 500;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  background: linear-gradient(135deg, #fff9e6 0%, #fff3d6 100%);
  padding: 4px 10px;
  border-radius: 4px;
  border: 1px solid #ffe6a1;
  display: inline-block;
  cursor: help;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.path-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.path-text {
  font-size: 13px;
  font-weight: 600;
  color: #1f2329;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  background: linear-gradient(135deg, #fff9e6 0%, #fff3d6 100%);
  padding: 4px 10px;
  border-radius: 4px;
  border: 1px solid #ffe6a1;
  display: inline-block;
}

.service-name-cell {
  font-size: 13px;
  font-weight: 600;
  color: #409eff;
  background: linear-gradient(135deg, #ecf5ff 0%, #d9ecff 100%);
  padding: 4px 10px;
  border-radius: 4px;
  border: 1px solid #b3d8ff;
  display: inline-block;
}

.port-number-cell {
  font-size: 13px;
  font-weight: 600;
  color: #e6a23c;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  background: linear-gradient(135deg, #fef3e6 0%, #fde7d0 100%);
  padding: 4px 10px;
  border-radius: 4px;
  border: 1px solid #fad295;
  display: inline-block;
  box-shadow: 0 1px 4px rgba(230, 162, 60, 0.1);
}

.restart-high {
  color: #f56c6c;
  font-weight: 600;
}

/* 下拉菜单样式 */
.pods-table :deep(.el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
}

.pods-table :deep(.el-dropdown-menu__item .el-icon) {
  color: #409eff;
  font-size: 14px;
}

/* 运行时信息表格样式 */
.runtime-content {
  background: #fff;
  border-radius: 8px;
  padding: 0;
}

.runtime-table {
  font-size: 13px;
  border-radius: 8px;
  overflow: hidden;
}

.runtime-table :deep(.el-table__header) {
  background: linear-gradient(180deg, #f5f7fa 0%, #eef1f6 100%);
}

.runtime-table :deep(.el-table__header th) {
  background: transparent !important;
  color: #1f2329;
  font-weight: 700;
  font-size: 13px;
  border-bottom: 2px solid #e8eaed;
}

.runtime-table :deep(.el-table__body tr) {
  transition: all 0.2s ease;
}

.runtime-table :deep(.el-table__body tr:hover) {
  background: linear-gradient(90deg, #f5f7ff 0%, #ffffff 100%) !important;
}

.runtime-table :deep(.el-table__body td) {
  border-bottom: 1px solid #f0f2f5;
}

.runtime-category {
  display: flex;
  align-items: center;
  gap: 8px;
}

.category-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.category-icon.icon-pod {
  color: #409eff;
}

.category-icon.icon-replica {
  color: #67c23a;
}

.category-icon.icon-update {
  color: #e6a23c;
}

.category-icon.icon-available {
  color: #67c23a;
}

.category-icon.icon-paused {
  color: #909399;
}

.category-icon.icon-collision {
  color: #f56c6c;
}

.category-icon.icon-observer {
  color: #909399;
}

.category-text {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.status-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.status-indicator {
  font-size: 18px;
  flex-shrink: 0;
}

.status-indicator.status-success {
  color: #67c23a;
}

.status-indicator.status-warning {
  color: #e6a23c;
}

.status-indicator.status-danger {
  color: #f56c6c;
}

.status-indicator.status-primary {
  color: #409eff;
}

.status-indicator.status-info {
  color: #909399;
}

.status-indicator.is-loading {
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.status-text {
  font-size: 13px;
  font-weight: 700;
}

.status-text.status-success {
  color: #67c23a;
}

.status-text.status-warning {
  color: #e6a23c;
}

.status-text.status-danger {
  color: #f56c6c;
}

.status-text.status-primary {
  color: #409eff;
}

.status-text.status-info {
  color: #909399;
}

.message-cell {
  display: flex;
  align-items: center;
}

.message-text {
  font-size: 13px;
  color: #606266;
  line-height: 1.6;
}

.time-text {
  font-size: 13px;
  color: #909399;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
}

/* 暂停页面样式 */
.paused-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
  background: #ffffff;
  border-radius: 8px;
  padding: 24px;
}

.paused-header {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 24px;
  background: linear-gradient(135deg, #f5f7fa 0%, #eef1f6 100%);
  border-radius: 12px;
  border: 1px solid #e8eaed;
}

.paused-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: #ffffff;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.paused-icon {
  font-size: 40px;
  color: #67c23a;
  transition: all 0.3s ease;
}

.paused-icon.is-paused {
  color: #f56c6c;
}

.paused-title {
  flex: 1;
}

.paused-title h3 {
  margin: 0 0 8px 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.paused-status-text {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  color: #67c23a;
}

.paused-status-text.paused {
  color: #f56c6c;
}

.paused-control {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 24px;
  background: #ffffff;
  border-radius: 12px;
  border: 1px solid #ebeef5;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.paused-switch-wrapper {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px;
  background: linear-gradient(135deg, #f5f7fa 0%, #eef1f6 100%);
  border-radius: 8px;
}

.switch-label {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.paused-description {
  margin-top: 8px;
}

.paused-info {
  background: #ffffff;
  border-radius: 8px;
  overflow: hidden;
}

.paused-info :deep(.el-descriptions__label) {
  font-weight: 600;
  background: #f5f7fa !important;
}

.paused-info :deep(.el-descriptions__content) {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
}

.container-group-header {
  font-size: 12px;
  color: #909399;
  font-weight: 600;
  padding: 4px 0;
  border-bottom: 1px solid #e4e7ed;
  margin-bottom: 4px;
}

/* 历史版本表格样式 */
.revisions-content {
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
}

.revisions-table {
  font-size: 13px;
  border-radius: 8px;
  overflow: hidden;
}

.revisions-table :deep(.el-table__header) {
  background: linear-gradient(180deg, #f5f7fa 0%, #eef1f6 100%);
}

.revisions-table :deep(.el-table__header th) {
  background: transparent !important;
  color: #1f2329;
  font-weight: 700;
  font-size: 13px;
  border-bottom: 2px solid #e8eaed;
}

.revisions-table :deep(.el-table__body tr) {
  transition: all 0.2s ease;
}

.revisions-table :deep(.el-table__body tr:hover) {
  background: linear-gradient(90deg, #f5f7ff 0%, #ffffff 100%) !important;
}

.revisions-table :deep(.el-table__body td) {
  border-bottom: 1px solid #f0f2f5;
}

/* 版本单元格样式 */
.revision-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
}

.revision-number-wrapper {
  display: flex;
  align-items: baseline;
  gap: 2px;
}

.revision-icon {
  font-size: 14px;
  font-weight: 600;
  color: #409eff;
}

.revision-number {
  font-size: 16px;
  font-weight: 700;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
}

.current-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 12px;
}

/* 镜像列样式增强 */
.images-column-enhanced {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.image-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: linear-gradient(135deg, #f5f7fa 0%, #eef1f6 100%);
  border-radius: 6px;
  border: 1px solid #e4e7ed;
  transition: all 0.2s ease;
}

.image-card:hover {
  background: linear-gradient(135deg, #ecf5ff 0%, #d9ecff 100%);
  border-color: #b3d8ff;
  transform: translateX(4px);
}

.image-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: linear-gradient(135deg, #409eff 0%, #66b1ff 100%);
  border-radius: 6px;
  color: #fff;
  font-size: 14px;
  flex-shrink: 0;
}

.image-info {
  flex: 1;
  min-width: 0;
}

.image-name {
  font-size: 13px;
  font-weight: 500;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 副本信息样式 */
.replicas-info {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 8px 16px;
  background: linear-gradient(135deg, #f5f7fa 0%, #eef1f6 100%);
  border-radius: 6px;
}

.replica-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.replica-label {
  font-size: 11px;
  color: #909399;
  font-weight: 500;
  text-transform: uppercase;
}

.replica-value {
  font-size: 16px;
  font-weight: 700;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
}

.replica-value.ready {
  color: #67c23a;
}

.replica-divider {
  width: 1px;
  height: 30px;
  background: #dcdfe6;
}

/* 时间单元格样式 */
.time-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}

.time-icon {
  color: #909399;
  font-size: 14px;
}

/* 增强的状态单元格 */
.status-cell-enhanced {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 6px;
  background: linear-gradient(135deg, #f5f7fa 0%, #eef1f6 100%);
}

.status-dot {
  font-size: 16px;
}

.status-dot.status-success {
  color: #67c23a;
}

.status-dot.status-warning {
  color: #e6a23c;
}

.status-dot.status-danger {
  color: #f56c6c;
}

.status-dot.status-info {
  color: #909399;
}

.status-text-enhanced {
  font-size: 13px;
  font-weight: 600;
}

.status-text-enhanced.status-success {
  color: #67c23a;
}

.status-text-enhanced.status-warning {
  color: #e6a23c;
}

.status-text-enhanced.status-danger {
  color: #f56c6c;
}

.status-text-enhanced.status-info {
  color: #909399;
}

/* 操作按钮样式 */
.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s ease;
}

.action-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.view-btn {
  background: linear-gradient(135deg, #ecf5ff 0%, #d9ecff 100%);
  border-color: #b3d8ff;
  color: #409eff;
}

.view-btn:hover {
  background: linear-gradient(135deg, #d9ecff 0%, #b3d8ff 100%);
  border-color: #409eff;
}

.rollback-btn {
  background: linear-gradient(135deg, #fef3e6 0%, #fde7d0 100%);
  border-color: #fad295;
  color: #e6a23c;
}

.rollback-btn:hover {
  background: linear-gradient(135deg, #fde7d0 0%, #fbd6b6 100%);
  border-color: #e6a23c;
}

/* 终端对话框样式 */
.terminal-container {
  position: relative;
  width: 100%;
  height: 600px;
}

.terminal-wrapper {
  width: 100%;
  height: 100%;
  background: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
}

.terminal-loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #1e1e1e;
  border-radius: 8px;
  z-index: 10;
  color: #409eff;
  font-size: 16px;
  gap: 12px;
}

.terminal-loading-overlay .el-icon {
  font-size: 32px;
}

.terminal-iframe {
  width: 100%;
  height: 100%;
  border: none;
}

/* 日志对话框样式 */
.logs-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 8px;
}

.logs-wrapper {
  width: 100%;
  height: 500px;
  overflow: auto;
  background: #1e1e1e;
  border-radius: 8px;
  padding: 16px;
}

.logs-content {
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #d4af37;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.logs-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #409eff;
  font-size: 16px;
  gap: 16px;
}

.logs-loading .el-icon {
  font-size: 32px;
}

.detail-content {
  max-height: 600px;
  overflow-y: auto;
}

.detail-section {
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid #e4e7ed;
}

.detail-section:last-child {
  border-bottom: none;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.info-item.full-width {
  grid-column: 1 / -1;
}

.info-label {
  font-size: 13px;
  color: #909399;
  font-weight: 500;
}

.info-value {
  font-size: 14px;
  color: #303133;
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  margin: 0;
}

.annotations-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.annotation-item {
  display: flex;
  gap: 12px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.annotation-key {
  font-size: 13px;
  font-weight: 500;
  color: #606266;
  min-width: 200px;
}

.annotation-value {
  font-size: 13px;
  color: #909399;
  word-break: break-all;
}

.images-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.image-item {
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.image-name {
  font-size: 13px;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
}

.image-in-cell, .port-in-cell {
  font-size: 12px;
  color: #606266;
  padding: 2px 0;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
}

.yaml-editor-wrapper {
  display: flex;
  border: 1px solid #333;
  border-radius: 8px;
  overflow: hidden;
  background-color: #000000;
}

.yaml-line-numbers {
  background-color: #0a0a0a;
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
  .search-inputs {
    flex-wrap: wrap;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .header-actions {
    width: 100%;
    flex-direction: column;
  }

  .cluster-select,
  .filter-select {
    width: 100%;
  }
}

/* 工作负载编辑对话框 - 白金风格 */
.workload-edit-dialog :deep(.el-dialog__wrapper) {
  overflow: hidden;
}

.workload-edit-dialog :deep(.el-dialog) {
  background: #ffffff;
  border: 1px solid #e8e8e8;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  margin: auto;
  max-height: calc(100vh - 100px);
  display: flex;
  flex-direction: column;
}

.workload-edit-dialog :deep(.el-dialog__header) {
  background: #d4af37;
  border-bottom: 2px solid #c9a227;
  padding: 24px 32px;
  margin: 0;
  position: relative;
}

.workload-edit-dialog :deep(.el-dialog__header::before) {
  display: none;
}

.workload-edit-dialog :deep(.el-dialog__title) {
  font-size: 20px;
  font-weight: 700;
  color: #1a1a1a;
  letter-spacing: 0.5px;
  font-family: 'Helvetica Neue', Arial, sans-serif;
}

.workload-edit-dialog :deep(.el-dialog__headerbtn .el-dialog__close) {
  color: #1a1a1a;
  font-size: 20px;
  transition: all 0.3s ease;
  font-weight: bold;
}

.workload-edit-dialog :deep(.el-dialog__headerbtn .el-dialog__close:hover) {
  color: #000000;
  transform: rotate(90deg);
}

.workload-edit-dialog :deep(.el-dialog__body) {
  padding: 0;
  background: #ffffff;
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

.workload-edit-dialog :deep(.el-dialog__footer) {
  padding: 16px 32px;
  background: #ffffff;
  border-top: 1px solid #e8e8e8;
  flex-shrink: 0;
}

.workload-edit-content {
  display: flex;
  height: calc(100vh - 200px);
  max-height: 800px;
  background: #ffffff;
}

.edit-sidebar {
  width: 360px;
  flex-shrink: 0;
  background: linear-gradient(135deg, #fafafa 0%, #f5f5f5 100%);
  border-right: 2px solid #e8e8e8;
  overflow-y: auto;
}

.edit-sidebar::-webkit-scrollbar {
  width: 8px;
}

.edit-sidebar::-webkit-scrollbar-track {
  background: #f5f5f5;
}

.edit-sidebar::-webkit-scrollbar-thumb {
  background: #d4af37;
  border-radius: 4px;
}

.edit-sidebar::-webkit-scrollbar-thumb:hover {
  background: #c9a227;
}

.edit-main {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: #ffffff;
}

.edit-main :deep(.el-tabs) {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: transparent;
}

.edit-main :deep(.el-tabs__header) {
  margin: 0;
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 100%);
  border-bottom: 2px solid #e8e8e8;
  padding: 0 32px;
}

.edit-main :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.edit-main :deep(.el-tabs__item) {
  color: #666;
  font-weight: 500;
  font-size: 15px;
  padding: 0 28px;
  height: 54px;
  line-height: 54px;
  border: none;
  transition: all 0.3s ease;
  letter-spacing: 0.3px;
}

.edit-main :deep(.el-tabs__item:hover) {
  color: #d4af37;
}

.edit-main :deep(.el-tabs__item.is-active) {
  color: #d4af37;
  background: transparent;
  font-weight: 600;
}

.edit-main :deep(.el-tabs__active-bar) {
  height: 3px;
  background: #d4af37;
}

.edit-main :deep(.el-tabs__content) {
  flex: 1;
  overflow-y: auto;
  padding: 0;
  background: transparent;
}

.edit-main :deep(.el-tabs__content)::-webkit-scrollbar {
  width: 10px;
}

.edit-main :deep(.el-tabs__content)::-webkit-scrollbar-track {
  background: #fafafa;
}

.edit-main :deep(.el-tabs__content)::-webkit-scrollbar-thumb {
  background: #d4af37;
  border-radius: 5px;
}

.edit-main :deep(.el-tabs__content)::-webkit-scrollbar-thumb:hover {
  background: #c9a227;
}

/* 调度页面样式 */
.scheduling-tab-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.info-panel {
  background: #fff;
  border-radius: 4px;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  padding: 12px 20px;
  background: #d4af37;
  border-bottom: 1px solid #d4af37;
}

.panel-icon {
  font-size: 18px;
  margin-right: 8px;
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #ffffff;
  flex: 1;
}

.panel-content {
  padding: 16px;
  background: #ffffff;
}

.placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 450px;
  color: #999;
  font-size: 16px;
  gap: 20px;
  background: #fafafa;
  border-radius: 12px;
  border: 1px dashed #e0e0e0;
}

.placeholder :deep(.el-icon) {
  font-size: 64px;
  opacity: 0.4;
  color: #d4af37;
}

/* 白金风格按钮样式 */
.edit-main :deep(.el-button--primary),
.edit-sidebar :deep(.el-button--primary) {
  background: #d4af37;
  border: none;
  color: #1a1a1a;
  font-weight: 600;
  letter-spacing: 0.3px;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
  transition: all 0.3s ease;
}

.edit-main :deep(.el-button--primary:hover),
.edit-sidebar :deep(.el-button--primary:hover) {
  background: #c9a227;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.4);
  transform: translateY(-1px);
}

.edit-main :deep(.el-button--primary:active),
.edit-sidebar :deep(.el-button--primary:active) {
  transform: translateY(0);
}

.edit-main :deep(.el-button--default),
.edit-sidebar :deep(.el-button--default) {
  background: #ffffff;
  border: 1px solid #e0e0e0;
  color: #666;
  font-weight: 500;
  transition: all 0.3s ease;
}

.edit-main :deep(.el-button--default:hover),
.edit-sidebar :deep(.el-button--default:hover) {
  background: #fafafa;
  border-color: #d4af37;
  color: #d4af37;
}

.edit-main :deep(.el-button--danger) {
  background: #ff4d4f;
  border: none;
  color: #ffffff;
  font-weight: 500;
}

.edit-main :deep(.el-button--danger:hover) {
  background: #ff7875;
}

/* 白金风格输入框 */
.edit-main :deep(.el-input__wrapper),
.edit-main :deep(.el-textarea__inner),
.edit-main :deep(.el-select .el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.edit-main :deep(.el-input__wrapper:hover),
.edit-main :deep(.el-textarea__inner:hover),
.edit-main :deep(.el-select .el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.edit-main :deep(.el-input__wrapper.is-focus),
.edit-main :deep(.el-textarea__inner:focus),
.edit-main :deep(.el-select .el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 4px rgba(212, 175, 55, 0.15);
}

.edit-main :deep(.el-input__inner) {
  color: #333;
  font-weight: 500;
}

.edit-main :deep(.el-input__inner::placeholder) {
  color: #aaa;
}

.edit-main :deep(.el-textarea__inner) {
  color: #333;
  background: #fafafa;
}

.edit-main :deep(.el-select .el-input__inner) {
  color: #333;
}

/* 白金风格标签 */
.edit-main :deep(.el-tag) {
  background: rgba(212, 175, 55, 0.1);
  border: 1px solid #d4af37;
  color: #d4af37;
  font-weight: 600;
}

.edit-main :deep(.el-tag--success) {
  background: rgba(82, 196, 26, 0.1);
  border-color: #52c41a;
  color: #52c41a;
}

.edit-main :deep(.el-tag--warning) {
  background: rgba(250, 173, 20, 0.1);
  border-color: #faad14;
  color: #faad14;
}

.edit-main :deep(.el-tag--danger) {
  background: rgba(255, 77, 79, 0.1);
  border-color: #ff4d4f;
  color: #ff4d4f;
}

/* 白金风格表单 */
.edit-main :deep(.el-form-item__label) {
  color: #333;
  font-weight: 600;
  font-size: 14px;
  letter-spacing: 0.3px;
}

.edit-main :deep(.el-checkbox__label) {
  color: #333;
  font-weight: 500;
}

.edit-main :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background: #d4af37;
  border-color: #d4af37;
}

/* 白金风格表格 */
.edit-main :deep(.el-table) {
  background: #ffffff;
  color: #333;
}

.edit-main :deep(.el-table th) {
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 100%);
  color: #333;
  font-weight: 600;
  border-bottom: 2px solid #e8e8e8;
}

.edit-main :deep(.el-table tr) {
  transition: all 0.3s ease;
}

.edit-main :deep(.el-table tr:hover) {
  background: #fafafa;
}

.edit-main :deep(.el-table td) {
  border-bottom: 1px solid #f0f0f0;
}

/* 白金风格折叠面板 */
.edit-main :deep(.el-collapse-item__header) {
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 100%);
  border: 1px solid #e8e8e8;
  color: #333;
  font-weight: 600;
  transition: all 0.3s ease;
}

.edit-main :deep(.el-collapse-item__header:hover) {
  background: #ffffff;
  border-color: #d4af37;
}

.edit-main :deep(.el-collapse-item__wrap) {
  background: #ffffff;
  border: none;
}

/* 白金风格开关 */
.edit-main :deep(.el-switch.is-checked .el-switch__core) {
  background: #d4af37;
  border-color: #d4af37;
}

/* 白金风格选择器下拉 */
.edit-main :deep(.el-select-dropdown) {
  background: #ffffff;
  border: 1px solid #e8e8e8;
}

.edit-main :deep(.el-select-dropdown__item) {
  color: #333;
}

.edit-main :deep(.el-select-dropdown__item:hover) {
  background: #fafafa;
  color: #d4af37;
}

.edit-main :deep(.el-select-dropdown__item.is-selected) {
  background: rgba(212, 175, 55, 0.1);
  color: #d4af37;
}

/* 白金风格数字输入框 */
.edit-main :deep(.el-input-number .el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
}

.edit-main :deep(.el-input-number__decrease),
.edit-main :deep(.el-input-number__increase) {
  background: #f5f5f5;
  border-left: 1px solid #e0e0e0;
  color: #d4af37;
}

.edit-main :deep(.el-input-number__decrease:hover),
.edit-main :deep(.el-input-number__increase:hover) {
  color: #c9a227;
}

/* 创建工作负载弹窗样式 */
.yaml-create-mode {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.yaml-editor-container {
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
  background: #fafbfc;
}

.yaml-create-mode .yaml-editor-wrapper {
  max-height: 500px;
  overflow: hidden;
}

.create-workload-dialog :deep(.el-dialog__footer) {
  padding: 16px 20px;
  border-top: 1px solid #ebeef5;
}

/* Pod 操作菜单样式 */
.pod-action-menu {
  min-width: 200px;
}

.container-actions {
  margin-bottom: 12px;
}

.container-actions:last-of-type {
  margin-bottom: 0;
}

.container-name {
  font-size: 12px;
  font-weight: 600;
  color: #909399;
  padding: 4px 8px;
  background: #f5f7fa;
  border-radius: 4px;
  margin-bottom: 6px;
}

.container-menu-items {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.2s ease;
  font-size: 14px;
  color: #606266;
}

.menu-item:hover {
  background: #f5f7fa;
  color: #d4af37;
}

.menu-item.danger {
  color: #f56c6c;
}

.menu-item.danger:hover {
  background: #fef0f0;
  color: #f56c6c;
}

.menu-item .el-icon {
  font-size: 16px;
}

.menu-error {
  text-align: center;
  padding: 20px;
  color: #909399;
  font-size: 14px;
}

/* 容器管理美化：工作负载详情与编辑器 */
.workloads-container {
  --k8s-primary: #2563eb;
  --k8s-primary-soft: #eff6ff;
  --k8s-ink: #111827;
  --k8s-muted: #667085;
  --k8s-border: #e6ebf2;
  --k8s-surface: #ffffff;
}

.detail-dialog :deep(.el-dialog) {
  border-radius: 24px;
  overflow: hidden;
  background: #f7faff;
  box-shadow: 0 28px 80px rgba(15, 23, 42, 0.2);
}

.detail-dialog :deep(.el-dialog__header) {
  padding: 20px 26px;
  margin: 0;
  background: #ffffff;
  border-bottom: 1px solid var(--k8s-border);
}

.detail-dialog :deep(.el-dialog__title) {
  color: var(--k8s-ink);
  font-size: 18px;
  font-weight: 800;
}

.detail-dialog :deep(.el-dialog__body) {
  padding: 22px;
  background:
    radial-gradient(circle at top left, rgba(37, 99, 235, 0.08), transparent 32%),
    #f7faff;
}

.detail-wrapper {
  gap: 16px;
}

.detail-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 22px;
  align-items: stretch;
  padding: 24px;
  border: 1px solid var(--k8s-border);
  border-radius: 22px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.98), rgba(248, 251, 255, 0.92)),
    radial-gradient(circle at 92% 12%, rgba(37, 99, 235, 0.14), transparent 30%);
  box-shadow: 0 18px 42px rgba(15, 23, 42, 0.08);
}

.detail-kind-chip {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  padding: 7px 12px;
  border-radius: 999px;
  background: #eff6ff;
  color: var(--k8s-primary);
  border: 1px solid #bfdbfe;
  font-weight: 800;
  font-size: 12px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.detail-title {
  margin: 12px 0 10px;
  color: var(--k8s-ink);
  font-size: 30px;
  font-weight: 900;
  letter-spacing: -0.04em;
  word-break: break-word;
}

.detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.detail-meta span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 10px;
  border: 1px solid #d7e2f2;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.78);
  color: var(--k8s-muted);
  font-size: 13px;
}

.detail-summary-grid {
  display: grid;
  grid-template-columns: repeat(3, 118px);
  gap: 10px;
}

.detail-summary-card {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 86px;
  padding: 14px;
  border: 1px solid #dbe7f5;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.82);
}

.summary-label {
  color: var(--k8s-muted);
  font-size: 12px;
  font-weight: 700;
}

.detail-summary-card strong {
  margin-top: 6px;
  color: var(--k8s-ink);
  font-size: 28px;
  line-height: 1;
}

.detail-meta-section {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.detail-meta-card {
  padding: 16px;
  border: 1px solid var(--k8s-border);
  border-radius: 18px;
  background: #ffffff;
  box-shadow: 0 10px 26px rgba(15, 23, 42, 0.05);
}

.detail-meta-card.images-card {
  grid-column: span 2;
}

.meta-card-title {
  margin-bottom: 12px;
  color: var(--k8s-ink);
  font-size: 14px;
  font-weight: 800;
}

.image-tag {
  border: 1px solid #dbeafe;
  border-radius: 12px;
  background: #f8fbff;
  color: #1d4ed8;
}

.labels-list .label-tag,
.annotations-text {
  border-radius: 999px;
  background: #f8fbff;
  border-color: #dbeafe;
  color: #344054;
}

.detail-tabs {
  border: 1px solid var(--k8s-border);
  border-radius: 18px;
  overflow: hidden;
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.06);
}

.detail-tabs :deep(.el-tabs__header) {
  background: #ffffff;
  border-bottom: 1px solid var(--k8s-border);
}

.detail-tabs :deep(.el-tabs__item) {
  color: #475467;
  font-weight: 700;
}

.detail-tabs :deep(.el-tabs__item.is-active) {
  color: var(--k8s-primary);
}

.detail-tabs :deep(.el-tabs__content) {
  background: #ffffff;
}

@media (max-width: 1280px) {
  .detail-hero {
    grid-template-columns: 1fr;
  }

  .detail-summary-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .detail-meta-section {
    grid-template-columns: 1fr;
  }

  .detail-meta-card.images-card {
    grid-column: auto;
  }

}

</style>
