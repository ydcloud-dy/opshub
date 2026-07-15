package logagent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const redactedValue = "[REDACTED]"

var defaultSensitiveFields = []string{
	"password", "passwd", "pwd", "token", "access_token", "refresh_token", "authorization",
	"cookie", "set-cookie", "secret", "client_secret", "api_key", "apikey", "access_key", "secret_key",
}

type RedactionConfig struct {
	Configured      bool            `json:"configured"`
	Enabled         bool            `json:"enabled"`
	UseDefaultRules bool            `json:"useDefaultRules"`
	SensitiveFields []string        `json:"sensitiveFields,omitempty"`
	Rules           []RedactionRule `json:"rules,omitempty"`
}

type RedactionRule struct {
	Name        string `json:"name"`
	Target      string `json:"target"`
	Field       string `json:"field,omitempty"`
	Pattern     string `json:"pattern,omitempty"`
	Action      string `json:"action"`
	Replacement string `json:"replacement,omitempty"`
}

type compiledRedactionRule struct {
	RedactionRule
	pattern *regexp.Regexp
}

type Redactor struct {
	enabled         bool
	sensitiveFields map[string]struct{}
	rules           []compiledRedactionRule
}

func DefaultRedactionConfig() RedactionConfig {
	return RedactionConfig{
		Configured: true, Enabled: true, UseDefaultRules: true,
		SensitiveFields: append([]string(nil), defaultSensitiveFields...),
	}
}

func (config *RedactionConfig) normalize() {
	if !config.Configured {
		*config = DefaultRedactionConfig()
		return
	}
	fields := normalizeStringList(config.SensitiveFields)
	if config.UseDefaultRules {
		fields = normalizeStringList(append(fields, defaultSensitiveFields...))
	}
	for index := range fields {
		fields[index] = normalizeSensitiveField(fields[index])
	}
	config.SensitiveFields = fields
	for index := range config.Rules {
		rule := &config.Rules[index]
		rule.Name = strings.TrimSpace(rule.Name)
		rule.Target = strings.ToLower(strings.TrimSpace(rule.Target))
		rule.Field = normalizeSensitiveField(rule.Field)
		rule.Pattern = strings.TrimSpace(rule.Pattern)
		rule.Action = strings.ToLower(strings.TrimSpace(rule.Action))
		rule.Replacement = strings.TrimSpace(rule.Replacement)
		if rule.Target == "" {
			rule.Target = "field"
		}
		if rule.Action == "" {
			rule.Action = "replace"
		}
	}
}

func (config RedactionConfig) Validate(sourceID string) error {
	if !config.Enabled {
		return nil
	}
	for index, rule := range config.Rules {
		name := firstNonEmptyString(rule.Name, fmt.Sprintf("规则 %d", index+1))
		switch rule.Target {
		case "field", "json_path":
			if rule.Field == "" {
				return fmt.Errorf("日志源 %s 的脱敏%s缺少字段路径", sourceID, name)
			}
		case "regex":
			if rule.Pattern == "" {
				return fmt.Errorf("日志源 %s 的脱敏%s缺少正则", sourceID, name)
			}
			if _, err := regexp.Compile(rule.Pattern); err != nil {
				return fmt.Errorf("日志源 %s 的脱敏%s正则无效: %w", sourceID, name, err)
			}
		default:
			return fmt.Errorf("日志源 %s 的脱敏%s目标无效: %s", sourceID, name, rule.Target)
		}
		switch rule.Action {
		case "replace", "hash", "drop_field":
		default:
			return fmt.Errorf("日志源 %s 的脱敏%s动作无效: %s", sourceID, name, rule.Action)
		}
	}
	return nil
}

func NewRedactor(config RedactionConfig) (*Redactor, error) {
	config.normalize()
	if err := config.Validate("redaction"); err != nil {
		return nil, err
	}
	redactor := &Redactor{enabled: config.Enabled, sensitiveFields: make(map[string]struct{})}
	for _, field := range config.SensitiveFields {
		redactor.sensitiveFields[normalizeSensitiveField(field)] = struct{}{}
	}
	for _, rule := range config.Rules {
		compiled := compiledRedactionRule{RedactionRule: rule}
		if rule.Target == "regex" {
			compiled.pattern = regexp.MustCompile(rule.Pattern)
		}
		redactor.rules = append(redactor.rules, compiled)
	}
	return redactor, nil
}

