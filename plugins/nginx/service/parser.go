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
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/nginx/model"
)

// ParserService 日志解析服务
type ParserService struct {
	uaParser *UAParser
}

// NewParserService 创建解析服务
func NewParserService() *ParserService {
	return &ParserService{
		uaParser: NewUAParser(),
	}
}

// ParseLogs 解析日志内容
func (s *ParserService) ParseLogs(content string, format string) []model.ParsedLogEntry {
	var entries []model.ParsedLogEntry

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry model.ParsedLogEntry
		var err error

		switch format {
		case "json":
			entry, err = s.parseJSONLine(line)
		case "combined", "":
			entry, err = s.parseCombinedLine(line)
		default:
			entry, err = s.parseCombinedLine(line)
		}

		if err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	return entries
}

// parseCombinedLine 解析 combined 格式日志
func (s *ParserService) parseCombinedLine(line string) (model.ParsedLogEntry, error) {
	var entry model.ParsedLogEntry

	// combined 格式: $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"
	// 可选: $request_time $upstream_response_time $host
	pattern := regexp.MustCompile(`^(\S+)\s+-\s+(\S+)\s+\[([^\]]+)\]\s+"([^"]+)"\s+(\d+)\s+(\d+)\s+"([^"]*)"\s+"([^"]*)"(?:\s+(\S+))?(?:\s+(\S+))?(?:\s+(\S+))?`)

	matches := pattern.FindStringSubmatch(line)
	if matches == nil {
		return entry, fmt.Errorf("cannot parse log line")
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
		t, err = time.Parse("02/Jan/2006:15:04:05", timeStr)
		if err != nil {
			return entry, fmt.Errorf("parse time failed: %w", err)
		}
	}
	entry.Timestamp = t

	// 解析请求行
	request := matches[4]
	parts := strings.SplitN(request, " ", 3)
	if len(parts) >= 2 {
		entry.Method = parts[0]
		entry.URI = parts[1]
		if len(parts) >= 3 {
			entry.Protocol = parts[2]
		}
	}

	entry.Status, _ = strconv.Atoi(matches[5])
	entry.BodyBytesSent, _ = strconv.ParseInt(matches[6], 10, 64)

	entry.HTTPReferer = matches[7]
	if entry.HTTPReferer == "-" {
		entry.HTTPReferer = ""
	}

	entry.HTTPUserAgent = matches[8]
	if entry.HTTPUserAgent == "-" {
		entry.HTTPUserAgent = ""
	}

	// 可选字段
	if len(matches) > 9 && matches[9] != "" && matches[9] != "-" {
		entry.RequestTime, _ = strconv.ParseFloat(matches[9], 64)
	}
	if len(matches) > 10 && matches[10] != "" && matches[10] != "-" {
		entry.UpstreamTime, _ = strconv.ParseFloat(matches[10], 64)
	}
	if len(matches) > 11 && matches[11] != "" && matches[11] != "-" {
		entry.Host = matches[11]
	}

	// 尝试从 referer 提取 host
	if entry.Host == "" {
		entry.Host = s.extractHost(entry.HTTPReferer)
	}

	return entry, nil
}

// JSONLogEntry JSON 日志格式
type JSONLogEntry struct {
	RemoteAddr     string  `json:"remote_addr"`
	RemoteUser     string  `json:"remote_user"`
	TimeLocal      string  `json:"time_local"`
	Time           string  `json:"time"`
	Timestamp      string  `json:"@timestamp"`
	Request        string  `json:"request"`
	RequestMethod  string  `json:"request_method"`
	RequestURI     string  `json:"request_uri"`
	URI            string  `json:"uri"`
	ServerProtocol string  `json:"server_protocol"`
	Status         int     `json:"status"`
	BodyBytesSent  int64   `json:"body_bytes_sent"`
	HTTPReferer    string  `json:"http_referer"`
	HTTPUserAgent  string  `json:"http_user_agent"`
	RequestTime    float64 `json:"request_time"`
	UpstreamTime   string  `json:"upstream_response_time"`
	Host           string  `json:"host"`
	ServerName     string  `json:"server_name"`

	// K8s Ingress 特有字段
	IngressName  string `json:"ingress_name"`
	ServiceName  string `json:"service_name"`
	ServicePort  string `json:"service_port"`
	Namespace    string `json:"namespace"`
	PodName      string `json:"pod_name"`
	UpstreamAddr string `json:"upstream_addr"`
}

