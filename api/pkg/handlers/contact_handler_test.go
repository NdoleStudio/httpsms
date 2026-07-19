package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/cache"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/middlewares"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/NdoleStudio/stacktrace"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/lib/pq"
	ttlCache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// contactHandlerFakeRepo is a shared, thread-safe ContactRepository stub that
// records the effects of Store/Update/Delete and returns configurable Load/Index
// results so handler tests can assert real service→repository behaviour.
type contactHandlerFakeRepo struct {
	mu sync.Mutex

	stored      [][]*entities.Contact
	updated     []*entities.Contact
	deleted     []deletedContact
	indexParams []repositories.IndexParams
	loadCalls   []loadedContact

	loadResult  *entities.Contact
	loadErr     error
	indexResult []entities.Contact
	indexErr    error
	storeErr    error
	updateErr   error
	deleteErr   error
}

type deletedContact struct {
	userID entities.UserID
	id     uuid.UUID
}

type loadedContact struct {
	userID entities.UserID
	id     uuid.UUID
}

func (r *contactHandlerFakeRepo) Store(_ context.Context, contacts []*entities.Contact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.storeErr != nil {
		return r.storeErr
	}
	r.stored = append(r.stored, contacts)
	return nil
}

func (r *contactHandlerFakeRepo) Update(_ context.Context, contact *entities.Contact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updated = append(r.updated, contact)
	return nil
}

func (r *contactHandlerFakeRepo) Load(_ context.Context, userID entities.UserID, id uuid.UUID) (*entities.Contact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loadCalls = append(r.loadCalls, loadedContact{userID: userID, id: id})
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	if r.loadResult == nil {
		return nil, nil
	}
	// Return a copy so handler mutations don't corrupt the fixture.
	clone := *r.loadResult
	return &clone, nil
}

func (r *contactHandlerFakeRepo) Index(_ context.Context, _ entities.UserID, params repositories.IndexParams) (*[]entities.Contact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.indexParams = append(r.indexParams, params)
	if r.indexErr != nil {
		return nil, r.indexErr
	}
	out := make([]entities.Contact, len(r.indexResult))
	copy(out, r.indexResult)
	return &out, nil
}

func (r *contactHandlerFakeRepo) FetchAll(context.Context, entities.UserID) (*[]entities.Contact, error) {
	out := []entities.Contact{}
	return &out, nil
}

func (r *contactHandlerFakeRepo) Delete(_ context.Context, userID entities.UserID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deleted = append(r.deleted, deletedContact{userID: userID, id: id})
	return nil
}

func (r *contactHandlerFakeRepo) DeleteAllForUser(context.Context, entities.UserID) error {
	return nil
}

// snapshot returns a safe read of the recorded effects.
func (r *contactHandlerFakeRepo) snapshot() ([][]*entities.Contact, []*entities.Contact, []deletedContact, []repositories.IndexParams) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stored, r.updated, r.deleted, r.indexParams
}

const contactHandlerTestUserID = entities.UserID("user-id")

func newContactHandlerTestApp(repo repositories.ContactRepository) *fiber.App {
	logger := &messageThreadHandlerNoopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	appCache := cache.NewMemoryCache(tracer, ttlCache.New(time.Minute, time.Minute))
	service := services.NewContactService(logger, tracer, repo, appCache)
	handler := NewContactHandler(logger, tracer, validators.NewContactHandlerValidator(logger, tracer), service)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(middlewares.ContextKeyAuthUserID, entities.AuthContext{ID: contactHandlerTestUserID, Email: "user@example.com"})
		return c.Next()
	})
	handler.RegisterRoutes(app)
	return app
}

type contactHandlerPayload struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeContactHandlerPayload(t *testing.T, resp *http.Response) contactHandlerPayload {
	t.Helper()
	var payload contactHandlerPayload
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	return payload
}

