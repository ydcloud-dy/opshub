package logagent

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type KubernetesResolver struct {
	pods        corelisters.PodLister
	replicaSets appslisters.ReplicaSetLister
	jobs        batchlisters.JobLister
}

func NewKubernetesResolver(ctx context.Context, client kubernetes.Interface, nodeName string) (*KubernetesResolver, error) {
	if client == nil {
		return nil, fmt.Errorf("Kubernetes client 不能为空")
	}
	nodeName = strings.TrimSpace(nodeName)
	if nodeName == "" {
		return nil, fmt.Errorf("Kubernetes 节点名称不能为空")
	}
	podFactory := informers.NewSharedInformerFactoryWithOptions(client, 10*time.Minute, informers.WithTweakListOptions(func(options *metav1.ListOptions) {
		options.FieldSelector = fields.OneTermEqualSelector("spec.nodeName", nodeName).String()
	}))
	ownerFactory := informers.NewSharedInformerFactory(client, 10*time.Minute)
	podInformer := podFactory.Core().V1().Pods()
	replicaSetInformer := ownerFactory.Apps().V1().ReplicaSets()
	jobInformer := ownerFactory.Batch().V1().Jobs()
	podFactory.Start(ctx.Done())
	ownerFactory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), podInformer.Informer().HasSynced, replicaSetInformer.Informer().HasSynced, jobInformer.Informer().HasSynced) {
		return nil, fmt.Errorf("等待 Kubernetes 元数据缓存同步失败")
	}
	return &KubernetesResolver{
		pods: podInformer.Lister(), replicaSets: replicaSetInformer.Lister(), jobs: jobInformer.Lister(),
	}, nil
}

func (resolver *KubernetesResolver) Resolve(path string, config KubernetesSourceConfig) (KubernetesMetadata, bool, error) {
	podName, namespace, containerName, ok := parseKubernetesContainerLogPath(path)
	if !ok {
		return KubernetesMetadata{}, false, nil
	}
	metadata := KubernetesMetadata{
		ClusterID: config.ClusterID, ClusterName: config.ClusterName,
		Namespace: namespace, PodName: podName, ContainerName: containerName,
		Labels: make(map[string]string), Annotations: make(map[string]string),
	}
	pod, err := resolver.pods.Pods(namespace).Get(podName)
	if err == nil {
		metadata.PodUID = string(pod.UID)
		metadata.NodeName = pod.Spec.NodeName
		if owner := metav1.GetControllerOf(pod); owner == nil {
			metadata.WorkloadKind, metadata.WorkloadName = "Pod", pod.Name
		} else {
			metadata.WorkloadKind, metadata.WorkloadName = resolver.resolveWorkload(namespace, owner)
		}
		for _, container := range pod.Spec.Containers {
			if container.Name == containerName {
				metadata.ContainerImage = container.Image
				break
			}
		}
		metadata.Labels = cloneStringMap(pod.Labels)
		metadata.Annotations = cloneStringMap(pod.Annotations)
		metadata.Service = firstLabelValue(pod.Labels, config.ServiceLabelKeys)
		metadata.Environment = firstLabelValue(pod.Labels, config.EnvironmentLabelKeys)
	}
	selected, err := kubernetesMetadataSelected(metadata, config)
	if err != nil || !selected {
		return metadata, selected, err
	}
	metadata.Labels = selectMetadataValues(metadata.Labels, config.LabelAllowlist)
	metadata.Annotations = selectMetadataValues(metadata.Annotations, config.AnnotationAllowlist)
	metadata.ResourceAttributes = map[string]string{
		"k8s.cluster.id": strconvUint(config.ClusterID), "k8s.cluster.name": config.ClusterName,
		"k8s.namespace.name": metadata.Namespace, "k8s.pod.name": metadata.PodName,
		"k8s.pod.uid": metadata.PodUID, "k8s.container.name": metadata.ContainerName,
		"k8s.container.image": metadata.ContainerImage, "k8s.node.name": metadata.NodeName,
		"k8s.workload.kind": metadata.WorkloadKind, "k8s.workload.name": metadata.WorkloadName,
	}
	return metadata, true, nil
}

func (resolver *KubernetesResolver) resolveWorkload(namespace string, owner *metav1.OwnerReference) (string, string) {
	if owner == nil {
		return "Pod", ""
	}
	kind, name := owner.Kind, owner.Name
	switch strings.ToLower(kind) {
	case "replicaset":
		if replicaSet, err := resolver.replicaSets.ReplicaSets(namespace).Get(name); err == nil {
			if parent := metav1.GetControllerOf(replicaSet); parent != nil {
				return parent.Kind, parent.Name
			}
		}
	case "job":
		if job, err := resolver.jobs.Jobs(namespace).Get(name); err == nil {
			if parent := metav1.GetControllerOf(job); parent != nil {
				return parent.Kind, parent.Name
			}
		}
	}
	return kind, name
}

func parseKubernetesContainerLogPath(path string) (string, string, string, bool) {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	parts := strings.SplitN(base, "_", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	container := parts[2]
	if separator := strings.LastIndex(container, "-"); separator > 0 {
		container = container[:separator]
	}
	if container == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], container, true
}

func kubernetesMetadataSelected(metadata KubernetesMetadata, config KubernetesSourceConfig) (bool, error) {
	if stringInList(metadata.Namespace, config.ExcludeNamespaces) && !hasExplicitNamespace(metadata.Namespace, config.Selectors) {
		return false, nil
	}
	for _, selector := range config.Selectors {
		if selector.Namespace != "" && selector.Namespace != metadata.Namespace {
			continue
		}
		if selector.WorkloadKind != "" && !strings.EqualFold(selector.WorkloadKind, metadata.WorkloadKind) {
			continue
		}
		if selector.WorkloadName != "" && selector.WorkloadName != metadata.WorkloadName {
			continue
		}
		if selector.LabelSelector != "" {
			parsed, err := labels.Parse(selector.LabelSelector)
			if err != nil {
				return false, fmt.Errorf("Pod 标签选择器无效: %w", err)
			}
			if !parsed.Matches(labels.Set(metadata.Labels)) {
				continue
			}
		}
		if len(selector.ContainerInclude) > 0 && !matchesAnyName(metadata.ContainerName, selector.ContainerInclude) {
			continue
		}
		if matchesAnyName(metadata.ContainerName, selector.ContainerExclude) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func selectMetadataValues(values map[string]string, allowlist []string) map[string]string {
	result := make(map[string]string)
	for key, value := range values {
		if matchesAnyName(key, allowlist) {
			result[key] = value
		}
	}
	return result
}

func firstLabelValue(values map[string]string, keys []string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func matchesAnyName(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == value {
			return true
		}
		if matched, err := filepath.Match(pattern, value); err == nil && matched {
			return true
		}
	}
	return false
}

func stringInList(value string, values []string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func hasExplicitNamespace(namespace string, selectors []KubernetesSelector) bool {
	for _, selector := range selectors {
		if selector.Namespace == namespace {
			return true
		}
	}
	return false
}

func strconvUint(value uint64) string {
	return fmt.Sprintf("%d", value)
}
