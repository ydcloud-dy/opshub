package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/conf"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

type DataSourceHandler struct {
	db *gorm.DB
}

type monitorSchedulerRuntimeStatus struct {
	StartedAt            atomic.Value
	InstanceID           atomic.Value
	LeaderID             atomic.Value
	IsLeader             atomic.Bool
	LeaderUpdatedAt      atomic.Value
	LeaderError          atomic.Value
	SchedulerMode        atomic.Value
	LastTickAt           atomic.Value
	LastFinishedAt       atomic.Value
	LastDurationMS       atomic.Int64
	LastRuleTotal        atomic.Int64
	LastRuleDue          atomic.Int64
	LastRuleClaimed      atomic.Int64
	LastRuleEvaluated    atomic.Int64
	LastRuleSkipped      atomic.Int64
	LastRuleFailed       atomic.Int64
	LastError            atomic.Value
	LastProbeTickAt      atomic.Value
	LastProbeFinishedAt  atomic.Value
	LastProbeDurationMS  atomic.Int64
	LastProbeTaskTotal   atomic.Int64
	LastProbeTaskStarted atomic.Int64
	LastProbeError       atomic.Value
}

var monitorSchedulerStatus monitorSchedulerRuntimeStatus

var (
	noticeLabelTemplatePattern      = regexp.MustCompile(`\$\{labels\.([A-Za-z0-9_.:-]+)\}|{{\s*\$?labels\.([A-Za-z0-9_.:-]+)\s*}}`)
	noticeAnnotationTemplatePattern = regexp.MustCompile(`\$\{annotations\.([A-Za-z0-9_.:-]+)\}|{{\s*\$?annotations\.([A-Za-z0-9_.:-]+)\s*}}`)
	alertLabelTemplatePattern       = regexp.MustCompile(`\$\{labels\.([A-Za-z0-9_.:-]+)\}|{{\s*\$?labels\.([A-Za-z0-9_.:-]+)\s*}}`)
	alertAnnotationTemplatePattern  = regexp.MustCompile(`\$\{annotations\.([A-Za-z0-9_.:-]+)\}|{{\s*\$?annotations\.([A-Za-z0-9_.:-]+)\s*}}`)
	alertTemplateVariableNameRegexp = regexp.MustCompile(`[A-Za-z0-9_.:-]+`)
	lokiRangeSelectorRegexp         = regexp.MustCompile(`(?is)\s*\[((?:\d+(?:\.\d+)?(?:ms|s|m|h|d|w|y))+)\]\s*$`)
	lokiAnyRangeSelectorRegexp      = regexp.MustCompile(`(?is)\[((?:\d+(?:\.\d+)?(?:ms|s|m|h|d|w|y))+)\]`)
	elasticNoMappingSortRegexp      = regexp.MustCompile(`No mapping found for \[([^\]]+)\] in order to sort`)
)

const (
	maxMatchedLogLinesPerEvent       = 5
	maxMatchedLogLineChars           = 700
	maxMatchedLogsTextChars          = 2400
	maxLokiMatchedLogLookbackSeconds = 7 * 24 * 60 * 60
	alertRuleEvaluationConcurrency   = 8
	alertRuleEvaluationBatchSize     = 64
	alertRuleEvaluationTimeout       = 45 * time.Second
)

func NewDataSourceHandler(db *gorm.DB) *DataSourceHandler {
	return &DataSourceHandler{db: db}
}

type dataSourceQueryRequest struct {
	Query     string `json:"query"`
	QueryMode string `json:"queryMode"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Step      string `json:"step"`
	Index     string `json:"index"`
	Limit     int    `json:"limit"`
}

type alertCallbackQueryConfig struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Query        string `json:"query"`
	Value        string `json:"value"`
	DataSourceID uint   `json:"dataSourceId"`
	QueryMode    string `json:"queryMode"`
	Index        string `json:"index"`
	Step         string `json:"step"`
	RangeSeconds int    `json:"rangeSeconds"`
	Enabled      *bool  `json:"enabled"`
}

type notificationCallbackContext struct {
	Items         []notificationCallbackItem
	Images        []notificationCallbackImage
	ImageWarnings []string
	Summary       string
	DetailText    string
}

type notificationCallbackItem struct {
	Key            string
	Name           string
	Query          string
	RenderedQuery  string
	QueryMode      string
	DataSourceID   uint
	DataSourceName string
	DataSourceType string
	ValueText      string
	Status         string
	Error          string
	Result         interface{}
	Image          *notificationCallbackImage
}

type notificationCallbackImage struct {
	Title    string
	Query    string
	ImageKey string
	PNG      []byte
}

func (h *DataSourceHandler) ListDataSources(c *gin.Context) {
	var dataSources []model.DataSource
	query := h.db.Model(&model.DataSource{})
	if c.Query("includeInternal") != "true" {
		query = query.Where("type <> ?", opsHubLogDataSourceType)
	}

	if dsType := strings.TrimSpace(c.Query("type")); dsType != "" {
		query = query.Where("type = ?", dsType)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR url LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Order("id DESC").Find(&dataSources).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取数据源列表失败", "error": err.Error()})
		return
	}

	if isReadonlyContext(c) {
		maskDataSourceSecrets(dataSources)
	}

	c.JSON(200, gin.H{"code": 0, "message": "success", "data": dataSources})
}

func (h *DataSourceHandler) GetDataSource(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var dataSource model.DataSource
	if err := h.db.First(&dataSource, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "数据源不存在"})
		return
	}

	if isReadonlyContext(c) {
		maskDataSourceSecret(&dataSource)
	}

	c.JSON(200, gin.H{"code": 0, "message": "success", "data": dataSource})
}

func isReadonlyContext(c *gin.Context) bool {
	value, ok := c.Get("readonly_user")
	if !ok {
		return false
	}
	readonly, ok := value.(bool)
	return ok && readonly
}

func maskDataSourceSecrets(dataSources []model.DataSource) {
	for i := range dataSources {
		maskDataSourceSecret(&dataSources[i])
	}
}

func maskDataSourceSecret(dataSource *model.DataSource) {
	dataSource.Password = maskSecret(dataSource.Password)
	dataSource.Token = maskSecret(dataSource.Token)
	dataSource.Headers = maskSecret(dataSource.Headers)
	dataSource.RemoteWritePassword = maskSecret(dataSource.RemoteWritePassword)
	dataSource.RemoteWriteToken = maskSecret(dataSource.RemoteWriteToken)
	dataSource.RemoteWriteHeaders = maskSecret(dataSource.RemoteWriteHeaders)
}

func maskSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "******"
}

func (h *DataSourceHandler) CreateDataSource(c *gin.Context) {
	var req model.DataSource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}

	normalizeDataSource(&req)
	if req.Type == opsHubLogDataSourceType {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "OpsHub 内置日志数据源由日志中心自动维护"})
		return
	}
	if err := validateDataSource(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "创建数据源失败", "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "创建成功", "data": req})
}

func (h *DataSourceHandler) UpdateDataSource(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var dataSource model.DataSource
	if err := h.db.First(&dataSource, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "数据源不存在"})
		return
	}
	if dataSource.Type == opsHubLogDataSourceType {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "OpsHub 内置日志数据源请在日志库中管理"})
		return
	}

	var req model.DataSource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}

	normalizeDataSource(&req)
	if err := validateDataSource(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}

	dataSource.Name = req.Name
	dataSource.Type = req.Type
	dataSource.URL = req.URL
	dataSource.AuthType = req.AuthType
	dataSource.Username = req.Username
	dataSource.Password = req.Password
	dataSource.Token = req.Token
	dataSource.Headers = req.Headers
	dataSource.Timeout = req.Timeout
	dataSource.SkipTLSVerify = req.SkipTLSVerify
	dataSource.Enabled = req.Enabled
	dataSource.RemoteWriteEnabled = req.RemoteWriteEnabled
	dataSource.RemoteWriteURL = req.RemoteWriteURL
	dataSource.RemoteWriteAuthType = req.RemoteWriteAuthType
	dataSource.RemoteWriteUsername = req.RemoteWriteUsername
	dataSource.RemoteWritePassword = req.RemoteWritePassword
	dataSource.RemoteWriteToken = req.RemoteWriteToken
	dataSource.RemoteWriteHeaders = req.RemoteWriteHeaders
	dataSource.RemoteWriteSkipTLSVerify = req.RemoteWriteSkipTLSVerify
	dataSource.Description = req.Description

	if err := h.db.Save(&dataSource).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "更新数据源失败", "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": dataSource})
}

func (h *DataSourceHandler) DeleteDataSource(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var datasource model.DataSource
	if err := h.db.First(&datasource, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "数据源不存在"})
		return
	}
	if datasource.Type == opsHubLogDataSourceType {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "OpsHub 内置日志数据源不能在监控中心删除"})
		return
	}
	if err := h.db.Delete(&datasource).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除数据源失败", "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func (h *DataSourceHandler) TestDataSource(c *gin.Context) {
	dataSource, ok := h.loadDataSource(c)
	if !ok {
		return
	}

	start := time.Now()
	err := h.testDataSource(c.Request.Context(), dataSource)
	now := time.Now()
	dataSource.LastTestAt = &now

	if err != nil {
		dataSource.Status = "abnormal"
		dataSource.LastError = err.Error()
		_ = h.db.Save(dataSource).Error
		c.JSON(200, gin.H{
			"code":    0,
			"message": "测试完成",
			"data": gin.H{
				"ok":       false,
				"duration": int(time.Since(start).Milliseconds()),
				"error":    err.Error(),
			},
		})
		return
	}

	dataSource.Status = "normal"
	dataSource.LastError = ""
	_ = h.db.Save(dataSource).Error
	c.JSON(200, gin.H{
		"code":    0,
		"message": "测试成功",
		"data": gin.H{
			"ok":       true,
			"duration": int(time.Since(start).Milliseconds()),
		},
	})
}

func (h *DataSourceHandler) TestDataSourceRemoteWrite(c *gin.Context) {
	dataSource, ok := h.loadDataSource(c)
	if !ok {
		return
	}

	h.respondRemoteWriteTest(c, dataSource, true)
}

func (h *DataSourceHandler) TestRemoteWriteConfig(c *gin.Context) {
	var req model.DataSource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeDataSource(&req)

	h.respondRemoteWriteTest(c, &req, false)
}

func (h *DataSourceHandler) respondRemoteWriteTest(c *gin.Context, dataSource *model.DataSource, persist bool) {
	start := time.Now()
	err := h.testRemoteWrite(c.Request.Context(), dataSource)
	now := time.Now()

	if persist {
		dataSource.LastTestAt = &now
		if err != nil {
			dataSource.Status = "abnormal"
			dataSource.LastError = err.Error()
		} else {
			dataSource.Status = "normal"
			dataSource.LastError = ""
		}
		_ = h.db.Save(dataSource).Error
	}

	if err != nil {
		c.JSON(200, gin.H{
			"code":    0,
			"message": "远程写入测试完成",
			"data": gin.H{
				"ok":       false,
				"duration": int(time.Since(start).Milliseconds()),
				"error":    err.Error(),
			},
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "远程写入测试成功",
		"data": gin.H{
			"ok":       true,
			"duration": int(time.Since(start).Milliseconds()),
		},
	})
}

func (h *DataSourceHandler) QueryDataSource(c *gin.Context) {
	dataSource, ok := h.loadDataSource(c)
	if !ok {
		return
	}

	var req dataSourceQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}

	start := time.Now()
	result, statusCode, err := h.queryDataSource(c.Request.Context(), dataSource, req)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    0,
			"message": "查询失败",
			"data": gin.H{
				"statusCode": statusCode,
				"duration":   int(time.Since(start).Milliseconds()),
				"ok":         false,
				"message":    alertRuleErrorMessage(err),
				"error":      err.Error(),
			},
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "查询成功",
		"data": gin.H{
			"statusCode": statusCode,
			"duration":   int(time.Since(start).Milliseconds()),
			"ok":         true,
			"result":     result,
		},
	})
}

func (h *DataSourceHandler) GetAlertEventCallbackQueries(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var event model.AlertEvent
	if err := h.db.First(&event, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "告警事件不存在"})
		return
	}

	var rule model.AlertRule
	if err := h.db.First(&rule, event.RuleID).Error; err != nil {
		c.JSON(200, gin.H{"code": 0, "message": "success", "data": []gin.H{}})
		return
	}

	callbacks := parseAlertCallbackQueries(rule.CallbackQueries)
	results := make([]gin.H, 0, len(callbacks))
	for _, callback := range callbacks {
		if callback.Enabled != nil && !*callback.Enabled {
			continue
		}
		queryText := strings.TrimSpace(firstNonEmpty(callback.Query, callback.Value))
		name := strings.TrimSpace(firstNonEmpty(callback.Name, callback.Key))
		if name == "" {
			name = "关联查询"
		}
		item := gin.H{
			"key":          callback.Key,
			"name":         name,
			"query":        queryText,
			"queryMode":    callback.QueryMode,
			"dataSourceId": callback.DataSourceID,
		}
		if queryText == "" {
			item["error"] = "查询语句为空"
			results = append(results, item)
			continue
		}

		dsID := callback.DataSourceID
		if dsID == 0 {
			dsID = rule.DataSourceID
		}
		var ds model.DataSource
		if err := h.db.First(&ds, dsID).Error; err != nil {
			item["error"] = "数据源不存在"
			results = append(results, item)
			continue
		}
		req := buildCallbackQueryRequest(callback, queryText, rule, event)
		item["renderedQuery"] = req.Query
		start := time.Now()
		rawResult, statusCode, err := h.queryDataSource(c.Request.Context(), &ds, req)
		item["dataSourceId"] = ds.ID
		item["dataSourceName"] = ds.Name
		item["dataSourceType"] = ds.Type
		item["queryMode"] = req.QueryMode
		item["statusCode"] = statusCode
		item["duration"] = int(time.Since(start).Milliseconds())
		if err != nil {
			item["error"] = err.Error()
		} else {
			item["result"] = rawResult
		}
		results = append(results, item)
	}

	c.JSON(200, gin.H{"code": 0, "message": "success", "data": results})
}

func (h *DataSourceHandler) loadDataSource(c *gin.Context) (*model.DataSource, bool) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return nil, false
	}

	var dataSource model.DataSource
	if err := h.db.First(&dataSource, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "数据源不存在"})
		return nil, false
	}

	return &dataSource, true
}

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		c.JSON(400, gin.H{"code": 400, "message": "无效的ID"})
		return 0, false
	}
	return uint(id), true
}

func normalizeDataSource(ds *model.DataSource) {
	ds.Name = strings.TrimSpace(ds.Name)
	ds.Type = strings.ToLower(strings.TrimSpace(ds.Type))
	ds.URL = strings.TrimRight(strings.TrimSpace(ds.URL), "/")
	ds.AuthType = strings.ToLower(strings.TrimSpace(ds.AuthType))
	ds.Username = strings.TrimSpace(ds.Username)
	ds.Headers = strings.TrimSpace(ds.Headers)
	ds.RemoteWriteURL = strings.TrimRight(strings.TrimSpace(ds.RemoteWriteURL), "/")
	ds.RemoteWriteAuthType = strings.ToLower(strings.TrimSpace(ds.RemoteWriteAuthType))
	ds.RemoteWriteUsername = strings.TrimSpace(ds.RemoteWriteUsername)
	ds.RemoteWriteHeaders = strings.TrimSpace(ds.RemoteWriteHeaders)
	if ds.AuthType == "" {
		ds.AuthType = "none"
	}
	if ds.RemoteWriteAuthType == "" {
		ds.RemoteWriteAuthType = "none"
	}
	if ds.Timeout <= 0 {
		ds.Timeout = 10
	}
	if ds.Status == "" {
		ds.Status = "unknown"
	}
}

func validateDataSource(ds *model.DataSource) error {
	if ds.Name == "" {
		return fmt.Errorf("请输入数据源名称")
	}
	if ds.URL == "" {
		return fmt.Errorf("请输入数据源地址")
	}
	if _, err := url.ParseRequestURI(ds.URL); err != nil {
		return fmt.Errorf("数据源地址格式不正确")
	}
	switch ds.Type {
	case "prometheus", "victoriametrics", "loki", "elasticsearch", "opensearch":
	default:
		return fmt.Errorf("不支持的数据源类型: %s", ds.Type)
	}
	switch ds.AuthType {
	case "none", "basic", "bearer":
	default:
		return fmt.Errorf("不支持的认证方式: %s", ds.AuthType)
	}
	if ds.AuthType == "basic" && (ds.Username == "" || ds.Password == "") {
		return fmt.Errorf("Basic 认证必须填写用户名和密码")
	}
	if ds.AuthType == "bearer" && strings.TrimSpace(ds.Token) == "" {
		return fmt.Errorf("Bearer 认证必须填写 Token")
	}
	switch ds.RemoteWriteAuthType {
	case "none", "basic", "bearer":
	default:
		return fmt.Errorf("不支持的远程写入认证方式: %s", ds.RemoteWriteAuthType)
	}
	if ds.RemoteWriteEnabled {
		return validateRemoteWriteConfig(ds)
	}
	return nil
}

func validateRemoteWriteConfig(ds *model.DataSource) error {
	if ds.Type != "prometheus" && ds.Type != "victoriametrics" {
		return fmt.Errorf("只有 Prometheus 或 VictoriaMetrics 数据源支持远程写入")
	}
	if !ds.RemoteWriteEnabled {
		return fmt.Errorf("请先启用远程写入")
	}
	if ds.RemoteWriteURL == "" {
		return fmt.Errorf("启用远程写入后必须填写远程写入地址")
	}
	if _, err := url.ParseRequestURI(ds.RemoteWriteURL); err != nil {
		return fmt.Errorf("远程写入地址格式不正确")
	}
	if ds.RemoteWriteAuthType == "basic" && (ds.RemoteWriteUsername == "" || ds.RemoteWritePassword == "") {
		return fmt.Errorf("远程写入 Basic 认证必须填写用户名和密码")
	}
	if ds.RemoteWriteAuthType == "bearer" && strings.TrimSpace(ds.RemoteWriteToken) == "" {
		return fmt.Errorf("远程写入 Bearer 认证必须填写 Token")
	}
	if ds.RemoteWriteHeaders != "" {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(ds.RemoteWriteHeaders), &parsed); err != nil {
			return fmt.Errorf("远程写入请求头必须是 JSON 对象")
		}
	}
	return nil
}

func (h *DataSourceHandler) testDataSource(ctx context.Context, ds *model.DataSource) error {
	switch ds.Type {
	case "prometheus", "victoriametrics":
		_, _, err := h.doDataSourceRequest(ctx, ds, http.MethodGet, "/api/v1/query", map[string]string{"query": "vector(1)"}, nil)
		if err != nil {
			return h.enrichPrometheusEndpointError(ctx, ds, err)
		}
		return nil
	case "loki":
		_, _, err := h.doDataSourceRequest(ctx, ds, http.MethodGet, "/ready", nil, nil)
		if err == nil {
			return nil
		}
		_, _, err = h.doDataSourceRequest(ctx, ds, http.MethodGet, "/loki/api/v1/labels", nil, nil)
		return err
	case "elasticsearch", "opensearch":
		_, _, err := h.doDataSourceRequest(ctx, ds, http.MethodGet, "/_cluster/health", nil, nil)
		return err
	case opsHubLogDataSourceType:
		_, _, err := h.queryOpsHubLogs(ctx, ds, dataSourceQueryRequest{Query: `{"version":1,"windowSeconds":60,"keyword":"*","aggregation":"count","sampleLimit":1}`})
		return err
	default:
		return fmt.Errorf("不支持的数据源类型: %s", ds.Type)
	}
}

func (h *DataSourceHandler) testRemoteWrite(ctx context.Context, ds *model.DataSource) error {
	if err := validateRemoteWriteConfig(ds); err != nil {
		return err
	}
	now := time.Now()
	labels := map[string]string{
		"job":             "opshub_remote_write_test",
		"datasource_type": ds.Type,
	}
	if ds.ID > 0 {
		labels["datasource_id"] = strconv.FormatUint(uint64(ds.ID), 10)
	}
	if ds.Name != "" {
		labels["datasource_name"] = ds.Name
	}
	series := []remoteWriteSeries{
		withMetricName(labels, "opshub_remote_write_test", 1, now.UnixMilli()),
	}
	return sendPrometheusRemoteWrite(ctx, ds, series)
}

func (h *DataSourceHandler) enrichPrometheusEndpointError(ctx context.Context, ds *model.DataSource, original error) error {
	if original == nil {
		return nil
	}
	errText := original.Error()
	if !strings.Contains(errText, "404") && !strings.Contains(strings.ToLower(errText), "not found") {
		return original
	}
	title, err := h.readDataSourceRootTitle(ctx, ds)
	if err != nil || title == "" {
		return original
	}
	if strings.Contains(strings.ToLower(title), "argo cd") {
		return fmt.Errorf("当前地址返回的是 Argo CD 页面，不是 %s API 地址；请填写 Prometheus 或 VictoriaMetrics 的访问地址", dataSourceDisplayName(ds.Type))
	}
	return fmt.Errorf("当前地址首页标题为「%s」，但 /api/v1/query 不存在；请确认它是否为 %s API 地址", title, dataSourceDisplayName(ds.Type))
}

func (h *DataSourceHandler) readDataSourceRootTitle(ctx context.Context, ds *model.DataSource) (string, error) {
	timeout := time.Duration(ds.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, strings.TrimRight(ds.URL, "/")+"/", nil)
	if err != nil {
		return "", err
	}
	applyDataSourceAuth(req, ds)
	if err := applyDataSourceHeaders(req, ds.Headers); err != nil {
		return "", err
	}
	resp, err := newMonitorHTTPClient(timeout, ds.SkipTLSVerify).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return "", err
	}
	text := string(body)
	lower := strings.ToLower(text)
	start := strings.Index(lower, "<title>")
	end := strings.Index(lower, "</title>")
	if start < 0 || end <= start {
		return "", nil
	}
	return strings.TrimSpace(text[start+len("<title>") : end]), nil
}

func dataSourceDisplayName(dsType string) string {
	switch dsType {
	case "prometheus":
		return "Prometheus"
	case "victoriametrics":
		return "VictoriaMetrics"
	case "loki":
		return "Loki"
	case "elasticsearch":
		return "Elasticsearch"
	case "opensearch":
		return "OpenSearch"
	case opsHubLogDataSourceType:
		return "OpsHub 内置日志"
	default:
		return dsType
	}
}

func (h *DataSourceHandler) queryDataSource(ctx context.Context, ds *model.DataSource, req dataSourceQueryRequest) (interface{}, int, error) {
	queryMode := strings.ToLower(strings.TrimSpace(req.QueryMode))
	if queryMode == "" {
		queryMode = "instant"
	}

	switch ds.Type {
	case "prometheus", "victoriametrics":
		if strings.TrimSpace(req.Query) == "" {
			return nil, 0, fmt.Errorf("请输入 PromQL 查询语句")
		}
		path := "/api/v1/query"
		params := map[string]string{"query": req.Query}
		if queryMode == "range" {
			path = "/api/v1/query_range"
			params["start"] = req.Start
			params["end"] = req.End
			params["step"] = req.Step
		}
		return h.doDataSourceRequest(ctx, ds, http.MethodGet, path, params, nil)
	case "loki":
		if strings.TrimSpace(req.Query) == "" {
			return nil, 0, fmt.Errorf("请输入 LogQL 查询语句")
		}
		path := "/loki/api/v1/query"
		params := map[string]string{"query": req.Query}
		if req.Limit > 0 {
			params["limit"] = strconv.Itoa(req.Limit)
		}
		if queryMode == "range" {
			path = "/loki/api/v1/query_range"
			params["start"] = req.Start
			params["end"] = req.End
			params["step"] = req.Step
			params["direction"] = "backward"
		}
		return h.doDataSourceRequest(ctx, ds, http.MethodGet, path, params, nil)
	case "elasticsearch", "opensearch":
		body := strings.TrimSpace(req.Query)
		if body == "" {
			body = `{"size":0}`
		}
		path := "/_search"
		if index := strings.Trim(req.Index, "/ "); index != "" {
			path = "/" + index + "/_search"
		}
		return h.doDataSourceRequest(ctx, ds, http.MethodPost, path, nil, []byte(body))
	case opsHubLogDataSourceType:
		return h.queryOpsHubLogs(ctx, ds, req)
	default:
		return nil, 0, fmt.Errorf("不支持的数据源类型: %s", ds.Type)
	}
}

func parseAlertCallbackQueries(raw string) []alertCallbackQueryConfig {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []alertCallbackQueryConfig{}
	}
	var items []alertCallbackQueryConfig
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return items
	}
	var object map[string]string
	if err := json.Unmarshal([]byte(raw), &object); err == nil {
		items = make([]alertCallbackQueryConfig, 0, len(object))
		for key, value := range object {
			items = append(items, alertCallbackQueryConfig{Key: key, Name: key, Query: value})
		}
		return items
	}
	return []alertCallbackQueryConfig{}
}

func buildCallbackQueryRequest(callback alertCallbackQueryConfig, queryText string, rule model.AlertRule, event model.AlertEvent) dataSourceQueryRequest {
	queryMode := strings.ToLower(strings.TrimSpace(callback.QueryMode))
	if queryMode == "" {
		queryMode = "instant"
	}
	req := dataSourceQueryRequest{
		Query:     renderAlertCallbackQuery(queryText, rule, event),
		QueryMode: queryMode,
		Index:     firstNonEmpty(callback.Index, rule.Index),
		Step:      firstNonEmpty(callback.Step, "30s"),
	}
	if queryMode == "range" || callback.RangeSeconds > 0 {
		if callback.RangeSeconds <= 0 {
			callback.RangeSeconds = 1800
		}
		end := event.LastEvalAt
		if end.IsZero() {
			end = time.Now()
		}
		start := end.Add(-time.Duration(callback.RangeSeconds) * time.Second)
		req.QueryMode = "range"
		req.Start = start.Format(time.RFC3339)
		req.End = end.Format(time.RFC3339)
	}
	return req
}

func (h *DataSourceHandler) doDataSourceRequest(ctx context.Context, ds *model.DataSource, method, path string, params map[string]string, body []byte) (interface{}, int, error) {
	timeout := time.Duration(ds.Timeout) * time.Second
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	targetURL := ds.URL + path
	req, err := http.NewRequestWithContext(reqCtx, method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}

	query := req.URL.Query()
	for key, value := range params {
		if strings.TrimSpace(value) != "" {
			query.Set(key, value)
		}
	}
	req.URL.RawQuery = query.Encode()

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	applyDataSourceAuth(req, ds)
	if err := applyDataSourceHeaders(req, ds.Headers); err != nil {
		return nil, 0, err
	}

	client := newMonitorHTTPClient(timeout, ds.SkipTLSVerify)
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, formatDataSourceRequestError(ds, targetURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, formatDataSourceResponseError(resp.StatusCode, respBody)
	}

	var decoded interface{}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return string(respBody), resp.StatusCode, nil
	}
	if err := dataSourceAPIError(decoded); err != nil {
		return decoded, resp.StatusCode, err
	}
	return decoded, resp.StatusCode, nil
}

func formatDataSourceResponseError(statusCode int, body []byte) error {
	message := strings.TrimSpace(string(body))
	var parsed map[string]interface{}
	if json.Unmarshal(body, &parsed) == nil {
		message = dataSourceErrorText(parsed)
	}
	if strings.TrimSpace(message) == "" {
		message = http.StatusText(statusCode)
	}
	message = friendlyDataSourceError(message)
	return fmt.Errorf("数据源查询失败（HTTP %d）：%s", statusCode, clipPlainText(message, 800))
}

func dataSourceAPIError(decoded interface{}) error {
	root, ok := decoded.(map[string]interface{})
	if !ok {
		return nil
	}
	status := strings.TrimSpace(fmt.Sprint(root["status"]))
	if !strings.EqualFold(status, "error") {
		return nil
	}
	message := dataSourceErrorText(root)
	if strings.TrimSpace(message) == "" {
		message = "数据源返回查询错误"
	}
	message = friendlyDataSourceError(message)
	return fmt.Errorf("%s", clipPlainText(message, 800))
}

func friendlyDataSourceError(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	if matches := elasticNoMappingSortRegexp.FindStringSubmatch(message); len(matches) == 2 {
		return fmt.Sprintf("当前索引没有字段 %q，但查询 DSL 正在按它排序。请删除 sort，或把 sort/range 中的字段改成索引真实的时间字段；如果只是查看日志样例，可以使用 match_all 查询。", matches[1])
	}
	return message
}

func alertRuleErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "duplicate time series") || strings.Contains(lower, "many-to-many matching"):
		return "查询语句存在重复时间序列，数据源无法完成标签匹配，请检查 PromQL 的 on()/group_left()/聚合维度"
	case strings.Contains(lower, "cannot execute") || strings.Contains(lower, "cannot evaluate"):
		return "数据源无法执行当前查询语句，请检查查询条件和指标标签是否匹配"
	case strings.Contains(lower, "parse error") || strings.Contains(lower, "bad_data") || strings.Contains(lower, "invalid parameter"):
		return "查询语句解析失败，请检查 PromQL / LogQL / DSL 语法"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
		return "数据源查询超时，请缩小查询范围或检查数据源状态"
	}
	return clipPlainText(message, 220)
}

func dataSourceErrorText(root map[string]interface{}) string {
	parts := make([]string, 0, 3)
	for _, key := range []string{"errorType", "error", "message", "msg"} {
		if value := strings.TrimSpace(fmt.Sprint(root[key])); value != "" && value != "<nil>" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "：")
}

func newMonitorHTTPClient(timeout time.Duration, skipTLSVerify bool) *http.Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if skipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func formatDataSourceRequestError(ds *model.DataSource, targetURL string, err error) error {
	if err == nil || ds == nil {
		return err
	}
	baseURL, parseErr := url.Parse(strings.TrimSpace(targetURL))
	if parseErr != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		baseURL, parseErr = url.Parse(strings.TrimSpace(ds.URL))
	}
	if parseErr != nil || baseURL.Scheme != "http" || baseURL.Host == "" {
		return err
	}
	errText := err.Error()
	httpsPrefix := "https://" + baseURL.Host
	if !strings.Contains(errText, httpsPrefix) {
		return err
	}
	if strings.Contains(errText, "certificate") || strings.Contains(errText, "x509") {
		return fmt.Errorf("服务端将 HTTP 请求重定向到 HTTPS（%s），HTTPS 证书校验失败；请在数据源中开启“跳过 TLS 证书校验”，或关闭服务端强制 HTTPS。原始错误：%w", httpsPrefix, err)
	}
	return fmt.Errorf("服务端将 HTTP 请求重定向到 HTTPS（%s）。原始错误：%w", httpsPrefix, err)
}

func applyDataSourceAuth(req *http.Request, ds *model.DataSource) {
	switch ds.AuthType {
	case "basic":
		req.SetBasicAuth(ds.Username, ds.Password)
	case "bearer":
		if token := strings.TrimSpace(ds.Token); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
}

func applyDataSourceHeaders(req *http.Request, headers string) error {
	if strings.TrimSpace(headers) == "" {
		return nil
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(headers), &parsed); err != nil {
		return fmt.Errorf("自定义请求头必须是 JSON 对象")
	}
	for key, value := range parsed {
		if strings.TrimSpace(key) != "" {
			req.Header.Set(key, value)
		}
	}
	return nil
}

func (h *DataSourceHandler) ListAlertRules(c *gin.Context) {
	var rules []model.AlertRule
	query := h.db.Model(&model.AlertRule{})

	if dataSourceID := strings.TrimSpace(c.Query("dataSourceId")); dataSourceID != "" {
		query = query.Where("data_source_id = ?", dataSourceID)
	}
	if ruleGroupID := strings.TrimSpace(c.Query("ruleGroupId")); ruleGroupID != "" {
		query = query.Where("rule_group_id = ?", ruleGroupID)
	}
	if faultCenterID := strings.TrimSpace(c.Query("faultCenterId")); faultCenterID != "" {
		query = query.Where("fault_center_id = ?", faultCenterID)
	}
	if dataSourceType := strings.TrimSpace(c.Query("dataSourceType")); dataSourceType != "" {
		query = query.Where("data_source_type = ?", dataSourceType)
	}
	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		query = query.Where("enabled = ?", enabled == "true" || enabled == "1")
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR query LIKE ? OR detail_template LIKE ? OR annotations LIKE ?", like, like, like, like)
	}

	if err := query.Order("id DESC").Find(&rules).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取告警规则列表失败", "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "success", "data": rules})
}

func (h *DataSourceHandler) GetAlertRule(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var rule model.AlertRule
	if err := h.db.First(&rule, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "告警规则不存在"})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "success", "data": rule})
}

func (h *DataSourceHandler) CreateAlertRule(c *gin.Context) {
	var req model.AlertRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}

	if err := h.normalizeAlertRule(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "创建告警规则失败", "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "创建成功", "data": req})
}

func (h *DataSourceHandler) UpdateAlertRule(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var rule model.AlertRule
	if err := h.db.First(&rule, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "告警规则不存在"})
		return
	}

	var req model.AlertRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}

	if err := h.normalizeAlertRule(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}

	originalRule := rule
	wasActive := isActiveAlertRuleState(originalRule.LastState)
	rule.Name = req.Name
	rule.RuleGroupID = req.RuleGroupID
	rule.FaultCenterID = req.FaultCenterID
	rule.DataSourceID = req.DataSourceID
	rule.DataSourceIDs = req.DataSourceIDs
	rule.DataSourceType = req.DataSourceType
	rule.Query = req.Query
	rule.QueryMode = req.QueryMode
	rule.Index = req.Index
	rule.Condition = req.Condition
	rule.Threshold = req.Threshold
	rule.SeverityRules = req.SeverityRules
	rule.ForSeconds = req.ForSeconds
	rule.EvaluateInterval = req.EvaluateInterval
	rule.Severity = req.Severity
	rule.Enabled = req.Enabled
	rule.ChannelIDs = req.ChannelIDs
	rule.NotifyRecovery = req.NotifyRecovery
	rule.RepeatInterval = req.RepeatInterval
	rule.Labels = req.Labels
	rule.Annotations = req.Annotations
	rule.DetailTemplate = req.DetailTemplate
	rule.CallbackQueries = req.CallbackQueries
	rule.EffectiveTime = req.EffectiveTime

	if alertRuleEvaluationConfigChanged(originalRule, req) {
		rule.LastEvalAt = nil
		rule.LastError = ""
		rule.PendingSince = nil
		rule.FiringSince = nil
	}

	if !req.Enabled && wasActive {
		now := time.Now()
		message := fmt.Sprintf("规则「%s」已停用，活跃告警自动恢复", originalRule.Name)
		if _, err := h.recoverActiveAlertEvents(&originalRule, now, message); err != nil {
			c.JSON(500, gin.H{"code": 500, "message": "恢复活跃告警失败", "error": err.Error()})
			return
		}
		rule.LastState = "inactive"
		rule.PendingSince = nil
		rule.FiringSince = nil
		rule.LastEvalAt = &now
	}

	if err := h.db.Save(&rule).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "更新告警规则失败", "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": rule})
}

func alertRuleEvaluationConfigChanged(before, after model.AlertRule) bool {
	if before.DataSourceID != after.DataSourceID ||
		strings.TrimSpace(before.DataSourceIDs) != strings.TrimSpace(after.DataSourceIDs) ||
		strings.TrimSpace(before.DataSourceType) != strings.TrimSpace(after.DataSourceType) ||
		strings.TrimSpace(before.Query) != strings.TrimSpace(after.Query) ||
		strings.TrimSpace(before.QueryMode) != strings.TrimSpace(after.QueryMode) ||
		strings.TrimSpace(before.Index) != strings.TrimSpace(after.Index) ||
		normalizeCondition(before.Condition) != normalizeCondition(after.Condition) ||
		before.Threshold != after.Threshold ||
		strings.TrimSpace(before.SeverityRules) != strings.TrimSpace(after.SeverityRules) ||
		before.ForSeconds != after.ForSeconds ||
		before.EvaluateInterval != after.EvaluateInterval ||
		normalizeSeverity(before.Severity) != normalizeSeverity(after.Severity) ||
		strings.TrimSpace(before.Labels) != strings.TrimSpace(after.Labels) ||
		strings.TrimSpace(before.EffectiveTime) != strings.TrimSpace(after.EffectiveTime) {
		return true
	}
	return false
}

func (h *DataSourceHandler) DeleteAlertRule(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var rule model.AlertRule
	if err := h.db.First(&rule, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "告警规则不存在"})
		return
	}

	now := time.Now()
	message := fmt.Sprintf("规则「%s」已删除，活跃告警自动恢复", rule.Name)
	if _, err := h.recoverActiveAlertEvents(&rule, now, message); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "恢复活跃告警失败", "error": err.Error()})
		return
	}

	if err := h.db.Delete(&rule).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除告警规则失败", "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

type alertRuleBatchUpdateRequest struct {
	IDs           []uint `json:"ids"`
	RuleGroupID   *uint  `json:"ruleGroupId"`
	DataSourceID  *uint  `json:"dataSourceId"`
	FaultCenterID *uint  `json:"faultCenterId"`
	Enabled       *bool  `json:"enabled"`
}

type alertRuleBatchDeleteRequest struct {
	IDs []uint `json:"ids"`
}

func (h *DataSourceHandler) BatchUpdateAlertRules(c *gin.Context) {
	var req alertRuleBatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	ids := uniqueUintIDs(req.IDs)
	if len(ids) == 0 {
		c.JSON(400, gin.H{"code": 400, "message": "请选择需要批量更新的告警规则"})
		return
	}
	if req.RuleGroupID == nil && req.DataSourceID == nil && req.FaultCenterID == nil && req.Enabled == nil {
		c.JSON(400, gin.H{"code": 400, "message": "请选择至少一个需要更新的字段"})
		return
	}

	var ds model.DataSource
	if req.DataSourceID != nil {
		if *req.DataSourceID == 0 {
			c.JSON(400, gin.H{"code": 400, "message": "请选择数据源"})
			return
		}
		if err := h.db.First(&ds, *req.DataSourceID).Error; err != nil {
			c.JSON(400, gin.H{"code": 400, "message": "数据源不存在"})
			return
		}
	}
	if req.RuleGroupID != nil && *req.RuleGroupID > 0 {
		var count int64
		if err := h.db.Model(&model.AlertRuleGroup{}).Where("id = ?", *req.RuleGroupID).Count(&count).Error; err != nil || count == 0 {
			c.JSON(400, gin.H{"code": 400, "message": "规则组不存在"})
			return
		}
	}
	if req.FaultCenterID != nil && *req.FaultCenterID > 0 {
		var count int64
		if err := h.db.Model(&model.FaultCenter{}).Where("id = ?", *req.FaultCenterID).Count(&count).Error; err != nil || count == 0 {
			c.JSON(400, gin.H{"code": 400, "message": "故障中心不存在"})
			return
		}
	}

	var rules []model.AlertRule
	if err := h.db.Where("id IN ?", ids).Find(&rules).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取告警规则失败", "error": err.Error()})
		return
	}
	if len(rules) == 0 {
		c.JSON(404, gin.H{"code": 404, "message": "未找到可更新的告警规则"})
		return
	}

	now := time.Now()
	updated := 0
	for i := range rules {
		rule := &rules[i]
		original := *rule
		if req.RuleGroupID != nil {
			rule.RuleGroupID = *req.RuleGroupID
		}
		if req.FaultCenterID != nil {
			rule.FaultCenterID = *req.FaultCenterID
		}
		if req.DataSourceID != nil {
			rule.DataSourceID = ds.ID
			rule.DataSourceType = ds.Type
			if encoded, err := json.Marshal([]uint{ds.ID}); err == nil {
				rule.DataSourceIDs = string(encoded)
			}
		}
		if req.Enabled != nil {
			rule.Enabled = *req.Enabled
			if !*req.Enabled && isActiveAlertRuleState(original.LastState) {
				message := fmt.Sprintf("规则「%s」已批量停用，活跃告警自动恢复", original.Name)
				if _, err := h.recoverActiveAlertEvents(&original, now, message); err != nil {
					c.JSON(500, gin.H{"code": 500, "message": "恢复活跃告警失败", "error": err.Error()})
					return
				}
				rule.LastState = "inactive"
				rule.PendingSince = nil
				rule.FiringSince = nil
				rule.LastEvalAt = &now
			}
		}
		if err := h.db.Save(rule).Error; err != nil {
			c.JSON(500, gin.H{"code": 500, "message": "批量更新失败", "error": err.Error()})
			return
		}
		updated++
	}

	c.JSON(200, gin.H{"code": 0, "message": "批量更新成功", "data": gin.H{"updated": updated}})
}

func (h *DataSourceHandler) BatchDeleteAlertRules(c *gin.Context) {
	var req alertRuleBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	ids := uniqueUintIDs(req.IDs)
	if len(ids) == 0 {
		c.JSON(400, gin.H{"code": 400, "message": "请选择需要删除的告警规则"})
		return
	}
	var rules []model.AlertRule
	if err := h.db.Where("id IN ?", ids).Find(&rules).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取告警规则失败", "error": err.Error()})
		return
	}
	if len(rules) == 0 {
		c.JSON(404, gin.H{"code": 404, "message": "未找到可删除的告警规则"})
		return
	}
	now := time.Now()
	for i := range rules {
		rule := &rules[i]
		message := fmt.Sprintf("规则「%s」已批量删除，活跃告警自动恢复", rule.Name)
		if _, err := h.recoverActiveAlertEvents(rule, now, message); err != nil {
			c.JSON(500, gin.H{"code": 500, "message": "恢复活跃告警失败", "error": err.Error()})
			return
		}
	}
	if err := h.db.Where("id IN ?", ids).Delete(&model.AlertRule{}).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "批量删除失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "批量删除成功", "data": gin.H{"deleted": len(rules)}})
}

func (h *DataSourceHandler) ExportAlertRules(c *gin.Context) {
	var req alertRuleBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	ids := uniqueUintIDs(req.IDs)
	query := h.db.Model(&model.AlertRule{})
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}
	var rules []model.AlertRule
	if err := query.Order("id DESC").Find(&rules).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "导出告警规则失败", "error": err.Error()})
		return
	}

	exportRules := make([]watchAlertRuleExport, 0, len(rules))
	for _, rule := range rules {
		exportRules = append(exportRules, buildWatchAlertRuleExport(rule))
	}
	data, err := json.MarshalIndent(exportRules, "", "  ")
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "生成导出文件失败", "error": err.Error()})
		return
	}
	fileName := fmt.Sprintf("rules_export_%s.json", time.Now().Format("2006-01-02"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	c.Data(200, "application/json; charset=utf-8", data)
}

type watchAlertRuleExport struct {
	TenantID              string                 `json:"tenantId"`
	RuleID                string                 `json:"ruleId"`
	RuleGroupID           string                 `json:"ruleGroupId"`
	ExternalLabels        map[string]string      `json:"externalLabels"`
	DatasourceType        string                 `json:"datasourceType"`
	DatasourceID          []string               `json:"datasourceId"`
	RuleName              string                 `json:"ruleName"`
	EvalInterval          int                    `json:"evalInterval"`
	RepeatNoticeInterval  int                    `json:"repeatNoticeInterval"`
	Description           string                 `json:"description"`
	EffectiveTime         map[string]interface{} `json:"effectiveTime"`
	Severity              string                 `json:"severity"`
	PrometheusConfig      *watchAlertRuleConfig  `json:"prometheusConfig,omitempty"`
	VictoriaMetricsConfig *watchAlertRuleConfig  `json:"victoriaMetricsConfig,omitempty"`
	LokiConfig            *watchAlertRuleConfig  `json:"lokiConfig,omitempty"`
	ElasticSearchConfig   *watchAlertRuleConfig  `json:"elasticSearchConfig,omitempty"`
	ElasticsearchConfig   *watchAlertRuleConfig  `json:"elasticsearchConfig,omitempty"`
}

type watchAlertRuleConfig struct {
	PromQL          string                     `json:"promQL,omitempty"`
	LogQL           string                     `json:"logQL,omitempty"`
	Query           string                     `json:"query,omitempty"`
	Index           string                     `json:"index,omitempty"`
	Annotations     string                     `json:"annotations"`
	Rules           []watchAlertRuleCondition  `json:"rules"`
	CallbackPromQLs []watchAlertCallbackPromQL `json:"callbackPromQLs"`
}

type watchAlertRuleCondition struct {
	ForDuration int    `json:"forDuration"`
	Severity    string `json:"severity"`
	Expr        string `json:"expr"`
}

type watchAlertCallbackPromQL struct {
	Key          string `json:"key,omitempty"`
	Name         string `json:"name,omitempty"`
	Title        string `json:"title,omitempty"`
	PromQL       string `json:"promQL,omitempty"`
	Query        string `json:"query,omitempty"`
	Value        string `json:"value,omitempty"`
	DataSourceID string `json:"datasourceId,omitempty"`
	QueryMode    string `json:"queryMode,omitempty"`
	RangeSeconds int    `json:"rangeSeconds,omitempty"`
}

func buildWatchAlertRuleExport(rule model.AlertRule) watchAlertRuleExport {
	conditions, err := parseSeverityConditions(&rule)
	if err != nil || len(conditions) == 0 {
		conditions = []severityCondition{{
			Severity:   normalizeSeverity(rule.Severity),
			Condition:  normalizeCondition(rule.Condition),
			Threshold:  rule.Threshold,
			ForSeconds: positiveInt(rule.ForSeconds, 60),
		}}
	}
	watchRules := make([]watchAlertRuleCondition, 0, len(conditions))
	for _, condition := range conditions {
		watchRules = append(watchRules, watchAlertRuleCondition{
			ForDuration: positiveInt(condition.ForSeconds, rule.ForSeconds),
			Severity:    normalizeSeverityLevel(condition.Severity),
			Expr:        watchAlertConditionExpr(condition.Condition, condition.Threshold),
		})
	}

	config := &watchAlertRuleConfig{
		Annotations:     firstNonEmpty(rule.DetailTemplate, rule.Annotations),
		Rules:           watchRules,
		CallbackPromQLs: buildWatchAlertCallbackPromQLs(rule.CallbackQueries),
	}
	switch strings.ToLower(strings.TrimSpace(rule.DataSourceType)) {
	case "loki":
		config.LogQL = rule.Query
	case "elasticsearch", "opensearch":
		config.Query = rule.Query
		config.Index = rule.Index
	default:
		config.PromQL = rule.Query
	}

	exported := watchAlertRuleExport{
		TenantID:             "default",
		RuleID:               fmt.Sprintf("opshub-rule-%d", rule.ID),
		RuleGroupID:          fmt.Sprintf("opshub-rule-group-%d", rule.RuleGroupID),
		ExternalLabels:       parseStringMap(rule.Labels),
		DatasourceType:       watchAlertDatasourceType(rule.DataSourceType),
		DatasourceID:         watchAlertDatasourceIDs(rule),
		RuleName:             rule.Name,
		EvalInterval:         positiveInt(rule.EvaluateInterval, 60),
		RepeatNoticeInterval: rule.RepeatInterval,
		Description:          firstNonEmpty(rule.DetailTemplate, rule.Annotations),
		EffectiveTime:        watchAlertEffectiveTime(rule.EffectiveTime),
		Severity:             normalizeSeverityLevel(rule.Severity),
	}
	switch strings.ToLower(strings.TrimSpace(rule.DataSourceType)) {
	case "victoriametrics":
		exported.VictoriaMetricsConfig = config
	case "loki":
		exported.LokiConfig = config
	case "elasticsearch", "opensearch":
		exported.ElasticSearchConfig = config
	default:
		exported.PrometheusConfig = config
	}
	return exported
}

func watchAlertDatasourceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "victoriametrics":
		return "VictoriaMetrics"
	case "loki":
		return "Loki"
	case "elasticsearch", "opensearch":
		return "Elasticsearch"
	default:
		return "Prometheus"
	}
}

func watchAlertDatasourceIDs(rule model.AlertRule) []string {
	ids, err := parseRuleDataSourceIDs(rule.DataSourceIDs)
	if err != nil || len(ids) == 0 {
		ids = []uint{rule.DataSourceID}
	}
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		result = append(result, fmt.Sprintf("opshub-datasource-%d", id))
	}
	return result
}

func watchAlertConditionExpr(condition string, threshold float64) string {
	operator := map[string]string{
		"gt":  ">",
		"gte": ">=",
		"lt":  "<",
		"lte": "<=",
		"eq":  "==",
		"neq": "!=",
	}[normalizeCondition(condition)]
	if operator == "" {
		operator = ">"
	}
	return operator + formatRuleValue(threshold)
}

func watchAlertEffectiveTime(raw string) map[string]interface{} {
	result := map[string]interface{}{"week": nil, "startTime": 0, "endTime": 86340}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return result
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return result
	}
	if len(parsed) == 0 {
		return result
	}
	return parsed
}

func buildWatchAlertCallbackPromQLs(raw string) []watchAlertCallbackPromQL {
	callbacks := parseAlertCallbackQueries(raw)
	if len(callbacks) == 0 {
		return nil
	}
	result := make([]watchAlertCallbackPromQL, 0, len(callbacks))
	for _, callback := range callbacks {
		query := firstNonEmpty(callback.Query, callback.Value)
		if strings.TrimSpace(query) == "" {
			continue
		}
		result = append(result, watchAlertCallbackPromQL{
			Key:          firstNonEmpty(callback.Key, callback.Name),
			Name:         firstNonEmpty(callback.Name, callback.Key),
			Title:        firstNonEmpty(callback.Name, callback.Key),
			PromQL:       query,
			Query:        query,
			Value:        query,
			DataSourceID: fmt.Sprintf("opshub-datasource-%d", callback.DataSourceID),
			QueryMode:    firstNonEmpty(callback.QueryMode, "range"),
			RangeSeconds: positiveInt(callback.RangeSeconds, 1800),
		})
	}
	return result
}

type prometheusRuleImportRequest struct {
	YAML            string `json:"yaml"`
	Content         string `json:"content"`
	JSON            string `json:"json"`
	DataSourceID    uint   `json:"dataSourceId"`
	RuleGroupID     uint   `json:"ruleGroupId"`
	FaultCenterID   uint   `json:"faultCenterId"`
	DefaultSeverity string `json:"defaultSeverity"`
}

type alertRuleImportResult struct {
	Imported int               `json:"imported"`
	Skipped  []string          `json:"skipped"`
	Rules    []model.AlertRule `json:"rules"`
}

type prometheusRuleDocument struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Groups []prometheusRuleGroup `yaml:"groups"`
	} `yaml:"spec"`
	Groups []prometheusRuleGroup `yaml:"groups"`
	Rules  []prometheusRuleSpec  `yaml:"rules"`
}

type prometheusRuleGroup struct {
	Name  string               `yaml:"name"`
	Rules []prometheusRuleSpec `yaml:"rules"`
}

type prometheusRuleSpec struct {
	Alert       string                 `yaml:"alert"`
	Expr        string                 `yaml:"expr"`
	For         string                 `yaml:"for"`
	Labels      map[string]interface{} `yaml:"labels"`
	Annotations map[string]interface{} `yaml:"annotations"`
}

func (h *DataSourceHandler) ImportPrometheusRuleYAML(c *gin.Context) {
	var req prometheusRuleImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	result, statusCode, message, err := h.importPrometheusRuleYAMLData(req)
	respondAlertRuleImport(c, result, statusCode, message, err)
}

func (h *DataSourceHandler) ImportAlertRules(c *gin.Context) {
	var req prometheusRuleImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	content := strings.TrimSpace(firstNonEmpty(req.Content, req.JSON, req.YAML))
	if content == "" {
		c.JSON(400, gin.H{"code": 400, "message": "请粘贴 WatchAlert/OpsHub JSON 或 PrometheusRule YAML"})
		return
	}
	if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
		result, statusCode, message, err := h.importWatchAlertRuleJSON(content, req)
		respondAlertRuleImport(c, result, statusCode, message, err)
		return
	}
	req.YAML = content
	result, statusCode, message, err := h.importPrometheusRuleYAMLData(req)
	respondAlertRuleImport(c, result, statusCode, message, err)
}

func respondAlertRuleImport(c *gin.Context, result alertRuleImportResult, statusCode int, message string, err error) {
	if statusCode == 0 {
		statusCode = 500
	}
	if message == "" {
		message = "导入失败"
	}
	if err != nil {
		payload := gin.H{"code": statusCode, "message": message, "error": err.Error()}
		if len(result.Skipped) > 0 {
			payload["data"] = gin.H{"skipped": result.Skipped}
		}
		c.JSON(statusCode, payload)
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "导入成功",
		"data": gin.H{
			"imported": result.Imported,
			"skipped":  result.Skipped,
			"rules":    result.Rules,
		},
	})
}

func (h *DataSourceHandler) importPrometheusRuleYAMLData(req prometheusRuleImportRequest) (alertRuleImportResult, int, string, error) {
	req.YAML = strings.TrimSpace(firstNonEmpty(req.YAML, req.Content))
	if req.YAML == "" {
		return alertRuleImportResult{}, 400, "请粘贴 PrometheusRule YAML", fmt.Errorf("内容为空")
	}
	if req.DataSourceID == 0 {
		return alertRuleImportResult{}, 400, "请选择 Prometheus 或 VictoriaMetrics 数据源", fmt.Errorf("数据源为空")
	}

	var ds model.DataSource
	if err := h.db.First(&ds, req.DataSourceID).Error; err != nil {
		return alertRuleImportResult{}, 400, "数据源不存在", err
	}
	if ds.Type != "prometheus" && ds.Type != "victoriametrics" {
		return alertRuleImportResult{}, 400, "PrometheusRule 只能导入到 Prometheus 或 VictoriaMetrics 数据源", fmt.Errorf("数据源类型为 %s", ds.Type)
	}
	if req.RuleGroupID == 0 {
		req.RuleGroupID = findDefaultRuleGroup(h.db)
	}
	if req.FaultCenterID == 0 {
		req.FaultCenterID = findDefaultFaultCenter(h.db)
	}
	if req.RuleGroupID == 0 || req.FaultCenterID == 0 {
		return alertRuleImportResult{}, 400, "请先创建规则组和故障中心", fmt.Errorf("规则组或故障中心为空")
	}

	docs, err := decodePrometheusRuleDocuments(req.YAML)
	if err != nil {
		return alertRuleImportResult{}, 400, "解析 PrometheusRule YAML 失败", err
	}
	defaultSeverity := normalizePrometheusRuleSeverity(req.DefaultSeverity)
	if defaultSeverity == "" {
		defaultSeverity = "p1"
	}

	importedRules := make([]model.AlertRule, 0)
	skipped := make([]string, 0)
	for _, doc := range docs {
		groups := prometheusRuleGroupsFromDocument(doc)
		for _, group := range groups {
			for _, promRule := range group.Rules {
				if strings.TrimSpace(promRule.Alert) == "" {
					continue
				}
				alertRule, skipReason := buildAlertRuleFromPrometheusRule(promRule, group.Name, req, ds, defaultSeverity)
				if skipReason != "" {
					skipped = append(skipped, fmt.Sprintf("%s: %s", firstNonEmpty(promRule.Alert, "未命名规则"), skipReason))
					continue
				}
				if err := h.normalizeAlertRule(&alertRule); err != nil {
					skipped = append(skipped, fmt.Sprintf("%s: %s", firstNonEmpty(promRule.Alert, "未命名规则"), err.Error()))
					continue
				}
				if err := h.db.Create(&alertRule).Error; err != nil {
					return alertRuleImportResult{Skipped: skipped, Rules: importedRules, Imported: len(importedRules)}, 500, "保存导入规则失败", err
				}
				importedRules = append(importedRules, alertRule)
			}
		}
	}
	if len(importedRules) == 0 {
		return alertRuleImportResult{Skipped: skipped}, 400, "没有可导入的 alert 规则", fmt.Errorf("没有可导入的 alert 规则")
	}

	return alertRuleImportResult{Imported: len(importedRules), Skipped: skipped, Rules: importedRules}, 200, "导入成功", nil
}

func (h *DataSourceHandler) importWatchAlertRuleJSON(raw string, req prometheusRuleImportRequest) (alertRuleImportResult, int, string, error) {
	exports, err := decodeWatchAlertRuleExports(raw)
	if err != nil {
		return alertRuleImportResult{}, 400, "解析 WatchAlert/OpsHub JSON 失败", err
	}
	if len(exports) == 0 {
		return alertRuleImportResult{}, 400, "没有可导入的告警规则", fmt.Errorf("JSON 中没有规则")
	}
	if req.RuleGroupID == 0 {
		req.RuleGroupID = findDefaultRuleGroup(h.db)
	}
	if req.FaultCenterID == 0 {
		req.FaultCenterID = findDefaultFaultCenter(h.db)
	}
	if req.RuleGroupID == 0 || req.FaultCenterID == 0 {
		return alertRuleImportResult{}, 400, "请先创建规则组和故障中心", fmt.Errorf("规则组或故障中心为空")
	}

	defaultSeverity := normalizePrometheusRuleSeverity(req.DefaultSeverity)
	if defaultSeverity == "" {
		defaultSeverity = "p1"
	}
	importedRules := make([]model.AlertRule, 0, len(exports))
	skipped := make([]string, 0)
	for _, exported := range exports {
		alertRule, skipReason := h.buildAlertRuleFromWatchAlertExport(exported, req, defaultSeverity)
		if skipReason != "" {
			skipped = append(skipped, fmt.Sprintf("%s: %s", firstNonEmpty(exported.RuleName, exported.RuleID, "未命名规则"), skipReason))
			continue
		}
		if err := h.normalizeAlertRule(&alertRule); err != nil {
			skipped = append(skipped, fmt.Sprintf("%s: %s", firstNonEmpty(exported.RuleName, exported.RuleID, "未命名规则"), err.Error()))
			continue
		}
		if err := h.db.Create(&alertRule).Error; err != nil {
			return alertRuleImportResult{Imported: len(importedRules), Skipped: skipped, Rules: importedRules}, 500, "保存导入规则失败", err
		}
		importedRules = append(importedRules, alertRule)
	}
	if len(importedRules) == 0 {
		return alertRuleImportResult{Skipped: skipped}, 400, "没有可导入的告警规则", fmt.Errorf("没有可导入的告警规则")
	}
	return alertRuleImportResult{Imported: len(importedRules), Skipped: skipped, Rules: importedRules}, 200, "导入成功", nil
}

func decodeWatchAlertRuleExports(raw string) ([]watchAlertRuleExport, error) {
	var list []watchAlertRuleExport
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		return filterWatchAlertRuleExports(list), nil
	}
	var wrapper struct {
		Data  []watchAlertRuleExport `json:"data"`
		Rules []watchAlertRuleExport `json:"rules"`
		Items []watchAlertRuleExport `json:"items"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapper); err == nil {
		switch {
		case len(wrapper.Data) > 0:
			return filterWatchAlertRuleExports(wrapper.Data), nil
		case len(wrapper.Rules) > 0:
			return filterWatchAlertRuleExports(wrapper.Rules), nil
		case len(wrapper.Items) > 0:
			return filterWatchAlertRuleExports(wrapper.Items), nil
		}
	}
	var single watchAlertRuleExport
	if err := json.Unmarshal([]byte(raw), &single); err != nil {
		return nil, err
	}
	return filterWatchAlertRuleExports([]watchAlertRuleExport{single}), nil
}

