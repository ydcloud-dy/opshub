package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/pkg/response"
	k8smodel "github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	k8sservice "github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

const (
	kubernetesCollectorNamespace = "opshub-system"
	kubernetesCollectorName      = "opshub-log-agent"
)

type kubernetesCollectorRequest struct {
	Namespace string `json:"namespace"`
	Image     string `json:"image"`
	ServerURL string `json:"serverUrl"`
}

type kubernetesWorkloadOption struct {
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
}

func (h *Handler) GetKubernetesPolicyOptions(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	clusterID := parseUint(c.Param("id"))
	clientset, err := k8sservice.NewClusterService(h.db).GetCachedClientset(c.Request.Context(), clusterID)
	if err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "连接 Kubernetes 集群失败: "+err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "读取 Namespace 失败: "+err.Error())
		return
	}
	workloads, err := listKubernetesWorkloadOptions(ctx, clientset)
	if err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "读取 Workload 失败: "+err.Error())
		return
	}
	namespaceNames := make([]string, 0, len(namespaces.Items))
	for _, namespace := range namespaces.Items {
		namespaceNames = append(namespaceNames, namespace.Name)
	}
	sort.Strings(namespaceNames)
	response.Success(c, gin.H{"namespaces": namespaceNames, "workloads": workloads})
}

func (h *Handler) GetKubernetesCollectorStatus(c *gin.Context) {
	clusterID, cluster, ok := h.kubernetesCollectorCluster(c)
	if !ok {
		return
	}
	now := time.Now()
	var credential logmodel.ClusterCollectorCredential
	credentialConfigured := h.db.WithContext(c.Request.Context()).Where("cluster_id = ? AND status = ?", clusterID, "active").First(&credential).Error == nil
	var instances []logmodel.CollectorInstance
	h.db.WithContext(c.Request.Context()).Where("mode = ? AND cluster_id = ?", "kubernetes-node", clusterID).Order("node_name ASC").Find(&instances)
	online := 0
	for index := range instances {
		if instances[index].LastHeartbeatAt != nil && now.Sub(*instances[index].LastHeartbeatAt) <= 90*time.Second {
			online++
		} else {
			instances[index].Status = "offline"
		}
	}
	var policyCount int64
	h.db.Model(&logmodel.CollectionPolicy{}).Where("source_mode = ? AND status = ? AND id IN (?)", "kubernetes", "published",
		h.db.Model(&logmodel.PolicyTarget{}).Select("policy_id").Where("target_type = ? AND target_id = ?", "cluster", clusterID)).Count(&policyCount)
	installed := false
	desired, ready, available := int32(0), int32(0), int32(0)
	lastError := ""
	collectorImage := ""
	imagePullPolicy := ""
	clientset, err := k8sservice.NewClusterService(h.db).GetCachedClientset(c.Request.Context(), clusterID)
	if err == nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		daemonSet, getErr := clientset.AppsV1().DaemonSets(kubernetesCollectorNamespace).Get(ctx, kubernetesCollectorName, metav1.GetOptions{})
		if getErr == nil {
			installed = true
			desired, ready, available = daemonSet.Status.DesiredNumberScheduled, daemonSet.Status.NumberReady, daemonSet.Status.NumberAvailable
			if len(daemonSet.Spec.Template.Spec.Containers) > 0 {
				collectorImage = daemonSet.Spec.Template.Spec.Containers[0].Image
				imagePullPolicy = string(daemonSet.Spec.Template.Spec.Containers[0].ImagePullPolicy)
			}
			pods, listErr := clientset.CoreV1().Pods(kubernetesCollectorNamespace).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/name=" + kubernetesCollectorName})
			if listErr != nil {
				lastError = listErr.Error()
			} else {
				lastError = kubernetesCollectorPodError(pods.Items)
			}
		} else if !apierrors.IsNotFound(getErr) {
			lastError = getErr.Error()
		}
	} else {
		lastError = err.Error()
	}
	response.Success(c, gin.H{
		"clusterId": clusterID, "clusterName": firstNonEmpty(cluster.Alias, cluster.Name),
		"installed": installed, "credentialConfigured": credentialConfigured, "tokenHint": credential.TokenHint,
		"desiredNodes": desired, "readyNodes": ready, "availableNodes": available,
		"instanceTotal": len(instances), "instanceOnline": online, "policyCount": policyCount,
		"instances": instances, "lastError": lastError, "image": collectorImage, "imagePullPolicy": imagePullPolicy,
	})
}

