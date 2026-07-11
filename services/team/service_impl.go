package team

import (
	"context"
	"strconv"

	"weave/models"

	"gorm.io/gorm"
)

type teamServiceImpl struct {
	db *gorm.DB
}

// NewTeamService 创建团队服务实例
func NewTeamService(db *gorm.DB) TeamService {
	return &teamServiceImpl{db: db}
}

func (s *teamServiceImpl) GetTeams(ctx context.Context, userID, tenantID uint) ([]models.Team, error) {
	var teamMembers []models.TeamMember
	if err := s.db.WithContext(ctx).Where("user_id = ? AND tenant_id = ?", userID, tenantID).Find(&teamMembers).Error; err != nil {
		return nil, err
	}

	var teamIDs []uint
	for _, member := range teamMembers {
		teamIDs = append(teamIDs, member.TeamID)
	}

	var teams []models.Team
	if len(teamIDs) > 0 {
		if err := s.db.WithContext(ctx).Where("id IN ? AND tenant_id = ?", teamIDs, tenantID).Find(&teams).Error; err != nil {
			return nil, err
		}
	}

	return teams, nil
}

func (s *teamServiceImpl) CreateTeam(ctx context.Context, name, description string, ownerID, tenantID uint) (*models.Team, error) {
	team := models.Team{
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		TenantID:    tenantID,
	}
	if err := s.db.WithContext(ctx).Create(&team).Error; err != nil {
		return nil, err
	}

	// 将创建者加入团队成员，角色为owner
	_ = s.db.WithContext(ctx).Create(&models.TeamMember{TeamID: team.ID, UserID: ownerID, Role: "owner", TenantID: tenantID}).Error

	// 更新团队成员列表字段
	s.updateTeamMembers(team.ID)

	return &team, nil
}

func (s *teamServiceImpl) UpdateTeam(ctx context.Context, teamID uint, name, description string, userID, tenantID uint) (*models.Team, error) {
	var team models.Team
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", teamID, tenantID).First(&team).Error; err != nil {
		return nil, err
	}

	// 检查权限：只有团队所有者可以更新团队信息
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ? AND role = 'owner'", team.ID, userID).First(&models.TeamMember{}).Error; err != nil {
		return nil, err
	}

	// 如果要更新名称，检查名称是否已存在
	if name != "" && name != team.Name {
		var existingTeam models.Team
		if err := s.db.WithContext(ctx).Where("name = ? AND tenant_id = ? AND id != ?", name, tenantID, teamID).First(&existingTeam).Error; err == nil {
			return nil, gorm.ErrDuplicatedKey
		}
		team.Name = name
	}

	if description != team.Description {
		team.Description = description
	}

	if err := s.db.WithContext(ctx).Save(&team).Error; err != nil {
		return nil, err
	}

	return &team, nil
}

func (s *teamServiceImpl) GetTeamMembers(ctx context.Context, teamID, userID, tenantID uint) ([]models.TeamMember, error) {
	var team models.Team
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", teamID, tenantID).First(&team).Error; err != nil {
		return nil, err
	}

	// 检查用户是否为团队成员
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ?", teamID, userID).First(&models.TeamMember{}).Error; err != nil {
		return nil, err
	}

	var members []models.TeamMember
	if err := s.db.WithContext(ctx).Where("team_id = ? AND tenant_id = ?", teamID, tenantID).Find(&members).Error; err != nil {
		return nil, err
	}

	return members, nil
}

func (s *teamServiceImpl) AddTeamMember(ctx context.Context, teamID, newMemberUserID uint, role string, requesterID, tenantID uint) (*models.TeamMember, error) {
	var team models.Team
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", teamID, tenantID).First(&team).Error; err != nil {
		return nil, err
	}

	// 检查权限：只有团队所有者或管理员可以添加成员
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ? AND role IN ('owner', 'admin')", teamID, requesterID).First(&models.TeamMember{}).Error; err != nil {
		return nil, err
	}

	// 检查用户是否已存在
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ?", teamID, newMemberUserID).First(&models.TeamMember{}).Error; err == nil {
		return nil, gorm.ErrDuplicatedKey
	}

	newMember := models.TeamMember{
		TeamID:   teamID,
		UserID:   newMemberUserID,
		Role:     role,
		TenantID: tenantID,
	}

	if err := s.db.WithContext(ctx).Create(&newMember).Error; err != nil {
		return nil, err
	}

	s.updateTeamMembers(teamID)

	return &newMember, nil
}

