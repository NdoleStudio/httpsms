package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// PhoneAPIKeyResponse is the payload containing an entities.PhoneAPIKey
type PhoneAPIKeyResponse struct {
	response
	Data *entities.PhoneAPIKey `json:"data"`
}

// PhoneAPIKeysResponse is the payload containing []entities.PhoneAPIKey
type PhoneAPIKeysResponse struct {
	response
	Data []*entities.PhoneAPIKey `json:"data"`
}
