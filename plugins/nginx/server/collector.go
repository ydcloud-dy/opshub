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

package server

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"github.com/ydcloud-dy/opshub/plugins/nginx/model"
)

// 加密密钥（与凭证仓库相同）
var encryptionKey = []byte("opshub-enc-key-32-bytes-long!!!!")

// NginxLogEntry 解析后的日志条目
type NginxLogEntry struct {
	Timestamp     time.Time
	RemoteAddr    string
	RemoteUser    string
	Request       string
	Method        string
	URI           string
	Protocol      string
	Status        int
	BodyBytesSent int64
	HTTPReferer   string
	HTTPUserAgent string
	RequestTime   float64
	Host          string
}

// CollectLogs 手动触发日志采集
// @Summary 手动触发日志采集
// @Description 立即采集指定数据源或所有数据源的Nginx日志
// @Tags Nginx统计-数据源
// @Accept json
// @Produce json
// @Security Bearer
// @Param sourceId query int false "数据源ID（不指定则采集所有）"
// @Success 200 {object} response.Response "采集成功"
// @Router /nginx/collect [post]
func (h *Handler) CollectLogs(c *gin.Context) {
	sourceIDStr := c.Query("sourceId")

	var sources []model.NginxSource
	var err error

	if sourceIDStr != "" {
		// 采集指定数据源
		id, _ := strconv.ParseUint(sourceIDStr, 10, 32)
		source, err := h.repo.GetSourceByID(uint(id))
		if err != nil {
			response.ErrorCode(c, http.StatusNotFound, "数据源不存在")
			return
		}
		sources = []model.NginxSource{*source}
	} else {
		// 采集所有活跃数据源
		sources, err = h.repo.GetActiveSources()
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "获取数据源列表失败")
			return
		}
	}

	if len(sources) == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "没有可采集的数据源")
		return
	}

	// 采集结果
	results := make([]map[string]interface{}, 0)

	for _, source := range sources {
		result := map[string]interface{}{
			"sourceId":   source.ID,
			"sourceName": source.Name,
			"type":       source.Type,
		}

		switch source.Type {
		case model.SourceTypeHost:
			collected, parseErr := h.collectHostLogs(&source)
			if parseErr != nil {
				result["status"] = "failed"
				result["error"] = parseErr.Error()
			} else {
				result["status"] = "success"
				result["logsCollected"] = collected
			}
		case model.SourceTypeK8sIngress:
			// TODO: K8s Ingress 采集
			result["status"] = "skipped"
			result["error"] = "K8s Ingress 采集暂未实现"
		default:
			result["status"] = "skipped"
			result["error"] = "不支持的数据源类型"
		}

		results = append(results, result)
	}

	response.Success(c, gin.H{
		"message": "采集完成",
		"results": results,
	})
}

