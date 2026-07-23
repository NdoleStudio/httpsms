package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/cache"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
)

// contactMapCacheTTL bounds staleness of the cached phone-number -> contact map.
// A 24h ceiling lets self-healing kick in even if an invalidation write is ever lost.
const contactMapCacheTTL = 24 * time.Hour

// ContactService owns contact CRUD and the cached phone-number-to-contact map
// used by higher-level services (e.g. GetThreads) to resolve display names.
type ContactService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.ContactRepository
	cache      cache.Cache
}

// NewContactService creates a new ContactService.
func NewContactService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.ContactRepository,
	appCache cache.Cache,
) (s *ContactService) {
	return &ContactService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		repository: repository,
		cache:      appCache,
	}
}

func (service *ContactService) cacheKey(userID entities.UserID) string {
	return fmt.Sprintf("contacts.map.%s", userID)
}

// CreateMany persists one or many contacts in a single batch and invalidates the cached map.
func (service *ContactService) CreateMany(ctx context.Context, userID entities.UserID, contacts []*entities.Contact) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.repository.Store(ctx, contacts); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot store [%d] contacts for user [%s]", len(contacts), userID))
	}

	service.invalidate(ctx, ctxLogger, userID)
	return nil
}

// Get returns a single contact scoped to the user.
func (service *ContactService) Get(ctx context.Context, userID entities.UserID, contactID uuid.UUID) (*entities.Contact, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	contact, err := service.repository.Load(ctx, userID, contactID)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, stacktrace.GetCode(err), "cannot load contact [%s] for user [%s]", contactID, userID))
	}
	return contact, nil
}

// Index lists contacts for a user with the provided search/pagination params.
func (service *ContactService) Index(ctx context.Context, userID entities.UserID, params repositories.IndexParams) (*[]entities.Contact, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	contacts, err := service.repository.Index(ctx, userID, params)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot index contacts for user [%s]", userID))
	}
	return contacts, nil
}

// Count returns the total number of contacts for a user matching the same
// search filter as Index, ignoring pagination. It lets callers report an
// accurate total independent of the current page's skip/limit.
func (service *ContactService) Count(ctx context.Context, userID entities.UserID, params repositories.IndexParams) (int64, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	total, err := service.repository.Count(ctx, userID, params)
	if err != nil {
		return 0, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot count contacts for user [%s]", userID))
	}
	return total, nil
}

// Update persists changes to a contact and invalidates the cached map.
func (service *ContactService) Update(ctx context.Context, contact *entities.Contact) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.repository.Update(ctx, contact); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update contact [%s] for user [%s]", contact.ID, contact.UserID))
	}

	service.invalidate(ctx, ctxLogger, contact.UserID)
	return nil
}

// Delete removes a contact scoped to the user and invalidates the cached map.
func (service *ContactService) Delete(ctx context.Context, userID entities.UserID, contactID uuid.UUID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.repository.Delete(ctx, userID, contactID); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete contact [%s] for user [%s]", contactID, userID))
	}

	service.invalidate(ctx, ctxLogger, userID)
	return nil
}

// GetContactMap returns a phone_number -> *Contact map for the given user, backed by a per-user cache.
// FetchAll orders by updated_at ASC, so map construction naturally lets the most-recently-updated
// contact win when two contacts share a phone number.
func (service *ContactService) GetContactMap(ctx context.Context, userID entities.UserID) (map[string]*entities.Contact, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	key := service.cacheKey(userID)

	raw, cacheErr := service.cache.Get(ctx, key)
	switch {
	case cacheErr != nil:
		// The Cache interface has no typed miss signal, so a Get error is
		// treated as "no cached value, rebuild from source". This is expected
		// on cold starts, so log at Debug — a real fault surfaces via the
		// subsequent repository/set calls or via cache-level metrics.
		ctxLogger.Debug(fmt.Sprintf("contact map cache miss for user [%s]: %s", userID, cacheErr.Error()))
	case raw == "":
		// Empty string is the invalidation marker written by mutations.
		ctxLogger.Debug(fmt.Sprintf("contact map cache invalidated for user [%s], rebuilding", userID))
	default:
		result := map[string]*entities.Contact{}
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			ctxLogger.Error(stacktrace.Propagatef(err, "cannot unmarshal contact map cache for user [%s], rebuilding", userID))
		} else {
			return result, nil
		}
	}

	contacts, err := service.repository.FetchAll(ctx, userID)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch contacts to build map for user [%s]", userID))
	}

	result := make(map[string]*entities.Contact, len(*contacts))
	for index := range *contacts {
		// Take a fresh copy of the slice element so the map holds a stable pointer
		// that does not alias the underlying slice storage.
		contact := (*contacts)[index]
		for _, number := range contact.PhoneNumbers {
			result[number] = &contact
		}
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot marshal contact map cache payload for user [%s]", userID))
		return result, nil
	}
	if err := service.cache.Set(ctx, key, string(encoded), contactMapCacheTTL); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot populate contact map cache for user [%s]", userID))
	}

	return result, nil
}

// invalidate writes an empty marker under the user's cache key. The Cache
// interface has no Delete; the marker is treated as a rebuild trigger by
// GetContactMap. Failures are logged at Error but do not fail the calling
// mutation: the DB write already succeeded, and the 24h TTL bounds staleness.
// See docs in contact_service_test.go / task-6-report.md for the rationale.
func (service *ContactService) invalidate(ctx context.Context, ctxLogger telemetry.Logger, userID entities.UserID) {
	key := service.cacheKey(userID)
	if err := service.cache.Set(ctx, key, "", contactMapCacheTTL); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot invalidate contact map cache for user [%s] with key [%s]", userID, key))
	}
}
