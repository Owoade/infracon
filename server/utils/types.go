package utils

type ResponsePayload struct {
	StatusCode int
	Message    string `json:"message"`
	Status     bool   `json:"status"`
	Data       any    `json:"data"`
}