func kubernetesCollectorPodError(pods []corev1.Pod) string {
	for _, pod := range pods {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse && condition.Reason != "" {
				return fmt.Sprintf("%s: %s: %s", pod.Name, condition.Reason, condition.Message)
			}
		}
		statuses := append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
		for _, status := range statuses {
			if waiting := status.State.Waiting; waiting != nil && waiting.Reason != "" && waiting.Reason != "ContainerCreating" && waiting.Reason != "PodInitializing" {
				return strings.TrimSpace(fmt.Sprintf("%s: %s: %s", pod.Name, waiting.Reason, waiting.Message))
			}
			if terminated := status.State.Terminated; terminated != nil && terminated.ExitCode != 0 {
				return strings.TrimSpace(fmt.Sprintf("%s: %s (exit %d): %s", pod.Name, firstNonEmpty(terminated.Reason, "ContainerError"), terminated.ExitCode, terminated.Message))
			}
		}
	}
	return ""
}

func (h *Handler) GenerateKubernetesCollectorManifest(c *gin.Context) {
	clusterID, cluster, ok := h.kubernetesCollectorCluster(c)
	if !ok {
		return
	}
	request := parseKubernetesCollectorRequest(c)
	token, credential, err := h.rotateKubernetesCollectorCredential(c, clusterID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "生成集群采集 Token 失败: "+err.Error())
		return
	}
	resources := buildKubernetesCollectorResources(clusterID, token, resolveCollectorServerURL(c, request.ServerURL), resolveCollectorImage(request.Image), request.Namespace)
	manifest, err := marshalKubernetesResources(resources)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "生成 DaemonSet 清单失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{
		"clusterId": clusterID, "clusterName": firstNonEmpty(cluster.Alias, cluster.Name),
		"namespace": resources.Namespace.Name, "tokenHint": credential.TokenHint, "yaml": manifest,
		"warning": "生成清单会轮换集群采集 Token，请立即在目标集群应用该 YAML",
	})
}

func (h *Handler) InstallKubernetesCollector(c *gin.Context) {
	clusterID, cluster, ok := h.kubernetesCollectorCluster(c)
	if !ok {
		return
	}
	request := parseKubernetesCollectorRequest(c)
	clientset, err := k8sservice.NewClusterService(h.db).GetCachedClientset(c.Request.Context(), clusterID)
	if err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "连接 Kubernetes 集群失败: "+err.Error())
		return
	}
	token, err := newKubernetesCollectorToken()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "生成集群采集 Token 失败: "+err.Error())
		return
	}
	resources := buildKubernetesCollectorResources(clusterID, token, resolveCollectorServerURL(c, request.ServerURL), resolveCollectorImage(request.Image), request.Namespace)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	if err := applyKubernetesCollectorResources(ctx, clientset, resources); err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "安装 Kubernetes 日志采集器失败: "+err.Error())
		return
	}
	credential, err := h.saveKubernetesCollectorCredential(c, clusterID, token)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存集群采集 Token 失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{
		"clusterId": clusterID, "clusterName": firstNonEmpty(cluster.Alias, cluster.Name),
		"namespace": resources.Namespace.Name, "daemonSet": resources.DaemonSet.Name, "tokenHint": credential.TokenHint,
	})
}

