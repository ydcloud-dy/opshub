package model

import (
	"time"

	"gorm.io/gorm"
)

// Application is the ownership root for every runtime and dependency asset.
type Application struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	Code             string         `gorm:"type:varchar(80);not null;index" json:"code"`
	Name             string         `gorm:"type:varchar(120);not null;index" json:"name"`
	Description      string         `gorm:"type:varchar(1000)" json:"description"`
	EnvironmentID    uint           `gorm:"not null;default:0;index" json:"environmentId"`
	OwnerName        string         `gorm:"type:varchar(120);index" json:"ownerName"`
	OwnerUserID      uint           `gorm:"index;default:0" json:"ownerUserId"`
	DepartmentID     uint           `gorm:"index;default:0" json:"departmentId"`
	Team             string         `gorm:"type:varchar(120);index" json:"team"`
	Criticality      string         `gorm:"type:varchar(20);not null;default:'medium';index" json:"criticality"`
	Status           string         `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	Lifecycle        string         `gorm:"type:varchar(20);not null;default:'production';index" json:"lifecycle"`
	HealthStatus     string         `gorm:"type:varchar(20);not null;default:'unknown';index" json:"healthStatus"`
	HealthCheckedAt  *time.Time     `gorm:"type:datetime;index" json:"healthCheckedAt"`
	HealthMessage    string         `gorm:"type:varchar(500)" json:"healthMessage"`
	HealthSource     string         `gorm:"type:varchar(40);not null;default:'asset-aggregation'" json:"healthSource"`
	RepositoryURL    string         `gorm:"type:varchar(500)" json:"repositoryUrl"`
	DocumentationURL string         `gorm:"type:varchar(500)" json:"documentationUrl"`
	Language         string         `gorm:"type:varchar(80)" json:"language"`
	Tags             string         `gorm:"type:text" json:"tags"`
	CreatedBy        uint           `gorm:"index;default:0" json:"createdBy"`
	UpdatedBy        uint           `gorm:"default:0" json:"updatedBy"`
	OwnerUsername    string         `gorm:"-" json:"ownerUsername,omitempty"`
	DepartmentName   string         `gorm:"-" json:"departmentName,omitempty"`
	EnvironmentName  string         `gorm:"-" json:"environmentName,omitempty"`
}

func (Application) TableName() string { return "app_inventory_applications" }

type Environment struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	Code             string         `gorm:"type:varchar(40);not null;index" json:"code"`
	Name             string         `gorm:"type:varchar(80);not null" json:"name"`
	Kind             string         `gorm:"type:varchar(20);not null;default:'production';index" json:"kind"`
	Region           string         `gorm:"type:varchar(100)" json:"region"`
	Status           string         `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	Description      string         `gorm:"type:varchar(500)" json:"description"`
	CreatedBy        uint           `gorm:"default:0" json:"createdBy"`
	UpdatedBy        uint           `gorm:"default:0" json:"updatedBy"`
	ApplicationCount int            `gorm:"-" json:"applicationCount"`
}

func (Environment) TableName() string { return "app_inventory_environments" }

type Domain struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	ApplicationID     uint           `gorm:"not null;index" json:"applicationId"`
	EnvironmentID     uint           `gorm:"index;default:0" json:"environmentId"`
	Domain            string         `gorm:"type:varchar(255);not null;index" json:"domain"`
	Protocol          string         `gorm:"type:varchar(10);not null;default:'https'" json:"protocol"`
	Port              int            `gorm:"not null;default:443" json:"port"`
	Path              string         `gorm:"type:varchar(255);not null;default:'/'" json:"path"`
	DNSProvider       string         `gorm:"type:varchar(80)" json:"dnsProvider"`
	CertificateID     uint           `gorm:"index;default:0" json:"certificateId"`
	IsPrimary         bool           `gorm:"type:tinyint(1);not null;default:0;index" json:"isPrimary"`
	Status            string         `gorm:"type:varchar(20);not null;default:'unknown';index" json:"status"`
	Source            string         `gorm:"type:varchar(20);not null;default:'manual'" json:"source"`
	Description       string         `gorm:"type:varchar(500)" json:"description"`
	LastCheckedAt     *time.Time     `gorm:"type:datetime;index" json:"lastCheckedAt"`
	ResponseTimeMS    int            `gorm:"not null;default:0" json:"responseTimeMs"`
	HTTPStatusCode    int            `gorm:"not null;default:0" json:"httpStatusCode"`
	ProbeMessage      string         `gorm:"type:varchar(500)" json:"probeMessage"`
	ResolvedAddress   string         `gorm:"type:varchar(255)" json:"resolvedAddress"`
	TLSExpiresAt      *time.Time     `gorm:"type:datetime;index" json:"tlsExpiresAt"`
	TLSIssuer         string         `gorm:"type:varchar(255)" json:"tlsIssuer"`
	CertificateName   string         `gorm:"-" json:"certificateName,omitempty"`
	CertificateStatus string         `gorm:"-" json:"certificateStatus,omitempty"`
	CertificateExpiry *time.Time     `gorm:"-" json:"certificateExpiry,omitempty"`
}

