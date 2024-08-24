package profile

type CreateRequest struct {
	FirstName string `validate:"required,max=72"`
	LastName  string `validate:"required,max=72"`
	Email     string `validate:"required,min=3"`
}

type UpdateRequest struct {
	Id        string `validate:"required"`
	FirstName string `validate:"max=72"`
	LastName  string `validate:"max=72"`
	Email     string `validate:"min=3"`
}
