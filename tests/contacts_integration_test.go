package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type integrationContact struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Emails       []string          `json:"emails"`
	PhoneNumbers []string          `json:"phone_numbers"`
	Properties   map[string]string `json:"properties"`
}

type integrationContactsResponse struct {
	Data  []integrationContact `json:"data"`
	Total int64                `json:"total"`
}

type integrationContactResponse struct {
	Data integrationContact `json:"data"`
}

type integrationContactThread struct {
	ID             string              `json:"id"`
	Contact        string              `json:"contact"`
	ContactDetails *integrationContact `json:"contact_details"`
}

func createIntegrationContacts(
	ctx context.Context,
	t *testing.T,
	contacts []map[string]any,
) []integrationContact {
	t.Helper()

	var response integrationContactsResponse
	requestJSON(
		ctx,
		t,
		http.MethodPost,
		"/v1/contacts",
		userAPIKey,
		contacts,
		http.StatusCreated,
		&response,
	)
	require.Len(t, response.Data, len(contacts))
	return response.Data
}

func deleteIntegrationContact(ctx context.Context, t *testing.T, contactID string) {
	t.Helper()

	requestJSON(
		ctx,
		t,
		http.MethodDelete,
		"/v1/contacts/"+contactID,
		userAPIKey,
		nil,
		http.StatusNoContent,
		nil,
	)
}

func cleanupIntegrationContact(t *testing.T, contactID string) {
	t.Helper()

	request, err := http.NewRequest(
		http.MethodDelete,
		apiBaseURL+"/v1/contacts/"+contactID,
		nil,
	)
	require.NoError(t, err)
	request.Header.Set("x-api-key", userAPIKey)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Contains(
		t,
		[]int{http.StatusNoContent, http.StatusNotFound},
		response.StatusCode,
		"response: %s",
		string(responseBody),
	)
}

func listIntegrationContacts(
	ctx context.Context,
	t *testing.T,
	query string,
	skip int,
	limit int,
) integrationContactsResponse {
	t.Helper()

	path := fmt.Sprintf(
		"/v1/contacts?query=%s&skip=%d&limit=%d",
		url.QueryEscape(query),
		skip,
		limit,
	)
	var response integrationContactsResponse
	requestJSON(ctx, t, http.MethodGet, path, userAPIKey, nil, http.StatusOK, &response)
	return response
}

func uploadIntegrationContactsCSV(
	ctx context.Context,
	t *testing.T,
	contents string,
) integrationContactsResponse {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("document", "contacts.csv")
	require.NoError(t, err)
	_, err = part.Write([]byte(contents))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		apiBaseURL+"/v1/contacts/upload",
		&body,
	)
	require.NoError(t, err)
	request.Header.Set("x-api-key", userAPIKey)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode, "response: %s", string(responseBody))

	var result integrationContactsResponse
	require.NoError(t, json.Unmarshal(responseBody, &result))
	return result
}

