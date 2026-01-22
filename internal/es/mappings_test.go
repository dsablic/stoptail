package es

import (
	"encoding/json"
	"testing"
)

func TestParseMappingPropertiesResponse(t *testing.T) {
	raw := `{
		"products": {
			"mappings": {
				"properties": {
					"name": {"type": "text"},
					"price": {"type": "float"},
					"category": {
						"type": "text",
						"analyzer": "custom_analyzer",
						"fields": {
							"keyword": {"type": "keyword"}
						}
					},
					"address": {
						"properties": {
							"city": {"type": "keyword"},
							"zip": {"type": "keyword", "index": false}
						}
					}
				}
			}
		}
	}`

	var response map[string]struct {
		Mappings struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"mappings"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	props := response["products"].Mappings.Properties
	if len(props) != 4 {
		t.Errorf("got %d properties, want 4", len(props))
	}
}

func TestParseMappingFields(t *testing.T) {
	fields := parseMappingProperties(map[string]json.RawMessage{
		"name":  json.RawMessage(`{"type": "text"}`),
		"price": json.RawMessage(`{"type": "float"}`),
	}, "")

	if len(fields) != 2 {
		t.Fatalf("got %d fields, want 2", len(fields))
	}

	nameFound := false
	for _, f := range fields {
		if f.Name == "name" && f.Type == "text" {
			nameFound = true
		}
	}
	if !nameFound {
		t.Error("name field not found or wrong type")
	}
}

func TestParseMappingPropertiesComprehensive(t *testing.T) {
	props := map[string]json.RawMessage{
		"title": json.RawMessage(`{
			"type": "text",
			"analyzer": "custom_text",
			"fields": {
				"keyword": {"type": "keyword"},
				"autocomplete": {"type": "text", "analyzer": "edge_ngram"}
			}
		}`),
		"price": json.RawMessage(`{"type": "float", "doc_values": false}`),
		"in_stock": json.RawMessage(`{"type": "boolean", "index": false}`),
		"description": json.RawMessage(`{"type": "text", "norms": false}`),
		"metadata": json.RawMessage(`{"type": "keyword", "store": true}`),
		"default_value": json.RawMessage(`{"type": "keyword", "null_value": "N/A"}`),
		"address": json.RawMessage(`{
			"properties": {
				"city": {"type": "keyword"},
				"zip": {"type": "keyword", "index": false},
				"geo": {
					"properties": {
						"lat": {"type": "float"},
						"lon": {"type": "float"}
					}
				}
			}
		}`),
	}

	fields := parseMappingProperties(props, "")

	fieldMap := make(map[string]MappingField)
	for _, f := range fields {
		fieldMap[f.Name] = f
	}

	tests := []struct {
		name       string
		wantType   string
		wantProps  map[string]string
		wantNested int
	}{
		{"title", "text", map[string]string{"analyzer": "custom_text"}, 2},
		{"price", "float", map[string]string{"doc_values": "false"}, 0},
		{"in_stock", "boolean", map[string]string{"index": "false"}, 0},
		{"description", "text", map[string]string{"norms": "false"}, 0},
		{"metadata", "keyword", map[string]string{"store": "true"}, 0},
		{"default_value", "keyword", map[string]string{"null_value": "N/A"}, 0},
		{"address", "object", nil, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, ok := fieldMap[tt.name]
			if !ok {
				t.Fatalf("field %q not found", tt.name)
			}
			if f.Type != tt.wantType {
				t.Errorf("field %q type = %q, want %q", tt.name, f.Type, tt.wantType)
			}
			for k, v := range tt.wantProps {
				if f.Properties[k] != v {
					t.Errorf("field %q property %q = %q, want %q", tt.name, k, f.Properties[k], v)
				}
			}
			if len(f.Children) != tt.wantNested {
				t.Errorf("field %q children = %d, want %d", tt.name, len(f.Children), tt.wantNested)
			}
		})
	}

	addressField := fieldMap["address"]
	childMap := make(map[string]MappingField)
	for _, c := range addressField.Children {
		childMap[c.Name] = c
	}

	if geo, ok := childMap["address.geo"]; !ok {
		t.Error("address.geo not found")
	} else if len(geo.Children) != 2 {
		t.Errorf("address.geo children = %d, want 2", len(geo.Children))
	}

	if zip, ok := childMap["address.zip"]; !ok {
		t.Error("address.zip not found")
	} else if zip.Properties["index"] != "false" {
		t.Errorf("address.zip index = %q, want %q", zip.Properties["index"], "false")
	}
}
