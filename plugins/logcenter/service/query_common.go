package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const histogramMaxBuckets = 240

type LogQueryResponse struct {
	Items        []LogItem      `json:"items"`
	Total        int            `json:"total"`
	DurationMS   int64          `json:"durationMs"`
	Histogram    []HistogramBar `json:"histogram"`
	Fields       []FieldSummary `json:"fields"`
	NextCursor   string         `json:"nextCursor,omitempty"`
	HasMore      bool           `json:"hasMore,omitempty"`
	ScannedBytes int64          `json:"scannedBytes,omitempty"`
	Raw          any            `json:"raw,omitempty"`
}

type LogItem struct {
	Timestamp       string                 `json:"timestamp"`
	Message         string                 `json:"message"`
	Level           string                 `json:"level"`
	Labels          map[string]string      `json:"labels"`
	Fields          map[string]interface{} `json:"fields"`
	Raw             interface{}            `json:"raw,omitempty"`
	ContextSelected bool                   `json:"contextSelected,omitempty"`
}

type HistogramBar struct {
	Time  string `json:"time"`
	Count int    `json:"count"`
}

type FieldSummary struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Count  int    `json:"count"`
	Sample string `json:"sample"`
}

type FieldOption struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	DisplayName    string `json:"displayName,omitempty"`
	IsTimeField    bool   `json:"isTimeField,omitempty"`
	IsMessageField bool   `json:"isMessageField,omitempty"`
	IsLevelField   bool   `json:"isLevelField,omitempty"`
	Masked         bool   `json:"masked,omitempty"`
}

func applyLogFieldSecurity(items []LogItem, deniedFields, maskFields []string) []LogItem {
	denied := normalizedFieldSet(deniedFields)
	masked := normalizedFieldSet(maskFields)
	for index := range items {
		item := &items[index]
		if fieldSetMatches(denied, "message") || fieldSetMatches(denied, "body") {
			item.Message = ""
		} else if fieldSetMatches(masked, "message") || fieldSetMatches(masked, "body") {
			item.Message = "******"
		}
		if fieldSetMatches(denied, "level") {
			item.Level = ""
		} else if fieldSetMatches(masked, "level") {
			item.Level = "******"
		}
		secureStringMap(item.Labels, "labels", denied, masked)
		secureInterfaceMap(item.Fields, "fields", denied, masked)
		if raw, ok := item.Raw.(map[string]interface{}); ok {
			secureInterfaceMap(raw, "raw", denied, masked)
			item.Raw = raw
		}
	}
	return items
}

func FilterInternalFieldOptions(options []FieldOption, deniedFields, maskFields []string) []FieldOption {
	denied := normalizedFieldSet(deniedFields)
	masked := normalizedFieldSet(maskFields)
	result := make([]FieldOption, 0, len(options))
	for _, option := range options {
		if fieldSetMatches(denied, option.Name) {
			continue
		}
		option.Masked = fieldSetMatches(masked, option.Name)
		result = append(result, option)
	}
	return result
}

func FieldAccessDenied(field string, deniedFields []string) bool {
	return fieldSetMatches(normalizedFieldSet(deniedFields), field)
}

func appendKubernetesAccessCondition(conditions *[]string, params map[string]string, scopes map[uint64][]string, prefix string) {
	if scopes == nil {
		return
	}
	parts := []string{"cluster_id = 0"}
	clusterIDs := make([]uint64, 0, len(scopes))
	for clusterID := range scopes {
		clusterIDs = append(clusterIDs, clusterID)
	}
	sort.Slice(clusterIDs, func(left, right int) bool { return clusterIDs[left] < clusterIDs[right] })
	parameterIndex := 0
	for _, clusterID := range clusterIDs {
		namespaces := scopes[clusterID]
		if len(namespaces) == 0 {
			parts = append(parts, fmt.Sprintf("cluster_id = %d", clusterID))
			continue
		}
		placeholders := make([]string, 0, len(namespaces))
		for _, namespace := range namespaces {
			name := fmt.Sprintf("%s_%d", prefix, parameterIndex)
			parameterIndex++
			params[name] = namespace
			placeholders = append(placeholders, fmt.Sprintf("{%s:String}", name))
		}
		parts = append(parts, fmt.Sprintf("(cluster_id = %d AND namespace IN (%s))", clusterID, strings.Join(placeholders, ",")))
	}
	*conditions = append(*conditions, "("+strings.Join(parts, " OR ")+")")
}

func normalizedFieldSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = normalizeFieldName(value)
		if value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}

func normalizeFieldName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "raw.")
	value = strings.TrimPrefix(value, "fields.")
	return value
}

