# OpsHub 插件开发文档

<p align="center">
  <img src="https://img.shields.io/badge/Architecture-Plugin--based-purple?style=flat" alt="Plugin">
  <img src="https://img.shields.io/badge/Backend-Go%201.21+-00ADD8?style=flat&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Frontend-Vue%203-4FC08D?style=flat&logo=vue.js" alt="Vue">
</p>

---

## 文档目录

| 文档 | 说明 | 适合人群 |
|:-----|:-----|:---------|
| [快速开始](quick-start.md) | 5 分钟创建一个简单插件 | 新手开发者 |
| [完整开发指南](development-guide.md) | 详细的插件开发流程和规范 | 所有开发者 |
| [API 参考](api-reference.md) | 插件接口和数据结构定义 | 进阶开发者 |
| [高级主题](advanced-topics.md) | 菜单排序、权限控制、后台任务等 | 进阶开发者 |

---

## 插件系统概述

OpsHub 采用**前后端分离的插件化架构**，允许开发者以插件的形式扩展系统功能。

### 核心特性

| 特性 | 说明 |
|:-----|:-----|
| **模块化** | 每个插件独立开发、测试、部署 |
| **可插拔** | 支持一键启用/禁用，无需重启 |
| **低耦合** | 插件之间互不影响 |
| **易扩展** | 新功能作为插件开发，不修改核心代码 |
| **状态持久化** | 插件启用状态保存到数据库 |

### 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      OpsHub 核心系统                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   后端 (Go)                        前端 (Vue 3)              │
│   ┌─────────────────┐              ┌─────────────────┐       │
│   │  Plugin Manager │◄────────────►│  Plugin Manager │       │
│   │  (plugin.go)    │   REST API   │  (manager.ts)   │       │
│   └────────┬────────┘              └────────┬────────┘       │
│            │                                │                │
│   ┌────────▼────────┐              ┌────────▼────────┐       │
│   │    Plugins      │              │    Plugins      │       │
│   │  ┌───────────┐  │              │  ┌───────────┐  │       │
│   │  │Kubernetes │  │              │  │Kubernetes │  │       │
│   │  ├───────────┤  │              │  ├───────────┤  │       │
│   │  │  Monitor  │  │              │  │  Monitor  │  │       │
│   │  ├───────────┤  │              │  ├───────────┤  │       │
│   │  │   Task    │  │              │  │   Task    │  │       │
│   │  └───────────┘  │              │  └───────────┘  │       │
│   └─────────────────┘              └─────────────────┘       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   MySQL     │
                    │ plugin_states│
                    └─────────────┘
```

### 内置插件

| 插件 | 说明 | 功能 |
|:-----|:-----|:-----|
| **kubernetes** | 容器管理 | 多集群管理、工作负载、终端、巡检、Arthas 诊断 |
| **monitor** | 监控中心 | 域名监控、SSL 证书、告警通知 |
| **task** | 任务中心 | 脚本执行、模板管理、文件分发 |

---

## 插件目录结构

### 后端插件

```
plugins/
└── your-plugin/
    ├── plugin.go          # 插件入口，实现 Plugin 接口
    ├── model/             # 数据模型
    │   └── model.go
    └── server/            # HTTP 服务
        ├── router.go      # 路由定义
        └── handler.go     # 请求处理
```

### 前端插件

```
web/src/
├── plugins/
│   └── your-plugin/
│       └── index.ts       # 插件入口，实现 Plugin 接口
├── views/
│   └── your-plugin/
│       ├── Index.vue      # 主页面
│       └── Detail.vue     # 详情页面
└── api/
    └── your-plugin.ts     # API 请求封装
```

---

## 快速开始

### 1. 创建后端插件

```go
// plugins/hello/plugin.go
package hello

import (
    "github.com/gin-gonic/gin"
    "github.com/ydcloud-dy/opshub/internal/plugin"
    "gorm.io/gorm"
)

type Plugin struct{}

func New() *Plugin {
    return &Plugin{}
}

func (p *Plugin) Name() string        { return "hello" }
func (p *Plugin) Description() string { return "Hello World 示例插件" }
func (p *Plugin) Version() string     { return "1.0.0" }
func (p *Plugin) Author() string      { return "OpsHub" }

func (p *Plugin) Enable(db *gorm.DB) error {
    // 初始化数据库表等
    return nil
}

func (p *Plugin) Disable(db *gorm.DB) error {
    return nil
}

func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    router.GET("/hello", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello from plugin!"})
    })
}

func (p *Plugin) GetMenus() []plugin.MenuConfig {
    return []plugin.MenuConfig{
        {Name: "Hello", Path: "/hello", Icon: "Star", Sort: 100},
    }
}
```

### 2. 注册插件

```go
// internal/server/http.go
import helloplugin "github.com/ydcloud-dy/opshub/plugins/hello"

// 在 NewHTTPServer() 中
s.pluginMgr.Register(helloplugin.New())
```

### 3. 创建前端插件

```typescript
// web/src/plugins/hello/index.ts
import type { Plugin } from '@/plugins/types'
import { pluginManager } from '@/plugins/manager'

class HelloPlugin implements Plugin {
    name = 'hello'
    description = 'Hello World 示例插件'
    version = '1.0.0'
    author = 'OpsHub'

    async install() {}
    async uninstall() {}

    getMenus() {
        return [{ name: 'Hello', path: '/hello', icon: 'Star', sort: 100, hidden: false, parentPath: '' }]
    }

    getRoutes() {
        return [{
            path: '/hello',
            name: 'Hello',
            component: () => import('@/views/hello/Index.vue'),
            meta: { title: 'Hello' }
        }]
    }
}

pluginManager.register(new HelloPlugin())
export default new HelloPlugin()
```

### 4. 导入插件

```typescript
// web/src/main.ts
import '@/plugins/hello'
```

---

## 下一步

- [快速开始](quick-start.md) - 完整的 Hello World 示例
- [完整开发指南](development-guide.md) - 深入了解插件开发
- [API 参考](api-reference.md) - 查看接口定义

---

## 相关文档

- [OpsHub 主文档](../../README.md)
- [部署指南](../deployment.md)
- [Kubernetes 插件](../plugins/kubernetes.md)
- [任务中心插件](../plugins/task.md)
- [监控中心插件](../plugins/monitor.md)
