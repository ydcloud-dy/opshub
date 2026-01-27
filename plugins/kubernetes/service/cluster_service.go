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
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/api/core/v1"
	authenticationv1 "k8s.io/api/authentication/v1"

	rbacBiz "github.com/ydcloud-dy/opshub/internal/biz/rbac"
	rbacData "github.com/ydcloud-dy/opshub/internal/data/rbac"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/biz"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/model"
)

const (
	// OpsHubAuthNamespace OpsHub 认证专用命名空间
	OpsHubAuthNamespace = "opshub-auth"
)

// ClusterService 集群服务层
type ClusterService struct {
	clusterBiz *biz.ClusterBiz
	db         *gorm.DB

	// 缓存已连接的集群 clientset (key: "clusterID-userID")
	clientsetCache map[string]*kubernetes.Clientset
	metricsCache   map[uint]*metricsv.Clientset
	cacheMutex     sync.RWMutex
}

// NewClusterService 创建集群服务
func NewClusterService(db *gorm.DB) *ClusterService {
	return &ClusterService{
		clusterBiz:     biz.NewClusterBiz(db),
		db:             db,
		clientsetCache: make(map[string]*kubernetes.Clientset),
		metricsCache:   make(map[uint]*metricsv.Clientset),
	}
}

// CreateClusterRequest 创建集群请求
type CreateClusterRequest struct {
	Name        string `json:"name" binding:"required"`
	Alias       string `json:"alias"`
	APIEndpoint string `json:"apiEndpoint"` // 移除 required，因为 KubeConfig 中已经包含
	KubeConfig  string `json:"kubeConfig" binding:"required"`
	Region      string `json:"region"`
	Provider    string `json:"provider"`
	Description string `json:"description"`
	UserID      uint   `json:"userId"`
}

// UpdateClusterRequest 更新集群请求
type UpdateClusterRequest struct {
	Name        string `json:"name"`
	Alias       string `json:"alias"`
	APIEndpoint string `json:"apiEndpoint"`
	KubeConfig  string `json:"kubeConfig"`
	Region      string `json:"region"`
	Provider    string `json:"provider"`
	Description string `json:"description"`
}

