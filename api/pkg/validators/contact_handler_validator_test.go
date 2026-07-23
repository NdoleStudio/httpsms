package validators

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func newContactValidator() *ContactHandlerValidator {
	return &ContactHandlerValidator{}
}

func newContactUploadValidator() *ContactHandlerValidator {
	logger := &contactValidatorNoopLogger{}
	return NewContactHandlerValidator(logger, telemetry.NewOtelLogger("test", logger))
}

func TestContactValidator_ValidateStore_ValidOneAndMany(t *testing.T) {
	validator := newContactValidator()

	tests := []struct {
		name    string
		request requests.ContactStoreRequest
	}{
		{
			name: "one contact",
			request: requests.ContactStoreRequest{Contacts: []requests.ContactItem{{
				Name:         "Alice",
				PhoneNumbers: []string{"+18005550199"},
				Emails:       []string{"alice@example.com"},
			}}},
		},
		{
			name: "many contacts",
			request: requests.ContactStoreRequest{Contacts: []requests.ContactItem{
				{
					Name:         "Alice",
					PhoneNumbers: []string{"+18005550199", "+18005550100"},
					Emails:       []string{"alice@example.com", "alice.work@example.com"},
				},
				{
					Name:         "Bob",
					PhoneNumbers: []string{"+14155552671"},
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.ValidateStore(context.Background(), tt.request)

			assert.Empty(t, errs)
		})
	}
}

func TestContactValidator_ValidateStore_MissingName(t *testing.T) {
	validator := newContactValidator()

	errs := validator.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: []requests.ContactItem{{
		PhoneNumbers: []string{"+18005550199"},
	}}})

	require.NotEmpty(t, errs.Get("contacts"))
	assert.Contains(t, errs["contacts"][0], "Contact [1]")
	assert.Contains(t, errs["contacts"][0], "name")
}

func TestContactValidator_ValidateStore_NoPhoneNumber(t *testing.T) {
	validator := newContactValidator()

	errs := validator.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: []requests.ContactItem{{
		Name: "Alice",
	}}})

	require.NotEmpty(t, errs.Get("contacts"))
	assert.Contains(t, errs["contacts"][0], "Contact [1]")
	assert.Contains(t, errs["contacts"][0], "phone number")
}

func TestContactValidator_ValidateStore_EmptyBatch(t *testing.T) {
	validator := newContactValidator()

	errs := validator.ValidateStore(context.Background(), requests.ContactStoreRequest{})

	require.NotEmpty(t, errs.Get("contacts"))
	assert.Contains(t, errs["contacts"][0], "at least one contact")
}

func TestContactValidator_ValidateStore_InvalidPhoneNumber(t *testing.T) {
	validator := newContactValidator()

	errs := validator.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: []requests.ContactItem{{
		Name:         "Alice",
		PhoneNumbers: []string{"not-a-number"},
	}}})

	require.NotEmpty(t, errs.Get("contacts"))
	assert.Contains(t, errs["contacts"][0], "Contact [1]")
	assert.Contains(t, errs["contacts"][0], "not-a-number")
}

func TestContactValidator_ValidateStore_InvalidEmail(t *testing.T) {
	validator := newContactValidator()

	errs := validator.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: []requests.ContactItem{{
		Name:         "Alice",
		PhoneNumbers: []string{"+18005550199"},
		Emails:       []string{"not-an-email"},
	}}})

	require.NotEmpty(t, errs.Get("contacts"))
	assert.Contains(t, errs["contacts"][0], "Contact [1]")
	assert.Contains(t, errs["contacts"][0], "not-an-email")
}

func TestContactValidator_ValidateStore_RejectsEmptyPhoneAndEmailElements(t *testing.T) {
	validator := newContactValidator()

	errs := validator.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: []requests.ContactItem{{
		Name:         "Alice",
		PhoneNumbers: []string{"+18005550199", " "},
		Emails:       []string{" "},
	}}})

	contactsErrors := strings.Join(errs["contacts"], "\n")
	assert.Contains(t, contactsErrors, "phone number")
	assert.Contains(t, contactsErrors, "email")
}

