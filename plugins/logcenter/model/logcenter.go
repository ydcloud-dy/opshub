package model

import "time"

// StorageCluster stores metadata for OpsHub-owned log storage clusters.
type StorageCluster struct {
	ID                   uint       `gorm:"primarykey" json:"id"`
	Name                 string     `gorm:"type:varchar(120);not null;uniqueIndex" json:"name"`
	StorageType          string     `gorm:"type:varchar(32);not null;default:'clickhouse';index" json:"storageType"`
	Endpoints            string     `gorm:"type:text;not null" json:"endpoints"`
	DatabaseName         string     `gorm:"type:varchar(128);not null;default:'opshub_logs'" json:"databaseName"`
	Username             string     `gorm:"type:varchar(128)" json:"username"`
	PasswordEncrypted    string     `gorm:"type:text" json:"-"`
	PasswordConfigured   bool       `gorm:"-" json:"passwordConfigured"`
	SkipTLSVerify        bool       `gorm:"type:tinyint(1);not null;default:0" json:"skipTlsVerify"`
	Timeout              int        `gorm:"type:int;not null;default:300" json:"timeout"`
	QueueMode            string     `gorm:"type:varchar(32);not null;default:'direct'" json:"queueMode"`
	QueueEndpoints       string     `gorm:"type:text" json:"queueEndpoints"`
	QueueAuthEncrypted   string     `gorm:"type:text" json:"-"`
	DefaultRetentionDays int        `gorm:"type:int;not null;default:30" json:"defaultRetentionDays"`
	Status               string     `gorm:"type:varchar(20);not null;default:'unknown';index" json:"status"`
	LastTestAt           *time.Time `gorm:"type:datetime" json:"lastTestAt"`
	LastError            string     `gorm:"type:text" json:"lastError"`
	InitializedAt        *time.Time `gorm:"type:datetime" json:"initializedAt"`
	Enabled              bool       `gorm:"type:tinyint(1);not null;default:1;index" json:"enabled"`
	IsPrimary            bool       `gorm:"type:tinyint(1);not null;default:0;index" json:"isPrimary"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

func (StorageCluster) TableName() string {
	return "log_storage_clusters"
}

// QueryHistory records every log query execution.
type QueryHistory struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	UserID         uint      `gorm:"index" json:"userId"`
	DataSourceID   uint      `gorm:"not null;index" json:"datasourceId"`
	DataSourceType string    `gorm:"type:varchar(32);not null;index" json:"datasourceType"`
	QueryLanguage  string    `gorm:"type:varchar(32);not null" json:"queryLanguage"`
	Query          string    `gorm:"type:text;not null" json:"query"`
	Index          string    `gorm:"type:varchar(255)" json:"index"`
	StartTime      time.Time `gorm:"type:datetime" json:"startTime"`
	EndTime        time.Time `gorm:"type:datetime" json:"endTime"`
	Limit          int       `gorm:"type:int;not null;default:200" json:"limit"`
	DurationMS     int64     `gorm:"type:bigint;not null;default:0" json:"durationMs"`
	ResultCount    int       `gorm:"type:int;not null;default:0" json:"resultCount"`
	Status         string    `gorm:"type:varchar(20);not null;default:'success';index" json:"status"`
	ErrorMessage   string    `gorm:"type:text" json:"errorMessage"`
	SourceMode     string    `gorm:"type:varchar(32);not null;default:'external';index" json:"sourceMode"`
	AssetScope     string    `gorm:"type:text" json:"assetScope"`
	ScannedBytes   int64     `gorm:"type:bigint;not null;default:0" json:"scannedBytes"`
	CreatedAt      time.Time `gorm:"type:datetime;index" json:"createdAt"`
}

func (QueryHistory) TableName() string {
	return "log_query_histories"
}

// LogExportTask stores asynchronous ClickHouse export progress and artifacts.
type LogExportTask struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"userId"`
	StorageID      uint       `gorm:"not null;index" json:"storageId"`
	Format         string     `gorm:"type:varchar(16);not null;default:'ndjson'" json:"format"`
	QueryPayload   string     `gorm:"type:longtext;not null" json:"-"`
	Status         string     `gorm:"type:varchar(20);not null;default:'pending';index;index:idx_log_export_claim,priority:1" json:"status"`
	Progress       int        `gorm:"type:int;not null;default:0" json:"progress"`
	ExportedRows   int64      `gorm:"type:bigint;not null;default:0" json:"exportedRows"`
	MaxRows        int        `gorm:"type:int;not null;default:100000" json:"maxRows"`
	AttemptCount   int        `gorm:"type:int;not null;default:0" json:"attemptCount"`
	MaxAttempts    int        `gorm:"type:int;not null;default:3" json:"maxAttempts"`
	LeaseOwner     string     `gorm:"type:varchar(160);not null;default:''" json:"-"`
	LeaseExpiresAt *time.Time `gorm:"type:datetime;index" json:"-"`
	NextAttemptAt  *time.Time `gorm:"type:datetime;index;index:idx_log_export_claim,priority:2" json:"nextAttemptAt,omitempty"`
	FileName       string     `gorm:"type:varchar(255)" json:"fileName"`
	FilePath       string     `gorm:"type:text" json:"-"`
	FileSize       int64      `gorm:"type:bigint;not null;default:0" json:"fileSize"`
	ErrorMessage   string     `gorm:"type:text" json:"errorMessage"`
	StartedAt      *time.Time `gorm:"type:datetime" json:"startedAt"`
	CompletedAt    *time.Time `gorm:"type:datetime" json:"completedAt"`
	ExpiresAt      *time.Time `gorm:"type:datetime;index" json:"expiresAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (LogExportTask) TableName() string {
	return "log_export_tasks"
}

// QueryTemplate stores reusable structured filters for the built-in log store.
type QueryTemplate struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Name           string    `gorm:"type:varchar(120);not null;uniqueIndex" json:"name"`
	Category       string    `gorm:"type:varchar(64);index" json:"category"`
	DataSourceType string    `gorm:"type:varchar(32);not null;index" json:"datasourceType"`
	DataSourceID   uint      `gorm:"index;default:0" json:"datasourceId"`
	QueryLanguage  string    `gorm:"type:varchar(32);not null" json:"queryLanguage"`
	Query          string    `gorm:"type:text;not null" json:"query"`
	Index          string    `gorm:"type:varchar(255)" json:"index"`
	TimeRange      string    `gorm:"type:varchar(32)" json:"timeRange"`
	Variables      string    `gorm:"type:text" json:"variables"`
	Description    string    `gorm:"type:varchar(500)" json:"description"`
	IsPublic       bool      `gorm:"type:tinyint(1);not null;default:0;index" json:"isPublic"`
	OwnerID        uint      `gorm:"index;default:0" json:"ownerId"`
	Sort           int       `gorm:"type:int;not null;default:0" json:"sort"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (QueryTemplate) TableName() string {
	return "log_query_templates"
}

// SavedView stores a full query view layout for a user.
type SavedView struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Name           string    `gorm:"type:varchar(120);not null" json:"name"`
	UserID         uint      `gorm:"index" json:"userId"`
	DataSourceID   uint      `gorm:"index;default:0" json:"datasourceId"`
	QueryLanguage  string    `gorm:"type:varchar(32)" json:"queryLanguage"`
	Query          string    `gorm:"type:text" json:"query"`
	Index          string    `gorm:"type:varchar(255)" json:"index"`
	Filters        string    `gorm:"type:text" json:"filters"`
	Columns        string    `gorm:"type:text" json:"columns"`
	TimeRange      string    `gorm:"type:varchar(32)" json:"timeRange"`
	DisplayOptions string    `gorm:"type:text" json:"displayOptions"`
	IsPublic       bool      `gorm:"type:tinyint(1);not null;default:0;index" json:"isPublic"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (SavedView) TableName() string {
	return "log_saved_views"
}

// LibraryItem stores logical resources discovered in the built-in log storage.
type LibraryItem struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	DataSourceID   uint       `gorm:"not null;index:idx_log_library_source_name,priority:1" json:"datasourceId"`
	DataSourceType string     `gorm:"type:varchar(32);not null;index" json:"datasourceType"`
	ItemType       string     `gorm:"type:varchar(32);not null;index:idx_log_library_source_name,priority:2" json:"itemType"`
	Name           string     `gorm:"type:varchar(255);not null;index:idx_log_library_source_name,priority:3" json:"name"`
	DisplayName    string     `gorm:"type:varchar(255)" json:"displayName"`
	Description    string     `gorm:"type:varchar(500)" json:"description"`
	Owner          string     `gorm:"type:varchar(80)" json:"owner"`
	Environment    string     `gorm:"type:varchar(32);index" json:"environment"`
	RetentionDays  int        `gorm:"type:int;not null;default:0" json:"retentionDays"`
	DocCount       int64      `gorm:"type:bigint;not null;default:0" json:"docCount"`
	StoreSize      string     `gorm:"type:varchar(64)" json:"storeSize"`
	LastSyncAt     *time.Time `gorm:"type:datetime" json:"lastSyncAt"`
	Status         string     `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	RawMeta        string     `gorm:"type:text" json:"rawMeta"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (LibraryItem) TableName() string {
	return "log_library_items"
}

// FieldCatalog caches and annotates queryable fields.
type FieldCatalog struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	DataSourceID   uint      `gorm:"not null;index" json:"datasourceId"`
	LibraryItemID  uint      `gorm:"index" json:"libraryItemId"`
	FieldName      string    `gorm:"type:varchar(255);not null;index" json:"fieldName"`
	FieldType      string    `gorm:"type:varchar(64)" json:"fieldType"`
	DisplayName    string    `gorm:"type:varchar(255)" json:"displayName"`
	SampleValue    string    `gorm:"type:text" json:"sampleValue"`
	IsTimeField    bool      `gorm:"type:tinyint(1);not null;default:0" json:"isTimeField"`
	IsMessageField bool      `gorm:"type:tinyint(1);not null;default:0" json:"isMessageField"`
	IsLevelField   bool      `gorm:"type:tinyint(1);not null;default:0" json:"isLevelField"`
	IsSensitive    bool      `gorm:"type:tinyint(1);not null;default:0" json:"isSensitive"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (FieldCatalog) TableName() string {
	return "log_field_catalogs"
}

// AlertContext stores log samples and event jump configuration for monitor rules.
type AlertContext struct {
	ID                 uint      `gorm:"primarykey" json:"id"`
	RuleID             uint      `gorm:"not null;uniqueIndex" json:"ruleId"`
	SampleQuery        string    `gorm:"type:text" json:"sampleQuery"`
	SampleLimit        int       `gorm:"type:int;not null;default:5" json:"sampleLimit"`
	ContextBefore      int       `gorm:"type:int;not null;default:0" json:"contextBefore"`
	ContextAfter       int       `gorm:"type:int;not null;default:0" json:"contextAfter"`
	IncludeLogInNotice bool      `gorm:"type:tinyint(1);not null;default:1" json:"includeLogInNotice"`
	JumpTimeWindow     int       `gorm:"type:int;not null;default:900" json:"jumpTimeWindow"`
	HighlightKeywords  string    `gorm:"type:text" json:"highlightKeywords"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

func (AlertContext) TableName() string {
	return "log_alert_contexts"
}

// CollectorInstance stores observed collector runtime status.
type CollectorInstance struct {
	ID               uint       `gorm:"primarykey" json:"id"`
	ConfigID         uint       `gorm:"index;default:0" json:"configId"`
	InstanceID       string     `gorm:"type:varchar(160);not null;uniqueIndex" json:"instanceId"`
	AgentID          string     `gorm:"type:varchar(80);index" json:"agentId"`
	Mode             string     `gorm:"type:varchar(20);index" json:"mode"`
	HostID           uint       `gorm:"index;default:0" json:"hostId"`
	ClusterID        uint       `gorm:"index;default:0" json:"clusterId"`
	Hostname         string     `gorm:"type:varchar(255)" json:"hostname"`
	PodName          string     `gorm:"type:varchar(255)" json:"podName"`
	Namespace        string     `gorm:"type:varchar(120)" json:"namespace"`
	NodeName         string     `gorm:"type:varchar(255)" json:"nodeName"`
	CollectorType    string     `gorm:"type:varchar(32);index" json:"collectorType"`
	Version          string     `gorm:"type:varchar(64)" json:"version"`
	ConfigVersion    uint64     `gorm:"type:bigint unsigned;not null;default:0" json:"configVersion"`
	Status           string     `gorm:"type:varchar(20);not null;default:'offline';index" json:"status"`
	LastHeartbeatAt  *time.Time `gorm:"type:datetime" json:"lastHeartbeatAt"`
	LastIngestAt     *time.Time `gorm:"type:datetime" json:"lastIngestAt"`
	WALBytes         int64      `gorm:"type:bigint;not null;default:0" json:"walBytes"`
	InputEPS         float64    `gorm:"type:double;not null;default:0" json:"inputEps"`
	OutputEPS        float64    `gorm:"type:double;not null;default:0" json:"outputEps"`
	DroppedTotal     uint64     `gorm:"type:bigint unsigned;not null;default:0" json:"droppedTotal"`
	RetryTotal       uint64     `gorm:"type:bigint unsigned;not null;default:0" json:"retryTotal"`
	ReloadGeneration uint64     `gorm:"type:bigint unsigned;not null;default:0" json:"reloadGeneration"`
	LastError        string     `gorm:"type:text" json:"lastError"`
	Metrics          string     `gorm:"type:text" json:"metrics"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// ClusterCollectorCredential stores the hashed installation credential for one Kubernetes cluster.
type ClusterCollectorCredential struct {
	ID        uint       `gorm:"primarykey" json:"id"`
	ClusterID uint       `gorm:"not null;uniqueIndex" json:"clusterId"`
	TokenHash string     `gorm:"type:varchar(64);not null" json:"-"`
	TokenHint string     `gorm:"type:varchar(16);not null" json:"tokenHint"`
	Status    string     `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	RotatedAt *time.Time `gorm:"type:datetime" json:"rotatedAt"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

func (ClusterCollectorCredential) TableName() string {
	return "log_cluster_collector_credentials"
}

// CollectionPolicy stores the editable desired state of a log collection policy.
type CollectionPolicy struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	Name              string    `gorm:"type:varchar(120);not null;uniqueIndex" json:"name"`
	SourceMode        string    `gorm:"type:varchar(20);not null;default:'host';index" json:"sourceMode"`
	Description       string    `gorm:"type:varchar(500)" json:"description"`
	Paths             string    `gorm:"type:text;not null" json:"paths"`
	SourceOptions     string    `gorm:"type:text" json:"sourceOptions"`
	ParserType        string    `gorm:"type:varchar(32);not null;default:'raw'" json:"parserType"`
	ParserConfig      string    `gorm:"type:text" json:"parserConfig"`
	MultilineConfig   string    `gorm:"type:text" json:"multilineConfig"`
	FilterConfig      string    `gorm:"type:text" json:"filterConfig"`
	MaskConfig        string    `gorm:"type:text" json:"maskConfig"`
	MetadataConfig    string    `gorm:"type:text" json:"metadataConfig"`
	RetentionPolicyID uint      `gorm:"index;default:0" json:"retentionPolicyId"`
	RetentionDays     int       `gorm:"type:int;not null;default:30" json:"retentionDays"`
	RetentionConfig   string    `gorm:"type:text" json:"retentionConfig"`
	WALConfig         string    `gorm:"type:text" json:"walConfig"`
	Status            string    `gorm:"type:varchar(20);not null;default:'draft';index" json:"status"`
	Version           uint64    `gorm:"type:bigint unsigned;not null;default:0" json:"version"`
	CreatedBy         uint      `gorm:"index;default:0" json:"createdBy"`
	UpdatedBy         uint      `gorm:"index;default:0" json:"updatedBy"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (CollectionPolicy) TableName() string {
	return "log_collection_policies"
}

type PolicyTarget struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	PolicyID         uint      `gorm:"not null;index;uniqueIndex:idx_log_policy_target,priority:1" json:"policyId"`
	TargetType       string    `gorm:"type:varchar(32);not null;uniqueIndex:idx_log_policy_target,priority:2" json:"targetType"`
	TargetID         uint      `gorm:"not null;index;uniqueIndex:idx_log_policy_target,priority:3" json:"targetId"`
	Namespace        string    `gorm:"type:varchar(128);uniqueIndex:idx_log_policy_target,priority:4" json:"namespace"`
	WorkloadKind     string    `gorm:"type:varchar(64);uniqueIndex:idx_log_policy_target,priority:5" json:"workloadKind"`
	WorkloadName     string    `gorm:"type:varchar(255);uniqueIndex:idx_log_policy_target,priority:6" json:"workloadName"`
	LabelSelector    string    `gorm:"type:varchar(1000)" json:"labelSelector"`
	ContainerInclude string    `gorm:"type:text" json:"containerInclude"`
	ContainerExclude string    `gorm:"type:text" json:"containerExclude"`
	CreatedAt        time.Time `json:"createdAt"`
}

func (PolicyTarget) TableName() string {
	return "log_policy_targets"
}

type PolicyRevision struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	PolicyID      uint      `gorm:"not null;index;uniqueIndex:idx_log_policy_revision,priority:1" json:"policyId"`
	Version       uint64    `gorm:"type:bigint unsigned;not null;uniqueIndex:idx_log_policy_revision,priority:2" json:"version"`
	Content       string    `gorm:"type:longtext;not null" json:"content"`
	Checksum      string    `gorm:"type:varchar(64);not null" json:"checksum"`
	ChangeSummary string    `gorm:"type:varchar(500)" json:"changeSummary"`
	CreatedBy     uint      `gorm:"index;default:0" json:"createdBy"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (PolicyRevision) TableName() string {
	return "log_policy_revisions"
}

type CollectorAssignment struct {
	ID            uint       `gorm:"primarykey" json:"id"`
	InstanceID    string     `gorm:"type:varchar(160);not null;index;uniqueIndex:idx_log_assignment,priority:1" json:"instanceId"`
	PolicyID      uint       `gorm:"not null;index;uniqueIndex:idx_log_assignment,priority:2" json:"policyId"`
	PolicyVersion uint64     `gorm:"type:bigint unsigned;not null" json:"policyVersion"`
	DesiredState  string     `gorm:"type:varchar(20);not null;default:'active';index" json:"desiredState"`
	ApplyStatus   string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"applyStatus"`
	AppliedAt     *time.Time `gorm:"type:datetime" json:"appliedAt"`
	LastError     string     `gorm:"type:text" json:"lastError"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (CollectorAssignment) TableName() string {
	return "log_collector_assignments"
}

func (CollectorInstance) TableName() string {
	return "log_collector_instances"
}

// AccessPolicy stores log field and action access-control rules.
type AccessPolicy struct {
	ID                 uint                `gorm:"primarykey" json:"id"`
	Name               string              `gorm:"type:varchar(120);not null;default:''" json:"name"`
	Description        string              `gorm:"type:varchar(500)" json:"description"`
	SubjectType        string              `gorm:"type:varchar(20);not null;index" json:"subjectType"`
	SubjectID          uint                `gorm:"not null;index" json:"subjectId"`
	DataSourceID       uint                `gorm:"not null;index" json:"datasourceId"`
	LibraryItemPattern string              `gorm:"type:varchar(255)" json:"libraryItemPattern"`
	ScopeMode          string              `gorm:"type:varchar(32);not null;default:'all';index" json:"scopeMode"`
	AllowedActions     string              `gorm:"type:varchar(255)" json:"allowedActions"`
	DeniedFields       string              `gorm:"type:text" json:"deniedFields"`
	MaskFields         string              `gorm:"type:text" json:"maskFields"`
	Enabled            bool                `gorm:"type:tinyint(1);not null;default:1;index" json:"enabled"`
	CreatedBy          uint                `gorm:"index;default:0" json:"createdBy"`
	UpdatedBy          uint                `gorm:"index;default:0" json:"updatedBy"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
	Scopes             []AccessPolicyScope `gorm:"foreignKey:AccessPolicyID;constraint:OnDelete:CASCADE" json:"-"`
}

// AccessPolicyScope binds an access policy to a collection policy.
type AccessPolicyScope struct {
	ID                 uint      `gorm:"primarykey" json:"id"`
	AccessPolicyID     uint      `gorm:"not null;index;uniqueIndex:idx_log_access_policy_scope,priority:1" json:"accessPolicyId"`
	CollectionPolicyID uint      `gorm:"not null;index;uniqueIndex:idx_log_access_policy_scope,priority:2" json:"collectionPolicyId"`
	CreatedAt          time.Time `json:"createdAt"`
}

func (AccessPolicyScope) TableName() string {
	return "log_access_policy_scopes"
}

// RetentionPolicy stores reusable differential retention profiles.
type RetentionPolicy struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Name        string    `gorm:"type:varchar(120);not null;uniqueIndex" json:"name"`
	Description string    `gorm:"type:varchar(500)" json:"description"`
	StorageID   uint      `gorm:"index;default:0" json:"storageId"`
	DefaultDays int       `gorm:"type:int;not null;default:30" json:"defaultDays"`
	LevelDays   string    `gorm:"type:text" json:"levelDays"`
	Priority    int       `gorm:"type:int;not null;default:100" json:"priority"`
	Enabled     bool      `gorm:"type:tinyint(1);not null;default:1;index" json:"enabled"`
	CreatedBy   uint      `gorm:"index;default:0" json:"createdBy"`
	UpdatedBy   uint      `gorm:"index;default:0" json:"updatedBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (RetentionPolicy) TableName() string {
	return "log_retention_policies"
}

func (AccessPolicy) TableName() string {
	return "log_access_policies"
}
