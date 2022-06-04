package responses

type response struct {
	Status  string `json:"status" example:"success"`
	Message string `json:"message" example:"item created successfully"`
}

// InternalServerError is the response with status code is 500
type InternalServerError struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"We ran into an internal error while handling the request."`
}

// BadRequest is the response with status code is 400
type BadRequest struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"The request isn't properly formed"`
	Data    string `json:"data" example:"The request body is not a valid JSON string"`
}

// UnprocessableEntity is the response with status code is 422
type UnprocessableEntity struct {
	Status  string              `json:"status" example:"error"`
	Message string              `json:"message" example:"validation errors while sending message"`
	Data    map[string][]string `json:"data"`
}