func (Domain) TableName() string { return "app_inventory_domains" }

type Resource struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	ApplicationID  uint           `gorm:"not null;index" json:"applicationId"`
	EnvironmentID  uint           `gorm:"index;default:0" json:"environmentId"`
	Kind           string         `gorm:"type:varchar(40);not null;index" json:"kind"`
	Name           string         `gorm:"type:varchar(180);not null;index" json:"name"`
	Address        string         `gorm:"type:varchar(500)" json:"address"`
	Port           int            `gorm:"default:0" json:"port"`
	HostID         uint           `gorm:"index;default:0" json:"hostId"`
	ClusterID      uint           `gorm:"index;default:0" json:"clusterId"`
	Namespace      string         `gorm:"type:varchar(120);index" json:"namespace"`
	ExternalID     string         `gorm:"type:varchar(500);index" json:"externalId"`
	CredentialID   uint           `gorm:"index;default:0" json:"credentialId"`
	Status         string         `gorm:"type:varchar(20);not null;default:'unknown';index" json:"status"`
	Source         string         `gorm:"type:varchar(20);not null;default:'manual';index" json:"source"`
	Metadata       string         `gorm:"type:longtext" json:"metadata"`
	Description    string         `gorm:"type:varchar(500)" json:"description"`
	LastSyncedAt   *time.Time     `gorm:"type:datetime" json:"lastSyncedAt"`
	LastCheckedAt  *time.Time     `gorm:"type:datetime;index" json:"lastCheckedAt"`
	ResponseTimeMS int            `gorm:"not null;default:0" json:"responseTimeMs"`
	HealthMessage  string         `gorm:"type:varchar(500)" json:"healthMessage"`
}

func (Resource) TableName() string { return "app_inventory_resources" }

// Component covers databases, middleware, caches, queues, search engines and external services.
type Component struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	ApplicationID  uint           `gorm:"not null;index" json:"applicationId"`
	EnvironmentID  uint           `gorm:"index;default:0" json:"environmentId"`
	Category       string         `gorm:"type:varchar(30);not null;index" json:"category"`
	Type           string         `gorm:"type:varchar(60);not null;index" json:"type"`
	Name           string         `gorm:"type:varchar(150);not null;index" json:"name"`
	Address        string         `gorm:"type:varchar(500)" json:"address"`
	Port           int            `gorm:"default:0" json:"port"`
	DatabaseName   string         `gorm:"type:varchar(120)" json:"databaseName"`
	Version        string         `gorm:"type:varchar(80)" json:"version"`
	CredentialID   uint           `gorm:"index;default:0" json:"credentialId"`
	TLSEnabled     bool           `gorm:"type:tinyint(1);not null;default:0" json:"tlsEnabled"`
	Status         string         `gorm:"type:varchar(20);not null;default:'unknown';index" json:"status"`
	Source         string         `gorm:"type:varchar(20);not null;default:'manual'" json:"source"`
	Metadata       string         `gorm:"type:longtext" json:"metadata"`
	Description    string         `gorm:"type:varchar(500)" json:"description"`
	LastCheckedAt  *time.Time     `gorm:"type:datetime;index" json:"lastCheckedAt"`
	ResponseTimeMS int            `gorm:"not null;default:0" json:"responseTimeMs"`
	HealthMessage  string         `gorm:"type:varchar(500)" json:"healthMessage"`
}

func (Component) TableName() string { return "app_inventory_components" }

type Dependency struct {
	ID                  uint           `gorm:"primarykey" json:"id"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
	SourceApplicationID uint           `gorm:"not null;index" json:"sourceApplicationId"`
	SourceEnvironmentID uint           `gorm:"index;default:0" json:"sourceEnvironmentId"`
	TargetApplicationID uint           `gorm:"index;default:0" json:"targetApplicationId"`
	TargetComponentID   uint           `gorm:"index;default:0" json:"targetComponentId"`
	TargetResourceID    uint           `gorm:"index;default:0" json:"targetResourceId"`
	TargetName          string         `gorm:"type:varchar(180)" json:"targetName"`
	RelationType        string         `gorm:"type:varchar(30);not null;index" json:"relationType"`
	Protocol            string         `gorm:"type:varchar(30)" json:"protocol"`
	Endpoint            string         `gorm:"type:varchar(500)" json:"endpoint"`
	Port                int            `gorm:"default:0" json:"port"`
	Criticality         string         `gorm:"type:varchar(20);not null;default:'medium'" json:"criticality"`
	Status              string         `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	Description         string         `gorm:"type:varchar(500)" json:"description"`
}

