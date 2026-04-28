package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// SetD6ApprovalService 注入高级审批服务，并注册 D6 审批路由
func (s *Server) SetD6ApprovalService(advApprovalSvc interfaces.AdvancedApprovalService) {
	s.advApprovalSvc = advApprovalSvc
	s.registerD6ApprovalRoutes(s.engine)
}

func (s *Server) registerD6ApprovalRoutes(r *gin.Engine) {
	approval := r.Group("/api/approval")
	{
		approval.POST("/add-sign", s.addSign)
		approval.POST("/transfer", s.transferApproval)
		approval.POST("/return", s.returnApproval)
		approval.POST("/withdraw", s.withdrawApproval)
		approval.POST("/delegate", s.setDelegate)
		approval.GET("/delegates/:userId", s.getDelegates)
		approval.POST("/cc", s.sendCC)
		approval.GET("/cc/list", s.getCCList)
		approval.PUT("/cc/:ccId/read", s.markCCRead)
	}
}

// ─── 高级审批处理函数 ─────────────────────────────────────────────────────────

func (s *Server) addSign(c *gin.Context) {
	if s.advApprovalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "高级审批服务不可用"})
		return
	}
	var req types.AddSignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.advApprovalSvc.AddSign(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "加签成功"})
}

func (s *Server) transferApproval(c *gin.Context) {
	if s.advApprovalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "高级审批服务不可用"})
		return
	}
	var req types.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.advApprovalSvc.Transfer(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "转签成功"})
}

func (s *Server) returnApproval(c *gin.Context) {
	if s.advApprovalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "高级审批服务不可用"})
		return
	}
	var req types.ReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.advApprovalSvc.Return(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "退回成功"})
}

func (s *Server) withdrawApproval(c *gin.Context) {
	if s.advApprovalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "高级审批服务不可用"})
		return
	}
	var req types.WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.advApprovalSvc.Withdraw(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "撤回成功"})
}

func (s *Server) setDelegate(c *gin.Context) {
	if s.advApprovalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "高级审批服务不可用"})
		return
	}
	var req types.DelegateConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.advApprovalSvc.SetDelegate(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "委托设置成功"})
}

func (s *Server) getDelegates(c *gin.Context) {
	if s.advApprovalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "高级审批服务不可用"})
		return
	}
	userID := c.Param("userId")
	delegates, err := s.advApprovalSvc.GetDelegates(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": delegates})
}

func (s *Server) sendCC(c *gin.Context) {
	if s.advApprovalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "高级审批服务不可用"})
		return
	}
	var req types.CCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.advApprovalSvc.SendCC(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "抄送成功"})
}

type getCCListQuery struct {
	UserID   string `form:"userId" binding:"required"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

func (s *Server) getCCList(c *gin.Context) {
	if s.advApprovalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "高级审批服务不可用"})
		return
	}
	var q getCCListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	list, total, err := s.advApprovalSvc.GetCCList(c.Request.Context(), q.UserID, q.Page, q.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": list, "total": total})
}

func (s *Server) markCCRead(c *gin.Context) {
	if s.advApprovalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "高级审批服务不可用"})
		return
	}
	ccIDStr := c.Param("ccId")
	ccID, err := strconv.ParseInt(ccIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的 ccId 参数"})
		return
	}
	if err := s.advApprovalSvc.MarkCCRead(c.Request.Context(), ccID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "标记已读成功"})
}
