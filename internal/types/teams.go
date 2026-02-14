package types

// Создание команды
type TeamCreateRequest struct {
	Name string `json:"name"`
}

type TeamResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Список команд, где состоит пользователь.
type TeamListResponse struct {
	Teams []TeamResponse `json:"teams"`
}

// Приглашение пользователя в команду.
type TeamInviteRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"` // "owner" или "member" (по умолчанию member)
}
