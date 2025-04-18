package requests

// PhoneAPIKeyStoreRequest is the payload for storing a phone API key
type PhoneAPIKeyStoreRequest struct {
	request
	Name string `json:"name" example:"My Phone API Key"`
}

// Sanitize sets defaults to MessageReceive
func (input *PhoneAPIKeyStoreRequest) Sanitize() PhoneAPIKeyStoreRequest {
	input.Name = input.sanitizeAddress(input.Name)
	return *input
}