func filterWatchAlertRuleExports(items []watchAlertRuleExport) []watchAlertRuleExport {
	result := make([]watchAlertRuleExport, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.RuleName) == "" && item.ruleConfig() == nil {
			continue
		}
		result = append(result, item)
	}
	return result
}

func (item watchAlertRuleExport) ruleConfig() *watchAlertRuleConfig {
	switch normalizeWatchAlertDatasourceType(item.DatasourceType) {
	case "victoriametrics":
		if item.VictoriaMetricsConfig != nil {
			return item.VictoriaMetricsConfig
		}
	case "loki":
		if item.LokiConfig != nil {
			return item.LokiConfig
		}
	case "elasticsearch":
		if item.ElasticSearchConfig != nil {
			return item.ElasticSearchConfig
		}
		if item.ElasticsearchConfig != nil {
			return item.ElasticsearchConfig
		}
	default:
		if item.PrometheusConfig != nil {
			return item.PrometheusConfig
		}
	}
	for _, candidate := range []*watchAlertRuleConfig{
		item.PrometheusConfig,
		item.VictoriaMetricsConfig,
		item.LokiConfig,
		item.ElasticSearchConfig,
		item.ElasticsearchConfig,
	} {
		if candidate != nil {
			return candidate
		}
	}
	return nil
}

func (h *DataSourceHandler) buildAlertRuleFromWatchAlertExport(exported watchAlertRuleExport, req prometheusRuleImportRequest, defaultSeverity string) (model.AlertRule, string) {
	config := exported.ruleConfig()
	if config == nil {
		return model.AlertRule{}, "缺少数据源查询配置"
	}
	ds, err := h.resolveImportDataSource(req, exported)
	if err != nil {
		return model.AlertRule{}, err.Error()
	}
	query := strings.TrimSpace(firstNonEmpty(config.PromQL, config.LogQL, config.Query))
	if query == "" {
		return model.AlertRule{}, "查询语句为空"
	}
	conditions := watchAlertSeverityConditions(config.Rules, exported.Severity, defaultSeverity)
	if len(conditions) == 0 {
		conditions = []severityCondition{{Severity: defaultSeverity, Condition: "gt", Threshold: 0, ForSeconds: 60}}
	}
	primary := conditions[0]
	severityRules, _ := json.Marshal(conditions)
	dataSourceIDs, _ := json.Marshal([]uint{ds.ID})
	effectiveTime := watchAlertImportEffectiveTime(exported.EffectiveTime)
	callbackQueries := watchAlertImportCallbackQueries(config.CallbackPromQLs, ds.ID)
	detail := firstNonEmpty(config.Annotations, exported.Description)
	if detail == "" {
		detail = fmt.Sprintf("WatchAlert 规则 %s 触发，当前值 ${labels.value} %s ${threshold}", strings.TrimSpace(exported.RuleName), getConditionText(primary.Condition))
	}
	return model.AlertRule{
		Name:             firstNonEmpty(strings.TrimSpace(exported.RuleName), strings.TrimSpace(exported.RuleID), "WatchAlert 导入规则"),
		RuleGroupID:      req.RuleGroupID,
		FaultCenterID:    req.FaultCenterID,
		DataSourceID:     ds.ID,
		DataSourceIDs:    string(dataSourceIDs),
		DataSourceType:   ds.Type,
		Query:            query,
		QueryMode:        "instant",
		Index:            strings.Trim(config.Index, "/ "),
		Condition:        primary.Condition,
		Threshold:        primary.Threshold,
		SeverityRules:    string(severityRules),
		ForSeconds:       positiveInt(primary.ForSeconds, 60),
		EvaluateInterval: positiveInt(exported.EvalInterval, 60),
		Severity:         primary.Severity,
		Enabled:          true,
		NotifyRecovery:   true,
		RepeatInterval:   positiveInt(exported.RepeatNoticeInterval, 3600),
		Labels:           marshalStringMap(exported.ExternalLabels),
		DetailTemplate:   detail,
		CallbackQueries:  callbackQueries,
		EffectiveTime:    effectiveTime,
		LastState:        "inactive",
	}, ""
}

func (h *DataSourceHandler) resolveImportDataSource(req prometheusRuleImportRequest, exported watchAlertRuleExport) (model.DataSource, error) {
	if req.DataSourceID > 0 {
		var ds model.DataSource
		if err := h.db.First(&ds, req.DataSourceID).Error; err != nil {
			return model.DataSource{}, fmt.Errorf("数据源不存在")
		}
		return ds, nil
	}
	for _, rawID := range exported.DatasourceID {
		if id := parseTrailingUint(rawID); id > 0 {
			var ds model.DataSource
			if err := h.db.First(&ds, id).Error; err == nil {
				return ds, nil
			}
		}
	}
	dsType := normalizeWatchAlertDatasourceType(exported.DatasourceType)
	if dsType == "" {
		dsType = "prometheus"
	}
	var ds model.DataSource
	if err := h.db.Where("type = ? AND enabled = ?", dsType, true).Order("id ASC").First(&ds).Error; err == nil {
		return ds, nil
	}
	return model.DataSource{}, fmt.Errorf("请选择数据源")
}

func normalizeWatchAlertDatasourceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "prometheus", "prom":
		return "prometheus"
	case "victoriametrics", "victoria-metrics", "vm":
		return "victoriametrics"
	case "loki":
		return "loki"
	case "elasticsearch", "elastic-search", "opensearch", "open-search", "os", "es":
		return "elasticsearch"
	default:
		return ""
	}
}

func parseTrailingUint(value string) uint {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if id, err := strconv.ParseUint(value, 10, 64); err == nil {
		return uint(id)
	}
	matches := regexp.MustCompile(`(\d+)$`).FindStringSubmatch(value)
	if len(matches) != 2 {
		return 0
	}
	id, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

func watchAlertSeverityConditions(rules []watchAlertRuleCondition, rootSeverity, defaultSeverity string) []severityCondition {
	result := make([]severityCondition, 0, len(rules))
	for _, rule := range rules {
		condition, threshold, ok := parseWatchAlertConditionExpr(rule.Expr)
		if !ok {
			continue
		}
		severity := normalizePrometheusRuleSeverity(firstNonEmpty(rule.Severity, rootSeverity, defaultSeverity))
		if severity == "" {
			severity = defaultSeverity
		}
		result = append(result, severityCondition{
			Severity:   severity,
			Condition:  condition,
			Threshold:  threshold,
			ForSeconds: positiveInt(rule.ForDuration, 60),
		})
	}
	return result
}

func parseWatchAlertConditionExpr(expr string) (string, float64, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "gt", 0, true
	}
	matches := regexp.MustCompile(`^(>=|<=|==|!=|>|<|=)\s*(-?\d+(?:\.\d+)?)$`).FindStringSubmatch(expr)
	if len(matches) != 3 {
		return "", 0, false
	}
	threshold, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return "", 0, false
	}
	condition := map[string]string{
		">":  "gt",
		">=": "gte",
		"<":  "lt",
		"<=": "lte",
		"=":  "eq",
		"==": "eq",
		"!=": "neq",
	}[matches[1]]
	return condition, threshold, condition != ""
}

func watchAlertImportEffectiveTime(value map[string]interface{}) string {
	if len(value) == 0 {
		return `{"week":[],"startTime":"","endTime":""}`
	}
	data, err := json.Marshal(value)
	if err != nil || len(data) == 0 {
		return `{"week":[],"startTime":"","endTime":""}`
	}
	return string(data)
}

func watchAlertImportCallbackQueries(callbacks []watchAlertCallbackPromQL, defaultDataSourceID uint) string {
	if len(callbacks) == 0 {
		return ""
	}
	result := make([]alertCallbackQueryConfig, 0, len(callbacks))
	for _, callback := range callbacks {
		query := strings.TrimSpace(firstNonEmpty(callback.PromQL, callback.Query, callback.Value))
		if query == "" {
			continue
		}
		dsID := parseTrailingUint(callback.DataSourceID)
		if dsID == 0 {
			dsID = defaultDataSourceID
		}
		result = append(result, alertCallbackQueryConfig{
			Key:          firstNonEmpty(callback.Key, callback.Name, callback.Title, "callback"),
			Name:         firstNonEmpty(callback.Name, callback.Title, callback.Key),
			Query:        query,
			Value:        query,
			DataSourceID: dsID,
			QueryMode:    firstNonEmpty(callback.QueryMode, "range"),
			RangeSeconds: positiveInt(callback.RangeSeconds, 1800),
		})
	}
	if len(result) == 0 {
		return ""
	}
	data, _ := json.Marshal(result)
	return string(data)
}

func decodePrometheusRuleDocuments(raw string) ([]prometheusRuleDocument, error) {
	decoder := yaml.NewDecoder(strings.NewReader(raw))
	docs := make([]prometheusRuleDocument, 0)
	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if node.Kind == 0 {
			continue
		}
		var doc prometheusRuleDocument
		if err := node.Decode(&doc); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("YAML 内容为空")
	}
	return docs, nil
}

func prometheusRuleGroupsFromDocument(doc prometheusRuleDocument) []prometheusRuleGroup {
	if len(doc.Spec.Groups) > 0 {
		return doc.Spec.Groups
	}
	if len(doc.Groups) > 0 {
		return doc.Groups
	}
	if len(doc.Rules) > 0 {
		return []prometheusRuleGroup{{Name: firstNonEmpty(doc.Metadata.Name, "PrometheusRule"), Rules: doc.Rules}}
	}
	return nil
}

func buildAlertRuleFromPrometheusRule(promRule prometheusRuleSpec, groupName string, req prometheusRuleImportRequest, ds model.DataSource, defaultSeverity string) (model.AlertRule, string) {
	expr := strings.TrimSpace(promRule.Expr)
	if expr == "" {
		return model.AlertRule{}, "expr 为空"
	}
	labels := stringMapFromYAML(promRule.Labels)
	annotations := stringMapFromYAML(promRule.Annotations)
	query, condition, threshold, ok := splitPrometheusAlertExpression(expr)
	if !ok {
		query = expr
		condition = "gt"
		threshold = 0
	}
	severity := normalizePrometheusRuleSeverity(firstNonEmpty(labels["severity"], labels["level"], defaultSeverity))
	if severity == "" {
		severity = defaultSeverity
	}
	forSeconds := parsePrometheusDurationSeconds(promRule.For)
	if forSeconds <= 0 {
		forSeconds = 60
	}
	detail := firstNonEmpty(annotations["description"], annotations["summary"], annotations["message"], annotations["runbook_url"])
	if detail == "" {
		detail = fmt.Sprintf("PrometheusRule %s 触发，当前值 ${labels.value} %s ${threshold}", strings.TrimSpace(promRule.Alert), getConditionText(condition))
	}
	labelJSON := marshalStringMap(labels)
	annotationJSON := marshalStringMap(annotations)
	dataSourceIDs, _ := json.Marshal([]uint{ds.ID})
	severityRules, _ := json.Marshal([]severityCondition{{
		Severity:   severity,
		Condition:  condition,
		Threshold:  threshold,
		ForSeconds: forSeconds,
	}})
	name := strings.TrimSpace(promRule.Alert)
	if groupName = strings.TrimSpace(groupName); groupName != "" {
		name = fmt.Sprintf("%s / %s", groupName, name)
	}
	return model.AlertRule{
		Name:             name,
		RuleGroupID:      req.RuleGroupID,
		FaultCenterID:    req.FaultCenterID,
		DataSourceID:     ds.ID,
		DataSourceIDs:    string(dataSourceIDs),
		DataSourceType:   ds.Type,
		Query:            query,
		QueryMode:        "instant",
		Condition:        condition,
		Threshold:        threshold,
		SeverityRules:    string(severityRules),
		ForSeconds:       forSeconds,
		EvaluateInterval: 60,
		Severity:         severity,
		Enabled:          true,
		NotifyRecovery:   true,
		RepeatInterval:   3600,
		Labels:           labelJSON,
		Annotations:      annotationJSON,
		DetailTemplate:   detail,
		EffectiveTime:    `{"week":[],"startTime":"","endTime":""}`,
		LastState:        "inactive",
	}, ""
}

func stringMapFromYAML(values map[string]interface{}) map[string]string {
	result := map[string]string{}
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		result[key] = strings.TrimSpace(fmt.Sprint(value))
	}
	return result
}

func normalizePrometheusRuleSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "p0", "critical", "fatal", "emergency", "page":
		return "p0"
	case "p1", "warning", "warn", "major", "serious", "high":
		return "p1"
	case "p2", "info", "notice", "minor", "low":
		return "p2"
	default:
		return ""
	}
}

func splitPrometheusAlertExpression(expr string) (string, string, float64, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", "", 0, false
	}
	for i := len(expr) - 1; i >= 0; i-- {
		if !isTopLevelPromQLOperatorPosition(expr, i) {
			continue
		}
		operator := ""
		for _, candidate := range []string{">=", "<=", "==", "!=", ">", "<"} {
			if strings.HasPrefix(expr[i:], candidate) {
				operator = candidate
				break
			}
		}
		if operator == "" {
			continue
		}
		left := strings.TrimSpace(expr[:i])
		right := strings.TrimSpace(expr[i+len(operator):])
		if strings.HasPrefix(strings.ToLower(right), "bool ") {
			right = strings.TrimSpace(right[5:])
		}
		threshold, err := strconv.ParseFloat(right, 64)
		if err != nil || left == "" {
			continue
		}
		return left, prometheusComparisonToCondition(operator), threshold, true
	}
	return expr, "gt", 0, false
}

func isTopLevelPromQLOperatorPosition(expr string, pos int) bool {
	depthParen := 0
	depthBrace := 0
	depthBracket := 0
	inQuote := byte(0)
	escaped := false
	for i := 0; i < pos; i++ {
		ch := expr[i]
		if inQuote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			inQuote = ch
		case '(':
			depthParen++
		case ')':
			if depthParen > 0 {
				depthParen--
			}
		case '{':
			depthBrace++
		case '}':
			if depthBrace > 0 {
				depthBrace--
			}
		case '[':
			depthBracket++
		case ']':
			if depthBracket > 0 {
				depthBracket--
			}
		}
	}
	if inQuote != 0 || depthParen != 0 || depthBrace != 0 || depthBracket != 0 {
		return false
	}
	if pos > 0 {
		prev := expr[pos-1]
		if prev == '!' || prev == '=' || prev == '~' {
			return false
		}
	}
	ch := expr[pos]
	return ch == '>' || ch == '<' || ch == '=' || ch == '!'
}

func prometheusComparisonToCondition(operator string) string {
	switch operator {
	case ">":
		return "gt"
	case ">=":
		return "gte"
	case "<":
		return "lt"
	case "<=":
		return "lte"
	case "==":
		return "eq"
	case "!=":
		return "neq"
	default:
		return "gt"
	}
}

