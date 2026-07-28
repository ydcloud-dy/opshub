package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/ydcloud-dy/opshub/plugins/app-inventory/model"
	"gorm.io/gorm"
)

// applicationReferences contains authoritative directory references resolved on the server.
type applicationReferences struct {
	OwnerName      string
	OwnerUsername  string
	DepartmentID   uint
	DepartmentName string
	Environment    model.Environment
}

func (s *Service) resolveApplicationReferences(ctx context.Context, in *ApplicationInput) (applicationReferences, error) {
	var result applicationReferences
	var user struct {
		ID           uint
		Username     string
		RealName     string
		DepartmentID uint `gorm:"column:department_id"`
	}
	if err := s.db.WithContext(ctx).Table("sys_user").
		Select("id, username, real_name, department_id").
		Where("id = ? AND deleted_at IS NULL AND status = 1", in.OwnerUserID).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, errors.New("负责人不存在或已停用")
		}
		return result, err
	}
	result.OwnerUsername = strings.TrimSpace(user.Username)
	result.OwnerName = strings.TrimSpace(user.RealName)
	if result.OwnerName == "" {
		result.OwnerName = result.OwnerUsername
	}

	departmentID := user.DepartmentID
	if departmentID == 0 {
		return result, errors.New("负责人尚未关联有效部门，请先完善平台用户资料")
	}
	var department struct {
		ID   uint
		Name string
	}
	if err := s.db.WithContext(ctx).Table("sys_department").
		Select("id, name").
		Where("id = ? AND deleted_at IS NULL AND status = 1", departmentID).
		First(&department).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, errors.New("所属部门不存在或已停用")
		}
		return result, err
	}
	result.DepartmentID = department.ID
	result.DepartmentName = strings.TrimSpace(department.Name)
	in.DepartmentID = result.DepartmentID

	if in.EnvironmentID == 0 {
		return result, errors.New("运行环境不能为空")
	}
	if err := s.db.WithContext(ctx).Where("id = ? AND status = ?", in.EnvironmentID, "active").First(&result.Environment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, errors.New("运行环境不存在或已停用")
		}
		return result, err
	}
	return result, nil
}

func applyApplicationReferences(app *model.Application, refs applicationReferences) {
	app.OwnerName = refs.OwnerName
	app.OwnerUsername = refs.OwnerUsername
	app.DepartmentID = refs.DepartmentID
	app.DepartmentName = refs.DepartmentName
	app.Team = refs.DepartmentName
	app.EnvironmentName = refs.Environment.Name
}

func lifecycleFromEnvironment(kind string) string {
	if oneOf(kind, "production", "staging", "test", "development") {
		return kind
	}
	return "production"
}

func (s *Service) enrichApplications(ctx context.Context, apps []model.Application) error {
	if len(apps) == 0 {
		return nil
	}
	userIDs := make([]uint, 0, len(apps))
	departmentIDs := make([]uint, 0, len(apps))
	environmentIDs := make([]uint, 0, len(apps))
	for _, app := range apps {
		if app.OwnerUserID > 0 {
			userIDs = append(userIDs, app.OwnerUserID)
		}
		if app.DepartmentID > 0 {
			departmentIDs = append(departmentIDs, app.DepartmentID)
		}
		if app.EnvironmentID > 0 {
			environmentIDs = append(environmentIDs, app.EnvironmentID)
		}
	}
	type directoryUser struct {
		username     string
		realName     string
		departmentID uint
	}
	users := map[uint]directoryUser{}
	if len(userIDs) > 0 {
		var rows []struct {
			ID           uint
			Username     string
			RealName     string
			DepartmentID uint
		}
		if err := s.db.WithContext(ctx).Table("sys_user").Select("id, username, real_name, department_id").Where("id IN ? AND deleted_at IS NULL", userIDs).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			users[row.ID] = directoryUser{username: row.Username, realName: row.RealName, departmentID: row.DepartmentID}
			if row.DepartmentID > 0 {
				departmentIDs = append(departmentIDs, row.DepartmentID)
			}
		}
	}
	departments := map[uint]string{}
	if len(departmentIDs) > 0 {
		var rows []struct {
			ID   uint
			Name string
		}
		if err := s.db.WithContext(ctx).Table("sys_department").Select("id, name").Where("id IN ?", departmentIDs).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			departments[row.ID] = row.Name
		}
	}
	environments := map[uint]model.Environment{}
	if len(environmentIDs) > 0 {
		var rows []model.Environment
		if err := s.db.WithContext(ctx).Where("id IN ?", environmentIDs).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			environments[row.ID] = row
		}
	}
	for i := range apps {
		if user, ok := users[apps[i].OwnerUserID]; ok {
			apps[i].OwnerUsername = user.username
			apps[i].OwnerName = strings.TrimSpace(user.realName)
			if apps[i].OwnerName == "" {
				apps[i].OwnerName = user.username
			}
			apps[i].DepartmentID = user.departmentID
		}
		if name, ok := departments[apps[i].DepartmentID]; ok {
			apps[i].DepartmentName = name
			apps[i].Team = name
		}
		if env, ok := environments[apps[i].EnvironmentID]; ok {
			apps[i].EnvironmentName = env.Name
		}
	}
	return nil
}

