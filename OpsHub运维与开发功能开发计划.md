# OpsHub 运维与开发功能开发计划

编写日期：2026-06-30  
计划周期：2026-07-01 至 2026-12-31  
范围说明：本计划聚焦 OpsHub 作为“运行态运维与可观测平台”的能力建设，不包含 CI/CD 发布流水线、构建流水线、代码仓库管理、制品仓库管理等能力。CI/CD 仅作为外部系统通过 Webhook 上报变更事件，不在 OpsHub 内执行发布。

## 1. 当前能力基线

OpsHub 已经具备以下基础模块：

| 模块 | 已有能力 | 关键表/资源 |
| --- | --- | --- |
| 资产管理 | 主机、凭据、业务分组、云账号、Agent、终端、终端审计、资产权限 | `hosts`、`credentials`、`asset_group`、`cloud_accounts`、`ssh_terminal_sessions`、`sys_role_asset_permission` |
| 容器管理 | K8s 集群、节点、命名空间、工作负载、网络、配置、存储、访问控制、终端审计、巡检、应用诊断 | `k8s_clusters`、`k8s_user_kube_configs`、`k8s_user_role_bindings`、`k8s_cluster_inspections`、`k8s_terminal_sessions` |
| 监控中心 | 数据源、故障中心、告警规则、告警事件、拨测任务、即时拨测、通知对象、通知模板、值班表 | `monitor_datasources`、`monitor_fault_centers`、`monitor_rule_groups`、`monitor_alert_rules`、`monitor_alert_events`、`monitor_probe_tasks`、`monitor_notice_objects`、`monitor_notice_templates`、`monitor_duty_tables`、`monitor_duty_schedules` |
| SSL 证书 | DNS 提供商、证书管理、部署配置、续期任务 | `ssl_certificates`、`ssl_dns_providers`、`ssl_deploy_configs`、`ssl_renew_tasks` |
| Nginx 统计 | 日志源、访问明细、小时/日聚合、多维统计 | `nginx_sources`、`nginx_fact_access_logs`、`nginx_dim_ip`、`nginx_dim_url`、`nginx_dim_referer`、`nginx_dim_user_agent`、`nginx_agg_hourly`、`nginx_agg_daily` |
| 任务中心 | 脚本任务、任务模板、文件分发、执行历史 | `job_templates`、`job_tasks`、`ansible_tasks` |
| AIOps | AI 助手、智能诊断、日志分析、告警分析、会话记录、AI 配置 | `ai_providers`、`ai_sessions`、`ai_messages`、`ai_tool_calls`、`ai_diagnosis_tasks`、`ai_root_cause_analyses` |
| 审计与系统 | 用户、角色、菜单、部门、岗位、系统配置、操作日志、登录日志、MFA | `sys_user`、`sys_role`、`sys_menu`、`sys_role_menu`、`sys_department`、`sys_position`、`sys_config`、`sys_operation_log`、`sys_login_log`、`sys_data_log`、`user_mfa`、`sys_mfa_log` |

## 2. 总体建设目标

OpsHub 后续重点不是继续堆工具菜单，而是建立围绕“业务应用”的运行态闭环：

1. 应用中心：把主机、K8s 工作负载、域名、证书、告警、日志、负责人统一挂到应用上。
2. 事件中心：把告警升级为可跟踪、可复盘、可统计的 Incident。
3. 变更中心：接收外部 CI/CD、人工发布、K8s rollout、配置变更，只做记录和关联分析，不做发布。
4. Runbook：沉淀标准处置流程，支持关联告警、应用、任务模板和 AIOps 推荐。
5. 应用观测：从开发视角查看应用指标、日志、事件、错误和最近变更。
6. 拓扑与容量：展示应用依赖关系、资源关系和容量趋势。
7. 基线巡检与备份视图：补齐运维日常检查、安全基线、备份状态和通知健康。

## 3. 菜单规划

### 3.1 新增一级菜单

| 一级菜单 | 路径 | 说明 |
| --- | --- | --- |
| 应用中心 | `/applications` | 面向开发和运维的业务应用入口 |
| 事件中心 | `/incidents` | 告警事件升级后的故障协同、处置和复盘中心 |
| 变更中心 | `/changes` | 记录外部发布、配置、扩缩容、K8s rollout 等变更事件 |
| 运维知识库 | `/runbooks` | Runbook、处置步骤、命令模板、执行记录 |
| 健康巡检 | `/health-checks` | 主机/K8s/应用基线巡检、容量趋势、备份状态 |

### 3.2 新增/调整子菜单

