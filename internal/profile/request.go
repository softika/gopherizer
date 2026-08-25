package profile

type GetRequest struct {
	Id string `validate:"required,uuid"`
}

type CreateRequest struct {
	FirstName string `json:"firstName" validate:"required,max=72"`
	LastName  string `json:"lastName" validate:"required,max=72"`
}

type UpdateRequest struct {
	Id        string `json:"id" validate:"required,uuid"`
	FirstName string `json:"firstName" validate:"required,max=72"`
	LastName  string `json:"lastName" validate:"required,max=72"`
}

type DeleteRequest struct {
	Id string `validate:"required,uuid"`
}
