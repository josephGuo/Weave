package controllers

import (
	"fmt"
	"net/http"
	"time"

	"weave/models"
	"weave/pkg"
	usersvc "weave/services/user"
	"weave/utils"

	"github.com/gin-gonic/gin"
)

// UserController 用户控制器
type UserController struct {
	userService usersvc.UserService
}

// NewUserController 创建用户控制器实例
func NewUserController(userSvc usersvc.UserService) *UserController {
	return &UserController{
		userService: userSvc,
	}
}

// Register 用户注册
func (uc *UserController) Register(c *gin.Context) {
	var registerRequest usersvc.RegisterRequest
	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		err := pkg.NewValidationError("Invalid registration data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	if registerRequest.Password != registerRequest.ConfirmPassword {
		err := pkg.NewValidationError("Passwords do not match", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	newUser, err := uc.userService.Register(c.Request.Context(), registerRequest)
	if err != nil {
		appErr := pkg.NewConflictError(err.Error(), nil)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "注册成功", "user": newUser})
}

// SendVerificationCodeRequest 发送验证码请求结构
type SendVerificationCodeRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"omitempty,email"`
}

// LoginWithCodeRequest 验证码登录请求结构
type LoginWithCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

// SendVerificationCode 发送邮箱验证码
func (uc *UserController) SendVerificationCode(c *gin.Context) {
	var req SendVerificationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		err := pkg.NewValidationError("请输入有效的用户名", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	tenantID := c.GetUint("tenant_id")

	user, err := uc.userService.SendVerificationCode(c.Request.Context(), req.Username, tenantID)
	if err != nil {
		appErr := pkg.NewValidationError(err.Error(), nil)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "send_verification_code",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", user.ID),
		OldValue:     nil,
		NewValue: map[string]interface{}{
			"username": req.Username,
			"email":    user.Email,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送到您的邮箱，请查收"})
}

// LoginWithVerificationCode 使用邮箱验证码登录
func (uc *UserController) LoginWithVerificationCode(c *gin.Context) {
	var req LoginWithCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.userService.RecordLoginHistory(c.Request.Context(), req.Email, c.ClientIP(), c.Request.UserAgent(), "请求参数验证失败: "+err.Error(), false, 0)
		err := pkg.NewValidationError("Invalid request data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	tenantID := c.GetUint("tenant_id")

	user, err := uc.userService.LoginWithCode(c.Request.Context(), req.Email, req.Code, tenantID)
	if err != nil {
		uc.userService.RecordLoginHistory(c.Request.Context(), req.Email, c.ClientIP(), c.Request.UserAgent(), "验证码验证失败: "+err.Error(), false, tenantID)
		appErr := pkg.NewAuthError("验证码错误或已过期", nil)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	accessToken, err := utils.GenerateToken(user.ID, user.TenantID)
	if err != nil {
		uc.userService.RecordLoginHistory(c.Request.Context(), req.Email, c.ClientIP(), c.Request.UserAgent(), "生成访问令牌失败: "+err.Error(), false, user.TenantID)
		err := pkg.NewInternalError("Failed to generate access token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.TenantID)
	if err != nil {
		uc.userService.RecordLoginHistory(c.Request.Context(), req.Email, c.ClientIP(), c.Request.UserAgent(), "生成刷新令牌失败: "+err.Error(), false, user.TenantID)
		err := pkg.NewInternalError("Failed to generate refresh token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	uc.userService.RecordLoginHistory(c.Request.Context(), req.Email, c.ClientIP(), c.Request.UserAgent(), "邮箱验证码登录成功", true, user.TenantID)

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "login_with_code",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", user.ID),
		OldValue:     nil,
		NewValue: map[string]interface{}{
			"email":      user.Email,
			"ip_address": c.ClientIP(),
			"success":    true,
		},
	})

	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"message": "登录成功", "access_token": accessToken, "refresh_token": refreshToken, "user": user})
}

// Login 用户登录
func (uc *UserController) Login(c *gin.Context) {
	var loginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required,len=6"`
	}

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		uc.userService.RecordLoginHistory(c.Request.Context(), loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), "请求参数验证失败: "+err.Error(), false, 0)
		err := pkg.NewValidationError("请输入用户名、密码和验证码", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	tenantID := c.GetUint("tenant_id")

	user, err := uc.userService.Login(c.Request.Context(), tenantID, usersvc.LoginRequest{
		Username: loginRequest.Username,
		Password: loginRequest.Password,
		Code:     loginRequest.Code,
	})
	if err != nil {
		uc.userService.RecordLoginHistory(c.Request.Context(), loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), err.Error(), false, tenantID)
		appErr := pkg.NewAuthError(err.Error(), nil)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	accessToken, err := utils.GenerateToken(user.ID, user.TenantID)
	if err != nil {
		uc.userService.RecordLoginHistory(c.Request.Context(), loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), "生成访问令牌失败: "+err.Error(), false, user.TenantID)
		err := pkg.NewInternalError("Failed to generate access token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.TenantID)
	if err != nil {
		uc.userService.RecordLoginHistory(c.Request.Context(), loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), "生成刷新令牌失败: "+err.Error(), false, user.TenantID)
		err := pkg.NewInternalError("Failed to generate refresh token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	uc.userService.RecordLoginHistory(c.Request.Context(), loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), "登录成功", true, user.TenantID)

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "login_multi_factor",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", user.ID),
		OldValue:     nil,
		NewValue: map[string]interface{}{
			"username":   user.Username,
			"email":      user.Email,
			"ip_address": c.ClientIP(),
			"success":    true,
		},
	})

	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"message": "登录成功", "access_token": accessToken, "refresh_token": refreshToken, "user": user})
}

