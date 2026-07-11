package audit

import (
	"context"

	"weave/models"
)

// AuditLogFilter 审计日志查询过滤条件
type AuditLogFilter struct {
	Page        int
	PageSize    int
	Action      string
	ResourceType string
	Username    string
	StartTime   string
	EndTime     string
}

// AuditLogPageResult 审计日志分页结果
type AuditLogPageResult struct {
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
	Logs       []models.AuditLog  `json:"logs"`
}

// ActionStat 按操作类型统计
type ActionStat struct {
	Action string `json:"action"`
	Count  int64  `json:"count"`
}

// ResourceStat 按资源类型统计
type ResourceStat struct {
	ResourceType string `json:"resource_type"`
	Count        int64  `json:"count"`
}

// DailyStat 每日统计
type DailyStat struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// AuditStats 审计日志统计信息
type AuditStats struct {
	ActionStats   []ActionStat   `json:"action_stats"`
	ResourceStats []ResourceStat `json:"resource_stats"`
	DailyStats    []DailyStat    `json:"daily_stats"`
}

// AuditService 审计日志服务接口
type AuditService interface {
	GetAuditLogs(ctx context.Context, tenantID uint, filter AuditLogFilter) (*AuditLogPageResult, error)
	GetAuditLog(ctx context.Context, id string, tenantID uint) (*models.AuditLog, error)
	GetAuditStats(ctx context.Context, tenantID uint) (*AuditStats, error)
}