package handler

import (
	"errors"

	"github.com/G0tem/go-service-task/internal/model"
	"github.com/G0tem/go-service-task/internal/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const (
	TeamRoleOwner  = "owner" // владелец
	TeamRoleMember = "member"
)

// createTeam
// @Summary createTeam info
// @Description CreateTeam создает команду и делает текущего пользователя её owner.
// @Tags teams
// @Produce json
// @Param team body types.TeamCreateRequest true "team data"
// @Success 200 {object} types.TeamResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /teams [post]
func (h *Handler) createTeam(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*JwtClaims)

	var req types.TeamCreateRequest
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid request body",
		})
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid user id in token",
		})
	}

	team := model.Team{
		Name:      req.Name,
		CreatedBy: userID,
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&team).Error; err != nil {
			return err
		}
		member := model.TeamMember{
			TeamID: team.ID,
			UserID: userID,
			Role:   TeamRoleOwner,
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to create team",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(types.TeamResponse{
		ID:   team.ID.String(),
		Name: team.Name,
	})
}

// listUserTeams
// @Summary listUserTeams info
// @Description ListUserTeams возвращает список команд, где состоит текущий пользователь.
// @Tags teams
// @Produce json
// @Success 200 {object} types.TeamListResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /teams [get]
func (h *Handler) listUserTeams(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*JwtClaims)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid user id in token",
		})
	}

	var teams []model.Team
	if err := h.db.
		Joins("JOIN team_members ON team_members.team_id = teams.id AND team_members.user_id = ?", userID).
		Find(&teams).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to load teams",
			Error:   err.Error(),
		})
	}

	resp := types.TeamListResponse{Teams: make([]types.TeamResponse, 0, len(teams))}
	for _, t := range teams {
		resp.Teams = append(resp.Teams, types.TeamResponse{
			ID:   t.ID.String(),
			Name: t.Name,
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// inviteToTeam
// @Summary inviteToTeam info
// @Description InviteToTeam приглашает пользователя в команду (может только owner).
// @Tags teams
// @Produce json
// @Param id path string true "team ID"
// @Param invite body types.TeamInviteRequest true "invite data"
// @Success 200 {object} types.TeamListResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /teams/{id}/invite [post]
func (h *Handler) inviteToTeam(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*JwtClaims)
	currentUserID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid user id in token",
		})
	}

	teamIDParam := c.Params("id")
	teamID, err := uuid.Parse(teamIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid team id",
		})
	}

	var req types.TeamInviteRequest
	if err := c.BodyParser(&req); err != nil || req.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid request body",
		})
	}

	inviteUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid invited user id",
		})
	}

	role := req.Role
	if role == "" {
		role = TeamRoleMember
	}
	if role != TeamRoleOwner && role != TeamRoleMember {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid role",
		})
	}

	// Проверяем, что текущий пользователь владелец
	var ownerMembership model.TeamMember
	if err := h.db.Where(&model.TeamMember{
		TeamID: teamID,
		UserID: currentUserID,
		Role:   TeamRoleOwner,
	}).First(&ownerMembership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusForbidden).JSON(types.FailureResponse{
				Status:  "error",
				Message: "only team owner can invite members",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to check team ownership",
			Error:   err.Error(),
		})
	}

	// Проверяем, что приглашаемый пользователь существует
	var user model.User
	if err := h.db.First(&user, "id = ?", inviteUserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
				Status:  "error",
				Message: "user not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to load user",
			Error:   err.Error(),
		})
	}

	// Создаем или обновляем membership.
	member := model.TeamMember{
		TeamID: teamID,
		UserID: inviteUserID,
		Role:   role,
	}
	if err := h.db.Where("team_id = ? AND user_id = ?", teamID, inviteUserID).
		Assign(map[string]interface{}{"role": role}).
		FirstOrCreate(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to invite user to team",
			Error:   err.Error(),
		})
	}

	// Запрос в mail сервис для реализации Circuit Breaker
	err = h.sendInviteEmailWithBreaker(user.Email, teamID, inviteUserID)
	if err != nil {
		log.Warn().Msgf("[inviteToTeam] Error send notify mail, Test Circuit Breaker, err: %s", err)
	}

	return c.Status(fiber.StatusOK).JSON(types.SuccessResponse{
		Status:  "ok",
		Message: "user invited to team",
	})
}
