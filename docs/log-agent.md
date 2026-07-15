# OpsHub Log Agent

OpsHub Log Agent v0.3.0 使用同一个 Linux 二进制支持主机与 Kubernetes 节点两种模式。主机模式复用资产管理中的 Agent 身份；Kubernetes 模式由平台生成 DaemonSet，每个节点运行一个采集实例。日志统一经过独立 Log Gateway 和 Log Writer 写入 ClickHouse，不占用 OpsHub 主业务 API 的数据面连接。

## 运行模式

| 模式 | 启动方式 | 资产身份 | 日志来源 |
| --- | --- | --- | --- |
| `host` | systemd 或手动启动 | `hostId + agentId` | 主机文件与通配路径 |
| `kubernetes-node` | 平台生成的 DaemonSet | `clusterId + nodeName` | CRI/containerd 与 Docker 容器日志 |

Agent 会每 30 秒拉取一次平台策略。配置没有变化时平台返回 `ETag/304`；新配置校验失败时，Agent 会继续运行上一版采集流水线并回报失败原因。

## 主机模式配置示例

编辑 `/etc/opshub-agent/agent.json`，保留平台注册后生成的 `agentId`、`agentToken` 和 `hostId`，增加 `logCollection`：

```json
{
  "serverUrl": "https://opshub.example.com",
  "agentId": "agt_xxx",
  "agentToken": "平台生成的主机 Agent Token",
  "hostId": 1,
  "interval": 30,
  "logMetricsAddress": "127.0.0.1:19877",
  "logCollection": {
    "enabled": true,
    "gatewayUrl": "http://opshub.example.com:19880",
    "gatewayToken": "OPSHUB_LOG_INGEST_TOKEN",
    "stateDir": "/var/lib/opshub-agent/logs",
    "scanIntervalSeconds": 1,
    "batchSize": 500,
    "flushIntervalSeconds": 2,
    "maxWalBytes": 2147483648,
    "sources": [
      {
        "id": "application-log",
        "paths": ["/data/apps/*/logs/*.log"],
        "excludePaths": ["*.gz", "*.tmp"],
        "readFrom": "latest",
        "environment": "production",
        "service": "application",
        "maxLineBytes": 262144,
        "parser": { "type": "raw" },
        "multiline": {
          "enabled": true,
          "preset": "java",
          "maxLines": 500,
          "maxBytes": 1048576,
          "flushSeconds": 2
        }
      }
    ]
  }
}
```

正常使用时，`gatewayToken` 和采集策略由平台自动下发。静态配置仅用于本地调试或控制面不可用时排障。

## Kubernetes DaemonSet 模式

### 安装前提

1. Kubernetes 集群已在“容器管理 -> 集群管理”中注册，OpsHub 后端能够访问该集群 API Server。
2. 目标集群的节点能够访问 OpsHub 对外域名；该地址不能是 `localhost`、容器名或仅 OpsHub 集群内可见的 Service 地址。
3. OpsHub 前端 Nginx 或 Helm Ingress 已把 `/api/v1/logs/` 转发到 Log Gateway，其余 `/api/` 请求转发到后端。
4. `OPSHUB_LOG_AGENT_IMAGE` 指向与当前控制面兼容的独立 `opshub-log-agent` 镜像，不再复用 API 镜像。
5. 目标节点可读取 `/var/log/containers`；Docker 节点还需要读取 `/var/lib/docker/containers`。

### 平台安装

1. 进入“容器管理 -> 集群详情 -> 日志采集”。
2. 点击“安装采集器”。平台会创建 `opshub-system` Namespace、只读 ServiceAccount/RBAC、Secret、ConfigMap 和滚动更新 DaemonSet。
3. 在“日志中心 -> 采集接入 -> 采集策略”中新建 Kubernetes 策略，选择集群并配置 Namespace、Workload、Pod Label 和容器选择器，然后发布。
4. DaemonSet 节点在 30 秒内拉取策略；集群详情会显示在线实例、配置版本、吞吐、WAL 和最近错误。
5. 从集群详情点击“查询集群日志”，查询页会自动固定当前 `clusterId`。

如果 OpsHub 的集群凭据不允许自动创建资源，可点击“下载 YAML”，然后在目标集群执行：

```bash
kubectl apply -f opshub-log-agent-cluster-<cluster-id>.yaml
```

每次重新生成 YAML 都会轮换集群采集 Token，必须应用最新文件；旧 DaemonSet 使用的 Token 随即失效。

### Kubernetes 选择器

