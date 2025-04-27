package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/nyaruka/phonenumbers"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// MessageService is handles message requests
type MessageService struct {
	service
	logger          telemetry.Logger
	tracer          telemetry.Tracer
	eventDispatcher *EventDispatcher
	phoneService    *PhoneService
	repository      repositories.MessageRepository
}

// NewMessageService creates a new MessageService
func NewMessageService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.MessageRepository,
	eventDispatcher *EventDispatcher,
	phoneService *PhoneService,
) (s *MessageService) {
	return &MessageService{
		logger:          logger.WithService(fmt.Sprintf("%T", s)),
		tracer:          tracer,
		repository:      repository,
		phoneService:    phoneService,
		eventDispatcher: eventDispatcher,
	}
}

// MessageGetOutstandingParams parameters for sending a new message
type MessageGetOutstandingParams struct {
	Source       string
	UserID       entities.UserID
	PhoneNumbers []string
	Timestamp    time.Time
	MessageID    uuid.UUID
}

// GetOutstanding fetches messages that still to be sent to the phone
func (service *MessageService) GetOutstanding(ctx context.Context, params MessageGetOutstandingParams) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message, err := service.repository.GetOutstanding(ctx, params.UserID, params.MessageID, params.PhoneNumbers)
	if err != nil {
		msg := fmt.Sprintf("could not fetch outstanding messages with params [%s]", spew.Sdump(params))
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	event, err := service.createMessagePhoneSendingEvent(params.Source, events.MessagePhoneSendingPayload{
		ID:        message.ID,
		Owner:     message.Owner,
		Contact:   message.Contact,
		Timestamp: params.Timestamp,
		Encrypted: message.Encrypted,
		UserID:    message.UserID,
		Content:   message.Content,
		SIM:       message.SIM,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create [%T] for message with ID [%s]", event, message.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("created event [%s] with id [%s] for message [%s]", event.Type(), event.ID(), message.ID))

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] with id [%s] for message [%s]", event.Type(), event.ID(), message.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("dispatched event [%s] with id [%s] for message [%s]", event.Type(), event.ID(), message.ID))
	return message, nil
}

// DeleteAllForUser deletes all entities.Message for an entities.UserID.
func (service *MessageService) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.repository.DeleteAllForUser(ctx, userID); err != nil {
		msg := fmt.Sprintf("could not delete [entities.Message] for user with ID [%s]", userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted all [entities.Message] for user with ID [%s]", userID))
	return nil
}

// DeleteMessage deletes a message from the database
func (service *MessageService) DeleteMessage(ctx context.Context, source string, message *entities.Message) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if err := service.repository.Delete(ctx, message.UserID, message.ID); err != nil {
		msg := fmt.Sprintf("could not delete message with ID [%s] for user wit ID [%s]", message.ID, message.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	var prevID *uuid.UUID
	var prevStatus *entities.MessageStatus
	var prevContent *string
	previousMessage, err := service.repository.LastMessage(ctx, message.UserID, message.Owner, message.Contact)
	if err == nil {
		prevID = &previousMessage.ID
		prevStatus = &previousMessage.Status
		prevContent = &previousMessage.Content
	}

	event, err := service.createEvent(events.MessageAPIDeleted, source, &events.MessageAPIDeletedPayload{
		MessageID:              message.ID,
		UserID:                 message.UserID,
		Owner:                  message.Owner,
		Encrypted:              message.Encrypted,
		RequestID:              message.RequestID,
		Contact:                message.Contact,
		Timestamp:              time.Now().UTC(),
		Content:                message.Content,
		PreviousMessageID:      prevID,
		PreviousMessageStatus:  prevStatus,
		PreviousMessageContent: prevContent,
		SIM:                    message.SIM,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create [%T] for message with ID [%s]", event, message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("created event [%s] with id [%s] for message [%s]", event.Type(), event.ID(), message.ID))
	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] with id [%s] for message [%s]", event.Type(), event.ID(), message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("dispatched event [%s] with id [%s] for message [%s]", event.Type(), event.ID(), message.ID))
	return nil
}

// DeleteByOwnerAndContact deletes all the messages between an owner and a contact
func (service *MessageService) DeleteByOwnerAndContact(ctx context.Context, userID entities.UserID, owner, contact string) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if err := service.repository.DeleteByOwnerAndContact(ctx, userID, owner, contact); err != nil {
		msg := fmt.Sprintf("could not all delete messages for user with ID [%s] between owner [%s] and contact [%s] ", userID, owner, contact)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted all messages for user with ID [%s] between owner [%s] and contact [%s] ", userID, owner, contact))
	return nil
}

// RespondToMissedCall creates an SMS response to a missed phone call on the android phone
func (service *MessageService) RespondToMissedCall(ctx context.Context, source string, payload *events.MessageCallMissedPayload) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	phone, err := service.phoneService.Load(ctx, payload.UserID, payload.Owner)
	if err != nil {
		msg := fmt.Sprintf("cannot find phone with owner [%s] for user with ID [%s] when handling missed phone call message [%s]", payload.Owner, payload.UserID, payload.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if phone.MissedCallAutoReply == nil || strings.TrimSpace(*phone.MissedCallAutoReply) == "" {
		ctxLogger.Info(fmt.Sprintf("no auto reply set for phone [%s] for message [%s] with user [%s]", payload.Owner, payload.MessageID, payload.UserID))
		return nil
	}

	requestID := fmt.Sprintf("missed-call-%s", payload.MessageID)
	owner, _ := phonenumbers.Parse(payload.Owner, phonenumbers.UNKNOWN_REGION)
	message, err := service.SendMessage(ctx, MessageSendParams{
		Owner:             owner,
		Contact:           payload.Contact,
		Encrypted:         false,
		Content:           *phone.MissedCallAutoReply,
		Source:            source,
		SendAt:            nil,
		RequestID:         &requestID,
		UserID:            payload.UserID,
		RequestReceivedAt: time.Now().UTC(),
	})
	if err != nil {
		msg := fmt.Sprintf("cannot send auto response message for owner [%s] for user with ID [%s] when handling missed phone call message [%s]", payload.Owner, payload.UserID, payload.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("created response message with ID [%s] for missed call event [%s] for user [%s]", message.ID, payload.MessageID, message.UserID))
	return nil
}

// MessageGetParams parameters for sending a new message
type MessageGetParams struct {
	repositories.IndexParams
	UserID  entities.UserID
	Owner   string
	Contact string
}

// GetMessages fetches sent between 2 phone numbers
func (service *MessageService) GetMessages(ctx context.Context, params MessageGetParams) (*[]entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	messages, err := service.repository.Index(ctx, params.UserID, params.Owner, params.Contact, params.IndexParams)
	if err != nil {
		msg := fmt.Sprintf("could not fetch messages with parms [%+#v]", params)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] messages with prams [%+#v]", len(*messages), params))
	return messages, nil
}

// GetMessage fetches a message by the ID
func (service *MessageService) GetMessage(ctx context.Context, userID entities.UserID, messageID uuid.UUID) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	message, err := service.repository.Load(ctx, userID, messageID)
	if err != nil {
		msg := fmt.Sprintf("could not fetch messages with ID [%s]", messageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	return message, nil
}

// MessageStoreEventParams parameters registering a message event
type MessageStoreEventParams struct {
	MessageID    uuid.UUID
	EventName    entities.MessageEventName
	Timestamp    time.Time
	ErrorMessage *string
	Source       string
}

// StoreEvent handles event generated by a mobile phone
func (service *MessageService) StoreEvent(ctx context.Context, message *entities.Message, params MessageStoreEventParams) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	var err error

	switch params.EventName {
	case entities.MessageEventNameSent:
		err = service.handleMessageSentEvent(ctx, params, message)
	case entities.MessageEventNameDelivered:
		err = service.handleMessageDeliveredEvent(ctx, params, message)
	case entities.MessageEventNameFailed:
		err = service.handleMessageFailedEvent(ctx, params, message)
	default:
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.NewError(fmt.Sprintf("cannot handle message event [%s]", params.EventName)))
	}

	if err != nil {
		msg := fmt.Sprintf("could not handle phone event [%s] for message with id [%s]", params.EventName, message.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return service.repository.Load(ctx, message.UserID, params.MessageID)
}

// MessageReceiveParams parameters registering a message event
type MessageReceiveParams struct {
	Contact   string
	UserID    entities.UserID
	Owner     phonenumbers.PhoneNumber
	Content   string
	SIM       entities.SIM
	Timestamp time.Time
	Encrypted bool
	Source    string
}

// ReceiveMessage handles message received by a mobile phone
func (service *MessageService) ReceiveMessage(ctx context.Context, params *MessageReceiveParams) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	eventPayload := events.MessagePhoneReceivedPayload{
		MessageID: uuid.New(),
		UserID:    params.UserID,
		Encrypted: params.Encrypted,
		Owner:     phonenumbers.Format(&params.Owner, phonenumbers.E164),
		Contact:   params.Contact,
		Timestamp: params.Timestamp,
		Content:   params.Content,
		SIM:       params.SIM,
	}

	ctxLogger.Info(fmt.Sprintf("creating cloud event for received with ID [%s]", eventPayload.MessageID))

	event, err := service.createMessagePhoneReceivedEvent(params.Source, eventPayload)
	if err != nil {
		msg := fmt.Sprintf("cannot create %T from payload with message id [%s]", event, eventPayload.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("created event [%s] with id [%s] and message id [%s]", event.Type(), event.ID(), eventPayload.MessageID))

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event type [%s] and id [%s]", event.Type(), event.ID())
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	ctxLogger.Info(fmt.Sprintf("event [%s] dispatched succesfully", event.ID()))

	return service.storeReceivedMessage(ctx, eventPayload)
}

func (service *MessageService) handleMessageSentEvent(ctx context.Context, params MessageStoreEventParams, message *entities.Message) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	event, err := service.createMessagePhoneSentEvent(params.Source, events.MessagePhoneSentPayload{
		ID:        message.ID,
		Owner:     message.Owner,
		UserID:    message.UserID,
		RequestID: message.RequestID,
		Timestamp: params.Timestamp,
		Contact:   message.Contact,
		Encrypted: message.Encrypted,
		Content:   message.Content,
		SIM:       message.SIM,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event [%s] for message [%s]", events.EventTypeMessagePhoneSent, message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event type [%s] and id [%s]", event.Type(), event.ID())
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}

func (service *MessageService) handleMessageDeliveredEvent(ctx context.Context, params MessageStoreEventParams, message *entities.Message) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	event, err := service.createMessagePhoneDeliveredEvent(params.Source, events.MessagePhoneDeliveredPayload{
		ID:        message.ID,
		Owner:     message.Owner,
		UserID:    message.UserID,
		RequestID: message.RequestID,
		Timestamp: params.Timestamp,
		Encrypted: message.Encrypted,
		Contact:   message.Contact,
		Content:   message.Content,
		SIM:       message.SIM,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event [%s] for message [%s]", events.EventTypeMessagePhoneSent, message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if _, err = service.eventDispatcher.DispatchWithTimeout(ctx, event, time.Second); err != nil {
		msg := fmt.Sprintf("cannot dispatch event type [%s] and id [%s] for message [%s]", event.Type(), event.ID(), message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}

func (service *MessageService) handleMessageFailedEvent(ctx context.Context, params MessageStoreEventParams, message *entities.Message) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	errorMessage := "UNKNOWN ERROR"
	if params.ErrorMessage != nil {
		errorMessage = *params.ErrorMessage
	}

	event, err := service.createMessageSendFailedEvent(params.Source, events.MessageSendFailedPayload{
		ID:           message.ID,
		Owner:        message.Owner,
		ErrorMessage: errorMessage,
		Timestamp:    params.Timestamp,
		Encrypted:    message.Encrypted,
		Contact:      message.Contact,
		RequestID:    message.RequestID,
		UserID:       message.UserID,
		Content:      message.Content,
		SIM:          message.SIM,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event [%s] for message [%s]", events.EventTypeMessageSendFailed, message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event type [%s] and id [%s]", event.Type(), event.ID())
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}

// MessageSendParams parameters for sending a new message
type MessageSendParams struct {
	Owner             *phonenumbers.PhoneNumber
	Contact           string
	Encrypted         bool
	Content           string
	Source            string
	SendAt            *time.Time
	RequestID         *string
	UserID            entities.UserID
	RequestReceivedAt time.Time
}

// SendMessage a new message
func (service *MessageService) SendMessage(ctx context.Context, params MessageSendParams) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	sendAttempts, sim := service.phoneSettings(ctx, params.UserID, phonenumbers.Format(params.Owner, phonenumbers.E164))

	eventPayload := events.MessageAPISentPayload{
		MessageID:         uuid.New(),
		UserID:            params.UserID,
		Encrypted:         params.Encrypted,
		MaxSendAttempts:   sendAttempts,
		RequestID:         params.RequestID,
		Owner:             phonenumbers.Format(params.Owner, phonenumbers.E164),
		Contact:           params.Contact,
		RequestReceivedAt: params.RequestReceivedAt,
		Content:           params.Content,
		ScheduledSendTime: params.SendAt,
		SIM:               sim,
	}

	event, err := service.createMessageAPISentEvent(params.Source, eventPayload)
	if err != nil {
		msg := fmt.Sprintf("cannot create %T from payload with message id [%s]", event, eventPayload.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	ctxLogger.Info(fmt.Sprintf("created event [%s] with id [%s] and message id [%s] and user [%s]", event.Type(), event.ID(), eventPayload.MessageID, eventPayload.UserID))

	message, err := service.storeSentMessage(ctx, eventPayload)
	if err != nil {
		msg := fmt.Sprintf("cannot store message with id [%s]", eventPayload.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	timeout := service.getSendDelay(ctxLogger, eventPayload, params.SendAt)
	if _, err = service.eventDispatcher.DispatchWithTimeout(ctx, event, timeout); err != nil {
		msg := fmt.Sprintf("cannot dispatch event type [%s] and id [%s]", event.Type(), event.ID())
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("[%s] event with ID [%s] dispatched succesfully for message [%s] with user [%s] and delay [%s]", event.Type(), event.ID(), eventPayload.MessageID, eventPayload.UserID, timeout))
	return message, err
}

// MissedCallParams parameters for sending a new message
type MissedCallParams struct {
	Owner     *phonenumbers.PhoneNumber
	Contact   string
	Source    string
	SIM       entities.SIM
	Timestamp time.Time
	UserID    entities.UserID
}

// RegisterMissedCall a new message
func (service *MessageService) RegisterMissedCall(ctx context.Context, params *MissedCallParams) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	eventPayload := &events.MessageCallMissedPayload{
		MessageID: uuid.New(),
		UserID:    params.UserID,
		Timestamp: params.Timestamp,
		Owner:     phonenumbers.Format(params.Owner, phonenumbers.E164),
		Contact:   params.Contact,
		SIM:       params.SIM,
	}

	event, err := service.createEvent(events.MessageCallMissed, params.Source, eventPayload)
	if err != nil {
		msg := fmt.Sprintf("cannot create [%T] from payload with message id [%s]", event, eventPayload.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("created event [%s] with id [%s] and message id [%s] and user [%s]", event.Type(), event.ID(), eventPayload.MessageID, eventPayload.UserID))

	message, err := service.storeMissedCallMessage(ctx, eventPayload)
	if err != nil {
		msg := fmt.Sprintf("cannot store missed call message message with id [%s]", eventPayload.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event type [%s] and id [%s]", event.Type(), event.ID())
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("[%s] event with ID [%s] dispatched succesfully for message [%s] with user [%s]", event.Type(), event.ID(), eventPayload.MessageID, eventPayload.UserID))
	return message, err
}

func (service *MessageService) getSendDelay(ctxLogger telemetry.Logger, eventPayload events.MessageAPISentPayload, sendAt *time.Time) time.Duration {
	if sendAt == nil {
		return time.Duration(0)
	}

	delay := sendAt.Sub(time.Now().UTC())
	if delay < 0 {
		ctxLogger.Info(fmt.Sprintf("message [%s] has send time [%s] in the past. sending immediately", eventPayload.MessageID, sendAt.String()))
		return time.Duration(0)
	}

	return delay
}

// StoreReceivedMessage a new message
func (service *MessageService) storeReceivedMessage(ctx context.Context, params events.MessagePhoneReceivedPayload) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message := &entities.Message{
		ID:                params.MessageID,
		Owner:             params.Owner,
		UserID:            params.UserID,
		Contact:           params.Contact,
		Content:           params.Content,
		SIM:               params.SIM,
		Encrypted:         params.Encrypted,
		Type:              entities.MessageTypeMobileOriginated,
		Status:            entities.MessageStatusReceived,
		RequestReceivedAt: params.Timestamp,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
		OrderTimestamp:    params.Timestamp,
		ReceivedAt:        &params.Timestamp,
	}

	if err := service.repository.Store(ctx, message); err != nil {
		msg := fmt.Sprintf("cannot save message with id [%s]", params.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message saved with id [%s]", message.ID))
	return message, nil
}

// HandleMessageParams are parameters for handling a message event
type HandleMessageParams struct {
	ID        uuid.UUID
	Source    string
	UserID    entities.UserID
	Timestamp time.Time
}

// HandleMessageSending handles when a message is being sent
func (service *MessageService) HandleMessageSending(ctx context.Context, params HandleMessageParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message, err := service.repository.Load(ctx, params.UserID, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot find message with id [%s]", params.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !message.IsSending() {
		msg := fmt.Sprintf("message has wrong status [%s]. expected %s", message.Status, entities.MessageStatusSending)
		return service.tracer.WrapErrorSpan(span, stacktrace.NewError(msg))
	}

	if err = service.repository.Update(ctx, message.AddSendAttempt(params.Timestamp)); err != nil {
		msg := fmt.Sprintf("cannot update message with id [%s] after sending", message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message with id [%s] updated after adding send attempt", message.ID))
	return nil
}

// HandleMessageSent handles when a message has been sent by a mobile phone
func (service *MessageService) HandleMessageSent(ctx context.Context, params HandleMessageParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message, err := service.repository.Load(ctx, params.UserID, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot find message with id [%s]", params.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if message.IsSent() || message.IsDelivered() {
		ctxLogger.Info(fmt.Sprintf("message [%s] for [%s] has already been processed with status [%s]", message.ID, message.UserID, message.Status))
		return nil
	}

	if !message.IsSending() && !message.IsExpired() && !message.IsScheduled() {
		msg := fmt.Sprintf("message has wrong status [%s]. expected [%s, %s, %s]", message.Status, entities.MessageStatusSending, entities.MessageStatusExpired, entities.MessageStatusScheduled)
		return service.tracer.WrapErrorSpan(span, stacktrace.NewError(msg))
	}

	if err = service.repository.Update(ctx, message.Sent(params.Timestamp)); err != nil {
		msg := fmt.Sprintf("cannot update message with id [%s] as sent", message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message with id [%s] has been updated to status [%s]", message.ID, message.Status))
	return nil
}

// HandleMessageFailedParams are parameters for handling a failed message event
type HandleMessageFailedParams struct {
	ID           uuid.UUID
	UserID       entities.UserID
	ErrorMessage string
	Timestamp    time.Time
}

// HandleMessageFailed handles when a message could not be sent by a mobile phone
func (service *MessageService) HandleMessageFailed(ctx context.Context, params HandleMessageFailedParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message, err := service.repository.Load(ctx, params.UserID, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot find message with id [%s]", params.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if message.IsDelivered() {
		msg := fmt.Sprintf("message has already been delivered with status [%s]", message.Status)
		return service.tracer.WrapErrorSpan(span, stacktrace.NewError(msg))
	}

	if err = service.repository.Update(ctx, message.Failed(params.Timestamp, params.ErrorMessage)); err != nil {
		msg := fmt.Sprintf("cannot update message with id [%s] as sent", message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message with id [%s] has been updated to status [%s]", message.ID, message.Status))
	return nil
}

// HandleMessageDelivered handles when a message is has been delivered by a mobile phone
func (service *MessageService) HandleMessageDelivered(ctx context.Context, params HandleMessageParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message, err := service.repository.Load(ctx, params.UserID, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot find message with id [%s]", params.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !message.IsSent() && !message.IsSending() && !message.IsExpired() && !message.IsScheduled() {
		msg := fmt.Sprintf("message has wrong status [%s]. expected [%s, %s, %s, %s]", message.Status, entities.MessageStatusSent, entities.MessageStatusScheduled, entities.MessageStatusSending, entities.MessageStatusExpired)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return nil
	}

	if err = service.repository.Update(ctx, message.Delivered(params.Timestamp)); err != nil {
		msg := fmt.Sprintf("cannot update message with id [%s] as delivered", message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message with id [%s] has been updated to status [%s]", message.ID, message.Status))
	return nil
}

// HandleMessageNotificationScheduled handles the event when the notification of a message has been scheduled
func (service *MessageService) HandleMessageNotificationScheduled(ctx context.Context, params HandleMessageParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message, err := service.repository.Load(ctx, params.UserID, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot find message with id [%s]", params.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !message.IsPending() && !message.IsExpired() && !message.IsSending() {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("received scheduled event for message with id [%s] message has status [%s]", message.ID, message.Status)))
	}

	if err = service.repository.Update(ctx, message.NotificationScheduled(params.Timestamp)); err != nil {
		msg := fmt.Sprintf("cannot update message with id [%s] as expired", message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message with id [%s] has been scheduled to send at [%s]", message.ID, message.NotificationScheduledAt.String()))
	return nil
}

// HandleMessageNotificationSent handles the event when the notification of a message has been sent
func (service *MessageService) HandleMessageNotificationSent(ctx context.Context, params HandleMessageParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message, err := service.repository.Load(ctx, params.UserID, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot find message with id [%s]", params.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.repository.Update(ctx, message.AddSendAttemptCount()); err != nil {
		msg := fmt.Sprintf("cannot update message with id [%s] as expired", message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("notification for message with id [%s] has been sent at [%s]", message.ID, params.Timestamp.String()))
	return nil
}

// HandleMessageExpired handles when a message is has been expired
func (service *MessageService) HandleMessageExpired(ctx context.Context, params HandleMessageParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message, err := service.repository.Load(ctx, params.UserID, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot find message with id [%s]", params.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !message.IsSending() && !message.IsScheduled() && !message.IsPending() {
		msg := fmt.Sprintf("message has wrong status [%s]. expected [%s, %s, %s]", message.Status, entities.MessageStatusSending, entities.MessageStatusScheduled, entities.MessageStatusPending)
		return service.tracer.WrapErrorSpan(span, stacktrace.NewError(msg))
	}

	if err = service.repository.Update(ctx, message.Expired(params.Timestamp)); err != nil {
		msg := fmt.Sprintf("cannot update message with id [%s] as expired", message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message with id [%s] has been updated to status [%s]", message.ID, message.Status))

	if !message.CanBeRescheduled() {
		return nil
	}

	event, err := service.createMessageSendRetryEvent(params.Source, &events.MessageSendRetryPayload{
		MessageID: message.ID,
		Timestamp: time.Now().UTC(),
		Contact:   message.Contact,
		Owner:     message.Owner,
		Encrypted: message.Encrypted,
		UserID:    message.UserID,
		Content:   message.Content,
		SIM:       message.SIM,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create [%s] event for expired message with ID [%s]", events.EventTypeMessageSendRetry, message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch [%s] event for message with ID [%s]", event.Type(), message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("retried sending message with ID [%s]", message.ID))
	return nil
}

// MessageScheduleExpirationParams are parameters for scheduling the expiration of a message event
type MessageScheduleExpirationParams struct {
	MessageID                 uuid.UUID
	UserID                    entities.UserID
	NotificationSentAt        time.Time
	PhoneID                   uuid.UUID
	MessageExpirationDuration time.Duration
	Source                    string
}

// ScheduleExpirationCheck schedules an event to check if a message is expired
func (service *MessageService) ScheduleExpirationCheck(ctx context.Context, params MessageScheduleExpirationParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if params.MessageExpirationDuration == 0 {
		ctxLogger.Info(fmt.Sprintf("message expiration duration not set for message [%s] using phone [%s]", params.MessageID, params.PhoneID))
		return nil
	}

	event, err := service.createMessageSendExpiredCheckEvent(params.Source, &events.MessageSendExpiredCheckPayload{
		MessageID:   params.MessageID,
		ScheduledAt: params.NotificationSentAt.Add(params.MessageExpirationDuration),
		UserID:      params.UserID,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event [%s] for message with id [%s]", events.EventTypeMessageSendExpiredCheck, params.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if _, err = service.eventDispatcher.DispatchWithTimeout(ctx, event, params.MessageExpirationDuration); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for message with ID [%s]", event.Type(), params.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("scheduled message id [%s] to expire at [%s]", params.MessageID, params.NotificationSentAt.Add(params.MessageExpirationDuration)))
	return nil
}

// MessageCheckExpired are parameters for checking if a message is expired
type MessageCheckExpired struct {
	MessageID uuid.UUID
	UserID    entities.UserID
	Source    string
}

// CheckExpired checks if a message has expired
func (service *MessageService) CheckExpired(ctx context.Context, params MessageCheckExpired) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message, err := service.repository.Load(ctx, params.UserID, params.MessageID)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		ctxLogger.Info(fmt.Sprintf("message has been deleted for userID [%s] and messageID [%s]", params.UserID, params.MessageID))
		return nil
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load message with userID [%s] and messageID [%s]", params.UserID, params.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !message.IsPending() && !message.IsSending() && !message.IsScheduled() {
		ctxLogger.Info(fmt.Sprintf("message with ID [%s] has status [%s] and is not expired", message.ID, message.Status))
		return nil
	}

	event, err := service.createMessageSendExpiredEvent(params.Source, events.MessageSendExpiredPayload{
		MessageID:        message.ID,
		Owner:            message.Owner,
		Contact:          message.Contact,
		Encrypted:        message.Encrypted,
		RequestID:        message.RequestID,
		IsFinal:          message.SendAttemptCount == message.MaxSendAttempts,
		SendAttemptCount: message.SendAttemptCount,
		UserID:           message.UserID,
		Timestamp:        time.Now().UTC(),
		Content:          message.Content,
		SIM:              message.SIM,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event [%s] for message with id [%s]", events.EventTypeMessageSendExpired, params.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for message with ID [%s]", event.Type(), params.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message [%s] has expired with status [%s]", params.MessageID, message.Status))
	return nil
}

// MessageSearchParams are parameters for searching messages
type MessageSearchParams struct {
	repositories.IndexParams
	UserID   entities.UserID
	Owners   []string
	Types    []entities.MessageType
	Statuses []entities.MessageStatus
}

// SearchMessages fetches all the messages for a user
func (service *MessageService) SearchMessages(ctx context.Context, params *MessageSearchParams) ([]*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	messages, err := service.repository.Search(ctx, params.UserID, params.Owners, params.Types, params.Statuses, params.IndexParams)
	if err != nil {
		msg := fmt.Sprintf("could not search messages with parms [%+#v]", params)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] messages with prams [%+#v]", len(messages), params))
	return messages, nil
}

func (service *MessageService) phoneSettings(ctx context.Context, userID entities.UserID, owner string) (uint, entities.SIM) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	phone, err := service.phoneService.Load(ctx, userID, owner)
	if err != nil {
		msg := fmt.Sprintf("cannot load phone for userID [%s] and owner [%s]. using default max send attempt of 2", userID, owner)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return 2, entities.SIM1
	}

	return phone.MaxSendAttemptsSanitized(), phone.SIM
}

// storeSentMessage a new message
func (service *MessageService) storeSentMessage(ctx context.Context, payload events.MessageAPISentPayload) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	timestamp := payload.RequestReceivedAt
	if payload.ScheduledSendTime != nil {
		timestamp = *payload.ScheduledSendTime
	}

	message := &entities.Message{
		ID:                payload.MessageID,
		Owner:             payload.Owner,
		Contact:           payload.Contact,
		UserID:            payload.UserID,
		Content:           payload.Content,
		RequestID:         payload.RequestID,
		SIM:               payload.SIM,
		Encrypted:         payload.Encrypted,
		ScheduledSendTime: payload.ScheduledSendTime,
		Type:              entities.MessageTypeMobileTerminated,
		Status:            entities.MessageStatusPending,
		RequestReceivedAt: payload.RequestReceivedAt,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
		MaxSendAttempts:   payload.MaxSendAttempts,
		OrderTimestamp:    timestamp,
	}

	if err := service.repository.Store(ctx, message); err != nil {
		msg := fmt.Sprintf("cannot save message with id [%s]", payload.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message saved with id [%s]", payload.MessageID))
	return message, nil
}

// storeMissedCallMessage a new message
func (service *MessageService) storeMissedCallMessage(ctx context.Context, payload *events.MessageCallMissedPayload) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message := &entities.Message{
		ID:                payload.MessageID,
		Owner:             payload.Owner,
		Contact:           payload.Contact,
		UserID:            payload.UserID,
		SIM:               payload.SIM,
		Type:              entities.MessageTypeCallMissed,
		Status:            entities.MessageStatusReceived,
		RequestReceivedAt: payload.Timestamp,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
		OrderTimestamp:    payload.Timestamp,
	}

	if err := service.repository.Store(ctx, message); err != nil {
		msg := fmt.Sprintf("cannot save missed call message with id [%s]", payload.MessageID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("missed call message saved with id [%s]", payload.MessageID))
	return message, nil
}

func (service *MessageService) createMessageSendExpiredEvent(source string, payload events.MessageSendExpiredPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessageSendExpired, source, payload)
}

func (service *MessageService) createMessageSendExpiredCheckEvent(source string, payload *events.MessageSendExpiredCheckPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessageSendExpiredCheck, source, payload)
}

func (service *MessageService) createMessageAPISentEvent(source string, payload events.MessageAPISentPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessageAPISent, source, payload)
}

func (service *MessageService) createMessagePhoneReceivedEvent(source string, payload events.MessagePhoneReceivedPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessagePhoneReceived, source, payload)
}

func (service *MessageService) createMessagePhoneSendingEvent(source string, payload events.MessagePhoneSendingPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessagePhoneSending, source, payload)
}

func (service *MessageService) createMessagePhoneSentEvent(source string, payload events.MessagePhoneSentPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessagePhoneSent, source, payload)
}

func (service *MessageService) createMessageSendFailedEvent(source string, payload events.MessageSendFailedPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessageSendFailed, source, payload)
}

func (service *MessageService) createMessagePhoneDeliveredEvent(source string, payload events.MessagePhoneDeliveredPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessagePhoneDelivered, source, payload)
}

func (service *MessageService) createMessageSendRetryEvent(source string, payload *events.MessageSendRetryPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessageSendRetry, source, payload)
}
