package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/app-inventory/model"
	"gorm.io/gorm"
)

type CredentialSecret struct {
	Password   string            `json:"password,omitempty"`
	Token      string            `json:"token,omitempty"`
	AccessKey  string            `json:"accessKey,omitempty"`
	SecretKey  string            `json:"secretKey,omitempty"`
	PrivateKey string            `json:"privateKey,omitempty"`
	Passphrase string            `json:"passphrase,omitempty"`
	Extra      map[string]string `json:"extra,omitempty"`
}

func (s CredentialSecret) Empty() bool {
	return strings.TrimSpace(s.Password) == "" && strings.TrimSpace(s.Token) == "" && strings.TrimSpace(s.AccessKey) == "" && strings.TrimSpace(s.SecretKey) == "" && strings.TrimSpace(s.PrivateKey) == "" && strings.TrimSpace(s.Passphrase) == "" && len(s.Extra) == 0
}

type CredentialInput struct {
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	Username    string            `json:"username"`
	Secret      *CredentialSecret `json:"secret"`
	Scope       string            `json:"scope"`
	Status      string            `json:"status"`
	Description string            `json:"description"`
	OwnerUserID uint              `json:"ownerUserId"`
	ExpiresAt   *time.Time        `json:"expiresAt"`
}

type CredentialView struct {
	model.Credential
	HasSecret  bool   `json:"hasSecret"`
	SecretMask string `json:"secretMask"`
	CanReveal  bool   `json:"canReveal"`
	CanManage  bool   `json:"canManage"`
	GrantCount int    `json:"grantCount"`
}

type CredentialGrantInput struct {
	SubjectType string `json:"subjectType"`
	SubjectID   uint   `json:"subjectId"`
	Permissions uint   `json:"permissions"`
}

type CredentialReveal struct {
	CredentialID uint             `json:"credentialId"`
	Name         string           `json:"name"`
	Username     string           `json:"username"`
	Secret       CredentialSecret `json:"secret"`
	RevealedAt   time.Time        `json:"revealedAt"`
}

func validateCredentialInput(in *CredentialInput, requireSecret bool) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Kind = strings.TrimSpace(in.Kind)
	in.Username = strings.TrimSpace(in.Username)
	in.Scope = strings.TrimSpace(in.Scope)
	in.Status = strings.TrimSpace(in.Status)
	in.Description = strings.TrimSpace(in.Description)
	if in.Name == "" || in.Kind == "" {
		return errors.New("凭据名称和类型不能为空")
	}
	if len([]rune(in.Name)) > 120 || len([]rune(in.Kind)) > 40 || len([]rune(in.Username)) > 255 || len([]rune(in.Description)) > 500 {
		return errors.New("凭据名称、类型、用户名或说明超过长度限制")
	}
	if requireSecret && (in.Secret == nil || in.Secret.Empty()) {
		return errors.New("至少填写一项凭据密文")
	}
	if in.Scope == "" {
		in.Scope = "private"
	}
	if in.Scope != "private" && in.Scope != "application" && in.Scope != "shared" {
		return errors.New("凭据共享范围无效")
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.Status != "active" && in.Status != "disabled" {
		return errors.New("凭据状态无效")
	}
	return nil
}

func (s *Service) CreateCredential(ctx context.Context, in *CredentialInput, userID uint) (*CredentialView, error) {
	if s.cipher == nil {
		return nil, errors.New("凭据加密密钥未配置，暂不能创建凭据")
	}
	if err := validateCredentialInput(in, true); err != nil {
		return nil, err
	}
	item := &model.Credential{Name: in.Name, Kind: in.Kind, Username: in.Username, KeyVersion: s.cipher.KeyVersion(), Scope: in.Scope, Status: in.Status, Description: in.Description, OwnerUserID: userID, ExpiresAt: in.ExpiresAt}
	if in.OwnerUserID > 0 {
		item.OwnerUserID = in.OwnerUserID
	}
	if err := s.ensureCredentialOwner(ctx, item.OwnerUserID); err != nil {
		return nil, err
	}
	ciphertext, err := s.sealCredentialSecret(in.Secret)
	if err != nil {
		return nil, err
	}
	item.SecretCiphertext = ciphertext
	if err := s.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, err
	}
	view := s.toCredentialView(item, credentialAvailable(item), true, 0)
	return &view, nil
}

