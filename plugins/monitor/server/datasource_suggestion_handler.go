package server

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
)

type dataSourceSuggestion struct {
	Value      string `json:"value"`
	InsertText string `json:"insertText"`
	Kind       string `json:"kind"`
	Type       string `json:"type"`
	Help       string `json:"help"`
}

type dataSourceIndexOption struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Status    string `json:"status,omitempty"`
	DocsCount string `json:"docsCount,omitempty"`
	StoreSize string `json:"storeSize,omitempty"`
}

type prometheusMetadata struct {
	Type string
	Help string
}

func (h *DataSourceHandler) ListDataSourceIndices(c *gin.Context) {
	dataSource, ok := h.loadDataSource(c)
	if !ok {
		return
	}
	if dataSource.Type != "elasticsearch" {
		c.JSON(400, gin.H{"code": 400, "message": "只有 Elasticsearch 数据源支持索引发现"})
		return
	}

	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	options, err := h.listElasticsearchIndices(c, dataSource, keyword, limit)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取 Elasticsearch 索引失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": options})
}

func (h *DataSourceHandler) SuggestDataSource(c *gin.Context) {
	dataSource, ok := h.loadDataSource(c)
	if !ok {
		return
	}
	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var (
		suggestions []dataSourceSuggestion
		err         error
	)
	switch dataSource.Type {
	case "prometheus", "victoriametrics":
		suggestions, err = h.suggestPrometheusMetrics(c, dataSource, keyword, limit)
	case "loki":
		suggestions, err = h.suggestLokiLabels(c, dataSource, keyword, limit)
	case "elasticsearch":
		suggestions, err = h.suggestElasticsearchFields(c, dataSource, keyword, strings.Trim(c.Query("index"), "/ "), limit)
	default:
		err = fmt.Errorf("不支持的数据源类型: %s", dataSource.Type)
	}
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取查询建议失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": suggestions})
}

func (h *DataSourceHandler) suggestPrometheusMetrics(c *gin.Context, ds *model.DataSource, keyword string, limit int) ([]dataSourceSuggestion, error) {
	raw, _, err := h.doDataSourceRequest(c.Request.Context(), ds, http.MethodGet, "/api/v1/label/__name__/values", nil, nil)
	if err != nil {
		return nil, err
	}
	metadata := h.loadPrometheusMetadata(c, ds)
	names := extractStringArrayFromData(raw)
	sort.Strings(names)

	suggestions := make([]dataSourceSuggestion, 0, limit)
	for _, name := range names {
		if keyword != "" && !strings.Contains(strings.ToLower(name), keyword) {
			continue
		}
		item := dataSourceSuggestion{
			Value:      name,
			InsertText: name,
			Kind:       "metric",
		}
		if meta, ok := metadata[name]; ok {
			item.Type = meta.Type
			item.Help = meta.Help
		}
		suggestions = append(suggestions, item)
		if len(suggestions) >= limit {
			break
		}
	}
	return suggestions, nil
}

func (h *DataSourceHandler) loadPrometheusMetadata(c *gin.Context, ds *model.DataSource) map[string]prometheusMetadata {
	raw, _, err := h.doDataSourceRequest(c.Request.Context(), ds, http.MethodGet, "/api/v1/metadata", map[string]string{"limit": "5000"}, nil)
	if err != nil {
		return map[string]prometheusMetadata{}
	}
	root, ok := raw.(map[string]interface{})
	if !ok {
		return map[string]prometheusMetadata{}
	}
	data, ok := root["data"].(map[string]interface{})
	if !ok {
		return map[string]prometheusMetadata{}
	}
	result := make(map[string]prometheusMetadata, len(data))
	for metric, value := range data {
		items, ok := value.([]interface{})
		if !ok || len(items) == 0 {
			continue
		}
		first, ok := items[0].(map[string]interface{})
		if !ok {
			continue
		}
		result[metric] = prometheusMetadata{
			Type: fmt.Sprint(first["type"]),
			Help: fmt.Sprint(first["help"]),
		}
	}
	return result
}

func (h *DataSourceHandler) suggestLokiLabels(c *gin.Context, ds *model.DataSource, keyword string, limit int) ([]dataSourceSuggestion, error) {
	raw, _, err := h.doDataSourceRequest(c.Request.Context(), ds, http.MethodGet, "/loki/api/v1/labels", nil, nil)
	if err != nil {
		return nil, err
	}
	labels := extractStringArrayFromData(raw)
	sort.Strings(labels)
	suggestions := make([]dataSourceSuggestion, 0, limit)
	for _, label := range labels {
		if keyword != "" && !strings.Contains(strings.ToLower(label), keyword) {
			continue
		}
		suggestions = append(suggestions, dataSourceSuggestion{
			Value:      label,
			InsertText: fmt.Sprintf(`%s=~".*"`, label),
			Kind:       "label",
			Type:       "loki_label",
			Help:       "Loki stream label",
		})
		if len(suggestions) >= limit {
			break
		}
	}
	return suggestions, nil
}