// parseJSONLine 解析 JSON 格式日志
func (s *ParserService) parseJSONLine(line string) (model.ParsedLogEntry, error) {
	var entry model.ParsedLogEntry
	var jsonEntry JSONLogEntry

	if err := json.Unmarshal([]byte(line), &jsonEntry); err != nil {
		return entry, fmt.Errorf("json unmarshal failed: %w", err)
	}

	entry.RemoteAddr = jsonEntry.RemoteAddr
	entry.RemoteUser = jsonEntry.RemoteUser
	if entry.RemoteUser == "-" {
		entry.RemoteUser = ""
	}

	// 解析时间
	timeStr := jsonEntry.TimeLocal
	if timeStr == "" {
		timeStr = jsonEntry.Time
	}
	if timeStr == "" {
		timeStr = jsonEntry.Timestamp
	}

	if timeStr != "" {
		// 尝试多种时间格式
		formats := []string{
			"02/Jan/2006:15:04:05 -0700",
			"02/Jan/2006:15:04:05",
			time.RFC3339,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05",
		}
		for _, f := range formats {
			t, err := time.Parse(f, timeStr)
			if err == nil {
				entry.Timestamp = t
				break
			}
		}
	}

	// 解析请求
	if jsonEntry.RequestMethod != "" {
		entry.Method = jsonEntry.RequestMethod
	}
	if jsonEntry.RequestURI != "" {
		entry.URI = jsonEntry.RequestURI
	} else if jsonEntry.URI != "" {
		entry.URI = jsonEntry.URI
	}
	if jsonEntry.ServerProtocol != "" {
		entry.Protocol = jsonEntry.ServerProtocol
	}

	// 如果有完整的 request 字段，解析它
	if entry.Method == "" && jsonEntry.Request != "" {
		parts := strings.SplitN(jsonEntry.Request, " ", 3)
		if len(parts) >= 2 {
			entry.Method = parts[0]
			entry.URI = parts[1]
			if len(parts) >= 3 {
				entry.Protocol = parts[2]
			}
		}
	}

	entry.Status = jsonEntry.Status
	entry.BodyBytesSent = jsonEntry.BodyBytesSent
	entry.HTTPReferer = jsonEntry.HTTPReferer
	if entry.HTTPReferer == "-" {
		entry.HTTPReferer = ""
	}
	entry.HTTPUserAgent = jsonEntry.HTTPUserAgent
	if entry.HTTPUserAgent == "-" {
		entry.HTTPUserAgent = ""
	}

	entry.RequestTime = jsonEntry.RequestTime
	if jsonEntry.UpstreamTime != "" && jsonEntry.UpstreamTime != "-" {
		entry.UpstreamTime, _ = strconv.ParseFloat(jsonEntry.UpstreamTime, 64)
	}

	entry.Host = jsonEntry.Host
	if entry.Host == "" {
		entry.Host = jsonEntry.ServerName
	}

	// K8s 字段
	entry.IngressName = jsonEntry.IngressName
	entry.ServiceName = jsonEntry.ServiceName
	entry.PodName = jsonEntry.PodName

	return entry, nil
}

// extractHost 从 URL 提取主机名
func (s *ParserService) extractHost(referer string) string {
	if referer == "" || referer == "-" {
		return ""
	}

	u, err := url.Parse(referer)
	if err != nil {
		return ""
	}

	return u.Host
}

// HashString 计算字符串的 SHA256 哈希
func HashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

// NormalizeURL 规范化 URL (去除查询参数)
func NormalizeURL(uri string) string {
	if uri == "" {
		return ""
	}

	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	// 返回不带查询参数的路径
	return u.Path
}

// ExtractRefererDomain 从 Referer 提取域名
func ExtractRefererDomain(referer string) string {
	if referer == "" || referer == "-" {
		return ""
	}

	u, err := url.Parse(referer)
	if err != nil {
		return ""
	}

	return u.Host
}

// ClassifyReferer 分类 Referer 来源
func ClassifyReferer(referer string) string {
	if referer == "" || referer == "-" {
		return "direct"
	}

	domain := ExtractRefererDomain(referer)
	domainLower := strings.ToLower(domain)

	// 搜索引擎
	searchEngines := []string{"google", "baidu", "bing", "yahoo", "sogou", "360", "soso", "yandex", "duckduckgo"}
	for _, se := range searchEngines {
		if strings.Contains(domainLower, se) {
			return "search"
		}
	}

	// 社交网络
	socialNetworks := []string{"facebook", "twitter", "linkedin", "weibo", "wechat", "qq", "instagram", "tiktok", "douyin", "reddit"}
	for _, sn := range socialNetworks {
		if strings.Contains(domainLower, sn) {
			return "social"
		}
	}

	return "other"
}

// IsPVRequest 判断是否为页面访问请求
func IsPVRequest(uri string, status int) bool {
	// 排除静态资源
	staticExtensions := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".woff", ".woff2", ".ttf", ".eot", ".map"}
	uriLower := strings.ToLower(uri)
	for _, ext := range staticExtensions {
		if strings.HasSuffix(uriLower, ext) {
			return false
		}
	}

	// 排除 API 请求路径
	if strings.Contains(uriLower, "/api/") || strings.HasPrefix(uriLower, "/api") {
		return false
	}

	// 排除健康检查
	healthPaths := []string{"/health", "/ping", "/ready", "/live", "/metrics"}
	for _, hp := range healthPaths {
		if strings.HasPrefix(uriLower, hp) {
			return false
		}
	}

	// 只有成功的请求才算 PV
	if status < 200 || status >= 400 {
		return false
	}

	return true
}

