<template>
  <div class="inventory-page" v-loading="loading">
    <div class="inventory-detail-header">
      <div class="inventory-detail-header__main">
        <div class="inventory-detail-header__title">
          <el-button text :icon="ArrowLeft" aria-label="返回应用台账" @click="router.push('/app-inventory/apps')" />
          <h2>{{ detail?.name || '应用详情' }}</h2>
          <el-tag v-if="detail" effect="plain">{{ detail.code }}</el-tag>
          <StatusTag v-if="detail" :status="detail.healthStatus" />
        </div>
        <div class="inventory-detail-header__meta">
          <span>负责人：{{ detail?.ownerName || detail?.ownerUsername || '未设置' }}</span>
          <span>部门：{{ detail?.departmentName || detail?.team || '未设置' }}</span>
          <span>环境：{{ environment?.name || '未关联' }}</span>
          <span>重要级别：{{ criticalityLabel(detail?.criticality) }}</span>
        </div>
      </div>
      <div class="inventory-page__actions">
        <el-button :icon="Share" @click="router.push(`/app-inventory/topology?app_id=${appId}`)">查看拓扑</el-button>
        <el-button :icon="Aim" :loading="probingKey === `application:${appId}`" @click="runApplicationProbe">立即检测</el-button>
        <el-button :icon="Refresh" @click="loadDetail">刷新</el-button>
      </div>
    </div>

    <section v-if="detail" class="inventory-panel">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="应用概况" name="overview">
          <div class="inventory-health-summary" :class="`is-${healthTone(detail.healthStatus)}`">
            <div><span>自动健康检测</span><strong>{{ detail.healthMessage || '暂无可探测资产' }}</strong></div>
            <div class="inventory-health-summary__meta"><StatusTag :status="detail.healthStatus" /><span>{{ formatDateTime(detail.healthCheckedAt) }}</span></div>
          </div>
          <div class="inventory-stat-grid inventory-stat-grid--detail">
            <div class="inventory-stat"><div class="inventory-stat__label">所属环境</div><div class="inventory-stat__value">{{ environment ? 1 : 0 }}</div></div>
            <div class="inventory-stat"><div class="inventory-stat__label">域名</div><div class="inventory-stat__value">{{ detail.domains?.length || 0 }}</div></div>
            <div class="inventory-stat"><div class="inventory-stat__label">部署资源</div><div class="inventory-stat__value">{{ detail.resources?.length || 0 }}</div></div>
            <div class="inventory-stat"><div class="inventory-stat__label">数据库与中间件</div><div class="inventory-stat__value">{{ detail.components?.length || 0 }}</div></div>
            <div class="inventory-stat"><div class="inventory-stat__label">调用依赖</div><div class="inventory-stat__value">{{ detail.dependencies?.length || 0 }}</div></div>
          </div>
          <div class="inventory-two-column">
            <div>
              <div class="inventory-panel__heading"><h3>基本信息</h3></div>
              <el-descriptions :column="2" border size="small">
                <el-descriptions-item label="代码仓库"><a v-if="detail.repositoryUrl" :href="detail.repositoryUrl" target="_blank" rel="noreferrer">{{ detail.repositoryUrl }}</a><span v-else>-</span></el-descriptions-item>
                <el-descriptions-item label="技术栈">{{ detail.language || '-' }}</el-descriptions-item>
                <el-descriptions-item label="文档地址"><a v-if="detail.documentationUrl" :href="detail.documentationUrl" target="_blank" rel="noreferrer">{{ detail.documentationUrl }}</a><span v-else>-</span></el-descriptions-item>
                <el-descriptions-item label="治理状态"><StatusTag :status="detail.status" /></el-descriptions-item>
                <el-descriptions-item label="业务标签" :span="2"><div v-if="parseTagList(detail.tags).length" class="inventory-tag-list"><el-tag v-for="tag in parseTagList(detail.tags)" :key="tag" size="small" effect="plain">{{ tag }}</el-tag></div><span v-else>-</span></el-descriptions-item>
                <el-descriptions-item label="说明" :span="2">{{ detail.description || '暂无说明' }}</el-descriptions-item>
              </el-descriptions>
            </div>
            <div>
              <div class="inventory-panel__heading"><h3>运行环境</h3><el-button link :icon="SetUp" @click="router.push(`/app-inventory/environments?app_id=${appId}`)">环境管理</el-button></div>
              <div v-if="environment" class="inventory-environment-card">
                <div class="inventory-environment-card__head"><div><strong>{{ environment.name }}</strong><span>{{ environment.code }}</span></div><StatusTag :status="environment.status" /></div>
                <dl><div><dt>类型</dt><dd>{{ lifecycleLabel(environment.kind) }}</dd></div><div><dt>区域</dt><dd>{{ environment.region || '未设置' }}</dd></div><div><dt>共享应用</dt><dd>{{ environment.applicationCount || 0 }} 个</dd></div></dl>
                <p>{{ environment.description || '暂无环境说明' }}</p>
              </div>
              <el-empty v-else description="应用尚未关联运行环境" :image-size="60" />
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane name="domains">
          <template #label>域名与证书 <el-badge :value="detail.domains?.length || 0" :max="99" /></template>
          <div class="inventory-panel__heading"><h3>应用入口</h3><el-button type="primary" :icon="Plus" @click="openDomain">新增域名</el-button></div>
          <el-table :data="detail.domains" stripe>
            <el-table-column label="访问入口" min-width="250"><template #default="{ row }"><strong>{{ domainEndpoint(row) }}</strong><div class="inventory-table-cell__sub">{{ row.source === 'kubernetes' ? 'Kubernetes 自动发现' : '手工登记' }} · {{ environment?.name || '未关联环境' }}</div></template></el-table-column>
            <el-table-column label="自动探测" min-width="230"><template #default="{ row }"><StatusTag :status="row.status" /><div class="inventory-table-cell__sub" :title="row.probeMessage">{{ row.probeMessage || '等待检测' }}</div><div class="inventory-table-cell__sub">{{ probeMeta(row) }}</div></template></el-table-column>
            <el-table-column label="DNS 与证书" min-width="190"><template #default="{ row }">{{ row.certificateName || row.tlsIssuer || '未关联证书' }}<div v-if="row.certificateExpiry || row.tlsExpiresAt" class="inventory-table-cell__sub">到期 {{ formatDate(row.certificateExpiry || row.tlsExpiresAt) }}</div><div class="inventory-table-cell__sub">{{ row.dnsProvider || 'DNS 服务商待识别' }}</div></template></el-table-column>
            <el-table-column label="主入口" width="80"><template #default="{ row }"><el-icon v-if="row.isPrimary" color="#16803c"><CircleCheck /></el-icon><span v-else>-</span></template></el-table-column>
            <el-table-column label="操作" width="190" fixed="right"><template #default="{ row }"><el-button link type="primary" :loading="probingKey === `domain:${row.id}`" @click="runDomainProbe(row)">检测</el-button><el-button link type="primary" @click="openDomain(row)">编辑</el-button><el-button link type="danger" @click="removeDomain(row)">删除</el-button></template></el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane name="resources">
          <template #label>部署资源 <el-badge :value="detail.resources?.length || 0" :max="99" /></template>
          <div class="inventory-panel__heading"><h3>主机与 Kubernetes 资源</h3><div><el-button :icon="Download" @click="router.push(`/app-inventory/resources?sync=1&app_id=${appId}`)">K8s 同步</el-button><el-button type="primary" :icon="Plus" @click="openResource">新增资源</el-button></div></div>
          <el-table :data="detail.resources" stripe>
            <el-table-column label="资源" min-width="210"><template #default="{ row }"><strong>{{ row.name }}</strong><div class="inventory-table-cell__sub">{{ resourceKindLabel(row.kind) }} · {{ sourceLabel(row.source) }}</div></template></el-table-column>
            <el-table-column label="部署位置" min-width="220"><template #default="{ row }">{{ resourceLocation(row) }}<div class="inventory-table-cell__sub">{{ environment?.name || '未关联环境' }}</div></template></el-table-column>
            <el-table-column label="自动探测" min-width="240"><template #default="{ row }"><StatusTag :status="row.status" /><div class="inventory-table-cell__sub" :title="row.healthMessage">{{ row.healthMessage || '等待检测' }}</div><div class="inventory-table-cell__sub">{{ probeMeta(row) }}</div></template></el-table-column>
            <el-table-column label="操作" width="190" fixed="right"><template #default="{ row }"><el-button link type="primary" :loading="probingKey === `resource:${row.id}`" @click="runResourceProbe(row)">检测</el-button><el-button link type="primary" @click="openResource(row)">编辑</el-button><el-button link type="danger" @click="removeResource(row)">删除</el-button></template></el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane name="components">
          <template #label>数据库与中间件 <el-badge :value="detail.components?.length || 0" :max="99" /></template>
          <div class="inventory-panel__heading"><h3>运行依赖组件</h3><el-button type="primary" :icon="Plus" @click="openComponent">新增组件</el-button></div>
          <el-table :data="detail.components" stripe>
            <el-table-column label="组件" min-width="210"><template #default="{ row }"><strong>{{ row.name }}</strong><div class="inventory-table-cell__sub">{{ categoryLabel(row.category) }} · {{ row.type }}{{ row.version ? ` ${row.version}` : '' }}</div></template></el-table-column>
            <el-table-column label="连接地址" min-width="210"><template #default="{ row }">{{ endpoint(row.address, row.port) || '-' }}<div v-if="row.databaseName" class="inventory-table-cell__sub">数据库：{{ row.databaseName }}</div></template></el-table-column>
            <el-table-column label="凭据" width="100"><template #default="{ row }"><el-tag v-if="row.credentialId" size="small" type="success" effect="plain">已托管</el-tag><span v-else>-</span></template></el-table-column>
            <el-table-column label="自动探测" min-width="230"><template #default="{ row }"><StatusTag :status="row.status" /><div class="inventory-table-cell__sub" :title="row.healthMessage">{{ row.healthMessage || '等待检测' }}</div><div class="inventory-table-cell__sub">{{ probeMeta(row) }}</div></template></el-table-column>
            <el-table-column label="操作" width="190" fixed="right"><template #default="{ row }"><el-button link type="primary" :loading="probingKey === `component:${row.id}`" @click="runComponentProbe(row)">检测</el-button><el-button link type="primary" @click="openComponent(row)">编辑</el-button><el-button link type="danger" @click="removeComponent(row)">删除</el-button></template></el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane name="dependencies">
          <template #label>调用依赖 <el-badge :value="detail.dependencies?.length || 0" :max="99" /></template>
          <div class="inventory-panel__heading"><h3>出向调用关系</h3><el-button type="primary" :icon="Plus" @click="openDependency">新增依赖</el-button></div>
          <el-table :data="detail.dependencies" stripe>
            <el-table-column label="目标" min-width="190"><template #default="{ row }"><strong>{{ dependencyTarget(row) }}</strong><div class="inventory-table-cell__sub">{{ relationTypeLabel(row.relationType) }} · {{ row.protocol }}</div></template></el-table-column>
            <el-table-column label="调用地址" min-width="240"><template #default="{ row }">{{ endpoint(row.endpoint, row.port) || '-' }}</template></el-table-column>
            <el-table-column label="重要级别" width="100"><template #default="{ row }">{{ criticalityLabel(row.criticality) }}</template></el-table-column>
            <el-table-column label="启用状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
            <el-table-column label="说明" min-width="150" prop="description" show-overflow-tooltip />
            <el-table-column label="操作" width="130"><template #default="{ row }"><el-button link type="primary" @click="openDependency(row)">编辑</el-button><el-button link type="danger" @click="removeDependency(row)">删除</el-button></template></el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </section>

    <el-dialog v-model="dialogs.domain" width="800px" class="inventory-editor-dialog" destroy-on-close @closed="domainFormRef?.resetFields()">
      <template #header><EditorDialogHeader :icon="Link" eyebrow="APPLICATION ENTRY" :title="domainForm.id ? '编辑应用域名' : '新增应用域名'" description="环境由当前应用自动继承，保存后自动执行 DNS、HTTP/TCP 与 TLS 检测。" /></template>
      <el-form ref="domainFormRef" :model="domainForm" :rules="domainRules" label-position="top" class="inventory-editor-form">
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>01</span><div><h4>访问入口</h4><p>健康状态由探测生成，不需要人工判断。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="所属应用"><div class="inventory-readonly-field"><span>{{ detail.name }}</span><small>{{ detail.code }}</small></div></el-form-item>
          <el-form-item label="运行环境"><div class="inventory-readonly-field"><span>{{ environment?.name || '未关联' }}</span><small>由应用自动继承</small></div></el-form-item>
          <el-form-item label="域名" prop="domain"><el-input v-model="domainForm.domain" maxlength="255" placeholder="api.example.com" /></el-form-item>
          <el-form-item label="访问路径" prop="path"><el-input v-model="domainForm.path" maxlength="255" placeholder="/" /></el-form-item>
          <el-form-item label="协议" prop="protocol"><el-segmented v-model="domainForm.protocol" :options="protocolOptions" @change="handleDomainProtocolChange" /></el-form-item>
          <el-form-item label="端口" prop="port"><el-input-number v-model="domainForm.port" :min="1" :max="65535" /></el-form-item>
        </div></section>
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>02</span><div><h4>证书与流量</h4><p>HTTPS 可关联 SSL 证书中心中的证书。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="SSL 证书"><el-select v-model="domainForm.certificateId" :disabled="domainForm.protocol !== 'https'" clearable filterable placeholder="暂不关联"><el-option v-for="item in references.certificates || []" :key="item.id" :label="referenceLabel(item)" :value="item.id" /></el-select></el-form-item>
          <el-form-item label="DNS 服务商"><div class="inventory-readonly-field"><span>保存后自动识别</span><small>根据权威 NS 记录探测</small></div></el-form-item>
          <el-form-item label="设为主入口"><el-switch v-model="domainForm.isPrimary" inline-prompt active-text="是" inactive-text="否" /></el-form-item>
          <el-form-item label="检测状态"><div class="inventory-readonly-field"><StatusTag status="checking" /><span>保存后自动检测</span></div></el-form-item>
          <el-form-item label="说明" class="el-form-item--full"><el-input v-model="domainForm.description" type="textarea" :rows="2" maxlength="500" show-word-limit /></el-form-item>
        </div></section>
      </el-form>
      <template #footer><div class="inventory-dialog-footer"><span><i>*</i> 为必填项；状态由平台自动检测</span><div><el-button @click="dialogs.domain = false">取消</el-button><el-button type="primary" :loading="saving.domain" @click="saveDomain">保存并检测</el-button></div></div></template>
    </el-dialog>

    <el-dialog v-model="dialogs.resource" width="820px" class="inventory-editor-dialog" destroy-on-close @closed="resourceFormRef?.resetFields()">
      <template #header><EditorDialogHeader :icon="Monitor" eyebrow="DEPLOYMENT RESOURCE" :title="resourceForm.id ? '编辑部署资源' : '新增部署资源'" description="Host IP 来自资产管理，Kubernetes 状态来自关联集群。" /></template>
      <el-form ref="resourceFormRef" :model="resourceForm" :rules="resourceRules" label-position="top" class="inventory-editor-form">
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>01</span><div><h4>资源身份</h4><p>应用和运行环境已经固定，无需重复选择。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="所属应用"><div class="inventory-readonly-field"><span>{{ detail.name }}</span><small>{{ environment?.name || '未关联环境' }}</small></div></el-form-item>
          <el-form-item label="资源类型" prop="kind"><el-select v-model="resourceForm.kind" :disabled="Boolean(resourceForm.id)" @change="handleResourceKindChange"><el-option v-for="item in resourceKinds" :key="item.value" :label="item.label" :value="item.value" /></el-select><div v-if="resourceForm.id" class="inventory-form-hint">资源类型创建后不可修改，避免部署位置关系失真。</div></el-form-item>
          <el-form-item label="资源名称" prop="name" class="el-form-item--full"><el-input v-model="resourceForm.name" maxlength="180" :placeholder="isKubernetesResource ? 'Kubernetes 对象名称' : '例如：order-api-prod'" /></el-form-item>
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
      <template #footer><div class="inventory-dialog-footer"><span><i>*</i> 为必填项；Host 地址自动回填</span><div><el-button @click="dialogs.resource = false">取消</el-button><el-button type="primary" :loading="saving.resource" @click="saveResource">保存并检测</el-button></div></div></template>
    </el-dialog>

    <el-dialog v-model="dialogs.component" width="820px" class="inventory-editor-dialog" destroy-on-close @closed="componentFormRef?.resetFields()">
      <template #header><EditorDialogHeader :icon="Connection" eyebrow="RUNTIME COMPONENT" :title="componentForm.id ? '编辑数据库或中间件' : '新增数据库或中间件'" description="凭据从统一凭据中心授权使用，状态通过 TCP/TLS 连接自动检测。" /></template>
      <el-form ref="componentFormRef" :model="componentForm" :rules="componentRules" label-position="top" class="inventory-editor-form">
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>01</span><div><h4>组件身份</h4><p>运行环境自动继承当前应用。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="所属应用"><div class="inventory-readonly-field"><span>{{ detail.name }}</span><small>{{ environment?.name || '未关联环境' }}</small></div></el-form-item>
          <el-form-item label="组件分类" prop="category"><el-select v-model="componentForm.category"><el-option v-for="item in componentCategories" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
          <el-form-item label="组件类型" prop="type"><el-select v-model="componentForm.type" filterable allow-create default-first-option><el-option v-for="item in componentTypes" :key="item" :label="item" :value="item" /></el-select></el-form-item>
          <el-form-item label="组件名称" prop="name"><el-input v-model="componentForm.name" maxlength="150" placeholder="例如：订单库主实例" /></el-form-item>
          <el-form-item label="版本"><el-input v-model="componentForm.version" maxlength="80" placeholder="例如：8.0.36" /></el-form-item>
        </div></section>
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>02</span><div><h4>连接与安全</h4><p>账号密码不会写入组件记录。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="连接地址" prop="address"><el-input v-model="componentForm.address" maxlength="500" placeholder="主机名、IP 或服务地址" /></el-form-item>
          <el-form-item label="端口" prop="port"><el-input-number v-model="componentForm.port" :min="1" :max="65535" /></el-form-item>
          <el-form-item label="数据库 / 实例名"><el-input v-model="componentForm.databaseName" maxlength="120" /></el-form-item>
          <el-form-item label="关联凭据"><el-select v-model="componentForm.credentialId" clearable filterable placeholder="从统一凭据中心选择"><el-option v-for="item in credentials" :key="item.id" :label="`${item.name} (${item.username || '无用户名'})`" :value="item.id" /></el-select></el-form-item>
          <el-form-item label="启用 TLS"><el-switch v-model="componentForm.tlsEnabled" inline-prompt active-text="是" inactive-text="否" /></el-form-item>
          <el-form-item label="说明" class="el-form-item--full"><el-input v-model="componentForm.description" type="textarea" :rows="2" maxlength="500" show-word-limit /></el-form-item>
        </div></section>
      </el-form>
      <template #footer><div class="inventory-dialog-footer"><span><i>*</i> 为必填项；状态由连接探测生成</span><div><el-button @click="dialogs.component = false">取消</el-button><el-button type="primary" :loading="saving.component" @click="saveComponent">保存并检测</el-button></div></div></template>
    </el-dialog>

    <el-dialog v-model="dialogs.dependency" width="840px" class="inventory-editor-dialog" destroy-on-close @closed="dependencyFormRef?.resetFields()">
      <template #header><EditorDialogHeader :icon="Share" eyebrow="SERVICE DEPENDENCY" :title="dependencyForm.id ? '编辑调用依赖' : '新增调用依赖'" description="来源环境自动继承当前应用；启用状态仅表示是否纳入拓扑，不代表健康状态。" /></template>
      <el-form ref="dependencyFormRef" :model="dependencyForm" :rules="dependencyRules" label-position="top" class="inventory-editor-form">
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>01</span><div><h4>调用两端</h4><p>目标可选择已登记应用、组件、资源或外部服务。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="来源应用"><div class="inventory-readonly-field"><span>{{ detail.name }}</span><small>{{ environment?.name || '未关联环境' }}</small></div></el-form-item>
          <el-form-item label="目标类型" prop="targetType"><el-segmented v-model="dependencyForm.targetType" :options="dependencyTargetOptions" @change="resetDependencyTarget" /></el-form-item>
          <el-form-item v-if="dependencyForm.targetType === 'application'" label="目标应用" prop="targetApplicationId" class="el-form-item--full"><el-select v-model="dependencyForm.targetApplicationId" filterable><el-option v-for="item in allApplications.filter(item => item.id !== appId)" :key="item.id" :label="`${item.name} (${item.code})`" :value="item.id" /></el-select></el-form-item>
          <el-form-item v-else-if="dependencyForm.targetType === 'component'" label="目标组件" prop="targetComponentId" class="el-form-item--full"><el-select v-model="dependencyForm.targetComponentId" filterable @change="fillComponentEndpoint"><el-option v-for="item in targetComponents" :key="item.id" :label="`${item.name} · ${item.type} · ${applicationName(item.applicationId)}`" :value="item.id" /></el-select></el-form-item>
          <el-form-item v-else-if="dependencyForm.targetType === 'resource'" label="目标资源" prop="targetResourceId" class="el-form-item--full"><el-select v-model="dependencyForm.targetResourceId" filterable @change="fillResourceEndpoint"><el-option v-for="item in targetResources" :key="item.id" :label="`${item.name} · ${item.kind} · ${applicationName(item.applicationId)}`" :value="item.id" /></el-select></el-form-item>
          <el-form-item v-else label="外部目标名称" prop="targetName" class="el-form-item--full"><el-input v-model="dependencyForm.targetName" maxlength="180" placeholder="例如：微信支付开放平台" /></el-form-item>
        </div></section>
        <section class="inventory-form-section"><div class="inventory-form-section__heading"><span>02</span><div><h4>调用配置</h4><p>协议和地址用于拓扑连线与影响分析。</p></div></div><div class="inventory-form-grid">
          <el-form-item label="关系类型" prop="relationType"><el-select v-model="dependencyForm.relationType"><el-option v-for="item in relationTypes" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
          <el-form-item label="协议" prop="protocol"><el-select v-model="dependencyForm.protocol" filterable allow-create default-first-option><el-option v-for="item in dependencyProtocols" :key="item" :label="item" :value="item" /></el-select></el-form-item>
          <el-form-item label="调用地址" prop="endpoint" class="el-form-item--full"><el-input v-model="dependencyForm.endpoint" maxlength="500" placeholder="https://service/api 或 service.namespace" /></el-form-item>
          <el-form-item label="端口"><el-input-number v-model="dependencyForm.port" :min="0" :max="65535" /></el-form-item>
          <el-form-item label="重要级别" prop="criticality"><el-select v-model="dependencyForm.criticality"><el-option label="核心" value="critical" /><el-option label="高" value="high" /><el-option label="中" value="medium" /><el-option label="低" value="low" /></el-select></el-form-item>
          <el-form-item label="启用状态" prop="status"><el-select v-model="dependencyForm.status"><el-option label="启用并显示在拓扑" value="active" /><el-option label="停用并从拓扑隐藏" value="disabled" /></el-select></el-form-item>
          <el-form-item label="说明" class="el-form-item--full"><el-input v-model="dependencyForm.description" type="textarea" :rows="2" maxlength="500" show-word-limit /></el-form-item>
        </div></section>
      </el-form>
      <template #footer><div class="inventory-dialog-footer"><span><i>*</i> 为必填项；依赖状态是配置开关</span><div><el-button @click="dialogs.dependency = false">取消</el-button><el-button type="primary" :loading="saving.dependency" @click="saveDependency">保存依赖</el-button></div></div></template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Aim, ArrowLeft, CircleCheck, Connection, Download, Link, Monitor, Plus, Refresh, SetUp, Share } from '@element-plus/icons-vue'
