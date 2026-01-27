<p align="center">
  <img src="web/public/logo.png" alt="OpsHub Logo" width="180"/>
</p>

<h3 align="center">OpsHub —— 现代化、插件化的云原生运维管理平台</h3>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Vue-3.5+-4FC08D?style=flat&logo=vue.js" alt="Vue">
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat" alt="License">
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat" alt="PRs Welcome">
</p>

---

## 💎 OpsHub 是什么？

**🎯 一站式运维管理平台，让运维更简单**

OpsHub 是一个功能强大的**插件化运维管理平台**，采用前后端分离架构，支持多集群 Kubernetes 管理、主机资产管理、RBAC 权限控制、任务编排、监控告警等功能。平台以**插件形式**组织功能模块，支持**一键安装与卸载**，可根据实际需求灵活扩展。

**🔌 插件化架构，按需加载**

通过插件系统实现功能模块的解耦，Kubernetes 管理、任务中心、监控中心等核心功能均以插件形式提供，团队可根据实际需求选择性启用，降低系统复杂度。

---

## 🌟 核心亮点

### 🔌 插件化架构

- 功能模块以插件形式存在，支持一键安装/卸载
- 前后端插件系统联动，按需加载
- 完整的插件开发规范，易于扩展

### ☸️ 多集群 Kubernetes 管理

- 统一管理多个 Kubernetes 集群
- 完整的工作负载管理：Deployment、StatefulSet、DaemonSet、Job、CronJob
- 网络与存储：Service、Ingress、ConfigMap、Secret、PV/PVC
- Web Terminal 终端连接，支持会话录制与回放
- 集群健康巡检，一键生成巡检报告

### 🔐 精细化权限控制

- 平台级 + Kubernetes 级双重 RBAC
- 资产级权限隔离（查看、编辑、删除、终端、文件）

### 📊 操作审计

- 操作日志完整记录
- SSH 终端会话录制与回放
- 数据变更追溯

---

## 🚀 功能特性

### 基础功能

| 功能模块 | 描述 |
|:---------|:-----|
| 👥 用户管理 | 用户增删改查、密码重置、状态管理 |
| 🎭 角色管理 | 角色定义、权限分配、角色继承 |
| 🏢 部门管理 | 组织架构管理、部门层级 |
| 📋 菜单管理 | 动态菜单配置、权限绑定 |
| 📝 操作审计 | 完整的操作日志记录与查询 |

### 插件功能

#### ☸️ Kubernetes 容器管理

| 功能 | 描述 |
|:-----|:-----|
| 集群管理 | 多集群接入、集群概览、健康检查 |
| 节点管理 | 节点列表、资源监控、污点/标签管理 |
| 工作负载 | Deployment、StatefulSet、DaemonSet、Job 管理 |
| 网络管理 | Service、Ingress、NetworkPolicy 管理 |
| 配置存储 | ConfigMap、Secret、PV/PVC 管理 |
| 终端审计 | Web Terminal、会话录制与回放 |
| 集群巡检 | 一键生成 K8S 巡检报告 |

#### ✅ 任务中心

| 功能 | 描述 |
|:-----|:-----|
| 执行任务 | 脚本执行、批量操作 |
| 模板管理 | 任务模板定义与复用 |
| 文件分发 | 批量文件分发到目标主机 |
| 执行历史 | 任务执行记录与日志查看 |

#### 📊 监控中心

| 功能 | 描述 |
|:-----|:-----|
| 域名监控 | SSL 证书监控、到期提醒 |
| 告警管理 | 告警规则配置、多渠道通知 |

---

## 🛠️ 技术栈

### 后端

| 技术 | 版本 | 描述 |
|:-----|:-----|:-----|
| `Go` | 1.21+ | 后端开发语言 |
| `Gin` | 1.11+ | 高性能 HTTP Web 框架 |
| `GORM` | 1.31+ | Go 语言 ORM 库 |
| `client-go` | 0.35+ | Kubernetes Go 客户端 |
| `jwt-go` | 5.3+ | JWT 认证 |
| `zap` | 1.27+ | 高性能日志库 |

### 前端

| 技术 | 版本 | 描述 |
|:-----|:-----|:-----|
| `Vue` | 3.5+ | 渐进式 JavaScript 框架 |
| `TypeScript` | 5.9+ | 类型安全的 JavaScript |
| `Element Plus` | 2.13+ | Vue 3 UI 组件库 |
| `Vite` | 5.4+ | 下一代前端构建工具 |
| `xterm.js` | 6.0+ | Web 终端模拟器 |

---

## 📦 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- MySQL 8.0+
- Redis 6.0+

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

### 4. 启动服务

```bash
# 启动后端
go run main.go server

# 启动前端（新终端）
cd web && npm install && npm run dev
```

