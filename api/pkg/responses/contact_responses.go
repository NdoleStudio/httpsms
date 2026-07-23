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
	// Total is the number of contacts matching the request filter for the
	// user, independent of the pagination skip/limit applied to Data.
	Total int64 `json:"total" example:"57"`
}