// ClusterDetailResponse 集群详情响应
type ClusterDetailResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Alias       string `json:"alias"`
	APIEndpoint string `json:"apiEndpoint"`
	Version     string `json:"version"`
	Status      int    `json:"status"`
	NodeCount   int    `json:"nodeCount"`   // 节点数量
	Region      string `json:"region"`
	Provider    string `json:"provider"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// CreateCluster 创建集群
func (s *ClusterService) CreateCluster(ctx context.Context, req *CreateClusterRequest) (*ClusterDetailResponse, error) {
	bizReq := &biz.CreateClusterRequest{
		Name:        req.Name,
		Alias:       req.Alias,
		APIEndpoint: req.APIEndpoint,
		KubeConfig:  req.KubeConfig,
		Region:      req.Region,
		Provider:    req.Provider,
		Description: req.Description,
		CreatedBy:   req.UserID,
	}

	cluster, err := s.clusterBiz.CreateCluster(ctx, bizReq)
	if err != nil {
		return nil, err
	}

	return s.toClusterResponse(cluster), nil
}

// UpdateCluster 更新集群
func (s *ClusterService) UpdateCluster(ctx context.Context, id uint, req *UpdateClusterRequest) (*ClusterDetailResponse, error) {
	bizReq := &biz.UpdateClusterRequest{
		Name:        req.Name,
		Alias:       req.Alias,
		APIEndpoint: req.APIEndpoint,
		KubeConfig:  req.KubeConfig,
		Region:      req.Region,
		Provider:    req.Provider,
		Description: req.Description,
	}

	cluster, err := s.clusterBiz.UpdateCluster(ctx, id, bizReq)
	if err != nil {
		return nil, err
	}

	// 清除缓存
	s.clearClientsetCache(id)

	return s.toClusterResponse(cluster), nil
}

// DeleteCluster 删除集群（并行优化版本）
func (s *ClusterService) DeleteCluster(ctx context.Context, id uint) error {
	// 1. 获取该集群的所有用户凭据记录
	var kubeConfigs []model.UserKubeConfig
	err := s.db.Where("cluster_id = ?", id).Find(&kubeConfigs).Error
	if err != nil {
		return fmt.Errorf("查询集群凭据失败: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 10) // 缓冲channel用于收集错误

	// 2. 并行清理每个用户的 K8s 资源和数据库记录
	for _, kc := range kubeConfigs {
		wg.Add(1)
		go func(kc model.UserKubeConfig) {
			defer wg.Done()
			// 获取用户名（从 ServiceAccount 提取，格式为 opshub-{username}）
			username := strings.TrimPrefix(kc.ServiceAccount, "opshub-")

			// 清理 K8s 中的 ServiceAccount 和 RoleBinding
			if err := s.cleanupClusterK8sResources(ctx, id, kc.ServiceAccount, username); err != nil {
				// 记录错误但继续清理其他资源
				errChan <- fmt.Errorf("清理用户 %s 的 K8s 资源失败: %w", username, err)
			}

			// 删除数据库记录 - k8s_user_kube_configs
			s.db.Where("cluster_id = ? AND id = ?", id, kc.ID).Delete(&model.UserKubeConfig{})
		}(kc)
	}

	// 3. 并行清理默认角色（ClusterRole）
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.cleanupDefaultRoles(ctx, id); err != nil {
			errChan <- fmt.Errorf("清理默认角色失败: %w", err)
		}
	}()

	// 4. 并行清理所有角色绑定（ClusterRoleBinding 和 RoleBinding）
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.cleanupAllRoleBindings(ctx, id); err != nil {
			errChan <- fmt.Errorf("清理角色绑定失败: %w", err)
		}
	}()

	// 5. 等待所有清理任务完成
	wg.Wait()
	close(errChan)

	// 6. 删除数据库中的角色绑定记录
	s.db.Table("k8s_user_role_bindings").
		Where("cluster_id = ?", id).
		Delete(&model.K8sUserRoleBinding{})

	// 7. 清除缓存
	s.clearClientsetCache(id)

	// 8. 删除集群
	if err := s.clusterBiz.DeleteCluster(ctx, id); err != nil {
		return err
	}

	// 9. 收集所有错误
	var errors []error
	for e := range errChan {
		errors = append(errors, e)
	}

	if len(errors) > 0 {
		return fmt.Errorf("删除集群完成，但有 %d 个清理错误: %v", len(errors), errors[0])
	}

	return nil
}

// cleanupClusterK8sResources 清理集群的 K8s 资源（SA 和 RoleBinding）
func (s *ClusterService) cleanupClusterK8sResources(ctx context.Context, clusterID uint, serviceAccount string, username string) error {
	// 获取 clientset
	clientset, err := s.GetCachedClientset(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取集群 clientset 失败: %w", err)
	}

	// 1. 删除 RoleBinding
	// 尝试删除命名空间级别的 RoleBinding
	if err := clientset.RbacV1().RoleBindings(OpsHubAuthNamespace).Delete(ctx, serviceAccount, metav1.DeleteOptions{}); err != nil {
		// 忽略不存在的错误
		_ = err
	}

	// 尝试删除集群级别的 ClusterRoleBinding
	if err := clientset.RbacV1().ClusterRoleBindings().Delete(ctx, serviceAccount, metav1.DeleteOptions{}); err != nil {
		// 忽略不存在的错误
		_ = err
	}

	// 2. 删除 ServiceAccount
	if err := clientset.CoreV1().ServiceAccounts(OpsHubAuthNamespace).Delete(ctx, serviceAccount, metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return fmt.Errorf("删除 ServiceAccount 失败: %w", err)
		}
	}

	return nil
}

// cleanupDefaultRoles 清理集群的默认角色（使用 DeleteCollection 批量删除）
func (s *ClusterService) cleanupDefaultRoles(ctx context.Context, clusterID uint) error {
	clientset, err := s.GetCachedClientset(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取集群 clientset 失败: %w", err)
	}

	// 使用 DeleteCollection 批量删除（一次 API 调用）
	labelSelector := "opshub.ydcloud-dy.com/managed-by=opshub"
	err = clientset.RbacV1().ClusterRoles().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("批量删除 ClusterRole 失败: %w", err)
	}

	return nil
}

// cleanupAllRoleBindings 清理集群的所有角色绑定（使用 DeleteCollection 批量删除）
func (s *ClusterService) cleanupAllRoleBindings(ctx context.Context, clusterID uint) error {
	clientset, err := s.GetCachedClientset(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取集群 clientset 失败: %w", err)
	}

	labelSelector := "opshub.ydcloud-dy.com/managed-by=opshub"
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// 1. 并行：批量删除 ClusterRoleBinding（通过标签）
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := clientset.RbacV1().ClusterRoleBindings().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil && !k8serrors.IsNotFound(err) {
			errChan <- err
		}
	}()

	// 2. 并行：批量删除所有命名空间中的 RoleBinding（通过标签）
	wg.Add(1)
	go func() {
		defer wg.Done()
		namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			errChan <- err
			return
		}

		// 使用 worker pool 限制并发数
		sem := make(chan struct{}, 10) // 最多 10 个并发
		var nsWg sync.WaitGroup

		for _, ns := range namespaces.Items {
			nsWg.Add(1)
			go func(namespace string) {
				defer nsWg.Done()
				sem <- struct{}{}        // 获取信号量
				defer func() { <-sem }() // 释放信号量

				err := clientset.RbacV1().RoleBindings(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
					LabelSelector: labelSelector,
				})
				if err != nil && !k8serrors.IsNotFound(err) {
					errChan <- err
				}
			}(ns.Name)
		}
		nsWg.Wait()
	}()

	// 3. 并行：删除旧的没有标签的 ClusterRoleBinding（通过前缀）
	// 注意：前缀匹配无法使用 DeleteCollection，只能先 List 再逐个 Delete
	wg.Add(1)
	go func() {
		defer wg.Done()
		allCRBs, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
		if err != nil {
			return
		}

		var deleteWg sync.WaitGroup
		for _, crb := range allCRBs.Items {
			if strings.HasPrefix(crb.Name, "opshub-") {
				deleteWg.Add(1)
				go func(name string) {
					defer deleteWg.Done()
					_ = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
				}(crb.Name)
			}
		}
		deleteWg.Wait()
	}()

	// 4. 并行：删除旧的没有标签的 RoleBinding（通过前缀）
	wg.Add(1)
	go func() {
		defer wg.Done()
		namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return
		}

		var nsWg sync.WaitGroup
		sem := make(chan struct{}, 10)

		for _, ns := range namespaces.Items {
			nsWg.Add(1)
			go func(namespace string) {
				defer nsWg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				allRBs, err := clientset.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return
				}

				var deleteWg sync.WaitGroup
				for _, rb := range allRBs.Items {
					if strings.HasPrefix(rb.Name, "opshub-") {
						deleteWg.Add(1)
						go func(ns, name string) {
							defer deleteWg.Done()
							_ = clientset.RbacV1().RoleBindings(ns).Delete(ctx, name, metav1.DeleteOptions{})
						}(namespace, rb.Name)
					}
				}
				deleteWg.Wait()
			}(ns.Name)
		}
		nsWg.Wait()
	}()

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for e := range errChan {
		errors = append(errors, e)
	}

	if len(errors) > 0 {
		return fmt.Errorf("清理角色绑定完成，但有 %d 个错误", len(errors))
	}

	return nil
}

// GetCluster 获取集群详情
func (s *ClusterService) GetCluster(ctx context.Context, id uint) (*ClusterDetailResponse, error) {
	cluster, err := s.clusterBiz.GetCluster(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toClusterResponse(cluster), nil
}

// ListClusters 获取集群列表
func (s *ClusterService) ListClusters(ctx context.Context) ([]ClusterDetailResponse, error) {
	clusters, err := s.clusterBiz.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]ClusterDetailResponse, 0, len(clusters))
	for _, cluster := range clusters {
		responses = append(responses, *s.toClusterResponse(&cluster))
	}

	return responses, nil
}

// TestClusterConnection 测试集群连接
func (s *ClusterService) TestClusterConnection(ctx context.Context, id uint) (string, error) {
	// 清除缓存，强制重新连接
	s.clearClientsetCache(id)

	return s.clusterBiz.TestClusterConnection(ctx, id)
}

// GetCachedClientset 获取缓存的 clientset（使用管理员权限）
// 注意：此方法使用集群管理员权限，建议使用 GetClientsetForUser 实现用户级权限控制
func (s *ClusterService) GetCachedClientset(ctx context.Context, id uint) (*kubernetes.Clientset, error) {
	cacheKey := fmt.Sprintf("%d-admin", id)

	s.cacheMutex.RLock()
	clientset, exists := s.clientsetCache[cacheKey]
	s.cacheMutex.RUnlock()

	if exists {
		return clientset, nil
	}

	// 缓存不存在，创建新的
	clientset, err := s.clusterBiz.GetClusterClientset(ctx, id)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	s.cacheMutex.Lock()
	s.clientsetCache[cacheKey] = clientset
	s.cacheMutex.Unlock()

	return clientset, nil
}

// GetClientsetForUser 获取基于用户权限的 clientset
// 这个方法会使用用户在 K8s 集群中的 ServiceAccount 凭据创建连接
// 这样可以实现真正的用户级权限隔离
// 平台管理员（role code == "admin"）会直接使用集群注册的 kubeconfig
func (s *ClusterService) GetClientsetForUser(ctx context.Context, clusterID uint, userID uint) (*kubernetes.Clientset, error) {
	cacheKey := fmt.Sprintf("%d-%d", clusterID, userID)

	s.cacheMutex.RLock()
	clientset, exists := s.clientsetCache[cacheKey]
	s.cacheMutex.RUnlock()

	if exists {
		return clientset, nil
	}

	// 检查用户是否是平台管理员（role code == "admin"）
	isPlatformAdmin, err := s.isPlatformAdmin(ctx, userID)
	if err != nil {
		// 如果检查角色失败，继续检查K8s角色
	}

	if isPlatformAdmin {
		// 平台管理员直接使用集群注册的 kubeconfig
		adminClientset, err := s.clusterBiz.GetClusterClientset(ctx, clusterID)
		if err != nil {
			return nil, fmt.Errorf("获取集群 clientset 失败: %w", err)
		}

		// 存入缓存
		s.cacheMutex.Lock()
		s.clientsetCache[cacheKey] = adminClientset
		s.cacheMutex.Unlock()

		return adminClientset, nil
	}

	// 检查用户在K8s集群中是否有管理员角色（如cluster-owner）
	hasK8sAdminRole, err := s.hasK8sClusterAdminRole(ctx, clusterID, userID)
	if err != nil {
		// 如果检查失败，继续使用普通用户逻辑
	} else if hasK8sAdminRole {
		// 有K8s管理员角色的用户也使用集群注册的 kubeconfig
		adminClientset, err := s.clusterBiz.GetClusterClientset(ctx, clusterID)
		if err != nil {
			return nil, fmt.Errorf("获取集群 clientset 失败: %w", err)
		}

		// 存入缓存
		s.cacheMutex.Lock()
		s.clientsetCache[cacheKey] = adminClientset
		s.cacheMutex.Unlock()

		return adminClientset, nil
	}

	// 非平台管理员，使用用户个人的 ServiceAccount 凭据

	// 缓存不存在，查询用户的 ServiceAccount 凭据
	var config model.UserKubeConfig
	err = s.db.Where("cluster_id = ? AND user_id = ? AND is_active = 1", clusterID, userID).
		Order("created_at DESC").
		First(&config).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户尚未申请该集群的访问凭据，请先申请 kubeconfig")
		}
		return nil, fmt.Errorf("查询用户凭据失败: %w", err)
	}

	// 获取集群信息
	cluster, err := s.clusterBiz.GetCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// 先获取管理员 clientset 用于生成用户的 kubeconfig
	adminClientset, err := s.clusterBiz.GetClusterClientset(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// 为用户的 ServiceAccount 生成 kubeconfig
	kubeConfigContent, err := s.generateKubeConfigForServiceAccount(adminClientset, cluster, config.ServiceAccount)
	if err != nil {
		return nil, fmt.Errorf("生成用户 kubeconfig 失败: %w", err)
	}

	// 使用用户的 kubeconfig 创建 clientset
	userClientset, err := biz.CreateClientsetFromKubeConfig(kubeConfigContent)
	if err != nil {
		return nil, fmt.Errorf("创建用户 clientset 失败: %w", err)
	}

	// 存入缓存
	s.cacheMutex.Lock()
	s.clientsetCache[cacheKey] = userClientset
	s.cacheMutex.Unlock()

	return userClientset, nil
}

// isPlatformAdmin 检查用户是否是平台管理员（是否有 code == "admin" 的角色）
func (s *ClusterService) isPlatformAdmin(ctx context.Context, userID uint) (bool, error) {
	// 先获取用户信息
	userRepo := rbacData.NewUserRepo(s.db)
	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("查询用户信息失败: %w", err)
	}

	// admin用户自动拥有平台管理员权限
	if user.Username == "admin" {
		return true, nil
	}

	roleRepo := rbacData.NewRoleRepo(s.db)
	roleUseCase := rbacBiz.NewRoleUseCase(roleRepo)

	roles, err := roleUseCase.GetByUserID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("查询用户角色失败: %w", err)
	}

	// 检查是否有 admin 角色的用户
	for _, role := range roles {
		if role.Code == "admin" {
			return true, nil
		}
	}

	return false, nil
}

// hasK8sClusterAdminRole 检查用户在K8s集群中是否有管理员角色（如cluster-owner、cluster-admin等）
func (s *ClusterService) hasK8sClusterAdminRole(ctx context.Context, clusterID, userID uint) (bool, error) {
	// 查询用户在该集群的角色绑定
	var bindings []model.K8sUserRoleBinding
	err := s.db.WithContext(ctx).
		Where("cluster_id = ? AND user_id = ?", clusterID, userID).
		Find(&bindings).Error

	if err != nil {
		return false, fmt.Errorf("查询用户K8s角色失败: %w", err)
	}

	// 检查是否有集群管理员级别的角色
	for _, binding := range bindings {
		// 检查是否有cluster-owner、cluster-admin等管理员角色
		if binding.RoleName == "cluster-owner" ||
		   binding.RoleName == "cluster-admin" ||
		   binding.RoleName == "admin" {
			return true, nil
		}
	}

	return false, nil
}

// GetClusterKubeConfig 获取集群的 KubeConfig（解密后的）
func (s *ClusterService) GetClusterKubeConfig(ctx context.Context, id uint) (string, error) {
	cluster, err := s.clusterBiz.GetCluster(ctx, id)
	if err != nil {
		return "", err
	}

	// 调用 biz 层的解密方法
	kubeConfig, err := biz.DecryptKubeConfig(cluster.KubeConfig)
	if err != nil {
		return "", err
	}

	return kubeConfig, nil
}

// ClearClientsetCache 清除指定集群的 clientset 缓存
func (s *ClusterService) ClearClientsetCache(id uint) {
	s.clearClientsetCache(id)
}

// clearClientsetCache 内部方法：清除缓存
// 清除所有与该集群相关的 clientset 缓存（包括所有用户的）
func (s *ClusterService) clearClientsetCache(id uint) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// 由于缓存 key 格式为 "clusterID-userID" 或 "clusterID-admin"
	// 需要遍历并删除所有以该 clusterID 开头的缓存
	clusterPrefix := fmt.Sprintf("%d-", id)
	for key := range s.clientsetCache {
		if strings.HasPrefix(key, clusterPrefix) {
			delete(s.clientsetCache, key)
		}
	}
}

// toClusterResponse 转换为响应对象
func (s *ClusterService) toClusterResponse(cluster *models.Cluster) *ClusterDetailResponse {
	if cluster == nil {
		return nil
	}

	// 使用数据库中缓存的节点数和 Pod 数
	return &ClusterDetailResponse{
		ID:          cluster.ID,
		Name:        cluster.Name,
		Alias:       cluster.Alias,
		APIEndpoint: cluster.APIEndpoint,
		Version:     cluster.Version,
		Status:      cluster.Status,
		NodeCount:   cluster.NodeCount,
		Region:      cluster.Region,
		Provider:    cluster.Provider,
		Description: cluster.Description,
		CreatedAt:   cluster.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   cluster.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// GetCachedMetricsClientset 获取缓存的 metrics clientset
func (s *ClusterService) GetCachedMetricsClientset(ctx context.Context, id uint) (*metricsv.Clientset, error) {
	s.cacheMutex.RLock()
	metricsClientset, exists := s.metricsCache[id]
	s.cacheMutex.RUnlock()

	if exists {
		return metricsClientset, nil
	}

	// 缓存不存在，创建新的
	// 先获取集群信息
	cluster, err := s.clusterBiz.GetCluster(ctx, id)
	if err != nil {
		return nil, err
	}

	// 使用 repository 的方法获取 config 和 clientset
	_, config, err := s.clusterBiz.GetRepo().GetClientset(cluster)
	if err != nil {
		return nil, err
	}

	// 创建 metrics clientset
	metricsClientset, err = metricsv.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	s.cacheMutex.Lock()
	s.metricsCache[id] = metricsClientset
	s.cacheMutex.Unlock()

	return metricsClientset, nil
}

// GetClusterAPIEndpoint 获取集群的 API Endpoint
func (s *ClusterService) GetClusterAPIEndpoint(ctx context.Context, id uint) (string, error) {
	cluster, err := s.clusterBiz.GetCluster(ctx, id)
	if err != nil {
		return "", err
	}

	// 如果数据库中已存储 API Endpoint，直接返回
	if cluster.APIEndpoint != "" {
		return cluster.APIEndpoint, nil
	}

	// 否则从 KubeConfig 中解析
	kubeConfig, err := biz.DecryptKubeConfig(cluster.KubeConfig)
	if err != nil {
		return "", err
	}

	// 从 KubeConfig 中提取 server 地址
	lines := strings.Split(kubeConfig, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "server:") {
			server := strings.TrimPrefix(line, "server:")
			server = strings.TrimSpace(server)
			// 去除引号
			server = strings.Trim(server, "\"")
			server = strings.Trim(server, "'")
			return server, nil
		}
	}

	return "", nil
}

// GetClusterConfig 获取集群的 KubeConfig（解密后的）
func (s *ClusterService) GetClusterConfig(ctx context.Context, id uint) (string, error) {
	cluster, err := s.clusterBiz.GetCluster(ctx, id)
	if err != nil {
		return "", err
	}

	// 调用 biz 层的解密方法
	kubeConfig, err := biz.DecryptKubeConfig(cluster.KubeConfig)
	if err != nil {
		return "", err
	}
	return kubeConfig, nil
}

// GenerateKubeConfigRequest 生成 KubeConfig 请求
type GenerateKubeConfigRequest struct {
	ClusterID uint   `json:"clusterId" binding:"required"`
	Username string `json:"username" binding:"required"`
}

// GenerateUserKubeConfig 为指定用户生成 KubeConfig
func (s *ClusterService) GenerateUserKubeConfig(ctx context.Context, clusterID uint, username string, userID uint) (string, string, error) {
	// 获取集群信息
	cluster, err := s.clusterBiz.GetCluster(ctx, clusterID)
	if err != nil {
		return "", "", err
	}

	// 获取集群的 clientset
	clientset, err := s.clusterBiz.GetClusterClientset(ctx, clusterID)
	if err != nil {
		return "", "", err
	}

	// 生成 KubeConfig
	kubeConfig, uniqueUsername, err := s.createKubeConfigForUser(clientset, cluster, username)
	if err != nil {
		return "", "", err
	}

	// 保存或更新凭据记录到数据库
	kubeConfigRecord := &model.UserKubeConfig{
		ClusterID:      uint64(clusterID),
		UserID:         uint64(userID),
		ServiceAccount: uniqueUsername,
		Namespace:      OpsHubAuthNamespace,
		IsActive:       true,
		CreatedBy:      uint64(userID),
	}

	// 使用 ON DUPLICATE KEY UPDATE 处理重复记录
	err = s.db.Where("cluster_id = ? AND user_id = ?", clusterID, userID).
		Assign(kubeConfigRecord).
		FirstOrCreate(kubeConfigRecord).Error

	if err != nil {
		return "", "", fmt.Errorf("保存凭据记录失败: %w", err)
	}

	return kubeConfig, uniqueUsername, nil
}

// GetUserExistingKubeConfig 获取用户现有的KubeConfig（如果存在）
func (s *ClusterService) GetUserExistingKubeConfig(ctx context.Context, clusterID uint, username string, userID uint) (string, string, error) {
	// 从数据库查询用户在该集群的激活凭据
	var config model.UserKubeConfig
	err := s.db.Where("cluster_id = ? AND user_id = ? AND is_active = 1", clusterID, userID).
		Order("created_at DESC").
		First(&config).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", "", fmt.Errorf("用户尚未申请凭据")
		}
		return "", "", fmt.Errorf("查询凭据失败: %w", err)
	}

	// 获取集群信息
	cluster, err := s.clusterBiz.GetCluster(ctx, clusterID)
	if err != nil {
		return "", "", err
	}

	// 获取集群的 clientset
	clientset, err := s.clusterBiz.GetClusterClientset(ctx, clusterID)
	if err != nil {
		return "", "", err
	}

	// 为现有的ServiceAccount生成kubeconfig
	kubeConfig, err := s.generateKubeConfigForServiceAccount(clientset, cluster, config.ServiceAccount)
	if err != nil {
		return "", "", fmt.Errorf("生成KubeConfig失败: %w", err)
	}

	return kubeConfig, config.ServiceAccount, nil
}

// GenerateKubeConfigForServiceAccount 为现有的ServiceAccount生成kubeconfig（公开方法）
func (s *ClusterService) GenerateKubeConfigForServiceAccount(ctx context.Context, clientset *kubernetes.Clientset, cluster *models.Cluster, saName string) (string, error) {
	return s.generateKubeConfigForServiceAccount(clientset, cluster, saName)
}

// GenerateKubeConfigForSA 为现有的ServiceAccount生成kubeconfig（不需要context的版本）
func (s *ClusterService) GenerateKubeConfigForSA(clientset *kubernetes.Clientset, cluster *models.Cluster, saName string) (string, error) {
	return s.generateKubeConfigForServiceAccount(clientset, cluster, saName)
}

// generateKubeConfigForServiceAccount 为现有的ServiceAccount生成kubeconfig
func (s *ClusterService) generateKubeConfigForServiceAccount(clientset *kubernetes.Clientset, cluster *models.Cluster, saName string) (string, error) {
	ctx := context.TODO()

	// 尝试从新命名空间获取 token
	expiration := int64(86400 * 365) // 1年有效期
	tr, err := clientset.CoreV1().ServiceAccounts(OpsHubAuthNamespace).CreateToken(ctx, saName, &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &expiration,
		},
	}, metav1.CreateOptions{})

	var targetNamespace string
	if err != nil {
		// 新命名空间失败，尝试旧的 default 命名空间（兼容旧数据）
		tr, err = clientset.CoreV1().ServiceAccounts("default").CreateToken(ctx, saName, &authenticationv1.TokenRequest{
			Spec: authenticationv1.TokenRequestSpec{
				ExpirationSeconds: &expiration,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			// 两个命名空间都失败，尝试查找 Secret
			targetNamespace = s.findServiceAccountNamespace(ctx, clientset, saName)
			if targetNamespace == "" {
				return "", fmt.Errorf("未找到 ServiceAccount: %s", saName)
			}
		} else {
			targetNamespace = "default"
		}
	} else {
		targetNamespace = OpsHubAuthNamespace
	}

	// 如果通过 TokenRequest 成功获取了 token
	if tr != nil && err == nil {
		token := tr.Status.Token
		kubeConfig, err := s.generateKubeConfigContent(clientset, cluster, saName, token)
		if err != nil {
			return "", err
		}
		return kubeConfig, nil
	}

	// TokenRequest 失败，尝试查找现有的 Secret
	var secretName string
	secrets, err := clientset.CoreV1().Secrets(targetNamespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, secret := range secrets.Items {
			if strings.HasPrefix(secret.Name, saName+"-token") {
				secretName = secret.Name
				break
			}
		}
	}

	if secretName == "" {
		return "", fmt.Errorf("获取 Token 失败且未找到现有 Secret")
	}

	secret, err := clientset.CoreV1().Secrets(targetNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("获取 Secret 失败: %w", err)
	}

	token, ok := secret.Data["token"]
	if !ok {
		return "", fmt.Errorf("Secret 中缺少 Token")
	}

	kubeConfig, err := s.generateKubeConfigContent(clientset, cluster, saName, string(token))
	if err != nil {
		return "", err
	}
	return kubeConfig, nil
}

// findServiceAccountNamespace 查找 ServiceAccount 所在的命名空间
func (s *ClusterService) findServiceAccountNamespace(ctx context.Context, clientset *kubernetes.Clientset, saName string) string {
	// 先检查新命名空间
	_, err := clientset.CoreV1().ServiceAccounts(OpsHubAuthNamespace).Get(ctx, saName, metav1.GetOptions{})
	if err == nil {
		return OpsHubAuthNamespace
	}

	// 再检查旧命名空间
	_, err = clientset.CoreV1().ServiceAccounts("default").Get(ctx, saName, metav1.GetOptions{})
	if err == nil {
		return "default"
	}

	return ""
}

// RevokeUserKubeConfig 吊销用户的 KubeConfig 凭据
func (s *ClusterService) RevokeUserKubeConfig(ctx context.Context, clusterID uint, username string) error {
	// 获取集群的 clientset
	clientset, err := s.clusterBiz.GetClusterClientset(ctx, clusterID)
	if err != nil {
		return err
	}

	// 删除 ClusterRoleBinding (如果有的话，例如 admin 用户)
	crbName := username + "-binding"
	err = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, crbName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		// ClusterRoleBinding 可能不存在（普通用户没有），继续删除 ServiceAccount
	}

	// 删除 ServiceAccount - 在 opshub-auth namespace 中
	err = clientset.CoreV1().ServiceAccounts(OpsHubAuthNamespace).Delete(ctx, username, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("删除 ServiceAccount 失败: %w", err)
	}

	// 更新数据库记录为已吊销（而不是删除）
	// 这样可以保留审计日志，并且GetClusterCredentialUsers会过滤这些记录
	now := time.Now()
	err = s.db.Where("cluster_id = ? AND service_account = ?", clusterID, username).
		Updates(map[string]interface{}{
			"is_active":  false,
			"revoked_at": now,
		}).Error
	if err != nil {
		// 记录错误但不影响整体流程（K8s资源已删除）
	}

	// 清除缓存
	s.clearClientsetCache(clusterID)

	return nil
}

// createKubeConfigForUser 创建用户专用的 KubeConfig
func (s *ClusterService) createKubeConfigForUser(clientset *kubernetes.Clientset, cluster *models.Cluster, username string) (string, string, error) {
	ctx := context.TODO()

	// 确保 OpsHub 认证命名空间存在
	if err := s.ensureOpsHubAuthNamespace(ctx, clientset); err != nil {
		return "", "", fmt.Errorf("确保命名空间存在失败: %w", err)
	}

	// ServiceAccount名称直接使用 opshub-{username}
	saName := fmt.Sprintf("opshub-%s", username)

	// 检查ServiceAccount是否已存在
	_, err := clientset.CoreV1().ServiceAccounts(OpsHubAuthNamespace).Get(ctx, saName, metav1.GetOptions{})
	if err != nil {
		// ServiceAccount不存在，创建新的
		if k8serrors.IsNotFound(err) {
			sa := &v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: saName,
					Labels: map[string]string{
						"opshub.ydcloud-dy.com/created-by": "opshub",
						"opshub.ydcloud-dy.com/username":  username,
					},
				},
			}
			_, err = clientset.CoreV1().ServiceAccounts(OpsHubAuthNamespace).Create(ctx, sa, metav1.CreateOptions{})
			if err != nil {
				return "", "", fmt.Errorf("创建 ServiceAccount 失败: %w", err)
			}

			// 不再自动创建任何权限绑定
			// 所有用户（包括 admin）都需要通过"角色授权"功能来分配权限
		} else {
			return "", "", fmt.Errorf("查询 ServiceAccount 失败: %w", err)
		}
	}
	// 如果ServiceAccount已存在，直接使用，不需要重新创建权限绑定

	// 使用 ServiceAccount 的 Token 创建请求
	// 通过创建 TokenRequest API 获取临时 token
	expiration := int64(86400 * 365) // 1年有效期
	tr, err := clientset.CoreV1().ServiceAccounts(OpsHubAuthNamespace).CreateToken(ctx, saName, &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &expiration,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		// 如果 TokenRequest 失败，尝试查找现有的 Secret
		var secretName string
		secrets, err := clientset.CoreV1().Secrets(OpsHubAuthNamespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, secret := range secrets.Items {
				if strings.HasPrefix(secret.Name, saName+"-token") {
					secretName = secret.Name
					break
				}
			}
		}

		if secretName == "" {
			return "", "", fmt.Errorf("获取 Token 失败且未找到现有 Secret: %w", err)
		}

		secret, err := clientset.CoreV1().Secrets(OpsHubAuthNamespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return "", "", fmt.Errorf("获取 Secret 失败: %w", err)
		}

		token, ok := secret.Data["token"]
		if !ok {
			return "", "", fmt.Errorf("Secret 中缺少 Token")
		}

		kubeConfig, err := s.generateKubeConfigContent(clientset, cluster, saName, string(token))
		if err != nil {
			return "", "", err
		}
		return kubeConfig, saName, nil
	}

	// 使用 TokenRequest 返回的 token
	token := tr.Status.Token
	kubeConfig, err := s.generateKubeConfigContent(clientset, cluster, saName, token)
	if err != nil {
		return "", "", err
	}
	return kubeConfig, saName, nil
}

// generateKubeConfigContent 生成 kubeconfig 内容
func (s *ClusterService) generateKubeConfigContent(clientset *kubernetes.Clientset, cluster *models.Cluster, username, token string) (string, error) {
	// 优先从集群获取 CA 证书
	caCert, err := s.getClusterCACert(clientset)
	if err != nil {
		// 如果从集群获取失败，尝试从原始 kubeconfig 中提取
		originalKubeConfig, decryptErr := biz.DecryptKubeConfig(cluster.KubeConfig)
		if decryptErr == nil {
			caCert = extractCAFromKubeconfig(originalKubeConfig)
			if caCert == "" {
				return "", fmt.Errorf("无法从原始 kubeconfig 中提取 CA 证书")
			}
		} else {
			return "", fmt.Errorf("解密集群 kubeconfig 失败: %w", decryptErr)
		}
	}

	// 验证 CA 证书不为空
	if caCert == "" {
		return "", fmt.Errorf("CA 证书为空")
	}

	// 解密集群的原始 kubeconfig 获取 server 地址
	serverURL := cluster.APIEndpoint
	if serverURL == "" {
		originalKubeConfig, err := biz.DecryptKubeConfig(cluster.KubeConfig)
		if err != nil {
			return "", fmt.Errorf("解密集群 kubeconfig 失败: %w", err)
		}

		lines := strings.Split(originalKubeConfig, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "server:") {
				serverURL = strings.TrimPrefix(line, "server:")
				serverURL = strings.TrimSpace(serverURL)
				serverURL = strings.Trim(serverURL, "\"")
				serverURL = strings.Trim(serverURL, "'")
				break
			}
		}
	}

	if serverURL == "" {
		return "", fmt.Errorf("无法确定集群的 API Server 地址")
	}

	// 生成 KubeConfig 内容（使用标准的 kubectl 格式）
	kubeConfigContent := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
  - name: %s
    cluster:
      certificate-authority-data: %s
      server: %s
contexts:
  - name: %s
    context:
      cluster: %s
      user: %s
current-context: %s
preferences: {}
users:
  - name: %s
    user:
      token: %s
`,
		cluster.Name,
		caCert,
		serverURL,
		cluster.Name+"-context",
		cluster.Name,
		username,
		cluster.Name+"-context",
		username,
		token,
	)

	return kubeConfigContent, nil
}

