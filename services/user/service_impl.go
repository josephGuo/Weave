package user

import (
	"context"
	"fmt"
	"time"

	"weave/models"
	"weave/utils"

	"gorm.io/gorm"
)

type userServiceImpl struct {
	db      *gorm.DB
	emailer *emailer
}

// NewUserService 创建用户服务实例
func NewUserService(db *gorm.DB, emailCfg EmailConfig) UserService {
	return &userServiceImpl{db: db, emailer: newEmailer(emailCfg)}
}

func (s *userServiceImpl) Register(ctx context.Context, req RegisterRequest) (*models.User, error) {
	// 检查用户名是否已存在
	var existingUser models.User
	if err := s.db.WithContext(ctx).Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("用户名已存在")
	}

	// 检查邮箱是否已存在
	if err := s.db.WithContext(ctx).Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("邮箱已注册")
	}

	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	newUser := models.User{
		Username: req.Username,
		Password: passwordHash,
		Email:    req.Email,
	}

	if err := s.db.WithContext(ctx).Create(&newUser).Error; err != nil {
		return nil, err
	}

	newUser.Password = ""
	return &newUser, nil
}

func (s *userServiceImpl) Login(ctx context.Context, tenantID uint, req LoginRequest) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("username = ? AND tenant_id = ?", req.Username, tenantID).First(&user).Error; err != nil {
		return nil, err
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	// 验证邮箱验证码
	isValid, err := s.verifyEmailCode(user.Email, req.Code, tenantID)
	if err != nil || !isValid {
		return nil, fmt.Errorf("验证码错误或已过期")
	}

	return &user, nil
}

func (s *userServiceImpl) LoginWithCode(ctx context.Context, email, code string, tenantID uint) (*models.User, error) {
	isValid, err := s.verifyEmailCode(email, code, tenantID)
	if err != nil || !isValid {
		return nil, fmt.Errorf("验证码错误或已过期")
	}

	var user models.User
	if err := s.db.WithContext(ctx).Where("email = ? AND tenant_id = ?", email, tenantID).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *userServiceImpl) SendVerificationCode(ctx context.Context, username string, tenantID uint) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("username = ? AND tenant_id = ?", username, tenantID).First(&user).Error; err != nil {
		return nil, err
	}

	userEmail := user.Email

	canSend, err := s.checkVerificationRateLimit(userEmail, tenantID)
	if err != nil {
		return nil, err
	}
	if !canSend {
		return nil, fmt.Errorf("验证码发送过于频繁，请稍后再试")
	}

	originalCode, verificationCode, err := s.createVerificationCodeRecord(userEmail, tenantID)
	if err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Create(verificationCode).Error; err != nil {
		return nil, err
	}

	// 异步发送邮件
	go func() {
		if err := s.emailer.sendVerificationCode(userEmail, originalCode); err != nil {
			fmt.Printf("Failed to send verification email to %s: %v\n", userEmail, err)
		}
	}()

	return &user, nil
}

func (s *userServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*models.User, error) {
	userID, _, err := utils.VerifyRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *userServiceImpl) GetUsers(ctx context.Context, tenantID uint) ([]models.User, error) {
	var users []models.User
	if err := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *userServiceImpl) GetUser(ctx context.Context, id, tenantID uint) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).
		Preload("AuditLogs", func(db *gorm.DB) *gorm.DB {
			return db.Where("created_at > ?", time.Now().AddDate(0, 0, -30)).Order("created_at DESC").Limit(100)
		}).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *userServiceImpl) CreateUser(ctx context.Context, user *models.User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *userServiceImpl) UpdateUser(ctx context.Context, id, tenantID uint, user *models.User) (*models.User, error) {
	var oldUser models.User
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&oldUser).Error; err != nil {
		return nil, err
	}

	user.ID = oldUser.ID
	user.TenantID = tenantID

	if user.Password == "" {
		user.Password = oldUser.Password
	}

	if err := s.db.WithContext(ctx).Save(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userServiceImpl) DeleteUser(ctx context.Context, id, tenantID uint) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&user).Error; err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Delete(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *userServiceImpl) ChangePassword(ctx context.Context, userID, tenantID uint, currentPassword, newPassword string) error {
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err != nil {
		return err
	}

	if !utils.CheckPasswordHash(currentPassword, user.Password) {
		return fmt.Errorf("当前密码不正确")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return s.db.WithContext(ctx).Save(&user).Error
}

func (s *userServiceImpl) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *userServiceImpl) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *userServiceImpl) RecordLoginHistory(ctx context.Context, username, ipAddress, userAgent, message string, success bool, tenantID uint) {
	loginHistory := models.LoginHistory{
		Username:  username,
		IPAddress: ipAddress,
		Success:   success,
		Message:   message,
		UserAgent: userAgent,
		TenantID:  tenantID,
		LoginTime: time.Now(),
	}

	go func() {
		if err := s.db.Create(&loginHistory).Error; err != nil {
			fmt.Printf("Failed to record login history: %v\n", err)
		}
	}()
}

// ----- 验证码内部方法 -----

// createVerificationCodeRecord 创建并保存验证码记录
func (s *userServiceImpl) createVerificationCodeRecord(email string, tenantID uint) (string, *models.EmailVerificationCode, error) {
	code, err := s.emailer.generateVerificationCode()
	if err != nil {
		return "", nil, err
	}

	hashedCode, err := utils.HashPassword(code)
	if err != nil {
		return "", nil, err
	}

	verificationCode := &models.EmailVerificationCode{
		Email:     email,
		Code:      hashedCode,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Used:      false,
		TenantID:  tenantID,
	}

	return code, verificationCode, nil
}

// verifyEmailCode 验证邮箱验证码
func (s *userServiceImpl) verifyEmailCode(email, code string, tenantID uint) (bool, error) {
	var verificationCode models.EmailVerificationCode
	result := s.db.Where("email = ? AND used = false AND expires_at > ? AND tenant_id = ?",
		email, time.Now(), tenantID).Order("created_at DESC").First(&verificationCode)

	if result.Error != nil {
		return false, result.Error
	}

	if !utils.CheckPasswordHash(code, verificationCode.Code) {
		return false, fmt.Errorf("invalid verification code")
	}

	verificationCode.Used = true
	if err := s.db.Save(&verificationCode).Error; err != nil {
		return false, err
	}

	return true, nil
}

// getLastVerificationTime 获取用户最近一次获取验证码的时间
func (s *userServiceImpl) getLastVerificationTime(email string, tenantID uint) (time.Time, error) {
	var verificationCode models.EmailVerificationCode
	result := s.db.Where("email = ? AND tenant_id = ?", email, tenantID).Order("created_at DESC").First(&verificationCode)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return time.Time{}, nil
		}
		return time.Time{}, result.Error
	}

	return verificationCode.CreatedAt, nil
}

// checkVerificationRateLimit 检查获取验证码的频率限制
func (s *userServiceImpl) checkVerificationRateLimit(email string, tenantID uint) (bool, error) {
	lastTime, err := s.getLastVerificationTime(email, tenantID)
	if err != nil {
		return false, err
	}

	if !lastTime.IsZero() && time.Since(lastTime) < 60*time.Second {
		return false, nil
	}

	var count int64
	s.db.Model(&models.EmailVerificationCode{}).
		Where("email = ? AND tenant_id = ? AND created_at > ?",
			email, tenantID, time.Now().Add(-24*time.Hour)).
		Count(&count)

	if count >= 15 {
		return false, nil
	}

	return true, nil
}
