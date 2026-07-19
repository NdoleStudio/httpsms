package responses

import "github.com/NdoleStudio/httpsms/pkg/entities"

// ContactResponse is the payload containing entities.Contact.
type ContactResponse struct {
	response
	Data entities.Contact `json:"data"`
}

// ContactsResponse is the payload containing []entities.Contact.
type ContactsResponse struct {
	response
	Data []entities.Contact `json:"data"`
}