// collectHostLogs 采集主机上的Nginx日志
func (h *Handler) collectHostLogs(source *model.NginxSource) (int, error) {
	if source.HostID == nil {
		return 0, fmt.Errorf("数据源未关联主机")
	}

	// 获取主机信息
	var host assetbiz.Host
	if err := h.db.First(&host, *source.HostID).Error; err != nil {
		return 0, fmt.Errorf("获取主机信息失败: %w", err)
	}

	// 获取凭证信息
	var credential assetbiz.Credential
	if err := h.db.First(&credential, host.CredentialID).Error; err != nil {
		return 0, fmt.Errorf("获取凭证信息失败: %w", err)
	}

	// 解密凭证
	if err := decryptCredential(&credential); err != nil {
		return 0, fmt.Errorf("解密凭证失败: %w", err)
	}

	// 创建SSH连接
	sshClient, err := createSSHClient(&host, &credential)
	if err != nil {
		return 0, fmt.Errorf("SSH连接失败: %w", err)
	}
	defer sshClient.Close()

	// 读取日志文件（最近1000行）
	logPath := source.LogPath
	if logPath == "" {
		logPath = "/var/log/nginx/access.log"
	}

	session, err := sshClient.NewSession()
	if err != nil {
		return 0, fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 读取最近的日志
	output, err := session.CombinedOutput(fmt.Sprintf("tail -n 1000 %s 2>/dev/null || cat %s 2>/dev/null | head -n 1000", logPath, logPath))
	if err != nil {
		return 0, fmt.Errorf("读取日志文件失败: %w", err)
	}

	// 解析日志
	logs := parseNginxLogs(string(output), source.LogFormat)
	if len(logs) == 0 {
		return 0, nil
	}

	// 转换为数据库模型，并截断超长字段以防止数据库错误
	// 同时填充地理位置和UA信息
	accessLogs := make([]model.NginxAccessLog, 0, len(logs))
	for _, log := range logs {
		// 解析地理位置
		geoInfo, _ := h.geoSvc.Lookup(log.RemoteAddr)

		// 解析 User-Agent
		uaInfo := h.uaParser.Parse(log.HTTPUserAgent)

		accessLog := model.NginxAccessLog{
			SourceID:       source.ID,
			Timestamp:      log.Timestamp,
			RemoteAddr:     truncateString(log.RemoteAddr, 50),
			RemoteUser:     truncateString(log.RemoteUser, 100),
			Request:        truncateString(log.Request, 2000),
			Method:         truncateString(log.Method, 20),
			URI:            truncateString(log.URI, 1000),
			Protocol:       truncateString(log.Protocol, 50),
			Status:         log.Status,
			BodyBytesSent:  log.BodyBytesSent,
			HTTPReferer:    truncateString(log.HTTPReferer, 1000),
			HTTPUserAgent:  truncateString(log.HTTPUserAgent, 500),
			RequestTime:    log.RequestTime,
			Host:           truncateString(log.Host, 255),
			CreatedAt:      time.Now(),
		}

		// 填充地理位置信息
		if geoInfo != nil {
			accessLog.Country = truncateString(geoInfo.Country, 50)
			accessLog.Province = truncateString(geoInfo.Province, 50)
			accessLog.City = truncateString(geoInfo.City, 50)
			accessLog.ISP = truncateString(geoInfo.ISP, 100)
		}

		// 填充 UA 解析信息 (uaInfo 是值类型，不需要判空)
		accessLog.Browser = truncateString(uaInfo.Browser, 50)
		accessLog.BrowserVersion = truncateString(uaInfo.BrowserVersion, 20)
		accessLog.OS = truncateString(uaInfo.OS, 50)
		accessLog.OSVersion = truncateString(uaInfo.OSVersion, 20)
		accessLog.DeviceType = truncateString(uaInfo.DeviceType, 20)

		accessLogs = append(accessLogs, accessLog)
	}

	// 批量插入日志
	if err := h.repo.BatchCreateAccessLogs(accessLogs); err != nil {
		return 0, fmt.Errorf("保存日志失败: %w", err)
	}

	// 更新统计数据
	if err := h.updateStats(source.ID, accessLogs); err != nil {
		// 统计更新失败不影响日志采集
		fmt.Printf("更新统计数据失败: %v\n", err)
	}

	return len(accessLogs), nil
}

// updateStats 更新统计数据
func (h *Handler) updateStats(sourceID uint, logs []model.NginxAccessLog) error {
	if len(logs) == 0 {
		return nil
	}

	// 按天分组统计
	dailyStats := make(map[string]*model.NginxDailyStats)
	// 按小时分组统计
	hourlyStats := make(map[string]*model.NginxHourlyStats)
	// 独立IP统计
	dailyIPs := make(map[string]map[string]bool)
	hourlyIPs := make(map[string]map[string]bool)

	for _, log := range logs {
		dateKey := log.Timestamp.Format("2006-01-02")
		hourKey := log.Timestamp.Format("2006-01-02 15:00:00")

		// 初始化日统计
		if _, ok := dailyStats[dateKey]; !ok {
			// 使用本地时区解析日期，确保与查询时的时区一致
			date, _ := time.ParseInLocation("2006-01-02", dateKey, time.Local)
			dailyStats[dateKey] = &model.NginxDailyStats{
				SourceID: sourceID,
				Date:     date,
			}
			dailyIPs[dateKey] = make(map[string]bool)
		}

		// 初始化小时统计
		if _, ok := hourlyStats[hourKey]; !ok {
			// 使用本地时区解析小时，确保与查询时的时区一致
			hour, _ := time.ParseInLocation("2006-01-02 15:04:05", hourKey+":00", time.Local)
			hourlyStats[hourKey] = &model.NginxHourlyStats{
				SourceID: sourceID,
				Hour:     hour,
			}
			hourlyIPs[hourKey] = make(map[string]bool)
		}

		// 更新日统计
		daily := dailyStats[dateKey]
		daily.TotalRequests++
		daily.TotalBandwidth += log.BodyBytesSent
		daily.AvgResponseTime = (daily.AvgResponseTime*float64(daily.TotalRequests-1) + log.RequestTime) / float64(daily.TotalRequests)
		dailyIPs[dateKey][log.RemoteAddr] = true

		switch {
		case log.Status >= 200 && log.Status < 300:
			daily.Status2xx++
		case log.Status >= 300 && log.Status < 400:
			daily.Status3xx++
		case log.Status >= 400 && log.Status < 500:
			daily.Status4xx++
		case log.Status >= 500:
			daily.Status5xx++
		}

		// 更新小时统计
		hourly := hourlyStats[hourKey]
		hourly.TotalRequests++
		hourly.TotalBandwidth += log.BodyBytesSent
		hourly.AvgResponseTime = (hourly.AvgResponseTime*float64(hourly.TotalRequests-1) + log.RequestTime) / float64(hourly.TotalRequests)
		hourlyIPs[hourKey][log.RemoteAddr] = true

		switch {
		case log.Status >= 200 && log.Status < 300:
			hourly.Status2xx++
		case log.Status >= 300 && log.Status < 400:
			hourly.Status3xx++
		case log.Status >= 400 && log.Status < 500:
			hourly.Status4xx++
		case log.Status >= 500:
			hourly.Status5xx++
		}
	}

	// 保存日统计
	for dateKey, stats := range dailyStats {
		stats.UniqueVisitors = int64(len(dailyIPs[dateKey]))

		// 尝试获取已有统计，合并数据
		existing, err := h.repo.GetDailyStats(sourceID, stats.Date)
		if err == nil && existing != nil {
			// 合并已有数据
			existing.TotalRequests += stats.TotalRequests
			existing.TotalBandwidth += stats.TotalBandwidth
			existing.Status2xx += stats.Status2xx
			existing.Status3xx += stats.Status3xx
			existing.Status4xx += stats.Status4xx
			existing.Status5xx += stats.Status5xx
			// 独立访客需要重新计算，这里简单相加（可能有重复）
			existing.UniqueVisitors += stats.UniqueVisitors
			// 平均响应时间重新计算
			totalReqs := existing.TotalRequests
			if totalReqs > 0 {
				existing.AvgResponseTime = (existing.AvgResponseTime*float64(existing.TotalRequests-stats.TotalRequests) +
					stats.AvgResponseTime*float64(stats.TotalRequests)) / float64(totalReqs)
			}
			stats = existing
		}

		if err := h.repo.CreateOrUpdateDailyStats(stats); err != nil {
			return fmt.Errorf("保存日统计失败: %w", err)
		}
	}

	// 保存小时统计
	for hourKey, stats := range hourlyStats {
		stats.UniqueVisitors = int64(len(hourlyIPs[hourKey]))

		// 尝试获取已有统计，合并数据
		existing, err := h.repo.GetHourlyStats(sourceID, stats.Hour)
		if err == nil && existing != nil {
			existing.TotalRequests += stats.TotalRequests
			existing.TotalBandwidth += stats.TotalBandwidth
			existing.Status2xx += stats.Status2xx
			existing.Status3xx += stats.Status3xx
			existing.Status4xx += stats.Status4xx
			existing.Status5xx += stats.Status5xx
			existing.UniqueVisitors += stats.UniqueVisitors
			totalReqs := existing.TotalRequests
			if totalReqs > 0 {
				existing.AvgResponseTime = (existing.AvgResponseTime*float64(existing.TotalRequests-stats.TotalRequests) +
					stats.AvgResponseTime*float64(stats.TotalRequests)) / float64(totalReqs)
			}
			stats = existing
		}

		if err := h.repo.CreateOrUpdateHourlyStats(stats); err != nil {
			return fmt.Errorf("保存小时统计失败: %w", err)
		}
	}

	return nil
}

// parseNginxLogs 解析Nginx日志
func parseNginxLogs(logContent string, format string) []NginxLogEntry {
	var logs []NginxLogEntry

	scanner := bufio.NewScanner(strings.NewReader(logContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry, err := parseNginxLogLine(line, format)
		if err != nil {
			// 解析失败，跳过这行
			continue
		}

		logs = append(logs, entry)
	}

	return logs
}

// parseNginxLogLine 解析单行Nginx日志
func parseNginxLogLine(line string, format string) (NginxLogEntry, error) {
	var entry NginxLogEntry

	// 支持 combined 和 main 格式
	// combined 格式: $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"
	// main 格式 (带 $request_time): $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $request_time

	// 正则表达式匹配 combined 格式
	combinedPattern := regexp.MustCompile(`^(\S+)\s+-\s+(\S+)\s+\[([^\]]+)\]\s+"([^"]+)"\s+(\d+)\s+(\d+)\s+"([^"]*)"\s+"([^"]*)"(?:\s+(\S+))?`)

	matches := combinedPattern.FindStringSubmatch(line)
	if matches == nil {
		return entry, fmt.Errorf("无法解析日志行")
	}

	entry.RemoteAddr = matches[1]
	entry.RemoteUser = matches[2]
	if entry.RemoteUser == "-" {
		entry.RemoteUser = ""
	}

	// 解析时间
	timeStr := matches[3]
	t, err := time.Parse("02/Jan/2006:15:04:05 -0700", timeStr)
	if err != nil {
		// 尝试不带时区
		t, err = time.Parse("02/Jan/2006:15:04:05", timeStr)
		if err != nil {
			return entry, fmt.Errorf("解析时间失败: %w", err)
		}
	}
	entry.Timestamp = t

	// 解析请求行
	request := matches[4]
	entry.Request = request
	parts := strings.SplitN(request, " ", 3)
	if len(parts) >= 2 {
		entry.Method = parts[0]
		entry.URI = parts[1]
		if len(parts) >= 3 {
			entry.Protocol = parts[2]
		}
	}

	// 状态码
	entry.Status, _ = strconv.Atoi(matches[5])

	// 响应体大小
	entry.BodyBytesSent, _ = strconv.ParseInt(matches[6], 10, 64)

	// Referer
	entry.HTTPReferer = matches[7]
	if entry.HTTPReferer == "-" {
		entry.HTTPReferer = ""
	}

	// User-Agent
	entry.HTTPUserAgent = matches[8]
	if entry.HTTPUserAgent == "-" {
		entry.HTTPUserAgent = ""
	}

	// 请求时间 (可选)
	if len(matches) > 9 && matches[9] != "" && matches[9] != "-" {
		entry.RequestTime, _ = strconv.ParseFloat(matches[9], 64)
	}

	// 从 Host header 或 URI 提取 host
	entry.Host = extractHost(entry.URI, entry.HTTPReferer)

	return entry, nil
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// extractHost 从URI或Referer提取主机名
func extractHost(uri, referer string) string {
	// 尝试从 referer 提取
	if referer != "" && referer != "-" {
		if strings.HasPrefix(referer, "http://") || strings.HasPrefix(referer, "https://") {
			parts := strings.SplitN(strings.TrimPrefix(strings.TrimPrefix(referer, "https://"), "http://"), "/", 2)
			if len(parts) > 0 && parts[0] != "" {
				return strings.Split(parts[0], ":")[0]
			}
		}
	}
	return ""
}

// createSSHClient 创建SSH客户端
func createSSHClient(host *assetbiz.Host, credential *assetbiz.Credential) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	switch credential.Type {
	case "password":
		authMethods = append(authMethods, ssh.Password(credential.Password))
	case "key", "private_key":
		signer, err := ssh.ParsePrivateKey([]byte(credential.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	default:
		return nil, fmt.Errorf("不支持的凭证类型: %s", credential.Type)
	}

	config := &ssh.ClientConfig{
		User:            credential.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host.IP, host.Port)
	return ssh.Dial("tcp", addr, config)
}

// decryptCredential 解密凭证
func decryptCredential(credential *assetbiz.Credential) error {
	if credential.Password != "" {
		decrypted, err := decrypt(credential.Password)
		if err != nil {
			return fmt.Errorf("解密密码失败: %w", err)
		}
		credential.Password = decrypted
	}

	if credential.PrivateKey != "" {
		decrypted, err := decrypt(credential.PrivateKey)
		if err != nil {
			return fmt.Errorf("解密私钥失败: %w", err)
		}
		credential.PrivateKey = decrypted
	}

	return nil
}

// decrypt AES-GCM 解密
func decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
