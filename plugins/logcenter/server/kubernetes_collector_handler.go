package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
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
	"gorm.io/gorm"
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
	kubernetesCollectorVersion   = "0.3.1"
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

type kubernetesCollectorShutdownResult struct {
	ClusterID         uint   `json:"clusterId"`
	ClusterName       string `json:"clusterName"`
	Status            string `json:"status"`
	ActivePolicyCount int64  `json:"activePolicyCount"`
	Message           string `json:"message"`
}

type activeKubernetesPolicyCounter func(context.Context, uint, uint) (int64, error)
type kubernetesCollectorUninstaller func(context.Context, uint) error

func shutdownUnusedKubernetesCollectors(
	ctx context.Context,
	policyID uint,
	clusters []k8smodel.Cluster,
	countActivePolicies activeKubernetesPolicyCounter,
	uninstallCollector kubernetesCollectorUninstaller,
) []kubernetesCollectorShutdownResult {
	results := make([]kubernetesCollectorShutdownResult, 0, len(clusters))
	for _, cluster := range clusters {
		result := kubernetesCollectorShutdownResult{
			ClusterID:   cluster.ID,
			ClusterName: firstNonEmpty(cluster.Alias, cluster.Name, fmt.Sprintf("集群 %d", cluster.ID)),
		}
		activePolicyCount, err := countActivePolicies(ctx, cluster.ID, policyID)
		result.ActivePolicyCount = activePolicyCount
		if err != nil {
			result.Status = "failed"
			result.Message = "检查共享采集器使用情况失败: " + err.Error()
			results = append(results, result)
			continue
		}
		if activePolicyCount > 0 {
			result.Status = "skipped"
			result.Message = fmt.Sprintf("仍有 %d 条已发布策略使用该集群采集器，已保留 DaemonSet", activePolicyCount)
			results = append(results, result)
			continue
		}
		if err := uninstallCollector(ctx, cluster.ID); err != nil {
			result.Status = "failed"
			result.Message = "关闭采集器失败: " + err.Error()
			results = append(results, result)
			continue
		}
		result.Status = "uninstalled"
		result.Message = "DaemonSet、RBAC、配置和采集凭据已回收"
		results = append(results, result)
	}
	return results
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
	collectorServerURL := ""
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
			if configMap, configErr := clientset.CoreV1().ConfigMaps(kubernetesCollectorNamespace).Get(ctx, kubernetesCollectorName, metav1.GetOptions{}); configErr == nil {
				collectorServerURL = strings.TrimSpace(configMap.Data["server-url"])
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
		"serverUrl": collectorServerURL,
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
	serverURL, err := resolveCollectorServerURL(c, request.ServerURL)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	token, credential, err := h.rotateKubernetesCollectorCredential(c, clusterID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "生成集群采集 Token 失败: "+err.Error())
		return
	}
	resources := buildKubernetesCollectorResources(clusterID, token, serverURL, resolveCollectorImage(request.Image), request.Namespace)
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
	serverURL, err := resolveCollectorServerURL(c, request.ServerURL)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
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
	resources := buildKubernetesCollectorResources(clusterID, token, serverURL, resolveCollectorImage(request.Image), request.Namespace)
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
	activePolicyCount, err := h.countOtherPublishedKubernetesPolicies(c.Request.Context(), clusterID, 0)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "检查采集策略失败: "+err.Error())
		return
	}
	if activePolicyCount > 0 {
		response.ErrorCode(c, http.StatusConflict, fmt.Sprintf("该集群仍有 %d 条已发布采集策略，请先停用策略后再卸载采集器", activePolicyCount))
		return
	}
	if err := h.uninstallKubernetesCollector(c.Request.Context(), clusterID); err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "卸载 Kubernetes 日志采集器失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"clusterId": clusterID})
}

