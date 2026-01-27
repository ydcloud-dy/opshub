# 插件开发完整指南

本文档详细介绍 OpsHub 插件开发的完整流程、规范和最佳实践。

---

## 目录

- [开发环境准备](#开发环境准备)
- [后端插件开发](#后端插件开发)
- [前端插件开发](#前端插件开发)
- [数据库设计](#数据库设计)
- [API 设计规范](#api-设计规范)
- [测试与调试](#测试与调试)
- [发布与部署](#发布与部署)

---

## 开发环境准备

### 环境要求

| 工具 | 版本 | 用途 |
|:-----|:-----|:-----|
| Go | 1.21+ | 后端开发 |
| Node.js | 18+ | 前端开发 |
| MySQL | 8.0+ | 数据存储 |
| Redis | 6.0+ | 缓存（可选） |

### 开发工具推荐

| 工具 | 说明 |
|:-----|:-----|
| VSCode / GoLand | IDE |
| Postman / Insomnia | API 测试 |
| DBeaver | 数据库管理 |
| Git | 版本控制 |

### 项目结构

```
opshub/
├── cmd/                    # 命令行入口
├── config/                 # 配置文件
├── internal/               # 核心模块（不可被外部引用）
│   ├── biz/               # 业务逻辑层
│   ├── data/              # 数据访问层
│   ├── plugin/            # 插件系统核心
│   │   ├── manager.go     # 插件管理器
│   │   ├── plugin.go      # 插件接口定义
│   │   └── menu.go        # 菜单配置
│   └── server/            # HTTP 服务
│       └── http.go        # 插件注册入口
├── plugins/                # 插件目录 ⭐
│   ├── kubernetes/        # K8S 管理插件
│   ├── task/              # 任务中心插件
│   └── monitor/           # 监控中心插件
├── web/                    # 前端代码
│   ├── src/
│   │   ├── plugins/       # 前端插件 ⭐
│   │   ├── views/         # 页面视图
│   │   └── api/           # API 请求
│   └── package.json
└── main.go
```

---

## 后端插件开发

### 1. 创建插件目录

```bash
# 创建插件目录结构
mkdir -p plugins/myplugin/{model,server}
```

目录结构：

```
plugins/myplugin/
├── plugin.go          # 插件入口，实现 Plugin 接口
├── model/             # 数据模型
│   └── model.go       # GORM 模型定义
└── server/            # HTTP 服务
    ├── router.go      # 路由定义
    └── handler.go     # 请求处理器
```

### 2. 实现插件接口

插件必须实现 `plugin.Plugin` 接口：

```go
// plugins/myplugin/plugin.go
package myplugin

import (
    "github.com/gin-gonic/gin"
    "github.com/ydcloud-dy/opshub/internal/plugin"
    "github.com/ydcloud-dy/opshub/plugins/myplugin/model"
    "github.com/ydcloud-dy/opshub/plugins/myplugin/server"
    "gorm.io/gorm"
)

type Plugin struct {
    db *gorm.DB
}

func New() *Plugin {
    return &Plugin{}
}

// ========== 插件元信息（必须实现） ==========

// Name 返回插件唯一标识符
// 用于路由前缀、数据库记录等
func (p *Plugin) Name() string {
    return "myplugin"
}

// Description 返回插件描述
func (p *Plugin) Description() string {
    return "我的自定义插件"
}

// Version 返回插件版本号
// 建议使用语义化版本：主版本.次版本.修订版本
func (p *Plugin) Version() string {
    return "1.0.0"
}

// Author 返回插件作者
func (p *Plugin) Author() string {
    return "Your Name"
}

// ========== 生命周期方法（必须实现） ==========

// Enable 插件启用时调用
// 用于初始化数据库表、加载配置等
func (p *Plugin) Enable(db *gorm.DB) error {
    p.db = db

    // 自动迁移数据库表
    if err := db.AutoMigrate(
        &model.MyModel{},
        &model.AnotherModel{},
    ); err != nil {
        return err
    }

    // 初始化默认数据（可选）
    // p.initDefaultData()

    return nil
}

// Disable 插件禁用时调用
// 用于清理资源、停止后台任务等
func (p *Plugin) Disable(db *gorm.DB) error {
    // 清理后台任务
    // 关闭连接
    // 注意：通常不删除数据表
    return nil
}

// ========== 路由注册（必须实现） ==========

// RegisterRoutes 注册插件的 HTTP 路由
// router 已经挂载到 /api/v1/plugins/{plugin_name} 路径下
func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    server.RegisterRoutes(router, db)
}

// ========== 菜单配置（必须实现） ==========

// GetMenus 返回插件的菜单配置
// 用于动态生成系统菜单
func (p *Plugin) GetMenus() []plugin.MenuConfig {
    return []plugin.MenuConfig{
        {
            Name:     "我的插件",
            Path:     "/myplugin",
            Icon:     "Setting",
            Sort:     90,
            Children: []plugin.MenuConfig{
                {
                    Name: "功能一",
                    Path: "/myplugin/feature1",
                    Icon: "Document",
                    Sort: 1,
                },
                {
                    Name: "功能二",
                    Path: "/myplugin/feature2",
                    Icon: "List",
                    Sort: 2,
                },
            },
        },
    }
}
```

### 3. 定义数据模型

```go
// plugins/myplugin/model/model.go
package model

import (
    "time"
    "gorm.io/gorm"
)

// MyModel 示例数据模型
type MyModel struct {
    ID          uint           `gorm:"primaryKey" json:"id"`
    Name        string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
    Description string         `gorm:"size:500" json:"description"`
    Status      int            `gorm:"default:1" json:"status"` // 1: 启用, 0: 禁用
    Config      string         `gorm:"type:text" json:"config"` // JSON 配置
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (MyModel) TableName() string {
    return "my_plugin_models"
}

// BeforeCreate 创建前钩子
func (m *MyModel) BeforeCreate(tx *gorm.DB) error {
    // 验证、设置默认值等
    return nil
}
```

### 4. 实现路由和处理器

```go
// plugins/myplugin/server/router.go
package server

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

var db *gorm.DB

func RegisterRoutes(router *gin.RouterGroup, database *gorm.DB) {
    db = database

    // 路由组
    // 最终路径: /api/v1/plugins/myplugin/...
    {
        router.GET("/list", listHandler)
        router.GET("/detail/:id", detailHandler)
        router.POST("/create", createHandler)
        router.PUT("/update/:id", updateHandler)
        router.DELETE("/delete/:id", deleteHandler)
    }

    // 子路由组
    feature := router.Group("/feature")
    {
        feature.GET("/stats", statsHandler)
        feature.POST("/action", actionHandler)
    }
}
```

```go
// plugins/myplugin/server/handler.go
package server

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/ydcloud-dy/opshub/plugins/myplugin/model"
)

// Response 统一响应结构
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// 成功响应
func success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Code:    0,
        Message: "success",
        Data:    data,
    })
}

// 错误响应
func fail(c *gin.Context, code int, message string) {
    c.JSON(http.StatusOK, Response{
        Code:    code,
        Message: message,
    })
}

// listHandler 列表查询
func listHandler(c *gin.Context) {
    var items []model.MyModel

    // 分页参数
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

    // 查询条件
    query := db.Model(&model.MyModel{})

    // 关键字搜索
    if keyword := c.Query("keyword"); keyword != "" {
        query = query.Where("name LIKE ?", "%"+keyword+"%")
    }

    // 状态筛选
    if status := c.Query("status"); status != "" {
        query = query.Where("status = ?", status)
    }

    // 统计总数
    var total int64
    query.Count(&total)

    // 分页查询
    offset := (page - 1) * pageSize
    if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&items).Error; err != nil {
        fail(c, 500, "查询失败: "+err.Error())
        return
    }

    success(c, gin.H{
        "list":      items,
        "total":     total,
        "page":      page,
        "page_size": pageSize,
    })
}

// detailHandler 详情查询
func detailHandler(c *gin.Context) {
    id := c.Param("id")

    var item model.MyModel
    if err := db.First(&item, id).Error; err != nil {
        fail(c, 404, "记录不存在")
        return
    }

    success(c, item)
}

// CreateRequest 创建请求
type CreateRequest struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
    Config      string `json:"config"`
}

// createHandler 创建记录
func createHandler(c *gin.Context) {
    var req CreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        fail(c, 400, "参数错误: "+err.Error())
        return
    }

    item := model.MyModel{
        Name:        req.Name,
        Description: req.Description,
        Config:      req.Config,
        Status:      1,
    }

    if err := db.Create(&item).Error; err != nil {
        fail(c, 500, "创建失败: "+err.Error())
        return
    }

    success(c, item)
}

// updateHandler 更新记录
func updateHandler(c *gin.Context) {
    id := c.Param("id")

    var item model.MyModel
    if err := db.First(&item, id).Error; err != nil {
        fail(c, 404, "记录不存在")
        return
    }

    var req CreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        fail(c, 400, "参数错误: "+err.Error())
        return
    }

    item.Name = req.Name
    item.Description = req.Description
    item.Config = req.Config

    if err := db.Save(&item).Error; err != nil {
        fail(c, 500, "更新失败: "+err.Error())
        return
    }

    success(c, item)
}

// deleteHandler 删除记录
func deleteHandler(c *gin.Context) {
    id := c.Param("id")

    if err := db.Delete(&model.MyModel{}, id).Error; err != nil {
        fail(c, 500, "删除失败: "+err.Error())
        return
    }

    success(c, nil)
}

// statsHandler 统计数据
func statsHandler(c *gin.Context) {
    var total, enabled, disabled int64

    db.Model(&model.MyModel{}).Count(&total)
    db.Model(&model.MyModel{}).Where("status = 1").Count(&enabled)
    db.Model(&model.MyModel{}).Where("status = 0").Count(&disabled)

    success(c, gin.H{
        "total":    total,
        "enabled":  enabled,
        "disabled": disabled,
    })
}

// actionHandler 执行操作
func actionHandler(c *gin.Context) {
    // 实现具体业务逻辑
    success(c, gin.H{"result": "action completed"})
}
```

### 5. 注册插件到系统

编辑 `internal/server/http.go`：

```go
import (
    // ... 其他导入
    myplugin "github.com/ydcloud-dy/opshub/plugins/myplugin"
)

func NewHTTPServer(/* ... */) *HTTPServer {
    // ...

    // 注册插件
    s.pluginMgr.Register(kubeplugin.New())
    s.pluginMgr.Register(monitorplugin.New())
    s.pluginMgr.Register(taskplugin.New())
    s.pluginMgr.Register(myplugin.New())  // 添加新插件

    // ...
}
```

---

## 前端插件开发

### 1. 创建插件目录

```bash
# 创建前端插件目录
mkdir -p web/src/plugins/myplugin
mkdir -p web/src/views/myplugin
mkdir -p web/src/api
```

### 2. 实现插件入口

```typescript
// web/src/plugins/myplugin/index.ts
import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '@/plugins/types'
import { pluginManager } from '@/plugins/manager'

class MyPlugin implements Plugin {
    name = 'myplugin'
    description = '我的自定义插件'
    version = '1.0.0'
    author = 'Your Name'

    async install() {
        console.log('MyPlugin installed')
        // 初始化逻辑：加载配置、注册事件等
    }

    async uninstall() {
        console.log('MyPlugin uninstalled')
        // 清理逻辑：移除事件监听、清理缓存等
    }

    getMenus(): PluginMenuConfig[] {
        return [
            {
                name: '我的插件',
                path: '/myplugin',
                icon: 'Setting',
                sort: 90,
                hidden: false,
                parentPath: '',
                children: [
                    {
                        name: '功能一',
                        path: '/myplugin/feature1',
                        icon: 'Document',
                        sort: 1,
                        hidden: false,
                        parentPath: '/myplugin',
                    },
                    {
                        name: '功能二',
                        path: '/myplugin/feature2',
                        icon: 'List',
                        sort: 2,
                        hidden: false,
                        parentPath: '/myplugin',
                    },
                ],
            },
        ]
    }

    getRoutes(): PluginRouteConfig[] {
        return [
            {
                path: '/myplugin',
                name: 'MyPlugin',
                component: () => import('@/views/myplugin/Index.vue'),
                redirect: '/myplugin/feature1',
                meta: {
                    title: '我的插件',
                    icon: 'Setting',
                },
                children: [
                    {
                        path: 'feature1',
                        name: 'Feature1',
                        component: () => import('@/views/myplugin/Feature1.vue'),
                        meta: {
                            title: '功能一',
                            icon: 'Document',
                        },
                    },
                    {
                        path: 'feature2',
                        name: 'Feature2',
                        component: () => import('@/views/myplugin/Feature2.vue'),
                        meta: {
                            title: '功能二',
                            icon: 'List',
                        },
                    },
                ],
            },
        ]
    }
}

// 创建实例并注册
const plugin = new MyPlugin()
pluginManager.register(plugin)

export default plugin
```

### 3. 创建 API 封装

```typescript
// web/src/api/myplugin.ts
import request from '@/utils/request'

const BASE_URL = '/api/v1/plugins/myplugin'

// 类型定义
export interface MyItem {
    id: number
    name: string
    description: string
    status: number
    config: string
    created_at: string
    updated_at: string
}

export interface ListParams {
    page?: number
    page_size?: number
    keyword?: string
    status?: number
}

export interface ListResponse {
    list: MyItem[]
    total: number
    page: number
    page_size: number
}

// 获取列表
export function getList(params: ListParams) {
    return request.get<ListResponse>(`${BASE_URL}/list`, { params })
}

// 获取详情
export function getDetail(id: number) {
    return request.get<MyItem>(`${BASE_URL}/detail/${id}`)
}

// 创建
export function create(data: Partial<MyItem>) {
    return request.post<MyItem>(`${BASE_URL}/create`, data)
}

// 更新
export function update(id: number, data: Partial<MyItem>) {
    return request.put<MyItem>(`${BASE_URL}/update/${id}`, data)
}

// 删除
export function remove(id: number) {
    return request.delete(`${BASE_URL}/delete/${id}`)
}

// 获取统计
export function getStats() {
    return request.get<{ total: number; enabled: number; disabled: number }>(
        `${BASE_URL}/feature/stats`
    )
}
```

### 4. 创建页面组件

```vue
<!-- web/src/views/myplugin/Index.vue -->
<template>
  <div class="myplugin-container">
    <router-view />
  </div>
</template>

<script setup lang="ts">
// 布局组件，用于嵌套路由
</script>

<style scoped>
.myplugin-container {
  padding: 20px;
}
</style>
```

```vue
<!-- web/src/views/myplugin/Feature1.vue -->
<template>
  <div class="feature1-page">
    <!-- 搜索栏 -->
    <el-card class="search-card">
      <el-form :inline="true" :model="searchForm">
        <el-form-item label="关键字">
          <el-input v-model="searchForm.keyword" placeholder="搜索名称" clearable />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" placeholder="全部" clearable>
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 操作栏 -->
    <el-card class="table-card">
      <template #header>
        <div class="card-header">
          <span>数据列表</span>
          <el-button type="primary" @click="handleCreate">新建</el-button>
        </div>
      </template>

      <!-- 数据表格 -->
      <el-table :data="tableData" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </el-card>

    <!-- 编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
      <el-form :model="formData" :rules="formRules" ref="formRef" label-width="80px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="formData.description" type="textarea" rows="3" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="配置" prop="config">
          <el-input v-model="formData.config" type="textarea" rows="5" placeholder="JSON 配置" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { getList, create, update, remove, type MyItem } from '@/api/myplugin'

// 搜索表单
const searchForm = reactive({
  keyword: '',
  status: undefined as number | undefined,
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0,
})

// 数据
const loading = ref(false)
const tableData = ref<MyItem[]>([])

// 弹窗
const dialogVisible = ref(false)
const dialogTitle = ref('新建')
const submitLoading = ref(false)
const formRef = ref<FormInstance>()
const formData = reactive({
  id: 0,
  name: '',
  description: '',
  config: '',
})

const formRules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

// 获取列表
const fetchList = async () => {
  loading.value = true
  try {
    const res = await getList({
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: searchForm.keyword,
      status: searchForm.status,
    })
    if (res.data.code === 0) {
      tableData.value = res.data.data.list
      pagination.total = res.data.data.total
    }
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  pagination.page = 1
  fetchList()
}

// 重置
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.status = undefined
  handleSearch()
}

// 分页
const handleSizeChange = () => {
  pagination.page = 1
  fetchList()
}

const handlePageChange = () => {
  fetchList()
}

// 新建
const handleCreate = () => {
  dialogTitle.value = '新建'
  formData.id = 0
  formData.name = ''
  formData.description = ''
  formData.config = ''
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: MyItem) => {
  dialogTitle.value = '编辑'
  formData.id = row.id
  formData.name = row.name
  formData.description = row.description
  formData.config = row.config
  dialogVisible.value = true
}

// 删除
const handleDelete = async (row: MyItem) => {
  await ElMessageBox.confirm('确定要删除该记录吗？', '提示', { type: 'warning' })
  await remove(row.id)
  ElMessage.success('删除成功')
  fetchList()
}

// 提交
const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()

  submitLoading.value = true
  try {
    if (formData.id) {
      await update(formData.id, formData)
      ElMessage.success('更新成功')
    } else {
      await create(formData)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchList()
  } finally {
    submitLoading.value = false
  }
}

onMounted(() => {
  fetchList()
})
</script>

<style scoped>
.search-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
```

### 5. 导入插件

编辑 `web/src/main.ts`：

```typescript
// 导入插件
import '@/plugins/kubernetes'
import '@/plugins/monitor'
import '@/plugins/task'
import '@/plugins/myplugin'  // 添加新插件
```

---

## 数据库设计

### 命名规范

| 类型 | 规范 | 示例 |
|:-----|:-----|:-----|
| 表名 | 小写，下划线分隔，插件前缀 | `my_plugin_models` |
| 字段名 | 小写，下划线分隔 | `created_at` |
| 主键 | `id`，bigint unsigned | `id` |
| 外键 | `关联表_id` | `user_id` |
| 时间字段 | `xxx_at` 或 `xxx_time` | `created_at` |
| 状态字段 | `status` 或 `xxx_status` | `status` |

### 常用字段类型

| 字段类型 | Go 类型 | 用途 |
|:---------|:--------|:-----|
| `bigint unsigned` | `uint` | 主键、外键 |
| `varchar(n)` | `string` | 短字符串 |
| `text` | `string` | 中等文本 |
| `longtext` | `string` | 大文本 |
| `json` | `string` | JSON 数据 |
| `tinyint` | `int` | 状态、布尔值 |
| `datetime` | `time.Time` | 时间戳 |

### 示例表结构

```sql
CREATE TABLE IF NOT EXISTS `my_plugin_models` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(100) NOT NULL COMMENT '名称',
    `description` varchar(500) DEFAULT '' COMMENT '描述',
    `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态: 1-启用, 0-禁用',
    `config` text COMMENT 'JSON 配置',
    `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_name` (`name`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='我的插件数据表';
```

---

## API 设计规范

### URL 规范

```
GET    /api/v1/plugins/{plugin}/list          # 列表
GET    /api/v1/plugins/{plugin}/detail/{id}   # 详情
POST   /api/v1/plugins/{plugin}/create        # 创建
PUT    /api/v1/plugins/{plugin}/update/{id}   # 更新
DELETE /api/v1/plugins/{plugin}/delete/{id}   # 删除
```

### 响应格式

```json
{
    "code": 0,
    "message": "success",
    "data": {}
}
```

| code | 说明 |
|:-----|:-----|
| 0 | 成功 |
| 400 | 参数错误 |
| 401 | 未授权 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器错误 |

### 分页响应

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [],
        "total": 100,
        "page": 1,
        "page_size": 10
    }
}
```

---

## 测试与调试

### 后端测试

```bash
# 启动后端
go run main.go server

# 测试 API
curl http://localhost:9876/api/v1/plugins/myplugin/list
curl http://localhost:9876/api/v1/plugins/myplugin/detail/1
curl -X POST http://localhost:9876/api/v1/plugins/myplugin/create \
    -H "Content-Type: application/json" \
    -d '{"name":"test","description":"test description"}'
```

### 前端测试

```bash
cd web
npm run dev

# 访问 http://localhost:5173
```

### 调试技巧

1. **后端日志**：使用 zap 日志库
2. **前端调试**：Vue DevTools
3. **API 调试**：Postman / Insomnia
4. **数据库调试**：DBeaver / MySQL Workbench

---

## 发布与部署

### 代码提交

1. 确保所有测试通过
2. 更新版本号
3. 更新文档
4. 提交代码并创建 PR

### 部署检查清单

- [ ] 数据库迁移脚本准备
- [ ] 配置文件更新
- [ ] 前端资源构建
- [ ] 后端服务编译
- [ ] 环境变量配置
- [ ] 健康检查验证

---

## 下一步

- [API 参考](api-reference.md) - 查看完整接口定义
- [高级主题](advanced-topics.md) - 菜单排序、权限控制等
