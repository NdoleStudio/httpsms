package entities

import (
	"time"

	"github.com/google/uuid"
)

// USSDType represents the type of USSD session
type USSDType string

const (
	// USSDTypeRequest is a USSD request from a mobile phone
	USSDTypeRequest = USSDType("request")
	// USSDTypeResponse is a USSD response from the application
	USSDTypeResponse = USSDType("response")
)

// USSDDirection represents the direction of USSD communication
type USSDDirection string

const (
	// USSDDirectionMoToApp means the USSD message comes from the mobile phone to the application
	USSDDirectionMoToApp = USSDDirection("MO-to-App")
	// USSDDirectionAppToMO means the USSD message goes from the application to the mobile phone
	USSDDirectionAppToMO = USSDDirection("App-to-MO")
)

// USSDStatus represents the status of a USSD session
type USSDStatus string

const (
	// USSDStatusPending means the USSD session is pending
	USSDStatusPending = USSDStatus("pending")
	// USSDStatusActive means the USSD session is active
	USSDStatusActive = USSDStatus("active")
	// USSDStatusCompleted means the USSD session is completed
	USSDStatusCompleted = USSDStatus("completed")
	// USSDStatusFailed means the USSD session has failed
	USSDStatusFailed = USSDStatus("failed")
	// USSDStatusTimedOut means the USSD session has timed out
	USSDStatusTimedOut = USSDStatus("timed-out")
)

// USSD represents a USSD session on the phone
type USSD struct {
	ID             uuid.UUID    `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID         UserID       `json:"user_id" gorm:"index:idx_ussds__user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	PhoneID        uuid.UUID    `json:"phone_id" gorm:"type:uuid;index:idx_ussds__phone_id" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	Owner          string       `json:"owner" example:"+18005550199"`
	SessionID      string       `json:"session_id" example:"USSDSESSION12345" gorm:"index:idx_ussds__session_id"`
	Type           USSDType     `json:"type" example:"request"`
	Direction      USSDDirection `json:"direction" example:"MO-to-App"`
	Content        string       `json:"content" example:"*123#"`
	Response       *string      `json:"response" example:"Welcome to USSD menu" validate:"optional"`
	Status         USSDStatus   `json:"status" example:"pending" gorm:"default:pending"`
	SIM            SIM          `json:"sim" example:"SIM1"`
	Timestamp      time.Time    `json:"timestamp" example:"2022-06-05T14:26:09.527976+03:00"`
	CreatedAt      time.Time    `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt      time.Time    `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}

// TableName specifies the table name for the USSD entity
func (USSD) TableName() string {
	return "ussds"
}

// IsActive checks if the USSD session is active
func (ussd *USSD) IsActive() bool {
	return ussd.Status == USSDStatusActive
}

// IsPending checks if the USSD session is pending
func (ussd *USSD) IsPending() bool {
	return ussd.Status == USSDStatusPending
}

// IsCompleted checks if the USSD session is completed
func (ussd *USSD) IsCompleted() bool {
	return ussd.Status == USSDStatusCompleted
}

// IsFailed checks if the USSD session has failed
func (ussd *USSD) IsFailed() bool {
	return ussd.Status == USSDStatusFailed
}

// IsTimedOut checks if the USSD session has timed out
func (ussd *USSD) IsTimedOut() bool {
	return ussd.Status == USSDStatusTimedOut
}

// MarkActive marks the USSD session as active
func (ussd *USSD) MarkActive() *USSD {
	ussd.Status = USSDStatusActive
	return ussd
}

// MarkCompleted marks the USSD session as completed
func (ussd *USSD) MarkCompleted(response string) *USSD {
	ussd.Status = USSDStatusCompleted
	ussd.Response = &response
	return ussd
}

// MarkFailed marks the USSD session as failed
func (ussd *USSD) MarkFailed() *USSD {
	ussd.Status = USSDStatusFailed
	return ussd
}

// MarkTimedOut marks the USSD session as timed out
func (ussd *USSD) MarkTimedOut() *USSD {
	ussd.Status = USSDStatusTimedOut
	return ussd
}