package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/G0tem/go-service-task/internal"
	"github.com/G0tem/go-service-task/internal/model"
	"github.com/G0tem/go-service-task/internal/types"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const (
	taskCacheTTL          = 5 * time.Minute
	taskTeamCacheKeyPrefx = "team_tasks:"
)

// invalidateTeamTasksCache удаляет все ключи кеша списка задач команды (team_tasks:<id>, team_tasks:<id>:ps*).
func invalidateTeamTasksCache(ctx context.Context, rds *redis.Client, teamID string) {
	pattern := taskTeamCacheKeyPrefx + teamID + "*"
	keys, err := rds.Keys(ctx, pattern).Result()
	if err != nil {
		log.Warn().Err(err).Str("pattern", pattern).Msg("failed to get cache keys for invalidation")
		return
	}
	if len(keys) > 0 {
		_ = rds.Del(ctx, keys...).Err()
	}
}

// isTeamMember проверяет, что пользователь состоит в команде.
func (h *Handler) isTeamMember(userID, teamID uuid.UUID) (bool, error) {
	var tm model.TeamMember
	err := h.db.Where(&model.TeamMember{
		UserID: userID,
		TeamID: teamID,
	}).First(&tm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// isTeamOwner проверяет, что пользователь — owner команды.
func (h *Handler) isTeamOwner(userID, teamID uuid.UUID) (bool, error) {
	var tm model.TeamMember
	err := h.db.Where(&model.TeamMember{
		UserID: userID,
		TeamID: teamID,
		Role:   TeamRoleOwner,
	}).First(&tm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// createTask
// @Summary createTask info
// @Description CreateTask создаёт задачу для команды (только член команды).
// @Tags task
// @Produce json
// @Param task body types.TaskCreateRequest true "task data"
// @Success 200 {object} types.TaskResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /tasks [post]
func (h *Handler) createTask(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*JwtClaims)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid user id in token",
		})
	}

	var req types.TaskCreateRequest
	if err := c.BodyParser(&req); err != nil || req.Title == "" || req.TeamID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid request body",
		})
	}

	teamID, err := uuid.Parse(req.TeamID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid team id",
		})
	}

	isMember, err := h.isTeamMember(userID, teamID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to check team membership",
			Error:   err.Error(),
		})
	}
	if !isMember {
		return c.Status(fiber.StatusForbidden).JSON(types.FailureResponse{
			Status:  "error",
			Message: "only team member can create tasks",
		})
	}

	var assigneeID uuid.UUID
	if req.AssigneeID != "" {
		assigneeID, err = uuid.Parse(req.AssigneeID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
				Status:  "error",
				Message: "invalid assignee id",
			})
		}
	}

	status := req.Status
	if status == "" {
		status = "todo"
	}

	task := model.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		TeamID:      teamID,
		AssigneeID:  assigneeID,
		CreatedBy:   userID,
	}

	if err := h.db.Create(&task).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to create task",
			Error:   err.Error(),
		})
	}

	invalidateTeamTasksCache(context.Background(), h.redis, task.TeamID.String())
	return c.Status(fiber.StatusCreated).JSON(taskToResponse(task))
}

