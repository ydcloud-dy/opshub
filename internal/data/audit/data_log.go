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

package audit

import (
	"context"
	"time"

	"github.com/ydcloud-dy/opshub/internal/biz/audit"
	"gorm.io/gorm"
)

type dataLogRepo struct {
	db *gorm.DB
}

func NewDataLogRepo(db *gorm.DB) audit.DataLogRepo {
	return &dataLogRepo{db: db}
}

func (r *dataLogRepo) Create(ctx context.Context, log *audit.SysDataLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *dataLogRepo) GetByID(ctx context.Context, id uint) (*audit.SysDataLog, error) {
	var log audit.SysDataLog
	err := r.db.WithContext(ctx).First(&log, id).Error
	return &log, err
}

func (r *dataLogRepo) List(ctx context.Context, page, pageSize int, username, tableName, action, startTime, endTime string) ([]*audit.SysDataLog, int64, error) {
	var logs []*audit.SysDataLog
	var total int64

	query := r.db.WithContext(ctx).Model(&audit.SysDataLog{})

	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if tableName != "" {
		query = query.Where("table_name = ?", tableName)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if startTime != "" {
		t, err := time.Parse("2006-01-02", startTime)
		if err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if endTime != "" {
		t, err := time.Parse("2006-01-02", endTime)
		if err == nil {
			t = t.AddDate(0, 0, 1)
			query = query.Where("created_at < ?", t)
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page-1)*pageSize).Limit(pageSize).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *dataLogRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&audit.SysDataLog{}, id).Error
}

func (r *dataLogRepo) DeleteBatch(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Delete(&audit.SysDataLog{}, ids).Error
}
