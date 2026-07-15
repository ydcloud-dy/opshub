package aiops

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	assetmodel "github.com/ydcloud-dy/opshub/internal/biz/asset"
)

const (
	opsHubAgentHostTopDefaultLimit = 100
	opsHubAgentHostTopMaxLimit     = 200
	opsHubAgentHostTopDefaultN     = 3
	opsHubAgentHostTopMaxN         = 10

	opsHubAgentHostSoftwareDefaultLimit = 100
	opsHubAgentHostSoftwareMaxLimit     = 200
	opsHubAgentHostSoftwareMaxEvidence  = 30
)

type opsHubAgentHostLiveRow struct {
	ID            uint       `gorm:"column:id"`
	Name          string     `gorm:"column:name"`
	Hostname      string     `gorm:"column:hostname"`
	GroupID       uint       `gorm:"column:group_id"`
	GroupName     string     `gorm:"column:group_name"`
	Type          string     `gorm:"column:type"`
	CloudProvider string     `gorm:"column:cloud_provider"`
	SSHUser       string     `gorm:"column:ssh_user"`
	IP            string     `gorm:"column:ip"`
	Port          int        `gorm:"column:port"`
	CredentialID  uint       `gorm:"column:credential_id"`
	Status        int        `gorm:"column:status"`
	AgentStatus   string     `gorm:"column:agent_status"`
	CPUUsage      float64    `gorm:"column:cpu_usage"`
	MemoryUsage   float64    `gorm:"column:memory_usage"`
	DiskUsage     float64    `gorm:"column:disk_usage"`
	LastSeen      *time.Time `gorm:"column:last_seen"`
}

type opsHubAgentHostProcessInfo struct {
	PID           int     `json:"pid"`
	PPID          int     `json:"ppid,omitempty"`
	User          string  `json:"user,omitempty"`
	Name          string  `json:"name,omitempty"`
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryPercent float64 `json:"memoryPercent"`
	RSSKB         int64   `json:"rssKB,omitempty"`
	RSS           string  `json:"rss,omitempty"`
	Command       string  `json:"command,omitempty"`
}

type opsHubAgentHostTopProcessItem struct {
	HostID             uint                         `json:"hostId"`
	Name               string                       `json:"name"`
	Hostname           string                       `json:"hostname,omitempty"`
	IP                 string                       `json:"ip"`
	GroupID            uint                         `json:"groupId,omitempty"`
	GroupName          string                       `json:"groupName,omitempty"`
	Type               string                       `json:"type,omitempty"`
	CloudProvider      string                       `json:"cloudProvider,omitempty"`
	Status             string                       `json:"status"`
	AgentStatus        string                       `json:"agentStatus,omitempty"`
	StoredCPUUsage     float64                      `json:"storedCPUUsage"`
	StoredMemoryUsage  float64                      `json:"storedMemoryUsage"`
	StoredDiskUsage    float64                      `json:"storedDiskUsage"`
	QuerySuccess       bool                         `json:"querySuccess"`
	QueryMethod        string                       `json:"queryMethod"`
	Error              string                       `json:"error,omitempty"`
	TopMemoryProcesses []opsHubAgentHostProcessInfo `json:"topMemoryProcesses,omitempty"`
	TopCPUProcesses    []opsHubAgentHostProcessInfo `json:"topCPUProcesses,omitempty"`
}

type opsHubAgentHostSoftwareEvidence struct {
	Type    string `json:"type"`
	Detail  string `json:"detail"`
	Running bool   `json:"running,omitempty"`
}

type opsHubAgentHostSoftwareProbeItem struct {
	HostID            uint                              `json:"hostId"`
	Name              string                            `json:"name"`
	Hostname          string                            `json:"hostname,omitempty"`
	IP                string                            `json:"ip"`
	GroupID           uint                              `json:"groupId,omitempty"`
	GroupName         string                            `json:"groupName,omitempty"`
	Type              string                            `json:"type,omitempty"`
	CloudProvider     string                            `json:"cloudProvider,omitempty"`
	Status            string                            `json:"status"`
	AgentStatus       string                            `json:"agentStatus,omitempty"`
	StoredCPUUsage    float64                           `json:"storedCPUUsage"`
	StoredMemoryUsage float64                           `json:"storedMemoryUsage"`
	StoredDiskUsage   float64                           `json:"storedDiskUsage"`
	Target            string                            `json:"target"`
	Deployed          bool                              `json:"deployed"`
	Running           bool                              `json:"running"`
	QuerySuccess      bool                              `json:"querySuccess"`
	QueryMethod       string                            `json:"queryMethod"`
	Error             string                            `json:"error,omitempty"`
	Evidence          []opsHubAgentHostSoftwareEvidence `json:"evidence,omitempty"`
}

func requiredOpsHubAgentLiveTool(question string, trace opsHubAgentTrace) string {
	if isHostSoftwareProbeQuestion(question) && !opsHubAgentToolAlreadyCalled(trace, opsHubAgentToolHostSoftwareProbe) {
		return opsHubAgentToolHostSoftwareProbe
	}
	if isHostTopProcessQuestion(question) && !opsHubAgentToolAlreadyCalled(trace, opsHubAgentToolHostTopProcesses) {
		return opsHubAgentToolHostTopProcesses
	}
	return ""
}

func enrichOpsHubAgentLiveToolParams(question, tool string, params map[string]any) {
	if params == nil {
		return
	}
	switch tool {
	case opsHubAgentToolHostTopProcesses:
		if _, ok := params["topN"]; !ok {
			params["topN"] = opsHubAgentHostTopDefaultN
		}
		if _, ok := params["metric"]; !ok {
			text := strings.ToLower(strings.TrimSpace(question))
			switch {
			case containsAnyText(text, []string{"内存", "memory", "mem", "rss"}) && !containsAnyText(text, []string{"cpu"}):
				params["metric"] = "memory"
			case containsAnyText(text, []string{"cpu"}) && !containsAnyText(text, []string{"内存", "memory", "mem", "rss"}):
				params["metric"] = "cpu"
			default:
				params["metric"] = "both"
			}
		}
		if _, ok := params["limit"]; !ok {
			params["limit"] = opsHubAgentHostTopDefaultLimit
		}
	case opsHubAgentToolHostSoftwareProbe:
		if firstStringParam(params, "target", "software", "service", "name") == "" {
			if target := inferHostSoftwareTarget(question); target != "" {
				params["target"] = target
			}
		}
		if _, ok := params["limit"]; !ok {
			params["limit"] = opsHubAgentHostSoftwareDefaultLimit
		}
	}
}