import {
  createComponent, createDependency, createDomain, createResource,
  deleteComponent, deleteDependency, deleteDomain, deleteResource,
  getApplication, getReferences, listApplications, listComponents, listCredentials, listResources,
  probeApplication, probeComponent, probeDomain, probeResource,
  updateComponent, updateDependency, updateDomain, updateResource,
  type Component, type Domain, type Resource,
} from '../api'
import { parseTagList, validateDomainName, validateForm } from '../form-utils'
import EditorDialogHeader from './EditorDialogHeader.vue'
import StatusTag from './StatusTag.vue'

const route = useRoute()
const router = useRouter()
const appId = Number(route.params.id)
const loading = ref(false)
const detail = ref<any>()
const activeTab = ref('overview')
const references = ref<Record<string, any[]>>({})
const credentials = ref<any[]>([])
const allApplications = ref<any[]>([])
const targetComponents = ref<Component[]>([])
const targetResources = ref<Resource[]>([])
const dialogs = reactive({ domain: false, resource: false, component: false, dependency: false })
const saving = reactive({ domain: false, resource: false, component: false, dependency: false })
const probingKey = ref('')
const domainFormRef = ref<FormInstance>()
const resourceFormRef = ref<FormInstance>()
const componentFormRef = ref<FormInstance>()
const dependencyFormRef = ref<FormInstance>()
const domainForm = reactive<any>({})
const resourceForm = reactive<any>({})
const componentForm = reactive<any>({})
const dependencyForm = reactive<any>({})
const resourceKinds = [{ label: '物理主机', value: 'Host' }, { label: '虚拟机', value: 'VirtualMachine' }, { label: 'K8s Deployment', value: 'Deployment' }, { label: 'K8s StatefulSet', value: 'StatefulSet' }, { label: 'K8s DaemonSet', value: 'DaemonSet' }, { label: 'K8s Service', value: 'Service' }, { label: 'K8s Ingress', value: 'Ingress' }, { label: 'K8s Container', value: 'Container' }, { label: '其他资源', value: 'Other' }]
const kubernetesKinds = new Set(['Deployment', 'StatefulSet', 'DaemonSet', 'Service', 'Ingress', 'Container'])
const componentCategories = [{ label: '数据库', value: 'database' }, { label: '中间件', value: 'middleware' }, { label: '缓存', value: 'cache' }, { label: '消息队列', value: 'queue' }, { label: '搜索', value: 'search' }, { label: '对象存储', value: 'storage' }, { label: '外部服务', value: 'external' }]
const componentTypes = ['MySQL', 'PostgreSQL', 'MongoDB', 'Redis', 'Kafka', 'RabbitMQ', 'RocketMQ', 'Elasticsearch', 'OpenSearch', 'MinIO']
const dependencyProtocols = ['HTTP', 'HTTPS', 'gRPC', 'Dubbo', 'TCP', 'MySQL', 'PostgreSQL', 'Redis', 'Kafka', 'AMQP']
const protocolOptions = [{ label: 'HTTPS', value: 'https' }, { label: 'HTTP', value: 'http' }, { label: 'TCP', value: 'tcp' }]
const dependencyTargetOptions = [{ label: '应用', value: 'application' }, { label: '组件', value: 'component' }, { label: '资源', value: 'resource' }, { label: '外部服务', value: 'external' }]
const relationTypes = [{ label: 'HTTP 调用', value: 'http' }, { label: 'RPC 调用', value: 'rpc' }, { label: '数据库', value: 'database' }, { label: '缓存', value: 'cache' }, { label: '消息队列', value: 'queue' }, { label: '外部 API', value: 'external' }]
const environment = computed(() => detail.value?.environment || detail.value?.environments?.[0])
const isKubernetesResource = computed(() => kubernetesKinds.has(resourceForm.kind))
const isHostResource = computed(() => ['Host', 'VirtualMachine'].includes(resourceForm.kind))
const isOtherResource = computed(() => resourceForm.kind === 'Other')
const selectedHost = computed(() => (references.value.hosts || []).find(item => item.id === resourceForm.hostId))
const resourceLocationHint = computed(() => isHostResource.value ? '选择资产主机后自动使用其 IP，不再让用户重复填写访问地址。' : isKubernetesResource.value ? '选择集群和命名空间，平台按对象名称获取实时状态。' : '只有其他资源允许手工登记访问地址。')

