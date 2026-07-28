package logagent

import "time"

type Event struct {
	Sequence           uint64            `json:"sequence"`
	SourceID           string            `json:"sourceId"`
	PolicyID           uint64            `json:"policyId"`
	PolicyVersion      uint64            `json:"policyVersion"`
	FilePath           string            `json:"filePath"`
	Environment        string            `json:"environment"`
	Service            string            `json:"service"`
	Stream             string            `json:"stream"`
	Namespace          string            `json:"namespace"`
	WorkloadKind       string            `json:"workloadKind"`
	WorkloadName       string            `json:"workloadName"`
	PodName            string            `json:"podName"`
	PodUID             string            `json:"podUid"`
	ContainerName      string            `json:"containerName"`
	ContainerImage     string            `json:"containerImage"`
	NodeName           string            `json:"nodeName"`
	Timestamp          time.Time         `json:"timestamp"`
	ObservedAt         time.Time         `json:"observedAt"`
	Body               string            `json:"body"`
	Level              string            `json:"level"`
	TraceID            string            `json:"traceId,omitempty"`
	SpanID             string            `json:"spanId,omitempty"`
	RetentionDays      int               `json:"retentionDays"`
	Attributes         map[string]string `json:"attributes"`
	ResourceAttributes map[string]string `json:"resourceAttributes"`
}

type KubernetesMetadata struct {
	ClusterID          uint64
	ClusterName        string
	Namespace          string
	WorkloadKind       string
	WorkloadName       string
	PodName            string
	PodUID             string
	ContainerName      string
	ContainerImage     string
	NodeName           string
	Environment        string
	Service            string
	Labels             map[string]string
	Annotations        map[string]string
	ResourceAttributes map[string]string
}

type KubernetesMetadataResolver interface {
	Resolve(path string, config KubernetesSourceConfig) (KubernetesMetadata, bool, error)
}

type AgentIdentity struct {
	AgentID   string
	AssetType string
	AssetID   uint64
	HostID    uint64
	ClusterID uint64
	NodeName  string
}

type EventSink interface {
	Append(Event) error
}
