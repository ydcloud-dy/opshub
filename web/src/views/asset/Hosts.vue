<template>
  <div class="hosts-page-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Monitor /></el-icon>
        </div>
        <div>
          <h2 class="page-title">主机管理</h2>
          <p class="page-subtitle">管理所有服务器和主机资源，支持多种方式导入</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="handleOpenTerminal" class="terminal-button">
          <el-icon style="margin-right: 6px;"><Monitor /></el-icon>
          终端
        </el-button>
        <el-dropdown @command="handleImportCommand" class="import-dropdown">
          <el-button class="black-button">
            <el-icon style="margin-right: 6px;"><Plus /></el-icon>
            新增主机
            <el-icon style="margin-left: 6px;"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="direct">
                <el-icon><DocumentAdd /></el-icon>
                直接导入
              </el-dropdown-item>
              <el-dropdown-item command="excel">
                <el-icon><Upload /></el-icon>
                Excel导入
              </el-dropdown-item>
              <el-dropdown-item command="cloud">
                <el-icon><Cloudy /></el-icon>
                云主机导入
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- 主内容区域：左侧分组树 + 右侧主机列表 -->
    <div class="main-content">
      <!-- 左侧分组树 - 终端视图时隐藏 -->
      <div class="left-panel" v-show="activeView === 'hosts'">
        <div class="panel-header">
          <div class="panel-title">
            <el-icon class="panel-icon"><Collection /></el-icon>
            <span>资产分组</span>
          </div>
          <div class="panel-actions">
            <el-tooltip content="新增分组" placement="top">
              <el-button circle size="small" @click="handleAddGroup">
                <el-icon><Plus /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip :content="isExpandAll ? '折叠全部' : '展开全部'" placement="top">
              <el-button circle size="small" @click="toggleExpandAll">
                <el-icon><Sort /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
        <div class="panel-body">
          <el-input
            v-model="groupSearchKeyword"
            placeholder="搜索分组..."
            clearable
            size="small"
            class="group-search"
            @input="filterGroupTree"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <div class="tree-container" v-loading="groupLoading">
            <el-tree
              ref="groupTreeRef"
              :data="filteredGroupTree"
              :props="treeProps"
              :default-expand-all="false"
              :expand-on-click-node="false"
              :highlight-current="true"
              node-key="id"
              class="group-tree"
              @node-click="handleGroupClick"
            >
              <template #default="{ node, data }">
                <div class="tree-node">
                  <span class="node-icon">
                    <el-icon v-if="!data.parentId || data.parentId === 0" color="#67c23a">
                      <Collection />
                    </el-icon>
                    <el-icon v-else color="#409eff">
                      <Folder />
                    </el-icon>
                  </span>
                  <span class="node-label">{{ node.label }}</span>
                  <span class="node-count">({{ data.hostCount || 0 }})</span>
                  <span class="node-actions" @click.stop>
                    <el-dropdown trigger="click" @command="(cmd) => handleGroupAction(cmd, data)">
                      <el-icon class="more-icon"><MoreFilled /></el-icon>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item command="edit">
                            <el-icon><Edit /></el-icon> 编辑
                          </el-dropdown-item>
                          <el-dropdown-item command="delete" divided>
                            <el-icon><Delete /></el-icon> 删除
                          </el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </span>
                </div>
              </template>
            </el-tree>
            <el-empty v-if="filteredGroupTree.length === 0 && !groupLoading" description="暂无分组" :image-size="60" />
          </div>
        </div>
      </div>

      <!-- 右侧主机列表/终端 -->
      <div class="right-panel">
        <!-- 主机列表视图 -->
        <div v-show="activeView === 'hosts'" class="view-container">
          <!-- 搜索和筛选栏 -->
          <div class="filter-bar">
          <div class="filter-inputs">
            <el-input
              v-model="searchForm.keyword"
              placeholder="搜索主机名/IP..."
              clearable
              class="filter-input"
              @input="handleSearch"
            >
              <template #prefix>
                <el-icon class="search-icon"><Search /></el-icon>
              </template>
            </el-input>

            <el-select
              v-model="searchForm.status"
              placeholder="主机状态"
              clearable
              class="filter-input"
              @change="handleSearch"
            >
              <el-option label="在线" :value="1" />
              <el-option label="离线" :value="0" />
              <el-option label="未知" :value="-1" />
            </el-select>
          </div>

          <div class="filter-actions">
            <el-button
              v-if="selectedHosts.length > 0"
              type="danger"
              plain
              @click="handleBatchDelete"
            >
              <el-icon style="margin-right: 4px;"><Delete /></el-icon>
              批量删除 ({{ selectedHosts.length }})
            </el-button>
            <el-button class="reset-btn" @click="handleReset">
              <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
              重置
            </el-button>
            <el-button @click="loadHostList">
              <el-icon style="margin-right: 4px;"><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>

        <!-- 当前选择的分组 -->
        <div v-if="selectedGroup" class="selected-group-bar">
          <el-icon><FolderOpened /></el-icon>
          <span class="group-path">{{ getGroupPath(selectedGroup) }}</span>
          <el-tag size="small" type="info">{{ hostPagination.total }} 台主机</el-tag>
          <el-button link size="small" @click="clearGroupSelection">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>

        <!-- 主机列表 -->
        <div class="table-wrapper">
          <el-table
            :data="hostList"
            v-loading="hostLoading"
            class="modern-table"
            :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
            @selection-change="handleHostSelectionChange"
          >
            <el-table-column type="selection" width="55" fixed="left" />
            <el-table-column label="主机" prop="name" min-width="140" fixed="left">
              <template #default="{ row }">
                <div class="hostname-cell" @click="handleShowHostDetail(row)">
                  <div class="host-avatar" :class="`host-status-${row.status}`">
                    <el-icon><Monitor /></el-icon>
                  </div>
                  <div class="host-info">
                    <div class="hostname hostname-clickable">{{ row.name }}</div>
                    <div class="host-meta">
                      <span class="ip">{{ row.ip }}</span>
                      <span class="port">:{{ row.port }}</span>
                    </div>
                  </div>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="状态" width="70" align="center">
              <template #default="{ row }">
                <div class="status-cell">
                  <span class="status-dot" :class="`status-dot-${row.status}`"></span>
                  <span class="status-text" :class="`status-text-${row.status}`">{{ row.statusText }}</span>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="类型" width="90" align="center">
              <template #default="{ row }">
                <el-tag v-if="row.type === 'cloud'" :icon="Cloudy" size="small" type="warning">
                  {{ row.cloudProviderText || '云主机' }}
                </el-tag>
                <el-tag v-else :icon="Monitor" size="small" type="info">
                  自建
                </el-tag>
              </template>
            </el-table-column>

            <el-table-column label="CPU" min-width="100" align="center">
              <template #default="{ row }">
                <div class="resource-cell">
                  <div v-if="row.cpuCores" class="resource-info">
                    <span class="resource-label">{{ row.cpuCores }}核</span>
                    <el-progress
                      :percentage="row.cpuUsage ? parseFloat(row.cpuUsage.toFixed(1)) : 0"
                      :color="getUsageColor(row.cpuUsage)"
                      :stroke-width="5"
                      :show-text="true"
                    />
                  </div>
                  <span v-else class="text-muted">-</span>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="内存" min-width="120" align="center">
              <template #default="{ row }">
                <div class="resource-cell">
                  <div v-if="row.memoryTotal" class="resource-info">
                    <span class="resource-label resource-compact">{{ formatBytesCompact(row.memoryUsed) }} / {{ formatBytesCompact(row.memoryTotal) }}</span>
                    <el-progress
                      :percentage="row.memoryUsage ? parseFloat(row.memoryUsage.toFixed(1)) : 0"
                      :color="getUsageColor(row.memoryUsage)"
                      :stroke-width="5"
                      :show-text="true"
                    />
                  </div>
                  <span v-else class="text-muted">-</span>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="磁盘" min-width="120" align="center">
              <template #default="{ row }">
                <div class="resource-cell">
                  <div v-if="row.diskTotal" class="resource-info">
                    <span class="resource-label resource-compact">{{ formatBytesCompact(row.diskUsed) }} / {{ formatBytesCompact(row.diskTotal) }}</span>
                    <el-progress
                      :percentage="row.diskUsage ? parseFloat(row.diskUsage.toFixed(1)) : 0"
                      :color="getUsageColor(row.diskUsage)"
                      :stroke-width="5"
                      :show-text="true"
                    />
                  </div>
                  <span v-else class="text-muted">-</span>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="标签" min-width="80">
              <template #default="{ row }">
                <div v-if="row.tags && row.tags.length > 0" class="tags-cell">
                  <el-tag
                    v-for="(tag, index) in row.tags.slice(0, 2)"
                    :key="index"
                    size="small"
                    class="tag-item"
                  >
                    {{ tag }}
                  </el-tag>
                  <el-tag v-if="row.tags.length > 2" size="small" type="info" class="tag-more">
                    +{{ row.tags.length - 2 }}
                  </el-tag>
                </div>
                <span v-else class="text-muted">-</span>
              </template>
            </el-table-column>

            <el-table-column label="系统信息" min-width="120" show-overflow-tooltip>
              <template #default="{ row }">
                <div v-if="row.os || row.arch" class="config-cell">
                  <div v-if="row.os" class="config-item">
                    <el-icon><Platform /></el-icon>
                    <span class="config-text">{{ row.os }}</span>
                  </div>
                  <div v-if="row.arch" class="config-item">
                    <el-icon><Cpu /></el-icon>
                    <span class="config-text">{{ row.arch }}</span>
                  </div>
                </div>
                <span v-else class="text-muted">-</span>
              </template>
            </el-table-column>

            <el-table-column label="操作" width="170" fixed="right" align="center">
              <template #default="{ row }">
                <div class="action-buttons">
                  <el-tooltip content="采集信息" placement="top">
                    <el-button
                      v-if="hasHostPermission(row.id, PERMISSION.COLLECT)"
                      link
                      class="action-btn action-refresh"
                      @click="handleCollectHost(row)"
                    >
                      <el-icon><Refresh /></el-icon>
                    </el-button>
                  </el-tooltip>
                  <el-tooltip content="文件管理" placement="top">
                    <el-button
                      v-if="hasHostPermission(row.id, PERMISSION.FILE)"
                      link
                      class="action-btn action-files"
                      @click="handleFileManager(row)"
                    >
                      <el-icon><Folder /></el-icon>
                    </el-button>
                  </el-tooltip>
                  <el-tooltip content="编辑" placement="top">
                    <el-button
                      v-if="hasHostPermission(row.id, PERMISSION.EDIT)"
                      link
                      class="action-btn action-edit"
                      @click="handleEditHost(row)"
                    >
                      <el-icon><Edit /></el-icon>
                    </el-button>
                  </el-tooltip>
                  <el-tooltip content="删除" placement="top">
                    <el-button
                      v-if="hasHostPermission(row.id, PERMISSION.DELETE)"
                      link
                      class="action-btn action-delete"
                      @click="handleDeleteHost(row)"
                    >
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </el-tooltip>
                </div>
              </template>
            </el-table-column>
          </el-table>

          <!-- 分页 -->
          <div class="pagination-wrapper">
            <el-pagination
              v-model:current-page="hostPagination.page"
              v-model:page-size="hostPagination.pageSize"
              :total="hostPagination.total"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="loadHostList"
              @current-change="loadHostList"
            />
          </div>
        </div>
        </div>

        <!-- 终端视图 -->
        <div v-show="activeView === 'terminal'" class="view-container terminal-view">
          <div class="terminal-view-header">
            <div class="terminal-view-title">
              <el-icon><Monitor /></el-icon>
              <span>Web终端</span>
              <span v-if="activeTerminalHost" class="terminal-current-group">
                / {{ activeTerminalHost.name }}
              </span>
            </div>
            <el-button size="small" @click="switchToHostsView">
              <el-icon style="margin-right: 4px;"><ArrowLeft /></el-icon>
              返回主机列表
            </el-button>
          </div>
          <div class="terminal-content">
            <!-- 资产分组树（左） -->
            <div class="terminal-sidebar">
              <div class="panel-header">
                <div class="panel-title">
                  <el-icon class="panel-icon"><Collection /></el-icon>
                  <span>资产分组</span>
                </div>
                <div class="panel-actions">
                  <el-tooltip :content="isExpandAll ? '折叠全部' : '展开全部'" placement="top">
                    <el-button circle size="small" @click="toggleExpandAll">
                      <el-icon><Sort /></el-icon>
                    </el-button>
                  </el-tooltip>
                </div>
              </div>
              <div class="panel-body">
                <el-input
                  v-model="groupSearchKeyword"
                  placeholder="搜索分组..."
                  clearable
                  size="small"
                  class="group-search"
                  @input="filterGroupTree"
                >
                  <template #prefix>
                    <el-icon><Search /></el-icon>
                  </template>
                </el-input>
                <div class="tree-container" v-loading="groupLoading">
                  <el-tree
                    ref="groupTreeRef"
                    :data="terminalGroupTree"
                    :props="treeProps"
                    :default-expand-all="false"
                    :expand-on-click-node="false"
                    :highlight-current="true"
                    node-key="id"
                    class="group-tree"
                  >
                    <template #default="{ node, data }">
                      <div class="tree-node" @dblclick="handleHostDblClick(data)" :style="{ cursor: data.type === 'host' ? 'pointer' : 'default' }">
                        <span class="node-icon">
                          <el-icon v-if="data.type === 'group' || !data.parentId || data.parentId === 0" color="#67c23a">
                            <Collection />
                          </el-icon>
                          <el-icon v-else-if="data.type === 'host'" :color="getStatusColor(data.status)">
                            <Monitor />
                          </el-icon>
                          <el-icon v-else color="#409eff">
                            <Folder />
                          </el-icon>
                        </span>
                        <span class="node-label">{{ node.label }}</span>
                        <span v-if="data.type === 'group' || !data.type" class="node-count">({{ data.hostCount || 0 }})</span>
                        <span v-if="data.type === 'host'" class="node-ip">{{ data.ip }}</span>
                      </div>
                    </template>
                  </el-tree>
                  <el-empty v-if="!terminalGroupTree || terminalGroupTree.length === 0" description="暂无数据" :image-size="60" />
                </div>
              </div>
            </div>

            <!-- 终端区域（右） -->
            <div class="terminal-main">
              <div v-if="activeTerminalHost" class="terminal-header">
                <div class="terminal-info">
                  <el-icon class="terminal-icon"><Monitor /></el-icon>
                  <div class="terminal-details">
                    <div class="terminal-title">{{ activeTerminalHost.name }}</div>
                    <div class="terminal-meta">
                      <span class="terminal-ip">{{ activeTerminalHost.ip }}:{{ activeTerminalHost.port }}</span>
                      <span class="terminal-user">{{ activeTerminalHost.sshUser }}</span>
                    </div>
                  </div>
                </div>
                <div class="terminal-actions">
                  <el-button size="small" @click="closeTerminal" :icon="Close">关闭</el-button>
                </div>
              </div>
              <div v-else class="terminal-placeholder">
                <el-icon class="placeholder-icon"><Monitor /></el-icon>
                <div class="placeholder-text">请双击左侧主机连接终端</div>
                <div class="placeholder-hint">展开分组查看主机，双击主机即可连接</div>
              </div>
              <div v-if="activeTerminalHost" class="terminal-body">
                <div class="terminal-wrapper">
                  <div ref="terminalRef" class="xterm-container"></div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 直接导入对话框 -->
    <el-dialog
      v-model="directImportVisible"
      :title="hostForm.id && hostForm.id > 0 ? '编辑主机' : '新增主机'"
      width="60%"
      class="host-import-dialog responsive-dialog"
      @close="handleDirectImportClose"
    >
      <el-form :model="hostForm" :rules="hostRules" ref="hostFormRef" label-width="120px">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="主机名称" prop="name">
              <el-input v-model="hostForm.name" placeholder="请输入主机名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="主机类型" prop="type">
              <el-select v-model="hostForm.type" placeholder="请选择主机类型" style="width: 100%">
                <el-option label="自建主机" value="self">
                  <div style="display: flex; align-items: center; gap: 8px;">
                    <el-icon><Monitor /></el-icon>
                    <span>自建主机</span>
                  </div>
                </el-option>
                <el-option label="云主机" value="cloud">
                  <div style="display: flex; align-items: center; gap: 8px;">
                    <el-icon><Cloudy /></el-icon>
                    <span>云主机</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="所属分组" prop="groupId">
              <el-tree-select
                v-model="hostForm.groupId"
                :data="groupTreeOptions"
                :props="{ value: 'id', label: 'name', children: 'children' }"
                clearable
                check-strictly
                placeholder="请选择分组"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12"></el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="IP地址" prop="ip">
              <el-input v-model="hostForm.ip" placeholder="请输入IP地址" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="SSH端口" prop="port">
              <el-input-number v-model="hostForm.port" :min="1" :max="65535" :step="1" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="SSH用户名" prop="sshUser">
              <el-input v-model="hostForm.sshUser" placeholder="如：root" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="认证凭据">
              <el-select v-model="hostForm.credentialId" placeholder="选择或新建凭证" clearable filterable>
                <el-option
                  v-for="cred in credentialList"
                  :key="cred.id"
                  :label="`${cred.name} (${cred.typeText})`"
                  :value="cred.id"
                />
                <template #footer>
                  <el-button text @click="showCredentialDialog = true" style="width: 100%">
                    <el-icon><Plus /></el-icon> 新建凭证
                  </el-button>
                </template>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="24">
            <el-form-item label="主机标签">
              <el-input v-model="hostForm.tags" placeholder="多个标签用逗号分隔，如：web,prod" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="24">
            <el-form-item label="备注">
              <el-input v-model="hostForm.description" type="textarea" :rows="3" placeholder="请输入备注信息" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="directImportVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleDirectImportSubmit" :loading="hostSubmitting">
            {{ hostForm.id && hostForm.id > 0 ? '保存' : '确定' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 主机详情对话框 -->
    <el-dialog
      v-model="showHostDetailDialog"
      title=""
      width="65%"
      class="host-detail-dialog"
      @close="handleCloseHostDetail"
    >
      <template #header="{ close }">
        <div class="detail-dialog-header">
          <div class="detail-header-left">
            <div class="host-avatar-lg" :class="`host-status-${hostDetail?.status}`">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="detail-header-info">
              <div class="detail-hostname">{{ hostDetail?.name }}</div>
              <div class="detail-hostmeta">
                <span class="detail-ip">{{ hostDetail?.ip }}:{{ hostDetail?.port }}</span>
                <el-tag :type="getStatusType(hostDetail?.status)" size="small">{{ hostDetail?.statusText }}</el-tag>
              </div>
            </div>
          </div>
          <el-button link @click="close"><el-icon><Close /></el-icon></el-button>
        </div>
      </template>
      <div v-loading="hostDetailLoading" class="host-detail-content">
        <template v-if="hostDetail">
          <!-- 信息网格 -->
          <div class="info-grid">
            <!-- 基本信息 -->
            <div class="info-card">
              <div class="info-card-header">
                <div class="info-icon info-icon-basic">
                  <el-icon><InfoFilled /></el-icon>
                </div>
                <span class="info-card-title">基本信息</span>
              </div>
              <div class="info-card-body">
                <div class="info-row">
                  <span class="info-label">主机类型</span>
                  <span class="info-value">
                    <el-tag v-if="hostDetail.type === 'cloud'" :icon="Cloudy" size="small" type="warning">
                      {{ hostDetail.cloudProviderText || '云主机' }}
                    </el-tag>
                    <el-tag v-else :icon="Monitor" size="small" type="info">
                      自建主机
                    </el-tag>
                  </span>
                </div>
                <div class="info-row">
                  <span class="info-label">SSH用户</span>
                  <span class="info-value">{{ hostDetail.sshUser }}</span>
                </div>
                <div class="info-row">
                  <span class="info-label">分组</span>
                  <span class="info-value">{{ hostDetail.groupName || '未分组' }}</span>
                </div>
                <div class="info-row">
                  <span class="info-label">最后连接</span>
                  <span class="info-value">{{ hostDetail.lastSeen || '未连接' }}</span>
                </div>
                <div class="info-row" v-if="hostDetail.createTime">
                  <span class="info-label">创建时间</span>
                  <span class="info-value">{{ hostDetail.createTime }}</span>
                </div>
              </div>
            </div>

            <!-- 系统信息 -->
            <div class="info-card">
              <div class="info-card-header">
                <div class="info-icon info-icon-system">
                  <el-icon><Platform /></el-icon>
                </div>
                <span class="info-card-title">系统信息</span>
              </div>
              <div class="info-card-body">
                <div class="info-row">
                  <span class="info-label">操作系统</span>
                  <span class="info-value">{{ hostDetail.os || '-' }}</span>
                </div>
                <div class="info-row">
                  <span class="info-label">内核版本</span>
                  <span class="info-value">{{ hostDetail.kernel || '-' }}</span>
                </div>
                <div class="info-row">
                  <span class="info-label">系统架构</span>
                  <span class="info-value">{{ hostDetail.arch || '-' }}</span>
                </div>
                <div class="info-row">
                  <span class="info-label">主机名</span>
                  <span class="info-value">{{ hostDetail.hostname || '-' }}</span>
                </div>
              </div>
            </div>

            <!-- 认证信息 -->
            <div class="info-card" v-if="hostDetail.credential">
              <div class="info-card-header">
                <div class="info-icon info-icon-auth">
                  <el-icon><Lock /></el-icon>
                </div>
                <span class="info-card-title">认证信息</span>
              </div>
              <div class="info-card-body">
                <div class="info-row">
                  <span class="info-label">凭证名称</span>
                  <span class="info-value">{{ hostDetail.credential.name }}</span>
                </div>
                <div class="info-row">
                  <span class="info-label">认证方式</span>
                  <el-tag :type="hostDetail.credential.type === 'password' ? 'warning' : 'success'" size="small">
                    {{ hostDetail.credential.typeText }}
                  </el-tag>
                </div>
                <div class="info-row" v-if="hostDetail.credential.username">
                  <span class="info-label">用户名</span>
                  <span class="info-value">{{ hostDetail.credential.username }}</span>
                </div>
              </div>
            </div>

            <!-- 标签 -->
            <div class="info-card" v-if="hostDetail.tags && hostDetail.tags.length > 0">
              <div class="info-card-header">
                <div class="info-icon info-icon-tags">
                  <el-icon><Collection /></el-icon>
                </div>
                <span class="info-card-title">标签</span>
              </div>
              <div class="info-card-body">
                <div class="tags-list">
                  <el-tag v-for="(tag, index) in hostDetail.tags" :key="index" size="large">
                    {{ tag }}
                  </el-tag>
                </div>
              </div>
            </div>
          </div>

          <!-- 资源信息 -->
          <div class="resource-section" v-if="hostDetail.cpuCores || hostDetail.memoryTotal">
            <div class="info-card resource-card-wrapper">
              <div class="info-card-header">
                <div class="info-icon info-icon-resource">
                  <el-icon><DataLine /></el-icon>
                </div>
                <span class="info-card-title">资源信息</span>
              </div>
              <div class="info-card-body">
                <div class="resource-grid">
                  <!-- CPU -->
                  <div class="resource-card">
                    <div class="resource-card-header">
                      <el-icon class="resource-icon cpu-icon"><Cpu /></el-icon>
                      <span>CPU</span>
                    </div>
                    <div class="resource-card-body">
                      <div class="resource-value">{{ hostDetail.cpuCores }}核</div>
                      <div class="resource-usage" :class="`usage-${getUsageLevel(hostDetail.cpuUsage)}`">
                        <el-progress
                          :percentage="hostDetail.cpuUsage ? parseFloat(hostDetail.cpuUsage.toFixed(1)) : 0"
                          :color="getUsageColor(hostDetail.cpuUsage)"
                          :stroke-width="8"
                          :show-text="false"
                        />
                        <span class="usage-text" :class="getUsageLevel(hostDetail.cpuUsage)">{{ hostDetail.cpuUsage ? hostDetail.cpuUsage.toFixed(1) : '-' }}%</span>
                      </div>
                    </div>
                  </div>
                  <!-- 内存 -->
                  <div class="resource-card">
                    <div class="resource-card-header">
                      <el-icon class="resource-icon memory-icon"><Coin /></el-icon>
                      <span>内存</span>
                    </div>
                    <div class="resource-card-body">
                      <div class="resource-value">{{ formatBytes(hostDetail.memoryTotal) }}</div>
                      <div class="resource-usage" :class="`usage-${getUsageLevel(hostDetail.memoryUsage)}`">
                        <el-progress
                          :percentage="hostDetail.memoryUsage ? parseFloat(hostDetail.memoryUsage.toFixed(1)) : 0"
                          :color="getUsageColor(hostDetail.memoryUsage)"
                          :stroke-width="8"
                          :show-text="false"
                        />
                        <span class="usage-text" :class="getUsageLevel(hostDetail.memoryUsage)">{{ hostDetail.memoryUsage ? hostDetail.memoryUsage.toFixed(1) : '-' }}%</span>
                      </div>
                    </div>
                  </div>
                  <!-- 磁盘 -->
                  <div class="resource-card">
                    <div class="resource-card-header">
                      <el-icon class="resource-icon disk-icon"><Files /></el-icon>
                      <span>磁盘</span>
                    </div>
                    <div class="resource-card-body">
                      <div class="resource-value">{{ formatBytes(hostDetail.diskTotal) }}</div>
                      <div class="resource-usage" :class="`usage-${getUsageLevel(hostDetail.diskUsage)}`">
                        <el-progress
                          :percentage="hostDetail.diskUsage ? parseFloat(hostDetail.diskUsage.toFixed(1)) : 0"
                          :color="getUsageColor(hostDetail.diskUsage)"
                          :stroke-width="8"
                          :show-text="false"
                        />
                        <span class="usage-text" :class="getUsageLevel(hostDetail.diskUsage)">{{ hostDetail.diskUsage ? hostDetail.diskUsage.toFixed(1) : '-' }}%</span>
                      </div>
                    </div>
                  </div>
                </div>
                <div class="resource-legend">
                  <span class="legend-item"><span class="legend-dot legend-low"></span>正常 &lt;70%</span>
                  <span class="legend-item"><span class="legend-dot legend-high"></span>繁忙 ≥70%</span>
                  <span class="legend-item"><span class="legend-dot legend-critical"></span>严重 ≥90%</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 备注 -->
          <div class="remark-section" v-if="hostDetail.description">
            <div class="remark-header">
              <div class="info-icon info-icon-remark">
                <el-icon><Document /></el-icon>
              </div>
              <span class="section-title">备注</span>
            </div>
            <div class="remark-content">{{ hostDetail.description }}</div>
          </div>
        </template>
      </div>
      <template #footer>
        <el-button @click="showHostDetailDialog = false">关闭</el-button>
        <el-button type="primary" @click="handleCollectHostFromDetail" :loading="hostDetailLoading">
          <el-icon><Refresh /></el-icon>
          采集信息
        </el-button>
      </template>
    </el-dialog>

    <!-- Excel导入对话框 -->
    <el-dialog
      v-model="excelImportVisible"
      title="Excel批量导入"
      width="50%"
      class="excel-import-dialog responsive-dialog"
      @close="handleExcelImportClose"
    >
      <div class="excel-import-content">
        <el-alert title="导入说明" type="info" :closable="false" style="margin-bottom: 20px;">
          <ul style="margin: 8px 0 0 0; padding-left: 20px;">
            <li>请先下载Excel模板文件</li>
            <li>按照模板格式填写主机信息</li>
            <li>支持批量导入多台主机</li>
            <li>IP地址重复的主机会自动跳过</li>
          </ul>
        </el-alert>

        <el-form :model="excelImportForm" label-width="120px">
          <el-form-item label="主机类型">
            <el-select v-model="excelImportForm.type" placeholder="请选择主机类型" style="width: 100%">
              <el-option label="自建主机" value="self">
                <div style="display: flex; align-items: center; gap: 8px;">
                  <el-icon><Monitor /></el-icon>
                  <span>自建主机</span>
                </div>
              </el-option>
              <el-option label="云主机" value="cloud">
                <div style="display: flex; align-items: center; gap: 8px;">
                  <el-icon><Cloudy /></el-icon>
                  <span>云主机</span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>

          <el-form-item label="所属分组">
            <el-tree-select
              v-model="excelImportForm.groupId"
              :data="groupTreeOptions"
              :props="{ value: 'id', label: 'name', children: 'children' }"
              clearable
              check-strictly
              placeholder="请选择默认分组"
            />
          </el-form-item>

          <el-form-item label="下载模板">
            <el-button @click="downloadTemplate">
              <el-icon style="margin-right: 6px;"><Download /></el-icon>
              下载Excel模板
            </el-button>
          </el-form-item>

          <el-form-item label="上传文件">
            <el-upload
              ref="uploadRef"
              class="upload-demo"
              drag
              action="#"
              :auto-upload="false"
              :on-change="handleFileChange"
              :limit="1"
              accept=".xlsx,.xls"
            >
              <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
              <div class="el-upload__text">
                将文件拖到此处，或<em>点击上传</em>
              </div>
              <template #tip>
                <div class="el-upload__tip">
                  只支持 .xlsx 或 .xls 格式的Excel文件
                </div>
              </template>
            </el-upload>
          </el-form-item>

          <el-form-item v-if="uploadedFile" label="已选择文件">
            <div class="file-info">
              <el-icon><Document /></el-icon>
              <span>{{ uploadedFile.name }}</span>
              <el-tag size="small" type="success">{{ (uploadedFile.size / 1024).toFixed(2) }} KB</el-tag>
            </div>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="excelImportVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleExcelImportSubmit" :loading="excelImporting">开始导入</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 云主机导入对话框 -->
    <el-dialog
      v-model="cloudImportVisible"
      title="云主机导入"
      width="70%"
      class="cloud-import-dialog responsive-dialog"
      @close="handleCloudImportClose"
    >
      <el-steps :active="cloudImportStep" align-center style="margin-bottom: 30px;">
        <el-step title="选择云平台" />
        <el-step title="选择主机" />
        <el-step title="确认导入" />
      </el-steps>

      <!-- 步骤1: 选择云平台 -->
      <div v-if="cloudImportStep === 0" class="cloud-step-content">
        <el-form :model="cloudImportForm" label-width="120px">
          <el-form-item label="云平台">
            <el-select v-model="cloudImportForm.accountId" placeholder="请选择云平台账号" filterable>
              <el-option
                v-for="account in enabledCloudAccounts"
                :key="account.id"
                :label="`${account.name} (${account.providerText})`"
                :value="account.id"
              >
                <div style="display: flex; justify-content: space-between; align-items: center;">
                  <span>{{ account.name }}</span>
                  <el-tag size="small" :type="account.provider === 'aliyun' ? 'warning' : 'primary'">
                    {{ account.providerText }}
                  </el-tag>
                </div>
              </el-option>
              <template #footer>
                <el-button text @click="showCloudAccountDialog = true" style="width: 100%">
                  <el-icon><Plus /></el-icon> 新增云平台账号
                </el-button>
              </template>
            </el-select>
          </el-form-item>

          <el-form-item label="区域">
            <el-select v-model="cloudImportForm.region" placeholder="请先选择云平台账号" filterable :loading="loadingCloudRegions">
              <el-option v-for="region in cloudRegions" :key="region.value" :label="region.label" :value="region.value" />
            </el-select>
          </el-form-item>

          <el-form-item label="导入到分组">
            <el-tree-select
              v-model="cloudImportForm.groupId"
              :data="groupTreeOptions"
              :props="{ value: 'id', label: 'name', children: 'children' }"
              clearable
              check-strictly
              placeholder="请选择分组"
            />
          </el-form-item>
        </el-form>

        <el-alert title="提示" type="info" :closable="false">
          <ul style="margin: 8px 0 0 0; padding-left: 20px;">
            <li>请先添加云平台账号（Access Key / Secret Key）</li>
            <li>系统将自动获取该账号下指定区域的ECS实例</li>
            <li>支持阿里云、腾讯云等主流云厂商</li>
          </ul>
        </el-alert>
      </div>

      <!-- 步骤2: 选择主机 -->
      <div v-if="cloudImportStep === 1" class="cloud-step-content">
        <div class="step-header">
          <span>找到 {{ cloudHostList.length }} 台云主机，请选择要导入的主机</span>
          <el-checkbox v-model="selectAllCloudHosts" @change="handleSelectAllCloudHosts">全选</el-checkbox>
        </div>

        <el-table
          :data="cloudHostList"
          v-loading="loadingCloudHosts"
          @selection-change="handleCloudHostSelectionChange"
          max-height="400"
        >
          <el-table-column type="selection" width="55" />
          <el-table-column label="实例名称" prop="name" min-width="150" />
          <el-table-column label="实例ID" prop="instanceId" min-width="180" />
          <el-table-column label="公网IP" prop="publicIp" min-width="140" />
          <el-table-column label="私网IP" prop="privateIp" min-width="140" />
          <el-table-column label="操作系统" prop="os" min-width="150" show-overflow-tooltip />
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.status === 'Running' ? 'success' : 'info'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>

        <el-empty v-if="cloudHostList.length === 0 && !loadingCloudHosts" description="暂无云主机数据" />
      </div>

      <!-- 步骤3: 确认导入 -->
      <div v-if="cloudImportStep === 2" class="cloud-step-content">
        <el-result icon="success" title="准备就绪" sub-title="以下主机将被导入到系统中">
          <template #extra>
            <div class="import-summary">
              <el-descriptions :column="1" border>
                <el-descriptions-item label="云平台账号">{{ selectedCloudAccount?.name }}</el-descriptions-item>
                <el-descriptions-item label="目标分组">{{ selectedCloudGroup?.name }}</el-descriptions-item>
                <el-descriptions-item label="待导入主机">{{ selectedCloudHosts.length }} 台</el-descriptions-item>
              </el-descriptions>

              <div class="host-list-preview">
                <h4>主机列表：</h4>
                <el-tag
                  v-for="host in selectedCloudHosts"
                  :key="host.instanceId"
                  style="margin: 4px;"
                >
                  {{ host.name }}
                </el-tag>
              </div>
            </div>
          </template>
        </el-result>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button v-if="cloudImportStep > 0" @click="cloudImportStep--">上一步</el-button>
          <el-button @click="cloudImportVisible = false">取消</el-button>
          <el-button
            v-if="cloudImportStep === 0"
            class="black-button"
            @click="handleGetCloudHosts"
            :loading="loadingCloudHosts"
          >
            下一步
          </el-button>
          <el-button
            v-if="cloudImportStep === 1"
            class="black-button"
            @click="cloudImportStep++"
            :disabled="selectedCloudHosts.length === 0"
          >
            下一步
          </el-button>
          <el-button
            v-if="cloudImportStep === 2"
            class="black-button"
            @click="handleCloudImportSubmit"
            :loading="cloudImporting"
          >
            开始导入
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 新建凭证对话框 -->
    <el-dialog
      v-model="showCredentialDialog"
      title="新建凭证"
      width="50%"
      class="credential-dialog responsive-dialog"
      @close="handleCredentialDialogClose"
    >
      <el-form :model="credentialForm" :rules="credentialRules" ref="credentialFormRef" label-width="120px">
        <el-form-item label="凭证名称" prop="name">
          <el-input v-model="credentialForm.name" placeholder="请输入凭证名称，如：生产环境root凭证" />
        </el-form-item>

        <el-form-item label="认证方式" prop="type">
          <el-radio-group v-model="credentialForm.type" @change="handleAuthTypeChange">
            <el-radio label="password">密码认证</el-radio>
            <el-radio label="key">密钥认证</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="credentialForm.type === 'password'" label="用户名" prop="username">
          <el-input v-model="credentialForm.username" placeholder="如：root" />
        </el-form-item>

        <el-form-item v-if="credentialForm.type === 'password'" label="密码" prop="password">
          <el-input v-model="credentialForm.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>

        <el-form-item v-if="credentialForm.type === 'key'" label="用户名">
          <el-input v-model="credentialForm.username" placeholder="如：root（可选）" />
        </el-form-item>

        <el-form-item v-if="credentialForm.type === 'key'" label="私钥" prop="privateKey">
          <el-input
            v-model="credentialForm.privateKey"
            type="textarea"
            :rows="8"
            placeholder="请粘贴PEM格式的私钥内容"
          />
        </el-form-item>

        <el-form-item v-if="credentialForm.type === 'key'" label="私钥密码">
          <el-input v-model="credentialForm.passphrase" type="password" placeholder="如果私钥有密码请输入（可选）" show-password />
        </el-form-item>

        <el-form-item label="备注">
          <el-input v-model="credentialForm.description" type="textarea" :rows="2" placeholder="请输入备注信息" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showCredentialDialog = false">取消</el-button>
          <el-button class="black-button" @click="handleCredentialSubmit" :loading="credentialSubmitting">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 新增云平台账号对话框 -->
    <el-dialog
      v-model="showCloudAccountDialog"
      title="新增云平台账号"
      width="50%"
      class="cloud-account-dialog responsive-dialog"
      @close="handleCloudAccountDialogClose"
    >
      <el-form :model="cloudAccountForm" :rules="cloudAccountRules" ref="cloudAccountFormRef" label-width="120px">
        <el-form-item label="账号名称" prop="name">
          <el-input v-model="cloudAccountForm.name" placeholder="请输入账号名称，如：阿里云生产账号" />
        </el-form-item>

        <el-form-item label="云厂商" prop="provider">
          <el-select v-model="cloudAccountForm.provider" placeholder="请选择云厂商">
            <el-option label="阿里云" value="aliyun" />
            <el-option label="腾讯云" value="tencent" />
            <el-option label="AWS" value="aws" />
            <el-option label="华为云" value="huawei" />
          </el-select>
        </el-form-item>

        <el-form-item label="Access Key" prop="accessKey">
          <el-input v-model="cloudAccountForm.accessKey" placeholder="请输入Access Key ID" />
        </el-form-item>

        <el-form-item label="Secret Key" prop="secretKey">
          <el-input v-model="cloudAccountForm.secretKey" type="password" placeholder="请输入Access Key Secret" show-password />
        </el-form-item>

        <el-form-item label="默认区域">
          <el-input v-model="cloudAccountForm.region" placeholder="如：cn-hangzhou（可选）" />
        </el-form-item>

        <el-form-item label="备注">
          <el-input v-model="cloudAccountForm.description" type="textarea" :rows="2" placeholder="请输入备注信息" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showCloudAccountDialog = false">取消</el-button>
          <el-button class="black-button" @click="handleCloudAccountSubmit" :loading="cloudAccountSubmitting">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 新增/编辑分组对话框 -->
    <el-dialog
      v-model="groupDialogVisible"
      :title="groupDialogTitle"
      width="50%"
      class="group-edit-dialog responsive-dialog"
      @close="handleGroupDialogClose"
    >
      <el-form :model="groupForm" :rules="groupRules" ref="groupFormRef" label-width="100px">
        <el-form-item label="上级分组">
          <el-tree-select
            v-model="groupForm.parentId"
            :data="groupTreeOptions"
            :props="{ value: 'id', label: 'name', children: 'children' }"
            clearable
            check-strictly
            placeholder="不选择则为顶级分组"
          />
        </el-form-item>
        <el-form-item label="分组名称" prop="name">
          <el-input v-model="groupForm.name" placeholder="请输入分组名称" />
        </el-form-item>
        <el-form-item label="分组编码" prop="code">
          <el-input v-model="groupForm.code" placeholder="请输入分组编码" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="groupForm.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="groupForm.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-radio-group v-model="groupForm.status">
            <el-radio :label="1">正常</el-radio>
            <el-radio :label="0">停用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="groupDialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleGroupSubmit" :loading="groupSubmitting">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 文件浏览器对话框 -->
    <HostFileBrowser
      v-model:visible="fileBrowserVisible"
      :hostId="selectedHostId"
      :hostName="selectedHostName"
    />

  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch, nextTick, onBeforeUnmount } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import 'xterm/css/xterm.css'
import 'xterm/lib/xterm.js'
import { ElMessage, ElMessageBox, FormInstance, FormRules, UploadFile, UploadProps } from 'element-plus'
import {
  Plus,
  Edit,
  Delete,
  Monitor,
  Search,
  Refresh,
  RefreshLeft,
  Collection,
  Folder,
  FolderOpened,
  Sort,
  Close,
  MoreFilled,
  ArrowDown,
  ArrowLeft,
  DocumentAdd,
  Upload,
  Cloudy,
  Download,
  Document,
  UploadFilled,
  Platform,
  Operation,
  Cpu,
  Lock,
  Key,
  DataLine,
  InfoFilled,
  Coin,
  Files
} from '@element-plus/icons-vue'
import HostFileBrowser from './components/HostFileBrowser.vue'
import {
  getGroupTree,
  createGroup,
  updateGroup,
  deleteGroup
} from '@/api/assetGroup'
import {
  getHostList,
  getHost,
  createHost,
  updateHost,
  deleteHost,
  getCredentials,
  createCredential,
  getCloudAccounts,
  createCloudAccount,
  importFromCloud,
  getCloudRegions,
  getCloudInstances,
  collectHostInfo,
  testHostConnection,
  batchCollectHostInfo,
  downloadExcelTemplate,
  importFromExcel,
  batchDeleteHosts
} from '@/api/host'
import type { CloudInstanceVO, CloudRegionVO } from '@/api/host'
import { PERMISSION, hasPermission } from '@/utils/permission'
import { getUserHostPermissions } from '@/api/assetPermission'

// 加载状态
const groupLoading = ref(false)
const hostLoading = ref(false)
const hostSubmitting = ref(false)
const excelImporting = ref(false)
const cloudImporting = ref(false)
const loadingCloudHosts = ref(false)
const credentialSubmitting = ref(false)
const cloudAccountSubmitting = ref(false)
const groupSubmitting = ref(false)

// 视图状态
const activeView = ref('hosts') // 'hosts' | 'terminal'
const activeTerminalHost = ref<any>(null)
const terminalHostList = ref<any[]>([])
const terminalSearchKeyword = ref('')

// 终端相关
const terminalRef = ref<HTMLElement | null>(null)
const terminal = ref<Terminal | null>(null)
const fitAddon = ref<FitAddon | null>(null)
const ws = ref<WebSocket | null>(null)

// 对话框状态
const directImportVisible = ref(false)
const excelImportVisible = ref(false)
const cloudImportVisible = ref(false)
const showCredentialDialog = ref(false)
const showCloudAccountDialog = ref(false)
const fileBrowserVisible = ref(false)
const selectedHostId = ref(0)
const selectedHostName = ref('')

// 主机详情
const showHostDetailDialog = ref(false)
const hostDetail = ref<any>(null)
const hostDetailLoading = ref(false)
const groupDialogVisible = ref(false)

const groupDialogTitle = ref('')
const isGroupEdit = ref(false)

// 表单引用
const hostFormRef = ref<FormInstance>()
const credentialFormRef = ref<FormInstance>()
const cloudAccountFormRef = ref<FormInstance>()
const groupFormRef = ref<FormInstance>()
const groupTreeRef = ref()
const uploadRef = ref()

// 分组树数据
const groupTree = ref<any[]>([])
const filteredGroupTree = ref<any[]>([])
const groupSearchKeyword = ref('')
const selectedGroup = ref<any>(null)
const isExpandAll = ref(false)

// 主机列表数据
const hostList = ref([])
const hostPermissions = ref<Map<number, number>>(new Map()) // 存储每个主机的用户权限
const credentialList = ref([])
const cloudAccountList = ref([])

// 搜索表单
const searchForm = reactive({
  keyword: '',
  status: undefined as number | undefined
})

// 主机分页
const hostPagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 树形组件配置
const treeProps = {
  children: 'children',
  label: 'name',
  value: 'id'
}

// 主机表单
const hostForm = reactive({
  id: 0,
  name: '',
  groupId: null as number | null,
  type: 'self',
  cloudProvider: '',
  cloudInstanceId: '',
  cloudAccountId: null as number | null,
  sshUser: 'root',
  ip: '',
  port: 22,
  credentialId: null as number | null,
  tags: '',
  description: ''
})

// 凭证表单
const credentialForm = reactive({
  name: '',
  type: 'password',
  username: '',
  password: '',
  privateKey: '',
  passphrase: '',
  description: ''
})

// 云平台账号表单
const cloudAccountForm = reactive({
  name: '',
  provider: 'aliyun',
  accessKey: '',
  secretKey: '',
  region: '',
  description: '',
  status: 1
})

// Excel导入表单
const excelImportForm = reactive({
  type: 'self',
  groupId: null as number | null
})
const uploadedFile = ref<UploadFile | null>(null)

// 云主机导入
const cloudImportStep = ref(0)
const cloudImportForm = reactive({
  accountId: null as number | null,
  region: '',
  groupId: null as number | null
})
const cloudHostList = ref<any[]>([])
const selectedCloudHosts = ref<any[]>([])
const selectAllCloudHosts = ref(false)
const cloudRegions = ref<any[]>([])
const loadingCloudRegions = ref(false)

// 主机批量选择
const selectedHosts = ref<any[]>([])

const selectedCloudAccount = ref<any>(null)
const selectedCloudGroup = ref<any>(null)

// 分组表单
const groupForm = reactive({
  id: 0,
  parentId: null,
  name: '',
  code: '',
  description: '',
  sort: 0,
  status: 1
})

// 表单验证规则
const hostRules: FormRules = {
  name: [{ required: true, message: '请输入主机名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择主机类型', trigger: 'change' }],
  ip: [{ required: true, message: '请输入IP地址', trigger: 'blur' }],
  sshUser: [{ required: true, message: '请输入SSH用户名', trigger: 'blur' }],
  port: [{ required: true, message: '请输入SSH端口', trigger: 'blur' }]
}

const credentialRules: FormRules = {
  name: [{ required: true, message: '请输入凭证名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择认证方式', trigger: 'change' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  privateKey: [{ required: true, message: '请输入私钥', trigger: 'blur' }]
}

const cloudAccountRules: FormRules = {
  name: [{ required: true, message: '请输入账号名称', trigger: 'blur' }],
  provider: [{ required: true, message: '请选择云厂商', trigger: 'change' }],
  accessKey: [{ required: true, message: '请输入Access Key', trigger: 'blur' }],
  secretKey: [{ required: true, message: '请输入Secret Key', trigger: 'blur' }]
}

const groupRules: FormRules = {
  name: [{ required: true, message: '请输入分组名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入分组编码', trigger: 'blur' }],
  status: [{ required: true, message: '请选择状态', trigger: 'change' }]
}

// 分组树选项（用于表单中的选择器）
const groupTreeOptions = computed(() => {
  return buildTreeOptions(groupTree.value)
})

// 构建树形选项
const buildTreeOptions = (nodes: any[]): any[] => {
  return nodes.map(node => ({
    id: node.id,
    name: node.name,
    children: node.children ? buildTreeOptions(node.children) : undefined
  }))
}

// 过滤分组树
const filterGroupTree = () => {
  if (!groupSearchKeyword.value) {
    filteredGroupTree.value = groupTree.value
    return
  }
  filteredGroupTree.value = searchTreeNodes(groupTree.value, groupSearchKeyword.value)
}

// 递归搜索树节点
const searchTreeNodes = (nodes: any[], keyword: string): any[] => {
  const result: any[] = []
  for (const node of nodes) {
    const matchName = node.name?.toLowerCase().includes(keyword.toLowerCase())
    let filteredChildren: any[] = []
    if (node.children && node.children.length > 0) {
      filteredChildren = searchTreeNodes(node.children, keyword)
    }
    if (matchName || filteredChildren.length > 0) {
      result.push({
        ...node,
        children: filteredChildren.length > 0 ? filteredChildren : node.children
      })
    }
  }
  return result
}

// 展开/折叠全部
const toggleExpandAll = () => {
  isExpandAll.value = !isExpandAll.value
  const allNodeKeys = getAllNodeKeys(filteredGroupTree.value)
  if (isExpandAll.value) {
    allNodeKeys.forEach(key => groupTreeRef.value?.store.nodesMap[key]?.expand())
  } else {
    allNodeKeys.forEach(key => groupTreeRef.value?.store.nodesMap[key]?.collapse())
  }
}

// 获取所有节点key
const getAllNodeKeys = (nodes: any[]): any[] => {
  const keys: any[] = []
  const traverse = (nodeList: any[]) => {
    nodeList.forEach(node => {
      keys.push(node.id)
      if (node.children && node.children.length > 0) {
        traverse(node.children)
      }
    })
  }
  traverse(nodes)
  return keys
}

// 点击分组节点
const handleGroupClick = (data: any) => {
  selectedGroup.value = data

  if (activeView.value === 'terminal') {
    // 终端视图：加载该分组的主机到终端列表
    loadTerminalHostList(data.id)
  } else {
    // 主机列表视图：加载该分组的主机到表格
    hostPagination.page = 1
    loadHostList()
  }
}

// 清除分组选择
const clearGroupSelection = () => {
  selectedGroup.value = null
  groupTreeRef.value?.setCurrentKey(null)
  hostPagination.page = 1
  loadHostList()
}

// 获取分组路径
const getGroupPath = (group: any): string => {
  return group.name || '未知分组'
}

// 分组操作
const handleGroupAction = (command: string, data: any) => {
  if (command === 'edit') {
    handleEditGroup(data)
  } else if (command === 'delete') {
    handleDeleteGroup(data)
  }
}

// 获取状态颜色
const getStatusColor = (status: number) => {
  switch (status) {
    case 1: return '#67c23a'
    case 0: return '#909399'
    default: return '#c0c4cc'
  }
}

// 获取状态类型
const getStatusType = (status: number) => {
  switch (status) {
    case 1: return 'success'
    case 0: return 'info'
    default: return ''
  }
}

// 加载分组树
const loadGroupTree = async () => {
  groupLoading.value = true
  try {
    const data = await getGroupTree()
    groupTree.value = data || []
    filteredGroupTree.value = data || []
  } catch (error) {
    console.error('获取分组树失败:', error)
    ElMessage.error('获取分组树失败')
  } finally {
    groupLoading.value = false
  }
}

// 加载主机列表
const loadHostList = async () => {
  hostLoading.value = true
  try {
    const params: any = {
      page: hostPagination.page,
      pageSize: hostPagination.pageSize,
      keyword: searchForm.keyword || undefined
    }
    if (searchForm.status !== undefined) {
      params.status = searchForm.status
    }
    // 添加分组ID筛选
    if (selectedGroup.value && selectedGroup.value.id) {
      params.groupId = selectedGroup.value.id
    }

    const res = await getHostList(params)
    hostList.value = res.list || []
    hostPagination.total = res.total || 0

    // 加载每个主机的用户权限
    if (hostList.value && hostList.value.length > 0) {
      const permissionsMap = new Map<number, number>()
      for (const host of hostList.value) {
        try {
          const permRes = await getUserHostPermissions(host.id)
          if (permRes && permRes.permissions !== undefined) {
            permissionsMap.set(host.id, permRes.permissions)
          }
        } catch (err) {
          console.error(`获取主机 ${host.id} 的权限失败:`, err)
        }
      }
      hostPermissions.value = permissionsMap
    }
  } catch (error) {
    console.error('获取主机列表失败:', error)
    ElMessage.error('获取主机列表失败')
  } finally {
    hostLoading.value = false
  }
}

// 加载凭证列表
const loadCredentialList = async () => {
  try {
    const data = await getCredentials()
    credentialList.value = data || []
  } catch (error) {
    console.error('获取凭证列表失败:', error)
  }
}

// 加载云平台账号列表
const loadCloudAccountList = async () => {
  try {
    const data = await getCloudAccounts()
    cloudAccountList.value = data || []
  } catch (error) {
    console.error('获取云平台账号列表失败:', error)
  }
}

// 获取启用的云平台账号列表
const enabledCloudAccounts = computed(() => {
  return cloudAccountList.value.filter((a: any) => a.status === 1)
})

// 终端相关方法
const openTerminalTab = () => {
  const url = window.location.origin + '/terminal'
  window.open(url, '_blank')
}

const handleHostDblClick = async (data: any) => {
  // 如果是主机节点，跳转到终端页面
  if (data.type === 'host' || data.ip) {
    // 将主机信息存储到 sessionStorage
    const dblClickHosts = JSON.parse(sessionStorage.getItem('dblClickHosts') || '[]')
    dblClickHosts.push(data)
    sessionStorage.setItem('dblClickHosts', JSON.stringify(dblClickHosts))
    // 跳转到终端页面
    window.location.href = '/terminal'
  } else if (activeView.value !== 'terminal') {
    // 如果是分组节点且不在终端视图，切换到终端视图
    await openTerminalView()
  }
}

// 终端视图相关方法
const loadTerminalHostList = async (groupId?: number) => {
  try {
    const params: any = {
      page: 1,
      pageSize: 10000
    }
    if (groupId) {
      params.groupId = groupId
    }
    const res = await getHostList(params)
    console.log('终端主机列表数据:', res)
    terminalHostList.value = res.list || []
    console.log('terminalHostList:', terminalHostList.value)
  } catch (error) {
    console.error('获取主机列表失败:', error)
    terminalHostList.value = []
  }
}

// 过滤终端主机列表
const filteredTerminalHosts = computed(() => {
  if (!terminalSearchKeyword.value) {
    return terminalHostList.value
  }
  const keyword = terminalSearchKeyword.value.toLowerCase()
  return terminalHostList.value.filter((host: any) => {
    return host.name?.toLowerCase().includes(keyword) ||
           host.ip?.includes(keyword) ||
           host.groupName?.toLowerCase().includes(keyword)
  })
})

// 构建终端视图的分组+主机树
const terminalGroupTree = computed(() => {
  if (!groupTree.value || groupTree.value.length === 0) {
    return []
  }

  // 深拷贝分组树
  const copyTree = (groups: any[]): any[] => {
    return groups.map((group: any) => ({
      ...group,
      type: 'group',
      label: group.name,
      children: group.children ? copyTree(group.children) : []
    }))
  }

  const tree = copyTree(groupTree.value)

  // 将主机添加到对应的分组
  const addHostsToGroups = (groups: any[], hosts: any[]) => {
    groups.forEach((group: any) => {
      // 查找属于该分组的主机
      const groupHosts = hosts.filter((h: any) => h.groupId === group.id)
      if (groupHosts.length > 0) {
        // 将主机转换为树节点格式
        const hostNodes = groupHosts.map((host: any) => ({
          ...host,
          type: 'host',
          label: host.name
        }))
        // 将主机添加到分组的children中
        group.children = [...(group.children || []), ...hostNodes]
      }

      // 递归处理子分组
      if (group.children && group.children.length > 0) {
        addHostsToGroups(group.children, hosts)
      }
    })
  }

  addHostsToGroups(tree, terminalHostList.value)

  return tree
})

// 初始化终端
const initTerminal = async () => {
  await nextTick()

  if (!terminalRef.value) return

  // 清理旧的终端
  if (terminal.value) {
    terminal.value.dispose()
  }

  // 创建新终端
  terminal.value = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#cccccc',
      cursor: '#cccccc',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79',
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#56b6c2',
      white: '#ffffff',
      brightBlack: '#666666',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#f5f543',
      brightBlue: '#3b8eea',
      brightMagenta: '#d3b9d8',
      brightCyan: '#61bfff',
      brightWhite: '#ffffff',
    }
  })

  // 加载插件
  fitAddon.value = new FitAddon()
  terminal.value.loadAddon(fitAddon.value)
  terminal.value.loadAddon(new WebLinksAddon())

  // 打开终端
  terminal.value.open(terminalRef.value)

  // 欢迎信息
  terminal.value.writeln('\x1b[1;32m欢迎使用 SSH Web 终端\x1b[0m')
  terminal.value.writeln('正在连接...')
}