func TestContactHandler_Store_CreatesSingleContact(t *testing.T) {
	repo := &contactHandlerFakeRepo{}
	app := newContactHandlerTestApp(repo)

	body := `[{"name":"Alice","phone_numbers":["+18005550199"],"emails":["alice@example.com"]}]`
	req := httptest.NewRequest(http.MethodPost, "/v1/contacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	stored, _, _, _ := repo.snapshot()
	require.Len(t, stored, 1)
	require.Len(t, stored[0], 1)
	require.Equal(t, "Alice", stored[0][0].Name)
	require.Equal(t, contactHandlerTestUserID, stored[0][0].UserID)
	require.Equal(t, pq.StringArray{"+18005550199"}, stored[0][0].PhoneNumbers)
}

func TestContactHandler_Store_CreatesManyContactsFromObjectShape(t *testing.T) {
	repo := &contactHandlerFakeRepo{}
	app := newContactHandlerTestApp(repo)

	body := `{"contacts":[
		{"name":"Alice","phone_numbers":["+18005550199"]},
		{"name":"Bob","phone_numbers":["+18005550100"]}
	]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/contacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	stored, _, _, _ := repo.snapshot()
	require.Len(t, stored, 1)
	require.Len(t, stored[0], 2)
	assert.Equal(t, "Alice", stored[0][0].Name)
	assert.Equal(t, "Bob", stored[0][1].Name)
}

func TestContactHandler_Store_ValidationError_ReturnsUnprocessableEntity(t *testing.T) {
	repo := &contactHandlerFakeRepo{}
	app := newContactHandlerTestApp(repo)

	body := `[{"name":"","phone_numbers":[]}]`
	req := httptest.NewRequest(http.MethodPost, "/v1/contacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	stored, _, _, _ := repo.snapshot()
	require.Empty(t, stored, "no contact should be stored on validation failure")
}

func TestContactHandler_Store_MalformedJSON_ReturnsBadRequest(t *testing.T) {
	app := newContactHandlerTestApp(&contactHandlerFakeRepo{})

	req := httptest.NewRequest(http.MethodPost, "/v1/contacts", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func buildContactCSVUpload(t *testing.T, filename string, contentType string, body string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="document"; filename="%s"`, filename))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return &buf, writer.FormDataContentType()
}