func parsePrometheusDurationSeconds(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if duration, err := time.ParseDuration(raw); err == nil {
		return int(math.Ceil(duration.Seconds()))
	}
	pattern := regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)(ms|s|m|h|d|w|y)`)
	matches := pattern.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return 0
	}
	consumed := strings.Builder{}
	total := 0.0
	for _, match := range matches {
		consumed.WriteString(match[0])
		value, _ := strconv.ParseFloat(match[1], 64)
		switch strings.ToLower(match[2]) {
		case "ms":
			total += value / 1000
		case "s":
			total += value
		case "m":
			total += value * 60
		case "h":
			total += value * 3600
		case "d":
			total += value * 86400
		case "w":
			total += value * 7 * 86400
		case "y":
			total += value * 365 * 86400
		}
	}
	if strings.ReplaceAll(strings.ToLower(raw), " ", "") != strings.ToLower(consumed.String()) {
		return 0
	}
	if total <= 0 {
		return 0
	}
	return int(math.Ceil(total))
}

type alertRuleEvaluationResult struct {
	RuleID          uint                              `json:"ruleId"`
	RuleName        string                            `json:"ruleName"`
	RuleGroupID     uint                              `json:"ruleGroupId"`
	FaultCenterID   uint                              `json:"faultCenterId"`
	DataSourceID    uint                              `json:"dataSourceId"`
	DataSourceName  string                            `json:"dataSourceName"`
	DataSourceType  string                            `json:"dataSourceType"`
	Severity        string                            `json:"severity"`
	State           string                            `json:"state"`
	PreviousState   string                            `json:"previousState"`
	Matched         bool                              `json:"matched"`
	Value           float64                           `json:"value"`
	Condition       string                            `json:"condition"`
	Threshold       float64                           `json:"threshold"`
	ForSeconds      int                               `json:"forSeconds"`
	Labels          map[string]string                 `json:"labels,omitempty"`
	Fingerprint     string                            `json:"fingerprint,omitempty"`
	MatchedLogs     []matchedLogEntry                 `json:"matchedLogs,omitempty"`
	MatchedLogCount int                               `json:"matchedLogCount,omitempty"`
	MatchedLogQuery string                            `json:"matchedLogQuery,omitempty"`
	Message         string                            `json:"message"`
	StatusCode      int                               `json:"statusCode"`
	QueryResult     interface{}                       `json:"queryResult,omitempty"`
	EvaluatedAt     time.Time                         `json:"evaluatedAt"`
	StartedAt       *time.Time                        `json:"startedAt,omitempty"`
	Samples         []alertRuleEvaluationSampleResult `json:"samples,omitempty"`
}

type alertRuleEvaluationSampleResult struct {
	Fingerprint     string            `json:"fingerprint,omitempty"`
	Matched         bool              `json:"matched"`
	State           string            `json:"state"`
	Value           float64           `json:"value"`
	Severity        string            `json:"severity"`
	Condition       string            `json:"condition"`
	Threshold       float64           `json:"threshold"`
	ForSeconds      int               `json:"forSeconds"`
	Labels          map[string]string `json:"labels,omitempty"`
	MatchedLogs     []matchedLogEntry `json:"matchedLogs,omitempty"`
	MatchedLogCount int               `json:"matchedLogCount,omitempty"`
	MatchedLogQuery string            `json:"matchedLogQuery,omitempty"`
	Message         string            `json:"message"`
}

type matchedLogEntry struct {
	Timestamp string            `json:"timestamp,omitempty"`
	Line      string            `json:"line"`
	Labels    map[string]string `json:"labels,omitempty"`
}

func (r *alertRuleEvaluationResult) copyForSample(sample ruleEvaluationSample, condition severityCondition) *alertRuleEvaluationResult {
	return &alertRuleEvaluationResult{
		RuleID:          r.RuleID,
		RuleName:        r.RuleName,
		RuleGroupID:     r.RuleGroupID,
		FaultCenterID:   r.FaultCenterID,
		DataSourceID:    r.DataSourceID,
		DataSourceName:  r.DataSourceName,
		DataSourceType:  r.DataSourceType,
		Severity:        condition.Severity,
		PreviousState:   r.PreviousState,
		Matched:         condition.Matched,
		Value:           sample.Value,
		Condition:       condition.Condition,
		Threshold:       condition.Threshold,
		ForSeconds:      condition.ForSeconds,
		Labels:          cloneStringMap(sample.Labels),
		MatchedLogs:     cloneMatchedLogEntries(sample.MatchedLogs),
		MatchedLogCount: sample.MatchedLogCount,
		MatchedLogQuery: sample.MatchedLogQuery,
		StatusCode:      r.StatusCode,
		EvaluatedAt:     r.EvaluatedAt,
	}
}

func (r *alertRuleEvaluationResult) toSampleResult() alertRuleEvaluationSampleResult {
	return alertRuleEvaluationSampleResult{
		Fingerprint:     r.Fingerprint,
		Matched:         r.Matched,
		State:           r.State,
		Value:           r.Value,
		Severity:        r.Severity,
		Condition:       r.Condition,
		Threshold:       r.Threshold,
		ForSeconds:      r.ForSeconds,
		Labels:          cloneStringMap(r.Labels),
		MatchedLogs:     cloneMatchedLogEntries(r.MatchedLogs),
		MatchedLogCount: r.MatchedLogCount,
		MatchedLogQuery: r.MatchedLogQuery,
		Message:         r.Message,
	}
}

func applyEvaluationSummary(result, representative *alertRuleEvaluationResult, firingCount, pendingCount, recoveringCount, recoveredCount, total int) {
	result.Value = representative.Value
	result.Labels = cloneStringMap(representative.Labels)
	result.Fingerprint = representative.Fingerprint
	result.Severity = representative.Severity
	result.Condition = representative.Condition
	result.Threshold = representative.Threshold
	result.ForSeconds = representative.ForSeconds
	result.StartedAt = representative.StartedAt
	result.MatchedLogs = cloneMatchedLogEntries(representative.MatchedLogs)
	result.MatchedLogCount = representative.MatchedLogCount
	result.MatchedLogQuery = representative.MatchedLogQuery
	result.Matched = firingCount > 0 || pendingCount > 0

	switch {
	case firingCount > 0:
		result.State = "firing"
		result.Message = fmt.Sprintf("规则「%s」触发：%d/%d 条结果告警中", result.RuleName, firingCount, total)
		if pendingCount > 0 {
			result.Message = fmt.Sprintf("%s，%d 条预告警", result.Message, pendingCount)
		}
	case pendingCount > 0:
		result.State = "pending"
		result.Message = fmt.Sprintf("规则「%s」待触发：%d/%d 条结果处于预告警", result.RuleName, pendingCount, total)
	case recoveringCount > 0:
		result.State = "recovering"
		result.Matched = true
		result.Message = fmt.Sprintf("规则「%s」待恢复：%d 条活跃告警进入恢复等待", result.RuleName, recoveringCount)
	default:
		result.State = "inactive"
		result.Matched = false
		if recoveredCount > 0 {
			result.Message = fmt.Sprintf("规则「%s」已恢复：%d 条活跃告警已关闭", result.RuleName, recoveredCount)
		} else {
			result.Message = fmt.Sprintf("当前 %d 条结果均未满足任何告警条件", total)
		}
	}
}

func sampleResultPreferredForSummary(candidate, current *alertRuleEvaluationResult) bool {
	candidateRank := evaluationStateRank(candidate.State)
	currentRank := evaluationStateRank(current.State)
	if candidateRank != currentRank {
		return candidateRank < currentRank
	}
	candidateSeverityRank := severityRank(candidate.Severity)
	currentSeverityRank := severityRank(current.Severity)
	if candidateSeverityRank != currentSeverityRank {
		return candidateSeverityRank < currentSeverityRank
	}
	return ruleSamplePreferred(candidate.Value, severityCondition{
		Severity:  candidate.Severity,
		Condition: candidate.Condition,
		Threshold: candidate.Threshold,
		Matched:   candidate.Matched,
	}, current.Value, severityCondition{
		Severity:  current.Severity,
		Condition: current.Condition,
		Threshold: current.Threshold,
		Matched:   current.Matched,
	})
}

func evaluationStateRank(state string) int {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "firing":
		return 0
	case "pending":
		return 1
	default:
		return 2
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneMatchedLogEntries(values []matchedLogEntry) []matchedLogEntry {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]matchedLogEntry, len(values))
	for index, value := range values {
		cloned[index] = value
		cloned[index].Labels = cloneStringMap(value.Labels)
	}
	return cloned
}

func (h *DataSourceHandler) EvaluateAlertRule(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var rule model.AlertRule
	if err := h.db.First(&rule, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "告警规则不存在"})
		return
	}

	result, err := h.evaluateAlertRule(c.Request.Context(), &rule, true)
	if err != nil {
		c.JSON(200, gin.H{"code": 0, "message": "评估失败", "error": err.Error(), "data": result})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "评估完成", "data": result})
}

func (h *DataSourceHandler) EvaluateDueAlertRules(ctx context.Context) {
	startedAt := time.Now().In(time.Local)
	RecordMonitorSchedulerStarted(startedAt)
	defer func() {
		monitorSchedulerStatus.LastFinishedAt.Store(time.Now().In(time.Local))
		monitorSchedulerStatus.LastDurationMS.Store(time.Since(startedAt).Milliseconds())
	}()

	now := time.Now().In(time.Local)
	var totalRules int64
	if err := h.db.Model(&model.AlertRule{}).Where("enabled = ?", true).Count(&totalRules).Error; err != nil {
		monitorSchedulerStatus.LastError.Store(err.Error())
		return
	}
	var rules []model.AlertRule
	futureSkew := now.Add(1 * time.Minute)
	if err := h.db.Where("enabled = ?", true).
		Where("(last_eval_at IS NULL OR last_eval_at > ? OR TIMESTAMPDIFF(SECOND, last_eval_at, ?) >= evaluate_interval)", futureSkew, now).
		Order("COALESCE(last_eval_at, '1970-01-01 00:00:00') ASC, id ASC").
		Limit(alertRuleEvaluationBatchSize).
		Find(&rules).Error; err != nil {
		monitorSchedulerStatus.LastError.Store(err.Error())
		return
	}
	monitorSchedulerStatus.LastRuleTotal.Store(totalRules)
	monitorSchedulerStatus.LastRuleDue.Store(int64(len(rules)))
	monitorSchedulerStatus.LastRuleClaimed.Store(0)
	monitorSchedulerStatus.LastRuleEvaluated.Store(0)
	monitorSchedulerStatus.LastRuleSkipped.Store(0)
	monitorSchedulerStatus.LastRuleFailed.Store(0)
	monitorSchedulerStatus.LastError.Store("")

	sem := make(chan struct{}, alertRuleEvaluationConcurrency)
	var wg sync.WaitGroup
	for i := range rules {
		rule := &rules[i]
		if !h.claimDueAlertRule(rule, now) {
			monitorSchedulerStatus.LastRuleSkipped.Add(1)
			continue
		}
		monitorSchedulerStatus.LastRuleClaimed.Add(1)
		select {
		case <-ctx.Done():
			monitorSchedulerStatus.LastError.Store(ctx.Err().Error())
			waitAlertRuleEvaluationWorkers(&wg)
			return
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(rule model.AlertRule) {
			defer wg.Done()
			defer func() {
				<-sem
				if recovered := recover(); recovered != nil {
					monitorSchedulerStatus.LastRuleFailed.Add(1)
					h.markAlertRuleEvaluationPanic(rule.ID, fmt.Errorf("%v", recovered))
				}
			}()
			ruleCtx, cancel := context.WithTimeout(ctx, alertRuleEvaluationTimeout)
			defer cancel()
			if _, err := h.evaluateAlertRule(ruleCtx, &rule, false); err != nil {
				monitorSchedulerStatus.LastRuleFailed.Add(1)
				monitorSchedulerStatus.LastError.Store(err.Error())
				return
			}
			monitorSchedulerStatus.LastRuleEvaluated.Add(1)
		}(*rule)
	}
	waitAlertRuleEvaluationWorkers(&wg)
}

func waitAlertRuleEvaluationWorkers(wg *sync.WaitGroup) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
	}
}

func RecordMonitorSchedulerStarted(t time.Time) {
	if _, ok := monitorSchedulerStatus.StartedAt.Load().(time.Time); !ok {
		monitorSchedulerStatus.StartedAt.Store(t)
	}
	monitorSchedulerStatus.LastTickAt.Store(t)
}

func RecordMonitorProbeSchedulerStarted(t time.Time) {
	monitorSchedulerStatus.LastProbeTickAt.Store(t)
}

func RecordMonitorProbeSchedulerFinished(startedAt time.Time, total, started int, err error) {
	monitorSchedulerStatus.LastProbeFinishedAt.Store(time.Now())
	monitorSchedulerStatus.LastProbeDurationMS.Store(time.Since(startedAt).Milliseconds())
	monitorSchedulerStatus.LastProbeTaskTotal.Store(int64(total))
	monitorSchedulerStatus.LastProbeTaskStarted.Store(int64(started))
	if err != nil {
		monitorSchedulerStatus.LastProbeError.Store(err.Error())
	} else {
		monitorSchedulerStatus.LastProbeError.Store("")
	}
}

func SetMonitorSchedulerLeaderStatus(instanceID, leaderID string, isLeader bool, err error) {
	monitorSchedulerStatus.InstanceID.Store(instanceID)
	monitorSchedulerStatus.LeaderID.Store(leaderID)
	monitorSchedulerStatus.IsLeader.Store(isLeader)
	monitorSchedulerStatus.LeaderUpdatedAt.Store(time.Now())
	if leaderID == "local-fallback" {
		monitorSchedulerStatus.SchedulerMode.Store("local-fallback")
	} else {
		monitorSchedulerStatus.SchedulerMode.Store("redis-leader")
	}
	if err != nil {
		monitorSchedulerStatus.LeaderError.Store(err.Error())
	} else {
		monitorSchedulerStatus.LeaderError.Store("")
	}
}

func (h *DataSourceHandler) GetSchedulerStatus(c *gin.Context) {
	var dbNow string
	_ = h.db.Raw("SELECT DATE_FORMAT(NOW(), '%Y-%m-%d %H:%i:%s')").Scan(&dbNow).Error
	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"instanceId":           schedulerStringValue(monitorSchedulerStatus.InstanceID.Load()),
			"currentLeader":        schedulerStringValue(monitorSchedulerStatus.LeaderID.Load()),
			"isLeader":             monitorSchedulerStatus.IsLeader.Load(),
			"schedulerMode":        schedulerStringValue(monitorSchedulerStatus.SchedulerMode.Load()),
			"leaderUpdatedAt":      schedulerTimeValue(monitorSchedulerStatus.LeaderUpdatedAt.Load()),
			"leaderError":          schedulerStringValue(monitorSchedulerStatus.LeaderError.Load()),
			"startedAt":            schedulerTimeValue(monitorSchedulerStatus.StartedAt.Load()),
			"lastTickAt":           schedulerTimeValue(monitorSchedulerStatus.LastTickAt.Load()),
			"lastFinishedAt":       schedulerTimeValue(monitorSchedulerStatus.LastFinishedAt.Load()),
			"lastDurationMs":       monitorSchedulerStatus.LastDurationMS.Load(),
			"lastRuleTotal":        monitorSchedulerStatus.LastRuleTotal.Load(),
			"lastRuleDue":          monitorSchedulerStatus.LastRuleDue.Load(),
			"lastRuleClaimed":      monitorSchedulerStatus.LastRuleClaimed.Load(),
			"lastRuleEvaluated":    monitorSchedulerStatus.LastRuleEvaluated.Load(),
			"lastRuleSkipped":      monitorSchedulerStatus.LastRuleSkipped.Load(),
			"lastRuleFailed":       monitorSchedulerStatus.LastRuleFailed.Load(),
			"lastError":            schedulerStringValue(monitorSchedulerStatus.LastError.Load()),
			"lastProbeTickAt":      schedulerTimeValue(monitorSchedulerStatus.LastProbeTickAt.Load()),
			"lastProbeFinishedAt":  schedulerTimeValue(monitorSchedulerStatus.LastProbeFinishedAt.Load()),
			"lastProbeDurationMs":  monitorSchedulerStatus.LastProbeDurationMS.Load(),
			"lastProbeTaskTotal":   monitorSchedulerStatus.LastProbeTaskTotal.Load(),
			"lastProbeTaskStarted": monitorSchedulerStatus.LastProbeTaskStarted.Load(),
			"lastProbeError":       schedulerStringValue(monitorSchedulerStatus.LastProbeError.Load()),
			"serverNow":            time.Now().Format("2006-01-02 15:04:05"),
			"serverLocation":       time.Local.String(),
			"databaseNow":          dbNow,
		},
	})
}

func schedulerTimeValue(raw interface{}) string {
	if value, ok := raw.(time.Time); ok && !value.IsZero() {
		return value.Format("2006-01-02 15:04:05")
	}
	return ""
}

func schedulerStringValue(raw interface{}) string {
	if value, ok := raw.(string); ok {
		return value
	}
	return ""
}

func (h *DataSourceHandler) claimDueAlertRule(rule *model.AlertRule, now time.Time) bool {
	if h == nil || h.db == nil || rule == nil || rule.ID == 0 {
		return false
	}
	interval := positiveInt(rule.EvaluateInterval, 60)
	futureSkew := now.Add(1 * time.Minute)
	query := h.db.Model(&model.AlertRule{}).
		Where("id = ? AND enabled = ?", rule.ID, true).
		Where("(last_eval_at IS NULL OR last_eval_at > ? OR TIMESTAMPDIFF(SECOND, last_eval_at, ?) >= ?)", futureSkew, now, interval).
		Update("last_eval_at", now)
	if query.Error != nil || query.RowsAffected == 0 {
		return false
	}
	rule.LastEvalAt = &now
	return true
}

func (h *DataSourceHandler) markAlertRuleEvaluationPanic(ruleID uint, err error) {
	if h == nil || h.db == nil || ruleID == 0 || err == nil {
		return
	}
	now := time.Now().In(time.Local)
	message := fmt.Sprintf("后台评估异常：%s", err.Error())
	_ = h.updateAlertRuleRuntimeFields(ruleID, map[string]interface{}{
		"last_state":   "error",
		"last_eval_at": &now,
		"last_error":   message,
	})
}

func (h *DataSourceHandler) ListAlertEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 500 {
		pageSize = 500
	}

	activeScope := strings.TrimSpace(c.Query("scope")) == "active"
	if activeScope {
		if err := h.ensurePendingAlertEvents(c.Request.Context(), strings.TrimSpace(c.Query("faultCenterId"))); err != nil {
			c.JSON(500, gin.H{"code": 500, "message": "同步预告警事件失败", "error": err.Error()})
			return
		}
	}

	query := h.db.Model(&model.AlertEvent{})
	if ruleID := strings.TrimSpace(c.Query("ruleId")); ruleID != "" {
		query = query.Where("rule_id = ?", ruleID)
	}
	if ruleGroupID := strings.TrimSpace(c.Query("ruleGroupId")); ruleGroupID != "" {
		query = query.Where("rule_group_id = ?", ruleGroupID)
	}
	if faultCenterID := strings.TrimSpace(c.Query("faultCenterId")); faultCenterID != "" {
		query = query.Where("fault_center_id = ?", faultCenterID)
	}
	if dataSourceType := strings.TrimSpace(c.Query("dataSourceType")); dataSourceType != "" {
		query = query.Where("data_source_type = ?", dataSourceType)
	}
	if state := strings.TrimSpace(c.Query("state")); state != "" {
		query = query.Where("state IN ?", alertEventStateValues(state))
	}
	if severity := strings.TrimSpace(c.Query("severity")); severity != "" {
		query = query.Where("severity IN ?", alertSeverityValues(severity))
	}
	if startDate := strings.TrimSpace(c.Query("startDate")); startDate != "" {
		if t, err := parseAlertEventDate(startDate, false); err == nil {
			query = query.Where("started_at >= ?", t)
		}
	}
	if endDate := strings.TrimSpace(c.Query("endDate")); endDate != "" {
		if t, err := parseAlertEventDate(endDate, true); err == nil {
			query = query.Where("started_at <= ?", t)
		}
	}
	if scope := strings.TrimSpace(c.Query("scope")); scope == "active" {
		query = query.Where("state IN ? AND ended_at IS NULL", activeAlertEventStates())
	} else if scope == "history" {
		query = query.Where("state = ? OR ended_at IS NOT NULL", "recovered")
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("rule_name LIKE ? OR message LIKE ? OR labels LIKE ? OR annotations LIKE ?", like, like, like, like)
	}

	var total int64
	query.Count(&total)

	var events []model.AlertEvent
	if err := query.Order(alertEventListOrder(strings.TrimSpace(c.Query("scope")), strings.TrimSpace(c.Query("sort")))).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&events).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取告警事件失败", "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":     events,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

func alertEventListOrder(scope, sortBy string) string {
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "last_eval_at", "last_eval":
		return "last_eval_at DESC, id DESC"
	case "ended_at", "ended":
		return "COALESCE(ended_at, last_eval_at) DESC, id DESC"
	case "started_at", "started":
		return "started_at DESC, id DESC"
	}

	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "active":
		return "started_at DESC, id DESC"
	case "history":
		return "COALESCE(ended_at, last_eval_at) DESC, id DESC"
	default:
		return "started_at DESC, id DESC"
	}
}

func (h *DataSourceHandler) GetAlertEvent(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var event model.AlertEvent
	if err := h.db.First(&event, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "告警事件不存在"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": event})
}

func parseAlertEventDate(value string, endOfDay bool) (time.Time, error) {
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		if endOfDay {
			return t.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
		}
		return t, nil
	}
	return time.Parse(time.RFC3339, value)
}

func activeAlertEventStates() []string {
	return []string{"pending", "firing", "processing", "silenced", "recovering", "error"}
}

func shouldSendRecoveryNotification(event *model.AlertEvent) bool {
	if event == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(event.State)) {
	case "firing", "processing", "recovering":
	default:
		return false
	}
	switch strings.ToLower(strings.TrimSpace(event.NotifyStatus)) {
	case "success", "partial":
		return true
	default:
		return false
	}
}

func recoveryWaitRemainingForDuration(event *model.AlertEvent, now time.Time, wait time.Duration) time.Duration {
	if event == nil || wait <= 0 {
		return 0
	}
	if !strings.EqualFold(strings.TrimSpace(event.State), "recovering") {
		return wait
	}
	recoveryStartedAt := event.LastEvalAt
	if recoveryStartedAt.IsZero() {
		return wait
	}
	elapsed := now.Sub(recoveryStartedAt)
	if elapsed >= wait {
		return 0
	}
	return wait - elapsed
}

func (h *DataSourceHandler) effectiveRecoveryWait(rule *model.AlertRule) time.Duration {
	if h == nil || h.db == nil || rule == nil || rule.FaultCenterID == 0 {
		return 0
	}
	var center model.FaultCenter
	if err := h.db.Select("id, recover_wait_seconds").First(&center, rule.FaultCenterID).Error; err != nil {
		return 0
	}
	wait := center.RecoverWaitSeconds
	if wait <= 0 {
		wait = 30
	}
	return time.Duration(wait) * time.Second
}

func (h *DataSourceHandler) faultCenterAllowsRecoveryNotification(rule *model.AlertRule) bool {
	if h == nil || h.db == nil || rule == nil || rule.FaultCenterID == 0 {
		return true
	}
	var center model.FaultCenter
	if err := h.db.Select("id, recover_notify").First(&center, rule.FaultCenterID).Error; err != nil {
		return true
	}
	return center.RecoverNotify
}

func isActiveAlertRuleState(state string) bool {
	state = strings.ToLower(strings.TrimSpace(state))
	for _, item := range activeAlertEventStates() {
		if state == item {
			return true
		}
	}
	return false
}

func alertEventStateValues(state string) []string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "pending":
		return []string{"pending"}
	case "firing":
		return []string{"firing", "error"}
	case "processing":
		return []string{"processing"}
	case "silenced":
		return []string{"silenced"}
	case "recovering":
		return []string{"recovering"}
	case "error":
		return []string{"error"}
	case "active":
		return activeAlertEventStates()
	default:
		return []string{state}
	}
}

func alertSeverityValues(severity string) []string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "p0":
		return []string{"p0", "critical"}
	case "p1":
		return []string{"p1", "warning"}
	case "p2":
		return []string{"p2", "info"}
	default:
		return []string{severity}
	}
}

func (h *DataSourceHandler) GetAlertEventStats(c *gin.Context) {
	var stats struct {
		TotalRules       int64 `json:"totalRules"`
		EnabledRules     int64 `json:"enabledRules"`
		FiringRules      int64 `json:"firingRules"`
		PendingRules     int64 `json:"pendingRules"`
		TodayEvents      int64 `json:"todayEvents"`
		UnresolvedEvents int64 `json:"unresolvedEvents"`
	}

	h.db.Model(&model.AlertRule{}).Count(&stats.TotalRules)
	h.db.Model(&model.AlertRule{}).Where("enabled = ?", true).Count(&stats.EnabledRules)
	h.db.Model(&model.AlertRule{}).Where("last_state = ?", "firing").Count(&stats.FiringRules)
	h.db.Model(&model.AlertEvent{}).Distinct("rule_id").Where("state = ? AND ended_at IS NULL", "pending").Count(&stats.PendingRules)
	h.db.Model(&model.AlertEvent{}).Where("DATE(last_eval_at) = CURDATE()").Count(&stats.TodayEvents)
	h.db.Model(&model.AlertEvent{}).Where("state IN ? AND ended_at IS NULL", activeAlertEventStates()).Count(&stats.UnresolvedEvents)

	c.JSON(200, gin.H{"code": 0, "message": "success", "data": stats})
}

func (h *DataSourceHandler) GetAlertEventTrend(c *gin.Context) {
	now := time.Now().In(time.Local)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -6)
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), time.Local)

	if startDate := strings.TrimSpace(c.Query("startDate")); startDate != "" {
		parsed, err := parseAlertEventDate(startDate, false)
		if err != nil {
			c.JSON(400, gin.H{"code": 400, "message": "开始时间格式不正确"})
			return
		}
		start = parsed
	}
	if endDate := strings.TrimSpace(c.Query("endDate")); endDate != "" {
		parsed, err := parseAlertEventDate(endDate, true)
		if err != nil {
			c.JSON(400, gin.H{"code": 400, "message": "结束时间格式不正确"})
			return
		}
		end = parsed
	}
	if start.After(end) {
		c.JSON(400, gin.H{"code": 400, "message": "开始时间不能晚于结束时间"})
		return
	}

	type trendRow struct {
		Date     string `json:"date"`
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}

	dateExpr := alertEventTrendDateExpr(h.db)
	var rows []trendRow
	query := h.db.Model(&model.AlertEvent{}).
		Select(fmt.Sprintf("%s AS date, severity, COUNT(*) AS count", dateExpr)).
		Where("started_at >= ? AND started_at <= ?", start, end)

	if ruleID := strings.TrimSpace(c.Query("ruleId")); ruleID != "" {
		query = query.Where("rule_id = ?", ruleID)
	}
	if ruleGroupID := strings.TrimSpace(c.Query("ruleGroupId")); ruleGroupID != "" {
		query = query.Where("rule_group_id = ?", ruleGroupID)
	}
	if faultCenterID := strings.TrimSpace(c.Query("faultCenterId")); faultCenterID != "" {
		query = query.Where("fault_center_id = ?", faultCenterID)
	}
	if dataSourceType := strings.TrimSpace(c.Query("dataSourceType")); dataSourceType != "" {
		query = query.Where("data_source_type = ?", dataSourceType)
	}
	if state := strings.TrimSpace(c.Query("state")); state != "" {
		query = query.Where("state IN ?", alertEventStateValues(state))
	}
	if severity := strings.TrimSpace(c.Query("severity")); severity != "" {
		query = query.Where("severity IN ?", alertSeverityValues(severity))
	}

	if err := query.Group(fmt.Sprintf("%s, severity", dateExpr)).
		Order("date ASC").
		Scan(&rows).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取告警趋势失败", "error": err.Error()})
		return
	}

	points := make([]trendRow, 0, len(rows))
	for _, row := range rows {
		points = append(points, trendRow{
			Date:     strings.TrimSpace(row.Date),
			Severity: normalizeSeverityLevel(row.Severity),
			Count:    row.Count,
		})
	}

	c.JSON(200, gin.H{"code": 0, "message": "success", "data": points})
}

func alertEventTrendDateExpr(db *gorm.DB) string {
	if db == nil || db.Dialector == nil {
		return "DATE(started_at)"
	}
	switch db.Dialector.Name() {
	case "mysql":
		return "DATE_FORMAT(started_at, '%Y-%m-%d')"
	case "sqlite":
		return "strftime('%Y-%m-%d', started_at)"
	case "postgres":
		return "TO_CHAR(started_at, 'YYYY-MM-DD')"
	default:
		return "DATE(started_at)"
	}
}

func (h *DataSourceHandler) normalizeAlertRule(rule *model.AlertRule) error {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Query = strings.TrimSpace(rule.Query)
	rule.QueryMode = strings.ToLower(strings.TrimSpace(rule.QueryMode))
	rule.Index = strings.Trim(rule.Index, "/ ")
	rule.Condition = strings.ToLower(strings.TrimSpace(rule.Condition))
	rule.Severity = strings.ToLower(strings.TrimSpace(rule.Severity))
	rule.DataSourceIDs = strings.TrimSpace(rule.DataSourceIDs)
	rule.SeverityRules = strings.TrimSpace(rule.SeverityRules)
	rule.ChannelIDs = strings.TrimSpace(rule.ChannelIDs)
	rule.Labels = strings.TrimSpace(rule.Labels)
	rule.Annotations = strings.TrimSpace(rule.Annotations)
	rule.DetailTemplate = strings.TrimSpace(rule.DetailTemplate)
	rule.CallbackQueries = strings.TrimSpace(rule.CallbackQueries)
	rule.EffectiveTime = strings.TrimSpace(rule.EffectiveTime)

	if rule.Name == "" {
		return fmt.Errorf("请输入规则名称")
	}
	if rule.RuleGroupID == 0 {
		rule.RuleGroupID = findDefaultRuleGroup(h.db)
	}
	if rule.FaultCenterID == 0 {
		rule.FaultCenterID = findDefaultFaultCenter(h.db)
	}
	dataSourceIDs, err := parseRuleDataSourceIDs(rule.DataSourceIDs)
	if err != nil {
		return err
	}
	if rule.DataSourceID == 0 && len(dataSourceIDs) > 0 {
		rule.DataSourceID = dataSourceIDs[0]
	}
	if rule.DataSourceID == 0 {
		return fmt.Errorf("请选择数据源")
	}
	if len(dataSourceIDs) == 0 {
		dataSourceIDs = []uint{rule.DataSourceID}
	}
	if encoded, err := json.Marshal(dataSourceIDs); err == nil {
		rule.DataSourceIDs = string(encoded)
	}
	if rule.Query == "" {
		return fmt.Errorf("请输入查询语句")
	}

	var ds model.DataSource
	if err := h.db.First(&ds, rule.DataSourceID).Error; err != nil {
		return fmt.Errorf("数据源不存在")
	}
	rule.DataSourceType = ds.Type

	if rule.QueryMode == "" {
		rule.QueryMode = "instant"
	}
	switch rule.QueryMode {
	case "instant", "range":
	default:
		return fmt.Errorf("不支持的查询模式: %s", rule.QueryMode)
	}
	if rule.Condition == "" {
		rule.Condition = "gt"
	}
	switch rule.Condition {
	case "gt", "gte", "lt", "lte", "eq", "neq":
	default:
		return fmt.Errorf("不支持的判断条件: %s", rule.Condition)
	}
	if rule.ForSeconds <= 0 {
		rule.ForSeconds = 60
	}
	if rule.EvaluateInterval <= 0 {
		rule.EvaluateInterval = 60
	}
	if rule.Severity == "" {
		rule.Severity = "warning"
	}
	switch rule.Severity {
	case "info", "warning", "critical", "p0", "p1", "p2":
	default:
		return fmt.Errorf("不支持的告警等级: %s", rule.Severity)
	}
	if rule.SeverityRules != "" {
		if _, err := parseSeverityConditions(rule); err != nil {
			return err
		}
	}
	if rule.RepeatInterval <= 0 {
		rule.RepeatInterval = 3600
	}
	if rule.ChannelIDs != "" {
		if _, err := parseRuleChannelIDs(rule.ChannelIDs); err != nil {
			return err
		}
	}
	for name, raw := range map[string]string{
		"标签":   rule.Labels,
		"事件注解": rule.Annotations,
		"回调查询": rule.CallbackQueries,
		"生效时间": rule.EffectiveTime,
	} {
		if raw != "" && !json.Valid([]byte(raw)) {
			return fmt.Errorf("%s必须是合法 JSON", name)
		}
	}
	if rule.DetailTemplate == "" {
		rule.DetailTemplate = legacyDetailTemplateFromAnnotations(rule.Annotations)
	}
	if rule.LastState == "" {
		rule.LastState = "inactive"
	}
	return nil
}

func legacyDetailTemplateFromAnnotations(raw string) string {
	annotations := parseStringMap(raw)
	for _, key := range []string{"description", "detail", "summary", "message"} {
		if value := strings.TrimSpace(annotations[key]); value != "" {
			return value
		}
	}
	return ""
}

func (h *DataSourceHandler) evaluateAlertRule(ctx context.Context, rule *model.AlertRule, includeRawResult bool) (*alertRuleEvaluationResult, error) {
	previousState := rule.LastState
	if previousState == "" {
		previousState = "inactive"
	}

	var ds model.DataSource
	if err := h.db.First(&ds, rule.DataSourceID).Error; err != nil {
		return nil, fmt.Errorf("数据源不存在")
	}
	if !ds.Enabled {
		return nil, fmt.Errorf("数据源已禁用")
	}

	req := dataSourceQueryRequest{
		Query:     rule.Query,
		QueryMode: rule.QueryMode,
		Index:     rule.Index,
	}
	rawResult, statusCode, err := h.queryDataSource(ctx, &ds, req)
	now := time.Now()

	result := &alertRuleEvaluationResult{
		RuleID:         rule.ID,
		RuleName:       rule.Name,
		RuleGroupID:    rule.RuleGroupID,
		FaultCenterID:  rule.FaultCenterID,
		DataSourceID:   ds.ID,
		DataSourceName: ds.Name,
		DataSourceType: ds.Type,
		Severity:       rule.Severity,
		PreviousState:  previousState,
		Condition:      rule.Condition,
		Threshold:      rule.Threshold,
		ForSeconds:     rule.ForSeconds,
		StatusCode:     statusCode,
		EvaluatedAt:    now,
	}
	if includeRawResult {
		result.QueryResult = rawResult
	}

	if err != nil {
		message := fmt.Sprintf("规则「%s」评估失败：%s", rule.Name, alertRuleErrorMessage(err))
		result.State = "error"
		result.Message = message
		rule.LastState = "error"
		rule.LastEvalAt = &now
		rule.LastError = err.Error()
		_ = h.updateAlertRuleRuntimeFields(rule.ID, map[string]interface{}{
			"last_state":   rule.LastState,
			"last_eval_at": rule.LastEvalAt,
			"last_error":   rule.LastError,
		})
		if previousState != "error" {
			if isActiveAlertRuleState(previousState) {
				_, _ = h.recoverActiveAlertEvents(rule, now, fmt.Sprintf("规则「%s」评估失败，原活跃告警自动关闭", rule.Name))
			}
			_ = h.createAlertEvent(rule, &ds, result, "error", message)
		}
		return result, err
	}

	samples, err := extractRuleEvaluationSamples(ds.Type, rawResult)
	if err != nil {
		message := fmt.Sprintf("规则「%s」无法从查询结果中提取数值：%s", rule.Name, alertRuleErrorMessage(err))
		result.State = "error"
		result.Message = message
		rule.LastState = "error"
		rule.LastEvalAt = &now
		rule.LastError = err.Error()
		_ = h.updateAlertRuleRuntimeFields(rule.ID, map[string]interface{}{
			"last_state":   rule.LastState,
			"last_eval_at": rule.LastEvalAt,
			"last_error":   rule.LastError,
		})
		if previousState != "error" {
			if previousState == "firing" {
				_, _ = h.recoverActiveAlertEvents(rule, now, fmt.Sprintf("规则「%s」评估失败，原活跃告警自动关闭", rule.Name))
			}
			_ = h.createAlertEvent(rule, &ds, result, "error", message)
		}
		return result, err
	}

	if previousState == "error" {
		_, _ = h.recoverActiveAlertEventsWithResult(rule, result, now, fmt.Sprintf("规则「%s」评估恢复，错误状态自动关闭", rule.Name))
	}

	seenFingerprints := map[string]struct{}{}
	firingCount := 0
	pendingCount := 0
	recoveringCount := 0
	recoveredCount := 0
	var representative *alertRuleEvaluationResult
	var earliestPending *time.Time
	var earliestFiring *time.Time
	var firingNotifyEvents []*model.AlertEvent
	var recoveryNotifyEvents []*model.AlertEvent
	var escalationCandidates []*model.AlertEvent
	allowRecoveryNotification := rule.NotifyRecovery && h.faultCenterAllowsRecoveryNotification(rule)
	recoveryWait := time.Duration(0)
	if allowRecoveryNotification {
		recoveryWait = h.effectiveRecoveryWait(rule)
	}

	for _, sample := range samples {
		condition := selectSeverityCondition(rule, sample.Value)
		sampleResult := result.copyForSample(sample, condition)
		sampleResult.Fingerprint = buildAlertFingerprint(rule.ID, rule.Name, sampleResult.Severity, buildRuleLabelMap(rule, sampleResult))
		seenFingerprints[sampleResult.Fingerprint] = struct{}{}

		activeEvent, err := h.findActiveAlertEventByFingerprint(rule, sampleResult.Fingerprint, activeAlertEventStates())
		if err != nil {
			return result, err
		}

		state := "inactive"
		message := fmt.Sprintf("当前值 %s，未满足任何告警条件", formatRuleValue(sample.Value))
		if condition.Matched {
			if ds.Type == "loki" {
				h.enrichLokiMatchedLogs(ctx, &ds, rule, sampleResult, now)
			}
			startedAt := now
			if activeEvent != nil && !activeEvent.StartedAt.IsZero() {
				startedAt = activeEvent.StartedAt
			}
			sampleResult.StartedAt = &startedAt
			if now.Sub(startedAt) >= time.Duration(condition.ForSeconds)*time.Second {
				state = "firing"
				message = fmt.Sprintf("规则「%s」触发：当前值 %s %s %s", rule.Name, formatRuleValue(sample.Value), getConditionText(condition.Condition), formatRuleValue(condition.Threshold))
				firingCount++
				if earliestFiring == nil || startedAt.Before(*earliestFiring) {
					earliestFiring = &startedAt
				}
			} else {
				state = "pending"
				remaining := time.Duration(condition.ForSeconds)*time.Second - now.Sub(startedAt)
				message = fmt.Sprintf("规则「%s」待触发：当前值 %s %s %s，还需持续约 %ds", rule.Name, formatRuleValue(sample.Value), getConditionText(condition.Condition), formatRuleValue(condition.Threshold), int(math.Ceil(remaining.Seconds())))
				pendingCount++
				if earliestPending == nil || startedAt.Before(*earliestPending) {
					earliestPending = &startedAt
				}
			}
			sampleResult.State = state
			sampleResult.Message = message
			event, err := h.upsertActiveAlertEvent(rule, &ds, sampleResult, state, message)
			if err != nil {
				return result, err
			}
			result.Samples = append(result.Samples, sampleResult.toSampleResult())
			if representative == nil || sampleResultPreferredForSummary(sampleResult, representative) {
				representative = sampleResult
			}
			if state == "firing" {
				if h.applySilenceRulesToEvent(event) {
					_ = h.db.Save(event).Error
					continue
				}
				escalationCandidates = append(escalationCandidates, event)
				shouldNotify := activeEvent == nil || activeEvent.State != "firing" || h.shouldRepeatRuleNotification(rule, sampleResult.Severity, now)
				if shouldNotify {
					firingNotifyEvents = append(firingNotifyEvents, event)
				}
			}
			continue
		}

		sampleResult.State = "inactive"
		sampleResult.Message = message
		result.Samples = append(result.Samples, sampleResult.toSampleResult())
		if representative == nil {
			representative = sampleResult
		}
		if activeEvent != nil {
			recoverMessage := fmt.Sprintf("规则「%s」已恢复：当前值 %s 不再满足告警条件", rule.Name, formatRuleValue(sample.Value))
			recovery, err := h.handleRecoveredAlertEvent(rule, activeEvent, sampleResult, now, recoverMessage, recoveryWait)
			if err != nil {
				return result, err
			}
			if len(recovery.recovered) > 0 {
				recoveredCount++
			}
			if len(recovery.recovering) > 0 {
				recoveringCount++
			}
			if allowRecoveryNotification && len(recovery.notify) > 0 {
				recoveryNotifyEvents = append(recoveryNotifyEvents, recovery.notify...)
			}
		}
	}

	staleRecovery, err := h.recoverMissingActiveAlertEvents(rule, seenFingerprints, now, recoveryWait)
	if err != nil {
		return result, err
	}
	recoveredCount += len(staleRecovery.recovered)
	recoveringCount += len(staleRecovery.recovering)
	if allowRecoveryNotification {
		recoveryNotifyEvents = append(recoveryNotifyEvents, staleRecovery.notify...)
	}

	if h.sendAndRecordRuleNotifications(ctx, rule, firingNotifyEvents) {
		rule.LastNotifyAt = &now
	}
	if h.sendAndRecordRuleNotifications(ctx, rule, recoveryNotifyEvents) {
		rule.LastNotifyAt = &now
	}
	h.processFaultCenterEscalations(ctx, rule, escalationCandidates, now)

	if representative == nil {
		recoveryResult := result.copyForSample(ruleEvaluationSample{}, severityCondition{Severity: normalizeSeverity(rule.Severity), Condition: normalizeCondition(rule.Condition), Threshold: rule.Threshold, ForSeconds: positiveInt(rule.ForSeconds, 60)})
		representative = recoveryResult
	}
	applyEvaluationSummary(result, representative, firingCount, pendingCount, recoveringCount, recoveredCount, len(samples))

	rule.LastState = result.State
	rule.LastValue = result.Value
	rule.LastEvalAt = &now
	rule.LastError = ""
	rule.PendingSince = earliestPending
	rule.FiringSince = earliestFiring

	updates := map[string]interface{}{
		"last_state":    rule.LastState,
		"last_value":    rule.LastValue,
		"last_eval_at":  rule.LastEvalAt,
		"last_error":    rule.LastError,
		"pending_since": rule.PendingSince,
		"firing_since":  rule.FiringSince,
	}
	if rule.LastNotifyAt != nil {
		updates["last_notify_at"] = rule.LastNotifyAt
	}
	if err := h.updateAlertRuleRuntimeFields(rule.ID, updates); err != nil {
		return result, err
	}
	return result, nil
}

func (h *DataSourceHandler) updateAlertRuleRuntimeFields(ruleID uint, updates map[string]interface{}) error {
	if ruleID == 0 || len(updates) == 0 {
		return nil
	}
	return h.db.Model(&model.AlertRule{}).Where("id = ?", ruleID).Updates(updates).Error
}

func (h *DataSourceHandler) enrichLokiMatchedLogs(ctx context.Context, ds *model.DataSource, rule *model.AlertRule, result *alertRuleEvaluationResult, now time.Time) {
	if ds == nil || rule == nil || result == nil || ds.Type != "loki" {
		return
	}
	if result.MatchedLogCount > 0 || len(result.MatchedLogs) > 0 {
		if strings.TrimSpace(result.MatchedLogQuery) == "" {
			result.MatchedLogQuery = deriveLokiMatchedLogQuery(rule.Query)
		}
		return
	}
	logQuery := deriveLokiMatchedLogQuery(rule.Query)
	if strings.TrimSpace(logQuery) == "" {
		return
	}
	result.MatchedLogQuery = logQuery
	windowSeconds := lokiMatchedLogLookbackSeconds(rule)
	end := now
	start := end.Add(-time.Duration(windowSeconds) * time.Second)
	raw, _, err := h.queryDataSource(ctx, ds, dataSourceQueryRequest{
		Query:     logQuery,
		QueryMode: "range",
		Start:     start.Format(time.RFC3339Nano),
		End:       end.Format(time.RFC3339Nano),
		Limit:     maxMatchedLogLinesPerEvent * 3,
	})
	if err != nil {
		return
	}
	samples, err := extractRuleEvaluationSamples("loki", raw)
	if err != nil {
		return
	}
	logs, count := matchedLogsForSampleLabels(samples, result.Labels, maxMatchedLogLinesPerEvent)
	if len(logs) == 0 && count == 0 {
		return
	}
	result.MatchedLogs = logs
	result.MatchedLogCount = count
	result.MatchedLogQuery = logQuery
}

func (h *DataSourceHandler) ensureLokiMatchedLogsForNotificationEvents(ctx context.Context, events []*model.AlertEvent) {
	rules := map[uint]*model.AlertRule{}
	dataSources := map[uint]*model.DataSource{}
	for _, event := range events {
		if event == nil || !strings.EqualFold(strings.TrimSpace(event.DataSourceType), "loki") {
			continue
		}
		annotations := parseStringMap(event.Annotations)
		if strings.TrimSpace(firstNonEmpty(annotations["matched_logs"], annotations["matchedLogs"])) != "" {
			continue
		}
		if event.RuleID == 0 || event.DataSourceID == 0 {
			continue
		}

		rule := rules[event.RuleID]
		if rule == nil {
			var item model.AlertRule
			if err := h.db.First(&item, event.RuleID).Error; err != nil {
				continue
			}
			rule = &item
			rules[event.RuleID] = rule
		}
		ds := dataSources[event.DataSourceID]
		if ds == nil {
			var item model.DataSource
			if err := h.db.First(&item, event.DataSourceID).Error; err != nil {
				continue
			}
			ds = &item
			dataSources[event.DataSourceID] = ds
		}
		if !strings.EqualFold(strings.TrimSpace(ds.Type), "loki") {
			continue
		}

		labels := parseStringMap(event.Labels)
		evaluatedAt := event.LastEvalAt
		if evaluatedAt.IsZero() {
			evaluatedAt = time.Now()
		}
		result := &alertRuleEvaluationResult{
			RuleID:          event.RuleID,
			RuleName:        event.RuleName,
			DataSourceName:  event.DataSourceName,
			DataSourceType:  event.DataSourceType,
			Severity:        event.Severity,
			State:           event.State,
			Value:           event.Value,
			Condition:       event.Condition,
			Threshold:       event.Threshold,
			Labels:          labels,
			Fingerprint:     event.Fingerprint,
			Message:         event.Message,
			MatchedLogQuery: strings.TrimSpace(firstNonEmpty(annotations["matched_log_query"], annotations["matchedLogQuery"])),
			EvaluatedAt:     evaluatedAt,
		}
		h.enrichLokiMatchedLogs(ctx, ds, rule, result, evaluatedAt)
		if strings.TrimSpace(result.MatchedLogQuery) == "" && len(result.MatchedLogs) == 0 && result.MatchedLogCount == 0 {
			continue
		}
		event.Annotations = buildRuleAnnotations(rule, result, labels, event.Fingerprint)
		_ = h.db.Model(event).Update("annotations", event.Annotations).Error
	}
}

func matchedLogsForSampleLabels(samples []ruleEvaluationSample, labels map[string]string, limit int) ([]matchedLogEntry, int) {
	if limit <= 0 {
		limit = maxMatchedLogLinesPerEvent
	}
	logs := make([]matchedLogEntry, 0, limit)
	count := 0
	for _, sample := range samples {
		if !lokiLabelsMatch(labels, sample.Labels) {
			continue
		}
		count += matchedLogSampleCount(sample)
		for _, entry := range sample.MatchedLogs {
			if len(logs) >= limit {
				continue
			}
			logs = append(logs, entry)
		}
	}
	if count > 0 || len(logs) > 0 {
		return logs, count
	}
	for _, sample := range samples {
		count += matchedLogSampleCount(sample)
		for _, entry := range sample.MatchedLogs {
			if len(logs) >= limit {
				continue
			}
			logs = append(logs, entry)
		}
	}
	return logs, count
}

func matchedLogSampleCount(sample ruleEvaluationSample) int {
	if sample.MatchedLogCount > 0 {
		return sample.MatchedLogCount
	}
	return len(sample.MatchedLogs)
}

func lokiLabelsMatch(expected, actual map[string]string) bool {
	if len(expected) == 0 {
		return true
	}
	if len(actual) == 0 {
		return false
	}
	for key, value := range expected {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if strings.TrimSpace(actual[key]) != value {
			return false
		}
	}
	return true
}

func (h *DataSourceHandler) shouldRepeatRuleNotification(rule *model.AlertRule, severity string, now time.Time) bool {
	if !h.ruleHasNotificationTarget(rule) {
		return false
	}
	interval := h.effectiveRepeatNotificationInterval(rule, severity)
	if interval <= 0 {
		return false
	}
	if rule.LastNotifyAt == nil {
		return true
	}
	return !rule.LastNotifyAt.Add(interval).After(now)
}

func (h *DataSourceHandler) effectiveRepeatNotificationInterval(rule *model.AlertRule, severity string) time.Duration {
	if rule.FaultCenterID > 0 {
		var center model.FaultCenter
		if err := h.db.Select("id, repeat_notice_interval").First(&center, rule.FaultCenterID).Error; err == nil {
			if interval, ok := faultCenterRepeatNoticeInterval(center.RepeatNoticeInterval, severity); ok {
				return interval
			}
		}
	}
	if rule.RepeatInterval <= 0 {
		return 0
	}
	return time.Duration(rule.RepeatInterval) * time.Second
}

func faultCenterRepeatNoticeInterval(raw string, severity string) (time.Duration, bool) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &config); err != nil || len(config) == 0 {
		return 0, false
	}
	for _, key := range repeatNoticeIntervalKeys(severity) {
		minutes, ok := noticeIntervalMinutes(config[key])
		if ok {
			return time.Duration(minutes) * time.Minute, true
		}
	}
	return 0, false
}

func repeatNoticeIntervalKeys(severity string) []string {
	switch normalizeSeverityLevel(severity) {
	case "P0":
		return []string{"p0", "critical"}
	case "P1":
		return []string{"p1", "warning"}
	case "P2":
		return []string{"p2", "info"}
	default:
		key := strings.ToLower(strings.TrimSpace(severity))
		if key == "" {
			return nil
		}
		return []string{key}
	}
}

func noticeIntervalMinutes(value interface{}) (int, bool) {
	switch item := value.(type) {
	case float64:
		if item > 0 {
			return int(item), true
		}
	case int:
		if item > 0 {
			return item, true
		}
	case json.Number:
		number, err := item.Int64()
		if err == nil && number > 0 {
			return int(number), true
		}
	case string:
		number, err := strconv.Atoi(strings.TrimSpace(item))
		if err == nil && number > 0 {
			return number, true
		}
	}
	return 0, false
}

func (h *DataSourceHandler) ruleHasNotificationTarget(rule *model.AlertRule) bool {
	if ids, err := parseRuleChannelIDs(rule.ChannelIDs); err == nil && len(ids) > 0 {
		return true
	}
	if rule.FaultCenterID == 0 {
		return false
	}
	var center model.FaultCenter
	if err := h.db.First(&center, rule.FaultCenterID).Error; err != nil {
		return false
	}
	if ids, err := parseRuleChannelIDs(center.NoticeObjectIDs); err == nil && len(ids) > 0 {
		return true
	}
	if ids, err := parseRuleChannelIDs(center.NoticeChannelIDs); err == nil && len(ids) > 0 {
		return true
	}
	return noticeRoutesHaveTargets(center.NoticeRoutes)
}

func noticeRoutesHaveTargets(raw string) bool {
	var routes []struct {
		NoticeObjectIDs []interface{} `json:"noticeObjectIds"`
		Enabled         *bool         `json:"enabled"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &routes); err != nil {
		return false
	}
	for _, route := range routes {
		if route.Enabled != nil && !*route.Enabled {
			continue
		}
		if len(route.NoticeObjectIDs) > 0 {
			return true
		}
	}
	return false
}

