# OpsHub Helm Chart

OpsHub 的官方 Helm Chart，用于在 Kubernetes 上部署 OpsHub 运维管理平台。

## 前置条件

- Kubernetes 1.24+
- Helm 3.0+
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
| `backend.replicaCount` | 副本数 | `2` |
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
| `frontend.image.repository` | 镜像仓库 | `ydcloud/opshub-frontend` |
| `frontend.image.tag` | 镜像标签 | `latest` |
| `frontend.resources.requests.memory` | 内存请求 | `64Mi` |
| `frontend.resources.requests.cpu` | CPU 请求 | `50m` |

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
| `server.httpPort` | HTTP 端口 | `9876` |
| `server.jwtSecret` | JWT 密钥 | `opshub-jwt-secret-...` |
| `server.jwtExpire` | JWT 过期时间 | `24h` |

### Ingress 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `ingress.enabled` | 是否启用 Ingress | `true` |
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

### 生产环境配置

```yaml
backend:
  replicaCount: 3
  resources:
    requests:
      memory: "512Mi"
      cpu: "200m"
    limits:
      memory: "1Gi"
      cpu: "1000m"

frontend:
  replicaCount: 3

mysql:
  persistence:
    size: 100Gi

server:
  jwtSecret: "your-very-long-random-secret-key"
```

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