// extractCAFromKubeconfig 从 kubeconfig 内容中提取 CA 证书
func extractCAFromKubeconfig(kubeconfig string) string {
	lines := strings.Split(kubeconfig, "\n")

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 查找 certificate-authority-data 字段
		if strings.HasPrefix(trimmedLine, "certificate-authority-data:") {
			// 尝试获取同一行的值
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				data := strings.TrimSpace(parts[1])
				// 如果同一行有数据且不是其他字段
				if data != "" && !strings.HasPrefix(data, "server:") && !strings.HasPrefix(data, "client-") {
					return data
				}
			}

			// 尝试获取下一行的值（多行格式）
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				// 下一行应该是纯数据，不包含冒号（不是新的字段）
				if nextLine != "" && !strings.Contains(nextLine, ":") {
					return nextLine
				}
			}
		}
	}

	// 如果都找不到，返回空字符串
	return ""
}

// getClusterCACert 获取集群的 CA 证书
func (s *ClusterService) getClusterCACert(clientset *kubernetes.Clientset) (string, error) {
	// 尝试从 kube-system 命名空间的 ServiceAccount Secret 获取
	secrets, err := clientset.CoreV1().Secrets("kube-system").List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, secret := range secrets.Items {
			if strings.HasPrefix(secret.Name, "default-token-") || strings.HasPrefix(secret.Name, "coredns-token-") {
				if caCert, ok := secret.Data["ca.crt"]; ok {
					// CA 证书已经是 DER 格式，需要 base64 编码
					// k8s 存储的 ca.crt 已经是 PEM 格式，需要转换为 base64
					return base64.StdEncoding.EncodeToString(caCert), nil
				}
			}
		}
	}

	// 如果 kube-system 中没找到，尝试从所有命名空间查找任何包含 ca.crt 的 Secret
	allSecrets, err := clientset.CoreV1().Secrets("").List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, secret := range allSecrets.Items {
			if caCert, ok := secret.Data["ca.crt"]; ok && len(caCert) > 0 {
				return base64.StdEncoding.EncodeToString(caCert), nil
			}
		}
	}

	// 如果还没找到，尝试从 ConfigMap 获取
	cm, err := clientset.CoreV1().ConfigMaps("kube-public").Get(context.TODO(), "cluster-info", metav1.GetOptions{})
	if err == nil {
		if kubeconfig, ok := cm.Data["kubeconfig"]; ok {
			// 从 kubeconfig 中提取 CA 证书
			lines := strings.Split(kubeconfig, "\n")
			inCA := false
			var caData strings.Builder
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "certificate-authority-data:") {
					inCA = true
					// 尝试获取同一行的数据
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						data := strings.TrimSpace(parts[1])
						if data != "" && !strings.HasPrefix(data, "server:") {
							return data, nil
						}
					}
					continue
				}
				if inCA {
					// 如果遇到新的字段（包含冒号且不是续行），说明 CA 数据结束
					if strings.Contains(line, ":") {
						break
					}
					if line != "" {
						caData.WriteString(line)
					}
				}
			}
			if caData.String() != "" {
				return caData.String(), nil
			}
		}
	}

	return "", fmt.Errorf("无法获取集群 CA 证书")
}