func (h *DataSourceHandler) suggestElasticsearchFields(c *gin.Context, ds *model.DataSource, keyword, index string, limit int) ([]dataSourceSuggestion, error) {
	path := "/_mapping/field/*"
	if index != "" {
		path = "/" + index + "/_mapping/field/*"
	}
	raw, _, err := h.doDataSourceRequest(c.Request.Context(), ds, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	fields := map[string]string{}
	collectElasticsearchFields(raw, fields)
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)

	suggestions := make([]dataSourceSuggestion, 0, limit)
	for _, name := range names {
		if keyword != "" && !strings.Contains(strings.ToLower(name), keyword) {
			continue
		}
		suggestions = append(suggestions, dataSourceSuggestion{
			Value:      name,
			InsertText: name,
			Kind:       "field",
			Type:       fields[name],
			Help:       "Elasticsearch mapping field",
		})
		if len(suggestions) >= limit {
			break
		}
	}
	return suggestions, nil
}

func (h *DataSourceHandler) listElasticsearchIndices(c *gin.Context, ds *model.DataSource, keyword string, limit int) ([]dataSourceIndexOption, error) {
	raw, _, err := h.doDataSourceRequest(c.Request.Context(), ds, http.MethodGet, "/_cat/indices", map[string]string{
		"format": "json",
		"h":      "index,status,docs.count,store.size",
		"s":      "index:asc",
	}, nil)
	if err != nil {
		return nil, err
	}

	optionsByName := map[string]dataSourceIndexOption{}
	for _, item := range collectElasticsearchIndexOptions(raw) {
		optionsByName[item.Name] = item
	}

	if rawAliases, _, err := h.doDataSourceRequest(c.Request.Context(), ds, http.MethodGet, "/_aliases", nil, nil); err == nil {
		for _, item := range collectElasticsearchAliasOptions(rawAliases) {
			if existing, ok := optionsByName[item.Name]; ok && existing.Type == "index" {
				continue
			}
			optionsByName[item.Name] = item
		}
	}

	names := make([]string, 0, len(optionsByName))
	for name := range optionsByName {
		if keyword != "" && !strings.Contains(strings.ToLower(name), keyword) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	options := make([]dataSourceIndexOption, 0, minInt(len(names), limit))
	for _, name := range names {
		options = append(options, optionsByName[name])
		if len(options) >= limit {
			break
		}
	}
	return options, nil
}

func extractStringArrayFromData(raw interface{}) []string {
	root, ok := raw.(map[string]interface{})
	if !ok {
		return []string{}
	}
	data, ok := root["data"].([]interface{})
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(data))
	for _, item := range data {
		if value := strings.TrimSpace(fmt.Sprint(item)); value != "" && value != "<nil>" {
			result = append(result, value)
		}
	}
	return result
}

func collectElasticsearchIndexOptions(raw interface{}) []dataSourceIndexOption {
	items, ok := raw.([]interface{})
	if !ok {
		return []dataSourceIndexOption{}
	}
	options := make([]dataSourceIndexOption, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name := strings.TrimSpace(fmt.Sprint(row["index"]))
		if name == "" || name == "<nil>" {
			continue
		}
		options = append(options, dataSourceIndexOption{
			Name:      name,
			Type:      "index",
			Status:    strings.TrimSpace(fmt.Sprint(row["status"])),
			DocsCount: strings.TrimSpace(fmt.Sprint(row["docs.count"])),
			StoreSize: strings.TrimSpace(fmt.Sprint(row["store.size"])),
		})
	}
	return options
}

func collectElasticsearchAliasOptions(raw interface{}) []dataSourceIndexOption {
	root, ok := raw.(map[string]interface{})
	if !ok {
		return []dataSourceIndexOption{}
	}
	aliases := map[string]struct{}{}
	for _, rawIndex := range root {
		indexInfo, ok := rawIndex.(map[string]interface{})
		if !ok {
			continue
		}
		rawAliases, ok := indexInfo["aliases"].(map[string]interface{})
		if !ok {
			continue
		}
		for name := range rawAliases {
			name = strings.TrimSpace(name)
			if name != "" {
				aliases[name] = struct{}{}
			}
		}
	}
	names := make([]string, 0, len(aliases))
	for name := range aliases {
		names = append(names, name)
	}
	sort.Strings(names)
	options := make([]dataSourceIndexOption, 0, len(names))
	for _, name := range names {
		options = append(options, dataSourceIndexOption{Name: name, Type: "alias"})
	}
	return options
}

func collectElasticsearchFields(raw interface{}, fields map[string]string) {
	switch value := raw.(type) {
	case map[string]interface{}:
		if mappings, ok := value["mappings"].(map[string]interface{}); ok {
			for field, rawField := range mappings {
				fieldInfo, ok := rawField.(map[string]interface{})
				if !ok {
					continue
				}
				if mapping, ok := fieldInfo["mapping"].(map[string]interface{}); ok {
					for _, rawType := range mapping {
						if typeInfo, ok := rawType.(map[string]interface{}); ok {
							fields[field] = fmt.Sprint(typeInfo["type"])
							break
						}
					}
				}
			}
		}
		for _, child := range value {
			collectElasticsearchFields(child, fields)
		}
	case []interface{}:
		for _, child := range value {
			collectElasticsearchFields(child, fields)
		}
	}
}
