# Grafana Loki 日志查询指南

## 目录

- [环境与数据源](#环境与数据源)
- [基础查询](#基础查询)
- [标签筛选](#标签筛选)
- [内容筛选](#内容筛选)
- [聚合操作](#聚合操作)
- [常见查询场景](#常见查询场景)
- [查询技巧](#查询技巧)

---

## 环境与数据源

Grafana 已配置三个 Loki 数据源，对应不同的 Kubernetes 集群环境：

| 环境 | 数据源名称 | 用途 |
|------|-----------|------|
| **Test** | loki-test | 测试环境日志查询 |
| **UAT**  | loki-uat  | 预发布环境日志查询 |
| **Prod** | loki-prod | 生产环境日志查询 |

**切换数据源步骤：**
1. 在 Grafana Explore 页面
2. 点击左上角数据源下拉框
3. 选择对应环境的 Loki 数据源

---

## 基础查询

### 1. 查询所有日志

```
{job="varlogcontainers"}
```

### 2. 查询特定命名空间的日志

```
{job="varlogcontainers", namespace="default"}
```

### 3. 查询特定 Pod 的日志

```
{job="varlogcontainers", namespace="default", pod="my-app-xxx"}
```

### 4. 查询特定容器的日志

```
{job="varlogcontainers", namespace="default", pod="my-app-xxx", container="my-container"}
```

---

## 标签筛选

标签是日志流的索引，查询时使用标签选择器可以快速定位日志。

### 常用标签

| 标签 | 说明 | 示例 |
|------|------|------|
| `job` | 日志采集任务 | `varlogcontainers` |
| `namespace` | K8s 命名空间 | `default`, `production` |
| `pod` | Pod 名称 | `app-deployment-xxx` |
| `container` | 容器名称 | `app-container` |
| `app` | 应用名称（通过标签传递） | `myapp` |
| `node` | 节点名称 | `node-1` |

### 标签匹配操作符

| 操作符 | 说明 | 示例 |
|--------|------|------|
| `=` | 完全匹配 | `{namespace="default"}` |
| `!=` | 不等于 | `{namespace!="kube-system"}` |
| `=~` | 正则匹配 | `{pod=~"app-.*"}` |
| `!~` | 正则不匹配 | `{pod!~"debug-.*"}` |

### 示例

```
# 查询所有非 kube-system 的日志
{job="varlogcontainers", namespace!="kube-system"}

# 使用正则查询多个命名空间
{job="varlogcontainers", namespace=~"default|production"}

# 查询 Pod 名称前缀为 app 的日志
{job="varlogcontainers", pod=~"app-.*"}
```

---

## 内容筛选

内容筛选用于在日志行中查找特定内容。

### 行过滤操作符

| 操作符 | 说明 | 示例 |
|--------|------|------|
| `\|=` | 包含字符串 | `{...} |= "error"` |
| `!=`   | 不包含字符串 | `{...} != "debug"` |
| `\|~`  | 正则匹配 | `{...} \|=~ "error.*timeout"` |
| `!~`   | 正则不匹配 | `{...} !~ "healthcheck"` |

### 示例

```
# 查询包含错误信息的日志
{job="varlogcontainers", namespace="default"} |= "error"

# 查询包含 HTTP 5xx 错误的日志
{job="varlogcontainers"} |=~ "5[0-9]{2}"

# 查询同时包含 "ERROR" 和 "database" 的日志
{job="varlogcontainers"} |= "ERROR" |= "database"

# 查询包含 "error" 但排除 "heartbeat" 的日志
{job="varlogcontainers"} |= "error" != "heartbeat"
```

---

## 聚合操作

### 1. 统计操作

```
# 统计日志行数
count_over_time({job="varlogcontainers", namespace="default"}[5m])

# 统计特定错误的出现次数
count_over_time({job="varlogcontainers"} |= "ERROR"[1h])

# 计算错误率
(sum(count_over_time({job="varlogcontainers"} |= "ERROR"[1h])) / sum(count_over_time({job="varlogcontainers"}[1h]))) * 100
```

### 2. 比率计算

```
# 查询包含特定字符串的日志比例
rate({job="varlogcontainers"} |= "timeout"[5m])

# 按 Pod 统计错误率
sum by (pod) (rate({job="varlogcontainers"} |= "ERROR"[5m]))
```

### 3. Top N 查询

```
# 统计日志最多的前 10 个 Pod
topk(10, sum by (pod) (count_over_time({job="varlogcontainers"}[1h])))

# 统计错误最多的 Pod
topk(10, sum by (pod) (count_over_time({job="varlogcontainers"} |= "ERROR"[1h])))
```

---

## 常见查询场景

### 1. 排查应用错误

```
# 查询特定应用的错误日志
{namespace="default", app="myapp"} |= "ERROR" |= "Exception"

# 查询最近的 panic 堆栈
{namespace="production"} |=~ "panic|fatal"
```

### 2. HTTP 请求分析

```
# 查询 4xx 错误
{namespace="default", container="nginx"} |=~ "4[0-9]{2}"

# 查询慢请求（响应时间超过 1s）
{namespace="default"} |=~ "duration.*[1-9][0-9]{3,}"

# 查询特定 API 的调用
{namespace="default"} |= "POST /api/v1/users"
```

### 3. 数据库查询问题

```
# 查询慢 SQL
{namespace="default", container="app"} |= "slow query"

# 查询数据库连接错误
{namespace="default"} |=~ "connection.*refused|timeout"
```

### 4. 追踪请求链路

```
# 根据 trace_id 查询相关日志
{job="varlogcontainers"} |= "trace-id=abc123"

# 查询特定用户的操作
{namespace="default"} |= "user_id=10086"
```

### 5. 监控告警相关

```
# 查询 OOMKilled 事件
{job="varlogcontainers"} |= "OOMKilled"

# 查询 CrashLoopBackOff 相关
{job="varlogcontainers"} |=~ "BackOff|crash"
```

---

## 查询技巧

### 1. 时间范围选择

- **快捷选择**: 点击时间选择器，选择 `Last 5 minutes`, `Last 1 hour` 等
- **自定义**: 直接输入时间范围，如 `now-1h` 到 `now`
- **绝对时间**: 选择具体的起止时间

### 2. 实时日志查看

勾选 **Live** 模式，实时接收新日志。

### 3. 日志格式化

点击日志行右侧的 **Format** 按钮，可以格式化 JSON 日志：

```json
{"level":"error","msg":"database connection failed","time":"2024-01-01T10:00:00Z"}
```

### 4. 提取字段

对于 JSON 格式日志，可以使用 `line_format` 提取字段：

```
{job="varlogcontainers", app="myapp"}
| json
| line_format "{{.level}}: {{.msg}}"
```

### 5. 标签提取

从日志内容中提取标签用于聚合：

```
{job="varlogcontainers"}
| json
| label_format level={{.level}}
```

### 6. 查询性能优化

- **尽量使用标签筛选**，减少搜索范围
- **避免过宽的正则表达式**，如 `.*`
- **合理设置时间范围**，避免查询过长时间段
- **使用 `| json` 解析 JSON 日志**后再过滤

### 7. 常用查询模板

**保存常用查询：**
1. 编写好查询语句
2. 点击右上角 **Save** 保存为查询库
3. 下次直接从查询库加载

---

## 快捷键参考

| 快捷键 | 功能 |
|--------|------|
| `Ctrl + Enter` | 运行查询 |
| `Ctrl + Shift + F` | 全屏显示 |
| `Esc` | 关闭详情/退出全屏 |
| `Ctrl + /` | 查看快捷键帮助 |

---

## 注意事项

1. **权限控制**: 不同环境可能有不同的访问权限
2. **日志保留期**: Loki 日志有保留期限，查询历史日志注意时间范围
3. **查询限制**: 单次查询返回的日志行数有限制，可调整时间范围
4. **性能影响**: 复杂的查询可能影响 Loki 性能，请合理使用

---

## 获取帮助

- 官方文档: https://grafana.com/docs/loki/latest/
- LogQL 参考: https://grafana.com/docs/loki/latest/logql/
- Grafana Explore: https://grafana.com/docs/grafana/latest/explore/

---

**最后更新**: 2026-01-06
**维护团队**: Platform Team