func TestContactValidator_ValidateStore_RejectsMoreThan1000Contacts(t *testing.T) {
	validator := newContactValidator()
	contacts := make([]requests.ContactItem, 1001)
	for index := range contacts {
		contacts[index] = requests.ContactItem{Name: "Alice", PhoneNumbers: []string{"+18005550199"}}
	}

	errs := validator.ValidateStore(context.Background(), requests.ContactStoreRequest{Contacts: contacts})

	require.NotEmpty(t, errs.Get("contacts"))
	assert.Contains(t, errs["contacts"][0], "1000")
}

func TestContactValidator_ValidateUpdate_ValidAndInvalid(t *testing.T) {
	validator := newContactValidator()

	validErrs := validator.ValidateUpdate(context.Background(), requests.ContactUpdateRequest{
		Name:         "Alice",
		PhoneNumbers: []string{"+18005550199"},
		Emails:       []string{"alice@example.com"},
	})
	assert.Empty(t, validErrs)

	invalidErrs := validator.ValidateUpdate(context.Background(), requests.ContactUpdateRequest{
		Name:         "",
		PhoneNumbers: []string{"not-a-number"},
		Emails:       []string{"not-an-email"},
	})
	assert.NotEmpty(t, invalidErrs.Get("contacts"))
}

func TestContactValidator_ValidateIndex_Bounds(t *testing.T) {
	validator := newContactValidator()

	validErrs := validator.ValidateIndex(context.Background(), requests.ContactIndex{Skip: "0", Limit: "100", Query: strings.Repeat("a", 100)})
	assert.Empty(t, validErrs)

	tests := []struct {
		name    string
		request requests.ContactIndex
		key     string
	}{
		{name: "limit too low", request: requests.ContactIndex{Skip: "0", Limit: "0"}, key: "limit"},
		{name: "limit too high", request: requests.ContactIndex{Skip: "0", Limit: "101"}, key: "limit"},
		{name: "skip negative", request: requests.ContactIndex{Skip: "-1", Limit: "20"}, key: "skip"},
		{name: "query too long", request: requests.ContactIndex{Skip: "0", Limit: "20", Query: strings.Repeat("a", 101)}, key: "query"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.ValidateIndex(context.Background(), tt.request)

			assert.NotEmpty(t, errs.Get(tt.key))
		})
	}
}

func TestContactValidator_ValidateUpload_ParsesValidCSVWithQuotedMultiValueCells(t *testing.T) {
	validator := newContactUploadValidator()
	header := multipartFileHeader(t, "contacts.csv", "text/csv", strings.Join([]string{
		"Name,Emails,PhoneNumbers",
		`Alice,"alice@example.com,alice.work@example.com","+18005550199,+18005550100"`,
		`Bob,bob@example.com;bob.work@example.com,+14155552671;+14155552672`,
	}, "\n"))

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	require.Empty(t, errs)
	require.Len(t, items, 2)
	assert.Equal(t, "Alice", items[0].Name)
	assert.Equal(t, []string{"alice@example.com", "alice.work@example.com"}, items[0].Emails)
	assert.Equal(t, []string{"+18005550199", "+18005550100"}, items[0].PhoneNumbers)
	assert.Equal(t, "Bob", items[1].Name)
	assert.Equal(t, []string{"bob@example.com", "bob.work@example.com"}, items[1].Emails)
	assert.Equal(t, []string{"+14155552671", "+14155552672"}, items[1].PhoneNumbers)
}

