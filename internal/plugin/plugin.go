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
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Plugin 插件接口
// All plugins must implement this interface
type Plugin interface {
	// Name Plugin unique identifier
	Name() string

	// Description Plugin description
	Description() string

	// Version Plugin version
	Version() string

	// Author Plugin author
	Author() string

	// Enable Enable plugin
	// Initialize plugin resources, database tables, etc.
	Enable(db *gorm.DB) error

	// Disable Disable plugin
	// Clean up plugin resources (note: won't delete database tables by default)
	Disable(db *gorm.DB) error

	// RegisterRoutes Register routes
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
