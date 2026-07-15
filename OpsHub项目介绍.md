# OpsHub：一站式企业级运维管理平台

OpsHub 是一个面向企业内部运维、SRE、DevOps 场景打造的一站式运维管理平台，基于 **Go + Vue3** 实现，支持资产管理、容器管理、终端审计、监控告警、SSL 证书管理、Nginx 日志统计、任务中心、插件管理等能力。

它的目标不是只做一个“工具集合”，而是把日常运维中分散的能力统一到一个平台里，让服务器、Kubernetes、告警、审计、任务、证书、日志分析等工作可以在同一个入口完成。

---

## 项目亮点

- **开箱即用**：支持 Docker Compose / Helm 部署，适合本地测试和生产环境部署。
- **前后端分离**：后端使用 Go，前端使用 Vue3，结构清晰，便于二次开发。
- **插件化架构**：核心能力按插件拆分，方便后续扩展更多运维场景。
- **完整监控中心**：支持 Prometheus、Loki、VictoriaMetrics、Elasticsearch 等数据源，具备告警规则、故障中心、通知对象、值班表、拨测任务等能力。
- **统一审计能力**：终端操作、平台操作、登录行为等都可以纳入审计。
- **面向真实运维场景**：围绕资产、容器、监控、告警、审计、自动化任务等实际需求设计。

---

## 技术架构

OpsHub 使用前后端分离架构：

- 前端：Vue3、Element Plus、TypeScript
- 后端：Go、Gin、Gorm
- 数据库：MySQL
- 缓存与协调：Redis
- 部署方式：Docker Compose、Helm
- 可观测数据源：Prometheus、Loki、VictoriaMetrics、Elasticsearch

![系统架构图](./images/opshub/architecture-01.png)

---

## 仪表盘

仪表盘提供平台整体运行概览，可以快速查看资产数量、告警状态、资源统计、最近操作、活跃告警等信息。

![仪表盘总览](./images/opshub/dashboard-01.png)

![资源概览](./images/opshub/dashboard-02.png)

![活跃告警](./images/opshub/dashboard-03.png)

---

## 资产管理

资产管理用于统一维护服务器、主机分组、登录凭据等信息。通过资产管理，可以将服务器资源和后续的终端连接、文件管理、任务执行等功能关联起来。

主要能力包括：

- 主机管理
- 主机分组
- 凭据管理
- 资产授权
- 主机状态展示

![资产管理列表](./images/opshub/asset-01.png)

![主机详情](./images/opshub/asset-02.png)

![凭据管理](./images/opshub/asset-03.png)

---

## 容器管理

容器管理面向 Kubernetes 场景，支持集群接入、命名空间、节点、工作负载、网络、配置、存储等资源管理。

主要能力包括：

- 集群管理
- 节点管理
- 命名空间管理
- 工作负载管理
- Service / Ingress 管理
- ConfigMap / Secret 管理
- 存储资源管理
- 集群巡检
- 应用诊断

![集群管理](./images/opshub/kubernetes-01.png)

![节点管理](./images/opshub/kubernetes-02.png)

![工作负载](./images/opshub/kubernetes-03.png)

![应用诊断](./images/opshub/kubernetes-04.png)

---

## 应用诊断

应用诊断基于 Arthas 实现，面向 Java 应用运行时排障场景。

在 Kubernetes Pod 中选择 Java 进程后，可以查看线程、JVM 信息、系统信息、线程堆栈，并支持方法追踪、方法监控、方法观察、火焰图等能力。

适合用于排查：

- CPU 飙高
- 线程阻塞
- 接口耗时高
- 方法调用异常
- JVM 运行状态异常
- Java 应用线上问题定位

![应用诊断入口](./images/opshub/diagnosis-01.png)

![线程信息](./images/opshub/diagnosis-02.png)

![JVM 信息](./images/opshub/diagnosis-03.png)

![方法追踪](./images/opshub/diagnosis-04.png)

---

## 终端与文件管理

OpsHub 支持通过 Web 方式连接服务器终端，并记录终端操作审计。

同时，平台也提供文件管理能力，可以进行文件浏览、上传、下载等操作，方便运维人员处理服务器上的常见文件操作。

主要能力包括：

- Web SSH 终端
- 终端会话管理
- 终端操作审计
- 文件浏览
- 文件上传与下载
- 文件分发

![Web 终端](./images/opshub/terminal-01.png)

![终端审计](./images/opshub/terminal-02.png)

![文件管理](./images/opshub/file-01.png)

---

## 监控中心

监控中心是 OpsHub 近期重点增强的模块，目标是提供一套完整的监控告警闭环能力。

它支持接入多种数据源，并基于数据源创建告警规则、故障中心、通知对象、通知模板、值班表、拨测任务等配置。

支持的数据源包括：

- Prometheus
- VictoriaMetrics
- Loki
- Elasticsearch

主要能力包括：

