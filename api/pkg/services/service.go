package services

import (
	"regexp"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/nyaruka/phonenumbers"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

type service struct{}

func (service *service) createEvent(eventType string, source string, payload any) (cloudevents.Event, error) {
	event := cloudevents.NewEvent()

	event.SetSource(source)
	event.SetType(eventType)
	event.SetTime(time.Now().UTC())
	event.SetID(uuid.New().String())

	if err := event.SetData(cloudevents.ApplicationJSON, payload); err != nil {
		return event, stacktrace.Propagate(err, "cannot encode %T [%#+v] as JSON", payload, payload)
	}

	return event, nil
}

func (service *service) getFormattedNumber(ctxLogger telemetry.Logger, phoneNumber string) string {
	matched, err := regexp.MatchString("^\\+?[1-9]\\d{9,14}$", phoneNumber)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "error while matching phoneNumber [%s] with regex [%s]", phoneNumber, "^\\+?[1-9]\\d{10,14}$"))
		return phoneNumber
	}
	if !matched {
		return phoneNumber
	}

	number, err := phonenumbers.Parse(phoneNumber, phonenumbers.UNKNOWN_REGION)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot parse number [%s]", phoneNumber))
		return phoneNumber
	}

	return phonenumbers.Format(number, phonenumbers.INTERNATIONAL)
}
