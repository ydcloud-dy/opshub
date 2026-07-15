package logagent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ParsedRecord struct {
	Timestamp  time.Time
	Body       string
	Level      string
	Attributes map[string]string
}

type LineParser interface {
	Parse(line string, observedAt time.Time) (ParsedRecord, error)
}

func NewLineParser(config ParserConfig) (LineParser, error) {
	switch strings.ToLower(strings.TrimSpace(config.Type)) {
	case "", "raw":
		return rawParser{}, nil
	case "json":
		return jsonParser{config: config}, nil
	case "regex":
		pattern, err := regexp.Compile(config.Pattern)
		if err != nil {
			return nil, err
		}
		return regexParser{config: config, pattern: pattern}, nil
	default:
		return nil, fmt.Errorf("不支持的日志解析器: %s", config.Type)
	}
}

type rawParser struct{}

func (rawParser) Parse(line string, observedAt time.Time) (ParsedRecord, error) {
	return ParsedRecord{Timestamp: observedAt, Body: line, Level: detectLevel(line), Attributes: map[string]string{}}, nil
}

type jsonParser struct {
	config ParserConfig
}

func (parser jsonParser) Parse(line string, observedAt time.Time) (ParsedRecord, error) {
	var value map[string]interface{}
	if err := json.Unmarshal([]byte(line), &value); err != nil {
		return ParsedRecord{}, fmt.Errorf("JSON 日志解析失败: %w", err)
	}
	messageField := firstNonEmptyString(parser.config.MessageField, "message", "msg", "body")
	timestampField := firstNonEmptyString(parser.config.TimestampField, "timestamp", "time", "@timestamp")
	levelField := firstNonEmptyString(parser.config.LevelField, "level", "severity")
	body := firstMapString(value, messageField, "message", "msg", "body")
	if body == "" {
		body = line
	}
	timestamp := parseTimestamp(firstMapString(value, timestampField, "timestamp", "time", "@timestamp"), parser.config.TimestampLayout, observedAt)
	level := strings.ToUpper(firstMapString(value, levelField, "level", "severity"))
	if level == "" {
		level = detectLevel(body)
	}
	attributes := make(map[string]string)
	for key, item := range value {
		if key == messageField || key == timestampField || key == levelField {
			continue
		}
		switch typed := item.(type) {
		case string:
			attributes[key] = typed
		case float64, bool, json.Number:
			attributes[key] = fmt.Sprint(typed)
		}
	}
	return ParsedRecord{Timestamp: timestamp, Body: body, Level: level, Attributes: attributes}, nil
}

type regexParser struct {
	config  ParserConfig
	pattern *regexp.Regexp
}

func (parser regexParser) Parse(line string, observedAt time.Time) (ParsedRecord, error) {
	match := parser.pattern.FindStringSubmatch(line)
	if match == nil {
		return ParsedRecord{}, fmt.Errorf("日志不匹配解析正则")
	}
	values := make(map[string]string)
	for index, name := range parser.pattern.SubexpNames() {
		if index > 0 && name != "" && index < len(match) {
			values[name] = match[index]
		}
	}
	messageField := firstNonEmptyString(parser.config.MessageField, "message")
	timestampField := firstNonEmptyString(parser.config.TimestampField, "timestamp")
	levelField := firstNonEmptyString(parser.config.LevelField, "level")
	body := firstNonEmptyString(values[messageField], line)
	level := strings.ToUpper(values[levelField])
	if level == "" {
		level = detectLevel(body)
	}
	attributes := make(map[string]string)
	for key, value := range values {
		if key != messageField && key != timestampField && key != levelField {
			attributes[key] = value
		}
	}
	return ParsedRecord{
		Timestamp:  parseTimestamp(values[timestampField], parser.config.TimestampLayout, observedAt),
		Body:       body,
		Level:      level,
		Attributes: attributes,
	}, nil
}

func parseTimestamp(value, layout string, fallback time.Time) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if layout != "" {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	for _, candidate := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05.000", "2006-01-02 15:04:05"} {
		if parsed, err := time.ParseInLocation(candidate, value, time.Local); err == nil {
			return parsed
		}
	}
	if unixValue, err := strconv.ParseInt(value, 10, 64); err == nil {
		if unixValue > 1_000_000_000_000_000 {
			return time.Unix(0, unixValue)
		}
		if unixValue > 1_000_000_000_000 {
			return time.UnixMilli(unixValue)
		}
		return time.Unix(unixValue, 0)
	}
	return fallback
}

func detectLevel(line string) string {
	upper := strings.ToUpper(line)
	for _, level := range []string{"FATAL", "ERROR", "WARN", "DEBUG", "TRACE", "INFO"} {
		if strings.Contains(upper, level) {
			return level
		}
	}
	return "INFO"
}

func firstMapString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, exists := values[key]; exists && value != nil {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
