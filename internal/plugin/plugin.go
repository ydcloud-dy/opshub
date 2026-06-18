// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package plugin

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Plugin 插件接口
// 所有插件必须实现该接口
type Plugin interface {
	// 插件名
	Name() string

	// 插件描述
	Description() string

	// 插件版本
	Version() string

	// 插件作者
	Author() string

	// 启用插件
	// Initialize plugin resources, database tables, etc.
	Enable(db *gorm.DB) error

	// 关闭插件
	// Clean up plugin resources (note: won't delete database tables by default)
	Disable(db *gorm.DB) error

	// 注册插件路由到系统路由
	// Plugin can register its API routes here
	RegisterRoutes(router *gin.RouterGroup, db *gorm.DB)

	// GetMenus Get plugin menu configuration
	// Return menu items to be added to the system
	GetMenus() []MenuConfig
}

// MenuConfig Menu configuration
type MenuConfig struct {
	// Menu name
	Name string `json:"name"`

	// Menu path (frontend route)
	Path string `json:"path"`

	// Icon name
	Icon string `json:"icon"`

	// Sort order (smaller number comes first)
	Sort int `json:"sort"`

	// Hidden or not
	Hidden bool `json:"hidden"`

	// Parent menu path (if this is a submenu)
	ParentPath string `json:"parentPath"`

	// Permission identifier (optional, for access control)
	Permission string `json:"permission"`
}

