package model

type Profile struct {
	Base
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`
}

func NewProfile() *Profile {
	return &Profile{}
}

func (p *Profile) WithId(id string) *Profile {
	p.Id = id
	return p
}

func (p *Profile) WithFirstName(firstName string) *Profile {
	p.FirstName = firstName
	return p
}

func (p *Profile) WithLastName(lastName string) *Profile {
	p.LastName = lastName
	return p
}

func (p *Profile) WithEmail(email string) *Profile {
	p.Email = email
	return p
}
