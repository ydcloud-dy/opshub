package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/app-inventory/model"
	k8sservice "github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultProbeInterval = 5 * time.Minute
	probeTimeout         = 8 * time.Second
)

type healthMonitor struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func newHealthMonitor() *healthMonitor { return &healthMonitor{} }

func (s *Service) StartHealthMonitor() {
	if s == nil || s.monitor == nil {
		return
	}
	s.monitor.mu.Lock()
	defer s.monitor.mu.Unlock()
	if s.monitor.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.monitor.cancel = cancel
	s.monitor.done = done
	interval := probeInterval()
	go func() {
		defer close(done)
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			_ = s.ProbeAll(ctx)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = s.ProbeAll(ctx)
			}
		}
	}()
}

func (s *Service) StopHealthMonitor() {
	if s == nil || s.monitor == nil {
		return
	}
	s.monitor.mu.Lock()
	cancel, done := s.monitor.cancel, s.monitor.done
	s.monitor.cancel, s.monitor.done = nil, nil
	s.monitor.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
}

func probeInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv("OPSHUB_APP_INVENTORY_PROBE_INTERVAL"))
	if value == "" {
		return defaultProbeInterval
	}
	interval, err := time.ParseDuration(value)
	if err != nil || interval < 30*time.Second {
		return defaultProbeInterval
	}
	return interval
}

func (s *Service) ProbeAll(ctx context.Context) error {
	var appIDs []uint
	if err := s.db.WithContext(ctx).Model(&model.Application{}).Where("status = ?", "active").Pluck("id", &appIDs).Error; err != nil {
		return err
	}
	for _, appID := range appIDs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := s.ProbeApplication(ctx, appID); err != nil {
			continue
		}
	}
	return nil
}

