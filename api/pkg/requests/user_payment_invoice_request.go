package requests

import (
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
)

// UserPaymentInvoice is the payload for generating a subscription payment invoice
type UserPaymentInvoice struct {
	request
	Name                  string `json:"name" example:"Acme Corp"`
	Address               string `json:"address" example:"221B Baker Street, London"`
	City                  string `json:"city" example:"Los Angeles"`
	State                 string `json:"state" example:"CA"`
	Country               string `json:"country" example:"US"`
	ZipCode               string `json:"zip_code" example:"9800"`
	Notes                 string `json:"notes" example:"Thank you for your business!"`
	SubscriptionInvoiceID string `json:"subscriptionInvoiceID" swaggerignore:"true"` // used internally for validation
}

// Sanitize sets defaults to MessageReceive
func (input *UserPaymentInvoice) Sanitize() UserPaymentInvoice {
	input.Name = input.sanitizeAddress(input.Name)
	input.Address = input.sanitizeAddress(input.Address)
	input.City = input.sanitizeAddress(input.City)
	input.State = input.sanitizeAddress(input.State)
	input.Country = input.sanitizeAddress(input.Country)
	input.ZipCode = input.sanitizeAddress(input.ZipCode)
	input.Notes = input.sanitizeAddress(input.Notes)
	return *input
}

// UserInvoiceGenerateParams converts UserPaymentInvoice to services.UserInvoiceGenerateParams
func (input *UserPaymentInvoice) UserInvoiceGenerateParams(userID entities.UserID) *services.UserInvoiceGenerateParams {
	return &services.UserInvoiceGenerateParams{
		UserID:                userID,
		SubscriptionInvoiceID: input.SubscriptionInvoiceID,
		Name:                  input.Name,
		Address:               input.Address,
		City:                  input.City,
		State:                 input.State,
		Country:               input.Country,
		Notes:                 input.Notes,
		ZipCode:               input.ZipCode,
	}
}
