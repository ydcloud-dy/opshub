package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
)

type probeRequest struct {
	Protocol       string `json:"protocol"`
	Endpoint       string `json:"endpoint"`
	Method         string `json:"method"`
	Headers        string `json:"headers"`
	Body           string `json:"body"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
	ICMPCount      int    `json:"icmpCount"`
	ICMPIntervalMS int    `json:"icmpIntervalMs"`
}

type probeEndpointResult struct {
	Protocol    string     `json:"protocol"`
	Endpoint    string     `json:"endpoint"`
	Success     bool       `json:"success"`
	StatusCode  int        `json:"statusCode"`
	DurationMS  int64      `json:"durationMs"`
	SSLExpireAt *time.Time `json:"sslExpireAt"`
	SSLDaysLeft int        `json:"sslDaysLeft"`
	Message     string     `json:"message"`
	Error       string     `json:"error"`
	CheckedAt   time.Time  `json:"checkedAt"`
}

type probeRunSummary struct {
	Success        bool                  `json:"success"`
	Status         string                `json:"status"`
	DurationMS     int64                 `json:"durationMs"`
	RemoteWriteOK  bool                  `json:"remoteWriteOk"`
	RemoteWriteErr string                `json:"remoteWriteErr"`
	Results        []probeEndpointResult `json:"results"`
}

type remoteWriteSeries struct {
	Labels    map[string]string
	Value     float64
	Timestamp int64
}

func (h *DataSourceHandler) ListProbeTasks(c *gin.Context) {
	var tasks []model.ProbeTask
	query := h.db.Model(&model.ProbeTask{})

	if protocol := strings.TrimSpace(c.Query("protocol")); protocol != "" {
		query = query.Where("protocol = ?", strings.ToLower(protocol))
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", strings.ToLower(status))
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR endpoint LIKE ? OR description LIKE ?", like, like, like)
	}

	if err := query.Order("id DESC").Find(&tasks).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取拨测任务失败", "error": err.Error()})
		return
	}
	h.fillProbeTaskRuntime(tasks)

	c.JSON(200, gin.H{"code": 0, "message": "success", "data": tasks})
}

func (h *DataSourceHandler) GetProbeTask(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var task model.ProbeTask
	if err := h.db.First(&task, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "拨测任务不存在"})
		return
	}
	tasks := []model.ProbeTask{task}
	h.fillProbeTaskRuntime(tasks)
	task = tasks[0]
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": task})
}

func (h *DataSourceHandler) CreateProbeTask(c *gin.Context) {
	var req model.ProbeTask
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeProbeTask(&req)
	if err := h.validateProbeTask(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	now := time.Now()
	if req.Enabled {
		next := now.Add(time.Duration(req.FrequencySeconds) * time.Second)
		req.NextProbeAt = &next
	}
	if req.Status == "" {
		req.Status = "unknown"
	}
	if req.Operator == "" {
		req.Operator = "admin"
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "创建拨测任务失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "创建成功", "data": req})
}

func (h *DataSourceHandler) UpdateProbeTask(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var task model.ProbeTask
	if err := h.db.First(&task, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "拨测任务不存在"})
		return
	}
	var req model.ProbeTask
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeProbeTask(&req)
	if err := h.validateProbeTask(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}

	task.Name = req.Name
	task.Protocol = req.Protocol
	task.Endpoint = req.Endpoint
	task.Method = req.Method
	task.Headers = req.Headers
	task.Body = req.Body
	task.FrequencySeconds = req.FrequencySeconds
	task.TimeoutSeconds = req.TimeoutSeconds
	task.ICMPCount = req.ICMPCount
	task.ICMPIntervalMS = req.ICMPIntervalMS
	task.DataSourceID = req.DataSourceID
	task.WriteRuleEnabled = req.WriteRuleEnabled
	task.Enabled = req.Enabled
	task.Description = req.Description
	task.Operator = req.Operator
	if task.Operator == "" {
		task.Operator = "admin"
	}
	next := time.Now().Add(time.Duration(task.FrequencySeconds) * time.Second)
	if task.Enabled {
		task.NextProbeAt = &next
	} else {
		task.NextProbeAt = nil
	}

	if err := h.db.Save(&task).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "更新拨测任务失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": task})
}

func (h *DataSourceHandler) DeleteProbeTask(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.db.Delete(&model.ProbeTask{}, id).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除拨测任务失败", "error": err.Error()})
		return
	}
	_ = h.db.Where("probe_task_id = ?", id).Delete(&model.ProbeHistory{}).Error
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func (h *DataSourceHandler) RunProbeTask(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var task model.ProbeTask
	if err := h.db.First(&task, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "拨测任务不存在"})
		return
	}
	summary := h.executeProbeTask(c.Request.Context(), &task)
	c.JSON(200, gin.H{"code": 0, "message": "拨测完成", "data": summary})
}

func (h *DataSourceHandler) InstantProbe(c *gin.Context) {
	var req probeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	req.Protocol = normalizeProbeProtocol(req.Protocol)
	req.Method = strings.ToUpper(strings.TrimSpace(req.Method))
	if req.Method == "" {
		req.Method = http.MethodGet
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 5
	}
	if req.ICMPCount <= 0 {
		req.ICMPCount = 3
	}
	if req.ICMPIntervalMS <= 0 {
		req.ICMPIntervalMS = 1000
	}
	if err := validateProbeProtocol(req.Protocol); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if _, err := splitProbeEndpoints(req.Endpoint); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	results := h.executeProbeEndpoints(c.Request.Context(), req.Protocol, req.Endpoint, req.Method, req.Headers, req.Body, req.TimeoutSeconds, req.ICMPCount, req.ICMPIntervalMS)
	success := true
	var duration int64
	for _, result := range results {
		duration += result.DurationMS
		if !result.Success {
			success = false
		}
	}
	status := "normal"
	if !success {
		status = "abnormal"
	}
	c.JSON(200, gin.H{"code": 0, "message": "拨测完成", "data": probeRunSummary{
		Success:    success,
		Status:     status,
		DurationMS: duration,
		Results:    results,
	}})
}

func (h *DataSourceHandler) RunDueProbeTasks(ctx context.Context) {
	now := time.Now()
	var tasks []model.ProbeTask
	if err := h.db.Where("enabled = ? AND (next_probe_at IS NULL OR next_probe_at <= ?)", true, now).Find(&tasks).Error; err != nil {
		return
	}
	for i := range tasks {
		task := tasks[i]
		go h.executeProbeTask(ctx, &task)
	}
}

func (h *DataSourceHandler) executeProbeTask(ctx context.Context, task *model.ProbeTask) probeRunSummary {
	results := h.executeProbeEndpoints(ctx, task.Protocol, task.Endpoint, task.Method, task.Headers, task.Body, task.TimeoutSeconds, task.ICMPCount, task.ICMPIntervalMS)
	now := time.Now()
	success := len(results) > 0
	var duration int64
	var errors []string
	for _, result := range results {
		duration += result.DurationMS
		if !result.Success {
			success = false
		}
		if result.Error != "" {
			errors = append(errors, fmt.Sprintf("%s: %s", result.Endpoint, result.Error))
		}
	}

	status := "normal"
	if !success {
		status = "abnormal"
	}
	remoteWriteOK := false
	remoteWriteErr := ""
	if task.WriteRuleEnabled {
		if err := h.writeProbeResults(ctx, task, results); err != nil {
			remoteWriteErr = err.Error()
		} else {
			remoteWriteOK = true
			task.LastWriteAt = &now
			task.LastWriteError = ""
		}
	}
	if remoteWriteErr != "" {
		task.LastWriteError = remoteWriteErr
	}

	task.Status = status
	task.LastStatus = status
	task.LastProbeAt = &now
	task.LastDurationMS = duration
	task.LastError = strings.Join(errors, "\n")
	next := now.Add(time.Duration(task.FrequencySeconds) * time.Second)
	if task.Enabled {
		task.NextProbeAt = &next
	} else {
		task.NextProbeAt = nil
	}
	_ = h.db.Save(task).Error
	h.saveProbeHistories(task.ID, results, remoteWriteOK, remoteWriteErr)

	return probeRunSummary{
		Success:        success,
		Status:         status,
		DurationMS:     duration,
		RemoteWriteOK:  remoteWriteOK,
		RemoteWriteErr: remoteWriteErr,
		Results:        results,
	}
}

func (h *DataSourceHandler) executeProbeEndpoints(ctx context.Context, protocol, rawEndpoints, method, headers, body string, timeoutSeconds, icmpCount, icmpIntervalMS int) []probeEndpointResult {
	endpoints, err := splitProbeEndpoints(rawEndpoints)
	if err != nil {
		return []probeEndpointResult{{
			Protocol:  protocol,
			Endpoint:  rawEndpoints,
			Success:   false,
			Error:     err.Error(),
			CheckedAt: time.Now(),
		}}
	}
	results := make([]probeEndpointResult, 0, len(endpoints))
	for _, endpoint := range endpoints {
		switch protocol {
		case "http":
			results = append(results, probeHTTP(ctx, endpoint, method, headers, body, timeoutSeconds))
		case "icmp":
			results = append(results, probeICMP(ctx, endpoint, timeoutSeconds, icmpCount, icmpIntervalMS))
		case "tcp":
			results = append(results, probeTCP(ctx, endpoint, timeoutSeconds))
		case "ssl":
			results = append(results, probeSSL(ctx, endpoint, timeoutSeconds))
		default:
			results = append(results, probeEndpointResult{
				Protocol:  protocol,
				Endpoint:  endpoint,
				Success:   false,
				Error:     fmt.Sprintf("不支持的拨测协议: %s", protocol),
				CheckedAt: time.Now(),
			})
		}
	}
	return results
}

func (h *DataSourceHandler) writeProbeResults(ctx context.Context, task *model.ProbeTask, results []probeEndpointResult) error {
	if task.DataSourceID == 0 {
		return fmt.Errorf("请选择开启远程写入的数据源")
	}
	var ds model.DataSource
	if err := h.db.First(&ds, task.DataSourceID).Error; err != nil {
		return fmt.Errorf("数据源不存在")
	}
	if !ds.Enabled {
		return fmt.Errorf("数据源未启用")
	}
	if ds.Type != "prometheus" && ds.Type != "victoriametrics" {
		return fmt.Errorf("拨测结果只能写入 Prometheus 或 VictoriaMetrics 数据源")
	}
	if !ds.RemoteWriteEnabled {
		return fmt.Errorf("数据源未开启远程写入")
	}

	series := make([]remoteWriteSeries, 0, len(results)*4)
	for _, result := range results {
		timestamp := result.CheckedAt.UnixMilli()
		baseLabels := map[string]string{
			"job":       "opshub_probe",
			"task_id":   strconv.FormatUint(uint64(task.ID), 10),
			"task_name": task.Name,
			"protocol":  result.Protocol,
			"endpoint":  result.Endpoint,
		}
		series = append(series,
			withMetricName(baseLabels, "opshub_probe_success", boolToFloat(result.Success), timestamp),
			withMetricName(baseLabels, "opshub_probe_duration_seconds", float64(result.DurationMS)/1000, timestamp),
			withMetricName(baseLabels, "opshub_probe_status_code", float64(result.StatusCode), timestamp),
		)
		if result.SSLExpireAt != nil || result.Protocol == "ssl" {
			series = append(series, withMetricName(baseLabels, "opshub_probe_ssl_days_remaining", float64(result.SSLDaysLeft), timestamp))
		}
	}
	return sendPrometheusRemoteWrite(ctx, &ds, series)
}

func (h *DataSourceHandler) saveProbeHistories(taskID uint, results []probeEndpointResult, remoteWriteOK bool, remoteWriteErr string) {
	for _, result := range results {
		_ = h.db.Create(&model.ProbeHistory{
			ProbeTaskID:    taskID,
			Protocol:       result.Protocol,
			Endpoint:       result.Endpoint,
			Success:        result.Success,
			StatusCode:     result.StatusCode,
			DurationMS:     result.DurationMS,
			SSLExpireAt:    result.SSLExpireAt,
			SSLDaysLeft:    result.SSLDaysLeft,
			Error:          result.Error,
			Message:        result.Message,
			RemoteWriteOK:  remoteWriteOK,
			RemoteWriteErr: remoteWriteErr,
			CheckedAt:      result.CheckedAt,
		}).Error
	}

	var count int64
	h.db.Model(&model.ProbeHistory{}).Where("probe_task_id = ?", taskID).Count(&count)
	if count <= 500 {
		return
	}
	var keepMinID uint
	if err := h.db.Model(&model.ProbeHistory{}).
		Select("id").
		Where("probe_task_id = ?", taskID).
		Order("id DESC").
		Offset(499).
		Limit(1).
		Scan(&keepMinID).Error; err == nil && keepMinID > 0 {
		_ = h.db.Where("probe_task_id = ? AND id < ?", taskID, keepMinID).Delete(&model.ProbeHistory{}).Error
	}
}

func (h *DataSourceHandler) fillProbeTaskRuntime(tasks []model.ProbeTask) {
	if len(tasks) == 0 {
		return
	}
	sourceIDs := make([]uint, 0, len(tasks))
	for _, task := range tasks {
		if task.DataSourceID > 0 {
			sourceIDs = append(sourceIDs, task.DataSourceID)
		}
	}
	if len(sourceIDs) == 0 {
		return
	}
	var sources []model.DataSource
	if err := h.db.Where("id IN ?", sourceIDs).Find(&sources).Error; err != nil {
		return
	}
	sourceMap := make(map[uint]model.DataSource, len(sources))
	for _, source := range sources {
		sourceMap[source.ID] = source
	}
	for i := range tasks {
		if source, ok := sourceMap[tasks[i].DataSourceID]; ok {
			tasks[i].DataSourceName = source.Name
			tasks[i].DataSourceRemoteOK = source.Enabled && source.RemoteWriteEnabled && (source.Type == "prometheus" || source.Type == "victoriametrics")
		}
	}
}

func normalizeProbeTask(task *model.ProbeTask) {
	task.Name = strings.TrimSpace(task.Name)
	task.Protocol = normalizeProbeProtocol(task.Protocol)
	task.Endpoint = strings.TrimSpace(task.Endpoint)
	task.Method = strings.ToUpper(strings.TrimSpace(task.Method))
	task.Headers = strings.TrimSpace(task.Headers)
	task.Body = strings.TrimSpace(task.Body)
	task.Description = strings.TrimSpace(task.Description)
	task.Operator = strings.TrimSpace(task.Operator)
	if task.Method == "" {
		task.Method = http.MethodGet
	}
	if task.FrequencySeconds <= 0 {
		task.FrequencySeconds = 60
	}
	if task.TimeoutSeconds <= 0 {
		task.TimeoutSeconds = 5
	}
	if task.ICMPCount <= 0 {
		task.ICMPCount = 3
	}
	if task.ICMPIntervalMS <= 0 {
		task.ICMPIntervalMS = 1000
	}
	if task.Status == "" {
		task.Status = "unknown"
	}
}

func (h *DataSourceHandler) validateProbeTask(task *model.ProbeTask) error {
	if task.Name == "" {
		return fmt.Errorf("请输入任务名称")
	}
	if err := validateProbeProtocol(task.Protocol); err != nil {
		return err
	}
	if _, err := splitProbeEndpoints(task.Endpoint); err != nil {
		return err
	}
	if task.FrequencySeconds < 10 {
		return fmt.Errorf("执行频率不能小于 10 秒")
	}
	if task.TimeoutSeconds < 1 {
		return fmt.Errorf("超时时间不能小于 1 秒")
	}
	if task.WriteRuleEnabled {
		if task.DataSourceID == 0 {
			return fmt.Errorf("请选择写入数据源")
		}
		var ds model.DataSource
		if err := h.db.First(&ds, task.DataSourceID).Error; err != nil {
			return fmt.Errorf("数据源不存在")
		}
		if ds.Type != "prometheus" && ds.Type != "victoriametrics" {
			return fmt.Errorf("拨测任务只能写入 Prometheus 或 VictoriaMetrics 数据源")
		}
		if !ds.RemoteWriteEnabled {
			return fmt.Errorf("数据源未开启远程写入，无法创建拨测任务")
		}
		if strings.TrimSpace(ds.RemoteWriteURL) == "" {
			return fmt.Errorf("数据源已开启远程写入，但远程写入地址为空")
		}
	}
	return nil
}

func validateProbeProtocol(protocol string) error {
	switch protocol {
	case "http", "icmp", "tcp", "ssl":
		return nil
	default:
		return fmt.Errorf("不支持的拨测协议: %s", protocol)
	}
}

func normalizeProbeProtocol(protocol string) string {
	return strings.ToLower(strings.TrimSpace(protocol))
}

func splitProbeEndpoints(raw string) ([]string, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t'
	})
	endpoints := make([]string, 0, len(parts))
	for _, part := range parts {
		endpoint := strings.TrimSpace(part)
		if endpoint != "" {
			endpoints = append(endpoints, endpoint)
		}
	}
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("请输入拨测端点")
	}
	return endpoints, nil
}

func probeHTTP(ctx context.Context, endpoint, method, headers, body string, timeoutSeconds int) probeEndpointResult {
	start := time.Now()
	result := probeEndpointResult{Protocol: "http", Endpoint: endpoint, CheckedAt: start}
	target := strings.TrimSpace(endpoint)
	if !strings.Contains(target, "://") {
		target = "https://" + target
	}
	if _, err := url.ParseRequestURI(target); err != nil {
		result.DurationMS = time.Since(start).Milliseconds()
		result.Error = "URL 格式不正确"
		return result
	}
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(reqCtx, method, target, strings.NewReader(body))
	if err != nil {
		result.DurationMS = time.Since(start).Milliseconds()
		result.Error = err.Error()
		return result
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if err := applyDataSourceHeaders(req, headers); err != nil {
		result.DurationMS = time.Since(start).Milliseconds()
		result.Error = err.Error()
		return result
	}
	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
	resp, err := client.Do(req)
	result.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 400
	if result.Success {
		result.Message = fmt.Sprintf("HTTP %d，耗时 %dms", resp.StatusCode, result.DurationMS)
	} else {
		result.Error = fmt.Sprintf("HTTP 状态码 %d", resp.StatusCode)
	}
	return result
}

func probeTCP(ctx context.Context, endpoint string, timeoutSeconds int) probeEndpointResult {
	start := time.Now()
	result := probeEndpointResult{Protocol: "tcp", Endpoint: endpoint, CheckedAt: start}
	target, err := normalizeHostPort(endpoint, "")
	if err != nil {
		result.DurationMS = time.Since(start).Milliseconds()
		result.Error = err.Error()
		return result
	}
	dialer := net.Dialer{Timeout: time.Duration(timeoutSeconds) * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", target)
	result.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	_ = conn.Close()
	result.Success = true
	result.Message = fmt.Sprintf("TCP 连接成功，耗时 %dms", result.DurationMS)
	return result
}

func probeSSL(ctx context.Context, endpoint string, timeoutSeconds int) probeEndpointResult {
	start := time.Now()
	result := probeEndpointResult{Protocol: "ssl", Endpoint: endpoint, CheckedAt: start}
	target, err := normalizeHostPort(endpoint, "443")
	if err != nil {
		result.DurationMS = time.Since(start).Milliseconds()
		result.Error = err.Error()
		return result
	}
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		result.DurationMS = time.Since(start).Milliseconds()
		result.Error = err.Error()
		return result
	}
	dialer := &net.Dialer{Timeout: time.Duration(timeoutSeconds) * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", target, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	result.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer conn.Close()
	if len(conn.ConnectionState().PeerCertificates) == 0 {
		result.Error = "未获取到证书"
		return result
	}
	cert := conn.ConnectionState().PeerCertificates[0]
	daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
	result.SSLExpireAt = &cert.NotAfter
	result.SSLDaysLeft = daysLeft
	result.Success = time.Now().Before(cert.NotAfter)
	if result.Success {
		result.Message = fmt.Sprintf("证书有效，剩余 %d 天", daysLeft)
	} else {
		result.Error = "证书已过期"
	}
	return result
}

func probeICMP(ctx context.Context, endpoint string, timeoutSeconds, count, intervalMS int) probeEndpointResult {
	start := time.Now()
	result := probeEndpointResult{Protocol: "icmp", Endpoint: endpoint, CheckedAt: start}
	host := strings.TrimSpace(endpoint)
	if strings.Contains(host, "://") {
		parsed, err := url.Parse(host)
		if err == nil {
			host = parsed.Hostname()
		}
	}
	if host == "" {
		result.Error = "请输入 ICMP 目标"
		return result
	}
	pingCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	args := buildPingArgs(host, timeoutSeconds, count, intervalMS)
	cmd := exec.CommandContext(pingCtx, "ping", args...)
	output, err := cmd.CombinedOutput()
	result.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			text = err.Error()
		}
		result.Error = text
		return result
	}
	result.Success = true
	result.Message = fmt.Sprintf("ICMP 探测成功，耗时 %dms", result.DurationMS)
	return result
}

func buildPingArgs(host string, timeoutSeconds, count, intervalMS int) []string {
	if count <= 0 {
		count = 1
	}
	interval := fmt.Sprintf("%.3f", float64(intervalMS)/1000)
	if runtime.GOOS == "darwin" {
		return []string{"-c", strconv.Itoa(count), "-i", interval, "-W", strconv.Itoa(timeoutSeconds * 1000), host}
	}
	return []string{"-c", strconv.Itoa(count), "-i", interval, "-W", strconv.Itoa(timeoutSeconds), host}
}

func normalizeHostPort(endpoint, defaultPort string) (string, error) {
	target := strings.TrimSpace(endpoint)
	if strings.Contains(target, "://") {
		parsed, err := url.Parse(target)
		if err != nil {
			return "", fmt.Errorf("端点格式不正确")
		}
		target = parsed.Host
	}
	if target == "" {
		return "", fmt.Errorf("请输入端点")
	}
	if _, _, err := net.SplitHostPort(target); err == nil {
		return target, nil
	}
	if defaultPort == "" {
		return "", fmt.Errorf("TCP 端点必须包含端口，如 127.0.0.1:80")
	}
	return net.JoinHostPort(strings.Trim(target, "[]"), defaultPort), nil
}

func withMetricName(base map[string]string, metric string, value float64, timestamp int64) remoteWriteSeries {
	labels := make(map[string]string, len(base)+1)
	for key, item := range base {
		labels[key] = item
	}
	labels["__name__"] = metric
	return remoteWriteSeries{Labels: labels, Value: value, Timestamp: timestamp}
}

func boolToFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func sendPrometheusRemoteWrite(ctx context.Context, ds *model.DataSource, series []remoteWriteSeries) error {
	if len(series) == 0 {
		return nil
	}
	writeURL := strings.TrimSpace(ds.RemoteWriteURL)
	if writeURL == "" {
		return fmt.Errorf("启用远程写入后必须填写远程写入地址")
	}
	if _, err := url.ParseRequestURI(writeURL); err != nil {
		return fmt.Errorf("远程写入地址格式不正确")
	}
	payload := snappyEncodeBlock(marshalRemoteWriteRequest(series))
	timeout := time.Duration(ds.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, writeURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	applyRemoteWriteAuth(req, ds)
	if err := applyDataSourceHeaders(req, ds.RemoteWriteHeaders); err != nil {
		return err
	}
	client := newMonitorHTTPClient(timeout, ds.RemoteWriteSkipTLSVerify || ds.SkipTLSVerify)
	resp, err := client.Do(req)
	if err != nil {
		return formatDataSourceRequestError(ds, writeURL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return formatRemoteWriteStatusError(resp.StatusCode, string(body))
	}
	return nil
}

func formatRemoteWriteStatusError(statusCode int, body string) error {
	text := strings.TrimSpace(body)
	if strings.Contains(text, "remote write receiver needs to be enabled") {
		return fmt.Errorf("远程写入接收端未开启：Prometheus 需要启用 remote-write-receiver 后才能接收拨测指标")
	}
	if text == "" {
		return fmt.Errorf("remote write 返回状态码 %d", statusCode)
	}
	return fmt.Errorf("remote write 返回状态码 %d: %s", statusCode, text)
}

func applyRemoteWriteAuth(req *http.Request, ds *model.DataSource) {
	switch ds.RemoteWriteAuthType {
	case "basic":
		req.SetBasicAuth(ds.RemoteWriteUsername, ds.RemoteWritePassword)
	case "bearer":
		if token := strings.TrimSpace(ds.RemoteWriteToken); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
}

func marshalRemoteWriteRequest(series []remoteWriteSeries) []byte {
	var dst []byte
	for _, item := range series {
		dst = appendBytesField(dst, 1, marshalTimeSeries(item))
	}
	return dst
}

func marshalTimeSeries(series remoteWriteSeries) []byte {
	var dst []byte
	keys := make([]string, 0, len(series.Labels))
	for key := range series.Labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		dst = appendBytesField(dst, 1, marshalLabel(key, series.Labels[key]))
	}
	dst = appendBytesField(dst, 2, marshalSample(series.Value, series.Timestamp))
	return dst
}

func marshalLabel(name, value string) []byte {
	var dst []byte
	dst = appendStringField(dst, 1, name)
	dst = appendStringField(dst, 2, value)
	return dst
}

func marshalSample(value float64, timestamp int64) []byte {
	var dst []byte
	dst = append(dst, byte(1<<3|1))
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(value))
	dst = append(dst, buf[:]...)
	dst = appendVarintField(dst, 2, uint64(timestamp))
	return dst
}

func appendStringField(dst []byte, fieldNumber int, value string) []byte {
	return appendBytesField(dst, fieldNumber, []byte(value))
}

func appendBytesField(dst []byte, fieldNumber int, value []byte) []byte {
	dst = appendVarint(dst, uint64(fieldNumber<<3|2))
	dst = appendVarint(dst, uint64(len(value)))
	dst = append(dst, value...)
	return dst
}

func appendVarintField(dst []byte, fieldNumber int, value uint64) []byte {
	dst = appendVarint(dst, uint64(fieldNumber<<3))
	return appendVarint(dst, value)
}

func appendVarint(dst []byte, value uint64) []byte {
	for value >= 0x80 {
		dst = append(dst, byte(value)|0x80)
		value >>= 7
	}
	return append(dst, byte(value))
}

func snappyEncodeBlock(src []byte) []byte {
	dst := appendVarint(nil, uint64(len(src)))
	for len(src) > 0 {
		chunkSize := len(src)
		if chunkSize > 1<<20 {
			chunkSize = 1 << 20
		}
		dst = appendSnappyLiteral(dst, src[:chunkSize])
		src = src[chunkSize:]
	}
	return dst
}

func appendSnappyLiteral(dst []byte, literal []byte) []byte {
	length := len(literal)
	if length == 0 {
		return dst
	}
	n := length - 1
	if n < 60 {
		dst = append(dst, byte(n<<2))
	} else {
		var buf [4]byte
		size := 0
		for n > 0 {
			buf[size] = byte(n)
			n >>= 8
			size++
		}
		dst = append(dst, byte((59+size)<<2))
		dst = append(dst, buf[:size]...)
	}
	return append(dst, literal...)
}
