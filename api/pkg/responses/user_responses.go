package responses

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// UserResponse is the payload containing entities.User
type UserResponse struct {
	response
	Data entities.User `json:"data"`
}

// UserSubscriptionPaymentsResponse is the payload containing lemonsqueezy.SubscriptionInvoicesAPIResponse
type UserSubscriptionPaymentsResponse struct {
	response
	Data []struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			BillingReason           string    `json:"billing_reason"`
			CardBrand               string    `json:"card_brand"`
			CardLastFour            string    `json:"card_last_four"`
			Currency                string    `json:"currency"`
			CurrencyRate            string    `json:"currency_rate"`
			Status                  string    `json:"status"`
			StatusFormatted         string    `json:"status_formatted"`
			Refunded                bool      `json:"refunded"`
			RefundedAt              any       `json:"refunded_at"`
			Subtotal                int       `json:"subtotal"`
			DiscountTotal           int       `json:"discount_total"`
			Tax                     int       `json:"tax"`
			TaxInclusive            bool      `json:"tax_inclusive"`
			Total                   int       `json:"total"`
			RefundedAmount          int       `json:"refunded_amount"`
			SubtotalUsd             int       `json:"subtotal_usd"`
			DiscountTotalUsd        int       `json:"discount_total_usd"`
			TaxUsd                  int       `json:"tax_usd"`
			TotalUsd                int       `json:"total_usd"`
			RefundedAmountUsd       int       `json:"refunded_amount_usd"`
			SubtotalFormatted       string    `json:"subtotal_formatted"`
			DiscountTotalFormatted  string    `json:"discount_total_formatted"`
			TaxFormatted            string    `json:"tax_formatted"`
			TotalFormatted          string    `json:"total_formatted"`
			RefundedAmountFormatted string    `json:"refunded_amount_formatted"`
			CreatedAt               time.Time `json:"created_at"`
			UpdatedAt               time.Time `json:"updated_at"`
		} `json:"attributes"`
	} `json:"data"`
}