func (s *Service) UpdateCredential(ctx context.Context, id uint, in *CredentialInput, userID uint) (*CredentialView, error) {
	if s.cipher == nil {
		return nil, errors.New("凭据加密密钥未配置，暂不能更新凭据")
	}
	var item model.Credential
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	if err := validateCredentialInput(in, false); err != nil {
		return nil, err
	}
	item.Name, item.Kind, item.Username = in.Name, in.Kind, in.Username
	item.Scope, item.Status, item.Description, item.ExpiresAt = in.Scope, in.Status, in.Description, in.ExpiresAt
	if in.OwnerUserID > 0 {
		item.OwnerUserID = in.OwnerUserID
	}
	if err := s.ensureCredentialOwner(ctx, item.OwnerUserID); err != nil {
		return nil, err
	}
	if in.Secret != nil && !in.Secret.Empty() {
		ciphertext, err := s.sealCredentialSecret(in.Secret)
		if err != nil {
			return nil, err
		}
		item.SecretCiphertext = ciphertext
		item.KeyVersion = s.cipher.KeyVersion()
		item.LastRotatedAt = timePtr(time.Now())
	}
	if err := s.db.WithContext(ctx).Save(&item).Error; err != nil {
		return nil, err
	}
	view := s.toCredentialView(&item, credentialAvailable(&item), true, 0)
	return &view, nil
}

func (s *Service) sealCredentialSecret(secret *CredentialSecret) (string, error) {
	if secret == nil || secret.Empty() {
		return "", errors.New("凭据密文不能为空")
	}
	payload, err := json.Marshal(secret)
	if err != nil {
		return "", err
	}
	ciphertext, err := s.cipher.Encrypt(payload)
	if err != nil {
		return "", err
	}
	return ciphertext, nil
}

func (s *Service) DeleteCredential(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("credential_id = ?", id).Delete(&model.CredentialGrant{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Resource{}).Where("credential_id = ?", id).Update("credential_id", 0).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Component{}).Where("credential_id = ?", id).Update("credential_id", 0).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Credential{}, id).Error
	})
}

