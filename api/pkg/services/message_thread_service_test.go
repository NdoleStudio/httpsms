package services

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/stretchr/testify/assert"
)

func TestShouldCheckUnarchive(t *testing.T) {
	service := &MessageThreadService{}

	archived := &entities.MessageThread{IsArchived: true}
	notArchived := &entities.MessageThread{IsArchived: false}

	received := MessageThreadUpdateParams{Status: entities.MessageStatusReceived}
	sent := MessageThreadUpdateParams{Status: entities.MessageStatusSent}

	assert.True(t, service.shouldCheckUnarchive(archived, received), "archived + inbound -> consult phone setting")
	assert.False(t, service.shouldCheckUnarchive(archived, sent), "outbound status -> no check")
	assert.False(t, service.shouldCheckUnarchive(notArchived, received), "already unarchived -> no check")
	assert.False(t, service.shouldCheckUnarchive(notArchived, sent), "not archived + outbound -> no check")
}
