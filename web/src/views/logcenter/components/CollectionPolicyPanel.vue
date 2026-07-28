<template>
  <div class="policy-page">
    <section class="section-head panel">
      <div>
        <h3>采集策略</h3>
        <p>{{ policyViewDescription }}</p>
      </div>
      <div class="section-actions">
        <el-segmented v-model="policyView" :options="policyViewOptions" @change="handlePolicyViewChange" />
        <el-input v-model="keyword" clearable placeholder="搜索策略" @keyup.enter="loadPolicies">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-if="policyView === 'current'" v-model="statusFilter" clearable placeholder="全部状态" @change="loadPolicies">
          <el-option label="草稿" value="draft" />
          <el-option label="已发布" value="published" />
          <el-option label="已停用" value="disabled" />
        </el-select>
        <el-select v-else model-value="archived" disabled>
          <el-option label="已归档" value="archived" />
        </el-select>
        <el-button :loading="loading" @click="loadPolicies"><el-icon><Refresh /></el-icon>刷新</el-button>
        <el-button v-if="isAdmin" type="primary" class="primary-action" @click="openCreateFromToolbar"><el-icon><Plus /></el-icon>新建策略</el-button>
      </div>
    </section>

    <section class="policy-table panel" v-loading="loading">
      <el-table :data="policies" :empty-text="policyView === 'archived' ? '暂无已归档策略' : '暂无采集策略'">
        <el-table-column label="策略" min-width="230">
          <template #default="{ row }">
            <div class="policy-name"><strong>{{ row.payload.name }}</strong><small>{{ row.payload.description || '未填写说明' }}</small></div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="105">
          <template #default="{ row }"><div class="policy-status"><el-tag size="small" :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag><el-tag v-if="row.hasUnpublishedChanges" size="small" type="warning">待发布</el-tag></div></template>
        </el-table-column>
        <el-table-column label="版本" width="86"><template #default="{ row }">v{{ row.version || 0 }}</template></el-table-column>
        <el-table-column label="目标资产" width="130">
          <template #default="{ row }"><strong>{{ row.targetCount }}</strong> {{ row.payload.sourceMode === 'kubernetes' ? '个集群' : '台主机' }}</template>
        </el-table-column>
        <el-table-column min-width="245">
          <template #header>
            <span class="column-heading">下发进度<el-tooltip content="Agent 每 30 秒拉取策略，成功切换配置并回报后计为已应用" placement="top"><el-icon><InfoFilled /></el-icon></el-tooltip></span>
          </template>
          <template #default="{ row }">
            <el-popover trigger="hover" placement="bottom-start" :width="340">
              <template #reference>
                <div class="rollout-cell">
                  <div class="rollout-line">
                    <el-progress :percentage="rolloutPercent(row)" :stroke-width="7" :show-text="false" />
                    <strong>{{ rolloutText(row) }}</strong>
                    <el-tag v-if="row.errorInstances" size="small" type="danger">{{ row.errorInstances }} 失败</el-tag>
                    <el-tag v-else-if="rolloutWaiting(row)" size="small" type="warning">等待确认</el-tag>
                  </div>
                  <small>{{ rolloutSubtext(row) }}</small>
                </div>
              </template>
              <div class="rollout-detail">
                <div class="rollout-detail-head"><strong>策略 v{{ row.version || 0 }} 下发状态</strong><el-tag size="small" :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag></div>
                <dl>
                  <div><dt>{{ rolloutTargetLabel(row) }}</dt><dd>{{ rolloutExpected(row) }}</dd></div>
                  <div><dt>已接入</dt><dd>{{ row.instanceTotal || 0 }}</dd></div>
                  <div><dt>当前在线</dt><dd>{{ row.instanceOnline || 0 }}</dd></div>
                  <div><dt>已应用</dt><dd class="success">{{ row.instanceApplied || 0 }}</dd></div>
                  <div><dt>待确认</dt><dd class="warning">{{ rolloutPending(row) }}</dd></div>
                  <div><dt>失败</dt><dd :class="{ danger: row.errorInstances }">{{ row.errorInstances || 0 }}</dd></div>
                </dl>
                <p>每个 Agent/Collector 拉取并成功应用当前版本后，进度增加 1；页面每 10 秒刷新一次。</p>
                <el-button link type="primary" @click="openInstances(row)">查看采集实例与失败原因</el-button>
              </div>
            </el-popover>
          </template>
        </el-table-column>
        <el-table-column label="日志路径" min-width="220" show-overflow-tooltip>
          <template #default="{ row }"><code>{{ row.payload.paths.join(', ') }}</code></template>
        </el-table-column>
        <el-table-column label="更新时间" width="172"><template #default="{ row }">{{ formatTime(row.updatedAt) }}</template></el-table-column>
        <el-table-column label="操作" fixed="right" width="350">
          <template #default="{ row }">
            <template v-if="isAdmin">
              <el-button v-if="policyView === 'archived'" link type="success" @click="restorePolicy(row)">恢复为草稿</el-button>
              <template v-else>
                <el-button link @click="openEdit(row)">编辑</el-button>
                <el-button v-if="needsCollectorSetup(row)" link type="primary" @click="openCollectorSetup(row)">接入采集器</el-button>
                <el-button v-if="row.hasUnpublishedChanges || row.status !== 'published'" link type="primary" @click="publishExisting(row)">{{ row.hasUnpublishedChanges ? '发布变更' : '发布' }}</el-button>
                <el-button v-if="row.status === 'published'" link type="warning" @click="disableExisting(row)">停用</el-button>
                <el-button v-if="row.status === 'draft' && row.version === 0" link type="danger" @click="removePolicy(row)">删除</el-button>
                <el-button v-else-if="row.status === 'disabled'" link type="danger" @click="removePolicy(row)">移入归档</el-button>
              </template>
            </template>
            <span v-else class="readonly-text">只读</span>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑采集策略' : '新建采集策略'" width="min(920px, calc(100vw - 32px))" destroy-on-close class="policy-dialog">
      <el-steps :active="step" finish-status="success" align-center class="policy-steps">
        <el-step title="基本信息" />
        <el-step title="目标资产" />
        <el-step title="采集规则" />
        <el-step title="确认发布" />
      </el-steps>

      <el-form :model="form" label-position="top" class="policy-form">
        <div v-show="step === 0" class="form-stage two-column">
          <el-form-item label="采集类型" class="full">
            <el-segmented v-model="form.sourceMode" :options="sourceModeOptions" @change="handleSourceModeChange" />
          </el-form-item>
          <el-form-item label="策略名称" prop="name" :rules="[{ required: true, message: '请输入策略名称' }]">
            <el-input v-model="form.name" maxlength="120" placeholder="例如：生产环境 Nginx 日志" />
          </el-form-item>
          <el-form-item label="运行环境" prop="environment" class="required-label" :rules="[{ required: true, message: '请选择运行环境' }]">
            <el-select v-model="form.environment" filterable allow-create default-first-option placeholder="请选择或输入环境">
              <el-option v-for="item in environmentOptions" :key="item.value" :label="`${item.label}（${item.value}）`" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="服务名称" prop="service" class="required-label" :rules="[{ required: true, message: '请输入服务名称' }]">
            <el-input v-model="form.service" maxlength="120" placeholder="例如：nginx、order-api" />
          </el-form-item>
          <div class="retention-config full">
            <div class="retention-toolbar">
              <el-form-item label="保留策略模板">
                <el-select v-model="form.retentionPolicyId" clearable placeholder="自定义" @change="applyRetentionProfile">
                  <el-option v-for="item in retentionProfiles" :key="item.id" :label="`${item.payload.name}${item.payload.enabled ? '' : '（已停用）'}`" :value="item.id" :disabled="!item.payload.enabled" />
                </el-select>
              </el-form-item>
              <el-form-item label="默认保留周期"><div class="days-control"><el-input-number v-model="form.retention.defaultDays" :min="1" :max="3650" controls-position="right" :disabled="Boolean(form.retentionPolicyId)" /><span>天</span></div></el-form-item>
            </div>
            <el-alert v-if="form.retentionPolicyId" class="retention-profile-alert" type="info" :closable="false" show-icon title="保留周期由绑定策略管理；发布时后端会重新读取最新配置并生成不可变快照" />
            <div class="retention-title"><strong>按日志级别保留时长</strong><span>这里只控制清理天数，不参与日志级别判定</span><el-tooltip content="识别到日志级别时使用对应天数；未识别级别时使用默认保留周期" placement="top"><el-icon><InfoFilled /></el-icon></el-tooltip></div>
            <div class="retention-level-grid">
              <label v-for="level in retentionLevelOptions" :key="level.value">
                <span class="level-name" :class="`level-${level.value.toLowerCase()}`">{{ level.value }}</span>
                <small>{{ level.label }}</small>
                <div class="days-control"><el-input-number v-model="form.retention.levelDays[level.value]" :min="1" :max="3650" controls-position="right" :disabled="Boolean(form.retentionPolicyId)" /><span>天</span></div>
              </label>
            </div>
          </div>
          <el-form-item label="策略说明" class="full"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="500" show-word-limit /></el-form-item>
        </div>

        <div v-show="step === 1" class="form-stage">
          <template v-if="form.sourceMode === 'host'">
            <el-alert type="info" :closable="false" show-icon title="可同时绑定具体主机和主机分组；分组中新加入的主机会自动继承已发布策略。" />
            <div class="target-grid">
              <el-form-item label="选择主机">
                <el-select v-model="selectedHostIds" multiple filterable collapse-tags :max-collapse-tags="4" placeholder="请选择主机">
                  <el-option v-for="host in targetOptions.hosts" :key="host.id" :label="`${host.name} (${host.ip})`" :value="host.id">
                    <div class="host-option"><span>{{ host.name }} <small>{{ host.ip }}</small></span><el-tag size="small" :type="host.agentId ? 'success' : 'info'">{{ host.agentId ? 'Agent 已安装' : '未安装 Agent' }}</el-tag></div>
                  </el-option>
                </el-select>
              </el-form-item>
              <el-form-item label="选择主机分组">
                <el-select v-model="selectedGroupIds" multiple filterable collapse-tags placeholder="可选">
                  <el-option v-for="group in targetOptions.groups" :key="group.id" :label="`${group.name} (${group.hostCount} 台)`" :value="group.id" />
                </el-select>
              </el-form-item>
            </div>
            <div class="target-summary">
              <span>已选择 <strong>{{ selectedHostIds.length }}</strong> 台主机、<strong>{{ selectedGroupIds.length }}</strong> 个分组</span>
              <span v-if="hostsWithoutAgent" class="warning-text">其中 {{ hostsWithoutAgent }} 台主机尚未安装 Agent，可先发布后安装</span>
            </div>
          </template>
          <template v-else>
            <el-alert type="info" :closable="false" show-icon title="DaemonSet 只采集命中 Namespace、Workload、Pod 标签和容器名称条件的日志。" />
            <div class="target-grid kubernetes-target-grid">
              <el-form-item label="Kubernetes 集群" class="full required-label">
                <el-select v-model="selectedClusterId" filterable placeholder="请选择已纳管集群" @change="loadKubernetesOptions">
                  <el-option v-for="cluster in targetOptions.clusters" :key="cluster.id" :label="`${cluster.alias || cluster.name} (${cluster.nodeCount} 节点)`" :value="cluster.id" />
                </el-select>
              </el-form-item>
              <div v-if="selectedClusterId" v-loading="collectorStatusLoading" class="collector-readiness full" :class="{ ready: collectorStatus?.installed && !collectorStatus?.lastError, error: collectorStatusError || collectorStatus?.lastError }">
                <div class="collector-readiness-main">
                  <i></i>
                  <div>
                    <strong>节点日志采集器</strong>
                    <small>{{ collectorStatusDescription }}</small>
                  </div>
                </div>
                <div v-if="collectorStatus" class="collector-readiness-metrics">
                  <span>DaemonSet <strong>{{ collectorStatus.readyNodes }}/{{ collectorStatus.desiredNodes }}</strong></span>
                  <span>在线实例 <strong>{{ collectorStatus.instanceOnline }}/{{ collectorStatus.instanceTotal }}</strong></span>
                </div>
                <el-button type="primary" class="primary-action" :loading="collectorInstalling" @click="installSelectedClusterCollector">
                  {{ collectorStatus?.installed ? '升级采集器' : '安装采集器' }}
                </el-button>
              </div>
              <el-form-item label="Namespace">
                <el-select v-model="selectedNamespaces" multiple filterable collapse-tags :loading="kubernetesOptionsLoading" placeholder="留空表示全集群" @change="handleNamespaceChange">
                  <el-option v-for="namespace in kubernetesOptions.namespaces" :key="namespace" :label="namespace" :value="namespace" />
                </el-select>
              </el-form-item>
              <el-form-item label="Workload">
                <el-select v-model="selectedWorkloads" multiple filterable collapse-tags :loading="kubernetesOptionsLoading" placeholder="可选，支持多选">
                  <el-option v-for="item in filteredWorkloads" :key="workloadValue(item)" :label="`${item.namespace} / ${item.kind} / ${item.name}`" :value="workloadValue(item)" />
                </el-select>
              </el-form-item>
              <el-form-item label="Pod Label Selector" class="full"><el-input v-model="labelSelector" placeholder="例如：app.kubernetes.io/name=order-api,environment=production" /></el-form-item>
              <el-form-item label="包含容器"><el-input v-model="containerIncludeText" type="textarea" :rows="2" placeholder="每行一个名称或通配符，留空表示全部" /></el-form-item>
              <el-form-item label="排除容器"><el-input v-model="containerExcludeText" type="textarea" :rows="2" placeholder="例如：istio-proxy 或 *-sidecar" /></el-form-item>
            </div>
            <div class="target-summary"><span>采集范围：<strong>{{ kubernetesTargetSummary }}</strong></span><span>系统 Namespace 默认排除，可显式选择后采集</span></div>
          </template>
        </div>

        <div v-show="step === 2" class="form-stage two-column">
          <el-form-item :label="form.sourceMode === 'kubernetes' ? '容器日志路径' : '采集路径'" class="full required-label">
            <el-input v-model="pathText" type="textarea" :rows="form.sourceMode === 'kubernetes' ? 2 : 4" :disabled="form.sourceMode === 'kubernetes'" placeholder="每行一个路径，例如：&#10;/var/log/nginx/*.log&#10;/data/apps/*/logs/*.log" />
            <div class="field-tip">{{ form.sourceMode === 'kubernetes' ? '固定读取节点 /var/log/containers，自动兼容 CRI partial 与 Docker JSON 日志。' : '每行一个路径，按回车分隔；支持 * 和 ? glob 通配符，至少填写一行。' }}</div>
          </el-form-item>
          <el-form-item label="排除路径" class="full"><el-input v-model="excludePathText" type="textarea" :rows="2" placeholder="每行一个，例如：*.gz 或 *.tmp" /><div class="field-tip">同样按回车分隔，命中的文件不会被采集。</div></el-form-item>
          <el-form-item label="首次读取位置">
            <el-radio-group v-model="form.readFrom"><el-radio-button value="latest">只读新日志</el-radio-button><el-radio-button value="beginning">从头读取</el-radio-button></el-radio-group>
          </el-form-item>
          <el-form-item label="日志解析器">
            <el-select v-model="form.parser.type"><el-option label="原始文本（自动识别级别）" value="raw" /><el-option label="JSON 结构化日志" value="json" /><el-option label="正则命名分组提取" value="regex" /></el-select>
            <div class="field-tip">{{ parserHelpText }}</div>
          </el-form-item>
          <el-form-item v-if="form.parser.type === 'regex'" label="解析正则" class="full required-label">
            <el-input v-model="form.parser.pattern" type="textarea" :rows="2" placeholder="例如：^(?P<timestamp>\\S+\\s+\\S+)\\s+(?P<level>\\w+)\\s+(?P<message>.*)$" />
            <div class="field-tip">支持 timestamp、level、message、trace_id、span_id 命名分组；不匹配的行会保留原文并标记解析错误。</div>
          </el-form-item>
          <template v-if="form.parser.type === 'json'">
            <el-form-item label="消息字段"><el-input v-model="form.parser.messageField" placeholder="message" /></el-form-item>
            <el-form-item label="级别字段"><el-input v-model="form.parser.levelField" placeholder="level" /></el-form-item>
            <el-form-item label="时间字段"><el-input v-model="form.parser.timestampField" placeholder="timestamp" /></el-form-item>
            <el-form-item label="时间格式（可选）"><el-input v-model="form.parser.timestampLayout" placeholder="留空自动识别 RFC3339 / 常见日期" /></el-form-item>
          </template>
          <el-form-item label="多行日志"><el-switch v-model="form.multiline.enabled" active-text="合并堆栈日志" /><div class="field-tip">在解析前将异常堆栈的多个物理行合并为一条完整日志。</div></el-form-item>
          <el-form-item v-if="form.multiline.enabled" label="多行模板">
            <el-select v-model="form.multiline.preset"><el-option label="Java 堆栈" value="java" /><el-option label="Go panic" value="go" /><el-option label="Python traceback" value="python" /><el-option label="自定义" value="custom" /></el-select>
            <div class="field-tip">{{ multilineHelpText }}</div>
          </el-form-item>
          <el-form-item v-if="form.multiline.enabled && form.multiline.preset === 'custom'" label="新日志起始行正则" class="full required-label"><el-input v-model="form.multiline.startPattern" placeholder="匹配时表示一条新日志开始" /></el-form-item>
          <template v-if="form.multiline.enabled">
            <el-form-item label="最多合并行数"><el-input-number v-model="form.multiline.maxLines" :min="2" :max="5000" controls-position="right" /></el-form-item>
            <el-form-item label="单条合并上限"><el-select v-model="form.multiline.maxBytes"><el-option v-for="option in multilineSizeOptions" :key="option.value" :label="option.label" :value="option.value" /></el-select></el-form-item>
            <el-form-item label="空闲刷新等待"><div class="number-with-unit"><el-input-number v-model="form.multiline.flushSeconds" :min="1" :max="60" controls-position="right" /><span>秒</span></div></el-form-item>
          </template>
          <el-form-item label="单行最大长度">
            <el-select v-model="form.maxLineBytes"><el-option v-for="option in lineSizeOptions" :key="option.value" :label="option.label" :value="option.value" /></el-select>
            <div class="field-tip">超过上限的物理行会被截断并添加 log.truncated 标记，Agent 最大支持 1 MiB。</div>
          </el-form-item>
          <el-form-item label="WAL 最大容量"><el-select v-model="form.walMaxBytes"><el-option label="512 MiB" :value="536870912" /><el-option label="1 GiB" :value="1073741824" /><el-option label="2 GiB" :value="2147483648" /><el-option label="5 GiB" :value="5368709120" /></el-select><div class="field-tip">日志发送失败时暂存在 Agent 本地，达到上限后暂停接收新日志，发送恢复后自动继续。</div></el-form-item>
          <div class="security-config full">
            <div class="security-head"><div><strong>采集前脱敏</strong><span>日志进入本地 WAL 之前就替换或删除敏感值，原始密码和 Token 不会发往 OpsHub</span></div><el-switch v-model="form.redaction.enabled" /></div>
            <template v-if="form.redaction.enabled">
              <div class="security-options">
                <el-checkbox v-model="form.redaction.useDefaultRules">启用 password、token、authorization、cookie、secret 默认规则</el-checkbox>
                <div class="sensitive-field-setting">
                  <span>补充敏感字段</span>
                  <el-select v-model="form.redaction.sensitiveFields" multiple filterable allow-create default-first-option collapse-tags :max-collapse-tags="8" placeholder="选择或输入后回车">
                    <el-option v-for="field in sensitiveFieldOptions" :key="field" :label="field" :value="field" />
                  </el-select>
                  <small>匹配 JSON 字段名、解析后属性名，以及 password=xxx 这类文本键值。</small>
                </div>
              </div>
              <div class="redaction-rules">
                <div class="rules-head"><span>自定义脱敏规则</span><el-button link type="primary" @click="addRedactionRule"><el-icon><Plus /></el-icon>添加规则</el-button></div>
                <p class="rules-help">字段适合 authorization 这类属性；JSON Path 适合 user.password 这类嵌套字段；正则可在日志正文中替换、哈希或删除匹配内容。</p>
                <div v-for="(rule, index) in form.redaction.rules" :key="index" class="rule-row">
                  <el-select v-model="rule.target"><el-option label="字段" value="field" /><el-option label="JSON Path" value="json_path" /><el-option label="正则" value="regex" /></el-select>
                  <el-input v-if="rule.target === 'regex'" v-model="rule.pattern" placeholder="正则表达式" /><el-input v-else v-model="rule.field" :placeholder="rule.target === 'json_path' ? '$.user.password' : 'authorization'" />
                  <el-select v-model="rule.action"><el-option label="替换" value="replace" /><el-option label="不可逆哈希" value="hash" /><el-option label="删除字段/匹配内容" value="drop_field" /></el-select>
                  <el-input v-if="rule.action === 'replace'" v-model="rule.replacement" placeholder="默认 [REDACTED]" />
                  <span v-else></span>
                  <el-button link type="danger" @click="form.redaction.rules.splice(index, 1)">删除</el-button>
                </div>
                <el-empty v-if="!form.redaction.rules.length" :image-size="42" description="暂无自定义规则" />
              </div>
            </template>
          </div>
        </div>

        <div v-show="step === 3" class="form-stage review-stage">
          <div class="review-card"><span>策略</span><strong>{{ form.name || '-' }}</strong><small>{{ form.environment || '未指定环境' }} · {{ form.service || '未指定服务' }}</small></div>
          <div class="review-card"><span>目标范围</span><strong>{{ form.sourceMode === 'kubernetes' ? kubernetesTargetSummary : `${selectedHostIds.length} 台主机 + ${selectedGroupIds.length} 个分组` }}</strong><small>发布后由在线采集实例自动拉取</small></div>
          <div class="review-card wide"><span>采集路径</span><code v-for="path in normalizedPaths" :key="path">{{ path }}</code></div>
          <div class="review-card"><span>解析方式</span><strong>{{ parserText }}</strong><small>{{ form.multiline.enabled ? `${form.multiline.preset} 多行合并` : '单行日志' }}</small></div>
          <div class="review-card"><span>可靠性</span><strong>{{ formatBytes(form.walMaxBytes) }} WAL</strong><small>ACK 后清理，断线自动重放</small></div>
          <div class="review-card"><span>保留与安全</span><strong>{{ form.retention.defaultDays }} 天默认保留</strong><small>{{ form.redaction.enabled ? '采集前脱敏已启用' : '采集前脱敏未启用' }}</small></div>
          <el-form-item label="发布说明" class="wide"><el-input v-model="changeSummary" placeholder="例如：首次上线 Nginx 访问日志采集" /></el-form-item>
        </div>
      </el-form>

      <template #footer>
        <div class="dialog-footer"><el-button @click="dialogVisible = false">取消</el-button><span class="footer-spacer" /><el-button v-if="step > 0" @click="step--">上一步</el-button><el-button v-if="step < 3" type="primary" class="primary-action" @click="nextStep">下一步</el-button><template v-else><el-button :loading="saving" @click="savePolicy(false)">保存草稿</el-button><el-button type="primary" class="primary-action" :loading="publishing" @click="savePolicy(true)">保存并发布</el-button></template></div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { InfoFilled, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import {
  createLogCollectionPolicy, deleteLogCollectionPolicy, disableLogCollectionPolicy,
  getLogCollectionPolicies, getLogKubernetesCollectorStatus, getLogKubernetesPolicyOptions, getLogPolicyTargetOptions, getLogRetentionPolicies,
  installLogKubernetesCollector, publishLogCollectionPolicy,
  restoreLogCollectionPolicy, updateLogCollectionPolicy, type LogCollectionPolicy, type LogCollectionPolicyPayload,
  type LogKubernetesCollectorStatus, type LogKubernetesWorkloadOption, type LogPolicyTargetCluster, type LogPolicyTargetGroup, type LogPolicyTargetHost, type LogRetentionPolicy,
} from '@/api/logcenter'

const props = defineProps<{ presetHostId?: number; presetClusterId?: number }>()
const emit = defineEmits<{ (event: 'open-instances', policyId?: number): void }>()
const userStore = useUserStore()
const isAdmin = computed(() => (userStore.userInfo?.roles || []).some((role: any) => role.code === 'admin'))

const emptyForm = (): LogCollectionPolicyPayload => ({
  name: '', sourceMode: 'host', description: '', paths: [], excludePaths: [], readFrom: 'latest', encoding: 'utf-8',
  environment: 'prod', service: '', stream: 'stdout', maxLineBytes: 262144,
  parser: { type: 'raw', messageField: 'message', levelField: 'level', timestampField: 'timestamp' },
  multiline: { enabled: false, preset: 'java', maxLines: 500, maxBytes: 1048576, flushSeconds: 2 },
  redaction: { configured: true, enabled: true, useDefaultRules: true, sensitiveFields: ['password', 'token', 'authorization', 'cookie', 'secret'], rules: [] },
  retentionPolicyId: undefined,
  retention: { defaultDays: 30, levelDays: { TRACE: 7, DEBUG: 7, INFO: 30, WARN: 60, ERROR: 90, FATAL: 180 } },
  retentionDays: 30, walMaxBytes: 2147483648, targets: [],
})

const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const dialogVisible = ref(false)
const keyword = ref('')
const policyView = ref<'current' | 'archived'>('current')
const statusFilter = ref('')
const policies = ref<LogCollectionPolicy[]>([])
const retentionProfiles = ref<LogRetentionPolicy[]>([])
const policyViewOptions = [{ label: '当前策略', value: 'current' }, { label: '已归档', value: 'archived' }]
const policyViewDescription = computed(() => policyView.value === 'archived'
  ? '已归档策略不再下发，发布版本、目标快照和审计记录会继续保留'
  : '统一管理主机文件与 Kubernetes 容器日志，发布后采集实例将在 30 秒内自动应用')
const environmentOptions = [
  { value: 'dev', label: '开发环境' },
  { value: 'test', label: '测试环境' },
  { value: 'sit', label: '集成测试环境' },
  { value: 'uat', label: '用户验收环境' },
  { value: 'staging', label: '预发布环境' },
  { value: 'prod', label: '生产环境' },
]
const retentionLevelOptions = [
  { value: 'TRACE', label: '链路跟踪' },
  { value: 'DEBUG', label: '调试信息' },
  { value: 'INFO', label: '运行信息' },
  { value: 'WARN', label: '风险警告' },
  { value: 'ERROR', label: '错误日志' },
  { value: 'FATAL', label: '致命错误' },
]
const lineSizeOptions = [
  { label: '64 KiB', value: 64 * 1024 },
  { label: '128 KiB', value: 128 * 1024 },
  { label: '256 KiB（推荐）', value: 256 * 1024 },
  { label: '512 KiB', value: 512 * 1024 },
  { label: '1 MiB', value: 1024 * 1024 },
]
const multilineSizeOptions = [
  { label: '256 KiB', value: 256 * 1024 },
  { label: '512 KiB', value: 512 * 1024 },
  { label: '1 MiB（推荐）', value: 1024 * 1024 },
]
const sensitiveFieldOptions = [
  'password', 'passwd', 'pwd', 'token', 'access_token', 'refresh_token', 'authorization',
  'cookie', 'set-cookie', 'secret', 'client_secret', 'api_key', 'apikey', 'access_key', 'secret_key',
]
const targetOptions = reactive<{ hosts: LogPolicyTargetHost[]; groups: LogPolicyTargetGroup[]; clusters: LogPolicyTargetCluster[] }>({ hosts: [], groups: [], clusters: [] })
const kubernetesOptions = reactive<{ namespaces: string[]; workloads: LogKubernetesWorkloadOption[] }>({ namespaces: [], workloads: [] })
const kubernetesOptionsLoading = ref(false)
const collectorStatusLoading = ref(false)
const collectorInstalling = ref(false)
const collectorStatus = ref<LogKubernetesCollectorStatus>()
const collectorStatusError = ref('')
const editingId = ref<number>()
const step = ref(0)
const form = reactive<LogCollectionPolicyPayload>(emptyForm())
const selectedHostIds = ref<number[]>([])
const selectedGroupIds = ref<number[]>([])
const selectedClusterId = ref<number>()
const selectedNamespaces = ref<string[]>([])
const selectedWorkloads = ref<string[]>([])
const labelSelector = ref('')
const containerIncludeText = ref('')
const containerExcludeText = ref('')
const pathText = ref('')
const excludePathText = ref('*.gz\n*.tmp')
const changeSummary = ref('')
const COLLECTOR_SERVER_URL_KEY = 'opshub_log_collector_server_url'

const normalizedPaths = computed(() => lines(pathText.value))
const hostsWithoutAgent = computed(() => selectedHostIds.value.filter(id => !targetOptions.hosts.find(item => item.id === id)?.agentId).length)
const parserText = computed(() => ({ raw: '原始文本', json: 'JSON 结构化', regex: '正则提取' }[form.parser.type] || form.parser.type))
const parserHelpText = computed(() => ({
  raw: '保留整行原文，从独立的 FATAL / ERROR / WARN / INFO / DEBUG / TRACE 标记中识别级别。',
  json: '解析 JSON 字段，提取正文、时间、级别以及其他结构化属性。',
  regex: '使用 Go 正则命名分组提取正文、时间、级别和 Trace 上下文。',
}[form.parser.type] || ''))
const multilineHelpText = computed(() => ({
  java: '识别 at、Caused by、Suppressed 和缩进堆栈行。',
  go: '识别 panic、goroutine、runtime 和 Go 调用栈。',
  python: '识别 Traceback、缩进栈帧和异常链。',
  custom: '用“新日志起始行正则”决定何时结束上一条多行日志。',
}[form.multiline.preset] || ''))
const sourceModeOptions = [{ label: '主机文件', value: 'host' }, { label: 'Kubernetes 容器', value: 'kubernetes' }]
const filteredWorkloads = computed(() => kubernetesOptions.workloads.filter(item => !selectedNamespaces.value.length || selectedNamespaces.value.includes(item.namespace)))
const selectedCluster = computed(() => targetOptions.clusters.find(item => item.id === selectedClusterId.value))
const kubernetesTargetSummary = computed(() => {
  if (!selectedClusterId.value) return '未选择集群'
  const clusterName = selectedCluster.value?.alias || selectedCluster.value?.name || `集群 ${selectedClusterId.value}`
  if (selectedWorkloads.value.length) return `${clusterName} · ${selectedWorkloads.value.length} 个 Workload`
  if (selectedNamespaces.value.length) return `${clusterName} · ${selectedNamespaces.value.length} 个 Namespace`
  return `${clusterName} · 全集群`
})
const collectorStatusDescription = computed(() => {
  if (collectorStatusError.value) return collectorStatusError.value
  if (!collectorStatus.value) return '正在读取集群采集状态'
  if (collectorStatus.value.lastError) return `采集器异常：${collectorStatus.value.lastError}`
  if (!collectorStatus.value.installed) return '尚未安装 DaemonSet，策略可以发布，但不会开始下发'
  if (!collectorStatus.value.desiredNodes) return 'DaemonSet 已创建，正在等待 Kubernetes 调度'
  if (collectorStatus.value.readyNodes < collectorStatus.value.desiredNodes) return `DaemonSet 正在启动，${collectorStatus.value.readyNodes}/${collectorStatus.value.desiredNodes} 个节点就绪`
  return `采集器已就绪，策略将在节点下一次配置拉取时自动应用`
})

let policyRefreshTimer: ReturnType<typeof setInterval> | undefined

const loadPolicies = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const rows = await getLogCollectionPolicies({
      keyword: keyword.value || undefined,
      status: policyView.value === 'archived' ? 'archived' : statusFilter.value || undefined,
    }) as any || []
    policies.value = rows.map((row: LogCollectionPolicy) => ({
      ...row,
      payload: normalizePolicyPayload(row.payload),
      targetHosts: Array.isArray(row.targetHosts) ? row.targetHosts : [],
      targetClusters: Array.isArray(row.targetClusters) ? row.targetClusters : [],
    }))
  } finally { loading.value = false }
}