// 连接SSH
const connectSSH = (host: any) => {
  const token = localStorage.getItem('token') || ''
  const wsUrl = `ws://localhost:9876/api/v1/asset/terminal/${host.id}?token=${token}`

  ws.value = new WebSocket(wsUrl)

  ws.value.onopen = () => {
    if (terminal.value) {
      terminal.value.writeln('\x1b[1;32m连接成功！\x1b[0m')
      terminal.value.writeln(`已连接到: ${host.name} (${host.ip}:${host.port})`)
      terminal.value.writeln('')
    }
  }

  ws.value.onmessage = (event) => {
    if (terminal.value) {
      terminal.value.write(event.data)
    }
  }

  ws.value.onerror = (error) => {
    console.error('WebSocket error:', error)
    if (terminal.value) {
      terminal.value.writeln('\x1b[1;31m连接错误\x1b[0m')
    }
  }

  ws.value.onclose = () => {
    if (terminal.value) {
      terminal.value.writeln('\r\n\x1b[1;33m连接已关闭\x1b[0m')
    }
  }
}

const getTerminalUrl = (host: any): string => {
  const token = localStorage.getItem('token') || ''
  return `/api/v1/asset/terminal/${host.id}?token=${token}`
}

const closeTerminal = () => {
  // 关闭WebSocket
  if (ws.value) {
    ws.value.close()
    ws.value = null
  }

  // 清理终端
  if (terminal.value) {
    terminal.value.dispose()
    terminal.value = null
  }

  activeTerminalHost.value = null
}