const domainRules: FormRules = {
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }, { validator: validateDomainName, trigger: 'blur' }],
  protocol: [{ required: true, message: '请选择访问协议', trigger: 'change' }],
  port: [{ required: true, message: '请输入端口', trigger: 'change' }],
  path: [{ required: true, message: '请输入访问路径', trigger: 'blur' }, { pattern: /^\//, message: '访问路径必须以 / 开头', trigger: 'blur' }],
}
const resourceRules: FormRules = {
  kind: [{ required: true, message: '请选择资源类型', trigger: 'change' }],
  name: [{ required: true, message: '请输入资源名称', trigger: 'blur' }],
  hostId: [{ validator: (_rule, value, callback) => isHostResource.value && !value ? callback(new Error('请选择关联主机')) : callback(), trigger: 'change' }],
  clusterId: [{ validator: (_rule, value, callback) => isKubernetesResource.value && !value ? callback(new Error('请选择 Kubernetes 集群')) : callback(), trigger: 'change' }],
  namespace: [{ validator: (_rule, value, callback) => isKubernetesResource.value && !String(value || '').trim() ? callback(new Error('请输入命名空间')) : callback(), trigger: 'blur' }],
  address: [{ validator: (_rule, value, callback) => isOtherResource.value && !String(value || '').trim() && !String(resourceForm.externalId || '').trim() ? callback(new Error('请输入访问地址或外部标识')) : callback(), trigger: 'blur' }],
}
const componentRules: FormRules = {
  category: [{ required: true, message: '请选择组件分类', trigger: 'change' }],
  type: [{ required: true, message: '请选择或输入组件类型', trigger: 'change' }],
  name: [{ required: true, message: '请输入组件名称', trigger: 'blur' }],
  address: [{ required: true, message: '请输入连接地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入连接端口', trigger: 'change' }],
}
const dependencyRules: FormRules = {
  targetType: [{ required: true, message: '请选择目标类型', trigger: 'change' }],
  targetApplicationId: [{ validator: (_rule, value, callback) => dependencyForm.targetType === 'application' && !value ? callback(new Error('请选择目标应用')) : callback(), trigger: 'change' }],
  targetComponentId: [{ validator: (_rule, value, callback) => dependencyForm.targetType === 'component' && !value ? callback(new Error('请选择目标组件')) : callback(), trigger: 'change' }],
  targetResourceId: [{ validator: (_rule, value, callback) => dependencyForm.targetType === 'resource' && !value ? callback(new Error('请选择目标资源')) : callback(), trigger: 'change' }],
  targetName: [{ validator: (_rule, value, callback) => dependencyForm.targetType === 'external' && !String(value || '').trim() ? callback(new Error('请输入外部目标名称')) : callback(), trigger: 'blur' }],
  relationType: [{ required: true, message: '请选择关系类型', trigger: 'change' }],
  protocol: [{ required: true, message: '请选择或输入协议', trigger: 'change' }],
  endpoint: [{ required: true, message: '请输入调用地址', trigger: 'blur' }],
  criticality: [{ required: true, message: '请选择重要级别', trigger: 'change' }],
  status: [{ required: true, message: '请选择启用状态', trigger: 'change' }],
}

const loadDetail = async () => { loading.value = true; try { detail.value = await getApplication(appId) } catch { ElMessage.error('加载应用详情失败') } finally { loading.value = false } }
const loadReferences = async () => {
  const [refs, creds, apps, components, resources] = await Promise.all([getReferences(), listCredentials({ page: 1, page_size: 200 }).catch(() => ({ list: [] } as any)), listApplications({ page: 1, page_size: 200 }), listComponents({ page: 1, page_size: 200 }), listResources({ page: 1, page_size: 200 })])
  references.value = refs || {}; credentials.value = creds?.list || []; allApplications.value = apps?.list || []; targetComponents.value = components?.list || []; targetResources.value = resources?.list || []
}
const requireEnvironment = () => { if (environment.value?.id) return true; ElMessage.warning('请先在应用编辑中关联运行环境'); return false }
const endpoint = (address?: string, port?: number) => address ? `${address}${port && port > 0 ? `:${port}` : ''}` : ''
const domainEndpoint = (row: Domain) => `${row.protocol}://${row.domain}${row.port && !((row.protocol === 'https' && row.port === 443) || (row.protocol === 'http' && row.port === 80)) ? `:${row.port}` : ''}${row.path || '/'}`
const formatDate = (value?: string) => value && !Number.isNaN(new Date(value).getTime()) ? new Date(value).toLocaleDateString('zh-CN') : '-'
const formatDateTime = (value?: string) => value && !Number.isNaN(new Date(value).getTime()) ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '尚未检测'
const probeMeta = (row: any) => `${formatDateTime(row.lastCheckedAt)}${row.responseTimeMs > 0 ? ` · ${row.responseTimeMs} ms` : ''}${row.httpStatusCode > 0 ? ` · HTTP ${row.httpStatusCode}` : ''}`
const criticalityLabel = (value?: string) => ({ critical: '核心', high: '高', medium: '中', low: '低' } as Record<string, string>)[value || ''] || value || '-'
const lifecycleLabel = (value?: string) => ({ production: '生产', staging: '预发布', test: '测试', development: '开发' } as Record<string, string>)[value || ''] || value || '-'
const healthTone = (value?: string) => value === 'healthy' ? 'healthy' : value === 'warning' || value === 'checking' ? 'warning' : value === 'unhealthy' ? 'danger' : 'unknown'
const categoryLabel = (value: string) => componentCategories.find(item => item.value === value)?.label || value
const resourceKindLabel = (value: string) => resourceKinds.find(item => item.value === value)?.label || value
const sourceLabel = (value?: string) => value === 'kubernetes' ? 'Kubernetes' : value === 'discovered' ? '自动发现' : '手工登记'
const clusterName = (id?: number) => (references.value.clusters || []).find(item => item.id === id)?.name || (id ? `集群 #${id}` : 'Kubernetes')
const resourceLocation = (row: Resource) => row.namespace ? `${clusterName(row.clusterId)} / ${row.namespace}` : endpoint(row.address, row.port) || '未获取位置'
const referenceLabel = (item: any) => item.detail ? `${item.name} (${item.detail})` : item.name
const dependencyTarget = (row: any) => allApplications.value.find(item => item.id === row.targetApplicationId)?.name || targetComponents.value.find(item => item.id === row.targetComponentId)?.name || targetResources.value.find(item => item.id === row.targetResourceId)?.name || row.targetName || row.endpoint || '未命名目标'
const applicationName = (id: number) => allApplications.value.find(item => item.id === id)?.name || `应用 #${id}`
const relationTypeLabel = (value: string) => relationTypes.find(item => item.value === value)?.label || value
const optionalId = (value?: number) => value && value > 0 ? value : undefined

const openDomain = (row?: Domain) => { if (!requireEnvironment()) return; const value = { id: 0, applicationId: appId, domain: '', protocol: 'https', port: 443, path: '/', dnsProvider: '', certificateId: undefined as number | undefined, isPrimary: false, description: '', ...(row || {}) }; value.certificateId = optionalId(value.certificateId); Object.assign(domainForm, value); dialogs.domain = true }
const openResource = (row?: Resource) => { if (!requireEnvironment()) return; const value = { id: 0, applicationId: appId, kind: 'Host', name: '', address: '', port: 0, hostId: undefined as number | undefined, clusterId: undefined as number | undefined, namespace: '', externalId: '', credentialId: undefined as number | undefined, description: '', ...(row || {}) }; value.hostId = optionalId(value.hostId); value.clusterId = optionalId(value.clusterId); value.credentialId = optionalId(value.credentialId); Object.assign(resourceForm, value); if (isHostResource.value) handleHostChange(resourceForm.hostId); dialogs.resource = true }
const openComponent = (row?: Component) => { if (!requireEnvironment()) return; const value = { id: 0, applicationId: appId, category: 'database', type: 'MySQL', name: '', address: '', port: 3306, databaseName: '', version: '', credentialId: undefined as number | undefined, tlsEnabled: false, description: '', ...(row || {}) }; value.credentialId = optionalId(value.credentialId); Object.assign(componentForm, value); dialogs.component = true }
const openDependency = (row?: any) => {
  if (!requireEnvironment()) return
  const hasTargetApplication = allApplications.value.some(item => item.id !== appId)
  const targetType = row?.targetApplicationId ? 'application' : row?.targetComponentId ? 'component' : row?.targetResourceId ? 'resource' : row ? 'external' : hasTargetApplication ? 'application' : 'external'
  const value = { id: 0, sourceApplicationId: appId, targetApplicationId: undefined as number | undefined, targetComponentId: undefined as number | undefined, targetResourceId: undefined as number | undefined, targetName: '', relationType: 'http', protocol: 'HTTP', endpoint: '', port: 0, criticality: 'medium', status: 'active', description: '', ...(row || {}), targetType }
  value.targetApplicationId = optionalId(value.targetApplicationId)
  value.targetComponentId = optionalId(value.targetComponentId)
  value.targetResourceId = optionalId(value.targetResourceId)
  Object.assign(dependencyForm, value)
  dialogs.dependency = true
}
const handleDomainProtocolChange = (value: string) => { domainForm.port = value === 'https' ? 443 : value === 'http' ? 80 : domainForm.port || 80; if (value !== 'https') domainForm.certificateId = 0 }
const handleResourceKindChange = () => { resourceForm.hostId = undefined; resourceForm.clusterId = undefined; resourceForm.namespace = ''; resourceForm.address = ''; resourceForm.externalId = '' }
const handleHostChange = (id?: number) => { const host = (references.value.hosts || []).find(item => item.id === id); resourceForm.address = host?.address || '' }
const resetDependencyTarget = () => { dependencyForm.targetApplicationId = undefined; dependencyForm.targetComponentId = undefined; dependencyForm.targetResourceId = undefined; dependencyForm.targetName = ''; dependencyForm.endpoint = ''; dependencyForm.port = 0; dependencyFormRef.value?.clearValidate(['targetApplicationId', 'targetComponentId', 'targetResourceId', 'targetName']) }
const fillComponentEndpoint = (id: number) => { const item = targetComponents.value.find(row => row.id === id); if (!item) return; dependencyForm.endpoint = item.address || ''; dependencyForm.port = item.port || 0; dependencyForm.protocol = item.type || dependencyForm.protocol }
const fillResourceEndpoint = (id: number) => { const item = targetResources.value.find(row => row.id === id); if (!item) return; dependencyForm.endpoint = item.address || item.name || ''; dependencyForm.port = item.port || 0; dependencyForm.protocol = 'TCP' }

const saveDomain = async () => { if (!await validateForm(domainFormRef.value)) return; saving.domain = true; try { const payload = { applicationId: appId, domain: domainForm.domain, protocol: domainForm.protocol, port: domainForm.port, path: domainForm.path, certificateId: domainForm.certificateId || 0, isPrimary: domainForm.isPrimary, description: domainForm.description }; domainForm.id ? await updateDomain(domainForm.id, payload) : await createDomain(payload); dialogs.domain = false; ElMessage.success('域名已保存并完成检测'); await loadDetail() } finally { saving.domain = false } }
const saveResource = async () => { if (!await validateForm(resourceFormRef.value)) return; saving.resource = true; try { const payload = { applicationId: appId, kind: resourceForm.kind, name: resourceForm.name, address: isOtherResource.value ? resourceForm.address : '', port: resourceForm.port || 0, hostId: isHostResource.value ? resourceForm.hostId : 0, clusterId: isKubernetesResource.value ? resourceForm.clusterId : 0, namespace: isKubernetesResource.value ? resourceForm.namespace : '', externalId: isOtherResource.value ? resourceForm.externalId : '', credentialId: resourceForm.credentialId || 0, description: resourceForm.description }; resourceForm.id ? await updateResource(resourceForm.id, payload) : await createResource(payload); dialogs.resource = false; ElMessage.success('资源已保存并完成检测'); await loadDetail() } finally { saving.resource = false } }
const saveComponent = async () => { if (!await validateForm(componentFormRef.value)) return; saving.component = true; try { const payload = { applicationId: appId, category: componentForm.category, type: componentForm.type, name: componentForm.name, address: componentForm.address, port: componentForm.port, databaseName: componentForm.databaseName, version: componentForm.version, credentialId: componentForm.credentialId || 0, tlsEnabled: componentForm.tlsEnabled, description: componentForm.description }; componentForm.id ? await updateComponent(componentForm.id, payload) : await createComponent(payload); dialogs.component = false; ElMessage.success('组件已保存并完成检测'); await Promise.all([loadDetail(), loadReferences()]) } finally { saving.component = false } }
const saveDependency = async () => { if (!await validateForm(dependencyFormRef.value)) return; saving.dependency = true; try { const { id, targetType: _targetType, ...payload } = dependencyForm; id ? await updateDependency(id, payload) : await createDependency(payload); dialogs.dependency = false; ElMessage.success('依赖已保存'); await Promise.all([loadDetail(), loadReferences()]) } finally { saving.dependency = false } }
const runProbe = async (key: string, action: () => Promise<unknown>) => { probingKey.value = key; try { await action(); ElMessage.success('检测完成'); await loadDetail() } finally { probingKey.value = '' } }
const runApplicationProbe = () => runProbe(`application:${appId}`, () => probeApplication(appId))
const runDomainProbe = (row: Domain) => runProbe(`domain:${row.id}`, () => probeDomain(row.id))
const runResourceProbe = (row: Resource) => runProbe(`resource:${row.id}`, () => probeResource(row.id))
const runComponentProbe = (row: Component) => runProbe(`component:${row.id}`, () => probeComponent(row.id))
const confirmDelete = (name: string) => ElMessageBox.confirm(`确认删除“${name}”？`, '删除确认', { type: 'warning' })
const removeDomain = async (row: Domain) => { await confirmDelete(row.domain); await deleteDomain(row.id); await loadDetail() }
const removeResource = async (row: Resource) => { await confirmDelete(row.name); await deleteResource(row.id); await loadDetail() }
const removeComponent = async (row: Component) => { await confirmDelete(row.name); await deleteComponent(row.id); await loadDetail() }
const removeDependency = async (row: any) => { await confirmDelete(dependencyTarget(row)); await deleteDependency(row.id); await loadDetail() }

onMounted(async () => { await Promise.all([loadDetail(), loadReferences()]) })
</script>