func isHostTopProcessQuestion(question string) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" {
		return false
	}
	processHit := containsAnyText(text, []string{
		"进程", "process", "top", "pid", "占用最高", "占用最多", "资源最多", "高占用",
	})
	resourceHit := containsAnyText(text, []string{
		"cpu", "内存", "memory", "mem", "rss", "资源", "占用",
	})
	hostHit := containsAnyText(text, []string{
		"主机", "服务器", "机器", "资产", "云主机", "host", "server", "每台", "所有",
	})
	return processHit && resourceHit && hostHit
}

func isHostSoftwareProbeQuestion(question string) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" || inferHostSoftwareTarget(text) == "" {
		return false
	}
	hostHit := containsAnyText(text, []string{
		"主机", "服务器", "机器", "资产", "云主机", "host", "server", "哪台", "哪些", "所有",
	})
	probeHit := containsAnyText(text, []string{
		"部署", "安装", "装了", "运行", "启动", "在哪", "哪里", "哪些", "哪台", "有没有", "是否有",
		"服务", "软件", "进程", "installed", "running", "deployed", "service",
	})
	if !hostHit || !probeHit {
		return false
	}
	if containsAnyText(text, []string{"deployment", "工作负载", "pod", "ingress", "service yaml", "service配置"}) &&
		!containsAnyText(text, []string{"主机", "服务器", "机器", "资产", "云主机"}) {
		return false
	}
	return true
}

func inferHostSoftwareTarget(question string) string {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" {
		return ""
	}
	if target := normalizeHostSoftwareTarget(firstStringParam(map[string]any{"question": question}, "target", "software", "service", "name")); target != "" {
		return target
	}
	for _, name := range knownHostSoftwareNames() {
		re := regexp.MustCompile(`(^|[^a-z0-9._+-])` + regexp.QuoteMeta(name) + `([^a-z0-9._+-]|$)`)
		if re.FindStringIndex(text) != nil {
			return name
		}
	}
	patterns := []string{
		`(?:部署了?|安装了?|运行了?|启动了?|装了|有)\s*([a-zA-Z][a-zA-Z0-9._+-]{1,63})`,
		`([a-zA-Z][a-zA-Z0-9._+-]{1,63})\s*(?:部署|安装|运行|启动|服务|进程|在哪|在不在)`,
		`(?:查一下|看下|看看|找出|查询)\s*(?:哪些|哪台|所有|每台)?\s*(?:主机|服务器|机器|资产)?\s*(?:中|上|里)?\s*(?:的)?\s*([a-zA-Z][a-zA-Z0-9._+-]{1,63})`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(question); len(match) > 1 {
			if target := normalizeHostSoftwareTarget(match[1]); target != "" {
				return target
			}
		}
	}
	return ""
}