| 选择器 | 示例 | 行为 |
| --- | --- | --- |
| Namespace | `production` | 只采集指定命名空间 |
| Workload | `Deployment/api` | 通过 owner 链匹配 Deployment、StatefulSet、DaemonSet、Job、CronJob 或独立 Pod |
| Pod Label | `app.kubernetes.io/name=api` | 使用 Kubernetes LabelSelector 语法匹配 |
| 包含容器 | `api,sidecar` | 只采集列表中的容器 |
| 排除容器 | `istio-proxy` | 在包含规则之后排除指定容器 |

默认排除 `kube-system`、`kube-public` 和 `kube-node-lease`。Agent 使用 Informer 缓存补全 Cluster、Namespace、Node、Pod、Pod UID、Container、Image、Workload、Service、Environment 和允许列表中的 Pod Label。

### 容器日志兼容性

- CRI/containerd：解析 `<timestamp> <stdout|stderr> <F|P> <content>`，自动合并 `P/F` partial 片段。
- Docker JSON：解析 `log`、`stream` 和 `time` 字段。
- `/var/log/containers/*.log` 符号链接用于识别 Pod、Namespace 和 Container；Informer 用 Pod UID 校验并补全 owner 元数据。
- 节点状态和 WAL 保存在宿主机 `/var/lib/opshub-log-agent`，DaemonSet 滚动升级不会删除未确认日志。

## 部署地址配置

Gateway、Writer 和 Kubernetes Log Agent 使用独立镜像，不再由 `opshub-api` 镜像切换启动命令。五个应用镜像的职责如下：

| 镜像 | 进程 | 职责 |
| --- | --- | --- |
| `opshub-api` | `opshub server` | 控制面、业务 API、策略与权限 |
| `opshub-web` | Nginx | 前端静态资源与反向代理 |
| `opshub-log-gateway` | `opshub-log-gateway` | Agent 鉴权、限流、gRPC/HTTP 接入和 Kafka 发布 |
| `opshub-log-writer` | `opshub-log-writer` | Kafka 消费、批处理、去重、ClickHouse 写入和 deadletter |
| `opshub-log-agent` | `opshub-agent --mode kubernetes-node` | 节点容器日志读取、解析、WAL 和策略拉取 |

本地构建独立日志服务镜像：

```bash
docker build --platform linux/amd64 \
  -f Dockerfile.log-gateway \
  -t dyclouds/opshub-log-gateway:v0.0.8 .

docker build --platform linux/amd64 \
  -f Dockerfile.log-writer \
  -t dyclouds/opshub-log-writer:v0.0.8 .

docker build --platform linux/amd64 \
  -f Dockerfile.log-agent \
  -t dyclouds/opshub-log-agent:v0.0.8 .
```

包含 amd64 和 arm64 节点的集群应直接构建并推送多架构清单：

```bash
docker buildx build --platform linux/amd64,linux/arm64 \
  -f Dockerfile.log-agent \
  -t dyclouds/opshub-log-agent:v0.0.8 \
  --push .
```

Docker Compose 可以通过环境变量覆盖镜像，Gateway 和 Writer 可单独更新：

```bash
export OPSHUB_LOG_GATEWAY_IMAGE=dyclouds/opshub-log-gateway:v0.0.8
export OPSHUB_LOG_WRITER_IMAGE=dyclouds/opshub-log-writer:v0.0.8

docker compose pull log-gateway log-writer
docker compose up -d --no-deps --force-recreate log-writer log-gateway
```

Docker Compose 使用域名部署时，至少配置以下变量并重建后端：

```bash
export OPSHUB_SERVER_EXTERNAL_URL=https://opshub.example.com
export OPSHUB_SERVER_FRONTEND_URL=https://opshub.example.com
export OPSHUB_LOG_GATEWAY_PUBLIC_URL=https://opshub.example.com
export OPSHUB_LOG_AGENT_IMAGE=dyclouds/opshub-log-agent:v0.0.8

docker compose up -d --no-deps --force-recreate backend
```

Helm 使用 Ingress 时，设置同一个外部域名，并显式使用独立 Collector 镜像：

```bash
helm upgrade --install opshub ./charts/opshub \
  -n opshub --create-namespace \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=opshub.example.com \
  --set server.externalURL=https://opshub.example.com \
  --set server.frontendURL=https://opshub.example.com \
  --set logCenter.ingest.gateway.publicURL=https://opshub.example.com \
  --set logCenter.ingest.gateway.image.repository=dyclouds/opshub-log-gateway \
  --set logCenter.ingest.gateway.image.tag=v0.0.8 \
  --set logCenter.ingest.writer.image.repository=dyclouds/opshub-log-writer \
  --set logCenter.ingest.writer.image.tag=v0.0.8 \
  --set logCenter.kubernetesCollector.image=dyclouds/opshub-log-agent:v0.0.8
```

