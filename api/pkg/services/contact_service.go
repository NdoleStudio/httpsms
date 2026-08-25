package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/google/uuid"
)

// contactCacheTTL bounds staleness if an invalidation is ever missed.
const (
	contactMapCacheTTL               = 24 * time.Hour
	contactGenerationCleanupInterval = time.Hour
)

// ContactCacheEntry stores a contact together with its user's cache generation.
type ContactCacheEntry struct {
	contact    *entities.Contact
	generation uint64
}

type contactGeneration struct {
	value      uint64
	lastAccess time.Time
}

// ContactService owns contact CRUD and phone-number contact lookups.
type ContactService struct {
	service
	logger                telemetry.Logger
	tracer                telemetry.Tracer
	repository            repositories.ContactRepository
	cache                 *ristretto.Cache[string, ContactCacheEntry]
	generationMu          sync.Mutex
	generations           map[entities.UserID]contactGeneration
	nextGeneration        uint64
	lastGenerationCleanup time.Time
}

// NewContactService creates a new ContactService.
func NewContactService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.ContactRepository,
	contactCache *ristretto.Cache[string, ContactCacheEntry],
) (s *ContactService) {
	return &ContactService{
		logger:      logger.WithService(fmt.Sprintf("%T", s)),
		tracer:      tracer,
		repository:  repository,
		cache:       contactCache,
		generations: make(map[entities.UserID]contactGeneration),
	}
}

func (service *ContactService) cacheKey(userID entities.UserID, phoneNumber string) string {
	return fmt.Sprintf("%s|%s", userID, phoneNumber)
}

func (service *ContactService) generation(userID entities.UserID) uint64 {
	now := time.Now()
	service.generationMu.Lock()
	defer service.generationMu.Unlock()

	service.expireGenerationsLocked(now)
	if generation, ok := service.generations[userID]; ok {
		generation.lastAccess = now
		service.generations[userID] = generation
		return generation.value
	}

	service.nextGeneration++
	service.generations[userID] = contactGeneration{
		value:      service.nextGeneration,
		lastAccess: now,
	}
	return service.nextGeneration
}

func (service *ContactService) advanceGeneration(userID entities.UserID) {
	now := time.Now()
	service.generationMu.Lock()
	defer service.generationMu.Unlock()

	service.expireGenerationsLocked(now)
	service.nextGeneration++
	service.generations[userID] = contactGeneration{
		value:      service.nextGeneration,
		lastAccess: now,
	}
}

func (service *ContactService) expireGenerations(now time.Time) {
	service.generationMu.Lock()
	defer service.generationMu.Unlock()
	service.expireGenerationsLocked(now)
}

func (service *ContactService) expireGenerationsLocked(now time.Time) {
	if !service.lastGenerationCleanup.IsZero() &&
		now.Sub(service.lastGenerationCleanup) < contactGenerationCleanupInterval {
		return
	}
	for userID, generation := range service.generations {
		if now.Sub(generation.lastAccess) >= contactMapCacheTTL {
			delete(service.generations, userID)
		}
	}
	service.lastGenerationCleanup = now
}

// CreateMany persists one or many contacts in a single batch.
func (service *ContactService) CreateMany(ctx context.Context, userID entities.UserID, contacts []*entities.Contact) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	if err := service.repository.Store(ctx, contacts); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot store [%d] contacts for user [%s]", len(contacts), userID))
	}

	phoneNumbers := make([]string, 0)
	for _, contact := range contacts {
		phoneNumbers = append(phoneNumbers, contact.PhoneNumbers...)
	}
	service.invalidate(userID, phoneNumbers)
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

// Update persists changes to a contact and invalidates its old and new numbers.
func (service *ContactService) Update(ctx context.Context, contact *entities.Contact, previousPhoneNumbers []string) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	if err := service.repository.Update(ctx, contact); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update contact [%s] for user [%s]", contact.ID, contact.UserID))
	}

	phoneNumbers := make([]string, 0, len(previousPhoneNumbers)+len(contact.PhoneNumbers))
	phoneNumbers = append(phoneNumbers, previousPhoneNumbers...)
	phoneNumbers = append(phoneNumbers, contact.PhoneNumbers...)
	service.invalidate(contact.UserID, phoneNumbers)
	return nil
}

// Delete removes a contact scoped to the user and invalidates its phone numbers.
func (service *ContactService) Delete(ctx context.Context, userID entities.UserID, contactID uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	contact, err := service.repository.Load(ctx, userID, contactID)
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, stacktrace.GetCode(err), "cannot load contact [%s] for user [%s] before delete", contactID, userID))
	}

	if err := service.repository.Delete(ctx, userID, contactID); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete contact [%s] for user [%s]", contactID, userID))
	}

	service.invalidate(userID, contact.PhoneNumbers)
	return nil
}

// DeleteAllForUser removes every contact owned by a user and clears cached contacts.
func (service *ContactService) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	if err := service.repository.DeleteAllForUser(ctx, userID); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete all contacts for user [%s]", userID))
	}

	service.invalidate(userID, nil)
	return nil
}

// GetContactMap resolves only the requested phone numbers. Each contact is
// cached independently by user and phone number.
func (service *ContactService) GetContactMap(ctx context.Context, userID entities.UserID, phoneNumbers []string) (map[string]*entities.Contact, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	generation := service.generation(userID)
	result := make(map[string]*entities.Contact, len(phoneNumbers))
	missing := make([]string, 0, len(phoneNumbers))
	missingSet := make(map[string]struct{}, len(phoneNumbers))
	requested := make(map[string]struct{}, len(phoneNumbers))
	for _, phoneNumber := range phoneNumbers {
		if _, seen := requested[phoneNumber]; seen {
			continue
		}
		requested[phoneNumber] = struct{}{}
		key := service.cacheKey(userID, phoneNumber)
		if entry, found := service.cache.Get(key); found {
			if entry.generation == generation {
				if entry.contact != nil {
					result[phoneNumber] = entry.contact
				}
				continue
			}
			service.cache.Del(key)
		}
		missing = append(missing, phoneNumber)
		missingSet[phoneNumber] = struct{}{}
	}

	if len(missing) == 0 {
		return result, nil
	}

	contacts, err := service.repository.FetchByPhoneNumbers(ctx, userID, missing)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch contacts by phone numbers for user [%s]", userID))
	}

	for index := range *contacts {
		contact := (*contacts)[index]
		for _, phoneNumber := range contact.PhoneNumbers {
			if _, requested := missingSet[phoneNumber]; requested {
				result[phoneNumber] = &contact
			}
		}
	}

	if service.generation(userID) == generation {
		for _, phoneNumber := range missing {
			contact := result[phoneNumber]
			entry := ContactCacheEntry{contact: contact, generation: generation}
			if accepted := service.cache.SetWithTTL(service.cacheKey(userID, phoneNumber), entry, 1, contactMapCacheTTL); !accepted {
				ctxLogger.Error(stacktrace.NewErrorf("cannot cache contact lookup for user [%s] and phone number [%s]", userID, phoneNumber))
			}
		}
	}

	return result, nil
}

func (service *ContactService) invalidate(userID entities.UserID, phoneNumbers []string) {
	service.advanceGeneration(userID)
	service.cache.Wait()
	for _, phoneNumber := range phoneNumbers {
		service.cache.Del(service.cacheKey(userID, phoneNumber))
	}
}
