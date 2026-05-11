package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

var _ api.FormService = (*fakeFormService)(nil)

type fakeFormService struct{}

func (f *fakeFormService) CreateForm(context.Context, *types.FormDefinition) error { return nil }
func (f *fakeFormService) GetForm(context.Context, string) (*types.FormDefinition, error) {
	return &types.FormDefinition{}, nil
}
func (f *fakeFormService) UpdateForm(context.Context, *types.FormDefinition) error { return nil }
func (f *fakeFormService) ListForms(context.Context, types.FormListParams) ([]*types.FormDefinition, int64, error) {
	return []*types.FormDefinition{}, 0, nil
}
func (f *fakeFormService) PublishForm(context.Context, string, string, string) error { return nil }
func (f *fakeFormService) RollbackForm(context.Context, string, int, string) error   { return nil }
func (f *fakeFormService) GetFormVersions(context.Context, string) ([]*types.FormVersionHistory, error) {
	return []*types.FormVersionHistory{}, nil
}
func (f *fakeFormService) SaveFormInstance(context.Context, *types.FormInstance) error { return nil }
func (f *fakeFormService) GetFormInstance(context.Context, string) (*types.FormInstance, error) {
	return &types.FormInstance{}, nil
}
func (f *fakeFormService) GetFormInstanceByWorkflow(context.Context, string, string) (*types.FormInstance, error) {
	return &types.FormInstance{}, nil
}

func TestD6FormRoutes_NotRegisteredWithoutInjection(t *testing.T) {
	srv := NewServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/forms", nil)
	srv.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestD6FormRoutes_RegisteredAfterInjection(t *testing.T) {
	srv := NewServer()
	srv.SetD6FormService(nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/forms", nil)
	srv.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestD6FormList_AvailableAfterInjection(t *testing.T) {
	srv := NewServer()
	srv.SetD6FormService(&fakeFormService{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/forms", nil)
	srv.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