const switchToHostsView = async () => {
  activeView.value = 'hosts'
  activeTerminalHost.value = null
}

const handleOpenTerminal = () => {
  // 打开新标签页到终端页面
  const url = window.location.origin + '/terminal'
  window.open(url, '_blank')
}

// 搜索
const handleSearch = () => {
  hostPagination.page = 1
  loadHostList()
}

// 重置
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.status = undefined
  clearGroupSelection()
}

// 导入命令处理
const handleImportCommand = (command: string) => {
  if (command === 'direct') {
    handleDirectImport()
  } else if (command === 'excel') {
    handleExcelImport()
  } else if (command === 'cloud') {
    handleCloudImport()
  }
}

// 直接导入
const handleDirectImport = async () => {
  // 确保凭证列表已加载
  await loadCredentialList()

  Object.assign(hostForm, {
    id: 0,
    name: '',
    groupId: selectedGroup.value?.id || null,
    type: 'self',
    sshUser: 'root',
    ip: '',
    port: 22,
    credentialId: null,
    tags: '',
    description: ''
  })
  directImportVisible.value = true
}

// 直接导入关闭
const handleDirectImportClose = () => {
  hostFormRef.value?.resetFields()
}

// 直接导入提交
const handleDirectImportSubmit = async () => {
  if (!hostFormRef.value) return
  await hostFormRef.value.validate(async (valid) => {
    if (!valid) return
    hostSubmitting.value = true
    try {
      let hostId = 0
      // 判断是创建还是更新
      if (hostForm.id && hostForm.id > 0) {
        // 更新主机
        await updateHost(hostForm.id, hostForm)
        hostId = hostForm.id
        ElMessage.success('主机更新成功')
      } else {
        // 创建主机
        const result = await createHost(hostForm)
        hostId = result.id
        ElMessage.success('主机导入成功')
      }

      directImportVisible.value = false
      loadHostList()
      loadGroupTree()

      // 如果配置了凭证，自动采集主机信息
      if (hostForm.credentialId && hostId > 0) {
        setTimeout(async () => {
          try {
            await collectHostInfo(hostId)
            ElMessage.success('主机信息采集成功')
            loadHostList()
          } catch (error: any) {
            console.error('自动采集失败:', error)
            // 采集失败不阻塞主流程，只记录错误
          }
        }, 500)
      }
    } catch (error: any) {
      ElMessage.error(error.message || '操作失败')
    } finally {
      hostSubmitting.value = false
    }
  })
}

