<template>
  <div class="nodes-container">
    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon stat-icon-blue">
          <el-icon><Monitor /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">节点总数</div>
          <div class="stat-value">{{ nodeList.length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-green">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">运行正常</div>
          <div class="stat-value">{{ readyNodeCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-orange">
          <el-icon><Odometer /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">Pod总数</div>
          <div class="stat-value">{{ totalPodCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-purple">
          <el-icon><CPU /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">总CPU核数</div>
          <div class="stat-value">{{ totalCPUCores }}</div>
        </div>
      </div>
    </div>

    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Monitor /></el-icon>
        </div>
        <div>
          <h2 class="page-title">节点管理</h2>
          <p class="page-subtitle">管理 Kubernetes 集群节点，监控节点状态和资源使用</p>
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
        <el-button class="black-button" @click="loadNodes">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchName"
          placeholder="搜索节点名称..."
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
          placeholder="节点状态"
          clearable
          @change="handleSearch"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><CircleCheck /></el-icon>
          </template>
          <el-option label="正常" value="Ready" />
          <el-option label="异常" value="NotReady" />
        </el-select>

        <el-select
          v-model="searchRole"
          placeholder="节点角色"
          clearable
          @change="handleSearch"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><User /></el-icon>
          </template>
          <el-option label="Master" value="master" />
          <el-option label="Control Plane" value="control-plane" />
          <el-option label="Worker" value="worker" />
        </el-select>

        <el-button
          type="warning"
          :icon="cloudttyInstalled ? Monitor : Download"
          :loading="cloudttyLoading"
          @click="handleCloudTTY"
          class="cloudtty-action-btn"
        >
          {{ cloudttyInstalled ? '打开 CloudTTY' : '部署 CloudTTY' }}
        </el-button>
      </div>
    </div>

    <!-- CloudTTY 部署对话框 -->
    <el-dialog
      v-model="cloudttyDialogVisible"
      title="部署 CloudTTY"
      width="600px"
      :close-on-click-modal="false"
    >
      <div class="cloudtty-dialog-content">
        <el-alert
          title="CloudTTY 是一个 Kubernetes Web Terminal 解决方案"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 20px"
        >
          <template #default>
            <p>CloudTTY 可以提供更强大的节点 Shell 功能，支持：</p>
            <ul style="margin: 10px 0; padding-left: 20px">
              <li>完整的终端模拟</li>
              <li>文件上传/下载</li>
              <li>多标签页支持</li>
              <li>会话审计和录制</li>
            </ul>
          </template>
        </el-alert>

        <div v-if="cloudttyDeploying" class="deploy-status">
          <el-progress :percentage="deployProgress" :status="deployStatus" />
          <p style="margin-top: 10px; text-align: center">{{ deployMessage }}</p>
        </div>

        <div v-else class="deploy-methods">
          <h4>部署步骤：</h4>
          <el-steps direction="vertical" :active="1" class="deploy-steps">
            <el-step title="复制下方命令" />
            <el-step title="在控制台执行命令" />
            <el-step title="等待部署完成（约2-3分钟）" />
            <el-step title="点击已完成部署按钮刷新状态" />
          </el-steps>

          <h4 style="margin-top: 20px">部署命令：</h4>
          <div class="code-block-wrapper">
            <div class="code-line-numbers">
              <div v-for="line in cloudttyCommandLines" :key="line" class="code-line-number">{{ line }}</div>
            </div>
            <textarea
              readonly
              :value="cloudttyCommands"
              class="code-textarea"
            ></textarea>
          </div>
          <el-button
            type="primary"
            @click="copyCommands"
            style="margin-top: 10px"
          >
            复制命令
          </el-button>
        </div>
      </div>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="cloudttyDialogVisible = false">关闭</el-button>
          <el-button
            type="success"
            @click="handleDeployComplete"
          >
            我已完成部署
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- CloudTTY 终端对话框 -->
    <el-dialog
      v-model="cloudttyTerminalVisible"
      :title="`Shell - ${selectedNode?.name || ''}`"
      width="900px"
      :close-on-click-modal="false"
      @close="handleCloseCloudTTY"
      class="cloudtty-terminal-dialog"
    >
      <div class="cloudtty-terminal-wrapper" @click="focusCloudTTYIframe">
        <iframe
          v-if="cloudttyTerminalVisible"
          id="cloudtty-iframe"
          class="cloudtty-iframe"
          frameborder="0"
          allow="clipboard-read; clipboard-write"
        ></iframe>
      </div>
    </el-dialog>

    <!-- 节点列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedNodeList"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        :row-style="{ height: '56px' }"
        :cell-style="{ padding: '8px 0' }"
      >
      <el-table-column label="节点名称" min-width="220" fixed="left">
        <template #header>
          <span class="header-with-icon">
            <el-icon class="header-icon header-icon-blue"><Monitor /></el-icon>
            节点名称
          </span>
        </template>
        <template #default="{ row }">
          <div class="node-name-cell">
            <div class="node-icon-wrapper">
              <el-icon class="node-icon" :size="18"><Platform /></el-icon>
            </div>
            <div class="node-name-content">
              <div class="node-name link-text" @click="goToNodeDetail(row)">{{ row.name }}</div>
              <div class="node-ip">{{ row.internalIP }}</div>
            </div>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="状态" width="100" align="center">
        <template #default="{ row }">
          <el-tag
            :type="row.status === 'Ready' ? 'success' : 'danger'"
            effect="dark"
            size="large"
            class="status-tag"
          >
            {{ row.status === 'Ready' ? '正常' : '异常' }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="角色" width="130" align="center">
        <template #default="{ row }">
          <div :class="['role-badge', 'role-' + (row.roles || 'worker')]">
            <el-icon :size="14"><User /></el-icon>
            <span>{{ getRoleText(row.roles) }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="Kubelet版本" width="140">
        <template #default="{ row }">
          <div class="version-cell">
            <el-icon class="version-icon"><InfoFilled /></el-icon>
            <span>{{ row.version || '-' }}</span>
          </div>
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

      <el-table-column label="运行时间" width="140">
        <template #default="{ row }">
          <div class="age-cell">
            <el-icon class="age-icon"><Clock /></el-icon>
            <span>{{ row.age || '-' }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="CPU" width="180">
        <template #default="{ row }">
          <div class="resource-cell">
            <div class="resource-icon resource-icon-cpu">
              <el-icon><Cpu /></el-icon>
            </div>
            <span class="resource-value">{{ formatCPUWithUsage(row) }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="内存" width="180">
        <template #default="{ row }">
          <div class="resource-cell">
            <div class="resource-icon resource-icon-memory">
              <el-icon><Coin /></el-icon>
            </div>
            <span class="resource-value">{{ formatMemoryWithUsage(row) }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="Pod数量" width="110" align="center">
        <template #default="{ row }">
          <div class="pod-count-cell">
            <span class="pod-count">{{ row.podCount ?? 0 }}/{{ row.podCapacity ?? 0 }}</span>
            <span class="pod-label">Pods</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="调度状态" width="120" align="center">
        <template #default="{ row }">
          <el-tag
            :type="row.schedulable ? 'success' : 'warning'"
            effect="dark"
            size="large"
          >
            {{ row.schedulable ? '可调度' : '不可调度' }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="污点" width="120" align="center">
        <template #default="{ row }">
          <div class="taint-cell" @click="showTaints(row)">
            <div class="taint-badge-wrapper">
              <el-icon class="taint-icon"><WarnTriangleFilled /></el-icon>
              <span class="taint-count">{{ row.taintCount ?? 0 }}</span>
            </div>
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
                <el-dropdown-item command="shell">
                  <el-icon><Monitor /></el-icon>
                  <span>Shell</span>
                </el-dropdown-item>
                <el-dropdown-item command="monitor">
                  <el-icon><DataAnalysis /></el-icon>
                  <span>监控</span>
                </el-dropdown-item>
                <el-dropdown-item command="yaml">
                  <el-icon><Document /></el-icon>
                  <span>YAML</span>
                </el-dropdown-item>
                <el-dropdown-item command="drain" divided>
                  <el-icon><CircleClose /></el-icon>
                  <span>节点排空</span>
                </el-dropdown-item>
                <el-dropdown-item command="cordon">
                  <el-icon><Warning /></el-icon>
                  <span>设为不可调度</span>
                </el-dropdown-item>
                <el-dropdown-item command="uncordon">
                  <el-icon><CircleCheck /></el-icon>
                  <span>设为可调度</span>
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
          :total="filteredNodeList.length"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 标签弹窗 -->
    <el-dialog
      v-model="labelDialogVisible"
      :title="labelEditMode ? '编辑节点标签' : '节点标签'"
      width="850px"
      class="label-dialog"
      :close-on-click-modal="!labelEditMode"
    >
      <div class="label-dialog-content">
        <!-- 编辑模式 -->
        <div v-if="labelEditMode" class="label-edit-container">
          <div class="label-edit-header">
            <div class="label-edit-info">
              <el-icon class="info-icon"><InfoFilled /></el-icon>
              <span>编辑 {{ selectedNode?.name }} 的标签</span>
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
            <el-button @click="labelDialogVisible = false" size="large">关闭</el-button>
            <el-button type="primary" @click="startLabelEdit" :icon="Edit" size="large" class="edit-btn">
              编辑标签
            </el-button>
          </template>
        </div>
      </template>
    </el-dialog>

    <!-- 污点弹窗 -->
    <el-dialog
      v-model="taintDialogVisible"
      :title="taintEditMode ? '编辑节点污点' : '节点污点'"
      width="900px"
      class="taint-dialog"
      :close-on-click-modal="!taintEditMode"
    >
      <div class="taint-dialog-content">
        <!-- 编辑模式 -->
        <div v-if="taintEditMode" class="taint-edit-container">
          <div class="taint-edit-header">
            <div class="taint-edit-info">
              <el-icon class="info-icon"><WarnTriangleFilled /></el-icon>
              <span>编辑 {{ selectedNode?.name }} 的污点</span>
            </div>
            <div class="taint-edit-count">
              共 {{ editTaintList.length }} 个污点
            </div>
          </div>

          <div class="taint-edit-list">
            <div v-for="(taint, index) in editTaintList" :key="index" class="taint-edit-row">
              <div class="taint-row-number">{{ index + 1 }}</div>
              <div class="taint-row-content">
                <div class="taint-input-group">
                  <div class="taint-input-wrapper">
                    <span class="taint-input-label">Key</span>
                    <el-input
                      v-model="taint.key"
                      placeholder="如: key1"
                      size="default"
                      class="taint-edit-input"
                    />
                  </div>
                  <span class="taint-separator">=</span>
                  <div class="taint-input-wrapper">
                    <span class="taint-input-label">Value</span>
                    <el-input
                      v-model="taint.value"
                      placeholder="可选"
                      size="default"
                      class="taint-edit-input"
                    />
                  </div>
                  <span class="taint-separator">:</span>
                  <div class="taint-effect-wrapper">
                    <span class="taint-input-label">Effect</span>
                    <el-select
                      v-model="taint.effect"
                      placeholder="选择"
                      size="default"
                      class="taint-effect-select"
                    >
                      <el-option label="NoSchedule" value="NoSchedule">
                        <div class="effect-option">
                          <el-tag type="warning" size="small" effect="plain">NoSchedule</el-tag>
                          <span class="effect-desc">Pod 不会被调度</span>
                        </div>
                      </el-option>
                      <el-option label="PreferNoSchedule" value="PreferNoSchedule">
                        <div class="effect-option">
                          <el-tag type="info" size="small" effect="plain">PreferNoSchedule</el-tag>
                          <span class="effect-desc">尽量不调度</span>
                        </div>
                      </el-option>
                      <el-option label="NoExecute" value="NoExecute">
                        <div class="effect-option">
                          <el-tag type="danger" size="small" effect="plain">NoExecute</el-tag>
                          <span class="effect-desc">驱逐已有 Pod</span>
                        </div>
                      </el-option>
                    </el-select>
                  </div>
                </div>
              </div>
              <el-button
                type="danger"
                :icon="Delete"
                size="default"
                @click="removeEditTaint(index)"
                class="remove-btn"
                circle
              />
            </div>
            <div v-if="editTaintList.length === 0" class="empty-taints">
              <el-icon class="empty-icon"><WarnTriangleFilled /></el-icon>
              <p>暂无污点</p>
              <span>点击下方按钮添加新污点</span>
            </div>
          </div>

          <el-button
            type="primary"
            :icon="Plus"
            @click="addEditTaint"
            class="add-taint-btn"
            plain
          >
            添加污点
          </el-button>
        </div>

        <!-- 查看模式 -->
        <el-table v-else :data="taintList" class="taint-table" max-height="500">
          <el-table-column prop="key" label="Key" min-width="200">
            <template #default="{ row }">
              <div class="taint-key-wrapper" @click="copyToClipboard(row.key, 'Key')">
                <span class="taint-key-text">{{ row.key }}</span>
                <el-icon class="copy-icon"><CopyDocument /></el-icon>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="value" label="Value" min-width="200">
            <template #default="{ row }">
              <span class="taint-value">{{ row.value || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="effect" label="Effect" width="120" align="center">
            <template #default="{ row }">
              <el-tag :type="getEffectTagType(row.effect)" class="effect-tag">
                {{ row.effect }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <template v-if="taintEditMode">
            <el-button @click="cancelTaintEdit" size="large">取消</el-button>
            <el-button type="primary" @click="saveTaints" :loading="taintSaving" size="large" class="save-btn">
              <el-icon v-if="!taintSaving"><Check /></el-icon>
              {{ taintSaving ? '保存中...' : '保存更改' }}
            </el-button>
          </template>
          <template v-else>
            <el-button @click="taintDialogVisible = false" size="large">关闭</el-button>
            <el-button type="primary" @click="startTaintEdit" :icon="Edit" size="large" class="edit-btn">
              编辑污点
            </el-button>
          </template>
        </div>
      </template>
    </el-dialog>

    <!-- YAML 编辑弹窗 -->
    <el-dialog
      v-model="yamlDialogVisible"
      :title="`节点 YAML - ${selectedNode?.name || ''}`"
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

    <!-- Shell 终端弹窗 -->
    <el-dialog
      v-model="shellDialogVisible"
      :title="`Shell - ${selectedNode?.name || ''}`"
      width="900px"
      class="shell-dialog"
      @close="handleCloseShell"
      @opened="handleShellOpened"
    >
      <div class="shell-dialog-content">
        <div ref="terminalRef" class="terminal-container"></div>
      </div>
      <template #footer>
        <span></span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from 'axios'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import {
  Search,
  PriceTag,
  Monitor,
  Platform,
  CircleCheck,
  User,
  InfoFilled,
  Clock,
  Cpu,
  Coin,
  View,
  Odometer,
  Refresh,
  CopyDocument,
  WarnTriangleFilled,
  Edit,
  DataAnalysis,
  Document,
  CircleClose,
  Setting,
  Delete,
  Warning,
  Plus,
  Check
} from '@element-plus/icons-vue'
import { getClusterList, type Cluster, getNodes, type NodeInfo } from '@/api/kubernetes'

const loading = ref(false)
const router = useRouter()
const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const nodeList = ref<NodeInfo[]>([])

// 搜索条件
const searchName = ref('')
const searchStatus = ref('')
const searchRole = ref('')

// 分页状态
const currentPage = ref(1)
const pageSize = ref(10)
const paginationStorageKey = ref('nodes_pagination')

// 标签弹窗
const labelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])
const labelEditMode = ref(false)
const labelSaving = ref(false)
const editLabelList = ref<{ key: string; value: string }[]>([])
const labelOriginalYaml = ref('')

// 污点弹窗
const taintDialogVisible = ref(false)
const taintList = ref<{ key: string; value: string; effect: string }[]>([])
const taintEditMode = ref(false)
const taintSaving = ref(false)
const editTaintList = ref<{ key: string; value: string; effect: string }[]>([])
const taintOriginalYaml = ref('')

// YAML 编辑弹窗
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const yamlSaving = ref(false)
const selectedNode = ref<NodeInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)

// Shell 终端弹窗
const shellDialogVisible = ref(false)
const terminalRef = ref<HTMLElement | null>(null)
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null

// CloudTTY 相关状态
const cloudttyInstalled = ref(false)
const cloudttyLoading = ref(false)
const cloudttyDialogVisible = ref(false)
const cloudttyTerminalVisible = ref(false)
const cloudttyDeploying = ref(false)
const deployProgress = ref(0)
const deployStatus = ref<'success' | 'exception' | ''>('')
const deployMessage = ref('')
const deployMethod = ref('auto')
const cloudttyCommands = ref(`#1、安装并等待 Pod 运行起来
helm repo add cloudtty https://cloudtty.github.io/cloudtty
helm repo update
helm install cloudtty-operator cloudtty/cloudtty \\
  --version 0.5.0 \\
  --create-namespace \\
  --namespace cloudtty-system

#2、创建 cloudshell.yaml
cat <<EOF > cloudshell.yaml
apiVersion: cloudshell.cloudtty.io/v1alpha1
kind: CloudShell
metadata:
  name: permanent-terminal
  namespace: cloudtty-system
spec:
  # 命令 - 使用交互式 bash
  commandAction: "bash -il"
  # 暴露方式
  exposureMode: NodePort
  # 单次连接模式 - false 表示允许多次连接
  once: false
  # 不自动清理
  cleanup: false
  ttlSecondsAfterStarted: 315360000
  # 允许 URL 参数
  urlArg: true
  # 环境变量 - 配置 ttyd 参数
  env:
  - name: TTYD_WRITABLE
    value: "true"
  - name: TTYD_SERVER_BUFFER_SIZE
    value: "4096"
  - name: TTYD_CLIENT_BUFFER_SIZE
    value: "4096"
  # ttyd 客户端选项
  ttydClientOptions:
    fontSize: "14"
    fontFamily: "Monaco, Menlo, Consolas, 'Courier New', monospace"
    cursorBlink: "true"
    rendererType: "canvas"
  # 镜像
  image: "cloudshell/cloudshell:latest"
EOF

kubectl apply -f cloudshell.yaml

#3、观察 CR 状态，获取访问接入点
kubectl get cloudshell -w`)

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 计算CloudTTY命令行数
const cloudttyCommandLines = computed(() => {
  if (!cloudttyCommands.value) return 1
  return cloudttyCommands.value.split('\n').length
})

// 过滤后的节点列表
const filteredNodeList = computed(() => {
  let result = nodeList.value

  if (searchName.value) {
    result = result.filter(node =>
      node.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (searchStatus.value) {
    result = result.filter(node => node.status === searchStatus.value)
  }

  if (searchRole.value) {
    result = result.filter(node => node.roles === searchRole.value)
  }

  return result
})

// 分页后的节点列表
const paginatedNodeList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredNodeList.value.slice(start, end)
})

// 统计数据
const readyNodeCount = computed(() => {
  return nodeList.value.filter(node => node.status === 'Ready').length
})

const totalPodCount = computed(() => {
  const usedPods = nodeList.value.reduce((sum, node) => sum + (node.podCount || 0), 0)
  const totalPods = nodeList.value.reduce((sum, node) => sum + (node.podCapacity || 0), 0)
  return `${usedPods}/${totalPods}`
})

const totalCPUCores = computed(() => {
  let totalCores = 0
  nodeList.value.forEach(node => {
    if (node.cpuCapacity) {
      const cores = parseCPU(node.cpuCapacity)
      totalCores += cores
    }
  })
  return totalCores.toFixed(1)
})

// 解析 CPU 核数
const parseCPU = (cpu: string): number => {
  if (!cpu) return 0
  if (cpu.endsWith('m')) {
    return parseInt(cpu) / 1000
  }
  return parseFloat(cpu) || 0
}

// 格式化 CPU 显示
const formatCPU = (cpu: string) => {
  if (!cpu) return '-'
  if (cpu.endsWith('m')) {
    const millicores = parseInt(cpu)
    if (isNaN(millicores)) return cpu
    return (millicores / 1000).toFixed(2) + ' 核'
  }
  return cpu + ' 核'
}

// 格式化内存显示
const formatMemory = (memory: string) => {
  if (!memory) return '-'

  const match = memory.match(/^(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti)?$/i)
  if (!match) return memory

  const value = parseFloat(match[1])
  const unit = match[2]?.toUpperCase()

  if (!unit) {
    const bytes = value
    const tb = bytes / (1024 * 1024 * 1024 * 1024)
    if (tb >= 1) return Math.ceil(tb) + ' TB'
    const gb = bytes / (1024 * 1024 * 1024)
    if (gb >= 1) return Math.ceil(gb) + ' GB'
    const mb = bytes / (1024 * 1024)
    if (mb >= 1) return Math.ceil(mb) + ' MB'
    return memory
  }

  let bytes = 0
  switch (unit) {
    case 'KI':
      bytes = value * 1024
      break
    case 'MI':
      bytes = value * 1024 * 1024
      break
    case 'GI':
      bytes = value * 1024 * 1024 * 1024
      break
    case 'TI':
      bytes = value * 1024 * 1024 * 1024 * 1024
      break
  }

  const tb = bytes / (1024 * 1024 * 1024 * 1024)
  if (tb >= 1) return Math.ceil(tb) + ' TB'

  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return Math.ceil(gb) + ' GB'

  const mb = bytes / (1024 * 1024)
  if (mb >= 1) return Math.ceil(mb) + ' MB'

  return memory
}

// 格式化 CPU 显示（包含使用量）
const formatCPUWithUsage = (node: NodeInfo) => {
  const usedCores = (node.cpuUsed || 0) / 1000 // 毫核转核
  const totalCores = parseCPU(node.cpuCapacity)

  const used = usedCores.toFixed(1) // 已使用保留1位小数
  const total = Math.round(totalCores) // 总数不保留小数

  return `${used}/${total}核`
}

// 格式化内存显示（包含使用量）
const formatMemoryWithUsage = (node: NodeInfo) => {
  const usedBytes = node.memoryUsed || 0

  // 解析总内存
  const match = node.memoryCapacity.match(/^(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti)?$/i)
  if (!match) return '-'

  const value = parseFloat(match[1])
  const unit = match[2]?.toUpperCase()

  let totalBytes = 0
  if (!unit) {
    totalBytes = value
  } else {
    switch (unit) {
      case 'KI':
        totalBytes = value * 1024
        break
      case 'MI':
        totalBytes = value * 1024 * 1024
        break
      case 'GI':
        totalBytes = value * 1024 * 1024 * 1024
        break
      case 'TI':
        totalBytes = value * 1024 * 1024 * 1024 * 1024
        break
    }
  }

  // 转换为GB
  const usedGB = usedBytes / (1024 * 1024 * 1024)
  const totalGB = totalBytes / (1024 * 1024 * 1024)

  const used = usedGB >= 1 ? usedGB.toFixed(1) : (usedBytes / (1024 * 1024)).toFixed(1)
  const total = totalGB >= 1 ? Math.ceil(totalGB) + 'G' : Math.ceil(totalBytes / (1024 * 1024)) + 'M'

  return `内存:${used}/${total}`
}

// 获取角色文本
const getRoleText = (role: string | undefined) => {
  if (!role) return 'Worker'
  if (role === 'master') return 'Master'
  if (role === 'control-plane') return 'Control Plane'
  if (role === 'worker') return 'Worker'
  return role
}

// 获取 Effect 标签类型
const getEffectTagType = (effect: string) => {
  switch (effect) {
    case 'NoSchedule':
      return 'warning'
    case 'NoExecute':
      return 'danger'
    case 'PreferNoSchedule':
      return 'info'
    default:
      return ''
  }
}

// 显示标签弹窗
const showLabels = (row: NodeInfo) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelEditMode.value = false
  selectedNode.value = row
  labelDialogVisible.value = true
}

// 开始编辑标签
const startLabelEdit = async () => {
  if (!selectedNode.value) return

  try {
    const token = localStorage.getItem('token')
    const nodeName = selectedNode.value.name

    // 获取节点当前 YAML
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/yaml?clusterId=${selectedClusterId.value}`,
      {
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
    console.error('获取节点YAML失败:', error)
    ElMessage.error('获取节点信息失败')
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
  if (!selectedNode.value) return

  // 验证标签 - 只验证 key，value 可以为空
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
    const nodeName = selectedNode.value.name

    console.log('保存标签 - 节点:', nodeName)
    console.log('有效标签:', validLabels)

    // 判断是否为系统标签
    const isSystemLabel = (key: string) => {
      return key.startsWith('kubernetes.io/') ||
             key.startsWith('node-role.') ||
             key.startsWith('node.kubernetes.io/') ||
             key.startsWith('beta.kubernetes.io/')
    }

    // 从 editLabelList 中分离系统标签和用户标签
    const systemLabels: { key: string; value: string }[] = []
    const userLabels: { key: string; value: string }[] = []

    validLabels.forEach(l => {
      if (isSystemLabel(l.key)) {
        systemLabels.push(l)
      } else {
        userLabels.push(l)
      }
    })

    console.log('系统标签:', systemLabels)
    console.log('用户标签:', userLabels)

    // 合并系统标签和用户标签
    const allLabels = [...systemLabels, ...userLabels]

    // 构建包含所有标签的 YAML
    const labelsStr = allLabels
      .map(l => {
        if (l.value === '') {
          return `    ${l.key}: ""`
        }
        return `    ${l.key}: ${l.value}`
      })
      .join('\n')

    const labelsYaml = `apiVersion: v1
kind: Node
metadata:
  name: ${nodeName}
  labels:
${labelsStr}
`

    console.log('发送的 labels YAML:', labelsYaml)

    // 调用 API 保存
    const response = await axios.put(
      `/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/yaml`,
      {
        clusterId: selectedClusterId.value,
        yaml: labelsYaml
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    console.log('API 响应:', response.data)
    ElMessage.success('标签保存成功')
    labelEditMode.value = false
    // 刷新节点列表
    await loadNodes()
    console.log('刷新节点列表完成')
    // 从刷新后的节点数据中重新获取标签
    const updatedNode = nodeList.value.find(n => n.name === nodeName)
    console.log('查找更新后的节点:', updatedNode)
    if (updatedNode) {
      console.log('节点标签:', updatedNode.labels)
      selectedNode.value = updatedNode
      labelList.value = Object.keys(updatedNode.labels || {}).map(key => ({
        key,
        value: updatedNode.labels![key]
      }))
      console.log('最终 labelList:', labelList.value)
    }
  } catch (error: any) {
    console.error('保存标签失败:', error)
    ElMessage.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    labelSaving.value = false
  }
}

// 跳转到节点详情页
const goToNodeDetail = (row: NodeInfo) => {
  const cluster = clusterList.value.find(c => c.id === selectedClusterId.value)
  router.push({
    name: 'K8sNodeDetail',
    params: {
      clusterId: selectedClusterId.value,
      nodeName: row.name
    },
    query: {
      clusterName: cluster?.alias || cluster?.name
    }
  })
}

// 显示污点弹窗
const showTaints = (row: NodeInfo) => {
  const taints = row.taints || []
  taintList.value = taints.map(taint => ({
    key: taint.key,
    value: taint.value || '',
    effect: taint.effect
  }))
  // 重置编辑模式
  taintEditMode.value = false
  selectedNode.value = row
  taintDialogVisible.value = true
}

// 开始编辑污点
const startTaintEdit = async () => {
  if (!selectedNode.value) return

  try {
    const token = localStorage.getItem('token')
    const nodeName = selectedNode.value.name

    // 获取节点当前 YAML
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/yaml?clusterId=${selectedClusterId.value}`,
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    taintOriginalYaml.value = response.data.data?.yaml || ''

    // 复制当前污点到编辑列表
    editTaintList.value = taintList.value.map(taint => ({
      key: taint.key,
      value: taint.value || '',
      effect: taint.effect
    }))

    taintEditMode.value = true
  } catch (error) {
    console.error('获取节点YAML失败:', error)
    ElMessage.error('获取节点信息失败')
  }
}

// 取消编辑污点
const cancelTaintEdit = () => {
  taintEditMode.value = false
  editTaintList.value = []
}

// 添加编辑污点
const addEditTaint = () => {
  editTaintList.value.push({ key: '', value: '', effect: 'NoSchedule' })
}

// 删除编辑污点
const removeEditTaint = (index: number) => {
  editTaintList.value.splice(index, 1)
}

// 保存污点
const saveTaints = async () => {
  if (!selectedNode.value) return

  // 验证污点
  const validTaints = editTaintList.value.filter(taint => taint.key.trim() !== '')
  if (validTaints.some(taint => !taint.key || !taint.effect)) {
    ElMessage.warning('请填写完整的污点键和Effect')
    return
  }

  // 检查是否有重复的键
  const keys = validTaints.map(t => t.key)
  const uniqueKeys = new Set(keys)
  if (keys.length !== uniqueKeys.size) {
    ElMessage.warning('存在重复的污点键，请检查')
    return
  }

  taintSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const nodeName = selectedNode.value.name

    // 将 YAML 按行分割处理
    const lines = taintOriginalYaml.value.split('\n')
    let updatedLines: string[] = []
    let i = 0

    while (i < lines.length) {
      const line = lines[i]

      // 检测 taints: 开始（在 spec 下的 2 空格缩进）
      if (/^  taints:\s*$/.test(line)) {
        // 如果有污点，保留 taints 并添加新内容
        if (validTaints.length > 0) {
          updatedLines.push(line)
          // 添加污点内容（列表项2空格，属性4空格）
          for (const taint of validTaints) {
            if (taint.value) {
              updatedLines.push(`  - key: ${taint.key}`)
              updatedLines.push(`    value: ${taint.value}`)
              updatedLines.push(`    effect: ${taint.effect}`)
            } else {
              updatedLines.push(`  - key: ${taint.key}`)
              updatedLines.push(`    effect: ${taint.effect}`)
            }
          }
        }
        // 跳过原有的污点内容
        i++
        // 跳过所有污点条目（2 空格缩进的 "- " 开头）
        while (i < lines.length && /^  -\s/.test(lines[i])) {
          i++
          // 跳过污点的属性行（4 空格缩进）
          while (i < lines.length && /^    /.test(lines[i])) {
            i++
          }
        }
        continue
      }

      updatedLines.push(line)
      i++
    }

    const updatedYaml = updatedLines.join('\n')

    // 调用 API 保存
    await axios.put(
      `/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/yaml`,
      {
        clusterId: selectedClusterId.value,
        yaml: updatedYaml
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('污点保存成功')
    taintEditMode.value = false
    // 刷新节点列表
    await loadNodes()
    // 更新当前显示的污点列表
    taintList.value = validTaints
  } catch (error: any) {
    console.error('保存污点失败:', error)
    ElMessage.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    taintSaving.value = false
  }
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

// 保存分页状态到 localStorage
const savePaginationState = () => {
  try {
    localStorage.setItem(paginationStorageKey.value, JSON.stringify({
      currentPage: currentPage.value,
      pageSize: pageSize.value
    }))
  } catch (error) {
    console.error('保存分页状态失败:', error)
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
    console.error('恢复分页状态失败:', error)
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
  const maxPage = Math.ceil(filteredNodeList.value.length / size)
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
      const savedClusterId = localStorage.getItem('nodes_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      await loadNodes()
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取集群列表失败')
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('nodes_selected_cluster_id', selectedClusterId.value.toString())
  }
  // 切换集群时重置分页
  currentPage.value = 1
  await loadNodes()
}

// 加载节点列表
const loadNodes = async () => {
  if (!selectedClusterId.value) return

  loading.value = true
  try {
    const data = await getNodes(selectedClusterId.value)
    nodeList.value = data || []
    // 恢复分页状态
    restorePaginationState()
  } catch (error) {
    console.error(error)
    nodeList.value = []
    ElMessage.error('获取节点列表失败')
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  // 搜索时重置到第一页
  currentPage.value = 1
  savePaginationState()
}

// 查看详情
const handleViewDetails = (row: NodeInfo) => {
  console.log('查看节点详情:', row)
  ElMessage.info('详情功能开发中...')
}

// 处理下拉菜单命令
const handleActionCommand = (command: string, row: NodeInfo) => {
  selectedNode.value = row

  switch (command) {
    case 'shell':
      handleShell()
      break
    case 'monitor':
      ElMessage.info('监控功能开发中...')
      break
    case 'yaml':
      handleShowYAML()
      break
    case 'drain':
      handleDrainNode()
      break
    case 'cordon':
      handleCordonNode()
      break
    case 'uncordon':
      handleUncordonNode()
      break
    case 'delete':
      handleDeleteNode()
      break
    case 'schedule':
      ElMessage.info('调度设置功能开发中...')
      break
  }
}

// 节点排空
const handleDrainNode = async () => {
  if (!selectedNode.value) return

  try {
    await ElMessageBox.confirm(
      `确定要排空节点 ${selectedNode.value.name} 吗？这将会驱逐该节点上的所有 Pod。`,
      '节点排空确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'black-button'
      }
    )

    // 用户确认后执行排空
    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/nodes/${selectedNode.value.name}/drain`,
      {
        clusterId: selectedClusterId.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('节点排空成功')
    await loadNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('节点排空失败:', error)
      ElMessage.error(`节点排空失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 设为不可调度
const handleCordonNode = async () => {
  if (!selectedNode.value) return

  try {
    await ElMessageBox.confirm(
      `确定要将节点 ${selectedNode.value.name} 设为不可调度吗？该节点将不再接受新的Pod调度。`,
      '设为不可调度确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'black-button'
      }
    )

    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/nodes/${selectedNode.value.name}/cordon`,
      {
        clusterId: selectedClusterId.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('节点已设为不可调度')
    await loadNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('设为不可调度失败:', error)
      ElMessage.error(`设为不可调度失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 设为可调度
const handleUncordonNode = async () => {
  if (!selectedNode.value) return

  try {
    await ElMessageBox.confirm(
      `确定要将节点 ${selectedNode.value.name} 设为可调度吗？该节点将重新接受新的Pod调度。`,
      '设为可调度确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info',
        confirmButtonClass: 'black-button'
      }
    )

    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/nodes/${selectedNode.value.name}/uncordon`,
      {
        clusterId: selectedClusterId.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('节点已设为可调度')
    await loadNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('设为可调度失败:', error)
      ElMessage.error(`设为可调度失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 删除节点
const handleDeleteNode = async () => {
  if (!selectedNode.value) return

  try {
    await ElMessageBox.confirm(
      `确定要删除节点 ${selectedNode.value.name} 吗？此操作不可恢复！`,
      '删除节点确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error',
        confirmButtonClass: 'black-button'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/nodes/${selectedNode.value.name}?clusterId=${selectedClusterId.value}`,
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('节点删除成功')
    await loadNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('删除节点失败:', error)
      ElMessage.error(`删除节点失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 显示 YAML 编辑器
const handleShowYAML = async () => {
  if (!selectedNode.value) return

  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const nodeName = selectedNode.value.name

    console.log('获取 YAML:', { clusterId, nodeName })

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/yaml?clusterId=${clusterId}`,
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    console.log('YAML 响应:', response.data)

    yamlContent.value = response.data.data?.yaml || ''
    yamlDialogVisible.value = true
  } catch (error: any) {
    console.error('获取 YAML 失败:', error)
    console.error('错误响应:', error.response?.data)
    ElMessage.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  }
}

// 保存 YAML
const handleSaveYAML = async () => {
  if (!selectedNode.value) return

  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token')
    await axios.put(
      `/api/v1/plugins/kubernetes/resources/nodes/${selectedNode.value.name}/yaml`,
      {
        clusterId: selectedClusterId.value,
        yaml: yamlContent.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    await loadNodes()
  } catch (error) {
    console.error('保存 YAML 失败:', error)
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

// 打开 Shell 终端
const handleShell = async () => {
  if (!selectedNode.value) return

  try {
    const token = localStorage.getItem('token')

    // 先获取CloudTTY的Service信息
    const serviceResponse = await axios.get(
      `/api/v1/plugins/kubernetes/cloudtty/service`,
      {
        params: {
          clusterId: selectedClusterId.value
        },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    if (serviceResponse.data.code !== 0) {
      ElMessage.error('CloudTTY服务未找到，请先部署CloudTTY')
      return
    }

    const service = serviceResponse.data.data
    const nodeIp = service.nodeIP || selectedNode.value.internalIP
    const port = service.port || 30000
    const path = service.path || '/cloudtty'

    // 构建CloudTTY访问地址
    // CloudTTY NodePort 模式下，直接访问服务地址即可
    const cloudttyUrl = `http://${nodeIp}:${port}/`

    console.log('CloudTTY URL:', cloudttyUrl)
    console.log('Node:', selectedNode.value.name)

    // 打开 CloudTTY 终端对话框
    cloudttyTerminalVisible.value = true

    nextTick(() => {
      const iframe = document.getElementById('cloudtty-iframe') as HTMLIFrameElement
      if (iframe) {
        iframe.src = cloudttyUrl
        // 添加焦点处理，确保 iframe 可以接收键盘输入
        iframe.addEventListener('load', () => {
          try {
            iframe.contentWindow?.focus()
          } catch (e) {
            console.log('无法设置 iframe 焦点:', e)
          }
        })
      }
    })
  } catch (error: any) {
    console.error('获取CloudTTY服务失败:', error)
    ElMessage.error('无法连接到CloudTTY服务: ' + (error.response?.data?.message || error.message))
  }
}

// 关闭 CloudTTY 终端
const handleCloseCloudTTY = () => {
  const iframe = document.getElementById('cloudtty-iframe') as HTMLIFrameElement
  if (iframe) {
    iframe.src = '' // 清空iframe以停止加载
  }
  cloudttyTerminalVisible.value = false
}

// 聚焦 CloudTTY iframe
const focusCloudTTYIframe = () => {
  const iframe = document.getElementById('cloudtty-iframe') as HTMLIFrameElement
  if (iframe && iframe.contentWindow) {
    try {
      iframe.contentWindow.focus()
      console.log('CloudTTY iframe 已聚焦')
    } catch (e) {
      console.log('无法聚焦 iframe:', e)
    }
  }
}

// Shell 终端初始化
const handleShellOpened = async () => {
  await nextTick()
  const container = terminalRef.value
  if (!container || !selectedNode.value) return

  // 清空容器
  container.innerHTML = ''

  // 创建终端实例
  terminal = new Terminal({
    theme: {
      background: '#000000',
      foreground: '#d4af37',
      cursor: '#d4af37',
      selection: '#ffffff40'
    },
    fontFamily: 'Monaco, Menlo, Courier New, monospace',
    fontSize: 14,
    lineHeight: 1.2,
    cursorBlink: true,
    scrollback: 1000
  })

  // 加载插件
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())

  // 打开终端
  terminal.open(container)
  fitAddon.fit()

  // 建立WebSocket连接
  const token = localStorage.getItem('token')
  const wsUrl = `ws://localhost:9876/api/v1/plugins/kubernetes/shell/nodes/${selectedNode.value.name}?clusterId=${selectedClusterId.value}&token=${token}`

  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    terminal.writeln('连接成功...\r\n')
  }

  ws.onmessage = (event) => {
    terminal.write(event.data)
  }

  ws.onerror = (error) => {
    terminal.writeln('\r\n\x1b[31m连接错误\x1b[0m')
    console.error('WebSocket error:', error)
  }

  ws.onclose = () => {
    terminal.writeln('\r\n\x1b[33m连接已关闭\x1b[0m')
  }

  // 监听终端输入
  terminal.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })

  // 监听窗口大小变化
  const resizeObserver = new ResizeObserver(() => {
    if (fitAddon) {
      fitAddon.fit()
    }
  })
  resizeObserver.observe(container)
}

// 关闭 Shell 终端
const handleCloseShell = () => {
  if (ws) {
    ws.close()
    ws = null
  }
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  if (fitAddon) {
    fitAddon = null
  }
  shellDialogVisible.value = false
}

// 检查 CloudTTY 是否已安装
const checkCloudTTY = async () => {
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/cloudtty/status`,
      {
        params: { clusterId: selectedClusterId.value },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    cloudttyInstalled.value = response.data.data?.installed || false
  } catch (error) {
    console.log('CloudTTY check failed:', error)
  }
}

// 处理 CloudTTY 按钮
const handleCloudTTY = () => {
  console.log('CloudTTY button clicked, installed:', cloudttyInstalled.value)
  cloudttyDialogVisible.value = true
  console.log('Dialog visible set to:', cloudttyDialogVisible.value)
}

// 开始部署
const startDeploy = async () => {
  // 自动部署改为提示用户使用手动部署
  deployMethod.value = 'manual'
  await copyCommands()
  ElMessage.success('命令已复制，请在控制台执行')
}

// 复制命令
const copyCommands = () => {
  navigator.clipboard.writeText(cloudttyCommands.value)
  ElMessage.success('命令已复制到剪贴板')
}

// 处理部署完成
const handleDeployComplete = async () => {
  cloudttyLoading.value = true
  await checkCloudTTY()
  cloudttyLoading.value = false

  if (cloudttyInstalled.value) {
    ElMessage.success('CloudTTY 部署成功！')
    cloudttyDialogVisible.value = false
  } else {
    ElMessage.warning('未检测到 CloudTTY，请确认部署已完成')
  }
}

// 打开 CloudTTY (已废弃，使用对话框替代)
const openCloudTTY = () => {
  ElMessage.info('请手动部署 CloudTTY 或使用自动部署功能')
}

onMounted(() => {
  loadClusters()
  checkCloudTTY()
})
</script>

<style scoped>
.nodes-container {
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

.cloudtty-action-btn {
  background: linear-gradient(135deg, #D4AF37 0%, #B8860B 100%);
  color: #000000;
  border: none;
  font-weight: 600;
  padding: 12px 24px;
  border-radius: 8px;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  letter-spacing: 0.5px;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.3);
  margin-left: 12px;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(212, 175, 55, 0.5);
    background: linear-gradient(135deg, #E5C158 0%, #C9961C 100%);
  }

  &:active {
    transform: translateY(0);
    box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
  }

  :deep(.el-icon) {
    font-size: 16px;
  }
}

.cloudtty-dialog-content {
  .deploy-status {
    padding: 20px 0;
  }

  .deploy-methods {
    h4 {
      margin: 20px 0 10px 0;
      color: #303133;
    }

    .el-radio {
      display: block;
      margin-bottom: 15px;
      padding: 15px;
      border: 1px solid #dcdfe6;
      border-radius: 8px;
      transition: all 0.3s;

      &:hover {
        border-color: #D4AF37;
        background-color: rgba(212, 175, 55, 0.05);
      }
    }

    .code-block-wrapper {
      display: flex;
      border: 1px solid #d4af37;
      border-radius: 6px;
      overflow: hidden;
      background-color: #000000;
      margin-top: 10px;
    }

    .code-line-numbers {
      background-color: #0d0d0d;
      color: #666;
      padding: 16px 8px;
      text-align: right;
      font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
      font-size: 12px;
      line-height: 1.6;
      user-select: none;
      overflow: hidden;
      min-width: 40px;
      border-right: 1px solid #333;
    }

    .code-line-number {
      height: 19.2px;
      line-height: 1.6;
    }

    .code-textarea {
      flex: 1;
      background-color: #000000;
      color: #d4af37;
      border: none;
      outline: none;
      padding: 16px;
      font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
      font-size: 12px;
      line-height: 1.6;
      resize: none;
      min-height: 400px;
    }

    .code-textarea::placeholder {
      color: #555;
    }

    .code-textarea:focus {
      outline: none;
    }
  }
}

.cloudtty-terminal-wrapper {
  width: 100%;
  height: calc(100vh - 200px);
  background-color: #000000;
  border-radius: 4px;
  overflow: hidden;
}

.cloudtty-iframe {
  width: 100%;
  height: 100%;
  border: none;
}

.cloudtty-terminal-dialog {
  .el-dialog__body {
    padding: 0;
  }
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

/* 节点名称单元格 */
.node-name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.node-icon-wrapper {
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

.node-icon {
  color: #d4af37;
}

.node-name-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.node-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.link-text {
  color: #d4af37;
  cursor: pointer;
  transition: all 0.3s;
}

.link-text:hover {
  color: #bfa13f;
  text-decoration: underline;
}

.node-ip {
  font-size: 12px;
  color: #909399;
}

/* 角色标签 */
.role-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 14px;
  border-radius: 16px;
  font-size: 13px;
  font-weight: 500;
}

.role-master {
  background: transparent;
  color: #d4af37;
  border: 1px solid #d4af37;
}

.role-control-plane {
  background: transparent;
  color: #d4af37;
  border: 1px solid #d4af37;
}

.role-worker {
  background: transparent;
  color: #606266;
  border: 1px solid #dcdfe6;
}

/* 版本单元格 */
.version-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #606266;
}

.version-icon {
  color: #d4af37;
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

/* 资源单元格 */
.resource-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-icon {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.resource-icon-cpu {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.resource-icon-memory {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.resource-value {
  font-size: 13px;
  font-weight: 500;
  color: #303133;
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

/* 污点单元格 */
.taint-cell {
  display: flex;
  justify-content: center;
  align-items: center;
  cursor: pointer;
  padding: 5px 0;
}

.taint-badge-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.taint-icon {
  color: #d4af37;
  font-size: 20px;
  transition: all 0.3s;
}

.taint-count {
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

.taint-cell:hover .taint-icon {
  color: #bfa13f;
  transform: scale(1.1);
}

.taint-cell:hover .taint-count {
  background-color: #bfa13f;
  border-color: #bfa13f;
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

/* 污点弹窗 */
.taint-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.taint-dialog :deep(.el-dialog__title) {
  color: #d4af37;
  font-size: 16px;
  font-weight: 600;
}

.taint-dialog-content {
  padding: 8px 0;
}

/* 污点编辑模式 */
.taint-edit-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.taint-edit-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  border-radius: 8px;
  border: 1px solid #d4af37;
}

.taint-edit-info {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 15px;
  color: #d4af37;
  font-weight: 500;
}

.taint-edit-info .info-icon {
  font-size: 18px;
}

.taint-edit-count {
  font-size: 14px;
  color: #d4af37;
  padding: 6px 14px;
  background: rgba(212, 175, 55, 0.15);
  border-radius: 20px;
  font-weight: 500;
}

.taint-edit-list {
  max-height: 420px;
  overflow-y: auto;
  padding: 12px;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  border-radius: 8px;
  border: 1px solid #e0e0e0;
}

.taint-edit-row {
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

.taint-edit-row:hover {
  border-color: #d4af37;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.15);
  transform: translateY(-2px);
}

.taint-row-number {
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

.taint-row-content {
  flex: 1;
  min-width: 0;
}

.taint-input-group {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.taint-input-wrapper {
  flex: 1;
  min-width: 120px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.taint-input-label {
  font-size: 12px;
  color: #909399;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.taint-edit-input {
  width: 100%;
}

.taint-edit-input :deep(.el-input__wrapper) {
  background: #fafbfc;
  border: 1px solid #d0d0d0;
  border-radius: 6px;
  transition: all 0.3s;
}

.taint-edit-input :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  background: #fff;
}

.taint-edit-input :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  background: #fff;
  box-shadow: 0 0 0 2px rgba(212, 175, 55, 0.1);
}

.taint-separator {
  color: #909399;
  font-weight: 600;
  font-size: 14px;
  flex-shrink: 0;
}

.taint-effect-wrapper {
  min-width: 160px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.taint-effect-select {
  width: 100%;
}

.taint-effect-select :deep(.el-input__wrapper) {
  background: #fafbfc;
  border: 1px solid #d0d0d0;
  border-radius: 6px;
}

.taint-effect-select :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  background: #fff;
}

.effect-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}

.effect-desc {
  font-size: 12px;
  color: #909399;
}

.empty-taints {
  padding: 40px 20px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.empty-taints .empty-icon {
  font-size: 48px;
  color: #d4af37;
  opacity: 0.5;
}

.empty-taints p {
  font-size: 16px;
  color: #606266;
  margin: 0 0 4px 0;
}

.empty-taints span {
  font-size: 14px;
  color: #909399;
}

.add-taint-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  border: 2px dashed #d4af37;
  border-radius: 8px;
  transition: all 0.3s;
}

.add-taint-btn:hover {
  border-style: solid;
  border-color: #bfa13f;
  background: rgba(212, 175, 55, 0.05);
  transform: translateY(-2px);
}

/* 表格样式 */
.taint-table {
  width: 100%;
}

.taint-table :deep(.el-table__cell) {
  padding: 8px 0;
}

.taint-key-wrapper {
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

.taint-key-wrapper:hover {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%) !important;
  border-color: #bfa13f !important;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3) !important;
  transform: translateY(-1px);
}

.taint-key-wrapper:active {
  transform: translateY(0);
}

.taint-key-text {
  flex: 1;
  word-break: break-all;
  line-height: 1.4;
  white-space: pre-wrap;
}

.copy-icon {
  font-size: 14px;
  flex-shrink: 0;
  opacity: 0.6;
  transition: opacity 0.3s;
}

.taint-key-wrapper:hover .copy-icon {
  opacity: 1;
}

.taint-value {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #606266;
  word-break: break-all;
  white-space: pre-wrap;
}

.effect-tag {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  font-weight: 500;
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

/* Shell 终端对话框 */
.shell-dialog-content {
  padding: 0;
}

.terminal-container {
  width: 100%;
  height: 500px;
  background-color: #000000;
  border-radius: 4px;
  overflow: hidden;
}
</style>
