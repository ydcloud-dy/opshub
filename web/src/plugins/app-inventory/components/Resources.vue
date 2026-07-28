<template>
  <div class="inventory-page">
    <PageHeader :icon="Coin" title="资源与域名" description="维护应用入口、主机与 Kubernetes 部署资源，以及数据库和中间件连接信息。">
      <template #metrics>
        <div class="inventory-header-metrics">
          <div class="inventory-header-metric"><strong>{{ domains.length }}</strong><span>入口域名</span></div>
          <div class="inventory-header-metric"><strong>{{ resources.length }}</strong><span>部署资源</span></div>
          <div class="inventory-header-metric"><strong>{{ components.length }}</strong><span>数据组件</span></div>
          <div class="inventory-header-metric"><strong>{{ unhealthyCount }}</strong><span>异常资产</span></div>
        </div>
      </template>
      <el-button :icon="Refresh" :loading="loading" @click="loadAll">刷新</el-button>
      <el-button :icon="Download" @click="openSync">从 K8s 同步</el-button>
      <el-button type="primary" :icon="Plus" @click="openCreateForTab">新增{{ activeTabLabel }}</el-button>
    </PageHeader>

    <div class="inventory-toolbar">
      <el-select v-model="filters.appId" clearable filterable placeholder="全部应用" @change="handleAppFilter"><el-option v-for="app in applications" :key="app.id" :label="app.name" :value="app.id" /></el-select>
      <el-select v-model="filters.environmentId" clearable filterable placeholder="全部共享环境" :disabled="Boolean(filters.appId)" @change="loadAll"><el-option v-for="env in environments" :key="env.id" :label="env.name" :value="env.id" /></el-select>
      <el-input v-model="filters.keyword" clearable placeholder="名称、地址或域名" :prefix-icon="Search" @keyup.enter="loadAll" />
      <el-select v-model="filters.status" clearable placeholder="自动检测状态" @change="loadAll"><el-option label="健康" value="healthy" /><el-option label="关注" value="warning" /><el-option label="异常" value="unhealthy" /><el-option label="检测中" value="checking" /><el-option label="未知" value="unknown" /></el-select>
      <el-button type="primary" :icon="Search" @click="loadAll">查询</el-button>
      <el-button text @click="resetFilters">重置</el-button>
    </div>

    <section class="inventory-panel">
      <el-tabs v-model="activeTab">
        <el-tab-pane name="domains">
          <template #label>域名与证书 <el-badge :value="domains.length" :max="99" /></template>
          <el-table v-loading="loading" :data="domains" stripe>
            <el-table-column label="访问入口" min-width="250"><template #default="{ row }"><strong>{{ domainEndpoint(row) }}</strong><div class="inventory-table-cell__sub">{{ appName(row.applicationId) }} · {{ envName(row.environmentId) }}</div></template></el-table-column>
            <el-table-column label="自动探测" min-width="220"><template #default="{ row }"><StatusTag :status="row.status" /><div class="inventory-table-cell__sub" :title="row.probeMessage">{{ row.probeMessage || '等待检测' }}</div><div class="inventory-table-cell__sub">{{ probeMeta(row) }}</div></template></el-table-column>
            <el-table-column label="DNS 与证书" min-width="190"><template #default="{ row }">{{ row.certificateName || row.tlsIssuer || '未关联证书' }}<div v-if="row.certificateExpiry || row.tlsExpiresAt" class="inventory-table-cell__sub">{{ certificateHint(row.certificateExpiry || row.tlsExpiresAt) }}</div><div class="inventory-table-cell__sub">{{ row.dnsProvider || 'DNS 服务商待识别' }}</div></template></el-table-column>
            <el-table-column label="来源" width="110"><template #default="{ row }"><el-tag size="small" effect="plain">{{ sourceLabel(row.source) }}</el-tag></template></el-table-column>
            <el-table-column label="操作" width="190" fixed="right"><template #default="{ row }"><el-button link type="primary" :loading="probingKey === `domain:${row.id}`" @click="runDomainProbe(row)">检测</el-button><el-button link type="primary" @click="openDomain(row)">编辑</el-button><el-button link type="danger" @click="removeDomain(row)">删除</el-button></template></el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane name="resources">
          <template #label>部署资源 <el-badge :value="resources.length" :max="99" /></template>
          <el-table v-loading="loading" :data="resources" stripe>
            <el-table-column label="资源" min-width="220"><template #default="{ row }"><strong>{{ row.name }}</strong><div class="inventory-table-cell__sub">{{ resourceKindLabel(row.kind) }} · {{ appName(row.applicationId) }}</div></template></el-table-column>
            <el-table-column label="位置" min-width="210"><template #default="{ row }">{{ resourceLocation(row) }}<div class="inventory-table-cell__sub">{{ envName(row.environmentId) }}</div></template></el-table-column>
            <el-table-column label="自动探测" min-width="220"><template #default="{ row }"><StatusTag :status="row.status" /><div class="inventory-table-cell__sub" :title="row.healthMessage">{{ row.healthMessage || '等待检测' }}</div><div class="inventory-table-cell__sub">{{ probeMeta(row) }}</div></template></el-table-column>
            <el-table-column label="来源" width="110"><template #default="{ row }"><el-tag size="small" effect="plain">{{ sourceLabel(row.source) }}</el-tag></template></el-table-column>
            <el-table-column label="操作" width="190" fixed="right"><template #default="{ row }"><el-button link type="primary" :loading="probingKey === `resource:${row.id}`" @click="runResourceProbe(row)">检测</el-button><el-button link type="primary" @click="openResource(row)">编辑</el-button><el-button link type="danger" @click="removeResource(row)">删除</el-button></template></el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane name="components">
          <template #label>数据库与中间件 <el-badge :value="components.length" :max="99" /></template>
          <el-table v-loading="loading" :data="components" stripe>
            <el-table-column label="组件" min-width="220"><template #default="{ row }"><strong>{{ row.name }}</strong><div class="inventory-table-cell__sub">{{ categoryLabel(row.category) }} · {{ row.type }}{{ row.version ? ` ${row.version}` : '' }}</div></template></el-table-column>
            <el-table-column label="连接" min-width="210"><template #default="{ row }">{{ endpoint(row.address, row.port) || '-' }}<div class="inventory-table-cell__sub">{{ appName(row.applicationId) }} · {{ envName(row.environmentId) }}</div></template></el-table-column>
            <el-table-column label="凭据" width="110"><template #default="{ row }"><el-tag v-if="row.credentialId" type="success" size="small" effect="plain">已托管</el-tag><span v-else class="inventory-muted">未关联</span></template></el-table-column>
            <el-table-column label="自动探测" min-width="220"><template #default="{ row }"><StatusTag :status="row.status" /><div class="inventory-table-cell__sub" :title="row.healthMessage">{{ row.healthMessage || '等待检测' }}</div><div class="inventory-table-cell__sub">{{ probeMeta(row) }}</div></template></el-table-column>
            <el-table-column label="操作" width="190" fixed="right"><template #default="{ row }"><el-button link type="primary" :loading="probingKey === `component:${row.id}`" @click="runComponentProbe(row)">检测</el-button><el-button link type="primary" @click="openComponent(row)">编辑</el-button><el-button link type="danger" @click="removeComponent(row)">删除</el-button></template></el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
      <div v-if="!loading && !activeRows.length" class="inventory-empty">当前筛选条件下没有{{ activeTabLabel }}记录。</div>
    </section>

    <el-dialog v-model="dialogs.domain" width="820px" class="inventory-editor-dialog" destroy-on-close @closed="domainFormRef?.resetFields()">
      <template #header><EditorDialogHeader :icon="Link" eyebrow="APPLICATION ENTRY" :title="domainForm.id ? '编辑应用域名' : '新增应用域名'" description="运行环境由所属应用自动继承，保存后立即执行 DNS、HTTP/TCP 和 TLS 检测。" /></template>
      <el-form ref="domainFormRef" :model="domainForm" :rules="domainRules" label-position="top" class="inventory-editor-form">
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>01</span><div><h4>访问入口</h4><p>完整入口由协议、域名、端口和路径组成。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="所属应用" prop="applicationId"><el-select v-model="domainForm.applicationId" filterable :disabled="Boolean(domainForm.id)" @change="validateDomainApplication"><el-option v-for="app in applications" :key="app.id" :label="applicationOptionLabel(app)" :value="app.id" :disabled="!app.environmentId" /></el-select><div v-if="domainForm.id" class="inventory-form-hint">所属应用创建后不可修改。</div></el-form-item>
          <el-form-item label="运行环境"><div class="inventory-readonly-field"><span>{{ inheritedEnvironmentName(domainForm.applicationId) }}</span><small>由应用自动继承</small></div></el-form-item>
          <el-form-item label="域名" prop="domain"><el-input v-model="domainForm.domain" maxlength="255" placeholder="api.example.com" /></el-form-item>
          <el-form-item label="访问路径" prop="path"><el-input v-model="domainForm.path" maxlength="255" placeholder="/" /></el-form-item>
          <el-form-item label="协议" prop="protocol"><el-segmented v-model="domainForm.protocol" :options="protocolOptions" @change="handleDomainProtocolChange" /></el-form-item>
          <el-form-item label="端口" prop="port"><el-input-number v-model="domainForm.port" :min="1" :max="65535" /></el-form-item>
        </div></section>
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>02</span><div><h4>证书与流量</h4><p>健康状态不在这里填写，检测结果会回写到列表。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="SSL 证书"><el-select v-model="domainForm.certificateId" :disabled="domainForm.protocol !== 'https'" clearable filterable placeholder="暂不关联"><el-option v-for="item in references.certificates || []" :key="item.id" :label="referenceLabel(item)" :value="item.id" /></el-select></el-form-item>
          <el-form-item label="DNS 服务商"><div class="inventory-readonly-field"><span>{{ domainForm.dnsProvider || '保存后自动识别' }}</span><small>根据权威 NS 记录探测</small></div></el-form-item>
          <el-form-item label="设为主入口"><el-switch v-model="domainForm.isPrimary" inline-prompt active-text="是" inactive-text="否" /></el-form-item>
          <el-form-item label="探测方式"><div class="inventory-readonly-field"><StatusTag status="checking" /><span>保存后自动检测</span></div></el-form-item>
          <el-form-item label="说明" class="el-form-item--full"><el-input v-model="domainForm.description" type="textarea" :rows="2" maxlength="500" show-word-limit /></el-form-item>
        </div></section>
      </el-form>
      <template #footer><div class="inventory-dialog-footer"><span><i>*</i> 为必填项；状态由平台自动探测</span><div><el-button @click="dialogs.domain = false">取消</el-button><el-button type="primary" :loading="saving.domain" @click="saveDomain">保存并检测</el-button></div></div></template>
    </el-dialog>

    <el-dialog v-model="dialogs.resource" width="840px" class="inventory-editor-dialog" destroy-on-close @closed="resourceFormRef?.resetFields()">
      <template #header><EditorDialogHeader :icon="Monitor" eyebrow="DEPLOYMENT RESOURCE" :title="resourceForm.id ? '编辑部署资源' : '新增部署资源'" description="主机 IP 来自资产管理，Kubernetes 状态来自关联集群，避免重复和错误录入。" /></template>
      <el-form ref="resourceFormRef" :model="resourceForm" :rules="resourceRules" label-position="top" class="inventory-editor-form">
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>01</span><div><h4>资源身份</h4><p>运行环境由应用自动继承，资源状态由平台自动检测。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="所属应用" prop="applicationId"><el-select v-model="resourceForm.applicationId" filterable :disabled="Boolean(resourceForm.id)"><el-option v-for="app in applications" :key="app.id" :label="applicationOptionLabel(app)" :value="app.id" :disabled="!app.environmentId" /></el-select><div v-if="resourceForm.id" class="inventory-form-hint">所属应用创建后不可修改。</div></el-form-item>
          <el-form-item label="运行环境"><div class="inventory-readonly-field"><span>{{ inheritedEnvironmentName(resourceForm.applicationId) }}</span><small>由应用自动继承</small></div></el-form-item>
          <el-form-item label="资源类型" prop="kind"><el-select v-model="resourceForm.kind" :disabled="Boolean(resourceForm.id)" @change="handleResourceKindChange"><el-option v-for="item in resourceKinds" :key="item.value" :label="item.label" :value="item.value" /></el-select><div v-if="resourceForm.id" class="inventory-form-hint">资源类型创建后不可修改。</div></el-form-item>
          <el-form-item label="资源名称" prop="name"><el-input v-model="resourceForm.name" maxlength="180" :placeholder="isKubernetesResource ? 'Kubernetes 对象名称' : '例如：order-api-prod'" /></el-form-item>
        </div></section>
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>02</span><div><h4>部署位置</h4><p>{{ resourceLocationHint }}</p></div></div><div class="inventory-form-grid">
          <el-form-item v-if="isHostResource" label="关联主机" prop="hostId"><el-select v-model="resourceForm.hostId" filterable placeholder="选择资产管理中的主机" @change="handleHostChange"><el-option v-for="item in references.hosts || []" :key="item.id" :label="referenceLabel(item)" :value="item.id" /></el-select></el-form-item>
          <el-form-item v-if="isHostResource" label="主机 IP"><div class="inventory-readonly-field"><span>{{ selectedHost?.address || '选择主机后自动带出' }}</span><small>来自资产管理，不可手工覆盖</small></div></el-form-item>
          <el-form-item v-if="isKubernetesResource" label="关联集群" prop="clusterId"><el-select v-model="resourceForm.clusterId" filterable placeholder="选择有权限的可用集群"><el-option v-for="item in references.clusters || []" :key="item.id" :label="`${referenceLabel(item)}${item.status === '1' ? '' : ' · 不可用'}`" :value="item.id" :disabled="item.status !== '1'" /></el-select></el-form-item>
          <el-form-item v-if="isKubernetesResource" label="命名空间" prop="namespace"><el-input v-model="resourceForm.namespace" maxlength="120" placeholder="例如：production" /></el-form-item>
          <el-form-item v-if="isOtherResource" label="访问地址" prop="address"><el-input v-model="resourceForm.address" maxlength="500" placeholder="IP、域名或 API 地址" /></el-form-item>
          <el-form-item v-if="isOtherResource" label="外部标识"><el-input v-model="resourceForm.externalId" maxlength="500" placeholder="云资源 ID 等，可选" /></el-form-item>
          <el-form-item v-if="!isKubernetesResource" label="业务端口"><el-input-number v-model="resourceForm.port" :min="0" :max="65535" /><div class="inventory-form-hint">不填端口时仅依据主机在线状态判断。</div></el-form-item>
          <el-form-item label="说明" class="el-form-item--full"><el-input v-model="resourceForm.description" type="textarea" :rows="2" maxlength="500" show-word-limit /></el-form-item>
        </div></section>
      </el-form>
      <template #footer><div class="inventory-dialog-footer"><span><i>*</i> 为必填项；Host 地址由平台资产自动回填</span><div><el-button @click="dialogs.resource = false">取消</el-button><el-button type="primary" :loading="saving.resource" @click="saveResource">保存并检测</el-button></div></div></template>
    </el-dialog>

    <el-dialog v-model="dialogs.component" width="840px" class="inventory-editor-dialog" destroy-on-close @closed="componentFormRef?.resetFields()">
      <template #header><EditorDialogHeader :icon="Connection" eyebrow="RUNTIME COMPONENT" :title="componentForm.id ? '编辑数据库或中间件' : '新增数据库或中间件'" description="连接凭据从统一凭据中心授权使用，健康状态通过 TCP/TLS 自动检测。" /></template>
      <el-form ref="componentFormRef" :model="componentForm" :rules="componentRules" label-position="top" class="inventory-editor-form">
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>01</span><div><h4>组件归属</h4><p>运行环境由所属应用自动继承。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="所属应用" prop="applicationId"><el-select v-model="componentForm.applicationId" filterable :disabled="Boolean(componentForm.id)"><el-option v-for="app in applications" :key="app.id" :label="applicationOptionLabel(app)" :value="app.id" :disabled="!app.environmentId" /></el-select><div v-if="componentForm.id" class="inventory-form-hint">所属应用创建后不可修改。</div></el-form-item>
          <el-form-item label="运行环境"><div class="inventory-readonly-field"><span>{{ inheritedEnvironmentName(componentForm.applicationId) }}</span><small>由应用自动继承</small></div></el-form-item>
          <el-form-item label="组件分类" prop="category"><el-select v-model="componentForm.category"><el-option v-for="item in categories" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
          <el-form-item label="组件类型" prop="type"><el-select v-model="componentForm.type" filterable allow-create default-first-option><el-option v-for="item in componentTypes" :key="item" :label="item" :value="item" /></el-select></el-form-item>
          <el-form-item label="组件名称" prop="name"><el-input v-model="componentForm.name" maxlength="150" placeholder="例如：订单库主实例" /></el-form-item>
          <el-form-item label="版本"><el-input v-model="componentForm.version" maxlength="80" placeholder="例如：8.0.36" /></el-form-item>
        </div></section>
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>02</span><div><h4>连接与安全</h4><p>账号密码不写入组件记录，只保存统一凭据的授权引用。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="连接地址" prop="address"><el-input v-model="componentForm.address" maxlength="500" placeholder="主机名、IP 或服务地址" /></el-form-item>
          <el-form-item label="端口" prop="port"><el-input-number v-model="componentForm.port" :min="1" :max="65535" /></el-form-item>
          <el-form-item label="数据库 / 实例名"><el-input v-model="componentForm.databaseName" maxlength="120" /></el-form-item>
          <el-form-item label="托管凭据"><el-select v-model="componentForm.credentialId" clearable filterable placeholder="从统一凭据中心选择"><el-option v-for="item in credentials" :key="item.id" :label="`${item.name} (${item.username || '无用户名'})`" :value="item.id" /></el-select></el-form-item>
          <el-form-item label="启用 TLS"><el-switch v-model="componentForm.tlsEnabled" inline-prompt active-text="是" inactive-text="否" /></el-form-item>
          <el-form-item label="说明" class="el-form-item--full"><el-input v-model="componentForm.description" type="textarea" :rows="2" maxlength="500" show-word-limit /></el-form-item>
        </div></section>
      </el-form>
      <template #footer><div class="inventory-dialog-footer"><span><i>*</i> 为必填项；状态由连接探测生成</span><div><el-button @click="dialogs.component = false">取消</el-button><el-button type="primary" :loading="saving.component" @click="saveComponent">保存并检测</el-button></div></div></template>
    </el-dialog>

    <el-dialog v-model="dialogs.sync" width="920px" class="inventory-editor-dialog" destroy-on-close>
      <template #header><EditorDialogHeader :icon="Download" eyebrow="KUBERNETES DISCOVERY" title="扫描并导入 Kubernetes 资源" description="从有权限的集群发现工作负载、Service、Ingress 和域名，并自动继承目标应用的环境。" /></template>
      <div class="inventory-editor-form">
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>01</span><div><h4>选择扫描范围</h4><p>选择集群后自动加载命名空间和资源，不需要手工填写。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="集群" required><el-select v-model="syncForm.clusterId" filterable @change="handleSyncClusterChange"><el-option v-for="item in references.clusters || []" :key="item.id" :label="`${referenceLabel(item)}${item.status === '1' ? '' : ' · 不可用'}`" :value="item.id" :disabled="item.status !== '1'" /></el-select></el-form-item>
          <el-form-item label="命名空间"><el-select v-model="syncForm.namespace" filterable :loading="syncing.namespaces" placeholder="选择命名空间" @change="scanKubernetes"><el-option label="全部命名空间" value="__all__" /><el-option v-for="item in syncNamespaces" :key="item.name" :label="`${item.name}${item.status === 'Active' ? '' : ` · ${item.status}`}`" :value="item.name" /></el-select></el-form-item>
          <el-form-item label="高级标签过滤（可选）" class="el-form-item--full"><el-input v-model="syncForm.selector" clearable placeholder="无需填写；仅在资源很多时使用，如 app.kubernetes.io/name=order-api" @keyup.enter="scanKubernetes" /><div class="inventory-form-help">留空会自动发现 Deployment、StatefulSet、DaemonSet、Service、Ingress 及域名。</div></el-form-item>
        </div><div class="inventory-form-action"><el-button type="primary" :loading="syncing.preview" @click="scanKubernetes">重新扫描</el-button></div></section>
        <section v-if="syncCandidates.length || syncScanned" class="inventory-form-section"><div class="inventory-form-section__heading"><span>02</span><div><h4>选择资源组</h4><p>资源按命名空间和应用标签自动归组，可展开查看具体对象。</p></div></div>
          <el-table v-if="syncCandidates.length" :data="syncCandidates" row-key="key" highlight-current-row @current-change="selectCandidate">
            <el-table-column type="expand"><template #default="{ row }"><div class="inventory-discovery-resources"><div v-for="resource in row.resources" :key="`${resource.kind}:${resource.namespace}:${resource.name}`" class="inventory-discovery-resource"><el-tag size="small" effect="plain">{{ resource.kind }}</el-tag><strong>{{ resource.name }}</strong><span>{{ resource.address || resource.namespace }}</span><StatusTag :status="resource.status" /></div></div></template></el-table-column>
            <el-table-column width="48"><template #default="{ row }"><el-radio v-model="syncForm.candidateKey" :value="row.key"><span /></el-radio></template></el-table-column>
            <el-table-column prop="name" label="发现的应用" min-width="160" />
            <el-table-column prop="namespace" label="命名空间" min-width="130" />
            <el-table-column label="资源构成" min-width="230"><template #default="{ row }"><div class="inventory-resource-kinds"><el-tag v-for="item in resourceKindSummary(row)" :key="item.kind" size="small" effect="plain">{{ item.kind }} {{ item.count }}</el-tag></div></template></el-table-column>
            <el-table-column label="域名" min-width="190"><template #default="{ row }">{{ row.domains?.join(', ') || '-' }}</template></el-table-column>
          </el-table>
          <el-empty v-else description="当前范围没有发现可导入的 Kubernetes 资源" :image-size="72" />
          <div class="inventory-form-grid inventory-sync-target"><el-form-item label="导入应用" required><el-select v-model="syncForm.applicationId" filterable><el-option v-for="app in applications" :key="app.id" :label="applicationOptionLabel(app)" :value="app.id" :disabled="!app.environmentId" /></el-select></el-form-item><el-form-item label="继承环境"><div class="inventory-readonly-field"><span>{{ inheritedEnvironmentName(syncForm.applicationId) }}</span><small>无需再次选择</small></div></el-form-item></div>
        </section>
      </div>
      <template #footer><div class="inventory-dialog-footer"><span>导入后会持续使用平台健康探测</span><div><el-button @click="dialogs.sync = false">取消</el-button><el-button type="primary" :disabled="!syncForm.candidateKey || !syncForm.applicationId" :loading="syncing.import" @click="runImport">导入选中资源</el-button></div></div></template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Coin, Connection, Download, Link, Monitor, Plus, Refresh, Search } from '@element-plus/icons-vue'