func (h *DataSourceHandler) createAlertEvent(rule *model.AlertRule, ds *model.DataSource, result *alertRuleEvaluationResult, state, message string) *model.AlertEvent {
	startedAt := result.EvaluatedAt
	if result.StartedAt != nil && !result.StartedAt.IsZero() {
		startedAt = *result.StartedAt
	}
	var endedAt *time.Time
	if state == "recovered" {
		endedAt = &result.EvaluatedAt
	}
	labels := buildRuleLabelMap(rule, result)
	fingerprint := strings.TrimSpace(result.Fingerprint)
	if fingerprint == "" {
		fingerprint = buildAlertFingerprint(rule.ID, rule.Name, result.Severity, labels)
		result.Fingerprint = fingerprint
	}

	event := &model.AlertEvent{
		RuleID:         rule.ID,
		RuleGroupID:    rule.RuleGroupID,
		FaultCenterID:  rule.FaultCenterID,
		RuleName:       rule.Name,
		DataSourceID:   ds.ID,
		DataSourceName: ds.Name,
		DataSourceType: ds.Type,
		Severity:       result.Severity,
		State:          state,
		Value:          result.Value,
		Condition:      result.Condition,
		Threshold:      result.Threshold,
		Message:        message,
		Labels:         marshalStringMap(labels),
		Annotations:    buildRuleAnnotations(rule, result, labels, fingerprint),
		Fingerprint:    fingerprint,
		NotifyStatus:   "none",
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		LastEvalAt:     result.EvaluatedAt,
	}
	if err := h.db.Create(event).Error; err != nil {
		return nil
	}
	return event
}

func (h *DataSourceHandler) upsertActiveAlertEvent(rule *model.AlertRule, ds *model.DataSource, result *alertRuleEvaluationResult, state, message string) (*model.AlertEvent, error) {
	labels := buildRuleLabelMap(rule, result)
	fingerprint := strings.TrimSpace(result.Fingerprint)
	if fingerprint == "" {
		fingerprint = buildAlertFingerprint(rule.ID, rule.Name, result.Severity, labels)
		result.Fingerprint = fingerprint
	}
	event, err := h.findActiveAlertEventByFingerprint(rule, fingerprint, activeAlertEventStates())
	if err != nil {
		return nil, err
	}
	if event == nil {
		event = h.createAlertEvent(rule, ds, result, state, message)
		if event == nil {
			return nil, fmt.Errorf("创建告警事件失败")
		}
		return event, nil
	}
	event.RuleGroupID = rule.RuleGroupID
	event.FaultCenterID = rule.FaultCenterID
	event.RuleName = rule.Name
	event.DataSourceID = ds.ID
	event.DataSourceName = ds.Name
	event.DataSourceType = ds.Type
	event.Severity = result.Severity
	event.State = state
	event.Value = result.Value
	event.Condition = result.Condition
	event.Threshold = result.Threshold
	event.Message = message
	event.Labels = marshalStringMap(labels)
	event.Annotations = buildRuleAnnotations(rule, result, labels, fingerprint)
	event.Fingerprint = fingerprint
	event.EndedAt = nil
	event.LastEvalAt = result.EvaluatedAt
	if result.StartedAt != nil && !result.StartedAt.IsZero() {
		event.StartedAt = *result.StartedAt
	}
	if err := h.db.Save(event).Error; err != nil {
		return nil, err
	}
	return event, nil
}

func (h *DataSourceHandler) applySilenceRulesToEvent(event *model.AlertEvent) bool {
	if event == nil || event.FaultCenterID == 0 || event.EndedAt != nil {
		return false
	}
	if event.State != "firing" && event.State != "pending" && event.State != "error" {
		return false
	}
	var center model.FaultCenter
	if err := h.db.First(&center, event.FaultCenterID).Error; err != nil {
		return false
	}
	if !center.SilenceEnabled {
		return false
	}
	rules := parseFaultCenterSilenceRules(center.SilenceRules)
	if len(rules) == 0 {
		return false
	}
	now := time.Now()
	if ruleName, matched := silenceRuleMatchName(rules, event, now); matched {
		event.State = "silenced"
		event.Acknowledged = true
		event.AcknowledgedBy = firstNonEmpty(ruleName, "静默规则")
		event.AcknowledgedAt = &now
		event.LastEvalAt = now
		event.NotifyStatus = "none"
		if !strings.Contains(event.Message, "静默规则") {
			event.Message = strings.TrimSpace(event.Message + "；已命中静默规则：" + ruleName)
		}
		return true
	}
	return false
}

func (h *DataSourceHandler) resyncFaultCenterSilenceEvents(faultCenterID uint) error {
	var center model.FaultCenter
	if err := h.db.First(&center, faultCenterID).Error; err != nil {
		return err
	}
	var events []model.AlertEvent
	if err := h.db.Where("fault_center_id = ? AND state IN ? AND ended_at IS NULL", faultCenterID, activeAlertEventStates()).
		Find(&events).Error; err != nil {
		return err
	}
	rules := parseFaultCenterSilenceRules(center.SilenceRules)
	now := time.Now()
	for i := range events {
		event := &events[i]
		matchedRuleName := ""
		matched := false
		if center.SilenceEnabled && len(rules) > 0 {
			matchedRuleName, matched = silenceRuleMatchName(rules, event, now)
		}
		if matched {
			if event.State != "silenced" {
				event.State = "silenced"
				event.Acknowledged = true
				event.AcknowledgedBy = firstNonEmpty(matchedRuleName, "静默规则")
				event.AcknowledgedAt = &now
				event.NotifyStatus = "none"
				event.Message = appendSilenceMessage(event.Message, matchedRuleName)
				event.LastEvalAt = now
				if err := h.db.Save(event).Error; err != nil {
					return err
				}
			}
			continue
		}
		if event.State != "silenced" {
			continue
		}
		event.State = h.currentStateForUnsilencedEvent(event)
		event.Message = trimSilenceMessage(event.Message)
		clearSilenceAcknowledgement(event)
		event.LastEvalAt = now
		if err := h.db.Save(event).Error; err != nil {
			return err
		}
	}
	return nil
}

func silenceRuleMatchName(rules []faultCenterSilenceRuleConfig, event *model.AlertEvent, now time.Time) (string, bool) {
	payload := buildRuleNotificationPayload(event)
	for _, rule := range rules {
		if rule.Enabled != nil && !*rule.Enabled {
			continue
		}
		if strings.TrimSpace(rule.Matcher) == "" {
			continue
		}
		if !silenceEffectiveTimeMatched(rule.EffectiveTime, now) {
			continue
		}
		if !noticeRouteMatcherMatched(rule.Matcher, payload) {
			continue
		}
		return firstNonEmpty(rule.Name, rule.Matcher, "静默规则"), true
	}
	return "", false
}

func (h *DataSourceHandler) currentStateForUnsilencedEvent(event *model.AlertEvent) string {
	var rule model.AlertRule
	if err := h.db.First(&rule, event.RuleID).Error; err == nil {
		state := strings.ToLower(strings.TrimSpace(rule.LastState))
		if state == "pending" || state == "firing" || state == "error" {
			return state
		}
	}
	if strings.TrimSpace(event.Condition) != "" && compareRuleValue(event.Value, event.Condition, event.Threshold) {
		return "firing"
	}
	return "firing"
}

func appendSilenceMessage(message, ruleName string) string {
	if strings.Contains(message, "；已命中静默规则：") {
		return message
	}
	return strings.TrimSpace(message + "；已命中静默规则：" + firstNonEmpty(ruleName, "静默规则"))
}

func trimSilenceMessage(message string) string {
	if left, _, ok := strings.Cut(message, "；已命中静默规则："); ok {
		return strings.TrimSpace(left)
	}
	return message
}

func clearSilenceAcknowledgement(event *model.AlertEvent) {
	if event == nil {
		return
	}
	event.Acknowledged = false
	event.AcknowledgedBy = ""
	event.AcknowledgedAt = nil
}

type faultCenterSilenceRuleConfig struct {
	Name          string `json:"name"`
	Matcher       string `json:"matcher"`
	EffectiveTime string `json:"effectiveTime"`
	Enabled       *bool  `json:"enabled"`
}

func parseFaultCenterSilenceRules(raw string) []faultCenterSilenceRuleConfig {
	var rules []faultCenterSilenceRuleConfig
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &rules); err != nil {
		return []faultCenterSilenceRuleConfig{}
	}
	return rules
}

func silenceEffectiveTimeMatched(raw string, now time.Time) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "00:00-23:59" {
		return true
	}
	start, end, ok := strings.Cut(raw, "-")
	if !ok {
		return true
	}
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if start == "" || end == "" {
		return true
	}
	current := now.Format("15:04")
	if start <= end {
		return current >= start && current <= end
	}
	return current >= start || current <= end
}

func (h *DataSourceHandler) recoverActiveAlertEvents(rule *model.AlertRule, endedAt time.Time, message string) ([]*model.AlertEvent, error) {
	return h.recoverActiveAlertEventsWithResult(rule, nil, endedAt, message)
}

type alertRecoveryResult struct {
	recovered  []*model.AlertEvent
	recovering []*model.AlertEvent
	notify     []*model.AlertEvent
}

func (h *DataSourceHandler) handleRecoveredAlertEvent(rule *model.AlertRule, event *model.AlertEvent, result *alertRuleEvaluationResult, now time.Time, message string, recoveryWait time.Duration) (alertRecoveryResult, error) {
	out := alertRecoveryResult{}
	if event == nil {
		return out, nil
	}
	shouldNotifyRecovery := shouldSendRecoveryNotification(event)
	if shouldNotifyRecovery && recoveryWaitRemainingForDuration(event, now, recoveryWait) > 0 {
		recovering, err := h.markAlertEventRecoveringWithResult(rule, event, result, now, message)
		if err != nil {
			return out, err
		}
		if recovering != nil {
			out.recovering = append(out.recovering, recovering)
		}
		return out, nil
	}
	recovered, err := h.recoverActiveAlertEventWithResult(rule, event, result, now, message)
	if err != nil {
		return out, err
	}
	if recovered != nil {
		out.recovered = append(out.recovered, recovered)
		if shouldNotifyRecovery {
			out.notify = append(out.notify, recovered)
		}
	}
	return out, nil
}

func (h *DataSourceHandler) recoverActiveAlertEventsWithResult(rule *model.AlertRule, result *alertRuleEvaluationResult, endedAt time.Time, message string) ([]*model.AlertEvent, error) {
	var events []model.AlertEvent
	if err := h.db.Where("rule_id = ? AND state IN ? AND ended_at IS NULL", rule.ID, activeAlertEventStates()).
		Order("id ASC").
		Find(&events).Error; err != nil {
		return nil, err
	}

	recoveredEvents := make([]*model.AlertEvent, 0, len(events))
	for i := range events {
		event := &events[i]
		previousAnnotations := event.Annotations
		event.State = "recovered"
		event.EndedAt = &endedAt
		event.LastEvalAt = endedAt
		if strings.TrimSpace(message) != "" {
			event.Message = message
		}
		if result != nil {
			labels := buildRuleLabelMap(rule, result)
			event.Severity = result.Severity
			event.Value = result.Value
			event.Condition = result.Condition
			event.Threshold = result.Threshold
			event.DataSourceID = result.DataSourceID
			event.DataSourceName = result.DataSourceName
			event.DataSourceType = result.DataSourceType
			event.Labels = marshalStringMap(labels)
			event.Annotations = mergeRecoveredMatchedLogAnnotations(previousAnnotations, buildRuleAnnotations(rule, result, labels, event.Fingerprint))
		}
		if err := h.db.Save(event).Error; err != nil {
			return nil, err
		}
		recoveredEvents = append(recoveredEvents, event)
	}
	return recoveredEvents, nil
}

func (h *DataSourceHandler) recoverActiveAlertEventWithResult(rule *model.AlertRule, event *model.AlertEvent, result *alertRuleEvaluationResult, endedAt time.Time, message string) (*model.AlertEvent, error) {
	if event == nil {
		return nil, nil
	}
	previousAnnotations := event.Annotations
	event.State = "recovered"
	event.EndedAt = &endedAt
	event.LastEvalAt = endedAt
	if strings.TrimSpace(message) != "" {
		event.Message = message
	}
	if result != nil {
		labels := buildRuleLabelMap(rule, result)
		fingerprint := firstNonEmpty(event.Fingerprint, result.Fingerprint)
		event.Severity = result.Severity
		event.Value = result.Value
		event.Condition = result.Condition
		event.Threshold = result.Threshold
		event.DataSourceID = result.DataSourceID
		event.DataSourceName = result.DataSourceName
		event.DataSourceType = result.DataSourceType
		event.Labels = marshalStringMap(labels)
		event.Annotations = mergeRecoveredMatchedLogAnnotations(previousAnnotations, buildRuleAnnotations(rule, result, labels, fingerprint))
		if fingerprint != "" {
			event.Fingerprint = fingerprint
		}
	}
	if err := h.db.Save(event).Error; err != nil {
		return nil, err
	}
	return event, nil
}

func (h *DataSourceHandler) markAlertEventRecoveringWithResult(rule *model.AlertRule, event *model.AlertEvent, result *alertRuleEvaluationResult, now time.Time, message string) (*model.AlertEvent, error) {
	if event == nil {
		return nil, nil
	}
	previousAnnotations := event.Annotations
	alreadyRecovering := strings.EqualFold(strings.TrimSpace(event.State), "recovering")
	event.State = "recovering"
	event.EndedAt = nil
	if !alreadyRecovering {
		event.LastEvalAt = now
	}
	if strings.TrimSpace(message) != "" {
		event.Message = message
	}
	if result != nil {
		labels := buildRuleLabelMap(rule, result)
		fingerprint := firstNonEmpty(event.Fingerprint, result.Fingerprint)
		event.Severity = result.Severity
		event.Value = result.Value
		event.Condition = result.Condition
		event.Threshold = result.Threshold
		event.DataSourceID = result.DataSourceID
		event.DataSourceName = result.DataSourceName
		event.DataSourceType = result.DataSourceType
		event.Labels = marshalStringMap(labels)
		event.Annotations = mergeRecoveredMatchedLogAnnotations(previousAnnotations, buildRuleAnnotations(rule, result, labels, fingerprint))
		if fingerprint != "" {
			event.Fingerprint = fingerprint
		}
	}
	if err := h.db.Save(event).Error; err != nil {
		return nil, err
	}
	return event, nil
}

func (h *DataSourceHandler) recoverMissingActiveAlertEvents(rule *model.AlertRule, seenFingerprints map[string]struct{}, endedAt time.Time, recoveryWait time.Duration) (alertRecoveryResult, error) {
	out := alertRecoveryResult{}
	var events []model.AlertEvent
	if err := h.db.Where("rule_id = ? AND state IN ? AND ended_at IS NULL", rule.ID, activeAlertEventStates()).
		Order("id ASC").
		Find(&events).Error; err != nil {
		return out, err
	}

	for i := range events {
		event := &events[i]
		if _, ok := seenFingerprints[event.Fingerprint]; ok {
			continue
		}
		result := &alertRuleEvaluationResult{
			RuleID:         event.RuleID,
			RuleName:       event.RuleName,
			RuleGroupID:    event.RuleGroupID,
			FaultCenterID:  event.FaultCenterID,
			DataSourceID:   event.DataSourceID,
			DataSourceName: event.DataSourceName,
			DataSourceType: event.DataSourceType,
			Severity:       event.Severity,
			State:          "recovered",
			Matched:        false,
			Value:          event.Value,
			Condition:      event.Condition,
			Threshold:      event.Threshold,
			Labels:         parseStringMap(event.Labels),
			Fingerprint:    event.Fingerprint,
			Message:        event.Message,
			EvaluatedAt:    endedAt,
		}
		message := fmt.Sprintf("规则「%s」已恢复：查询结果中未再返回该序列", rule.Name)
		recovery, err := h.handleRecoveredAlertEvent(rule, event, result, endedAt, message, recoveryWait)
		if err != nil {
			return out, err
		}
		out.recovered = append(out.recovered, recovery.recovered...)
		out.recovering = append(out.recovering, recovery.recovering...)
		out.notify = append(out.notify, recovery.notify...)
	}
	return out, nil
}

func (h *DataSourceHandler) findLatestActiveAlertEvent(rule *model.AlertRule, states []string) (*model.AlertEvent, error) {
	var event model.AlertEvent
	err := h.db.Where("rule_id = ? AND state IN ? AND ended_at IS NULL", rule.ID, states).
		Order("last_eval_at DESC, id DESC").
		First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (h *DataSourceHandler) findActiveAlertEventByFingerprint(rule *model.AlertRule, fingerprint string, states []string) (*model.AlertEvent, error) {
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return nil, nil
	}
	var event model.AlertEvent
	err := h.db.Where("rule_id = ? AND fingerprint = ? AND state IN ? AND ended_at IS NULL", rule.ID, fingerprint, states).
		Order("last_eval_at DESC, id DESC").
		First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (h *DataSourceHandler) ensurePendingAlertEvents(ctx context.Context, faultCenterID string) error {
	var rules []model.AlertRule
	query := h.db.WithContext(ctx).Where("enabled = ? AND last_state = ?", true, "pending")
	if strings.TrimSpace(faultCenterID) != "" {
		query = query.Where("fault_center_id = ?", faultCenterID)
	}
	if err := query.Find(&rules).Error; err != nil {
		return err
	}
	for i := range rules {
		rule := &rules[i]
		event, err := h.findLatestActiveAlertEvent(rule, activeAlertEventStates())
		if err != nil {
			return err
		}
		if event != nil {
			continue
		}
		var ds model.DataSource
		if err := h.db.WithContext(ctx).First(&ds, rule.DataSourceID).Error; err != nil {
			continue
		}
		evaluatedAt := time.Now()
		if rule.LastEvalAt != nil {
			evaluatedAt = *rule.LastEvalAt
		}
		result := &alertRuleEvaluationResult{
			RuleID:         rule.ID,
			RuleName:       rule.Name,
			RuleGroupID:    rule.RuleGroupID,
			FaultCenterID:  rule.FaultCenterID,
			DataSourceID:   ds.ID,
			DataSourceName: ds.Name,
			DataSourceType: ds.Type,
			State:          "pending",
			Matched:        true,
			Value:          rule.LastValue,
			Severity:       rule.Severity,
			Condition:      rule.Condition,
			Threshold:      rule.Threshold,
			ForSeconds:     rule.ForSeconds,
			Message:        fmt.Sprintf("规则「%s」处于预告警状态：当前值 %s %s %s", rule.Name, formatRuleValue(rule.LastValue), getConditionText(rule.Condition), formatRuleValue(rule.Threshold)),
			EvaluatedAt:    evaluatedAt,
		}
		_, err = h.upsertActiveAlertEvent(rule, &ds, result, "pending", result.Message)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildAlertFingerprint(ruleID uint, ruleName, severity string, labels map[string]string) string {
	fingerprintLabels := cloneStringMap(labels)
	fingerprintLabels["rule_id"] = strconv.FormatUint(uint64(ruleID), 10)
	fingerprintLabels["rule_name"] = strings.TrimSpace(ruleName)
	fingerprintLabels["severity"] = normalizeSeverity(severity)

	keys := make([]string, 0, len(fingerprintLabels))
	for key := range fingerprintLabels {
		key = strings.TrimSpace(key)
		if key != "" && key != "value" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+strings.TrimSpace(fingerprintLabels[key]))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\xff")))
	return fmt.Sprintf("%x", sum)
}

func compareRuleValue(value float64, condition string, threshold float64) bool {
	switch condition {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return math.Abs(value-threshold) < 0.000001
	case "neq":
		return math.Abs(value-threshold) >= 0.000001
	default:
		return false
	}
}

func getConditionText(condition string) string {
	switch condition {
	case "gt":
		return ">"
	case "gte":
		return ">="
	case "lt":
		return "<"
	case "lte":
		return "<="
	case "eq":
		return "="
	case "neq":
		return "!="
	default:
		return condition
	}
}

func formatRuleValue(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

type ruleEvaluationSample struct {
	Value           float64
	Labels          map[string]string
	MatchedLogs     []matchedLogEntry
	MatchedLogCount int
	MatchedLogQuery string
}

func extractRuleEvaluationSamples(dsType string, raw interface{}) ([]ruleEvaluationSample, error) {
	switch dsType {
	case "prometheus", "victoriametrics", "loki":
		return extractPromLikeSamples(raw)
	case opsHubLogDataSourceType:
		return extractOpsHubLogSamples(raw)
	case "elasticsearch", "opensearch":
		value, err := extractElasticsearchValue(raw)
		if err != nil {
			return nil, err
		}
		return []ruleEvaluationSample{{Value: value, Labels: extractElasticsearchLabels(raw)}}, nil
	default:
		return nil, fmt.Errorf("不支持的数据源类型: %s", dsType)
	}
}

func extractRuleNumericValue(dsType string, raw interface{}) (float64, error) {
	switch dsType {
	case "prometheus", "victoriametrics", "loki":
		return extractPromLikeValue(raw)
	case opsHubLogDataSourceType:
		samples, err := extractOpsHubLogSamples(raw)
		if err != nil || len(samples) == 0 {
			return 0, err
		}
		return samples[0].Value, nil
	case "elasticsearch", "opensearch":
		return extractElasticsearchValue(raw)
	default:
		return 0, fmt.Errorf("不支持的数据源类型: %s", dsType)
	}
}

func extractRuleResultLabels(dsType string, raw interface{}) map[string]string {
	switch dsType {
	case "prometheus", "victoriametrics", "loki":
		return extractPromLikeLabels(raw)
	case opsHubLogDataSourceType:
		samples, _ := extractOpsHubLogSamples(raw)
		if len(samples) > 0 {
			return samples[0].Labels
		}
		return map[string]string{}
	case "elasticsearch", "opensearch":
		return extractElasticsearchLabels(raw)
	default:
		return map[string]string{}
	}
}

func extractPromLikeLabels(raw interface{}) map[string]string {
	root, ok := raw.(map[string]interface{})
	if !ok {
		return map[string]string{}
	}
	data, _ := root["data"].(map[string]interface{})
	result, _ := data["result"].([]interface{})
	for _, item := range result {
		series, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if metric := labelsFromInterfaceMap(series["metric"]); len(metric) > 0 {
			return metric
		}
		if stream := labelsFromInterfaceMap(series["stream"]); len(stream) > 0 {
			return stream
		}
	}
	return map[string]string{}
}

func extractElasticsearchLabels(raw interface{}) map[string]string {
	root, ok := raw.(map[string]interface{})
	if !ok {
		return map[string]string{}
	}
	hits, _ := root["hits"].(map[string]interface{})
	items, _ := hits["hits"].([]interface{})
	for _, item := range items {
		hit, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		source := labelsFromInterfaceMap(hit["_source"])
		if len(source) > 0 {
			if index, ok := hit["_index"].(string); ok && index != "" {
				source["_index"] = index
			}
			return source
		}
	}
	return map[string]string{}
}

func labelsFromInterfaceMap(raw interface{}) map[string]string {
	values := map[string]string{}
	switch data := raw.(type) {
	case map[string]string:
		for key, value := range data {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			values[key] = strings.TrimSpace(value)
		}
	case map[string]interface{}:
		for key, value := range data {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			switch typed := value.(type) {
			case string:
				values[key] = strings.TrimSpace(typed)
			case float64, float32, int, int64, uint, uint64, bool:
				values[key] = fmt.Sprint(typed)
			}
		}
	}
	return values
}

func extractPromLikeValue(raw interface{}) (float64, error) {
	root, ok := raw.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("响应不是 JSON 对象")
	}
	data, ok := root["data"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("响应缺少 data 字段")
	}

	resultType, _ := data["resultType"].(string)
	switch resultType {
	case "scalar":
		return valueFromPair(data["result"])
	case "vector":
		return maxValueFromPromResults(data["result"], "value")
	case "matrix":
		return maxValueFromPromResults(data["result"], "values")
	case "streams":
		return countLokiStreamEntries(data["result"])
	default:
		if value, err := maxValueFromPromResults(data["result"], "value"); err == nil {
			return value, nil
		}
		if value, err := maxValueFromPromResults(data["result"], "values"); err == nil {
			return value, nil
		}
		return 0, fmt.Errorf("无法识别的结果类型: %s", resultType)
	}
}

func extractPromLikeSamples(raw interface{}) ([]ruleEvaluationSample, error) {
	root, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("响应不是 JSON 对象")
	}
	data, ok := root["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("响应缺少 data 字段")
	}

	resultType, _ := data["resultType"].(string)
	switch resultType {
	case "scalar":
		value, err := valueFromPair(data["result"])
		if err != nil {
			return nil, err
		}
		return []ruleEvaluationSample{{Value: value, Labels: map[string]string{}}}, nil
	case "vector":
		return samplesFromPromResults(data["result"], "value")
	case "matrix":
		return samplesFromPromResults(data["result"], "values")
	case "streams":
		return samplesFromLokiStreams(data["result"])
	default:
		if samples, err := samplesFromPromResults(data["result"], "value"); err == nil {
			return samples, nil
		}
		if samples, err := samplesFromPromResults(data["result"], "values"); err == nil {
			return samples, nil
		}
		return nil, fmt.Errorf("无法识别的结果类型: %s", resultType)
	}
}

func valueFromPair(raw interface{}) (float64, error) {
	items, ok := raw.([]interface{})
	if !ok || len(items) < 2 {
		return 0, fmt.Errorf("数值格式不正确")
	}
	return parseFloat(items[1])
}

func samplesFromPromResults(raw interface{}, key string) ([]ruleEvaluationSample, error) {
	results, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("查询结果为空")
	}
	if len(results) == 0 {
		return []ruleEvaluationSample{}, nil
	}

	samples := make([]ruleEvaluationSample, 0, len(results))
	for _, item := range results {
		series, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		var value float64
		var err error
		if key == "value" {
			value, err = valueFromPair(series["value"])
		} else {
			values, ok := series["values"].([]interface{})
			if !ok || len(values) == 0 {
				continue
			}
			value, err = valueFromPair(values[len(values)-1])
		}
		if err != nil {
			continue
		}
		labels := labelsFromInterfaceMap(series["metric"])
		if len(labels) == 0 {
			labels = labelsFromInterfaceMap(series["stream"])
		}
		samples = append(samples, ruleEvaluationSample{Value: value, Labels: labels})
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("查询结果中没有可用数值")
	}
	return samples, nil
}

func samplesFromLokiStreams(raw interface{}) ([]ruleEvaluationSample, error) {
	streams, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Loki stream 结果格式不正确")
	}
	samples := make([]ruleEvaluationSample, 0, len(streams))
	total := 0
	for _, item := range streams {
		stream, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		values, _ := stream["values"].([]interface{})
		count := len(values)
		total += count
		labels := labelsFromInterfaceMap(stream["stream"])
		samples = append(samples, ruleEvaluationSample{
			Value:           float64(count),
			Labels:          labels,
			MatchedLogs:     matchedLogsFromLokiValues(values, labels, maxMatchedLogLinesPerEvent),
			MatchedLogCount: count,
		})
	}
	if len(samples) == 0 {
		return []ruleEvaluationSample{{Value: float64(total), Labels: map[string]string{}}}, nil
	}
	return samples, nil
}

