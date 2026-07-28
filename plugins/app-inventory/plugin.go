package appinventory

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/internal/plugin"
	"github.com/ydcloud-dy/opshub/plugins/app-inventory/model"
	"github.com/ydcloud-dy/opshub/plugins/app-inventory/server"
	"github.com/ydcloud-dy/opshub/plugins/app-inventory/service"
)

type Plugin struct {
	db     *gorm.DB
	svc    *service.Service
	cipher *service.SecretCipher
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return "app-inventory" }
func (p *Plugin) Description() string {
	return "应用资产中心：应用、环境、域名、证书、资源、数据库、中间件和依赖拓扑"
}
func (p *Plugin) Version() string { return "0.1.0" }
func (p *Plugin) Author() string  { return "OpsHub" }

func (p *Plugin) Enable(db *gorm.DB) error {
	p.db = db
	for _, item := range model.AllModels() {
		if err := db.AutoMigrate(item); err != nil {
			return fmt.Errorf("应用资产中心迁移 %T 失败: %w", item, err)
		}
	}
	if err := service.MigrateLegacyRelationships(db); err != nil {
		return fmt.Errorf("应用资产中心关系迁移失败: %w", err)
	}
	var err error
	p.cipher, err = service.NewSecretCipher(os.Getenv("OPSHUB_APP_INVENTORY_SECRET_KEY"))
	if err != nil {
		// Keep the inventory read/write APIs available, but never fall back to a hardcoded key.
		fmt.Printf("app-inventory: credential vault disabled: %v\n", err)
		p.cipher = nil
	}
	p.svc = service.New(db, p.cipher)
	p.svc.StartHealthMonitor()
	return nil
}

func (p *Plugin) Disable(db *gorm.DB) error {
	if p.svc != nil {
		p.svc.StopHealthMonitor()
	}
	p.svc = nil
	p.cipher = nil
	return nil
}

func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	if p.svc == nil {
		p.svc = service.New(db, p.cipher)
	}
	server.RegisterRoutes(router, p.svc)
}

func (p *Plugin) GetMenus() []plugin.MenuConfig {
	parent := "/app-inventory"
	return []plugin.MenuConfig{
		{Name: "应用资产", Path: parent, Icon: "Collection", Sort: 18, ParentPath: ""},
		{Name: "资产总览", Path: parent + "/overview", Icon: "DataAnalysis", Sort: 1, ParentPath: parent},
		{Name: "应用台账", Path: parent + "/apps", Icon: "Grid", Sort: 2, ParentPath: parent},
		{Name: "环境管理", Path: parent + "/environments", Icon: "SetUp", Sort: 3, ParentPath: parent},
		{Name: "依赖拓扑", Path: parent + "/topology", Icon: "Share", Sort: 4, ParentPath: parent},
		{Name: "资源与域名", Path: parent + "/resources", Icon: "Coin", Sort: 5, ParentPath: parent},
		{Name: "凭据中心", Path: parent + "/credentials", Icon: "Lock", Sort: 6, ParentPath: parent},
	}
}