func (s *Service) ListCredentials(ctx context.Context, opts ListOptions, userID uint) (Page[CredentialView], error) {
	page, pageSize := normalizePage(opts)
	q := s.db.WithContext(ctx).Model(&model.Credential{})
	admin := s.isAdmin(ctx, userID)
	if !admin {
		roleIDs := s.db.WithContext(ctx).Table("sys_user_role").Select("role_id").Where("user_id = ?", userID)
		grantedIDs := s.db.WithContext(ctx).Model(&model.CredentialGrant{}).
			Select("credential_id").
			Where("(permissions & ?) <> 0", credentialPermissionAll).
			Where("(subject_type = ? AND subject_id = ?) OR (subject_type = ? AND subject_id IN (?))", "user", userID, "role", roleIDs)
		q = q.Where("owner_user_id = ? OR id IN (?)", userID, grantedIDs)
	}
	if opts.Keyword != "" {
		like := "%" + strings.TrimSpace(opts.Keyword) + "%"
		q = q.Where("name LIKE ? OR kind LIKE ? OR username LIKE ?", like, like, like)
	}
	if opts.Kind != "" {
		q = q.Where("kind = ?", opts.Kind)
	}
	if opts.Status == "expiring" {
		now := time.Now()
		q = q.Where("expires_at IS NOT NULL AND expires_at >= ? AND expires_at <= ?", now, now.Add(30*24*time.Hour))
	} else if opts.Status != "" {
		q = q.Where("status = ?", opts.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return Page[CredentialView]{}, err
	}
	var items []model.Credential
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return Page[CredentialView]{}, err
	}
	permissions, err := s.credentialPermissions(ctx, items, userID, admin)
	if err != nil {
		return Page[CredentialView]{}, err
	}
	grantCounts, err := s.credentialGrantCounts(ctx, items)
	if err != nil {
		return Page[CredentialView]{}, err
	}
	views := make([]CredentialView, 0, len(items))
	for i := range items {
		permission := permissions[items[i].ID]
		views = append(views, s.toCredentialView(&items[i], credentialAvailable(&items[i]) && permission&model.CredentialPermissionReveal != 0, permission&model.CredentialPermissionManage != 0, grantCounts[items[i].ID]))
	}
	return Page[CredentialView]{List: views, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) toCredentialView(item *model.Credential, canReveal, canManage bool, grantCount int) CredentialView {
	return CredentialView{Credential: *item, HasSecret: strings.TrimSpace(item.SecretCiphertext) != "", SecretMask: "********", CanReveal: canReveal, CanManage: canManage, GrantCount: grantCount}
}

func (s *Service) RevealCredential(ctx context.Context, id, userID uint, username, ip, userAgent, reason string) (*CredentialReveal, error) {
	var item model.Credential
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	audit := &model.SecretAudit{CredentialID: id, UserID: userID, Username: username, Action: "reveal", Reason: strings.TrimSpace(reason), IP: ip, UserAgent: userAgent, CreatedAt: time.Now()}
	defer func() {
		_ = s.db.WithContext(context.Background()).Create(audit).Error
	}()
	if !s.canReveal(ctx, &item, userID) {
		audit.Success = false
		return nil, errors.New("没有查看该凭据明文的权限")
	}
	if !credentialAvailable(&item) {
		audit.Success = false
		return nil, errors.New("凭据已停用或已过期，不能查看明文")
	}
	if err := validateRevealReason(audit.Reason); err != nil {
		audit.Success = false
		return nil, err
	}
	if s.cipher == nil {
		audit.Success = false
		return nil, errors.New("凭据加密密钥未配置")
	}
	plaintext, err := s.cipher.Decrypt(item.SecretCiphertext)
	if err != nil {
		audit.Success = false
		return nil, fmt.Errorf("解密凭据失败: %w", err)
	}
	var secret CredentialSecret
	if err := json.Unmarshal(plaintext, &secret); err != nil {
		audit.Success = false
		return nil, fmt.Errorf("解析凭据密文失败: %w", err)
	}
	audit.Success = true
	return &CredentialReveal{CredentialID: item.ID, Name: item.Name, Username: item.Username, Secret: secret, RevealedAt: time.Now()}, nil
}

func (s *Service) canReveal(ctx context.Context, item *model.Credential, userID uint) bool {
	if userID == 0 || item == nil {
		return false
	}
	if s.isAdmin(ctx, userID) || item.OwnerUserID == userID {
		return true
	}
	return s.hasCredentialPermission(ctx, item.ID, userID, model.CredentialPermissionReveal)
}

const credentialPermissionAll = model.CredentialPermissionView | model.CredentialPermissionUse | model.CredentialPermissionReveal | model.CredentialPermissionManage

func credentialAvailable(item *model.Credential) bool {
	if item == nil || item.Status != "active" {
		return false
	}
	return item.ExpiresAt == nil || item.ExpiresAt.After(time.Now())
}

func validateRevealReason(reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("查看凭据明文必须填写理由")
	}
	if len([]rune(reason)) > 500 {
		return errors.New("查看理由不能超过500个字符")
	}
	return nil
}