| 模块 | 子菜单 | 路径 | 主要用户 |
| --- | --- | --- | --- |
| 应用中心 | 应用列表 | `/applications/services` | 开发、运维 |
| 应用中心 | 应用详情 | `/applications/services/:id` | 开发、运维 |
| 应用中心 | 应用拓扑 | `/applications/topology` | 运维、架构 |
| 应用中心 | 应用观测 | `/applications/observability` | 开发、运维 |
| 应用中心 | 依赖管理 | `/applications/dependencies` | 开发、运维 |
| 事件中心 | 活跃事件 | `/incidents/active` | 值班人员 |
| 事件中心 | 历史事件 | `/incidents/history` | 运维、开发 |
| 事件中心 | 事件复盘 | `/incidents/reviews` | 运维、开发、管理者 |
| 事件中心 | 改进项 | `/incidents/actions` | 负责人、管理者 |
| 变更中心 | 变更列表 | `/changes/events` | 开发、运维 |
| 变更中心 | 接入源 | `/changes/sources` | 管理员 |
| 变更中心 | Webhook 配置 | `/changes/webhooks` | 管理员 |
| 运维知识库 | Runbook 列表 | `/runbooks/list` | 运维、开发 |
| 运维知识库 | Runbook 执行 | `/runbooks/executions` | 值班人员 |
| 运维知识库 | 命令模板 | `/runbooks/commands` | 运维 |
| 健康巡检 | 主机基线 | `/health-checks/hosts` | 运维 |
| 健康巡检 | K8s 基线 | `/health-checks/kubernetes` | 运维 |
| 健康巡检 | 容量趋势 | `/health-checks/capacity` | 运维、管理者 |
| 健康巡检 | 备份状态 | `/health-checks/backups` | 运维 |
| 监控中心 | 通知健康 | `/monitor/notification-health` | 运维 |

## 4. 阶段开发计划

### 阶段 0：技术准备与数据模型设计

时间：2026-07-01 至 2026-07-05  
目标：统一后续模块的命名、权限、菜单、迁移和基础 API 风格，避免后续重复返工。

#### 开发内容

| 内容 | 说明 |
| --- | --- |
| 模块命名规范 | 新增模块采用 `application`、`incident`、`change`、`runbook`、`health_check` 前缀 |
| 菜单初始化 | 在 `cmd/server/server.go` 或独立迁移中新增菜单初始化逻辑 |
| 权限模型 | 每个新模块补齐查看、新增、编辑、删除、执行、导出等权限点 |
| 迁移策略 | 每个阶段单独提供 SQL migration，并同步 GORM model |
| 审计规范 | 所有新增、编辑、删除、执行、Webhook 调用写入操作审计 |

#### 涉及菜单

- 暂不开放新菜单，只新增隐藏菜单或仅在测试环境开放。

#### 新增表

| 表名 | 用途 |
| --- | --- |
| `module_feature_flags` | 控制新模块灰度开关，可选实现 |
| `platform_object_tags` | 统一标签表，可关联应用、主机、事件、Runbook |
| `platform_object_relations` | 通用对象关系表，可选，用于跨模块关联 |

#### 实现方式

1. 后端新增 `internal/biz/application`、`internal/data/application`、`internal/server/application` 目录骨架。
2. 同步建立 `incident`、`change`、`runbook`、`healthcheck` 目录骨架。
3. 新增 migrations，例如 `migrations/20260701_application_base.sql`。
4. 前端新增 API 文件和空路由页面，先只显示空状态。
5. 为 admin 自动授权新增菜单。

#### 验收标准

- 新增模块菜单可按开关显示。
- 后端启动可自动迁移新表。
- admin 可以访问空页面。
- 普通用户无权限时不可访问。

---

### 阶段 1：应用中心一期 - 服务目录与资源绑定

时间：2026-07-06 至 2026-07-24  
目标：建立“业务应用”这个核心对象，把已有资源挂到应用上。

#### 开发内容

| 内容 | 怎么开发 |
| --- | --- |
| 应用列表 | 支持按名称、编码、环境、负责人、状态、标签筛选 |
| 应用创建/编辑 | 支持应用名称、编码、环境、等级、负责人、研发负责人、运维负责人、描述、标签 |
| 资源绑定 | 应用可绑定主机、K8s 集群、Namespace、Deployment、StatefulSet、Service、Ingress、域名、SSL 证书、监控规则、故障中心 |
| 应用详情 | 展示基本信息、负责人、关联资源、最近告警、最近变更、证书状态 |
| 权限控制 | 开发只能看自己负责或被授权的应用，运维可看全部 |

#### 涉及菜单

| 菜单 | 路径 |
| --- | --- |
| 应用中心 / 应用列表 | `/applications/services` |
| 应用中心 / 应用详情 | `/applications/services/:id` |

#### 新增表

| 表名 | 关键字段 | 说明 |
| --- | --- | --- |
| `app_services` | `id`、`name`、`code`、`env`、`tier`、`status`、`owner_user_id`、`dev_owner_id`、`ops_owner_id`、`description`、`tags` | 应用主表 |
| `app_service_members` | `service_id`、`user_id`、`role` | 应用成员，如 owner/dev/ops/viewer |
| `app_service_resources` | `service_id`、`resource_type`、`resource_id`、`resource_name`、`external_key`、`cluster_id`、`namespace`、`resource_role` | 应用与资源关联 |
| `app_service_domains` | `service_id`、`domain`、`protocol`、`url`、`ssl_certificate_id`、`probe_task_id` | 应用域名入口 |
| `app_service_labels` | `service_id`、`label_key`、`label_value` | 应用标签，便于筛选 |