func (h *Handler) UninstallKubernetesCollector(c *gin.Context) {
	clusterID, _, ok := h.kubernetesCollectorCluster(c)
	if !ok {
		return
	}
	clientset, err := k8sservice.NewClusterService(h.db).GetCachedClientset(c.Request.Context(), clusterID)
	if err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "连接 Kubernetes 集群失败: "+err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	deletePolicy := metav1.DeletePropagationBackground
	_ = clientset.AppsV1().DaemonSets(kubernetesCollectorNamespace).Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	_ = clientset.CoreV1().Secrets(kubernetesCollectorNamespace).Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{})
	_ = clientset.CoreV1().ConfigMaps(kubernetesCollectorNamespace).Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{})
	_ = clientset.CoreV1().ServiceAccounts(kubernetesCollectorNamespace).Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{})
	_ = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{})
	_ = clientset.RbacV1().ClusterRoles().Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{})
	h.db.WithContext(c.Request.Context()).Model(&logmodel.ClusterCollectorCredential{}).Where("cluster_id = ?", clusterID).Update("status", "revoked")
	h.db.WithContext(c.Request.Context()).Model(&logmodel.CollectorInstance{}).Where("mode = ? AND cluster_id = ?", "kubernetes-node", clusterID).Update("status", "offline")
	response.Success(c, gin.H{"clusterId": clusterID})
}

func (h *Handler) kubernetesCollectorCluster(c *gin.Context) (uint, k8smodel.Cluster, bool) {
	if !h.requirePolicyAdmin(c) {
		return 0, k8smodel.Cluster{}, false
	}
	clusterID := parseUint(c.Param("id"))
	var cluster k8smodel.Cluster
	if clusterID == 0 || h.db.WithContext(c.Request.Context()).First(&cluster, clusterID).Error != nil {
		response.ErrorCode(c, http.StatusNotFound, "Kubernetes 集群不存在")
		return 0, cluster, false
	}
	return clusterID, cluster, true
}

func (h *Handler) rotateKubernetesCollectorCredential(c *gin.Context, clusterID uint) (string, logmodel.ClusterCollectorCredential, error) {
	token, err := newKubernetesCollectorToken()
	if err != nil {
		return "", logmodel.ClusterCollectorCredential{}, err
	}
	credential, err := h.saveKubernetesCollectorCredential(c, clusterID, token)
	return token, credential, err
}

func newKubernetesCollectorToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (h *Handler) saveKubernetesCollectorCredential(c *gin.Context, clusterID uint, token string) (logmodel.ClusterCollectorCredential, error) {
	hash := sha256.Sum256([]byte(token))
	now := time.Now()
	credential := logmodel.ClusterCollectorCredential{ClusterID: clusterID}
	err := h.db.WithContext(c.Request.Context()).Where("cluster_id = ?", clusterID).Assign(map[string]any{
		"token_hash": hex.EncodeToString(hash[:]), "token_hint": token[:8], "status": "active", "rotated_at": &now,
	}).FirstOrCreate(&credential).Error
	return credential, err
}

type kubernetesCollectorResources struct {
	Namespace          *corev1.Namespace
	ServiceAccount     *corev1.ServiceAccount
	ClusterRole        *rbacv1.ClusterRole
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
	Secret             *corev1.Secret
	ConfigMap          *corev1.ConfigMap
	DaemonSet          *appsv1.DaemonSet
}

