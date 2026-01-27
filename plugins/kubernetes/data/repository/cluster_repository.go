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

package repository

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"

	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
)

// ClusterRepository 集群数据访问层
type ClusterRepository struct {
	db *gorm.DB
}

// NewClusterRepository 创建集群仓储
func NewClusterRepository(db *gorm.DB) *ClusterRepository {
	return &ClusterRepository{db: db}
}

// Create 创建集群
func (r *ClusterRepository) Create(cluster *models.Cluster) error {
	return r.db.Create(cluster).Error
}

// Update 更新集群
func (r *ClusterRepository) Update(cluster *models.Cluster) error {
	return r.db.Model(cluster).Select("*").Updates(cluster).Error
}

// Delete 删除集群（硬删除）
func (r *ClusterRepository) Delete(id uint) error {
	return r.db.Delete(&models.Cluster{}, id).Error
}

// GetByID 根据ID获取集群
func (r *ClusterRepository) GetByID(id uint) (*models.Cluster, error) {
	var cluster models.Cluster
	err := r.db.First(&cluster, id).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// GetByName 根据名称获取集群
func (r *ClusterRepository) GetByName(name string) (*models.Cluster, error) {
	var cluster models.Cluster
	err := r.db.Where("name = ?", name).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// List 获取集群列表
func (r *ClusterRepository) List() ([]models.Cluster, error) {
	var clusters []models.Cluster
	err := r.db.Order("created_at DESC").Find(&clusters).Error
	return clusters, err
}

// ListByProvider 根据服务商获取集群列表
func (r *ClusterRepository) ListByProvider(provider string) ([]models.Cluster, error) {
	var clusters []models.Cluster
	err := r.db.Where("provider = ?", provider).Order("created_at DESC").Find(&clusters).Error
	return clusters, err
}

// ListByStatus 根据状态获取集群列表
func (r *ClusterRepository) ListByStatus(status int) ([]models.Cluster, error) {
	var clusters []models.Cluster
	err := r.db.Where("status = ?", status).Order("created_at DESC").Find(&clusters).Error
	return clusters, err
}

// UpdateStatus 更新集群状态
func (r *ClusterRepository) UpdateStatus(id uint, status int) error {
	return r.db.Model(&models.Cluster{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateVersion 更新集群版本
func (r *ClusterRepository) UpdateVersion(id uint, version string) error {
	return r.db.Model(&models.Cluster{}).Where("id = ?", id).Update("version", version).Error
}

// TestConnection 测试集群连接
func (r *ClusterRepository) TestConnection(cluster *models.Cluster) (*kubernetes.Clientset, string, error) {
	var kubeConfig string

	// 尝试解密，如果失败说明可能是未加密的（用于创建时的测试连接）
	decryptedConfig, decryptErr := decryptKubeConfig(cluster.KubeConfig)
	if decryptErr != nil {
		// 解密失败，直接使用原始值（可能是创建集群时传入的未加密数据）
		kubeConfig = cluster.KubeConfig
	} else {
		// 解密成功，使用解密后的值
		kubeConfig = decryptedConfig
	}

	// 从 kubeConfig 创建配置
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		return nil, "", err
	}

	// 创建 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, "", err
	}

	// 获取服务器版本
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, "", err
	}

	return clientset, version.GitVersion, nil
}

// GetClientset 获取集群的 clientset
func (r *ClusterRepository) GetClientset(cluster *models.Cluster) (*kubernetes.Clientset, *rest.Config, error) {
	var kubeConfig string

	// 尝试解密，如果失败说明可能是未加密的（用于创建时的测试连接）
	decryptedConfig, decryptErr := decryptKubeConfig(cluster.KubeConfig)
	if decryptErr != nil {
		// 解密失败，直接使用原始值
		kubeConfig = cluster.KubeConfig
	} else {
		// 解密成功，使用解密后的值
		kubeConfig = decryptedConfig
	}

	// 从 kubeConfig 创建配置
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		return nil, nil, err
	}

	// 创建 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return clientset, config, nil
}

// 加密密钥（必须与 biz 包中的密钥相同）
const encryptionKey = "opshub-k8s-encrypt-key-32bytes!!"

// decryptKubeConfig 解密 kubeconfig（内部使用）
func decryptKubeConfig(cipherText string) (string, error) {
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
