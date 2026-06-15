package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/jwtutil"
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

func TestD6FormPublish_FallsBackToUsernameInContext(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	srv := NewServer()
	srv.engine.Use(func(c *gin.Context) {
		c.Set("username", "alice")
		c.Next()
	})
	svc := &capturingFormService{}
	srv.SetD6FormService(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/forms/demo/publish", strings.NewReader(`{"changeLog":"v1"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if svc.publishOperator != "alice" {
		t.Fatalf("expected operator %q, got %q", "alice", svc.publishOperator)
	}
}

func TestD6FormPublish_UsesBearerTokenWhenContextMissing(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	srv := NewServer()
	srv.jwtSecret = "test-secret"
	svc := &capturingFormService{}
	srv.SetD6FormService(svc)

	token := mustSignToken(t, srv.jwtSecret, jwtutil.Claims{UserID: 88, ExpireAt: time.Now().Add(time.Hour).Unix()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/forms/demo/publish", strings.NewReader(`{"changeLog":"v1","operatorId":"client-forged"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	srv.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if svc.publishOperator != "88" {
		t.Fatalf("expected operator %q, got %q", "88", svc.publishOperator)
	}
}

func TestD6FormRollback_UsesBearerTokenWhenContextMissing(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	srv := NewServer()
	srv.jwtSecret = "test-secret"
	svc := &capturingFormService{}
	srv.SetD6FormService(svc)

	token := mustSignToken(t, srv.jwtSecret, jwtutil.Claims{Username: "bob", ExpireAt: time.Now().Add(time.Hour).Unix()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/forms/demo/rollback", strings.NewReader(`{"targetVersion":2,"operatorId":"client-forged"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	srv.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if svc.rollbackOperator != "bob" {
		t.Fatalf("expected operator %q, got %q", "bob", svc.rollbackOperator)
	}
}

func TestD6FormPublish_RejectsOperatorOnlyWithoutLoginState(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	srv := NewServer()
	srv.jwtSecret = "test-secret"
	svc := &capturingFormService{}
	srv.SetD6FormService(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/forms/demo/publish", strings.NewReader(`{"changeLog":"v1","operatorId":"client-only"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if svc.publishOperator != "" {
		t.Fatalf("expected no operator to be passed, got %q", svc.publishOperator)
	}
}

func mustSignToken(t *testing.T, secret string, claims jwtutil.Claims) string {
	t.Helper()
	token, err := jwtutil.Sign(claims, secret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return token
}

func TestD6FormList_ReturnsPagedDataShape(t *testing.T) {
	srv := NewServer()
	svc := &capturingFormService{
		listCount: 42,
		listItems: []*types.FormDefinition{{FormID: "f1"}},
	}
	srv.SetD6FormService(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/forms", nil)
	srv.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["total"]; ok {
		t.Fatalf("did not expect top-level total field")
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", resp["data"])
	}
	if _, ok := data["list"]; !ok {
		t.Fatalf("expected data.list field")
	}
	if total, ok := data["total"]; !ok || total.(float64) != 42 {
		t.Fatalf("expected data.total=42, got %#v", data["total"])
	}
}