func matchedLogsFromLokiValues(values []interface{}, labels map[string]string, limit int) []matchedLogEntry {
	if limit <= 0 {
		limit = maxMatchedLogLinesPerEvent
	}
	logs := make([]matchedLogEntry, 0, minInt(len(values), limit))
	for _, raw := range values {
		if len(logs) >= limit {
			break
		}
		pair, ok := raw.([]interface{})
		if !ok || len(pair) < 2 {
			continue
		}
		line := strings.TrimSpace(fmt.Sprint(pair[1]))
		if line == "" {
			continue
		}
		logs = append(logs, matchedLogEntry{
			Timestamp: formatLokiLogTimestamp(pair[0]),
			Line:      clipPlainText(line, maxMatchedLogLineChars),
			Labels:    cloneStringMap(labels),
		})
	}
	return logs
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatLokiLogTimestamp(raw interface{}) string {
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" || text == "<nil>" {
		return ""
	}
	if ns, err := strconv.ParseInt(text, 10, 64); err == nil {
		if ns > 1_000_000_000_000_000 {
			return time.Unix(0, ns).Format("2006-01-02 15:04:05.000")
		}
		if ns > 1_000_000_000 {
			return time.Unix(ns, 0).Format("2006-01-02 15:04:05")
		}
	}
	if seconds, err := strconv.ParseFloat(text, 64); err == nil {
		sec, frac := math.Modf(seconds)
		return time.Unix(int64(sec), int64(frac*1e9)).Format("2006-01-02 15:04:05.000")
	}
	if ts, err := time.Parse(time.RFC3339Nano, text); err == nil {
		return ts.Format("2006-01-02 15:04:05.000")
	}
	return text
}

func deriveLokiMatchedLogQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	if strings.HasPrefix(query, "{") {
		return strings.TrimSpace(stripTrailingLokiRangeSelector(query))
	}
	for _, fn := range []string{"count_over_time", "rate", "bytes_over_time", "bytes_rate", "absent_over_time"} {
		if arg, ok := lokiFunctionArgument(query, fn); ok {
			arg = strings.TrimSpace(stripTrailingLokiRangeSelector(arg))
			if strings.HasPrefix(arg, "{") {
				return arg
			}
		}
	}
	return ""
}

func lokiFunctionArgument(query, fn string) (string, bool) {
	lowerQuery := strings.ToLower(query)
	lowerFn := strings.ToLower(fn)
	searchFrom := 0
	for {
		index := strings.Index(lowerQuery[searchFrom:], lowerFn)
		if index < 0 {
			return "", false
		}
		index += searchFrom
		beforeOK := index == 0 || !isIdentifierRune(rune(lowerQuery[index-1]))
		afterIndex := index + len(lowerFn)
		if !beforeOK || afterIndex >= len(query) || query[afterIndex] != '(' {
			searchFrom = afterIndex
			continue
		}
		start := afterIndex + 1
		depth := 1
		var quote rune
		escaped := false
		for pos, r := range query[start:] {
			abs := start + pos
			if quote != 0 {
				if escaped {
					escaped = false
					continue
				}
				if r == '\\' {
					escaped = true
					continue
				}
				if r == quote {
					quote = 0
				}
				continue
			}
			if r == '"' || r == '\'' || r == '`' {
				quote = r
				continue
			}
			switch r {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					return query[start:abs], true
				}
			}
		}
		return "", false
	}
}

func isIdentifierRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func stripTrailingLokiRangeSelector(query string) string {
	return strings.TrimSpace(lokiRangeSelectorRegexp.ReplaceAllString(query, ""))
}

func lokiMatchedLogLookbackSeconds(rule *model.AlertRule) int {
	if rule == nil {
		return 300
	}
	window := extractLastLokiRangeSeconds(rule.Query)
	window = maxInt(window, rule.ForSeconds, rule.EvaluateInterval)
	if window <= 0 {
		window = 300
	}
	if window < 60 {
		window = 60
	}
	if window > maxLokiMatchedLogLookbackSeconds {
		window = maxLokiMatchedLogLookbackSeconds
	}
	return window
}

func extractLastLokiRangeSeconds(query string) int {
	matches := lokiAnyRangeSelectorRegexp.FindAllStringSubmatch(query, -1)
	if len(matches) == 0 {
		return 0
	}
	return parsePrometheusDurationSeconds(matches[len(matches)-1][1])
}

func matchedLogsText(logs []matchedLogEntry, total int) string {
	if len(logs) == 0 {
		return ""
	}
	lines := make([]string, 0, len(logs)+1)
	for _, log := range logs {
		line := strings.TrimSpace(log.Line)
		if line == "" {
			continue
		}
		if log.Timestamp != "" {
			line = fmt.Sprintf("[%s] %s", log.Timestamp, line)
		}
		lines = append(lines, clipPlainText(line, maxMatchedLogLineChars))
	}
	if total > len(logs) {
		lines = append(lines, fmt.Sprintf("... 还有 %d 行命中日志未展示", total-len(logs)))
	}
	return clipPlainText(strings.Join(lines, "\n"), maxMatchedLogsTextChars)
}

func maxValueFromPromResults(raw interface{}, key string) (float64, error) {
	results, ok := raw.([]interface{})
	if !ok || len(results) == 0 {
		return 0, fmt.Errorf("查询结果为空")
	}

	found := false
	maxValue := -math.MaxFloat64
	for _, item := range results {
		series, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if key == "value" {
			value, err := valueFromPair(series["value"])
			if err == nil {
				found = true
				if value > maxValue {
					maxValue = value
				}
			}
			continue
		}
		values, ok := series["values"].([]interface{})
		if !ok || len(values) == 0 {
			continue
		}
		value, err := valueFromPair(values[len(values)-1])
		if err == nil {
			found = true
			if value > maxValue {
				maxValue = value
			}
		}
	}
	if !found {
		return 0, fmt.Errorf("查询结果中没有可用数值")
	}
	return maxValue, nil
}

func countLokiStreamEntries(raw interface{}) (float64, error) {
	streams, ok := raw.([]interface{})
	if !ok {
		return 0, fmt.Errorf("Loki stream 结果格式不正确")
	}
	var total int
	for _, item := range streams {
		stream, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		values, ok := stream["values"].([]interface{})
		if ok {
			total += len(values)
		}
	}
	return float64(total), nil
}

func extractElasticsearchValue(raw interface{}) (float64, error) {
	root, ok := raw.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("响应不是 JSON 对象")
	}
	if aggs, ok := root["aggregations"].(map[string]interface{}); ok {
		if value, ok := firstNumericInAggregation(aggs); ok {
			return value, nil
		}
	}
	if hits, ok := root["hits"].(map[string]interface{}); ok {
		if total, ok := hits["total"].(map[string]interface{}); ok {
			if value, err := parseFloat(total["value"]); err == nil {
				return value, nil
			}
		}
		if hitList, ok := hits["hits"].([]interface{}); ok {
			return float64(len(hitList)), nil
		}
	}
	return 0, fmt.Errorf("ES 响应中没有 hits.total 或可用聚合数值")
}

func firstNumericInAggregation(raw interface{}) (float64, bool) {
	switch value := raw.(type) {
	case map[string]interface{}:
		if direct, ok := value["value"]; ok {
			if parsed, err := parseFloat(direct); err == nil {
				return parsed, true
			}
		}
		if buckets, ok := value["buckets"].([]interface{}); ok {
			for _, bucket := range buckets {
				if parsed, ok := firstNumericInAggregation(bucket); ok {
					return parsed, true
				}
			}
		}
		for _, child := range value {
			if parsed, ok := firstNumericInAggregation(child); ok {
				return parsed, true
			}
		}
	case []interface{}:
		for _, child := range value {
			if parsed, ok := firstNumericInAggregation(child); ok {
				return parsed, true
			}
		}
	}
	return 0, false
}

func parseFloat(raw interface{}) (float64, error) {
	switch value := raw.(type) {
	case float64:
		return value, nil
	case float32:
		return float64(value), nil
	case int:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case json.Number:
		return value.Float64()
	case string:
		return strconv.ParseFloat(value, 64)
	default:
		return 0, fmt.Errorf("无法解析数值")
	}
}

type severityCondition struct {
	Severity   string  `json:"severity"`
	Condition  string  `json:"condition"`
	Threshold  float64 `json:"threshold"`
	ForSeconds int     `json:"forSeconds"`
	Matched    bool    `json:"-"`
}

func parseRuleDataSourceIDs(raw string) ([]uint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var ids []uint
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &ids); err != nil {
			return nil, fmt.Errorf("数据源列表必须是 ID 数组 JSON")
		}
		return filterPositiveIDs(ids), nil
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 64)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("数据源 ID 格式不正确")
		}
		ids = append(ids, uint(id))
	}
	return filterPositiveIDs(ids), nil
}

func filterPositiveIDs(ids []uint) []uint {
	result := make([]uint, 0, len(ids))
	seen := map[uint]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func parseSeverityConditions(rule *model.AlertRule) ([]severityCondition, error) {
	raw := strings.TrimSpace(rule.SeverityRules)
	if raw == "" {
		return []severityCondition{{
			Severity:   normalizeSeverity(rule.Severity),
			Condition:  normalizeCondition(rule.Condition),
			Threshold:  rule.Threshold,
			ForSeconds: positiveInt(rule.ForSeconds, 60),
		}}, nil
	}
	var conditions []severityCondition
	if err := json.Unmarshal([]byte(raw), &conditions); err != nil {
		return nil, fmt.Errorf("告警等级条件必须是合法 JSON 数组")
	}
	if len(conditions) == 0 {
		return nil, fmt.Errorf("告警等级条件不能为空")
	}
	for i := range conditions {
		conditions[i].Severity = normalizeSeverity(conditions[i].Severity)
		conditions[i].Condition = normalizeCondition(conditions[i].Condition)
		conditions[i].ForSeconds = positiveInt(conditions[i].ForSeconds, rule.ForSeconds)
		if !isSupportedSeverity(conditions[i].Severity) {
			return nil, fmt.Errorf("不支持的告警等级: %s", conditions[i].Severity)
		}
		if !isSupportedCondition(conditions[i].Condition) {
			return nil, fmt.Errorf("不支持的判断条件: %s", conditions[i].Condition)
		}
	}
	return conditions, nil
}

func selectSeverityCondition(rule *model.AlertRule, value float64) severityCondition {
	conditions, err := parseSeverityConditions(rule)
	if err != nil || len(conditions) == 0 {
		conditions = []severityCondition{{
			Severity:   normalizeSeverity(rule.Severity),
			Condition:  normalizeCondition(rule.Condition),
			Threshold:  rule.Threshold,
			ForSeconds: positiveInt(rule.ForSeconds, 60),
		}}
	}
	selected := conditions[0]
	for _, condition := range conditions {
		if compareRuleValue(value, condition.Condition, condition.Threshold) {
			condition.Matched = true
			return condition
		}
	}
	selected.Matched = false
	return selected
}

func ruleSamplePreferred(candidateValue float64, candidateCondition severityCondition, currentValue float64, currentCondition severityCondition) bool {
	candidateSeverityRank := severityRank(candidateCondition.Severity)
	currentSeverityRank := severityRank(currentCondition.Severity)
	if candidateCondition.Matched && currentCondition.Matched && candidateSeverityRank != currentSeverityRank {
		return candidateSeverityRank < currentSeverityRank
	}
	switch candidateCondition.Condition {
	case "lt", "lte":
		return candidateValue < currentValue
	case "gt", "gte":
		return candidateValue > currentValue
	case "eq":
		return math.Abs(candidateValue-candidateCondition.Threshold) < math.Abs(currentValue-currentCondition.Threshold)
	case "neq":
		return math.Abs(candidateValue-candidateCondition.Threshold) > math.Abs(currentValue-currentCondition.Threshold)
	default:
		return candidateValue > currentValue
	}
}

func severityRank(severity string) int {
	switch normalizeSeverity(severity) {
	case "p0", "critical":
		return 0
	case "p1", "warning":
		return 1
	case "p2", "info":
		return 2
	default:
		return 3
	}
}

func normalizeSeverity(severity string) string {
	severity = strings.ToLower(strings.TrimSpace(severity))
	if severity == "" {
		return "warning"
	}
	return severity
}

func normalizeCondition(condition string) string {
	condition = strings.ToLower(strings.TrimSpace(condition))
	if condition == "" {
		return "gt"
	}
	return condition
}

func positiveInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return 60
}

func maxInt(values ...int) int {
	max := 0
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

func isSupportedSeverity(severity string) bool {
	switch severity {
	case "info", "warning", "critical", "p0", "p1", "p2":
		return true
	default:
		return false
	}
}

func isSupportedCondition(condition string) bool {
	switch condition {
	case "gt", "gte", "lt", "lte", "eq", "neq":
		return true
	default:
		return false
	}
}

func parseRuleChannelIDs(raw string) ([]uint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var ids []uint
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &ids); err != nil {
			return nil, fmt.Errorf("告警通道必须是 ID 数组 JSON")
		}
		return ids, nil
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 64)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("告警通道 ID 格式不正确")
		}
		ids = append(ids, uint(id))
	}
	return ids, nil
}

func buildRuleLabelMap(rule *model.AlertRule, result *alertRuleEvaluationResult) map[string]string {
	labels := map[string]string{}
	if result != nil {
		for key, value := range result.Labels {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			labels[key] = strings.TrimSpace(value)
		}
		labels["value"] = formatRuleValue(result.Value)
	}
	for key, value := range parseStringMap(rule.Labels) {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		labels[key] = renderAlertTemplateText(value, rule, result, labels, nil, "")
	}
	if result != nil {
		labels["value"] = formatRuleValue(result.Value)
	}
	return labels
}

func buildRuleAnnotations(rule *model.AlertRule, result *alertRuleEvaluationResult, labels map[string]string, fingerprint string) string {
	annotations := parseStringMap(rule.Annotations)
	rendered := map[string]string{}
	for key, value := range annotations {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		rendered[key] = renderAlertTemplateText(value, rule, result, labels, annotations, fingerprint)
	}
	if detail := strings.TrimSpace(rule.DetailTemplate); detail != "" {
		rendered["description"] = renderAlertTemplateText(detail, rule, result, labels, annotations, fingerprint)
	}
	appendMatchedLogAnnotations(rendered, result)
	if len(rendered) > 0 {
		return marshalStringMap(rendered)
	}
	if strings.TrimSpace(rule.Annotations) != "" && len(annotations) == 0 {
		return renderAlertTemplateText(rule.Annotations, rule, result, labels, annotations, fingerprint)
	}
	return ""
}

func appendMatchedLogAnnotations(rendered map[string]string, result *alertRuleEvaluationResult) {
	if rendered == nil || result == nil {
		return
	}
	count := result.MatchedLogCount
	if count <= 0 {
		count = len(result.MatchedLogs)
	}
	logText := matchedLogsText(result.MatchedLogs, count)
	if count <= 0 && logText == "" {
		return
	}
	rendered["matched_log_count"] = strconv.Itoa(count)
	if logText != "" {
		rendered["matched_logs"] = logText
	}
	if strings.TrimSpace(result.MatchedLogQuery) != "" {
		rendered["matched_log_query"] = result.MatchedLogQuery
	}
}

func carryMatchedLogAnnotations(target, source map[string]string) {
	if target == nil || len(source) == 0 {
		return
	}
	carryAnnotationValue(target, source, "matched_log_count", "matchedLogCount")
	carryAnnotationValue(target, source, "matched_logs", "matchedLogs")
	carryAnnotationValue(target, source, "matched_log_query", "matchedLogQuery")
}

func carryAnnotationValue(target, source map[string]string, key string, aliases ...string) {
	if strings.TrimSpace(target[key]) != "" && strings.TrimSpace(target[key]) != "0" {
		return
	}
	keys := append([]string{key}, aliases...)
	for _, item := range keys {
		if value := strings.TrimSpace(source[item]); value != "" && value != "0" {
			target[key] = value
			return
		}
	}
}

func mergeRecoveredMatchedLogAnnotations(previousRaw, nextRaw string) string {
	next := parseStringMap(nextRaw)
	if len(next) == 0 {
		return nextRaw
	}
	previous := parseStringMap(previousRaw)
	carryMatchedLogAnnotations(next, previous)
	return marshalStringMap(next)
}

func marshalStringMap(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	cleaned := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		cleaned[key] = strings.TrimSpace(value)
	}
	if len(cleaned) == 0 {
		return ""
	}
	data, err := json.Marshal(cleaned)
	if err != nil {
		return ""
	}
	return string(data)
}

func renderAlertCallbackQuery(query string, rule model.AlertRule, event model.AlertEvent) string {
	labels := parseStringMap(event.Labels)
	annotations := parseStringMap(event.Annotations)
	result := &alertRuleEvaluationResult{
		RuleID:         event.RuleID,
		RuleName:       event.RuleName,
		RuleGroupID:    event.RuleGroupID,
		FaultCenterID:  event.FaultCenterID,
		DataSourceID:   event.DataSourceID,
		DataSourceName: event.DataSourceName,
		DataSourceType: event.DataSourceType,
		Severity:       event.Severity,
		State:          event.State,
		Value:          event.Value,
		Condition:      event.Condition,
		Threshold:      event.Threshold,
		Message:        event.Message,
		EvaluatedAt:    event.LastEvalAt,
	}
	if rule.Name == "" {
		rule.Name = event.RuleName
	}
	return renderAlertTemplateText(query, &rule, result, labels, annotations, event.Fingerprint)
}

func renderAlertTemplateText(text string, rule *model.AlertRule, result *alertRuleEvaluationResult, labels, annotations map[string]string, fingerprint string) string {
	if result == nil {
		result = &alertRuleEvaluationResult{}
	}
	if labels == nil {
		labels = map[string]string{}
	}
	labels = cloneStringMap(labels)
	labels["value"] = formatRuleValue(result.Value)
	if annotations == nil {
		annotations = map[string]string{}
	}
	ruleName := firstNonEmpty(result.RuleName, rule.Name)
	matchedLogCount := result.MatchedLogCount
	if matchedLogCount <= 0 {
		matchedLogCount = len(result.MatchedLogs)
	}
	matchedLogs := matchedLogsText(result.MatchedLogs, matchedLogCount)
	matchedLogsBlock := matchedLogsCodeBlock(matchedLogs)
	labelText := formatTemplateStringMap(labels)
	replacements := map[string]string{
		"{{ruleName}}":           ruleName,
		"{{rule_name}}":          ruleName,
		"${rule_name}":           ruleName,
		"{{severity}}":           normalizeSeverityLevel(result.Severity),
		"${severity}":            normalizeSeverityLevel(result.Severity),
		"{{state}}":              result.State,
		"${state}":               result.State,
		"{{value}}":              formatRuleValue(result.Value),
		"{{ $value }}":           formatRuleValue(result.Value),
		"${value}":               formatRuleValue(result.Value),
		"{{condition}}":          getConditionText(result.Condition),
		"${condition}":           getConditionText(result.Condition),
		"{{threshold}}":          formatRuleValue(result.Threshold),
		"${threshold}":           formatRuleValue(result.Threshold),
		"{{dataSourceName}}":     result.DataSourceName,
		"${data_source_name}":    result.DataSourceName,
		"{{dataSourceType}}":     result.DataSourceType,
		"${data_source_type}":    result.DataSourceType,
		"{{message}}":            result.Message,
		"${message}":             result.Message,
		"{{fingerprint}}":        fingerprint,
		"${fingerprint}":         fingerprint,
		"{{labels}}":             labelText,
		"{{ $labels }}":          labelText,
		"${labels}":              labelText,
		"{{matchedLogs}}":        matchedLogs,
		"{{matched_logs}}":       matchedLogs,
		"${matched_logs}":        matchedLogs,
		"{{matchedLogsBlock}}":   matchedLogsBlock,
		"{{matched_logs_block}}": matchedLogsBlock,
		"${matched_logs_block}":  matchedLogsBlock,
		"{{matchedLogCount}}":    strconv.Itoa(matchedLogCount),
		"{{matched_log_count}}":  strconv.Itoa(matchedLogCount),
		"${matched_log_count}":   strconv.Itoa(matchedLogCount),
		"{{matchedLogQuery}}":    result.MatchedLogQuery,
		"{{matched_log_query}}":  result.MatchedLogQuery,
		"${matched_log_query}":   result.MatchedLogQuery,
	}
	for key, value := range replacements {
		text = strings.ReplaceAll(text, key, value)
	}
	text = regexp.MustCompile(`{{\s*\$?value\s*}}`).ReplaceAllString(text, formatRuleValue(result.Value))
	text = regexp.MustCompile(`{{\s*\$?labels\s*}}|\$\{labels\}`).ReplaceAllString(text, labelText)
	text = replaceAlertScopedVariables(text, alertLabelTemplatePattern, labels)
	text = replaceAlertScopedVariables(text, alertAnnotationTemplatePattern, annotations)
	return text
}

func replaceAlertScopedVariables(text string, pattern *regexp.Regexp, values map[string]string) string {
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		for _, part := range pattern.FindStringSubmatch(match)[1:] {
			if part == "" || !alertTemplateVariableNameRegexp.MatchString(part) {
				continue
			}
			return values[part]
		}
		return match
	})
}

func formatTemplateStringMap(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.TrimSpace(values[key])))
	}
	return strings.Join(parts, ", ")
}

type ruleNotificationPayload struct {
	RuleID         uint                           `json:"ruleId"`
	EventID        uint                           `json:"eventId"`
	RuleName       string                         `json:"ruleName"`
	FaultCenterID  uint                           `json:"faultCenterId"`
	DataSourceID   uint                           `json:"dataSourceId"`
	DataSourceName string                         `json:"dataSourceName"`
	DataSourceType string                         `json:"dataSourceType"`
	Severity       string                         `json:"severity"`
	State          string                         `json:"state"`
	Value          float64                        `json:"value"`
	Condition      string                         `json:"condition"`
	Threshold      float64                        `json:"threshold"`
	Message        string                         `json:"message"`
	Labels         string                         `json:"labels"`
	Annotations    string                         `json:"annotations"`
	Fingerprint    string                         `json:"fingerprint"`
	StartedAt      time.Time                      `json:"startedAt"`
	EndedAt        *time.Time                     `json:"endedAt"`
	Time           time.Time                      `json:"time"`
	Aggregated     bool                           `json:"aggregated,omitempty"`
	EventCount     int                            `json:"eventCount,omitempty"`
	Events         []ruleNotificationEventPayload `json:"events,omitempty"`
	Escalated      bool                           `json:"escalated,omitempty"`
	EscalationText string                         `json:"escalationText,omitempty"`
}

type ruleNotificationEventPayload struct {
	RuleID         uint              `json:"ruleId"`
	EventID        uint              `json:"eventId"`
	RuleName       string            `json:"ruleName"`
	FaultCenterID  uint              `json:"faultCenterId"`
	DataSourceID   uint              `json:"dataSourceId"`
	DataSourceName string            `json:"dataSourceName"`
	DataSourceType string            `json:"dataSourceType"`
	Severity       string            `json:"severity"`
	State          string            `json:"state"`
	Value          float64           `json:"value"`
	Condition      string            `json:"condition"`
	Threshold      float64           `json:"threshold"`
	Message        string            `json:"message"`
	Labels         map[string]string `json:"labels"`
	Annotations    map[string]string `json:"annotations"`
	AnnotationText string            `json:"annotationText"`
	Fingerprint    string            `json:"fingerprint"`
	Instance       string            `json:"instance"`
	EventURL       string            `json:"eventUrl"`
	StartedAt      time.Time         `json:"startedAt"`
	EndedAt        *time.Time        `json:"endedAt"`
	Time           time.Time         `json:"time"`
}

type alertChannelConfig struct {
	WebhookURL      string `json:"webhookUrl"`
	WeChatWebhook   string `json:"wechatWebhook"`
	DingTalkWebhook string `json:"dingtalkWebhook"`
	DingTalkSecret  string `json:"dingtalkSecret"`
	FeishuWebhook   string `json:"feishuWebhook"`
	SMTPHost        string `json:"smtpHost"`
	SMTPPort        int    `json:"smtpPort"`
	SMTPUser        string `json:"smtpUser"`
	SMTPPassword    string `json:"smtpPassword"`
	FromEmail       string `json:"fromEmail"`
	FromName        string `json:"fromName"`
}

func buildRuleNotificationPayload(event *model.AlertEvent) ruleNotificationPayload {
	if event == nil {
		return ruleNotificationPayload{}
	}
	return ruleNotificationPayload{
		RuleID:         event.RuleID,
		EventID:        event.ID,
		RuleName:       event.RuleName,
		FaultCenterID:  event.FaultCenterID,
		DataSourceID:   event.DataSourceID,
		DataSourceName: event.DataSourceName,
		DataSourceType: event.DataSourceType,
		Severity:       event.Severity,
		State:          event.State,
		Value:          event.Value,
		Condition:      event.Condition,
		Threshold:      event.Threshold,
		Message:        event.Message,
		Labels:         event.Labels,
		Annotations:    event.Annotations,
		Fingerprint:    event.Fingerprint,
		StartedAt:      event.StartedAt,
		EndedAt:        event.EndedAt,
		Time:           event.LastEvalAt,
	}
}

func buildRuleNotificationEventPayload(event *model.AlertEvent) ruleNotificationEventPayload {
	if event == nil {
		return ruleNotificationEventPayload{}
	}
	labels := parseStringMap(event.Labels)
	annotations := parseStringMap(event.Annotations)
	payload := buildRuleNotificationPayload(event)
	return ruleNotificationEventPayload{
		RuleID:         event.RuleID,
		EventID:        event.ID,
		RuleName:       event.RuleName,
		FaultCenterID:  event.FaultCenterID,
		DataSourceID:   event.DataSourceID,
		DataSourceName: event.DataSourceName,
		DataSourceType: event.DataSourceType,
		Severity:       event.Severity,
		State:          event.State,
		Value:          event.Value,
		Condition:      event.Condition,
		Threshold:      event.Threshold,
		Message:        event.Message,
		Labels:         labels,
		Annotations:    annotations,
		AnnotationText: noticeAnnotationText(event.Annotations, annotations),
		Fingerprint:    event.Fingerprint,
		Instance:       notificationEventInstance(event),
		EventURL:       buildNoticeEventURL(payload),
		StartedAt:      event.StartedAt,
		EndedAt:        event.EndedAt,
		Time:           event.LastEvalAt,
	}
}

func buildAggregatedRuleNotificationPayload(events []*model.AlertEvent) ruleNotificationPayload {
	events = compactNotificationEvents(events)
	if len(events) == 0 {
		return ruleNotificationPayload{}
	}
	sortNotificationEvents(events)
	representative := representativeNotificationEvent(events)
	payload := buildRuleNotificationPayload(representative)
	payload.Aggregated = len(events) > 1
	payload.EventCount = len(events)
	payload.EventID = 0
	payload.Fingerprint = representative.Fingerprint
	payload.Labels = marshalStringMap(aggregateNotificationLabels(representative, events))
	payload.Annotations = marshalStringMap(aggregateNotificationAnnotations(representative, len(events)))
	payload.Message = aggregateNotificationMessage(representative, len(events))
	payload.Value = representative.Value
	payload.Severity = representative.Severity
	payload.State = representative.State
	payload.StartedAt = representative.StartedAt
	payload.Time = representative.LastEvalAt
	payload.EndedAt = latestNotificationEndedAt(events)
	payload.Events = make([]ruleNotificationEventPayload, 0, len(events))
	for _, event := range events {
		payload.Events = append(payload.Events, buildRuleNotificationEventPayload(event))
	}
	return payload
}

func compactNotificationEvents(events []*model.AlertEvent) []*model.AlertEvent {
	result := make([]*model.AlertEvent, 0, len(events))
	seen := map[uint]struct{}{}
	for _, event := range events {
		if event == nil {
			continue
		}
		if event.ID > 0 {
			if _, ok := seen[event.ID]; ok {
				continue
			}
			seen[event.ID] = struct{}{}
		}
		result = append(result, event)
	}
	return result
}

func sortNotificationEvents(events []*model.AlertEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		left := events[i]
		right := events[j]
		if left == nil || right == nil {
			return right != nil
		}
		if severityRank(left.Severity) != severityRank(right.Severity) {
			return severityRank(left.Severity) < severityRank(right.Severity)
		}
		leftInstance := notificationEventInstance(left)
		rightInstance := notificationEventInstance(right)
		if leftInstance != rightInstance {
			return leftInstance < rightInstance
		}
		return left.ID < right.ID
	})
}

func representativeNotificationEvent(events []*model.AlertEvent) *model.AlertEvent {
	if len(events) == 0 {
		return nil
	}
	sortNotificationEvents(events)
	return events[0]
}

func notificationEventInstance(event *model.AlertEvent) string {
	if event == nil {
		return "-"
	}
	labels := parseStringMap(event.Labels)
	for _, key := range []string{"instance", "host", "hostname", "pod", "container", "endpoint", "job"} {
		if value := strings.TrimSpace(labels[key]); value != "" {
			return value
		}
	}
	return firstNonEmpty(event.DataSourceName, event.RuleName, "-")
}

func aggregateNotificationLabels(representative *model.AlertEvent, events []*model.AlertEvent) map[string]string {
	labels := map[string]string{}
	if representative != nil {
		for key, value := range parseStringMap(representative.Labels) {
			labels[key] = value
		}
	}
	instances := aggregateNotificationInstances(events)
	instanceText := strings.Join(instances, "、")
	if instanceText == "" {
		instanceText = "-"
	}
	labels["instances"] = instanceText
	labels["event_count"] = strconv.Itoa(len(events))
	return labels
}

func aggregateNotificationInstances(events []*model.AlertEvent) []string {
	instances := make([]string, 0, len(events))
	seen := map[string]struct{}{}
	for _, event := range events {
		instance := notificationEventInstance(event)
		if instance == "" || instance == "-" {
			continue
		}
		if _, ok := seen[instance]; ok {
			continue
		}
		seen[instance] = struct{}{}
		instances = append(instances, instance)
	}
	sort.Strings(instances)
	return instances
}

func aggregateNotificationAnnotation(representative *model.AlertEvent, count int) string {
	if representative == nil {
		return aggregationNoticeText(count)
	}
	annotations := parseStringMap(representative.Annotations)
	text := firstNonEmpty(noticeAnnotationText(representative.Annotations, annotations), representative.Message)
	if notice := aggregationNoticeText(count); notice != "" {
		text = strings.TrimSpace(text + notice)
	}
	return text
}

func aggregateNotificationAnnotations(representative *model.AlertEvent, count int) map[string]string {
	annotations := map[string]string{
		"description": aggregateNotificationAnnotation(representative, count),
	}
	if representative == nil {
		return annotations
	}
	source := parseStringMap(representative.Annotations)
	carryMatchedLogAnnotations(annotations, source)
	return annotations
}

func aggregateNotificationMessage(representative *model.AlertEvent, count int) string {
	if representative == nil {
		return ""
	}
	message := representative.Message
	if notice := aggregationNoticeText(count); notice != "" {
		message = strings.TrimSpace(message + notice)
	}
	return message
}

func aggregationNoticeText(count int) string {
	if count <= 1 {
		return ""
	}
	return fmt.Sprintf("\n聚合 %d 条消息，详情请前往 OpsHub 查看", count)
}

func latestNotificationEndedAt(events []*model.AlertEvent) *time.Time {
	var latest *time.Time
	for _, event := range events {
		if event == nil || event.EndedAt == nil {
			continue
		}
		if latest == nil || event.EndedAt.After(*latest) {
			value := *event.EndedAt
			latest = &value
		}
	}
	return latest
}

