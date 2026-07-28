# OpsHub Helm Chart

OpsHub 的官方 Helm Chart，用于在 Kubernetes 上部署 OpsHub 运维管理平台。

## 前置条件

- Kubernetes 1.24+
- Helm 3.12+
- PV provisioner（如需持久化存储）
- Ingress Controller（如需外部访问）

## 安装

### 方式一：本地安装

```bash

# 克隆项目
git clone https://github.com/ydcloud-dy/opshub.git
cd opshub

# 使用默认配置安装
helm install opshub ./charts/opshub \
  --namespace opshub \
  --create-namespace

# 使用自定义配置安装
helm install opshub ./charts/opshub \
  --namespace opshub \
  --create-namespace \
  -f my-values.yaml
```

### 方式二：指定参数安装

```bash
helm install opshub ./charts/opshub \
  --namespace opshub \
  --create-namespace \
  --set ingress.hosts[0].host=opshub.mycompany.com \
  --set mysql.auth.rootPassword=MySecurePassword \
  --set server.jwtSecret=my-jwt-secret-key
```

## 卸载

```bash
helm uninstall opshub -n opshub
kubectl delete namespace opshub
```

## 配置参数

### 全局配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `global.storageClass` | 全局存储类 | `""` |
| `global.imagePullPolicy` | 镜像拉取策略 | `IfNotPresent` |
| `global.imagePullSecrets` | 镜像拉取密钥 | `[]` |

### 后端配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `backend.replicaCount` | 副本数；大于 1 时日志导出必须配置 RWX 共享卷 | `1` |
| `backend.podDisruptionBudget.enabled` | 是否保护后端滚动维护期间的可用副本 | `true` |
| `backend.podDisruptionBudget.minAvailable` | 后端最少可用副本 | `1` |
| `backend.image.repository` | 镜像仓库 | `ydcloud/opshub-backend` |
| `backend.image.tag` | 镜像标签 | `latest` |
| `backend.resources.requests.memory` | 内存请求 | `256Mi` |
| `backend.resources.requests.cpu` | CPU 请求 | `100m` |
| `backend.resources.limits.memory` | 内存限制 | `512Mi` |
| `backend.resources.limits.cpu` | CPU 限制 | `500m` |

### 前端配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `frontend.replicaCount` | 副本数 | `2` |
| `frontend.podDisruptionBudget.enabled` | 是否保护前端滚动维护期间的可用副本 | `true` |
| `frontend.podDisruptionBudget.minAvailable` | 前端最少可用副本 | `1` |
| `frontend.image.repository` | 镜像仓库 | `ydcloud/opshub-frontend` |
| `frontend.image.tag` | 镜像标签 | `latest` |
| `frontend.resources.requests.memory` | 内存请求 | `64Mi` |
| `frontend.resources.requests.cpu` | CPU 请求 | `50m` |

### 日志数据面配置

Log Gateway 与 Log Writer 使用独立镜像和 Deployment，不复用后端 API 镜像，可分别发布、扩容和回滚。

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `logCenter.ingest.gateway.replicaCount` | Gateway 副本数 | `2` |
| `logCenter.ingest.gateway.image.repository` | Gateway 镜像仓库 | `docker.1ms.run/dyclouds/opshub-log-gateway` |
| `logCenter.ingest.gateway.image.tag` | Gateway 镜像标签 | `v0.0.8` |
| `logCenter.ingest.writer.replicaCount` | Writer 副本数 | `2` |
| `logCenter.ingest.writer.image.repository` | Writer 镜像仓库 | `docker.1ms.run/dyclouds/opshub-log-writer` |
| `logCenter.ingest.writer.image.tag` | Writer 镜像标签 | `v0.0.8` |
| `logCenter.ingest.queue.mode` | 数据面模式：`direct` 或 `kafka` | `direct` |
| `logCenter.export.maxAttempts` | 导出任务最大执行次数 | `3` |
| `logCenter.export.leaseSeconds` | 导出 worker 租约秒数 | `300` |
| `logCenter.export.persistence.enabled` | 是否持久化导出文件；后端多副本时必须开启 | `false` |

