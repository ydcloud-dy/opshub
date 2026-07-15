package logagent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type CRIRecord struct {
	Timestamp  time.Time
	Stream     string
	Body       string
	Attributes map[string]string
}

type CRIAssembler struct {
	partial   strings.Builder
	timestamp time.Time
	stream    string
	maxBytes  int
	truncated bool
	active    bool
}

func NewCRIAssembler(maxBytes int) *CRIAssembler {
	if maxBytes <= 0 || maxBytes > maxLineBytes {
		maxBytes = maxLineBytes
	}
	return &CRIAssembler{maxBytes: maxBytes}
}

func (assembler *CRIAssembler) Add(line string, observedAt time.Time) []CRIRecord {
	record, flag, ok := parseContainerRuntimeLine(line, observedAt)
	if !ok {
		result := assembler.Flush()
		record.Attributes = map[string]string{"log.cri_parse_error": "invalid container runtime log format"}
		return append(result, record)
	}
	if flag == "P" {
		if !assembler.active {
			assembler.timestamp = record.Timestamp
			assembler.stream = record.Stream
			assembler.active = true
		}
		assembler.append(record.Body)
		return nil
	}
	if !assembler.active {
		return []CRIRecord{record}
	}
	if assembler.stream != "" && record.Stream != assembler.stream {
		result := assembler.Flush()
		return append(result, record)
	}
	assembler.append(record.Body)
	return assembler.Flush()
}

func (assembler *CRIAssembler) Flush() []CRIRecord {
	if !assembler.active {
		return nil
	}
	attributes := map[string]string{"log.cri_partial_merged": "true"}
	if assembler.truncated {
		attributes["log.truncated"] = "true"
	}
	record := CRIRecord{Timestamp: assembler.timestamp, Stream: assembler.stream, Body: assembler.partial.String(), Attributes: attributes}
	assembler.Reset()
	return []CRIRecord{record}
}

func (assembler *CRIAssembler) Reset() {
	assembler.partial.Reset()
	assembler.timestamp = time.Time{}
	assembler.stream = ""
	assembler.truncated = false
	assembler.active = false
}

func (assembler *CRIAssembler) MarkTruncated() {
	assembler.truncated = true
}

func (assembler *CRIAssembler) append(value string) {
	remaining := assembler.maxBytes - assembler.partial.Len()
	if remaining <= 0 {
		assembler.truncated = true
		return
	}
	if len(value) > remaining {
		assembler.partial.WriteString(value[:remaining])
		assembler.truncated = true
		return
	}
	assembler.partial.WriteString(value)
}

func parseContainerRuntimeLine(line string, observedAt time.Time) (CRIRecord, string, bool) {
	if strings.HasPrefix(strings.TrimSpace(line), "{") {
		var docker struct {
			Log    string `json:"log"`
			Stream string `json:"stream"`
			Time   string `json:"time"`
		}
		if json.Unmarshal([]byte(line), &docker) == nil && docker.Log != "" {
			timestamp := observedAt
			if parsed, err := time.Parse(time.RFC3339Nano, docker.Time); err == nil {
				timestamp = parsed
			}
			return CRIRecord{Timestamp: timestamp, Stream: docker.Stream, Body: strings.TrimSuffix(docker.Log, "\n")}, "F", true
		}
	}
	parts := strings.SplitN(line, " ", 4)
	if len(parts) != 4 || (parts[2] != "P" && parts[2] != "F") {
		return CRIRecord{Timestamp: observedAt, Body: line}, "", false
	}
	timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return CRIRecord{Timestamp: observedAt, Body: line}, "", false
	}
	if parts[1] != "stdout" && parts[1] != "stderr" {
		return CRIRecord{Timestamp: observedAt, Body: line}, "", false
	}
	return CRIRecord{Timestamp: timestamp, Stream: parts[1], Body: parts[3]}, parts[2], true
}

func (record CRIRecord) validate() error {
	if record.Timestamp.IsZero() {
		return fmt.Errorf("CRI 日志时间为空")
	}
	return nil
}
