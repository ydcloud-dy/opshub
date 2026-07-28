# OpsHub 生产发布与恢复手册

本文档对应日志中心开发计划阶段 9，覆盖发布前检查、备份、灰度、升级、回滚和故障恢复。命令默认使用固定版本镜像，不建议在生产环境使用 `latest`。

## 1. 发布前检查

在仓库根目录执行：

```bash
./scripts/opshub-release-check.sh --compose --helm --strict
```

使用生产 values 文件时：

```bash
./scripts/opshub-release-check.sh --helm \
  --values ./deploy/production-values.yaml \
  --release opshub \
  --namespace opshub \
  --strict
```

该命令只执行 `docker compose config`、`helm lint` 和 `helm template`，不会创建、更新、重启或删除资源。

严格模式必须通过以下检查：

- JWT、日志写入 Token、Writer Token、日志中心加密密钥不是默认值。
- API、Web、Gateway、Writer、Collector 都使用不可变版本标签。
- Helm 多副本 backend/frontend 存在 PDB；单副本环境可在 values 中显式关闭。
- Compose 包含 ClickHouse、Gateway、Writer 和 backend/frontend 全部服务。
- 日志写入使用 Kafka/Redpanda 持久队列，不使用 `direct` 模式。
- backend 多副本时，日志导出目录使用支持 `ReadWriteMany` 的共享卷。
- 导出任务已配置最大重试次数和租约时间，后端重启后可从 MySQL 恢复任务。

Compose 生产检查需要显式启用 standard profile，并在 `.env` 设置：

```bash
OPSHUB_LOG_QUEUE_MODE=kafka
OPSHUB_LOG_EXPORT_MAX_ATTEMPTS=3
OPSHUB_LOG_EXPORT_LEASE_SECONDS=300

docker compose --profile standard config >/dev/null
./scripts/opshub-release-check.sh --compose --strict
```

Helm 多副本示例：

```yaml
backend:
  replicaCount: 2

logCenter:
  ingest:
    queue:
      mode: kafka
      redpanda:
        enabled: true
  export:
    maxAttempts: 3
    leaseSeconds: 300
    persistence:
      enabled: true
      storageClass: your-rwx-storage-class
      accessModes:
        - ReadWriteMany
```

## 2. 备份

### 2.1 Docker Compose

先创建带时间戳的目录。备份期间不要执行 `docker compose down -v`：

```bash
backup_directory="./backups/$(date +%Y%m%d-%H%M%S)"
mkdir -p "${backup_directory}"

docker compose exec -T mysql sh -c \
  'exec mysqldump --single-transaction --routines --triggers -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE"' \
  > "${backup_directory}/opshub.sql"

docker compose exec -T clickhouse clickhouse-client \
  --user "$CLICKHOUSE_USERNAME" \
  --password "$CLICKHOUSE_PASSWORD" \
  --query 'SHOW CREATE DATABASE opshub_logs' \
  > "${backup_directory}/clickhouse-database.sql"

docker compose config > "${backup_directory}/compose-rendered.yaml"
sha256sum "${backup_directory}"/* > "${backup_directory}/SHA256SUMS"
```

ClickHouse 日志数据应使用底层卷快照或 ClickHouse 官方备份工具。不要在高写入期间直接复制正在使用的 ClickHouse 数据目录；如果只能做卷级备份，先暂停 Writer 和 Gateway，确认写入队列为空，再创建快照。

### 2.2 Helm

MySQL 逻辑备份示例：

```bash
backup_directory="./backups/$(date +%Y%m%d-%H%M%S)"
mkdir -p "${backup_directory}"

kubectl exec -n opshub statefulset/opshub-mysql -- \
  sh -c 'exec mysqldump --single-transaction --routines --triggers -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE"' \
  > "${backup_directory}/opshub.sql"

helm get values opshub -n opshub -a > "${backup_directory}/helm-values.yaml"
helm get manifest opshub -n opshub > "${backup_directory}/helm-manifest.yaml"
sha256sum "${backup_directory}"/* > "${backup_directory}/SHA256SUMS"
```

