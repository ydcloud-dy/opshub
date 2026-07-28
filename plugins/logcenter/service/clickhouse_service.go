package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

type CapacityEstimate struct {
	StorageID            uint            `json:"storageId"`
	StorageName          string          `json:"storageName"`
	CurrentRows          int64           `json:"currentRows"`
	CurrentCompressed    int64           `json:"currentCompressedBytes"`
	CurrentStored        int64           `json:"currentStoredBytes"`
	CurrentUncompressed  int64           `json:"currentUncompressedBytes"`
	CompressionRatio     float64         `json:"compressionRatio"`
	LogsLast24Hours      int64           `json:"logsLast24Hours"`
	RawBytesLast24Hours  int64           `json:"rawBytesLast24Hours"`
	AverageRecordBytes   float64         `json:"averageRecordBytes"`
	AverageStoredBytes   float64         `json:"averageStoredRecordBytes"`
	DailyStoredBytes     int64           `json:"dailyStoredBytes"`
	ForecastBasis        string          `json:"forecastBasis"`
	SafetyFactor         float64         `json:"safetyFactor"`
	RetentionDays        int             `json:"retentionDays"`
	ProjectedStoredBytes int64           `json:"projectedStoredBytes"`
	RecommendedBytes     int64           `json:"recommendedBytes"`
	DiskTotalBytes       int64           `json:"diskTotalBytes"`
	DiskFreeBytes        int64           `json:"diskFreeBytes"`
	DiskReservedBytes    int64           `json:"diskReservedBytes"`
	UsableDiskBytes      int64           `json:"usableDiskBytes"`
	UsableFreeBytes      int64           `json:"usableFreeBytes"`
	ProjectedUsage       float64         `json:"projectedUsagePercent"`
	DaysUntilFull        float64         `json:"daysUntilFull"`
	Retention            RetentionHealth `json:"retention"`
}

type RetentionHealth struct {
	Status                 string  `json:"status"`
	ExpiredRows            int64   `json:"expiredRows"`
	ExpiredParts           int64   `json:"expiredParts"`
	OldestExpiredAt        string  `json:"oldestExpiredAt,omitempty"`
	TTLLagSeconds          int64   `json:"ttlLagSeconds"`
	TTLMergeActive         bool    `json:"ttlMergeActive"`
	TTLMergeProgress       float64 `json:"ttlMergeProgress"`
	TTLMergeTimeoutSeconds int     `json:"ttlMergeTimeoutSeconds"`
}

var clickHouseIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const (
	capacitySafetyFactor     = 1.3
	capacityDiskReserveRatio = 0.2
)

// ClickHouseService provides the small HTTP surface needed by the log center.
type ClickHouseService struct {
	secureClient   *http.Client
	insecureClient *http.Client
}

func NewClickHouseService() *ClickHouseService {
	return &ClickHouseService{
		secureClient:   newHTTPClient(false),
		insecureClient: newHTTPClient(true),
	}
}

func (s *ClickHouseService) Ping(ctx context.Context, cluster logmodel.StorageCluster, password string) error {
	raw, err := s.request(ctx, cluster, password, http.MethodGet, "/ping", nil, nil)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(raw)) != "Ok." {
		return fmt.Errorf("ClickHouse 健康检查返回异常: %s", strings.TrimSpace(string(raw)))
	}
	return nil
}