const handlePolicyViewChange = () => {
  statusFilter.value = ''
  void loadPolicies()
}
const loadTargets = async () => {
  const [targets, profiles] = await Promise.all([getLogPolicyTargetOptions(), getLogRetentionPolicies()])
  Object.assign(targetOptions, targets as any)
  retentionProfiles.value = profiles as any || []
}
const loadCollectorStatus = async (clusterId: number) => {
  collectorStatusLoading.value = true
  collectorStatusError.value = ''
  try {
    collectorStatus.value = await getLogKubernetesCollectorStatus(clusterId) as any
  } catch (error: any) {
    collectorStatus.value = undefined
    collectorStatusError.value = error?.response?.data?.message || error?.message || '读取采集器状态失败'
  } finally {
    collectorStatusLoading.value = false
  }
}
const loadKubernetesOptions = async (clusterId?: number, resetSelection = true) => {
  const id = Number(clusterId || selectedClusterId.value)
  kubernetesOptions.namespaces = []; kubernetesOptions.workloads = []
  if (resetSelection) { selectedNamespaces.value = []; selectedWorkloads.value = [] }
  collectorStatus.value = undefined; collectorStatusError.value = ''
  if (!id) return
  kubernetesOptionsLoading.value = true
  await Promise.all([
    getLogKubernetesPolicyOptions(id)
      .then(result => Object.assign(kubernetesOptions, result as any))
      .catch((error: any) => ElMessage.warning(error?.response?.data?.message || error?.message || '读取 Kubernetes 资源范围失败'))
      .finally(() => { kubernetesOptionsLoading.value = false }),
    loadCollectorStatus(id),
  ])
}
const handleNamespaceChange = () => {
  if (!selectedNamespaces.value.length) return
  selectedWorkloads.value = selectedWorkloads.value.filter(value => selectedNamespaces.value.includes(value.split('|')[0] || ''))
}