#### 复用表

- `hosts`
- `asset_group`
- `k8s_clusters`
- `ssl_certificates`
- `monitor_alert_rules`
- `monitor_fault_centers`
- `monitor_probe_tasks`

#### API 设计

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/applications/services` | 应用列表 |
| `POST` | `/api/v1/applications/services` | 创建应用 |
| `GET` | `/api/v1/applications/services/:id` | 应用详情 |
| `PUT` | `/api/v1/applications/services/:id` | 更新应用 |
| `DELETE` | `/api/v1/applications/services/:id` | 删除应用 |
| `POST` | `/api/v1/applications/services/:id/resources` | 绑定资源 |
| `DELETE` | `/api/v1/applications/services/:id/resources/:resourceId` | 解绑资源 |
| `GET` | `/api/v1/applications/services/:id/summary` | 应用运行摘要 |

#### 前端页面

1. `web/src/views/application/Services.vue`
2. `web/src/views/application/ServiceDetail.vue`
3. `web/src/api/application.ts`

#### 验收标准

- 可以创建应用并绑定至少以下资源：主机、K8s 工作负载、告警规则、故障中心、证书。
- 应用详情能看到最近 10 条告警、最近 10 条变更、证书到期状态。
- 数据权限可限制开发用户只看到自己负责的应用。

---

### 阶段 2：应用中心二期 - 依赖管理与应用拓扑

时间：2026-07-27 至 2026-08-14  
目标：展示应用之间、应用与基础设施之间的关系。

#### 开发内容

| 内容 | 怎么开发 |
| --- | --- |
| 应用依赖 | 支持配置依赖服务、数据库、Redis、外部 API、MQ、对象存储 |
| 拓扑图 | 使用应用资源关系和依赖关系生成节点与边 |
| 影响分析 | 某应用故障时展示上游/下游影响范围 |
| 自动关联 | 根据 Ingress、Service、域名、告警 labels 自动提示可关联资源 |

#### 涉及菜单

| 菜单 | 路径 |
| --- | --- |
| 应用中心 / 应用拓扑 | `/applications/topology` |
| 应用中心 / 依赖管理 | `/applications/dependencies` |

#### 新增表

| 表名 | 关键字段 | 说明 |
| --- | --- | --- |
| `app_service_dependencies` | `service_id`、`target_service_id`、`dependency_type`、`name`、`endpoint`、`criticality`、`description` | 应用依赖 |
| `app_topology_snapshots` | `service_id`、`snapshot_time`、`nodes_json`、`edges_json` | 拓扑快照，便于历史对比 |
| `app_resource_discovery_rules` | `service_id`、`rule_type`、`matcher`、`enabled` | 自动发现规则 |

#### API 设计

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/applications/topology` | 全局拓扑 |
| `GET` | `/api/v1/applications/services/:id/topology` | 单应用拓扑 |
| `POST` | `/api/v1/applications/services/:id/dependencies` | 新增依赖 |
| `PUT` | `/api/v1/applications/dependencies/:id` | 更新依赖 |
| `DELETE` | `/api/v1/applications/dependencies/:id` | 删除依赖 |
| `POST` | `/api/v1/applications/services/:id/discovery` | 自动发现关联资源 |

#### 验收标准

- 应用详情可展示拓扑图。
- 拓扑节点至少包含应用、主机、K8s 工作负载、域名、证书、告警规则。
- 点击拓扑节点可跳转到对应资源详情。

---

### 阶段 3：变更中心一期 - 外部变更事件接入

时间：2026-08-17 至 2026-09-04  
目标：不做 CI/CD，但接入外部系统变更记录，用于告警排查和故障分析。

#### 开发内容

| 内容 | 怎么开发 |
| --- | --- |
| 变更源管理 | 配置 Jenkins、GitLab、ArgoCD、K8s、人工变更、脚本任务等来源 |
| Webhook Token | 为每个变更源生成独立 token，支持启停和过期 |
| 变更事件接入 | 通过 Webhook 接收变更事件，写入变更中心 |
| 手工登记变更 | 支持运维手动录入配置变更、扩缩容、数据库变更 |
| 变更关联应用 | 根据应用编码、标签、namespace、workload、host 自动关联应用 |

#### 涉及菜单

| 菜单 | 路径 |
| --- | --- |
| 变更中心 / 变更列表 | `/changes/events` |
| 变更中心 / 接入源 | `/changes/sources` |
| 变更中心 / Webhook 配置 | `/changes/webhooks` |

#### 新增表

