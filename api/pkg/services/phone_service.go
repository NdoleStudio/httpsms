package services

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"
	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// PhoneService is handles phone requests
type PhoneService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.PhoneRepository
	dispatcher *EventDispatcher
}

// NewPhoneService creates a new PhoneService
func NewPhoneService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.PhoneRepository,
	dispatcher *EventDispatcher,
) (s *PhoneService) {
	return &PhoneService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		dispatcher: dispatcher,
		repository: repository,
	}
}

// DeleteAllForUser deletes all entities.Phone for an entities.UserID.
func (service *PhoneService) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.repository.DeleteAllForUser(ctx, userID); err != nil {
		msg := fmt.Sprintf("could not delete all [entities.Phone] for user with ID [%s]", userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted all [entities.Phone] for user with ID [%s]", userID))
	return nil
}

// Index fetches the heartbeats for a phone number
func (service *PhoneService) Index(ctx context.Context, authUser entities.AuthContext, params repositories.IndexParams) (*[]entities.Phone, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	phones, err := service.repository.Index(ctx, authUser.ID, params)
	if err != nil {
		msg := fmt.Sprintf("could not fetch phones with parms [%+#v]", params)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] phones with prams [%+#v]", len(*phones), params))
	return phones, nil
}

// Load a phone by userID and owner
func (service *PhoneService) Load(ctx context.Context, userID entities.UserID, owner string) (*entities.Phone, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	return service.repository.Load(ctx, userID, owner)
}

// PhoneUpsertParams are parameters for creating a new entities.Phone
type PhoneUpsertParams struct {
	PhoneNumber               *phonenumbers.PhoneNumber
	FcmToken                  *string
	MessagesPerMinute         *uint
	MaxSendAttempts           *uint
	WebhookURL                *string
	MessageExpirationDuration *time.Duration
	MissedCallAutoReply       *string
	SIM                       entities.SIM
	Source                    string
	UserID                    entities.UserID
}

// Upsert a new entities.Phone
func (service *PhoneService) Upsert(ctx context.Context, params *PhoneUpsertParams) (*entities.Phone, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	phone, err := service.repository.Load(ctx, params.UserID, phonenumbers.Format(params.PhoneNumber, phonenumbers.E164))
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return service.createPhone(ctx, &PhoneFCMTokenParams{
			Source:        params.Source,
			PhoneNumber:   params.PhoneNumber,
			PhoneAPIKeyID: nil,
			UserID:        params.UserID,
			FcmToken:      params.FcmToken,
			SIM:           params.SIM,
		})
	}

	if err != nil {
		msg := fmt.Sprintf("cannot upsert phone with id [%s] and number [%s]", phone.ID, phone.PhoneNumber)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.repository.Save(ctx, service.update(phone, params)); err != nil {
		msg := fmt.Sprintf("cannot update phone with id [%s] and number [%s]", phone.ID, phone.PhoneNumber)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("phone updated with id [%s] in the phone repository for user [%s]", phone.ID, phone.UserID))
	return phone, service.dispatchPhoneUpdatedEvent(ctx, phone, &PhoneFCMTokenParams{
		Source:        params.Source,
		PhoneNumber:   params.PhoneNumber,
		PhoneAPIKeyID: nil,
		UserID:        params.UserID,
		FcmToken:      params.FcmToken,
		SIM:           params.SIM,
	})
}

func (service *PhoneService) dispatchPhoneUpdatedEvent(ctx context.Context, phone *entities.Phone, input *PhoneFCMTokenParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	event, err := service.createPhoneUpdatedEvent(input.Source, events.PhoneUpdatedPayload{
		PhoneID:       phone.ID,
		UserID:        phone.UserID,
		Timestamp:     phone.UpdatedAt,
		PhoneAPIKeyID: input.PhoneAPIKeyID,
		Owner:         phone.PhoneNumber,
		SIM:           phone.SIM,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event when phone [%s] is updated for user [%s]", phone.ID, phone.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for phone with id [%s]", event.Type(), phone.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}

// Delete an entities.Phone
func (service *PhoneService) Delete(ctx context.Context, source string, userID entities.UserID, phoneID uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	phone, err := service.repository.LoadByID(ctx, userID, phoneID)
	if err != nil {
		msg := fmt.Sprintf("cannot load phone with userID [%s] and phoneID [%s]", userID, phoneID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.repository.Delete(ctx, userID, phoneID); err != nil {
		msg := fmt.Sprintf("cannot delete phone with id [%s] and user id [%s]", phoneID, userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted phone with id [%s] and user id [%s]", phoneID, userID))

	event, err := service.createPhoneDeletedEvent(source, events.PhoneDeletedPayload{
		PhoneID:   phone.ID,
		UserID:    phone.UserID,
		Timestamp: phone.UpdatedAt,
		Owner:     phone.PhoneNumber,
		SIM:       phone.SIM,
	})
	if err != nil {
		msg := "cannot create event when phone is deleted"
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for phone with id [%s]", event.Type(), phone.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// PhoneFCMTokenParams are parameters for upserting an entities.Phone
type PhoneFCMTokenParams struct {
	Source        string
	PhoneNumber   *phonenumbers.PhoneNumber
	PhoneAPIKeyID *uuid.UUID
	UserID        entities.UserID
	FcmToken      *string
	SIM           entities.SIM
}

// UpsertFCMToken the FCM token for an entities.Phone
func (service *PhoneService) UpsertFCMToken(ctx context.Context, params *PhoneFCMTokenParams) (*entities.Phone, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	phone, err := service.repository.Load(ctx, params.UserID, phonenumbers.Format(params.PhoneNumber, phonenumbers.E164))
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return service.createPhone(ctx, params)
	}

	if err != nil {
		msg := fmt.Sprintf("cannot upsert FCM token for user with id [%s] and number [%s]", params.UserID, params.PhoneNumber)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	phone.FcmToken = params.FcmToken
	phone.SIM = params.SIM

	if err = service.repository.Save(ctx, phone); err != nil {
		msg := fmt.Sprintf("cannot update phone with id [%s] and number [%s]", phone.ID, phone.PhoneNumber)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("phone updated with id [%s] in the phone repository for user [%s]", phone.ID, phone.UserID))
	return phone, service.dispatchPhoneUpdatedEvent(ctx, phone, params)
}

func (service *PhoneService) createPhone(ctx context.Context, params *PhoneFCMTokenParams) (*entities.Phone, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	phone := &entities.Phone{
		ID:       uuid.New(),
		UserID:   params.UserID,
		FcmToken: params.FcmToken,
		// Android has a limit of 30 SMS messages per minute without user permission, to be safe let's use 10 messages per minute
		// https://android.googlesource.com/platform/frameworks/opt/telephony/+/master/src/java/com/android/internal/telephony/SmsUsageMonitor.java#80
		MessagesPerMinute:        10,
		MessageExpirationSeconds: 10 * 60, // 10 minutes
		MaxSendAttempts:          2,
		SIM:                      params.SIM,
		MissedCallAutoReply:      nil,
		PhoneNumber:              phonenumbers.Format(params.PhoneNumber, phonenumbers.E164),
		CreatedAt:                time.Now().UTC(),
		UpdatedAt:                time.Now().UTC(),
	}

	if err := service.repository.Save(ctx, phone); err != nil {
		msg := fmt.Sprintf("cannot create phone with id [%s] and number [%s]", phone.ID, phone.PhoneNumber)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("phone updated with id [%s] in the phone repository for user [%s]", phone.ID, phone.UserID))
	return phone, service.dispatchPhoneUpdatedEvent(ctx, phone, params)
}

func (service *PhoneService) createPhoneUpdatedEvent(source string, payload events.PhoneUpdatedPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypePhoneUpdated, source, payload)
}

func (service *PhoneService) createPhoneDeletedEvent(source string, payload events.PhoneDeletedPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypePhoneDeleted, source, payload)
}

func (service *PhoneService) update(phone *entities.Phone, params *PhoneUpsertParams) *entities.Phone {
	if phone.FcmToken != nil {
		phone.FcmToken = params.FcmToken
	}
	if params.MessagesPerMinute != nil && *params.MessagesPerMinute > 0 {
		phone.MessagesPerMinute = *params.MessagesPerMinute
	}

	if params.MaxSendAttempts != nil && *params.MaxSendAttempts > 0 {
		phone.MaxSendAttempts = *params.MaxSendAttempts
	}

	if params.MessageExpirationDuration != nil {
		phone.MessageExpirationSeconds = uint(params.MessageExpirationDuration.Seconds())
	}

	if params.MissedCallAutoReply != nil {
		phone.MissedCallAutoReply = params.MissedCallAutoReply
	}

	phone.SIM = params.SIM

	return phone
}
