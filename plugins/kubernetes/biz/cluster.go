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

package biz

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/data/repository"
)

// ClusterBiz 集群业务逻辑层
type ClusterBiz struct {
	repo *repository.ClusterRepository
	db   *gorm.DB
}

// NewClusterBiz 创建集群业务逻辑
func NewClusterBiz(db *gorm.DB) *ClusterBiz {
	return &ClusterBiz{
		repo: repository.NewClusterRepository(db),
		db:   db,
	}
}

// CreateClusterRequest 创建集群请求
type CreateClusterRequest struct {
	Name        string `json:"name" binding:"required"`
	Alias       string `json:"alias"`
	APIEndpoint string `json:"apiEndpoint" binding:"required"`
	KubeConfig  string `json:"kubeConfig" binding:"required"`
	Region      string `json:"region"`
	Provider    string `json:"provider"`
	Description string `json:"description"`
	CreatedBy   uint   `json:"createdBy"`
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

// CreateCluster 创建集群
func (b *ClusterBiz) CreateCluster(ctx context.Context, req *CreateClusterRequest) (*models.Cluster, error) {
	// 检查集群名称是否已存在
	existCluster, err := b.repo.GetByName(req.Name)
	if err == nil && existCluster != nil {
		return nil, errors.New("集群名称已存在")
	}

	// 先用原始 KubeConfig 测试连接
	testCluster := &models.Cluster{
		Name:        req.Name,
		KubeConfig:  req.KubeConfig, // 使用原始的、未加密的 KubeConfig 测试
		APIEndpoint: req.APIEndpoint,
	}

	clientset, version, err := b.repo.TestConnection(testCluster)
	if err != nil {
		return nil, fmt.Errorf("测试集群连接失败: %w", err)
	}
	_ = clientset

	// 测试成功后，加密 KubeConfig
	encryptedConfig, err := encryptKubeConfig(req.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("加密 kubeconfig 失败: %w", err)
	}

	cluster := &models.Cluster{
		Name:        req.Name,
		Alias:       req.Alias,
		APIEndpoint: req.APIEndpoint,
		KubeConfig:  encryptedConfig, // 存储加密后的 KubeConfig
		Region:      req.Region,
		Provider:    req.Provider,
		Description: req.Description,
		CreatedBy:   req.CreatedBy,
		Status:      models.ClusterStatusNormal,
		Version:     version, // 设置从测试连接获取的版本
	}

	// 保存到数据库
	if err := b.repo.Create(cluster); err != nil {
		return nil, fmt.Errorf("保存集群失败: %w", err)
	}

	return cluster, nil
}

// UpdateCluster 更新集群
func (b *ClusterBiz) UpdateCluster(ctx context.Context, id uint, req *UpdateClusterRequest) (*models.Cluster, error) {
	cluster, err := b.repo.GetByID(id)
	if err != nil {
		return nil, errors.New("集群不存在")
	}

	// 如果要更新名称，检查新名称是否已被其他集群使用
	if req.Name != "" && req.Name != cluster.Name {
		existCluster, err := b.repo.GetByName(req.Name)
		if err == nil && existCluster != nil && existCluster.ID != id {
			return nil, errors.New("集群名称已存在")
		}
		cluster.Name = req.Name
	}

	// 如果更新了 kubeconfig，需要重新加密
	if req.KubeConfig != "" && req.KubeConfig != cluster.KubeConfig {
		encryptedConfig, err := encryptKubeConfig(req.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("加密 kubeconfig 失败: %w", err)
		}
		cluster.KubeConfig = encryptedConfig
	}

	// 更新别名（支持清空）
	cluster.Alias = req.Alias

	// 更新 API Endpoint（支持清空）
	cluster.APIEndpoint = req.APIEndpoint

	// 更新 Region
	if req.Region != "" {
		cluster.Region = req.Region
	}

	// 更新 Provider
	if req.Provider != "" {
		cluster.Provider = req.Provider
	}

	// 更新 Description
	if req.Description != "" {
		cluster.Description = req.Description
	}

	// 测试连接
	clientset, version, err := b.repo.TestConnection(cluster)
	if err != nil {
		// 连接失败，更新状态为失败
		b.repo.UpdateStatus(id, models.ClusterStatusFailed)
		return nil, fmt.Errorf("测试集群连接失败: %w", err)
	}
	_ = clientset

	cluster.Version = version
	cluster.Status = models.ClusterStatusNormal

	// 更新数据库
	if err := b.repo.Update(cluster); err != nil {
		return nil, fmt.Errorf("更新集群失败: %w", err)
	}

	return cluster, nil
}

// DeleteCluster 删除集群
func (b *ClusterBiz) DeleteCluster(ctx context.Context, id uint) error {
	// 检查集群是否存在
	_, err := b.repo.GetByID(id)
	if err != nil {
		return errors.New("集群不存在")
	}

	return b.repo.Delete(id)
}

// GetCluster 获取集群详情
func (b *ClusterBiz) GetCluster(ctx context.Context, id uint) (*models.Cluster, error) {
	cluster, err := b.repo.GetByID(id)
	if err != nil {
		return nil, errors.New("集群不存在")
	}

	// 解密 KubeConfig（如果需要返回给前端）
	// 注意：实际使用时根据安全要求决定是否解密
	return cluster, nil
}

// ListClusters 获取集群列表
func (b *ClusterBiz) ListClusters(ctx context.Context) ([]models.Cluster, error) {
	return b.repo.List()
}

// TestClusterConnection 测试集群连接
func (b *ClusterBiz) TestClusterConnection(ctx context.Context, id uint) (string, error) {
	cluster, err := b.repo.GetByID(id)
	if err != nil {
		return "", errors.New("集群不存在")
	}

	_, version, err := b.repo.TestConnection(cluster)
	if err != nil {
		// 更新状态为失败
		b.repo.UpdateStatus(id, models.ClusterStatusFailed)
		return "", fmt.Errorf("连接失败: %w", err)
	}

	// 更新状态和版本
	b.repo.UpdateStatus(id, models.ClusterStatusNormal)
	b.repo.UpdateVersion(id, version)

	return version, nil
}

// GetClusterClientset 获取集群的 Kubernetes clientset
func (b *ClusterBiz) GetClusterClientset(ctx context.Context, id uint) (*kubernetes.Clientset, error) {
	cluster, err := b.repo.GetByID(id)
	if err != nil {
		return nil, errors.New("集群不存在")
	}

	clientset, _, err := b.repo.GetClientset(cluster)
	if err != nil {
		return nil, fmt.Errorf("获取集群 clientset 失败: %w", err)
	}

	return clientset, nil
}

// 加密密钥（实际生产环境应该从配置中心获取）
const encryptionKey = "opshub-k8s-encrypt-key-32bytes!!"

// encryptKubeConfig 加密 kubeconfig
func encryptKubeConfig(plainText string) (string, error) {
	key := []byte(encryptionKey)
	plaintext := []byte(plainText)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptKubeConfig 解密 kubeconfig（导出供其他包使用）
func DecryptKubeConfig(cipherText string) (string, error) {
	key := []byte(encryptionKey)
	ciphertext, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GetRepo 获取 repository 实例
func (b *ClusterBiz) GetRepo() *repository.ClusterRepository {
	return b.repo
}

// CreateClientsetFromKubeConfig 从 kubeconfig 字符串创建 clientset
// 这个方法用于创建基于用户凭据的 clientset
func CreateClientsetFromKubeConfig(kubeConfigContent string) (*kubernetes.Clientset, error) {
	// 需要导入 k8s.io/client-go/tools/clientcmd
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfigContent))
	if err != nil {
		return nil, fmt.Errorf("从 kubeconfig 创建配置失败: %w", err)
	}

	// 创建 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("创建 clientset 失败: %w", err)
	}

	return clientset, nil
}

// GetClusterRESTConfig 获取集群的 REST Config
func (b *ClusterBiz) GetClusterRESTConfig(ctx context.Context, id uint) (*rest.Config, error) {
	cluster, err := b.repo.GetByID(id)
	if err != nil {
		return nil, errors.New("集群不存在")
	}

	_, restConfig, err := b.repo.GetClientset(cluster)
	if err != nil {
		return nil, fmt.Errorf("获取集群 REST Config 失败: %w", err)
	}

	return restConfig, nil
}

// CreateRESTConfigFromKubeConfig 从 kubeconfig 字符串创建 REST Config
func CreateRESTConfigFromKubeConfig(kubeConfigContent string) (*rest.Config, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfigContent))
	if err != nil {
		return nil, fmt.Errorf("从 kubeconfig 创建 REST config 失败: %w", err)
	}
	return config, nil
}