import {
  createComponent, createDomain, createResource, deleteComponent, deleteDomain, deleteResource,
  getReferences, importKubernetes, listApplications, listComponents, listCredentials, listDomains, listKubernetesNamespaces,
  listEnvironments, listResources, previewKubernetes, probeComponent, probeDomain, probeResource,
  updateComponent, updateDomain, updateResource,
  type Application, type Component, type Domain, type Environment, type Resource,
} from '../api'
import { validateDomainName, validateForm } from '../form-utils'
import EditorDialogHeader from './EditorDialogHeader.vue'
import PageHeader from './PageHeader.vue'
import StatusTag from './StatusTag.vue'

const route = useRoute()
const loading = ref(false)
const activeTab = ref('domains')
const applications = ref<Application[]>([])
const environments = ref<Environment[]>([])
const references = ref<Record<string, any[]>>({})
const credentials = ref<any[]>([])
const domains = ref<Domain[]>([])
const resources = ref<Resource[]>([])
const components = ref<Component[]>([])
const dialogs = reactive({ domain: false, resource: false, component: false, sync: false })
const saving = reactive({ domain: false, resource: false, component: false })
const probingKey = ref('')
const domainFormRef = ref<FormInstance>()
const resourceFormRef = ref<FormInstance>()
const componentFormRef = ref<FormInstance>()
const filters = reactive({ appId: route.query.app_id ? Number(route.query.app_id) : undefined as number | undefined, environmentId: undefined as number | undefined, keyword: '', status: '' })
const domainForm = reactive<any>({})
const resourceForm = reactive<any>({})
const componentForm = reactive<any>({})
const syncForm = reactive<any>({ clusterId: undefined as number | undefined, namespace: '__all__', selector: '', candidateKey: '', applicationId: filters.appId || undefined })
const syncCandidates = ref<any[]>([])
const syncNamespaces = ref<Array<{ name: string; status: string }>>([])
const syncScanned = ref(false)
const syncing = reactive({ namespaces: false, preview: false, import: false })
const resourceKinds = [{ label: '物理主机', value: 'Host' }, { label: '虚拟机', value: 'VirtualMachine' }, { label: 'K8s Deployment', value: 'Deployment' }, { label: 'K8s StatefulSet', value: 'StatefulSet' }, { label: 'K8s DaemonSet', value: 'DaemonSet' }, { label: 'K8s Service', value: 'Service' }, { label: 'K8s Ingress', value: 'Ingress' }, { label: 'K8s Container', value: 'Container' }, { label: '其他资源', value: 'Other' }]
const kubernetesKinds = new Set(['Deployment', 'StatefulSet', 'DaemonSet', 'Service', 'Ingress', 'Container'])
const categories = [{ label: '数据库', value: 'database' }, { label: '中间件', value: 'middleware' }, { label: '缓存', value: 'cache' }, { label: '消息队列', value: 'queue' }, { label: '搜索', value: 'search' }, { label: '对象存储', value: 'storage' }, { label: '外部服务', value: 'external' }]
const componentTypes = ['MySQL', 'PostgreSQL', 'MongoDB', 'Redis', 'Kafka', 'RabbitMQ', 'RocketMQ', 'Elasticsearch', 'OpenSearch', 'MinIO']
const protocolOptions = [{ label: 'HTTPS', value: 'https' }, { label: 'HTTP', value: 'http' }, { label: 'TCP', value: 'tcp' }]
const isKubernetesResource = computed(() => kubernetesKinds.has(resourceForm.kind))
const isHostResource = computed(() => ['Host', 'VirtualMachine'].includes(resourceForm.kind))
const isOtherResource = computed(() => resourceForm.kind === 'Other')
const selectedHost = computed(() => (references.value.hosts || []).find(item => item.id === resourceForm.hostId))
const resourceLocationHint = computed(() => isHostResource.value ? '选择资产主机后自动使用其 IP，用户无需也不能重复填写。' : isKubernetesResource.value ? '选择集群和命名空间，平台按对象名称读取实时状态。' : '只有其他资源允许手工登记访问地址。')

