package model

import "time"

// DataSource stores external observability backends such as Prometheus, Loki, VictoriaMetrics, and Elasticsearch.
type DataSource struct {
	ID                       uint       `gorm:"primarykey" json:"id"`
	Name                     string     `gorm:"type:varchar(100);not null" json:"name"`
	Type                     string     `gorm:"type:varchar(32);not null;index" json:"type"`
	URL                      string     `gorm:"type:varchar(500);not null" json:"url"`
	AuthType                 string     `gorm:"type:varchar(32);not null;default:'none'" json:"authType"`
	Username                 string     `gorm:"type:varchar(255)" json:"username"`
	Password                 string     `gorm:"type:varchar(500)" json:"password"`
	Token                    string     `gorm:"type:text" json:"token"`
	Headers                  string     `gorm:"type:text" json:"headers"`
	Timeout                  int        `gorm:"type:int;not null;default:10" json:"timeout"`
	SkipTLSVerify            bool       `gorm:"type:tinyint(1);not null;default:0" json:"skipTlsVerify"`
	Enabled                  bool       `gorm:"type:tinyint(1);not null;default:1;index" json:"enabled"`
	RemoteWriteEnabled       bool       `gorm:"type:tinyint(1);not null;default:0;index" json:"remoteWriteEnabled"`
	RemoteWriteURL           string     `gorm:"type:varchar(500)" json:"remoteWriteUrl"`
	RemoteWriteAuthType      string     `gorm:"type:varchar(32);not null;default:'none'" json:"remoteWriteAuthType"`
	RemoteWriteUsername      string     `gorm:"type:varchar(255)" json:"remoteWriteUsername"`
	RemoteWritePassword      string     `gorm:"type:varchar(500)" json:"remoteWritePassword"`
	RemoteWriteToken         string     `gorm:"type:text" json:"remoteWriteToken"`
	RemoteWriteHeaders       string     `gorm:"type:text" json:"remoteWriteHeaders"`
	RemoteWriteSkipTLSVerify bool       `gorm:"type:tinyint(1);not null;default:0" json:"remoteWriteSkipTlsVerify"`
	Status                   string     `gorm:"type:varchar(20);not null;default:'unknown'" json:"status"`
	LastTestAt               *time.Time `gorm:"type:datetime" json:"lastTestAt"`
	LastError                string     `gorm:"type:text" json:"lastError"`
	Description              string     `gorm:"type:varchar(500)" json:"description"`
	CreatedAt                time.Time  `json:"createdAt"`
	UpdatedAt                time.Time  `json:"updatedAt"`
}

func (DataSource) TableName() string {
	return "monitor_datasources"
}

