# 插件开发快速开始

本文档将指导你在 5 分钟内创建一个简单的 OpsHub 插件。

---

## 目标

创建一个名为 `example` 的示例插件，包含：
- 一个 API 接口 `/api/v1/plugins/example/info`
- 一个前端页面 `/example`
- 一个菜单项

---

## 第一步：创建后端插件

### 1.1 创建目录结构

```bash
mkdir -p plugins/example/model
mkdir -p plugins/example/server
```

### 1.2 创建插件入口

```go
// plugins/example/plugin.go
package example

import (
    "github.com/gin-gonic/gin"
    "github.com/ydcloud-dy/opshub/internal/plugin"
    "github.com/ydcloud-dy/opshub/plugins/example/server"
    "gorm.io/gorm"
)

type Plugin struct {
    db *gorm.DB
}

func New() *Plugin {
    return &Plugin{}
}

// ========== 插件元信息 ==========

func (p *Plugin) Name() string {
    return "example"
}

func (p *Plugin) Description() string {
    return "示例插件 - 演示插件开发流程"
}

func (p *Plugin) Version() string {
    return "1.0.0"
}

func (p *Plugin) Author() string {
    return "Your Name"
}

// ========== 生命周期方法 ==========

func (p *Plugin) Enable(db *gorm.DB) error {
    p.db = db
    // 在这里初始化数据库表
    // 例如: return db.AutoMigrate(&model.YourModel{})
    return nil
}

func (p *Plugin) Disable(db *gorm.DB) error {
    // 在这里清理资源
    return nil
}

// ========== 路由注册 ==========

func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    server.RegisterRoutes(router, db)
}

// ========== 菜单配置 ==========

func (p *Plugin) GetMenus() []plugin.MenuConfig {
    return []plugin.MenuConfig{
        {
            Name: "示例插件",
            Path: "/example",
            Icon: "MagicStick",
            Sort: 95,  // 排序，越小越靠前
        },
    }
}
```

### 1.3 创建路由

```go
// plugins/example/server/router.go
package server

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    example := router.Group("/example")
    {
        example.GET("/info", getInfo)
        example.GET("/status", getStatus)
    }
}

func getInfo(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "code":    0,
        "message": "success",
        "data": gin.H{
            "name":        "Example Plugin",
            "version":     "1.0.0",
            "description": "这是一个示例插件",
        },
    })
}

func getStatus(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "code":    0,
        "message": "success",
        "data": gin.H{
            "status":    "running",
            "uptime":    time.Now().Format(time.RFC3339),
            "healthy":   true,
        },
    })
}
```

### 1.4 注册插件到系统

编辑 `internal/server/http.go`，在 `NewHTTPServer` 函数中添加：

```go
import (
    // ... 其他导入
    exampleplugin "github.com/ydcloud-dy/opshub/plugins/example"
)

// 在 NewHTTPServer() 函数中，找到插件注册部分
s.pluginMgr.Register(exampleplugin.New())
```

---

## 第二步：创建前端插件

### 2.1 创建插件入口

```typescript
// web/src/plugins/example/index.ts
import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '@/plugins/types'
import { pluginManager } from '@/plugins/manager'

class ExamplePlugin implements Plugin {
    name = 'example'
    description = '示例插件 - 演示插件开发流程'
    version = '1.0.0'
    author = 'Your Name'

    async install() {
        console.log('Example plugin installed')
    }

    async uninstall() {
        console.log('Example plugin uninstalled')
    }

    getMenus(): PluginMenuConfig[] {
        return [
            {
                name: '示例插件',
                path: '/example',
                icon: 'MagicStick',
                sort: 95,
                hidden: false,
                parentPath: '',
            },
        ]
    }

    getRoutes(): PluginRouteConfig[] {
        return [
            {
                path: '/example',
                name: 'Example',
                component: () => import('@/views/example/Index.vue'),
                meta: {
                    title: '示例插件',
                    icon: 'MagicStick',
                },
            },
        ]
    }
}

// 创建实例并注册
const plugin = new ExamplePlugin()
pluginManager.register(plugin)

export default plugin
```