func (s *Service) ProbeApplication(ctx context.Context, appID uint, userIDs ...uint) (*model.Application, error) {
	if err := s.ensureApplication(ctx, appID); err != nil {
		return nil, err
	}
	var domainIDs, resourceIDs, componentIDs []uint
	if err := s.db.WithContext(ctx).Model(&model.Domain{}).Where("application_id = ?", appID).Pluck("id", &domainIDs).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.Resource{}).Where("application_id = ?", appID).Pluck("id", &resourceIDs).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.Component{}).Where("application_id = ?", appID).Pluck("id", &componentIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range domainIDs {
		_, _ = s.probeDomain(ctx, id, false)
	}
	for _, id := range resourceIDs {
		_, _ = s.probeResource(ctx, id, false, userIDs...)
	}
	for _, id := range componentIDs {
		_, _ = s.probeComponent(ctx, id, false)
	}
	if err := s.RecomputeApplicationHealth(ctx, appID); err != nil {
		return nil, err
	}
	var app model.Application
	if err := s.db.WithContext(ctx).First(&app, appID).Error; err != nil {
		return nil, err
	}
	items := []model.Application{app}
	if err := s.enrichApplications(ctx, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (s *Service) ProbeDomain(ctx context.Context, id uint) (*model.Domain, error) {
	return s.probeDomain(ctx, id, true)
}

func (s *Service) probeDomain(ctx context.Context, id uint, recompute bool) (*model.Domain, error) {
	var item model.Domain
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	started := time.Now()
	status, message := "checking", "正在检测"
	item.ResponseTimeMS = 0
	item.HTTPStatusCode = 0
	item.ResolvedAddress = ""
	item.DNSProvider = ""
	item.TLSExpiresAt = nil
	item.TLSIssuer = ""

	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	host := normalizeProbeHost(item.Domain)
	addresses, resolveErr := net.DefaultResolver.LookupHost(probeCtx, host)
	if resolveErr != nil {
		status, message = "unhealthy", "DNS 解析失败: "+resolveErr.Error()
	} else {
		item.ResolvedAddress = strings.Join(addresses, ", ")
		dnsContext, dnsCancel := context.WithTimeout(ctx, 2*time.Second)
		item.DNSProvider = detectDNSProvider(dnsContext, host)
		dnsCancel()
		endpoint := net.JoinHostPort(host, strconv.Itoa(item.Port))
		if item.Protocol == "tcp" {
			conn, err := (&net.Dialer{Timeout: probeTimeout}).DialContext(probeCtx, "tcp", endpoint)
			if err != nil {
				status, message = "unhealthy", "TCP 连接失败: "+err.Error()
			} else {
				_ = conn.Close()
				status, message = "healthy", "TCP 连接正常"
			}
		} else {
			status, message = probeHTTPDomain(probeCtx, &item, host)
		}
	}
	item.Status = status
	item.ProbeMessage = truncateProbeMessage(message)
	now := time.Now()
	item.LastCheckedAt = &now
	item.ResponseTimeMS = elapsedMilliseconds(started)
	if err := s.db.WithContext(ctx).Model(&model.Domain{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
		"status":           item.Status,
		"last_checked_at":  item.LastCheckedAt,
		"response_time_ms": item.ResponseTimeMS,
		"http_status_code": item.HTTPStatusCode,
		"probe_message":    item.ProbeMessage,
		"resolved_address": item.ResolvedAddress,
		"dns_provider":     item.DNSProvider,
		"tls_expires_at":   item.TLSExpiresAt,
		"tls_issuer":       item.TLSIssuer,
	}).Error; err != nil {
		return nil, err
	}
	if recompute {
		_ = s.RecomputeApplicationHealth(ctx, item.ApplicationID)
	}
	return &item, nil
}

func detectDNSProvider(ctx context.Context, host string) string {
	labels := strings.Split(strings.Trim(strings.ToLower(host), "."), ".")
	for index := 0; index < len(labels)-1; index++ {
		nameservers, err := net.DefaultResolver.LookupNS(ctx, strings.Join(labels[index:], "."))
		if err != nil || len(nameservers) == 0 {
			continue
		}
		hosts := make([]string, 0, len(nameservers))
		for _, nameserver := range nameservers {
			hosts = append(hosts, strings.TrimSuffix(strings.ToLower(nameserver.Host), "."))
		}
		return dnsProviderFromNameservers(hosts)
	}
	return ""
}

func dnsProviderFromNameservers(nameservers []string) string {
	joined := strings.Join(nameservers, " ")
	providers := []struct {
		needles []string
		name    string
	}{
		{needles: []string{"alidns", "hichina"}, name: "阿里云 DNS"},
		{needles: []string{"cloudflare"}, name: "Cloudflare"},
		{needles: []string{"dnspod", "dnsv"}, name: "DNSPod"},
		{needles: []string{"awsdns"}, name: "AWS Route 53"},
		{needles: []string{"azure-dns"}, name: "Azure DNS"},
		{needles: []string{"googledomains"}, name: "Google Cloud DNS"},
		{needles: []string{"huaweicloud-dns"}, name: "华为云 DNS"},
		{needles: []string{"nsone"}, name: "NS1"},
	}
	for _, provider := range providers {
		for _, needle := range provider.needles {
			if strings.Contains(joined, needle) {
				return provider.name
			}
		}
	}
	if len(nameservers) > 0 {
		return nameservers[0]
	}
	return ""
}

func probeHTTPDomain(ctx context.Context, item *model.Domain, host string) (string, string) {
	hostPort := host
	if (item.Protocol == "https" && item.Port != 443) || (item.Protocol == "http" && item.Port != 80) {
		hostPort = net.JoinHostPort(host, strconv.Itoa(item.Port))
	}
	path := item.Path
	if path == "" {
		path = "/"
	}
	target := (&url.URL{Scheme: item.Protocol, Host: hostPort, Path: path}).String()
	client := &http.Client{
		Timeout: probeTimeout,
		Transport: &http.Transport{
			DialContext:         (&net.Dialer{Timeout: probeTimeout}).DialContext,
			TLSHandshakeTimeout: probeTimeout,
		},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("重定向次数过多")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "unhealthy", "构造探测请求失败: " + err.Error()
	}
	req.Header.Set("User-Agent", "OpsHub-AppInventory-Probe/1.0")
	resp, err := client.Do(req)
	if err != nil {
		if item.Protocol == "https" {
			inspectDomainTLS(ctx, item, host)
		}
		return "unhealthy", "访问失败: " + err.Error()
	}
	defer resp.Body.Close()
	item.HTTPStatusCode = resp.StatusCode
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		applyTLSCertificate(item, resp.TLS.PeerCertificates[0])
	}
	status := "healthy"
	message := fmt.Sprintf("HTTP %d，访问正常", resp.StatusCode)
	if resp.StatusCode >= 500 {
		status, message = "unhealthy", fmt.Sprintf("HTTP %d，服务端响应异常", resp.StatusCode)
	} else if resp.StatusCode >= 400 {
		status, message = "warning", fmt.Sprintf("HTTP %d，入口可达但业务响应需关注", resp.StatusCode)
	}
	if item.TLSExpiresAt != nil {
		remaining := time.Until(*item.TLSExpiresAt)
		if remaining <= 0 {
			return "unhealthy", "TLS 证书已过期"
		}
		if remaining <= 30*24*time.Hour && status == "healthy" {
			return "warning", fmt.Sprintf("TLS 证书将在 %d 天内到期", int(remaining.Hours()/24)+1)
		}
	}
	return status, message
}

func inspectDomainTLS(ctx context.Context, item *model.Domain, host string) {
	dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: probeTimeout}, Config: &tls.Config{ServerName: host, InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(item.Port)))
	if err != nil {
		return
	}
	defer conn.Close()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok || len(tlsConn.ConnectionState().PeerCertificates) == 0 {
		return
	}
	applyTLSCertificate(item, tlsConn.ConnectionState().PeerCertificates[0])
}

