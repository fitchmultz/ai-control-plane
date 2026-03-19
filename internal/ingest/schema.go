// Package ingest provides typed vendor-evidence ingest and normalization.
//
// Purpose:
//   - Load the tracked normalized evidence schema and validate normalized vendor
//     records against its required-field and type contracts.
//
// Responsibilities:
//   - Parse `demo/config/normalized_schema.yaml` into typed field definitions.
//   - Validate required fields, primitive types, timestamps, and enums.
//   - Provide reusable dotted-path lookup helpers for nested normalized records.
//
// Scope:
//   - Schema loading and normalized-record validation only.
//
// Usage:
//   - Used by the ingest workflow before persisting normalized evidence.
//
// Invariants/Assumptions:
//   - Validation stays deterministic and side-effect free.
package ingest

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	validationissues "github.com/mitchfultz/ai-control-plane/internal/validation"
	"gopkg.in/yaml.v3"
)

type schemaDocument struct {
	NormalizedEvidenceSchema struct {
		Version      string            `yaml:"version"`
		EntityGroups []schemaGroupNode `yaml:"entity_groups"`
	} `yaml:"normalized_evidence_schema"`
}

type schemaGroupNode struct {
	Name   string            `yaml:"name"`
	Fields []schemaFieldNode `yaml:"fields"`
}

type schemaFieldNode struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Required bool   `yaml:"required"`
}

// Schema captures the tracked normalized evidence field contract.
type Schema struct {
	Version string
	Fields  map[string]SchemaField
}

// SchemaField captures one normalized evidence field definition.
type SchemaField struct {
	Type     string
	Required bool
}

// LoadSchema loads the tracked normalized evidence schema from disk.
func LoadSchema(path string) (Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Schema{}, fmt.Errorf("read normalized schema: %w", err)
	}
	var doc schemaDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Schema{}, fmt.Errorf("parse normalized schema: %w", err)
	}
	fields := make(map[string]SchemaField)
	for _, group := range doc.NormalizedEvidenceSchema.EntityGroups {
		for _, field := range group.Fields {
			name := strings.TrimSpace(field.Name)
			if name == "" {
				continue
			}
			fields[name] = SchemaField{Type: strings.TrimSpace(field.Type), Required: field.Required}
		}
	}
	if len(fields) == 0 {
		return Schema{}, fmt.Errorf("normalized schema did not define any fields")
	}
	return Schema{Version: strings.TrimSpace(doc.NormalizedEvidenceSchema.Version), Fields: fields}, nil
}

// ValidateRecord validates one normalized record against the tracked schema.
func (s Schema) ValidateRecord(record map[string]any, index int) []string {
	issues := validationissues.NewIssues(len(s.Fields))
	for name, field := range s.Fields {
		value, ok := lookupPath(record, name)
		if !ok || value == nil || isEmptyValue(value) {
			if field.Required {
				issues.Addf("record[%d]: %s is required", index, name)
			}
			continue
		}
		issues.Extend(validateValueType(index, name, field.Type, value))
	}
	return issues.Sorted()
}

func validateValueType(index int, name string, fieldType string, value any) []string {
	issues := validationissues.NewIssues(1)
	typeName, enumValues := parseFieldType(fieldType)
	switch typeName {
	case "string":
		if _, ok := value.(string); !ok {
			issues.Addf("record[%d]: %s must be a string", index, name)
		}
	case "timestamp":
		raw, ok := value.(string)
		if !ok {
			issues.Addf("record[%d]: %s must be an RFC3339 timestamp string", index, name)
			break
		}
		if _, err := time.Parse(time.RFC3339, raw); err != nil {
			issues.Addf("record[%d]: %s must be RFC3339 (got %q)", index, name, raw)
		}
	case "integer":
		if !isIntegerValue(value) {
			issues.Addf("record[%d]: %s must be an integer", index, name)
		}
	case "number":
		if !isNumberValue(value) {
			issues.Addf("record[%d]: %s must be numeric", index, name)
		}
	case "enum":
		raw, ok := value.(string)
		if !ok {
			issues.Addf("record[%d]: %s must be one of [%s]", index, name, strings.Join(enumValues, ", "))
			break
		}
		matched := false
		for _, candidate := range enumValues {
			if raw == candidate {
				matched = true
				break
			}
		}
		if !matched {
			issues.Addf("record[%d]: %s must be one of [%s] (got %q)", index, name, strings.Join(enumValues, ", "), raw)
		}
	}
	return issues.ToSlice()
}

func parseFieldType(fieldType string) (string, []string) {
	trimmed := strings.TrimSpace(fieldType)
	if strings.HasPrefix(trimmed, "enum[") && strings.HasSuffix(trimmed, "]") {
		inner := strings.TrimSuffix(strings.TrimPrefix(trimmed, "enum["), "]")
		parts := strings.Split(inner, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			candidate := strings.TrimSpace(part)
			if candidate != "" {
				values = append(values, candidate)
			}
		}
		return "enum", values
	}
	return trimmed, nil
}

func setPath(root map[string]any, dottedPath string, value any) {
	parts := strings.Split(strings.TrimSpace(dottedPath), ".")
	if len(parts) == 0 {
		return
	}
	cursor := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := cursor[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			cursor[part] = next
		}
		cursor = next
	}
	cursor[parts[len(parts)-1]] = value
}

func lookupPath(root map[string]any, dottedPath string) (any, bool) {
	parts := strings.Split(strings.TrimSpace(dottedPath), ".")
	if len(parts) == 0 {
		return nil, false
	}
	var current any = root
	for _, part := range parts {
		mapping, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := mapping[part]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func isEmptyValue(value any) bool {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case []string:
		return len(typed) == 0
	default:
		return false
	}
}

func isIntegerValue(value any) bool {
	switch typed := value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return typed == float32(math.Trunc(float64(typed)))
	case float64:
		return typed == math.Trunc(typed)
	default:
		return false
	}
}

func isNumberValue(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	default:
		return false
	}
}
