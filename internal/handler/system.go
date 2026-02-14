package handler

import (
	"github.com/G0tem/go-service-task/internal/types"
	"github.com/gofiber/fiber/v2"
)

// getTeamStats
// @Summary getTeamStats оборачивает сложный SQL info
// @Description "Получить для каждой команды: название, количество участников, количество задач в статусе done за последние 7 дней".
// @Tags complex-sql
// @Produce json
// @Success 200 {object} types.TaskResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /system/teams-stats [get]
func (h *Handler) getTeamStats(c *fiber.Ctx) error {
	var rows []types.TeamStatsResponse

	sql := `
SELECT
    t.id   AS team_id,
    t.name AS team_name,
    COUNT(DISTINCT tm.user_id)                                            AS members_count,
    COALESCE(SUM(CASE WHEN tasks.status = 'done' THEN 1 ELSE 0 END), 0)   AS done_tasks_last_7d
FROM teams t
LEFT JOIN team_members tm ON tm.team_id = t.id
LEFT JOIN tasks ON tasks.team_id = t.id
               AND tasks.status = 'done'
               AND tasks.created_at >= NOW() - INTERVAL 7 DAY
GROUP BY t.id, t.name
ORDER BY t.name;
`

	if err := h.db.Raw(sql).Scan(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to execute team stats query",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(rows)
}

// getTopCreatorsByTeam
// @Summary getTopCreatorsByTeam оборачивает сложный SQL с оконной функцией info
// @Description "Получить топ-3 пользователя по количеству созданных задач в каждой команде за месяц".
// @Tags complex-sql
// @Produce json
// @Success 200 {object} types.TaskResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /system/teams-top-creators [get]
func (h *Handler) getTopCreatorsByTeam(c *fiber.Ctx) error {
	var rows []types.TeamTopCreatorsItem

	sql := `
WITH task_counts AS (
    SELECT
        t.team_id,
        t.created_by     AS user_id,
        COUNT(*)         AS tasks_count
    FROM tasks t
    WHERE t.created_at >= NOW() - INTERVAL 1 MONTH
    GROUP BY t.team_id, t.created_by
), ranked AS (
    SELECT
        tc.team_id,
        tc.user_id,
        tc.tasks_count,
        ROW_NUMBER() OVER (PARTITION BY tc.team_id ORDER BY tc.tasks_count DESC) AS rn
    FROM task_counts tc
)
SELECT
    ranked.team_id            AS team_id,
    teams.name                AS team_name,
    ranked.user_id            AS user_id,
    users.username            AS username,
    ranked.tasks_count        AS tasks_count,
    ranked.rn                 AS rank_in_team
FROM ranked
JOIN teams  ON teams.id = ranked.team_id
JOIN users  ON users.id = ranked.user_id
WHERE ranked.rn <= 3
ORDER BY team_name, rank_in_team;
`

	if err := h.db.Raw(sql).Scan(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to execute top creators query",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(rows)
}

// getTasksWithInvalidAssignee
// @Summary getTasksWithInvalidAssignee оборачивает запрос info
// @Description "Найти задачи, где assignee не является членом команды этой задачи".
// @Tags complex-sql
// @Produce json
// @Success 200 {object} types.TaskResponse
// @Failure 401 {object} types.FailureResponse
// @Failure 500 {object} types.FailureErrorResponse
// @Security ApiKeyAuth
// @Router /system/invalid-assignees [get]
func (h *Handler) getTasksWithInvalidAssignee(c *fiber.Ctx) error {
	var rows []types.InvalidAssigneeTask

	sql := `
SELECT
    tasks.id        AS task_id,
    tasks.team_id   AS team_id,
    tasks.assignee_id AS assignee_id
FROM tasks
LEFT JOIN team_members tm
    ON tm.team_id = tasks.team_id
   AND tm.user_id = tasks.assignee_id
WHERE tasks.assignee_id IS NOT NULL
  AND tm.user_id IS NULL;
`

	if err := h.db.Raw(sql).Scan(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.FailureErrorResponse{
			Status:  "error",
			Message: "failed to execute invalid assignees query",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(rows)
}
