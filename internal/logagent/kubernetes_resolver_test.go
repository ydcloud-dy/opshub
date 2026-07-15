package logagent

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	appslisters "k8s.io/client-go/listers/apps/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

func TestKubernetesResolverEnrichesDeploymentMetadata(t *testing.T) {
	controller := true
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "api-7d9c8", Namespace: "production", UID: types.UID("pod-uid"),
		Labels:          map[string]string{"app.kubernetes.io/name": "api", "environment": "production", "private": "omit"},
		OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "api-7d9c8", Controller: &controller}},
	}, Spec: corev1.PodSpec{
		NodeName: "worker-01", Containers: []corev1.Container{{Name: "api", Image: "registry/api:v1"}},
	}}
	replicaSet := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
		Name: "api-7d9c8", Namespace: "production",
		OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "api", Controller: &controller}},
	}}
	podIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	replicaSetIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	jobIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	_ = podIndexer.Add(pod)
	_ = replicaSetIndexer.Add(replicaSet)
	resolver := &KubernetesResolver{
		pods: corelisters.NewPodLister(podIndexer), replicaSets: appslisters.NewReplicaSetLister(replicaSetIndexer),
		jobs: batchlisters.NewJobLister(jobIndexer),
	}
	metadata, selected, err := resolver.Resolve("/var/log/containers/api-7d9c8_production_api-aabbcc.log", KubernetesSourceConfig{
		ClusterID: 3, ClusterName: "prod", LabelAllowlist: []string{"app.kubernetes.io/name", "environment"},
		ServiceLabelKeys: []string{"app.kubernetes.io/name"}, EnvironmentLabelKeys: []string{"environment"},
		Selectors: []KubernetesSelector{{Namespace: "production", WorkloadKind: "Deployment", WorkloadName: "api", LabelSelector: "app.kubernetes.io/name=api", ContainerInclude: []string{"api"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !selected || metadata.WorkloadKind != "Deployment" || metadata.WorkloadName != "api" {
		t.Fatalf("unexpected metadata: selected=%v metadata=%#v", selected, metadata)
	}
	if metadata.PodUID != "pod-uid" || metadata.ContainerImage != "registry/api:v1" || metadata.Service != "api" {
		t.Fatalf("unexpected pod metadata: %#v", metadata)
	}
	if _, exists := metadata.Labels["private"]; exists {
		t.Fatalf("non-allowlisted label leaked: %#v", metadata.Labels)
	}
}

func TestParseKubernetesContainerLogPath(t *testing.T) {
	pod, namespace, container, ok := parseKubernetesContainerLogPath("/var/log/containers/frontend-abc_default_web-nginx-0123456789abcdef.log")
	if !ok || pod != "frontend-abc" || namespace != "default" || container != "web-nginx" {
		t.Fatalf("parsed = %q %q %q %v", pod, namespace, container, ok)
	}
}