func (s *ClickHouseService) Initialize(ctx context.Context, cluster logmodel.StorageCluster, password string) error {
	database, err := normalizeClickHouseIdentifier(cluster.DatabaseName, "opshub_logs")
	if err != nil {
		return err
	}
	clusterName, err := configuredClickHouseClusterName(cluster)
	if err != nil {
		return err
	}
	retentionDays := cluster.DefaultRetentionDays
	if retentionDays <= 0 {
		retentionDays = 30
	}
	if retentionDays > 3650 {
		return fmt.Errorf("默认保留天数不能超过 3650 天")
	}
	metricsRetentionDays := retentionDays
	if metricsRetentionDays < 400 {
		metricsRetentionDays = 400
	}
	ttlMergeTimeoutSeconds := clickHouseTTLMergeTimeoutSeconds()
	onCluster := ""
	logsEngine := "MergeTree"
	metricsEngine := "SummingMergeTree"
	deduplicationSetting := "non_replicated_deduplication_window = 100000"
	if clusterName != "" {
		onCluster = fmt.Sprintf(" ON CLUSTER `%s`", clusterName)
		logsEngine = "ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}')"
		metricsEngine = "ReplicatedSummingMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}')"
		deduplicationSetting = "replicated_deduplication_window = 100000"
	}

	statements := []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`%s", database, onCluster),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.opshub_logs%s
(
    tenant_id UInt64 DEFAULT 1,
    event_date Date MATERIALIZED toDate(timestamp),
    timestamp DateTime64(9, 'UTC'),
    observed_at DateTime64(9, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
    source_type LowCardinality(String),
    asset_type LowCardinality(String),
    asset_id UInt64,
    host_id UInt64 DEFAULT 0,
    cluster_id UInt64 DEFAULT 0,
    environment LowCardinality(String),
    service LowCardinality(String),
    level LowCardinality(String),
	retention_days UInt16 DEFAULT %d,
	expire_at DateTime('UTC') DEFAULT toDateTime(timestamp, 'UTC') + toIntervalDay(retention_days),
    namespace LowCardinality(String),
    workload_kind LowCardinality(String),
    workload_name String,
    pod_name String,
    pod_uid String,
    container_name LowCardinality(String),
    container_image String,
    node_name String,
    file_path String,
    stream LowCardinality(String),
    body String CODEC(ZSTD(3)),
    attributes Map(String, String) CODEC(ZSTD(3)),
    resource_attributes Map(String, String) CODEC(ZSTD(3)),
    trace_id String DEFAULT '',
    span_id String DEFAULT '',
    agent_id String,
    policy_id UInt64,
    policy_version UInt64,
    batch_id UUID,
    sequence UInt64,
    fingerprint UInt64,
    INDEX idx_level level TYPE set(64) GRANULARITY 4,
    INDEX idx_body lowerUTF8(body) TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 4,
    INDEX idx_trace trace_id TYPE bloom_filter(0.01) GRANULARITY 4
)
ENGINE = %s
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, asset_type, asset_id, service, event_date, timestamp, fingerprint, sequence)
TTL expire_at DELETE
SETTINGS index_granularity = 8192, %s, merge_with_ttl_timeout = %d`, database, onCluster, retentionDays, logsEngine, deduplicationSetting, ttlMergeTimeoutSeconds),
		fmt.Sprintf(`ALTER TABLE %s.opshub_logs%s
MODIFY SETTING %s, merge_with_ttl_timeout = %d`, database, onCluster, deduplicationSetting, ttlMergeTimeoutSeconds),
		fmt.Sprintf(`ALTER TABLE %s.opshub_logs%s
ADD COLUMN IF NOT EXISTS retention_days UInt16 DEFAULT %d AFTER level`, database, onCluster, retentionDays),
		fmt.Sprintf(`ALTER TABLE %s.opshub_logs%s
ADD COLUMN IF NOT EXISTS expire_at DateTime('UTC') DEFAULT toDateTime(timestamp, 'UTC') + toIntervalDay(retention_days) AFTER retention_days`, database, onCluster),
		fmt.Sprintf(`ALTER TABLE %s.opshub_logs%s
MODIFY TTL expire_at DELETE`, database, onCluster),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.opshub_log_metrics_1m%s
(
    minute DateTime('UTC'),
    tenant_id UInt64,
    asset_type LowCardinality(String),
    asset_id UInt64,
    cluster_id UInt64,
    namespace LowCardinality(String),
    service LowCardinality(String),
    level LowCardinality(String),
    log_count UInt64,
    byte_count UInt64
)
ENGINE = %s
PARTITION BY toYYYYMM(minute)
ORDER BY (tenant_id, asset_type, asset_id, cluster_id, namespace, service, level, minute)
TTL toDate(minute) + INTERVAL %d DAY DELETE
SETTINGS merge_with_ttl_timeout = %d`, database, onCluster, metricsEngine, metricsRetentionDays, ttlMergeTimeoutSeconds),
		fmt.Sprintf(`ALTER TABLE %s.opshub_log_metrics_1m%s