// listTasks
// @Summary listTasks info
// @Description ListTasks возвращает список задач с фильтрами и пагинацией.
// @Tags task
// @Produce json
// @Param team_id query string false "team ID"
// @Param status query string false "status"
// @Param assignee_id query string false "assignee ID"
// @Param page query string false "page"
// @Param page_size query string false "page_size"
// @Success 200 {object} types.TaskResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /tasks [get]
func (h *Handler) listTasks(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*JwtClaims)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid user id in token",
		})
	}

	teamIDStr := c.Query("team_id", "")
	status := c.Query("status", "")
	assigneeIDStr := c.Query("assignee_id", "")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var teamID uuid.UUID
	if teamIDStr != "" {
		teamID, err = uuid.Parse(teamIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
				Status:  "error",
				Message: "invalid team id",
			})
		}

		// Проверяем, что пользователь состоит в этой команде.
		isMember, err := h.isTeamMember(userID, teamID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
				Status:  "error",
				Message: "failed to check team membership",
				Error:   err.Error(),
			})
		}
		if !isMember {
			return c.Status(fiber.StatusForbidden).JSON(types.FailureResponse{
				Status:  "error",
				Message: "access denied for this team",
			})
		}
	}

	// Пробуем читать из кеша только для первой страницы без фильтров (ключ учитывает page_size).
	ctx := context.Background()
	if teamIDStr != "" && status == "" && assigneeIDStr == "" && page == 1 {
		cacheKey := fmt.Sprintf("%s%s:ps%d", taskTeamCacheKeyPrefx, teamIDStr, pageSize)
		if data, err := h.redis.Get(ctx, cacheKey).Bytes(); err == nil && len(data) > 0 {
			c.Response().Header.SetContentType(fiber.MIMEApplicationJSONCharsetUTF8)
			return c.Send(data)
		}
	}

	var assigneeID uuid.UUID
	if assigneeIDStr != "" {
		assigneeID, err = uuid.Parse(assigneeIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
				Status:  "error",
				Message: "invalid assignee id",
			})
		}
	}

	var tasks []model.Task
	baseQuery := h.db.Model(&model.Task{})
	if teamIDStr != "" {
		baseQuery = baseQuery.Where("tasks.team_id = ?", teamID)
	} else {
		baseQuery = baseQuery.Joins("JOIN team_members ON team_members.team_id = tasks.team_id AND team_members.user_id = ?", userID)
	}
	if status != "" {
		baseQuery = baseQuery.Where("tasks.status = ?", status)
	}
	if assigneeIDStr != "" {
		baseQuery = baseQuery.Where("tasks.assignee_id = ?", assigneeID)
	}

	var total int64
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to count tasks",
			Error:   err.Error(),
		})
	}

	if err := baseQuery.
		Order("tasks.created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&tasks).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to load tasks",
			Error:   err.Error(),
		})
	}

	resp := types.TaskListResponse{
		Tasks:        make([]types.TaskResponse, 0, len(tasks)),
		CurrentPage:  page,
		PageSize:     pageSize,
		TotalRecords: total,
		TotalPages:   internal.PaginationTotalPages(total, pageSize),
	}

	for _, t := range tasks {
		resp.Tasks = append(resp.Tasks, taskToResponse(t))
	}

	// Кешируем только первую страницу по команде без фильтров (ключ с page_size).
	if teamIDStr != "" && status == "" && assigneeIDStr == "" && page == 1 {
		cacheKey := fmt.Sprintf("%s%s:ps%d", taskTeamCacheKeyPrefx, teamIDStr, pageSize)
		if data, err := c.App().Config().JSONEncoder(resp); err == nil {
			if err := h.redis.Set(ctx, cacheKey, data, taskCacheTTL).Err(); err != nil {
				log.Warn().Err(err).Msg("failed to cache team tasks")
			}
		}
	}

	return c.JSON(resp)
}

