package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ydcloud-dy/opshub/plugins/app-inventory/model"
	"gorm.io/gorm"
)

// MigrateLegacyRelationships converts the original application-owned environment model
// into the current shared-environment model without deleting historical records.
func MigrateLegacyRelationships(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.Environment{}) {
		return nil
	}
	hasLegacyApplicationID := db.Migrator().HasColumn("app_inventory_environments", "application_id")
	if hasLegacyApplicationID && db.Dialector.Name() == "mysql" {
		if err := db.Exec("ALTER TABLE app_inventory_environments MODIFY COLUMN application_id bigint unsigned NOT NULL DEFAULT 0").Error; err != nil {
			return fmt.Errorf("调整旧环境关联字段失败: %w", err)
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if hasLegacyApplicationID {
			if err := migrateLegacyApplicationEnvironments(tx); err != nil {
				return err
			}
		}
		var apps []model.Application
		if err := tx.Select("id, environment_id").Find(&apps).Error; err != nil {
			return err
		}
		for _, app := range apps {
			if app.EnvironmentID == 0 {
				continue
			}
			if err := syncApplicationEnvironment(tx, app.ID, app.EnvironmentID); err != nil {
				return fmt.Errorf("同步应用 %d 的环境关系失败: %w", app.ID, err)
			}
		}
		return tx.Model(&model.Application{}).
			Where("health_source = '' OR health_source IS NULL").
			Updates(map[string]interface{}{"health_status": "unknown", "health_message": "等待自动探测", "health_source": "asset-aggregation"}).Error
	})
}

func migrateLegacyApplicationEnvironments(tx *gorm.DB) error {
	type legacyEnvironment struct {
		ID            uint
		ApplicationID uint
		Kind          string
	}
	var legacy []legacyEnvironment
	if err := tx.Table("app_inventory_environments").
		Select("id, application_id, kind").
		Where("application_id > 0 AND deleted_at IS NULL").
		Find(&legacy).Error; err != nil {
		return fmt.Errorf("读取旧环境关系失败: %w", err)
	}
	grouped := make(map[uint][]legacyEnvironment)
	for _, item := range legacy {
		grouped[item.ApplicationID] = append(grouped[item.ApplicationID], item)
	}
	for appID, candidates := range grouped {
		sort.SliceStable(candidates, func(i, j int) bool {
			left, right := environmentPriority(candidates[i].Kind), environmentPriority(candidates[j].Kind)
			if left == right {
				return candidates[i].ID < candidates[j].ID
			}
			return left < right
		})
		var current struct {
			ID            uint
			EnvironmentID uint
		}
		if err := tx.Model(&model.Application{}).Select("id, environment_id").Where("id = ?", appID).First(&current).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		if current.EnvironmentID > 0 || len(candidates) == 0 {
			continue
		}
		selected := candidates[0]
		if err := tx.Model(&model.Application{}).Where("id = ?", appID).Updates(map[string]interface{}{
			"environment_id": selected.ID,
			"lifecycle":      lifecycleFromEnvironment(selected.Kind),
		}).Error; err != nil {
			return fmt.Errorf("迁移应用 %d 的运行环境失败: %w", appID, err)
		}
	}
	return nil
}

func environmentPriority(kind string) int {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "production":
		return 0
	case "staging":
		return 1
	case "test":
		return 2
	case "development":
		return 3
	default:
		return 4
	}
}