func (s *Service) credentialPermissions(ctx context.Context, items []model.Credential, userID uint, admin bool) (map[uint]uint, error) {
	result := make(map[uint]uint, len(items))
	ids := make([]uint, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
		if admin || items[i].OwnerUserID == userID {
			result[items[i].ID] = credentialPermissionAll
		}
	}
	if admin || len(ids) == 0 {
		return result, nil
	}
	roleIDs := s.db.WithContext(ctx).Table("sys_user_role").Select("role_id").Where("user_id = ?", userID)
	var grants []model.CredentialGrant
	err := s.db.WithContext(ctx).
		Where("credential_id IN ?", ids).
		Where("(subject_type = ? AND subject_id = ?) OR (subject_type = ? AND subject_id IN (?))", "user", userID, "role", roleIDs).
		Find(&grants).Error
	if err != nil {
		return nil, err
	}
	for _, grant := range grants {
		result[grant.CredentialID] |= grant.Permissions
	}
	return result, nil
}

func (s *Service) credentialGrantCounts(ctx context.Context, items []model.Credential) (map[uint]int, error) {
	result := make(map[uint]int, len(items))
	ids := make([]uint, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	if len(ids) == 0 {
		return result, nil
	}
	var rows []struct {
		CredentialID uint
		Count        int
	}
	if err := s.db.WithContext(ctx).Model(&model.CredentialGrant{}).Select("credential_id, COUNT(*) AS count").Where("credential_id IN ?", ids).Group("credential_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.CredentialID] = row.Count
	}
	return result, nil
}

func (s *Service) hasCredentialPermission(ctx context.Context, credentialID, userID, permission uint) bool {
	roleIDs := s.db.WithContext(ctx).Table("sys_user_role").Select("role_id").Where("user_id = ?", userID)
	var count int64
	err := s.db.WithContext(ctx).Model(&model.CredentialGrant{}).
		Where("credential_id = ? AND (permissions & ?) <> 0", credentialID, permission).
		Where("(subject_type = ? AND subject_id = ?) OR (subject_type = ? AND subject_id IN (?))", "user", userID, "role", roleIDs).
		Count(&count).Error
	return err == nil && count > 0
}

func (s *Service) CanManageCredential(ctx context.Context, credentialID, userID uint) bool {
	var item model.Credential
	if err := s.db.WithContext(ctx).First(&item, credentialID).Error; err != nil {
		return false
	}
	return s.isAdmin(ctx, userID) || item.OwnerUserID == userID || s.hasCredentialPermission(ctx, item.ID, userID, model.CredentialPermissionManage)
}

func (s *Service) CanManageGrant(ctx context.Context, grantID, userID uint) bool {
	var grant model.CredentialGrant
	if err := s.db.WithContext(ctx).First(&grant, grantID).Error; err != nil {
		return false
	}
	return s.CanManageCredential(ctx, grant.CredentialID, userID)
}

func (s *Service) CanManageAnyCredential(ctx context.Context, userID uint) bool {
	if s.isAdmin(ctx, userID) {
		return true
	}
	var owned int64
	if err := s.db.WithContext(ctx).Model(&model.Credential{}).Where("owner_user_id = ?", userID).Count(&owned).Error; err == nil && owned > 0 {
		return true
	}
	roleIDs := s.db.WithContext(ctx).Table("sys_user_role").Select("role_id").Where("user_id = ?", userID)
	var granted int64
	err := s.db.WithContext(ctx).Model(&model.CredentialGrant{}).
		Where("(permissions & ?) <> 0", model.CredentialPermissionManage).
		Where("(subject_type = ? AND subject_id = ?) OR (subject_type = ? AND subject_id IN (?))", "user", userID, "role", roleIDs).
		Count(&granted).Error
	return err == nil && granted > 0
}

func (s *Service) IsAdmin(ctx context.Context, userID uint) bool { return s.isAdmin(ctx, userID) }

func (s *Service) isAdmin(ctx context.Context, userID uint) bool {
	if userID == 0 {
		return false
	}
	var count int64
	if err := s.db.WithContext(ctx).Table("sys_user_role ur").Joins("JOIN sys_role r ON r.id = ur.role_id AND r.deleted_at IS NULL").Where("ur.user_id = ? AND r.code = ?", userID, "admin").Count(&count).Error; err == nil && count > 0 {
		return true
	}
	var username string
	if s.db.WithContext(ctx).Table("sys_user").Where("id = ?", userID).Pluck("username", &username).Error == nil {
		return strings.EqualFold(username, "admin")
	}
	return false
}

func (s *Service) ListGrants(ctx context.Context, credentialID uint) ([]model.CredentialGrant, error) {
	var items []model.CredentialGrant
	if err := s.db.WithContext(ctx).Where("credential_id = ?", credentialID).Order("id").Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		var name string
		table := "sys_user"
		field := "username"
		if items[i].SubjectType == "role" {
			table, field = "sys_role", "name"
		}
		if s.db.WithContext(ctx).Table(table).Where("id = ?", items[i].SubjectID).Pluck(field, &name).Error == nil {
			items[i].SubjectName = name
		}
	}
	return items, nil
}

