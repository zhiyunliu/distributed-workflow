package engine

import (
	"testing"
	"time"

	"github.com/zhiyunliu/distributed-workflow/types"
)

func makeWorker(id, nodeType string, maxCap, curLoad int) *types.NodeWorkerInfo {
	return &types.NodeWorkerInfo{
		ID:            id,
		Address:       "127.0.0.1:9001",
		IP:            "127.0.0.1",
		Capabilities:  map[string]int{nodeType: maxCap},
		CurrentLoad:   map[string]int{nodeType: curLoad},
		LastHeartbeat: time.Now(),
		Status:        types.WorkerStatusOnline,
	}
}

// newTestWorkerManager 创建不依赖 gRPC 的 WorkerManager（workerPool 为 nil 即可，不调 AssignTask）
func newTestWorkerManager() *WorkerManagerServiceImpl {
	return &WorkerManagerServiceImpl{}
}

func TestWorkerManager_RegisterAndGetOnline(t *testing.T) {
	m := newTestWorkerManager()

	err := m.RegisterWorker(makeWorker("w1", "log", 5, 0))
	if err != nil {
		t.Fatalf("RegisterWorker: %v", err)
	}

	list := m.GetOnlineWorkers()
	if len(list) != 1 {
		t.Fatalf("expected 1 online worker, got %d", len(list))
	}
	if list[0].ID != "w1" {
		t.Errorf("expected worker w1, got %s", list[0].ID)
	}
}

func TestWorkerManager_RegisterSetsStatusOnline(t *testing.T) {
	m := newTestWorkerManager()
	w := makeWorker("w2", "sleep", 3, 0)
	w.Status = types.WorkerStatusOffline // 注册前故意设为 offline
	_ = m.RegisterWorker(w)

	list := m.GetOnlineWorkers()
	if len(list) != 1 || list[0].Status != types.WorkerStatusOnline {
		t.Errorf("expected status online after register, got %v", list)
	}
}

func TestWorkerManager_UpdateHeartbeat(t *testing.T) {
	m := newTestWorkerManager()
	_ = m.RegisterWorker(makeWorker("w3", "log", 10, 0))

	err := m.UpdateHeartbeat("w3", map[string]int{"log": 7})
	if err != nil {
		t.Fatalf("UpdateHeartbeat: %v", err)
	}

	list := m.GetOnlineWorkers()
	if list[0].CurrentLoad["log"] != 7 {
		t.Errorf("expected load 7, got %d", list[0].CurrentLoad["log"])
	}
}

func TestWorkerManager_UpdateHeartbeat_NotFound(t *testing.T) {
	m := newTestWorkerManager()
	err := m.UpdateHeartbeat("ghost", map[string]int{"log": 1})
	if err == nil {
		t.Error("expected error for unknown worker")
	}
}

func TestWorkerManager_GetWorkersByNodeType(t *testing.T) {
	m := newTestWorkerManager()
	_ = m.RegisterWorker(makeWorker("w4", "log", 5, 0))
	_ = m.RegisterWorker(makeWorker("w5", "sleep", 5, 0))

	logWorkers := m.GetWorkersByNodeType("log")
	if len(logWorkers) != 1 {
		t.Errorf("expected 1 log worker, got %d", len(logWorkers))
	}
	if logWorkers[0].ID != "w4" {
		t.Errorf("expected w4, got %s", logWorkers[0].ID)
	}
}

func TestWorkerManager_SelectBestWorker_MinLoad(t *testing.T) {
	m := newTestWorkerManager()
	// w6: load ratio = 8/10 = 0.8
	w6 := makeWorker("w6", "http", 10, 8)
	// w7: load ratio = 1/10 = 0.1 → 应选 w7
	w7 := makeWorker("w7", "http", 10, 1)
	_ = m.RegisterWorker(w6)
	_ = m.RegisterWorker(w7)

	best, err := m.SelectBestWorker("http")
	if err != nil {
		t.Fatalf("SelectBestWorker: %v", err)
	}
	if best.ID != "w7" {
		t.Errorf("expected w7 (lower load), got %s", best.ID)
	}
}

func TestWorkerManager_SelectBestWorker_NoWorker(t *testing.T) {
	m := newTestWorkerManager()
	_, err := m.SelectBestWorker("unknown-type")
	if err == nil {
		t.Error("expected error when no worker available")
	}
}

func TestWorkerManager_SelectBestWorker_OfflineWorkerIgnored(t *testing.T) {
	m := newTestWorkerManager()
	w := makeWorker("w8", "log", 5, 0)
	_ = m.RegisterWorker(w)
	// 将 w8 标记为 offline
	w.Status = types.WorkerStatusOffline

	_, err := m.SelectBestWorker("log")
	if err == nil {
		t.Error("expected error: offline worker should not be selected")
	}
}

func TestWorkerManager_HandleWorkerOffline(t *testing.T) {
	m := newTestWorkerManager()
	_ = m.RegisterWorker(makeWorker("w9", "log", 5, 0))

	err := m.HandleWorkerOffline("w9")
	if err != nil {
		t.Fatalf("HandleWorkerOffline: %v", err)
	}

	list := m.GetOnlineWorkers()
	if len(list) != 0 {
		t.Errorf("expected 0 online workers after offline, got %d", len(list))
	}
}