### 5. 访问系统

- 前端地址：http://localhost:5173
- 后端 API：http://localhost:9876
- Swagger 文档：http://localhost:9876/swagger/index.html

### 默认账号

| 用户名 | 密码 |
|:-------|:-----|
| `admin` | `123456` |

> ⚠️ **重要**: 生产环境请立即修改默认密码！

---

## 🚢 部署方式

### 方式一：Docker Compose 部署（推荐）

最简单的部署方式，一键启动所有服务。

```bash
# 1. 克隆项目
git clone https://github.com/ydcloud-dy/opshub.git
cd opshub

# 2. 创建环境变量文件（可选）
cp .env.example .env
# 编辑 .env 文件修改配置

# 3. 启动服务
docker-compose up -d

# 4. 查看服务状态
docker-compose ps

# 5. 查看日志
docker-compose logs -f
```

**访问地址：**
| 服务 | 地址 |
|:-----|:-----|
| 前端 | http://localhost:3000 |
| 后端 API | http://localhost:9876 |
| Swagger 文档 | http://localhost:9876/swagger/index.html |

```bash
# 停止服务
docker-compose down

# 停止并删除数据卷
docker-compose down -v
```

---

### 方式二：脚本一键部署

提供一键部署脚本，适合快速部署到服务器。

```bash
# 下载并执行安装脚本
curl -fsSL https://raw.githubusercontent.com/ydcloud-dy/opshub/main/scripts/install.sh | bash

# 或者手动下载后执行
curl -fsSL https://raw.githubusercontent.com/ydcloud-dy/opshub/main/scripts/install.sh -o install.sh
chmod +x install.sh
./install.sh
```

**脚本支持的参数：**

```bash
# 指定安装目录
./install.sh --install-dir /opt/opshub

# 指定数据库配置
./install.sh --db-host 127.0.0.1 --db-password your-password

# 跳过依赖检查
./install.sh --skip-deps

# 查看帮助
./install.sh --help
```

---

### 方式三：Kubernetes 部署

适合生产环境的容器化部署。

#### 使用 YAML 部署

```bash
# 1. 创建命名空间
kubectl create namespace opshub

# 2. 创建配置密钥
kubectl create secret generic opshub-secrets \
  --from-literal=db-password=your-db-password \
  --from-literal=jwt-secret=your-jwt-secret \
  -n opshub

# 3. 部署应用
kubectl apply -f deploy/kubernetes/ -n opshub

# 4. 查看部署状态
kubectl get pods -n opshub

# 5. 查看服务
kubectl get svc -n opshub
```

**Deployment 示例：**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: opshub
  labels:
    app: opshub
spec:
  replicas: 2
  selector:
    matchLabels:
      app: opshub
  template:
    metadata:
      labels:
        app: opshub
    spec:
      containers:
      - name: opshub
        image: opshub:latest
        ports:
        - containerPort: 9876
        env:
        - name: OPSHUB_SERVER_MODE
          value: "release"
        - name: OPSHUB_DATABASE_HOST
          value: "mysql-service"
        - name: OPSHUB_DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: opshub-secrets
              key: db-password
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        readinessProbe:
          httpGet:
            path: /api/health
            port: 9876
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: opshub-service
spec:
  selector:
    app: opshub
  ports:
  - port: 80
    targetPort: 9876
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: opshub-ingress
spec:
  rules:
  - host: opshub.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: opshub-service
            port:
              number: 80
```

#### 使用 Helm 部署

```bash
# 添加 Helm 仓库
helm repo add opshub https://charts.opshub.io
helm repo update

# 安装
helm install opshub opshub/opshub \
  --namespace opshub \
  --create-namespace \
  --set database.host=mysql-service \
  --set database.password=your-password \
  --set server.jwtSecret=your-jwt-secret

# 使用自定义 values.yaml
helm install opshub opshub/opshub \
  --namespace opshub \
  --create-namespace \
  -f values.yaml

# 升级
helm upgrade opshub opshub/opshub -n opshub

# 卸载
helm uninstall opshub -n opshub
```

**values.yaml 示例：**

```yaml
replicaCount: 2

image:
  repository: opshub
  tag: latest
  pullPolicy: IfNotPresent

server:
  mode: release
  httpPort: 9876
  jwtSecret: "your-jwt-secret"

database:
  host: mysql-service
  port: 3306
  database: opshub
  username: root
  password: "your-password"

redis:
  host: redis-service
  port: 6379
  password: ""

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: opshub.example.com
      paths:
        - path: /
          pathType: Prefix

resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

---

### 方式四：Docker 单独部署

单独使用 Docker 部署后端服务。

