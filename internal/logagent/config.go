package logagent

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	defaultScanIntervalSeconds = 1
	defaultBatchSize           = 500
	defaultFlushInterval       = 2
	defaultMaxWALBytes         = int64(2 * 1024 * 1024 * 1024)
	defaultMaxLineBytes        = 256 * 1024
	maxLineBytes               = 1024 * 1024
)

type Config struct {
	Enabled              bool           `json:"enabled"`
	GatewayURL           string         `json:"gatewayUrl"`
	GatewayToken         string         `json:"gatewayToken"`
	StateDir             string         `json:"stateDir"`
	ScanIntervalSeconds  int            `json:"scanIntervalSeconds"`
	BatchSize            int            `json:"batchSize"`
	FlushIntervalSeconds int            `json:"flushIntervalSeconds"`
	MaxWALBytes          int64          `json:"maxWalBytes"`
	Sources              []SourceConfig `json:"sources"`
}

type SourceConfig struct {
	ID            string                  `json:"id"`
	PolicyID      uint64                  `json:"policyId"`
	PolicyVersion uint64                  `json:"policyVersion"`
	Format        string                  `json:"format"`
	Paths         []string                `json:"paths"`
	ExcludePaths  []string                `json:"excludePaths"`
	ReadFrom      string                  `json:"readFrom"`
	Encoding      string                  `json:"encoding"`
	Environment   string                  `json:"environment"`
	Service       string                  `json:"service"`
	Stream        string                  `json:"stream"`
	MaxLineBytes  int                     `json:"maxLineBytes"`
	Parser        ParserConfig            `json:"parser"`
	Multiline     MultilineConfig         `json:"multiline"`
	Redaction     RedactionConfig         `json:"redaction"`
	Retention     RetentionConfig         `json:"retention"`
	Kubernetes    *KubernetesSourceConfig `json:"kubernetes,omitempty"`
}

type RetentionConfig struct {
	DefaultDays int            `json:"defaultDays"`
	LevelDays   map[string]int `json:"levelDays,omitempty"`
}

type KubernetesSourceConfig struct {
	ClusterID            uint64               `json:"clusterId"`
	ClusterName          string               `json:"clusterName"`
	Selectors            []KubernetesSelector `json:"selectors"`
	ExcludeNamespaces    []string             `json:"excludeNamespaces"`
	LabelAllowlist       []string             `json:"labelAllowlist"`
	AnnotationAllowlist  []string             `json:"annotationAllowlist"`
	ServiceLabelKeys     []string             `json:"serviceLabelKeys"`
	EnvironmentLabelKeys []string             `json:"environmentLabelKeys"`
}

type KubernetesSelector struct {
	Namespace        string   `json:"namespace"`
	WorkloadKind     string   `json:"workloadKind"`
	WorkloadName     string   `json:"workloadName"`
	LabelSelector    string   `json:"labelSelector"`
	ContainerInclude []string `json:"containerInclude"`
	ContainerExclude []string `json:"containerExclude"`
}

type ParserConfig struct {
	Type            string `json:"type"`
	Pattern         string `json:"pattern"`
	MessageField    string `json:"messageField"`
	TimestampField  string `json:"timestampField"`
	LevelField      string `json:"levelField"`
	TimestampLayout string `json:"timestampLayout"`
}

type MultilineConfig struct {
	Enabled      bool   `json:"enabled"`
	Preset       string `json:"preset"`
	StartPattern string `json:"startPattern"`
	MaxLines     int    `json:"maxLines"`
	MaxBytes     int    `json:"maxBytes"`
	FlushSeconds int    `json:"flushSeconds"`
}

func (config *Config) Normalize() {
	config.GatewayURL = strings.TrimRight(strings.TrimSpace(config.GatewayURL), "/")
	if strings.TrimSpace(config.StateDir) == "" {
		config.StateDir = "/var/lib/opshub-agent/logs"
	}
	if config.ScanIntervalSeconds <= 0 {
		config.ScanIntervalSeconds = defaultScanIntervalSeconds
	}
	if config.BatchSize <= 0 || config.BatchSize > 2000 {
		config.BatchSize = defaultBatchSize
	}
	if config.FlushIntervalSeconds <= 0 {
		config.FlushIntervalSeconds = defaultFlushInterval
	}
	if config.MaxWALBytes <= 0 {
		config.MaxWALBytes = defaultMaxWALBytes
	}
	for index := range config.Sources {
		config.Sources[index].normalize()
	}
}

