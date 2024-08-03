package model

import (
	"time"

	"github.com/oklog/ulid/v2"
)

type User struct {
	Base
	FirstName string
	LastName  string
	Email     string
	Password  string
	Enabled   bool
}

func NewUser() *User {
	return &User{
		Base: Base{
			Id:        ulid.Make(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Enabled: true,
	}
}

func (u *User) WithFirstName(firstName string) *User {
	u.FirstName = firstName
	return u
}

func (u *User) WithLastName(lastName string) *User {
	u.LastName = lastName
	return u
}

func (u *User) WithEmail(email string) *User {
	u.Email = email
	return u
}

func (u *User) WithPassword(password string) *User {
	u.Password = password
	return u
}
