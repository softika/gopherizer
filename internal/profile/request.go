package profile

// Validation is expressed with JSON Schema constraints, which huma derives the
// OpenAPI schema from and enforces on every request.
//
// Note that JSON Schema `required` only asserts presence, so a non-empty string
// needs minLength: without it `{"firstName": ""}` would be accepted.

type GetRequest struct {
	Id string
}

type CreateRequest struct {
	FirstName string `json:"firstName" minLength:"1" maxLength:"72" doc:"Given name" example:"John"`
	LastName  string `json:"lastName" minLength:"1" maxLength:"72" doc:"Family name" example:"Doe"`
}

type UpdateRequest struct {
	Id        string `json:"id" pattern:"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$" doc:"Profile identifier"`
	FirstName string `json:"firstName" minLength:"1" maxLength:"72" doc:"Given name" example:"John"`
	LastName  string `json:"lastName" minLength:"1" maxLength:"72" doc:"Family name" example:"Doe"`
}

type DeleteRequest struct {
	Id string
}
