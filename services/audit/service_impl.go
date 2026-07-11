package audit

import (
	"context"
	"time"

	"weave/models"

	"gorm.io/gorm"
)

type auditServiceImpl struct {
	db *gorm.DB
}

// NewAuditService 创建审计日志服务实例
func NewAuditService(db *gorm.DB) AuditService {
	return &auditServiceImpl{db: db}
}

func (s *auditServiceImpl) GetAuditLogs(ctx context.Context, tenantID uint, filter AuditLogFilter) (*AuditLogPageResult, error) {
	query := s.db.WithContext(ctx).Model(&models.AuditLog{})
	query = query.Where("tenant_id = ?", tenantID)

	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.Username != "" {
		query = query.Where("username = ?", filter.Username)
	}
	if filter.StartTime != "" {
		if startTime, err := time.Parse(time.RFC3339, filter.StartTime); err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if filter.EndTime != "" {
		if endTime, err := time.Parse(time.RFC3339, filter.EndTime); err == nil {
			query = query.Where("created_at <= ?", endTime)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (filter.Page - 1) * filter.PageSize

	var auditLogs []models.AuditLog
	if err := query.Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Preload("User").Find(&auditLogs).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &AuditLogPageResult{
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
		Logs:       auditLogs,
	}, nil
}

func (s *auditServiceImpl) GetAuditLog(ctx context.Context, id string, tenantID uint) (*models.AuditLog, error) {
	var auditLog models.AuditLog
	result := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).Preload("User").First(&auditLog)
	if result.Error != nil {
		return nil, result.Error
	}
	return &auditLog, nil
}

func (s *auditServiceImpl) GetAuditStats(ctx context.Context, tenantID uint) (*AuditStats, error) {
	// 按操作类型统计
	var actionStats []ActionStat
	if err := s.db.WithContext(ctx).Model(&models.AuditLog{}).
		Select("action, COUNT(*) as count").
		Where("tenant_id = ?", tenantID).
		Group("action").
		Find(&actionStats).Error; err != nil {
		return nil, err
	}

	// 按资源类型统计
	var resourceStats []ResourceStat
	if err := s.db.WithContext(ctx).Model(&models.AuditLog{}).
		Select("resource_type, COUNT(*) as count").
		Where("tenant_id = ?", tenantID).
		Group("resource_type").
		Find(&resourceStats).Error; err != nil {
		return nil, err
	}

	// 最近7天的操作统计
	sevenDaysAgo := time.Now().AddDate(0, 0, -6).Truncate(24 * time.Hour)
	today := time.Now().Truncate(24 * time.Hour)

	countMap := make(map[string]int64)
	for i := 0; i < 7; i++ {
		date := sevenDaysAgo.AddDate(0, 0, i).Format("2006-01-02")
		countMap[date] = 0
	}

	var results []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	if err := s.db.WithContext(ctx).Model(&models.AuditLog{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("tenant_id = ? AND created_at >= ? AND created_at < ?", tenantID, sevenDaysAgo, today.Add(24*time.Hour)).
		Group("DATE(created_at)").
		Find(&results).Error; err != nil {
		return nil, err
	}

	for _, result := range results {
		countMap[result.Date] = result.Count
	}

	var dailyStats []DailyStat
	for i := 0; i < 7; i++ {
		date := sevenDaysAgo.AddDate(0, 0, i).Format("2006-01-02")
		dailyStats = append(dailyStats, DailyStat{
			Date:  date,
			Count: countMap[date],
		})
	}

	return &AuditStats{
		ActionStats:   actionStats,
		ResourceStats: resourceStats,
		DailyStats:    dailyStats,
	}, nil
}