// ProbeTask stores WatchAlert-style blackbox probing jobs.
type ProbeTask struct {
	ID                 uint       `gorm:"primarykey" json:"id"`
	Name               string     `gorm:"type:varchar(120);not null" json:"name"`
	Protocol           string     `gorm:"type:varchar(20);not null;index" json:"protocol"`
	Endpoint           string     `gorm:"type:text;not null" json:"endpoint"`
	Method             string     `gorm:"type:varchar(16);not null;default:'GET'" json:"method"`
	Headers            string     `gorm:"type:text" json:"headers"`
	Body               string     `gorm:"type:text" json:"body"`
	FrequencySeconds   int        `gorm:"type:int;not null;default:60" json:"frequencySeconds"`
	TimeoutSeconds     int        `gorm:"type:int;not null;default:5" json:"timeoutSeconds"`
	ICMPCount          int        `gorm:"type:int;not null;default:3" json:"icmpCount"`
	ICMPIntervalMS     int        `gorm:"type:int;not null;default:1000" json:"icmpIntervalMs"`
	DataSourceID       uint       `gorm:"index;default:0" json:"dataSourceId"`
	WriteRuleEnabled   bool       `gorm:"type:tinyint(1);not null;default:1" json:"writeRuleEnabled"`
	Enabled            bool       `gorm:"type:tinyint(1);not null;default:1;index" json:"enabled"`
	Status             string     `gorm:"type:varchar(20);not null;default:'unknown';index" json:"status"`
	LastStatus         string     `gorm:"type:varchar(20)" json:"lastStatus"`
	LastProbeAt        *time.Time `gorm:"type:datetime" json:"lastProbeAt"`
	NextProbeAt        *time.Time `gorm:"type:datetime;index" json:"nextProbeAt"`
	LastDurationMS     int64      `gorm:"type:bigint;not null;default:0" json:"lastDurationMs"`
	LastError          string     `gorm:"type:text" json:"lastError"`
	LastWriteAt        *time.Time `gorm:"type:datetime" json:"lastWriteAt"`
	LastWriteError     string     `gorm:"type:text" json:"lastWriteError"`
	Description        string     `gorm:"type:varchar(500)" json:"description"`
	Operator           string     `gorm:"type:varchar(80)" json:"operator"`
	DataSourceName     string     `gorm:"-" json:"dataSourceName"`
	DataSourceRemoteOK bool       `gorm:"-" json:"dataSourceRemoteOk"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

func (ProbeTask) TableName() string {
	return "monitor_probe_tasks"
}

// ProbeHistory records the latest execution results of probe tasks.
type ProbeHistory struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	ProbeTaskID    uint       `gorm:"not null;index" json:"probeTaskId"`
	Protocol       string     `gorm:"type:varchar(20);not null;index" json:"protocol"`
	Endpoint       string     `gorm:"type:varchar(500);not null" json:"endpoint"`
	Success        bool       `gorm:"type:tinyint(1);not null;default:0;index" json:"success"`
	StatusCode     int        `gorm:"type:int;not null;default:0" json:"statusCode"`
	DurationMS     int64      `gorm:"type:bigint;not null;default:0" json:"durationMs"`
	SSLExpireAt    *time.Time `gorm:"type:datetime" json:"sslExpireAt"`
	SSLDaysLeft    int        `gorm:"type:int;not null;default:0" json:"sslDaysLeft"`
	Error          string     `gorm:"type:text" json:"error"`
	Message        string     `gorm:"type:text" json:"message"`
	RemoteWriteOK  bool       `gorm:"type:tinyint(1);not null;default:0" json:"remoteWriteOk"`
	RemoteWriteErr string     `gorm:"type:text" json:"remoteWriteErr"`
	CheckedAt      time.Time  `gorm:"type:datetime;not null;index" json:"checkedAt"`
	CreatedAt      time.Time  `json:"createdAt"`
}

func (ProbeHistory) TableName() string {
	return "monitor_probe_histories"
}

// AlertRule is the persisted rule skeleton used by the upcoming evaluator.
type AlertRule struct {
	ID               uint       `gorm:"primarykey" json:"id"`
	Name             string     `gorm:"type:varchar(120);not null" json:"name"`
	RuleGroupID      uint       `gorm:"index;default:0" json:"ruleGroupId"`
	FaultCenterID    uint       `gorm:"index;default:0" json:"faultCenterId"`
	DataSourceID     uint       `gorm:"not null;index" json:"dataSourceId"`
	DataSourceIDs    string     `gorm:"type:text" json:"dataSourceIds"`
	DataSourceType   string     `gorm:"type:varchar(32);not null" json:"dataSourceType"`
	Query            string     `gorm:"type:text;not null" json:"query"`
	QueryMode        string     `gorm:"type:varchar(20);not null;default:'instant'" json:"queryMode"`
	Index            string     `gorm:"type:varchar(255)" json:"index"`
	Condition        string     `gorm:"type:varchar(32);not null;default:'gt'" json:"condition"`
	Threshold        float64    `gorm:"type:double;not null;default:0" json:"threshold"`
	SeverityRules    string     `gorm:"type:text" json:"severityRules"`
	ForSeconds       int        `gorm:"type:int;not null;default:60" json:"forSeconds"`
	EvaluateInterval int        `gorm:"type:int;not null;default:60" json:"evaluateInterval"`
	Severity         string     `gorm:"type:varchar(20);not null;default:'warning'" json:"severity"`
	Enabled          bool       `gorm:"type:tinyint(1);not null;default:1;index" json:"enabled"`
	ChannelIDs       string     `gorm:"type:text" json:"channelIds"`
	NotifyRecovery   bool       `gorm:"type:tinyint(1);not null;default:1" json:"notifyRecovery"`
	RepeatInterval   int        `gorm:"type:int;not null;default:3600" json:"repeatInterval"`
	Labels           string     `gorm:"type:text" json:"labels"`
	Annotations      string     `gorm:"type:text" json:"annotations"`
	DetailTemplate   string     `gorm:"type:text" json:"detailTemplate"`
	CallbackQueries  string     `gorm:"type:text" json:"callbackQueries"`
	EffectiveTime    string     `gorm:"type:text" json:"effectiveTime"`
	LastState        string     `gorm:"type:varchar(20);not null;default:'inactive'" json:"lastState"`
	LastValue        float64    `gorm:"type:double;not null;default:0" json:"lastValue"`
	LastEvalAt       *time.Time `gorm:"type:datetime" json:"lastEvalAt"`
	PendingSince     *time.Time `gorm:"type:datetime" json:"pendingSince"`
	FiringSince      *time.Time `gorm:"type:datetime" json:"firingSince"`
	LastNotifyAt     *time.Time `gorm:"type:datetime" json:"lastNotifyAt"`
	LastError        string     `gorm:"type:text" json:"lastError"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (AlertRule) TableName() string {
	return "monitor_alert_rules"
}

// AlertEvent records generic alert lifecycle events produced by data-source rules.
type AlertEvent struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	RuleID         uint       `gorm:"not null;index" json:"ruleId"`
	RuleGroupID    uint       `gorm:"index;default:0" json:"ruleGroupId"`
	FaultCenterID  uint       `gorm:"index;default:0" json:"faultCenterId"`
	RuleName       string     `gorm:"type:varchar(120);not null" json:"ruleName"`
	DataSourceID   uint       `gorm:"not null;index" json:"dataSourceId"`
	DataSourceName string     `gorm:"type:varchar(100);not null" json:"dataSourceName"`
	DataSourceType string     `gorm:"type:varchar(32);not null;index" json:"dataSourceType"`
	Severity       string     `gorm:"type:varchar(20);not null;index" json:"severity"`
	State          string     `gorm:"type:varchar(20);not null;index" json:"state"`
	Value          float64    `gorm:"type:double;not null;default:0" json:"value"`
	Condition      string     `gorm:"type:varchar(32);not null" json:"condition"`
	Threshold      float64    `gorm:"type:double;not null;default:0" json:"threshold"`
	Message        string     `gorm:"type:text" json:"message"`
	Labels         string     `gorm:"type:text" json:"labels"`
	Annotations    string     `gorm:"type:text" json:"annotations"`
	Fingerprint    string     `gorm:"type:varchar(80);not null;index" json:"fingerprint"`
	Acknowledged   bool       `gorm:"type:tinyint(1);not null;default:0;index" json:"acknowledged"`
	AcknowledgedBy string     `gorm:"type:varchar(80)" json:"acknowledgedBy"`
	AcknowledgedAt *time.Time `gorm:"type:datetime" json:"acknowledgedAt"`
	NotifyStatus   string     `gorm:"type:varchar(20);not null;default:'none'" json:"notifyStatus"`
	NotifyError    string     `gorm:"type:text" json:"notifyError"`
	Escalated      bool       `gorm:"type:tinyint(1);not null;default:0;index" json:"escalated"`
	EscalatedAt    *time.Time `gorm:"type:datetime" json:"escalatedAt"`
	LastEscalateAt *time.Time `gorm:"type:datetime" json:"lastEscalateAt"`
	EscalateStatus string     `gorm:"type:varchar(20);not null;default:'none'" json:"escalateStatus"`
	EscalateError  string     `gorm:"type:text" json:"escalateError"`
	StartedAt      time.Time  `gorm:"type:datetime;not null;index" json:"startedAt"`
	EndedAt        *time.Time `gorm:"type:datetime" json:"endedAt"`
	LastEvalAt     time.Time  `gorm:"type:datetime;not null;index" json:"lastEvalAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (AlertEvent) TableName() string {
	return "monitor_alert_events"
}

// FaultCenter groups alert events and owns notification/escalation policy.
type FaultCenter struct {
	ID                    uint      `gorm:"primarykey" json:"id"`
	Name                  string    `gorm:"type:varchar(120);not null;uniqueIndex" json:"name"`
	Description           string    `gorm:"type:varchar(500)" json:"description"`
	NoticeObjectIDs       string    `gorm:"type:text" json:"noticeObjectIds"`
	NoticeChannelIDs      string    `gorm:"type:text" json:"noticeChannelIds"`
	NoticeRoutes          string    `gorm:"type:text" json:"noticeRoutes"`
	RepeatNoticeInterval  string    `gorm:"type:text" json:"repeatNoticeInterval"`
	RecoverNotify         bool      `gorm:"type:tinyint(1);not null;default:1" json:"recoverNotify"`
	AggregationType       string    `gorm:"type:varchar(32);not null;default:'Rule'" json:"aggregationType"`
	SilenceEnabled        bool      `gorm:"type:tinyint(1);not null;default:0" json:"silenceEnabled"`
	SilenceRules          string    `gorm:"type:text" json:"silenceRules"`
	RecoverWaitSeconds    int       `gorm:"type:int;not null;default:30" json:"recoverWaitSeconds"`
	UpgradeEnabled        bool      `gorm:"type:tinyint(1);not null;default:0" json:"upgradeEnabled"`
	UpgradableSeverities  string    `gorm:"type:text" json:"upgradableSeverities"`
	UpgradeStrategy       string    `gorm:"type:text" json:"upgradeStrategy"`
	CurrentPreAlertNumber int64     `gorm:"-" json:"currentPreAlertNumber"`
	CurrentAlertNumber    int64     `gorm:"-" json:"currentAlertNumber"`
	CurrentRecoverNumber  int64     `gorm:"-" json:"currentRecoverNumber"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

func (FaultCenter) TableName() string {
	return "monitor_fault_centers"
}

// AlertRuleGroup mirrors WatchAlert's rule grouping sidebar.
type AlertRuleGroup struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Name        string    `gorm:"type:varchar(120);not null;uniqueIndex" json:"name"`
	Description string    `gorm:"type:varchar(500)" json:"description"`
	Sort        int       `gorm:"type:int;not null;default:0" json:"sort"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (AlertRuleGroup) TableName() string {
	return "monitor_rule_groups"
}

// NoticeTemplate stores WatchAlert-style notification templates for each route type.
type NoticeTemplate struct {
	ID                   uint      `gorm:"primarykey" json:"id"`
	Name                 string    `gorm:"type:varchar(120);not null;uniqueIndex" json:"name"`
	NoticeType           string    `gorm:"type:varchar(32);not null;index" json:"noticeType"`
	Description          string    `gorm:"type:varchar(500)" json:"description"`
	Template             string    `gorm:"type:text" json:"template"`
	TemplateFiring       string    `gorm:"type:text" json:"templateFiring"`
	TemplateRecover      string    `gorm:"type:text" json:"templateRecover"`
	EnableFeiShuJSONCard bool      `gorm:"type:tinyint(1);not null;default:0" json:"enableFeiShuJsonCard"`
	Enabled              bool      `gorm:"type:tinyint(1);not null;default:1;index" json:"enabled"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

func (NoticeTemplate) TableName() string {
	return "monitor_notice_templates"
}

// NoticeObject defines notification routes. Each route may reference a template and a duty table.
type NoticeObject struct {
	ID               uint       `gorm:"primarykey" json:"id"`
	UUID             string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"uuid"`
	Name             string     `gorm:"type:varchar(120);not null;uniqueIndex" json:"name"`
	Description      string     `gorm:"type:varchar(500)" json:"description"`
	DutyTableID      uint       `gorm:"index;default:0" json:"dutyTableId"`
	Routes           string     `gorm:"type:text" json:"routes"`
	Enabled          bool       `gorm:"type:tinyint(1);not null;default:1;index" json:"enabled"`
	LastStatus       string     `gorm:"type:varchar(32);not null;default:'unused'" json:"lastStatus"`
	DutyTableName    string     `gorm:"-" json:"dutyTableName"`
	CurrentDutyUsers []DutyUser `gorm:"-" json:"currentDutyUsers"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (NoticeObject) TableName() string {
	return "monitor_notice_objects"
}

// DutyUser is embedded in duty schedules so the monitor module can keep a stable roster snapshot.
type DutyUser struct {
	ID             uint   `gorm:"column:id" json:"id"`
	Username       string `gorm:"column:username" json:"username"`
	RealName       string `gorm:"column:real_name" json:"realName"`
	Email          string `gorm:"column:email" json:"email"`
	Phone          string `gorm:"column:phone" json:"phone"`
	NotifyUserID   string `gorm:"column:notify_user_id" json:"notifyUserId"`
	FeishuUserID   string `gorm:"column:feishu_user_id" json:"feishuUserId"`
	FeishuOpenID   string `gorm:"column:feishu_open_id" json:"feishuOpenId"`
	DingTalkUserID string `gorm:"column:dingtalk_user_id" json:"dingtalkUserId"`
	WeComUserID    string `gorm:"column:wecom_user_id" json:"wecomUserId"`
}

// DutyTable groups schedules and exposes the current on-call users.
type DutyTable struct {
	ID               uint       `gorm:"primarykey" json:"id"`
	Name             string     `gorm:"type:varchar(120);not null;uniqueIndex" json:"name"`
	Description      string     `gorm:"type:varchar(500)" json:"description"`
	ManagerUserID    uint       `gorm:"index;default:0" json:"managerUserId"`
	ManagerUsername  string     `gorm:"type:varchar(80)" json:"managerUsername"`
	Enabled          bool       `gorm:"type:tinyint(1);not null;default:1;index" json:"enabled"`
	CurrentDutyUsers []DutyUser `gorm:"-" json:"currentDutyUsers"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (DutyTable) TableName() string {
	return "monitor_duty_tables"
}

// DutySchedule stores day-level on-call users. Users is a JSON array of DutyUser.
type DutySchedule struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	DutyTableID uint      `gorm:"not null;index:idx_monitor_duty_day,unique" json:"dutyTableId"`
	DutyDate    string    `gorm:"type:varchar(10);not null;index:idx_monitor_duty_day,unique" json:"dutyDate"`
	Users       string    `gorm:"type:text" json:"users"`
	Status      string    `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (DutySchedule) TableName() string {
	return "monitor_duty_schedules"
}