func ruleNotificationEventToPayload(base ruleNotificationPayload, event ruleNotificationEventPayload) ruleNotificationPayload {
	payload := base
	payload.EventID = event.EventID
	payload.DataSourceID = event.DataSourceID
	payload.DataSourceName = event.DataSourceName
	payload.DataSourceType = event.DataSourceType
	payload.Severity = event.Severity
	payload.State = event.State
	payload.Value = event.Value
	payload.Condition = event.Condition
	payload.Threshold = event.Threshold
	payload.Message = event.Message
	payload.Labels = marshalStringMap(event.Labels)
	payload.Annotations = marshalStringMap(event.Annotations)
	payload.Fingerprint = event.Fingerprint
	payload.StartedAt = event.StartedAt
	payload.EndedAt = event.EndedAt
	payload.Time = event.Time
	payload.Aggregated = false
	payload.EventCount = 0
	payload.Events = nil
	return payload
}

type noticeObjectRouteConfig struct {
	NoticeType       string              `json:"noticeType"`
	NoticeTemplateID interface{}         `json:"noticeTemplateId"`
	Severitys        []string            `json:"severitys"`
	Hook             string              `json:"hook"`
	FeishuAppID      string              `json:"feishuAppId"`
	FeishuAppSecret  string              `json:"feishuAppSecret"`
	Headers          map[string]string   `json:"headers"`
	Sign             string              `json:"sign"`
	Subject          string              `json:"subject"`
	SMTPHost         string              `json:"smtpHost"`
	SMTPPort         int                 `json:"smtpPort"`
	SMTPUser         string              `json:"smtpUser"`
	SMTPPassword     string              `json:"smtpPassword"`
	FromEmail        string              `json:"fromEmail"`
	FromName         string              `json:"fromName"`
	To               []string            `json:"to"`
	CC               []string            `json:"cc"`
	EffectiveTime    noticeEffectiveTime `json:"effectiveTime"`
	Enabled          *bool               `json:"enabled"`
}

type noticeEffectiveTime struct {
	Week      []string    `json:"week"`
	StartTime interface{} `json:"startTime"`
	EndTime   interface{} `json:"endTime"`
}

type faultCenterNoticeRouteConfig struct {
	Name            string        `json:"name"`
	Matcher         string        `json:"matcher"`
	NoticeObjectIDs []interface{} `json:"noticeObjectIds"`
	Enabled         *bool         `json:"enabled"`
}

func (h *DataSourceHandler) sendRuleNotifications(ctx context.Context, rule *model.AlertRule, event *model.AlertEvent) (string, error) {
	return h.sendRuleNotificationsWithPayload(ctx, rule, buildRuleNotificationPayload(event))
}

func (h *DataSourceHandler) sendRuleNotificationsWithPayload(ctx context.Context, rule *model.AlertRule, payload ruleNotificationPayload) (string, error) {
	var center *model.FaultCenter
	if rule.FaultCenterID > 0 {
		var item model.FaultCenter
		if err := h.db.First(&item, rule.FaultCenterID).Error; err == nil {
			center = &item
		}
	}

	var statuses []string
	var errors []string
	if center != nil {
		objectIDs := notificationObjectIDsForFaultCenter(*center, payload)
		if len(objectIDs) > 0 {
			status, err := h.sendRuleNotificationsToNoticeObjects(ctx, objectIDs, payload)
			statuses = append(statuses, status)
			if err != nil {
				errors = append(errors, err.Error())
			}
		}
	}

	ids, err := parseRuleChannelIDs(rule.ChannelIDs)
	if err != nil {
		return "failed", err
	}
	if len(ids) == 0 && center != nil {
		ids, err = parseRuleChannelIDs(center.NoticeChannelIDs)
		if err != nil {
			return "failed", err
		}
	}
	if len(ids) > 0 {
		status, err := h.sendRuleNotificationsToChannels(ctx, ids, payload)
		statuses = append(statuses, status)
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	return mergeNotifyStatus(statuses, errors)
}

func (h *DataSourceHandler) sendAndRecordRuleNotifications(ctx context.Context, rule *model.AlertRule, events []*model.AlertEvent) bool {
	events = compactNotificationEvents(events)
	if len(events) == 0 {
		return false
	}
	h.ensureLokiMatchedLogsForNotificationEvents(ctx, events)

	if h.shouldAggregateRuleNotifications(rule, events) {
		payload := buildAggregatedRuleNotificationPayload(events)
		status, notifyErr := h.sendRuleNotificationsWithPayload(ctx, rule, payload)
		h.recordRuleNotificationResult(events, status, notifyErr)
		return status != "none"
	}

	notified := false
	for _, event := range events {
		status, notifyErr := h.sendRuleNotifications(ctx, rule, event)
		h.recordRuleNotificationResult([]*model.AlertEvent{event}, status, notifyErr)
		if status != "none" {
			notified = true
		}
	}
	return notified
}

func (h *DataSourceHandler) shouldAggregateRuleNotifications(rule *model.AlertRule, events []*model.AlertEvent) bool {
	if len(events) < 2 || rule == nil || rule.FaultCenterID == 0 {
		return false
	}
	var center model.FaultCenter
	if err := h.db.Select("id, aggregation_type").First(&center, rule.FaultCenterID).Error; err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(center.AggregationType), "Rule")
}

type faultCenterUpgradeStrategyConfig struct {
	Enabled         *bool         `json:"enabled"`
	Timeout         int           `json:"timeout"`
	RepeatInterval  int           `json:"repeatInterval"`
	NoticeObjectIDs []interface{} `json:"noticeObjectIds"`
}

func (h *DataSourceHandler) processFaultCenterEscalations(ctx context.Context, rule *model.AlertRule, events []*model.AlertEvent, now time.Time) {
	events = compactNotificationEvents(events)
	if rule == nil || rule.FaultCenterID == 0 || len(events) == 0 {
		return
	}
	var center model.FaultCenter
	if err := h.db.First(&center, rule.FaultCenterID).Error; err != nil {
		return
	}
	strategy, ok := parseFaultCenterUpgradeStrategy(center.UpgradeStrategy)
	if !center.UpgradeEnabled || !ok || (strategy.Enabled != nil && !*strategy.Enabled) {
		return
	}
	if strategy.Timeout <= 0 {
		strategy.Timeout = 30
	}
	if strategy.RepeatInterval <= 0 {
		strategy.RepeatInterval = 60
	}
	objectIDs := uintIDsFromValues(strategy.NoticeObjectIDs)
	if len(objectIDs) == 0 {
		objectIDs, _ = parseRuleChannelIDs(center.NoticeObjectIDs)
	}
	objectIDs = uniqueUintIDs(objectIDs)
	if len(objectIDs) == 0 {
		return
	}

	dueEvents := make([]*model.AlertEvent, 0, len(events))
	for _, event := range events {
		if shouldEscalateAlertEvent(center, strategy, event, now) {
			dueEvents = append(dueEvents, event)
		}
	}
	if len(dueEvents) == 0 {
		return
	}
	h.ensureLokiMatchedLogsForNotificationEvents(ctx, dueEvents)

	if strings.EqualFold(strings.TrimSpace(center.AggregationType), "Rule") && len(dueEvents) > 1 {
		payload := buildAggregatedRuleNotificationPayload(dueEvents)
		markEscalationPayload(&payload, strategy.Timeout)
		status, notifyErr := h.sendRuleNotificationsToNoticeObjects(ctx, objectIDs, payload)
		h.recordEscalationResult(dueEvents, now, status, notifyErr)
		return
	}
	for _, event := range dueEvents {
		payload := buildRuleNotificationPayload(event)
		markEscalationPayload(&payload, strategy.Timeout)
		status, notifyErr := h.sendRuleNotificationsToNoticeObjects(ctx, objectIDs, payload)
		h.recordEscalationResult([]*model.AlertEvent{event}, now, status, notifyErr)
	}
}

func parseFaultCenterUpgradeStrategy(raw string) (faultCenterUpgradeStrategyConfig, bool) {
	var strategy faultCenterUpgradeStrategyConfig
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &strategy); err != nil {
		return strategy, false
	}
	return strategy, true
}

func shouldEscalateAlertEvent(center model.FaultCenter, strategy faultCenterUpgradeStrategyConfig, event *model.AlertEvent, now time.Time) bool {
	if event == nil || event.EndedAt != nil {
		return false
	}
	if event.Acknowledged || event.State == "silenced" || event.State == "processing" || event.State == "pending" {
		return false
	}
	if event.State != "firing" && event.State != "error" {
		return false
	}
	if !faultCenterSeverityUpgradable(center.UpgradableSeverities, event.Severity) {
		return false
	}
	timeout := time.Duration(strategy.Timeout) * time.Minute
	if timeout <= 0 || now.Sub(event.StartedAt) < timeout {
		return false
	}
	repeat := time.Duration(strategy.RepeatInterval) * time.Minute
	if event.LastEscalateAt == nil {
		return true
	}
	return repeat > 0 && !event.LastEscalateAt.Add(repeat).After(now)
}

func faultCenterSeverityUpgradable(raw string, severity string) bool {
	var levels []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &levels); err != nil || len(levels) == 0 {
		levels = []string{"p0", "p1"}
	}
	target := normalizeSeverityLevel(severity)
	for _, level := range levels {
		if normalizeSeverityLevel(level) == target {
			return true
		}
	}
	return false
}

func markEscalationPayload(payload *ruleNotificationPayload, timeoutMinutes int) {
	if payload == nil {
		return
	}
	payload.Escalated = true
	payload.EscalationText = fmt.Sprintf("告警已持续超过 %d 分钟，触发升级通知", timeoutMinutes)
	if strings.TrimSpace(payload.Message) != "" && !strings.Contains(payload.Message, "告警升级") {
		payload.Message = "告警升级：" + payload.Message
	}
	if payload.Aggregated {
		for i := range payload.Events {
			if !strings.Contains(payload.Events[i].Message, "告警升级") {
				payload.Events[i].Message = "告警升级：" + payload.Events[i].Message
			}
		}
	}
}

func (h *DataSourceHandler) recordEscalationResult(events []*model.AlertEvent, now time.Time, status string, notifyErr error) {
	for _, event := range events {
		if event == nil {
			continue
		}
		if event.EscalatedAt == nil {
			event.EscalatedAt = &now
		}
		event.Escalated = true
		event.LastEscalateAt = &now
		event.EscalateStatus = status
		if notifyErr != nil {
			event.EscalateError = notifyErr.Error()
		} else {
			event.EscalateError = ""
		}
		_ = h.db.Save(event).Error
	}
}

func (h *DataSourceHandler) recordRuleNotificationResult(events []*model.AlertEvent, status string, notifyErr error) {
	for _, event := range events {
		if event == nil {
			continue
		}
		event.NotifyStatus = status
		if notifyErr != nil {
			event.NotifyError = notifyErr.Error()
		} else {
			event.NotifyError = ""
		}
		_ = h.db.Save(event).Error
	}
}

func (h *DataSourceHandler) buildNotificationCallbackContextForPayload(ctx context.Context, payload ruleNotificationPayload) notificationCallbackContext {
	if payload.Aggregated && len(payload.Events) > 0 {
		return h.buildAggregatedNotificationCallbackContext(ctx, payload)
	}
	return h.buildNotificationCallbackContext(ctx, payload)
}

func (h *DataSourceHandler) buildNotificationCallbackContext(ctx context.Context, payload ruleNotificationPayload) notificationCallbackContext {
	context := notificationCallbackContext{}
	if payload.EventID == 0 || payload.RuleID == 0 {
		return context
	}

	var event model.AlertEvent
	if err := h.db.First(&event, payload.EventID).Error; err != nil {
		return context
	}
	var rule model.AlertRule
	if err := h.db.First(&rule, payload.RuleID).Error; err != nil {
		return context
	}

	callbacks := parseAlertCallbackQueries(rule.CallbackQueries)
	items := make([]notificationCallbackItem, 0, len(callbacks))
	for _, callback := range callbacks {
		if callback.Enabled != nil && !*callback.Enabled {
			continue
		}
		queryText := strings.TrimSpace(firstNonEmpty(callback.Query, callback.Value))
		name := strings.TrimSpace(firstNonEmpty(callback.Name, callback.Key))
		if name == "" {
			name = "关联查询"
		}
		item := notificationCallbackItem{
			Key:          callback.Key,
			Name:         name,
			Query:        queryText,
			DataSourceID: callback.DataSourceID,
			QueryMode:    strings.ToLower(strings.TrimSpace(callback.QueryMode)),
		}
		if item.QueryMode == "" {
			item.QueryMode = "instant"
		}
		if queryText == "" {
			item.Status = "failed"
			item.Error = "查询语句为空"
			items = append(items, item)
			continue
		}

		dsID := callback.DataSourceID
		if dsID == 0 {
			dsID = rule.DataSourceID
		}
		var ds model.DataSource
		if err := h.db.First(&ds, dsID).Error; err != nil {
			item.Status = "failed"
			item.Error = "数据源不存在"
			items = append(items, item)
			continue
		}

		req := buildCallbackQueryRequest(callback, queryText, rule, event)
		item.RenderedQuery = req.Query
		item.QueryMode = req.QueryMode
		item.DataSourceID = ds.ID
		item.DataSourceName = ds.Name
		item.DataSourceType = ds.Type
		rawResult, _, err := h.queryDataSource(ctx, &ds, req)
		if err != nil {
			item.Status = "failed"
			item.Error = err.Error()
		} else {
			item.Status = "success"
			item.ValueText = callbackResultValueText(ds.Type, rawResult)
			item.Result = rawResult
			if imageItem := buildCallbackChartImage(item, rawResult, event); imageItem != nil {
				item.Image = imageItem
				context.Images = append(context.Images, *imageItem)
			}
		}
		items = append(items, item)
	}

	context.Items = items
	context.Summary = formatCallbackSummary(items)
	context.DetailText = formatCallbackDetailText(items, buildNoticeEventURL(payload))
	return context
}

func (h *DataSourceHandler) buildAggregatedNotificationCallbackContext(ctx context.Context, payload ruleNotificationPayload) notificationCallbackContext {
	context := notificationCallbackContext{}
	if len(payload.Events) == 0 {
		return context
	}
	summaryCount := 0
	detailLines := make([]string, 0, len(payload.Events))
	for _, event := range payload.Events {
		singlePayload := ruleNotificationEventToPayload(payload, event)
		itemContext := h.buildNotificationCallbackContext(ctx, singlePayload)
		if len(itemContext.Items) == 0 {
			continue
		}
		instance := firstNonEmpty(event.Instance, "-")
		for _, imageItem := range itemContext.Images {
			if instance != "" && instance != "-" {
				imageItem.Title = fmt.Sprintf("%s / %s", instance, imageItem.Title)
			}
			context.Images = append(context.Images, imageItem)
		}
		detail := formatCallbackItemsInline(itemContext.Items)
		if detail == "" {
			continue
		}
		summaryCount++
		detailLines = append(detailLines, fmt.Sprintf("- **%s**：%s", instance, detail))
	}
	if summaryCount > 0 {
		context.Summary = fmt.Sprintf("%d 个实例已完成回调查询", summaryCount)
	} else {
		context.Summary = "未配置"
	}
	if len(detailLines) > 0 {
		context.DetailText = strings.Join(detailLines, "\n")
	} else {
		context.DetailText = "未配置回调查询"
	}
	return context
}

func formatCallbackItemsInline(items []notificationCallbackItem) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		name := firstNonEmpty(item.Name, item.Key, "关联查询")
		if item.Status == "success" {
			parts = append(parts, fmt.Sprintf("%s=%s", name, firstNonEmpty(item.ValueText, "-")))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s失败（%s）", name, firstNonEmpty(item.Error, "未知错误")))
	}
	return strings.Join(parts, "；")
}

func notificationObjectIDsForFaultCenter(center model.FaultCenter, payload ruleNotificationPayload) []uint {
	if payload.Aggregated && len(payload.Events) > 0 {
		var ids []uint
		for _, event := range payload.Events {
			ids = append(ids, matchedNoticeRouteObjectIDs(center.NoticeRoutes, ruleNotificationEventToPayload(payload, event))...)
		}
		if len(ids) > 0 {
			return uniqueUintIDs(ids)
		}
	}
	matchedIDs := matchedNoticeRouteObjectIDs(center.NoticeRoutes, payload)
	if len(matchedIDs) > 0 {
		return matchedIDs
	}
	defaultIDs, err := parseRuleChannelIDs(center.NoticeObjectIDs)
	if err != nil {
		return []uint{}
	}
	return defaultIDs
}

func callbackResultValueText(dsType string, raw interface{}) string {
	switch dsType {
	case "prometheus", "victoriametrics", "loki", "elasticsearch", "opensearch":
		if value, err := extractRuleNumericValue(dsType, raw); err == nil {
			return formatRuleValue(value)
		}
	}
	return summarizeCallbackResult(raw)
}

type callbackChartPoint struct {
	Time  float64
	Value float64
}

type callbackChartSeries struct {
	Name   string
	Points []callbackChartPoint
}

func buildCallbackChartImage(item notificationCallbackItem, raw interface{}, event model.AlertEvent) *notificationCallbackImage {
	switch strings.ToLower(strings.TrimSpace(item.DataSourceType)) {
	case "prometheus", "victoriametrics":
	default:
		return nil
	}
	series := extractCallbackChartSeries(raw)
	if len(series) == 0 {
		return nil
	}
	imageBytes, err := renderCallbackChartPNG(series, item, event)
	if err != nil || len(imageBytes) == 0 {
		return nil
	}
	return &notificationCallbackImage{
		Title: firstNonEmpty(item.Name, item.Key, "回调查询"),
		Query: firstNonEmpty(item.RenderedQuery, item.Query),
		PNG:   imageBytes,
	}
}

func extractCallbackChartSeries(raw interface{}) []callbackChartSeries {
	root, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	data, _ := root["data"].(map[string]interface{})
	if data == nil {
		data = root
	}
	result, _ := data["result"].([]interface{})
	if len(result) == 0 {
		return nil
	}
	series := make([]callbackChartSeries, 0, len(result))
	for index, item := range result {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		points := make([]callbackChartPoint, 0)
		if values, ok := entry["values"].([]interface{}); ok {
			for _, rawPair := range values {
				if point, ok := callbackChartPointFromPair(rawPair); ok {
					points = append(points, point)
				}
			}
		} else if value, ok := callbackChartPointFromPair(entry["value"]); ok {
			points = append(points, value)
		}
		if len(points) == 0 {
			continue
		}
		sort.Slice(points, func(i, j int) bool { return points[i].Time < points[j].Time })
		series = append(series, callbackChartSeries{
			Name:   firstNonEmpty(callbackChartSeriesName(labelsFromInterfaceMap(entry["metric"]), index), fmt.Sprintf("Series %d", index+1)),
			Points: points,
		})
	}
	return series
}

func callbackChartPointFromPair(raw interface{}) (callbackChartPoint, bool) {
	pair, ok := raw.([]interface{})
	if !ok || len(pair) < 2 {
		return callbackChartPoint{}, false
	}
	timestamp, err := parseFloat(pair[0])
	if err != nil {
		return callbackChartPoint{}, false
	}
	value, err := parseFloat(pair[1])
	if err != nil || !isFiniteFloat(value) {
		return callbackChartPoint{}, false
	}
	return callbackChartPoint{Time: timestamp, Value: value}, true
}

func callbackChartSeriesName(labels map[string]string, index int) string {
	for _, key := range []string{"instance", "pod", "container", "job", "namespace", "__name__"} {
		if value := strings.TrimSpace(labels[key]); value != "" {
			return value
		}
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, 2)
	for _, key := range keys {
		if len(parts) >= 2 {
			break
		}
		if value := strings.TrimSpace(labels[key]); value != "" {
			parts = append(parts, key+"="+value)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return fmt.Sprintf("Series %d", index+1)
}

func renderCallbackChartPNG(series []callbackChartSeries, item notificationCallbackItem, event model.AlertEvent) ([]byte, error) {
	const (
		width  = 940
		height = 430
		left   = 72
		right  = 32
		top    = 126
		bottom = 86
	)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)

	titleFace := callbackChartFontFace(19)
	labelFace := callbackChartFontFace(12)
	smallFace := callbackChartFontFace(11)
	defer closeFontFace(titleFace)
	defer closeFontFace(labelFace)
	defer closeFontFace(smallFace)

	unit := inferCallbackMetricUnit(firstNonEmpty(item.Name, item.Key) + " " + firstNonEmpty(item.RenderedQuery, item.Query))
	title := clipTextToWidth(firstNonEmpty(item.Name, item.Key, "回调查询"), titleFace, 430)
	latestText := "-"
	if latest, ok := callbackLatestValue(series); ok {
		latestText = formatCallbackMetricValue(latest, unit)
	}
	thresholdText := ""
	if threshold := event.Threshold; isFiniteFloat(threshold) {
		thresholdText = getConditionText(event.Condition) + " " + formatCallbackMetricValue(threshold, unit)
	}
	drawChartText(img, "指标趋势", 26, 30, labelFace, color.RGBA{R: 102, G: 112, B: 133, A: 255})
	drawChartText(img, title, 26, 58, titleFace, color.RGBA{R: 17, G: 24, B: 39, A: 255})
	badgeRight := width - 26
	if thresholdText != "" {
		badgeRight = drawChartBadge(img, "阈值 "+thresholdText, badgeRight, 24, labelFace)
	}
	badgeRight = drawChartBadge(img, fmt.Sprintf("序列 %d", len(series)), badgeRight, 24, labelFace)
	drawChartBadge(img, "最新值 "+latestText, badgeRight, 24, labelFace)

	plotLeft := left
	plotRight := width - right
	plotTop := top
	plotBottom := height - bottom
	axisColor := color.RGBA{R: 209, G: 217, B: 230, A: 255}
	gridColor := color.RGBA{R: 237, G: 241, B: 247, A: 255}
	textColor := color.RGBA{R: 102, G: 112, B: 133, A: 255}

	minTime, maxTime, rawMinValue, rawMaxValue := callbackChartBounds(series)
	if minTime == maxTime {
		minTime -= 60
		maxTime += 60
	}
	scale := callbackChartScale(rawMinValue, rawMaxValue, event, unit)
	minValue := scale.Min
	maxValue := scale.Max

	for _, value := range scale.Ticks {
		y := callbackValueToY(value, minValue, maxValue, plotTop, plotBottom)
		drawLine(img, plotLeft, y, plotRight, y, gridColor)
		label := formatCallbackAxisValue(value, unit)
		drawChartText(img, label, plotLeft-measureChartText(label, smallFace)-12, y+5, smallFace, textColor)
	}
	for i := 0; i <= 4; i++ {
		x := plotLeft + (plotRight-plotLeft)*i/4
		drawLine(img, x, plotTop, x, plotBottom, gridColor)
		timestamp := minTime + (maxTime-minTime)*float64(i)/4
		label := formatCallbackAxisTime(timestamp)
		labelX := x - measureChartText(label, smallFace)/2
		if i == 0 {
			labelX = plotLeft
		} else if i == 4 {
			labelX = plotRight - measureChartText(label, smallFace)
		}
		drawChartText(img, label, labelX, plotBottom+28, smallFace, textColor)
	}
	drawLine(img, plotLeft, plotBottom, plotRight, plotBottom, axisColor)
	drawLine(img, plotLeft, plotTop, plotLeft, plotBottom, axisColor)
	timeRange := fmt.Sprintf("%s - %s", formatCallbackAxisTime(minTime), formatCallbackAxisTime(maxTime))
	drawChartText(img, timeRange, 26, height-18, labelFace, color.RGBA{R: 152, G: 162, B: 179, A: 255})
	unitText := "单位：" + unit.Label
	drawChartText(img, unitText, width-26-measureChartText(unitText, labelFace), height-18, labelFace, color.RGBA{R: 152, G: 162, B: 179, A: 255})

	valueToY := func(value float64) int {
		return callbackValueToY(value, minValue, maxValue, plotTop, plotBottom)
	}
	timeToX := func(timestamp float64) int {
		ratio := (timestamp - minTime) / (maxTime - minTime)
		return plotLeft + int(math.Round(ratio*float64(plotRight-plotLeft)))
	}

	if threshold := event.Threshold; isFiniteFloat(threshold) {
		y := valueToY(threshold)
		drawDashedLine(img, plotLeft, y, plotRight, y, color.RGBA{R: 220, G: 38, B: 38, A: 255})
		if thresholdText != "" {
			labelY := y - 8
			if labelY < plotTop+12 {
				labelY = y + 18
			}
			drawChartText(img, thresholdText, plotRight-measureChartText(thresholdText, smallFace)-8, labelY, smallFace, color.RGBA{R: 220, G: 38, B: 38, A: 255})
		}
	}
	palette := []color.RGBA{
		{R: 37, G: 99, B: 235, A: 255},
		{R: 22, G: 163, B: 74, A: 255},
		{R: 217, G: 119, B: 6, A: 255},
		{R: 220, G: 38, B: 38, A: 255},
		{R: 124, G: 58, B: 237, A: 255},
		{R: 8, G: 145, B: 178, A: 255},
	}
	for index, item := range series {
		if index >= 8 {
			break
		}
		lineColor := palette[index%len(palette)]
		if index < 4 {
			legendX := plotLeft + index*210
			legendY := 96
			fillCircle(img, legendX, legendY-4, 5, lineColor)
			drawChartText(img, clipTextToWidth(item.Name, smallFace, 150), legendX+14, legendY, smallFace, textColor)
		}
		for pointIndex, point := range item.Points {
			x := timeToX(point.Time)
			y := valueToY(point.Value)
			fillCircle(img, x, y, 3, lineColor)
			if pointIndex == 0 {
				continue
			}
			prev := item.Points[pointIndex-1]
			drawThickLine(img, timeToX(prev.Time), valueToY(prev.Value), x, y, lineColor, 2)
		}
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type callbackMetricUnit struct {
	Key      string
	Label    string
	AxisName string
}

type callbackChartScaleValue struct {
	Min   float64
	Max   float64
	Ticks []float64
}

func callbackChartScale(minValue, maxValue float64, event model.AlertEvent, unit callbackMetricUnit) callbackChartScaleValue {
	if threshold := event.Threshold; isFiniteFloat(threshold) {
		minValue = math.Min(minValue, threshold)
		maxValue = math.Max(maxValue, threshold)
	}
	if !isFiniteFloat(minValue) || !isFiniteFloat(maxValue) {
		minValue, maxValue = 0, 1
	}
	nonNegative := minValue >= 0 || unit.Key == "percent" || unit.Key == "bytes" || unit.Key == "count"
	if nonNegative {
		minValue = 0
	}
	if unit.Key == "percent" && maxValue <= 100 {
		maxValue = 100
	}
	if minValue == maxValue {
		if maxValue == 0 {
			maxValue = 1
		} else {
			maxValue += math.Abs(maxValue) * 0.15
			if !nonNegative {
				minValue -= math.Abs(minValue) * 0.15
			}
		}
	}
	rawRange := maxValue - minValue
	if rawRange <= 0 || !isFiniteFloat(rawRange) {
		rawRange = 1
	}
	step := niceChartStep(rawRange / 4)
	if step <= 0 || !isFiniteFloat(step) {
		step = 1
	}
	niceMin := math.Floor(minValue/step) * step
	niceMax := math.Ceil(maxValue/step) * step
	if nonNegative && niceMin < 0 {
		niceMin = 0
	}
	if niceMin == niceMax {
		niceMax = niceMin + step
	}
	ticks := make([]float64, 0, 8)
	for value := niceMin; value <= niceMax+step*0.5 && len(ticks) < 8; value += step {
		if math.Abs(value) < step/1000000 {
			value = 0
		}
		ticks = append(ticks, value)
	}
	if len(ticks) < 2 {
		ticks = []float64{niceMin, niceMax}
	}
	return callbackChartScaleValue{Min: niceMin, Max: niceMax, Ticks: ticks}
}

func niceChartStep(value float64) float64 {
	if value <= 0 || !isFiniteFloat(value) {
		return 1
	}
	exponent := math.Floor(math.Log10(value))
	fraction := value / math.Pow(10, exponent)
	niceFraction := 1.0
	switch {
	case fraction <= 1:
		niceFraction = 1
	case fraction <= 2:
		niceFraction = 2
	case fraction <= 5:
		niceFraction = 5
	default:
		niceFraction = 10
	}
	return niceFraction * math.Pow(10, exponent)
}

func callbackValueToY(value, minValue, maxValue float64, plotTop, plotBottom int) int {
	if maxValue == minValue {
		return plotBottom
	}
	ratio := (value - minValue) / (maxValue - minValue)
	ratio = math.Max(0, math.Min(1, ratio))
	return plotBottom - int(math.Round(ratio*float64(plotBottom-plotTop)))
}

func inferCallbackMetricUnit(text string) callbackMetricUnit {
	source := strings.ToLower(text)
	compact := strings.ReplaceAll(source, " ", "")
	if strings.Contains(source, "%") ||
		strings.Contains(source, "使用率") ||
		strings.Contains(source, "利用率") ||
		strings.Contains(source, "percent") ||
		strings.Contains(source, "percentage") ||
		strings.Contains(source, "utilization") ||
		strings.Contains(compact, "*100") {
		return callbackMetricUnit{Key: "percent", Label: "%", AxisName: "单位：%"}
	}
	if strings.Contains(source, "bytes") || strings.Contains(source, "_bytes") || strings.Contains(source, "byte") {
		return callbackMetricUnit{Key: "bytes", Label: "bytes", AxisName: "单位：bytes"}
	}
	if strings.Contains(source, "milliseconds") || strings.Contains(source, "_ms") || strings.Contains(source, "毫秒") {
		return callbackMetricUnit{Key: "milliseconds", Label: "ms", AxisName: "单位：ms"}
	}
	if strings.Contains(source, "seconds") || strings.Contains(source, "_seconds") || strings.Contains(source, "耗时") || strings.Contains(source, "延迟") {
		return callbackMetricUnit{Key: "seconds", Label: "s", AxisName: "单位：s"}
	}
	if strings.Contains(source, "count") || strings.Contains(source, "total") || strings.Contains(source, "次数") {
		return callbackMetricUnit{Key: "count", Label: "count", AxisName: "单位：count"}
	}
	return callbackMetricUnit{Key: "none", Label: "value", AxisName: "单位：数值"}
}

func callbackLatestValue(series []callbackChartSeries) (float64, bool) {
	values := make([]float64, 0, len(series))
	hasLineData := false
	for _, item := range series {
		if len(item.Points) > 1 {
			hasLineData = true
		}
		if len(item.Points) == 0 {
			continue
		}
		value := item.Points[len(item.Points)-1].Value
		if isFiniteFloat(value) {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return 0, false
	}
	if !hasLineData {
		return values[0], true
	}
	latest := values[0]
	for _, value := range values[1:] {
		latest = math.Max(latest, value)
	}
	return latest, true
}

func formatCallbackMetricValue(value float64, unit callbackMetricUnit) string {
	if !isFiniteFloat(value) {
		return "-"
	}
	switch unit.Key {
	case "percent":
		return formatCompactFloat(value) + "%"
	case "bytes":
		return formatCallbackBytes(value)
	case "seconds":
		return formatCompactFloat(value) + "s"
	case "milliseconds":
		return formatCompactFloat(value) + "ms"
	case "count":
		return formatCompactCount(value)
	default:
		return formatCompactFloat(value)
	}
}

func formatCallbackAxisValue(value float64, unit callbackMetricUnit) string {
	if !isFiniteFloat(value) {
		return "-"
	}
	switch unit.Key {
	case "percent":
		return formatCompactFloat(value) + "%"
	case "bytes":
		return formatCallbackBytes(value)
	case "seconds":
		return formatCompactFloat(value) + "s"
	case "milliseconds":
		return formatCompactFloat(value) + "ms"
	case "count":
		return formatCompactCount(value)
	default:
		return formatCompactFloat(value)
	}
}

func formatCompactCount(value float64) string {
	if math.Abs(value-math.Round(value)) < 0.000001 {
		return strconv.FormatInt(int64(math.Round(value)), 10)
	}
	return formatCompactFloat(value)
}

func formatCompactFloat(value float64) string {
	if !isFiniteFloat(value) {
		return "-"
	}
	text := strconv.FormatFloat(value, 'f', 2, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "-0" {
		return "0"
	}
	return text
}

func formatCallbackBytes(value float64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	normalized := math.Abs(value)
	index := 0
	for normalized >= 1024 && index < len(units)-1 {
		normalized /= 1024
		index++
	}
	if value < 0 {
		normalized = -normalized
	}
	return formatCompactFloat(normalized) + " " + units[index]
}

func formatCallbackAxisTime(timestamp float64) string {
	if !isFiniteFloat(timestamp) {
		return "-"
	}
	if timestamp > 100000000000 {
		timestamp = timestamp / 1000
	}
	return time.Unix(int64(timestamp), 0).Local().Format("01-02 15:04")
}

func callbackChartFontFace(size float64) font.Face {
	for _, path := range []string{
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/Library/Fonts/Arial Unicode.ttf",
		"/System/Library/Fonts/Hiragino Sans GB.ttc",
		"/System/Library/Fonts/STHeiti Medium.ttc",
		"/System/Library/Fonts/STHeiti Light.ttc",
		"/System/Library/Fonts/SFNS.ttf",
	} {
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			continue
		}
		collection, err := opentype.ParseCollection(data)
		if err != nil {
			continue
		}
		for i := 0; i < collection.NumFonts(); i++ {
			fontItem, err := collection.Font(i)
			if err != nil {
				continue
			}
			face, err := opentype.NewFace(fontItem, &opentype.FaceOptions{
				Size:    size,
				DPI:     96,
				Hinting: font.HintingFull,
			})
			if err == nil {
				return face
			}
		}
	}
	return basicfont.Face7x13
}

func closeFontFace(face font.Face) {
	if closer, ok := face.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

func drawChartText(img *image.RGBA, text string, x, y int, face font.Face, c color.RGBA) {
	if strings.TrimSpace(text) == "" {
		return
	}
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	drawer.DrawString(text)
}

func measureChartText(text string, face font.Face) int {
	return font.MeasureString(face, text).Ceil()
}

func clipTextToWidth(text string, face font.Face, maxWidth int) string {
	text = strings.TrimSpace(text)
	if text == "" || measureChartText(text, face) <= maxWidth {
		return text
	}
	ellipsis := "..."
	runes := []rune(text)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := strings.TrimSpace(string(runes)) + ellipsis
		if measureChartText(candidate, face) <= maxWidth {
			return candidate
		}
	}
	return ellipsis
}

func drawChartBadge(img *image.RGBA, text string, right, top int, face font.Face) int {
	paddingX := 13
	height := 32
	width := measureChartText(text, face) + paddingX*2
	left := right - width
	fillRoundedRect(img, left, top, right, top+height, 16, color.RGBA{R: 248, G: 250, B: 252, A: 255})
	drawChartText(img, text, left+paddingX, top+22, face, color.RGBA{R: 17, G: 24, B: 39, A: 255})
	return left - 8
}

func callbackChartBounds(series []callbackChartSeries) (float64, float64, float64, float64) {
	minTime := math.MaxFloat64
	maxTime := -math.MaxFloat64
	minValue := math.MaxFloat64
	maxValue := -math.MaxFloat64
	for _, item := range series {
		for _, point := range item.Points {
			minTime = math.Min(minTime, point.Time)
			maxTime = math.Max(maxTime, point.Time)
			minValue = math.Min(minValue, point.Value)
			maxValue = math.Max(maxValue, point.Value)
		}
	}
	if minTime == math.MaxFloat64 {
		now := float64(time.Now().Unix())
		return now - 60, now + 60, 0, 1
	}
	return minTime, maxTime, minValue, maxValue
}

func isFiniteFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			setPixel(img, x, y, c)
		}
	}
}

func fillRoundedRect(img *image.RGBA, x0, y0, x1, y1, radius int, c color.RGBA) {
	if radius <= 0 {
		fillRect(img, x0, y0, x1, y1, c)
		return
	}
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	radius = int(math.Min(float64(radius), math.Min(float64(x1-x0), float64(y1-y0))/2))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dx := 0
			if x < x0+radius {
				dx = x0 + radius - x
			} else if x >= x1-radius {
				dx = x - (x1 - radius - 1)
			}
			dy := 0
			if y < y0+radius {
				dy = y0 + radius - y
			} else if y >= y1-radius {
				dy = y - (y1 - radius - 1)
			}
			if dx == 0 || dy == 0 || dx*dx+dy*dy <= radius*radius {
				setPixel(img, x, y, c)
			}
		}
	}
}

func drawRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	drawLine(img, x0, y0, x1, y0, c)
	drawLine(img, x1, y0, x1, y1, c)
	drawLine(img, x1, y1, x0, y1, c)
	drawLine(img, x0, y1, x0, y0, c)
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := int(math.Abs(float64(x1 - x0)))
	dy := -int(math.Abs(float64(y1 - y0)))
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		setPixel(img, x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func drawThickLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA, thickness int) {
	if thickness <= 1 {
		drawLine(img, x0, y0, x1, y1, c)
		return
	}
	for offset := -thickness / 2; offset <= thickness/2; offset++ {
		drawLine(img, x0, y0+offset, x1, y1+offset, c)
		drawLine(img, x0+offset, y0, x1+offset, y1, c)
	}
}

func drawDashedLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dash := 10
	gap := 6
	if x0 == x1 {
		for y := y0; y <= y1; y += dash + gap {
			drawLine(img, x0, y, x1, int(math.Min(float64(y+dash), float64(y1))), c)
		}
		return
	}
	for x := x0; x <= x1; x += dash + gap {
		drawLine(img, x, y0, int(math.Min(float64(x+dash), float64(x1))), y1, c)
	}
}

func fillCircle(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= radius*radius {
				setPixel(img, cx+x, cy+y, c)
			}
		}
	}
}

func setPixel(img *image.RGBA, x, y int, c color.RGBA) {
	if x < img.Bounds().Min.X || x >= img.Bounds().Max.X || y < img.Bounds().Min.Y || y >= img.Bounds().Max.Y {
		return
	}
	img.SetRGBA(x, y, c)
}

func summarizeCallbackResult(raw interface{}) string {
	switch value := raw.(type) {
	case nil:
		return "-"
	case map[string]interface{}:
		if data, ok := value["data"].(map[string]interface{}); ok {
			if result, ok := data["result"].([]interface{}); ok {
				return fmt.Sprintf("%d 条结果", len(result))
			}
		}
		if hits, ok := value["hits"].(map[string]interface{}); ok {
			if total, ok := hits["total"].(map[string]interface{}); ok {
				if number, ok := total["value"]; ok {
					return fmt.Sprintf("%v 条命中", number)
				}
			}
		}
	case []interface{}:
		return fmt.Sprintf("%d 条结果", len(value))
	}
	data, err := json.Marshal(raw)
	if err != nil || len(data) == 0 {
		return "-"
	}
	text := string(data)
	if len(text) > 80 {
		return text[:80] + "..."
	}
	return text
}

func formatCallbackSummary(items []notificationCallbackItem) string {
	if len(items) == 0 {
		return "未配置"
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		name := firstNonEmpty(item.Name, item.Key, "关联查询")
		if item.Status == "success" {
			parts = append(parts, fmt.Sprintf("%s=%s", name, firstNonEmpty(item.ValueText, "-")))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s失败", name))
	}
	return strings.Join(parts, "；")
}

func formatCallbackDetailText(items []notificationCallbackItem, eventURL string) string {
	if len(items) == 0 {
		return "未配置回调查询"
	}
	lines := make([]string, 0, len(items)+1)
	for _, item := range items {
		name := firstNonEmpty(item.Name, item.Key, "关联查询")
		mode := "即时"
		if strings.EqualFold(item.QueryMode, "range") {
			mode = "范围"
		}
		if item.Status == "success" {
			lines = append(lines, fmt.Sprintf("- %s（%s查询）：%s", name, mode, firstNonEmpty(item.ValueText, "-")))
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s（%s查询）：失败，%s", name, mode, firstNonEmpty(item.Error, "未知错误")))
	}
	if strings.TrimSpace(eventURL) != "" {
		lines = append(lines, fmt.Sprintf("[查看事件回调查询](%s)", eventURL))
	}
	return strings.Join(lines, "\n")
}

func buildNoticeEventURL(payload ruleNotificationPayload) string {
	if payload.DataSourceType == opsHubLogDataSourceType {
		if logURL := buildOpsHubLogNoticeURL(payload); logURL != "" {
			return logURL
		}
	}
	path := fmt.Sprintf("/monitor/fault-centers/%d?tab=%s&query=%s", payload.FaultCenterID, noticeEventTab(payload), url.QueryEscape(payload.RuleName))
	if payload.EventID > 0 && !payload.Aggregated {
		path += fmt.Sprintf("&eventId=%d", payload.EventID)
	}
	base := noticeFrontendBaseURL()
	if base == "" {
		return path
	}
	return strings.TrimRight(base, "/") + path
}

func noticeFrontendBaseURL() string {
	return strings.TrimSpace(conf.GetNotificationFrontendURL())
}

func matchedNoticeRouteObjectIDs(raw string, payload ruleNotificationPayload) []uint {
	var routes []faultCenterNoticeRouteConfig
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &routes); err != nil {
		return []uint{}
	}
	var ids []uint
	for _, route := range routes {
		if route.Enabled != nil && !*route.Enabled {
			continue
		}
		if !noticeRouteMatcherMatched(route.Matcher, payload) {
			continue
		}
		ids = append(ids, uintIDsFromValues(route.NoticeObjectIDs)...)
	}
	return uniqueUintIDs(ids)
}

func noticeRouteMatcherMatched(matcher string, payload ruleNotificationPayload) bool {
	matcher = strings.TrimSpace(matcher)
	if matcher == "" {
		return false
	}
	labels := parseStringMap(payload.Labels)
	for _, part := range strings.FieldsFunc(matcher, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	}) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		operator := ""
		key := ""
		expected := ""
		for _, item := range []string{"!~", "=~", "!=", "==", "=", ":"} {
			if left, right, ok := strings.Cut(part, item); ok {
				key = left
				expected = right
				operator = item
				break
			}
		}
		if operator == "" {
			return false
		}
		key = strings.TrimSpace(key)
		expected = strings.Trim(strings.TrimSpace(expected), `"'`)
		actual := noticeMatcherFieldValue(key, payload, labels)
		if !noticeMatcherValueMatched(key, operator, actual, expected) {
			return false
		}
	}
	return true
}

func noticeMatcherFieldValue(key string, payload ruleNotificationPayload, labels map[string]string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "severity", "level":
		return normalizeSeverityLevel(payload.Severity)
	case "state", "status":
		return payload.State
	case "fingerprint":
		return payload.Fingerprint
	case "ruleid", "rule_id":
		return strconv.FormatUint(uint64(payload.RuleID), 10)
	case "rule", "rulename", "rule_name":
		return payload.RuleName
	case "datasourceid", "data_source_id":
		return strconv.FormatUint(uint64(payload.DataSourceID), 10)
	case "datasource", "datasourcename", "data_source_name":
		return payload.DataSourceName
	case "datasourcetype", "data_source_type":
		return payload.DataSourceType
	default:
		return labels[key]
	}
}

func noticeMatcherValueMatched(key, operator, actual, expected string) bool {
	if strings.EqualFold(strings.TrimSpace(key), "severity") || strings.EqualFold(strings.TrimSpace(key), "level") {
		actual = normalizeSeverityLevel(actual)
		expected = normalizeSeverityLevel(expected)
	}
	switch operator {
	case "!=", "!~":
		if operator == "!~" {
			matched, err := regexp.MatchString(expected, actual)
			return err == nil && !matched
		}
		return !strings.EqualFold(strings.TrimSpace(actual), strings.TrimSpace(expected))
	case "=~":
		matched, err := regexp.MatchString(expected, actual)
		return err == nil && matched
	default:
		return strings.EqualFold(strings.TrimSpace(actual), strings.TrimSpace(expected))
	}
}

func parseStringMap(raw string) map[string]string {
	result := map[string]string{}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &data); err != nil {
		return result
	}
	for key, value := range data {
		result[key] = fmt.Sprint(value)
	}
	return result
}

func uintIDsFromValues(values []interface{}) []uint {
	ids := make([]uint, 0, len(values))
	for _, item := range values {
		switch value := item.(type) {
		case float64:
			if value > 0 {
				ids = append(ids, uint(value))
			}
		case string:
			id, _ := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
			if id > 0 {
				ids = append(ids, uint(id))
			}
		}
	}
	return ids
}

func uniqueUintIDs(values []uint) []uint {
	result := make([]uint, 0, len(values))
	seen := map[uint]struct{}{}
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (h *DataSourceHandler) sendRuleNotificationsToChannels(ctx context.Context, ids []uint, payload ruleNotificationPayload) (string, error) {
	var channels []model.AlertChannel
	if err := h.db.Where("enabled = ? AND id IN ?", true, ids).Find(&channels).Error; err != nil {
		return "failed", err
	}
	if len(channels) == 0 {
		return "failed", fmt.Errorf("未找到启用的告警通道")
	}

	var errors []string
	successCount := 0
	for _, channel := range channels {
		if err := sendRuleNotificationToChannel(ctx, channel, payload); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", channel.Name, err.Error()))
			continue
		}
		successCount++
	}

	if successCount > 0 {
		if len(errors) > 0 {
			return "partial", fmt.Errorf("%s", strings.Join(errors, "; "))
		}
		return "success", nil
	}
	if len(errors) == 0 {
		return "none", nil
	}
	return "failed", fmt.Errorf("%s", strings.Join(errors, "; "))
}

func (h *DataSourceHandler) sendRuleNotificationsToNoticeObjects(ctx context.Context, ids []uint, payload ruleNotificationPayload) (string, error) {
	var objects []model.NoticeObject
	if err := h.db.Where("enabled = ? AND id IN ?", true, ids).Find(&objects).Error; err != nil {
		return "failed", err
	}
	if len(objects) == 0 {
		return "failed", fmt.Errorf("未找到启用的通知对象")
	}

	var errors []string
	var skipReasons []string
	successCount := 0
	for i := range objects {
		object := &objects[i]
		h.fillNoticeObjectRuntime(object)
		routes := parseNoticeObjectRoutes(object.Routes)
		if len(routes) == 0 {
			errors = append(errors, fmt.Sprintf("%s: 未配置通知路由", object.Name))
			continue
		}
		for _, route := range routes {
			if reason := noticeRouteSkipReason(route, payload.Severity); reason != "" {
				skipReasons = append(skipReasons, fmt.Sprintf("%s/%s: %s", object.Name, normalizeNoticeType(route.NoticeType), reason))
				continue
			}
			if err := h.sendRuleNotificationToNoticeRoute(ctx, *object, route, payload); err != nil {
				errors = append(errors, fmt.Sprintf("%s/%s: %s", object.Name, normalizeNoticeType(route.NoticeType), err.Error()))
				continue
			}
			successCount++
		}
	}

	if successCount > 0 {
		if len(errors) > 0 {
			return "partial", fmt.Errorf("%s", strings.Join(errors, "; "))
		}
		return "success", nil
	}
	if len(errors) == 0 && len(skipReasons) > 0 {
		return "failed", fmt.Errorf("%s", strings.Join(skipReasons, "; "))
	}
	if len(errors) == 0 {
		return "none", nil
	}
	return "failed", fmt.Errorf("%s", strings.Join(errors, "; "))
}

func parseNoticeObjectRoutes(raw string) []noticeObjectRouteConfig {
	var routes []noticeObjectRouteConfig
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &routes); err != nil {
		return []noticeObjectRouteConfig{}
	}
	return routes
}

func noticeRouteEnabled(route noticeObjectRouteConfig) bool {
	return route.Enabled == nil || *route.Enabled
}

func noticeRouteSkipReason(route noticeObjectRouteConfig, severity string) string {
	if !noticeRouteEnabled(route) {
		return "通知路由已停用"
	}
	if !noticeRouteSeverityMatched(route, severity) {
		return fmt.Sprintf("告警等级 %s 不在适用级别 %s", normalizeSeverityLevel(severity), formatNoticeSeveritys(route.Severitys))
	}
	if !noticeRouteTimeMatched(route) {
		return fmt.Sprintf("当前时间不在生效时间 %s", formatNoticeEffectiveTime(route.EffectiveTime))
	}
	return ""
}

func noticeRouteSeverityMatched(route noticeObjectRouteConfig, severity string) bool {
	if len(route.Severitys) == 0 {
		return true
	}
	target := normalizeSeverityLevel(severity)
	for _, item := range route.Severitys {
		if normalizeSeverityLevel(item) == target {
			return true
		}
	}
	return false
}

func formatNoticeSeveritys(severitys []string) string {
	if len(severitys) == 0 {
		return "全部"
	}
	items := make([]string, 0, len(severitys))
	seen := map[string]struct{}{}
	for _, item := range severitys {
		normalized := normalizeSeverityLevel(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}
	if len(items) == 0 {
		return "全部"
	}
	return strings.Join(items, "/")
}

func normalizeSeverityLevel(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "p0", "critical":
		return "P0"
	case "p1", "warning":
		return "P1"
	case "p2", "info":
		return "P2"
	default:
		return strings.ToUpper(strings.TrimSpace(severity))
	}
}

func noticeRouteTimeMatched(route noticeObjectRouteConfig) bool {
	now := time.Now()
	if len(route.EffectiveTime.Week) > 0 {
		today := now.Weekday().String()
		matched := false
		for _, item := range route.EffectiveTime.Week {
			if strings.EqualFold(strings.TrimSpace(item), today) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	start := effectiveClockText(route.EffectiveTime.StartTime)
	end := effectiveClockText(route.EffectiveTime.EndTime)
	if start == "" || end == "" {
		return true
	}
	current := now.Format("15:04")
	if start <= end {
		return current >= start && current <= end
	}
	return current >= start || current <= end
}

func formatNoticeEffectiveTime(effective noticeEffectiveTime) string {
	parts := []string{}
	if len(effective.Week) > 0 {
		parts = append(parts, "星期 "+strings.Join(effective.Week, "/"))
	}
	start := effectiveClockText(effective.StartTime)
	end := effectiveClockText(effective.EndTime)
	if start != "" && end != "" {
		parts = append(parts, start+"-"+end)
	}
	if len(parts) == 0 {
		return "全天"
	}
	return strings.Join(parts, " ")
}

func effectiveClockText(raw interface{}) string {
	switch value := raw.(type) {
	case string:
		text := strings.TrimSpace(value)
		if text == "0" {
			return ""
		}
		return text
	case float64:
		if value == 0 {
			return ""
		}
		return fmt.Sprintf("%02.0f:00", value)
	case int:
		if value == 0 {
			return ""
		}
		return fmt.Sprintf("%02d:00", value)
	default:
		return ""
	}
}

func (h *DataSourceHandler) sendRuleNotificationToNoticeRoute(ctx context.Context, object model.NoticeObject, route noticeObjectRouteConfig, payload ruleNotificationPayload) error {
	noticeType := normalizeNoticeType(route.NoticeType)
	callbacks := h.buildNotificationCallbackContextForPayload(ctx, payload)
	text := h.renderNoticeRouteText(object, route, payload, callbacks)
	receivers := buildNoticeReceivers(object, route)

	switch noticeType {
	case "WebHook":
		if strings.TrimSpace(route.Hook) == "" {
			return fmt.Errorf("Hook 地址未配置")
		}
		var customPayload interface{}
		if json.Unmarshal([]byte(strings.TrimSpace(text)), &customPayload) == nil {
			return postJSONWithHeaders(ctx, route.Hook, customPayload, route.Headers)
		}
		return postJSONWithHeaders(ctx, route.Hook, gin.H{"event": payload, "text": text, "receivers": receivers}, route.Headers)
	case "WeChat":
		if strings.TrimSpace(route.Hook) == "" {
			return fmt.Errorf("企业微信 Hook 地址未配置")
		}
		return postJSONWithHeaders(ctx, route.Hook, gin.H{"msgtype": "markdown", "markdown": gin.H{"content": text}}, route.Headers)
	case "DingDing":
		if strings.TrimSpace(route.Hook) == "" {
			return fmt.Errorf("钉钉 Hook 地址未配置")
		}
		targetURL, err := buildDingTalkWebhookURL(route.Hook, route.Sign, time.Now())
		if err != nil {
			return fmt.Errorf("钉钉 Hook 地址无效: %w", err)
		}
		atMobiles, atUserIDs := noticeDingTalkAtLists(object)
		return postJSONWithHeaders(ctx, targetURL, gin.H{
			"msgtype": "markdown",
			"markdown": gin.H{
				"title": "监控告警",
				"text":  strings.ReplaceAll(text, "\n", "\n\n"),
			},
			"at": gin.H{
				"atMobiles": atMobiles,
				"atUserIds": atUserIDs,
				"isAtAll":   false,
			},
		}, route.Headers)
	case "FeiShu":
		if strings.TrimSpace(route.Hook) == "" {
			return fmt.Errorf("飞书 Hook 地址未配置")
		}
		h.attachFeishuCallbackImages(ctx, route, &callbacks)
		var jsonPayload interface{}
		if json.Unmarshal([]byte(strings.TrimSpace(text)), &jsonPayload) == nil {
			return postJSONWithHeaders(ctx, route.Hook, buildFeishuWebhookPayload(injectFeishuCallbackImages(jsonPayload, callbacks)), route.Headers)
		}
		var cardPayload interface{}
		if looksLikeJSONTemplate(text) {
			cardPayload = buildDefaultFeishuCard(object, payload, callbacks)
		} else {
			cardPayload = buildFeishuMarkdownCard(object, payload, text, callbacks)
		}
		return postJSONWithHeaders(ctx, route.Hook, gin.H{"msg_type": "interactive", "card": cardPayload}, route.Headers)
	case "Email":
		return h.sendNoticeRouteEmail(ctx, object, route, payload, text)
	default:
		return fmt.Errorf("不支持的通知类型: %s", noticeType)
	}
}

func buildFeishuWebhookPayload(payload interface{}) gin.H {
	if message, ok := payload.(map[string]interface{}); ok {
		msgType, hasMsgType := message["msg_type"].(string)
		if hasMsgType && strings.TrimSpace(msgType) != "" {
			return gin.H(message)
		}
	}
	return gin.H{"msg_type": "interactive", "card": payload}
}

func (h *DataSourceHandler) sendNoticeRouteEmail(ctx context.Context, object model.NoticeObject, route noticeObjectRouteConfig, payload ruleNotificationPayload, text string) error {
	config, err := h.loadNoticeEmailSMTPConfig(route)
	if err != nil {
		return err
	}
	to := emailNoticeRecipients(object, route)
	cc := normalizeEmailRecipients(route.CC)
	recipients := uniqueStrings(append(append([]string{}, to...), cc...))
	if len(recipients) == 0 {
		return fmt.Errorf("邮件通知没有可用收件人：请填写收件人，或确认值班表当天值班人员已配置邮箱")
	}
	subject := renderNoticeEmailSubject(route.Subject, payload)
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("邮件主题未配置")
	}
	body := buildNoticeEmailBody(text)
	return sendSMTPEmail(ctx, config, emailMessage{
		Subject: subject,
		To:      to,
		CC:      cc,
		Body:    body,
	})
}

type emailMessage struct {
	Subject string
	To      []string
	CC      []string
	Body    string
}

func (h *DataSourceHandler) loadNoticeEmailSMTPConfig(route noticeObjectRouteConfig) (alertChannelConfig, error) {
	if config, ok, err := buildNoticeRouteSMTPConfig(route); ok || err != nil {
		return config, err
	}
	var channels []model.AlertChannel
	if err := h.db.Where("enabled = ? AND channel_type = ?", true, "email").Order("id ASC").Find(&channels).Error; err != nil {
		return alertChannelConfig{}, err
	}
	for _, channel := range channels {
		var config alertChannelConfig
		if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
			continue
		}
		config.SMTPHost = strings.TrimSpace(config.SMTPHost)
		config.SMTPUser = strings.TrimSpace(config.SMTPUser)
		config.FromEmail = strings.TrimSpace(firstNonEmpty(config.FromEmail, config.SMTPUser))
		config.FromName = strings.TrimSpace(firstNonEmpty(config.FromName, "OpsHub"))
		if config.SMTPPort <= 0 {
			config.SMTPPort = 25
		}
		if config.SMTPHost != "" && config.FromEmail != "" {
			return config, nil
		}
	}
	return alertChannelConfig{}, fmt.Errorf("邮件通知缺少 SMTP 配置：请在通知对象的邮件策略中填写 SMTP 服务器和发件邮箱")
}

func buildNoticeRouteSMTPConfig(route noticeObjectRouteConfig) (alertChannelConfig, bool, error) {
	hasRouteConfig := strings.TrimSpace(route.SMTPHost) != "" ||
		strings.TrimSpace(route.SMTPUser) != "" ||
		strings.TrimSpace(route.FromEmail) != ""
	if !hasRouteConfig {
		return alertChannelConfig{}, false, nil
	}
	config := alertChannelConfig{
		SMTPHost:     strings.TrimSpace(route.SMTPHost),
		SMTPPort:     route.SMTPPort,
		SMTPUser:     strings.TrimSpace(route.SMTPUser),
		SMTPPassword: route.SMTPPassword,
		FromEmail:    strings.TrimSpace(firstNonEmpty(route.FromEmail, route.SMTPUser)),
		FromName:     strings.TrimSpace(firstNonEmpty(route.FromName, "OpsHub")),
	}
	if config.SMTPPort <= 0 {
		config.SMTPPort = 465
	}
	if config.SMTPHost == "" {
		return alertChannelConfig{}, true, fmt.Errorf("邮件通知缺少 SMTP 服务器")
	}
	if config.FromEmail == "" {
		return alertChannelConfig{}, true, fmt.Errorf("邮件通知缺少发件邮箱")
	}
	if config.SMTPUser != "" && strings.TrimSpace(config.SMTPPassword) == "" {
		return alertChannelConfig{}, true, fmt.Errorf("邮件通知缺少 SMTP 密码")
	}
	return config, true, nil
}

func emailNoticeRecipients(object model.NoticeObject, route noticeObjectRouteConfig) []string {
	recipients := append([]string{}, route.To...)
	for _, user := range object.CurrentDutyUsers {
		if email := strings.TrimSpace(user.Email); email != "" {
			recipients = append(recipients, email)
		}
	}
	return normalizeEmailRecipients(recipients)
}

func normalizeEmailRecipients(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || !strings.Contains(value, "@") {
			continue
		}
		result = append(result, value)
	}
	return uniqueStrings(result)
}

func renderNoticeEmailSubject(template string, payload ruleNotificationPayload) string {
	subject := strings.TrimSpace(template)
	if subject == "" {
		return ""
	}
	replacements := map[string]string{
		"{{ruleName}}":       payload.RuleName,
		"{{rule_name}}":      payload.RuleName,
		"${rule_name}":       payload.RuleName,
		"{{severity}}":       normalizeSeverityLevel(payload.Severity),
		"${severity}":        normalizeSeverityLevel(payload.Severity),
		"{{state}}":          getStateText(payload.State),
		"${state}":           getStateText(payload.State),
		"{{value}}":          formatRuleValue(payload.Value),
		"${value}":           formatRuleValue(payload.Value),
		"{{dataSourceName}}": payload.DataSourceName,
		"${data_source}":     payload.DataSourceName,
		"{{time}}":           payload.Time.Format("2006-01-02 15:04:05"),
		"${time}":            payload.Time.Format("2006-01-02 15:04:05"),
	}
	for key, value := range replacements {
		subject = strings.ReplaceAll(subject, key, value)
	}
	return subject
}

func buildNoticeEmailBody(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		text = "OpsHub 监控通知"
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<body") {
		return text
	}
	if looksLikeHTMLFragment(text) {
		return wrapNoticeEmailHTML(text)
	}
	if looksLikeJSONTemplate(text) {
		var pretty bytes.Buffer
		if json.Indent(&pretty, []byte(text), "", "  ") == nil {
			text = pretty.String()
		}
		return wrapNoticeEmailHTML(`<pre style="margin:0;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;white-space:pre-wrap;line-height:1.6;color:#111827;">` + html.EscapeString(text) + `</pre>`)
	}
	return wrapNoticeEmailHTML(`<div style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Arial,sans-serif;line-height:1.7;color:#1f2937;white-space:normal;">` +
		strings.ReplaceAll(html.EscapeString(text), "\n", "<br>") +
		`</div>`)
}

