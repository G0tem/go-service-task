package types

// "для каждой команды: название, количество участников, количество задач в статусе done за последние 7 дней".
type TeamStatsResponse struct {
	TeamID          string `json:"team_id"`
	TeamName        string `json:"team_name"`
	MembersCount    int64  `json:"members_count"`
	DoneTasksLast7d int64  `json:"done_tasks_last_7d"`
}

// "топ-3 пользователя по количеству созданных задач в каждой команде за месяц".
type TeamTopCreatorsItem struct {
	TeamID     string `json:"team_id"`
	TeamName   string `json:"team_name"`
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	TasksCount int64  `json:"tasks_count"`
	RankInTeam int64  `json:"rank_in_team"`
}

// "задачи, где assignee не является членом команды этой задачи".
type InvalidAssigneeTask struct {
	TaskID     string `json:"task_id"`
	TeamID     string `json:"team_id"`
	AssigneeID string `json:"assignee_id"`
}
