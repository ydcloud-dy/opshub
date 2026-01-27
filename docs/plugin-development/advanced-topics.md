# 插件开发高级主题

本文档介绍 OpsHub 插件开发的高级功能，包括菜单排序、权限控制、后台任务等。

---

## 目录

- [菜单系统](#菜单系统)
  - [菜单排序](#菜单排序)
  - [动态菜单](#动态菜单)
  - [菜单权限](#菜单权限)
- [权限控制](#权限控制)
  - [API 权限](#api-权限)
  - [数据权限](#数据权限)
- [后台任务](#后台任务)
  - [定时任务](#定时任务)
  - [异步任务](#异步任务)
- [WebSocket](#websocket)
- [文件处理](#文件处理)
- [缓存策略](#缓存策略)
- [日志记录](#日志记录)
- [错误处理](#错误处理)
- [性能优化](#性能优化)

---

## 菜单系统

### 菜单排序

菜单按 `Sort` 字段升序排列，数值越小越靠前。

#### 推荐排序范围

| 功能模块 | Sort 范围 | 说明 |
|:---------|:----------|:-----|
| 仪表盘 | 1-9 | 首页、概览 |
| 核心功能 | 10-29 | 资产管理等 |
| 插件功能 | 30-79 | 各插件菜单 |
| 系统管理 | 80-89 | 用户、角色等 |
| 新插件 | 90-99 | 新开发的插件 |

#### 现有插件排序

| 插件 | 菜单名称 | Sort |
|:-----|:---------|:-----|
| - | 仪表盘 | 1 |
| - | 资产管理 | 10 |
| kubernetes | 容器管理 | 20 |
| task | 任务中心 | 30 |
| monitor | 监控中心 | 40 |
| - | 操作审计 | 50 |
| - | 插件管理 | 60 |
| - | 系统管理 | 80 |

#### 子菜单排序

子菜单在父菜单内部独立排序：

```go
func (p *Plugin) GetMenus() []plugin.MenuConfig {
    return []plugin.MenuConfig{
        {
            Name: "我的插件",
            Path: "/myplugin",
            Sort: 35,  // 在任务中心和监控中心之间
            Children: []plugin.MenuConfig{
                {Name: "功能A", Path: "/myplugin/a", Sort: 1},
                {Name: "功能B", Path: "/myplugin/b", Sort: 2},
                {Name: "功能C", Path: "/myplugin/c", Sort: 3},
            },
        },
    }
}
```

### 动态菜单

根据用户权限或配置动态生成菜单：

```go
func (p *Plugin) GetMenus() []plugin.MenuConfig {
    menus := []plugin.MenuConfig{
        {Name: "基础功能", Path: "/myplugin/basic", Sort: 1},
    }

    // 根据配置添加高级功能菜单
    if p.config.EnableAdvanced {
        menus = append(menus, plugin.MenuConfig{
            Name: "高级功能",
            Path: "/myplugin/advanced",
            Sort: 2,
        })
    }

    return menus
}
```

### 菜单权限

菜单与权限系统集成：

```go
// 在菜单配置中添加权限标识
type MenuConfig struct {
    Name       string       `json:"name"`
    Path       string       `json:"path"`
    Icon       string       `json:"icon"`
    Sort       int          `json:"sort"`
    Permission string       `json:"permission"` // 权限标识
    Children   []MenuConfig `json:"children"`
}

// 使用示例
func (p *Plugin) GetMenus() []plugin.MenuConfig {
    return []plugin.MenuConfig{
        {
            Name:       "数据管理",
            Path:       "/myplugin/data",
            Permission: "myplugin:data:view",  // 需要此权限才能看到
        },
        {
            Name:       "系统配置",
            Path:       "/myplugin/config",
            Permission: "myplugin:config:view",
        },
    }
}
```

---

## 权限控制

### API 权限

使用中间件进行 API 权限控制：

```go
// plugins/myplugin/server/middleware.go
package server

import (
    "github.com/gin-gonic/gin"
)

// 权限检查中间件
func RequirePermission(permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从上下文获取用户信息
        userID, exists := c.Get("user_id")
        if !exists {
            c.JSON(401, gin.H{"code": 401, "message": "未登录"})
            c.Abort()
            return
        }

        // 检查用户是否有权限
        if !checkPermission(userID.(uint), permission) {
            c.JSON(403, gin.H{"code": 403, "message": "权限不足"})
            c.Abort()
            return
        }

        c.Next()
    }
}

// 检查权限（示例实现）
func checkPermission(userID uint, permission string) bool {
    // 从数据库或缓存获取用户权限
    // 返回是否有权限
    return true
}
```

使用中间件：

```go
// plugins/myplugin/server/router.go
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    // 公开接口
    router.GET("/info", getInfo)

    // 需要权限的接口
    router.GET("/list", RequirePermission("myplugin:list"), listHandler)
    router.POST("/create", RequirePermission("myplugin:create"), createHandler)
    router.PUT("/update/:id", RequirePermission("myplugin:update"), updateHandler)
    router.DELETE("/delete/:id", RequirePermission("myplugin:delete"), deleteHandler)
}
```

### 数据权限

实现行级数据权限：

```go
// 根据用户过滤数据
func listHandler(c *gin.Context) {
    userID, _ := c.Get("user_id")
    roleCode, _ := c.Get("role_code")

    query := db.Model(&model.MyModel{})

    // 管理员可以看所有数据
    if roleCode != "admin" {
        // 普通用户只能看自己创建的
        query = query.Where("created_by = ?", userID)
    }

    // 继续查询...
}
```

---

## 后台任务

### 定时任务

使用 cron 实现定时任务：

```go
// plugins/myplugin/plugin.go
package myplugin

import (
    "github.com/robfig/cron/v3"
)

type Plugin struct {
    db   *gorm.DB
    cron *cron.Cron
}

func (p *Plugin) Enable(db *gorm.DB) error {
    p.db = db

    // 创建定时任务调度器
    p.cron = cron.New()

    // 添加定时任务：每分钟执行
    p.cron.AddFunc("* * * * *", func() {
        p.checkStatus()
    })

    // 添加定时任务：每天凌晨 2 点执行
    p.cron.AddFunc("0 2 * * *", func() {
        p.cleanupOldData()
    })

    // 启动调度器
    p.cron.Start()

    return nil
}

func (p *Plugin) Disable(db *gorm.DB) error {
    // 停止定时任务
    if p.cron != nil {
        p.cron.Stop()
    }
    return nil
}

func (p *Plugin) checkStatus() {
    // 实现状态检查逻辑
}

func (p *Plugin) cleanupOldData() {
    // 清理 30 天前的数据
    p.db.Where("created_at < ?", time.Now().AddDate(0, 0, -30)).Delete(&model.MyModel{})
}
```

### 异步任务

使用 goroutine 和 channel 实现异步任务：

```go
// plugins/myplugin/server/task.go
package server

import (
    "context"
    "sync"
)

type TaskManager struct {
    tasks chan Task
    wg    sync.WaitGroup
    ctx   context.Context
    cancel context.CancelFunc
}

type Task struct {
    ID      string
    Handler func() error
}

func NewTaskManager(workerCount int) *TaskManager {
    ctx, cancel := context.WithCancel(context.Background())
    tm := &TaskManager{
        tasks:  make(chan Task, 100),
        ctx:    ctx,
        cancel: cancel,
    }

    // 启动 worker
    for i := 0; i < workerCount; i++ {
        tm.wg.Add(1)
        go tm.worker()
    }

    return tm
}

func (tm *TaskManager) worker() {
    defer tm.wg.Done()
    for {
        select {
        case task := <-tm.tasks:
            if err := task.Handler(); err != nil {
                // 记录错误
            }
        case <-tm.ctx.Done():
            return
        }
    }
}

func (tm *TaskManager) Submit(task Task) {
    tm.tasks <- task
}

func (tm *TaskManager) Stop() {
    tm.cancel()
    tm.wg.Wait()
}
```

使用示例：

```go
var taskManager *TaskManager

func init() {
    taskManager = NewTaskManager(5)
}

func asyncHandler(c *gin.Context) {
    // 提交异步任务
    taskManager.Submit(Task{
        ID: "task-123",
        Handler: func() error {
            // 执行耗时操作
            return nil
        },
    })

    c.JSON(200, gin.H{"message": "任务已提交"})
}
```

---

## WebSocket

实现实时通信：

```go
// plugins/myplugin/server/websocket.go
package server

import (
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "net/http"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

func wsHandler(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    // 处理消息
    for {
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            break
        }

        // 处理接收到的消息
        response := processMessage(message)

        // 发送响应
        if err := conn.WriteMessage(messageType, response); err != nil {
            break
        }
    }
}

func processMessage(message []byte) []byte {
    // 处理消息逻辑
    return []byte("received")
}
```

注册路由：

```go
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    router.GET("/ws", wsHandler)
}
```

前端连接：

```typescript
const ws = new WebSocket('ws://localhost:9876/api/v1/plugins/myplugin/ws')

ws.onopen = () => {
    console.log('WebSocket connected')
    ws.send(JSON.stringify({ type: 'subscribe', topic: 'updates' }))
}

ws.onmessage = (event) => {
    const data = JSON.parse(event.data)
    console.log('Received:', data)
}

ws.onclose = () => {
    console.log('WebSocket disconnected')
}
```

---

## 文件处理

### 文件上传

```go
// plugins/myplugin/server/upload.go
package server

import (
    "path/filepath"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func uploadHandler(c *gin.Context) {
    file, err := c.FormFile("file")
    if err != nil {
        fail(c, 400, "请选择文件")
        return
    }

    // 验证文件类型
    ext := filepath.Ext(file.Filename)
    allowedExts := map[string]bool{".jpg": true, ".png": true, ".pdf": true}
    if !allowedExts[ext] {
        fail(c, 400, "不支持的文件类型")
        return
    }

    // 验证文件大小（10MB）
    if file.Size > 10*1024*1024 {
        fail(c, 400, "文件大小超过限制")
        return
    }

    // 生成唯一文件名
    filename := uuid.New().String() + ext
    dst := filepath.Join("uploads", "myplugin", filename)

    // 保存文件
    if err := c.SaveUploadedFile(file, dst); err != nil {
        fail(c, 500, "保存文件失败")
        return
    }

    success(c, gin.H{
        "filename": filename,
        "url":      "/uploads/myplugin/" + filename,
    })
}
```

### 文件下载

```go
func downloadHandler(c *gin.Context) {
    filename := c.Param("filename")

    // 安全检查：防止路径遍历
    if filepath.Clean(filename) != filename {
        fail(c, 400, "无效的文件名")
        return
    }

    filepath := filepath.Join("uploads", "myplugin", filename)

    // 检查文件是否存在
    if _, err := os.Stat(filepath); os.IsNotExist(err) {
        fail(c, 404, "文件不存在")
        return
    }

    c.File(filepath)
}
```

---

## 缓存策略

使用 Redis 缓存：

```go
// plugins/myplugin/server/cache.go
package server

import (
    "context"
    "encoding/json"
    "time"

    "github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func initCache(addr string) {
    rdb = redis.NewClient(&redis.Options{
        Addr: addr,
    })
}

// 缓存键前缀
const cachePrefix = "myplugin:"

// 获取缓存
func getCache[T any](ctx context.Context, key string) (*T, error) {
    val, err := rdb.Get(ctx, cachePrefix+key).Result()
    if err == redis.Nil {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    var result T
    if err := json.Unmarshal([]byte(val), &result); err != nil {
        return nil, err
    }
    return &result, nil
}

// 设置缓存
func setCache[T any](ctx context.Context, key string, value T, expiration time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return rdb.Set(ctx, cachePrefix+key, data, expiration).Err()
}

// 删除缓存
func delCache(ctx context.Context, key string) error {
    return rdb.Del(ctx, cachePrefix+key).Err()
}
```

使用示例：

```go
func getItemWithCache(ctx context.Context, id uint) (*model.MyModel, error) {
    cacheKey := fmt.Sprintf("item:%d", id)

    // 尝试从缓存获取
    if cached, err := getCache[model.MyModel](ctx, cacheKey); err == nil && cached != nil {
        return cached, nil
    }

    // 从数据库获取
    var item model.MyModel
    if err := db.First(&item, id).Error; err != nil {
        return nil, err
    }

    // 写入缓存（5 分钟过期）
    setCache(ctx, cacheKey, item, 5*time.Minute)

    return &item, nil
}
```

---

## 日志记录

使用 zap 进行结构化日志：

```go
// plugins/myplugin/server/logger.go
package server

import (
    "go.uber.org/zap"
)

var logger *zap.Logger

func initLogger() {
    logger, _ = zap.NewProduction()
}

// 记录操作日志
func logOperation(userID uint, action string, resource string, detail string) {
    logger.Info("operation",
        zap.Uint("user_id", userID),
        zap.String("action", action),
        zap.String("resource", resource),
        zap.String("detail", detail),
    )
}

// 记录错误
func logError(err error, context string) {
    logger.Error("error",
        zap.Error(err),
        zap.String("context", context),
    )
}
```

使用示例：

```go
func createHandler(c *gin.Context) {
    userID, _ := c.Get("user_id")

    // ... 创建逻辑

    // 记录操作日志
    logOperation(userID.(uint), "create", "my_model", fmt.Sprintf("created item: %s", item.Name))

    success(c, item)
}
```

---

## 错误处理

统一错误处理：

```go
// plugins/myplugin/server/errors.go
package server

import (
    "errors"
)

// 业务错误码
const (
    ErrCodeSuccess     = 0
    ErrCodeBadRequest  = 400
    ErrCodeUnauthorized = 401
    ErrCodeForbidden   = 403
    ErrCodeNotFound    = 404
    ErrCodeConflict    = 409
    ErrCodeInternal    = 500
)

// 业务错误
type BizError struct {
    Code    int
    Message string
    Err     error
}

func (e *BizError) Error() string {
    if e.Err != nil {
        return e.Message + ": " + e.Err.Error()
    }
    return e.Message
}

// 预定义错误
var (
    ErrNotFound      = &BizError{Code: ErrCodeNotFound, Message: "资源不存在"}
    ErrUnauthorized  = &BizError{Code: ErrCodeUnauthorized, Message: "未授权"}
    ErrForbidden     = &BizError{Code: ErrCodeForbidden, Message: "权限不足"}
    ErrBadRequest    = &BizError{Code: ErrCodeBadRequest, Message: "参数错误"}
    ErrNameExists    = &BizError{Code: ErrCodeConflict, Message: "名称已存在"}
)

// 创建错误
func NewBizError(code int, message string) *BizError {
    return &BizError{Code: code, Message: message}
}

// 包装错误
func WrapError(err error, message string) *BizError {
    return &BizError{Code: ErrCodeInternal, Message: message, Err: err}
}
```

使用示例：

```go
func getDetailHandler(c *gin.Context) {
    id := c.Param("id")

    var item model.MyModel
    if err := db.First(&item, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            handleError(c, ErrNotFound)
            return
        }
        handleError(c, WrapError(err, "查询失败"))
        return
    }

    success(c, item)
}

func handleError(c *gin.Context, err error) {
    var bizErr *BizError
    if errors.As(err, &bizErr) {
        c.JSON(200, gin.H{
            "code":    bizErr.Code,
            "message": bizErr.Message,
        })
        return
    }

    c.JSON(200, gin.H{
        "code":    ErrCodeInternal,
        "message": "服务器内部错误",
    })
}
```

---

## 性能优化

### 数据库优化

```go
// 使用索引
type MyModel struct {
    ID        uint   `gorm:"primaryKey"`
    Name      string `gorm:"size:100;index"`           // 单列索引
    Status    int    `gorm:"index:idx_status_created"` // 组合索引
    CreatedAt time.Time `gorm:"index:idx_status_created"`
}

// 批量操作
func batchCreate(items []model.MyModel) error {
    return db.CreateInBatches(items, 100).Error
}

// 预加载关联
func getWithRelations(id uint) (*model.MyModel, error) {
    var item model.MyModel
    err := db.Preload("RelatedItems").First(&item, id).Error
    return &item, err
}

// 只查询需要的字段
func getList() ([]model.MyModel, error) {
    var items []model.MyModel
    err := db.Select("id", "name", "status").Find(&items).Error
    return items, err
}
```

### 并发控制

```go
// 使用连接池
db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
sqlDB, _ := db.DB()
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(time.Hour)

// 使用互斥锁
var mu sync.Mutex

func updateCounter() {
    mu.Lock()
    defer mu.Unlock()
    // 更新操作
}

// 使用读写锁
var rwmu sync.RWMutex

func readData() {
    rwmu.RLock()
    defer rwmu.RUnlock()
    // 读取操作
}

func writeData() {
    rwmu.Lock()
    defer rwmu.Unlock()
    // 写入操作
}
```

### 请求限流

```go
import "golang.org/x/time/rate"

// 创建限流器：每秒 100 个请求，最多积累 10 个
var limiter = rate.NewLimiter(100, 10)

func rateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(429, gin.H{
                "code":    429,
                "message": "请求过于频繁，请稍后重试",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

---

## 相关文档

- [快速开始](quick-start.md) - 5 分钟创建简单插件
- [完整开发指南](development-guide.md) - 详细开发流程
- [API 参考](api-reference.md) - 接口定义
