package es

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func parseMappingProperties(props map[string]json.RawMessage, prefix string) []MappingField {
	var fields []MappingField

	for name, raw := range props {
		var prop struct {
			Type       string                     `json:"type"`
			Properties map[string]json.RawMessage `json:"properties"`
			Analyzer   string                     `json:"analyzer"`
			Index      *bool                      `json:"index"`
			DocValues  *bool                      `json:"doc_values"`
			Norms      *bool                      `json:"norms"`
			Store      *bool                      `json:"store"`
			NullValue  any                        `json:"null_value"`
			Fields     map[string]json.RawMessage `json:"fields"`
		}
		if err := json.Unmarshal(raw, &prop); err != nil {
			continue
		}

		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}

		field := MappingField{
			Name:       fullName,
			Type:       prop.Type,
			Properties: make(map[string]string),
		}

		if prop.Type == "" && prop.Properties != nil {
			field.Type = "object"
		}

		if prop.Analyzer != "" {
			field.Properties["analyzer"] = prop.Analyzer
		}
		if prop.Index != nil && !*prop.Index {
			field.Properties["index"] = "false"
		}
		if prop.DocValues != nil && !*prop.DocValues {
			field.Properties["doc_values"] = "false"
		}
		if prop.Norms != nil && !*prop.Norms {
			field.Properties["norms"] = "false"
		}
		if prop.Store != nil && *prop.Store {
			field.Properties["store"] = "true"
		}
		if prop.NullValue != nil {
			field.Properties["null_value"] = fmt.Sprintf("%v", prop.NullValue)
		}

		if prop.Fields != nil {
			for subName, subRaw := range prop.Fields {
				var subProp struct {
					Type     string `json:"type"`
					Analyzer string `json:"analyzer"`
				}
				if err := json.Unmarshal(subRaw, &subProp); err == nil {
					subField := MappingField{
						Name:       fullName + "." + subName,
						Type:       subProp.Type,
						Properties: make(map[string]string),
					}
					if subProp.Analyzer != "" {
						subField.Properties["analyzer"] = subProp.Analyzer
					}
					subField.Properties["multi_field"] = "true"
					field.Children = append(field.Children, subField)
				}
			}
		}

		if prop.Properties != nil {
			children := parseMappingProperties(prop.Properties, fullName)
			directPrefix := fullName + "."
			for _, c := range children {
				suffix := strings.TrimPrefix(c.Name, directPrefix)
				if !strings.Contains(suffix, ".") {
					field.Children = append(field.Children, c)
				}
			}
			fields = append(fields, field)
		} else {
			fields = append(fields, field)
		}
	}

	return fields
}

func flattenFields(fields []MappingField) []string {
	var result []string
	for _, f := range fields {
		if f.Type != "object" {
			result = append(result, f.Name)
		}
		if len(f.Children) > 0 {
			result = append(result, flattenFields(f.Children)...)
		}
	}
	return result
}

func (c *Client) FetchIndexMappings(ctx context.Context, indexName string) (*IndexMappings, error) {
	res, err := c.es.Indices.GetMapping(
		c.es.Indices.GetMapping.WithContext(ctx),
		c.es.Indices.GetMapping.WithIndex(indexName),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching mappings: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "mappings")
	if err != nil {
		return nil, err
	}

	var response map[string]struct {
		Mappings struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"mappings"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing mappings: %w", err)
	}

	result := &IndexMappings{IndexName: indexName}
	if indexData, ok := response[indexName]; ok {
		result.Fields = parseMappingProperties(indexData.Mappings.Properties, "")
		result.FieldCount = len(flattenFields(result.Fields))
	}

	return result, nil
}

func (c *Client) FetchMapping(ctx context.Context, index string) ([]string, error) {
	mappings, err := c.FetchIndexMappings(ctx, index)
	if err != nil {
		return nil, err
	}
	return flattenFields(mappings.Fields), nil
}