func waitForThreadWithContact(
	ctx context.Context,
	t *testing.T,
	owner string,
	phoneNumber string,
	timeout time.Duration,
) integrationContactThread {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		path := fmt.Sprintf(
			"/v1/message-threads?owner=%s&contacts=true&skip=0&limit=20",
			url.QueryEscape(owner),
		)
		var response struct {
			Data []integrationContactThread `json:"data"`
		}
		requestJSON(ctx, t, http.MethodGet, path, userAPIKey, nil, http.StatusOK, &response)
		for _, thread := range response.Data {
			if thread.Contact == phoneNumber && thread.ContactDetails != nil {
				return thread
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("thread for %s did not include contact details within %v", phoneNumber, timeout)
	return integrationContactThread{}
}

func TestContacts_CRUDSearchAndPagination(t *testing.T) {
	ctx := context.Background()
	marker := strings.ToLower(uuid.New().String())
	firstPhone := randomPhoneNumber()
	secondPhone := randomPhoneNumber()

	created := createIntegrationContacts(ctx, t, []map[string]any{
		{
			"name":          "  Alice " + marker + "  ",
			"emails":        []string{" ALICE-" + marker + "@EXAMPLE.COM "},
			"phone_numbers": []string{strings.TrimPrefix(firstPhone, "+")},
			"properties":    map[string]string{"company": "Acme"},
		},
		{
			"name":          "Bob " + marker,
			"emails":        []string{"bob-" + marker + "@example.com"},
			"phone_numbers": []string{secondPhone},
		},
	})
	for _, contact := range created {
		contactID := contact.ID
		t.Cleanup(func() {
			cleanupIntegrationContact(t, contactID)
		})
	}

	require.Len(t, created, 2)
	assert.Equal(t, "Alice "+marker, created[0].Name)
	assert.Equal(t, []string{"alice-" + marker + "@example.com"}, created[0].Emails)
	assert.Equal(t, []string{firstPhone}, created[0].PhoneNumbers)
	assert.Equal(t, map[string]string{"company": "Acme"}, created[0].Properties)

	page := listIntegrationContacts(ctx, t, marker, 0, 1)
	assert.EqualValues(t, 2, page.Total)
	require.Len(t, page.Data, 1)

	emailMatch := listIntegrationContacts(ctx, t, "ALICE-"+marker+"@EXAMPLE.COM", 0, 20)
	assert.EqualValues(t, 1, emailMatch.Total)
	require.Len(t, emailMatch.Data, 1)
	assert.Equal(t, created[0].ID, emailMatch.Data[0].ID)

	var updated integrationContactResponse
	requestJSON(
		ctx,
		t,
		http.MethodPut,
		"/v1/contacts/"+created[0].ID,
		userAPIKey,
		map[string]any{
			"name":          "Updated " + marker,
			"emails":        []string{"updated-" + marker + "@example.com"},
			"phone_numbers": []string{firstPhone},
			"properties":    map[string]string{"company": "NdoleStudio"},
		},
		http.StatusOK,
		&updated,
	)
	assert.Equal(t, "Updated "+marker, updated.Data.Name)
	assert.Equal(t, []string{"updated-" + marker + "@example.com"}, updated.Data.Emails)
	assert.Equal(t, map[string]string{"company": "NdoleStudio"}, updated.Data.Properties)

	deleteIntegrationContact(ctx, t, created[0].ID)

	deletedMatch := listIntegrationContacts(ctx, t, "updated-"+marker+"@example.com", 0, 20)
	assert.Zero(t, deletedMatch.Total)
	assert.Empty(t, deletedMatch.Data)
}

func TestContacts_CSVUpload(t *testing.T) {
	ctx := context.Background()
	marker := strings.ToLower(uuid.New().String())
	phoneNumber := randomPhoneNumber()
	csv := fmt.Sprintf(
		"Name,Emails,PhoneNumbers\nCSV %s,\"FIRST-%s@EXAMPLE.COM;second-%s@example.com\",%s\n",
		marker,
		marker,
		marker,
		strings.TrimPrefix(phoneNumber, "+"),
	)

	response := uploadIntegrationContactsCSV(ctx, t, csv)
	require.Len(t, response.Data, 1)
	contact := response.Data[0]
	t.Cleanup(func() {
		cleanupIntegrationContact(t, contact.ID)
	})

	assert.Equal(t, "CSV "+marker, contact.Name)
	assert.Equal(t, []string{
		"first-" + marker + "@example.com",
		"second-" + marker + "@example.com",
	}, contact.Emails)
	assert.Equal(t, []string{phoneNumber}, contact.PhoneNumbers)

	listed := listIntegrationContacts(ctx, t, marker, 0, 20)
	assert.EqualValues(t, 1, listed.Total)
	require.Len(t, listed.Data, 1)
	assert.Equal(t, contact.ID, listed.Data[0].ID)
}

func TestContacts_MessageThreadsIncludeContactDetails(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)
	contactPhone := randomPhoneNumber()
	marker := strings.ToLower(uuid.New().String())

	created := createIntegrationContacts(ctx, t, []map[string]any{
		{
			"name":          "Thread Contact " + marker,
			"emails":        []string{"thread-" + marker + "@example.com"},
			"phone_numbers": []string{contactPhone},
			"properties":    map[string]string{"source": "integration-test"},
		},
	})
	contact := created[0]
	t.Cleanup(func() {
		cleanupIntegrationContact(t, contact.ID)
	})

	messageID := receiveInbound(
		ctx,
		t,
		phone.PhoneAPIKey,
		contactPhone,
		phone.PhoneNumber,
		"Contact enrichment "+marker,
		time.Now(),
	)
	pollMessageStatus(ctx, t, messageID, "received", 15*time.Second)

	thread := waitForThreadWithContact(ctx, t, phone.PhoneNumber, contactPhone, 20*time.Second)
	require.NotNil(t, thread.ContactDetails)
	assert.Equal(t, contact.ID, thread.ContactDetails.ID)
	assert.Equal(t, contact.Name, thread.ContactDetails.Name)
	assert.Equal(t, contact.PhoneNumbers, thread.ContactDetails.PhoneNumbers)
	assert.Equal(t, contact.Properties, thread.ContactDetails.Properties)
}
