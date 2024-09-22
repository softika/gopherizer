package model

type AccountRole struct {
	Base
	AccountId string `db:"account_id"`
	RoleId    string `db:"role_id"`
}

// Identity represents the user identity
type Identity struct {
	AccountId string
	Email     string
	Password  string
	Roles     []string
}