// UAParser User-Agent 解析器
type UAParser struct {
	botPatterns     []*regexp.Regexp
	browserPatterns map[string]*regexp.Regexp
	osPatterns      map[string]*regexp.Regexp
	mobilePatterns  []*regexp.Regexp
	tabletPatterns  []*regexp.Regexp
}

// NewUAParser 创建 UA 解析器
func NewUAParser() *UAParser {
	parser := &UAParser{
		browserPatterns: make(map[string]*regexp.Regexp),
		osPatterns:      make(map[string]*regexp.Regexp),
	}

	// Bot 模式
	parser.botPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)bot|crawler|spider|scraper|curl|wget|python|java|go-http|node-fetch|axios|httpclient`),
		regexp.MustCompile(`(?i)googlebot|bingbot|yandex|baidu|slurp|duckduck|facebookexternalhit`),
		regexp.MustCompile(`(?i)monitoring|uptime|pingdom|newrelic|datadog|prometheus`),
	}

	// 浏览器模式 (顺序很重要)
	parser.browserPatterns["Edge"] = regexp.MustCompile(`Edg[eA]?/(\d+[\.\d]*)`)
	parser.browserPatterns["Chrome"] = regexp.MustCompile(`Chrome/(\d+[\.\d]*)`)
	parser.browserPatterns["Firefox"] = regexp.MustCompile(`Firefox/(\d+[\.\d]*)`)
	parser.browserPatterns["Safari"] = regexp.MustCompile(`Version/(\d+[\.\d]*).* Safari`)
	parser.browserPatterns["Opera"] = regexp.MustCompile(`(?:Opera|OPR)/(\d+[\.\d]*)`)
	parser.browserPatterns["IE"] = regexp.MustCompile(`(?:MSIE |rv:)(\d+[\.\d]*)`)

	// 操作系统模式
	parser.osPatterns["Windows"] = regexp.MustCompile(`Windows NT (\d+[\.\d]*)`)
	parser.osPatterns["macOS"] = regexp.MustCompile(`Mac OS X (\d+[_\.\d]*)`)
	parser.osPatterns["iOS"] = regexp.MustCompile(`(?:iPhone|iPad|iPod).*OS (\d+[_\.\d]*)`)
	parser.osPatterns["Android"] = regexp.MustCompile(`Android (\d+[\.\d]*)`)
	parser.osPatterns["Linux"] = regexp.MustCompile(`Linux`)

	// 移动设备模式
	parser.mobilePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)iPhone|iPod|Android.*Mobile|Windows Phone|BlackBerry|IEMobile|Opera Mini|Opera Mobi`),
	}

	// 平板设备模式
	parser.tabletPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)iPad|Android(?!.*Mobile)|Tablet|PlayBook|Silk`),
	}

	return parser
}

// UAInfo User-Agent 解析结果
type UAInfo struct {
	Browser        string
	BrowserVersion string
	OS             string
	OSVersion      string
	DeviceType     string
	IsBot          bool
}

// Parse 解析 User-Agent
func (p *UAParser) Parse(ua string) UAInfo {
	info := UAInfo{
		Browser:    "Unknown",
		OS:         "Unknown",
		DeviceType: "desktop",
	}

	if ua == "" || ua == "-" {
		return info
	}

	// 检测 Bot
	for _, pattern := range p.botPatterns {
		if pattern.MatchString(ua) {
			info.IsBot = true
			info.DeviceType = "bot"
			break
		}
	}

	// 检测浏览器 (按顺序检查以确保正确识别)
	browserOrder := []string{"Edge", "Chrome", "Firefox", "Safari", "Opera", "IE"}
	for _, browser := range browserOrder {
		pattern := p.browserPatterns[browser]
		if matches := pattern.FindStringSubmatch(ua); matches != nil {
			info.Browser = browser
			if len(matches) > 1 {
				info.BrowserVersion = matches[1]
			}
			break
		}
	}

	// 检测操作系统 (按顺序检查)
	osOrder := []string{"iOS", "Android", "Windows", "macOS", "Linux"}
	for _, os := range osOrder {
		pattern := p.osPatterns[os]
		if matches := pattern.FindStringSubmatch(ua); matches != nil {
			info.OS = os
			if len(matches) > 1 {
				version := matches[1]
				version = strings.ReplaceAll(version, "_", ".")
				info.OSVersion = version
			}
			break
		}
	}

	// 检测设备类型 (如果不是 bot)
	if !info.IsBot {
		for _, pattern := range p.tabletPatterns {
			if pattern.MatchString(ua) {
				info.DeviceType = "tablet"
				break
			}
		}
		if info.DeviceType == "desktop" {
			for _, pattern := range p.mobilePatterns {
				if pattern.MatchString(ua) {
					info.DeviceType = "mobile"
					break
				}
			}
		}
	}

	return info
}

// GetUAParser 获取 UA 解析器
func (s *ParserService) GetUAParser() *UAParser {
	return s.uaParser
}
