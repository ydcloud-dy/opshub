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
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ydcloud-dy/opshub/plugins/nginx/model"
	"github.com/ydcloud-dy/opshub/plugins/nginx/repository"
)

// CollectorService 日志采集服务
type CollectorService struct {
	repo       *repository.NginxRepository
	parser     *ParserService
	geo        *GeolocationService
	aggregator *AggregatorService
}

// NewCollectorService 创建采集服务
func NewCollectorService(repo *repository.NginxRepository, parser *ParserService, geo *GeolocationService, aggregator *AggregatorService) *CollectorService {
	return &CollectorService{
		repo:       repo,
		parser:     parser,
		geo:        geo,
		aggregator: aggregator,
	}
}

// CollectResult 采集结果
type CollectResult struct {
	SourceID      uint
	SourceName    string
	Status        string
	LogsCollected int
	Error         string
}

// CollectK8sIngressLogs 采集 K8s Ingress-Nginx 日志
func (s *CollectorService) CollectK8sIngressLogs(ctx context.Context, source *model.NginxSource, clientset *kubernetes.Clientset) (*CollectResult, error) {
	result := &CollectResult{
		SourceID:   source.ID,
		SourceName: source.Name,
	}

	if source.ClusterID == nil {
		result.Status = "failed"
		result.Error = "数据源未关联集群"
		return result, fmt.Errorf(result.Error)
	}

	namespace := source.Namespace
	if namespace == "" {
		namespace = "ingress-nginx"
	}

	podSelector := source.K8sPodSelector
	if podSelector == "" {
		podSelector = "app.kubernetes.io/name=ingress-nginx,app.kubernetes.io/component=controller"
	}

	containerName := source.K8sContainerName
	if containerName == "" {
		containerName = "controller"
	}

	// 列出 Ingress-Nginx Controller Pods
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: podSelector,
	})
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("列出 Pod 失败: %v", err)
		return result, err
	}

	if len(pods.Items) == 0 {
		result.Status = "failed"
		result.Error = "未找到 Ingress-Nginx Controller Pod"
		return result, fmt.Errorf(result.Error)
	}

	// 计算采集时间范围
	sinceSeconds := int64(source.CollectInterval)
	if sinceSeconds < 60 {
		sinceSeconds = 60
	}

	var allLogs []model.ParsedLogEntry

	// 从每个 Pod 获取日志
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		podLogOptions := &corev1.PodLogOptions{
			Container:    containerName,
			Timestamps:   true,
			SinceSeconds: &sinceSeconds,
		}

		logStream, err := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, podLogOptions).Stream(ctx)
		if err != nil {
			// 记录错误但继续处理其他 Pod
			continue
		}

		logContent, err := io.ReadAll(logStream)
		logStream.Close()
		if err != nil {
			continue
		}

		// 解析日志
		entries := s.parser.ParseLogs(string(logContent), source.LogFormat)
		for i := range entries {
			entries[i].PodName = pod.Name
		}
		allLogs = append(allLogs, entries...)
	}

	if len(allLogs) == 0 {
		result.Status = "success"
		result.LogsCollected = 0
		return result, nil
	}

	// 处理并存储日志
	logsStored, err := s.processAndStoreLogs(source, allLogs)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("存储日志失败: %v", err)
		return result, err
	}

	// 更新数据源采集状态
	s.repo.UpdateSourceCollectStatus(source.ID, int64(logsStored), "")

	result.Status = "success"
	result.LogsCollected = logsStored

	return result, nil
}

// ProcessHostLogs 处理主机日志内容
func (s *CollectorService) ProcessHostLogs(source *model.NginxSource, logContent string) (int, error) {
	// 解析日志
	entries := s.parser.ParseLogs(logContent, source.LogFormat)
	if len(entries) == 0 {
		return 0, nil
	}

	// 处理并存储日志
	logsStored, err := s.processAndStoreLogs(source, entries)
	if err != nil {
		return 0, err
	}

	// 更新数据源采集状态
	s.repo.UpdateSourceCollectStatus(source.ID, int64(logsStored), "")

	return logsStored, nil
}