// PluginState 插件状态数据模型
type PluginState struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Enabled   bool      `gorm:"default:false;not null" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (PluginState) TableName() string {
	return "plugin_states"
}

// Manager Plugin manager
type Manager struct {
	plugins map[string]Plugin
	db      *gorm.DB
}

// NewManager Create plugin manager
func NewManager(db *gorm.DB) *Manager {
	mgr := &Manager{
		plugins: make(map[string]Plugin),
		db:      db,
	}

	// 自动迁移插件状态表
	_ = db.AutoMigrate(&PluginState{})

	return mgr
}

// Register 注册插件
func (m *Manager) Register(plugin Plugin) error {
	name := plugin.Name()

	// Check if plugin already registered
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	// Register plugin
	m.plugins[name] = plugin

	// 初始化插件状态（如果不存在）
	var state PluginState
	if err := m.db.Where("name = ?", name).First(&state).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 插件状态不存在，创建新记录（默认禁用）
			state = PluginState{
				Name:    name,
				Enabled: false,
			}
			if err := m.db.Create(&state).Error; err != nil {
				return fmt.Errorf("failed to create plugin state: %w", err)
			}
		}
	}

	return nil
}

// Enable 启用插件
func (m *Manager) Enable(name string) error {
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Execute plugin Enable method
	if err := plugin.Enable(m.db); err != nil {
		return err
	}

	// 同步插件菜单到数据库
	if err := m.syncPluginMenus(name, plugin.GetMenus()); err != nil {
		return fmt.Errorf("failed to sync plugin menus: %w", err)
	}

	// 更新插件状态为已启用
	if err := m.db.Model(&PluginState{}).Where("name = ?", name).Update("enabled", true).Error; err != nil {
		return fmt.Errorf("failed to update plugin state: %w", err)
	}

	return nil
}

// Disable 禁用插件
func (m *Manager) Disable(name string) error {
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Execute plugin Disable method
	if err := plugin.Disable(m.db); err != nil {
		return err
	}

	// 从数据库移除插件菜单
	if err := m.removePluginMenus(name); err != nil {
		return fmt.Errorf("failed to remove plugin menus: %w", err)
	}

	// 更新插件状态为已禁用
	if err := m.db.Model(&PluginState{}).Where("name = ?", name).Update("enabled", false).Error; err != nil {
		return fmt.Errorf("failed to update plugin state: %w", err)
	}

	return nil
}

// IsEnabled 检查插件是否已启用
func (m *Manager) IsEnabled(name string) bool {
	var state PluginState
	if err := m.db.Where("name = ?", name).First(&state).Error; err != nil {
		return false
	}
	return state.Enabled
}

// GetPlugin Get plugin
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	plugin, exists := m.plugins[name]
	return plugin, exists
}

// GetAllPlugins Get all plugins
func (m *Manager) GetAllPlugins() []Plugin {
	plugins := make([]Plugin, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// RegisterAllRoutes Register all plugin routes
func (m *Manager) RegisterAllRoutes(router *gin.RouterGroup) {
	for _, plugin := range m.plugins {
		// 只有启用的插件才注册路由
		if m.IsEnabled(plugin.Name()) {
			// 直接将 router 传给插件，让插件自己决定路径前缀
			plugin.RegisterRoutes(router, m.db)
		}
	}
}

// GetAllMenus Get all plugin menu configurations
func (m *Manager) GetAllMenus() []MenuConfig {
	allMenus := make([]MenuConfig, 0)
	for _, plugin := range m.plugins {
		// 只有启用的插件才返回菜单
		if m.IsEnabled(plugin.Name()) {
			menus := plugin.GetMenus()
			allMenus = append(allMenus, menus...)
		}
	}
	return allMenus
}

// pluginSysMenu 插件菜单数据库模型（本地定义，避免循环引用）
type pluginSysMenu struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string         `gorm:"type:varchar(50);not null;comment:菜单名称"`
	Code      string         `gorm:"type:varchar(50);uniqueIndex;comment:菜单编码"`
	Type      int            `gorm:"type:tinyint;not null;comment:类型 1:目录 2:菜单 3:按钮"`
	ParentID  uint           `gorm:"default:0;comment:父菜单ID"`
	Path      string         `gorm:"type:varchar(200);comment:路由路径"`
	Component string         `gorm:"type:varchar(200);comment:组件路径"`
	Icon      string         `gorm:"type:varchar(100);comment:图标"`
	Sort      int            `gorm:"type:int;default:0;comment:排序"`
	Visible   int            `gorm:"type:tinyint;default:1;comment:是否显示 1:显示 0:隐藏"`
	Status    int            `gorm:"type:tinyint;default:1;comment:状态 1:启用 0:禁用"`
}

func (pluginSysMenu) TableName() string {
	return "sys_menu"
}

// syncPluginMenus 同步插件菜单到数据库
func (m *Manager) syncPluginMenus(pluginName string, menus []MenuConfig) error {
	if len(menus) == 0 {
		return nil
	}

	// 生成菜单code前缀（将连字符替换为下划线）
	codePrefix := "_" + strings.ReplaceAll(pluginName, "-", "_")
	// 原始前缀（保留连字符，用于清理旧格式记录）
	originalPrefix := "_" + pluginName

	// 第一步：收集所有菜单路径
	allPaths := make([]string, 0, len(menus))
	for _, menu := range menus {
		allPaths = append(allPaths, menu.Path)
	}

	// 第二步：删除这些菜单的角色权限关联（先删除关联，再删除菜单）
	if err := m.db.Exec("DELETE FROM sys_role_menu WHERE menu_id IN (SELECT id FROM sys_menu WHERE path IN ?)", allPaths).Error; err != nil {
		return fmt.Errorf("failed to clean role_menu relations by path: %w", err)
	}

	// 第三步：硬删除所有已存在的同路径菜单（使用原生SQL确保完全删除）
	if err := m.db.Exec("DELETE FROM sys_menu WHERE path IN ?", allPaths).Error; err != nil {
		return fmt.Errorf("failed to clean existing menus by path: %w", err)
	}

	// 第四步：删除插件code前缀菜单的角色权限关联
	if err := m.db.Exec("DELETE FROM sys_role_menu WHERE menu_id IN (SELECT id FROM sys_menu WHERE code = ? OR code LIKE ?)", codePrefix, codePrefix+"_%").Error; err != nil {
		return fmt.Errorf("failed to clean role_menu relations by code prefix: %w", err)
	}

	// 第五步：硬删除所有以此插件code前缀开头的菜单
	if err := m.db.Exec("DELETE FROM sys_menu WHERE code = ? OR code LIKE ?", codePrefix, codePrefix+"_%").Error; err != nil {
		return fmt.Errorf("failed to clean existing menus by code prefix: %w", err)
	}

	// 第六步：如果插件名包含连字符，也删除旧格式（如 _ssl-cert）
	if codePrefix != originalPrefix {
		if err := m.db.Exec("DELETE FROM sys_role_menu WHERE menu_id IN (SELECT id FROM sys_menu WHERE code = ? OR code LIKE ?)", originalPrefix, originalPrefix+"_%").Error; err != nil {
			return fmt.Errorf("failed to clean role_menu relations by original prefix: %w", err)
		}
		if err := m.db.Exec("DELETE FROM sys_menu WHERE code = ? OR code LIKE ?", originalPrefix, originalPrefix+"_%").Error; err != nil {
			return fmt.Errorf("failed to clean existing menus by original prefix: %w", err)
		}
	}

	// 第七步：找出所有父菜单（ParentPath为空的）和子菜单
	parentMenus := make([]MenuConfig, 0)
	childMenus := make([]MenuConfig, 0)
	for _, menu := range menus {
		if menu.ParentPath == "" {
			parentMenus = append(parentMenus, menu)
		} else {
			childMenus = append(childMenus, menu)
		}
	}

	// 第八步：创建父菜单，建立 path -> id 映射
	pathToID := make(map[string]uint)
	// 收集所有创建的菜单ID，用于分配权限
	allMenuIDs := make([]uint, 0)

	for _, menu := range parentMenus {
		// 简化code生成：直接用 _pluginName 作为父菜单code
		menuCode := codePrefix
		visible := 1
		if menu.Hidden {
			visible = 0
		}

		newMenu := pluginSysMenu{
			Name:     menu.Name,
			Code:     menuCode,
			Type:     1, // 目录
			ParentID: 0,
			Path:     menu.Path,
			Icon:     menu.Icon,
			Sort:     menu.Sort,
			Visible:  visible,
			Status:   1,
		}
		if err := m.db.Create(&newMenu).Error; err != nil {
			return fmt.Errorf("failed to create parent menu %s: %w", menu.Path, err)
		}
		pathToID[menu.Path] = newMenu.ID
		allMenuIDs = append(allMenuIDs, newMenu.ID)
	}

	// 第九步：创建子菜单
	for _, menu := range childMenus {
		// 简化code生成：_pluginName_childPath
		menuCode := codePrefix + pathToCode(menu.Path)
		visible := 1
		if menu.Hidden {
			visible = 0
		}

		// 获取父菜单ID
		parentID, ok := pathToID[menu.ParentPath]
		if !ok {
			// 父菜单可能是系统菜单，尝试从数据库查找
			var parentMenu pluginSysMenu
			if err := m.db.Where("path = ?", menu.ParentPath).First(&parentMenu).Error; err != nil {
				return fmt.Errorf("parent menu %s not found for child %s: %w", menu.ParentPath, menu.Path, err)
			}
			parentID = parentMenu.ID
		}

		newMenu := pluginSysMenu{
			Name:     menu.Name,
			Code:     menuCode,
			Type:     2, // 菜单
			ParentID: parentID,
			Path:     menu.Path,
			Icon:     menu.Icon,
			Sort:     menu.Sort,
			Visible:  visible,
			Status:   1,
		}
		if err := m.db.Create(&newMenu).Error; err != nil {
			return fmt.Errorf("failed to create child menu %s: %w", menu.Path, err)
		}
		allMenuIDs = append(allMenuIDs, newMenu.ID)
	}

	// 第十步：为内置角色分配插件菜单权限
	if len(allMenuIDs) > 0 {
		var roles []struct {
			ID uint
		}
		if err := m.db.Table("sys_role").Select("id").Where("code IN ?", []string{"admin", "test_viewer"}).Find(&roles).Error; err != nil {
			fmt.Printf("warning: built-in roles not found, skip assigning plugin menu permissions: %v\n", err)
			return nil
		}

		for _, role := range roles {
			for _, menuID := range allMenuIDs {
				if err := m.db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?) ON DUPLICATE KEY UPDATE role_id = role_id", role.ID, menuID).Error; err != nil {
					fmt.Printf("warning: failed to assign menu %d to role %d: %v\n", menuID, role.ID, err)
				}
			}
		}
	}

	return nil
}

// removePluginMenus 从数据库移除插件菜单
func (m *Manager) removePluginMenus(pluginName string) error {
	codePrefix := "_" + strings.ReplaceAll(pluginName, "-", "_")
	originalPrefix := "_" + pluginName

	// 硬删除所有以此前缀开头的菜单
	if err := m.db.Exec("DELETE FROM sys_menu WHERE code = ? OR code LIKE ?", codePrefix, codePrefix+"_%").Error; err != nil {
		return fmt.Errorf("failed to remove plugin menus: %w", err)
	}

	// 如果插件名包含连字符，也删除旧格式
	if codePrefix != originalPrefix {
		if err := m.db.Exec("DELETE FROM sys_menu WHERE code = ? OR code LIKE ?", originalPrefix, originalPrefix+"_%").Error; err != nil {
			return fmt.Errorf("failed to remove plugin menus with original prefix: %w", err)
		}
	}

	return nil
}

// pathToCode 将路径转换为菜单code
// 例如: /task/execute -> _task_execute
func pathToCode(path string) string {
	// 移除开头的斜杠，然后将斜杠替换为下划线
	code := strings.TrimPrefix(path, "/")
	code = strings.ReplaceAll(code, "/", "_")
	code = strings.ReplaceAll(code, "-", "_")
	return "_" + code
}
