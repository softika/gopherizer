package account

type RegisterRequest struct {
	Email    string `validate:"required,min=3,max=72"`
	Password string `validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `validate:"required,min=3,max=72"`
	Password string `validate:"required,min=8"`
}

type ChangePasswordRequest struct {
	AccountId   string `validate:"required"`
	OldPassword string `validate:"required,min=8"`
	NewPassword string `validate:"required,min=8"`
}