func (config Config) Validate() error {
	if !config.Enabled {
		return nil
	}
	if config.GatewayURL == "" {
		return fmt.Errorf("日志采集缺少 gatewayUrl")
	}
	if config.GatewayToken == "" {
		return fmt.Errorf("日志采集缺少 gatewayToken")
	}
	if len(config.Sources) == 0 {
		return fmt.Errorf("日志采集至少需要一个文件源")
	}
	seen := make(map[string]struct{}, len(config.Sources))
	for _, source := range config.Sources {
		if err := source.Validate(); err != nil {
			return err
		}
		if _, exists := seen[source.ID]; exists {
			return fmt.Errorf("日志源 ID 重复: %s", source.ID)
		}
		seen[source.ID] = struct{}{}
	}
	return nil
}

func (config Config) ScanInterval() time.Duration {
	return time.Duration(config.ScanIntervalSeconds) * time.Second
}

func (config Config) FlushInterval() time.Duration {
	return time.Duration(config.FlushIntervalSeconds) * time.Second
}

func (source *SourceConfig) normalize() {
	source.ID = strings.TrimSpace(source.ID)
	source.Format = strings.ToLower(strings.TrimSpace(source.Format))
	if source.Format == "" {
		source.Format = "plain"
	}
	if source.Kubernetes != nil {
		source.Format = "cri"
		source.Kubernetes.normalize()
	}
	if source.ReadFrom != "beginning" && source.ReadFrom != "latest" {
		source.ReadFrom = "latest"
	}
	if strings.TrimSpace(source.Encoding) == "" {
		source.Encoding = "utf-8"
	}
	if source.MaxLineBytes <= 0 {
		source.MaxLineBytes = defaultMaxLineBytes
	}
	if source.MaxLineBytes > maxLineBytes {
		source.MaxLineBytes = maxLineBytes
	}
	source.Parser.Type = strings.ToLower(strings.TrimSpace(source.Parser.Type))
	if source.Parser.Type == "" {
		source.Parser.Type = "raw"
	}
	source.Multiline.normalize()
	source.Redaction.normalize()
	source.Retention.normalize()
}

func (source SourceConfig) Validate() error {
	if source.ID == "" {
		return fmt.Errorf("日志源 ID 不能为空")
	}
	if len(source.Paths) == 0 {
		return fmt.Errorf("日志源 %s 至少需要一个文件路径", source.ID)
	}
	for _, pattern := range append(append([]string{}, source.Paths...), source.ExcludePaths...) {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("日志源 %s 包含空路径", source.ID)
		}
		if _, err := filepath.Match(pattern, pattern); err != nil {
			return fmt.Errorf("日志源 %s 路径通配符无效: %w", source.ID, err)
		}
	}
	if source.MaxLineBytes <= 0 || source.MaxLineBytes > maxLineBytes {
		return fmt.Errorf("日志源 %s 单行上限必须在 1 到 1048576 字节之间", source.ID)
	}
	if source.Format != "plain" && source.Format != "cri" {
		return fmt.Errorf("日志源 %s 不支持格式 %s", source.ID, source.Format)
	}
	if source.Kubernetes != nil {
		if source.Kubernetes.ClusterID == 0 {
			return fmt.Errorf("日志源 %s 缺少 Kubernetes 集群 ID", source.ID)
		}
		if len(source.Kubernetes.Selectors) == 0 {
			return fmt.Errorf("日志源 %s 至少需要一个 Kubernetes 选择器", source.ID)
		}
	}
	switch source.Parser.Type {
	case "raw", "json":
	case "regex":
		if strings.TrimSpace(source.Parser.Pattern) == "" {
			return fmt.Errorf("日志源 %s 的 regex 解析器缺少 pattern", source.ID)
		}
		if _, err := regexp.Compile(source.Parser.Pattern); err != nil {
			return fmt.Errorf("日志源 %s 的解析正则无效: %w", source.ID, err)
		}
	default:
		return fmt.Errorf("日志源 %s 不支持解析器 %s", source.ID, source.Parser.Type)
	}
	if err := source.Multiline.Validate(source.ID); err != nil {
		return err
	}
	return source.Redaction.Validate(source.ID)
}

