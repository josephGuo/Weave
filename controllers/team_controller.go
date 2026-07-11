package controllers

import (
	"net/http"
	"strconv"

	"weave/pkg"
	teamsvc "weave/services/team"

	"github.com/gin-gonic/gin"
)

// TeamController 团队控制器
type TeamController struct {
	teamService teamsvc.TeamService
}

// NewTeamController 创建团队控制器实例
func NewTeamController(teamSvc teamsvc.TeamService) *TeamController {
	return &TeamController{teamService: teamSvc}
}

// UpdateTeam 更新团队信息
func (tc *TeamController) UpdateTeam(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"omitempty,min=2,max=100"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		err := pkg.NewValidationError("Invalid team data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	teamIDStr := c.Param("id")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		err := pkg.NewValidationError("Team ID is required", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	team, err := tc.teamService.UpdateTeam(c.Request.Context(), uint(teamID), req.Name, req.Description, userID, tenantID)
	if err != nil {
		appErr := pkg.NewDatabaseError("Failed to update team", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "update",
		ResourceType: "team",
		ResourceID:   team.Name,
		NewValue:     team,
	})

	c.JSON(http.StatusOK, team)
}

// CreateTeam 创建团队
func (tc *TeamController) CreateTeam(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required,min=2,max=100"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		err := pkg.NewValidationError("Invalid team data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	tenantID := c.GetUint("tenant_id")
	ownerID := c.GetUint("user_id")

	team, err := tc.teamService.CreateTeam(c.Request.Context(), req.Name, req.Description, ownerID, tenantID)
	if err != nil {
		appErr := pkg.NewDatabaseError("Failed to create team", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "create",
		ResourceType: "team",
		ResourceID:   team.Name,
		NewValue:     team,
	})

	c.JSON(http.StatusCreated, team)
}

// GetTeamMembers 获取团队成员列表
func (tc *TeamController) GetTeamMembers(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		err := pkg.NewValidationError("Invalid team ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	members, err := tc.teamService.GetTeamMembers(c.Request.Context(), uint(teamID), userID, tenantID)
	if err != nil {
		appErr := pkg.NewDatabaseError("Failed to query team members", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, members)
}

// AddTeamMember 添加团队成员
func (tc *TeamController) AddTeamMember(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		err := pkg.NewValidationError("Invalid team ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Role   string `json:"role" binding:"required,oneof=admin member"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		err := pkg.NewValidationError("Invalid member data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	newMember, err := tc.teamService.AddTeamMember(c.Request.Context(), uint(teamID), req.UserID, req.Role, userID, tenantID)
	if err != nil {
		appErr := pkg.NewDatabaseError("Failed to add team member", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "add_member",
		ResourceType: "team",
		ResourceID:   teamIDStr,
		NewValue:     newMember,
	})

	c.JSON(http.StatusCreated, newMember)
}

// RemoveTeamMember 移除团队成员
func (tc *TeamController) RemoveTeamMember(c *gin.Context) {
	teamIDStr := c.Param("id")
	memberIDStr := c.Param("memberId")

	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		err := pkg.NewValidationError("Invalid team ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		err := pkg.NewValidationError("Invalid member ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	if err := tc.teamService.RemoveTeamMember(c.Request.Context(), uint(teamID), uint(memberID), userID, tenantID); err != nil {
		appErr := pkg.NewDatabaseError("Failed to remove team member", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "remove_member",
		ResourceType: "team",
		ResourceID:   teamIDStr,
		OldValue:     map[string]interface{}{"user_id": memberID},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Team member removed successfully"})
}

// GetTeams 获取当前租户下的团队列表
func (tc *TeamController) GetTeams(c *gin.Context) {
	tenantID := c.GetUint("tenant_id")
	userID := c.GetUint("user_id")

	teams, err := tc.teamService.GetTeams(c.Request.Context(), userID, tenantID)
	if err != nil {
		appErr := pkg.NewDatabaseError("Failed to query teams", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, teams)
}

// TransferTeamOwner 转让团队所有权
func (tc *TeamController) TransferTeamOwner(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		err := pkg.NewValidationError("Invalid team ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	var req struct {
		NewOwnerID uint `json:"new_owner_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		err := pkg.NewValidationError("Invalid owner data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	result, err := tc.teamService.TransferTeamOwner(c.Request.Context(), uint(teamID), req.NewOwnerID, userID, tenantID)
	if err != nil {
		appErr := pkg.NewDatabaseError("Failed to transfer team ownership", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "transfer_ownership",
		ResourceType: "team",
		ResourceID:   result.Team.Name,
		OldValue:     map[string]interface{}{"owner_id": userID},
		NewValue:     map[string]interface{}{"owner_id": req.NewOwnerID},
	})

	c.JSON(http.StatusOK, gin.H{
		"message":   "Team ownership transferred successfully",
		"team":      result.Team,
		"new_owner": result.NewOwner,
		"old_owner": result.OldOwner,
	})
}

// SearchTeamMembers 搜索团队成员
func (tc *TeamController) SearchTeamMembers(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		err := pkg.NewValidationError("Invalid team ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	keyword := c.Query("keyword")
	if keyword == "" {
		err := pkg.NewValidationError("Search keyword is required", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	members, err := tc.teamService.SearchTeamMembers(c.Request.Context(), uint(teamID), userID, tenantID, keyword)
	if err != nil {
		appErr := pkg.NewDatabaseError("Failed to search team members", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, members)
}

// UpdateMemberRole 更新团队成员角色
func (tc *TeamController) UpdateMemberRole(c *gin.Context) {
	teamIDStr := c.Param("id")
	memberIDStr := c.Param("memberId")

	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		err := pkg.NewValidationError("Invalid team ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		err := pkg.NewValidationError("Invalid member ID", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=admin member"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		err := pkg.NewValidationError("Invalid role data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	userID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	member, err := tc.teamService.UpdateMemberRole(c.Request.Context(), uint(teamID), uint(memberID), req.Role, userID, tenantID)
	if err != nil {
		appErr := pkg.NewDatabaseError("Failed to update member role", err)
		c.JSON(pkg.GetHTTPStatus(appErr), gin.H{"code": string(appErr.Code), "message": appErr.Message})
		return
	}

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "update_member_role",
		ResourceType: "team",
		ResourceID:   teamIDStr,
		NewValue:     map[string]interface{}{"user_id": member.UserID, "role": member.Role},
	})

	c.JSON(http.StatusOK, member)
}