MODIFY TTL toDate(minute) + INTERVAL %d DAY DELETE`, database, onCluster, metricsRetentionDays),
		fmt.Sprintf(`ALTER TABLE %s.opshub_log_metrics_1m%s
MODIFY SETTING merge_with_ttl_timeout = %d`, database, onCluster, ttlMergeTimeoutSeconds),
		fmt.Sprintf(`CREATE MATERIALIZED VIEW IF NOT EXISTS %s.opshub_log_metrics_1m_mv%s
TO %s.opshub_log_metrics_1m
AS SELECT
    toStartOfMinute(timestamp) AS minute,
    tenant_id,
    asset_type,
    asset_id,
    cluster_id,
    namespace,
    service,
    level,
    count() AS log_count,
    sum(length(body)) AS byte_count
FROM %s.opshub_logs
GROUP BY minute, tenant_id, asset_type, asset_id, cluster_id, namespace, service, level`, database, onCluster, database, database),
	}
	for _, statement := range statements {
		if _, err := s.Execute(ctx, cluster, password, statement, nil); err != nil {
			return err
		}
	}
	return nil
}

func configuredClickHouseClusterName(cluster logmodel.StorageCluster) (string, error) {
	name := strings.TrimSpace(os.Getenv("OPSHUB_LOGCENTER_CLICKHOUSE_CLUSTER"))
	if name == "" {
		return "", nil
	}
	if !clickHouseIdentifierPattern.MatchString(name) {
		return "", fmt.Errorf("ClickHouse 集群名称不合法: %s", name)
	}
	configuredEndpoint, err := firstClickHouseEndpoint(os.Getenv("OPSHUB_LOGCENTER_CLICKHOUSE_ENDPOINT"))
	if err != nil {
		return "", fmt.Errorf("配置 ClickHouse 集群名称时必须同时配置内置存储地址: %w", err)
	}
	storageEndpoint, err := firstClickHouseEndpoint(cluster.Endpoints)
	if err != nil {
		return "", err
	}
	if configuredEndpoint != storageEndpoint {
		return "", nil
	}
	return name, nil
}

func (s *ClickHouseService) Execute(ctx context.Context, cluster logmodel.StorageCluster, password, query string, params map[string]string) ([]byte, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("ClickHouse 查询不能为空")
	}
	return s.request(ctx, cluster, password, http.MethodPost, "/", params, strings.NewReader(query))
}

func (s *ClickHouseService) QueryJSONEachRow(ctx context.Context, cluster logmodel.StorageCluster, password, query string, params map[string]string) ([]map[string]interface{}, error) {
	raw, err := s.Execute(ctx, cluster, password, strings.TrimSpace(query)+" FORMAT JSONEachRow", params)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]interface{}, 0)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var row map[string]interface{}
		if err := json.Unmarshal(line, &row); err != nil {
			return nil, fmt.Errorf("解析 ClickHouse 查询结果失败: %w", err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 ClickHouse 查询结果失败: %w", err)
	}
	return rows, nil
}

func (s *ClickHouseService) EstimateCapacity(ctx context.Context, cluster logmodel.StorageCluster, password string, retentionDays int) (CapacityEstimate, error) {
	database, err := normalizeClickHouseIdentifier(cluster.DatabaseName, "opshub_logs")
	if err != nil {
		return CapacityEstimate{}, err
	}
	if retentionDays <= 0 {
		retentionDays = cluster.DefaultRetentionDays
	}
	if retentionDays <= 0 {
		retentionDays = 30
	}
	if retentionDays > 3650 {
		return CapacityEstimate{}, fmt.Errorf("容量预测保留天数不能超过 3650 天")
	}
	parts, err := s.QueryJSONEachRow(ctx, cluster, password, `SELECT
    toString(sum(rows)) AS rows,
    toString(sum(data_compressed_bytes)) AS compressedBytes,
	toString(sum(bytes_on_disk)) AS storedBytes,
    toString(sum(data_uncompressed_bytes)) AS uncompressedBytes
