package profile

import (
	"time"
)

type Response struct {
	Id        string    `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (r *Response) fromModel(u *Profile) *Response {
	r.Id = u.Id
	r.FirstName = u.FirstName
	r.LastName = u.LastName
	r.CreatedAt = u.CreatedAt
	r.UpdatedAt = u.UpdatedAt
	return r
}
