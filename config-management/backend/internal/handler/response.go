package handler

import "github.com/gin-gonic/gin"

// responsePayload 构造兼容前端现有约定的响应体。
// 仅在值存在时输出 message、data、total，避免改变既有字段结构。
func responsePayload(code int, message string, data any, total any) gin.H {
	payload := gin.H{"code": code}
	if message != "" {
		payload["message"] = message
	}
	if data != nil {
		payload["data"] = data
	}
	if total != nil {
		payload["total"] = total
	}
	return payload
}

// respondJSON 统一输出 JSON 响应。
func (s *Server) respondJSON(c *gin.Context, httpStatus int, code int, message string, data any, total any) {
	c.JSON(httpStatus, responsePayload(code, message, data, total))
}

// respondError 输出仅包含错误码和错误信息的响应。
func (s *Server) respondError(c *gin.Context, httpStatus int, code int, message string) {
	s.respondJSON(c, httpStatus, code, message, nil, nil)
}