FROM system.parts
WHERE active AND database = {database:String} AND table = 'opshub_logs'`, map[string]string{"database": database})
	if err != nil {
		return CapacityEstimate{}, err
	}
	metrics, err := s.QueryJSONEachRow(ctx, cluster, password, fmt.Sprintf(`SELECT
    toString(sum(log_count)) AS logs,
    toString(sum(byte_count)) AS rawBytes
FROM %s.opshub_log_metrics_1m
WHERE minute >= now() - INTERVAL 24 HOUR`, database), nil)
	if err != nil {
		return CapacityEstimate{}, err
	}
	disks, err := s.QueryJSONEachRow(ctx, cluster, password, `SELECT
    toString(sum(total_space)) AS totalBytes,
    toString(sum(free_space)) AS freeBytes
FROM system.disks`, nil)
	if err != nil {
		return CapacityEstimate{}, err
	}
	retention, err := s.RetentionHealth(ctx, cluster, password)
	if err != nil {
		return CapacityEstimate{}, err
	}
	expiredRows, err := s.QueryJSONEachRow(ctx, cluster, password, fmt.Sprintf(`SELECT
    toString(countIf(expire_at < now64(9))) AS expiredRows
FROM %s.opshub_logs`, database), nil)
	if err != nil {
		return CapacityEstimate{}, err
	}
	if len(expiredRows) > 0 {
		retention.ExpiredRows = capacityInt64(expiredRows[0]["expiredRows"])
	}
	result := CapacityEstimate{StorageID: cluster.ID, StorageName: cluster.Name, RetentionDays: retentionDays}
	result.Retention = retention
	if len(parts) > 0 {
		result.CurrentRows = capacityInt64(parts[0]["rows"])
		result.CurrentCompressed = capacityInt64(parts[0]["compressedBytes"])
		result.CurrentStored = capacityInt64(parts[0]["storedBytes"])
		result.CurrentUncompressed = capacityInt64(parts[0]["uncompressedBytes"])
	}
	if len(metrics) > 0 {
		result.LogsLast24Hours = capacityInt64(metrics[0]["logs"])
		result.RawBytesLast24Hours = capacityInt64(metrics[0]["rawBytes"])
	}
	if len(disks) > 0 {
		result.DiskTotalBytes = capacityInt64(disks[0]["totalBytes"])
		result.DiskFreeBytes = capacityInt64(disks[0]["freeBytes"])
	}
	result.CompressionRatio = 5
	if result.CurrentCompressed > 0 && result.CurrentUncompressed > 0 {
		result.CompressionRatio = float64(result.CurrentUncompressed) / float64(result.CurrentCompressed)
	}
	if result.CompressionRatio < 1 {
		result.CompressionRatio = 1
	}
	if result.LogsLast24Hours > 0 {
		result.AverageRecordBytes = float64(result.RawBytesLast24Hours) / float64(result.LogsLast24Hours)
	}
	if result.CurrentRows > 0 && result.CurrentStored > 0 {
		result.AverageStoredBytes = float64(result.CurrentStored) / float64(result.CurrentRows)
	}
	result.DailyStoredBytes, result.ForecastBasis = estimateDailyStoredBytes(result)
	if result.DailyStoredBytes == 0 && result.CurrentCompressed > 0 {
		storedBytes := result.CurrentStored
		if storedBytes == 0 {
			storedBytes = result.CurrentCompressed
		}
		result.DailyStoredBytes = storedBytes / int64(maxInt(retentionDays, 1))
		result.ForecastBasis = "retention_average"
	}
	result.SafetyFactor = capacitySafetyFactor
	result.ProjectedStoredBytes = result.DailyStoredBytes * int64(retentionDays)
	result.RecommendedBytes = int64(float64(result.ProjectedStoredBytes)*capacitySafetyFactor + 0.5)
	if result.DiskTotalBytes > 0 {
		result.DiskReservedBytes = int64(float64(result.DiskTotalBytes)*capacityDiskReserveRatio + 0.5)
		result.UsableDiskBytes = result.DiskTotalBytes - result.DiskReservedBytes
		if result.UsableDiskBytes > 0 {
			result.ProjectedUsage = float64(result.RecommendedBytes) / float64(result.UsableDiskBytes) * 100
		}
	}
	result.UsableFreeBytes = result.DiskFreeBytes - result.DiskReservedBytes
	if result.UsableFreeBytes < 0 {
		result.UsableFreeBytes = 0
	}
	if result.DailyStoredBytes > 0 {
		result.DaysUntilFull = float64(result.UsableFreeBytes) / (float64(result.DailyStoredBytes) * capacitySafetyFactor)
	}
	return result, nil
}

func estimateDailyStoredBytes(result CapacityEstimate) (int64, string) {
	rowEstimate := 0.0
	if result.LogsLast24Hours > 0 && result.AverageStoredBytes > 0 {
		rowEstimate = float64(result.LogsLast24Hours) * result.AverageStoredBytes
	}
	rawEstimate := 0.0
	if result.RawBytesLast24Hours > 0 && result.CompressionRatio > 0 {
		rawEstimate = float64(result.RawBytesLast24Hours) / result.CompressionRatio
	}
	if rowEstimate >= rawEstimate && rowEstimate > 0 {
		return int64(rowEstimate + 0.5), "stored_bytes_per_row"
	}
	if rawEstimate > 0 {
		return int64(rawEstimate + 0.5), "raw_bytes_compression"
	}
	return 0, "insufficient_data"
}

func (s *ClickHouseService) RetentionHealth(ctx context.Context, cluster logmodel.StorageCluster, password string) (RetentionHealth, error) {
	database, err := normalizeClickHouseIdentifier(cluster.DatabaseName, "opshub_logs")
	if err != nil {
		return RetentionHealth{}, err
	}
	parts, err := s.QueryJSONEachRow(ctx, cluster, password, `SELECT
    toString(countIf(delete_ttl_info_min > toDateTime(0) AND delete_ttl_info_min < now())) AS expiredParts,
    if(countIf(delete_ttl_info_min > toDateTime(0) AND delete_ttl_info_min < now()) = 0, '',
       formatDateTime(minIf(delete_ttl_info_min, delete_ttl_info_min > toDateTime(0) AND delete_ttl_info_min < now()), '%FT%TZ', 'UTC')) AS oldestExpiredAt,
    toString(if(countIf(delete_ttl_info_min > toDateTime(0) AND delete_ttl_info_min < now()) = 0, 0,
       dateDiff('second', minIf(delete_ttl_info_min, delete_ttl_info_min > toDateTime(0) AND delete_ttl_info_min < now()), now()))) AS ttlLagSeconds
