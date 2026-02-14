package types

// Структура для получения инфы о пользователе
type GetMeResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}
