package model

type Account struct {
	Base
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAccount() *Account {
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
