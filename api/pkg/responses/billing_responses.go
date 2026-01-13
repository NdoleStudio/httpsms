package responses

import (
	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// BillingUsagesResponse is the payload containing []entities.BillingUsage
type BillingUsagesResponse struct {
	response
	Data []entities.BillingUsage `json:"data"`
}

// BillingUsageResponse is the payload containing entities.BillingUsage
type BillingUsageResponse struct {
	response
	Data entities.BillingUsage `json:"data"`
}
