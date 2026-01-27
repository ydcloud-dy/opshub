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
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/biz"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/model"
)

// RoleBindingService 角色绑定服务
type RoleBindingService struct {
	db         *gorm.DB
	clusterBiz *biz.ClusterBiz
}

// NewRoleBindingService 创建角色绑定服务
func NewRoleBindingService(db *gorm.DB) *RoleBindingService {
	return &RoleBindingService{
		db:         db,
		clusterBiz: biz.NewClusterBiz(db),
	}
}

// BindUserRole 绑定用户到K8s角色
func (s *RoleBindingService) BindUserRole(ctx context.Context, clusterID, userID uint64, roleName, roleNamespace, roleType string, boundBy uint64) error {
	// 检查表是否存在
	if !s.db.Migrator().HasTable(&model.K8sUserRoleBinding{}) {
		return errors.New("数据表不存在，请重启服务")
	}

	// 检查是否已经绑定
	var existing model.K8sUserRoleBinding
	err := s.db.Where("cluster_id = ? AND user_id = ? AND role_name = ? AND role_namespace = ?",
		clusterID, userID, roleName, roleNamespace).First(&existing).Error

	if err == nil {
		return errors.New("用户已绑定该角色")
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	// 获取集群信息
	cluster, err := s.clusterBiz.GetCluster(ctx, uint(clusterID))
	if err != nil {
		return fmt.Errorf("获取集群信息失败: %w", err)
	}

	// 获取 K8s 客户端
	clientset, _, err := s.clusterBiz.GetRepo().GetClientset(cluster)
	if err != nil {
		return fmt.Errorf("获取K8s客户端失败: %w", err)
	}

	// 获取用户信息
	var user struct {
		Username string
	}
	if err := s.db.Table("sys_user").Select("username").Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// ServiceAccount 名称格式: opshub-{username}
	saName := fmt.Sprintf("opshub-%s", user.Username)

	// 确保 OpsHub 认证命名空间存在
	if err := s.ensureOpsHubAuthNamespace(ctx, clientset); err != nil {
		return fmt.Errorf("确保命名空间存在失败: %w", err)
	}

	// 确保用户有 ServiceAccount（用于 kubeconfig 凭据）
	if err := s.ensureServiceAccount(ctx, clientset, saName, OpsHubAuthNamespace); err != nil {
		return fmt.Errorf("确保ServiceAccount存在失败: %w", err)
	}

	// 确保用户在数据库中有凭据记录（这样才能通过 GetClientsetForUser 访问集群）
	if err := s.ensureUserKubeConfigRecord(clusterID, userID, saName, OpsHubAuthNamespace); err != nil {
		return fmt.Errorf("确保用户凭据记录存在失败: %w", err)
	}

	// 在 K8s 中创建 ClusterRoleBinding 或 RoleBinding
	if roleType == "ClusterRole" {
		if err := s.createClusterRoleBinding(ctx, clientset, roleName, saName, OpsHubAuthNamespace); err != nil {
			return fmt.Errorf("创建ClusterRoleBinding失败: %w", err)
		}
	} else {
		if err := s.createRoleBinding(ctx, clientset, roleName, roleNamespace, saName, OpsHubAuthNamespace); err != nil {
			return fmt.Errorf("创建RoleBinding失败: %w", err)
		}
	}

	// 创建数据库绑定记录
	binding := model.K8sUserRoleBinding{
		ClusterID:     clusterID,
		UserID:        userID,
		RoleName:      roleName,
		RoleNamespace: roleNamespace,
		RoleType:      roleType,
		BoundBy:       boundBy,
	}

	if err := s.db.Create(&binding).Error; err != nil {
		// 数据库创建失败，尝试回滚 K8s 绑定
		_ = s.rollbackK8sBinding(ctx, clientset, roleType, roleName, roleNamespace, saName, OpsHubAuthNamespace)
		return fmt.Errorf("创建角色绑定失败: %w", err)
	}

	return nil
}

// UnbindUserRole 解绑用户K8s角色
func (s *RoleBindingService) UnbindUserRole(ctx context.Context, clusterID, userID uint64, roleName, roleNamespace string) error {
	// 获取集群信息
	cluster, err := s.clusterBiz.GetCluster(ctx, uint(clusterID))
	if err != nil {
		return fmt.Errorf("获取集群信息失败: %w", err)
	}

	// 获取 K8s 客户端
	clientset, _, err := s.clusterBiz.GetRepo().GetClientset(cluster)
	if err != nil {
		return fmt.Errorf("获取K8s客户端失败: %w", err)
	}

	// 获取用户信息
	var user struct {
		Username string
	}
	if err := s.db.Table("sys_user").Select("username").Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// ServiceAccount 名称
	saName := fmt.Sprintf("opshub-%s", user.Username)

	// 确定角色类型
	var roleType string
	var count int64
	s.db.Table("k8s_user_role_bindings").
		Where("cluster_id = ? AND user_id = ? AND role_name = ? AND role_namespace = ?",
			clusterID, userID, roleName, roleNamespace).
		Count(&count)

	if count == 0 {
		return errors.New("绑定关系不存在")
	}

	// 获取角色类型
	s.db.Table("k8s_user_role_bindings").
		Select("role_type").
		Where("cluster_id = ? AND user_id = ? AND role_name = ? AND role_namespace = ?",
			clusterID, userID, roleName, roleNamespace).
		Scan(&roleType)

	// 删除 K8s 绑定
	if roleType == "ClusterRole" {
		bindingName := fmt.Sprintf("opshub-%s-%s", roleName, saName)
		if err := clientset.RbacV1().ClusterRoleBindings().Delete(ctx, bindingName, metav1.DeleteOptions{}); err != nil {
			// 忽略不存在的错误
			if !strings.Contains(err.Error(), "not found") {
				// 错误日志已移除
			}
		}
	} else {
		bindingName := fmt.Sprintf("opshub-%s-%s", roleName, saName)
		if err := clientset.RbacV1().RoleBindings(roleNamespace).Delete(ctx, bindingName, metav1.DeleteOptions{}); err != nil {
			// 忽略不存在的错误
			if !strings.Contains(err.Error(), "not found") {
				// 错误日志已移除
			}
		}
	}

	// 删除数据库记录
	result := s.db.Where("cluster_id = ? AND user_id = ? AND role_name = ? AND role_namespace = ?",
		clusterID, userID, roleName, roleNamespace).Delete(&model.K8sUserRoleBinding{})

	if result.Error != nil {
		return fmt.Errorf("解绑角色失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("绑定关系不存在")
	}

	return nil
}

// GetRoleBoundUsers 获取角色已绑定的用户列表
func (s *RoleBindingService) GetRoleBoundUsers(ctx context.Context, clusterID uint64, roleName, roleNamespace string) ([]map[string]interface{}, error) {
	type Result struct {
		UserID    uint64 `json:"userId"`
		Username  string `json:"username"`
		RealName  string `json:"realName"`
		BoundAt   string `json:"boundAt"`
	}

	var results []Result

	// 查询绑定关系及用户信息
	err := s.db.Table("k8s_user_role_bindings as b").
		Select("b.user_id as user_id, u.username, u.real_name, b.created_at as bound_at").
		Joins("LEFT JOIN sys_user u ON u.id = b.user_id").
		Where("b.cluster_id = ? AND b.role_name = ? AND b.role_namespace = ?", clusterID, roleName, roleNamespace).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 转换为返回格式
	users := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		users = append(users, map[string]interface{}{
			"userId":    r.UserID,
			"username":  r.Username,
			"realName":  r.RealName,
			"boundAt":   r.BoundAt,
		})
	}

	return users, nil
}

// GetUserClusterRoles 获取用户在指定集群的角色列表
func (s *RoleBindingService) GetUserClusterRoles(ctx context.Context, clusterID, userID uint64) ([]model.K8sUserRoleBinding, error) {
	var bindings []model.K8sUserRoleBinding
	err := s.db.Where("cluster_id = ? AND user_id = ?", clusterID, userID).
		Order("created_at DESC").
		Find(&bindings).Error

	return bindings, err
}

// GetUserRoleForCluster 获取用户在指定集群的所有角色
func (s *RoleBindingService) GetUserRoleForCluster(clusterID, userID uint64) ([]model.K8sUserRoleBinding, error) {
	var bindings []model.K8sUserRoleBinding
	err := s.db.Where("cluster_id = ? AND user_id = ?", clusterID, userID).
		Find(&bindings).Error

	return bindings, err
}

// GetAvailableUsers 获取可绑定的用户列表
func (s *RoleBindingService) GetAvailableUsers(ctx context.Context, keyword string, page, pageSize int) ([]map[string]interface{}, int64, error) {
	type UserResult struct {
		ID       uint64 `json:"id"`
		Username string `json:"username"`
		RealName string `json:"realName"`
		Email    string `json:"email"`
	}

	var results []UserResult
	var total int64

	query := s.db.Table("sys_user").Select("id, username, real_name, email").Where("deleted_at IS NULL")

	if keyword != "" {
		keywordLike := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR real_name LIKE ? OR email LIKE ?", keywordLike, keywordLike, keywordLike)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	// 转换为返回格式
	users := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		users = append(users, map[string]interface{}{
			"id":       r.ID,
			"username": r.Username,
			"realName": r.RealName,
			"email":    r.Email,
		})
	}

	return users, total, nil
}

// GetClusterCredentialUsers 获取集群的凭据用户列表（返回所有opshub开头的ServiceAccount）
func (s *RoleBindingService) GetClusterCredentialUsers(ctx context.Context, clusterID uint64, currentUserID uint64) ([]map[string]interface{}, error) {
	// 获取集群信息
	cluster, err := s.clusterBiz.GetCluster(ctx, uint(clusterID))
	if err != nil {
		return nil, fmt.Errorf("获取集群信息失败: %w", err)
	}

	// 获取 kubernetes clientset
	clientset, _, err := s.clusterBiz.GetRepo().GetClientset(cluster)
	if err != nil {
		return nil, fmt.Errorf("获取K8s客户端失败: %w", err)
	}

	// 收集所有 ServiceAccount（从两个命名空间）
	saMap := make(map[string]corev1.ServiceAccount)

	// 从新命名空间获取
	newSas, err := clientset.CoreV1().ServiceAccounts(OpsHubAuthNamespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, sa := range newSas.Items {
			saMap[sa.Name] = sa
		}
	}

	// 从旧命名空间获取（兼容旧数据）
	oldSas, err := clientset.CoreV1().ServiceAccounts("default").List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, sa := range oldSas.Items {
			// 如果新命名空间没有这个 SA，才添加
			if _, exists := saMap[sa.Name]; !exists {
				saMap[sa.Name] = sa
			}
		}
	}

	// 过滤出 opshub- 开头的 ServiceAccount，并从数据库查询用户信息
	credentialUsers := make([]map[string]interface{}, 0)

	for _, sa := range saMap {
		saName := sa.Name
		// 检查是否是 opshub- 开头的 ServiceAccount
		if !strings.HasPrefix(saName, "opshub-") {
			continue
		}

		// 解析格式: opshub-{username}-{suffix} 或 opshub-{username}
		parts := strings.SplitN(saName, "-", 2)
		if len(parts) != 2 {
			continue
		}

		// 提取 username (opshub 之后的部分)
		username := parts[1]

		// 从数据库查询用户信息
		var user struct {
			ID       uint64
			Username string
			RealName string
		}
		err = s.db.Table("sys_user").
			Select("id, username, real_name").
			Where("username = ?", username).
			First(&user).Error

		// 如果查询不到用户信息，跳过
		if err != nil {
			continue
		}

		// 获取用户在该集群的凭据记录 - 必须是激活状态且未被吊销
		var kubeConfig model.UserKubeConfig
		err = s.db.Where("cluster_id = ? AND user_id = ? AND service_account = ? AND is_active = 1 AND revoked_at IS NULL",
			clusterID, user.ID, saName).
			First(&kubeConfig).Error

		// 如果没有找到激活的凭据记录，则不显示此ServiceAccount
		// 这确保了已吊销的凭据即使在Kubernetes中仍然存在也不会显示
		if err != nil {
			// 如果数据库中没有激活的凭据记录，说明：
			// 1. 凭据已被吊销 (is_active=0 或 revoked_at != null)
			// 2. 或者是直接在K8s中创建的无跟踪凭据
			// 为了安全起见，只显示数据库中有记录的凭据
			continue
		}

		// 使用数据库中的创建时间
		createdAt := kubeConfig.CreatedAt.Format("2006-01-02 15:04:05")

		credentialUsers = append(credentialUsers, map[string]interface{}{
			"username":       username,      // 平台用户名
			"realName":       user.RealName, // 真实姓名
			"serviceAccount": saName,       // K8s ServiceAccount 完整名称
			"namespace":      sa.Namespace, // 命名空间
			"userId":         user.ID,      // 平台用户ID
			"createdAt":      createdAt,    // 创建时间
		})
	}

	return credentialUsers, nil
}

