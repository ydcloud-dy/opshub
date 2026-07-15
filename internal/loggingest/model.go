package loggingest

import "time"

type LogBatch struct {
	BatchID        string      `json:"batchId"`
	AgentID        string      `json:"agentId"`
	PolicyID       uint64      `json:"policyId"`
	PolicyVersion  uint64      `json:"policyVersion"`
	SequenceStart  uint64      `json:"sequenceStart"`
	SequenceEnd    uint64      `json:"sequenceEnd"`
	SourceType     string      `json:"sourceType"`
	AssetType      string      `json:"assetType"`
	AssetID        uint64      `json:"assetId"`
	HostID         uint64      `json:"hostId"`
	ClusterID      uint64      `json:"clusterId"`
	Environment    string      `json:"environment"`
	Service        string      `json:"service"`
	Namespace      string      `json:"namespace"`
	WorkloadKind   string      `json:"workloadKind"`
	WorkloadName   string      `json:"workloadName"`
	PodName        string      `json:"podName"`
	PodUID         string      `json:"podUid"`
	ContainerName  string      `json:"containerName"`
	ContainerImage string      `json:"containerImage"`
	NodeName       string      `json:"nodeName"`
	FilePath       string      `json:"filePath"`
	Stream         string      `json:"stream"`
	Records        []LogRecord `json:"records"`
}

type LogRecord struct {
	Sequence                  uint64            `json:"sequence"`
	TimestampUnixNano         int64             `json:"timestampUnixNano"`
	ObservedTimestampUnixNano int64             `json:"observedTimestampUnixNano"`
	Body                      string            `json:"body"`
	SeverityText              string            `json:"severityText"`
	SeverityNumber            int32             `json:"severityNumber"`
	RetentionDays             int               `json:"retentionDays"`
	Attributes                map[string]string `json:"attributes"`
	ResourceAttributes        map[string]string `json:"resourceAttributes"`
	TraceID                   string            `json:"traceId"`
	SpanID                    string            `json:"spanId"`
}

type IngestAck struct {
	BatchID          string `json:"batchId"`
	AcceptedRecords  int    `json:"acceptedRecords"`
	AcceptedSequence uint64 `json:"acceptedSequence"`
	RetryAfterMS     int    `json:"retryAfterMs"`
	Duplicate        bool   `json:"duplicate"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

type ComponentStatus struct {
	Name              string     `json:"name"`
	InstanceID        string     `json:"instanceId,omitempty"`
	Status            string     `json:"status"`
	QueueMode         string     `json:"queueMode,omitempty"`
	QueueTopic        string     `json:"queueTopic,omitempty"`
	ConsumerGroup     string     `json:"consumerGroup,omitempty"`
	BrokerCount       int        `json:"brokerCount,omitempty"`
	QueueHealthy      bool       `json:"queueHealthy"`
	QueueLag          int64      `json:"queueLag,omitempty"`
	StartedAt         time.Time  `json:"startedAt"`
	UptimeSeconds     int64      `json:"uptimeSeconds"`
	AcceptedBatches   uint64     `json:"acceptedBatches"`
	AcceptedRecords   uint64     `json:"acceptedRecords"`
	RejectedBatches   uint64     `json:"rejectedBatches"`
	DuplicateBatches  uint64     `json:"duplicateBatches"`
	FailedBatches     uint64     `json:"failedBatches"`
	QueueDepth        int        `json:"queueDepth"`
	QueueCapacity     int        `json:"queueCapacity"`
	Inflight          int64      `json:"inflight,omitempty"`
	InflightLimit     int        `json:"inflightLimit,omitempty"`
	PublishLatencyMS  float64    `json:"publishLatencyMs,omitempty"`
	WriteLatencyMS    float64    `json:"writeLatencyMs,omitempty"`
	DeadletterBatches uint64     `json:"deadletterBatches,omitempty"`
	LastSuccessAt     *time.Time `json:"lastSuccessAt,omitempty"`
	LastErrorAt       *time.Time `json:"lastErrorAt,omitempty"`
	LastError         string     `json:"lastError,omitempty"`
	QueueLastError    string     `json:"queueLastError,omitempty"`
	WriterURL         string     `json:"writerUrl,omitempty"`
	HTTPAddress       string     `json:"httpAddress,omitempty"`
	GRPCAddress       string     `json:"grpcAddress,omitempty"`
}