func fieldSetMatches(fields map[string]struct{}, path string) bool {
	path = normalizeFieldName(path)
	if _, ok := fields[path]; ok {
		return true
	}
	leaves := strings.Split(path, ".")
	if len(leaves) > 1 {
		_, ok := fields[leaves[len(leaves)-1]]
		return ok
	}
	return false
}

func secureStringMap(values map[string]string, prefix string, denied, masked map[string]struct{}) {
	for key := range values {
		path := strings.TrimPrefix(prefix+"."+key, ".")
		if fieldSetMatches(denied, path) {
			delete(values, key)
		} else if fieldSetMatches(masked, path) {
			values[key] = "******"
		}
	}
}

func secureInterfaceMap(values map[string]interface{}, prefix string, denied, masked map[string]struct{}) {
	for key, value := range values {
		path := strings.TrimPrefix(prefix+"."+key, ".")
		if fieldSetMatches(denied, path) {
			delete(values, key)
			continue
		}
		if fieldSetMatches(masked, path) {
			values[key] = "******"
			continue
		}
		switch typed := value.(type) {
		case map[string]interface{}:
			secureInterfaceMap(typed, path, denied, masked)
		case map[string]string:
			secureStringMap(typed, path, denied, masked)
		case []interface{}:
			secureInterfaceSlice(typed, path, denied, masked)
		}
	}
}

func secureInterfaceSlice(values []interface{}, prefix string, denied, masked map[string]struct{}) {
	for _, value := range values {
		switch typed := value.(type) {
		case map[string]interface{}:
			secureInterfaceMap(typed, prefix, denied, masked)
		case map[string]string:
			secureStringMap(typed, prefix, denied, masked)
		case []interface{}:
			secureInterfaceSlice(typed, prefix, denied, masked)
		}
	}
}

func newHTTPClient(skipTLSVerify bool) *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: skipTLSVerify}, //nolint:gosec
	}
	return &http.Client{Transport: transport}
}

func parseFlexibleTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if seconds > 1_000_000_000_000 {
			return time.UnixMilli(seconds)
		}
		return time.Unix(seconds, 0)
	}
	return time.Time{}
}

func emptyHistogram(start, end time.Time, step time.Duration) []HistogramBar {
	if step <= 0 {
		step = time.Minute
	}
	if end.Before(start) {
		start, end = end, start
	}
	alignedStart := start.Truncate(step)
	alignedEnd := end.Truncate(step)
	if alignedEnd.Before(end) {
		alignedEnd = alignedEnd.Add(step)
	}
	count := int(alignedEnd.Sub(alignedStart) / step)
	if count < 1 {
		count = 1
	}
	if count > histogramMaxBuckets {
		count = histogramMaxBuckets
	}
	bars := make([]HistogramBar, 0, count)
	for index := 0; index < count; index++ {
		bars = append(bars, HistogramBar{Time: alignedStart.Add(time.Duration(index) * step).Format(time.RFC3339)})
	}
	return bars
}

func addCountToHistogram(bars []HistogramBar, start time.Time, step time.Duration, timestamp time.Time, count int) {
	if len(bars) == 0 || step <= 0 || timestamp.IsZero() || count <= 0 {
		return
	}
	alignedStart := parseFlexibleTime(bars[0].Time)
	if alignedStart.IsZero() {
		alignedStart = start.Truncate(step)
	}
	index := int(timestamp.Sub(alignedStart) / step)
	if index >= 0 && index < len(bars) {
		bars[index].Count += count
	}
}

func summarizeFields(items []LogItem) []FieldSummary {
	type accumulator struct {
		count  int
		sample string
		typ    string
	}
	fields := map[string]*accumulator{}
	for _, item := range items {
		for key, value := range item.Fields {
			if _, exists := fields[key]; !exists {
				fields[key] = &accumulator{sample: trimLongString(asString(value), 120), typ: guessType(value)}
			}
			fields[key].count++
		}
		for key, value := range item.Labels {
			name := "labels." + key
			if _, exists := fields[name]; !exists {
				fields[name] = &accumulator{sample: trimLongString(value, 120), typ: "keyword"}
			}
			fields[name].count++
		}
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]FieldSummary, 0, len(names))
	for _, name := range names {
		item := fields[name]
		result = append(result, FieldSummary{Name: name, Type: item.typ, Count: item.count, Sample: item.sample})
	}
	return result
}

func guessType(value interface{}) string {
	switch value.(type) {
	case string:
		return "keyword"
	case float64, float32, int, int64, int32, uint, uint64:
		return "number"
	case bool:
		return "boolean"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	default:
		return "unknown"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func asString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func trimLongString(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	if maxLength <= 0 || len(value) <= maxLength {
		return value
	}
	return value[:maxLength] + "..."
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
