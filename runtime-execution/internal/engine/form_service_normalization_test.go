package engine

import (
	"context"
	"reflect"
	"testing"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

type capturingFormRepo struct {
	created *types.FormDefinition
	updated *types.FormDefinition
	list    []*types.FormDefinition
	get     *types.FormDefinition
}

var _ api.FormRepository = (*capturingFormRepo)(nil)

func (r *capturingFormRepo) CreateFormDef(_ context.Context, def *types.FormDefinition) error {
	r.created = def
	return nil
}

func (r *capturingFormRepo) GetFormDef(_ context.Context, _ string) (*types.FormDefinition, error) {
	return r.get, nil
}

func (r *capturingFormRepo) UpdateFormDef(_ context.Context, def *types.FormDefinition) error {
	r.updated = def
	return nil
}

func (r *capturingFormRepo) ListFormDefs(context.Context, types.FormListParams) ([]*types.FormDefinition, int64, error) {
	return r.list, int64(len(r.list)), nil
}

func (r *capturingFormRepo) PublishFormDef(context.Context, string, string, string) error { return nil }
func (r *capturingFormRepo) RollbackFormDef(context.Context, string, int, string) error   { return nil }
func (r *capturingFormRepo) GetFormVersionHistory(context.Context, string) ([]*types.FormVersionHistory, error) {
	return []*types.FormVersionHistory{}, nil
}
func (r *capturingFormRepo) CreateFormInstance(context.Context, *types.FormInstance) error {
	return nil
}
func (r *capturingFormRepo) UpdateFormInstance(context.Context, *types.FormInstance) error {
	return nil
}
func (r *capturingFormRepo) GetFormInstance(context.Context, string) (*types.FormInstance, error) {
	return &types.FormInstance{}, nil
}
func (r *capturingFormRepo) GetFormInstanceByWorkflow(context.Context, string, string) (*types.FormInstance, error) {
	return &types.FormInstance{}, nil
}

func TestFormServiceCreateForm_NormalizesBeforePersisting(t *testing.T) {
	repo := &capturingFormRepo{}
	svc := NewFormService(repo)

	def := &types.FormDefinition{
		FormName: "demo",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
		},
		FormSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "number"},
			},
		},
	}

	if err := svc.CreateForm(context.Background(), def); err != nil {
		t.Fatalf("CreateForm() error = %v", err)
	}
	if repo.created == nil {
		t.Fatalf("repository should receive create request")
	}
	if !reflect.DeepEqual(repo.created.Schema, repo.created.FormSchema) {
		t.Fatalf("repository should receive normalized schema and formSchema")
	}
	properties, ok := repo.created.Schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("normalized schema.properties should be an object")
	}
	nameField, ok := properties["name"].(map[string]interface{})
	if !ok {
		t.Fatalf("normalized schema.properties.name should be an object")
	}
	if got, _ := nameField["type"].(string); got != "string" {
		t.Fatalf("schema should win before persistence, got type=%q", got)
	}
}

func TestFormServiceGetForm_NormalizesRepositoryResponse(t *testing.T) {
	repo := &capturingFormRepo{
		get: &types.FormDefinition{
			FormID: "F-1",
			FormSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"age": map[string]interface{}{"type": "number"},
				},
			},
		},
	}
	svc := NewFormService(repo)

	got, err := svc.GetForm(context.Background(), "F-1")
	if err != nil {
		t.Fatalf("GetForm() error = %v", err)
	}
	if got == nil {
		t.Fatalf("GetForm() should return a definition")
	}
	if !reflect.DeepEqual(got.Schema, got.FormSchema) {
		t.Fatalf("returned schema and formSchema should match")
	}
	properties, ok := got.Schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("returned schema.properties should be an object")
	}
	ageField, ok := properties["age"].(map[string]interface{})
	if !ok {
		t.Fatalf("returned schema.properties.age should be an object")
	}
	if gotType, _ := ageField["type"].(string); gotType != "number" {
		t.Fatalf("returned schema should preserve repository value, got type=%q", gotType)
	}
}

func TestFormServiceListForms_NormalizesAllResults(t *testing.T) {
	repo := &capturingFormRepo{
		list: []*types.FormDefinition{
			{
				FormID: "F-2",
				FormSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}
	svc := NewFormService(repo)

	list, total, err := svc.ListForms(context.Background(), types.FormListParams{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListForms() error = %v", err)
	}
	if total != 1 {
		t.Fatalf("ListForms() total = %d, want 1", total)
	}
	if len(list) != 1 {
		t.Fatalf("ListForms() len = %d, want 1", len(list))
	}
	if !reflect.DeepEqual(list[0].Schema, list[0].FormSchema) {
		t.Fatalf("list item schema and formSchema should match")
	}
}
