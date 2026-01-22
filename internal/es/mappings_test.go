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