// RevokeCredentialFully 完全吊销用户凭据（删除 SA、RoleBinding 和数据库记录）
func (s *ClusterService) RevokeCredentialFully(ctx context.Context, clusterID uint, serviceAccount, username string) error {
	// 获取集群的 clientset
	clientset, err := s.clusterBiz.GetClusterClientset(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取K8s客户端失败: %w", err)
	}

	// 1. 删除所有相关的 ClusterRoleBindings（检查两个命名空间）
	crbs, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, crb := range crbs.Items {
			// 检查是否有 Subject 引用了这个 ServiceAccount（支持两个命名空间）
			for _, subject := range crb.Subjects {
				if subject.Kind == "ServiceAccount" && subject.Name == serviceAccount &&
					(subject.Namespace == OpsHubAuthNamespace || subject.Namespace == "default") {
					_ = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, crb.Name, metav1.DeleteOptions{})
					break
				}
			}
		}
	}

	// 2. 删除所有命名空间中的 RoleBindings（检查两个命名空间）
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, ns := range namespaces.Items {
			rbs, err := clientset.RbacV1().RoleBindings(ns.Name).List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}
			for _, rb := range rbs.Items {
				// 检查是否有 Subject 引用了这个 ServiceAccount（支持两个命名空间）
				for _, subject := range rb.Subjects {
					if subject.Kind == "ServiceAccount" && subject.Name == serviceAccount &&
						(subject.Namespace == OpsHubAuthNamespace || subject.Namespace == "default") {
						_ = clientset.RbacV1().RoleBindings(ns.Name).Delete(ctx, rb.Name, metav1.DeleteOptions{})
						break
					}
				}
			}
		}
	}

	// 3. 删除 ServiceAccount（先尝试新命名空间，再尝试旧命名空间）
	deleted := false
	err = clientset.CoreV1().ServiceAccounts(OpsHubAuthNamespace).Delete(ctx, serviceAccount, metav1.DeleteOptions{})
	if err == nil {
		deleted = true
	} else if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("删除 ServiceAccount 失败: %w", err)
	}

	// 如果新命名空间没找到，尝试从旧命名空间删除
	if !deleted {
		err = clientset.CoreV1().ServiceAccounts("default").Delete(ctx, serviceAccount, metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return fmt.Errorf("删除 ServiceAccount 失败: %w", err)
		}
	}

	// 4. 获取用户ID
	var user struct {
		ID uint64
	}
	err = s.db.Table("sys_user").Select("id").Where("username = ?", username).First(&user).Error
	if err != nil {
		// 如果找不到用户，只删除SA即可
		return nil
	}

	// 5. 标记凭据记录为已吊销（而不是删除）
	// 这样可以保留审计日志，并且GetClusterCredentialUsers会过滤这些记录
	now := time.Now()
	s.db.Where("cluster_id = ? AND user_id = ? AND service_account = ?", clusterID, user.ID, serviceAccount).
		Updates(map[string]interface{}{
			"is_active":  false,
			"revoked_at": now,
		})

	// 6. 删除数据库记录 - k8s_user_role_bindings（角色绑定可以删除，因为凭据本身已吊销）
	s.db.Table("k8s_user_role_bindings").
		Where("cluster_id = ? AND user_id = ?", clusterID, user.ID).
		Delete(&model.K8sUserRoleBinding{})

	return nil
}

