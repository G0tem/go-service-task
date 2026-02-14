package types

type LoginSuccessData struct {
	Token string `json:"token"`
}

type LoginRequest struct {
	Identity string `json:"identity"`
	Password string `json:"password"`
}
type LoginSuccessResponse struct {
	Status string           `json:"status"`
	Data   LoginSuccessData `json:"data"`
}
