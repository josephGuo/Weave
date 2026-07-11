package tool

import (
	"context"

	"weave/models"
)

// ToolService 工具服务接口
type ToolService interface {
	GetTools(ctx context.Context, tenantID uint) ([]models.Tool, error)
	GetTool(ctx context.Context, id string, tenantID uint) (*models.Tool, error)
	CreateTool(ctx context.Context, tool *models.Tool) error
	UpdateTool(ctx context.Context, id string, tenantID uint, tool *models.Tool) (*models.Tool, error)
	DeleteTool(ctx context.Context, id string, tenantID uint) error
}