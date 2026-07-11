package user

import (
	"context"

	"weave/models"
)

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
	Email           string `json:"email" binding:"required"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

// LoginResult 登录结果
type LoginResult struct {
	User         models.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
}

// UserService 用户服务接口
type UserService interface {
	Register(ctx context.Context, req RegisterRequest) (*models.User, error)
	Login(ctx context.Context, tenantID uint, req LoginRequest) (*models.User, error)
	LoginWithCode(ctx context.Context, email, code string, tenantID uint) (*models.User, error)
	SendVerificationCode(ctx context.Context, username string, tenantID uint) (*models.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*models.User, error)
	GetUsers(ctx context.Context, tenantID uint) ([]models.User, error)
	GetUser(ctx context.Context, id, tenantID uint) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, id, tenantID uint, user *models.User) (*models.User, error)
	DeleteUser(ctx context.Context, id, tenantID uint) (*models.User, error)
	ChangePassword(ctx context.Context, userID, tenantID uint, currentPassword, newPassword string) error
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	RecordLoginHistory(ctx context.Context, username, ipAddress, userAgent, message string, success bool, tenantID uint)
}