func (c *Client) FetchIndexAnalyzers(ctx context.Context, indexName string) ([]AnalyzerInfo, error) {
	res, err := c.es.Indices.GetSettings(
		c.es.Indices.GetSettings.WithContext(ctx),
		c.es.Indices.GetSettings.WithIndex(indexName),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching settings: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "settings")
	if err != nil {
		return nil, err
	}

	var response map[string]struct {
		Settings struct {
			Index struct {
				Analysis struct {
					Analyzer   map[string]json.RawMessage `json:"analyzer"`
					Tokenizer  map[string]json.RawMessage `json:"tokenizer"`
					Filter     map[string]json.RawMessage `json:"filter"`
					Normalizer map[string]json.RawMessage `json:"normalizer"`
				} `json:"analysis"`
			} `json:"index"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}

	var analyzers []AnalyzerInfo
	indexData := response[indexName]
	analysis := indexData.Settings.Index.Analysis

	analyzerSources := []struct {
		items map[string]json.RawMessage
		kind  string
	}{
		{analysis.Analyzer, "analyzer"},
		{analysis.Tokenizer, "tokenizer"},
		{analysis.Filter, "filter"},
		{analysis.Normalizer, "normalizer"},
	}
	for _, src := range analyzerSources {
		for name, raw := range src.items {
			analyzers = append(analyzers, parseAnalyzerInfo(name, src.kind, raw))
		}
	}

	sort.Slice(analyzers, func(i, j int) bool {
		if analyzers[i].Kind != analyzers[j].Kind {
			kindOrder := map[string]int{"analyzer": 0, "tokenizer": 1, "filter": 2, "normalizer": 3}
			return kindOrder[analyzers[i].Kind] < kindOrder[analyzers[j].Kind]
		}
		return analyzers[i].Name < analyzers[j].Name
	})

	return analyzers, nil
}

func parseAnalyzerInfo(name, kind string, raw json.RawMessage) AnalyzerInfo {
	var settings map[string]any
	json.Unmarshal(raw, &settings)

	info := AnalyzerInfo{
		Name:     name,
		Kind:     kind,
		Settings: make(map[string]string),
	}

	for k, v := range settings {
		switch val := v.(type) {
		case string:
			info.Settings[k] = val
		case []any:
			strs := make([]string, len(val))
			for i, s := range val {
				strs[i] = fmt.Sprintf("%v", s)
			}
			info.Settings[k] = strings.Join(strs, ", ")
		default:
			info.Settings[k] = fmt.Sprintf("%v", v)
		}
	}

	return info
}

func (c *Client) CreateIndex(ctx context.Context, name string, shards, replicas int) error {
	reqBodyObj := struct {
		Settings struct {
			NumberOfShards   int `json:"number_of_shards"`
			NumberOfReplicas int `json:"number_of_replicas"`
		} `json:"settings"`
	}{}
	reqBodyObj.Settings.NumberOfShards = shards
	reqBodyObj.Settings.NumberOfReplicas = replicas
	reqBytes, _ := json.Marshal(reqBodyObj)
	res, err := c.es.Indices.Create(
		name,
		c.es.Indices.Create.WithContext(ctx),
		c.es.Indices.Create.WithBody(strings.NewReader(string(reqBytes))),
	)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer res.Body.Close()

	return checkError(res)
}

func (c *Client) DeleteIndex(ctx context.Context, name string) error {
	res, err := c.es.Indices.Delete(
		[]string{name},
		c.es.Indices.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("deleting index: %w", err)
	}
	defer res.Body.Close()

	return checkError(res)
}

func (c *Client) OpenIndex(ctx context.Context, name string) error {
	res, err := c.es.Indices.Open(
		[]string{name},
		c.es.Indices.Open.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer res.Body.Close()

	return checkError(res)
}

func (c *Client) CloseIndex(ctx context.Context, name string) error {
	res, err := c.es.Indices.Close(
		[]string{name},
		c.es.Indices.Close.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("closing index: %w", err)
	}
	defer res.Body.Close()

	return checkError(res)
}

func (c *Client) updateAlias(ctx context.Context, action, index, alias string) error {
	type aliasTarget struct {
		Index string `json:"index"`
		Alias string `json:"alias"`
	}
	reqBodyObj := struct {
		Actions []map[string]aliasTarget `json:"actions"`
	}{
		Actions: []map[string]aliasTarget{{action: {Index: index, Alias: alias}}},
	}
	reqBytes, _ := json.Marshal(reqBodyObj)
	res, err := c.es.Indices.UpdateAliases(
		strings.NewReader(string(reqBytes)),
		c.es.Indices.UpdateAliases.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("%s alias: %w", action, err)
	}
	defer res.Body.Close()

	return checkError(res)
}

func (c *Client) AddAlias(ctx context.Context, index, alias string) error {
	return c.updateAlias(ctx, "add", index, alias)
}

func (c *Client) RemoveAlias(ctx context.Context, index, alias string) error {
	return c.updateAlias(ctx, "remove", index, alias)
}

func (c *Client) FetchIndexSettings(ctx context.Context, indexName string) (*IndexSettings, error) {
	res, err := c.es.Indices.GetSettings(
		c.es.Indices.GetSettings.WithContext(ctx),
		c.es.Indices.GetSettings.WithIndex(indexName),
		c.es.Indices.GetSettings.WithFlatSettings(true),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching index settings: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "index settings")
	if err != nil {
		return nil, err
	}

	return parseIndexSettings(indexName, body)
}

func parseIndexSettings(indexName string, data []byte) (*IndexSettings, error) {
	var response map[string]struct {
		Settings map[string]any `json:"settings"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parsing index settings: %w", err)
	}

	settings := &IndexSettings{
		IndexName:   indexName,
		AllSettings: make(map[string]string),
	}

	if indexData, ok := response[indexName]; ok {
		for k, rawVal := range indexData.Settings {
			v := formatSettingValue(rawVal)
			settings.AllSettings[k] = v
			switch k {
			case "index.number_of_shards":
				settings.NumberOfShards = v
			case "index.number_of_replicas":
				settings.NumberOfReplicas = v
			case "index.refresh_interval":
				settings.RefreshInterval = v
			case "index.codec":
				settings.Codec = v
			case "index.creation_date":
				settings.CreationDate = v
			case "index.uuid":
				settings.UUID = v
			case "index.version.created":
				settings.Version = v
			case "index.routing.allocation.enable":
				settings.RoutingAllocation = v
			}
		}
	}

	return settings, nil
}

func (c *Client) FetchIndexTemplates(ctx context.Context) ([]IndexTemplate, error) {
	res, err := c.es.Indices.GetIndexTemplate(
		c.es.Indices.GetIndexTemplate.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching index templates: %w", err)
	}
	defer res.Body.Close()

	body, err := readBody(res, "templates")
	if err != nil {
		return nil, err
	}

	var response struct {
		IndexTemplates []struct {
			Name     string `json:"name"`
			Template struct {
				IndexPatterns []string `json:"index_patterns"`
				ComposedOf    []string `json:"composed_of"`
				Priority      int      `json:"priority"`
				Version       int      `json:"version"`
				DataStream    *struct{} `json:"data_stream"`
				Template      struct {
					Settings struct {
						Index struct {
							NumberOfShards   string `json:"number_of_shards"`
							NumberOfReplicas string `json:"number_of_replicas"`
						} `json:"index"`
					} `json:"settings"`
				} `json:"template"`
			} `json:"index_template"`
		} `json:"index_templates"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	var templates []IndexTemplate
	for _, item := range response.IndexTemplates {
		t := IndexTemplate{
			Name:             item.Name,
			IndexPatterns:    item.Template.IndexPatterns,
			Priority:         item.Template.Priority,
			Version:          item.Template.Version,
			ComposedOf:       item.Template.ComposedOf,
			NumberOfShards:   item.Template.Template.Settings.Index.NumberOfShards,
			NumberOfReplicas: item.Template.Template.Settings.Index.NumberOfReplicas,
			DataStream:       item.Template.DataStream != nil,
		}
		templates = append(templates, t)
	}

	sort.Slice(templates, func(i, j int) bool {
		if templates[i].Priority != templates[j].Priority {
			return templates[i].Priority > templates[j].Priority
		}
		return templates[i].Name < templates[j].Name
	})

	return templates, nil
}
