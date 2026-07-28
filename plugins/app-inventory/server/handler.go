package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"github.com/ydcloud-dy/opshub/plugins/app-inventory/service"
	"gorm.io/gorm"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Overview(c *gin.Context) {
	data, err := h.svc.Overview(c.Request.Context())
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ListApplications(c *gin.Context) {
	data, err := h.svc.ListApplications(c.Request.Context(), listOptions(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) GetApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	data, err := h.svc.GetApplication(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) CreateApplication(c *gin.Context) {
	var in service.ApplicationInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.CreateApplication(c.Request.Context(), &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) UpdateApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in service.ApplicationInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.UpdateApplication(c.Request.Context(), id, &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) DeleteApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteApplication(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) ProbeApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	data, err := h.svc.ProbeApplication(c.Request.Context(), id, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ListEnvironments(c *gin.Context) {
	appID := parseQueryUint(c, "app_id")
	items, err := h.svc.ListEnvironments(c.Request.Context(), appID)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateEnvironment(c *gin.Context) {
	var in service.EnvironmentInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.CreateEnvironment(c.Request.Context(), &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) UpdateEnvironment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in service.EnvironmentInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.UpdateEnvironment(c.Request.Context(), id, &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) DeleteEnvironment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteEnvironment(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) ListDomains(c *gin.Context) {
	data, err := h.svc.ListDomains(c.Request.Context(), listOptions(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) CreateDomain(c *gin.Context) {
	var in service.DomainInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.CreateDomain(c.Request.Context(), &in)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) UpdateDomain(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in service.DomainInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.UpdateDomain(c.Request.Context(), id, &in)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) DeleteDomain(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteDomain(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) ProbeDomain(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	data, err := h.svc.ProbeDomain(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ListResources(c *gin.Context) {
	data, err := h.svc.ListResources(c.Request.Context(), listOptions(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) CreateResource(c *gin.Context) {
	var in service.ResourceInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.CreateResource(c.Request.Context(), &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) UpdateResource(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in service.ResourceInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.UpdateResource(c.Request.Context(), id, &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) DeleteResource(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteResource(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) ProbeResource(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	data, err := h.svc.ProbeResource(c.Request.Context(), id, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ListComponents(c *gin.Context) {
	data, err := h.svc.ListComponents(c.Request.Context(), listOptions(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) CreateComponent(c *gin.Context) {
	var in service.ComponentInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.CreateComponent(c.Request.Context(), &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) UpdateComponent(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in service.ComponentInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.UpdateComponent(c.Request.Context(), id, &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) DeleteComponent(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteComponent(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) ProbeComponent(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	data, err := h.svc.ProbeComponent(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ListDependencies(c *gin.Context) {
	data, err := h.svc.ListDependencies(c.Request.Context(), listOptions(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) CreateDependency(c *gin.Context) {
	var in service.DependencyInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.CreateDependency(c.Request.Context(), &in)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) UpdateDependency(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in service.DependencyInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.UpdateDependency(c.Request.Context(), id, &in)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) DeleteDependency(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteDependency(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) Topology(c *gin.Context) {
	appID := parseQueryUint(c, "app_id")
	if c.Param("id") != "" {
		parsed, ok := parseID(c)
		if !ok {
			return
		}
		appID = parsed
	}
	data, err := h.svc.Topology(c.Request.Context(), appID)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ListCredentials(c *gin.Context) {
	data, err := h.svc.ListCredentials(c.Request.Context(), listOptions(c), userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) CreateCredential(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	var in service.CredentialInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.CreateCredential(c.Request.Context(), &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) UpdateCredential(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in service.CredentialInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.UpdateCredential(c.Request.Context(), id, &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) DeleteCredential(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteCredential(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) RevealCredential(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if c.Request.ContentLength > 0 && !bindJSON(c, &req) {
		return
	}
	data, err := h.svc.RevealCredential(c.Request.Context(), id, userID(c), rbac.GetUsername(c), c.ClientIP(), c.Request.UserAgent(), req.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "权限") {
			c.JSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "message": err.Error()})
			return
		}
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ListGrants(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireCredentialManager(c, id) {
		return
	}
	data, err := h.svc.ListGrants(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) UpsertGrant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireCredentialManager(c, id) {
		return
	}
	var in service.CredentialGrantInput
	if !bindJSON(c, &in) {
		return
	}
	data, err := h.svc.UpsertGrant(c.Request.Context(), id, &in, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) DeleteGrant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.svc.CanManageGrant(c.Request.Context(), id, userID(c)) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "message": "没有管理该凭据授权的权限"})
		return
	}
	if err := h.svc.DeleteGrant(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) ListSecretAudits(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	data, err := h.svc.ListSecretAudits(c.Request.Context(), parseQueryUint(c, "credential_id"), parseQueryInt(c, "limit"))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) References(c *gin.Context) {
	currentUserID := userID(c)
	data, err := h.svc.References(c.Request.Context(), currentUserID, h.svc.CanManageAnyCredential(c.Request.Context(), currentUserID))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) DiscoverKubernetes(c *gin.Context) {
	var req service.KubernetesDiscoveryRequest
	if !bindJSON(c, &req) {
		return
	}
	data, err := h.svc.DiscoverKubernetes(c.Request.Context(), req.ClusterID, userID(c), req.Namespace, req.Selector)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ListKubernetesNamespaces(c *gin.Context) {
	clusterID := parseQueryUint(c, "cluster_id")
	data, err := h.svc.ListKubernetesNamespaces(c.Request.Context(), clusterID, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ImportKubernetes(c *gin.Context) {
	var req service.KubernetesDiscoveryRequest
	if !bindJSON(c, &req) {
		return
	}
	data, err := h.svc.ImportKubernetes(c.Request.Context(), &req, userID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ListDiscoveryRuns(c *gin.Context) {
	items, err := h.svc.ListDiscoveryRuns(c.Request.Context(), parseQueryUint(c, "app_id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) requireAdmin(c *gin.Context) bool {
	if h.svc.IsAdmin(c.Request.Context(), userID(c)) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "message": "该操作仅限应用资产管理员"})
	return false
}

func (h *Handler) requireCredentialManager(c *gin.Context, credentialID uint) bool {
	if h.svc.CanManageCredential(c.Request.Context(), credentialID, userID(c)) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "message": "没有管理该凭据授权的权限"})
	return false
}

func (h *Handler) fail(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "记录不存在"})
		return
	}
	message := err.Error()
	if strings.Contains(message, "权限") {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "message": message})
		return
	}
	if strings.Contains(strings.ToLower(message), "duplicate") || strings.Contains(message, "已存在") {
		c.JSON(http.StatusConflict, gin.H{"code": http.StatusConflict, "message": message})
		return
	}
	// Input and dependency errors are returned as a business error to match the existing frontend client.
	if strings.Contains(message, "不能为空") || strings.Contains(message, "至少") || strings.Contains(message, "必须") || strings.Contains(message, "不属于") || strings.Contains(message, "未找到") || strings.Contains(message, "不存在") || strings.Contains(message, "无效") || strings.Contains(message, "不能") || strings.Contains(message, "不可") || strings.Contains(message, "超过") {
		response.ErrorCode(c, http.StatusBadRequest, message)
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": message})
}

func userID(c *gin.Context) uint { return c.GetUint(rbac.UserIdKey) }

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "无效的ID"})
		return 0, false
	}
	return uint(id), true
}

func parseQueryUint(c *gin.Context, key string) uint {
	value, _ := strconv.ParseUint(c.Query(key), 10, 32)
	return uint(value)
}

func parseQueryInt(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

func listOptions(c *gin.Context) service.ListOptions {
	return service.ListOptions{Page: parseQueryInt(c, "page"), PageSize: parseQueryInt(c, "page_size"), Keyword: c.Query("keyword"), AppID: parseQueryUint(c, "app_id"), EnvID: parseQueryUint(c, "environment_id"), Kind: c.Query("kind"), Category: c.Query("category"), Status: c.Query("status")}
}

func bindJSON(c *gin.Context, target interface{}) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": fmt.Sprintf("请求参数错误: %v", err)})
		return false
	}
	return true
}
