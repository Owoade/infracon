package auth

type AuthPayload struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ForgotPasswordPayload struct {
	Password string `json:"password"`
	SetupKey string `json:"setup_key"`
}