### MySQL 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `mysql.enabled` | 是否启用内置 MySQL | `true` |
| `mysql.auth.rootPassword` | root 密码 | `OpsHub@2024` |
| `mysql.auth.database` | 数据库名 | `opshub` |
| `mysql.persistence.enabled` | 是否启用持久化 | `true` |
| `mysql.persistence.size` | 存储大小 | `20Gi` |

### Redis 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `redis.enabled` | 是否启用内置 Redis | `true` |
| `redis.auth.password` | 密码 | `OpsHub@Redis` |
| `redis.persistence.enabled` | 是否启用持久化 | `false` |

### 外部数据库配置

当 `mysql.enabled=false` 时使用：

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `externalDatabase.host` | 主机地址 | `""` |
| `externalDatabase.port` | 端口 | `3306` |
| `externalDatabase.database` | 数据库名 | `opshub` |
| `externalDatabase.username` | 用户名 | `root` |
| `externalDatabase.password` | 密码 | `""` |

### 外部 Redis 配置

当 `redis.enabled=false` 时使用：

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `externalRedis.host` | 主机地址 | `""` |
| `externalRedis.port` | 端口 | `6379` |
| `externalRedis.password` | 密码 | `""` |

### 服务器配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `server.mode` | 运行模式 | `release` |
| `server.timezone` | 后端容器时区，建议与数据库保持一致 | `Asia/Shanghai` |
| `server.httpPort` | HTTP 端口 | `9876` |
| `server.jwtSecret` | JWT 密钥 | `opshub-jwt-secret-...` |
| `server.jwtExpire` | JWT 过期时间 | `24h` |
| `server.externalURL` | 后端外部访问 URL，用于 OAuth2 / 外部回调 | `""` |
| `server.frontendURL` | 前端外部访问 URL，用于告警通知里的事件链接；留空时会优先从 Ingress 自动推导 | `""` |

### Ingress 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `ingress.enabled` | 是否启用 Ingress | `false` |
| `ingress.className` | Ingress 类名 | `nginx` |
| `ingress.hosts[0].host` | 主机域名 | `opshub.example.com` |
| `ingress.tls` | TLS 配置 | `[]` |

## 常见配置示例

### 使用外部数据库

```yaml
mysql:
  enabled: false

externalDatabase:
  host: mysql.example.com
  port: 3306
  database: opshub
  username: opshub
  password: your-password
```

### 启用 HTTPS

```yaml
ingress:
  enabled: true
  hosts:
    - host: opshub.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: opshub-tls
      hosts:
        - opshub.example.com
```

### 告警事件链接

当 `ingress.enabled=true` 且配置了 `ingress.hosts[0].host` 时，Chart 会自动把 `OPSHUB_SERVER_FRONTEND_URL` 注入为 `http(s)://<Ingress Host>`，告警通知里的事件链接不需要手动配置。

如果没有使用 Ingress，或者实际访问入口不是第一个 Ingress Host，可以显式覆盖：

```yaml
server:
  timezone: "Asia/Shanghai"
  externalURL: "http://10.122.28.13"
  frontendURL: "http://10.122.28.13"
```

### 多副本监控调度

后端可以多副本部署。监控中心的规则评估和拨测调度会通过 Redis 进行 Leader 选举，只有当前 Leader 执行调度，Leader 不可用时其他副本会自动接管。可以通过 `/api/v1/plugins/monitor/scheduler/status` 查看当前实例、Leader、最后调度时间和错误信息。

### 生产环境配置

日志链路提供独立的高可用覆盖配置 `values-logcenter-ha.yaml`，其拓扑为：

