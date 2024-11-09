package account

type RegisterResponse struct {
	AccountId string `json:"account_id"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type ChangePasswordResponse struct {
	AccountId string `json:"account_id"`
}
