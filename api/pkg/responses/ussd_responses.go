package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// USSDsResponse is the payload containing a list of entities.USSD
type USSDsResponse struct {
	response
	Data []entities.USSD `json:"data"`
}

// USSDResponse is the payload containing a single entities.USSD
type USSDResponse struct {
	response
	Data entities.USSD `json:"data"`
}