const resetForm = () => {
  Object.assign(form, emptyForm())
  editingId.value = undefined; selectedHostIds.value = []; selectedGroupIds.value = []; selectedClusterId.value = undefined
  selectedNamespaces.value = []; selectedWorkloads.value = []; labelSelector.value = ''; containerIncludeText.value = ''; containerExcludeText.value = ''
  kubernetesOptions.namespaces = []; kubernetesOptions.workloads = []; collectorStatus.value = undefined; collectorStatusError.value = ''
  pathText.value = ''; excludePathText.value = '*.gz\n*.tmp'; changeSummary.value = ''; step.value = 0
}
const normalizePolicyPayload = (payload?: Partial<LogCollectionPolicyPayload>): LogCollectionPolicyPayload => {
  const defaults = emptyForm()
  const source = JSON.parse(JSON.stringify(payload || {})) as Partial<LogCollectionPolicyPayload>
  return {
    ...defaults,
    ...source,
    retentionPolicyId: Number(source.retentionPolicyId) || undefined,
    paths: Array.isArray(source.paths) ? source.paths : defaults.paths,
    excludePaths: Array.isArray(source.excludePaths) ? source.excludePaths : defaults.excludePaths,
    targets: Array.isArray(source.targets) ? source.targets : defaults.targets,
    parser: { ...defaults.parser, ...(source.parser || {}) },
    multiline: { ...defaults.multiline, ...(source.multiline || {}) },
    redaction: {
      ...defaults.redaction,
      ...(source.redaction || {}),
      sensitiveFields: Array.isArray(source.redaction?.sensitiveFields) ? source.redaction.sensitiveFields : defaults.redaction.sensitiveFields,
      rules: Array.isArray(source.redaction?.rules) ? source.redaction.rules : defaults.redaction.rules,
    },
    retention: {
      ...defaults.retention,
      ...(source.retention || {}),
      levelDays: { ...defaults.retention.levelDays, ...(source.retention?.levelDays || {}) },
    },
  }
}
const openCreate = async () => { resetForm(); dialogVisible.value = true; await loadTargets() }
const openCreateFromToolbar = async () => {
  if (policyView.value === 'archived') {
    policyView.value = 'current'
    statusFilter.value = ''
    void loadPolicies()
  }
  await openCreate()
}
const openEdit = async (row: LogCollectionPolicy) => {
  resetForm(); editingId.value = row.id; Object.assign(form, normalizePolicyPayload(row.draftPayload || row.payload)); dialogVisible.value = true
  pathText.value = form.paths.join('\n'); excludePathText.value = form.excludePaths.join('\n')
  await loadTargets()
  if (form.sourceMode === 'kubernetes') {
    const targets = form.targets.filter(item => item.targetType === 'cluster')
    selectedClusterId.value = targets[0]?.targetId
    selectedNamespaces.value = [...new Set(targets.map(item => item.namespace).filter(Boolean) as string[])]
    selectedWorkloads.value = targets.filter(item => item.workloadName).map(item => workloadValue({ namespace: item.namespace || '', kind: item.workloadKind || '', name: item.workloadName || '' }))
    labelSelector.value = targets[0]?.labelSelector || ''
    containerIncludeText.value = (targets[0]?.containerInclude || []).join('\n')
    containerExcludeText.value = (targets[0]?.containerExclude || []).join('\n')
    if (selectedClusterId.value) await loadKubernetesOptions(selectedClusterId.value, false)
  } else {
    selectedHostIds.value = form.targets.filter(item => item.targetType === 'host').map(item => item.targetId)
    selectedGroupIds.value = form.targets.filter(item => item.targetType === 'host_group').map(item => item.targetId)
  }
}
const openCollectorSetup = async (row: LogCollectionPolicy) => {
  const opening = openEdit(row)
  step.value = 1
  await opening
}
const applyRetentionProfile = (id?: string | number | boolean) => {
  const profile = retentionProfiles.value.find(item => item.id === Number(id))
  if (!profile) return
  if (!profile.payload.enabled) {
    ElMessage.warning('该保留策略已停用，请选择其他策略')
    form.retentionPolicyId = undefined
    return
  }
  form.retention.defaultDays = profile.payload.defaultDays
  form.retention.levelDays = {
    ...emptyForm().retention.levelDays,
    ...(profile.payload.levelDays || {}),
  }
}
const addRedactionRule = () => form.redaction.rules.push({ target: 'field', field: '', action: 'replace', replacement: '[REDACTED]' })
const handleSourceModeChange = (value: string | number | boolean) => {
  form.sourceMode = String(value)
  if (form.sourceMode === 'kubernetes') {
    pathText.value = '/var/log/containers/*.log'; excludePathText.value = '*.gz\n*.tmp'
  } else if (pathText.value === '/var/log/containers/*.log') {
    pathText.value = ''
    collectorStatus.value = undefined; collectorStatusError.value = ''
  }
}
const installSelectedClusterCollector = async () => {
  if (!selectedClusterId.value) return
  try {
    const serverUrl = await requestCollectorServerURL()
    if (!serverUrl) return
    await ElMessageBox.confirm(
      `${collectorStatus.value?.installed
        ? '升级会轮换集群采集 Token，并滚动更新所有节点采集 Pod。'
        : '将在目标集群创建只读 RBAC、ConfigMap、Secret 和日志采集 DaemonSet。'}\n\n采集器连接地址：${serverUrl}`,
      collectorStatus.value?.installed ? '升级日志采集器' : '安装日志采集器',
      { type: 'warning', confirmButtonText: '确认执行' },
    )
    collectorInstalling.value = true
    await installLogKubernetesCollector(selectedClusterId.value, { serverUrl })
    ElMessage.success(collectorStatus.value?.installed ? '采集器升级已提交' : '采集器安装已提交')
    await loadCollectorStatus(selectedClusterId.value)
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') throw error
  } finally {
    collectorInstalling.value = false
  }
}
const normalizeCollectorServerURL = (value: string) => value.trim().replace(/\/+$/, '')
const collectorServerURLValidationError = (value: string) => {
  try {
    const parsed = new URL(normalizeCollectorServerURL(value))
    if (!['http:', 'https:'].includes(parsed.protocol)) return '地址必须以 http:// 或 https:// 开头'
    if (['localhost', '127.0.0.1', '::1', '0.0.0.0'].includes(parsed.hostname)) {
      return '不能使用 localhost 或回环地址，请填写目标集群 Pod 能访问的 OpsHub 地址'
    }
    return ''
  } catch {
    return '请输入完整有效的 URL'
  }
}
const requestCollectorServerURL = async () => {
  const initialValue = collectorStatus.value?.serverUrl || localStorage.getItem(COLLECTOR_SERVER_URL_KEY) || window.location.origin
  const promptResult = await ElMessageBox.prompt(
    '该地址会写入目标集群的采集器配置。请填写集群 Pod 能访问且可请求 /api/v1/public/ 的 OpsHub 地址。',
    '配置采集器连接地址',
    {
      inputValue: initialValue,
      inputPlaceholder: '例如：https://opshub.example.com 或 http://10.0.0.8:9876',
      confirmButtonText: '继续',
      cancelButtonText: '取消',
      inputValidator: value => collectorServerURLValidationError(value) || true,
    },
  ) as any
  const normalized = normalizeCollectorServerURL(String(promptResult?.value || ''))
  localStorage.setItem(COLLECTOR_SERVER_URL_KEY, normalized)
  return normalized
}
const nextStep = async () => {
  if (step.value === 0 && !form.name.trim()) return ElMessage.warning('请输入策略名称')
  if (step.value === 0 && !String(form.environment || '').trim()) return ElMessage.warning('请选择运行环境')
  if (step.value === 0 && !String(form.service || '').trim()) return ElMessage.warning('请输入服务名称')
  if (step.value === 1 && form.sourceMode === 'host' && !selectedHostIds.value.length && !selectedGroupIds.value.length) return ElMessage.warning('至少选择一台主机或一个主机分组')
  if (step.value === 1 && form.sourceMode === 'kubernetes' && !selectedClusterId.value) return ElMessage.warning('请选择 Kubernetes 集群')
  if (step.value === 2 && !normalizedPaths.value.length) return ElMessage.warning('至少填写一个采集路径')
  if (step.value === 2 && form.parser.type === 'regex' && !String(form.parser.pattern || '').trim()) return ElMessage.warning('请填写日志解析正则')
  if (step.value === 2 && form.multiline.enabled && form.multiline.preset === 'custom' && !String(form.multiline.startPattern || '').trim()) return ElMessage.warning('请填写新日志起始行正则')
  if (step.value === 2 && form.redaction.enabled) {
    const invalidRuleIndex = form.redaction.rules.findIndex(rule => rule.target === 'regex' ? !String(rule.pattern || '').trim() : !String(rule.field || '').trim())
    if (invalidRuleIndex >= 0) return ElMessage.warning(`请补全第 ${invalidRuleIndex + 1} 条自定义脱敏规则`)
  }
  step.value++
}
const buildPayload = (): LogCollectionPolicyPayload => ({
  ...JSON.parse(JSON.stringify(form)), retentionDays: form.retention.defaultDays, paths: normalizedPaths.value, excludePaths: lines(excludePathText.value),
  targets: form.sourceMode === 'kubernetes' ? buildKubernetesTargets() : [
    ...selectedHostIds.value.map(targetId => ({ targetType: 'host', targetId })),
    ...selectedGroupIds.value.map(targetId => ({ targetType: 'host_group', targetId })),
  ],
})
const buildKubernetesTargets = () => {
  if (!selectedClusterId.value) return []
  const common = { targetType: 'cluster', targetId: selectedClusterId.value, labelSelector: labelSelector.value.trim(), containerInclude: assetPatterns(containerIncludeText.value), containerExclude: assetPatterns(containerExcludeText.value) }
  if (selectedWorkloads.value.length) return selectedWorkloads.value.map(value => {
    const [namespace, workloadKind, workloadName] = value.split('|')
    return { ...common, namespace, workloadKind, workloadName }
  })
  if (selectedNamespaces.value.length) return selectedNamespaces.value.map(namespace => ({ ...common, namespace }))
  return [common]
}
const savePolicy = async (publish: boolean) => {
  if (!normalizedPaths.value.length) return ElMessage.warning('至少填写一个采集路径')
  publish ? publishing.value = true : saving.value = true
  try {
    const saved = editingId.value ? await updateLogCollectionPolicy(editingId.value, buildPayload()) : await createLogCollectionPolicy(buildPayload())
    if (publish) await publishLogCollectionPolicy((saved as any).id, changeSummary.value || undefined)
    ElMessage.success(publish ? '策略已发布，Agent 将自动拉取' : '策略草稿已保存'); dialogVisible.value = false; await loadPolicies()
  } finally { saving.value = false; publishing.value = false }
}
const publishExisting = async (row: LogCollectionPolicy) => { await ElMessageBox.confirm(`确认发布策略“${row.payload.name}”吗？`, '发布策略', { type: 'warning' }); await publishLogCollectionPolicy(row.id); ElMessage.success('策略已发布'); await loadPolicies() }
const disableExisting = async (row: LogCollectionPolicy) => {
  let uninstallCollectors = false
  if (row.payload.sourceMode === 'kubernetes') {
    try {
      await ElMessageBox.confirm(
        '停用后节点采集器会移除该策略。你还可以同时关闭目标集群采集器；仅当集群没有其他已发布策略时，系统才会删除 DaemonSet。',
        `停用“${row.payload.name}”`,
        {
          type: 'warning',
          confirmButtonText: '停用并关闭采集器',
          cancelButtonText: '仅停用策略',
          distinguishCancelAndClose: true,
          closeOnClickModal: false,
        },
      )
      uninstallCollectors = true
    } catch (action) {
      if (action !== 'cancel') return
    }
  } else {
    await ElMessageBox.confirm(`停用后目标 Agent 将停止该策略，确认停用“${row.payload.name}”？`, '停用策略', { type: 'warning' })
  }
  const result = await disableLogCollectionPolicy(row.id, uninstallCollectors) as any as LogCollectionPolicy
  const shutdownResults = result.collectorShutdown || []
  const failed = shutdownResults.filter(item => item.status === 'failed')
  const skipped = shutdownResults.filter(item => item.status === 'skipped')
  const uninstalled = shutdownResults.filter(item => item.status === 'uninstalled')
  if (failed.length) {
    ElMessage.warning({ message: `策略已停用，但 ${failed.map(item => item.clusterName).join('、')} 的采集器关闭失败，请查看集群日志采集状态`, duration: 7000 })
  } else if (skipped.length) {
    ElMessage.warning({ message: `策略已停用；${skipped.map(item => item.clusterName).join('、')} 仍被其他策略使用，已保留 DaemonSet`, duration: 6000 })
  } else if (uninstalled.length) {
    ElMessage.success(`策略已停用，${uninstalled.map(item => item.clusterName).join('、')} 的采集器已关闭`)
  } else {
    ElMessage.success('策略已停用')
  }
  await loadPolicies()
}
const removePolicy = async (row: LogCollectionPolicy) => {
  const archived = row.status === 'disabled'
  await ElMessageBox.confirm(
    archived
      ? `归档后策略会移入“已归档”视图并停止下发，发布版本、目标和审计记录会保留，之后仍可恢复。确认归档“${row.payload.name}”？`
      : `确认永久删除未发布草稿“${row.payload.name}”？该草稿没有发布版本。`,
    archived ? '归档策略' : '删除策略',
    { type: 'warning', confirmButtonText: archived ? '确认归档' : '确认删除', cancelButtonText: '取消' },
  )
  const result = await deleteLogCollectionPolicy(row.id) as any
  if (result?.action === 'archived') {
    policyView.value = 'archived'
    statusFilter.value = ''
    ElMessage.success('策略已移入归档，可在已归档视图恢复')
  } else {
    ElMessage.success('策略已删除')
  }
  await loadPolicies()
}
const restorePolicy = async (row: LogCollectionPolicy) => {
  await ElMessageBox.confirm(`恢复后“${row.payload.name}”会回到草稿状态，需要重新发布才会采集。确认恢复吗？`, '恢复策略', { type: 'info', confirmButtonText: '确认恢复', cancelButtonText: '取消' })
  await restoreLogCollectionPolicy(row.id)
  policyView.value = 'current'
  statusFilter.value = 'draft'
  ElMessage.success('策略已恢复为草稿')
  await loadPolicies()
}