// updateTask
// @Summary updateTask info
// @Description UpdateTask обновляет задачу и пишет историю изменений.
// @Tags task
// @Produce json
// @Param id path string false "task ID"
// @Param task body types.TaskUpdateRequest true "task data"
// @Success 200 {object} types.TaskResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /tasks/{id} [put]
func (h *Handler) updateTask(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*JwtClaims)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid user id in token",
		})
	}

	taskIDStr := c.Params("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid task id",
		})
	}

	var req types.TaskUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid request body",
		})
	}

	var task model.Task
	if err := h.db.First(&task, "id = ?", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(types.FailureResponse{
				Status:  "error",
				Message: "task not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to load task",
			Error:   err.Error(),
		})
	}

	// Чек на право изменения
	isMember, err := h.isTeamMember(userID, task.TeamID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to check team membership",
			Error:   err.Error(),
		})
	}
	if !isMember {
		return c.Status(fiber.StatusForbidden).JSON(types.FailureResponse{
			Status:  "error",
			Message: "only team member can update task",
		})
	}

	isOwner, err := h.isTeamOwner(userID, task.TeamID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to check team owner",
			Error:   err.Error(),
		})
	}

	if userID != task.CreatedBy && userID != task.AssigneeID && !isOwner {
		return c.Status(fiber.StatusForbidden).JSON(types.FailureResponse{
			Status:  "error",
			Message: "not enough rights to update task",
		})
	}

	// Фиксируем старые значения.
	oldTask := task

	// Применяем изменения.
	var historyRecords []model.TaskHistory

	if req.Title != nil && *req.Title != task.Title {
		historyRecords = append(historyRecords, model.TaskHistory{
			TaskID:    task.ID,
			ChangedBy: userID,
			Field:     "title",
			OldValue:  task.Title,
			NewValue:  *req.Title,
		})
		task.Title = *req.Title
	}
	if req.Description != nil && *req.Description != task.Description {
		historyRecords = append(historyRecords, model.TaskHistory{
			TaskID:    task.ID,
			ChangedBy: userID,
			Field:     "description",
			OldValue:  task.Description,
			NewValue:  *req.Description,
		})
		task.Description = *req.Description
	}
	if req.Status != nil && *req.Status != task.Status {
		historyRecords = append(historyRecords, model.TaskHistory{
			TaskID:    task.ID,
			ChangedBy: userID,
			Field:     "status",
			OldValue:  task.Status,
			NewValue:  *req.Status,
		})
		task.Status = *req.Status
	}
	if req.AssigneeID != nil {
		var newAssignee uuid.UUID
		if *req.AssigneeID != "" {
			newAssignee, err = uuid.Parse(*req.AssigneeID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
					Status:  "error",
					Message: "invalid assignee id",
				})
			}
		}
		if newAssignee != task.AssigneeID {
			historyRecords = append(historyRecords, model.TaskHistory{
				TaskID:    task.ID,
				ChangedBy: userID,
				Field:     "assignee_id",
				OldValue:  oldTask.AssigneeID.String(),
				NewValue:  newAssignee.String(),
			})
			task.AssigneeID = newAssignee
		}
	}

	if len(historyRecords) == 0 {
		return c.Status(fiber.StatusOK).JSON(taskToResponse(task))
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&task).Error; err != nil {
			return err
		}
		if err := tx.Create(&historyRecords).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to update task",
			Error:   err.Error(),
		})
	}

	invalidateTeamTasksCache(context.Background(), h.redis, task.TeamID.String())
	return c.Status(fiber.StatusOK).JSON(taskToResponse(task))
}

// getTaskHistory
// @Summary getTaskHistory info
// @Description GetTaskHistory возвращает историю изменений задачи.
// @Tags task
// @Produce json
// @Param id path string false "task ID"
// @Success 200 {object} types.TaskResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /tasks/{id}/history [get]
func (h *Handler) getTaskHistory(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*JwtClaims)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid user id in token",
		})
	}

	taskIDStr := c.Params("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.FailureResponse{
			Status:  "error",
			Message: "invalid task id",
		})
	}

	var task model.Task
	if err := h.db.First(&task, "id = ?", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(types.FailureResponse{
				Status:  "error",
				Message: "task not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to load task",
			Error:   err.Error(),
		})
	}

	// Чек, что пользователь состоит в команде
	isMember, err := h.isTeamMember(userID, task.TeamID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to check team membership",
			Error:   err.Error(),
		})
	}
	if !isMember {
		return c.Status(fiber.StatusForbidden).JSON(types.FailureResponse{
			Status:  "error",
			Message: "only team member can view task history",
		})
	}

	var history []model.TaskHistory
	if err := h.db.Where("task_id = ?", taskID).Order("created_at ASC").Find(&history).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to load task history",
			Error:   err.Error(),
		})
	}

	resp := make([]types.TaskHistoryItemResponse, 0, len(history))
	for _, hItem := range history {
		resp = append(resp, types.TaskHistoryItemResponse{
			Field:     hItem.Field,
			OldValue:  hItem.OldValue,
			NewValue:  hItem.NewValue,
			ChangedBy: hItem.ChangedBy.String(),
			CreatedAt: hItem.CreatedAt.Format(time.RFC3339),
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func taskToResponse(t model.Task) types.TaskResponse {
	resp := types.TaskResponse{
		ID:          t.ID.String(),
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		TeamID:      t.TeamID.String(),
		CreatedBy:   t.CreatedBy.String(),
	}
	if t.AssigneeID != uuid.Nil {
		resp.AssigneeID = t.AssigneeID.String()
	}
	return resp
}
