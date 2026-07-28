package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/app-inventory/model"
	k8sservice "github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubernetesDiscoveryRequest struct {
	ClusterID     uint   `json:"clusterId"`
	ApplicationID uint   `json:"applicationId"`
	EnvironmentID uint   `json:"-"`
	Namespace     string `json:"namespace"`
	Selector      string `json:"selector"`
	CandidateKey  string `json:"candidateKey"`
	AppName       string `json:"appName"`
}

type KubernetesNamespaceOption struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type DiscoveredResource struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Address    string `json:"address"`
	Port       int    `json:"port"`
	ExternalID string `json:"externalId"`
	Status     string `json:"status"`
	Metadata   string `json:"metadata"`
}

type DiscoveredCandidate struct {
	Key         string               `json:"key"`
	Name        string               `json:"name"`
	Namespace   string               `json:"namespace"`
	ResourceCnt int                  `json:"resourceCount"`
	Domains     []string             `json:"domains"`
	Resources   []DiscoveredResource `json:"resources"`
}

type DiscoveryPreview struct {
	ClusterID uint                  `json:"clusterId"`
	Items     []DiscoveredCandidate `json:"items"`
	ScannedAt time.Time             `json:"scannedAt"`
}

func (s *Service) DiscoverKubernetes(ctx context.Context, clusterID, userID uint, namespace, selector string) (*DiscoveryPreview, error) {
	if clusterID == 0 {
		return nil, errors.New("Kubernetes 集群不能为空")
	}
	clusterService := k8sservice.NewClusterService(s.db)
	clientset, err := clusterService.GetClientsetForUser(ctx, clusterID, userID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	selector = strings.TrimSpace(selector)
	candidates := map[string]*DiscoveredCandidate{}
	getCandidate := func(name, namespace string, labels map[string]string) *DiscoveredCandidate {
		candidateName := discoveryName(name, labels)
		mapKey := namespace + "/" + candidateName
		if candidates[mapKey] == nil {
			candidates[mapKey] = &DiscoveredCandidate{Key: mapKey, Name: candidateName, Namespace: namespace, Resources: make([]DiscoveredResource, 0), Domains: make([]string, 0)}
		}
		return candidates[mapKey]
	}
	appendResource := func(candidate *DiscoveredCandidate, resource DiscoveredResource) {
		candidate.Resources = append(candidate.Resources, resource)
		candidate.ResourceCnt = len(candidate.Resources)
	}

	deployments, err := clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("扫描 Deployment 失败: %w", err)
	}
	for _, item := range deployments.Items {
		candidate := getCandidate(item.Name, item.Namespace, item.Labels)
		appendResource(candidate, workloadResource("Deployment", item.Name, item.Namespace, item.Status.AvailableReplicas > 0, item.Spec.Template.Spec.Containers, item.UID))
	}
	statefulSets, err := clientset.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("扫描 StatefulSet 失败: %w", err)
	}
	for _, item := range statefulSets.Items {
		candidate := getCandidate(item.Name, item.Namespace, item.Labels)
		appendResource(candidate, workloadResource("StatefulSet", item.Name, item.Namespace, item.Status.ReadyReplicas > 0, item.Spec.Template.Spec.Containers, item.UID))
	}
	daemonSets, err := clientset.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("扫描 DaemonSet 失败: %w", err)
	}
	for _, item := range daemonSets.Items {
		candidate := getCandidate(item.Name, item.Namespace, item.Labels)
		appendResource(candidate, workloadResource("DaemonSet", item.Name, item.Namespace, item.Status.NumberReady > 0, item.Spec.Template.Spec.Containers, item.UID))
	}
	services, err := clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("扫描 Service 失败: %w", err)
	}
	for _, item := range services.Items {
		labels := item.Labels
		if len(labels) == 0 {
			labels = item.Spec.Selector
		}
		candidate := getCandidate(item.Name, item.Namespace, labels)
		port := 0
		if len(item.Spec.Ports) > 0 {
			port = int(item.Spec.Ports[0].Port)
		}
		status := "healthy"
		if item.Spec.ClusterIP == corev1.ClusterIPNone {
			status = "warning"
		}
		metadata, _ := json.Marshal(map[string]interface{}{"type": string(item.Spec.Type), "selector": item.Spec.Selector, "ports": item.Spec.Ports})
		appendResource(candidate, DiscoveredResource{Kind: "Service", Name: item.Name, Namespace: item.Namespace, Address: item.Spec.ClusterIP, Port: port, ExternalID: string(item.UID), Status: status, Metadata: string(metadata)})
	}
	ingresses, err := clientset.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("扫描 Ingress 失败: %w", err)
	}
	for _, item := range ingresses.Items {
		candidate := getCandidate(item.Name, item.Namespace, item.Labels)
		metadata, _ := json.Marshal(map[string]interface{}{"class": ingressClass(item), "tls": item.Spec.TLS})
		for _, rule := range item.Spec.Rules {
			if strings.TrimSpace(rule.Host) == "" {
				continue
			}
			candidate.Domains = appendUnique(candidate.Domains, rule.Host)
		}
		address := ""
		if len(item.Status.LoadBalancer.Ingress) > 0 {
			address = item.Status.LoadBalancer.Ingress[0].IP
			if address == "" {
				address = item.Status.LoadBalancer.Ingress[0].Hostname
			}
		}
		appendResource(candidate, DiscoveredResource{Kind: "Ingress", Name: item.Name, Namespace: item.Namespace, Address: address, Port: 443, ExternalID: string(item.UID), Status: "healthy", Metadata: string(metadata)})
	}

	items := make([]DiscoveredCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, *candidate)
	}
	sortCandidates(items)
	return &DiscoveryPreview{ClusterID: clusterID, Items: items, ScannedAt: time.Now()}, nil
}

