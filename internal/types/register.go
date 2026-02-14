package types

type RegisterRequest struct {
	Email           string `json:"email"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}