func (redactor *Redactor) Apply(event *Event) {
	if redactor == nil || !redactor.enabled || event == nil {
		return
	}
	event.Body = redactor.redactText(event.Body)
	redactor.redactMap(event.Attributes)
	redactor.redactMap(event.ResourceAttributes)
}

func (redactor *Redactor) redactMap(values map[string]string) {
	for key, value := range values {
		normalized := normalizeSensitiveField(key)
		if redactor.isSensitiveField(normalized) {
			values[key] = redactedValue
			continue
		}
		for _, rule := range redactor.rules {
			switch rule.Target {
			case "field", "json_path":
				if fieldPathMatches(normalized, rule.Field) {
					if rule.Action == "drop_field" {
						delete(values, key)
					} else {
						values[key] = applyRedactionAction(value, rule.RedactionRule)
					}
					goto nextField
				}
			case "regex":
				values[key] = replacePattern(value, rule)
			}
		}
	nextField:
	}
}

func (redactor *Redactor) redactText(value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}
	var structured interface{}
	if json.Unmarshal([]byte(value), &structured) == nil {
		redactor.redactJSONValue(&structured, "")
		if raw, err := json.Marshal(structured); err == nil {
			value = string(raw)
		}
	} else {
		value = redactKeyValueText(value, redactor.sensitiveFields)
	}
	for _, rule := range redactor.rules {
		if rule.Target == "regex" {
			value = replacePattern(value, rule)
		}
	}
	return value
}

func (redactor *Redactor) redactJSONValue(value *interface{}, path string) {
	switch typed := (*value).(type) {
	case map[string]interface{}:
		for key, child := range typed {
			childPath := normalizeSensitiveField(strings.TrimPrefix(path+"."+key, "."))
			if redactor.isSensitiveField(childPath) {
				typed[key] = redactedValue
				continue
			}
			dropped := false
			for _, rule := range redactor.rules {
				if (rule.Target == "field" || rule.Target == "json_path") && fieldPathMatches(childPath, rule.Field) {
					if rule.Action == "drop_field" {
						delete(typed, key)
						dropped = true
					} else {
						typed[key] = applyRedactionAction(fmt.Sprint(child), rule.RedactionRule)
					}
					break
				}
			}
			if dropped {
				continue
			}
			child = typed[key]
			redactor.redactJSONValue(&child, childPath)
			typed[key] = child
		}
	case []interface{}:
		for index := range typed {
			child := typed[index]
			redactor.redactJSONValue(&child, path)
			typed[index] = child
		}
	case string:
		for _, rule := range redactor.rules {
			if rule.Target == "regex" {
				typed = replacePattern(typed, rule)
			}
		}
		*value = typed
	}
}

func (redactor *Redactor) isSensitiveField(path string) bool {
	if _, ok := redactor.sensitiveFields[path]; ok {
		return true
	}
	segments := strings.Split(path, ".")
	if len(segments) > 1 {
		_, ok := redactor.sensitiveFields[segments[len(segments)-1]]
		return ok
	}
	return false
}

func normalizeSensitiveField(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "$."))
	return strings.ToLower(value)
}

func fieldPathMatches(path, configured string) bool {
	configured = normalizeSensitiveField(configured)
	if path == configured {
		return true
	}
	return !strings.Contains(configured, ".") && strings.HasSuffix(path, "."+configured)
}

func redactKeyValueText(value string, fields map[string]struct{}) string {
	for field := range fields {
		if strings.Contains(field, ".") {
			continue
		}
		pattern := regexp.MustCompile(`(?i)(["']?` + regexp.QuoteMeta(field) + `["']?\s*[:=]\s*)(["']?)([^\s,"';}\]]+)`)
		value = pattern.ReplaceAllString(value, `${1}${2}`+redactedValue)
	}
	return value
}

func replacePattern(value string, rule compiledRedactionRule) string {
	if rule.pattern == nil {
		return value
	}
	return rule.pattern.ReplaceAllStringFunc(value, func(match string) string {
		return applyRedactionAction(match, rule.RedactionRule)
	})
}

func applyRedactionAction(value string, rule RedactionRule) string {
	switch rule.Action {
	case "hash":
		sum := sha256.Sum256([]byte(value))
		return "sha256:" + hex.EncodeToString(sum[:8])
	case "drop_field":
		return ""
	default:
		if rule.Replacement != "" {
			return rule.Replacement
		}
		return redactedValue
	}
}