// GetUserRoleBindings 获取用户的所有K8s角色绑定
func (s *RoleBindingService) GetUserRoleBindings(ctx context.Context, clusterID uint64, userID *uint64) ([]map[string]interface{}, error) {
	type Result struct {
		ID            uint64 `json:"id"`
		ClusterID     uint64 `json:"clusterId"`
		ClusterName   string `json:"clusterName"`
		UserID        uint64 `json:"userId"`
		Username      string `json:"username"`
		RealName      string `json:"realName"`
		RoleName      string `json:"roleName"`
		RoleNamespace string `json:"roleNamespace"`
		RoleType      string `json:"roleType"`
		CreatedAt     string `json:"createdAt"`
	}

	var results []Result

	query := s.db.Table("k8s_user_role_bindings as b").
		Select(`b.id, b.cluster_id, b.user_id, b.role_name, b.role_namespace, b.role_type, b.created_at,
				c.name as cluster_name, u.username, u.real_name`).
		Joins("LEFT JOIN k8s_clusters c ON c.id = b.cluster_id").
		Joins("LEFT JOIN sys_user u ON u.id = b.user_id").
		Where("b.cluster_id = ?", clusterID)

	if userID != nil {
		query = query.Where("b.user_id = ?", *userID)
	}

	err := query.Order("b.created_at DESC").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// 转换为返回格式
	bindings := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		bindings = append(bindings, map[string]interface{}{
			"id":            r.ID,
			"clusterId":     r.ClusterID,
			"clusterName":   r.ClusterName,
			"userId":        r.UserID,
			"username":      r.Username,
			"realName":      r.RealName,
			"roleName":      r.RoleName,
			"roleNamespace": r.RoleNamespace,
			"roleType":      r.RoleType,
			"createdAt":     r.CreatedAt,
		})
	}

	return bindings, nil
}

