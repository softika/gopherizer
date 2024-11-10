package account

import "github.com/softika/gopherizer/internal"

type Account struct {
	internal.Base

	Email    string `db:"email"`
	Password string `db:"password"`
}

func New() *Account {
	return &Account{}
}

func (a *Account) WithId(id string) *Account {
	a.Id = id
	return a
}

func (a *Account) WithEmail(email string) *Account {
	a.Email = email
	return a
}

func (a *Account) WithPassword(password string) *Account {
	a.Password = password
	return a
}

// Identity represents the user identity
type Identity struct {
	AccountId string
	Email     string
	Password  string
	Roles     []string
}