func normalizeHostSoftwareTarget(value string) string {
	target := strings.ToLower(strings.Trim(strings.TrimSpace(value), " \t\r\n，,。.!！?？:：;；'\"`“”‘’()（）[]【】{}"))
	if target == "" {
		return ""
	}
	if strings.HasSuffix(target, ".service") {
		target = strings.TrimSuffix(target, ".service")
	}
	if len(target) < 2 || len(target) > 64 {
		return ""
	}
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9._+-]*$`).MatchString(target) {
		return ""
	}
	excluded := map[string]bool{
		"opshub": true, "host": true, "server": true, "service": true, "systemd": true,
		"linux": true, "ssh": true, "agent": true, "cpu": true, "memory": true,
	}
	if excluded[target] {
		return ""
	}
	return target
}

func knownHostSoftwareNames() []string {
	return []string{
		"nginx", "openresty", "apache", "httpd", "tomcat", "java", "jdk",
		"redis", "mysql", "mysqld", "mariadb", "postgres", "postgresql", "mongodb", "mongo",
		"docker", "containerd", "podman", "kubelet", "kube-proxy",
		"jenkins", "gitlab", "harbor", "nexus", "sonarqube",
		"prometheus", "grafana", "alertmanager", "node_exporter", "victoriametrics", "loki",
		"elasticsearch", "logstash", "kibana", "kafka", "zookeeper", "etcd", "consul",
		"rabbitmq", "rocketmq", "minio", "clickhouse", "tidb", "canal",
	}
}

func (s *Service) collectOpsHubAgentHostTopProcesses(ctx context.Context, userID uint, question string, params map[string]any) (map[string]any, []string) {
	_ = userID
	limit := boundedIntParam(params, opsHubAgentHostTopDefaultLimit, 1, opsHubAgentHostTopMaxLimit, "limit", "pageSize")
	topN := boundedIntParam(params, opsHubAgentHostTopDefaultN, 1, opsHubAgentHostTopMaxN, "topN", "top", "count")
	metric := strings.ToLower(defaultString(firstStringParam(params, "metric", "sortBy"), "both"))
	if metric != "memory" && metric != "cpu" {
		metric = "both"
	}

	query := s.db.WithContext(ctx).
		Table("hosts AS h").
		Joins("LEFT JOIN asset_group AS g ON g.id = h.group_id AND g.deleted_at IS NULL").
		Where("h.deleted_at IS NULL")

	if hostID := uintParam(params, "hostId", "hostID", "id"); hostID > 0 {
		query = query.Where("h.id = ?", hostID)
	}
	if ip := firstStringParam(params, "ip", "hostIP", "hostIp"); ip != "" {
		query = query.Where("h.ip = ?", ip)
	}
	if hostName := firstStringParam(params, "hostName", "hostname", "name"); hostName != "" {
		like := "%" + hostName + "%"
		query = query.Where("(h.name LIKE ? OR h.hostname LIKE ?)", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return map[string]any{
			"topProcessSummary": map[string]any{"totalHosts": 0, "queriedHosts": 0, "failedHosts": 0},
			"hosts":             []opsHubAgentHostTopProcessItem{},
		}, []string{"主机列表查询失败: " + err.Error()}
	}

	var rows []opsHubAgentHostLiveRow
	if err := query.
		Select([]string{
			"h.id",
			"h.name",
			"h.hostname",
			"h.group_id",
			"g.name AS group_name",
			"h.type",
			"h.cloud_provider",
			"h.ssh_user",
			"h.ip",
			"h.port",
			"h.credential_id",
			"h.status",
			"h.agent_status",
			"h.cpu_usage",
			"h.memory_usage",
			"h.disk_usage",
			"h.last_seen",
		}).
		Order("h.id DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return map[string]any{
			"topProcessSummary": map[string]any{"totalHosts": total, "queriedHosts": 0, "failedHosts": 0},
			"hosts":             []opsHubAgentHostTopProcessItem{},
		}, []string{"主机列表读取失败: " + err.Error()}
	}

	credentials, credentialErrors, credentialWarnings := s.loadDecryptedHostCredentials(ctx, rows)
	items := make([]opsHubAgentHostTopProcessItem, len(rows))
	warnings := append([]string{}, credentialWarnings...)

	liveCtx, cancel := context.WithTimeout(ctx, hostTopProcessQueryTimeout(len(rows)))
	defer cancel()

	const concurrency = 6
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i := range rows {
		i := i
		row := rows[i]
		items[i] = newHostTopProcessItem(row)
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-liveCtx.Done():
				items[i].Error = "现场查询已超时或被取消"
				return
			}
			credential, ok := credentials[row.CredentialID]
			if row.CredentialID == 0 {
				items[i].Error = "主机未绑定 SSH 凭证，无法现场查询进程"
				return
			}
			if errText := credentialErrors[row.CredentialID]; errText != "" {
				items[i].Error = "SSH 凭证不可用: " + errText
				return
			}
			if !ok {
				items[i].Error = "未找到主机绑定的 SSH 凭证"
				return
			}
			s.querySingleHostTopProcesses(liveCtx, row, credential, topN, metric, &items[i])
		}()
	}
	wg.Wait()

	summary := summarizeHostTopProcessItems(items, total, limit, topN, metric)
	for _, item := range items {
		if !item.QuerySuccess && item.Error != "" && len(warnings) < 12 {
			warnings = append(warnings, fmt.Sprintf("%s(%s): %s", item.Name, item.IP, item.Error))
		}
	}
	if total > int64(len(rows)) {
		warnings = append(warnings, fmt.Sprintf("主机数量 %d 台，已按安全上限查询前 %d 台，可在问题中指定主机名/IP 或提高 limit 参数", total, len(rows)))
	}

	return map[string]any{
		"topProcessSummary": summary,
		"hostTopProcesses":  items,
		"hosts":             items,
		"overallTopMemory":  overallTopProcesses(items, "memory", 10),
		"overallTopCPU":     overallTopProcesses(items, "cpu", 10),
		"queryPolicy": map[string]any{
			"mode":        "read_only_live_ssh",
			"description": "使用后端预置白名单 ps 采集命令，仅查询进程、CPU、内存和 RSS，不执行变更操作",
			"topN":        topN,
			"metric":      metric,
			"limit":       limit,
		},
	}, uniqueStrings(warnings)
}

func (s *Service) collectOpsHubAgentHostSoftwareProbe(ctx context.Context, userID uint, question string, params map[string]any) (map[string]any, []string) {
	_ = userID
	target := normalizeHostSoftwareTarget(firstStringParam(params, "target", "software", "service", "name"))
	if target == "" {
		target = inferHostSoftwareTarget(question)
	}
	limit := boundedIntParam(params, opsHubAgentHostSoftwareDefaultLimit, 1, opsHubAgentHostSoftwareMaxLimit, "limit", "pageSize")

	if target == "" {
		return map[string]any{
			"softwareProbeSummary": map[string]any{"target": "", "totalHosts": 0, "queriedHosts": 0, "failedHosts": 0, "deployedHosts": 0, "runningHosts": 0},
			"softwareProbe":        []opsHubAgentHostSoftwareProbeItem{},
			"hosts":                []opsHubAgentHostSoftwareProbeItem{},
			"queryPolicy": map[string]any{
				"mode":        "read_only_live_ssh",
				"description": "未能从问题中识别要探测的软件/服务名，无法执行主机现场探测",
			},
		}, []string{"未能从问题中识别要探测的软件/服务名，例如 nginx、redis、mysql"}
	}

	query := s.db.WithContext(ctx).
		Table("hosts AS h").
		Joins("LEFT JOIN asset_group AS g ON g.id = h.group_id AND g.deleted_at IS NULL").
		Where("h.deleted_at IS NULL")

	if hostID := uintParam(params, "hostId", "hostID", "id"); hostID > 0 {
		query = query.Where("h.id = ?", hostID)
	}
	if ip := firstStringParam(params, "ip", "hostIP", "hostIp"); ip != "" {
		query = query.Where("h.ip = ?", ip)
	}
	if hostName := firstStringParam(params, "hostName", "hostname", "name"); hostName != "" {
		like := "%" + hostName + "%"
		query = query.Where("(h.name LIKE ? OR h.hostname LIKE ?)", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return map[string]any{
			"softwareProbeSummary": map[string]any{"target": target, "totalHosts": 0, "queriedHosts": 0, "failedHosts": 0, "deployedHosts": 0, "runningHosts": 0},
			"softwareProbe":        []opsHubAgentHostSoftwareProbeItem{},
			"hosts":                []opsHubAgentHostSoftwareProbeItem{},
		}, []string{"主机列表查询失败: " + err.Error()}
	}

	var rows []opsHubAgentHostLiveRow
	if err := query.
		Select([]string{
			"h.id",
			"h.name",
			"h.hostname",
			"h.group_id",
			"g.name AS group_name",
			"h.type",
			"h.cloud_provider",
			"h.ssh_user",
			"h.ip",
			"h.port",
			"h.credential_id",
			"h.status",
			"h.agent_status",
			"h.cpu_usage",
			"h.memory_usage",
			"h.disk_usage",
			"h.last_seen",
		}).
		Order("h.id DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return map[string]any{
			"softwareProbeSummary": map[string]any{"target": target, "totalHosts": total, "queriedHosts": 0, "failedHosts": 0, "deployedHosts": 0, "runningHosts": 0},
			"softwareProbe":        []opsHubAgentHostSoftwareProbeItem{},
			"hosts":                []opsHubAgentHostSoftwareProbeItem{},
		}, []string{"主机列表读取失败: " + err.Error()}
	}

	credentials, credentialErrors, credentialWarnings := s.loadDecryptedHostCredentials(ctx, rows)
	items := make([]opsHubAgentHostSoftwareProbeItem, len(rows))
	warnings := append([]string{}, credentialWarnings...)

	liveCtx, cancel := context.WithTimeout(ctx, hostSoftwareProbeQueryTimeout(len(rows)))
	defer cancel()

	const concurrency = 6
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i := range rows {
		i := i
		row := rows[i]
		items[i] = newHostSoftwareProbeItem(row, target)
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-liveCtx.Done():
				items[i].Error = "现场探测已超时或被取消"
				return
			}
			credential, ok := credentials[row.CredentialID]
			if row.CredentialID == 0 {
				items[i].Error = "主机未绑定 SSH 凭证，无法现场探测软件/服务"
				return
			}
			if errText := credentialErrors[row.CredentialID]; errText != "" {
				items[i].Error = "SSH 凭证不可用: " + errText
				return
			}
			if !ok {
				items[i].Error = "未找到主机绑定的 SSH 凭证"
				return
			}
			s.querySingleHostSoftwareProbe(liveCtx, row, credential, target, &items[i])
		}()
	}
	wg.Wait()

	summary := summarizeHostSoftwareProbeItems(items, total, limit, target)
	for _, item := range items {
		if !item.QuerySuccess && item.Error != "" && len(warnings) < 12 {
			warnings = append(warnings, fmt.Sprintf("%s(%s): %s", item.Name, item.IP, item.Error))
		}
	}
	if total > int64(len(rows)) {
		warnings = append(warnings, fmt.Sprintf("主机数量 %d 台，已按安全上限探测前 %d 台，可在问题中指定主机名/IP 或提高 limit 参数", total, len(rows)))
	}

	return map[string]any{
		"softwareProbeSummary": summary,
		"softwareProbe":        items,
		"hosts":                items,
		"deployedHosts":        deployedHostSoftwareProbeItems(items),
		"runningHosts":         runningHostSoftwareProbeItems(items),
		"queryPolicy": map[string]any{
			"mode":        "read_only_live_ssh",
			"description": "使用后端预置白名单只读 SSH 探测命令，仅读取进程、systemd/service、二进制路径、rpm/dpkg 包、配置目录和容器信息，不执行变更操作",
			"target":      target,
			"limit":       limit,
		},
	}, uniqueStrings(warnings)
}

func (s *Service) loadDecryptedHostCredentials(ctx context.Context, rows []opsHubAgentHostLiveRow) (map[uint]assetmodel.Credential, map[uint]string, []string) {
	ids := make([]uint, 0)
	seen := map[uint]bool{}
	for _, row := range rows {
		if row.CredentialID == 0 || seen[row.CredentialID] {
			continue
		}
		seen[row.CredentialID] = true
		ids = append(ids, row.CredentialID)
	}
	if len(ids) == 0 {
		return map[uint]assetmodel.Credential{}, map[uint]string{}, nil
	}

	var credentials []assetmodel.Credential
	if err := s.db.WithContext(ctx).Table("credentials").Where("id IN ? AND deleted_at IS NULL", ids).Find(&credentials).Error; err != nil {
		return map[uint]assetmodel.Credential{}, map[uint]string{}, []string{"SSH 凭证查询失败: " + err.Error()}
	}

	byID := make(map[uint]assetmodel.Credential, len(credentials))
	errorsByID := make(map[uint]string)
	for i := range credentials {
		credential := credentials[i]
		if err := decryptCredentialForAI(&credential); err != nil {
			errorsByID[credential.ID] = err.Error()
			continue
		}
		byID[credential.ID] = credential
	}
	return byID, errorsByID, nil
}

func (s *Service) querySingleHostTopProcesses(ctx context.Context, row opsHubAgentHostLiveRow, credential assetmodel.Credential, topN int, metric string, item *opsHubAgentHostTopProcessItem) {
	if ctx.Err() != nil {
		item.Error = "现场查询已被取消"
		return
	}
	if strings.TrimSpace(row.IP) == "" {
		item.Error = "主机 IP 为空，无法建立 SSH 连接"
		return
	}
	host := assetmodel.Host{
		Name:    row.Name,
		SSHUser: row.SSHUser,
		IP:      row.IP,
		Port:    defaultSSHPort(row.Port),
	}
	host.ID = row.ID

	client, err := newAIHostSSHClient(host, credential)
	if err != nil {
		item.Error = err.Error()
		return
	}
	defer client.Close()

	output, err := client.ExecuteWithTimeout(buildHostTopProcessCommand(topN, metric), 18*time.Second)
	if err != nil {
		item.Error = "读取进程列表失败: " + err.Error()
		return
	}
	memProcesses, cpuProcesses := parseHostTopProcessOutput(output, topN)
	if metric != "cpu" {
		item.TopMemoryProcesses = memProcesses
	}
	if metric != "memory" {
		item.TopCPUProcesses = cpuProcesses
	}
	if len(item.TopMemoryProcesses) == 0 && len(item.TopCPUProcesses) == 0 {
		item.Error = "ps 未返回可解析的进程数据，可能是目标系统 ps 参数不兼容或权限受限"
		return
	}
	item.QuerySuccess = true
}

func (s *Service) querySingleHostSoftwareProbe(ctx context.Context, row opsHubAgentHostLiveRow, credential assetmodel.Credential, target string, item *opsHubAgentHostSoftwareProbeItem) {
	if ctx.Err() != nil {
		item.Error = "现场探测已被取消"
		return
	}
	if strings.TrimSpace(row.IP) == "" {
		item.Error = "主机 IP 为空，无法建立 SSH 连接"
		return
	}
	host := assetmodel.Host{
		Name:    row.Name,
		SSHUser: row.SSHUser,
		IP:      row.IP,
		Port:    defaultSSHPort(row.Port),
	}
	host.ID = row.ID

	client, err := newAIHostSSHClient(host, credential)
	if err != nil {
		item.Error = err.Error()
		return
	}
	defer client.Close()

	output, err := client.ExecuteWithTimeout(buildHostSoftwareProbeCommand(target), 22*time.Second)
	if err != nil {
		item.Error = "读取软件/服务部署线索失败: " + err.Error()
		return
	}
	item.Evidence = parseHostSoftwareProbeOutput(output)
	if len(item.Evidence) > opsHubAgentHostSoftwareMaxEvidence {
		item.Evidence = item.Evidence[:opsHubAgentHostSoftwareMaxEvidence]
	}
	for _, evidence := range item.Evidence {
		if strings.TrimSpace(evidence.Detail) == "" {
			continue
		}
		item.Deployed = true
		if evidence.Running {
			item.Running = true
		}
	}
	item.QuerySuccess = true
}

func newHostTopProcessItem(row opsHubAgentHostLiveRow) opsHubAgentHostTopProcessItem {
	agentStatus := strings.TrimSpace(row.AgentStatus)
	if agentStatus == "" {
		agentStatus = "none"
	}
	hostType := strings.TrimSpace(row.Type)
	if hostType == "" {
		hostType = "self"
	}
	return opsHubAgentHostTopProcessItem{
		HostID:            row.ID,
		Name:              row.Name,
		Hostname:          row.Hostname,
		IP:                row.IP,
		GroupID:           row.GroupID,
		GroupName:         row.GroupName,
		Type:              hostType,
		CloudProvider:     row.CloudProvider,
		Status:            hostStatusText(row.Status),
		AgentStatus:       agentStatus,
		StoredCPUUsage:    roundFloat(row.CPUUsage),
		StoredMemoryUsage: roundFloat(row.MemoryUsage),
		StoredDiskUsage:   roundFloat(row.DiskUsage),
		QueryMethod:       "ssh.ps.readonly",
	}
}

func newHostSoftwareProbeItem(row opsHubAgentHostLiveRow, target string) opsHubAgentHostSoftwareProbeItem {
	agentStatus := strings.TrimSpace(row.AgentStatus)
	if agentStatus == "" {
		agentStatus = "none"
	}
	hostType := strings.TrimSpace(row.Type)
	if hostType == "" {
		hostType = "self"
	}
	return opsHubAgentHostSoftwareProbeItem{
		HostID:            row.ID,
		Name:              row.Name,
		Hostname:          row.Hostname,
		IP:                row.IP,
		GroupID:           row.GroupID,
		GroupName:         row.GroupName,
		Type:              hostType,
		CloudProvider:     row.CloudProvider,
		Status:            hostStatusText(row.Status),
		AgentStatus:       agentStatus,
		StoredCPUUsage:    roundFloat(row.CPUUsage),
		StoredMemoryUsage: roundFloat(row.MemoryUsage),
		StoredDiskUsage:   roundFloat(row.DiskUsage),
		Target:            target,
		QueryMethod:       "ssh.software.probe.readonly",
	}
}

func defaultSSHPort(port int) int {
	if port <= 0 {
		return 22
	}
	return port
}

func buildHostTopProcessCommand(topN int, metric string) string {
	if topN <= 0 {
		topN = opsHubAgentHostTopDefaultN
	}
	if topN > opsHubAgentHostTopMaxN {
		topN = opsHubAgentHostTopMaxN
	}
	sections := make([]string, 0, 2)
	if metric != "cpu" {
		sections = append(sections, fmt.Sprintf("printf '__OPSHUB_MEM__\\n'; LC_ALL=C ps -eo pid=,ppid=,user=,comm=,%%mem=,%%cpu=,rss=,args= --sort=-%%mem 2>/dev/null | head -n %d", topN))
	}
	if metric != "memory" {
		sections = append(sections, fmt.Sprintf("printf '__OPSHUB_CPU__\\n'; LC_ALL=C ps -eo pid=,ppid=,user=,comm=,%%cpu=,%%mem=,rss=,args= --sort=-%%cpu 2>/dev/null | head -n %d", topN))
	}
	return strings.Join(sections, "; ")
}

func buildHostSoftwareProbeCommand(target string) string {
	target = normalizeHostSoftwareTarget(target)
	if target == "" {
		target = "unknown"
	}
	quotedTarget := shellQuote(target)
	return fmt.Sprintf(`T=%s