// Excel导入
const handleExcelImport = () => {
  Object.assign(excelImportForm, {
    groupId: selectedGroup.value?.id || null
  })
  uploadedFile.value = null
  excelImportVisible.value = true
}

// 下载模板
const downloadTemplate = async () => {
  try {
    const blob = await downloadExcelTemplate()
    // 创建下载链接
    const url = window.URL.createObjectURL(new Blob([blob], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }))
    const link = document.createElement('a')
    link.href = url
    link.download = 'host_import_template.xlsx'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    ElMessage.success('模板下载成功')
  } catch (error) {
    ElMessage.error('模板下载失败')
  }
}

// 文件变化
const handleFileChange: UploadProps['onChange'] = (uploadFile) => {
  uploadedFile.value = uploadFile
}

// Excel导入关闭
const handleExcelImportClose = () => {
  uploadedFile.value = null
  uploadRef.value?.clearFiles()
}

// Excel导入提交
const handleExcelImportSubmit = async () => {
  if (!uploadedFile.value) {
    ElMessage.warning('请先上传Excel文件')
    return
  }
  try {
    excelImporting.value = true
    const file = uploadedFile.value.raw
    const result = await importFromExcel(file, excelImportForm.type, excelImportForm.groupId || undefined)

    if (result.successCount > 0) {
      ElMessage.success(`成功导入 ${result.successCount} 台主机`)
      // 刷新列表和分组树
      await loadHostList()
      loadGroupTree()

      // 自动采集新导入的主机（状态为-1的主机）
      await new Promise(resolve => setTimeout(resolve, 500))
      const newHosts = hostList.value.filter((h: any) => h.status === -1)
      if (newHosts.length > 0) {
        ElMessage.info('正在自动采集主机信息...')
        const hostIds = newHosts.map((h: any) => h.id)
        try {
          await batchCollectHostInfo({ hostIds })
          await loadHostList()
          ElMessage.success(`成功采集 ${newHosts.length} 台主机信息`)
        } catch (error) {
          console.error('批量采集失败:', error)
        }
      }
    }
    if (result.failedCount > 0) {
      ElMessage.warning(`${result.failedCount} 台主机导入失败`)
      // 显示错误详情
      if (result.errors && result.errors.length > 0) {
        ElMessageBox.alert(
          result.errors.join('<br>'),
          '导入详情',
          { dangerouslyUseHTMLString: true }
        )
      }
    }

    // 关闭对话框
    excelImportVisible.value = false
    uploadedFile.value = null
    uploadRef.value?.clearFiles()
  } catch (error: any) {
    ElMessage.error(error.message || '导入失败')
  } finally {
    excelImporting.value = false
  }
}