func (s *Service) applicationEnvironmentID(ctx context.Context, appID uint) (uint, error) {
	if appID == 0 {
		return 0, errors.New("应用不能为空")
	}
	var app model.Application
	if err := s.db.WithContext(ctx).Select("id, environment_id").First(&app, appID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("应用不存在")
		}
		return 0, err
	}
	if app.EnvironmentID == 0 {
		return 0, errors.New("该应用尚未关联运行环境")
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Environment{}).Where("id = ? AND status = ?", app.EnvironmentID, "active").Count(&count).Error; err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, errors.New("该应用关联的运行环境不存在或已停用")
	}
	return app.EnvironmentID, nil
}

func syncApplicationEnvironment(tx *gorm.DB, appID, environmentID uint) error {
	for _, item := range []interface{}{&model.Domain{}, &model.Resource{}, &model.Component{}} {
		if err := tx.Model(item).Where("application_id = ?", appID).Update("environment_id", environmentID).Error; err != nil {
			return err
		}
	}
	if err := tx.Model(&model.Dependency{}).Where("source_application_id = ?", appID).Update("source_environment_id", environmentID).Error; err != nil {
		return err
	}
	return tx.Model(&model.DiscoveryRun{}).Where("application_id = ?", appID).Update("environment_id", environmentID).Error
}

func (s *Service) resolveResourceLocation(ctx context.Context, in *ResourceInput, userID uint) error {
	in.Kind = strings.TrimSpace(in.Kind)
	if isKubernetesKind(in.Kind) {
		if in.ClusterID == 0 {
			return errors.New("Kubernetes 资源必须关联集群")
		}
		var cluster struct {
			ID     uint
			Name   string
			Status int
		}
		if err := s.db.WithContext(ctx).Table("k8s_clusters").Where("id = ?", in.ClusterID).First(&cluster).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("关联的 Kubernetes 集群不存在")
			}
			return err
		}
		if cluster.Status != 1 {
			return errors.New("关联的 Kubernetes 集群当前不可用")
		}
		accessibleIDs, err := s.accessibleClusterIDs(ctx, userID)
		if err != nil {
			return err
		}
		allowed := false
		for _, id := range accessibleIDs {
			if id == in.ClusterID {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.New("当前用户无权访问该 Kubernetes 集群")
		}
		if strings.TrimSpace(in.Namespace) == "" {
			return errors.New("Kubernetes 资源必须填写命名空间")
		}
		in.Address = ""
		in.HostID = 0
		return nil
	}
	if oneOf(in.Kind, "Host", "VirtualMachine") {
		if in.HostID == 0 {
			return errors.New("主机类资源必须选择关联主机")
		}
		if err := s.ensureHostAccess(ctx, in.HostID, userID); err != nil {
			return err
		}
		var host struct {
			ID     uint
			IP     string
			Status int
		}
		if err := s.db.WithContext(ctx).Table("hosts").Select("id, ip, status").Where("id = ? AND deleted_at IS NULL", in.HostID).First(&host).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("关联的主机不存在")
			}
			return err
		}
		if strings.TrimSpace(host.IP) == "" {
			return errors.New("关联主机没有有效 IP 地址")
		}
		// The host registry is authoritative; never persist a client-supplied address.
		in.Address = strings.TrimSpace(host.IP)
		in.ClusterID = 0
		in.Namespace = ""
		return nil
	}
	in.HostID = 0
	in.ClusterID = 0
	in.Namespace = ""
	if strings.TrimSpace(in.Address) == "" && strings.TrimSpace(in.ExternalID) == "" {
		return errors.New("该资源必须填写访问地址或外部标识")
	}
	return nil
}

func (s *Service) ensureHostAccess(ctx context.Context, hostID, userID uint) error {
	if hostID == 0 {
		return errors.New("主机不能为空")
	}
	var exists int64
	if err := s.db.WithContext(ctx).Table("hosts").Where("id = ? AND deleted_at IS NULL", hostID).Count(&exists).Error; err != nil {
		return err
	}
	if exists == 0 {
		return errors.New("关联的主机不存在")
	}
	if userID == 0 {
		return errors.New("当前用户无权访问该主机")
	}
	var admin int64
	if err := s.db.WithContext(ctx).Table("sys_user_role ur").
		Joins("JOIN sys_role r ON r.id = ur.role_id AND r.deleted_at IS NULL").
		Where("ur.user_id = ? AND r.code = ? AND r.status = 1", userID, "admin").Count(&admin).Error; err != nil {
		return err
	}
	if admin > 0 {
		return nil
	}
	var groupID uint
	if err := s.db.WithContext(ctx).Table("hosts").Where("id = ?", hostID).Pluck("group_id", &groupID).Error; err != nil {
		return err
	}
	var allowed int64
	query := s.db.WithContext(ctx).Table("sys_role_asset_permission p").
		Joins("JOIN sys_user_role ur ON ur.role_id = p.role_id").
		Where("ur.user_id = ? AND p.asset_group_id = ? AND p.deleted_at IS NULL", userID, groupID).
		Where("JSON_LENGTH(COALESCE(p.host_ids, JSON_ARRAY())) = 0 OR JSON_CONTAINS(p.host_ids, CAST(? AS JSON))", hostID)
	if err := query.Count(&allowed).Error; err != nil {
		return err
	}
	if allowed == 0 {
		return errors.New("当前用户无权访问该主机")
	}
	return nil
}

