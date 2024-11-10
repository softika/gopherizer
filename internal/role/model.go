package role

import "github.com/softika/gopherizer/internal"

type Role struct {
	internal.Base

	Name string `db:"name"`
}

func New() *Role {
	return &Role{}
}

func (r *Role) WithId(id string) *Role {
	r.Id = id
	return r
}

func (r *Role) WithName(name string) *Role {
	r.Name = name
	return r
}

type AccountRole struct {
	internal.Base

	AccountId string `db:"account_id"`
	RoleId    string `db:"role_id"`
}
