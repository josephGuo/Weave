package controllers_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"weave/controllers"
	"weave/models"
	"weave/pkg"
	"weave/services/audit"
	"weave/services/health"
	"weave/services/team"
	"weave/services/tool"
	"weave/services/user"
)

// setupTestDB 创建并配置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("gorm open error: %v", err)
	}

	// 使用models包中定义的标准表迁移顺序
	if err := models.MigrateTables(db); err != nil {
		t.Fatalf("migrate tables error: %v", err)
	}

	// 设置全局DB实例
	pkg.DB = db

	return db
}

// newTestUserController 创建测试用用户控制器
func newTestUserController(db *gorm.DB) *controllers.UserController {
	userSvc := user.NewUserService(db, user.EmailConfig{})
	return controllers.NewUserController(userSvc)
}

// newTestTeamController 创建测试用团队控制器
func newTestTeamController(db *gorm.DB) *controllers.TeamController {
	teamSvc := team.NewTeamService(db)
	return controllers.NewTeamController(teamSvc)
}

// newTestAuditController 创建测试用审计控制器
func newTestAuditController(db *gorm.DB) *controllers.AuditController {
	auditSvc := audit.NewAuditService(db)
	return controllers.NewAuditController(auditSvc)
}

// newTestToolController 创建测试用工具控制器
func newTestToolController(db *gorm.DB) *controllers.ToolController {
	toolSvc := tool.NewToolService(db)
	return controllers.NewToolController(toolSvc)
}

// newTestHealthController 创建测试用健康检查控制器
func newTestHealthController(db *gorm.DB) *controllers.HealthController {
	healthSvc := health.NewHealthService(db)
	return controllers.NewHealthController(healthSvc)
}