| 表名 | 关键字段 | 说明 |
| --- | --- | --- |
| `change_sources` | `id`、`name`、`source_type`、`enabled`、`description` | 变更来源 |
| `change_webhook_tokens` | `source_id`、`token_hash`、`secret`、`enabled`、`expired_at`、`last_used_at` | Webhook 凭证 |
| `change_events` | `id`、`source_id`、`event_type`、`title`、`status`、`operator`、`commit_id`、`branch`、`version`、`started_at`、`finished_at`、`raw_payload` | 变更事件 |
| `change_event_resources` | `change_event_id`、`resource_type`、`resource_id`、`resource_name`、`service_id` | 变更关联资源 |
| `change_event_tags` | `change_event_id`、`tag_key`、`tag_value` | 变更标签 |

#### API 设计

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/changes/events` | 变更列表 |
| `POST` | `/api/v1/changes/events` | 手工登记变更 |
| `GET` | `/api/v1/changes/events/:id` | 变更详情 |
| `POST` | `/api/v1/changes/webhook/:token` | 外部系统上报变更 |
| `GET` | `/api/v1/changes/sources` | 变更源列表 |
| `POST` | `/api/v1/changes/sources` | 创建变更源 |

#### Webhook 示例字段

```json
{
  "eventType": "deploy",
  "title": "platform-website-biz-prod 发布",
  "serviceCode": "platform-website-biz",
  "environment": "prod",
  "operator": "dujie",
  "version": "v2026.08.18-001",
  "commitId": "abc123",
  "status": "success",
  "startedAt": "2026-08-18 10:00:00",
  "finishedAt": "2026-08-18 10:05:00",
  "resources": [
    {"type": "k8s_deployment", "namespace": "prod", "name": "platform-website-biz-prod"}
  ]
}
```

#### 验收标准

- 外部系统可通过 Webhook 上报变更事件。
- 应用详情页能展示最近变更。
- 告警事件详情能展示告警前后 30 分钟相关变更。

---

### 阶段 4：事件中心一期 - Incident 生命周期

时间：2026-09-07 至 2026-09-25  
目标：将监控告警升级为可管理的故障事件。

#### 开发内容

| 内容 | 怎么开发 |
| --- | --- |
| 事件创建 | 告警事件可手动/自动创建 Incident |
| 事件认领 | 支持认领、转派、升级、恢复、关闭 |
| 事件时间线 | 自动记录告警、通知、认领、静默、升级、评论、恢复 |
| 影响范围 | 关联应用、资源、变更和故障中心 |
| 协同评论 | 支持处理备注、Markdown、附件链接 |
| SLA/SLO 统计 | 按 P0/P1/P2 统计确认时长、恢复时长 |

#### 涉及菜单

| 菜单 | 路径 |
| --- | --- |
| 事件中心 / 活跃事件 | `/incidents/active` |
| 事件中心 / 历史事件 | `/incidents/history` |

#### 新增表

| 表名 | 关键字段 | 说明 |
| --- | --- | --- |
| `incidents` | `id`、`title`、`severity`、`status`、`source_type`、`service_id`、`fault_center_id`、`owner_user_id`、`acknowledged_at`、`recovered_at`、`closed_at`、`impact`、`summary` | 故障事件主表 |
| `incident_alert_events` | `incident_id`、`alert_event_id`、`relation_type` | 关联监控告警事件 |
| `incident_resources` | `incident_id`、`resource_type`、`resource_id`、`resource_name` | 影响资源 |
| `incident_timeline` | `incident_id`、`event_type`、`operator_id`、`content`、`created_at` | 事件时间线 |
| `incident_comments` | `incident_id`、`user_id`、`content`、`created_at` | 协同评论 |

#### 复用表

- `monitor_alert_events`
- `monitor_fault_centers`
- `monitor_duty_tables`
- `change_events`
- `app_services`

#### API 设计

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/incidents` | 事件列表 |
| `POST` | `/api/v1/incidents` | 创建事件 |
| `GET` | `/api/v1/incidents/:id` | 事件详情 |
| `POST` | `/api/v1/incidents/:id/ack` | 认领事件 |
| `POST` | `/api/v1/incidents/:id/assign` | 转派事件 |
| `POST` | `/api/v1/incidents/:id/recover` | 标记恢复 |
| `POST` | `/api/v1/incidents/:id/close` | 关闭事件 |
| `POST` | `/api/v1/incidents/:id/comments` | 添加评论 |

#### 验收标准

- 活跃告警可一键创建 Incident。
- P0/P1 告警可按故障中心策略自动创建 Incident。
- Incident 详情能看到关联告警、应用、资源、变更、时间线。

---

### 阶段 5：事件中心二期 - 复盘与改进项

时间：2026-09-28 至 2026-10-16  
目标：补齐故障复盘、改进项跟踪和统计分析。

#### 开发内容

| 内容 | 怎么开发 |
| --- | --- |
| 复盘模板 | 支持影响范围、时间线、根因、处置过程、改进项 |
| 改进项 | 支持负责人、截止时间、状态、优先级、验收说明 |
| 事件导出 | 支持导出复盘 Markdown/PDF |
| 统计看板 | 按应用、团队、等级统计故障数量、MTTA、MTTR |
| AIOps 辅助 | 根据告警、日志、变更生成复盘草稿 |