emit() {
  section="$1"
  line="$2"
  [ -n "$line" ] && printf '__OPSHUB_SOFTWARE__	%%s	%%s\n' "$section" "$line"
}
run_grep() {
  section="$1"
  shift
  "$@" 2>/dev/null | grep -i -- "$T" | grep -Ev 'grep -i|pgrep -af|__OPSHUB_SOFTWARE__|sh -c T=' | head -n 12 | while IFS= read -r line; do emit "$section" "$line"; done
}
check_path() {
  path="$1"
  [ -e "$path" ] && emit config "$path"
}
if command -v pgrep >/dev/null 2>&1; then
  pgrep -af -- "$T" 2>/dev/null | grep -Ev 'pgrep -af|__OPSHUB_SOFTWARE__|sh -c T=' | head -n 12 | while IFS= read -r line; do emit process "$line"; done
fi
if command -v ps >/dev/null 2>&1; then
  run_grep process ps -eo pid=,comm=,args=
fi
if command -v systemctl >/dev/null 2>&1; then
  run_grep systemd_unit systemctl list-units --type=service --all --no-pager --no-legend
  run_grep systemd_unit_file systemctl list-unit-files --type=service --no-pager --no-legend
  active=$(systemctl is-active "$T.service" 2>/dev/null || true)
  case "$active" in active|activating|reloading|inactive|failed) emit systemd_active "$T.service $active";; esac
  enabled=$(systemctl is-enabled "$T.service" 2>/dev/null || true)
  case "$enabled" in enabled|disabled|static|masked|indirect|generated) emit systemd_enabled "$T.service $enabled";; esac