func (Dependency) TableName() string { return "app_inventory_dependencies" }

type Credential struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	Name             string         `gorm:"type:varchar(120);not null;index" json:"name"`
	Kind             string         `gorm:"type:varchar(40);not null;index" json:"kind"`
	Username         string         `gorm:"type:varchar(255)" json:"username"`
	SecretCiphertext string         `gorm:"type:longtext;not null" json:"-"`
	KeyVersion       string         `gorm:"type:varchar(32);not null" json:"keyVersion"`
	Scope            string         `gorm:"type:varchar(20);not null;default:'private';index" json:"scope"`
	Status           string         `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	Description      string         `gorm:"type:varchar(500)" json:"description"`
	OwnerUserID      uint           `gorm:"not null;index" json:"ownerUserId"`
	LastRotatedAt    *time.Time     `gorm:"type:datetime" json:"lastRotatedAt"`
	ExpiresAt        *time.Time     `gorm:"type:datetime;index" json:"expiresAt"`
}

func (Credential) TableName() string { return "app_inventory_credentials" }

const (
	CredentialPermissionView uint = 1 << iota
	CredentialPermissionUse
	CredentialPermissionReveal
	CredentialPermissionManage
)

type CredentialGrant struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	CredentialID uint      `gorm:"not null;index:idx_credential_subject;uniqueIndex:uidx_app_inventory_credential_subject,priority:1" json:"credentialId"`
	SubjectType  string    `gorm:"type:varchar(20);not null;index:idx_credential_subject;uniqueIndex:uidx_app_inventory_credential_subject,priority:2" json:"subjectType"`
	SubjectID    uint      `gorm:"not null;index:idx_credential_subject;uniqueIndex:uidx_app_inventory_credential_subject,priority:3" json:"subjectId"`
	Permissions  uint      `gorm:"not null;default:1" json:"permissions"`
	CreatedBy    uint      `gorm:"not null;default:0" json:"createdBy"`
	SubjectName  string    `gorm:"-" json:"subjectName,omitempty"`
}

func (CredentialGrant) TableName() string { return "app_inventory_credential_grants" }

type SecretAudit struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CredentialID uint      `gorm:"not null;index" json:"credentialId"`
	UserID       uint      `gorm:"not null;index" json:"userId"`
	Username     string    `gorm:"type:varchar(80);index" json:"username"`
	Action       string    `gorm:"type:varchar(30);not null;index" json:"action"`
	Success      bool      `gorm:"type:tinyint(1);not null;default:0;index" json:"success"`
	Reason       string    `gorm:"type:varchar(500)" json:"reason"`
	IP           string    `gorm:"type:varchar(80)" json:"ip"`
	UserAgent    string    `gorm:"type:varchar(500)" json:"userAgent"`
	CreatedAt    time.Time `gorm:"index" json:"createdAt"`
}

func (SecretAudit) TableName() string { return "app_inventory_secret_audits" }

type DiscoveryRun struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	SourceType    string    `gorm:"type:varchar(30);not null;index" json:"sourceType"`
	SourceID      uint      `gorm:"not null;index" json:"sourceId"`
	ApplicationID uint      `gorm:"not null;index" json:"applicationId"`
	EnvironmentID uint      `gorm:"index;default:0" json:"environmentId"`
	Namespace     string    `gorm:"type:varchar(120)" json:"namespace"`
	Selector      string    `gorm:"type:varchar(500)" json:"selector"`
	Status        string    `gorm:"type:varchar(20);not null;index" json:"status"`
	ResourceCount int       `gorm:"not null;default:0" json:"resourceCount"`
	DomainCount   int       `gorm:"not null;default:0" json:"domainCount"`
	ErrorMessage  string    `gorm:"type:text" json:"errorMessage"`
	CreatedBy     uint      `gorm:"not null;default:0" json:"createdBy"`
	StartedAt     time.Time `gorm:"type:datetime;not null;index" json:"startedAt"`
	FinishedAt    time.Time `gorm:"type:datetime" json:"finishedAt"`
}

func (DiscoveryRun) TableName() string { return "app_inventory_discovery_runs" }

func AllModels() []interface{} {
	return []interface{}{
		&Application{},
		&Environment{},
		&Domain{},
		&Resource{},
		&Component{},
		&Dependency{},
		&Credential{},
		&CredentialGrant{},
		&SecretAudit{},
		&DiscoveryRun{},
	}
}