#### 涉及菜单

| 菜单 | 路径 |
| --- | --- |
| 事件中心 / 事件复盘 | `/incidents/reviews` |
| 事件中心 / 改进项 | `/incidents/actions` |

#### 新增表

| 表名 | 关键字段 | 说明 |
| --- | --- | --- |
| `incident_reviews` | `incident_id`、`root_cause`、`impact_analysis`、`resolution`、`lessons`、`review_status`、`reviewed_at` | 复盘记录 |
| `incident_improvement_items` | `incident_id`、`title`、`owner_user_id`、`priority`、`status`、`due_date`、`completed_at` | 改进项 |
| `incident_review_templates` | `name`、`content`、`enabled` | 复盘模板 |

#### API 设计

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/incidents/reviews` | 复盘列表 |
| `POST` | `/api/v1/incidents/:id/review` | 保存复盘 |
| `POST` | `/api/v1/incidents/:id/review/ai-draft` | AI 生成复盘草稿 |
| `GET` | `/api/v1/incidents/actions` | 改进项列表 |
| `PUT` | `/api/v1/incidents/actions/:id` | 更新改进项 |
| `GET` | `/api/v1/incidents/stats` | 故障统计 |

#### 验收标准

- 每个关闭事件都可以生成复盘。
- 改进项可以按负责人和状态筛选。
- 可导出复盘 Markdown。

---

### 阶段 6：Runbook 一期 - 知识库与处置流程

时间：2026-10-19 至 2026-11-06  
目标：把日常处置流程标准化，并和告警、应用、任务中心关联。

#### 开发内容

| 内容 | 怎么开发 |
| --- | --- |
| Runbook 列表 | 支持分类、标签、应用、故障类型、等级筛选 |
| Runbook 编辑 | Markdown 内容 + 结构化步骤 |
| 步骤类型 | 文档步骤、检查步骤、命令步骤、跳转链接、人工确认 |
| 关联对象 | 可关联应用、告警规则、故障中心、事件类型 |
| 执行记录 | 执行 Runbook 时记录执行人、步骤状态、输出、耗时 |

#### 涉及菜单

| 菜单 | 路径 |
| --- | --- |
| 运维知识库 / Runbook 列表 | `/runbooks/list` |
| 运维知识库 / Runbook 执行 | `/runbooks/executions` |
| 运维知识库 / 命令模板 | `/runbooks/commands` |

#### 新增表

| 表名 | 关键字段 | 说明 |
| --- | --- | --- |
| `runbooks` | `id`、`title`、`category`、`severity`、`content`、`owner_user_id`、`enabled`、`version` | Runbook 主表 |
| `runbook_steps` | `runbook_id`、`step_order`、`step_type`、`title`、`content`、`command_template_id`、`requires_confirm` | Runbook 步骤 |
| `runbook_bindings` | `runbook_id`、`object_type`、`object_id`、`matcher` | 关联应用/告警/故障中心 |
| `runbook_executions` | `runbook_id`、`incident_id`、`service_id`、`executor_id`、`status`、`started_at`、`finished_at` | 执行记录 |
| `runbook_execution_steps` | `execution_id`、`step_id`、`status`、`output`、`started_at`、`finished_at` | 步骤执行记录 |
| `runbook_command_templates` | `name`、`command`、`risk_level`、`timeout`、`description` | 命令模板 |

#### 复用表

- `job_templates`
- `job_tasks`
- `monitor_alert_rules`
- `monitor_fault_centers`
- `incidents`
- `app_services`

#### API 设计

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/runbooks` | Runbook 列表 |
| `POST` | `/api/v1/runbooks` | 创建 Runbook |
| `GET` | `/api/v1/runbooks/:id` | Runbook 详情 |
| `PUT` | `/api/v1/runbooks/:id` | 更新 Runbook |
| `POST` | `/api/v1/runbooks/:id/execute` | 执行 Runbook |
| `GET` | `/api/v1/runbooks/executions` | 执行记录 |

#### 验收标准

- 告警事件详情可以推荐关联 Runbook。
- Incident 详情可以启动 Runbook 执行。
- 命令步骤可调用任务中心执行，并回写执行结果。

---

### 阶段 7：应用观测一期 - 指标、日志、事件统一视图

时间：2026-11-09 至 2026-11-27  
目标：开发用户不需要理解底层数据源，也能从应用维度查看指标、日志和错误。

#### 开发内容

| 内容 | 怎么开发 |
| --- | --- |
| 应用观测页 | 应用维度展示指标、日志、告警、变更、事件 |
| 指标面板 | 支持 CPU、内存、QPS、错误率、延迟、Pod 重启、磁盘等默认面板 |
| 日志查询 | 根据应用绑定的 Loki/ES 数据源自动生成查询条件 |
| 错误分析 | 聚合 ERROR 日志、异常栈、Top 错误关键词 |
| 保存查询 | 用户可保存常用查询和面板 |

