package model

type User struct {
	Base
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`
	Password  string `db:"password"`
	Enabled   bool   `db:"enabled"`
}

func NewUser() *User {
	return &User{}
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
