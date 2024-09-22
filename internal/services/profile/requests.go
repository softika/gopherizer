package profile

type CreateRequest struct {
	FirstName string `validate:"required,max=72"`
	LastName  string `validate:"required,max=72"`
}

type UpdateRequest struct {
	Id        string `validate:"required"`
	FirstName string `validate:"max=72"`
	LastName  string `validate:"max=72"`
}
