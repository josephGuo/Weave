package controllers

import (
	"net/http"

	"weave/models"
	"weave/pkg"
	"weave/services/tool"

	"github.com/gin-gonic/gin"
)

// ToolController 工具控制器
type ToolController struct {
	toolService tool.ToolService
}

// NewToolController 创建工具控制器实例
func NewToolController(toolSvc tool.ToolService) *ToolController {
	return &ToolController{toolService: toolSvc}
}

// GetTools 获取所有工具
func (tc *ToolController) GetTools(c *gin.Context) {
	tenantID := c.GetUint("tenant_id")
	tools, err := tc.toolService.GetTools(c.Request.Context(), tenantID)
	if err != nil {
		dbErr := pkg.NewDatabaseError("Failed to fetch tools", err)
		c.JSON(pkg.GetHTTPStatus(dbErr), gin.H{"code": string(dbErr.Code), "message": dbErr.Message})
		return
	}
	c.JSON(http.StatusOK, tools)
}

// GetTool 获取单个工具
func (tc *ToolController) GetTool(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	tool, err := tc.toolService.GetTool(c.Request.Context(), id, tenantID)
	if err != nil {
		appErr := pkg.NewNotFoundError("Tool not found", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, tool)
}

// CreateTool 创建工具
func (tc *ToolController) CreateTool(c *gin.Context) {
	var tool models.Tool
	if err := c.ShouldBindJSON(&tool); err != nil {
		appErr := pkg.NewValidationError("Invalid tool data", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	tool.TenantID = c.GetUint("tenant_id")

	if err := tc.toolService.CreateTool(c.Request.Context(), &tool); err != nil {
		dbErr := pkg.NewDatabaseError("Failed to create tool", err)
		c.JSON(pkg.GetHTTPStatus(dbErr), gin.H{"code": string(dbErr.Code), "message": dbErr.Message})
		return
	}

	c.JSON(http.StatusCreated, tool)
}

// UpdateTool 更新工具
func (tc *ToolController) UpdateTool(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	var newTool models.Tool
	if err := c.ShouldBindJSON(&newTool); err != nil {
		appErr := pkg.NewValidationError("Invalid tool data", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	updated, err := tc.toolService.UpdateTool(c.Request.Context(), id, tenantID, &newTool)
	if err != nil {
		appErr := pkg.NewNotFoundError("Tool not found", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteTool 删除工具
func (tc *ToolController) DeleteTool(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	if err := tc.toolService.DeleteTool(c.Request.Context(), id, tenantID); err != nil {
		appErr := pkg.NewNotFoundError("Tool not found", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tool deleted successfully"})
}

// ExecuteTool 执行工具
func (tc *ToolController) ExecuteTool(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	tool, err := tc.toolService.GetTool(c.Request.Context(), id, tenantID)
	if err != nil {
		appErr := pkg.NewNotFoundError("Tool not found", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	// 执行逻辑保持不变
	c.JSON(http.StatusOK, gin.H{"message": "Tool execution started", "tool": tool})
}