- Log Gateway：3 个 Deployment 副本。
- Redpanda：3 个 StatefulSet Broker，Topic 副本因子为 3。
- Log Writer：3 个 Deployment 副本，通过同一 Consumer Group 消费。
- ClickHouse：1 个分片、2 个数据副本，使用 ReplicatedMergeTree。
- ClickHouse Keeper：3 个 StatefulSet 节点。

Gateway 和 Writer 不保存本地业务状态，分别通过 Kubernetes Service 和 Kafka Consumer Group 实现负载均衡与故障接管，因此使用 Deployment。Redpanda、ClickHouse、Keeper 需要稳定网络身份和独立数据盘，因此使用 StatefulSet；任意副本都不能共享同一个 RWO PVC。

集群至少需要 3 个可调度节点和支持动态供应的 StorageClass。安装前先渲染并检查资源：

```bash
helm lint ./charts/opshub -f ./charts/opshub/values-logcenter-ha.yaml
helm template opshub ./charts/opshub \
  -n opshub \
  -f ./charts/opshub/values-logcenter-ha.yaml > /tmp/opshub-ha.yaml
```

安装时必须覆盖默认密码和 Token：

```bash
helm upgrade --install opshub ./charts/opshub \
  -n opshub \
  --create-namespace \
  -f ./charts/opshub/values-logcenter-ha.yaml \
  --set-string server.jwtSecret='<JWT_SECRET>' \
  --set-string clickhouse.auth.password='<CLICKHOUSE_PASSWORD>' \
  --set-string logCenter.encryptionKey='<LOGCENTER_ENCRYPTION_KEY>' \
  --set-string logCenter.ingest.ingestToken='<INGEST_TOKEN>' \
  --set-string logCenter.ingest.writerToken='<WRITER_TOKEN>'
```

高可用模式使用硬反亲和，ClickHouse 副本、Keeper 和 Redpanda Broker 会尽量形成节点级故障域。节点数量不足时 Pod 会保持 Pending，而不会把同一组件的多个副本调度到同一节点制造“假高可用”。

#### 从单节点 ClickHouse 迁移

`clickhouse.highAvailability.enabled=true` 会创建一套新的 `clickhouse-ha` StatefulSet 和独立 PVC，不会复用旧数据。已有环境禁止直接从旧 Chart 一步切换到 HA；必须先让 Helm 记录旧 PVC 的保留策略，再迁移数据。

第一步，仍以单节点模式升级一次新 Chart，让 Helm 先记录旧 PVC 的保护注解：

```bash
helm upgrade opshub ./charts/opshub \
  -n opshub \
  -f <当前生产 values.yaml> \
  --set clickhouse.highAvailability.enabled=false

kubectl get pvc opshub-clickhouse-pvc -n opshub \
  -o jsonpath='{.metadata.annotations.helm\.sh/resource-policy}{"\n"}'
```

命令必须输出 `keep`。若没有输出，停止迁移，不能启用 HA。

第二步进入维护窗口，备份旧实例并创建 HA 存储，但暂不恢复业务流量：

```bash
# 先停止日志写入和 API，确认副本均降为 0 后再备份。
kubectl scale deployment/opshub-log-gateway deployment/opshub-log-writer deployment/opshub-backend \
  -n opshub --replicas=0
kubectl wait --for=delete pod \
  -l 'app.kubernetes.io/instance=opshub,app.kubernetes.io/component in (log-gateway,log-writer,backend)' \
  -n opshub --timeout=300s

# 使用 clickhouse-backup 或等价方案完成备份，并验证备份文件可读。

# 创建 ClickHouse HA、Keeper、Redpanda，并临时启动 1 个 Writer 完成复制表初始化。
# Gateway 为 0，不会接收新日志；Backend 为 0，查询 API 仍保持维护状态。
helm upgrade opshub ./charts/opshub \
  -n opshub \
  -f <当前生产 values.yaml> \
  -f ./charts/opshub/values-logcenter-ha.yaml \
  --set backend.replicaCount=0 \
  --set logCenter.ingest.highAvailability.enabled=false \
  --set logCenter.ingest.gateway.replicaCount=0 \
  --set logCenter.ingest.writer.replicaCount=1

kubectl rollout status statefulset/opshub-clickhouse-keeper -n opshub --timeout=10m
kubectl rollout status statefulset/opshub-clickhouse-ha -n opshub --timeout=10m
kubectl rollout status statefulset/opshub-redpanda -n opshub --timeout=10m
kubectl rollout status deployment/opshub-log-writer -n opshub --timeout=10m

# Writer Ready 表示 Replicated*MergeTree 表已在集群中创建；恢复数据前再次停止 Writer。
kubectl scale deployment/opshub-log-writer -n opshub --replicas=0
kubectl wait --for=delete pod \
  -l 'app.kubernetes.io/instance=opshub,app.kubernetes.io/component=log-writer' \
  -n opshub --timeout=300s
```

