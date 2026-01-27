# 插件 API 参考

本文档定义了 OpsHub 插件系统的所有接口和数据结构。

---

## 目录

- [后端接口](#后端接口)
  - [Plugin 接口](#plugin-接口)
  - [MenuConfig 结构](#menuconfig-结构)
  - [插件管理 API](#插件管理-api)
- [前端接口](#前端接口)
  - [Plugin 接口](#前端-plugin-接口)
  - [PluginMenuConfig 类型](#pluginmenuconfig-类型)
  - [PluginRouteConfig 类型](#pluginrouteconfig-类型)
- [HTTP API 规范](#http-api-规范)

---

## 后端接口

### Plugin 接口

所有后端插件必须实现此接口：

```go
// internal/plugin/plugin.go

type Plugin interface {
    // 元信息
    Name() string        // 插件唯一标识符
    Description() string // 插件描述
    Version() string     // 插件版本
    Author() string      // 插件作者

    // 生命周期
    Enable(db *gorm.DB) error   // 启用插件
    Disable(db *gorm.DB) error  // 禁用插件

    // 功能注册
    RegisterRoutes(router *gin.RouterGroup, db *gorm.DB)  // 注册路由
    GetMenus() []MenuConfig                                // 获取菜单配置
}
```

#### 方法说明

| 方法 | 返回类型 | 说明 |
|:-----|:---------|:-----|
| `Name()` | `string` | 返回插件唯一标识符，用于路由路径和数据库记录。只能包含小写字母、数字和连字符。 |
| `Description()` | `string` | 返回插件的人类可读描述。 |
| `Version()` | `string` | 返回插件版本，建议使用语义化版本格式（如 `1.0.0`）。 |
| `Author()` | `string` | 返回插件作者或维护者信息。 |
| `Enable(db)` | `error` | 插件启用时调用。用于初始化数据库表、加载配置等。返回错误将阻止插件启用。 |
| `Disable(db)` | `error` | 插件禁用时调用。用于清理资源、停止后台任务。返回错误将被记录但不会阻止禁用。 |
| `RegisterRoutes(router, db)` | - | 注册 HTTP 路由。`router` 已挂载到 `/api/v1/plugins/{name}` 路径下。 |
| `GetMenus()` | `[]MenuConfig` | 返回菜单配置数组，用于动态生成前端菜单。 |

#### 实现示例

```go
package myplugin

import (
    "github.com/gin-gonic/gin"
    "github.com/ydcloud-dy/opshub/internal/plugin"
    "gorm.io/gorm"
)

type Plugin struct {
    db *gorm.DB
}

func New() *Plugin {
    return &Plugin{}
}

func (p *Plugin) Name() string        { return "myplugin" }
func (p *Plugin) Description() string { return "我的自定义插件" }
func (p *Plugin) Version() string     { return "1.0.0" }
func (p *Plugin) Author() string      { return "开发者" }

func (p *Plugin) Enable(db *gorm.DB) error {
    p.db = db
    return db.AutoMigrate(&MyModel{})
}

func (p *Plugin) Disable(db *gorm.DB) error {
    return nil
}

func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    router.GET("/hello", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello"})
    })
}

func (p *Plugin) GetMenus() []plugin.MenuConfig {
    return []plugin.MenuConfig{
        {Name: "我的插件", Path: "/myplugin", Icon: "Setting", Sort: 90},
    }
}
```

---

### MenuConfig 结构

菜单配置结构：

```go
// internal/plugin/menu.go

type MenuConfig struct {
    Name       string       `json:"name"`        // 菜单显示名称
    Path       string       `json:"path"`        // 路由路径
    Icon       string       `json:"icon"`        // Element Plus 图标名称
    Sort       int          `json:"sort"`        // 排序值（越小越靠前）
    Hidden     bool         `json:"hidden"`      // 是否隐藏
    ParentPath string       `json:"parent_path"` // 父菜单路径
    Children   []MenuConfig `json:"children"`    // 子菜单
}
```

#### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|:-----|:-----|:-----|:-----|
| `Name` | `string` | 是 | 菜单显示名称，支持中文 |
| `Path` | `string` | 是 | 路由路径，必须以 `/` 开头 |
| `Icon` | `string` | 否 | Element Plus 图标名称，如 `Setting`、`Document` |
| `Sort` | `int` | 否 | 排序值，默认为 0，越小越靠前 |
| `Hidden` | `bool` | 否 | 是否隐藏菜单，默认 `false` |
| `ParentPath` | `string` | 否 | 父菜单路径，用于嵌套菜单 |
| `Children` | `[]MenuConfig` | 否 | 子菜单配置数组 |

#### 常用图标

| 图标名称 | 说明 |
|:---------|:-----|
| `HomeFilled` | 首页 |
| `Setting` | 设置 |
| `Document` | 文档 |
| `List` | 列表 |
| `User` | 用户 |
| `Tools` | 工具 |
| `Monitor` | 监控 |
| `Menu` | 菜单 |
| `Connection` | 连接 |
| `Upload` | 上传 |
| `Download` | 下载 |
| `Files` | 文件 |
| `Search` | 搜索 |

完整图标列表：[Element Plus Icons](https://element-plus.org/zh-CN/component/icon.html)

#### 菜单排序参考

| 插件 | Sort 值 |
|:-----|:--------|
| 仪表盘 | 1 |
| 资产管理 | 10 |
| 容器管理 | 20 |
| 任务中心 | 30 |
| 监控中心 | 40 |
| 系统管理 | 80 |
| 新插件推荐 | 90-95 |

---

### 插件管理 API

系统提供的插件管理 API：

#### 获取插件列表

```
GET /api/v1/plugins
```

响应：

```json
{
    "code": 0,
    "message": "success",
    "data": [
        {
            "name": "kubernetes",
            "description": "Kubernetes 容器管理",
            "version": "1.0.0",
            "author": "OpsHub",
            "enabled": true
        }
    ]
}
```

#### 获取插件详情

```
GET /api/v1/plugins/:name
```

响应：

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "name": "kubernetes",
        "description": "Kubernetes 容器管理",
        "version": "1.0.0",
        "author": "OpsHub",
        "enabled": true,
        "menus": [...]
    }
}
```

#### 启用插件

```
POST /api/v1/plugins/:name/enable
```

响应：

```json
{
    "code": 0,
    "message": "插件已启用"
}
```

#### 禁用插件

```
POST /api/v1/plugins/:name/disable
```

响应：

```json
{
    "code": 0,
    "message": "插件已禁用"
}
```

---

## 前端接口

### 前端 Plugin 接口

```typescript
// web/src/plugins/types.ts

export interface Plugin {
    // 元信息
    name: string          // 插件唯一标识符
    description: string   // 插件描述
    version: string       // 插件版本
    author: string        // 插件作者

    // 生命周期
    install(): Promise<void>    // 安装/初始化
    uninstall(): Promise<void>  // 卸载/清理

    // 配置
    getMenus(): PluginMenuConfig[]    // 获取菜单配置
    getRoutes(): PluginRouteConfig[]  // 获取路由配置
}
```

#### 方法说明

| 方法 | 返回类型 | 说明 |
|:-----|:---------|:-----|
| `install()` | `Promise<void>` | 插件初始化时调用，用于加载配置、注册事件等 |
| `uninstall()` | `Promise<void>` | 插件卸载时调用，用于清理资源、移除事件监听 |
| `getMenus()` | `PluginMenuConfig[]` | 返回菜单配置数组 |
| `getRoutes()` | `PluginRouteConfig[]` | 返回路由配置数组 |

---

### PluginMenuConfig 类型

```typescript
// web/src/plugins/types.ts

export interface PluginMenuConfig {
    name: string           // 菜单显示名称
    path: string           // 路由路径
    icon: string           // 图标名称
    sort: number           // 排序值
    hidden: boolean        // 是否隐藏
    parentPath: string     // 父菜单路径
    children?: PluginMenuConfig[]  // 子菜单
}
```

#### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|:-----|:-----|:-----|:-----|
| `name` | `string` | 是 | 菜单显示名称 |
| `path` | `string` | 是 | 路由路径 |
| `icon` | `string` | 是 | Element Plus 图标名称 |
| `sort` | `number` | 是 | 排序值，越小越靠前 |
| `hidden` | `boolean` | 是 | 是否隐藏 |
| `parentPath` | `string` | 是 | 父菜单路径，顶级菜单为空字符串 |
| `children` | `PluginMenuConfig[]` | 否 | 子菜单配置 |

---

### PluginRouteConfig 类型

```typescript
// web/src/plugins/types.ts

export interface PluginRouteConfig {
    path: string                              // 路由路径
    name: string                              // 路由名称（唯一）
    component: () => Promise<any>             // 组件懒加载
    redirect?: string                         // 重定向路径
    meta?: {
        title: string                         // 页面标题
        icon?: string                         // 图标
        hidden?: boolean                      // 是否隐藏
        keepAlive?: boolean                   // 是否缓存
        affix?: boolean                       // 是否固定标签
        breadcrumb?: boolean                  // 是否显示面包屑
    }
    children?: PluginRouteConfig[]            // 子路由
}
```

#### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|:-----|:-----|:-----|:-----|
| `path` | `string` | 是 | 路由路径 |
| `name` | `string` | 是 | 路由名称，必须全局唯一 |
| `component` | `() => Promise<any>` | 是 | 组件懒加载函数 |
| `redirect` | `string` | 否 | 重定向路径 |
| `meta` | `object` | 否 | 路由元信息 |
| `children` | `PluginRouteConfig[]` | 否 | 子路由配置 |

#### meta 字段说明

| 字段 | 类型 | 说明 |
|:-----|:-----|:-----|
| `title` | `string` | 页面标题，显示在标签页和面包屑 |
| `icon` | `string` | Element Plus 图标名称 |
| `hidden` | `boolean` | 是否在菜单中隐藏 |
| `keepAlive` | `boolean` | 是否启用页面缓存 |
| `affix` | `boolean` | 是否固定在标签栏 |
| `breadcrumb` | `boolean` | 是否显示在面包屑中 |

---

### PluginManager API

```typescript
// web/src/plugins/manager.ts

class PluginManager {
    // 注册插件
    register(plugin: Plugin): void

    // 获取所有插件
    getPlugins(): Plugin[]

    // 获取指定插件
    getPlugin(name: string): Plugin | undefined

    // 获取所有菜单
    getAllMenus(): PluginMenuConfig[]

    // 获取所有路由
    getAllRoutes(): PluginRouteConfig[]

    // 安装所有插件
    installAll(): Promise<void>

    // 卸载所有插件
    uninstallAll(): Promise<void>
}

// 导出单例
export const pluginManager: PluginManager
```

#### 使用示例

```typescript
import { pluginManager } from '@/plugins/manager'
import MyPlugin from '@/plugins/myplugin'

// 注册插件
pluginManager.register(MyPlugin)

// 获取所有菜单（在路由配置中使用）
const menus = pluginManager.getAllMenus()

// 获取所有路由（在路由配置中使用）
const routes = pluginManager.getAllRoutes()
```

---

## HTTP API 规范

### 请求格式

#### URL 规范

```
{HTTP_METHOD} /api/v1/plugins/{plugin_name}/{resource}[/{id}]
```

示例：

| 操作 | URL |
|:-----|:----|
| 获取列表 | `GET /api/v1/plugins/myplugin/items` |
| 获取详情 | `GET /api/v1/plugins/myplugin/items/123` |
| 创建 | `POST /api/v1/plugins/myplugin/items` |
| 更新 | `PUT /api/v1/plugins/myplugin/items/123` |
| 删除 | `DELETE /api/v1/plugins/myplugin/items/123` |

#### 请求头

| Header | 值 | 说明 |
|:-------|:---|:-----|
| `Content-Type` | `application/json` | 请求体格式 |
| `Authorization` | `Bearer {token}` | JWT 认证令牌 |

#### 分页参数

| 参数 | 类型 | 默认值 | 说明 |
|:-----|:-----|:-------|:-----|
| `page` | `int` | 1 | 页码 |
| `page_size` | `int` | 10 | 每页数量 |
| `keyword` | `string` | - | 搜索关键字 |
| `sort_by` | `string` | `id` | 排序字段 |
| `sort_order` | `string` | `desc` | 排序方向 |

### 响应格式

#### 成功响应

```json
{
    "code": 0,
    "message": "success",
    "data": { ... }
}
```

#### 错误响应

```json
{
    "code": 400,
    "message": "参数错误: name 不能为空"
}
```

#### 分页响应

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [ ... ],
        "total": 100,
        "page": 1,
        "page_size": 10
    }
}
```

### 状态码

| code | HTTP Status | 说明 |
|:-----|:------------|:-----|
| 0 | 200 | 成功 |
| 400 | 200 | 参数错误 |
| 401 | 401 | 未授权 |
| 403 | 403 | 权限不足 |
| 404 | 200 | 资源不存在 |
| 500 | 200 | 服务器错误 |

### 通用响应结构

```go
// 后端响应结构
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// 分页数据
type PageData struct {
    List     interface{} `json:"list"`
    Total    int64       `json:"total"`
    Page     int         `json:"page"`
    PageSize int         `json:"page_size"`
}
```

```typescript
// 前端类型定义
interface Response<T = any> {
    code: number
    message: string
    data?: T
}

interface PageData<T> {
    list: T[]
    total: number
    page: number
    page_size: number
}
```

---

## 相关文档

- [快速开始](quick-start.md) - 5 分钟创建简单插件
- [完整开发指南](development-guide.md) - 详细开发流程
- [高级主题](advanced-topics.md) - 菜单排序、权限控制等