const lines = (value: string) => value.split(/\r?\n/).map(item => item.trim()).filter(Boolean)
const assetPatterns = (value: string) => value.split(/[\r\n,;]+/).map(item => item.trim()).filter(Boolean)
const workloadValue = (item: LogKubernetesWorkloadOption) => `${item.namespace}|${item.kind}|${item.name}`
const rolloutExpected = (row: LogCollectionPolicy) => Number(row.targetExpected || (row.payload.sourceMode === 'kubernetes'
  ? (row.targetClusters || []).reduce((total, cluster) => total + Number(cluster.nodeCount || 0), 0)
  : (row.targetHosts || []).length))
const rolloutTotal = (row: LogCollectionPolicy) => Math.max(row.instanceTotal, rolloutExpected(row))
const rolloutPercent = (row: LogCollectionPolicy) => row.status === 'published' && rolloutTotal(row) ? Math.round(row.instanceApplied / rolloutTotal(row) * 100) : 0
const rolloutPending = (row: LogCollectionPolicy) => row.status === 'published' ? Math.max(rolloutExpected(row) - row.instanceApplied - row.errorInstances, 0) : 0
const rolloutWaiting = (row: LogCollectionPolicy) => row.status === 'published' && rolloutPending(row) > 0
const needsCollectorSetup = (row: LogCollectionPolicy) => row.payload.sourceMode === 'kubernetes' && row.status === 'published' && row.instanceTotal === 0 && rolloutExpected(row) > 0
const rolloutText = (row: LogCollectionPolicy) => {
  if (row.status === 'draft') return '尚未下发'
  if (row.status === 'disabled') return '已停止下发'
  if (row.status === 'archived') return '已归档'
  return `${row.instanceApplied}/${rolloutTotal(row)} 已应用`
}
const rolloutSubtext = (row: LogCollectionPolicy) => {
  if (row.status === 'draft') return '发布策略后开始下发'
  if (row.status === 'disabled') return '采集实例将移除该策略'
  if (row.status === 'archived') return '仅保留历史版本与审计记录'
  return `已接入 ${row.instanceTotal || 0} · 在线 ${row.instanceOnline || 0} · 待确认 ${rolloutPending(row)}`
}
const rolloutTargetLabel = (row: LogCollectionPolicy) => row.payload.sourceMode === 'kubernetes' ? '目标节点' : '目标主机'
const openInstances = (row: LogCollectionPolicy) => emit('open-instances', row.id)
const statusText = (status: string) => ({ draft: '草稿', published: '已发布', disabled: '已停用', archived: '已归档' }[status] || status)
const statusType = (status: string) => status === 'published' ? 'success' : status === 'disabled' || status === 'archived' ? 'info' : 'warning'
const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'
const formatBytes = (value: number) => value >= 1073741824 ? `${(value / 1073741824).toFixed(value % 1073741824 ? 1 : 0)} GiB` : `${Math.round(value / 1048576)} MiB`