// ensureServiceAccount 确保 ServiceAccount 存在
func (s *RoleBindingService) ensureServiceAccount(ctx context.Context, clientset *kubernetes.Clientset, saName, saNamespace string) error {
	saClient := clientset.CoreV1().ServiceAccounts(saNamespace)

	// 检查是否已存在
	_, err := saClient.Get(ctx, saName, metav1.GetOptions{})
	if err == nil {
		// 已存在，直接返回
		return nil
	}

	// 不存在，创建新的
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName,
			Annotations: map[string]string{
				"description": "Created by OpsHub for user authentication",
			},
		},
	}

	_, err = saClient.Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("创建ServiceAccount失败: %w", err)
	}

	return nil
}

// createClusterRoleBinding 创建集群角色绑定
func (s *RoleBindingService) createClusterRoleBinding(ctx context.Context, clientset *kubernetes.Clientset, roleName, saName, saNamespace string) error {
	crbClient := clientset.RbacV1().ClusterRoleBindings()

	// 绑定名称格式: opshub-{roleName}-{saName}
	bindingName := fmt.Sprintf("opshub-%s-%s", roleName, saName)

	// 检查是否已存在
	_, err := crbClient.Get(ctx, bindingName, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	// 创建 ClusterRoleBinding
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
			Labels: map[string]string{
				"opshub.ydcloud-dy.com/managed-by": "opshub",
			},
			Annotations: map[string]string{
				"description": "Created by OpsHub",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: saNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     roleName,
		},
	}

	_, err = crbClient.Create(ctx, crb, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("创建ClusterRoleBinding失败: %w", err)
	}

	return nil
}

