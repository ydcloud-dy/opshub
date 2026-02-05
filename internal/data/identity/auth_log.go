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

package identity

import (
	"context"
	"time"

	"github.com/ydcloud-dy/opshub/internal/biz/identity"
	"gorm.io/gorm"
)

type authLogRepo struct {
	db *gorm.DB
}

func NewAuthLogRepo(db *gorm.DB) identity.AuthLogRepo {
	return &authLogRepo{db: db}
}

func (r *authLogRepo) Create(ctx context.Context, log *identity.AuthLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *authLogRepo) List(ctx context.Context, page, pageSize int, userID *uint, action, result string, startTime, endTime string) ([]*identity.AuthLog, int64, error) {
	var logs []*identity.AuthLog
	var total int64

	query := r.db.WithContext(ctx).Model(&identity.AuthLog{})
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if result != "" {
		query = query.Where("result = ?", result)
	}
	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&logs).Error

	return logs, total, err
}

func (r *authLogRepo) GetStats(ctx context.Context, startTime, endTime string) (*identity.AuthLogStats, error) {
	stats := &identity.AuthLogStats{}

	query := r.db.WithContext(ctx).Model(&identity.AuthLog{})
	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	// 总登录次数
	query.Where("action = ?", "login").Count(&stats.TotalLogins)

	// 今日登录次数
	today := time.Now().Format("2006-01-02")
	r.db.WithContext(ctx).Model(&identity.AuthLog{}).
		Where("action = ? AND DATE(created_at) = ?", "login", today).
		Count(&stats.TodayLogins)

	// 失败登录次数
	r.db.WithContext(ctx).Model(&identity.AuthLog{}).
		Where("action = ? AND result = ?", "login", "failed").
		Count(&stats.FailedLogins)

	// 独立用户数
	r.db.WithContext(ctx).Model(&identity.AuthLog{}).
		Where("action = ? AND result = ?", "login", "success").
		Distinct("user_id").
		Count(&stats.UniqueUsers)

	// 应用访问次数
	r.db.WithContext(ctx).Model(&identity.AuthLog{}).
		Where("action = ?", "access_app").
		Count(&stats.AppAccessCount)

	// 登录趋势
	trend, _ := r.GetLoginTrend(ctx, 7)
	stats.LoginTrend = trend

	// Top应用
	topApps, _ := r.GetTopApps(ctx, 5, startTime, endTime)
	stats.TopApps = topApps

	// Top用户
	topUsers, _ := r.GetTopUsers(ctx, 5, startTime, endTime)
	stats.TopUsers = topUsers

	return stats, nil
}

func (r *authLogRepo) GetLoginTrend(ctx context.Context, days int) ([]identity.TrendPoint, error) {
	var results []identity.TrendPoint

	startDate := time.Now().AddDate(0, 0, -days+1).Format("2006-01-02")

	rows, err := r.db.WithContext(ctx).Model(&identity.AuthLog{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("action = ? AND result = ? AND DATE(created_at) >= ?", "login", "success", startDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var point identity.TrendPoint
		rows.Scan(&point.Date, &point.Count)
		results = append(results, point)
	}

	return results, nil
}

func (r *authLogRepo) GetTopApps(ctx context.Context, limit int, startTime, endTime string) ([]identity.TopAppStat, error) {
	var results []identity.TopAppStat

	query := r.db.WithContext(ctx).Model(&identity.AuthLog{}).
		Select("app_id, app_name, COUNT(*) as count").
		Where("action = ? AND app_id > 0", "access_app")

	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	err := query.Group("app_id, app_name").
		Order("count DESC").
		Limit(limit).
		Find(&results).Error

	return results, err
}

func (r *authLogRepo) GetTopUsers(ctx context.Context, limit int, startTime, endTime string) ([]identity.TopUserStat, error) {
	var results []identity.TopUserStat

	query := r.db.WithContext(ctx).Model(&identity.AuthLog{}).
		Select("user_id, username, COUNT(*) as count").
		Where("action = ? AND result = ?", "login", "success")

	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	err := query.Group("user_id, username").
		Order("count DESC").
		Limit(limit).
		Find(&results).Error

	return results, err
}