func (h *Handler) countOtherPublishedKubernetesPolicies(ctx context.Context, clusterID, excludedPolicyID uint) (int64, error) {
	query := h.db.WithContext(ctx).Model(&logmodel.CollectionPolicy{}).
		Where("source_mode = ? AND status = ?", "kubernetes", "published").
		Where("id IN (?)", h.db.Model(&logmodel.PolicyTarget{}).
			Select("policy_id").Where("target_type = ? AND target_id = ?", "cluster", clusterID))
	if excludedPolicyID > 0 {
		query = query.Where("id <> ?", excludedPolicyID)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (h *Handler) uninstallKubernetesCollector(ctx context.Context, clusterID uint) error {
	clientset, err := k8sservice.NewClusterService(h.db).GetCachedClientset(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("连接 Kubernetes 集群失败: %w", err)
	}
	kubernetesContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := deleteKubernetesCollectorResources(kubernetesContext, clientset); err != nil {
		return err
	}
	return h.markKubernetesCollectorUninstalled(ctx, clusterID, 0)
}

// markKubernetesCollectorUninstalled records a terminal disabled state after
// the remote collector resources have been removed. No agent can acknowledge
// the change after its credential is revoked, so leaving assignments pending
// would make the rollout appear stuck forever.
func (h *Handler) markKubernetesCollectorUninstalled(ctx context.Context, clusterID, policyID uint) error {
	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&logmodel.ClusterCollectorCredential{}).
			Where("cluster_id = ?", clusterID).Update("status", "revoked").Error; err != nil {
			return err
		}
		if err := tx.Model(&logmodel.CollectorInstance{}).
			Where("mode = ? AND cluster_id = ?", "kubernetes-node", clusterID).Update("status", "offline").Error; err != nil {
			return err
		}
		assignmentQuery := tx.Model(&logmodel.CollectorAssignment{}).
			Where("instance_id LIKE ? AND desired_state = ?", fmt.Sprintf("k8s:%d:%%", clusterID), "disabled")
		if policyID > 0 {
			assignmentQuery = assignmentQuery.Where("policy_id = ?", policyID)
		}
		now := time.Now()
		return assignmentQuery.Updates(map[string]any{
			"apply_status": "disabled", "applied_at": &now, "last_error": "",
		}).Error
	})
}

func deleteKubernetesCollectorResources(ctx context.Context, clientset kubernetes.Interface) error {
	deletePolicy := metav1.DeletePropagationBackground
	operations := []struct {
		name   string
		remove func() error
	}{
		{name: "DaemonSet", remove: func() error {
			return clientset.AppsV1().DaemonSets(kubernetesCollectorNamespace).Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		}},
		{name: "Secret", remove: func() error {
			return clientset.CoreV1().Secrets(kubernetesCollectorNamespace).Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{})
		}},
		{name: "ConfigMap", remove: func() error {
			return clientset.CoreV1().ConfigMaps(kubernetesCollectorNamespace).Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{})
		}},
		{name: "ServiceAccount", remove: func() error {
			return clientset.CoreV1().ServiceAccounts(kubernetesCollectorNamespace).Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{})
		}},
		{name: "ClusterRoleBinding", remove: func() error {
			return clientset.RbacV1().ClusterRoleBindings().Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{})
		}},
		{name: "ClusterRole", remove: func() error {
			return clientset.RbacV1().ClusterRoles().Delete(ctx, kubernetesCollectorName, metav1.DeleteOptions{})
		}},
	}
	var deleteErrors []error
	for _, operation := range operations {
		if err := operation.remove(); err != nil && !apierrors.IsNotFound(err) {
			deleteErrors = append(deleteErrors, fmt.Errorf("删除 %s 失败: %w", operation.name, err))
		}
	}
	return errors.Join(deleteErrors...)
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
	rolloutHash := sha256.Sum256([]byte(token + "\x00" + serverURL + "\x00" + image + "\x00" + kubernetesCollectorVersion))
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
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: map[string]string{"opshub.io/agent-version": kubernetesCollectorVersion, "opshub.io/config-checksum": hex.EncodeToString(rolloutHash[:])}}, Spec: corev1.PodSpec{
				ServiceAccountName: kubernetesCollectorName, AutomountServiceAccountToken: boolPointer(true),
				Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				InitContainers: []corev1.Container{{
					Name:            "download-agent",
					Image:           resolveCollectorDownloaderImage(),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/bin/sh", "-ec"},
					Args:            []string{collectorDownloadScript()},
					Env: []corev1.EnvVar{
						{Name: "OPSHUB_SERVER_URL", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: kubernetesCollectorName}, Key: "server-url"}}},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10m"), corev1.ResourceMemory: resource.MustParse("16Mi")},
						Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("64Mi")},
					},
					VolumeMounts: []corev1.VolumeMount{{Name: "agent-bin", MountPath: "/agent-bin"}},
				}},
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
						{Name: "agent-bin", MountPath: "/opshub-agent-bin", ReadOnly: true},
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
					{Name: "agent-bin", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
					{Name: "varlog", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/log", Type: hostPathTypePointer(corev1.HostPathDirectory)}}},
					{Name: "dockerlogs", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/docker/containers", Type: hostPathTypePointer(corev1.HostPathDirectoryOrCreate)}}},
					{Name: "state", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/opshub-log-agent", Type: hostPathTypePointer(corev1.HostPathDirectoryOrCreate)}}},
				},
			}},
		},
	}
	return resources
}

