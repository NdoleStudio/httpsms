package tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	httpsms "github.com/NdoleStudio/httpsms-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type phoneThreadSettings struct {
	ArchiveNewOutgoingMessageThreads bool `json:"archive_new_outgoing_message_threads"`
	UnarchiveThread                  bool `json:"unarchive_thread"`
}

func updatePhoneThreadSettings(ctx context.Context, t *testing.T, phone testPhone, settings map[string]bool) phoneThreadSettings {
	t.Helper()

	payload := map[string]any{
		"phone_number": phone.PhoneNumber,
		"fcm_token":    phone.FcmToken,
		"sim":          "SIM1",
	}
	for key, value := range settings {
		payload[key] = value
	}
	var response struct {
		Data phoneThreadSettings `json:"data"`
	}
	requestJSON(ctx, t, http.MethodPut, "/v1/phones", userAPIKey, payload, http.StatusOK, &response)
	return response.Data
}

func sendThreadMessage(ctx context.Context, t *testing.T, phone testPhone, contact, content string) {
	t.Helper()

	_, response, err := newAPIClient().Messages.Send(ctx, &httpsms.MessageSendParams{
		From:    phone.PhoneNumber,
		To:      contact,
		Content: content,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
}

func TestArchiveNewOutgoingMessageThreadsOnFirstRegistration(t *testing.T) {
	ctx := context.Background()
	phone := testPhone{
		PhoneNumber: randomPhoneNumber(),
		FcmToken:    "fcm-" + uuid.NewString(),
	}
	var response struct {
		Data phoneThreadSettings `json:"data"`
	}
	requestJSON(ctx, t, http.MethodPut, "/v1/phones", userAPIKey, map[string]any{
		"phone_number":                         phone.PhoneNumber,
		"fcm_token":                            phone.FcmToken,
		"sim":                                  "SIM1",
		"archive_new_outgoing_message_threads": true,
	}, http.StatusOK, &response)
	require.True(t, response.Data.ArchiveNewOutgoingMessageThreads)

	contact := "+18005540100"
	sendThreadMessage(ctx, t, phone, contact, "First registration archives outgoing")
	require.NotNil(t, waitForThread(ctx, t, phone.PhoneNumber, contact, true, "First registration archives outgoing", 20*time.Second))
	assert.Nil(t, findThreadByContact(fetchThreads(ctx, t, phone.PhoneNumber, false), contact))
}

func TestArchiveNewOutgoingMessageThreadsPerPhoneAndDirection(t *testing.T) {
	ctx := context.Background()
	enabledPhone := setupPhone(ctx, t, 60)
	defaultPhone := setupPhone(ctx, t, 60)
	require.NotEqual(t, enabledPhone.PhoneNumber, defaultPhone.PhoneNumber)

	existingContact := randomPhoneNumber()
	receiveInbound(ctx, t, enabledPhone.PhoneAPIKey, existingContact, enabledPhone.PhoneNumber, "Existing inbox conversation", time.Now())
	require.NotNil(t, waitForThread(ctx, t, enabledPhone.PhoneNumber, existingContact, false, "Existing inbox conversation", 20*time.Second))

	settings := updatePhoneThreadSettings(ctx, t, enabledPhone, map[string]bool{"archive_new_outgoing_message_threads": true})
	require.True(t, settings.ArchiveNewOutgoingMessageThreads)
	assert.False(t, updatePhoneThreadSettings(ctx, t, defaultPhone, nil).ArchiveNewOutgoingMessageThreads)

	contact := randomPhoneNumber()
	sendThreadMessage(ctx, t, enabledPhone, contact, "New outgoing archived")
	archived := waitForThread(ctx, t, enabledPhone.PhoneNumber, contact, true, "New outgoing archived", 20*time.Second)
	require.NotNil(t, archived)
	assert.True(t, archived.IsArchived)
	assert.Nil(t, findThreadByContact(fetchThreads(ctx, t, enabledPhone.PhoneNumber, false), contact))

	sendThreadMessage(ctx, t, defaultPhone, contact, "Other phone stays in inbox")
	require.NotNil(t, waitForThread(ctx, t, defaultPhone.PhoneNumber, contact, false, "Other phone stays in inbox", 20*time.Second))

	sendThreadMessage(ctx, t, enabledPhone, existingContact, "Existing conversation stays in inbox")
	require.NotNil(t, waitForThread(ctx, t, enabledPhone.PhoneNumber, existingContact, false, "Existing conversation stays in inbox", 20*time.Second))

	inboundContact := randomPhoneNumber()
	receiveInbound(ctx, t, enabledPhone.PhoneAPIKey, inboundContact, enabledPhone.PhoneNumber, "New incoming stays in inbox", time.Now())
	require.NotNil(t, waitForThread(ctx, t, enabledPhone.PhoneNumber, inboundContact, false, "New incoming stays in inbox", 20*time.Second))
}

func TestArchiveNewOutgoingMessageThreadsToggleAndPartialUpdates(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)

	require.True(t, updatePhoneThreadSettings(ctx, t, phone, map[string]bool{"archive_new_outgoing_message_threads": true}).ArchiveNewOutgoingMessageThreads)
	settings := updatePhoneThreadSettings(ctx, t, phone, map[string]bool{"unarchive_thread": true})
	assert.True(t, settings.ArchiveNewOutgoingMessageThreads, "omitting the setting must preserve it")
	assert.True(t, settings.UnarchiveThread)

	archivedContact := randomPhoneNumber()
	sendThreadMessage(ctx, t, phone, archivedContact, "Archived before disabling")
	require.NotNil(t, waitForThread(ctx, t, phone.PhoneNumber, archivedContact, true, "Archived before disabling", 20*time.Second))

	settings = updatePhoneThreadSettings(ctx, t, phone, map[string]bool{"archive_new_outgoing_message_threads": false})
	require.False(t, settings.ArchiveNewOutgoingMessageThreads)
	assert.True(t, settings.UnarchiveThread, "the two settings must be independent")

	inboxContact := randomPhoneNumber()
	sendThreadMessage(ctx, t, phone, inboxContact, "Inbox after disabling")
	require.NotNil(t, waitForThread(ctx, t, phone.PhoneNumber, inboxContact, false, "Inbox after disabling", 20*time.Second))
	assert.NotNil(t, findThreadByContact(fetchThreads(ctx, t, phone.PhoneNumber, true), archivedContact), "disabling must not change existing archived threads")
}

func TestArchiveNewOutgoingMessageThreadsReplyRespectsUnarchiveSetting(t *testing.T) {
	for _, test := range []struct {
		name      string
		unarchive bool
	}{
		{name: "reply remains archived", unarchive: false},
		{name: "reply returns to inbox", unarchive: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			phone := setupPhone(ctx, t, 60)
			updatePhoneThreadSettings(ctx, t, phone, map[string]bool{
				"archive_new_outgoing_message_threads": true,
				"unarchive_thread":                     test.unarchive,
			})
			contact := randomPhoneNumber()
			sendThreadMessage(ctx, t, phone, contact, "Outgoing starts archived")
			require.NotNil(t, waitForThread(ctx, t, phone.PhoneNumber, contact, true, "Outgoing starts archived", 20*time.Second))

			receiveInbound(ctx, t, phone.PhoneAPIKey, contact, phone.PhoneNumber, "Incoming reply", time.Now())

			thread := waitForThread(ctx, t, phone.PhoneNumber, contact, !test.unarchive, "Incoming reply", 20*time.Second)
			require.NotNil(t, thread)
			assert.Equal(t, !test.unarchive, thread.IsArchived)
			assert.Nil(t, findThreadByContact(fetchThreads(ctx, t, phone.PhoneNumber, test.unarchive), contact))
		})
	}
}