生产环境应使用固定版本镜像，不要给 Gateway、Writer 或 Collector 配置 `latest`。三个数据面组件拥有独立镜像标签，可以分别滚动升级和回滚。Collector DaemonSet 使用 `imagePullPolicy: Always`，即使测试环境重复推送同一标签也会重新拉取；正式环境仍应使用不可变版本标签。

## Redpanda/Kafka 生产队列

小规模环境默认使用 `direct`：Gateway 把批次直接交给 Writer，ClickHouse 成功后才向 Agent 返回 ACK。生产环境建议切换为 `kafka`，Agent 收到 ACK 时批次已经由 Redpanda/Kafka 持久化，ClickHouse 短时故障不会阻塞 Gateway。

Docker Compose 可启用内置单节点 Redpanda：

```bash
export OPSHUB_LOG_QUEUE_MODE=kafka
export OPSHUB_LOG_KAFKA_BROKERS=redpanda:9092
export OPSHUB_LOG_KAFKA_REPLICATION_FACTOR=1

docker compose --profile standard up -d
```

内置 Compose Redpanda 适合功能验证和单机部署。生产环境应使用至少 3 个 Broker，并把 `OPSHUB_LOG_KAFKA_BROKERS` 设置为外部集群的 bootstrap 地址，同时把复制因子调整为 3。

Helm 启用 3 副本内置 Redpanda：

```bash
helm upgrade --install opshub ./charts/opshub \
  -n opshub --create-namespace \
  --set logCenter.ingest.queue.mode=kafka \
  --set logCenter.ingest.queue.redpanda.enabled=true \
  --set logCenter.ingest.queue.redpanda.replicaCount=3 \
  --set logCenter.ingest.queue.replicationFactor=3
```

使用外部 Kafka 时关闭内置 Redpanda，并在 values 文件中配置：

```yaml
logCenter:
  ingest:
    queue:
      mode: kafka
      brokers:
        - kafka-0.kafka.svc:9092
        - kafka-1.kafka.svc:9092
        - kafka-2.kafka.svc:9092
      partitions: 12
      replicationFactor: 3
      tls:
        enabled: true
      sasl:
        mechanism: scram-sha-512
        username: opshub
        password: change-me
```

Kafka 模式按 `agentId` 作为消息 Key，同一 Agent 的批次保持分区内有序。Writer 只有在 ClickHouse 写入成功后才提交 offset；无效批次进入 deadletter Topic。ClickHouse 使用 `batchId` 作为 `insert_deduplication_token`，即使 Writer 在写入成功但提交 offset 前重启，重放也不会重复落库。

采集接入页面会展示当前 Gateway/Writer 实例、Broker 数、Topic、Consumer Lag、发布/写入延迟和 deadletter 数。对应 Prometheus 指标可从 Gateway、Writer 的 `/metrics` 获取，重点关注：

- `opshub_log_gateway_failed_batches_total`
- `opshub_log_gateway_publish_latency_ms`
- `opshub_log_gateway_inflight`
- `opshub_log_writer_queue_lag`
- `opshub_log_writer_write_latency_ms`
- `opshub_log_writer_deadletter_batches_total`
- `opshub_log_writer_queue_healthy`

## 解析方式

- `raw`：保留原始正文，并自动识别常见日志级别。
- `json`：默认识别 `timestamp/time/@timestamp`、`level/severity`、`message/msg/body`。
- `regex`：使用 RE2 命名分组，支持 `timestamp`、`level`、`message` 以及附加字段。
- 多行模板：支持 `java`、`go`、`python` 和带 `startPattern` 的 `custom`。

## 采集前脱敏

采集策略默认在日志写入 WAL 前执行脱敏，明文敏感信息不会进入本地 WAL、Gateway、Kafka 或 ClickHouse。内置规则覆盖 `password`、`token`、`authorization`、`cookie`、`secret` 等常见字段，大小写和下划线写法均会归一化处理。

自定义规则支持三种目标：

- `field`：匹配解析后的字段名。
- `json_path`：匹配 JSON 嵌套路径。
- `regex`：对日志正文执行 RE2 正则替换。

每条规则可选择 `replace`、`hash` 或 `drop_field`。策略发布前应使用测试日志确认业务字段没有被过度脱敏；禁止在自定义替换值中重新写入原始凭据。

## 差异化保留

采集策略可绑定日志库中的保留策略，并按日志级别覆盖默认天数，例如 ERROR 90 天、WARN 45 天、INFO/DEBUG 15 天。Agent 为每条日志计算 `retention_days`，Writer 写入 `expire_at`，ClickHouse 使用 `TTL expire_at DELETE` 异步清理。

升级已有日志库后，需要在“日志中心 -> 日志库 -> 存储配置”重新执行一次“初始化”，该操作幂等，会补齐 `retention_days`、`expire_at` 和表 TTL，不会清空历史日志。

## 查询权限