// 云主机导入
const handleCloudImport = () => {
  cloudImportStep.value = 0
  cloudHostList.value = []
  selectedCloudHosts.value = []
  cloudRegions.value = []
  Object.assign(cloudImportForm, {
    accountId: null,
    region: '',
    groupId: selectedGroup.value?.id || null
  })
  cloudImportVisible.value = true
}

// 获取云主机列表
const handleGetCloudHosts = async () => {
  if (!cloudImportForm.accountId) {
    ElMessage.warning('请选择云平台账号')
    return
  }
  if (!cloudImportForm.region) {
    ElMessage.warning('请选择区域')
    return
  }

  loadingCloudHosts.value = true
  try {
    // 调用真实的云平台API获取实例列表
    const res = await getCloudInstances(cloudImportForm.accountId!, cloudImportForm.region)
    cloudHostList.value = Array.isArray(res) ? res : []
    selectedCloudAccount.value = cloudAccountList.value.find(a => a.id === cloudImportForm.accountId)
    selectedCloudGroup.value = groupTree.value.find((g: any) => g.id === cloudImportForm.groupId)
    cloudImportStep.value = 1
  } catch (error: any) {
    console.error('获取云主机列表失败:', error)
    ElMessage.error(error.message || '获取云主机列表失败')
  } finally {
    loadingCloudHosts.value = false
  }
}