#### 涉及菜单

| 菜单 | 路径 |
| --- | --- |
| 应用中心 / 应用观测 | `/applications/observability` |
| AIOps / 日志分析 | `/aiops/logs`，增强现有页面 |

#### 新增表

| 表名 | 关键字段 | 说明 |
| --- | --- | --- |
| `app_observability_panels` | `service_id`、`panel_type`、`title`、`datasource_id`、`query`、`config_json`、`sort` | 应用观测面板 |
| `app_saved_queries` | `service_id`、`user_id`、`name`、`datasource_type`、`datasource_id`、`query`、`query_mode` | 保存查询 |
| `app_log_patterns` | `service_id`、`pattern`、`level`、`sample`、`count`、`last_seen_at` | 错误日志模式 |

#### API 设计

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/applications/services/:id/observability` | 应用观测汇总 |
| `POST` | `/api/v1/applications/services/:id/query/metrics` | 查询指标 |
| `POST` | `/api/v1/applications/services/:id/query/logs` | 查询日志 |
| `POST` | `/api/v1/applications/services/:id/logs/analyze` | 错误日志分析 |
| `POST` | `/api/v1/applications/saved-queries` | 保存查询 |

#### 验收标准

- 应用详情能直接打开指标和日志。
- Loki/ES 查询结果以友好表格和代码块展示，不直接抛 JSON。
- 支持最近 15 分钟、1 小时、6 小时、24 小时快捷范围。

---

### 阶段 8：健康巡检一期 - 主机/K8s 基线与容量趋势

时间：2026-11-30 至 2026-12-11  
目标：补齐运维日常巡检、风险发现和容量趋势能力。

#### 开发内容

| 内容 | 怎么开发 |
| --- | --- |
| 主机基线 | 检查磁盘、inode、时间同步、系统版本、开放端口、关键服务、弱配置 |
| K8s 基线 | 检查特权容器、资源限制、镜像拉取、Pending/Failed Pod、节点压力 |
| 巡检策略 | 支持按分组、集群、应用配置巡检周期 |
| 巡检结果 | 展示风险项、等级、建议、关联资源 |
| 容量趋势 | 从现有指标和 Agent 数据中汇总 CPU/内存/磁盘/Pod 使用率趋势 |

#### 涉及菜单

| 菜单 | 路径 |
| --- | --- |
| 健康巡检 / 主机基线 | `/health-checks/hosts` |
| 健康巡检 / K8s 基线 | `/health-checks/kubernetes` |
| 健康巡检 / 容量趋势 | `/health-checks/capacity` |

#### 新增表

| 表名 | 关键字段 | 说明 |
| --- | --- | --- |
| `health_check_profiles` | `name`、`scope_type`、`scope_ids`、`schedule`、`enabled` | 巡检策略 |
| `health_check_rules` | `profile_id`、`rule_code`、`rule_name`、`category`、`severity`、`config_json`、`enabled` | 巡检规则 |
| `health_check_tasks` | `profile_id`、`status`、`started_at`、`finished_at`、`summary_json` | 巡检任务 |
| `health_check_findings` | `task_id`、`resource_type`、`resource_id`、`severity`、`title`、`detail`、`suggestion`、`status` | 风险发现 |
| `capacity_snapshots` | `resource_type`、`resource_id`、`metric_name`、`value`、`unit`、`sampled_at` | 容量快照 |
| `capacity_forecasts` | `resource_type`、`resource_id`、`metric_name`、`forecast_json`、`risk_level` | 容量预测 |

#### 复用表

- `hosts`
- `k8s_clusters`
- `k8s_cluster_inspections`
- `monitor_datasources`
- `monitor_alert_events`

#### API 设计

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/health-checks/profiles` | 巡检策略列表 |
| `POST` | `/api/v1/health-checks/profiles` | 创建巡检策略 |
| `POST` | `/api/v1/health-checks/profiles/:id/run` | 立即巡检 |
| `GET` | `/api/v1/health-checks/tasks` | 巡检任务列表 |
| `GET` | `/api/v1/health-checks/findings` | 风险项列表 |
| `GET` | `/api/v1/health-checks/capacity` | 容量趋势 |

#### 验收标准

- 支持对主机分组执行巡检。
- 支持对 K8s 集群执行巡检。
- 巡检发现可转为 Incident 或改进项。

---

### 阶段 9：健康巡检二期 - 备份状态与通知健康

时间：2026-12-14 至 2026-12-18  
目标：补齐备份可见性和通知链路可用性，减少“告警没发出去”和“备份没人知道失败”的风险。

#### 开发内容

| 内容 | 怎么开发 |
| --- | --- |
| 备份源管理 | 接入外部备份系统、脚本任务、数据库备份结果 |
| 备份运行记录 | 通过 Webhook 或 API 上报备份结果 |
| 备份告警 | 备份失败、超时、长时间未运行时生成告警 |
| 通知健康 | 定时测试飞书、钉钉、企业微信、邮件、Webhook 通道 |
| 通知失败统计 | 汇总最近通知失败原因和通知对象健康度 |

