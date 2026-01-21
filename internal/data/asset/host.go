package asset

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"gorm.io/gorm"
)

type hostRepo struct {
	db *gorm.DB
}

// NewHostRepo 创建主机仓库
func NewHostRepo(db *gorm.DB) asset.HostRepo {
	return &hostRepo{db: db}
}

// Create 创建主机
func (r *hostRepo) Create(ctx context.Context, host *asset.Host) error {
	return r.db.WithContext(ctx).Create(host).Error
}

// CreateOrUpdate 创建或恢复主机（如果IP已被软删除则恢复）
func (r *hostRepo) CreateOrUpdate(ctx context.Context, host *asset.Host) error {
	// 先查找是否存在该IP的记录（包括软删除的）
	var existing asset.Host
	err := r.db.WithContext(ctx).Unscoped().Where("ip = ?", host.IP).First(&existing).Error

	if err == nil {
		// 找到了记录
		if existing.DeletedAt.Valid {
			// 记录已被软删除，恢复它
			existing.Name = host.Name
			existing.GroupID = host.GroupID
			existing.SSHUser = host.SSHUser
			existing.IP = host.IP
			existing.Port = host.Port
			existing.CredentialID = host.CredentialID
			existing.Tags = host.Tags
			existing.Description = host.Description
			existing.Status = host.Status
			existing.DeletedAt.Time = *new(time.Time) // 清除删除时间
			existing.DeletedAt.Valid = false
			return r.db.WithContext(ctx).Unscoped().Save(&existing).Error
		}
		// 记录未被删除，返回错误
		return fmt.Errorf("IP地址 %s 已存在", host.IP)
	}

	// 没找到记录，创建新的
	return r.db.WithContext(ctx).Create(host).Error
}

// Update 更新主机
func (r *hostRepo) Update(ctx context.Context, host *asset.Host) error {
	return r.db.WithContext(ctx).Save(host).Error
}

// Delete 删除主机
func (r *hostRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&asset.Host{}, id).Error
}