func (s *Service) accessibleHostIDs(ctx context.Context, userID uint) ([]uint, error) {
	if s.isAdmin(ctx, userID) {
		var ids []uint
		err := s.db.WithContext(ctx).Table("hosts").Where("deleted_at IS NULL").Pluck("id", &ids).Error
		return ids, err
	}
	var ids []uint
	err := s.db.WithContext(ctx).Raw(`
		SELECT DISTINCT h.id
		FROM hosts h
		JOIN sys_role_asset_permission p ON p.asset_group_id = h.group_id AND p.deleted_at IS NULL
		JOIN sys_user_role ur ON ur.role_id = p.role_id
		WHERE ur.user_id = ?
		  AND h.deleted_at IS NULL
		  AND (JSON_LENGTH(COALESCE(p.host_ids, JSON_ARRAY())) = 0 OR JSON_CONTAINS(p.host_ids, CAST(h.id AS JSON)))
	`, userID).Scan(&ids).Error
	return ids, err
}

func (s *Service) accessibleClusterIDs(ctx context.Context, userID uint) ([]uint, error) {
	if s.isAdmin(ctx, userID) {
		var ids []uint
		err := s.db.WithContext(ctx).Table("k8s_clusters").Pluck("id", &ids).Error
		return ids, err
	}
	seen := map[uint]bool{}
	var owned []uint
	if err := s.db.WithContext(ctx).Table("k8s_clusters").Where("created_by = ?", userID).Pluck("id", &owned).Error; err != nil {
		return nil, err
	}
	for _, id := range owned {
		seen[id] = true
	}
	if s.db.Migrator().HasTable("k8s_user_kube_configs") {
		var ids []uint
		if err := s.db.WithContext(ctx).Table("k8s_user_kube_configs").Where("user_id = ? AND is_active = 1", userID).Pluck("cluster_id", &ids).Error; err != nil {
			return nil, err
		}
		for _, id := range ids {
			seen[id] = true
		}
	}
	if s.db.Migrator().HasTable("k8s_user_role_bindings") {
		var ids []uint
		if err := s.db.WithContext(ctx).Table("k8s_user_role_bindings").Where("user_id = ?", userID).Pluck("cluster_id", &ids).Error; err != nil {
			return nil, err
		}
		for _, id := range ids {
			seen[id] = true
		}
	}
	ids := make([]uint, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Service) validateEnvironmentInput(in *EnvironmentInput) error {
	in.Code = strings.TrimSpace(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	in.Kind = strings.TrimSpace(in.Kind)
	in.Status = strings.TrimSpace(in.Status)
	if in.Code == "" || in.Name == "" {
		return errors.New("环境编码和名称不能为空")
	}
	if !regexpCode.MatchString(in.Code) || len(in.Code) > 40 || len(in.Name) > 80 {
		return errors.New("环境编码或名称格式无效")
	}
	if in.Kind == "" {
		in.Kind = "production"
	}
	if !oneOf(in.Kind, "production", "staging", "test", "development") {
		return errors.New("环境类型无效")
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if !oneOf(in.Status, "active", "disabled") {
		return errors.New("环境状态无效")
	}
	if len(strings.TrimSpace(in.Region)) > 100 || len(strings.TrimSpace(in.Description)) > 500 {
		return errors.New("环境资料超过长度限制")
	}
	return nil
}

var regexpCode = mustCompileCode()

func mustCompileCode() *regexp.Regexp {
	return regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
}

func (s *Service) validateAssetOwner(ctx context.Context, appID, envID uint) error {
	if err := s.ensureApplication(ctx, appID); err != nil {
		return err
	}
	var app model.Application
	if err := s.db.WithContext(ctx).Select("id, environment_id").First(&app, appID).Error; err != nil {
		return err
	}
	if app.EnvironmentID == 0 {
		return errors.New("应用尚未关联运行环境")
	}
	if envID > 0 && envID != app.EnvironmentID {
		return errors.New("环境不属于该应用")
	}
	return nil
}

func (s *Service) ensureApplication(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("应用不能为空")
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Application{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