### 2.2 创建页面组件

```vue
<!-- web/src/views/example/Index.vue -->
<template>
  <div class="example-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>示例插件</span>
          <el-button type="primary" @click="fetchInfo">获取信息</el-button>
        </div>
      </template>

      <el-descriptions :column="2" border>
        <el-descriptions-item label="插件名称">{{ info.name }}</el-descriptions-item>
        <el-descriptions-item label="版本">{{ info.version }}</el-descriptions-item>
        <el-descriptions-item label="描述" :span="2">{{ info.description }}</el-descriptions-item>
      </el-descriptions>

      <el-divider />

      <el-descriptions title="运行状态" :column="3" border>
        <el-descriptions-item label="状态">
          <el-tag :type="status.healthy ? 'success' : 'danger'">
            {{ status.status }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="启动时间">{{ status.uptime }}</el-descriptions-item>
        <el-descriptions-item label="健康状态">
          <el-icon v-if="status.healthy" color="green"><CircleCheckFilled /></el-icon>
          <el-icon v-else color="red"><CircleCloseFilled /></el-icon>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue'
import request from '@/utils/request'

const info = ref({
  name: '',
  version: '',
  description: '',
})

const status = ref({
  status: 'unknown',
  uptime: '',
  healthy: false,
})

const fetchInfo = async () => {
  try {
    const res = await request.get('/api/v1/plugins/example/info')
    if (res.data.code === 0) {
      info.value = res.data.data
    }
  } catch (error) {
    console.error('获取信息失败', error)
  }
}

const fetchStatus = async () => {
  try {
    const res = await request.get('/api/v1/plugins/example/status')
    if (res.data.code === 0) {
      status.value = res.data.data
    }
  } catch (error) {
    console.error('获取状态失败', error)
  }
}

onMounted(() => {
  fetchInfo()
  fetchStatus()
})
</script>

<style scoped>
.example-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
```

### 2.3 导入插件

编辑 `web/src/main.ts`，添加导入：

```typescript
// 导入插件
import '@/plugins/kubernetes'
import '@/plugins/monitor'
import '@/plugins/task'
import '@/plugins/example'  // 添加这行
```

---

## 第三步：测试插件

### 3.1 启动后端

```bash
go run main.go server
```

### 3.2 启动前端

```bash
cd web
npm run dev
```

### 3.3 验证

1. 打开浏览器访问 `http://localhost:5173`
2. 登录系统
3. 左侧菜单应该出现「示例插件」
4. 点击进入，可以看到插件页面
5. 点击「获取信息」按钮测试 API

### 3.4 测试 API

```bash
# 获取插件列表
curl http://localhost:9876/api/v1/plugins

# 测试插件 API
curl http://localhost:9876/api/v1/plugins/example/info
curl http://localhost:9876/api/v1/plugins/example/status
```

---

## 完成

恭喜！你已经成功创建了一个 OpsHub 插件。

### 下一步

- [完整开发指南](development-guide.md) - 学习更多高级功能
- [API 参考](api-reference.md) - 查看完整接口定义
- [高级主题](advanced-topics.md) - 菜单排序、权限控制等

---

## 常见问题

### Q: 插件菜单不显示？

检查：
1. 后端插件是否正确注册
2. `GetMenus()` 返回的配置是否正确
3. 前端是否导入了插件
4. 刷新页面或清除缓存

### Q: API 404？

检查：
1. 路由路径是否正确（以 `/api/v1/plugins/` 开头）
2. 后端是否重启
3. 插件是否启用

### Q: 前端页面白屏？

检查：
1. Vue 组件语法是否正确
2. 浏览器控制台错误信息
3. 路由配置是否正确
