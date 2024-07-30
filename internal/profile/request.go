package profile

type GetRequest struct {
	Id string `validate:"required,uuid"`
}

type CreateRequest struct {
	FirstName string `validate:"required,max=72"`
	LastName  string `validate:"required,max=72"`
}

type UpdateRequest struct {
	Id        string `validate:"required,uuid"`
	FirstName string `validate:"required,max=72"`
	LastName  string `validate:"required,max=72"`
}

type DeleteRequest struct {
	Id string `validate:"required,uuid"`
}
