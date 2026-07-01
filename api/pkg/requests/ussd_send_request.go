package requests

import (
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
)

// USSDReceive is the payload for receiving a USSD request from a phone
type USSDReceive struct {
	request
	From      string       `json:"from" example:"+18005550199"`
	To        string       `json:"to" example:"+18005550100"`
	Content   string       `json:"content" example:"*123#"`
	SessionID string       `json:"session_id" example:"USSDSESSION12345"`
	SIM       entities.SIM `json:"sim" example:"SIM1"`
	Timestamp time.Time    `json:"timestamp" example:"2022-06-05T14:26:09.527976+03:00"`
}

// Sanitize sets defaults to USSDReceive
func (input *USSDReceive) Sanitize() USSDReceive {
	input.To = input.sanitizeAddress(input.To)
	input.From = input.sanitizeContact(input.To, input.From)
	input.SessionID = strings.TrimSpace(input.SessionID)
	if strings.TrimSpace(string(input.SIM)) == "" || input.SIM == ("DEFAULT") {
		input.SIM = entities.SIM1
	}
	return *input
}

// ToUSSDReceiveParams converts USSDReceive to services.USSDReceiveParams
func (input *USSDReceive) ToUSSDReceiveParams(userID entities.UserID, source string) *services.USSDReceiveParams {
	return &services.USSDReceiveParams{
		Source:    source,
		UserID:    userID,
		Owner:     input.To,
		Contact:   input.From,
		Content:   input.Content,
		SessionID: input.SessionID,
		SIM:       input.SIM,
		Timestamp: input.Timestamp,
	}
}

// USSDSend is the payload for sending a USSD response to a phone
type USSDSend struct {
	request
	To      string `json:"to" example:"+18005550199"`
	From    string `json:"from" example:"+18005550100"`
	Content string `json:"content" example:"Welcome to USSD menu"`
	// SessionID is the USSD session identifier
	SessionID string `json:"session_id" example:"USSDSESSION12345"`
}

// Sanitize sets defaults to USSDSend
func (input *USSDSend) Sanitize() USSDSend {
	input.To = input.sanitizeAddress(input.To)
	input.From = input.sanitizeAddress(input.From)
	input.SessionID = strings.TrimSpace(input.SessionID)
	return *input
}

// ToUSSDSendParams converts USSDSend to services.USSDSendParams
func (input *USSDSend) ToUSSDSendParams(userID entities.UserID, source string) *services.USSDSendParams {
	return &services.USSDSendParams{
		Source:    source,
		UserID:    userID,
		Owner:     input.To,
		Contact:   input.From,
		Content:   input.Content,
		SessionID: input.SessionID,
	}
}

// USSDIndex is the payload for fetching USSD session history
type USSDIndex struct {
	request
	Skip    int    `query:"skip" json:"skip" example:"0"`
	Query   string `query:"query" json:"query" example:"*123#"`
	Limit   int    `query:"limit" json:"limit" example:"10"`
	PhoneID string `query:"phone_id" json:"phone_id" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
}

// Sanitize sets defaults to USSDIndex
func (input *USSDIndex) Sanitize() USSDIndex {
	if input.Limit <= 0 || input.Limit > 20 {
		input.Limit = 10
	}
	if input.Skip < 0 {
		input.Skip = 0
	}
	input.Query = strings.TrimSpace(input.Query)
	input.PhoneID = strings.TrimSpace(input.PhoneID)
	return *input
}

// ToIndexParams converts USSDIndex to repositories.IndexParams
func (input *USSDIndex) ToIndexParams() repositories.IndexParams {
	return repositories.IndexParams{
		Skip:  input.Skip,
		Query: input.Query,
		Limit: input.Limit,
	}
}

// USSDDelete is the payload for deleting a USSD session
type USSDDelete struct {
	request
	USSDID string `json:"ussd_id" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
}

// USSDIDUuid returns the USSDID as uuid.UUID
func (input *USSDDelete) USSDIDUuid() string {
	return input.USSDID
}