func TestContactValidator_ValidateUpload_RejectsNonCSVExtensionAndMIME(t *testing.T) {
	validator := newContactUploadValidator()

	tests := []struct {
		name        string
		filename    string
		contentType string
	}{
		{name: "xlsx extension", filename: "contacts.xlsx", contentType: "text/csv"},
		{name: "xlsx mime", filename: "contacts.csv", contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := multipartFileHeader(t, tt.filename, tt.contentType, "Name,Emails,PhoneNumbers\nAlice,alice@example.com,+18005550199")

			items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

			assert.Nil(t, items)
			assert.NotEmpty(t, errs.Get("document"))
		})
	}
}

func TestContactValidator_ValidateUpload_AcceptsCSVWithOctetStreamMIME(t *testing.T) {
	validator := newContactUploadValidator()
	header := multipartFileHeader(t, "contacts.csv", "application/octet-stream", "Name,Emails,PhoneNumbers\nAlice,alice@example.com,+18005550199")

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	require.Empty(t, errs)
	require.Len(t, items, 1)
	assert.Equal(t, "Alice", items[0].Name)
}

func TestContactValidator_ValidateUpload_AcceptsCSVWithVndMsExcelMIME(t *testing.T) {
	validator := newContactUploadValidator()
	header := multipartFileHeader(t, "contacts.csv", "application/vnd.ms-excel", "Name,Emails,PhoneNumbers\nAlice,alice@example.com,+18005550199")

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	require.Empty(t, errs)
	require.Len(t, items, 1)
	assert.Equal(t, "Alice", items[0].Name)
}

func TestContactValidator_ValidateUpload_AcceptsCSVWithEmptyMIME(t *testing.T) {
	validator := newContactUploadValidator()
	header := multipartFileHeader(t, "contacts.csv", "", "Name,Emails,PhoneNumbers\nAlice,alice@example.com,+18005550199")

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	require.Empty(t, errs)
	require.Len(t, items, 1)
	assert.Equal(t, "Alice", items[0].Name)
}

func TestContactValidator_ValidateUpload_RejectsXlsxDespiteSpreadsheetMIME(t *testing.T) {
	validator := newContactUploadValidator()
	header := multipartFileHeader(t, "contacts.xlsx", "application/vnd.ms-excel", "Name,Emails,PhoneNumbers\nAlice,alice@example.com,+18005550199")

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	assert.Nil(t, items)
	require.NotEmpty(t, errs.Get("document"))
	assert.Contains(t, errs["document"][0], "not a valid CSV file")
}

func TestContactValidator_ValidateUpload_RejectsCSVWithUnrelatedMIME(t *testing.T) {
	validator := newContactUploadValidator()
	header := multipartFileHeader(t, "contacts.csv", "image/png", "Name,Emails,PhoneNumbers\nAlice,alice@example.com,+18005550199")

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	assert.Nil(t, items)
	require.NotEmpty(t, errs.Get("document"))
	assert.Contains(t, errs["document"][0], "not a valid CSV file")
}

func TestContactValidator_ValidateUpload_RejectsOversizedFile(t *testing.T) {
	validator := newContactUploadValidator()
	header := multipartFileHeader(t, "contacts.csv", "text/csv", strings.Repeat("a", 500*1024+1))

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	assert.Nil(t, items)
	require.NotEmpty(t, errs.Get("document"))
	assert.Contains(t, errs["document"][0], "500 KB")
}

func TestContactValidator_ValidateUpload_RejectsMalformedCSV(t *testing.T) {
	validator := newContactUploadValidator()
	header := multipartFileHeader(t, "contacts.csv", "text/csv", "Name,Emails,PhoneNumbers\nAlice,\"alice@example.com,+18005550199")

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	assert.Nil(t, items)
	require.NotEmpty(t, errs.Get("document"))
	assert.Contains(t, errs["document"][0], "Cannot parse")
}

func TestContactValidator_ValidateUpload_RejectsMissingRequiredColumns(t *testing.T) {
	validator := newContactUploadValidator()
	header := multipartFileHeader(t, "contacts.csv", "text/csv", "Name,Emails\nAlice,alice@example.com")

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	assert.Nil(t, items)
	require.NotEmpty(t, errs.Get("document"))
	assert.Contains(t, errs["document"][0], "Name, Emails, PhoneNumbers")
}

