package services

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// USSDReceiveParams are parameters for receiving a USSD request from a phone
type USSDReceiveParams struct {
	Source    string
	UserID    entities.UserID
	Owner     string
	Contact   string
	Content   string
	SessionID string
	SIM       entities.SIM
	Timestamp time.Time
}

// USSDSendParams are parameters for sending a USSD response to a phone
type USSDSendParams struct {
	Source    string
	UserID    entities.UserID
	Owner     string
	Contact   string
	Content   string
	SessionID string
}

// USSDService handles USSD requests
type USSDService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.USSDRepository
	dispatcher *EventDispatcher
}

// NewUSSDService creates a new USSDService
func NewUSSDService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.USSDRepository,
	dispatcher *EventDispatcher,
) (s *USSDService) {
	return &USSDService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		repository: repository,
		dispatcher: dispatcher,
	}
}

// Receive handles an incoming USSD request from a mobile phone
func (service *USSDService) Receive(ctx context.Context, params *USSDReceiveParams, phoneID uuid.UUID) (*entities.USSD, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	ussd := &entities.USSD{
		ID:        uuid.New(),
		UserID:    params.UserID,
		PhoneID:   phoneID,
		Owner:     params.Owner,
		SessionID: params.SessionID,
		Type:      entities.USSDTypeRequest,
		Direction: entities.USSDDirectionMoToApp,
		Content:   params.Content,
		Status:    entities.USSDStatusPending,
		SIM:       params.SIM,
		Timestamp: params.Timestamp,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := service.repository.Store(ctx, ussd); err != nil {
		msg := fmt.Sprintf("cannot store USSD session with sessionID [%s]", params.SessionID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("USSD session stored with id [%s] and sessionID [%s]", ussd.ID, ussd.SessionID))

	if err := service.dispatchUSSDReceivedEvent(ctx, ussd); err != nil {
		msg := fmt.Sprintf("cannot dispatch USSD received event for session [%s]", ussd.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return ussd, nil
}

// Send handles sending a USSD response to a mobile phone
func (service *USSDService) Send(ctx context.Context, params *USSDSendParams) (*entities.USSD, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	ussd := &entities.USSD{
		ID:        uuid.New(),
		UserID:    params.UserID,
		Owner:     params.Owner,
		SessionID: params.SessionID,
		Type:      entities.USSDTypeResponse,
		Direction: entities.USSDDirectionAppToMO,
		Content:   params.Content,
		Status:    entities.USSDStatusPending,
		Timestamp: time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := service.repository.Store(ctx, ussd); err != nil {
		msg := fmt.Sprintf("cannot store USSD response with sessionID [%s]", params.SessionID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("USSD response stored with id [%s] and sessionID [%s]", ussd.ID, ussd.SessionID))

	if err := service.dispatchUSSDSentEvent(ctx, ussd); err != nil {
		msg := fmt.Sprintf("cannot dispatch USSD sent event for session [%s]", ussd.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return ussd, nil
}

// Index fetches paginated USSD sessions for a user
func (service *USSDService) Index(ctx context.Context, authUser entities.AuthContext, params repositories.IndexParams, phoneID *uuid.UUID) (*[]entities.USSD, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	var ussds *[]entities.USSD
	var err error

	if phoneID != nil {
		ussds, err = service.repository.IndexByPhoneID(ctx, authUser.ID, *phoneID, params)
	} else {
		ussds, err = service.repository.Index(ctx, authUser.ID, params)
	}

	if err != nil {
		msg := fmt.Sprintf("could not fetch USSD sessions with params [%+#v]", params)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] USSD sessions with params [%+#v]", len(*ussds), params))
	return ussds, nil
}

// Delete deletes a USSD session
func (service *USSDService) Delete(ctx context.Context, source string, userID entities.UserID, ussdID uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	ussd, err := service.repository.Load(ctx, userID, ussdID)
	if err != nil {
		msg := fmt.Sprintf("cannot load USSD session with ID [%s]", ussdID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.repository.Delete(ctx, userID, ussdID); err != nil {
		msg := fmt.Sprintf("cannot delete USSD session with ID [%s]", ussdID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted USSD session with ID [%s]", ussdID))

	// Dispatch deleted event
	event, err := service.createEvent("ussd.deleted", source, map[string]interface{}{
		"ussd_id":    ussd.ID,
		"user_id":    ussd.UserID,
		"session_id": ussd.SessionID,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create USSD deleted event for session [%s]", ussd.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch USSD deleted event for session [%s]", ussd.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (service *USSDService) dispatchUSSDReceivedEvent(ctx context.Context, ussd *entities.USSD) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	event, err := service.createUSSDReceivedEvent(ussd)
	if err != nil {
		msg := fmt.Sprintf("cannot create event when USSD is received for session [%s]", ussd.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for USSD session with id [%s]", event.Type(), ussd.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}

func (service *USSDService) dispatchUSSDSentEvent(ctx context.Context, ussd *entities.USSD) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	event, err := service.createUSSDSentEvent(ussd)
	if err != nil {
		msg := fmt.Sprintf("cannot create event when USSD is sent for session [%s]", ussd.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for USSD session with id [%s]", event.Type(), ussd.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}

func (service *USSDService) createUSSDReceivedEvent(ussd *entities.USSD) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeUSSDReceived, fmt.Sprintf("/v1/ussd/receive"), events.USSDReceivedPayload{
		USSDID:    ussd.ID,
		UserID:    ussd.UserID,
		PhoneID:   ussd.PhoneID,
		Owner:     ussd.Owner,
		SessionID: ussd.SessionID,
		Content:   ussd.Content,
		SIM:       ussd.SIM,
		Timestamp: ussd.Timestamp,
	})
}

func (service *USSDService) createUSSDSentEvent(ussd *entities.USSD) (cloudevents.Event, error) {
	response := ""
	if ussd.Response != nil {
		response = *ussd.Response
	}
	return service.createEvent(events.EventTypeUSSDResponse, fmt.Sprintf("/v1/ussd/send"), events.USSDResponsePayload{
		USSDID:    ussd.ID,
		UserID:    ussd.UserID,
		PhoneID:   ussd.PhoneID,
		Owner:     ussd.Owner,
		SessionID: ussd.SessionID,
		Response:  response,
		SIM:       ussd.SIM,
		Timestamp: ussd.Timestamp,
	})
}