// processAndStoreLogs 处理并存储日志
func (s *CollectorService) processAndStoreLogs(source *model.NginxSource, entries []model.ParsedLogEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}

	// 同时存储到旧表和新表
	var accessLogs []model.NginxAccessLog
	var factLogs []model.NginxFactAccessLog

	uaParser := s.parser.GetUAParser()

	for _, entry := range entries {
		// 存储到旧表 (兼容)
		accessLog := model.NginxAccessLog{
			SourceID:      source.ID,
			Timestamp:     entry.Timestamp,
			RemoteAddr:    entry.RemoteAddr,
			RemoteUser:    entry.RemoteUser,
			Request:       fmt.Sprintf("%s %s %s", entry.Method, entry.URI, entry.Protocol),
			Method:        entry.Method,
			URI:           entry.URI,
			Protocol:      entry.Protocol,
			Status:        entry.Status,
			BodyBytesSent: entry.BodyBytesSent,
			HTTPReferer:   entry.HTTPReferer,
			HTTPUserAgent: entry.HTTPUserAgent,
			RequestTime:   entry.RequestTime,
			UpstreamTime:  entry.UpstreamTime,
			Host:          entry.Host,
			IngressName:   entry.IngressName,
			ServiceName:   entry.ServiceName,
			CreatedAt:     time.Now(),
		}
		accessLogs = append(accessLogs, accessLog)

		// 创建维度记录
		var ipID, urlID, refererID, uaID uint64

		// IP 维度
		ipID, err := s.repo.GetOrCreateDimIP(entry.RemoteAddr)
		if err != nil {
			continue
		}

		// 更新地理位置信息 (异步或按需)
		if source.GeoEnabled {
			go s.updateIPGeo(ipID, entry.RemoteAddr)
		}

		// URL 维度
		urlHash := HashString(entry.URI + entry.Host)
		urlNormalized := NormalizeURL(entry.URI)
		urlID, err = s.repo.GetOrCreateDimURL(urlHash, entry.URI, urlNormalized, entry.Host)
		if err != nil {
			continue
		}

		// Referer 维度
		if entry.HTTPReferer != "" {
			refererHash := HashString(entry.HTTPReferer)
			refererDomain := ExtractRefererDomain(entry.HTTPReferer)
			refererType := ClassifyReferer(entry.HTTPReferer)
			refererID, _ = s.repo.GetOrCreateDimReferer(refererHash, entry.HTTPReferer, refererDomain, refererType)
		}

		// UserAgent 维度
		if entry.HTTPUserAgent != "" {
			uaHash := HashString(entry.HTTPUserAgent)
			uaInfo := uaParser.Parse(entry.HTTPUserAgent)
			uaID, _ = s.repo.GetOrCreateDimUserAgent(
				uaHash,
				entry.HTTPUserAgent,
				uaInfo.Browser,
				uaInfo.BrowserVersion,
				uaInfo.OS,
				uaInfo.OSVersion,
				uaInfo.DeviceType,
				uaInfo.IsBot,
			)
		}

		// 存储到新表
		factLog := model.NginxFactAccessLog{
			SourceID:      source.ID,
			Timestamp:     entry.Timestamp,
			IPID:          ipID,
			URLID:         urlID,
			RefererID:     refererID,
			UAID:          uaID,
			Method:        entry.Method,
			Protocol:      entry.Protocol,
			Status:        entry.Status,
			BodyBytesSent: entry.BodyBytesSent,
			RequestTime:   entry.RequestTime,
			UpstreamTime:  entry.UpstreamTime,
			IngressName:   entry.IngressName,
			ServiceName:   entry.ServiceName,
			PodName:       entry.PodName,
			IsPV:          IsPVRequest(entry.URI, entry.Status),
			CreatedAt:     time.Now(),
		}
		factLogs = append(factLogs, factLog)
	}

	// 批量插入旧表
	if err := s.repo.BatchCreateAccessLogs(accessLogs); err != nil {
		return 0, fmt.Errorf("保存访问日志失败: %w", err)
	}

	// 批量插入新表
	if err := s.repo.BatchCreateFactAccessLogs(factLogs); err != nil {
		// 新表插入失败不阻断流程
		fmt.Printf("保存事实表日志失败: %v\n", err)
	}

	// 更新聚合统计
	if s.aggregator != nil {
		go s.aggregator.UpdateStatsFromLogs(source.ID, accessLogs)
	}

	return len(accessLogs), nil
}

// updateIPGeo 更新 IP 地理位置信息
func (s *CollectorService) updateIPGeo(ipID uint64, ip string) {
	if s.geo == nil {
		return
	}

	geoInfo, err := s.geo.Lookup(ip)
	if err != nil {
		return
	}

	s.repo.UpdateDimIPGeo(ipID, geoInfo.Country, geoInfo.Province, geoInfo.City, geoInfo.ISP, false)
}
