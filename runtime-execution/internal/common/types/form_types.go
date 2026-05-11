package types

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// 表单状态常量
// ─────────────────────────────────────────────────────────────────────────────

const (
	// FormStatusDraft 草稿：表单定义尚未发布
	FormStatusDraft = 0
	// FormStatusPublished 已发布：表单定义已发布，可供使用
	FormStatusPublished = 1
	// FormStatusDisabled 已停用：表单定义已停用，不可新建实例
	FormStatusDisabled = 2
)

// ─────────────────────────────────────────────────────────────────────────────
// 表单核心类型
// ─────────────────────────────────────────────────────────────────────────────

// FormDefinition 表单定义
type FormDefinition struct {
	FormID      string                 `json:"formId"`
	FormName    string                 `json:"formName"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`
	FormSchema  map[string]interface{} `json:"formSchema"` // 表单设计器JSON Schema
	Version     int                    `json:"version"`    // 版本号，从1开始自增
	Status      int                    `json:"status"`     // 0=草稿 1=已发布 2=已停用
	CreatedBy   string                 `json:"createdBy"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Deleted     bool                   `json:"deleted"`
}

// NormalizeSchema 归一化 schema/formSchema，固定 schema 优先于 formSchema。
func (f *FormDefinition) NormalizeSchema(schema interface{}, formSchema interface{}) error {
	normalized, err := normalizeFormSchemaValue(schema, formSchema)
	if err != nil {
		return err
	}
	f.Schema = normalized
	f.FormSchema = normalized
	return nil
}

// UnmarshalJSON 兼容 schema/formSchema，支持 string|object，并固定 schema 优先。
func (f *FormDefinition) UnmarshalJSON(data []byte) error {
	type rawFormDefinition struct {
		FormID      string      `json:"formId"`
		FormName    string      `json:"formName"`
		Description string      `json:"description"`
		Schema      interface{} `json:"schema"`
		FormSchema  interface{} `json:"formSchema"`
		Version     int         `json:"version"`
		Status      int         `json:"status"`
		CreatedBy   string      `json:"createdBy"`
		CreatedAt   time.Time   `json:"createdAt"`
		UpdatedAt   time.Time   `json:"updatedAt"`
		Deleted     bool        `json:"deleted"`
	}

	var aux rawFormDefinition
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	f.FormID = aux.FormID
	f.FormName = aux.FormName
	f.Description = aux.Description
	f.Version = aux.Version
	f.Status = aux.Status
	f.CreatedBy = aux.CreatedBy
	f.CreatedAt = aux.CreatedAt
	f.UpdatedAt = aux.UpdatedAt
	f.Deleted = aux.Deleted

	return f.NormalizeSchema(aux.Schema, aux.FormSchema)
}

func normalizeFormSchemaValue(schema interface{}, formSchema interface{}) (map[string]interface{}, error) {
	if !isNilSchemaValue(schema) {
		return parseFormSchemaCandidate(schema)
	}
	if !isNilSchemaValue(formSchema) {
		return parseFormSchemaCandidate(formSchema)
	}
	return map[string]interface{}{}, nil
}

func isNilSchemaValue(value interface{}) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func parseFormSchemaCandidate(value interface{}) (map[string]interface{}, error) {
	switch v := value.(type) {
	case nil:
		return map[string]interface{}{}, nil
	case map[string]interface{}:
		if v == nil {
			return map[string]interface{}{}, nil
		}
		return v, nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return map[string]interface{}{}, nil
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
			return nil, fmt.Errorf("invalid schema string: %w", err)
		}
		if decoded == nil {
			return map[string]interface{}{}, nil
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("unsupported schema type: %T", value)
	}
}

// FormInstance 表单实例
type FormInstance struct {
	InstanceID         string                 `json:"instanceId"`
	FormID             string                 `json:"formId"`
	FormVersion        int                    `json:"formVersion"`
	WorkflowInstanceID string                 `json:"workflowInstanceId"`
	NodeID             string                 `json:"nodeId"`
	FormData           map[string]interface{} `json:"formData"`
	Status             int                    `json:"status"` // 0=草稿 1=已提交 2=已修改
	CreatedBy          string                 `json:"createdBy"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

// FormVersionHistory 表单版本历史
type FormVersionHistory struct {
	ID         int64                  `json:"id"`
	FormID     string                 `json:"formId"`
	Version    int                    `json:"version"`
	FormSchema map[string]interface{} `json:"formSchema"`
	ChangeLog  string                 `json:"changeLog"`
	CreatedBy  string                 `json:"createdBy"`
	CreatedAt  time.Time              `json:"createdAt"`
}

// FormFieldPermission 表单字段权限配置
type FormFieldPermission struct {
	FieldKey   string `json:"fieldKey"`
	NodeID     string `json:"nodeId"`
	Permission string `json:"permission"` // editable/readonly/hidden
}

// FormListParams 表单列表查询参数
type FormListParams struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Status   *int   `json:"status,omitempty"`
	Keyword  string `json:"keyword,omitempty"`
}