fi
if command -v service >/dev/null 2>&1; then
  service --status-all 2>/dev/null | grep -i -- "$T" | head -n 12 | while IFS= read -r line; do emit sysv_service "$line"; done
fi
bin_path=$(command -v "$T" 2>/dev/null || true)
[ -n "$bin_path" ] && emit binary "$bin_path"
if command -v whereis >/dev/null 2>&1; then
  whereis_line=$(whereis "$T" 2>/dev/null | grep -v ":$" || true)
  [ -n "$whereis_line" ] && emit binary "$whereis_line"
fi
if command -v rpm >/dev/null 2>&1; then
  run_grep package rpm -qa
fi
if command -v dpkg-query >/dev/null 2>&1; then
  run_grep package dpkg-query -W -f='${Package}\t${Version}\n'
fi
check_path "/etc/$T"
check_path "/etc/$T/$T.conf"
check_path "/etc/$T/conf.d"
check_path "/usr/local/$T"
check_path "/usr/local/$T/conf"
check_path "/opt/$T"
check_path "/var/log/$T"
if [ "$T" = "nginx" ]; then
  check_path "/etc/nginx/nginx.conf"
  check_path "/usr/sbin/nginx"
  check_path "/usr/local/nginx/sbin/nginx"
  check_path "/var/lib/nginx"
