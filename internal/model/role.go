package model

type Role struct {
	Base
	Name string `db:"name"`
}

func NewRole() *Role {
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