func applyTLSCertificate(item *model.Domain, certificate *x509.Certificate) {
	expiresAt := certificate.NotAfter
	item.TLSExpiresAt = &expiresAt
	item.TLSIssuer = certificate.Issuer.String()
}

func (s *Service) ProbeResource(ctx context.Context, id uint, userIDs ...uint) (*model.Resource, error) {
	return s.probeResource(ctx, id, true, userIDs...)
}

func (s *Service) probeResource(ctx context.Context, id uint, recompute bool, userIDs ...uint) (*model.Resource, error) {
	var item model.Resource
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	if len(userIDs) > 0 && userIDs[0] > 0 && item.HostID > 0 {
		if err := s.ensureHostAccess(ctx, item.HostID, userIDs[0]); err != nil {
			return nil, err
		}
	}
	started := time.Now()
	status, message := "unknown", "暂无可用探测方式"
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	switch {
	case oneOf(item.Kind, "Host", "VirtualMachine"):
		status, message = s.probeHostResource(probeCtx, &item)
	case isKubernetesKind(item.Kind):
		status, message = s.probeKubernetesResource(probeCtx, &item)
	case strings.TrimSpace(item.Address) != "" && item.Port > 0:
		status, message = probeTCP(probeCtx, item.Address, item.Port, false)
	}
	item.Status = status
	item.HealthMessage = truncateProbeMessage(message)
	now := time.Now()
	item.LastCheckedAt = &now
	item.ResponseTimeMS = elapsedMilliseconds(started)
	if err := s.db.WithContext(ctx).Model(&model.Resource{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
		"address":          item.Address,
		"port":             item.Port,
		"status":           item.Status,
		"last_checked_at":  item.LastCheckedAt,
		"response_time_ms": item.ResponseTimeMS,
		"health_message":   item.HealthMessage,
	}).Error; err != nil {
		return nil, err
	}
	if recompute {
		_ = s.RecomputeApplicationHealth(ctx, item.ApplicationID)
	}
	return &item, nil
}

func (s *Service) probeHostResource(ctx context.Context, item *model.Resource) (string, string) {
	var host struct {
		IP     string
		Status int
	}
	if err := s.db.WithContext(ctx).Table("hosts").Select("ip, status").Where("id = ? AND deleted_at IS NULL", item.HostID).First(&host).Error; err != nil {
		return "unhealthy", "关联主机不存在"
	}
	item.Address = strings.TrimSpace(host.IP)
	if item.Address == "" {
		return "unhealthy", "关联主机没有有效 IP 地址"
	}
	if host.Status == 0 {
		return "unhealthy", "资产管理显示主机离线"
	}
	if host.Status < 0 {
		return "unknown", "资产管理尚未确认主机在线状态"
	}
	if item.Port > 0 {
		return probeTCP(ctx, item.Address, item.Port, false)
	}
	return "healthy", "资产管理显示主机在线"
}