func buildKubernetesCollectorResources(clusterID uint, token, serverURL, image, namespace string) kubernetesCollectorResources {
	namespace = kubernetesCollectorNamespace
	labels := map[string]string{"app.kubernetes.io/name": kubernetesCollectorName, "app.kubernetes.io/component": "log-collector", "app.kubernetes.io/managed-by": "opshub"}
	rolloutHash := sha256.Sum256([]byte(token + "\x00" + serverURL + "\x00" + image))
	privileged := false
	readOnlyRoot := true
	runAsUser := int64(0)
	resources := kubernetesCollectorResources{
		Namespace:      &corev1.Namespace{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"}, ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{"app.kubernetes.io/managed-by": "opshub"}}},
		ServiceAccount: &corev1.ServiceAccount{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"}, ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName, Namespace: namespace, Labels: labels}},
		ClusterRole: &rbacv1.ClusterRole{TypeMeta: metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRole"}, ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName, Labels: labels}, Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{""}, Resources: []string{"pods", "namespaces", "nodes"}, Verbs: []string{"get", "list", "watch"}},
			{APIGroups: []string{"apps"}, Resources: []string{"replicasets", "deployments", "statefulsets", "daemonsets"}, Verbs: []string{"get", "list", "watch"}},
			{APIGroups: []string{"batch"}, Resources: []string{"jobs", "cronjobs"}, Verbs: []string{"get", "list", "watch"}},
		}},
		ClusterRoleBinding: &rbacv1.ClusterRoleBinding{TypeMeta: metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRoleBinding"}, ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName, Labels: labels}, RoleRef: rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: kubernetesCollectorName}, Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: kubernetesCollectorName, Namespace: namespace}}},
		Secret:             &corev1.Secret{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"}, ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName, Namespace: namespace, Labels: labels}, Type: corev1.SecretTypeOpaque, StringData: map[string]string{"cluster-token": token}},
		ConfigMap:          &corev1.ConfigMap{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"}, ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName, Namespace: namespace, Labels: labels}, Data: map[string]string{"server-url": serverURL, "cluster-id": strconv.FormatUint(uint64(clusterID), 10)}},
	}
	resources.DaemonSet = &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"}, ObjectMeta: metav1.ObjectMeta{Name: kubernetesCollectorName, Namespace: namespace, Labels: labels},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels}, UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: map[string]string{"opshub.io/config-checksum": hex.EncodeToString(rolloutHash[:])}}, Spec: corev1.PodSpec{
				ServiceAccountName: kubernetesCollectorName, AutomountServiceAccountToken: boolPointer(true),
				Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				Containers: []corev1.Container{{
					Name: kubernetesCollectorName, Image: image, ImagePullPolicy: corev1.PullAlways,
					Command: []string{"/bin/sh", "-ec"}, Args: []string{collectorEntrypointScript()},
					Env: []corev1.EnvVar{
						{Name: "OPSHUB_SERVER_URL", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: kubernetesCollectorName}, Key: "server-url"}}},
						{Name: "OPSHUB_CLUSTER_ID", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: kubernetesCollectorName}, Key: "cluster-id"}}},
						{Name: "OPSHUB_CLUSTER_TOKEN", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: kubernetesCollectorName}, Key: "cluster-token"}}},
						{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
						{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
						{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
					},
					Ports:           []corev1.ContainerPort{{Name: "metrics", ContainerPort: 19877}},
					SecurityContext: &corev1.SecurityContext{Privileged: &privileged, RunAsUser: &runAsUser, ReadOnlyRootFilesystem: &readOnlyRoot, AllowPrivilegeEscalation: boolPointer(false), Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m"), corev1.ResourceMemory: resource.MustParse("64Mi")},
						Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "varlog", MountPath: "/var/log", ReadOnly: true},
						{Name: "dockerlogs", MountPath: "/var/lib/docker/containers", ReadOnly: true},
						{Name: "state", MountPath: "/var/lib/opshub-log-agent"},
					},
					ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromString("metrics")}}, InitialDelaySeconds: 5, PeriodSeconds: 10},
					LivenessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromString("metrics")}}, InitialDelaySeconds: 20, PeriodSeconds: 20},
					StartupProbe: &corev1.Probe{
						ProbeHandler:  corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromString("metrics")}},
						PeriodSeconds: 5, TimeoutSeconds: 2, FailureThreshold: 30,
					},
				}},
				Volumes: []corev1.Volume{
					{Name: "varlog", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/log", Type: hostPathTypePointer(corev1.HostPathDirectory)}}},
					{Name: "dockerlogs", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/docker/containers", Type: hostPathTypePointer(corev1.HostPathDirectoryOrCreate)}}},
					{Name: "state", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/opshub-log-agent", Type: hostPathTypePointer(corev1.HostPathDirectoryOrCreate)}}},
				},
			}},
		},
	}
	return resources
}