// RefreshToken 刷新访问令牌
func (uc *UserController) RefreshToken(c *gin.Context) {
	var refreshRequest struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&refreshRequest); err != nil {
		err := pkg.NewValidationError("Refresh token is required", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	userID, tenantID, err := utils.VerifyRefreshToken(refreshRequest.RefreshToken)
	if err != nil {
		err := pkg.NewAuthError("Invalid refresh token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	user, err := uc.userService.RefreshToken(c.Request.Context(), refreshRequest.RefreshToken)
	if err != nil {
		appErr := pkg.NewNotFoundError("User not found", nil)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	accessToken, err := utils.GenerateToken(userID, tenantID)
	if err != nil {
		err := pkg.NewInternalError("Failed to generate access token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	newRefreshToken, err := utils.GenerateRefreshToken(userID, tenantID)
	if err != nil {
		err := pkg.NewInternalError("Failed to generate refresh token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"message": "令牌刷新成功", "access_token": accessToken, "refresh_token": newRefreshToken, "user": user})
}

// GetUsers 获取所有用户
func (uc *UserController) GetUsers(c *gin.Context) {
	tenantID := c.GetUint("tenant_id")

	users, err := uc.userService.GetUsers(c.Request.Context(), tenantID)
	if err != nil {
		appErr := pkg.NewDatabaseError("Failed to fetch users", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser 获取单个用户
func (uc *UserController) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		err := pkg.NewValidationError("Invalid user ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	user, err := uc.userService.GetUser(c.Request.Context(), id, tenantID)
	if err != nil {
		appErr := pkg.NewNotFoundError("User not found", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser 创建用户
func (uc *UserController) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		err := pkg.NewValidationError("Invalid user data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	user.TenantID = c.GetUint("tenant_id")

	logUser := user
	logUser.Password = "[REDACTED]"

	if err := uc.userService.CreateUser(c.Request.Context(), &user); err != nil {
		appErr := pkg.NewDatabaseError("Failed to create user", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "create",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", user.ID),
		OldValue:     nil,
		NewValue:     logUser,
	})

	user.Password = ""
	c.JSON(http.StatusCreated, user)
}

// UpdateUser 更新用户
func (uc *UserController) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		err := pkg.NewValidationError("Invalid user ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	var newUser models.User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		err := pkg.NewValidationError("Invalid user data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	updated, err := uc.userService.UpdateUser(c.Request.Context(), id, tenantID, &newUser)
	if err != nil {
		appErr := pkg.NewNotFoundError("User not found", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	auditNewUser := *updated
	auditNewUser.Password = "[REDACTED]"
	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "update",
		ResourceType: "user",
		ResourceID:   idStr,
		NewValue:     auditNewUser,
	})

	updated.Password = ""
	c.JSON(http.StatusOK, updated)
}

// DeleteUser 删除用户
func (uc *UserController) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		err := pkg.NewValidationError("Invalid user ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	deleted, err := uc.userService.DeleteUser(c.Request.Context(), id, tenantID)
	if err != nil {
		appErr := pkg.NewNotFoundError("User not found", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	auditUser := *deleted
	auditUser.Password = "[REDACTED]"
	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "delete",
		ResourceType: "user",
		ResourceID:   idStr,
		OldValue:     auditUser,
		NewValue:     nil,
	})

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// ChangePasswordRequest 修改密码请求结构
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword 修改用户密码
func (uc *UserController) ChangePassword(c *gin.Context) {
	currentUserID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		err := pkg.NewValidationError("Invalid request data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	if err := uc.userService.ChangePassword(c.Request.Context(), currentUserID, tenantID, req.CurrentPassword, req.NewPassword); err != nil {
		appErr := pkg.NewValidationError(err.Error(), nil)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "change_password",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", currentUserID),
		OldValue:     map[string]interface{}{"user_id": currentUserID},
		NewValue:     map[string]interface{}{"user_id": currentUserID, "password_changed": true},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// GetUserID 获取当前用户ID (辅助方法)
func (uc *UserController) GetUserID(c *gin.Context) uint {
	return c.GetUint("user_id")
}

// GetTenantID 获取当前租户ID (辅助方法)
func (uc *UserController) GetTenantID(c *gin.Context) uint {
	return c.GetUint("tenant_id")
}

// Ensure time is used (imported for future use)
var _ = time.Now