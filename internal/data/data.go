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

package data

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ydcloud-dy/opshub/internal/conf"
	// 导入 MySQL 驱动，确保 time.Time 类型正确处理
	_ "github.com/go-sql-driver/mysql"
)

// Data 数据层
type Data struct {
	db *gorm.DB
}

// NewData 创建数据层
func NewData(c *conf.Config) (*Data, error) {
	// 初始化 MySQL
	db, err := newMySQL(c.Database)
	if err != nil {
		return nil, fmt.Errorf("初始化MySQL失败: %w", err)
	}

	return &Data{
		db: db,
	}, nil
}

// newMySQL 创建 MySQL 连接
func newMySQL(cfg conf.DatabaseConfig) (*gorm.DB, error) {
	// GORM 配置 - 禁用外键约束
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 显示所有SQL
		DisableForeignKeyConstraintWhenMigrating: true, // 迁移时不创建外键约束
		NowFunc: func() time.Time {
			// 使用本地时区，确保时间字段正确处理
			return time.Now().Local()
		},
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(cfg.GetDSN()), gormConfig)
	if err != nil {
		return nil, err
	}

	// 禁用外键检查（只在当前会话）
	db.Exec("SET FOREIGN_KEY_CHECKS = 0;")

	// 获取底层 sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	return db, nil
}

// DB 获取数据库连接
func (d *Data) DB() *gorm.DB {
	return d.db
}

// Close 关闭连接
func (d *Data) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
