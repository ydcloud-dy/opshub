package kubernetes

import (
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/model"
	"gorm.io/gorm"
)

// AutoMigrate 自动迁移数据库表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.TerminalSession{},
	)
}