func looksLikeHTMLFragment(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	for _, marker := range []string{
		"<a ", "<article", "<b>", "<blockquote", "<br", "<button",
		"<div", "<font", "<h1", "<h2", "<h3", "<h4", "<hr",
		"<img", "<li", "<ol", "<p", "<pre", "<section", "<span",
		"<strong", "<table", "<tbody", "<td", "<th", "<thead", "<tr", "<ul",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func wrapNoticeEmailHTML(fragment string) string {
	return `<!doctype html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin:0;padding:0;background:#f3f4f6;">
  <div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">OpsHub 监控通知</div>
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;background:#f3f4f6;">
    <tr>
      <td align="center" style="padding:24px 12px;">
        <table role="presentation" width="760" cellspacing="0" cellpadding="0" border="0" style="width:100%;max-width:760px;background:#ffffff;border:1px solid #e5e7eb;border-radius:14px;box-shadow:0 12px 32px rgba(15,23,42,0.08);">
          <tr>
            <td style="padding:24px;">
` + fragment + `
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
}

func sendSMTPEmail(ctx context.Context, config alertChannelConfig, message emailMessage) error {
	host := strings.TrimSpace(config.SMTPHost)
	fromEmail := strings.TrimSpace(config.FromEmail)
	if host == "" || fromEmail == "" {
		return fmt.Errorf("SMTP 配置不完整")
	}
	port := config.SMTPPort
	if port <= 0 {
		port = 25
	}
	recipients := uniqueStrings(append(append([]string{}, message.To...), message.CC...))
	if len(recipients) == 0 {
		return fmt.Errorf("邮件收件人为空")
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	headers := map[string]string{
		"From":                      formatEmailAddress(config.FromName, fromEmail),
		"To":                        strings.Join(message.To, ", "),
		"Subject":                   mime.QEncoding.Encode("UTF-8", message.Subject),
		"Date":                      time.Now().Format(time.RFC1123Z),
		"MIME-Version":              "1.0",
		"Content-Type":              "text/html; charset=UTF-8",
		"Content-Transfer-Encoding": "8bit",
	}
	if len(message.CC) > 0 {
		headers["Cc"] = strings.Join(message.CC, ", ")
	}
	var builder strings.Builder
	for _, key := range []string{"From", "To", "Cc", "Subject", "Date", "MIME-Version", "Content-Type", "Content-Transfer-Encoding"} {
		if value := headers[key]; strings.TrimSpace(value) != "" {
			builder.WriteString(key)
			builder.WriteString(": ")
			builder.WriteString(value)
			builder.WriteString("\r\n")
		}
	}
	builder.WriteString("\r\n")
	builder.WriteString(message.Body)

	var auth smtp.Auth
	if strings.TrimSpace(config.SMTPUser) != "" {
		auth = smtp.PlainAuth("", strings.TrimSpace(config.SMTPUser), config.SMTPPassword, host)
	}
	if port == 465 {
		return sendSMTPEmailImplicitTLS(ctx, addr, host, auth, fromEmail, recipients, []byte(builder.String()))
	}
	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, fromEmail, recipients, []byte(builder.String()))
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func sendSMTPEmailImplicitTLS(ctx context.Context, addr, host string, auth smtp.Auth, from string, recipients []string, msg []byte) error {
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- client.Quit() }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func formatEmailAddress(name, address string) string {
	address = strings.TrimSpace(address)
	name = strings.TrimSpace(name)
	if name == "" {
		return address
	}
	return mime.QEncoding.Encode("UTF-8", name) + " <" + address + ">"
}

func buildDefaultFeishuCard(object model.NoticeObject, payload ruleNotificationPayload, callbacks notificationCallbackContext) gin.H {
	if payload.Aggregated {
		return buildDefaultAggregatedFeishuCard(object, payload, callbacks)
	}
	stateText := getStateText(payload.State)
	template := "red"
	title := "【告警中】- OpsHub 监控告警"
	if payload.Escalated {
		template = "red"
		title = "【告警升级】- OpsHub 监控告警"
	} else if payload.State == "recovered" || payload.EndedAt != nil {
		template = "green"
		title = "【已恢复】- OpsHub 监控告警"
	} else if payload.State == "pending" {
		template = "orange"
		title = "【预告警】- OpsHub 监控告警"
	} else if payload.State == "error" {
		template = "purple"
		title = "【评估异常】- OpsHub 监控告警"
	}
	annotationText := noticeAnnotationText(payload.Annotations, parseStringMap(payload.Annotations))
	matchedLogs := noticeMatchedLogsText(parseStringMap(payload.Annotations))
	lines := []string{
		fmt.Sprintf("**🤖 告警规则:** %s", payload.RuleName),
		fmt.Sprintf("**🫧 告警指纹:** %s", payload.Fingerprint),
		fmt.Sprintf("**📌 告警等级:** %s", normalizeSeverityLevel(payload.Severity)),
		fmt.Sprintf("**📍 事件状态:** %s", stateText),
		fmt.Sprintf("**🖥 告警实例:** %s", firstNonEmpty(parseStringMap(payload.Labels)["instance"], payload.DataSourceName, "-")),
		fmt.Sprintf("**📈 当前数值:** %s %s %s", formatRuleValue(payload.Value), getConditionText(payload.Condition), formatRuleValue(payload.Threshold)),
		fmt.Sprintf("**🕘 开始时间:** %s", payload.StartedAt.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("**👤 值班人员:** %s", noticeDutyUserFeishuAtText(object)),
		fmt.Sprintf("**📝 告警事件:** %s", firstNonEmpty(annotationText, payload.Message)),
		fmt.Sprintf("[%s](%s)", noticeEventLinkText(payload), buildNoticeEventURL(payload)),
	}
	if payload.Escalated && payload.EscalationText != "" {
		lines = append(lines[:4], append([]string{fmt.Sprintf("**🚨 升级说明:** %s", payload.EscalationText)}, lines[4:]...)...)
	}
	if payload.EndedAt != nil {
		lines = append(lines[:7], append([]string{fmt.Sprintf("**🕘 恢复时间:** %s", payload.EndedAt.Format("2006-01-02 15:04:05"))}, lines[7:]...)...)
	}
	if matchedLogs != "" && !annotationContainsMatchedLogs(annotationText, matchedLogs) {
		lines = append(lines[:len(lines)-1], append([]string{fmt.Sprintf("**📄 命中日志:**\n```text\n%s\n```", sanitizeMarkdownCodeBlock(matchedLogs))}, lines[len(lines)-1:]...)...)
	}
	elements := []gin.H{
		{"tag": "markdown", "content": strings.Join(lines, "\n")},
	}
	elements = append(elements, buildFeishuCallbackImageElements(callbacks)...)
	elements = append(elements,
		gin.H{"tag": "hr"},
		gin.H{"tag": "markdown", "content": "🧑‍💻 OpsHub - 运维团队"},
	)
	return gin.H{
		"schema": "2.0",
		"config": gin.H{
			"width_mode":     "fill",
			"enable_forward": true,
		},
		"header": gin.H{
			"template": template,
			"title":    gin.H{"tag": "plain_text", "content": title},
		},
		"body": gin.H{
			"elements": elements,
		},
	}
}

func buildDefaultAggregatedFeishuCard(object model.NoticeObject, payload ruleNotificationPayload, callbacks notificationCallbackContext) gin.H {
	template := "red"
	title := "【告警中】- OpsHub 监控告警"
	if payload.Escalated {
		template = "red"
		title = "【告警升级】- OpsHub 聚合告警"
	} else if payload.State == "recovered" || payload.EndedAt != nil {
		template = "green"
		title = "【已恢复】- OpsHub 监控告警"
	} else if payload.State == "pending" {
		template = "orange"
		title = "【预告警】- OpsHub 监控告警"
	}
	annotationText := noticeAnnotationText(payload.Annotations, parseStringMap(payload.Annotations))
	matchedLogs := noticeMatchedLogsText(parseStringMap(payload.Annotations))
	lines := []string{
		fmt.Sprintf("**🤖 告警类型:** %s", payload.RuleName),
		fmt.Sprintf("**🫧 告警指纹:** %s", payload.Fingerprint),
		fmt.Sprintf("**📌 告警等级:** %s", normalizeSeverityLevel(payload.Severity)),
		fmt.Sprintf("**🖥 告警主机:** %s", firstNonEmpty(parseStringMap(payload.Labels)["instance"], payload.DataSourceName, "-")),
		fmt.Sprintf("**🕘 开始时间:** %s", payload.StartedAt.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("**👤 值班人员:** %s", noticeDutyUserFeishuAtText(object)),
		fmt.Sprintf("**📝 告警事件:** %s", firstNonEmpty(annotationText, payload.Message)),
		fmt.Sprintf("[%s](%s)", noticeEventLinkText(payload), buildNoticeEventURL(payload)),
	}
	if payload.Escalated && payload.EscalationText != "" {
		lines = append(lines[:3], append([]string{fmt.Sprintf("**🚨 升级说明:** %s", payload.EscalationText)}, lines[3:]...)...)
	}
	if payload.EndedAt != nil {
		lines = append(lines[:5], append([]string{fmt.Sprintf("**🕘 恢复时间:** %s", payload.EndedAt.Format("2006-01-02 15:04:05"))}, lines[5:]...)...)
	}
	if matchedLogs != "" && !annotationContainsMatchedLogs(annotationText, matchedLogs) {
		lines = append(lines[:len(lines)-1], append([]string{fmt.Sprintf("**📄 命中日志:**\n```text\n%s\n```", sanitizeMarkdownCodeBlock(matchedLogs))}, lines[len(lines)-1:]...)...)
	}
	elements := []gin.H{
		{"tag": "markdown", "content": strings.Join(lines, "\n")},
	}
	elements = append(elements, buildFeishuCallbackImageElements(callbacks)...)
	elements = append(elements,
		gin.H{"tag": "hr"},
		gin.H{"tag": "markdown", "content": "🧑‍💻 OpsHub - 运维团队"},
	)
	return gin.H{
		"schema": "2.0",
		"config": gin.H{
			"width_mode":     "fill",
			"enable_forward": true,
		},
		"header": gin.H{
			"template": template,
			"title":    gin.H{"tag": "plain_text", "content": title},
		},
		"body": gin.H{
			"elements": elements,
		},
	}
}

func buildFeishuMarkdownCard(object model.NoticeObject, payload ruleNotificationPayload, text string, callbacks notificationCallbackContext) gin.H {
	template := "red"
	title := "OpsHub 监控告警"
	if payload.Escalated {
		template = "red"
		title = "OpsHub 告警升级"
	} else if payload.State == "recovered" || payload.EndedAt != nil {
		template = "green"
		title = "OpsHub 告警恢复"
	} else if payload.State == "pending" {
		template = "orange"
		title = "OpsHub 预告警"
	} else if payload.Aggregated {
		title = "OpsHub 告警聚合"
	}
	content := strings.TrimSpace(text)
	if content == "" {
		content = firstNonEmpty(payload.Message, payload.RuleName)
	}
	if mention := noticeDutyUserFeishuAtText(object); mention != "" && mention != "-" && !strings.Contains(content, mention) {
		content += "\n\n**📣 通知对象:** " + mention
	}
	elements := []gin.H{{"tag": "markdown", "content": content}}
	elements = append(elements, buildFeishuCallbackImageElements(callbacks)...)
	elements = append(elements,
		gin.H{"tag": "hr"},
		gin.H{"tag": "markdown", "content": "🧑‍💻 OpsHub - 运维团队"},
	)
	return gin.H{
		"schema": "2.0",
		"config": gin.H{
			"width_mode":     "fill",
			"enable_forward": true,
		},
		"header": gin.H{
			"template": template,
			"title":    gin.H{"tag": "plain_text", "content": title},
		},
		"body": gin.H{
			"elements": elements,
		},
	}
}

func buildFeishuCallbackImageElements(callbacks notificationCallbackContext) []gin.H {
	if len(callbacks.Images) == 0 && len(callbacks.ImageWarnings) == 0 {
		return nil
	}
	elements := make([]gin.H, 0, len(callbacks.Images)*2+len(callbacks.ImageWarnings)+1)
	added := 0
	for _, imageItem := range callbacks.Images {
		if strings.TrimSpace(imageItem.ImageKey) == "" {
			continue
		}
		if added == 0 {
			elements = append(elements, gin.H{"tag": "hr"})
			elements = append(elements, gin.H{"tag": "markdown", "content": "**📊 回调查询图表**"})
		}
		title := firstNonEmpty(imageItem.Title, "回调查询")
		query := strings.TrimSpace(imageItem.Query)
		content := fmt.Sprintf("**%s**", title)
		if query != "" {
			content += fmt.Sprintf("\n`%s`", clipPlainText(query, 480))
		}
		elements = append(elements,
			gin.H{"tag": "markdown", "content": content},
			gin.H{
				"tag":     "img",
				"img_key": imageItem.ImageKey,
				"alt":     gin.H{"tag": "plain_text", "content": title},
				"mode":    "fit_horizontal",
			},
		)
		added++
		if added >= 3 {
			break
		}
	}
	if added == 0 && len(callbacks.ImageWarnings) > 0 {
		elements = append(elements, gin.H{"tag": "hr"})
		warningText := strings.Join(callbacks.ImageWarnings, "\n")
		elements = append(elements, gin.H{
			"tag":     "markdown",
			"content": "**📊 回调查询图表**\n图表上传失败：" + clipPlainText(warningText, 900),
		})
	}
	return elements
}

func (h *DataSourceHandler) attachFeishuCallbackImages(ctx context.Context, route noticeObjectRouteConfig, callbacks *notificationCallbackContext) {
	if callbacks == nil || len(callbacks.Images) == 0 {
		return
	}
	appID := strings.TrimSpace(route.FeishuAppID)
	appSecret := strings.TrimSpace(route.FeishuAppSecret)
	if appID == "" || appSecret == "" {
		callbacks.ImageWarnings = append(callbacks.ImageWarnings, "已生成回调图表，但通知对象未配置飞书 App ID / App Secret")
		return
	}
	token, err := fetchFeishuTenantAccessToken(ctx, appID, appSecret)
	if err != nil || token == "" {
		callbacks.ImageWarnings = append(callbacks.ImageWarnings, firstNonEmpty(errString(err), "获取飞书 tenant_access_token 失败"))
		return
	}
	uploaded := 0
	for i := range callbacks.Images {
		if uploaded >= 3 {
			break
		}
		if len(callbacks.Images[i].PNG) == 0 {
			continue
		}
		imageKey, err := uploadFeishuMessageImage(ctx, token, callbacks.Images[i].PNG)
		if err != nil || strings.TrimSpace(imageKey) == "" {
			callbacks.ImageWarnings = append(callbacks.ImageWarnings, fmt.Sprintf("%s 上传失败：%s", firstNonEmpty(callbacks.Images[i].Title, "回调图表"), firstNonEmpty(feishuUploadUserMessage(err), "飞书未返回 image_key")))
			continue
		}
		callbacks.Images[i].ImageKey = imageKey
		uploaded++
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func fetchFeishuTenantAccessToken(ctx context.Context, appID, appSecret string) (string, error) {
	payload := gin.H{
		"app_id":     appID,
		"app_secret": appSecret,
	}
	body, err := marshalJSONBody(payload)
	if err != nil {
		return "", err
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("飞书获取 tenant_access_token 失败: HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", fmt.Errorf("飞书获取 tenant_access_token 失败: %s", firstNonEmpty(result.Msg, strconv.Itoa(result.Code)))
	}
	return strings.TrimSpace(result.TenantAccessToken), nil
}

func uploadFeishuMessageImage(ctx context.Context, tenantAccessToken string, imageBytes []byte) (string, error) {
	endpoints := []string{
		"https://open.feishu.cn/open-apis/im/v1/images",
		"https://open.feishu.cn/open-apis/image/v4/put/",
	}
	errMessages := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		imageKey, err := uploadFeishuMessageImageToEndpoint(ctx, tenantAccessToken, imageBytes, endpoint)
		if err == nil && strings.TrimSpace(imageKey) != "" {
			return imageKey, nil
		}
		errMessages = append(errMessages, feishuUploadUserMessage(err))
	}
	return "", errors.New(strings.Join(uniqueStrings(errMessages), "；"))
}

func uploadFeishuMessageImageToEndpoint(ctx context.Context, tenantAccessToken string, imageBytes []byte, endpoint string) (string, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	if err := writer.WriteField("image_type", "message"); err != nil {
		return "", err
	}
	part, err := writer.CreateFormFile("image", "opshub-callback.png")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(imageBytes); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, &buffer)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", feishuAPIError("飞书上传图片失败", resp.StatusCode, respBody)
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ImageKey string `json:"image_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", feishuAPIErrorFromFields("飞书上传图片失败", resp.StatusCode, result.Code, result.Msg, respBody)
	}
	return strings.TrimSpace(result.Data.ImageKey), nil
}

func feishuAPIError(operation string, statusCode int, body []byte) error {
	var result struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err == nil {
		return feishuAPIErrorFromFields(operation, statusCode, result.Code, firstNonEmpty(result.Msg, result.Error), body)
	}
	return fmt.Errorf("%s：HTTP %d，%s", operation, statusCode, clipPlainText(string(body), 260))
}

func feishuAPIErrorFromFields(operation string, statusCode int, code int, message string, body []byte) error {
	message = strings.TrimSpace(firstNonEmpty(message, string(body)))
	lowerMessage := strings.ToLower(message)
	if code == 99991672 || strings.Contains(lowerMessage, "access denied") || strings.Contains(lowerMessage, "im:resource") {
		return fmt.Errorf("%s：应用身份缺少图片资源上传权限（im:resource:upload 或 im:resource）。请在飞书开放平台给当前 App 开通「应用身份权限」后发布版本，并确认通知对象里填写的是同一个 App ID / App Secret", operation)
	}
	if code != 0 {
		return fmt.Errorf("%s：飞书返回 %d，%s", operation, code, clipPlainText(message, 260))
	}
	return fmt.Errorf("%s：HTTP %d，%s", operation, statusCode, clipPlainText(message, 260))
}

func feishuUploadUserMessage(err error) string {
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(err.Error())
	if strings.Contains(text, "应用身份缺少图片资源上传权限") {
		return "应用身份缺少图片资源上传权限（im:resource:upload 或 im:resource）。请确认权限加在「应用身份权限」而不是「用户身份权限」，并在飞书开放平台发布版本后重试"
	}
	return clipPlainText(text, 360)
}

func injectFeishuCallbackImages(payload interface{}, callbacks notificationCallbackContext) interface{} {
	elements := buildFeishuCallbackImageElements(callbacks)
	if len(elements) == 0 {
		return payload
	}
	if root, ok := payload.(map[string]interface{}); ok {
		if card, hasCard := root["card"]; hasCard {
			root["card"] = injectFeishuCallbackImages(card, callbacks)
			return root
		}
		injectFeishuElementsIntoCard(root, elements)
		return root
	}
	if root, ok := payload.(gin.H); ok {
		if card, hasCard := root["card"]; hasCard {
			root["card"] = injectFeishuCallbackImages(card, callbacks)
			return root
		}
		injectFeishuElementsIntoCard(root, elements)
		return root
	}
	return payload
}

func injectFeishuElementsIntoCard(card map[string]interface{}, elements []gin.H) {
	body, _ := card["body"].(map[string]interface{})
	if body == nil {
		body = map[string]interface{}{}
		card["body"] = body
	}
	existing, _ := body["elements"].([]interface{})
	for _, element := range elements {
		existing = append(existing, element)
	}
	body["elements"] = existing
}

func clipPlainText(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func (h *DataSourceHandler) renderNoticeRouteText(object model.NoticeObject, route noticeObjectRouteConfig, payload ruleNotificationPayload, callbacks notificationCallbackContext) string {
	text := ""
	templateID := noticeTemplateID(route.NoticeTemplateID)
	if templateID > 0 {
		var tmpl model.NoticeTemplate
		if err := h.db.Where("enabled = ?", true).First(&tmpl, templateID).Error; err == nil {
			if payload.State == "recovered" {
				text = firstNonEmpty(tmpl.TemplateRecover, tmpl.Template, tmpl.TemplateFiring)
			} else {
				text = firstNonEmpty(tmpl.TemplateFiring, tmpl.Template)
			}
		}
	}
	if strings.TrimSpace(text) == "" {
		text = defaultRuleNotificationText(payload)
	}
	jsonTemplate := looksLikeJSONTemplate(text)
	labels := parseStringMap(payload.Labels)
	annotations := parseStringMap(payload.Annotations)
	annotationText := noticeAnnotationText(payload.Annotations, annotations)
	matchedLogs := noticeMatchedLogsText(annotations)
	matchedLogCount := noticeMatchedLogCountText(annotations)
	matchedLogQuery := strings.TrimSpace(annotations["matched_log_query"])
	matchedLogsBlock := noticeMatchedLogsBlock(matchedLogs)
	if noticeTemplateUsesAnnotations(text) && annotationContainsMatchedLogs(annotationText, matchedLogs) {
		matchedLogsBlock = ""
	}
	dutyUser := noticeDutyUserText(object)
	noticeType := normalizeNoticeType(route.NoticeType)
	dutyMentions := noticeDutyUserMentionText(object, noticeType)
	routeDutyUser := noticeRouteDutyUserText(object, noticeType)
	eventURL := buildNoticeEventURL(payload)
	eventLinkText := noticeEventLinkText(payload)
	valueText := formatRuleValue(payload.Value)
	labels = cloneStringMap(labels)
	labels["value"] = valueText
	labelsText := formatTemplateStringMap(labels)
	labelsJSON := marshalStringMap(labels)
	instancesText := aggregatePayloadInstancesText(payload)
	fingerprintsText := aggregatePayloadFingerprintsText(payload)
	aggregateDetails := aggregatePayloadDetailsText(payload)
	escalatedText := "false"
	if payload.Escalated {
		escalatedText = "true"
	}
	replacements := map[string]string{
		"{{ruleName}}":                         noticeTemplateValue(payload.RuleName, jsonTemplate),
		"{{rule_name}}":                        noticeTemplateValue(payload.RuleName, jsonTemplate),
		"{{severity}}":                         noticeTemplateValue(normalizeSeverityLevel(payload.Severity), jsonTemplate),
		"{{state}}":                            noticeTemplateValue(getStateText(payload.State), jsonTemplate),
		"{{value}}":                            noticeTemplateValue(valueText, jsonTemplate),
		"{{ $value }}":                         noticeTemplateValue(valueText, jsonTemplate),
		"{{condition}}":                        noticeTemplateValue(getConditionText(payload.Condition), jsonTemplate),
		"{{threshold}}":                        noticeTemplateValue(formatRuleValue(payload.Threshold), jsonTemplate),
		"{{message}}":                          noticeTemplateValue(payload.Message, jsonTemplate),
		"{{dataSourceName}}":                   noticeTemplateValue(payload.DataSourceName, jsonTemplate),
		"{{dataSourceType}}":                   noticeTemplateValue(payload.DataSourceType, jsonTemplate),
		"{{time}}":                             noticeTemplateValue(payload.Time.Format("2006-01-02 15:04:05"), jsonTemplate),
		"{{labels}}":                           noticeTemplateValue(labelsJSON, jsonTemplate),
		"{{ $labels }}":                        noticeTemplateValue(labelsText, jsonTemplate),
		"{{annotations}}":                      noticeTemplateValue(annotationText, jsonTemplate),
		"{{matchedLogs}}":                      noticeTemplateValue(matchedLogs, jsonTemplate),
		"{{matched_logs}}":                     noticeTemplateValue(matchedLogs, jsonTemplate),
		"{{matchedLogsBlock}}":                 noticeTemplateValue(matchedLogsBlock, jsonTemplate),
		"{{matched_logs_block}}":               noticeTemplateValue(matchedLogsBlock, jsonTemplate),
		"{{matchedLogCount}}":                  noticeTemplateValue(matchedLogCount, jsonTemplate),
		"{{matched_log_count}}":                noticeTemplateValue(matchedLogCount, jsonTemplate),
		"{{matchedLogQuery}}":                  noticeTemplateValue(matchedLogQuery, jsonTemplate),
		"{{matched_log_query}}":                noticeTemplateValue(matchedLogQuery, jsonTemplate),
		"{{fingerprint}}":                      noticeTemplateValue(payload.Fingerprint, jsonTemplate),
		"{{faultCenterId}}":                    noticeTemplateValue(strconv.FormatUint(uint64(payload.FaultCenterID), 10), jsonTemplate),
		"{{eventId}}":                          noticeTemplateValue(strconv.FormatUint(uint64(payload.EventID), 10), jsonTemplate),
		"{{firstTriggerTime}}":                 noticeTemplateValue(payload.StartedAt.Format("2006-01-02 15:04:05"), jsonTemplate),
		"{{ .FirstTriggerTime | formatTime }}": noticeTemplateValue(payload.StartedAt.Format("2006-01-02 15:04:05"), jsonTemplate),
		"{{.FirstTriggerTime | formatTime}}":   noticeTemplateValue(payload.StartedAt.Format("2006-01-02 15:04:05"), jsonTemplate),
		"{{recoverTime}}":                      noticeTemplateValue(formatOptionalTime(payload.EndedAt), jsonTemplate),
		"{{ .RecoverTime | formatTime }}":      noticeTemplateValue(formatOptionalTime(payload.EndedAt), jsonTemplate),
		"{{.RecoverTime | formatTime}}":        noticeTemplateValue(formatOptionalTime(payload.EndedAt), jsonTemplate),
		"{{dutyUser}}":                         noticeTemplateValue(routeDutyUser, jsonTemplate),
		"{{dutyUserName}}":                     noticeTemplateValue(dutyUser, jsonTemplate),
		"{{dutyUserMentions}}":                 noticeTemplateValue(dutyMentions, jsonTemplate),
		"{{eventUrl}}":                         noticeTemplateValue(eventURL, jsonTemplate),
		"{{eventLinkText}}":                    noticeTemplateValue(eventLinkText, jsonTemplate),
		"{{callbackSummary}}":                  noticeTemplateValue(callbacks.Summary, jsonTemplate),
		"{{callbackLinks}}":                    noticeTemplateValue(callbacks.DetailText, jsonTemplate),
		"{{callbackDetails}}":                  noticeTemplateValue(callbacks.DetailText, jsonTemplate),
		"{{eventCount}}":                       noticeTemplateValue(strconv.Itoa(payload.EventCount), jsonTemplate),
		"{{instances}}":                        noticeTemplateValue(instancesText, jsonTemplate),
		"{{fingerprints}}":                     noticeTemplateValue(fingerprintsText, jsonTemplate),
		"{{aggregateDetails}}":                 noticeTemplateValue(aggregateDetails, jsonTemplate),
		"{{escalated}}":                        noticeTemplateValue(escalatedText, jsonTemplate),
		"{{escalationText}}":                   noticeTemplateValue(payload.EscalationText, jsonTemplate),
		"${rule_name}":                         noticeTemplateValue(payload.RuleName, jsonTemplate),
		"${fingerprint}":                       noticeTemplateValue(payload.Fingerprint, jsonTemplate),
		"${fingerprints}":                      noticeTemplateValue(fingerprintsText, jsonTemplate),
		"${severity}":                          noticeTemplateValue(normalizeSeverityLevel(payload.Severity), jsonTemplate),
		"${value}":                             noticeTemplateValue(valueText, jsonTemplate),
		"${labels}":                            noticeTemplateValue(labelsText, jsonTemplate),
		"${annotations}":                       noticeTemplateValue(annotationText, jsonTemplate),
		"${matched_logs}":                      noticeTemplateValue(matchedLogs, jsonTemplate),
		"${matched_logs_block}":                noticeTemplateValue(matchedLogsBlock, jsonTemplate),
		"${matched_log_count}":                 noticeTemplateValue(matchedLogCount, jsonTemplate),
		"${matched_log_query}":                 noticeTemplateValue(matchedLogQuery, jsonTemplate),
		"${duty_user}":                         noticeTemplateValue(routeDutyUser, jsonTemplate),
		"${duty_user_name}":                    noticeTemplateValue(dutyUser, jsonTemplate),
		"${duty_user_mentions}":                noticeTemplateValue(dutyMentions, jsonTemplate),
		"${duty_user_feishu_at}":               noticeTemplateValue(noticeDutyUserFeishuAtText(object), jsonTemplate),
		"${duty_user_dingtalk_at}":             noticeTemplateValue(noticeDutyUserDingTalkAtText(object), jsonTemplate),
		"${duty_user_wecom_at}":                noticeTemplateValue(noticeDutyUserWeComAtText(object), jsonTemplate),
		"${faultCenterId}":                     noticeTemplateValue(strconv.FormatUint(uint64(payload.FaultCenterID), 10), jsonTemplate),
		"${event_id}":                          noticeTemplateValue(strconv.FormatUint(uint64(payload.EventID), 10), jsonTemplate),
		"${event_url}":                         noticeTemplateValue(eventURL, jsonTemplate),
		"${event_link_text}":                   noticeTemplateValue(eventLinkText, jsonTemplate),
		"${callback_summary}":                  noticeTemplateValue(callbacks.Summary, jsonTemplate),
		"${callback_links}":                    noticeTemplateValue(callbacks.DetailText, jsonTemplate),
		"${callback_details}":                  noticeTemplateValue(callbacks.DetailText, jsonTemplate),
		"${event_count}":                       noticeTemplateValue(strconv.Itoa(payload.EventCount), jsonTemplate),
		"${instances}":                         noticeTemplateValue(instancesText, jsonTemplate),
		"${aggregate_details}":                 noticeTemplateValue(aggregateDetails, jsonTemplate),
		"${aggregate_events}":                  noticeTemplateValue(aggregateDetails, jsonTemplate),
		"${escalated}":                         noticeTemplateValue(escalatedText, jsonTemplate),
		"${escalation_text}":                   noticeTemplateValue(payload.EscalationText, jsonTemplate),
	}
	for key, value := range replacements {
		text = strings.ReplaceAll(text, key, value)
	}
	text = replaceNoticeLabelVariables(text, labels, jsonTemplate)
	text = replaceNoticeAnnotationVariables(text, annotations, jsonTemplate)
	text = appendMatchedLogsBlockIfMissing(text, matchedLogs, matchedLogsBlock, jsonTemplate)
	return text
}

func aggregateNotificationValuesText(events []ruleNotificationEventPayload) string {
	if len(events) == 0 {
		return "-"
	}
	lines := make([]string, 0, len(events))
	for _, event := range events {
		instance := firstNonEmpty(event.Instance, "-")
		lines = append(lines, fmt.Sprintf("%s=%s", instance, formatRuleValue(event.Value)))
	}
	return strings.Join(lines, "；")
}

func aggregatePayloadInstancesText(payload ruleNotificationPayload) string {
	if !payload.Aggregated || len(payload.Events) == 0 {
		return firstNonEmpty(parseStringMap(payload.Labels)["instance"], "-")
	}
	instances := make([]string, 0, len(payload.Events))
	seen := map[string]struct{}{}
	for _, event := range payload.Events {
		instance := strings.TrimSpace(event.Instance)
		if instance == "" || instance == "-" {
			continue
		}
		if _, ok := seen[instance]; ok {
			continue
		}
		seen[instance] = struct{}{}
		instances = append(instances, instance)
	}
	sort.Strings(instances)
	if len(instances) == 0 {
		return "-"
	}
	return strings.Join(instances, "、")
}

func aggregatePayloadFingerprintsText(payload ruleNotificationPayload) string {
	if !payload.Aggregated || len(payload.Events) == 0 {
		return payload.Fingerprint
	}
	values := make([]string, 0, len(payload.Events))
	for _, event := range payload.Events {
		if strings.TrimSpace(event.Fingerprint) == "" {
			continue
		}
		values = append(values, event.Fingerprint)
	}
	return strings.Join(values, "\n")
}

func aggregatePayloadDetailsText(payload ruleNotificationPayload) string {
	if !payload.Aggregated || len(payload.Events) == 0 {
		return firstNonEmpty(noticeAnnotationText(payload.Annotations, parseStringMap(payload.Annotations)), payload.Message)
	}
	lines := make([]string, 0, len(payload.Events))
	for _, event := range payload.Events {
		instance := firstNonEmpty(event.Instance, "-")
		annotation := firstNonEmpty(event.AnnotationText, event.Message)
		lines = append(lines, fmt.Sprintf("- %s：当前值 %s，%s", instance, formatRuleValue(event.Value), annotation))
	}
	return strings.Join(lines, "\n")
}

func replaceNoticeLabelVariables(text string, labels map[string]string, escapeJSON bool) string {
	return noticeLabelTemplatePattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := noticeLabelTemplatePattern.FindStringSubmatch(match)
		for _, part := range parts[1:] {
			if part != "" {
				return noticeTemplateValue(labels[part], escapeJSON)
			}
		}
		return match
	})
}

func replaceNoticeAnnotationVariables(text string, annotations map[string]string, escapeJSON bool) string {
	return noticeAnnotationTemplatePattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := noticeAnnotationTemplatePattern.FindStringSubmatch(match)
		for _, part := range parts[1:] {
			if part != "" {
				return noticeTemplateValue(annotations[part], escapeJSON)
			}
		}
		return match
	})
}

func looksLikeJSONTemplate(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

func noticeTemplateValue(value string, escapeJSON bool) string {
	if !escapeJSON {
		return value
	}
	data, err := json.Marshal(value)
	if err != nil || len(data) < 2 {
		return value
	}
	return string(data[1 : len(data)-1])
}

func noticeAnnotationText(raw string, annotations map[string]string) string {
	if len(annotations) == 0 {
		return raw
	}
	preferred := []string{"description", "detail", "summary", "message"}
	for _, key := range preferred {
		if value := strings.TrimSpace(annotations[key]); value != "" {
			return formatAnnotationMatchedLogsAsCodeBlock(value, annotations)
		}
	}
	keys := make([]string, 0, len(annotations))
	for key := range annotations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(annotations[key])
		if value == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", key, value))
	}
	return formatAnnotationMatchedLogsAsCodeBlock(strings.Join(lines, "\n"), annotations)
}

func noticeTemplateUsesAnnotations(text string) bool {
	return strings.Contains(text, "{{annotations}}") || strings.Contains(text, "${annotations}")
}

func formatAnnotationMatchedLogsAsCodeBlock(text string, annotations map[string]string) string {
	logs := strings.TrimSpace(noticeMatchedLogsText(annotations))
	if strings.TrimSpace(text) == "" || logs == "" || strings.Contains(text, "```") {
		return text
	}
	if !strings.Contains(text, logs) {
		return text
	}
	return strings.Replace(text, logs, matchedLogsCodeBlock(logs), 1)
}

func annotationContainsMatchedLogs(text, logs string) bool {
	text = strings.TrimSpace(text)
	logs = strings.TrimSpace(logs)
	if text == "" || logs == "" {
		return false
	}
	if strings.Contains(text, logs) || strings.Contains(text, matchedLogsCodeBlock(logs)) {
		return true
	}
	return strings.Contains(text, "```") && strings.Contains(text, sanitizeMarkdownCodeBlock(logs))
}

func noticeMatchedLogsText(annotations map[string]string) string {
	if len(annotations) == 0 {
		return ""
	}
	return strings.TrimSpace(firstNonEmpty(annotations["matched_logs"], annotations["matchedLogs"]))
}

func noticeMatchedLogCountText(annotations map[string]string) string {
	value := strings.TrimSpace(firstNonEmpty(annotations["matched_log_count"], annotations["matchedLogCount"]))
	if value != "" {
		return value
	}
	if logs := noticeMatchedLogsText(annotations); logs != "" {
		return strconv.Itoa(len(strings.Split(logs, "\n")))
	}
	return "0"
}

func noticeMatchedLogsBlock(logs string) string {
	logs = strings.TrimSpace(logs)
	if logs == "" {
		return ""
	}
	return "命中日志：\n" + matchedLogsCodeBlock(logs)
}

func appendMatchedLogsBlockIfMissing(text, logs, block string, jsonTemplate bool) string {
	text = strings.TrimSpace(text)
	logs = strings.TrimSpace(logs)
	block = strings.TrimSpace(block)
	if jsonTemplate || text == "" || logs == "" || block == "" {
		return text
	}
	if noticeTextContainsMatchedLogs(text, logs) {
		return text
	}
	return text + "\n\n" + block
}

func noticeTextContainsMatchedLogs(text, logs string) bool {
	text = strings.TrimSpace(text)
	logs = strings.TrimSpace(logs)
	if text == "" || logs == "" {
		return false
	}
	return strings.Contains(text, logs) || strings.Contains(text, sanitizeMarkdownCodeBlock(logs))
}

func matchedLogsCodeBlock(logs string) string {
	logs = strings.TrimSpace(logs)
	if logs == "" {
		return ""
	}
	return "```text\n" + sanitizeMarkdownCodeBlock(logs) + "\n```"
}

func sanitizeMarkdownCodeBlock(text string) string {
	return strings.ReplaceAll(text, "```", "'''")
}

func noticeEventTab(payload ruleNotificationPayload) string {
	if payload.State == "recovered" || payload.EndedAt != nil {
		return "history"
	}
	return "active"
}

func noticeEventLinkText(payload ruleNotificationPayload) string {
	if payload.Aggregated && noticeEventTab(payload) == "active" {
		return "查看活跃告警"
	}
	return "查看事件"
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func noticeDutyUserText(object model.NoticeObject) string {
	if len(object.CurrentDutyUsers) == 0 {
		return "-"
	}
	names := make([]string, 0, len(object.CurrentDutyUsers))
	for _, user := range object.CurrentDutyUsers {
		switch {
		case strings.TrimSpace(user.RealName) != "":
			names = append(names, strings.TrimSpace(user.RealName))
		case strings.TrimSpace(user.Username) != "":
			names = append(names, strings.TrimSpace(user.Username))
		}
	}
	if len(names) == 0 {
		return "-"
	}
	return strings.Join(names, "、")
}

func noticeDutyUserMentionText(object model.NoticeObject, noticeType string) string {
	switch normalizeNoticeType(noticeType) {
	case "FeiShu":
		return noticeDutyUserFeishuAtText(object)
	case "DingDing":
		return noticeDutyUserDingTalkAtText(object)
	case "WeChat":
		return noticeDutyUserWeComAtText(object)
	default:
		return noticeDutyUserText(object)
	}
}

func noticeRouteDutyUserText(object model.NoticeObject, noticeType string) string {
	if normalizeNoticeType(noticeType) == "FeiShu" {
		return noticeDutyUserFeishuAtText(object)
	}
	return noticeDutyUserText(object)
}

func noticeDutyUserFeishuAtText(object model.NoticeObject) string {
	items := make([]string, 0, len(object.CurrentDutyUsers))
	for _, user := range object.CurrentDutyUsers {
		id := firstNonEmpty(user.NotifyUserID, user.FeishuOpenID, user.FeishuUserID)
		name := dutyUserDisplayName(user)
		if strings.TrimSpace(id) != "" {
			if feishuIdentifierLooksResolvable(id) {
				items = append(items, feishuAtTag(strings.TrimSpace(id), name))
			} else if name != "" {
				items = append(items, "@"+name)
			} else {
				items = append(items, "@"+strings.TrimSpace(id))
			}
			continue
		}
		if name != "" {
			items = append(items, "@"+name)
		}
	}
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, " ")
}

func feishuAtTag(id string, name string) string {
	displayName := firstNonEmpty(name, id)
	return fmt.Sprintf(`<at id="%s">%s</at>`, html.EscapeString(strings.TrimSpace(id)), html.EscapeString(displayName))
}

func feishuIdentifierLooksResolvable(id string) bool {
	id = strings.TrimSpace(id)
	return strings.HasPrefix(id, "ou_") || strings.HasPrefix(id, "on_") || strings.HasPrefix(id, "un_")
}

func dutyUserDisplayName(user model.DutyUser) string {
	return firstNonEmpty(user.RealName, user.Username, user.Email, user.Phone)
}

func noticeDutyUserDingTalkAtText(object model.NoticeObject) string {
	items := make([]string, 0, len(object.CurrentDutyUsers))
	for _, user := range object.CurrentDutyUsers {
		id := firstNonEmpty(user.NotifyUserID, user.Phone, user.DingTalkUserID)
		if strings.TrimSpace(id) == "" {
			continue
		}
		items = append(items, "@"+strings.TrimSpace(id))
	}
	if len(items) == 0 {
		return noticeDutyUserText(object)
	}
	return strings.Join(items, " ")
}

func noticeDutyUserWeComAtText(object model.NoticeObject) string {
	items := make([]string, 0, len(object.CurrentDutyUsers))
	for _, user := range object.CurrentDutyUsers {
		id := firstNonEmpty(user.NotifyUserID, user.WeComUserID, user.Phone)
		if strings.TrimSpace(id) == "" {
			continue
		}
		if user.WeComUserID != "" || (user.NotifyUserID != "" && !isPhoneLike(user.NotifyUserID)) {
			items = append(items, fmt.Sprintf("<@%s>", strings.TrimSpace(id)))
		} else {
			items = append(items, "@"+strings.TrimSpace(id))
		}
	}
	if len(items) == 0 {
		return noticeDutyUserText(object)
	}
	return strings.Join(items, " ")
}

func noticeDingTalkAtLists(object model.NoticeObject) ([]string, []string) {
	mobiles := make([]string, 0, len(object.CurrentDutyUsers))
	userIDs := make([]string, 0, len(object.CurrentDutyUsers))
	for _, user := range object.CurrentDutyUsers {
		if notifyID := strings.TrimSpace(user.NotifyUserID); notifyID != "" {
			if isPhoneLike(notifyID) {
				mobiles = append(mobiles, notifyID)
			} else {
				userIDs = append(userIDs, notifyID)
			}
		}
		if phone := strings.TrimSpace(user.Phone); phone != "" {
			mobiles = append(mobiles, phone)
		}
		if userID := strings.TrimSpace(user.DingTalkUserID); userID != "" {
			userIDs = append(userIDs, userID)
		}
	}
	return uniqueStrings(mobiles), uniqueStrings(userIDs)
}

func isPhoneLike(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	digitCount := 0
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digitCount++
		case r == '+' || r == '-' || r == ' ':
		default:
			return false
		}
	}
	return digitCount >= 6
}

func defaultRuleNotificationText(payload ruleNotificationPayload) string {
	if payload.State == "recovered" || payload.EndedAt != nil {
		return "监控恢复\n规则：{{ruleName}}\n数据源：{{dataSourceName}}\n等级：{{severity}}\n最近值：{{value}}\n时间：{{time}}\n{{message}}\n{{matchedLogsBlock}}\n回调查询：{{callbackSummary}}\n{{callbackLinks}}\n查看事件：{{eventUrl}}"
	}
	if payload.Aggregated {
		return "监控告警\n规则：{{ruleName}}\n数据源：{{dataSourceName}}\n等级：{{severity}}\n状态：{{state}}\n当前值：{{value}}\n条件：{{condition}} {{threshold}}\n时间：{{time}}\n{{message}}\n{{matchedLogsBlock}}\n回调查询：{{callbackSummary}}\n{{callbackLinks}}\n查看活跃告警：{{eventUrl}}"
	}
	return "监控告警\n规则：{{ruleName}}\n数据源：{{dataSourceName}}\n等级：{{severity}}\n状态：{{state}}\n当前值：{{value}}\n条件：{{condition}} {{threshold}}\n时间：{{time}}\n{{message}}\n{{matchedLogsBlock}}\n回调查询：{{callbackSummary}}\n{{callbackLinks}}\n查看事件：{{eventUrl}}"
}

func noticeTemplateID(raw interface{}) uint {
	switch value := raw.(type) {
	case float64:
		return uint(value)
	case int:
		return uint(value)
	case uint:
		return value
	case string:
		id, _ := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
		return uint(id)
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func buildNoticeReceivers(object model.NoticeObject, route noticeObjectRouteConfig) []string {
	receivers := append([]string{}, route.To...)
	for _, user := range object.CurrentDutyUsers {
		switch {
		case strings.TrimSpace(user.Email) != "":
			receivers = append(receivers, user.Email)
		case strings.TrimSpace(user.Phone) != "":
			receivers = append(receivers, user.Phone)
		case strings.TrimSpace(user.NotifyUserID) != "":
			receivers = append(receivers, user.NotifyUserID)
		case strings.TrimSpace(user.Username) != "":
			receivers = append(receivers, user.Username)
		}
	}
	return uniqueStrings(receivers)
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func mergeNotifyStatus(statuses []string, errors []string) (string, error) {
	hasSuccess := false
	hasPartial := false
	hasFailure := len(errors) > 0
	hasTarget := false
	for _, status := range statuses {
		switch status {
		case "success":
			hasSuccess = true
			hasTarget = true
		case "partial":
			hasPartial = true
			hasTarget = true
		case "failed":
			hasFailure = true
			hasTarget = true
		case "none":
		default:
			if status != "" {
				hasTarget = true
			}
		}
	}
	if hasSuccess && !hasFailure && !hasPartial {
		return "success", nil
	}
	if hasSuccess || hasPartial {
		if len(errors) > 0 {
			return "partial", fmt.Errorf("%s", strings.Join(errors, "; "))
		}
		return "partial", nil
	}
	if hasFailure {
		return "failed", fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	if !hasTarget {
		return "none", nil
	}
	return "none", nil
}

func sendRuleNotificationToChannel(ctx context.Context, channel model.AlertChannel, payload ruleNotificationPayload) error {
	var config alertChannelConfig
	if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
		return fmt.Errorf("通道配置不是合法 JSON")
	}

	valueText := formatRuleValue(payload.Value)
	detailText := payload.Message
	text := fmt.Sprintf("监控告警\n规则：%s\n数据源：%s\n等级：%s\n状态：%s\n当前值：%s\n条件：%s %s\n时间：%s\n%s",
		payload.RuleName,
		payload.DataSourceName,
		payload.Severity,
		getStateText(payload.State),
		valueText,
		getConditionText(payload.Condition),
		formatRuleValue(payload.Threshold),
		payload.Time.Format("2006-01-02 15:04:05"),
		detailText,
	)

	switch channel.ChannelType {
	case "webhook":
		if strings.TrimSpace(config.WebhookURL) == "" {
			return fmt.Errorf("Webhook URL 未配置")
		}
		return postJSON(ctx, config.WebhookURL, payload)
	case "wechat":
		if strings.TrimSpace(config.WeChatWebhook) == "" {
			return fmt.Errorf("企业微信 Webhook 未配置")
		}
		return postJSON(ctx, config.WeChatWebhook, gin.H{"msgtype": "text", "text": gin.H{"content": text}})
	case "dingtalk":
		if strings.TrimSpace(config.DingTalkWebhook) == "" {
			return fmt.Errorf("钉钉 Webhook 未配置")
		}
		targetURL, err := buildDingTalkWebhookURL(config.DingTalkWebhook, config.DingTalkSecret, time.Now())
		if err != nil {
			return fmt.Errorf("钉钉 Webhook 地址无效: %w", err)
		}
		return postJSON(ctx, targetURL, gin.H{
			"msgtype": "markdown",
			"markdown": gin.H{
				"title": "监控告警",
				"text":  strings.ReplaceAll(text, "\n", "\n\n"),
			},
		})
	case "feishu":
		if strings.TrimSpace(config.FeishuWebhook) == "" {
			return fmt.Errorf("飞书 Webhook 未配置")
		}
		return postJSON(ctx, config.FeishuWebhook, gin.H{"msg_type": "text", "content": gin.H{"text": text}})
	case "email":
		return fmt.Errorf("邮件通道需要接收人编排，当前规则仅支持 Webhook、企业微信、钉钉、飞书直发")
	default:
		return fmt.Errorf("不支持的通道类型: %s", channel.ChannelType)
	}
}

func getStateText(state string) string {
	switch state {
	case "firing":
		return "告警中"
	case "pending":
		return "待触发"
	case "recovered":
		return "已恢复"
	case "error":
		return "评估失败"
	default:
		return state
	}
}

func postJSON(ctx context.Context, targetURL string, payload interface{}) error {
	return postJSONWithHeaders(ctx, targetURL, payload, nil)
}

func buildDingTalkWebhookURL(targetURL, secret string, now time.Time) (string, error) {
	targetURL = strings.TrimSpace(targetURL)
	secret = strings.TrimSpace(secret)
	if targetURL == "" || secret == "" {
		return targetURL, nil
	}
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}
	timestamp := strconv.FormatInt(now.UnixNano()/int64(time.Millisecond), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(timestamp + "\n" + secret)); err != nil {
		return "", err
	}
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	query := parsed.Query()
	query.Set("timestamp", timestamp)
	query.Set("sign", sign)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func postJSONWithHeaders(ctx context.Context, targetURL string, payload interface{}, headers map[string]string) error {
	body, err := marshalJSONBody(payload)
	if err != nil {
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || strings.EqualFold(key, "Content-Length") {
			continue
		}
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	if err := webhookBusinessResponseError(respBody); err != nil {
		return err
	}
	return nil
}

func webhookBusinessResponseError(respBody []byte) error {
	respText := strings.TrimSpace(string(respBody))
	if respText == "" {
		return nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal(respBody, &data); err != nil || len(data) == 0 {
		return nil
	}
	codeValue, ok := firstWebhookResponseField(data, "code", "errcode", "statusCode", "StatusCode", "status_code")
	if !ok {
		return nil
	}
	codeText := strings.TrimSpace(fmt.Sprint(codeValue))
	if codeText == "" || codeText == "0" || codeText == "0.0" {
		return nil
	}
	messageValue, _ := firstWebhookResponseField(data, "msg", "message", "errmsg", "StatusMessage", "statusMessage", "error")
	message := strings.TrimSpace(fmt.Sprint(messageValue))
	if message == "" || message == "<nil>" {
		message = respText
	}
	message = friendlyWebhookBusinessMessage(codeText, message)
	return fmt.Errorf("Webhook 返回错误 code=%s: %s", codeText, message)
}

func friendlyWebhookBusinessMessage(codeText, message string) string {
	text := strings.TrimSpace(message)
	if text == "" {
		return text
	}
	lower := strings.ToLower(text)
	hints := make([]string, 0, 2)
	if strings.Contains(lower, "keywords") || strings.Contains(lower, "keyword") || strings.Contains(lower, "not in content") {
		hints = append(hints, "钉钉机器人可能开启了关键词安全设置，请让通知模板正文包含机器人配置的关键词；默认钉钉模板已包含 OpsHub/告警")
	}
	if strings.Contains(lower, "sign") || strings.Contains(lower, "signature") {
		hints = append(hints, "如果机器人未开启加签，请不要填写签名密钥；如果开启了加签，请填写 SEC 开头的加签密钥")
	}
	if strings.Contains(lower, "access_token") || strings.Contains(lower, "token") {
		hints = append(hints, "请检查 Webhook 地址中的 access_token 是否完整有效")
	}
	if len(hints) == 0 {
		return text
	}
	return fmt.Sprintf("%s（%s）", text, strings.Join(hints, "；"))
}

func firstWebhookResponseField(data map[string]interface{}, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func marshalJSONBody(payload interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
