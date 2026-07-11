package team

import (
	"context"

	"weave/models"
)

// TeamService 团队服务接口
type TeamService interface {
	GetTeams(ctx context.Context, userID, tenantID uint) ([]models.Team, error)
	CreateTeam(ctx context.Context, name, description string, ownerID, tenantID uint) (*models.Team, error)
	UpdateTeam(ctx context.Context, teamID uint, name, description string, userID, tenantID uint) (*models.Team, error)
	GetTeamMembers(ctx context.Context, teamID, userID, tenantID uint) ([]models.TeamMember, error)
	AddTeamMember(ctx context.Context, teamID, newMemberUserID uint, role string, requesterID, tenantID uint) (*models.TeamMember, error)
	RemoveTeamMember(ctx context.Context, teamID, memberUserID uint, requesterID, tenantID uint) error
	SearchTeamMembers(ctx context.Context, teamID, userID, tenantID uint, keyword string) ([]MemberWithInfo, error)
	UpdateMemberRole(ctx context.Context, teamID, memberUserID uint, newRole string, requesterID, tenantID uint) (*models.TeamMember, error)
	TransferTeamOwner(ctx context.Context, teamID, newOwnerID, currentOwnerID, tenantID uint) (*TransferResult, error)
	IsMember(ctx context.Context, teamID, userID uint) bool
}

// MemberWithInfo 团队成员（含用户信息）
type MemberWithInfo struct {
	models.TeamMember
	Username string `json:"username"`
	Email    string `json:"email"`
}

// TransferResult 转让所有权结果
type TransferResult struct {
	Team      models.Team       `json:"team"`
	NewOwner  models.TeamMember `json:"new_owner"`
	OldOwner  models.TeamMember `json:"old_owner"`
}