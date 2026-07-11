package controllers

import (
	"net/http"
	"strconv"

	"weave/pkg"
	auditsvc "weave/services/audit"

	"github.com/gin-gonic/gin"
)

// AuditController 审计日志控制器
type AuditController struct {
	auditService auditsvc.AuditService
}

// NewAuditController 创建审计日志控制器实例
func NewAuditController(auditSvc auditsvc.AuditService) *AuditController {
	return &AuditController{auditService: auditSvc}
}

// GetAuditLogs 获取审计日志列表
func (ac *AuditController) GetAuditLogs(c *gin.Context) {
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filter := auditsvc.AuditLogFilter{
		Page:         page,
		PageSize:     pageSize,
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		Username:     c.Query("username"),
		StartTime:    c.Query("start_time"),
		EndTime:      c.Query("end_time"),
	}

	tenantID := c.GetUint("tenant_id")
	result, err := ac.auditService.GetAuditLogs(c.Request.Context(), tenantID, filter)
	if err != nil {
		dbErr := pkg.NewDatabaseError("Failed to fetch audit logs", err)
		c.JSON(pkg.GetHTTPStatus(dbErr), gin.H{"code": string(dbErr.Code), "message": dbErr.Message})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":       result.Total,
		"page":        result.Page,
		"page_size":   result.PageSize,
		"total_pages": result.TotalPages,
		"logs":        result.Logs,
	})
}

// GetAuditLog 获取单个审计日志详情
func (ac *AuditController) GetAuditLog(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	auditLog, err := ac.auditService.GetAuditLog(c.Request.Context(), id, tenantID)
	if err != nil {
		appErr := pkg.NewNotFoundError("Audit log not found", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, auditLog)
}

// GetAuditStats 获取审计日志统计信息
func (ac *AuditController) GetAuditStats(c *gin.Context) {
	tenantID := c.GetUint("tenant_id")

	stats, err := ac.auditService.GetAuditStats(c.Request.Context(), tenantID)
	if err != nil {
		dbErr := pkg.NewDatabaseError("Failed to get audit stats", err)
		c.JSON(pkg.GetHTTPStatus(dbErr), gin.H{"code": string(dbErr.Code), "message": dbErr.Message})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"action_stats":   stats.ActionStats,
		"resource_stats": stats.ResourceStats,
		"daily_stats":    stats.DailyStats,
	})
}