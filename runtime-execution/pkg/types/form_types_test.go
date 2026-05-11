package types

import (
	"encoding/json"
	"testing"
)

func TestFormDefinitionUnmarshal_SchemaObjectPreferred(t *testing.T) {
	payload := []byte(`{
		"formId":"F-1",
		"schema":{"type":"object","properties":{"name":{"type":"string"}}},
		"formSchema":{"type":"object","properties":{"name":{"type":"number"}}}
	}`)

	var def FormDefinition
	if err := json.Unmarshal(payload, &def); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	props, ok := def.FormSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("FormSchema.properties should be object")
	}
	nameField, ok := props["name"].(map[string]interface{})
	if !ok {
		t.Fatalf("FormSchema.properties.name should be object")
	}
	if got, _ := nameField["type"].(string); got != "string" {
		t.Fatalf("schema should override formSchema, got type=%q", got)
	}
}

func TestFormDefinitionUnmarshal_SchemaStringPreferred(t *testing.T) {
	payload := []byte(`{
		"formId":"F-2",
		"schema":"{\"type\":\"object\",\"properties\":{\"age\":{\"type\":\"number\"}}}",
		"formSchema":{"type":"object","properties":{"age":{"type":"string"}}}
	}`)

	var def FormDefinition
	if err := json.Unmarshal(payload, &def); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	props, ok := def.FormSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("FormSchema.properties should be object")
	}
	ageField, ok := props["age"].(map[string]interface{})
	if !ok {
		t.Fatalf("FormSchema.properties.age should be object")
	}
	if got, _ := ageField["type"].(string); got != "number" {
		t.Fatalf("schema string should override formSchema, got type=%q", got)
	}
}

func TestFormDefinitionUnmarshal_FormSchemaStringCompatible(t *testing.T) {
	payload := []byte(`{
		"formId":"F-3",
		"formSchema":"{\"type\":\"object\",\"properties\":{\"title\":{\"type\":\"string\"}}}"
	}`)

	var def FormDefinition
	if err := json.Unmarshal(payload, &def); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got, _ := def.FormSchema["type"].(string); got != "object" {
		t.Fatalf("FormSchema.type should be object, got=%q", got)
	}
}

func TestFormDefinitionNormalizeSchema_Method(t *testing.T) {
	var def FormDefinition
	if err := def.NormalizeSchema(
		`{"type":"object","properties":{"x":{"type":"string"}}}`,
		map[string]interface{}{"type": "object", "properties": map[string]interface{}{"x": map[string]interface{}{"type": "number"}}},
	); err != nil {
		t.Fatalf("NormalizeSchema() error = %v", err)
	}

	props, ok := def.FormSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("FormSchema.properties should be object")
	}
	xField, ok := props["x"].(map[string]interface{})
	if !ok {
		t.Fatalf("FormSchema.properties.x should be object")
	}
	if got, _ := xField["type"].(string); got != "string" {
		t.Fatalf("schema should override formSchema in NormalizeSchema(), got type=%q", got)
	}
}
