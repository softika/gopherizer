package model

type Account struct {
	Base
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAccount() *Account {
	return &Account{}
}

func (a *Account) WithId(id string) *Account {
	a.Id = id
	return a
}

func (a *Account) WithUsername(username string) *Account {
	a.Username = username
	return a
}

func (a *Account) WithPassword(password string) *Account {
	a.Password = password
	return a
}