ClickHouse PVC 使用存储平台的 `VolumeSnapshot` 或 ClickHouse 官方备份方案。备份前记录 PVC、StorageClass、ClickHouse 版本和快照 ID。

## 3. 灰度发布

灰度目标是 5 台主机和 1 个 Kubernetes 集群，不直接扩大到全量资产。

1. 先在日志中心创建只绑定灰度主机/集群的采集策略。
2. 使用固定版本 Agent/Collector，确认节点心跳、策略版本和 WAL 均正常。
3. 连续观察至少 30 分钟：采集实例在线率、Gateway 拒绝数、Writer Lag、ClickHouse 写入延迟、WAL 增长和查询耗时。
4. 灰度无异常后再扩大策略目标；发现异常时先停用策略，不要直接删除宿主机 WAL 目录。

Kubernetes Collector 是集群级 DaemonSet。停用策略时选择“停用并关闭采集器”只会在该集群没有其他已发布策略时删除 DaemonSet；存在共享策略时系统会保留它。

## 4. Docker Compose 升级

先备份并执行发布前检查，然后只替换需要升级的服务：

```bash
docker compose pull log-gateway log-writer backend frontend
docker compose up -d --no-deps --force-recreate log-writer log-gateway
docker compose up -d --no-deps --force-recreate backend frontend
docker compose ps
docker compose logs --since=5m backend log-gateway log-writer
```

检查 `/health`、日志中心采集链路和 ClickHouse 写入后，再扩大采集策略范围。旧版本回滚时把 `.env` 中的镜像标签改回上一版本，重复执行 `pull` 和 `up`；不要删除数据卷。

## 5. Helm 升级与回滚

```bash
helm upgrade --install opshub ./charts/opshub \
  --namespace opshub --create-namespace \
  --values ./deploy/production-values.yaml \
  --atomic --wait --timeout 15m --history-max 10

kubectl rollout status deployment/opshub-backend -n opshub --timeout=10m
kubectl rollout status deployment/opshub-frontend -n opshub --timeout=10m
kubectl rollout status deployment/opshub-log-gateway -n opshub --timeout=10m
kubectl rollout status deployment/opshub-log-writer -n opshub --timeout=10m
```

查看历史并回滚：

```bash
helm history opshub -n opshub
helm rollback opshub <REVISION> -n opshub --wait --timeout 15m
```

回滚后仍需检查数据库兼容性、ClickHouse 表结构、Writer Lag 和 Agent 配置版本。数据库 schema 只允许向前兼容，不能依赖 Helm 回滚自动删除字段。

## 6. 故障恢复

### API 或 Web 发布失败

```bash
kubectl get pods -n opshub
kubectl describe pod <pod> -n opshub
kubectl logs deployment/opshub-backend -n opshub --tail=200
helm rollback opshub <REVISION> -n opshub --wait
```

### ClickHouse 暂时不可用

Kafka 模式下保留 Writer，不要删除 Topic；观察 Lag 上升，恢复 ClickHouse 后确认 Lag 回落且相同 `batch_id` 没有重复。Direct 模式下观察 Agent WAL，恢复后确认 WAL 逐步下降。

### Collector 异常

```bash
kubectl get daemonset,pods -n opshub-system -o wide
kubectl logs -n opshub-system -l app.kubernetes.io/name=opshub-log-agent --tail=200 --prefix
```

先修复镜像、网络或 Token，再重新安装/升级 Collector。不要为了清理故障直接删除 `/var/lib/opshub-log-agent`，其中可能有尚未确认的日志。

## 7. 50,000 EPS 与 7 天观察

短时 50,000 条日志测试不能替代持续容量验收。正式验收需要独立压测环境连续运行 4 小时，并记录：

- Gateway 接收、拒绝、限流和发布延迟。
- Writer 消费 Lag、写入延迟、失败批次和死信数量。
- ClickHouse CPU、内存、磁盘、Merge、写入耗时和查询耗时。
- Agent WAL、重试、丢弃、截断和心跳在线率。

灰度扩大后连续观察 7 天；任一严重数据丢失、持续 OOM、Lag 不回落或查询阻塞都应停止扩大范围并保留现场信息。
