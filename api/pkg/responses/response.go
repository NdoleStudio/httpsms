package responses

type response struct {
	Status  string `json:"status" example:"success"`
	Message string `json:"message" example:"item created successfully"`
}