- 主机日志继承资产管理中的主机访问权限。
- Kubernetes 日志继承集群所有者、用户角色绑定和 kubeconfig Namespace 范围。
- 查询、字段选项、上下文、实时 Tail 和导出使用同一套服务端权限条件。
- 日志访问策略可单独控制 `query`、`tail`、`export`，并配置字段隐藏或掩码。
- OpsHub 内置日志告警使用结构化 Query AST，通知链接会携带存储、资产、筛选条件和准确时间窗口。

## 可靠性

- 使用 `device + inode + offset + 文件头指纹` 保存 checkpoint。
- 支持 rename 和 copytruncate 日志轮转。
- 日志先同步写入本地 WAL，随后才提交 checkpoint。
- Gateway 或 Writer 不可用时保留 WAL 并指数退避；收到 ACK 后删除对应段文件。
- WAL 达到容量上限时暂停读取并回退到最后已提交位置，不静默丢弃日志。
- 单行默认上限 256 KiB，最大 1 MiB；超长内容截断并写入 `log.truncated=true`。
- Kubernetes 节点使用独立 `clusterId + nodeName` 实例身份，配置和心跳互不覆盖。
- 集群 Token 只保存 SHA-256 摘要，明文仅写入目标集群 Secret。
- Collector ServiceAccount 只有 Pods、Namespaces、Nodes、常见 Workload 和 owner 解析所需资源的 `get/list/watch` 权限。

## CLI 快速启用

```bash
opshub-agent \
  --config /etc/opshub-agent/agent.json \
  --log-gateway http://opshub.example.com:19880 \
  --log-gateway-token 'OPSHUB_LOG_INGEST_TOKEN' \
  --log-paths '/var/log/nginx/access.log,/data/apps/*/logs/*.log' \
  --log-read-from latest \
  --log-service web \
  --log-environment production
```

CLI 参数会写回 Agent 配置文件，后续 systemd 重启仍会继续采集。

## 自监控

默认仅监听本机 `127.0.0.1:19877`：

```bash
curl http://127.0.0.1:19877/health
curl http://127.0.0.1:19877/metrics
```

核心指标包括输入/输出/重试/截断数量、WAL 字节数、配置版本和最近成功时间。

## Kubernetes 排障

先检查 DaemonSet 和每个节点实例：

```bash
kubectl get daemonset,pods -n opshub-system -o wide
kubectl logs -n opshub-system -l app.kubernetes.io/name=opshub-log-agent --tail=200 --prefix
kubectl describe daemonset opshub-log-agent -n opshub-system
```

检查只读权限与宿主机日志挂载：

```bash
kubectl auth can-i list pods --as=system:serviceaccount:opshub-system:opshub-log-agent --all-namespaces
kubectl exec -n opshub-system daemonset/opshub-log-agent -- sh -c 'ls -l /var/log/containers | head'
```

| 现象 | 重点检查 |
| --- | --- |
| 一直未注册 | `server.externalURL`/`server.frontendURL` 是否为节点可访问域名，DNS、TLS 和防火墙是否正常 |
| 返回 401 | 是否生成过新 YAML 导致 Token 轮换；重新应用最新 YAML |
| 实例在线但没有日志源 | Kubernetes 策略是否已发布并绑定当前集群，Namespace/Workload/Label/Container 条件是否过窄 |
| 有日志但缺少 Workload | ServiceAccount 是否能 `list/watch` Pod、ReplicaSet、Deployment、Job 等资源 |
| containerd 没有日志 | 节点 `/var/log/containers` 是否存在且链接目标可读 |
| Docker 没有日志 | `/var/lib/docker/containers` 是否存在；非 Docker 节点可忽略该目录 |
| Gateway 上传失败 | `OPSHUB_LOG_GATEWAY_PUBLIC_URL` 是否可从目标节点访问，反向代理是否转发 `/api/v1/logs/` |
| 节点显示离线 | 90 秒内未收到心跳；查看 Pod 重启、网络和认证错误 |
| `flag provided but not defined: -mode` | DaemonSet 使用了旧 API/Agent 镜像；构建并切换到当前 `opshub-log-agent` 镜像后重新升级 Collector |
| `exec format error` | `opshub-log-agent` 镜像是否包含当前节点架构；混合架构集群应发布 amd64/arm64 多架构镜像清单 |
| `ImagePullBackOff` | 镜像地址、标签、仓库凭据和节点到镜像仓库的网络是否正常 |

验证镜像能力：

```bash
docker run --rm --entrypoint /usr/local/bin/opshub-agent \
  dyclouds/opshub-log-agent:v0.0.8 --help 2>&1 | grep -- '-mode'
```

卸载采集器不会删除宿主机 `/var/lib/opshub-log-agent`，以免误删未确认 WAL。确认不再需要补传后，可自行清理该目录。
