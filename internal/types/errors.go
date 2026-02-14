package types

type FailureResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type FailureErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error"`
}
