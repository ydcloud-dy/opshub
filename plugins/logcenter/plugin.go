package logcenter

import (
	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/plugin"
	"github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	"github.com/ydcloud-dy/opshub/plugins/logcenter/server"
	"gorm.io/gorm"
)

// Plugin implements the OpsHub log center.
type Plugin struct {
	db   *gorm.DB
	name string
}

// New creates a log center plugin instance.
func New() *Plugin {
	return &Plugin{name: "logcenter"}
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Description() string {
	return "日志中心插件 - OpsHub 自带日志采集、ClickHouse 存储与统一查询"
}

func (p *Plugin) Version() string {
	return "1.0.0"
}

func (p *Plugin) Author() string {
	return "J"
}

func (p *Plugin) Enable(db *gorm.DB) error {
	p.db = db
	if err := db.AutoMigrate(
		&model.StorageCluster{},
		&model.QueryHistory{},
		&model.LogExportTask{},
		&model.QueryTemplate{},
		&model.SavedView{},
		&model.LibraryItem{},
		&model.FieldCatalog{},
		&model.AlertContext{},
		&model.CollectionPolicy{},
		&model.PolicyTarget{},
		&model.PolicyRevision{},
		&model.CollectorInstance{},
		&model.CollectorAssignment{},
		&model.ClusterCollectorCredential{},
		&model.AccessPolicy{},
		&model.RetentionPolicy{},
	); err != nil {
		return err
	}
	server.BootstrapStorageFromEnvironment(db)
	server.SyncInternalMonitorDataSources(db)
	return nil
}

func (p *Plugin) Disable(db *gorm.DB) error {
	return nil
}

func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	server.RegisterRoutes(router, db)
}

func (p *Plugin) GetMenus() []plugin.MenuConfig {
	const parentPath = "/logs"
	return []plugin.MenuConfig{
		{
			Name:       "日志中心",
			Path:       parentPath,
			Icon:       "Histogram",
			Sort:       25,
			ParentPath: "",
		},
		{
			Name:       "日志总览",
			Path:       "/logs/overview",
			Icon:       "DataAnalysis",
			Sort:       1,
			ParentPath: parentPath,
		},
		{
			Name:       "日志查询",
			Path:       "/logs/query",
			Icon:       "Search",
			Sort:       2,
			ParentPath: parentPath,
		},
		{
			Name:       "日志库",
			Path:       "/logs/library",
			Icon:       "FolderOpened",
			Sort:       3,
			ParentPath: parentPath,
		},
		{
			Name:       "查询模板",
			Path:       "/logs/templates",
			Icon:       "CollectionTag",
			Sort:       4,
			ParentPath: parentPath,
		},
		{
			Name:       "采集接入",
			Path:       "/logs/collectors",
			Icon:       "SetUp",
			Sort:       5,
			ParentPath: parentPath,
		},
	}
}
