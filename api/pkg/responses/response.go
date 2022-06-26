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

// NotFound is the response with status code is 404
type NotFound struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"cannot find message with ID [32343a19-da5e-4b1b-a767-3298a73703ca]"`
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

// Unauthorized is the response with status code is 403
type Unauthorized struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"You are not authorized to carry out this request."`
	Data    string `json:"data" example:"Make sure your API key is set in the [X-API-Key] header in the request"`
}