// GetByID 根据ID获取主机
func (r *hostRepo) GetByID(ctx context.Context, id uint) (*asset.Host, error) {
	var host asset.Host
	err := r.db.WithContext(ctx).First(&host, id).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// List 列表查询
func (r *hostRepo) List(ctx context.Context, page, pageSize int, keyword string, groupIDs []uint, accessibleHostIDs []uint) ([]*asset.Host, int64, error) {
	var hosts []*asset.Host
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.Host{})

	if keyword != "" {
		query = query.Where("name LIKE ? OR ip LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 添加分组ID筛选（支持多个分组ID）
	if len(groupIDs) > 0 {
		query = query.Where("group_id IN ?", groupIDs)
	}

	// 添加可访问主机ID筛选
	// 如果 accessibleHostIDs 为空切片（非nil），表示用户没有任何权限，应该返回空列表
	// 如果 accessibleHostIDs 为nil，表示不进行权限筛选（管理员或未启用权限控制）
	if accessibleHostIDs != nil {
		if len(accessibleHostIDs) == 0 {
			// 用户没有任何主机访问权限，返回空列表
			return []*asset.Host{}, 0, nil
		}
		query = query.Where("id IN ?", accessibleHostIDs)
	}

	err := query.Order("id DESC").Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&hosts).Error
	if err != nil {
		return nil, 0, err
	}

	return hosts, total, nil
}

// GetByGroupID 根据分组ID获取主机列表
func (r *hostRepo) GetByGroupID(ctx context.Context, groupID uint) ([]*asset.Host, error) {
	var hosts []*asset.Host
	err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&hosts).Error
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

// GetByIP 根据IP获取主机
func (r *hostRepo) GetByIP(ctx context.Context, ip string) (*asset.Host, error) {
	var host asset.Host
	err := r.db.WithContext(ctx).Where("ip = ?", ip).First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// GetByCloudInstanceID 根据云实例ID获取主机
func (r *hostRepo) GetByCloudInstanceID(ctx context.Context, instanceID string) (*asset.Host, error) {
	var host asset.Host
	err := r.db.WithContext(ctx).Where("cloud_instance_id = ?", instanceID).First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// CountByCredentialID 统计使用指定凭证的主机数量
func (r *hostRepo) CountByCredentialID(ctx context.Context, credentialID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&asset.Host{}).Where("credential_id = ?", credentialID).Count(&count).Error
	return count, err
}

// credentialRepo 凭证仓库
type credentialRepo struct {
	db           *gorm.DB
	encryptionKey []byte
}

// NewCredentialRepo 创建凭证仓库
func NewCredentialRepo(db *gorm.DB) asset.CredentialRepo {
	// AES-256要求密钥长度必须是32字节（256位）
	encryptionKey := []byte("opshub-enc-key-32-bytes-long!!!!")
	return &credentialRepo{
		db:           db,
		encryptionKey: encryptionKey,
	}
}

// encrypt 加密
func (r *credentialRepo) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(r.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 解密
func (r *credentialRepo) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(r.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// Create 创建凭证
func (r *credentialRepo) Create(ctx context.Context, credential *asset.Credential) error {
	// 加密敏感信息
	if credential.Password != "" {
		encrypted, err := r.encrypt(credential.Password)
		if err != nil {
			return fmt.Errorf("加密密码失败: %w", err)
		}
		credential.Password = encrypted
	}

	if credential.PrivateKey != "" {
		encrypted, err := r.encrypt(credential.PrivateKey)
		if err != nil {
			return fmt.Errorf("加密私钥失败: %w", err)
		}
		credential.PrivateKey = encrypted
	}

	if credential.Passphrase != "" {
		encrypted, err := r.encrypt(credential.Passphrase)
		if err != nil {
			return fmt.Errorf("加密私钥密码失败: %w", err)
		}
		credential.Passphrase = encrypted
	}

	return r.db.WithContext(ctx).Create(credential).Error
}

// Update 更新凭证
func (r *credentialRepo) Update(ctx context.Context, credential *asset.Credential) error {
	// 加密敏感信息
	if credential.Password != "" {
		encrypted, err := r.encrypt(credential.Password)
		if err != nil {
			return fmt.Errorf("加密密码失败: %w", err)
		}
		credential.Password = encrypted
	}

	if credential.PrivateKey != "" {
		encrypted, err := r.encrypt(credential.PrivateKey)
		if err != nil {
			return fmt.Errorf("加密私钥失败: %w", err)
		}
		credential.PrivateKey = encrypted
	}

	if credential.Passphrase != "" {
		encrypted, err := r.encrypt(credential.Passphrase)
		if err != nil {
			return fmt.Errorf("加密私钥密码失败: %w", err)
		}
		credential.Passphrase = encrypted
	}

	return r.db.WithContext(ctx).Save(credential).Error
}

// Delete 删除凭证
func (r *credentialRepo) Delete(ctx context.Context, id uint) error {
	// 检查是否有主机使用此凭证
	var count int64
	if err := r.db.WithContext(ctx).Model(&asset.Host{}).Where("credential_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该凭证正在被 %d 个主机使用，无法删除", count)
	}

	return r.db.WithContext(ctx).Delete(&asset.Credential{}, id).Error
}

// GetByID 根据ID获取凭证
func (r *credentialRepo) GetByID(ctx context.Context, id uint) (*asset.Credential, error) {
	var credential asset.Credential
	err := r.db.WithContext(ctx).First(&credential, id).Error
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

// GetByIDDecrypted 根据ID获取凭证（解密后的）
func (r *credentialRepo) GetByIDDecrypted(ctx context.Context, id uint) (*asset.Credential, error) {
	credential, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 解密敏感信息
	if credential.Password != "" {
		decrypted, err := r.decrypt(credential.Password)
		if err != nil {
			return nil, fmt.Errorf("解密密码失败: %w", err)
		}
		credential.Password = decrypted
	}

	if credential.PrivateKey != "" {
		decrypted, err := r.decrypt(credential.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("解密私钥失败: %w", err)
		}
		credential.PrivateKey = decrypted
	}

	if credential.Passphrase != "" {
		decrypted, err := r.decrypt(credential.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("解密私钥密码失败: %w", err)
		}
		credential.Passphrase = decrypted
	}

	return credential, nil
}

// List 列表查询
func (r *credentialRepo) List(ctx context.Context, page, pageSize int, keyword string) ([]*asset.Credential, int64, error) {
	var credentials []*asset.Credential
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.Credential{})

	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	err := query.Order("id DESC").Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&credentials).Error
	if err != nil {
		return nil, 0, err
	}

	return credentials, total, nil
}

// GetAll 获取所有凭证
func (r *credentialRepo) GetAll(ctx context.Context) ([]*asset.Credential, error) {
	var credentials []*asset.Credential
	err := r.db.WithContext(ctx).Order("id DESC").Find(&credentials).Error
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

// cloudAccountRepo 云平台账号仓库
type cloudAccountRepo struct {
	db *gorm.DB
}

// NewCloudAccountRepo 创建云平台账号仓库
func NewCloudAccountRepo(db *gorm.DB) asset.CloudAccountRepo {
	return &cloudAccountRepo{db: db}
}

// Create 创建云平台账号
func (r *cloudAccountRepo) Create(ctx context.Context, account *asset.CloudAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
}

// Update 更新云平台账号
func (r *cloudAccountRepo) Update(ctx context.Context, account *asset.CloudAccount) error {
	return r.db.WithContext(ctx).Save(account).Error
}

// Delete 删除云平台账号
func (r *cloudAccountRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&asset.CloudAccount{}, id).Error
}

// GetByID 根据ID获取云平台账号
func (r *cloudAccountRepo) GetByID(ctx context.Context, id uint) (*asset.CloudAccount, error) {
	var account asset.CloudAccount
	err := r.db.WithContext(ctx).First(&account, id).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// List 列表查询
func (r *cloudAccountRepo) List(ctx context.Context, page, pageSize int) ([]*asset.CloudAccount, int64, error) {
	var accounts []*asset.CloudAccount
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.CloudAccount{})

	err := query.Order("id DESC").Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&accounts).Error
	if err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

// GetAll 获取所有启用的云平台账号
func (r *cloudAccountRepo) GetAll(ctx context.Context) ([]*asset.CloudAccount, error) {
	var accounts []*asset.CloudAccount
	err := r.db.WithContext(ctx).Order("id DESC").Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}
