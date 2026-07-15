package server

import (
	"fmt"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	monitormodel "github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"gorm.io/gorm"
)

func SyncInternalMonitorDataSources(db *gorm.DB) {
	if db == nil || !db.Migrator().HasTable(&monitormodel.DataSource{}) {
		return
	}
	var storages []logmodel.StorageCluster
	if db.Find(&storages).Error != nil {
		return
	}
	for _, storage := range storages {
		internalURL := fmt.Sprintf("internal://opshub-log-storage/%d", storage.ID)
		var datasource monitormodel.DataSource
		err := db.Where("type = ? AND url = ?", "opshub_logs", internalURL).First(&datasource).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			continue
		}
		datasource.Name = "OpsHub 内置日志 · " + storage.Name
		datasource.Type = "opshub_logs"
		datasource.URL = internalURL
		datasource.AuthType = "none"
		datasource.Timeout = maxInt(storage.Timeout, 30)
		datasource.Enabled = storage.Enabled && storage.InitializedAt != nil
		if storage.Status == "healthy" {
			datasource.Status = "normal"
		} else {
			datasource.Status = "unknown"
		}
		datasource.LastTestAt = storage.LastTestAt
		datasource.LastError = storage.LastError
		datasource.Description = "由日志中心自动维护，用于内置 Query AST 日志告警"
		if datasource.ID == 0 {
			_ = db.Create(&datasource).Error
		} else {
			_ = db.Save(&datasource).Error
		}
	}
}

func LoadInternalStorage(db *gorm.DB, storageID uint) (logmodel.StorageCluster, string, error) {
	if db == nil {
		return logmodel.StorageCluster{}, "", fmt.Errorf("数据库连接不可用")
	}
	var item logmodel.StorageCluster
	query := db.Where("enabled = ? AND storage_type = ?", true, "clickhouse")
	if storageID > 0 {
		query = query.Where("id = ?", storageID)
	} else {
		query = query.Order("is_primary DESC, created_at ASC")
	}
	if err := query.First(&item).Error; err != nil {
		return item, "", fmt.Errorf("没有可用的 ClickHouse 内置日志存储")
	}
	if item.InitializedAt == nil {
		return item, "", fmt.Errorf("ClickHouse 日志存储尚未初始化")
	}
	password, err := decryptStorageSecret(item.PasswordEncrypted)
	if err != nil {
		return item, "", err
	}
	return item, password, nil
}
