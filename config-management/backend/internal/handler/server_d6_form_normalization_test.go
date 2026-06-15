package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

type capturingFormService struct {
	created          *types.FormDefinition
	updated          *types.FormDefinition
	publishOperator  string
	rollbackOperator string
	listCount        int64
	listItems        []*types.FormDefinition
}

var _ api.FormService = (*capturingFormService)(nil)

func (s *capturingFormService) CreateForm(_ context.Context, def *types.FormDefinition) error {
	s.created = def
	return nil
}

func (s *capturingFormService) GetForm(context.Context, string) (*types.FormDefinition, error) {
	return &types.FormDefinition{}, nil
}

func (s *capturingFormService) UpdateForm(_ context.Context, def *types.FormDefinition) error {
	s.updated = def
	return nil
}

func (s *capturingFormService) ListForms(context.Context, types.FormListParams) ([]*types.FormDefinition, int64, error) {
	if s.listItems != nil || s.listCount != 0 {
		return s.listItems, s.listCount, nil
	}
	return []*types.FormDefinition{}, 0, nil
}

func (s *capturingFormService) PublishForm(_ context.Context, _ string, _ string, operatorID string) error {
	s.publishOperator = operatorID
	return nil
}

func (s *capturingFormService) RollbackForm(_ context.Context, _ string, _ int, operatorID string) error {
	s.rollbackOperator = operatorID
	return nil
}
func (s *capturingFormService) GetFormVersions(context.Context, string) ([]*types.FormVersionHistory, error) {
	return []*types.FormVersionHistory{}, nil
}
func (s *capturingFormService) SaveFormInstance(context.Context, *types.FormInstance) error {
	return nil
}
func (s *capturingFormService) GetFormInstance(context.Context, string) (*types.FormInstance, error) {
	return &types.FormInstance{}, nil
}
func (s *capturingFormService) GetFormInstanceByWorkflow(context.Context, string, string) (*types.FormInstance, error) {
	return &types.FormInstance{}, nil
}

func TestCreateForm_NormalizesSchemaInHandlerAndResponse(t *testing.T) {
	srv := NewServer()
	svc := &capturingFormService{}
	srv.SetD6FormService(svc)

	body := `{
		"formName":"demo",
		"schema":{"type":"object","properties":{"name":{"type":"string"}}},
		"formSchema":{"type":"object","properties":{"name":{"type":"number"}}}
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/forms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	srv.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if svc.created == nil {
		t.Fatalf("service should receive create request")
	}
	if !reflect.DeepEqual(svc.created.Schema, svc.created.FormSchema) {
		t.Fatalf("service should receive normalized schema and formSchema")
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatalf("response data should be an object")
	}
	schema, ok := data["schema"].(map[string]any)
	if !ok {
		t.Fatalf("response should include schema object")
	}
	formSchema, ok := data["formSchema"].(map[string]any)
	if !ok {
		t.Fatalf("response should include formSchema object")
	}
	if !reflect.DeepEqual(schema, formSchema) {
		t.Fatalf("schema and formSchema should match in response")
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema.properties should be an object")
	}
	nameField, ok := properties["name"].(map[string]any)
	if !ok {
		t.Fatalf("schema.properties.name should be an object")
	}
	if got, _ := nameField["type"].(string); got != "string" {
		t.Fatalf("schema should take precedence, got type=%q", got)
	}
}