func collectorEntrypointScript() string {
	return `AGENT=/usr/local/bin/opshub-agent
if [ ! -x "${AGENT}" ]; then
  case "$(uname -m)" in
    x86_64|amd64) ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
  AGENT=/app/data/agent-binaries/opshub-agent-linux-${ARCH}
fi
if [ ! -x "${AGENT}" ]; then
  echo "incompatible collector image: OpsHub Agent binary was not found" >&2
  exit 64
fi
if ! "${AGENT}" --help 2>&1 | grep -q -- '-mode'; then
  echo "incompatible collector image: OpsHub Agent does not support kubernetes-node mode; rebuild or upgrade opshub-log-agent" >&2
  exit 64
fi
exec "${AGENT}" \
  --mode kubernetes-node \
  --server "${OPSHUB_SERVER_URL}" \
  --cluster-id "${OPSHUB_CLUSTER_ID}" \
  --cluster-token "${OPSHUB_CLUSTER_TOKEN}" \
  --node-name "${NODE_NAME}" \
  --pod-name "${POD_NAME}" \
  --pod-namespace "${POD_NAMESPACE}" \
  --log-metrics-address "0.0.0.0:19877"`
}

func marshalKubernetesResources(resources kubernetesCollectorResources) (string, error) {
	objects := []any{resources.Namespace, resources.ServiceAccount, resources.ClusterRole, resources.ClusterRoleBinding, resources.Secret, resources.ConfigMap, resources.DaemonSet}
	parts := make([]string, 0, len(objects))
	for _, object := range objects {
		raw, err := yaml.Marshal(object)
		if err != nil {
			return "", err
		}
		parts = append(parts, strings.TrimSpace(string(raw)))
	}
	return strings.Join(parts, "\n---\n") + "\n", nil
}