func collectorDownloadScript() string {
	return `case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac
SERVER="${OPSHUB_SERVER_URL%/}"
URL="${SERVER}/api/v1/public/agents/binaries/opshub-agent-linux-${ARCH}"
OUT=/agent-bin/opshub-agent
TMP=/agent-bin/opshub-agent.tmp
rm -f "${TMP}" "${OUT}"
if command -v wget >/dev/null 2>&1; then
  wget -q -O "${TMP}" "${URL}"
elif command -v curl >/dev/null 2>&1; then
  curl -fsSL -o "${TMP}" "${URL}"
else
  echo "downloader image must include wget or curl" >&2
  exit 1
fi
chmod 0755 "${TMP}"
"${TMP}" -version
mv "${TMP}" "${OUT}"`
}

func collectorEntrypointScript() string {
	return `AGENT=/opshub-agent-bin/opshub-agent
if [ ! -x "${AGENT}" ]; then
  AGENT=/usr/local/bin/opshub-agent
fi
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

func resolveCollectorServerURL(c *gin.Context, requested string) (string, error) {
	publicScheme := firstNonEmpty(c.GetHeader("X-Forwarded-Proto"), os.Getenv("OPSHUB_PUBLIC_SCHEME"), "http")
	if requested = strings.TrimRight(strings.TrimSpace(requested), "/"); requested != "" {
		return validateCollectorServerURL(normalizePublicURL(requested, publicScheme))
	}
	for _, key := range []string{"OPSHUB_SERVER_EXTERNAL_URL", "OPSHUB_SERVER_FRONTEND_URL"} {
		if value := strings.TrimRight(strings.TrimSpace(os.Getenv(key)), "/"); value != "" {
			return validateCollectorServerURL(normalizePublicURL(value, publicScheme))
		}
	}
	if publicHost := strings.TrimSpace(os.Getenv("OPSHUB_PUBLIC_HOST")); publicHost != "" {
		return validateCollectorServerURL(normalizePublicURL(publicHost, publicScheme))
	}
	host := firstNonEmpty(firstForwardedValue(c.GetHeader("X-Forwarded-Host")), c.Request.Host)
	return validateCollectorServerURL(normalizePublicURL(host, publicScheme))
}

func validateCollectorServerURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("Kubernetes 采集器控制面地址无效，请填写以 http:// 或 https:// 开头的完整地址")
	}
	hostname := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	ip := net.ParseIP(hostname)
	if hostname == "localhost" || (ip != nil && (ip.IsLoopback() || ip.IsUnspecified())) {
		return "", fmt.Errorf("Kubernetes 采集器控制面地址不能使用 %s；请填写目标集群 Pod 可访问的 OpsHub 地址，例如 https://opshub.example.com 或 http://10.0.0.8:9876", parsed.Host)
	}
	return value, nil
}

func resolveCollectorImage(requested string) string {
	return firstNonEmpty(strings.TrimSpace(requested), strings.TrimSpace(os.Getenv("OPSHUB_LOG_AGENT_IMAGE")), "docker.1ms.run/dyclouds/opshub-log-agent:v0.0.8")
}

func resolveCollectorDownloaderImage() string {
	return firstNonEmpty(strings.TrimSpace(os.Getenv("OPSHUB_LOG_AGENT_DOWNLOADER_IMAGE")), "swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/busybox:1.37.0")
}

func boolPointer(value bool) *bool                                       { return &value }
func hostPathTypePointer(value corev1.HostPathType) *corev1.HostPathType { return &value }
