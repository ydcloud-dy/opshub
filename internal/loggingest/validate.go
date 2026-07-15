package loggingest

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var batchIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type Limits struct {
	MaxBatchRecords int
	MaxRecordBytes  int
	MaxFutureSkew   time.Duration
	MaxPastAge      time.Duration
}

func DefaultLimits() Limits {
	return Limits{
		MaxBatchRecords: 2000,
		MaxRecordBytes:  1024 * 1024,
		MaxFutureSkew:   10 * time.Minute,
		MaxPastAge:      365 * 24 * time.Hour,
	}
}

func ValidateBatch(batch LogBatch, limits Limits) error {
	if !batchIDPattern.MatchString(strings.TrimSpace(batch.BatchID)) {
		return fmt.Errorf("batchId 必须是标准 UUID")
	}
	if strings.TrimSpace(batch.AgentID) == "" {
		return fmt.Errorf("agentId 不能为空")
	}
	if strings.TrimSpace(batch.AssetType) == "" || batch.AssetID == 0 {
		return fmt.Errorf("assetType 和 assetId 不能为空")
	}
	if len(batch.Records) == 0 {
		return fmt.Errorf("records 不能为空")
	}
	if limits.MaxBatchRecords <= 0 {
		limits = DefaultLimits()
	}
	if len(batch.Records) > limits.MaxBatchRecords {
		return fmt.Errorf("单批日志不能超过 %d 条", limits.MaxBatchRecords)
	}
	now := time.Now()
	for index, record := range batch.Records {
		if len(record.Body) > limits.MaxRecordBytes {
			return fmt.Errorf("第 %d 条日志超过 %d 字节", index+1, limits.MaxRecordBytes)
		}
		if record.TimestampUnixNano <= 0 {
			return fmt.Errorf("第 %d 条日志缺少时间戳", index+1)
		}
		timestamp := time.Unix(0, record.TimestampUnixNano)
		if limits.MaxFutureSkew > 0 && timestamp.After(now.Add(limits.MaxFutureSkew)) {
			return fmt.Errorf("第 %d 条日志时间超出未来允许范围", index+1)
		}
		if limits.MaxPastAge > 0 && timestamp.Before(now.Add(-limits.MaxPastAge)) {
			return fmt.Errorf("第 %d 条日志时间过旧", index+1)
		}
	}
	return nil
}