fi
if command -v docker >/dev/null 2>&1; then
  run_grep container docker ps -a --format '{{.ID}}	{{.Names}}	{{.Image}}	{{.Status}}'
fi
if command -v podman >/dev/null 2>&1; then
  run_grep container podman ps -a --format '{{.ID}}	{{.Names}}	{{.Image}}	{{.Status}}'
fi
if command -v crictl >/dev/null 2>&1; then
  run_grep container crictl ps -a
fi`, quotedTarget)
}

func parseHostTopProcessOutput(output string, topN int) ([]opsHubAgentHostProcessInfo, []opsHubAgentHostProcessInfo) {
	var memoryProcesses []opsHubAgentHostProcessInfo
	var cpuProcesses []opsHubAgentHostProcessInfo
	section := ""
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "":
			continue
		case "__OPSHUB_MEM__":
			section = "memory"
			continue
		case "__OPSHUB_CPU__":
			section = "cpu"
			continue
		}
		process, ok := parseHostTopProcessLine(line, section)
		if !ok {
			continue
		}
		if section == "memory" && len(memoryProcesses) < topN {
			memoryProcesses = append(memoryProcesses, process)
		}
		if section == "cpu" && len(cpuProcesses) < topN {
			cpuProcesses = append(cpuProcesses, process)
		}
	}
	return memoryProcesses, cpuProcesses
}

func parseHostTopProcessLine(line, section string) (opsHubAgentHostProcessInfo, bool) {
	fields := strings.Fields(line)
	if len(fields) < 7 {
		return opsHubAgentHostProcessInfo{}, false
	}
	pid, _ := strconv.Atoi(fields[0])
	ppid, _ := strconv.Atoi(fields[1])
	rssKB, _ := strconv.ParseInt(fields[6], 10, 64)
	var memPercent float64
	var cpuPercent float64
	if section == "cpu" {
		cpuPercent, _ = strconv.ParseFloat(fields[4], 64)
		memPercent, _ = strconv.ParseFloat(fields[5], 64)
	} else {
		memPercent, _ = strconv.ParseFloat(fields[4], 64)
		cpuPercent, _ = strconv.ParseFloat(fields[5], 64)
	}
	command := fields[3]
	if len(fields) > 7 {
		command = strings.Join(fields[7:], " ")
	}
	return opsHubAgentHostProcessInfo{
		PID:           pid,
		PPID:          ppid,
		User:          fields[2],
		Name:          fields[3],
		CPUPercent:    roundFloat(cpuPercent),
		MemoryPercent: roundFloat(memPercent),
		RSSKB:         rssKB,
		RSS:           formatBytesCN(uint64(rssKB * 1024)),
		Command:       truncateText(command, 500),
	}, pid > 0
}

func parseHostSoftwareProbeOutput(output string) []opsHubAgentHostSoftwareEvidence {
	items := make([]opsHubAgentHostSoftwareEvidence, 0)
	seen := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "__OPSHUB_SOFTWARE__\t") {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		evidenceType := strings.TrimSpace(parts[1])
		detail := truncateText(strings.TrimSpace(parts[2]), 500)
		if evidenceType == "" || detail == "" {
			continue
		}
		key := evidenceType + "\x00" + detail
		if seen[key] {
			continue
		}
		seen[key] = true
		items = append(items, opsHubAgentHostSoftwareEvidence{
			Type:    evidenceType,
			Detail:  detail,
			Running: isHostSoftwareEvidenceRunning(evidenceType, detail),
		})
	}
	return items
}

func isHostSoftwareEvidenceRunning(evidenceType, detail string) bool {
	kind := strings.ToLower(strings.TrimSpace(evidenceType))
	text := strings.ToLower(strings.TrimSpace(detail))
	switch kind {
	case "process":
		return true
	case "systemd_active":
		return strings.Contains(text, " active") || strings.HasSuffix(text, "active") || strings.Contains(text, "activating") || strings.Contains(text, "reloading")
	case "systemd_unit":
		return containsAnyText(text, []string{" active ", " running ", "activating", "reloading"})
	case "sysv_service":
		return strings.Contains(text, "[ + ]")
	case "container":
		return containsAnyText(text, []string{" up ", "running", "(running)"})
	default:
		return false
	}
}

func summarizeHostTopProcessItems(items []opsHubAgentHostTopProcessItem, total int64, limit, topN int, metric string) map[string]any {
	queried := 0
	failed := 0
	missingCredential := 0
	sshOrCommandFailed := 0
	for _, item := range items {
		if item.QuerySuccess {
			queried++
			continue
		}
		failed++
		if strings.Contains(item.Error, "凭证") {
			missingCredential++
		} else if item.Error != "" {
			sshOrCommandFailed++
		}
	}
	return map[string]any{
		"totalHosts":         total,
		"scannedHosts":       len(items),
		"queriedHosts":       queried,
		"failedHosts":        failed,
		"missingCredential":  missingCredential,
		"sshOrCommandFailed": sshOrCommandFailed,
		"truncated":          total > int64(len(items)),
		"limit":              limit,
		"topN":               topN,
		"metric":             metric,
	}
}

func summarizeHostSoftwareProbeItems(items []opsHubAgentHostSoftwareProbeItem, total int64, limit int, target string) map[string]any {
	queried := 0
	failed := 0
	deployed := 0
	running := 0
	missingCredential := 0
	sshOrCommandFailed := 0
	for _, item := range items {
		if item.QuerySuccess {
			queried++
		} else {
			failed++
			if strings.Contains(item.Error, "凭证") {
				missingCredential++
			} else if item.Error != "" {
				sshOrCommandFailed++
			}
		}
		if item.Deployed {
			deployed++
		}
		if item.Running {
			running++
		}
	}
	return map[string]any{
		"target":              target,
		"totalHosts":          total,
		"scannedHosts":        len(items),
		"queriedHosts":        queried,
		"failedHosts":         failed,
		"deployedHosts":       deployed,
		"runningHosts":        running,
		"notDetectedHosts":    queried - deployed,
		"missingCredential":   missingCredential,
		"sshOrCommandFailed":  sshOrCommandFailed,
		"truncated":           total > int64(len(items)),
		"limit":               limit,
		"evidenceDescription": "process/systemd/service/binary/package/config/container 任一命中即认为已部署，process/systemd active/sysv active/container running 命中即认为运行中",
	}
}

func deployedHostSoftwareProbeItems(items []opsHubAgentHostSoftwareProbeItem) []opsHubAgentHostSoftwareProbeItem {
	result := make([]opsHubAgentHostSoftwareProbeItem, 0)
	for _, item := range items {
		if item.Deployed {
			result = append(result, item)
		}
	}
	return result
}

func runningHostSoftwareProbeItems(items []opsHubAgentHostSoftwareProbeItem) []opsHubAgentHostSoftwareProbeItem {
	result := make([]opsHubAgentHostSoftwareProbeItem, 0)
	for _, item := range items {
		if item.Running {
			result = append(result, item)
		}
	}
	return result
}

func compactHostTopProcessDataForModel(data any) any {
	value, ok := data.(map[string]any)
	if !ok {
		return data
	}
	summary := value["topProcessSummary"]
	queryPolicy := value["queryPolicy"]
	compact := map[string]any{
		"topProcessSummary": summary,
		"queryPolicy":       queryPolicy,
		"overallTopMemory":  value["overallTopMemory"],
		"overallTopCPU":     value["overallTopCPU"],
	}
	items, ok := hostTopProcessItemsFromAny(value["hostTopProcesses"])
	if !ok {
		items, _ = hostTopProcessItemsFromAny(value["hosts"])
	}
	compactItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row := map[string]any{
			"hostId":            item.HostID,
			"name":              item.Name,
			"hostname":          item.Hostname,
			"ip":                item.IP,
			"groupName":         item.GroupName,
			"type":              item.Type,
			"status":            item.Status,
			"agentStatus":       item.AgentStatus,
			"storedCPUUsage":    item.StoredCPUUsage,
			"storedMemoryUsage": item.StoredMemoryUsage,
			"storedDiskUsage":   item.StoredDiskUsage,
			"querySuccess":      item.QuerySuccess,
			"error":             item.Error,
		}
		if len(item.TopCPUProcesses) > 0 {
			row["topCPUProcess"] = compactHostProcessForModel(item.TopCPUProcesses[0])
		}
		if len(item.TopMemoryProcesses) > 0 {
			row["topMemoryProcess"] = compactHostProcessForModel(item.TopMemoryProcesses[0])
		}
		compactItems = append(compactItems, row)
	}
	compact["hostTopProcesses"] = compactItems
	if summaryMap := mapStringAnyFromAny(summary); len(summaryMap) > 0 {
		queried := intFromAny(summaryMap["queriedHosts"])
		failed := intFromAny(summaryMap["failedHosts"])
		switch {
		case queried > 0 && failed > 0:
			compact["resultState"] = "partial_success"
		case queried > 0:
			compact["resultState"] = "success"
		default:
			compact["resultState"] = "failed"
		}
		compact["answerHint"] = "queriedHosts > 0 表示已经完成部分主机现场只读查询；不要把部分失败描述成整个工具失败。"
	}
	return compact
}

func compactHostSoftwareProbeDataForModel(data any) any {
	value, ok := data.(map[string]any)
	if !ok {
		return data
	}
	summary := value["softwareProbeSummary"]
	compact := map[string]any{
		"softwareProbeSummary": summary,
		"queryPolicy":          value["queryPolicy"],
	}
	items, ok := hostSoftwareProbeItemsFromAny(value["softwareProbe"])
	if !ok {
		items, _ = hostSoftwareProbeItemsFromAny(value["hosts"])
	}
	compactItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row := map[string]any{
			"hostId":            item.HostID,
			"name":              item.Name,
			"hostname":          item.Hostname,
			"ip":                item.IP,
			"groupName":         item.GroupName,
			"type":              item.Type,
			"status":            item.Status,
			"agentStatus":       item.AgentStatus,
			"storedCPUUsage":    item.StoredCPUUsage,
			"storedMemoryUsage": item.StoredMemoryUsage,
			"storedDiskUsage":   item.StoredDiskUsage,
			"target":            item.Target,
			"deployed":          item.Deployed,
			"running":           item.Running,
			"querySuccess":      item.QuerySuccess,
			"error":             item.Error,
		}
		if len(item.Evidence) > 0 {
			row["evidence"] = compactHostSoftwareEvidenceForModel(item.Evidence)
		}
		compactItems = append(compactItems, row)
	}
	compact["softwareProbe"] = compactItems
	if summaryMap := mapStringAnyFromAny(summary); len(summaryMap) > 0 {
		queried := intFromAny(summaryMap["queriedHosts"])
		failed := intFromAny(summaryMap["failedHosts"])
		deployed := intFromAny(summaryMap["deployedHosts"])
		switch {
		case queried > 0 && deployed > 0 && failed > 0:
			compact["resultState"] = "partial_success_with_deployments"
		case queried > 0 && deployed > 0:
			compact["resultState"] = "success_with_deployments"
		case queried > 0:
			compact["resultState"] = "success_no_deployment_detected"
		default:
			compact["resultState"] = "failed"
		}
		compact["answerHint"] = "软件部署问题必须以 deployed/running/evidence 字段为准；queriedHosts > 0 表示已进入主机现场只读探测，不能只按进程 Top 或上下文摘要回答。"
	}
	return compact
}

func mapStringAnyFromAny(value any) map[string]any {
	switch v := value.(type) {
	case map[string]any:
		return v
	default:
		return nil
	}
}

func hostSoftwareProbeItemsFromAny(value any) ([]opsHubAgentHostSoftwareProbeItem, bool) {
	switch v := value.(type) {
	case []opsHubAgentHostSoftwareProbeItem:
		return v, true
	case []any:
		items := make([]opsHubAgentHostSoftwareProbeItem, 0, len(v))
		for _, item := range v {
			if typed, ok := item.(opsHubAgentHostSoftwareProbeItem); ok {
				items = append(items, typed)
			}
		}
		return items, len(items) > 0
	default:
		return nil, false
	}
}

func compactHostSoftwareEvidenceForModel(evidence []opsHubAgentHostSoftwareEvidence) []map[string]any {
	limit := 8
	if len(evidence) < limit {
		limit = len(evidence)
	}
	result := make([]map[string]any, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, map[string]any{
			"type":    evidence[i].Type,
			"detail":  truncateText(evidence[i].Detail, 180),
			"running": evidence[i].Running,
		})
	}
	return result
}

func hostTopProcessItemsFromAny(value any) ([]opsHubAgentHostTopProcessItem, bool) {
	switch v := value.(type) {
	case []opsHubAgentHostTopProcessItem:
		return v, true
	case []any:
		items := make([]opsHubAgentHostTopProcessItem, 0, len(v))
		for _, item := range v {
			if typed, ok := item.(opsHubAgentHostTopProcessItem); ok {
				items = append(items, typed)
			}
		}
		return items, len(items) > 0
	default:
		return nil, false
	}
}

func compactHostProcessForModel(process opsHubAgentHostProcessInfo) map[string]any {
	return map[string]any{
		"pid":           process.PID,
		"name":          process.Name,
		"user":          process.User,
		"cpuPercent":    process.CPUPercent,
		"memoryPercent": process.MemoryPercent,
		"rss":           process.RSS,
		"command":       truncateText(process.Command, 180),
	}
}

func overallTopProcesses(items []opsHubAgentHostTopProcessItem, metric string, limit int) []map[string]any {
	type row struct {
		host    opsHubAgentHostTopProcessItem
		process opsHubAgentHostProcessInfo
	}
	rows := make([]row, 0, len(items))
	for _, item := range items {
		if !item.QuerySuccess {
			continue
		}
		var processes []opsHubAgentHostProcessInfo
		if metric == "cpu" {
			processes = item.TopCPUProcesses
		} else {
			processes = item.TopMemoryProcesses
		}
		if len(processes) == 0 {
			continue
		}
		rows = append(rows, row{host: item, process: processes[0]})
	}
	sort.Slice(rows, func(i, j int) bool {
		if metric == "cpu" {
			return rows[i].process.CPUPercent > rows[j].process.CPUPercent
		}
		return rows[i].process.MemoryPercent > rows[j].process.MemoryPercent
	})
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{
			"hostId":        row.host.HostID,
			"hostName":      row.host.Name,
			"ip":            row.host.IP,
			"groupName":     row.host.GroupName,
			"pid":           row.process.PID,
			"name":          row.process.Name,
			"cpuPercent":    row.process.CPUPercent,
			"memoryPercent": row.process.MemoryPercent,
			"rss":           row.process.RSS,
			"command":       row.process.Command,
		})
	}
	return result
}

func hostTopProcessQueryTimeout(hostCount int) time.Duration {
	if hostCount <= 0 {
		return 30 * time.Second
	}
	seconds := 30 + (hostCount/6)*18
	if seconds < 60 {
		seconds = 60
	}
	if seconds > 300 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}

func hostSoftwareProbeQueryTimeout(hostCount int) time.Duration {
	if hostCount <= 0 {
		return 30 * time.Second
	}
	seconds := 35 + (hostCount/6)*22
	if seconds < 75 {
		seconds = 75
	}
	if seconds > 360 {
		seconds = 360
	}
	return time.Duration(seconds) * time.Second
}

func boundedIntParam(params map[string]any, fallback, minValue, maxValue int, keys ...string) int {
	value := intParam(params, keys...)
	if value == 0 {
		value = fallback
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func intParam(params map[string]any, keys ...string) int {
	if params == nil {
		return 0
	}
	for _, key := range keys {
		value, ok := params[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case uint:
			return int(v)
		case uint32:
			return int(v)
		case uint64:
			return int(v)
		case float64:
			return int(v)
		case string:
			n, _ := strconv.Atoi(strings.TrimSpace(v))
			return n
		}
	}
	return 0
}

func uintParam(params map[string]any, keys ...string) uint {
	value := intParam(params, keys...)
	if value <= 0 {
		return 0
	}
	return uint(value)
}
