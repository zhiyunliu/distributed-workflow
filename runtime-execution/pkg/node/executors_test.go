package node

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// testCtx 构造测试用 WorkflowContext
func testCtx() *types.WorkflowContext {
	return &types.WorkflowContext{
		InstanceID: "test-instance-001",
		Data:       map[string]interface{}{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// LogExecutor
// ─────────────────────────────────────────────────────────────────────────────

func TestLogExecutor_Type(t *testing.T) {
	e := &LogExecutor{}
	if e.Type() != "log" {
		t.Errorf("expected type 'log', got %q", e.Type())
	}
}

func TestLogExecutor_Execute_WithMessage(t *testing.T) {
	e := &LogExecutor{}
	cfg := map[string]interface{}{"message": "hello test"}
	out, err := e.Execute(cfg, nil, testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["logged"] != true {
		t.Errorf("expected logged=true, got %v", out["logged"])
	}
	if out["message"] != "hello test" {
		t.Errorf("expected message='hello test', got %v", out["message"])
	}
}

func TestLogExecutor_Execute_EmptyMessage(t *testing.T) {
	e := &LogExecutor{}
	cfg := map[string]interface{}{}
	out, err := e.Execute(cfg, nil, testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 空消息时使用默认值
	if out["message"] != "(no message)" {
		t.Errorf("expected default message, got %v", out["message"])
	}
}

func TestLogExecutor_Execute_WarnLevel(t *testing.T) {
	e := &LogExecutor{}
	cfg := map[string]interface{}{"message": "warn msg", "level": "warn"}
	_, err := e.Execute(cfg, nil, testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogExecutor_Execute_ErrorLevel(t *testing.T) {
	e := &LogExecutor{}
	cfg := map[string]interface{}{"message": "err msg", "level": "error"}
	_, err := e.Execute(cfg, nil, testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SleepExecutor
// ─────────────────────────────────────────────────────────────────────────────

func TestSleepExecutor_Type(t *testing.T) {
	e := &SleepExecutor{}
	if e.Type() != "sleep" {
		t.Errorf("expected type 'sleep', got %q", e.Type())
	}
}

func TestSleepExecutor_Execute_ZeroDuration(t *testing.T) {
	e := &SleepExecutor{}
	cfg := map[string]interface{}{"duration_ms": 0}
	out, err := e.Execute(cfg, nil, testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["slept_ms"] != int64(0) {
		t.Errorf("expected slept_ms=0, got %v", out["slept_ms"])
	}
}

func TestSleepExecutor_Execute_PositiveDuration(t *testing.T) {
	e := &SleepExecutor{}
	cfg := map[string]interface{}{"duration_ms": float64(10)}
	start := time.Now()
	out, err := e.Execute(cfg, nil, testCtx())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["slept_ms"] != int64(10) {
		t.Errorf("expected slept_ms=10, got %v", out["slept_ms"])
	}
	if elapsed < 10*time.Millisecond {
		t.Errorf("expected sleep >= 10ms, elapsed %v", elapsed)
	}
}

func TestSleepExecutor_Execute_IntType(t *testing.T) {
	e := &SleepExecutor{}
	cfg := map[string]interface{}{"duration_ms": int(5)}
	out, err := e.Execute(cfg, nil, testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["slept_ms"] != int64(5) {
		t.Errorf("expected slept_ms=5, got %v", out["slept_ms"])
	}
}

func TestSleepExecutor_Execute_NegativeDuration(t *testing.T) {
	e := &SleepExecutor{}
	cfg := map[string]interface{}{"duration_ms": float64(-1)}
	_, err := e.Execute(cfg, nil, testCtx())
	if err == nil {
		t.Error("expected error for negative duration, got nil")
	}
}

func TestSleepExecutor_Execute_MissingConfig(t *testing.T) {
	e := &SleepExecutor{}
	// 未提供 duration_ms，默认 0
	out, err := e.Execute(map[string]interface{}{}, nil, testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["slept_ms"] != int64(0) {
		t.Errorf("expected slept_ms=0, got %v", out["slept_ms"])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTPExecutor
// ─────────────────────────────────────────────────────────────────────────────

func TestHTTPExecutor_Type(t *testing.T) {
	e := NewHTTPExecutor()
	if e.Type() != "http" {
		t.Errorf("expected type 'http', got %q", e.Type())
	}
}

func TestHTTPExecutor_Execute_MissingURL(t *testing.T) {
	e := NewHTTPExecutor()
	_, err := e.Execute(map[string]interface{}{}, nil, testCtx())
	if err == nil {
		t.Error("expected error when url is missing")
	}
}

func TestHTTPExecutor_Execute_GET_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	e := NewHTTPExecutor()
	cfg := map[string]interface{}{
		"url":    srv.URL,
		"method": "GET",
	}
	out, err := e.Execute(cfg, nil, testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["status_code"] != http.StatusOK {
		t.Errorf("expected status 200, got %v", out["status_code"])
	}
	if out["response_body"] != `{"ok":true}` {
		t.Errorf("unexpected response_body: %v", out["response_body"])
	}
}

func TestHTTPExecutor_Execute_POST_WithBody(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := http.MaxBytesReader(w, r.Body, 1024).Read(make([]byte, 1024))
		receivedBody = string(make([]byte, b))
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	e := NewHTTPExecutor()
	cfg := map[string]interface{}{
		"url":    srv.URL,
		"method": "POST",
		"body":   `{"key":"value"}`,
	}
	out, err := e.Execute(cfg, nil, testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["status_code"] != http.StatusCreated {
		t.Errorf("expected status 201, got %v", out["status_code"])
	}
	_ = receivedBody
}

func TestHTTPExecutor_Execute_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := NewHTTPExecutor()
	cfg := map[string]interface{}{"url": srv.URL}
	_, err := e.Execute(cfg, nil, testCtx())
	if err == nil {
		t.Error("expected error for 5xx response")
	}
}

func TestHTTPExecutor_Execute_WithHeaders(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Test")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewHTTPExecutor()
	cfg := map[string]interface{}{
		"url":     srv.URL,
		"headers": map[string]interface{}{"X-Test": "hello"},
	}
	_, err := e.Execute(cfg, nil, testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHeader != "hello" {
		t.Errorf("expected header X-Test='hello', got %q", gotHeader)
	}
}