func (s *Service) probeKubernetesResource(ctx context.Context, item *model.Resource) (string, string) {
	var cluster struct {
		Status int
	}
	if err := s.db.WithContext(ctx).Table("k8s_clusters").Select("status").Where("id = ?", item.ClusterID).First(&cluster).Error; err != nil {
		return "unhealthy", "关联的 Kubernetes 集群不存在"
	}
	if cluster.Status != 1 {
		return "unhealthy", "Kubernetes 集群当前不可用"
	}
	clientset, err := k8sservice.NewClusterService(s.db).GetCachedClientset(ctx, item.ClusterID)
	if err != nil {
		return "unhealthy", "连接 Kubernetes 集群失败: " + err.Error()
	}
	options := metav1.GetOptions{}
	switch item.Kind {
	case "Deployment":
		value, err := clientset.AppsV1().Deployments(item.Namespace).Get(ctx, item.Name, options)
		if err != nil {
			return "unhealthy", "读取 Deployment 失败: " + err.Error()
		}
		desired := int32(1)
		if value.Spec.Replicas != nil {
			desired = *value.Spec.Replicas
		}
		if desired == 0 {
			return "warning", "Deployment 副本数为 0"
		}
		if value.Status.AvailableReplicas < desired {
			return "unhealthy", fmt.Sprintf("Deployment 可用副本 %d/%d", value.Status.AvailableReplicas, desired)
		}
		return "healthy", fmt.Sprintf("Deployment 可用副本 %d/%d", value.Status.AvailableReplicas, desired)
	case "StatefulSet":
		value, err := clientset.AppsV1().StatefulSets(item.Namespace).Get(ctx, item.Name, options)
		if err != nil {
			return "unhealthy", "读取 StatefulSet 失败: " + err.Error()
		}
		desired := int32(1)
		if value.Spec.Replicas != nil {
			desired = *value.Spec.Replicas
		}
		if desired == 0 {
			return "warning", "StatefulSet 副本数为 0"
		}
		if value.Status.ReadyReplicas < desired {
			return "unhealthy", fmt.Sprintf("StatefulSet 就绪副本 %d/%d", value.Status.ReadyReplicas, desired)
		}
		return "healthy", fmt.Sprintf("StatefulSet 就绪副本 %d/%d", value.Status.ReadyReplicas, desired)
	case "DaemonSet":
		value, err := clientset.AppsV1().DaemonSets(item.Namespace).Get(ctx, item.Name, options)
		if err != nil {
			return "unhealthy", "读取 DaemonSet 失败: " + err.Error()
		}
		if value.Status.DesiredNumberScheduled == 0 {
			return "warning", "DaemonSet 尚未调度到节点"
		}
		if value.Status.NumberReady < value.Status.DesiredNumberScheduled {
			return "unhealthy", fmt.Sprintf("DaemonSet 就绪节点 %d/%d", value.Status.NumberReady, value.Status.DesiredNumberScheduled)
		}
		return "healthy", fmt.Sprintf("DaemonSet 就绪节点 %d/%d", value.Status.NumberReady, value.Status.DesiredNumberScheduled)
	case "Service":
		value, err := clientset.CoreV1().Services(item.Namespace).Get(ctx, item.Name, options)
		if err != nil {
			return "unhealthy", "读取 Service 失败: " + err.Error()
		}
		item.Address = value.Spec.ClusterIP
		if len(value.Spec.Ports) > 0 {
			item.Port = int(value.Spec.Ports[0].Port)
		}
		return "healthy", "Service 对象存在且集群可访问"
	case "Ingress":
		value, err := clientset.NetworkingV1().Ingresses(item.Namespace).Get(ctx, item.Name, options)
		if err != nil {
			return "unhealthy", "读取 Ingress 失败: " + err.Error()
		}
		if len(value.Status.LoadBalancer.Ingress) > 0 {
			item.Address = firstNonEmpty(value.Status.LoadBalancer.Ingress[0].IP, value.Status.LoadBalancer.Ingress[0].Hostname)
		}
		if item.Address == "" && len(value.Spec.Rules) > 0 {
			item.Address = value.Spec.Rules[0].Host
		}
		return "healthy", "Ingress 对象存在且集群可访问"
	default:
		return "warning", "资源类型暂不支持工作负载级探测，集群连接正常"
	}
}

func (s *Service) ProbeComponent(ctx context.Context, id uint) (*model.Component, error) {
	return s.probeComponent(ctx, id, true)
}

func (s *Service) probeComponent(ctx context.Context, id uint, recompute bool) (*model.Component, error) {
	var item model.Component
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	started := time.Now()
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	status, message := probeTCP(probeCtx, item.Address, item.Port, item.TLSEnabled)
	item.Status = status
	item.HealthMessage = truncateProbeMessage(message)
	now := time.Now()
	item.LastCheckedAt = &now
	item.ResponseTimeMS = elapsedMilliseconds(started)
	if err := s.db.WithContext(ctx).Model(&model.Component{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
		"status":           item.Status,
		"last_checked_at":  item.LastCheckedAt,
		"response_time_ms": item.ResponseTimeMS,
		"health_message":   item.HealthMessage,
	}).Error; err != nil {
		return nil, err
	}
	if recompute {
		_ = s.RecomputeApplicationHealth(ctx, item.ApplicationID)
	}
	return &item, nil
}

