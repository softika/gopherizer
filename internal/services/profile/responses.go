package profile

import (
	"time"

	"tldw/internal/model"
)

type Response struct {
	Id        string    `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (r *Response) fromModel(u *model.Profile) *Response {
	r.Id = u.Id
	r.FirstName = u.FirstName
	r.LastName = u.LastName
	r.Email = u.Email
	r.CreatedAt = u.CreatedAt
	r.UpdatedAt = u.UpdatedAt
	return r
}
