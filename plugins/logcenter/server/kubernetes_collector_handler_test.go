package server

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/ydcloud-dy/opshub/internal/logagent"
	k8smodel "github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestBuildKubernetesCollectorResourcesIncludesRuntimeMountsAndRBAC(t *testing.T) {
	resources := buildKubernetesCollectorResources(7, "collector-token", "https://opshub.example.com", "example/opshub:v1", "ignored")
	if resources.Namespace.Name != kubernetesCollectorNamespace || resources.DaemonSet.Namespace != kubernetesCollectorNamespace {
		t.Fatalf("unexpected namespace: %s", resources.DaemonSet.Namespace)
	}
	container := resources.DaemonSet.Spec.Template.Spec.Containers[0]
	if container.Image != "example/opshub:v1" || len(container.VolumeMounts) != 4 {
		t.Fatalf("unexpected collector container: %#v", container)
	}
	if len(resources.DaemonSet.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatalf("collector must download current agent binary before startup: %#v", resources.DaemonSet.Spec.Template.Spec.InitContainers)
	}
	initContainer := resources.DaemonSet.Spec.Template.Spec.InitContainers[0]
	if initContainer.Name != "download-agent" || len(initContainer.VolumeMounts) != 1 {
		t.Fatalf("unexpected download init container: %#v", initContainer)
	}
	if container.ImagePullPolicy != corev1.PullAlways {
		t.Fatalf("unexpected image pull policy: %s", container.ImagePullPolicy)
	}
	if container.StartupProbe == nil || container.StartupProbe.FailureThreshold < 24 {
		t.Fatalf("collector startup probe must allow metadata cache initialization: %#v", container.StartupProbe)
	}
	if len(resources.ClusterRole.Rules) != 3 {
		t.Fatalf("unexpected RBAC rules: %#v", resources.ClusterRole.Rules)
	}
	if resources.DaemonSet.Spec.Template.Annotations["opshub.io/config-checksum"] == "" {
		t.Fatal("rollout checksum missing")
	}
	if resources.DaemonSet.Spec.Template.Annotations["opshub.io/agent-version"] != kubernetesCollectorVersion {
		t.Fatalf("unexpected agent version annotation: %#v", resources.DaemonSet.Spec.Template.Annotations)
	}
	manifest, err := marshalKubernetesResources(resources)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"kind: DaemonSet", "kind: ClusterRole", "cluster-token: collector-token", "/var/log", "/var/lib/docker/containers", "download-agent", "/api/v1/public/agents/binaries/opshub-agent-linux-${ARCH}", "/opshub-agent-bin/opshub-agent", "does not support kubernetes-node mode"} {
		if !strings.Contains(manifest, expected) {
			t.Fatalf("manifest missing %q", expected)
		}
	}
}

func TestCollectorEntrypointScriptSyntax(t *testing.T) {
	command := exec.Command("sh", "-n")
	command.Stdin = strings.NewReader(collectorEntrypointScript())
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("invalid collector entrypoint: %v: %s", err, output)
	}
}

func TestCollectorDownloadScriptSyntax(t *testing.T) {
	command := exec.Command("sh", "-n")
	command.Stdin = strings.NewReader(collectorDownloadScript())
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("invalid collector download script: %v: %s", err, output)
	}
}

func TestValidateCollectorServerURLRejectsLoopback(t *testing.T) {
	for _, value := range []string{"http://localhost:5173", "http://127.0.0.1:9876", "http://[::1]:9876", "http://0.0.0.0:9876"} {
		if _, err := validateCollectorServerURL(value); err == nil {
			t.Fatalf("expected loopback URL %q to be rejected", value)
		}
	}
	value, err := validateCollectorServerURL("https://opshub.example.com/")
	if err != nil || value != "https://opshub.example.com" {
		t.Fatalf("valid collector URL was not normalized: value=%q err=%v", value, err)
	}
}

func TestKubernetesCollectorPodError(t *testing.T) {
	pods := []corev1.Pod{{
		ObjectMeta: metav1.ObjectMeta{Name: "opshub-log-agent-test"},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			Name: kubernetesCollectorName,
			State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{
				Reason: "CrashLoopBackOff", Message: "back-off restarting failed container",
			}},
		}}},
	}}
	message := kubernetesCollectorPodError(pods)
	if !strings.Contains(message, "opshub-log-agent-test") || !strings.Contains(message, "CrashLoopBackOff") {
		t.Fatalf("unexpected pod error: %q", message)
	}
}

