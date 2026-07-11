package tool

import (
	"context"

	"weave/models"

	"gorm.io/gorm"
)

type toolServiceImpl struct {
	db *gorm.DB
}

// NewToolService 创建工具服务实例
func NewToolService(db *gorm.DB) ToolService {
	return &toolServiceImpl{db: db}
}

func (s *toolServiceImpl) GetTools(ctx context.Context, tenantID uint) ([]models.Tool, error) {
	var tools []models.Tool
	result := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&tools)
	if result.Error != nil {
		return nil, result.Error
	}
	return tools, nil
}

func (s *toolServiceImpl) GetTool(ctx context.Context, id string, tenantID uint) (*models.Tool, error) {
	var tool models.Tool
	result := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&tool)
	if result.Error != nil {
		return nil, result.Error
	}
	return &tool, nil
}

func (s *toolServiceImpl) CreateTool(ctx context.Context, tool *models.Tool) error {
	return s.db.WithContext(ctx).Create(tool).Error
}

func (s *toolServiceImpl) UpdateTool(ctx context.Context, id string, tenantID uint, tool *models.Tool) (*models.Tool, error) {
	var oldTool models.Tool
	result := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&oldTool)
	if result.Error != nil {
		return nil, result.Error
	}

	tool.ID = oldTool.ID
	tool.TenantID = tenantID

	if err := s.db.WithContext(ctx).Save(tool).Error; err != nil {
		return nil, err
	}
	return tool, nil
}

func (s *toolServiceImpl) DeleteTool(ctx context.Context, id string, tenantID uint) error {
	var tool models.Tool
	result := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&tool)
	if result.Error != nil {
		return result.Error
	}
	return s.db.WithContext(ctx).Delete(&tool).Error
}