#### 涉及菜单

| 菜单 | 路径 |
| --- | --- |
| 健康巡检 / 备份状态 | `/health-checks/backups` |
| 监控中心 / 通知健康 | `/monitor/notification-health` |

#### 新增表

| 表名 | 关键字段 | 说明 |
| --- | --- | --- |
| `backup_sources` | `name`、`source_type`、`enabled`、`webhook_token_hash`、`description` | 备份来源 |
| `backup_jobs` | `source_id`、`name`、`resource_type`、`resource_name`、`schedule`、`owner_user_id`、`enabled` | 备份任务 |
| `backup_runs` | `job_id`、`status`、`started_at`、`finished_at`、`size_bytes`、`message`、`raw_payload` | 备份执行记录 |
| `monitor_notification_health_checks` | `notice_object_id`、`notice_type`、`status`、`latency_ms`、`error`、`checked_at` | 通知健康检查 |
| `monitor_notification_failures` | `notice_object_id`、`alert_event_id`、`notice_type`、`error`、`created_at` | 通知失败记录 |

#### API 设计

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/backups/webhook/:token` | 备份结果上报 |
| `GET` | `/api/v1/health-checks/backups/jobs` | 备份任务列表 |
| `GET` | `/api/v1/health-checks/backups/runs` | 备份运行记录 |
| `POST` | `/api/v1/monitor/notification-health/run` | 手动执行通知健康检查 |
| `GET` | `/api/v1/monitor/notification-health` | 通知健康列表 |

#### 验收标准

- 备份结果可以通过 Webhook 上报。
- 连续失败或超时未执行可以生成告警事件。
- 通知对象能展示最近测试状态和失败原因。

---

### 阶段 10：联动优化、权限、安全与上线稳定性

时间：2026-12-21 至 2026-12-31  
目标：把新增模块联动起来，完成权限、审计、文档、测试和线上演示环境能力。

#### 开发内容

| 内容 | 怎么开发 |
| --- | --- |
| 统一全局搜索 | 支持搜索应用、主机、Pod、告警、事件、变更、Runbook |
| 首页升级 | 仪表盘展示应用健康、活跃事件、最近变更、容量风险、备份风险 |
| 数据权限 | 开发用户默认只读，按应用成员控制可见范围 |
| 只读演示模式 | 支持 `test` 用户登录后禁用新增、编辑、删除、执行等按钮 |
| 审计补齐 | 所有新增模块写入操作审计和数据变更日志 |
| 文档补齐 | 更新 opshub-docs，提供应用中心、事件中心、变更接入、Runbook 使用说明 |
| 压测与索引 | 对事件、告警、日志、变更、应用资源关联表补齐索引 |

#### 涉及菜单

- 仪表盘
- 应用中心
- 事件中心
- 变更中心
- 运维知识库
- 健康巡检
- 系统管理 / 角色管理
- 操作审计

#### 新增/调整表

| 表名 | 说明 |
| --- | --- |
| `global_search_indexes` | 可选，保存全局搜索索引 |
| `demo_readonly_policies` | 可选，控制演示环境只读策略 |
| `sys_operation_log` | 复用，补齐新增模块操作审计 |
| `sys_data_log` | 复用，记录关键数据变更 |

#### 验收标准

- `test` 演示用户可以查看所有演示数据，但不能执行写操作。
- 首页可以从应用、事件、变更、告警、容量五个角度展示平台状态。
- 主要列表页均支持分页、筛选、导出。
- 所有新增模块有基础单元测试和关键接口测试。

## 5. 数据表总览

### 5.1 应用中心

| 表名 | 类型 | 是否必须 | 说明 |
| --- | --- | --- | --- |
| `app_services` | 新增 | 必须 | 应用主表 |
| `app_service_members` | 新增 | 必须 | 应用成员 |
| `app_service_resources` | 新增 | 必须 | 应用资源绑定 |
| `app_service_domains` | 新增 | 建议 | 应用域名入口 |
| `app_service_labels` | 新增 | 建议 | 应用标签 |
| `app_service_dependencies` | 新增 | 必须 | 应用依赖 |
| `app_topology_snapshots` | 新增 | 可选 | 拓扑快照 |
| `app_resource_discovery_rules` | 新增 | 可选 | 自动发现规则 |
| `app_observability_panels` | 新增 | 必须 | 应用观测面板 |
| `app_saved_queries` | 新增 | 建议 | 保存查询 |
| `app_log_patterns` | 新增 | 可选 | 错误日志模式 |

### 5.2 事件中心

| 表名 | 类型 | 是否必须 | 说明 |
| --- | --- | --- | --- |
| `incidents` | 新增 | 必须 | 故障事件主表 |
| `incident_alert_events` | 新增 | 必须 | 事件关联告警 |
| `incident_resources` | 新增 | 建议 | 影响资源 |
| `incident_timeline` | 新增 | 必须 | 事件时间线 |
| `incident_comments` | 新增 | 建议 | 协同评论 |
| `incident_reviews` | 新增 | 必须 | 故障复盘 |
| `incident_improvement_items` | 新增 | 必须 | 改进项 |
| `incident_review_templates` | 新增 | 可选 | 复盘模板 |

### 5.3 变更中心

| 表名 | 类型 | 是否必须 | 说明 |
| --- | --- | --- | --- |
| `change_sources` | 新增 | 必须 | 变更来源 |
| `change_webhook_tokens` | 新增 | 必须 | Webhook token |
| `change_events` | 新增 | 必须 | 变更事件 |
| `change_event_resources` | 新增 | 必须 | 变更关联资源 |
| `change_event_tags` | 新增 | 建议 | 变更标签 |

### 5.4 Runbook

| 表名 | 类型 | 是否必须 | 说明 |
| --- | --- | --- | --- |
| `runbooks` | 新增 | 必须 | Runbook 主表 |
| `runbook_steps` | 新增 | 必须 | Runbook 步骤 |
| `runbook_bindings` | 新增 | 必须 | Runbook 关联对象 |
| `runbook_executions` | 新增 | 必须 | Runbook 执行记录 |
| `runbook_execution_steps` | 新增 | 必须 | 步骤执行记录 |
| `runbook_command_templates` | 新增 | 建议 | 命令模板 |

### 5.5 健康巡检与备份

| 表名 | 类型 | 是否必须 | 说明 |
| --- | --- | --- | --- |
| `health_check_profiles` | 新增 | 必须 | 巡检策略 |
| `health_check_rules` | 新增 | 必须 | 巡检规则 |
| `health_check_tasks` | 新增 | 必须 | 巡检任务 |
| `health_check_findings` | 新增 | 必须 | 巡检风险项 |
| `capacity_snapshots` | 新增 | 建议 | 容量快照 |
| `capacity_forecasts` | 新增 | 可选 | 容量预测 |
| `backup_sources` | 新增 | 建议 | 备份来源 |
| `backup_jobs` | 新增 | 建议 | 备份任务 |
| `backup_runs` | 新增 | 建议 | 备份运行记录 |
| `monitor_notification_health_checks` | 新增 | 建议 | 通知健康检查 |
| `monitor_notification_failures` | 新增 | 建议 | 通知失败记录 |

## 6. 推荐开发顺序

| 优先级 | 功能 | 原因 |
| --- | --- | --- |
| P0 | 应用中心一期 | 后续所有能力都需要应用这个核心对象 |
| P0 | 变更中心一期 | 告警排查必须知道最近是否有发布或配置变更 |
| P0 | 事件中心一期 | 让告警进入可处理、可复盘的故障流程 |
| P1 | Runbook 一期 | 把处置经验沉淀为标准流程 |
| P1 | 应用观测一期 | 让开发真正能从应用视角使用平台 |
| P1 | 事件复盘与改进项 | 形成故障治理闭环 |
| P2 | 拓扑、容量、巡检、备份 | 提升平台深度和运维管理能力 |

## 7. 每阶段测试要求

| 测试类型 | 要求 |
| --- | --- |
| 后端单元测试 | usecase、handler、权限判断、数据关联逻辑必须覆盖 |
| 前端构建 | 每阶段执行 `npx vite build` |
| 后端编译 | 每阶段执行 `go build ./...` 或主程序编译 |
| API 测试 | 关键接口提供 curl/Postman 示例 |
| 权限测试 | admin、运维、开发、只读用户都要验证 |
| 数据迁移测试 | 空库、旧库升级、重复执行 migration 都要验证 |
| 审计测试 | 新增、编辑、删除、执行类操作必须写操作日志 |

## 8. 不纳入范围

以下能力不建议放入 OpsHub：

1. CI/CD 流水线编排。
2. 代码仓库管理。
3. 制品仓库管理。
4. 镜像构建系统。
5. 替代 Jenkins、GitLab CI、ArgoCD、Tekton 的发布系统。

OpsHub 只接收这些系统的变更事件，用于观测、审计、故障关联和复盘。

## 9. 最终目标形态

开发用户进入 OpsHub 后，应该先看到“我的应用”，并能完成：

1. 查看应用当前健康状态。
2. 查看最近告警、最近变更、最近错误日志。
3. 查询应用指标和日志。
4. 查看相关 Runbook。
5. 参与故障事件处理和复盘。

运维用户进入 OpsHub 后，应该能完成：

1. 查看全局应用健康。
2. 管理告警、值班、通知、故障中心。
3. 处理 Incident。
4. 追踪变更影响。
5. 执行标准 Runbook。
6. 查看基线巡检、容量风险、备份状态。
7. 通过 AIOps 辅助排障。

最终 OpsHub 的定位应是：以应用为核心，串联资产、容器、监控、告警、日志、变更、事件、Runbook、审计和 AIOps 的运行态运维平台。
