package server

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/ydcloud-dy/opshub/internal/logagent"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildKubernetesCollectorResourcesIncludesRuntimeMountsAndRBAC(t *testing.T) {
	resources := buildKubernetesCollectorResources(7, "collector-token", "https://opshub.example.com", "example/opshub:v1", "ignored")
	if resources.Namespace.Name != kubernetesCollectorNamespace || resources.DaemonSet.Namespace != kubernetesCollectorNamespace {
		t.Fatalf("unexpected namespace: %s", resources.DaemonSet.Namespace)
	}
	container := resources.DaemonSet.Spec.Template.Spec.Containers[0]
	if container.Image != "example/opshub:v1" || len(container.VolumeMounts) != 3 {
		t.Fatalf("unexpected collector container: %#v", container)
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
	manifest, err := marshalKubernetesResources(resources)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"kind: DaemonSet", "kind: ClusterRole", "cluster-token: collector-token", "/var/log", "/var/lib/docker/containers", "/usr/local/bin/opshub-agent", "does not support kubernetes-node mode"} {
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

func TestValidateKubernetesPolicyPayload(t *testing.T) {
	payload := policyPayload{
		Name: "production containers", SourceMode: "kubernetes", ReadFrom: "latest", MaxLineBytes: 262144,
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

func parserConfigRaw() logagent.ParserConfig {
	return logagent.ParserConfig{Type: "raw"}
}