// createRoleBinding 创建命名空间角色绑定
func (s *RoleBindingService) createRoleBinding(ctx context.Context, clientset *kubernetes.Clientset, roleName, roleNamespace, saName, saNamespace string) error {
	rbClient := clientset.RbacV1().RoleBindings(roleNamespace)

	// 绑定名称格式: opshub-{roleName}-{saName}
	bindingName := fmt.Sprintf("opshub-%s-%s", roleName, saName)

	// 检查是否已存在
	_, err := rbClient.Get(ctx, bindingName, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	// 创建 RoleBinding
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
			Labels: map[string]string{
				"opshub.ydcloud-dy.com/managed-by": "opshub",
			},
			Annotations: map[string]string{
				"description": "Created by OpsHub",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: saNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     roleName,
		},
	}

	_, err = rbClient.Create(ctx, rb, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("创建RoleBinding失败: %w", err)
	}

	return nil
}

// rollbackK8sBinding 回滚 K8s 绑定（用于数据库操作失败时）
func (s *RoleBindingService) rollbackK8sBinding(ctx context.Context, clientset *kubernetes.Clientset, roleType, roleName, roleNamespace, saName, saNamespace string) error {
	if roleType == "ClusterRole" {
		bindingName := fmt.Sprintf("opshub-%s-%s", roleName, saName)
		_ = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, bindingName, metav1.DeleteOptions{})
	} else {
		bindingName := fmt.Sprintf("opshub-%s-%s", roleName, saName)
		_ = clientset.RbacV1().RoleBindings(roleNamespace).Delete(ctx, bindingName, metav1.DeleteOptions{})
	}
	return nil
}

