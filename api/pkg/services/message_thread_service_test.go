package services

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/stretchr/testify/assert"
)

func TestShouldUnarchive(t *testing.T) {
	service := &MessageThreadService{}

	archived := &entities.MessageThread{IsArchived: true}
	notArchived := &entities.MessageThread{IsArchived: false}

	received := MessageThreadUpdateParams{Status: entities.MessageStatusReceived, UnarchiveThread: true}
	receivedFlagOff := MessageThreadUpdateParams{Status: entities.MessageStatusReceived, UnarchiveThread: false}
	sentFlagOn := MessageThreadUpdateParams{Status: entities.MessageStatusSent, UnarchiveThread: true}

	assert.True(t, service.shouldUnarchive(archived, received), "archived + inbound + flag on -> unarchive")
	assert.False(t, service.shouldUnarchive(archived, receivedFlagOff), "flag off -> no unarchive")
	assert.False(t, service.shouldUnarchive(archived, sentFlagOn), "outbound status -> no unarchive")
	assert.False(t, service.shouldUnarchive(notArchived, received), "already unarchived -> no change")
}