func (s *Service) ListKubernetesNamespaces(ctx context.Context, clusterID, userID uint) ([]KubernetesNamespaceOption, error) {
	if clusterID == 0 {
		return nil, errors.New("Kubernetes 集群不能为空")
	}
	clientset, err := k8sservice.NewClusterService(s.db).GetClientsetForUser(ctx, clusterID, userID)
	if err != nil {
		return nil, err
	}
	items, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("读取命名空间失败: %w", err)
	}
	result := make([]KubernetesNamespaceOption, 0, len(items.Items))
	for _, item := range items.Items {
		result = append(result, KubernetesNamespaceOption{Name: item.Name, Status: string(item.Status.Phase)})
	}
	return result, nil
}

func (s *Service) ImportKubernetes(ctx context.Context, req *KubernetesDiscoveryRequest, userID uint) (map[string]interface{}, error) {
	if req.ApplicationID == 0 {
		return nil, errors.New("导入前必须选择应用")
	}
	environmentID, err := s.applicationEnvironmentID(ctx, req.ApplicationID)
	if err != nil {
		return nil, err
	}
	req.EnvironmentID = environmentID
	started := time.Now()
	run := &model.DiscoveryRun{SourceType: "kubernetes", SourceID: req.ClusterID, ApplicationID: req.ApplicationID, EnvironmentID: req.EnvironmentID, Namespace: strings.TrimSpace(req.Namespace), Selector: strings.TrimSpace(req.Selector), Status: "running", CreatedBy: userID, StartedAt: started}
	if err := s.db.WithContext(ctx).Create(run).Error; err != nil {
		return nil, err
	}
	preview, err := s.DiscoverKubernetes(ctx, req.ClusterID, userID, req.Namespace, req.Selector)
	if err != nil {
		run.Status, run.ErrorMessage, run.FinishedAt = "failed", err.Error(), time.Now()
		_ = s.db.WithContext(ctx).Save(run).Error
		return nil, err
	}
	selected := preview.Items
	if strings.TrimSpace(req.CandidateKey) != "" {
		selected = nil
		for _, item := range preview.Items {
			if item.Key == strings.TrimSpace(req.CandidateKey) {
				selected = append(selected, item)
			}
		}
		if len(selected) == 0 {
			notFoundErr := fmt.Errorf("未找到 Kubernetes 资源组 %q", req.CandidateKey)
			run.Status, run.ErrorMessage, run.FinishedAt = "failed", notFoundErr.Error(), time.Now()
			_ = s.db.WithContext(ctx).Save(run).Error
			return nil, notFoundErr
		}
	} else if strings.TrimSpace(req.AppName) != "" {
		selected = nil
		for _, item := range preview.Items {
			if item.Name == strings.TrimSpace(req.AppName) {
				selected = append(selected, item)
			}
		}
		if len(selected) == 0 {
			notFoundErr := fmt.Errorf("未找到应用 %q 的 Kubernetes 资源", req.AppName)
			run.Status, run.ErrorMessage, run.FinishedAt = "failed", notFoundErr.Error(), time.Now()
			_ = s.db.WithContext(ctx).Save(run).Error
			return nil, notFoundErr
		}
	}
	resourceIDs := make(map[uint]struct{})
	domainIDs := make(map[uint]struct{})
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, candidate := range selected {
			for _, discovered := range candidate.Resources {
				var item model.Resource
				err := tx.Where("application_id = ? AND environment_id = ? AND cluster_id = ? AND namespace = ? AND kind = ? AND name = ?", req.ApplicationID, req.EnvironmentID, req.ClusterID, discovered.Namespace, discovered.Kind, discovered.Name).First(&item).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					now := time.Now()
					item = model.Resource{ApplicationID: req.ApplicationID, EnvironmentID: req.EnvironmentID, Kind: discovered.Kind, Name: discovered.Name, Address: discovered.Address, Port: discovered.Port, ClusterID: req.ClusterID, Namespace: discovered.Namespace, ExternalID: discovered.ExternalID, Status: discovered.Status, Source: "kubernetes", Metadata: discovered.Metadata, LastSyncedAt: &now}
					if err := tx.Create(&item).Error; err != nil {
						return err
					}
				} else if err != nil {
					return err
				} else {
					item.Address, item.Port, item.ExternalID, item.Status, item.Metadata, item.Source = discovered.Address, discovered.Port, discovered.ExternalID, discovered.Status, discovered.Metadata, "kubernetes"
					now := time.Now()
					item.LastSyncedAt = &now
					if err := tx.Save(&item).Error; err != nil {
						return err
					}
				}
				resourceIDs[item.ID] = struct{}{}
			}
			for _, domainName := range candidate.Domains {
				var domain model.Domain
				err := tx.Where("application_id = ? AND environment_id = ? AND domain = ?", req.ApplicationID, req.EnvironmentID, domainName).First(&domain).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					var primaryCount int64
					if err := tx.Model(&model.Domain{}).Where("application_id = ? AND environment_id = ? AND is_primary = ?", req.ApplicationID, req.EnvironmentID, true).Count(&primaryCount).Error; err != nil {
						return err
					}
					domain = model.Domain{ApplicationID: req.ApplicationID, EnvironmentID: req.EnvironmentID, Domain: domainName, Protocol: "https", Port: 443, Path: "/", IsPrimary: primaryCount == 0, Status: "checking", Source: "kubernetes", ProbeMessage: "等待自动探测"}
					if err := tx.Create(&domain).Error; err != nil {
						return err
					}
				} else if err != nil {
					return err
				} else {
					domain.Status, domain.ProbeMessage = "checking", "等待自动探测"
					if domain.Source == "" {
						domain.Source = "manual"
					}
					if err := tx.Save(&domain).Error; err != nil {
						return err
					}
				}
				domainIDs[domain.ID] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		run.Status, run.ErrorMessage, run.FinishedAt = "failed", err.Error(), time.Now()
		_ = s.db.WithContext(ctx).Save(run).Error
		return nil, err
	}
	for id := range domainIDs {
		_, _ = s.probeDomain(ctx, id, false)
	}
	resourceCount, domainCount := len(resourceIDs), len(domainIDs)
	run.Status, run.ResourceCount, run.DomainCount, run.FinishedAt = "success", resourceCount, domainCount, time.Now()
	if err := s.db.WithContext(ctx).Save(run).Error; err != nil {
		return nil, err
	}
	_ = s.RecomputeApplicationHealth(ctx, req.ApplicationID)
	return map[string]interface{}{"runId": run.ID, "resourceCount": resourceCount, "domainCount": domainCount, "candidateCount": len(selected)}, nil
}