func (config *RetentionConfig) normalize() {
	if config.DefaultDays <= 0 {
		config.DefaultDays = 30
	}
	if config.DefaultDays > 3650 {
		config.DefaultDays = 3650
	}
	normalized := make(map[string]int, len(config.LevelDays))
	for level, days := range config.LevelDays {
		level = strings.ToUpper(strings.TrimSpace(level))
		if level == "WARNING" {
			level = "WARN"
		}
		if days <= 0 || level == "" {
			continue
		}
		if days > 3650 {
			days = 3650
		}
		normalized[level] = days
	}
	config.LevelDays = normalized
}

func (config RetentionConfig) DaysForLevel(level string) int {
	level = strings.ToUpper(strings.TrimSpace(level))
	if level == "WARNING" {
		level = "WARN"
	}
	if days := config.LevelDays[level]; days > 0 {
		return days
	}
	if config.DefaultDays > 0 {
		return config.DefaultDays
	}
	return 30
}

func (config *KubernetesSourceConfig) normalize() {
	config.ClusterName = strings.TrimSpace(config.ClusterName)
	config.ExcludeNamespaces = normalizeStringList(config.ExcludeNamespaces)
	config.LabelAllowlist = normalizeStringList(config.LabelAllowlist)
	config.AnnotationAllowlist = normalizeStringList(config.AnnotationAllowlist)
	config.ServiceLabelKeys = normalizeStringList(config.ServiceLabelKeys)
	config.EnvironmentLabelKeys = normalizeStringList(config.EnvironmentLabelKeys)
	if len(config.ServiceLabelKeys) == 0 {
		config.ServiceLabelKeys = []string{"app.kubernetes.io/name", "app", "k8s-app"}
	}
	if len(config.EnvironmentLabelKeys) == 0 {
		config.EnvironmentLabelKeys = []string{"app.kubernetes.io/environment", "environment", "env"}
	}
	for index := range config.Selectors {
		selector := &config.Selectors[index]
		selector.Namespace = strings.TrimSpace(selector.Namespace)
		selector.WorkloadKind = strings.TrimSpace(selector.WorkloadKind)
		selector.WorkloadName = strings.TrimSpace(selector.WorkloadName)
		selector.LabelSelector = strings.TrimSpace(selector.LabelSelector)
		selector.ContainerInclude = normalizeStringList(selector.ContainerInclude)
		selector.ContainerExclude = normalizeStringList(selector.ContainerExclude)
	}
}

func normalizeStringList(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (config *MultilineConfig) normalize() {
	config.Preset = strings.ToLower(strings.TrimSpace(config.Preset))
	if config.MaxLines <= 0 {
		config.MaxLines = 500
	}
	if config.MaxBytes <= 0 || config.MaxBytes > maxLineBytes {
		config.MaxBytes = maxLineBytes
	}
	if config.FlushSeconds <= 0 {
		config.FlushSeconds = 2
	}
}

func (config MultilineConfig) Validate(sourceID string) error {
	if !config.Enabled {
		return nil
	}
	if config.StartPattern != "" {
		if _, err := regexp.Compile(config.StartPattern); err != nil {
			return fmt.Errorf("日志源 %s 的多行起始正则无效: %w", sourceID, err)
		}
	}
	switch config.Preset {
	case "", "java", "go", "python", "custom":
	default:
		return fmt.Errorf("日志源 %s 不支持多行模板 %s", sourceID, config.Preset)
	}
	if config.Preset == "custom" && config.StartPattern == "" {
		return fmt.Errorf("日志源 %s 的自定义多行模板缺少 startPattern", sourceID)
	}
	return nil
}