FROM system.parts
WHERE active AND database = {database:String} AND table = 'opshub_logs'`, map[string]string{"database": database})
	if err != nil {
		return RetentionHealth{}, err
	}
	merges, err := s.QueryJSONEachRow(ctx, cluster, password, `SELECT
    toString(count()) AS mergeCount,
    toString(if(count() = 0, 0, max(progress))) AS mergeProgress
FROM system.merges
WHERE database = {database:String} AND table = 'opshub_logs'`, map[string]string{"database": database})
	if err != nil {
		return RetentionHealth{}, err
	}
	result := RetentionHealth{TTLMergeTimeoutSeconds: clickHouseTTLMergeTimeoutSeconds()}
	if len(parts) > 0 {
		result.ExpiredParts = capacityInt64(parts[0]["expiredParts"])
		result.OldestExpiredAt = strings.TrimSpace(fmt.Sprint(parts[0]["oldestExpiredAt"]))
		result.TTLLagSeconds = capacityInt64(parts[0]["ttlLagSeconds"])
	}
	if len(merges) > 0 {
		result.TTLMergeActive = capacityInt64(merges[0]["mergeCount"]) > 0
		result.TTLMergeProgress = capacityFloat64(merges[0]["mergeProgress"])
	}
	result.Status = retentionHealthStatus(result.ExpiredParts, result.TTLLagSeconds, result.TTLMergeTimeoutSeconds)
	return result, nil
}

func retentionHealthStatus(expiredParts, lagSeconds int64, timeoutSeconds int) string {
	if expiredParts == 0 || lagSeconds <= int64(timeoutSeconds*2) {
		return "healthy"
	}
	if lagSeconds <= int64(timeoutSeconds*6) {
		return "warning"
	}
	return "critical"
}

func clickHouseTTLMergeTimeoutSeconds() int {
	const defaultValue = 3600
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv("OPSHUB_LOG_TTL_MERGE_TIMEOUT_SECONDS")))
	if err != nil || value < 300 || value > 86400 {
		return defaultValue
	}
	return value
}

func capacityInt64(value interface{}) int64 {
	parsed, _ := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), 64)
	return int64(parsed)
}

func capacityFloat64(value interface{}) float64 {
	parsed, _ := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), 64)
	return parsed
}

func (s *ClickHouseService) request(ctx context.Context, cluster logmodel.StorageCluster, password, method, requestPath string, params map[string]string, body io.Reader) ([]byte, error) {
	endpoint, err := firstClickHouseEndpoint(cluster.Endpoints)
	if err != nil {
		return nil, err
	}
	target, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("ClickHouse 地址无效: %w", err)
	}
	if requestPath == "/ping" {
		target.Path = strings.TrimRight(target.Path, "/") + "/ping"
	}
	queryValues := target.Query()
	for key, value := range params {
		queryValues.Set("param_"+key, value)
	}
	target.RawQuery = queryValues.Encode()

	timeout := cluster.Timeout
	if timeout <= 0 {
		timeout = 300
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, method, target.String(), body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	}
	if strings.TrimSpace(cluster.Username) != "" {
		req.SetBasicAuth(cluster.Username, password)
	}
	client := s.secureClient
	if cluster.SkipTLSVerify {
		client = s.insecureClient
	}
	resp, err := client.Do(req)
	if err != nil {
		if requestCtx.Err() != nil {
			return nil, fmt.Errorf("ClickHouse 请求超时或已取消: %w", requestCtx.Err())
		}
		return nil, fmt.Errorf("请求 ClickHouse 失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("读取 ClickHouse 响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(raw))
		if len(message) > 1200 {
			message = message[:1200]
		}
		return nil, fmt.Errorf("ClickHouse 返回 %d: %s", resp.StatusCode, message)
	}
	return raw, nil
}

func firstClickHouseEndpoint(raw string) (string, error) {
	items := make([]string, 0)
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "[") {
		_ = json.Unmarshal([]byte(trimmed), &items)
	}
	if len(items) == 0 {
		items = strings.FieldsFunc(trimmed, func(r rune) bool {
			return r == ',' || r == '\n' || r == ';'
		})
	}
	if len(items) == 0 || strings.TrimSpace(items[0]) == "" {
		return "", fmt.Errorf("ClickHouse 地址不能为空")
	}
	endpoint := strings.TrimRight(strings.TrimSpace(items[0]), "/")
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("ClickHouse 地址无效")
	}
	return endpoint, nil
}

func NormalizeClickHouseEndpoint(raw string) (string, error) {
	return firstClickHouseEndpoint(raw)
}

func normalizeClickHouseIdentifier(value, fallback string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if !clickHouseIdentifierPattern.MatchString(value) {
		return "", fmt.Errorf("ClickHouse 数据库名称只能包含字母、数字和下划线，且不能以数字开头")
	}
	return value, nil
}
