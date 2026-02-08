<template>
  <div class="certificate-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Key /></el-icon>
        </div>
        <div>
          <h2 class="page-title">证书管理</h2>
          <p class="page-subtitle">管理SSL证书，支持Let's Encrypt自动申请和手动导入</p>
        </div>
      </div>
      <div class="header-actions">
        <el-dropdown @command="handleCreate">
          <el-button class="black-button">
            <el-icon style="margin-right: 6px;"><Plus /></el-icon>
            新增证书
            <el-icon style="margin-left: 6px;"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="apply">申请证书 (Let's Encrypt)</el-dropdown-item>
              <el-dropdown-item command="import">导入证书</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button @click="loadData">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.domain"
          placeholder="搜索域名..."
          clearable
          class="search-input"
          @keyup.enter="loadData"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.status"
          placeholder="证书状态"
          clearable
          class="search-input"
          @change="loadData"
        >
          <el-option label="正常" value="active" />
          <el-option label="即将过期" value="expiring" />
          <el-option label="已过期" value="expired" />
          <el-option label="待申请" value="pending" />
          <el-option label="错误" value="error" />
        </el-select>

        <el-select
          v-model="searchForm.source_type"
          placeholder="证书来源"
          clearable
          class="search-input"
          @change="loadData"
        >
          <el-option label="Let's Encrypt" value="letsencrypt" />
          <el-option label="阿里云" value="aliyun" />
          <el-option label="手动导入" value="manual" />
        </el-select>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon stat-icon-primary">
          <el-icon><Document /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">证书总数</div>
          <div class="stat-value">{{ stats.total || 0 }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">正常</div>
          <div class="stat-value">{{ stats.active || 0 }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <el-icon><Warning /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">即将过期</div>
          <div class="stat-value">{{ stats.expiring || 0 }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <el-icon><CircleClose /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">已过期/错误</div>
          <div class="stat-value">{{ (stats.expired || 0) + (stats.error || 0) }}</div>
        </div>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <el-table
        :data="tableData"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="证书名称" prop="name" width="120" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="cert-name">{{ row.name }}</div>
          </template>
        </el-table-column>

        <el-table-column label="域名" prop="domain" width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="domain-cell">
              <span class="domain-main">{{ row.domain }}</span>
              <el-tag v-if="row.san_domains && JSON.parse(row.san_domains || '[]').length > 0" type="info" size="small" style="margin-left: 8px;">
                +{{ JSON.parse(row.san_domains || '[]').length }} SAN
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'active'" type="success" effect="dark">正常</el-tag>
            <el-tag v-else-if="row.status === 'expiring'" type="warning" effect="dark">即将过期</el-tag>
            <el-tag v-else-if="row.status === 'expired'" type="danger" effect="dark">已过期</el-tag>
            <el-tag v-else-if="row.status === 'pending'" type="info" effect="dark">待申请</el-tag>
            <el-tag v-else type="danger" effect="dark">错误</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="来源" min-width="100" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ getSourceTypeName(row.source_type) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="CA提供商" min-width="120" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ getCAProviderName(row.ca_provider) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="加密算法" min-width="90" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ getKeyAlgorithmName(row.key_algorithm) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="颁发者" prop="issuer" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ row.issuer || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="到期时间" min-width="110">
          <template #default="{ row }">
            <div v-if="row.not_after">
              <span :class="getExpiryClass(row.not_after)">{{ formatDateTime(row.not_after) }}</span>
              <div class="expiry-days">{{ getExpiryDays(row.not_after) }}</div>
            </div>
            <span v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="自动续期" width="80" align="center">
          <template #default="{ row }">
            <el-switch
              v-model="row.auto_renew"
              @change="handleAutoRenewChange(row)"
              :disabled="row.source_type === 'manual'"
            />
          </template>
        </el-table-column>

        <el-table-column label="操作" width="260" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="查看详情" placement="top">
                <el-button link class="action-btn action-view" @click="handleView(row)">
                  <el-icon><View /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="下载证书" placement="top">
                <el-button link class="action-btn action-download" @click="handleDownload(row)" :disabled="row.status === 'pending' || row.status === 'error'">
                  <el-icon><Download /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="手动续期" placement="top" v-if="row.source_type !== 'manual' && row.status !== 'pending'">
                <el-button link class="action-btn action-renew" @click="handleRenew(row)">
                  <el-icon><RefreshRight /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="同步状态" placement="top" v-if="row.source_type === 'aliyun' && row.status === 'pending'">
                <el-button link class="action-btn action-sync" @click="handleSync(row)">
                  <el-icon><Refresh /></el-icon>
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

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadData"
          @current-change="loadData"
        />
      </div>
    </div>

    <!-- 申请证书对话框 -->
    <el-dialog
      v-model="applyDialogVisible"
      title="申请证书"
      width="680px"
      :close-on-click-modal="false"
      class="beauty-dialog"
      destroy-on-close
    >
      <div class="dialog-scroll-body">
      <el-form :model="applyForm" :rules="applyRules" ref="applyFormRef" label-width="120px" class="beauty-form">
        <el-form-item label="证书名称" prop="name">
          <el-input v-model="applyForm.name" placeholder="请输入证书名称" />
        </el-form-item>
        <el-form-item label="主域名" prop="domain">
          <el-input v-model="applyForm.domain" placeholder="请输入主域名，如：example.com" />
        </el-form-item>
        <el-form-item label="SAN域名">
          <el-select
            v-model="applyForm.san_domains"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入域名后按回车添加，如：www.example.com"
            style="width: 100%"
          />
          <div class="form-tip">可选，让一张证书保护多个域名。例如主域名是 example.com，可添加 www.example.com、api.example.com 等</div>
        </el-form-item>

        <el-divider content-position="left">证书配置</el-divider>

        <el-form-item label="证书类型" prop="source_type">
          <el-radio-group v-model="applyForm.source_type" @change="handleSourceTypeChange">
            <el-radio-button value="acme">ACME免费证书</el-radio-button>
            <el-radio-button value="aliyun">阿里云CAS</el-radio-button>
          </el-radio-group>
          <div class="form-tip" v-if="applyForm.source_type === 'acme'">
            ACME免费证书支持 Let's Encrypt、ZeroSSL 等 CA 提供商，有效期90天，支持自动续期
          </div>
          <div class="form-tip" v-else-if="applyForm.source_type === 'aliyun'">
            <el-icon style="color: #e6a23c; margin-right: 4px;"><Warning /></el-icon>
            阿里云CAS免费证书，每个实名账号每年20张额度，有效期1年，不支持自动续期
          </div>
        </el-form-item>

        <!-- 云账号选择 (云厂商证书) -->
        <el-form-item label="云账号" prop="cloud_account_id" v-if="applyForm.source_type === 'aliyun'">
          <el-select
            v-model="applyForm.cloud_account_id"
            placeholder="请选择云账号"
            style="width: 100%"
            :loading="cloudAccountsLoading"
          >
            <el-option
              v-for="item in cloudAccounts"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
          <div class="form-tip">
            <el-icon style="color: #e6a23c; margin-right: 4px;"><Warning /></el-icon>
            请在「资产管理 - 云账号」中添加云账号，需要有证书服务的相关权限
          </div>
        </el-form-item>

        <el-form-item label="CA提供商" prop="ca_provider" v-if="applyForm.source_type === 'acme'">
          <el-select v-model="applyForm.ca_provider" placeholder="请选择CA提供商" style="width: 100%">
            <el-option label="Let's Encrypt (推荐)" value="letsencrypt">
              <span>Let's Encrypt</span>
              <span style="color: #67c23a; margin-left: 8px; font-size: 12px;">推荐</span>
            </el-option>
            <el-option label="ZeroSSL" value="zerossl" />
            <el-option label="Google Trust Services" value="google" />
            <el-option label="BuyPass" value="buypass" />
          </el-select>
          <div class="form-tip">不同CA提供商的证书有效期和签发策略可能不同</div>
        </el-form-item>

        <el-form-item label="加密算法" prop="key_algorithm">
          <el-select v-model="applyForm.key_algorithm" placeholder="请选择加密算法" style="width: 100%">
            <el-option label="EC P-256 (推荐)" value="ec256">
              <span>EC P-256</span>
              <span style="color: #67c23a; margin-left: 8px; font-size: 12px;">推荐</span>
            </el-option>
            <el-option label="EC P-384" value="ec384" />
            <el-option label="RSA 2048" value="rsa2048" />
            <el-option label="RSA 3072" value="rsa3072" />
            <el-option label="RSA 4096" value="rsa4096" />
          </el-select>
          <div class="form-tip">EC算法性能更好，RSA兼容性更广</div>
        </el-form-item>

        <el-divider content-position="left" v-if="applyForm.source_type === 'acme'">域名验证配置</el-divider>

        <el-form-item label="邮箱地址" prop="acme_email" v-if="applyForm.source_type === 'acme'">
          <el-input v-model="applyForm.acme_email" placeholder="请输入邮箱地址" />
          <div class="form-tip">用于ACME账户注册和接收证书过期提醒</div>
        </el-form-item>
        <el-form-item label="DNS验证" prop="dns_provider_id" v-if="applyForm.source_type === 'acme'">
          <el-select v-model="applyForm.dns_provider_id" placeholder="请选择DNS服务商" style="width: 100%">
            <el-option
              v-for="item in dnsProviders"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            >
              <span>{{ item.name }}</span>
              <span style="color: #999; margin-left: 8px;">{{ item.provider }}</span>
            </el-option>
          </el-select>
          <div class="form-tip">
            <el-icon style="color: #e6a23c; margin-right: 4px;"><Warning /></el-icon>
            选择你域名所在的DNS服务商，系统将通过其API自动完成域名所有权验证（DNS-01验证）
          </div>
        </el-form-item>

        <el-divider content-position="left">续期配置</el-divider>

        <el-form-item label="自动续期">
          <el-switch v-model="applyForm.auto_renew" />
          <span style="margin-left: 12px; color: #909399; font-size: 13px;">证书到期前自动续期</span>
        </el-form-item>
        <el-form-item label="提前续期天数" v-if="applyForm.auto_renew">
          <el-input-number v-model="applyForm.renew_days_before" :min="7" :max="90" />
          <span style="margin-left: 12px; color: #909399; font-size: 13px;">天（Let's Encrypt证书建议30天）</span>
        </el-form-item>
      </el-form>
      </div>

      <template #footer>
        <el-button @click="applyDialogVisible = false">取消</el-button>
        <el-button class="black-button" @click="handleApplySubmit" :loading="submitting">申请证书</el-button>
      </template>
    </el-dialog>

    <!-- 导入证书对话框 -->
    <el-dialog
      v-model="importDialogVisible"
      title="导入证书"
      width="750px"
      :close-on-click-modal="false"
      class="beauty-dialog"
      destroy-on-close
    >
      <CertificateUpload 
        v-if="importDialogVisible"
        v-model:visible="importDialogVisible"
        @submit="handleCertificateUploaded" />
    </el-dialog>

    <!-- 编辑对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑证书配置"
      width="540px"
      :close-on-click-modal="false"
      class="beauty-dialog"
      destroy-on-close
    >
      <el-form :model="editForm" ref="editFormRef" label-width="120px" class="beauty-form">
        <el-form-item label="证书名称" prop="name">
          <el-input v-model="editForm.name" placeholder="请输入证书名称" />
        </el-form-item>
        <el-form-item label="自动续期" v-if="editForm.source_type !== 'manual'">
          <el-switch v-model="editForm.auto_renew" />
        </el-form-item>
        <el-form-item label="提前续期天数" v-if="editForm.auto_renew && editForm.source_type !== 'manual'">
          <el-input-number v-model="editForm.renew_days_before" :min="7" :max="90" />
          <span style="margin-left: 12px; color: #909399; font-size: 13px;">天</span>
        </el-form-item>
        <el-form-item label="DNS Provider" v-if="editForm.source_type !== 'manual'">
          <el-select v-model="editForm.dns_provider_id" placeholder="请选择DNS Provider" style="width: 100%" clearable>
            <el-option
              v-for="item in dnsProviders"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
          <div class="form-tip" v-if="editForm.source_type === 'aliyun'">
            云厂商证书如需通过ACME自动续期，请配置DNS Provider
          </div>
        </el-form-item>
        <el-form-item label="ACME邮箱" v-if="editForm.source_type !== 'manual'">
          <el-input v-model="editForm.acme_email" placeholder="用于ACME账户注册和证书过期提醒" />
          <div class="form-tip">
            手动续期或自动续期需要配置ACME邮箱
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button class="black-button" @click="handleEditSubmit" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="证书详情"
      width="720px"
      class="beauty-dialog"
    >
      <div v-if="currentCert" class="detail-content">
        <div class="detail-status-bar">
          <el-tag v-if="currentCert.status === 'active'" type="success" effect="dark" size="large">正常</el-tag>
          <el-tag v-else-if="currentCert.status === 'expiring'" type="warning" effect="dark" size="large">即将过期</el-tag>
          <el-tag v-else-if="currentCert.status === 'expired'" type="danger" effect="dark" size="large">已过期</el-tag>
          <el-tag v-else-if="currentCert.status === 'pending'" type="info" effect="dark" size="large">待申请</el-tag>
          <el-tag v-else type="danger" effect="dark" size="large">错误</el-tag>
          <span class="detail-domain">{{ currentCert.domain }}</span>
        </div>
        <div class="detail-info">
          <div class="detail-info-section">
            <div class="detail-section-title">基本信息</div>
            <div class="detail-grid">
              <div class="info-item">
                <span class="info-label">证书名称</span>
                <span class="info-value">{{ currentCert.name }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">主域名</span>
                <span class="info-value">{{ currentCert.domain }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">来源</span>
                <span class="info-value">{{ getSourceTypeName(currentCert.source_type) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">CA提供商</span>
                <span class="info-value">{{ getCAProviderName(currentCert.ca_provider) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">加密算法</span>
                <span class="info-value">{{ getKeyAlgorithmName(currentCert.key_algorithm) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">颁发者</span>
                <span class="info-value">{{ currentCert.issuer || '-' }}</span>
              </div>
            </div>
          </div>
          <div class="detail-info-section">
            <div class="detail-section-title">有效期</div>
            <div class="detail-grid">
              <div class="info-item">
                <span class="info-label">生效时间</span>
                <span class="info-value">{{ formatDateTime(currentCert.not_before) || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">到期时间</span>
                <span class="info-value" :class="getExpiryClass(currentCert.not_after)">
                  {{ formatDateTime(currentCert.not_after) || '-' }}
                  <span v-if="currentCert.not_after" class="expiry-days-inline">{{ getExpiryDays(currentCert.not_after) }}</span>
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">自动续期</span>
                <span class="info-value">{{ currentCert.auto_renew ? '是' : '否' }}</span>
              </div>
              <div class="info-item" v-if="currentCert.auto_renew">
                <span class="info-label">提前续期</span>
                <span class="info-value">{{ currentCert.renew_days_before }} 天</span>
              </div>
              <div class="info-item" v-if="currentCert.last_renew_at">
                <span class="info-label">上次续期</span>
                <span class="info-value">{{ formatDateTime(currentCert.last_renew_at) }}</span>
              </div>
            </div>
          </div>
          <div class="detail-info-section">
            <div class="detail-section-title">安全信息</div>
            <div class="detail-grid">
              <div class="info-item full-width">
                <span class="info-label">指纹</span>
                <span class="info-value fingerprint">{{ currentCert.fingerprint || '-' }}</span>
              </div>
            </div>
          </div>
          <div class="detail-info-section" v-if="currentCert.last_error">
            <div class="detail-section-title error-section-title">错误信息</div>
            <div class="error-block">
              {{ formatErrorMessage(currentCert.last_error) }}
            </div>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 下载对话框 -->
    <el-dialog
      v-model="downloadDialogVisible"
      title="下载证书"
      width="620px"
      class="beauty-dialog"
    >
      <div class="download-card">
        <div class="download-card-header">
          <div class="download-domain-info">
            <el-icon class="domain-icon"><Key /></el-icon>
            <div class="domain-text">
              <span class="domain-name">{{ downloadCertDomain }}</span>
              <span class="domain-hint">SSL 证书文件</span>
            </div>
          </div>
        </div>

        <div class="download-format-selector">
          <div
            class="format-option"
            :class="{ active: downloadFormat === 'pem' }"
            @click="downloadFormat = 'pem'"
          >
            <el-icon class="format-icon"><Document /></el-icon>
            <div class="format-info">
              <span class="format-name">PEM 格式</span>
              <span class="format-desc">通用格式</span>
            </div>
          </div>
          <div
            class="format-option"
            :class="{ active: downloadFormat === 'nginx' }"
            @click="downloadFormat = 'nginx'"
          >
            <el-icon class="format-icon"><Connection /></el-icon>
            <div class="format-info">
              <span class="format-name">Nginx 格式</span>
              <span class="format-desc">fullchain + key</span>
            </div>
          </div>
        </div>

        <div class="download-files">
          <div class="file-item" v-if="downloadFormat === 'pem'">
            <div class="file-icon cert-icon">
              <el-icon><Document /></el-icon>
            </div>
            <div class="file-info">
              <span class="file-name">{{ downloadCertDomain }}.pem</span>
              <span class="file-size">证书文件</span>
            </div>
            <div class="file-actions">
              <el-button text type="primary" @click="copyToClipboard(downloadContent.certificate)">
                <el-icon><DocumentCopy /></el-icon>
              </el-button>
              <el-button text type="primary" @click="downloadFile(downloadContent.certificate, `${downloadCertDomain}.pem`)">
                <el-icon><Download /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="file-item" v-if="downloadFormat === 'pem'">
            <div class="file-icon key-icon">
              <el-icon><Key /></el-icon>
            </div>
            <div class="file-info">
              <span class="file-name">{{ downloadCertDomain }}.key</span>
              <span class="file-size">私钥文件</span>
            </div>
            <div class="file-actions">
              <el-button text type="primary" @click="copyToClipboard(downloadContent.private_key)">
                <el-icon><DocumentCopy /></el-icon>
              </el-button>
              <el-button text type="primary" @click="downloadFile(downloadContent.private_key, `${downloadCertDomain}.key`)">
                <el-icon><Download /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="file-item" v-if="downloadFormat === 'nginx'">
            <div class="file-icon cert-icon">
              <el-icon><Document /></el-icon>
            </div>
            <div class="file-info">
              <span class="file-name">{{ downloadCertDomain }}_fullchain.pem</span>
              <span class="file-size">完整证书链</span>
            </div>
            <div class="file-actions">
              <el-button text type="primary" @click="copyToClipboard(downloadContent.ssl_certificate)">
                <el-icon><DocumentCopy /></el-icon>
              </el-button>
              <el-button text type="primary" @click="downloadFile(downloadContent.ssl_certificate, `${downloadCertDomain}_fullchain.pem`)">
                <el-icon><Download /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="file-item" v-if="downloadFormat === 'nginx'">
            <div class="file-icon key-icon">
              <el-icon><Key /></el-icon>
            </div>
            <div class="file-info">
              <span class="file-name">{{ downloadCertDomain }}.key</span>
              <span class="file-size">私钥文件</span>
            </div>
            <div class="file-actions">
              <el-button text type="primary" @click="copyToClipboard(downloadContent.ssl_certificate_key)">
                <el-icon><DocumentCopy /></el-icon>
              </el-button>
              <el-button text type="primary" @click="downloadFile(downloadContent.ssl_certificate_key, `${downloadCertDomain}.key`)">
                <el-icon><Download /></el-icon>
              </el-button>
            </div>
          </div>
        </div>

        <div class="download-action">
          <el-button type="primary" size="large" @click="downloadAllAsZip" :loading="downloadingZip">
            <el-icon style="margin-right: 8px;"><Download /></el-icon>
            下载 ZIP 压缩包
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import {
  Plus,
  Search,
  RefreshLeft,
  Refresh,
  RefreshRight,
  Edit,
  Delete,
  View,
  Key,
  Document,
  CircleCheck,
  CircleClose,
  Warning,
  Download,
  ArrowDown,
  Connection,
  DocumentCopy
} from '@element-plus/icons-vue'
import JSZip from 'jszip'
import {
  getCertificates,
  getCertificate,
  createCertificate,
  importCertificate,
  updateCertificate,
  deleteCertificate,
  renewCertificate,
  syncCertificate,
  downloadCertificate,
  getCertificateStats,
  getAllDNSProviders,
  getCloudAccounts
} from '../api/ssl-cert'
import CertificateUpload from './CertificateUpload.vue'

const loading = ref(false)
const submitting = ref(false)

// 对话框状态
const applyDialogVisible = ref(false)
const importDialogVisible = ref(false)
const editDialogVisible = ref(false)
const detailDialogVisible = ref(false)
const downloadDialogVisible = ref(false)

// 表单引用
const applyFormRef = ref<FormInstance>()
const importFormRef = ref<FormInstance>()
const editFormRef = ref<FormInstance>()

// 搜索表单
const searchForm = reactive({
  domain: '',
  status: '',
  source_type: ''
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 统计数据
const stats = ref<Record<string, number>>({})

// 表格数据
const tableData = ref<any[]>([])

// DNS Providers
const dnsProviders = ref<any[]>([])

// Cloud Accounts
const cloudAccounts = ref<any[]>([])
const cloudAccountsLoading = ref(false)
// 当前查看的证书
const currentCert = ref<any>(null)

// 申请表单
const applyForm = reactive({
  name: '',
  domain: '',
  san_domains: [] as string[],
  acme_email: '',
  source_type: 'acme',
  ca_provider: 'letsencrypt',
  key_algorithm: 'ec256',
  dns_provider_id: null as number | null,
  cloud_account_id: null as number | null,
  auto_renew: true,
  renew_days_before: 30
})

// 导入表单
const importForm = reactive({
  name: '',
  domain: '',
  san_domains: [] as string[],
  certificate: '',
  private_key: '',
  cert_chain: ''
})

// 编辑表单
const editForm = reactive({
  id: 0,
  name: '',
  auto_renew: true,
  renew_days_before: 30,
  dns_provider_id: null as number | null,
  acme_email: '',
  source_type: ''
})

// 下载内容
const downloadFormat = ref('pem')
const downloadCertDomain = ref('')
const downloadingZip = ref(false)
const downloadContent = reactive({
  certificate: '',
  private_key: '',
  cert_chain: '',
  ssl_certificate: '',
  ssl_certificate_key: ''
})

// 表单验证规则
const applyRules: FormRules = {
  name: [{ required: true, message: '请输入证书名称', trigger: 'blur' }],
  domain: [{ required: true, message: '请输入主域名', trigger: 'blur' }],
  acme_email: [
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' }
  ]
  // dns_provider_id 和 cloud_account_id 在 handleApplySubmit 中根据 source_type 动态验证
}

const importRules: FormRules = {
  name: [{ required: true, message: '请输入证书名称', trigger: 'blur' }],
  domain: [{ required: true, message: '请输入主域名', trigger: 'blur' }],
  certificate: [{ required: true, message: '请输入证书内容', trigger: 'blur' }],
  private_key: [{ required: true, message: '请输入私钥内容', trigger: 'blur' }]
}

// 获取来源类型名称
const getSourceTypeName = (type: string) => {
  const names: Record<string, string> = {
    acme: 'ACME免费证书',
    letsencrypt: "Let's Encrypt",
    aliyun: '阿里云CAS',
    manual: '手动导入'
  }
  return names[type] || type
}

// 获取CA提供商名称
const getCAProviderName = (provider: string) => {
  const names: Record<string, string> = {
    letsencrypt: "Let's Encrypt",
    zerossl: 'ZeroSSL',
    google: 'Google Trust Services',
    buypass: 'BuyPass',
    aliyun: '阿里云CAS'
  }
  return names[provider] || provider || '-'
}

// 获取密钥算法名称
const getKeyAlgorithmName = (algorithm: string) => {
  const names: Record<string, string> = {
    rsa2048: 'RSA 2048',
    rsa3072: 'RSA 3072',
    rsa4096: 'RSA 4096',
    ec256: 'EC P-256',
    ec384: 'EC P-384'
  }
  return names[algorithm] || algorithm || '-'
}

// 格式化错误信息
const formatErrorMessage = (error: string) => {
  if (!error) return ''

  // 阿里云常见错误
  if (error.includes('InsufficientQuota')) {
    return '❌ 阿里云证书额度不足\n\n' +
           '可能原因：\n' +
           '1. 账号欠费（最常见！即使申请免费证书也需要账号无欠费）\n' +
           '2. 免费证书额度已用完（每个实名账号每年20张）\n' +
           '3. 未领取免费证书资源包\n\n' +
           '解决方案：\n' +
           '1. 【优先】检查账号是否欠费，如果欠费请先充值（哪怕1元也可以）\n' +
           '2. 登录阿里云SSL证书控制台，查看是否需要先"领取"免费证书资源包\n' +
           '3. 查看剩余免费证书额度\n' +
           '4. 如果额度用完，可购买证书资源包或使用 ACME 免费证书（Let\'s Encrypt）'
  }

  if (error.includes('InvalidDomain')) {
    return '❌ 域名验证失败\n\n' +
           '原因：域名格式不正确或域名不属于您\n\n' +
           '解决方案：\n' +
           '1. 检查域名是否已实名认证\n' +
           '2. 确认域名解析是否正常\n' +
           '3. 确保域名未被其他账号占用'
  }

  // 通用错误格式化
  if (error.includes('ErrorCode:')) {
    // 提取关键错误信息
    const lines = error.split('\n')
    const errorCode = lines.find(l => l.includes('ErrorCode:'))?.replace('ErrorCode:', '').trim()
    const message = lines.find(l => l.includes('Message:'))?.replace('Message:', '').trim()

    if (errorCode && message) {
      return `❌ ${message}\n\n错误码: ${errorCode}\n\n完整错误信息:\n${error}`
    }
  }

  return error
}

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return '-'
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 获取到期天数
const getExpiryDays = (notAfter: string) => {
  if (!notAfter) return ''
  const now = new Date()
  const expiry = new Date(notAfter)
  const days = Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
  if (days < 0) return `已过期 ${Math.abs(days)} 天`
  if (days === 0) return '今天到期'
  return `剩余 ${days} 天`
}

// 获取到期样式
const getExpiryClass = (notAfter: string) => {
  if (!notAfter) return ''
  const now = new Date()
  const expiry = new Date(notAfter)
  const days = Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
  if (days < 0) return 'expiry-expired'
  if (days <= 7) return 'expiry-danger'
  if (days <= 30) return 'expiry-warning'
  return 'expiry-normal'
}

// 重置搜索
const handleReset = () => {
  searchForm.domain = ''
  searchForm.status = ''
  searchForm.source_type = ''
  pagination.page = 1
  loadData()
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const [certsRes, statsRes] = await Promise.all([
      getCertificates({
        page: pagination.page,
        page_size: pagination.pageSize,
        domain: searchForm.domain || undefined,
        status: searchForm.status || undefined,
        source_type: searchForm.source_type || undefined
      }),
      getCertificateStats()
    ])
    tableData.value = certsRes.list || []
    pagination.total = certsRes.total || 0
    stats.value = statsRes || {}
    // 计算证书总数（后端按状态分组返回，需要前端汇总）
    if (!stats.value.total) {
      stats.value.total = Object.values(stats.value).reduce((sum: number, val: any) => sum + (Number(val) || 0), 0)
    }
  } catch (error: any) {
    // 错误已由 request 拦截器处理
  } finally {
    loading.value = false
  }
}

// 加载DNS Providers
const loadDNSProviders = async () => {
  try {
    const res = await getAllDNSProviders()
    dnsProviders.value = res || []
  } catch (error) {
    // ignore
  }
}

// 加载云账号列表
const loadCloudAccounts = async (provider?: string) => {
  cloudAccountsLoading.value = true
  try {
    const res = await getCloudAccounts(provider)
    cloudAccounts.value = res || []
  } catch (error) {
    cloudAccounts.value = []
  } finally {
    cloudAccountsLoading.value = false
  }
}

// 监听证书类型变化
const handleSourceTypeChange = (val: string) => {
  // 重置云账号选择
  applyForm.cloud_account_id = null

  if (val === 'aliyun') {
    // 加载对应的云账号
    loadCloudAccounts(val)
  }
}

// 处理创建
const handleCreate = (command: string) => {
  if (command === 'apply') {
    Object.assign(applyForm, {
      name: '',
      domain: '',
      san_domains: [],
      acme_email: '',
      source_type: 'acme',
      ca_provider: 'letsencrypt',
      key_algorithm: 'ec256',
      dns_provider_id: null,
      cloud_account_id: null,
      auto_renew: true,
      renew_days_before: 30
    })
    cloudAccounts.value = []
    applyDialogVisible.value = true
  } else if (command === 'import') {
    Object.assign(importForm, {
      name: '',
      domain: '',
      san_domains: [],
      certificate: '',
      private_key: '',
      cert_chain: ''
    })
    importDialogVisible.value = true
  }
}

// 申请证书提交
const handleApplySubmit = async () => {
  if (!applyFormRef.value) return
  await applyFormRef.value.validate(async (valid) => {
    if (valid) {
      // 云厂商证书验证
      if (applyForm.source_type === 'aliyun' && !applyForm.cloud_account_id) {
        ElMessage.warning('请选择云账号')
        return
      }
      // ACME证书验证
      if (applyForm.source_type === 'acme') {
        if (!applyForm.acme_email) {
          ElMessage.warning('请输入邮箱地址')
          return
        }
        if (!applyForm.dns_provider_id) {
          ElMessage.warning('请选择DNS验证配置')
          return
        }
      }

      submitting.value = true
      try {
        const data: any = {
          name: applyForm.name,
          domain: applyForm.domain,
          san_domains: applyForm.san_domains,
          source_type: applyForm.source_type,
          key_algorithm: applyForm.key_algorithm,
          auto_renew: applyForm.auto_renew,
          renew_days_before: applyForm.renew_days_before
        }

        // ACME证书需要额外参数
        if (applyForm.source_type === 'acme') {
          data.acme_email = applyForm.acme_email
          data.ca_provider = applyForm.ca_provider
          data.dns_provider_id = applyForm.dns_provider_id!
        }

        // 云厂商证书需要云账号
        if (applyForm.source_type === 'aliyun') {
          data.cloud_account_id = applyForm.cloud_account_id!
        }

        await createCertificate(data)
        ElMessage.success('证书申请已提交，正在后台处理')
        applyDialogVisible.value = false
        loadData()
      } catch (error: any) {
        // 错误已由 request 拦截器处理
      } finally {
        submitting.value = false
      }
    }
  })
}

// 导入证书提交
const handleImportSubmit = async () => {
  if (!importFormRef.value) return
  await importFormRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        await importCertificate({
          name: importForm.name,
          domain: importForm.domain,
          san_domains: importForm.san_domains,
          certificate: importForm.certificate,
          private_key: importForm.private_key,
          cert_chain: importForm.cert_chain
        })
        ElMessage.success('证书导入成功')
        importDialogVisible.value = false
        loadData()
      } catch (error: any) {
        // 错误已由 request 拦截器处理
      } finally {
        submitting.value = false
      }
    }
  })
}

// 处理证书上传
const handleCertificateUploaded = async (certData: any) => {
  submitting.value = true
  try {
    await importCertificate({
      name: certData.name,
      domain: certData.domain,
      certificate: certData.certificate,
      private_key: certData.privateKey,
      cert_chain: ''
    })
    ElMessage.success('证书导入成功')
    importDialogVisible.value = false
    loadData()
  } catch (error: any) {
    // 错误已由 request 拦截器处理
  } finally {
    submitting.value = false
  }
}

// 查看详情
const handleView = async (row: any) => {
  try {
    const res = await getCertificate(row.id)
    currentCert.value = res
    detailDialogVisible.value = true
  } catch (error) {
    // 错误已由 request 拦截器处理
  }
}

// 编辑
const handleEdit = async (row: any) => {
  try {
    // 获取证书详情以确保有完整数据
    const cert = await getCertificate(row.id)
    Object.assign(editForm, {
      id: cert.id,
      name: cert.name,
      auto_renew: cert.auto_renew,
      renew_days_before: cert.renew_days_before || 30,
      dns_provider_id: cert.dns_provider_id || null,
      acme_email: cert.acme_email || '',
      source_type: cert.source_type
    })
    editDialogVisible.value = true
  } catch (error) {
    // 如果获取详情失败，使用列表数据
    Object.assign(editForm, {
      id: row.id,
      name: row.name,
      auto_renew: row.auto_renew,
      renew_days_before: row.renew_days_before || 30,
      dns_provider_id: row.dns_provider_id || null,
      acme_email: row.acme_email || '',
      source_type: row.source_type
    })
    editDialogVisible.value = true
  }
}

// 编辑提交
const handleEditSubmit = async () => {
  submitting.value = true
  try {
    await updateCertificate(editForm.id, {
      name: editForm.name,
      auto_renew: editForm.auto_renew,
      renew_days_before: editForm.renew_days_before,
      dns_provider_id: editForm.dns_provider_id || undefined,
      acme_email: editForm.acme_email || undefined
    })
    ElMessage.success('保存成功')
    editDialogVisible.value = false
    loadData()
  } catch (error: any) {
    // 错误已由 request 拦截器处理
  } finally {
    submitting.value = false
  }
}

// 自动续期切换
const handleAutoRenewChange = async (row: any) => {
  try {
    await updateCertificate(row.id, { auto_renew: row.auto_renew })
    ElMessage.success('更新成功')
  } catch (error) {
    row.auto_renew = !row.auto_renew
  }
}

// 同步云证书状态
const handleSync = async (row: any) => {
  try {
    loading.value = true
    await syncCertificate(row.id)
    ElMessage.success('证书同步成功')
    loadData()
  } catch (error: any) {
    // 错误已由 request 拦截器处理
  } finally {
    loading.value = false
  }
}

// 手动续期证书
const handleRenew = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      `确定要手动续期证书 "${row.name}" 吗？续期需要配置DNS Provider和ACME邮箱。`,
      '手动续期',
      { type: 'warning' }
    )
    loading.value = true
    await renewCertificate(row.id)
    ElMessage.success('续期任务已提交，请在任务记录中查看进度')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      // 错误已由 request 拦截器处理
    }
  } finally {
    loading.value = false
  }
}

// 下载证书
const handleDownload = async (row: any) => {
  try {
    downloadCertDomain.value = row.domain
    const [pemRes, nginxRes] = await Promise.all([
      downloadCertificate(row.id, 'pem'),
      downloadCertificate(row.id, 'nginx')
    ])
    downloadContent.certificate = pemRes.certificate || ''
    downloadContent.private_key = pemRes.private_key || ''
    downloadContent.cert_chain = pemRes.cert_chain || ''
    downloadContent.ssl_certificate = nginxRes.ssl_certificate || ''
    downloadContent.ssl_certificate_key = nginxRes.ssl_certificate_key || ''
    downloadDialogVisible.value = true
  } catch (error) {
    // 错误已由 request 拦截器处理
  }
}

// 下载单个文件
const downloadFile = (content: string, filename: string) => {
  if (!content) {
    ElMessage.warning('文件内容为空')
    return
  }
  const blob = new Blob([content], { type: 'application/x-pem-file' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
  ElMessage.success(`已下载 ${filename}`)
}

// 下载全部文件（已弃用，保留兼容）
const downloadAllFiles = () => {
  const domain = downloadCertDomain.value
  if (downloadFormat.value === 'pem') {
    downloadFile(downloadContent.certificate, `${domain}.pem`)
    setTimeout(() => downloadFile(downloadContent.private_key, `${domain}.key`), 200)
  } else {
    downloadFile(downloadContent.ssl_certificate, `${domain}_fullchain.pem`)
    setTimeout(() => downloadFile(downloadContent.ssl_certificate_key, `${domain}.key`), 200)
  }
}

// 下载全部文件为ZIP压缩包
const downloadAllAsZip = async () => {
  downloadingZip.value = true
  try {
    const zip = new JSZip()
    const domain = downloadCertDomain.value || 'certificate'

    if (downloadFormat.value === 'pem') {
      if (downloadContent.certificate) {
        zip.file(`${domain}.pem`, downloadContent.certificate)
      }
      if (downloadContent.private_key) {
        zip.file(`${domain}.key`, downloadContent.private_key)
      }
      if (downloadContent.cert_chain) {
        zip.file(`${domain}_ca.pem`, downloadContent.cert_chain)
      }
    } else {
      if (downloadContent.ssl_certificate) {
        zip.file(`${domain}_fullchain.pem`, downloadContent.ssl_certificate)
      }
      if (downloadContent.ssl_certificate_key) {
        zip.file(`${domain}.key`, downloadContent.ssl_certificate_key)
      }
    }

    const content = await zip.generateAsync({ type: 'blob' })
    const url = URL.createObjectURL(content)
    const link = document.createElement('a')
    link.href = url
    link.download = `${domain}_ssl.zip`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
    ElMessage.success('证书已打包下载')
  } catch (error) {
    ElMessage.error('打包下载失败')
  } finally {
    downloadingZip.value = false
  }
}

// 删除
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该证书吗？关联的部署配置也将被删除。', '提示', { type: 'warning' })
    loading.value = true
    await deleteCertificate(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      // 错误已由 request 拦截器处理
    }
  } finally {
    loading.value = false
  }
}

// 复制到剪贴板
const copyToClipboard = (text: string) => {
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

onMounted(() => {
  loadData()
  loadDNSProviders()
})
</script>

<style scoped>
.certificate-container {
  padding: 0;
  background-color: transparent;
}

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
  width: 200px;
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

.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 12px;
}

.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
}

.stat-icon-primary {
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  color: #d4af37;
  border: 1px solid #d4af37;
}

.stat-icon-success {
  background: linear-gradient(135deg, #4caf50 0%, #45a049 100%);
  color: #fff;
}

.stat-icon-warning {
  background: linear-gradient(135deg, #e6a23c 0%, #d9972c 100%);
  color: #fff;
}

.stat-icon-danger {
  background: linear-gradient(135deg, #f56c6c 0%, #f4534a 100%);
  color: #fff;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 4px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
}

.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.cert-name {
  font-weight: 500;
}

.domain-cell {
  display: flex;
  align-items: center;
}

.domain-main {
  font-family: 'Monaco', 'Menlo', monospace;
}

.ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.expiry-days {
  font-size: 12px;
  color: #909399;
}

.expiry-normal {
  color: #67c23a;
}

.expiry-warning {
  color: #e6a23c;
}

.expiry-danger {
  color: #f56c6c;
}

.expiry-expired {
  color: #909399;
  text-decoration: line-through;
}

.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-view:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-download:hover {
  background-color: #e8f5e9;
  color: #67c23a;
}

.action-sync:hover {
  background-color: #e8f5e9;
  color: #67c23a;
}

.action-renew:hover {
  background-color: #fff3e0;
  color: #e6a23c;
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.pagination-wrapper {
  padding: 16px;
  display: flex;
  justify-content: flex-end;
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

.form-tip {
  font-size: 12px;
  color: #999;
  margin-top: 6px;
  line-height: 1.5;
}

/* ========== 统一弹窗美化样式 ========== */
:deep(.beauty-dialog) {
  border-radius: 16px;
  overflow: hidden;
}

:deep(.beauty-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  margin-right: 0;
  border-bottom: 1px solid #f0f0f0;
  background: #fafbfc;
}

:deep(.beauty-dialog .el-dialog__title) {
  font-size: 17px;
  font-weight: 600;
  color: #1a1a1a;
}

:deep(.beauty-dialog .el-dialog__headerbtn) {
  top: 20px;
  right: 20px;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.beauty-dialog .el-dialog__headerbtn:hover) {
  background: #f0f0f0;
}

:deep(.beauty-dialog .el-dialog__body) {
  padding: 24px;
  max-height: 65vh;
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

:deep(.beauty-dialog .el-dialog__body::-webkit-scrollbar) {
  display: none;
}

:deep(.beauty-dialog .el-dialog__footer) {
  padding: 16px 24px 20px;
  border-top: 1px solid #f0f0f0;
  background: #fafbfc;
}

/* 表单美化 */
:deep(.beauty-form .el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

:deep(.beauty-form .el-input__wrapper),
:deep(.beauty-form .el-textarea__inner) {
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.beauty-form .el-input__wrapper:hover),
:deep(.beauty-form .el-textarea__inner:hover) {
  box-shadow: 0 0 0 1px #c0c4cc inset;
}

:deep(.beauty-form .el-input__wrapper.is-focus),
:deep(.beauty-form .el-textarea__inner:focus) {
  box-shadow: 0 0 0 1px #000 inset;
}

:deep(.beauty-form .el-select .el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #000 inset !important;
}

:deep(.beauty-form .el-divider__text) {
  font-size: 13px;
  font-weight: 600;
  color: #909399;
  background: #fff;
}

/* ========== 详情弹窗样式 ========== */
.detail-content {
  padding: 0;
}

.detail-status-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  padding: 16px;
  background: #f8fafc;
  border-radius: 10px;
  border: 1px solid #e8ecf0;
}

.detail-domain {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
  font-family: 'Monaco', 'Menlo', monospace;
}

.detail-info {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.detail-info-section {
  background: #fff;
  border: 1px solid #e8ecf0;
  border-radius: 10px;
  overflow: hidden;
}

.detail-section-title {
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 600;
  color: #909399;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background: #f8fafc;
  border-bottom: 1px solid #e8ecf0;
}

.error-section-title {
  color: #f56c6c;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0;
}

.detail-grid .info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 16px;
  border-bottom: 1px solid #f5f5f5;
  border-right: 1px solid #f5f5f5;
}

.detail-grid .info-item:nth-child(2n) {
  border-right: none;
}

.detail-grid .info-item:last-child,
.detail-grid .info-item:nth-last-child(2):nth-child(odd) {
  border-bottom: none;
}

.detail-grid .info-item.full-width {
  grid-column: span 2;
  border-right: none;
}

.info-label {
  color: #909399;
  font-size: 12px;
  font-weight: 500;
}

.info-value {
  color: #303133;
  font-size: 14px;
  word-break: break-all;
  font-weight: 500;
}

.fingerprint {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
}

.expiry-days-inline {
  font-size: 12px;
  margin-left: 8px;
  font-weight: 400;
  color: #909399;
}

.error-block {
  padding: 14px 16px;
  font-size: 13px;
  color: #f56c6c;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
}

.error-text {
  color: #f56c6c;
}

/* ========== 下载弹窗样式 ========== */
.download-card {
  background: #fff;
  border-radius: 12px;
}

.download-card-header {
  padding: 16px 0;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 20px;
}

.download-domain-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.domain-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  border: 1px solid #d4af37;
}

.domain-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.domain-name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  font-family: 'Monaco', 'Menlo', monospace;
}

.domain-hint {
  font-size: 13px;
  color: #909399;
}

.download-format-selector {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.format-option {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border: 2px solid #e4e7ed;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.format-option:hover {
  border-color: #c0c4cc;
  background: #fafbfc;
}

.format-option.active {
  border-color: #000;
  background: #f5f5f5;
}

.format-icon {
  width: 40px;
  height: 40px;
  background: #f0f2f5;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  color: #606266;
}

.format-option.active .format-icon {
  background: #000;
  color: #d4af37;
}

.format-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.format-name {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.format-desc {
  font-size: 12px;
  color: #909399;
}

.download-files {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 24px;
  padding: 16px;
  background: #fafbfc;
  border-radius: 10px;
  border: 1px solid #e4e7ed;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #ebeef5;
  transition: all 0.2s ease;
}

.file-item:hover {
  border-color: #c0c4cc;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.file-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
}

.cert-icon {
  background: #e8f4ff;
  color: #409eff;
}

.key-icon {
  background: #fff3e0;
  color: #e6a23c;
}

.file-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.file-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  font-family: 'Monaco', 'Menlo', monospace;
}

.file-size {
  font-size: 12px;
  color: #909399;
}

.file-actions {
  display: flex;
  gap: 4px;
}

.download-action {
  display: flex;
  justify-content: center;
  padding-top: 8px;
}

.download-action .el-button {
  min-width: 200px;
  border-radius: 8px;
  background: #000;
  border-color: #000;
}

.download-action .el-button:hover {
  background: #333;
  border-color: #333;
}
</style>