func (s *Service) ListDiscoveryRuns(ctx context.Context, appID uint) ([]model.DiscoveryRun, error) {
	var items []model.DiscoveryRun
	q := s.db.WithContext(ctx).Order("id DESC").Limit(50)
	if appID > 0 {
		q = q.Where("application_id = ?", appID)
	}
	return items, q.Find(&items).Error
}

func workloadResource(kind, name, namespace string, ready bool, containers []corev1.Container, uid interface{}) DiscoveredResource {
	status := "unhealthy"
	if ready {
		status = "healthy"
	}
	images := make([]string, 0, len(containers))
	for _, container := range containers {
		images = append(images, container.Image)
	}
	metadata, _ := json.Marshal(map[string]interface{}{"images": images})
	return DiscoveredResource{Kind: kind, Name: name, Namespace: namespace, ExternalID: fmt.Sprint(uid), Status: status, Metadata: string(metadata)}
}

func discoveryName(fallback string, labels map[string]string) string {
	for _, key := range []string{"app.kubernetes.io/name", "app", "k8s-app", "app.kubernetes.io/instance"} {
		if name := strings.TrimSpace(labels[key]); name != "" {
			return name
		}
	}
	return fallback
}

func ingressClass(item networkingv1.Ingress) string {
	if item.Spec.IngressClassName != nil {
		return *item.Spec.IngressClassName
	}
	return ""
}

func appendUnique(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func sortCandidates(items []DiscoveredCandidate) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if strings.ToLower(items[j].Name) < strings.ToLower(items[i].Name) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