const domainRules: FormRules = {
  applicationId: [{ required: true, message: '请选择所属应用', trigger: 'change' }, { validator: validateApplicationEnvironment, trigger: 'change' }],
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }, { validator: validateDomainName, trigger: 'blur' }],
  path: [{ required: true, message: '请输入访问路径', trigger: 'blur' }, { pattern: /^\//, message: '访问路径必须以 / 开头', trigger: 'blur' }],
  protocol: [{ required: true, message: '请选择访问协议', trigger: 'change' }],
  port: [{ required: true, message: '请输入端口', trigger: 'change' }],
}
const resourceRules: FormRules = {
  applicationId: [{ required: true, message: '请选择所属应用', trigger: 'change' }, { validator: validateApplicationEnvironment, trigger: 'change' }],
  kind: [{ required: true, message: '请选择资源类型', trigger: 'change' }],
  name: [{ required: true, message: '请输入资源名称', trigger: 'blur' }],
  hostId: [{ validator: (_rule, value, callback) => isHostResource.value && !value ? callback(new Error('请选择关联主机')) : callback(), trigger: 'change' }],
  clusterId: [{ validator: (_rule, value, callback) => isKubernetesResource.value && !value ? callback(new Error('请选择 Kubernetes 集群')) : callback(), trigger: 'change' }],
  namespace: [{ validator: (_rule, value, callback) => isKubernetesResource.value && !String(value || '').trim() ? callback(new Error('请输入命名空间')) : callback(), trigger: 'blur' }],
  address: [{ validator: (_rule, value, callback) => isOtherResource.value && !String(value || '').trim() && !String(resourceForm.externalId || '').trim() ? callback(new Error('请输入访问地址或外部标识')) : callback(), trigger: 'blur' }],
}
const componentRules: FormRules = {
  applicationId: [{ required: true, message: '请选择所属应用', trigger: 'change' }, { validator: validateApplicationEnvironment, trigger: 'change' }],
  category: [{ required: true, message: '请选择组件分类', trigger: 'change' }],
  type: [{ required: true, message: '请选择或输入组件类型', trigger: 'change' }],
  name: [{ required: true, message: '请输入组件名称', trigger: 'blur' }],
  address: [{ required: true, message: '请输入连接地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入连接端口', trigger: 'change' }],
}

const activeTabLabel = computed(() => ({ domains: '域名', resources: '部署资源', components: '组件' } as Record<string, string>)[activeTab.value])
const activeRows = computed(() => activeTab.value === 'domains' ? domains.value : activeTab.value === 'resources' ? resources.value : components.value)
const unhealthyCount = computed(() => [...domains.value, ...resources.value, ...components.value].filter(item => ['unhealthy', 'error', 'down'].includes(item.status)).length)
const appName = (id: number) => applications.value.find(item => item.id === id)?.name || (id ? `应用 #${id}` : '-')
const applicationOptionLabel = (app: Application) => `${app.name} · ${app.environmentName || (app.environmentId ? envName(app.environmentId) : '未关联环境')}`
const envName = (id: number) => environments.value.find(item => item.id === id)?.name || (id ? `环境 #${id}` : '未关联')
const inheritedEnvironmentName = (appId: number) => {
  const app = applications.value.find(item => item.id === appId)
  return app?.environmentName || envName(app?.environmentId || 0)
}
const endpoint = (address?: string, port?: number) => address ? `${address}${port && port > 0 ? `:${port}` : ''}` : ''
const domainEndpoint = (row: Domain) => `${row.protocol}://${row.domain}${row.port && !((row.protocol === 'https' && row.port === 443) || (row.protocol === 'http' && row.port === 80)) ? `:${row.port}` : ''}${row.path || '/'}`
const sourceLabel = (value?: string) => value === 'kubernetes' ? 'Kubernetes' : value === 'discovered' ? '自动发现' : '手工登记'
const categoryLabel = (value: string) => categories.find(item => item.value === value)?.label || value
const resourceKindLabel = (value: string) => resourceKinds.find(item => item.value === value)?.label || value
const resourceLocation = (row: Resource) => row.namespace ? `${clusterName(row.clusterId)} / ${row.namespace}` : endpoint(row.address, row.port) || '未获取位置'
const clusterName = (id?: number) => (references.value.clusters || []).find(item => item.id === id)?.name || (id ? `集群 #${id}` : 'Kubernetes')
const referenceLabel = (item: any) => item.detail ? `${item.name} (${item.detail})` : item.name
const resourceKindSummary = (row: any) => Object.entries((row.resources || []).reduce((result: Record<string, number>, item: any) => { result[item.kind] = (result[item.kind] || 0) + 1; return result }, {})).map(([kind, count]) => ({ kind, count }))
const certificateHint = (value?: string) => {
  if (!value) return ''
  const timestamp = new Date(value).getTime()
  if (Number.isNaN(timestamp)) return '到期时间未知'
  const days = Math.ceil((timestamp - Date.now()) / 86400000)
  return days < 0 ? `已过期 ${Math.abs(days)} 天` : `${days} 天后到期`
}
const formatDateTime = (value?: string) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '尚未检测'
const probeMeta = (row: any) => `${formatDateTime(row.lastCheckedAt)}${row.responseTimeMs > 0 ? ` · ${row.responseTimeMs} ms` : ''}${row.httpStatusCode > 0 ? ` · HTTP ${row.httpStatusCode}` : ''}`
const optionalId = (value?: number) => value && value > 0 ? value : undefined

function validateApplicationEnvironment(_rule: unknown, value: number, callback: (error?: Error) => void) {
  const app = applications.value.find(item => item.id === value)
  app?.environmentId ? callback() : callback(new Error('所选应用尚未关联有效运行环境'))
}
const validateDomainApplication = () => domainFormRef.value?.validateField('applicationId')
const loadOptions = async () => {
  const [apps, envs, refs, creds] = await Promise.all([listApplications({ page: 1, page_size: 200 }), listEnvironments(), getReferences(), listCredentials({ page: 1, page_size: 200 }).catch(() => ({ list: [] } as any))])
  applications.value = apps?.list || []
  environments.value = envs || []
  references.value = refs || {}
  credentials.value = creds?.list || []
}
const loadAll = async () => {
  loading.value = true
  const params = { app_id: filters.appId || undefined, environment_id: filters.environmentId || undefined, keyword: filters.keyword || undefined, status: filters.status || undefined, page: 1, page_size: 200 }
  try {
    const [domainData, resourceData, componentData] = await Promise.all([listDomains(params), listResources(params), listComponents(params)])
    domains.value = domainData?.list || []
    resources.value = resourceData?.list || []
    components.value = componentData?.list || []
  } finally { loading.value = false }
}
const handleAppFilter = async () => { filters.environmentId = filters.appId ? applications.value.find(item => item.id === filters.appId)?.environmentId : undefined; await loadAll() }
const resetFilters = () => { filters.appId = undefined; filters.environmentId = undefined; filters.keyword = ''; filters.status = ''; loadAll() }
const preferredAppId = () => filters.appId || applications.value.find(item => item.environmentId)?.id || 0
const openCreateForTab = () => activeTab.value === 'domains' ? openDomain() : activeTab.value === 'resources' ? openResource() : openComponent()
const openDomain = (row?: Domain) => { const value = { id: 0, applicationId: preferredAppId() || undefined, domain: '', protocol: 'https', port: 443, path: '/', dnsProvider: '', certificateId: undefined as number | undefined, isPrimary: false, description: '', ...(row || {}) }; value.certificateId = optionalId(value.certificateId); Object.assign(domainForm, value); dialogs.domain = true }
const openResource = (row?: Resource) => { const value = { id: 0, applicationId: preferredAppId() || undefined, kind: 'Host', name: '', address: '', port: 0, hostId: undefined as number | undefined, clusterId: undefined as number | undefined, namespace: '', externalId: '', description: '', ...(row || {}) }; value.hostId = optionalId(value.hostId); value.clusterId = optionalId(value.clusterId); Object.assign(resourceForm, value); dialogs.resource = true; handleHostChange(resourceForm.hostId) }
const openComponent = (row?: Component) => { const value = { id: 0, applicationId: preferredAppId() || undefined, category: 'database', type: 'MySQL', name: '', address: '', port: 3306, databaseName: '', version: '', credentialId: undefined as number | undefined, tlsEnabled: false, description: '', ...(row || {}) }; value.credentialId = optionalId(value.credentialId); Object.assign(componentForm, value); dialogs.component = true }
const handleDomainProtocolChange = (value: string) => { domainForm.port = value === 'https' ? 443 : value === 'http' ? 80 : domainForm.port || 80; if (value !== 'https') domainForm.certificateId = 0 }
const handleResourceKindChange = () => { resourceForm.hostId = undefined; resourceForm.clusterId = undefined; resourceForm.namespace = ''; resourceForm.address = ''; resourceForm.externalId = '' }
const handleHostChange = (id?: number) => { const host = (references.value.hosts || []).find(item => item.id === id); resourceForm.address = host?.address || '' }
const saveDomain = async () => {
  if (!await validateForm(domainFormRef.value)) return
  saving.domain = true
  try {
    const payload = { applicationId: domainForm.applicationId, domain: domainForm.domain, protocol: domainForm.protocol, port: domainForm.port, path: domainForm.path, certificateId: domainForm.certificateId || 0, isPrimary: domainForm.isPrimary, description: domainForm.description }
    domainForm.id ? await updateDomain(domainForm.id, payload) : await createDomain(payload)
    dialogs.domain = false; ElMessage.success('域名已保存并完成检测'); await loadAll()
  } finally { saving.domain = false }
}
const saveResource = async () => {
  if (!await validateForm(resourceFormRef.value)) return
  saving.resource = true
  try {
    const payload = { applicationId: resourceForm.applicationId, kind: resourceForm.kind, name: resourceForm.name, address: isOtherResource.value ? resourceForm.address : '', port: resourceForm.port || 0, hostId: isHostResource.value ? resourceForm.hostId : 0, clusterId: isKubernetesResource.value ? resourceForm.clusterId : 0, namespace: isKubernetesResource.value ? resourceForm.namespace : '', externalId: isOtherResource.value ? resourceForm.externalId : '', description: resourceForm.description }
    resourceForm.id ? await updateResource(resourceForm.id, payload) : await createResource(payload)
    dialogs.resource = false; ElMessage.success('资源已保存并完成检测'); await loadAll()
  } finally { saving.resource = false }
}
const saveComponent = async () => {
  if (!await validateForm(componentFormRef.value)) return
  saving.component = true
  try {
    const payload = { applicationId: componentForm.applicationId, category: componentForm.category, type: componentForm.type, name: componentForm.name, address: componentForm.address, port: componentForm.port, databaseName: componentForm.databaseName, version: componentForm.version, credentialId: componentForm.credentialId || 0, tlsEnabled: componentForm.tlsEnabled, description: componentForm.description }
    componentForm.id ? await updateComponent(componentForm.id, payload) : await createComponent(payload)
    dialogs.component = false; ElMessage.success('组件已保存并完成检测'); await loadAll()
  } finally { saving.component = false }
}
const runProbe = async (key: string, action: () => Promise<unknown>) => { probingKey.value = key; try { await action(); ElMessage.success('检测完成'); await loadAll() } finally { probingKey.value = '' } }
const runDomainProbe = (row: Domain) => runProbe(`domain:${row.id}`, () => probeDomain(row.id))
const runResourceProbe = (row: Resource) => runProbe(`resource:${row.id}`, () => probeResource(row.id))
const runComponentProbe = (row: Component) => runProbe(`component:${row.id}`, () => probeComponent(row.id))
const confirmDelete = (name: string) => ElMessageBox.confirm(`确认删除“${name}”？`, '删除确认', { type: 'warning' })
const removeDomain = async (row: Domain) => { await confirmDelete(row.domain); await deleteDomain(row.id); await loadAll() }
const removeResource = async (row: Resource) => { await confirmDelete(row.name); await deleteResource(row.id); await loadAll() }
const removeComponent = async (row: Component) => { await confirmDelete(row.name); await deleteComponent(row.id); await loadAll() }
const openSync = async () => { dialogs.sync = true; syncCandidates.value = []; syncNamespaces.value = []; syncScanned.value = false; syncForm.candidateKey = ''; syncForm.namespace = '__all__'; syncForm.applicationId = filters.appId || (applications.value.find(item => item.id === syncForm.applicationId && item.environmentId)?.id) || applications.value.find(item => item.environmentId)?.id || undefined; const availableCluster = (references.value.clusters || []).find(item => item.status === '1'); if (!(references.value.clusters || []).some(item => item.id === syncForm.clusterId && item.status === '1')) syncForm.clusterId = availableCluster?.id; if (syncForm.clusterId) await handleSyncClusterChange() }
const handleSyncClusterChange = async () => { syncCandidates.value = []; syncNamespaces.value = []; syncScanned.value = false; syncForm.candidateKey = ''; syncForm.namespace = '__all__'; if (!syncForm.clusterId) return; syncing.namespaces = true; try { syncNamespaces.value = await listKubernetesNamespaces(syncForm.clusterId) || [] } finally { syncing.namespaces = false } await scanKubernetes() }
const scanKubernetes = async () => { if (!syncForm.clusterId || syncing.namespaces) return; syncing.preview = true; syncForm.candidateKey = ''; try { const data = await previewKubernetes({ clusterId: syncForm.clusterId, namespace: syncForm.namespace === '__all__' ? '' : syncForm.namespace, selector: syncForm.selector }); syncCandidates.value = data?.items || []; syncScanned.value = true } finally { syncing.preview = false } }
const selectCandidate = (row: any) => { if (row) syncForm.candidateKey = row.key }
const runImport = async () => { if (!syncForm.applicationId || !applications.value.find(item => item.id === syncForm.applicationId)?.environmentId) { ElMessage.warning('请选择已关联运行环境的应用'); return } syncing.import = true; try { const data = await importKubernetes({ clusterId: syncForm.clusterId, namespace: syncForm.namespace === '__all__' ? '' : syncForm.namespace, selector: syncForm.selector, candidateKey: syncForm.candidateKey, applicationId: syncForm.applicationId }); ElMessage.success(`已同步 ${data.resourceCount} 个资源和 ${data.domainCount} 个域名`); dialogs.sync = false; await loadAll() } finally { syncing.import = false } }

onMounted(async () => { await loadOptions(); if (filters.appId) filters.environmentId = applications.value.find(item => item.id === filters.appId)?.environmentId; await loadAll(); if (route.query.sync === '1') openSync() })
</script>