func TestDeleteKubernetesCollectorResources(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName, Namespace: kubernetesCollectorNamespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName, Namespace: kubernetesCollectorNamespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName, Namespace: kubernetesCollectorNamespace}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName, Namespace: kubernetesCollectorNamespace}},
		&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName}},
	)
	if err := deleteKubernetesCollectorResources(context.Background(), clientset); err != nil {
		t.Fatal(err)
	}
	if _, err := clientset.AppsV1().DaemonSets(kubernetesCollectorNamespace).Get(context.Background(), kubernetesCollectorName, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("daemonset was not deleted: %v", err)
	}
	if _, err := clientset.CoreV1().Secrets(kubernetesCollectorNamespace).Get(context.Background(), kubernetesCollectorName, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("secret was not deleted: %v", err)
	}
	if _, err := clientset.CoreV1().ConfigMaps(kubernetesCollectorNamespace).Get(context.Background(), kubernetesCollectorName, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("configmap was not deleted: %v", err)
	}
	if _, err := clientset.CoreV1().ServiceAccounts(kubernetesCollectorNamespace).Get(context.Background(), kubernetesCollectorName, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("service account was not deleted: %v", err)
	}
	if _, err := clientset.RbacV1().ClusterRoles().Get(context.Background(), kubernetesCollectorName, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("cluster role was not deleted: %v", err)
	}
	if _, err := clientset.RbacV1().ClusterRoleBindings().Get(context.Background(), kubernetesCollectorName, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("cluster role binding was not deleted: %v", err)
	}
}

func TestShutdownUnusedKubernetesCollectors(t *testing.T) {
	clusters := []k8smodel.Cluster{{ID: 7, Name: "production", Alias: "生产集群"}}
	tests := []struct {
		name           string
		activePolicies int64
		uninstallError error
		expectedStatus string
		expectedCalls  int
	}{
		{name: "last policy uninstalls collector", expectedStatus: "uninstalled", expectedCalls: 1},
		{name: "shared collector is retained", activePolicies: 2, expectedStatus: "skipped", expectedCalls: 0},
		{name: "uninstall failure is reported", uninstallError: errors.New("delete failed"), expectedStatus: "failed", expectedCalls: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			uninstallCalls := 0
			results := shutdownUnusedKubernetesCollectors(
				context.Background(), 11, clusters,
				func(context.Context, uint, uint) (int64, error) { return test.activePolicies, nil },
				func(context.Context, uint) error { uninstallCalls++; return test.uninstallError },
			)
			if len(results) != 1 || results[0].Status != test.expectedStatus {
				t.Fatalf("unexpected shutdown results: %#v", results)
			}
			if uninstallCalls != test.expectedCalls {
				t.Fatalf("unexpected uninstall calls: got %d want %d", uninstallCalls, test.expectedCalls)
			}
		})
	}
}

func TestValidateKubernetesPolicyPayload(t *testing.T) {
	payload := policyPayload{
		Name: "production containers", SourceMode: "kubernetes", Environment: "prod", Service: "api", ReadFrom: "latest", MaxLineBytes: 262144,
		Parser: parserConfigRaw(), Targets: []policyTargetInput{{TargetType: "cluster", TargetID: 7, Namespace: "production", LabelSelector: "app=api"}},
	}
	payload.normalize()
	if err := validatePolicyPayload(payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Paths) != 1 || payload.Paths[0] != "/var/log/containers/*.log" {
		t.Fatalf("unexpected paths: %#v", payload.Paths)
	}
}

func TestValidatePolicyPayloadRequiresLogIdentity(t *testing.T) {
	payload := policyPayload{
		Name: "host logs", SourceMode: "host", Paths: []string{"/var/log/app.log"},
		Targets: []policyTargetInput{{TargetType: "host", TargetID: 1}},
	}
	payload.normalize()
	if err := validatePolicyPayload(payload); err == nil || !strings.Contains(err.Error(), "运行环境") {
		t.Fatalf("expected environment validation error, got %v", err)
	}
	payload.Environment = "test"
	if err := validatePolicyPayload(payload); err == nil || !strings.Contains(err.Error(), "服务名称") {
		t.Fatalf("expected service validation error, got %v", err)
	}
}

func parserConfigRaw() logagent.ParserConfig {
	return logagent.ParserConfig{Type: "raw"}
}