func probeTCP(ctx context.Context, address string, port int, tlsEnabled bool) (string, string) {
	host := normalizeProbeHost(address)
	if host == "" || port <= 0 {
		return "unknown", "缺少可探测的地址或端口"
	}
	endpoint := net.JoinHostPort(host, strconv.Itoa(port))
	if tlsEnabled {
		dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: probeTimeout}, Config: &tls.Config{ServerName: host, InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}}
		conn, err := dialer.DialContext(ctx, "tcp", endpoint)
		if err != nil {
			return "unhealthy", "TLS 连接失败: " + err.Error()
		}
		_ = conn.Close()
		return "healthy", "TLS 端口连接正常"
	}
	conn, err := (&net.Dialer{Timeout: probeTimeout}).DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return "unhealthy", "TCP 连接失败: " + err.Error()
	}
	_ = conn.Close()
	return "healthy", "TCP 端口连接正常"
}

func normalizeProbeHost(value string) string {
	value = strings.TrimSpace(value)
	if parsed, err := url.Parse(value); err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(value, "[]")
}

func (s *Service) RecomputeApplicationHealth(ctx context.Context, appID uint) error {
	var app model.Application
	if err := s.db.WithContext(ctx).Select("id, status, environment_id").First(&app, appID).Error; err != nil {
		return err
	}
	if app.Status != "active" {
		now := time.Now()
		return s.db.WithContext(ctx).Model(&model.Application{}).Where("id = ?", appID).Updates(map[string]interface{}{
			"health_status":     "unknown",
			"health_checked_at": &now,
			"health_message":    "应用未处于运行状态",
			"health_source":     "asset-aggregation",
		}).Error
	}
	statuses := make([]string, 0)
	for _, spec := range []struct {
		model interface{}
		field string
	}{
		{model: &model.Domain{}, field: "status"},
		{model: &model.Resource{}, field: "status"},
		{model: &model.Component{}, field: "status"},
	} {
		var values []string
		if err := s.db.WithContext(ctx).Model(spec.model).Where("application_id = ?", appID).Pluck(spec.field, &values).Error; err != nil {
			return err
		}
		statuses = append(statuses, values...)
	}
	if app.EnvironmentID > 0 {
		var environmentStatus string
		if err := s.db.WithContext(ctx).Model(&model.Environment{}).Where("id = ?", app.EnvironmentID).Pluck("status", &environmentStatus).Error; err == nil && environmentStatus == "disabled" {
			statuses = append(statuses, "warning")
		}
	}
	status, message := aggregateHealth(statuses)
	now := time.Now()
	return s.db.WithContext(ctx).Model(&model.Application{}).Where("id = ?", appID).Updates(map[string]interface{}{
		"health_status":     status,
		"health_checked_at": &now,
		"health_message":    message,
		"health_source":     "asset-aggregation",
	}).Error
}

func aggregateHealth(statuses []string) (string, string) {
	counts := map[string]int{"healthy": 0, "warning": 0, "unhealthy": 0, "unknown": 0, "checking": 0}
	for _, status := range statuses {
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "healthy", "active", "headless", "normal", "success":
			counts["healthy"]++
		case "warning":
			counts["warning"]++
		case "unhealthy", "error", "down", "failed":
			counts["unhealthy"]++
		case "checking", "running":
			counts["checking"]++
		default:
			counts["unknown"]++
		}
	}
	if len(statuses) == 0 {
		return "unknown", "暂无可探测资产"
	}
	message := fmt.Sprintf("共 %d 项：健康 %d，关注 %d，异常 %d，未知 %d", len(statuses), counts["healthy"], counts["warning"], counts["unhealthy"], counts["unknown"]+counts["checking"])
	if counts["unhealthy"] > 0 {
		return "unhealthy", message
	}
	if counts["warning"] > 0 || (counts["healthy"] > 0 && (counts["unknown"] > 0 || counts["checking"] > 0)) {
		return "warning", message
	}
	if counts["checking"] > 0 {
		return "checking", message
	}
	if counts["healthy"] > 0 && counts["unknown"] == 0 {
		return "healthy", message
	}
	return "unknown", message
}

func elapsedMilliseconds(started time.Time) int {
	value := int(time.Since(started).Milliseconds())
	if value < 1 {
		return 1
	}
	return value
}

func truncateProbeMessage(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > 480 {
		return string(runes[:480])
	}
	return value
}
