package types

import "time"

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
	FormSchema  map[string]interface{} `json:"formSchema"` // 表单设计器JSON Schema
	Version     int                    `json:"version"`    // 版本号，从1开始自增
	Status      int                    `json:"status"`     // 0=草稿 1=已发布 2=已停用
	CreatedBy   string                 `json:"createdBy"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Deleted     bool                   `json:"deleted"`
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