```bash
# 构建镜像
docker build -t opshub:latest .

# 运行容器
docker run -d \
  --name opshub \
  -p 9876:9876 \
  -e OPSHUB_SERVER_MODE=release \
  -e OPSHUB_DATABASE_HOST=your-mysql-host \
  -e OPSHUB_DATABASE_PASSWORD=your-password \
  -e OPSHUB_REDIS_HOST=your-redis-host \
  -e OPSHUB_SERVER_JWT_SECRET=your-jwt-secret \
  opshub:latest

# 查看日志
docker logs -f opshub

# 停止并删除
docker stop opshub && docker rm opshub
```

---

## 📖 项目文档

| 文档 | 链接 |
|:-----|:-----|
| 📘 数据库初始化 | [migrations/README.md](migrations/README.md) |
| 📗 Kubernetes 插件 | [docs/plugins/kubernetes.md](docs/plugins/kubernetes.md) |
| 📙 任务中心插件 | [docs/plugins/task.md](docs/plugins/task.md) |
| 📕 监控中心插件 | [docs/plugins/monitor.md](docs/plugins/monitor.md) |

---

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                      👥 用户层                               │
│                  (浏览器 / API Client)                       │
└─────────────────────────┬───────────────────────────────────┘
                          │ HTTP/HTTPS
┌─────────────────────────▼───────────────────────────────────┐
│                    🎨 前端应用                               │
│            Vue 3 + TypeScript + Element Plus                 │
│  ┌──────────────┬──────────────┬──────────────┐            │
│  │  Kubernetes  │    Task      │   Monitor    │            │
│  │    插件      │    插件       │    插件      │            │
│  └──────────────┴──────────────┴──────────────┘            │
└─────────────────────────┬───────────────────────────────────┘
                          │ API
┌─────────────────────────▼───────────────────────────────────┐
│                    ⚙️ 后端服务                               │
│                  Go + Gin + GORM                             │
│  ┌──────────────┬──────────────┬──────────────┐            │
│  │  JWT 认证    │  RBAC 权限   │   审计日志   │            │
│  └──────────────┴──────────────┴──────────────┘            │
│  ┌──────────────┬──────────────┬──────────────┐            │
│  │  Kubernetes  │    Task      │   Monitor    │            │
│  │    插件      │    插件       │    插件      │            │
│  └──────────────┴──────────────┴──────────────┘            │
└───────┬─────────────────┬───────────────────┬───────────────┘
        │                 │                   │
┌───────▼────────┐ ┌──────▼──────┐ ┌─────────▼─────────┐
│   🗄️ MySQL    │ │  ⚡ Redis   │ │  ☸️ Kubernetes   │
│   数据持久化   │ │  缓存/会话  │ │     集群          │
└────────────────┘ └─────────────┘ └───────────────────┘
```

---

## 📁 项目结构

```
opshub/
├── cmd/                    # 命令行入口
├── config/                 # 配置文件
├── internal/               # 核心模块
│   ├── biz/               # 业务逻辑层
│   ├── data/              # 数据访问层
│   ├── plugin/            # 插件系统
│   └── server/            # HTTP 服务
├── plugins/                # 插件目录
│   ├── kubernetes/        # K8S 管理插件
│   ├── task/              # 任务中心插件
│   └── monitor/           # 监控中心插件
├── migrations/             # 数据库脚本
├── web/                    # 前端代码
│   ├── src/
│   │   ├── plugins/       # 前端插件
│   │   ├── views/         # 页面视图
│   │   └── api/           # API 请求
│   └── package.json
├── docker-compose.yml
├── Dockerfile
└── main.go
```

---

## 🔧 环境变量

| 变量名 | 描述 | 默认值 |
|:-------|:-----|:-------|
| `OPSHUB_SERVER_MODE` | 运行模式 | `debug` |
| `OPSHUB_SERVER_HTTP_PORT` | HTTP 端口 | `9876` |
| `OPSHUB_SERVER_JWT_SECRET` | JWT 密钥 | - |
| `OPSHUB_DATABASE_HOST` | MySQL 地址 | `127.0.0.1` |
| `OPSHUB_DATABASE_PORT` | MySQL 端口 | `3306` |
| `OPSHUB_DATABASE_PASSWORD` | MySQL 密码 | - |
| `OPSHUB_REDIS_HOST` | Redis 地址 | `127.0.0.1` |
| `OPSHUB_REDIS_PORT` | Redis 端口 | `6379` |

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

---

## 📄 许可证

本项目采用 [MIT License](LICENSE) 开源许可证。

---

## 📞 联系方式

- 📮 Issue: [GitHub Issues](https://github.com/ydcloud-dy/opshub/issues)
- 📧 Email: support@opshub.io

---

<p align="center">
  <b>如果觉得项目有帮助，欢迎 Star ⭐ 支持！</b>
</p>