此时使用 `clickhouse-backup`、ClickHouse `BACKUP/RESTORE` 或经过验证的迁移工具，把旧实例的数据恢复到 `opshub-clickhouse-ha` StatefulSet。必须执行“仅数据恢复”，不能用旧备份里的单机 `MergeTree` 建表 DDL 覆盖已经创建的 `ReplicatedMergeTree`/`ReplicatedSummingMergeTree` 表。客户端 Service 始终使用稳定名称 `opshub-clickhouse`，恢复后必须核对表引擎、总行数、最早/最新日志时间、各日志级别数量以及最近 24 小时聚合量。

第三步才恢复完整服务：

```bash
helm upgrade opshub ./charts/opshub \
  -n opshub \
  -f <当前生产 values.yaml> \
  -f ./charts/opshub/values-logcenter-ha.yaml

kubectl rollout status deployment/opshub-backend -n opshub --timeout=10m
kubectl rollout status statefulset/opshub-redpanda -n opshub --timeout=10m
kubectl rollout status deployment/opshub-log-writer -n opshub --timeout=10m
kubectl rollout status deployment/opshub-log-gateway -n opshub --timeout=10m
```

恢复采集后检查 Gateway 拒绝量、Kafka 消费积压、Writer 写入失败数、ClickHouse 副本延迟。观察至少一个完整保留周期后，再人工处理旧 StatefulSet 和 PVC。第二步失败时，先保持采集停止，再使用 `helm rollback` 回到第一步的版本；旧 PVC 仍在，可以重新拉起单节点 ClickHouse。

客户端始终访问 `opshub-clickhouse:8123`，因此数据库中已有的内置日志库地址不需要修改。HA Service 使用独立的 `opshub.io/clickhouse-mode=ha` 选择器，不会在迁移窗口把流量误发给旧单节点 Pod。旧 PVC 带有 `helm.sh/resource-policy=keep`，即使 HA 升级不再渲染单节点资源，Helm 也不会自动删除该 PVC。

当前内置 HA 模式面向单分片双副本。需要多分片、跨可用区或者滚动升级编排时，建议设置 `clickhouse.enabled=false`，接入由 ClickHouse Operator 或云服务管理的外部集群。

`values-logcenter-ha.yaml` 保障日志数据面的高可用，不会把内置单节点 MySQL 和 Redis 自动改造成集群。需要整个平台达到生产级高可用时，应设置 `mysql.enabled=false`、`redis.enabled=false`，并接入已有的 MySQL 高可用集群和 Redis Sentinel/Cluster 或云数据库服务。

## 升级

```bash
helm upgrade opshub ./charts/opshub -n opshub -f values.yaml
```

## 故障排查

```bash
# 查看 Pod 状态
kubectl get pods -n opshub

# 查看 Pod 日志
kubectl logs -f deployment/opshub-backend -n opshub
kubectl logs -f deployment/opshub-frontend -n opshub

# 查看 Pod 详情
kubectl describe pod <pod-name> -n opshub
```
