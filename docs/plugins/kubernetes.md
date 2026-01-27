# Kubernetes 容器管理插件

<p align="center">
  <img src="https://img.shields.io/badge/Plugin-Kubernetes-326CE5?style=flat&logo=kubernetes" alt="Kubernetes">
  <img src="https://img.shields.io/badge/Version-1.0.0-blue?style=flat" alt="Version">
  <img src="https://img.shields.io/badge/Status-Stable-green?style=flat" alt="Status">
</p>

---

## 概述

Kubernetes 插件是 OpsHub 的核心插件，提供完整的多集群 Kubernetes 管理能力，支持集群管理、工作负载、网络配置、存储管理、终端审计、应用诊断等功能。

---

## 功能特性

### 集群管理

| 功能 | 描述 |
|:-----|:-----|
| 多集群接入 | 支持接入多个 Kubernetes 集群统一管理 |
| 集群概览 | 查看集群资源使用情况、节点状态、Pod 分布 |
| 健康检查 | 实时检测集群连接状态和 API 可用性 |
| 集群事件 | 查看集群级别的事件和告警信息 |

### 节点管理

| 功能 | 描述 |
|:-----|:-----|
| 节点列表 | 查看所有节点的详细信息和状态 |
| 资源监控 | 实时监控 CPU、内存、磁盘使用情况 |
| 污点管理 | 添加、删除节点污点 (Taint) |
| 标签管理 | 管理节点标签 (Label) |
| 节点调度 | 设置节点为可调度或不可调度状态 |

### 命名空间管理

| 功能 | 描述 |
|:-----|:-----|
| 命名空间列表 | 查看所有命名空间及其资源配额 |
| 创建/删除 | 创建和删除命名空间 |
| 资源配额 | 设置命名空间的 CPU、内存限制 |
| 网络策略 | 配置命名空间的网络隔离策略 |

### 工作负载管理

支持以下 Kubernetes 工作负载类型的完整管理：

| 类型 | 描述 | 操作 |
|:-----|:-----|:-----|
| **Deployment** | 无状态应用部署 | 创建、编辑、删除、扩缩容、滚动更新、回滚 |
| **StatefulSet** | 有状态应用部署 | 创建、编辑、删除、扩缩容 |
| **DaemonSet** | 节点级应用部署 | 创建、编辑、删除、更新策略 |
| **Job** | 一次性任务 | 创建、删除、查看日志 |
| **CronJob** | 定时任务 | 创建、编辑、删除、暂停/恢复 |
| **Pod** | 容器组 | 查看详情、日志、终端、删除 |

### 网络管理

| 资源类型 | 功能 |
|:---------|:-----|
| **Service** | 服务创建、编辑、删除，支持 ClusterIP/NodePort/LoadBalancer |
| **Ingress** | 入口规则配置、TLS 证书管理、路由规则 |
| **NetworkPolicy** | 网络策略配置、入站/出站规则 |

### 配置管理

| 资源类型 | 功能 |
|:---------|:-----|
| **ConfigMap** | 配置映射的创建、编辑、删除、数据查看 |
| **Secret** | 密钥管理，支持 Opaque/TLS/dockerconfigjson 等类型 |

### 存储管理

| 资源类型 | 功能 |
|:---------|:-----|
| **PersistentVolume** | 持久卷管理、容量查看、回收策略 |
| **PersistentVolumeClaim** | 持久卷声明、存储申请、绑定状态 |
| **StorageClass** | 存储类配置、动态供给 |

### 访问控制

| 资源类型 | 功能 |
|:---------|:-----|
| **ServiceAccount** | 服务账户管理、Token 生成 |
| **Role/ClusterRole** | 角色定义、权限规则配置 |
| **RoleBinding/ClusterRoleBinding** | 角色绑定、用户/组授权 |

### 终端审计

| 功能 | 描述 |
|:-----|:-----|
| Web Terminal | 通过浏览器直接登录 Pod 容器执行命令 |
| 会话录制 | 自动录制所有终端操作，支持回放 |
| 操作审计 | 完整记录用户、时间、操作内容 |
| 日志查看 | 实时查看容器日志输出 |

### 集群巡检

| 功能 | 描述 |
|:-----|:-----|
| 一键巡检 | 自动检查集群健康状态 |
| 巡检报告 | 生成详细的巡检报告（评分、问题、建议） |
| 历史记录 | 保存巡检历史，支持对比分析 |
| 检查项 | 节点状态、Pod 状态、资源使用、安全配置等 |

### 应用诊断 (Arthas)

针对 Java 应用提供强大的在线诊断能力：

| 功能 | 描述 |
|:-----|:-----|
| 线程分析 | 查看线程列表、死锁检测、线程堆栈 |
| JVM 信息 | 查看 JVM 参数、内存使用、GC 状态 |
| 火焰图 | 生成 CPU 火焰图，定位性能热点 |
| 方法追踪 | 追踪方法调用链、执行耗时 |
| 反编译 | 在线反编译类文件 |

---

## 安装与启用

### 通过管理界面启用

1. 登录 OpsHub 系统
2. 进入「插件管理」-「插件列表」
3. 找到「Kubernetes」插件
4. 点击「启用」按钮
5. 刷新页面，左侧菜单出现「容器管理」

### 通过 API 启用

```bash
# 启用插件
curl -X POST http://localhost:9876/api/v1/plugins/kubernetes/enable \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 禁用插件
curl -X POST http://localhost:9876/api/v1/plugins/kubernetes/disable \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 查看插件状态
curl http://localhost:9876/api/v1/plugins \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

## 使用指南

### 添加 Kubernetes 集群

#### 步骤

1. 进入「容器管理」-「集群管理」
2. 点击「添加集群」按钮
3. 填写集群信息：
   - **集群名称**: 用于识别的名称
   - **API Server**: Kubernetes API 地址（如 `https://192.168.1.100:6443`）
   - **KubeConfig**: 上传或粘贴 kubeconfig 文件内容
