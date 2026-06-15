package handler

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResponsePayloadOmitsEmptyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payload := responsePayload(200, "", gin.H{"templateId": "tpl-1"}, nil)

	if payload["code"] != 200 {
		t.Fatalf("code = %v, want 200", payload["code"])
	}
	if _, ok := payload["message"]; ok {
		t.Fatalf("message should be omitted, got %v", payload["message"])
	}
	if _, ok := payload["total"]; ok {
		t.Fatalf("total should be omitted, got %v", payload["total"])
	}
	data, ok := payload["data"].(gin.H)
	if !ok {
		t.Fatalf("data type = %T, want gin.H", payload["data"])
	}
	if data["templateId"] != "tpl-1" {
		t.Fatalf("templateId = %v, want tpl-1", data["templateId"])
	}
}

func TestResponsePayloadIncludesMessageAndTotal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payload := responsePayload(200, "操作成功", gin.H{"list": []string{"a"}}, int64(3))

	if payload["message"] != "操作成功" {
		t.Fatalf("message = %v, want 操作成功", payload["message"])
	}
	if payload["total"] != int64(3) {
		t.Fatalf("total = %v, want 3", payload["total"])
	}
}