- 数据源管理
- 监控面板
- 告警规则
- 故障中心
- 活跃告警
- 历史告警
- 告警静默
- 告警升级
- 通知对象
- 通知模板
- 值班表
- 拨测任务
- 即时拨测

![监控面板](./images/opshub/monitor-01.png)

![数据源管理](./images/opshub/monitor-02.png)

![告警规则](./images/opshub/monitor-03.png)

![故障中心](./images/opshub/monitor-04.png)

![活跃告警](./images/opshub/monitor-05.png)

![通知对象](./images/opshub/monitor-06.png)

![值班表](./images/opshub/monitor-07.png)

![拨测任务](./images/opshub/monitor-08.png)

---

## 告警通知

OpsHub 的告警通知支持将告警发送到不同通知对象，并结合值班表、通知模板、告警等级、告警升级等配置，实现更完整的告警流转。

目前支持的通知类型包括：

- 飞书
- 钉钉
- 企业微信
- 邮件
- Webhook

告警内容支持携带规则名称、告警等级、标签、注释、详情、回调查询结果、日志样例、监控图表等信息。

![飞书告警](./images/opshub/notice-01.png)

![钉钉告警](./images/opshub/notice-02.png)

![企业微信告警](./images/opshub/notice-03.png)

![邮件告警](./images/opshub/notice-04.png)

---

## SSL 证书管理

SSL 证书管理用于统一维护域名证书信息，帮助运维人员及时发现证书即将过期的问题。

主要能力包括：

- 证书列表
- 到期时间展示
- 证书详情
- 证书状态检查
- 证书过期提醒

![SSL 证书列表](./images/opshub/ssl-01.png)

![证书详情](./images/opshub/ssl-02.png)

---

## Nginx 日志统计

Nginx 日志统计用于分析访问日志，帮助快速了解接口访问情况、状态码分布、访问趋势、Top URL、Top IP 等信息。

适合用于：

- 分析接口访问量
- 排查异常状态码
- 查看访问来源
- 统计热门接口
- 观察流量趋势

![Nginx 统计总览](./images/opshub/nginx-01.png)

![访问趋势](./images/opshub/nginx-02.png)

![Top URL](./images/opshub/nginx-03.png)

---

## 任务中心

任务中心用于统一管理自动化任务，可以将一些重复性的运维操作沉淀为标准化任务。

主要能力包括：

- 任务创建
- 任务执行
- 执行历史
- 执行结果查看
- 定时或手动触发

![任务列表](./images/opshub/task-01.png)

![任务执行](./images/opshub/task-02.png)

![执行历史](./images/opshub/task-03.png)

---

## 操作审计

操作审计用于记录平台中的关键操作行为，方便后续追踪、排查和合规审计。

审计内容包括：

- 登录日志
- 操作日志
- 终端审计
- 数据变更记录
- 用户行为追踪

![操作日志](./images/opshub/audit-01.png)

![登录日志](./images/opshub/audit-02.png)

![终端审计](./images/opshub/audit-03.png)

---

## 插件管理

OpsHub 支持插件化扩展，不同运维能力可以以插件形式接入平台。

插件管理可以查看插件状态、安装插件、启用插件，并方便后续扩展更多能力。

![插件列表](./images/opshub/plugin-01.png)

![插件详情](./images/opshub/plugin-02.png)

---

## 系统管理

系统管理提供平台基础配置能力，包括用户、角色、菜单、权限、LDAP、MFA 等配置。

主要能力包括：

- 用户管理
- 角色管理
- 菜单管理
- 权限管理
- LDAP 配置
- MFA 配置
- 系统参数配置

![用户管理](./images/opshub/system-01.png)

![角色管理](./images/opshub/system-02.png)

![LDAP 配置](./images/opshub/system-03.png)

---

## 适用场景

OpsHub 适合以下场景：

- 企业内部运维平台建设
- 中小团队统一管理服务器资产
- Kubernetes 集群日常管理
- Java 应用线上诊断
- Prometheus / Loki / Elasticsearch 告警整合
- 统一终端审计与操作审计
- 证书、日志、任务等运维能力集中管理
- 基于现有平台进行二次开发

---

## 总结

OpsHub 希望解决的是“运维能力分散”的问题。

在真实企业环境中，资产、终端、Kubernetes、监控、告警、日志、证书、审计、任务自动化往往分散在多个系统中。OpsHub 尝试把这些能力整合到一个统一平台里，让运维人员可以更高效地完成日常管理、故障排查和系统治理。

如果你正在寻找一个可以二次开发、可以私有化部署、并且覆盖常见运维场景的平台，OpsHub 是一个值得尝试的选择。

---

## 项目地址

- GitHub：`https://github.com/ydcloud-dy/opshub`
- 文档地址：`https://doc.dycloud.fun`
- 演示环境：`https://doc.dycloud.fun`
- 演示账号：`test`
- 演示密码：`test@123`
