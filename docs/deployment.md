# OpsHub 部署指南

本文档详细介绍 OpsHub 的多种部署方式，请根据实际环境选择合适的部署方案。

---

## 目录

- [环境要求](#环境要求)
- [方式一：Docker Compose 部署（推荐快速体验）](#方式一docker-compose-部署推荐快速体验)
- [方式二：Helm 部署（推荐生产环境）](#方式二helm-部署推荐生产环境)
- [方式三：源码部署](#方式三源码部署)
- [环境变量说明](#环境变量说明)
- [常见问题](#常见问题)

---

## 环境要求

| 组件 | 最低版本 | 推荐版本 |
|:-----|:---------|:---------|
| Go | 1.21+ | 1.24+ |
| Node.js | 18+ | 20+ |
| MySQL | 8.0+ | 8.0+ |
| Redis | 6.0+ | 7.0+ |
| Docker | 20.10+ | 24.0+ |
| Kubernetes | 1.24+ | 1.28+ |
| Helm | 3.0+ | 3.14+ |

---

## 方式一：Docker Compose 部署（推荐快速体验）

最简单的部署方式，一键启动所有服务，适合快速体验和开发测试。

### 1. 克隆项目

```bash
git clone https://github.com/ydcloud-dy/opshub.git
cd opshub
```

### 2. 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 3. 访问系统

| 服务 | 地址 |
|:-----|:-----|
| 前端 | http://localhost:5173 |
| 后端 API | http://localhost:9876 |
| Swagger 文档 | http://localhost:9876/swagger/index.html |

### 4. 常用命令

```bash
# 停止服务
docker-compose down

# 停止并删除数据卷
docker-compose down -v

# 重启单个服务
docker-compose restart opshub-backend

# 查看服务日志
docker-compose logs -f opshub-backend
```

---

## 方式二：Helm 部署（推荐生产环境）

使用 Helm Chart 在 Kubernetes 上部署 OpsHub，适合生产环境。

### 前置条件

- Kubernetes 1.24+ 集群
- Helm 3.0+
- kubectl 已配置好集群访问
- Ingress Controller（如 nginx-ingress）
- StorageClass（如需持久化存储）

### 1. 克隆项目

```bash
git clone https://github.com/ydcloud-dy/opshub.git
cd opshub
```

### 2. 快速安装（默认配置）

```bash
# 创建命名空间并安装
helm install opshub ./charts/opshub \
  --namespace opshub \
  --create-namespace
```

### 3. 自定义配置安装

创建 `my-values.yaml` 文件：

```yaml
# 后端配置
backend:
  replicaCount: 3
  resources:
    requests:
      memory: "512Mi"
      cpu: "200m"
    limits:
      memory: "1Gi"
      cpu: "1000m"

# 前端配置
frontend:
  replicaCount: 3

# MySQL 配置
mysql:
  auth:
    rootPassword: "YourSecurePassword123"
  persistence:
    size: 50Gi
    storageClass: "your-storage-class"

# Redis 配置
redis:
  auth:
    password: "YourRedisPassword"

# 服务器配置
server:
  mode: release
  jwtSecret: "your-very-long-random-secret-key-at-least-32-chars"
  jwtExpire: "24h"

# Ingress 配置
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: opshub.yourcompany.com
      paths:
        - path: /
          pathType: Prefix
  # 启用 HTTPS
  tls:
    - secretName: opshub-tls
      hosts:
        - opshub.yourcompany.com
```

安装：

```bash
helm install opshub ./charts/opshub \
  --namespace opshub \
  --create-namespace \
  -f my-values.yaml
```

### 4. 使用外部数据库

如果已有 MySQL 和 Redis，可以禁用内置服务：

```yaml
# 禁用内置 MySQL
mysql:
  enabled: false

# 外部 MySQL 配置
externalDatabase:
  host: mysql.example.com
  port: 3306
  database: opshub
  username: opshub
  password: "your-mysql-password"

# 禁用内置 Redis
redis:
  enabled: false

# 外部 Redis 配置
externalRedis:
  host: redis.example.com
  port: 6379
  password: "your-redis-password"
```

### 5. 验证安装

```bash
# 查看 Pod 状态
kubectl get pods -n opshub

# 查看所有资源
kubectl get all -n opshub

# 查看 Ingress
kubectl get ingress -n opshub
```

期望输出：

```
NAME                                  READY   STATUS    RESTARTS   AGE
opshub-backend-xxx-xxx                1/1     Running   0          2m
opshub-backend-xxx-xxx                1/1     Running   0          2m
opshub-frontend-xxx-xxx               1/1     Running   0          2m
opshub-frontend-xxx-xxx               1/1     Running   0          2m
opshub-mysql-0                        1/1     Running   0          2m
opshub-redis-xxx-xxx                  1/1     Running   0          2m
```

### 6. 访问系统

如果配置了 Ingress，通过域名访问：

```
https://opshub.yourcompany.com
```

如果没有 Ingress，可以使用端口转发：

```bash
# 转发前端服务
kubectl port-forward svc/opshub-frontend 8080:80 -n opshub

# 转发后端服务（另一个终端）
kubectl port-forward svc/opshub-backend 9876:9876 -n opshub

# 访问
# 前端: http://localhost:8080
# API: http://localhost:9876
```

### 7. 升级

```bash
# 更新配置后升级
helm upgrade opshub ./charts/opshub -n opshub -f my-values.yaml

# 查看升级历史
helm history opshub -n opshub

# 回滚到上一版本
helm rollback opshub -n opshub
```

### 8. 卸载

```bash
# 卸载 Helm release
helm uninstall opshub -n opshub

# 删除命名空间（会删除所有数据！）
kubectl delete namespace opshub
```

### 9. 常用运维命令

```bash
# 查看 Pod 日志
kubectl logs -f deployment/opshub-backend -n opshub
kubectl logs -f deployment/opshub-frontend -n opshub

# 进入 Pod 调试
kubectl exec -it deployment/opshub-backend -n opshub -- /bin/sh

# 重启 Deployment
kubectl rollout restart deployment/opshub-backend -n opshub
kubectl rollout restart deployment/opshub-frontend -n opshub

# 扩缩容
kubectl scale deployment opshub-backend --replicas=5 -n opshub

# 查看资源使用
kubectl top pods -n opshub
```

### 10. 完整配置参考

详细配置请参考 [charts/opshub/README.md](../charts/opshub/README.md) 和 [charts/opshub/values.yaml](../charts/opshub/values.yaml)。

---

## 方式三：源码部署

适合开发调试和二次开发场景。

### 1. 克隆项目

```bash
git clone https://github.com/ydcloud-dy/opshub.git
cd opshub
```

### 2. 初始化数据库

```bash
# 创建数据库
mysql -u root -p -e "CREATE DATABASE opshub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 导入初始化脚本
mysql -u root -p opshub < migrations/init.sql
```

### 3. 配置后端

```bash
cp config/config.yaml.example config/config.yaml
# 编辑 config.yaml 修改数据库连接信息
```

### 4. 启动后端

```bash
# 安装依赖
go mod download

# 启动服务
go run main.go server
```

### 5. 启动前端

```bash
cd web

# 安装依赖
npm install

# 开发模式
npm run dev

# 或者构建生产版本
npm run build
```

### 6. 访问系统

| 服务 | 地址 |
|:-----|:-----|
| 前端 | http://localhost:5173 |
| 后端 API | http://localhost:9876 |
| Swagger 文档 | http://localhost:9876/swagger/index.html |

---

## 环境变量说明

| 变量名 | 描述 | 默认值 |
|:-------|:-----|:-------|
| `OPSHUB_SERVER_MODE` | 运行模式 (debug/release) | `debug` |
| `OPSHUB_SERVER_HTTP_PORT` | HTTP 端口 | `9876` |
| `OPSHUB_SERVER_JWT_SECRET` | JWT 密钥 | - |
| `OPSHUB_SERVER_JWT_EXPIRE` | JWT 过期时间 | `24h` |
| `OPSHUB_DATABASE_HOST` | MySQL 地址 | `127.0.0.1` |
| `OPSHUB_DATABASE_PORT` | MySQL 端口 | `3306` |
| `OPSHUB_DATABASE_DATABASE` | 数据库名 | `opshub` |
| `OPSHUB_DATABASE_USERNAME` | 数据库用户名 | `root` |
| `OPSHUB_DATABASE_PASSWORD` | 数据库密码 | - |
| `OPSHUB_REDIS_HOST` | Redis 地址 | `127.0.0.1` |
| `OPSHUB_REDIS_PORT` | Redis 端口 | `6379` |
| `OPSHUB_REDIS_PASSWORD` | Redis 密码 | - |
| `OPSHUB_REDIS_DB` | Redis 数据库 | `0` |

---

## 常见问题

### 1. 数据库连接失败

**问题**: 启动时提示数据库连接失败

**解决方案**:
- 检查 MySQL 服务是否正常运行
- 确认数据库连接配置是否正确
- 检查防火墙是否开放 3306 端口
- 确认数据库用户权限

### 2. Redis 连接失败

**问题**: Redis 连接超时或拒绝

**解决方案**:
- 检查 Redis 服务是否正常运行
- 确认 Redis 配置（地址、端口、密码）
- 检查防火墙是否开放 6379 端口

### 3. 前端无法访问后端 API

**问题**: 前端报 CORS 或网络错误

**解决方案**:
- 检查后端服务是否正常运行
- 确认 Nginx 代理配置正确
- 检查 API 地址配置

### 4. Helm 安装后 Pod 启动失败

**问题**: Pod 处于 CrashLoopBackOff 或 Pending 状态

**解决方案**:

```bash
# 查看 Pod 日志
kubectl logs -f <pod-name> -n opshub

# 查看 Pod 详情
kubectl describe pod <pod-name> -n opshub

# 常见原因：
# - 镜像拉取失败：检查镜像名称和网络
# - PVC 创建失败：检查 StorageClass 是否存在
# - 配置错误：检查 values.yaml 配置
# - 资源不足：增加节点资源或调整 requests/limits
```

### 5. Ingress 无法访问

**问题**: 通过域名无法访问服务

**解决方案**:
- 确认 Ingress Controller 已安装并运行
- 检查域名 DNS 解析是否正确
- 确认 Ingress 规则配置正确
- 检查 TLS 证书是否有效（如启用 HTTPS）

```bash
# 检查 Ingress 状态
kubectl get ingress -n opshub
kubectl describe ingress opshub -n opshub
```

### 6. MySQL Pod 一直 Pending

**问题**: MySQL Pod 无法调度

**解决方案**:
- 检查是否有可用的 StorageClass
- 检查 PVC 是否创建成功

```bash
# 查看 PVC 状态
kubectl get pvc -n opshub

# 查看 StorageClass
kubectl get sc
```

---

## 下一步

- [数据库初始化说明](../migrations/README.md)
- [Helm Chart 详细配置](../charts/opshub/README.md)
- [Kubernetes 插件文档](plugins/kubernetes.md)
- [任务中心插件文档](plugins/task.md)
- [监控中心插件文档](plugins/monitor.md)
