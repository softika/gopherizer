package profile

import "github.com/softika/gopherizer/internal"

type Profile struct {
	internal.Base

	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	AccountId string `db:"account_id"`
}

func New() *Profile {
	return &Profile{}
}

func (p *Profile) WithId(id string) *Profile {
	p.Id = id
	return p
}

func (p *Profile) WithAccountId(accountId string) *Profile {
	p.AccountId = accountId
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