func (s *teamServiceImpl) RemoveTeamMember(ctx context.Context, teamID, memberUserID uint, requesterID, tenantID uint) error {
	var team models.Team
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", teamID, tenantID).First(&team).Error; err != nil {
		return err
	}

	// 查找要移除的成员
	var teamMember models.TeamMember
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ? AND tenant_id = ?", teamID, memberUserID, tenantID).First(&teamMember).Error; err != nil {
		return err
	}

	// 不能移除所有者
	if teamMember.Role == "owner" {
		return gorm.ErrInvalidData
	}

	// 检查权限：只有团队所有者或管理员可以移除成员
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ? AND role IN ('owner', 'admin')", teamID, requesterID).First(&models.TeamMember{}).Error; err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).Delete(&teamMember).Error; err != nil {
		return err
	}

	s.updateTeamMembers(teamID)

	return nil
}

func (s *teamServiceImpl) SearchTeamMembers(ctx context.Context, teamID, userID, tenantID uint, keyword string) ([]MemberWithInfo, error) {
	var team models.Team
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", teamID, tenantID).First(&team).Error; err != nil {
		return nil, err
	}

	// 检查用户是否为团队成员
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ?", teamID, userID).First(&models.TeamMember{}).Error; err != nil {
		return nil, err
	}

	query := s.db.WithContext(ctx).Table("team_member tm").
		Select("tm.*, u.username, u.email").
		Joins("JOIN user u ON tm.user_id = u.id").
		Where("tm.team_id = ? AND tm.tenant_id = ?", teamID, tenantID)

	if searchID, parseErr := strconv.ParseUint(keyword, 10, 32); parseErr == nil {
		query = query.Where("tm.user_id = ?", searchID)
	} else {
		query = query.Where("u.username LIKE ?", "%"+keyword+"%")
	}

	var members []MemberWithInfo
	if err := query.Find(&members).Error; err != nil {
		return nil, err
	}

	return members, nil
}

func (s *teamServiceImpl) UpdateMemberRole(ctx context.Context, teamID, memberUserID uint, newRole string, requesterID, tenantID uint) (*models.TeamMember, error) {
	var team models.Team
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", teamID, tenantID).First(&team).Error; err != nil {
		return nil, err
	}

	var teamMember models.TeamMember
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ? AND tenant_id = ?", teamID, memberUserID, tenantID).First(&teamMember).Error; err != nil {
		return nil, err
	}

	// 检查权限：只有团队所有者可以更新成员角色
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ? AND role = 'owner'", teamID, requesterID).First(&models.TeamMember{}).Error; err != nil {
		return nil, err
	}

	teamMember.Role = newRole
	if err := s.db.WithContext(ctx).Save(&teamMember).Error; err != nil {
		return nil, err
	}

	s.updateTeamMembers(teamID)

	return &teamMember, nil
}

func (s *teamServiceImpl) TransferTeamOwner(ctx context.Context, teamID, newOwnerID, currentOwnerID, tenantID uint) (*TransferResult, error) {
	var team models.Team
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", teamID, tenantID).First(&team).Error; err != nil {
		return nil, err
	}

	// 检查权限：只有团队所有者可以转让所有权
	var currentMember models.TeamMember
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ? AND role = 'owner'", teamID, currentOwnerID).First(&currentMember).Error; err != nil {
		return nil, err
	}

	// 检查新所有者是否为团队成员
	var newOwnerMember models.TeamMember
	if err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ? AND tenant_id = ?", teamID, newOwnerID, tenantID).First(&newOwnerMember).Error; err != nil {
		return nil, err
	}

	// 事务处理
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	currentMember.Role = "admin"
	if err := tx.Save(&currentMember).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	newOwnerMember.Role = "owner"
	if err := tx.Save(&newOwnerMember).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	team.OwnerID = newOwnerID
	if err := tx.Save(&team).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	s.updateTeamMembers(teamID)

	return &TransferResult{
		Team:     team,
		NewOwner: newOwnerMember,
		OldOwner: currentMember,
	}, nil
}

func (s *teamServiceImpl) IsMember(ctx context.Context, teamID, userID uint) bool {
	var member models.TeamMember
	err := s.db.WithContext(ctx).Where("team_id = ? AND user_id = ?", teamID, userID).First(&member).Error
	return err == nil
}

// updateTeamMembers 更新团队成员列表字段
func (s *teamServiceImpl) updateTeamMembers(teamID uint) {
	var usernames []string
	if err := s.db.Table("team_member tm").
		Select("u.username").
		Joins("JOIN user u ON tm.user_id = u.id").
		Where("tm.team_id = ?", teamID).
		Pluck("u.username", &usernames).Error; err != nil {
		return
	}

	membersStr := ""
	for i, username := range usernames {
		if i > 0 {
			membersStr += ","
		}
		membersStr += username
	}

	if membersStr == "" {
		s.db.Model(&models.Team{}).Where("id = ?", teamID).Update("members", nil)
	} else {
		s.db.Model(&models.Team{}).Where("id = ?", teamID).Update("members", membersStr)
	}
}