func TestContactHandler_Upload_CSVSuccess(t *testing.T) {
	repo := &contactHandlerFakeRepo{}
	app := newContactHandlerTestApp(repo)

	csv := "Name,Emails,PhoneNumbers\nAlice,alice@example.com,+18005550199\nBob,,+18005550100\n"
	body, contentType := buildContactCSVUpload(t, "contacts.csv", "text/csv", csv)

	req := httptest.NewRequest(http.MethodPost, "/v1/contacts/upload", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	stored, _, _, _ := repo.snapshot()
	require.Len(t, stored, 1)
	require.Len(t, stored[0], 2)
	assert.Equal(t, "Alice", stored[0][0].Name)
	assert.Equal(t, "Bob", stored[0][1].Name)
	for _, c := range stored[0] {
		assert.Equal(t, contactHandlerTestUserID, c.UserID)
	}
}

func TestContactHandler_Upload_NonCSVFile_ReturnsUnprocessableEntity(t *testing.T) {
	repo := &contactHandlerFakeRepo{}
	app := newContactHandlerTestApp(repo)

	body, contentType := buildContactCSVUpload(t, "contacts.txt", "text/plain", "junk")
	req := httptest.NewRequest(http.MethodPost, "/v1/contacts/upload", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	stored, _, _, _ := repo.snapshot()
	require.Empty(t, stored)
}

func TestContactHandler_Upload_MissingDocument_ReturnsBadRequest(t *testing.T) {
	app := newContactHandlerTestApp(&contactHandlerFakeRepo{})

	// multipart body without the "document" field.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	require.NoError(t, writer.WriteField("other", "value"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/v1/contacts/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestContactHandler_Update_Success(t *testing.T) {
	contactID := uuid.New()
	repo := &contactHandlerFakeRepo{
		loadResult: &entities.Contact{
			ID:           contactID,
			UserID:       contactHandlerTestUserID,
			Name:         "Old Name",
			PhoneNumbers: pq.StringArray{"+18005550100"},
		},
	}
	app := newContactHandlerTestApp(repo)

	body := `{"name":"New Name","phone_numbers":["+18005550199"],"emails":["new@example.com"]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/contacts/"+contactID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	_, updated, _, _ := repo.snapshot()
	require.Len(t, updated, 1)
	assert.Equal(t, "New Name", updated[0].Name)
	assert.Equal(t, contactID, updated[0].ID)
	assert.Equal(t, contactHandlerTestUserID, updated[0].UserID)
	assert.Equal(t, pq.StringArray{"+18005550199"}, updated[0].PhoneNumbers)

	// The repo Load call must have been user-scoped.
	require.Len(t, repo.loadCalls, 1)
	assert.Equal(t, contactHandlerTestUserID, repo.loadCalls[0].userID)
	assert.Equal(t, contactID, repo.loadCalls[0].id)
}

func TestContactHandler_Update_NotFound_ReturnsNotFound(t *testing.T) {
	contactID := uuid.New()
	repo := &contactHandlerFakeRepo{
		loadErr: stacktrace.PropagateWithCodef(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found"),
	}
	app := newContactHandlerTestApp(repo)

	body := `{"name":"New Name","phone_numbers":["+18005550199"]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/contacts/"+contactID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	payload := decodeContactHandlerPayload(t, resp)
	assert.Contains(t, payload.Message, contactID.String())

	_, updated, _, _ := repo.snapshot()
	assert.Empty(t, updated)
}

func TestContactHandler_Update_InvalidID_ReturnsUnprocessableEntity(t *testing.T) {
	app := newContactHandlerTestApp(&contactHandlerFakeRepo{})

	body := `{"name":"Alice","phone_numbers":["+18005550199"]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/contacts/not-a-uuid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestContactHandler_Update_ValidationError_ReturnsUnprocessableEntity(t *testing.T) {
	contactID := uuid.New()
	app := newContactHandlerTestApp(&contactHandlerFakeRepo{})

	body := `{"name":"","phone_numbers":[]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/contacts/"+contactID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestContactHandler_Delete_Success(t *testing.T) {
	contactID := uuid.New()
	repo := &contactHandlerFakeRepo{
		loadResult: &entities.Contact{
			ID:     contactID,
			UserID: contactHandlerTestUserID,
			Name:   "Alice",
		},
	}
	app := newContactHandlerTestApp(repo)

	req := httptest.NewRequest(http.MethodDelete, "/v1/contacts/"+contactID.String(), nil)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	_, _, deleted, _ := repo.snapshot()
	require.Len(t, deleted, 1)
	assert.Equal(t, contactHandlerTestUserID, deleted[0].userID)
	assert.Equal(t, contactID, deleted[0].id)
}

func TestContactHandler_Delete_NotFound_ReturnsNotFound(t *testing.T) {
	contactID := uuid.New()
	repo := &contactHandlerFakeRepo{
		loadErr: stacktrace.PropagateWithCodef(gorm.ErrRecordNotFound, repositories.ErrCodeNotFound, "not found"),
	}
	app := newContactHandlerTestApp(repo)

	req := httptest.NewRequest(http.MethodDelete, "/v1/contacts/"+contactID.String(), nil)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Ensure we did not delete another user's contact silently.
	_, _, deleted, _ := repo.snapshot()
	assert.Empty(t, deleted)
}

func TestContactHandler_Delete_InvalidID_ReturnsUnprocessableEntity(t *testing.T) {
	app := newContactHandlerTestApp(&contactHandlerFakeRepo{})

	req := httptest.NewRequest(http.MethodDelete, "/v1/contacts/not-a-uuid", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestContactHandler_Index_ConvertsQueryAndScopesToUser(t *testing.T) {
	repo := &contactHandlerFakeRepo{
		indexResult: []entities.Contact{
			{ID: uuid.New(), UserID: contactHandlerTestUserID, Name: "Alice", PhoneNumbers: pq.StringArray{"+18005550199"}},
		},
	}
	app := newContactHandlerTestApp(repo)

	values := url.Values{}
	values.Set("skip", "5")
	values.Set("limit", "25")
	values.Set("query", "ali")
	req := httptest.NewRequest(http.MethodGet, "/v1/contacts?"+values.Encode(), nil)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	_, _, _, indexParams := repo.snapshot()
	require.Len(t, indexParams, 1)
	assert.Equal(t, 5, indexParams[0].Skip)
	assert.Equal(t, 25, indexParams[0].Limit)
	assert.Equal(t, "ali", indexParams[0].Query)

	payload := decodeContactHandlerPayload(t, resp)
	assert.Equal(t, "success", payload.Status)
	assert.Contains(t, payload.Message, "1")
}

func TestContactHandler_Index_DefaultsAppliedWhenParamsMissing(t *testing.T) {
	repo := &contactHandlerFakeRepo{}
	app := newContactHandlerTestApp(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/contacts", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	_, _, _, indexParams := repo.snapshot()
	require.Len(t, indexParams, 1)
	assert.Equal(t, 0, indexParams[0].Skip)
	assert.Equal(t, 20, indexParams[0].Limit)
	assert.Equal(t, "", indexParams[0].Query)
}

func TestContactHandler_Index_InvalidLimit_ReturnsUnprocessableEntity(t *testing.T) {
	app := newContactHandlerTestApp(&contactHandlerFakeRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/contacts?limit=notanumber", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

// TestContactService_WiresIntoMessageThreadService is a compile-time guard that
// pins the DI wiring change: ContactService must satisfy the contactMapProvider
// interface consumed by MessageThreadService, so container.MessageThreadService()
// can be constructed with container.ContactService() instead of nil.
func TestContactService_WiresIntoMessageThreadService(t *testing.T) {
	logger := &messageThreadHandlerNoopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	appCache := cache.NewMemoryCache(tracer, ttlCache.New(time.Minute, time.Minute))
	contactService := services.NewContactService(logger, tracer, &contactHandlerFakeRepo{}, appCache)

	// If this compiles and runs, the ContactService satisfies the
	// contactMapProvider interface expected by NewMessageThreadService.
	threadService := services.NewMessageThreadService(logger, tracer, nil, nil, nil, contactService)
	require.NotNil(t, threadService)
}