func (s *Service) UpsertGrant(ctx context.Context, credentialID uint, in *CredentialGrantInput, userID uint) (*model.CredentialGrant, error) {
	if err := validateCredentialGrantInput(in); err != nil {
		return nil, err
	}
	var credential model.Credential
	if err := s.db.WithContext(ctx).First(&credential, credentialID).Error; err != nil {
		return nil, err
	}
	if err := s.ensureGrantSubject(ctx, in.SubjectType, in.SubjectID); err != nil {
		return nil, err
	}
	var grant model.CredentialGrant
	err := s.db.WithContext(ctx).Where("credential_id = ? AND subject_type = ? AND subject_id = ?", credentialID, in.SubjectType, in.SubjectID).First(&grant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		grant = model.CredentialGrant{CredentialID: credentialID, SubjectType: in.SubjectType, SubjectID: in.SubjectID, Permissions: in.Permissions, CreatedBy: userID}
		if err := s.db.WithContext(ctx).Create(&grant).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		grant.Permissions = in.Permissions
		if err := s.db.WithContext(ctx).Save(&grant).Error; err != nil {
			return nil, err
		}
	}
	return &grant, nil
}

func validateCredentialGrantInput(in *CredentialGrantInput) error {
	if in == nil || (in.SubjectType != "user" && in.SubjectType != "role") {
		return errors.New("授权主体必须是用户或角色")
	}
	if in.SubjectID == 0 || in.Permissions == 0 {
		return errors.New("授权主体和权限不能为空")
	}
	if in.Permissions&^credentialPermissionAll != 0 {
		return errors.New("凭据授权包含无效权限")
	}
	return nil
}

func (s *Service) DeleteGrant(ctx context.Context, grantID uint) error {
	return s.db.WithContext(ctx).Delete(&model.CredentialGrant{}, grantID).Error
}

func (s *Service) ListSecretAudits(ctx context.Context, credentialID uint, limit int) ([]model.SecretAudit, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var items []model.SecretAudit
	q := s.db.WithContext(ctx).Order("id DESC").Limit(limit)
	if credentialID > 0 {
		q = q.Where("credential_id = ?", credentialID)
	}
	return items, q.Find(&items).Error
}

func (s *Service) ensureCredentialOwner(ctx context.Context, userID uint) error {
	if userID == 0 {
		return errors.New("凭据所有者不能为空")
	}
	return s.ensureGrantSubject(ctx, "user", userID)
}

func (s *Service) ensureGrantSubject(ctx context.Context, subjectType string, subjectID uint) error {
	table, label := "sys_user", "授权用户"
	if subjectType == "role" {
		table, label = "sys_role", "授权角色"
	}
	var count int64
	if err := s.db.WithContext(ctx).Table(table).Where("id = ? AND deleted_at IS NULL AND status = 1", subjectID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%s不存在或已停用", label)
	}
	return nil
}

func timePtr(value time.Time) *time.Time { return &value }
