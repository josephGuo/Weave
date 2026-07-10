package health

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type healthServiceImpl struct {
	db *gorm.DB
}

// NewHealthService 创建健康检查服务实例
func NewHealthService(db *gorm.DB) HealthService {
	return &healthServiceImpl{db: db}
}

func (s *healthServiceImpl) CheckDatabase(ctx context.Context) DBHealthResult {
	startTime := time.Now()

	err := s.db.WithContext(ctx).Exec("SELECT 1").Error
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		return DBHealthResult{
			Healthy:      false,
			ResponseTime: duration,
			Error:        err.Error(),
		}
	}

	return DBHealthResult{
		Healthy:      true,
		ResponseTime: duration,
	}
}