onMounted(async () => {
  await loadPolicies()
  policyRefreshTimer = setInterval(() => loadPolicies(true), 10000)
  if (props.presetHostId && isAdmin.value) {
    await openCreate()
    selectedHostIds.value = [props.presetHostId]
  } else if (props.presetClusterId && isAdmin.value) {
    await openCreate()
    handleSourceModeChange('kubernetes')
    selectedClusterId.value = props.presetClusterId
    await loadKubernetesOptions(props.presetClusterId)
  }
})
onBeforeUnmount(() => policyRefreshTimer && clearInterval(policyRefreshTimer))
</script>

<style scoped>
.policy-page { display: flex; flex-direction: column; gap: 12px; }
.section-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; margin: 0; padding: 18px 20px; }
.section-head h3 { margin: 0; color: #111827; font-size: 15px; font-weight: 650; }
.section-head p { min-height: 20px; margin: 6px 0 0; color: #667085; font-size: 13px; line-height: 20px; }
.section-actions { display: grid; grid-template-columns: 176px 190px 126px 88px 124px; align-items: center; gap: 8px; }
.section-actions .el-input,.section-actions .el-select,.section-actions .el-button,.section-actions :deep(.el-segmented) { width: 100%; }
.policy-table { overflow: hidden; }.policy-name strong,.policy-name small { display: block; }.policy-name strong { color: #111827; }.policy-name small { margin-top: 5px; color: #98a2b3; font-size: 12px; }
.policy-status { display: flex; flex-wrap: wrap; gap: 4px; }
.column-heading { display:inline-flex;align-items:center;gap:5px; }.column-heading .el-icon { color:#98a2b3;cursor:help;font-size:14px; }
.rollout-cell { min-width:0;cursor:help;color:#667085;font-size:12px; }
.rollout-line { display:flex;align-items:center;gap:8px;min-width:0; }.rollout-line :deep(.el-progress) { width:88px;flex:0 0 auto; }.rollout-line strong { color:#344054;font-size:12px;white-space:nowrap; }
.rollout-cell>small { display:block;margin-top:6px;overflow:hidden;color:#98a2b3;text-overflow:ellipsis;white-space:nowrap; }
.rollout-detail-head { display:flex;align-items:center;justify-content:space-between;gap:12px;padding-bottom:10px;border-bottom:1px solid #eef0f3; }.rollout-detail-head strong { color:#101828;font-size:13px; }
.rollout-detail dl { display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin:12px 0; }.rollout-detail dl div { padding:8px 9px;background:#f8fafc; }.rollout-detail dt { color:#98a2b3;font-size:11px; }.rollout-detail dd { margin:4px 0 0;color:#344054;font-size:15px;font-weight:700; }.rollout-detail dd.success { color:#15803d; }.rollout-detail dd.warning { color:#b45309; }.rollout-detail dd.danger { color:#dc2626; }
.rollout-detail p { margin:0 0 4px;color:#667085;font-size:12px;line-height:1.55; }
code { color: #344054; font: 12px/1.5 ui-monospace,SFMono-Regular,Menlo,monospace; }
.policy-steps { margin: 4px 0 24px; }.policy-form { min-height: 400px; }.form-stage { padding: 4px 8px; }.two-column { display: grid; grid-template-columns: 1fr 1fr; gap: 0 20px; }.full,.wide { grid-column: 1/-1; }
.target-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-top: 20px; }.target-grid .el-select { width: 100%; }.host-option { display:flex;align-items:center;justify-content:space-between;gap:16px;width:100%;}.host-option small { color:#98a2b3; }
.target-summary { display:flex;justify-content:space-between;gap:16px;padding:14px 16px;border:1px solid #eaecf0;background:#f9fafb;color:#475467;font-size:13px; }.warning-text { color:#b54708; }
.collector-readiness { display:flex;align-items:center;gap:18px;margin-bottom:18px;padding:14px 16px;border:1px solid #fed7aa;border-radius:7px;background:#fffaf3; }
.collector-readiness.ready { border-color:#bbf7d0;background:#f7fef9; }.collector-readiness.error { border-color:#fecaca;background:#fff8f8; }
.collector-readiness-main { display:flex;align-items:center;gap:11px;min-width:240px;flex:1; }.collector-readiness-main>i { width:9px;height:9px;flex:0 0 auto;border-radius:50%;background:#f59e0b;box-shadow:0 0 0 4px #fef3c7; }
.collector-readiness.ready .collector-readiness-main>i { background:#22c55e;box-shadow:0 0 0 4px #dcfce7; }.collector-readiness.error .collector-readiness-main>i { background:#ef4444;box-shadow:0 0 0 4px #fee2e2; }
.collector-readiness-main strong,.collector-readiness-main small { display:block; }.collector-readiness-main strong { color:#1f2937;font-size:13px; }.collector-readiness-main small { margin-top:4px;color:#667085;font-size:12px;line-height:1.45; }
.collector-readiness-metrics { display:flex;align-items:center;gap:16px;color:#667085;font-size:12px;white-space:nowrap; }.collector-readiness-metrics strong { color:#1f2937; }
.field-tip { width:100%;margin-top:6px;color:#98a2b3;font-size:12px;line-height:1.55; }.required-label :deep(.el-form-item__label)::before { content:'*'; margin-right:4px; color:#f04438; }
.form-stage :deep(.el-select),.form-stage :deep(.el-input-number) { width:100%; }
.number-with-unit { display:flex;align-items:center;gap:8px;width:100%;color:#667085;font-size:12px; }.number-with-unit :deep(.el-input-number) { flex:1;min-width:0; }
.retention-config { margin-bottom:18px;border:1px solid #e4e7ec;background:#fbfcfe; }
.retention-toolbar { display:grid;grid-template-columns:1fr 1fr;gap:20px;padding:14px 16px 0;background:#fff; }.retention-toolbar :deep(.el-form-item) { margin-bottom:14px; }.retention-toolbar :deep(.el-select),.retention-toolbar :deep(.el-input-number) { width:100%; }
.retention-profile-alert { margin:0 16px 12px;width:auto; }
.retention-title { display:flex;align-items:center;gap:8px;padding:12px 16px 10px;border-top:1px solid #eef0f3;color:#344054;font-size:13px; }.retention-title>span { color:#98a2b3;font-size:12px;font-weight:400; }.retention-title .el-icon { color:#98a2b3;cursor:help; }
.retention-level-grid { display:grid;grid-template-columns:repeat(3,1fr);gap:1px;border-top:1px solid #eef0f3;background:#e8ecf1; }.retention-level-grid label { display:grid;grid-template-columns:62px minmax(62px,1fr) minmax(112px,1.25fr);align-items:center;gap:8px;min-width:0;padding:12px;background:#fff;color:#667085;font-size:12px; }.retention-level-grid label>small { overflow:hidden;text-overflow:ellipsis;white-space:nowrap; }
.level-name { display:inline-flex;align-items:center;justify-content:center;height:24px;border:1px solid transparent;font-size:11px;font-weight:700; }.level-trace { color:#475467;background:#f2f4f7;border-color:#e4e7ec; }.level-debug { color:#0369a1;background:#f0f9ff;border-color:#bae6fd; }.level-info { color:#047857;background:#ecfdf5;border-color:#a7f3d0; }.level-warn { color:#b45309;background:#fffbeb;border-color:#fde68a; }.level-error { color:#c2410c;background:#fff7ed;border-color:#fed7aa; }.level-fatal { color:#be123c;background:#fff1f2;border-color:#fecdd3; }
.days-control { display:flex;align-items:center;gap:7px;min-width:0;color:#98a2b3; }.days-control :deep(.el-input-number) { width:100%;min-width:0; }
.security-config { padding:16px;border:1px solid #eaecf0;border-radius:7px;background:#fff; }.security-head { display:flex;align-items:center;justify-content:space-between;gap:20px; }.security-head strong,.security-head span { display:block; }.security-head strong { color:#101828;font-size:14px; }.security-head span { margin-top:5px;color:#667085;font-size:12px;line-height:1.55; }.security-options { display:grid;grid-template-columns:1fr;gap:14px;margin-top:16px;padding-top:14px;border-top:1px solid #edf0f4; }.security-options .el-select { width:100%; }.sensitive-field-setting { display:grid;grid-template-columns:120px minmax(0,1fr);align-items:center;gap:8px 14px; }.sensitive-field-setting>span { color:#344054;font-size:13px;font-weight:600; }.sensitive-field-setting>small { grid-column:2;color:#98a2b3;font-size:12px;line-height:1.55; }.redaction-rules { margin-top:16px;padding-top:14px;border-top:1px solid #edf0f4; }.rules-head { display:flex;align-items:center;justify-content:space-between;color:#344054;font-size:13px;font-weight:600; }.rules-help { margin:4px 0 10px;color:#98a2b3;font-size:12px;line-height:1.55; }.rule-row { display:grid;grid-template-columns:120px minmax(180px,1fr) 120px minmax(140px,.7fr) 44px;gap:8px;align-items:center;margin-top:8px; }
.review-stage { display:grid;grid-template-columns:1fr 1fr;gap:14px; }.review-card { padding:16px;border:1px solid #eaecf0;border-radius:7px;background:#fff; }.review-card span,.review-card strong,.review-card small,.review-card code { display:block; }.review-card span { color:#98a2b3;font-size:12px; }.review-card strong { margin-top:8px;color:#101828;font-size:15px; }.review-card small { margin-top:5px;color:#667085; }.review-card code { margin-top:8px;padding:5px 8px;background:#f8fafc; }
.dialog-footer { display:flex;align-items:center;width:100%;}.footer-spacer { flex:1; }
.readonly-text { color:#98a2b3;font-size:12px; }
@media (max-width: 1280px) { .section-head { flex-direction:column; }.section-actions { width:100%; } }
@media (max-width: 900px) { .section-actions { grid-template-columns:176px minmax(180px,1fr) 126px 88px 124px;overflow-x:auto;padding-bottom:2px; }.two-column,.target-grid,.review-stage,.retention-toolbar { grid-template-columns:1fr; }.full,.wide { grid-column:auto; }.retention-title { align-items:flex-start;flex-wrap:wrap; }.retention-title>span { width:100%; }.retention-level-grid { grid-template-columns:1fr; }.sensitive-field-setting { grid-template-columns:1fr; }.sensitive-field-setting>small { grid-column:auto; }.rule-row { grid-template-columns:1fr 1fr; }.rule-row .el-button { justify-self:end; }.collector-readiness { align-items:flex-start;flex-wrap:wrap; }.collector-readiness-metrics { order:3;width:100%; } }
</style>