// ensureOpsHubAuthNamespace 确保 OpsHub 认证命名空间存在
func (s *ClusterService) ensureOpsHubAuthNamespace(ctx context.Context, clientset *kubernetes.Clientset) error {
	nsClient := clientset.CoreV1().Namespaces()

	// 检查命名空间是否已存在
	_, err := nsClient.Get(ctx, OpsHubAuthNamespace, metav1.GetOptions{})
	if err == nil {
		// 已存在，直接返回
		return nil
	}

	if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("检查命名空间失败: %w", err)
	}

	// 创建新的命名空间
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: OpsHubAuthNamespace,
			Labels: map[string]string{
				"name":                                 "opshub-auth",
				"opshub.ydcloud-dy.com/purpose":        "authentication",
				"opshub.ydcloud-dy.com/managed-by":     "opshub",
				"opshub.ydcloud-dy.com/namespace-type": "system",
			},
			Annotations: map[string]string{
				"description": "OpsHub user authentication namespace - managed by OpsHub, do not modify manually",
			},
		},
	}

	_, err = nsClient.Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("创建命名空间失败: %w", err)
	}

	return nil
}

// SyncClusterStatus 同步单个集群的状态信息
func (s *ClusterService) SyncClusterStatus(ctx context.Context, clusterID uint) error {
	// 获取 clientset
	clientset, err := s.GetCachedClientset(ctx, clusterID)
	if err != nil {
		// 连接失败，更新状态
		s.db.Model(&models.Cluster{}).Where("id = ?", clusterID).Update("status", models.ClusterStatusFailed)
		return fmt.Errorf("连接集群失败: %w", err)
	}

	// 获取节点列表
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("获取节点列表失败: %w", err)
	}
	nodeCount := len(nodes.Items)

	// 获取 Pod 列表
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("获取 Pod 列表失败: %w", err)
	}
	podCount := len(pods.Items)

	// 获取版本信息
	version, err := s.clusterBiz.TestClusterConnection(ctx, clusterID)
	if err != nil {
		// 无法获取版本，但已经连接成功，更新为正常状态
		s.db.Model(&models.Cluster{}).Where("id = ?", clusterID).Update("status", models.ClusterStatusNormal)
	} else {
		// 更新版本和状态
		s.db.Model(&models.Cluster{}).Where("id = ?", clusterID).Updates(map[string]interface{}{
			"version": version,
			"status":  models.ClusterStatusNormal,
		})
	}

	// 更新节点数和 Pod 数到数据库
	now := time.Now()
	err = s.db.Model(&models.Cluster{}).
		Where("id = ?", clusterID).
		Updates(map[string]interface{}{
			"node_count":       nodeCount,
			"pod_count":        podCount,
			"status_synced_at": &now,
		}).Error
	if err != nil {
		return fmt.Errorf("更新集群状态失败: %w", err)
	}

	return nil
}