4. 点击「测试连接」验证配置
5. 点击「保存」完成添加

#### 获取 KubeConfig

```bash
# 方式一：从服务器复制
cat ~/.kube/config

# 方式二：使用 kubectl 导出
kubectl config view --raw

# 方式三：为特定用户生成
kubectl config view --minify --flatten
```

### 管理工作负载

#### 创建 Deployment

1. 进入「容器管理」-「工作负载」
2. 选择目标集群和命名空间
3. 点击「创建 Deployment」
4. 填写配置：
   - 名称、副本数
   - 容器镜像、端口
   - 资源限制（CPU、内存）
   - 环境变量、挂载卷
5. 点击「创建」

#### 扩缩容

1. 在 Deployment 列表找到目标应用
2. 点击「扩缩容」按钮
3. 调整副本数量
4. 确认执行

#### 滚动更新

1. 点击目标 Deployment 的「编辑」
2. 修改镜像版本或其他配置
3. 保存后自动触发滚动更新
4. 在详情页查看更新进度

#### 回滚

1. 进入 Deployment 详情页
2. 点击「回滚」按钮
3. 选择要回滚到的版本
4. 确认执行

### 使用 Web Terminal

#### 连接到 Pod

1. 进入「容器管理」-「工作负载」
2. 找到目标 Pod
3. 点击「终端」按钮
4. 选择容器（如果 Pod 有多个容器）
5. 在 Web 终端中执行命令

#### 查看终端审计

1. 进入「容器管理」-「终端审计」
2. 查看所有终端会话记录
3. 点击「回放」查看操作录像
4. 支持按用户、时间、Pod 筛选

### 使用 Arthas 诊断

#### 启动诊断

1. 进入「容器管理」-「应用诊断」
2. 选择集群、命名空间、Pod
3. 点击「启动诊断」
4. 等待 Arthas 注入完成

#### 常用诊断命令

```bash
# 查看线程列表
thread

# 查看最忙的前 N 个线程
thread -n 3

# 查看死锁
thread -b

# 生成火焰图
profiler start
# ... 等待采样 ...
profiler stop --format html

# 查看 JVM 信息
jvm

# 查看内存使用
memory

# 追踪方法调用
trace com.example.UserService getUser

# 监控方法调用
monitor com.example.UserService getUser -c 5

# 反编译类
jad com.example.UserService
```

### 执行集群巡检

1. 进入「容器管理」-「集群巡检」
2. 选择要巡检的集群
3. 点击「开始巡检」
4. 等待巡检完成（通常 1-3 分钟）
5. 查看巡检报告

#### 巡检项目

| 检查类别 | 检查项 |
|:---------|:-------|
| 节点健康 | 节点状态、资源使用、磁盘压力、内存压力 |
| Pod 状态 | 运行状态、重启次数、资源限制 |
| 资源配额 | 命名空间配额、资源使用率 |
| 网络检查 | Service 端点、Ingress 配置 |
| 安全检查 | RBAC 配置、特权容器、敏感挂载 |
| 最佳实践 | 资源限制、健康检查、镜像标签 |

---

## 数据库表

Kubernetes 插件使用以下数据库表：

| 表名 | 说明 |
|:-----|:-----|
| `k8s_clusters` | 集群信息 |
| `k8s_user_kube_configs` | 用户 kubeconfig 配置 |
| `k8s_user_role_bindings` | 用户 K8S 角色绑定 |
| `k8s_cluster_inspections` | 集群巡检记录 |
| `k8s_terminal_sessions` | 终端会话录制 |

---

## 权限配置

### 平台级权限

在「系统管理」-「角色管理」中配置：

| 权限 | 说明 |
|:-----|:-----|
| 集群管理 | 添加、编辑、删除集群 |
| 节点管理 | 查看、管理节点 |
| 工作负载 | 管理 Deployment、StatefulSet 等 |
| 终端访问 | 使用 Web Terminal |
| 集群巡检 | 执行巡检、查看报告 |

### Kubernetes 级权限

通过「访问控制」功能配置 K8S RBAC：

```yaml
# 示例：只读角色
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: readonly
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
```

---

## 常见问题

### Q: 集群连接失败怎么办？

**A:** 检查以下几点：
1. API Server 地址是否正确
2. KubeConfig 是否有效
3. 网络是否能访问 API Server
4. 证书是否过期

```bash
# 测试连接
kubectl --kubeconfig=/path/to/config cluster-info
```

### Q: 终端连接超时？

**A:** 可能原因：
1. Pod 所在节点网络不通
2. 容器没有 shell（如 distroless 镜像）
3. 防火墙拦截了 WebSocket 连接

### Q: Arthas 诊断启动失败？

**A:** 检查：
1. 目标 Pod 必须是 Java 应用
2. 容器需要有足够的资源
3. 确保有 /tmp 目录的写入权限

### Q: 如何查看历史终端操作？

**A:** 进入「容器管理」-「终端审计」，所有终端会话自动录制保存，支持回放查看。

---

## 相关文档

- [Kubernetes 官方文档](https://kubernetes.io/docs/)
- [kubectl 命令参考](https://kubernetes.io/docs/reference/kubectl/)
- [Arthas 用户指南](https://arthas.aliyun.com/doc/)
- [OpsHub 主文档](../../README.md)
- [部署指南](../deployment.md)