// 云主机选择变化
const handleCloudHostSelectionChange = (selection: any[]) => {
  selectedCloudHosts.value = selection
}

// 主机选择变化
const handleHostSelectionChange = (selection: any[]) => {
  selectedHosts.value = selection
}

// 批量删除主机
const handleBatchDelete = async () => {
  if (selectedHosts.value.length === 0) {
    ElMessage.warning('请先选择要删除的主机')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedHosts.value.length} 台主机吗？`,
      '批量删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const hostIds = selectedHosts.value.map((h: any) => h.id)
    await batchDeleteHosts(hostIds)

    ElMessage.success('批量删除成功')
    selectedHosts.value = []
    loadHostList()
    loadGroupTree()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '批量删除失败')
    }
  }
}

// 全选云主机
const handleSelectAllCloudHosts = (checked: boolean) => {
  // TODO: 实现全选逻辑
}

// 云主机导入关闭
const handleCloudImportClose = () => {
  cloudImportStep.value = 0
  cloudHostList.value = []
  selectedCloudHosts.value = []
}

// 云主机导入提交
const handleCloudImportSubmit = async () => {
  cloudImporting.value = true
  try {
    const data = {
      accountId: cloudImportForm.accountId,
      region: cloudImportForm.region,
      groupId: cloudImportForm.groupId,
      instanceIds: selectedCloudHosts.value.map(h => h.instanceId)
    }
    await importFromCloud(data)
    ElMessage.success('云主机导入成功')
    cloudImportVisible.value = false
    loadHostList()
    loadGroupTree()
  } catch (error: any) {
    ElMessage.error(error.message || '导入失败')
  } finally {
    cloudImporting.value = false
  }
}

// 认证方式变化
const handleAuthTypeChange = (type: string) => {
  // 清空对应字段
  if (type === 'password') {
    credentialForm.privateKey = ''
    credentialForm.passphrase = ''
  } else {
    credentialForm.password = ''
  }
}

// 凭证对话框关闭
const handleCredentialDialogClose = () => {
  credentialFormRef.value?.resetFields()
}

// 提交凭证表单
const handleCredentialSubmit = async () => {
  if (!credentialFormRef.value) return
  await credentialFormRef.value.validate(async (valid) => {
    if (!valid) return
    credentialSubmitting.value = true
    try {
      await createCredential(credentialForm)
      ElMessage.success('凭证创建成功')
      showCredentialDialog.value = false
      loadCredentialList()
    } catch (error: any) {
      ElMessage.error(error.message || '创建失败')
    } finally {
      credentialSubmitting.value = false
    }
  })
}

// 云平台账号对话框关闭
const handleCloudAccountDialogClose = () => {
  cloudAccountFormRef.value?.resetFields()
}

// 提交云平台账号表单
const handleCloudAccountSubmit = async () => {
  if (!cloudAccountFormRef.value) return
  await cloudAccountFormRef.value.validate(async (valid) => {
    if (!valid) return
    cloudAccountSubmitting.value = true
    try {
      await createCloudAccount(cloudAccountForm)
      ElMessage.success('云平台账号添加成功')
      showCloudAccountDialog.value = false
      loadCloudAccountList()
    } catch (error: any) {
      ElMessage.error(error.message || '添加失败')
    } finally {
      cloudAccountSubmitting.value = false
    }
  })
}

// 编辑主机
const handleEditHost = async (row: any) => {
  // 重新加载凭证列表，确保显示最新的凭证
  await loadCredentialList()

  Object.assign(hostForm, {
    id: row.id,
    name: row.name,
    groupId: row.groupId,
    type: row.type || 'self',
    cloudProvider: row.cloudProvider || '',
    cloudInstanceId: row.cloudInstanceId || '',
    cloudAccountId: row.cloudAccountId || null,
    sshUser: row.sshUser,
    ip: row.ip,
    port: row.port,
    credentialId: row.credentialId,
    tags: Array.isArray(row.tags) ? row.tags.join(',') : row.tags,
    description: row.description
  })
  directImportVisible.value = true
}

// 文件管理
const handleFileManager = (row: any) => {
  selectedHostId.value = row.id
  selectedHostName.value = row.name
  fileBrowserVisible.value = true
}

// 显示主机详情
const handleShowHostDetail = async (row: any) => {
  try {
    hostDetailLoading.value = true
    showHostDetailDialog.value = true
    const data = await getHost(row.id)
    hostDetail.value = data
  } catch (error: any) {
    ElMessage.error(error.message || '获取主机详情失败')
  } finally {
    hostDetailLoading.value = false
  }
}

// 关闭主机详情
const handleCloseHostDetail = () => {
  showHostDetailDialog.value = false
  hostDetail.value = null
}

// 从详情页采集主机信息
const handleCollectHostFromDetail = async () => {
  if (!hostDetail.value) return
  try {
    hostDetailLoading.value = true
    await collectHostInfo(hostDetail.value.id)
    ElMessage.success('采集成功')
    // 重新获取详情
    const data = await getHost(hostDetail.value.id)
    hostDetail.value = data
    // 刷新列表
    loadHostList()
  } catch (error: any) {
    ElMessage.error(error.message || '采集失败')
  } finally {
    hostDetailLoading.value = false
  }
}

// 删除主机
const handleDeleteHost = (row: any) => {
  ElMessageBox.confirm(`确定要删除主机"${row.name}"吗？`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteHost(row.id)
      ElMessage.success('删除成功')
      loadHostList()
      loadGroupTree()
    } catch (error: any) {
      ElMessage.error(error.message || '删除失败')
    }
  }).catch(() => {})
}

// 新增分组
const handleAddGroup = () => {
  Object.assign(groupForm, {
    id: 0,
    parentId: null,
    name: '',
    code: '',
    description: '',
    sort: 0,
    status: 1
  })
  groupDialogTitle.value = '新增分组'
  isGroupEdit.value = false
  groupDialogVisible.value = true
}

// 编辑分组
const handleEditGroup = (data: any) => {
  Object.assign(groupForm, {
    id: data.id,
    parentId: data.parentId || null,
    name: data.name,
    code: data.code || '',
    description: data.description || '',
    sort: data.sort || 0,
    status: data.status
  })
  groupDialogTitle.value = '编辑分组'
  isGroupEdit.value = true
  groupDialogVisible.value = true
}

// 删除分组
const handleDeleteGroup = (data: any) => {
  const hasChildren = data.children && data.children.length > 0
  const confirmMsg = hasChildren
    ? `该分组下有 ${data.children.length} 个子分组，确定要删除吗？`
    : `确定要删除分组"${data.name}"吗？`

  ElMessageBox.confirm(confirmMsg, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteGroup(data.id)
      ElMessage.success('删除成功')
      loadGroupTree()
      if (selectedGroup.value?.id === data.id) {
        clearGroupSelection()
      }
    } catch (error: any) {
      ElMessage.error(error.message || '删除失败')
    }
  }).catch(() => {})
}

// 提交分组表单
const handleGroupSubmit = async () => {
  if (!groupFormRef.value) return
  await groupFormRef.value.validate(async (valid) => {
    if (!valid) return
    groupSubmitting.value = true
    try {
      const data = { ...groupForm }
      if (isGroupEdit.value) {
        await updateGroup(data.id, data)
      } else {
        await createGroup(data)
      }
      ElMessage.success(isGroupEdit.value ? '更新成功' : '创建成功')
      groupDialogVisible.value = false
      loadGroupTree()
    } catch (error: any) {
      ElMessage.error(error.message || (isGroupEdit.value ? '更新失败' : '创建失败'))
    } finally {
      groupSubmitting.value = false
    }
  })
}

// 分组对话框关闭
const handleGroupDialogClose = () => {
  groupFormRef.value?.resetFields()
}

// 格式化字节数

// 检查用户对主机的权限
const hasHostPermission = (hostId: number, permission: number): boolean => {
  const userPermissions = hostPermissions.value.get(hostId) || 0
  return hasPermission(userPermissions, permission)
}
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 格式化字节数（紧凑格式，不换行）
const formatBytesCompact = (bytes: number): string => {
  if (bytes === 0) return '0B'
  const k = 1024
  const sizes = ['B', 'K', 'M', 'G', 'T']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const value = bytes / Math.pow(k, i)
  if (value >= 100) {
    return Math.round(value) + sizes[i]
  }
  return parseFloat(value.toFixed(1)) + sizes[i]
}

// 获取使用率颜色
const getUsageColor = (usage: number): string => {
  if (usage >= 90) return '#f56c6c'
  if (usage >= 70) return '#e6a23c'
  return '#67c23a'
}

// 获取使用率等级
const getUsageLevel = (usage: number | undefined): string => {
  if (!usage) return 'low'
  if (usage >= 90) return 'critical'
  if (usage >= 70) return 'high'
  return 'low'
}

// 采集主机信息
const handleCollectHost = async (row: any) => {
  try {
    await collectHostInfo(row.id)
    ElMessage.success('采集成功')
    loadHostList()
  } catch (error: any) {
    ElMessage.error(error.message || '采集失败')
  }
}

// 监听activeTerminalHost变化，自动连接终端
watch(activeTerminalHost, async (newHost) => {
  if (newHost) {
    await initTerminal()
    await nextTick()
    connectSSH(newHost)
  } else {
    closeTerminal()
  }
})

// 监听云平台账号变化，加载区域列表
watch(() => cloudImportForm.accountId, async (accountId) => {
  if (accountId) {
    cloudRegions.value = []
    cloudImportForm.region = ''
    loadingCloudRegions.value = true
    try {
      const res = await getCloudRegions(accountId)
      cloudRegions.value = Array.isArray(res) ? res : []

      // 如果账号有默认区域且在列表中，自动选中
      const account = cloudAccountList.value.find((a: any) => a.id === accountId)
      if (account?.region && cloudRegions.value.some((r: any) => r.value === account.region)) {
        cloudImportForm.region = account.region
      }
    } catch (error: any) {
      console.error('加载区域列表失败:', error)
      ElMessage.error(error.message || '加载区域列表失败')
    } finally {
      loadingCloudRegions.value = false
    }
  } else {
    cloudRegions.value = []
    cloudImportForm.region = ''
  }
})

// 组件销毁时清理资源
onBeforeUnmount(() => {
  closeTerminal()
})

onMounted(() => {
  loadGroupTree()
  loadHostList()
  loadCredentialList()
  loadCloudAccountList()
})
</script>

<style scoped>
.hosts-page-container {
  padding: 0;
  background-color: transparent;
  height: 100%;
  display: flex;
  flex-direction: column;
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
  flex-shrink: 0;
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

.import-dropdown {
  display: inline-block;
}

/* 主内容区域 */
.main-content {
  display: flex;
  gap: 12px;
  flex: 1;
  min-height: 0;
}

/* 左侧分组面板 */
.left-panel {
  width: 280px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.panel-header {
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 15px;
  color: #303133;
}

.panel-icon {
  font-size: 18px;
  color: #d4af37;
}

.panel-actions {
  display: flex;
  gap: 8px;
}

.panel-body {
  flex: 1;
  padding: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.group-search {
  margin-bottom: 12px;
}

.group-search :deep(.el-input__wrapper) {
  border-radius: 20px;
}

.tree-container {
  flex: 1;
  overflow-y: auto;
}

.group-tree {
  background: transparent;
}

.group-tree :deep(.el-tree-node__content) {
  border-radius: 6px;
  padding: 6px 8px;
  transition: all 0.2s ease;
}

.group-tree :deep(.el-tree-node__content:hover) {
  background-color: #f5f7fa;
}

.group-tree :deep(.is-current > .el-tree-node__content) {
  background-color: #ecf5ff;
  color: #409eff;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  width: 0;
}

.node-icon {
  flex-shrink: 0;
}

.node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-count {
  font-size: 12px;
  color: #909399;
  flex-shrink: 0;
}

.node-actions {
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.2s;
}

.group-tree :deep(.el-tree-node__content:hover) .node-actions {
  opacity: 1;
}

.more-icon {
  font-size: 14px;
  cursor: pointer;
  color: #909399;
}

.more-icon:hover {
  color: #409eff;
}

/* 右侧主机列表面板 */
.right-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.filter-bar {
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px 8px 0 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.filter-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.filter-input {
  width: 220px;
}

.filter-actions {
  display: flex;
  gap: 10px;
}

.filter-bar :deep(.el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  transition: all 0.3s ease;
}

.filter-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
}

.filter-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
}

.search-icon {
  color: #d4af37;
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

.selected-group-bar {
  padding: 10px 16px;
  background: #f0f9ff;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.group-path {
  color: #409eff;
  font-weight: 500;
}

.table-wrapper {
  flex: 1;
  background: #fff;
  border-radius: 0 0 8px 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modern-table {
  flex: 1;
}

.modern-table :deep(.el-table__body-wrapper) {
  overflow-y: auto;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.modern-table :deep(.el-table__header th) {
  font-weight: 600;
  font-size: 13px;
}

.modern-table :deep(.el-table__body td) {
  font-size: 13px;
}

.hostname-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

/* 主机头像/状态图标 */
.host-avatar {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: #f0f2f5;
  color: #909399;
}

.host-avatar.host-status-1 {
  background: #e7f8e8;
  color: #67c23a;
}

.host-avatar.host-status-0 {
  background: #fef0f0;
  color: #f56c6c;
}

.host-avatar.host-status--1 {
  background: #f4f4f5;
  color: #909399;
}

.host-avatar .el-icon {
  font-size: 18px;
}

/* 主机信息 */
.host-info {
  flex: 1;
  min-width: 0;
}

.hostname {
  font-weight: 500;
  color: #303133;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.host-meta {
  font-size: 12px;
  color: #909399;
  display: flex;
  align-items: center;
  gap: 2px;
}

.ip {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.port {
  flex-shrink: 0;
}

/* 分组单元格 */
.group-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: #f5f7fa;
  border-radius: 6px;
  width: fit-content;
}

.group-icon {
  font-size: 14px;
  color: #409eff;
  flex-shrink: 0;
}

.group-name {
  font-size: 13px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 70px;
}

/* 凭证单元格新样式 */
.credential-cell-new {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: #f5f7fa;
  border-radius: 6px;
  width: fit-content;
  max-width: 100%;
}

.credential-icon {
  font-size: 14px;
  color: #909399;
  flex-shrink: 0;
}

.credential-icon.icon-key {
  color: #e6a23c;
}

.credential-icon.icon-lock {
  color: #67c23a;
}

.credential-cell-new .credential-name {
  font-size: 12px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 70px;
}

/* 空状态文本 */
.empty-text {
  font-size: 13px;
  color: #c0c4cc;
}

.empty-warning {
  color: #e6a23c;
}

/* 状态单元格 */
.status-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot-1 {
  background: #67c23a;
  box-shadow: 0 0 0 2px rgba(103, 194, 58, 0.2);
}

.status-dot-0 {
  background: #f56c6c;
  box-shadow: 0 0 0 2px rgba(245, 108, 108, 0.2);
}

.status-dot--1 {
  background: #909399;
  box-shadow: 0 0 0 2px rgba(144, 148, 153, 0.2);
}

.status-text {
  font-size: 13px;
  font-weight: 500;
}

.status-text-1 {
  color: #67c23a;
}

.status-text-0 {
  color: #f56c6c;
}

.status-text--1 {
  color: #909399;
}

.host-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.config-info {
  font-size: 13px;
  color: #606266;
}

.text-muted {
  color: #c0c4cc;
}

.pagination-wrapper {
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
  display: flex;
  justify-content: flex-end;
}

/* 旧的操作按钮样式 - 保留兼容 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn :deep(.el-icon) {
  font-size: 14px;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.action-refresh:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-files:hover {
  background-color: #fdf6ec;
  color: #e6a23c;
}

.hostname-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

/* 主机头像/状态图标 */
.host-avatar {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: #f0f2f5;
  color: #909399;
}

.host-avatar.host-status-1 {
  background: #e7f8e8;
  color: #67c23a;
}

.host-avatar.host-status-0 {
  background: #fef0f0;
  color: #f56c6c;
}

.host-avatar.host-status--1 {
  background: #f4f4f5;
  color: #909399;
}

.host-avatar .el-icon {
  font-size: 18px;
}

/* 主机信息 */
.host-info {
  flex: 1;
  min-width: 0;
}

.hostname {
  font-weight: 500;
  color: #303133;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.host-meta {
  font-size: 12px;
  color: #909399;
  display: flex;
  align-items: center;
  gap: 2px;
}

.ip {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.port {
  flex-shrink: 0;
}

/* 分组单元格 */
.group-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: #f5f7fa;
  border-radius: 6px;
  width: fit-content;
}

.group-icon {
  font-size: 14px;
  color: #409eff;
  flex-shrink: 0;
}

.group-name {
  font-size: 13px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 70px;
}

/* 凭证单元格新样式 */
.credential-cell-new {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: #f5f7fa;
  border-radius: 6px;
  width: fit-content;
  max-width: 100%;
}

.credential-icon {
  font-size: 14px;
  color: #909399;
  flex-shrink: 0;
}

.credential-icon.icon-key {
  color: #e6a23c;
}

.credential-icon.icon-lock {
  color: #67c23a;
}

.credential-cell-new .credential-name {
  font-size: 12px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 70px;
}

/* 空状态文本 */
.empty-text {
  font-size: 13px;
  color: #c0c4cc;
}

.empty-warning {
  color: #e6a23c;
}

/* 状态单元格 */
.status-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot-1 {
  background: #67c23a;
  box-shadow: 0 0 0 2px rgba(103, 194, 58, 0.2);
}

.status-dot-0 {
  background: #f56c6c;
  box-shadow: 0 0 0 2px rgba(245, 108, 108, 0.2);
}

.status-dot--1 {
  background: #909399;
  box-shadow: 0 0 0 2px rgba(144, 148, 153, 0.2);
}

.status-text {
  font-size: 13px;
  font-weight: 500;
}

.status-text-1 {
  color: #67c23a;
}

.status-text-0 {
  color: #f56c6c;
}

.status-text--1 {
  color: #909399;
}

.host-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.config-info {
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
  border-color: #333333 !important;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.host-import-dialog),
:deep(.excel-import-dialog),
:deep(.cloud-import-dialog),
:deep(.credential-dialog),
:deep(.cloud-account-dialog),
:deep(.group-edit-dialog) {
  border-radius: 12px;
}

:deep(.host-import-dialog .el-dialog__header),
:deep(.excel-import-dialog .el-dialog__header),
:deep(.cloud-import-dialog .el-dialog__header),
:deep(.credential-dialog .el-dialog__header),
:deep(.cloud-account-dialog .el-dialog__header),
:deep(.group-edit-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.host-import-dialog .el-dialog__body),
:deep(.excel-import-dialog .el-dialog__body),
:deep(.cloud-import-dialog .el-dialog__body),
:deep(.credential-dialog .el-dialog__body),
:deep(.cloud-account-dialog .el-dialog__body),
:deep(.group-edit-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.host-import-dialog .el-dialog__footer),
:deep(.excel-import-dialog .el-dialog__footer),
:deep(.cloud-import-dialog .el-dialog__footer),
:deep(.credential-dialog .el-dialog__footer),
:deep(.cloud-account-dialog .el-dialog__footer),
:deep(.group-edit-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

:deep(.el-tag) {
  border-radius: 4px;
  padding: 2px 8px;
  font-weight: 500;
  height: auto;
  line-height: 1.5;
}

:deep(.responsive-dialog) {
  max-width: 1200px;
  min-width: 500px;
}

@media (max-width: 768px) {
  :deep(.responsive-dialog .el-dialog) {
    width: 95% !important;
    max-width: none;
    min-width: auto;
  }
}

/* Excel导入样式 */
.excel-import-content {
  padding: 10px 0;
}

.upload-demo {
  width: 100%;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 6px;
}

/* 云主机导入样式 */
.cloud-step-content {
  padding: 20px 0;
  min-height: 300px;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 6px;
}

.import-summary {
  width: 100%;
  max-width: 600px;
  margin: 0 auto;
}

.host-list-preview {
  margin-top: 20px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 6px;
}

.host-list-preview h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #606266;
}

/* 新的表格样式 */
.host-table {
  border-radius: 0 0 8px 8px;
}

.host-table :deep(.el-table__header-wrapper) {
  border-radius: 0;
}

.host-table :deep(.el-table__header th) {
  border-bottom: 2px solid #e2e8f0;
}

.host-table :deep(.el-table__body tr) {
  transition: all 0.2s ease;
}

.host-table :deep(.el-table__body tr:hover > td) {
  background-color: #f8fafc !important;
}

/* 资源显示单元格样式 */
.resource-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 40px;
}

.resource-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  padding: 0 4px;
}

.resource-label {
  font-size: 12px;
  color: #606266;
  font-weight: 500;
  white-space: nowrap;
}

.resource-compact {
  font-size: 11px;
  white-space: nowrap;
}

.resource-info :deep(.el-progress) {
  margin: 0;
}

.resource-info :deep(.el-progress__text) {
  font-size: 10px !important;
  min-width: 28px;
}

.resource-info :deep(.el-progress-bar__outer) {
  height: 5px !important;
}

/* 配置单元格样式 */
.config-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  padding: 0 8px;
}

.config-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.config-item .el-icon {
  font-size: 14px;
  color: #909399;
  flex-shrink: 0;
}

.config-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 标签单元格样式 */
.tags-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
  justify-content: flex-start;
}

.tags-cell .tag-item {
  font-size: 11px;
  padding: 2px 6px;
  height: auto;
}

.tags-cell .tag-more {
  font-size: 11px;
  padding: 2px 6px;
  height: auto;
}

/* 凭证单元格样式 */
.credential-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  width: 100%;
}

.credential-type-tag {
  font-size: 10px;
  padding: 1px 6px;
  height: 18px;
  line-height: 16px;
}

.credential-name {
  font-size: 12px;
  color: #303133;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.text-danger {
  color: #f56c6c !important;
}

/* 终端按钮样式 */
/* 主机详情对话框样式 */
.host-detail-dialog {
  border-radius: 12px;
}

.host-detail-content {
  padding: 10px 0;
}

/* 信息网格布局 */
.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}

.info-card {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 12px;
  overflow: hidden;
}

.info-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
  border-bottom: 1px solid #e4e7ed;
}

.info-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: #fff;
  flex-shrink: 0;
}

.info-icon-basic {
  background: linear-gradient(135deg, #409eff 0%, #66b1ff 100%);
}

.info-icon-system {
  background: linear-gradient(135deg, #67c23a 0%, #85ce61 100%);
}

.info-icon-auth {
  background: linear-gradient(135deg, #e6a23c 0%, #ebb563 100%);
}

.info-icon-tags {
  background: linear-gradient(135deg, #909399 0%, #a6a9ad 100%);
}

.info-icon-resource {
  background: linear-gradient(135deg, #f56c6c 0%, #f78989 100%);
}

.info-icon-remark {
  background: linear-gradient(135deg, #606266 0%, #909399 100%);
}

.info-card-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.info-card-body {
  padding: 16px 20px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid #f5f5f5;
}

.info-row:last-child {
  border-bottom: none;
}

.info-label {
  font-size: 14px;
  color: #909399;
  flex-shrink: 0;
}

.info-value {
  font-size: 15px;
  color: #303133;
  font-weight: 500;
  text-align: right;
  word-break: break-all;
}

/* 资源区域 */
.resource-section {
  margin-bottom: 24px;
}

.resource-card-wrapper .info-card-body {
  padding: 20px;
}

.resource-section-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.section-title {
  font-size: 17px;
  font-weight: 600;
  color: #303133;
}

/* 备注区域 */
.remark-section {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 12px;
  overflow: hidden;
}

.remark-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
  border-bottom: 1px solid #e4e7ed;
}

.remark-content {
  padding: 16px 20px;
  font-size: 15px;
  color: #606266;
  line-height: 1.6;
  white-space: pre-wrap;
}

.tags-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

/* 主机名点击样式 */
.hostname-clickable {
  cursor: pointer;
  transition: color 0.2s ease;
}

.hostname-clickable:hover {
  color: #409eff;
}

.hostname-cell {
  cursor: pointer;
}

.terminal-button {
  background-color: #1a1a1a !important;
  color: #ffffff !important;
  border-color: #1a1a1a !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
  margin-right: 12px;
}

.terminal-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

/* 视图容器 */
.view-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* 终端视图 */
.terminal-view {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.terminal-view-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: #1a1a1a;
  border-bottom: 1px solid #333;
}

.terminal-view-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.terminal-view-title .el-icon {
  font-size: 20px;
  color: #d4af37;
}

.terminal-current-group {
  margin-left: 8px;
  font-size: 14px;
  color: #858585;
  font-weight: normal;
}

.terminal-content {
  display: flex;
  flex: 1;
  overflow: hidden;
  background: #1e1e1e;
}

/* 终端侧边栏 */
.terminal-sidebar {
  width: 280px;
  min-width: 280px;
  background: #252526;
  border-right: 1px solid #3e3e42;
  display: flex;
  flex-direction: column;
}

/* 主机列表面板（中间） */
.terminal-host-panel {
  width: 320px;
  min-width: 320px;
  background: #2d2d30;
  border-right: 1px solid #3e3e42;
  display: flex;
  flex-direction: column;
}

.terminal-host-panel-header {
  padding: 16px;
  border-bottom: 1px solid #3e3e42;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.host-panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #cccccc;
  font-size: 14px;
  font-weight: 600;
}

.host-panel-title .el-icon {
  color: #4ec9b0;
}

.terminal-host-panel-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.terminal-host-panel-content::-webkit-scrollbar {
  width: 8px;
}

.terminal-host-panel-content::-webkit-scrollbar-track {
  background: #1e1e1e;
}

.terminal-host-panel-content::-webkit-scrollbar-thumb {
  background: #424242;
  border-radius: 4px;
}

.terminal-host-panel-content::-webkit-scrollbar-thumb:hover {
  background: #4e4e4e;
}

.terminal-host-count {
  margin-left: auto;
}

.terminal-search {
  margin-bottom: 12px;
}

.terminal-search :deep(.el-input__wrapper) {
  background: #3c3c3c;
  border: 1px solid #3e3e42;
  box-shadow: none;
}

.terminal-search :deep(.el-input__inner) {
  color: #cccccc;
}

.terminal-search :deep(.el-input__wrapper:hover) {
  border-color: #555;
}

.terminal-search :deep(.el-input__wrapper.is-focus) {
  border-color: #007acc;
}

/* 终端主机列表 */
.terminal-host-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.terminal-host-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: #2d2d30;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.terminal-host-item:hover {
  background: #3e3e42;
}

.terminal-host-item.host-item-active {
  background: #094771;
  border: 1px solid #007acc;
}

.host-status-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.host-item-info {
  flex: 1;
  min-width: 0;
}

.host-item-name {
  font-size: 13px;
  font-weight: 500;
  color: #cccccc;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.host-item-ip {
  font-size: 11px;
  color: #858585;
  margin-top: 2px;
}

.host-item-group {
  font-size: 11px;
  color: #4ec9b0;
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 终端主区域 */
.terminal-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
  overflow: hidden;
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #252526;
  border-bottom: 1px solid #3e3e42;
  flex-shrink: 0;
}

.terminal-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.terminal-icon {
  font-size: 20px;
  color: #4ec9b0;
}

.terminal-details {
  display: flex;
  flex-direction: column;
}

.terminal-title {
  font-size: 14px;
  font-weight: 600;
  color: #cccccc;
}

.terminal-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #858585;
  margin-top: 2px;
}

.terminal-ip {
  color: #9cdcfe;
}

.terminal-user {
  color: #dcdcaa;
}

.terminal-actions {
  display: flex;
  gap: 8px;
}

.terminal-actions :deep(.el-button) {
  background: #3c3c3c;
  border: 1px solid #3e3e42;
  color: #cccccc;
}

.terminal-actions :deep(.el-button:hover) {
  background: #4e4e4e;
  border-color: #555;
}

.terminal-placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #858585;
}

.placeholder-icon {
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.placeholder-text {
  font-size: 16px;
  margin-bottom: 8px;
}

.placeholder-hint {
  font-size: 13px;
  color: #858585;
}

.terminal-body {
  flex: 1;
  overflow: hidden;
  background: #1e1e1e;
}

.terminal-wrapper {
  width: 100%;
  height: 100%;
  background: #1e1e1e;
  padding: 10px;
}

.xterm-container {
  width: 100%;
  height: 100%;
}

.xterm-container :deep(.xterm) {
  padding: 10px;
}

.xterm-container :deep(.xterm .xterm-viewport) {
  background-color: #1e1e1e !important;
}

.xterm-container :deep(.xterm .xterm-screen) {
  padding: 0;
}

/* 主机详情弹窗样式 */
.detail-dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0;
}

.detail-header-left {
  display: flex;
  align-items: center;
  gap: 20px;
  flex: 1;
}

.host-avatar-lg {
  width: 72px;
  height: 72px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  flex-shrink: 0;
  position: relative;
  overflow: hidden;
}

.host-avatar-lg::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 16px;
  padding: 2px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.3), transparent);
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
}

.host-avatar-lg.host-status-1 {
  background: linear-gradient(135deg, #67c23a 0%, #85ce61 100%);
  color: #fff;
}

.host-avatar-lg.host-status-0 {
  background: linear-gradient(135deg, #f56c6c 0%, #f78989 100%);
  color: #fff;
}

.host-avatar-lg.host-status--1 {
  background: linear-gradient(135deg, #909399 0%, #a6a9ad 100%);
  color: #fff;
}

.detail-header-info {
  flex: 1;
}

.detail-hostname {
  font-size: 26px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 8px;
}

.detail-hostmeta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.detail-ip {
  font-size: 16px;
  color: #606266;
  font-family: 'Monaco', 'Menlo', monospace;
}

/* 资源卡片网格 */
.resource-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.resource-card {
  background: #fafbfc;
  border: 1px solid #e8e8e8;
  border-radius: 10px;
  padding: 16px;
  transition: all 0.3s ease;
}

.resource-card:hover {
  background: #f5f6f7;
  border-color: #d8d8d8;
}

.resource-card-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: #606266;
  margin-bottom: 10px;
}

.resource-icon {
  font-size: 20px;
}

.cpu-icon {
  color: #409eff;
}

.memory-icon {
  color: #67c23a;
}

.disk-icon {
  color: #e6a23c;
}

.resource-card-body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.resource-value {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.resource-usage {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.resource-usage :deep(.el-progress) {
  flex: 1;
}

.resource-usage.usage-low :deep(.el-progress-bar__inner) {
  background: linear-gradient(90deg, #67c23a 0%, #85ce61 100%);
}

.resource-usage.usage-high :deep(.el-progress-bar__inner) {
  background: linear-gradient(90deg, #e6a23c 0%, #ebb563 100%);
}

.resource-usage.usage-critical :deep(.el-progress-bar__inner) {
  background: linear-gradient(90deg, #f56c6c 0%, #f78989 100%);
}

.usage-text {
  font-size: 15px;
  font-weight: 600;
  min-width: 50px;
  text-align: right;
}

.usage-text.low {
  color: #67c23a;
}

.usage-text.high {
  color: #e6a23c;
}

.usage-text.critical {
  color: #f56c6c;
}

.usage-legend {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
  margin-top: 16px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: #909399;
}

.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.legend-dot.low {
  background: #67c23a;
}

.legend-dot.high {
  background: #e6a23c;
}

.legend-dot.critical {
  background: #f56c6c;
}
</style>

// 文件管理
const handleFileManager = (row: any) => {
  selectedHostId.value = row.id
  selectedHostName.value = row.name
  fileBrowserVisible.value = true
}