// SyncAllClustersStatus 同步所有集群的状态信息（用于定时任务）
func (s *ClusterService) SyncAllClustersStatus(ctx context.Context) error {
	clusters, err := s.clusterBiz.ListClusters(ctx)
	if err != nil {
		return fmt.Errorf("获取集群列表失败: %w", err)
	}

	// 并发同步所有集群状态
	var wg sync.WaitGroup
	errChan := make(chan error, len(clusters))

	for _, cluster := range clusters {
		wg.Add(1)
		go func(id uint) {
			defer wg.Done()
			if err := s.SyncClusterStatus(ctx, id); err != nil {
				// 记录错误但继续处理其他集群
				errChan <- err
			}
		}(cluster.ID)
	}

	wg.Wait()
	close(errChan)

	// 收集错误（如果有）
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("部分集群同步失败，共 %d 个错误", len(errors))
	}

	return nil
}

// GetRESTConfig 获取用户的 REST Config（用于 WebSocket shell 等）
func (s *ClusterService) GetRESTConfig(clusterID uint, userID uint) (*rest.Config, error) {
	ctx := context.Background()

	// 检查用户是否是平台管理员
	isPlatformAdmin, err := s.isPlatformAdmin(ctx, userID)
	if err != nil {
		// 如果检查角色失败，继续使用普通用户逻辑
	} else if isPlatformAdmin {
		return s.clusterBiz.GetClusterRESTConfig(ctx, clusterID)
	}

	// 非平台管理员，使用用户个人的 ServiceAccount 凭据

	// 查询用户的 ServiceAccount 凭据
	var config model.UserKubeConfig
	err = s.db.Where("cluster_id = ? AND user_id = ? AND is_active = 1", clusterID, userID).
		Order("created_at DESC").
		First(&config).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户尚未申请该集群的访问凭据")
		}
		return nil, fmt.Errorf("查询用户凭据失败: %w", err)
	}

	// 获取集群信息
	cluster, err := s.clusterBiz.GetCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// 获取管理员 clientset 用于生成用户的 kubeconfig
	adminClientset, err := s.clusterBiz.GetClusterClientset(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// 为用户的 ServiceAccount 生成 kubeconfig
	kubeConfigContent, err := s.generateKubeConfigForServiceAccount(adminClientset, cluster, config.ServiceAccount)
	if err != nil {
		return nil, fmt.Errorf("生成用户 kubeconfig 失败: %w", err)
	}

	// 从 kubeconfig 创建 REST config
	restConfig, err := biz.CreateRESTConfigFromKubeConfig(kubeConfigContent)
	if err != nil {
		return nil, fmt.Errorf("创建 REST config 失败: %w", err)
	}

	return restConfig, nil
}

