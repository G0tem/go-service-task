package types

// Описывает создание задачи.
type TaskCreateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	TeamID      string `json:"team_id"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	Status      string `json:"status,omitempty"` // todo, in_progress, done, cancelled
}

// Описывает обновление задачи.
type TaskUpdateRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
}

// Упрощённое представление задачи.
type TaskResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	TeamID      string `json:"team_id"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	CreatedBy   string `json:"created_by"`
}

// Список задач с пагинацией.
type TaskListResponse struct {
	Tasks        []TaskResponse `json:"tasks"`
	CurrentPage  int            `json:"current_page"`
	PageSize     int            `json:"page_size"`
	TotalPages   int            `json:"total_pages"`
	TotalRecords int64          `json:"total_records"`
}

// Элемент истории изменений задачи.
type TaskHistoryItemResponse struct {
	Field     string `json:"field"`
	OldValue  string `json:"old_value"`
	NewValue  string `json:"new_value"`
	ChangedBy string `json:"changed_by"`
	CreatedAt string `json:"created_at"`
}