// ensureOpsHubAuthNamespace 确保 OpsHub 认证命名空间存在
func (s *RoleBindingService) ensureOpsHubAuthNamespace(ctx context.Context, clientset *kubernetes.Clientset) error {
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
	ns := &corev1.Namespace{
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

// ensureUserKubeConfigRecord 确保用户在数据库中有凭据记录
func (s *RoleBindingService) ensureUserKubeConfigRecord(clusterID, userID uint64, serviceAccount, namespace string) error {
	// 检查是否已存在记录
	var existing model.UserKubeConfig
	err := s.db.Where("cluster_id = ? AND user_id = ?", clusterID, userID).First(&existing).Error

	if err == nil {
		// 记录已存在，检查 service_account 是否匹配
		if existing.ServiceAccount == serviceAccount {
			// 完全匹配，无需更新
			return nil
		}

		// ServiceAccount 名称不同，更新记录
		existing.ServiceAccount = serviceAccount
		existing.Namespace = namespace
		existing.IsActive = true
		if err := s.db.Save(&existing).Error; err != nil {
			return fmt.Errorf("更新凭据记录失败: %w", err)
		}
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询凭据记录失败: %w", err)
	}

	// 记录不存在，创建新记录
	kubeConfigRecord := &model.UserKubeConfig{
		ClusterID:      clusterID,
		UserID:         userID,
		ServiceAccount: serviceAccount,
		Namespace:      namespace,
		IsActive:       true,
		CreatedBy:      userID,
	}

	if err := s.db.Create(kubeConfigRecord).Error; err != nil {
		return fmt.Errorf("创建凭据记录失败: %w", err)
	}

	return nil
}
