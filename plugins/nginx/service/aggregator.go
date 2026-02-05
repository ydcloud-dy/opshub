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

package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/nginx/model"
	"github.com/ydcloud-dy/opshub/plugins/nginx/repository"
)

// AggregatorService 聚合服务
type AggregatorService struct {
	repo  *repository.NginxRepository
	mutex sync.Mutex
}

// NewAggregatorService 创建聚合服务
func NewAggregatorService(repo *repository.NginxRepository) *AggregatorService {
	return &AggregatorService{
		repo: repo,
	}
}

// UpdateStatsFromLogs 根据日志更新统计数据
func (s *AggregatorService) UpdateStatsFromLogs(sourceID uint, logs []model.NginxAccessLog) error {
	if len(logs) == 0 {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 按天分组统计
	dailyStats := make(map[string]*dailyAggData)
	// 按小时分组统计
	hourlyStats := make(map[string]*hourlyAggData)

	for _, log := range logs {
		// 转换为本地时区，因为 MySQL 连接配置使用 loc=Local
		localTime := log.Timestamp.Local()
		dateKey := localTime.Format("2006-01-02")
		hourKey := localTime.Format("2006-01-02 15:00:00")

		// 初始化日统计
		if _, ok := dailyStats[dateKey]; !ok {
			date, _ := time.ParseInLocation("2006-01-02", dateKey, time.Local)
			dailyStats[dateKey] = &dailyAggData{
				sourceID: sourceID,
				date:     date,
				ips:      make(map[string]bool),
				methods:  make(map[string]int64),
			}
		}

		// 初始化小时统计
		if _, ok := hourlyStats[hourKey]; !ok {
			hour, _ := time.ParseInLocation("2006-01-02 15:04:05", hourKey, time.Local)
			hourlyStats[hourKey] = &hourlyAggData{
				sourceID: sourceID,
				hour:     hour,
				ips:      make(map[string]bool),
				methods:  make(map[string]int64),
			}
		}

		// 更新日统计
		daily := dailyStats[dateKey]
		daily.totalRequests++
		daily.totalBandwidth += log.BodyBytesSent
		daily.totalResponseTime += log.RequestTime
		daily.ips[log.RemoteAddr] = true
		daily.methods[log.Method]++

		if log.RequestTime > daily.maxResponseTime {
			daily.maxResponseTime = log.RequestTime
		}
		if daily.minResponseTime == 0 || log.RequestTime < daily.minResponseTime {
			daily.minResponseTime = log.RequestTime
		}

		// 判断是否为 PV
		if IsPVRequest(log.URI, log.Status) {
			daily.pvCount++
		}

		switch {
		case log.Status >= 200 && log.Status < 300:
			daily.status2xx++
		case log.Status >= 300 && log.Status < 400:
			daily.status3xx++
		case log.Status >= 400 && log.Status < 500:
			daily.status4xx++
		case log.Status >= 500:
			daily.status5xx++
		}

		// 更新小时统计
		hourly := hourlyStats[hourKey]
		hourly.totalRequests++
		hourly.totalBandwidth += log.BodyBytesSent
		hourly.totalResponseTime += log.RequestTime
		hourly.ips[log.RemoteAddr] = true
		hourly.methods[log.Method]++

		if log.RequestTime > hourly.maxResponseTime {
			hourly.maxResponseTime = log.RequestTime
		}
		if hourly.minResponseTime == 0 || log.RequestTime < hourly.minResponseTime {
			hourly.minResponseTime = log.RequestTime
		}

		if IsPVRequest(log.URI, log.Status) {
			hourly.pvCount++
		}

		switch {
		case log.Status >= 200 && log.Status < 300:
			hourly.status2xx++
		case log.Status >= 300 && log.Status < 400:
			hourly.status3xx++
		case log.Status >= 400 && log.Status < 500:
			hourly.status4xx++
		case log.Status >= 500:
			hourly.status5xx++
		}
	}

	// 保存日统计 (同时更新旧表和新表)
	for _, stats := range dailyStats {
		avgResponseTime := float64(0)
		if stats.totalRequests > 0 {
			avgResponseTime = stats.totalResponseTime / float64(stats.totalRequests)
		}

		// 新表: NginxAggDaily
		methodDist, _ := json.Marshal(stats.methods)
		aggDaily := &model.NginxAggDaily{
			SourceID:           stats.sourceID,
			Date:               stats.date,
			TotalRequests:      stats.totalRequests,
			PVCount:            stats.pvCount,
			UniqueIPs:          int64(len(stats.ips)),
			TotalBandwidth:     stats.totalBandwidth,
			AvgResponseTime:    avgResponseTime,
			MaxResponseTime:    stats.maxResponseTime,
			MinResponseTime:    stats.minResponseTime,
			Status2xx:          stats.status2xx,
			Status3xx:          stats.status3xx,
			Status4xx:          stats.status4xx,
			Status5xx:          stats.status5xx,
			MethodDistribution: string(methodDist),
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		// 合并已有数据
		existing, err := s.repo.GetAggDaily(stats.sourceID, stats.date)
		if err == nil && existing != nil {
			aggDaily.ID = existing.ID
			aggDaily.TotalRequests += existing.TotalRequests
			aggDaily.PVCount += existing.PVCount
			aggDaily.UniqueIPs += existing.UniqueIPs
			aggDaily.TotalBandwidth += existing.TotalBandwidth
			aggDaily.Status2xx += existing.Status2xx
			aggDaily.Status3xx += existing.Status3xx
			aggDaily.Status4xx += existing.Status4xx
			aggDaily.Status5xx += existing.Status5xx

			// 重新计算平均响应时间
			totalReqs := aggDaily.TotalRequests
			if totalReqs > 0 {
				aggDaily.AvgResponseTime = (existing.AvgResponseTime*float64(existing.TotalRequests) +
					avgResponseTime*float64(stats.totalRequests)) / float64(totalReqs)
			}

			// 更新最大最小值
			if existing.MaxResponseTime > aggDaily.MaxResponseTime {
				aggDaily.MaxResponseTime = existing.MaxResponseTime
			}
			if existing.MinResponseTime > 0 && existing.MinResponseTime < aggDaily.MinResponseTime {
				aggDaily.MinResponseTime = existing.MinResponseTime
			}
		}

		s.repo.CreateOrUpdateAggDaily(aggDaily)

		// 旧表: NginxDailyStats (兼容)
		dailyStatsOld := &model.NginxDailyStats{
			SourceID:        stats.sourceID,
			Date:            stats.date,
			TotalRequests:   stats.totalRequests,
			UniqueVisitors:  int64(len(stats.ips)),
			TotalBandwidth:  stats.totalBandwidth,
			AvgResponseTime: avgResponseTime,
			Status2xx:       stats.status2xx,
			Status3xx:       stats.status3xx,
			Status4xx:       stats.status4xx,
			Status5xx:       stats.status5xx,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		existingOld, err := s.repo.GetDailyStats(stats.sourceID, stats.date)
		if err == nil && existingOld != nil {
			dailyStatsOld.ID = existingOld.ID
			dailyStatsOld.TotalRequests += existingOld.TotalRequests
			dailyStatsOld.UniqueVisitors += existingOld.UniqueVisitors
			dailyStatsOld.TotalBandwidth += existingOld.TotalBandwidth
			dailyStatsOld.Status2xx += existingOld.Status2xx
			dailyStatsOld.Status3xx += existingOld.Status3xx
			dailyStatsOld.Status4xx += existingOld.Status4xx
			dailyStatsOld.Status5xx += existingOld.Status5xx

			totalReqs := dailyStatsOld.TotalRequests
			if totalReqs > 0 {
				dailyStatsOld.AvgResponseTime = (existingOld.AvgResponseTime*float64(existingOld.TotalRequests) +
					avgResponseTime*float64(stats.totalRequests)) / float64(totalReqs)
			}
		}

		s.repo.CreateOrUpdateDailyStats(dailyStatsOld)
	}

	// 保存小时统计 (同时更新旧表和新表)
	for _, stats := range hourlyStats {
		avgResponseTime := float64(0)
		if stats.totalRequests > 0 {
			avgResponseTime = stats.totalResponseTime / float64(stats.totalRequests)
		}

		// 新表: NginxAggHourly
		methodDist, _ := json.Marshal(stats.methods)
		aggHourly := &model.NginxAggHourly{
			SourceID:           stats.sourceID,
			Hour:               stats.hour,
			TotalRequests:      stats.totalRequests,
			PVCount:            stats.pvCount,
			UniqueIPs:          int64(len(stats.ips)),
			TotalBandwidth:     stats.totalBandwidth,
			AvgResponseTime:    avgResponseTime,
			MaxResponseTime:    stats.maxResponseTime,
			MinResponseTime:    stats.minResponseTime,
			Status2xx:          stats.status2xx,
			Status3xx:          stats.status3xx,
			Status4xx:          stats.status4xx,
			Status5xx:          stats.status5xx,
			MethodDistribution: string(methodDist),
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		existing, err := s.repo.GetAggHourly(stats.sourceID, stats.hour)
		if err == nil && existing != nil {
			aggHourly.ID = existing.ID
			aggHourly.TotalRequests += existing.TotalRequests
			aggHourly.PVCount += existing.PVCount
			aggHourly.UniqueIPs += existing.UniqueIPs
			aggHourly.TotalBandwidth += existing.TotalBandwidth
			aggHourly.Status2xx += existing.Status2xx
			aggHourly.Status3xx += existing.Status3xx
			aggHourly.Status4xx += existing.Status4xx
			aggHourly.Status5xx += existing.Status5xx

			totalReqs := aggHourly.TotalRequests
			if totalReqs > 0 {
				aggHourly.AvgResponseTime = (existing.AvgResponseTime*float64(existing.TotalRequests) +
					avgResponseTime*float64(stats.totalRequests)) / float64(totalReqs)
			}

			if existing.MaxResponseTime > aggHourly.MaxResponseTime {
				aggHourly.MaxResponseTime = existing.MaxResponseTime
			}
			if existing.MinResponseTime > 0 && existing.MinResponseTime < aggHourly.MinResponseTime {
				aggHourly.MinResponseTime = existing.MinResponseTime
			}
		}

		s.repo.CreateOrUpdateAggHourly(aggHourly)

		// 旧表: NginxHourlyStats (兼容)
		hourlyStatsOld := &model.NginxHourlyStats{
			SourceID:        stats.sourceID,
			Hour:            stats.hour,
			TotalRequests:   stats.totalRequests,
			UniqueVisitors:  int64(len(stats.ips)),
			TotalBandwidth:  stats.totalBandwidth,
			AvgResponseTime: avgResponseTime,
			Status2xx:       stats.status2xx,
			Status3xx:       stats.status3xx,
			Status4xx:       stats.status4xx,
			Status5xx:       stats.status5xx,
			CreatedAt:       time.Now(),
		}

		existingOld, err := s.repo.GetHourlyStats(stats.sourceID, stats.hour)
		if err == nil && existingOld != nil {
			hourlyStatsOld.ID = existingOld.ID
			hourlyStatsOld.TotalRequests += existingOld.TotalRequests
			hourlyStatsOld.UniqueVisitors += existingOld.UniqueVisitors
			hourlyStatsOld.TotalBandwidth += existingOld.TotalBandwidth
			hourlyStatsOld.Status2xx += existingOld.Status2xx
			hourlyStatsOld.Status3xx += existingOld.Status3xx
			hourlyStatsOld.Status4xx += existingOld.Status4xx
			hourlyStatsOld.Status5xx += existingOld.Status5xx

			totalReqs := hourlyStatsOld.TotalRequests
			if totalReqs > 0 {
				hourlyStatsOld.AvgResponseTime = (existingOld.AvgResponseTime*float64(existingOld.TotalRequests) +
					avgResponseTime*float64(stats.totalRequests)) / float64(totalReqs)
			}
		}

		s.repo.CreateOrUpdateHourlyStats(hourlyStatsOld)
	}

	return nil
}

// dailyAggData 日聚合数据
type dailyAggData struct {
	sourceID          uint
	date              time.Time
	totalRequests     int64
	pvCount           int64
	totalBandwidth    int64
	totalResponseTime float64
	maxResponseTime   float64
	minResponseTime   float64
	status2xx         int64
	status3xx         int64
	status4xx         int64
	status5xx         int64
	ips               map[string]bool
	methods           map[string]int64
}

// hourlyAggData 小时聚合数据
type hourlyAggData struct {
	sourceID          uint
	hour              time.Time
	totalRequests     int64
	pvCount           int64
	totalBandwidth    int64
	totalResponseTime float64
	maxResponseTime   float64
	minResponseTime   float64
	status2xx         int64
	status3xx         int64
	status4xx         int64
	status5xx         int64
	ips               map[string]bool
	methods           map[string]int64
}

// RunDailyAggregation 运行日聚合任务 (定时任务调用)
func (s *AggregatorService) RunDailyAggregation(sourceID uint, date time.Time) error {
	// 此方法可用于从事实表重新计算日聚合数据
	// 目前实现为增量更新，可选实现为全量重算
	fmt.Printf("运行日聚合任务: sourceID=%d, date=%s\n", sourceID, date.Format("2006-01-02"))
	return nil
}

// CleanupOldData 清理过期数据
func (s *AggregatorService) CleanupOldData(sourceID uint, retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 30
	}

	beforeTime := time.Now().AddDate(0, 0, -retentionDays)

	// 清理旧表
	if err := s.repo.DeleteOldAccessLogs(sourceID, beforeTime); err != nil {
		return fmt.Errorf("删除过期访问日志失败: %w", err)
	}

	// 清理新表
	if err := s.repo.DeleteOldFactAccessLogs(sourceID, beforeTime); err != nil {
		return fmt.Errorf("删除过期事实表日志失败: %w", err)
	}

	// 清理小时统计 (保留更久一些)
	hourlyBeforeTime := time.Now().AddDate(0, 0, -retentionDays*2)
	if err := s.repo.DeleteOldHourlyStats(sourceID, hourlyBeforeTime); err != nil {
		return fmt.Errorf("删除过期小时统计失败: %w", err)
	}

	if err := s.repo.DeleteOldAggHourly(sourceID, hourlyBeforeTime); err != nil {
		return fmt.Errorf("删除过期小时聚合失败: %w", err)
	}

	return nil
}