func applyKubernetesCollectorResources(ctx context.Context, client kubernetes.Interface, resources kubernetesCollectorResources) error {
	if _, err := client.CoreV1().Namespaces().Get(ctx, resources.Namespace.Name, metav1.GetOptions{}); apierrors.IsNotFound(err) {
		if _, err := client.CoreV1().Namespaces().Create(ctx, resources.Namespace, metav1.CreateOptions{}); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := applyServiceAccount(ctx, client, resources.ServiceAccount); err != nil {
		return err
	}
	if err := applyClusterRole(ctx, client, resources.ClusterRole); err != nil {
		return err
	}
	if err := applyClusterRoleBinding(ctx, client, resources.ClusterRoleBinding); err != nil {
		return err
	}
	if err := applySecret(ctx, client, resources.Secret); err != nil {
		return err
	}
	if err := applyConfigMap(ctx, client, resources.ConfigMap); err != nil {
		return err
	}
	return applyDaemonSet(ctx, client, resources.DaemonSet)
}

func applyServiceAccount(ctx context.Context, client kubernetes.Interface, desired *corev1.ServiceAccount) error {
	existing, err := client.CoreV1().ServiceAccounts(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.CoreV1().ServiceAccounts(desired.Namespace).Create(ctx, desired, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	desired.ResourceVersion = existing.ResourceVersion
	_, err = client.CoreV1().ServiceAccounts(desired.Namespace).Update(ctx, desired, metav1.UpdateOptions{})
	return err
}

func applyClusterRole(ctx context.Context, client kubernetes.Interface, desired *rbacv1.ClusterRole) error {
	existing, err := client.RbacV1().ClusterRoles().Get(ctx, desired.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.RbacV1().ClusterRoles().Create(ctx, desired, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	desired.ResourceVersion = existing.ResourceVersion
	_, err = client.RbacV1().ClusterRoles().Update(ctx, desired, metav1.UpdateOptions{})
	return err
}

func applyClusterRoleBinding(ctx context.Context, client kubernetes.Interface, desired *rbacv1.ClusterRoleBinding) error {
	existing, err := client.RbacV1().ClusterRoleBindings().Get(ctx, desired.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, desired, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	desired.ResourceVersion = existing.ResourceVersion
	_, err = client.RbacV1().ClusterRoleBindings().Update(ctx, desired, metav1.UpdateOptions{})
	return err
}

func applySecret(ctx context.Context, client kubernetes.Interface, desired *corev1.Secret) error {
	existing, err := client.CoreV1().Secrets(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.CoreV1().Secrets(desired.Namespace).Create(ctx, desired, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	desired.ResourceVersion = existing.ResourceVersion
	_, err = client.CoreV1().Secrets(desired.Namespace).Update(ctx, desired, metav1.UpdateOptions{})
	return err
}

func applyConfigMap(ctx context.Context, client kubernetes.Interface, desired *corev1.ConfigMap) error {
	existing, err := client.CoreV1().ConfigMaps(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.CoreV1().ConfigMaps(desired.Namespace).Create(ctx, desired, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	desired.ResourceVersion = existing.ResourceVersion
	_, err = client.CoreV1().ConfigMaps(desired.Namespace).Update(ctx, desired, metav1.UpdateOptions{})
	return err
}

func applyDaemonSet(ctx context.Context, client kubernetes.Interface, desired *appsv1.DaemonSet) error {
	existing, err := client.AppsV1().DaemonSets(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.AppsV1().DaemonSets(desired.Namespace).Create(ctx, desired, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	desired.ResourceVersion = existing.ResourceVersion
	_, err = client.AppsV1().DaemonSets(desired.Namespace).Update(ctx, desired, metav1.UpdateOptions{})
	return err
}

func listKubernetesWorkloadOptions(ctx context.Context, client kubernetes.Interface) ([]kubernetesWorkloadOption, error) {
	result := make([]kubernetesWorkloadOption, 0)
	deployments, err := client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range deployments.Items {
		result = append(result, kubernetesWorkloadOption{Namespace: item.Namespace, Kind: "Deployment", Name: item.Name})
	}
	statefulSets, err := client.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range statefulSets.Items {
		result = append(result, kubernetesWorkloadOption{Namespace: item.Namespace, Kind: "StatefulSet", Name: item.Name})
	}
	daemonSets, err := client.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range daemonSets.Items {
		result = append(result, kubernetesWorkloadOption{Namespace: item.Namespace, Kind: "DaemonSet", Name: item.Name})
	}
	jobs, err := client.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range jobs.Items {
		result = append(result, kubernetesWorkloadOption{Namespace: item.Namespace, Kind: "Job", Name: item.Name})
	}
	cronJobs, err := client.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range cronJobs.Items {
		result = append(result, kubernetesWorkloadOption{Namespace: item.Namespace, Kind: "CronJob", Name: item.Name})
	}
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range pods.Items {
		if metav1.GetControllerOf(&item) == nil {
			result = append(result, kubernetesWorkloadOption{Namespace: item.Namespace, Kind: "Pod", Name: item.Name})
		}
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Namespace != result[right].Namespace {
			return result[left].Namespace < result[right].Namespace
		}
		if result[left].Kind != result[right].Kind {
			return result[left].Kind < result[right].Kind
		}
		return result[left].Name < result[right].Name
	})
	return result, nil
}

func parseKubernetesCollectorRequest(c *gin.Context) kubernetesCollectorRequest {
	var request kubernetesCollectorRequest
	_ = c.ShouldBindJSON(&request)
	return request
}

func resolveCollectorServerURL(c *gin.Context, requested string) string {
	publicScheme := firstNonEmpty(c.GetHeader("X-Forwarded-Proto"), os.Getenv("OPSHUB_PUBLIC_SCHEME"), "http")
	if requested = strings.TrimRight(strings.TrimSpace(requested), "/"); requested != "" {
		return normalizePublicURL(requested, publicScheme)
	}
	for _, key := range []string{"OPSHUB_SERVER_EXTERNAL_URL", "OPSHUB_SERVER_FRONTEND_URL"} {
		if value := strings.TrimRight(strings.TrimSpace(os.Getenv(key)), "/"); value != "" {
			return normalizePublicURL(value, publicScheme)
		}
	}
	proto := publicScheme
	host := firstNonEmpty(c.GetHeader("X-Forwarded-Host"), c.Request.Host)
	return strings.TrimRight(proto+"://"+host, "/")
}

func resolveCollectorImage(requested string) string {
	return firstNonEmpty(strings.TrimSpace(requested), strings.TrimSpace(os.Getenv("OPSHUB_LOG_AGENT_IMAGE")), "docker.1ms.run/dyclouds/opshub-log-agent:v0.0.8")
}

func boolPointer(value bool) *bool                                       { return &value }
func hostPathTypePointer(value corev1.HostPathType) *corev1.HostPathType { return &value }
