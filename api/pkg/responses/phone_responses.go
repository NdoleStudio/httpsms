package responses

import "github.com/NdoleStudio/http-sms-manager/pkg/entities"

// PhonesResponse is the payload containing entities.Phone
type PhonesResponse struct {
	response
	Data []entities.Phone `json:"data"`
}

// PhoneResponse is the payload containing entities.Phone
type PhoneResponse struct {
	response
	Data entities.Phone `json:"data"`
}