func TestContactValidator_ValidateUpload_RejectsMoreThan1000Rows(t *testing.T) {
	validator := newContactUploadValidator()
	rows := []string{"Name,Emails,PhoneNumbers"}
	for range 1001 {
		rows = append(rows, "Alice,alice@example.com,+18005550199")
	}
	header := multipartFileHeader(t, "contacts.csv", "text/csv", strings.Join(rows, "\n"))

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	assert.Nil(t, items)
	require.NotEmpty(t, errs.Get("document"))
	assert.Contains(t, errs["document"][0], "1000")
}

func TestContactValidator_ValidateUpload_ReturnsRowIndexedValidationErrors(t *testing.T) {
	validator := newContactUploadValidator()
	header := multipartFileHeader(t, "contacts.csv", "text/csv", strings.Join([]string{
		"Name,Emails,PhoneNumbers",
		",alice@example.com,+18005550199",
		"Alice,,",
		"Bob,not-an-email,not-a-number",
	}, "\n"))

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	require.Len(t, items, 3)
	documentErrors := strings.Join(errs["document"], "\n")
	assert.Contains(t, documentErrors, "Row [2]")
	assert.Contains(t, documentErrors, "Row [3]")
	assert.Contains(t, documentErrors, "Row [4]")
	assert.Contains(t, documentErrors, "not-an-email")
	assert.Contains(t, documentErrors, "not-a-number")
}

func TestContactValidator_ValidateUpload_SanitizesItemsLikeJSONBeforeValidation(t *testing.T) {
	validator := newContactUploadValidator()
	// The phone number lacks a leading "+" and the email has mixed case and
	// surrounding whitespace. The JSON create path sanitizes these before
	// validation, so the CSV path must accept and normalize them identically.
	header := multipartFileHeader(t, "contacts.csv", "text/csv", strings.Join([]string{
		"Name,Emails,PhoneNumbers",
		"Alice, Alice@Example.com ,18005550199",
	}, "\n"))

	items, errs := validator.ValidateUpload(context.Background(), entities.UserID("user-1"), header)

	require.Empty(t, errs, "sanitized CSV rows must pass validation like the JSON path")
	require.Len(t, items, 1)
	assert.Equal(t, "Alice", items[0].Name)
	assert.Equal(t, []string{"+18005550199"}, items[0].PhoneNumbers)
	assert.Equal(t, []string{"alice@example.com"}, items[0].Emails)
}

func multipartFileHeader(t *testing.T, filename string, contentType string, content string) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="document"; filename="%s"`, filename))
	header.Set("Content-Type", contentType)

	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, request.ParseMultipartForm(int64(body.Len()+1024)))
	files := request.MultipartForm.File["document"]
	require.Len(t, files, 1)

	return files[0]
}

type contactValidatorNoopLogger struct{}

var _ telemetry.Logger = (*contactValidatorNoopLogger)(nil)

func (logger *contactValidatorNoopLogger) Error(_ error)                         {}
func (logger *contactValidatorNoopLogger) WithService(_ string) telemetry.Logger { return logger }

func (logger *contactValidatorNoopLogger) WithString(_, _ string) telemetry.Logger { return logger }

func (logger *contactValidatorNoopLogger) WithSpan(_ trace.SpanContext) telemetry.Logger {
	return logger
}
func (logger *contactValidatorNoopLogger) Trace(_ string)                    {}
func (logger *contactValidatorNoopLogger) Info(_ string)                     {}
func (logger *contactValidatorNoopLogger) Warn(_ error)                      {}
func (logger *contactValidatorNoopLogger) Debug(_ string)                    {}
func (logger *contactValidatorNoopLogger) Fatal(_ error)                     {}
func (logger *contactValidatorNoopLogger) Printf(_ string, _ ...interface{}) {}
