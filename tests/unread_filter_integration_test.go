package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fetchFilteredThreads(ctx context.Context, t *testing.T, owner string, archived, unreadOnly bool, limit, skip int) []integrationThread {
	t.Helper()

	query := url.Values{
		"owner":       {owner},
		"is_archived": {fmt.Sprint(archived)},
		"unread_only": {fmt.Sprint(unreadOnly)},
		"limit":       {fmt.Sprint(limit)},
		"skip":        {fmt.Sprint(skip)},
	}
	var response struct {
		Data []integrationThread `json:"data"`
	}
	requestJSON(ctx, t, http.MethodGet, "/v1/message-threads?"+query.Encode(), userAPIKey, nil, http.StatusOK, &response)
	return response.Data
}

func TestMessageThreadUnreadFilter(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)
	otherPhone := setupPhone(ctx, t, 60)
	require.NotEqual(t, phone.PhoneNumber, otherPhone.PhoneNumber)
	unreadContact := randomPhoneNumber()
	readContact := randomPhoneNumber()
	require.NotEqual(t, unreadContact, readContact)

	receiveInbound(ctx, t, phone.PhoneAPIKey, unreadContact, phone.PhoneNumber, "Older unread message", time.Now().Add(-2*time.Minute))
	unread := waitForThread(ctx, t, phone.PhoneNumber, unreadContact, false, "Older unread message", 20*time.Second)
	require.NotNil(t, unread)
	require.Equal(t, uint(1), unread.UnreadCount)

	receiveInbound(ctx, t, phone.PhoneAPIKey, readContact, phone.PhoneNumber, "Recent read message", time.Now().Add(-time.Minute))
	read := waitForThread(ctx, t, phone.PhoneNumber, readContact, false, "Recent read message", 20*time.Second)
	require.NotNil(t, read)
	clearMessageThreadUnreadCount(ctx, t, read.ID)

	receiveInbound(ctx, t, otherPhone.PhoneAPIKey, unreadContact, otherPhone.PhoneNumber, "Other phone unread", time.Now())
	require.NotNil(t, waitForThread(ctx, t, otherPhone.PhoneNumber, unreadContact, false, "Other phone unread", 20*time.Second))

	all := fetchThreads(ctx, t, phone.PhoneNumber, false)
	require.Len(t, all, 2, "All defaults to including read and unread conversations for this phone")
	assert.Equal(t, []string{read.ID, unread.ID}, []string{all[0].ID, all[1].ID})

	filtered := fetchFilteredThreads(ctx, t, phone.PhoneNumber, false, true, 1, 0)
	require.Len(t, filtered, 1, "the unread condition must run before the result limit")
	assert.Equal(t, unread.ID, filtered[0].ID)
	assert.Positive(t, filtered[0].UnreadCount)
	assert.Empty(t, fetchFilteredThreads(ctx, t, phone.PhoneNumber, false, true, 1, 1), "offset must apply to the filtered result")

	all = fetchFilteredThreads(ctx, t, phone.PhoneNumber, false, false, 1, 0)
	require.Len(t, all, 1)
	assert.Equal(t, read.ID, all[0].ID, "All must restore the most recent read conversation")

	archiveThread(ctx, t, unread.ID)
	assert.Empty(t, fetchFilteredThreads(ctx, t, phone.PhoneNumber, false, true, 20, 0), "archived conversations must not appear in the inbox unread filter")
	filtered = fetchFilteredThreads(ctx, t, phone.PhoneNumber, true, true, 20, 0)
	require.Len(t, filtered, 1)
	assert.Equal(t, unread.ID, filtered[0].ID)
	assert.True(t, filtered[0].IsArchived)

	clearMessageThreadUnreadCount(ctx, t, unread.ID)
	assert.Empty(t, fetchFilteredThreads(ctx, t, phone.PhoneNumber, true, true, 20, 0), "marking a conversation read removes it from Unread")
	all = fetchFilteredThreads(ctx, t, phone.PhoneNumber, true, false, 20, 0)
	require.Len(t, all, 1, "marking read must not delete or unarchive the conversation")
	assert.Equal(t, unread.ID, all[0].ID)
	assert.Zero(t, all[0].UnreadCount)
	assert.True(t, all[0].IsArchived)
}

func TestMessageThreadUnreadFilterRejectsInvalidValue(t *testing.T) {
	query := url.Values{
		"owner":       {randomPhoneNumber()},
		"unread_only": {"invalid"},
	}
	requestJSON(context.Background(), t, http.MethodGet, "/v1/message-threads?"+query.Encode(), userAPIKey, nil, http.StatusUnprocessableEntity, nil)
}
