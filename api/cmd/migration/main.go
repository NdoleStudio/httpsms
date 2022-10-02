package main

import (
	"context"
	"fmt"
	"log"

	"github.com/NdoleStudio/httpsms/pkg/di"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/cheggaaa/pb/v3"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/palantir/stacktrace"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	container := di.NewContainer("http-sms")
	eventRepo := container.EventRepository()
	logger := container.Logger()

	cloudEvents, err := eventRepo.FetchAll(context.Background())
	if err != nil {
		logger.Fatal(stacktrace.Propagate(err, "cannot fetch all cloudEvents"))
	}
	logger.Info(fmt.Sprintf("fetched %d cloudevents", len(*cloudEvents)))

	// create and start new bar
	bar := pb.StartNew(len(*cloudEvents))
	for _, event := range *cloudEvents {
		if event.Type() == events.EventTypeMessageAPISent {
			migrateSentEvent(logger, eventRepo, event)
		}
		if event.Type() == events.EventTypeMessagePhoneReceived {
			migrateReceivedEvent(logger, eventRepo, event)
		}
		bar.Increment()
	}

	bar.Finish()
}

func migrateReceivedEvent(logger telemetry.Logger, repository repositories.EventRepository, event cloudevents.Event) {
	var payload events.MessagePhoneReceivedPayloadV1
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		logger.Fatal(stacktrace.Propagate(err, msg))
	}

	if payload.MessageID != uuid.Nil {
		payloadV2 := events.MessagePhoneReceivedPayload{
			MessageID: payload.MessageID,
			UserID:    payload.UserID,
			Owner:     payload.Owner,
			Contact:   payload.Contact,
			Timestamp: payload.Timestamp,
			Content:   payload.Content,
		}

		if err := event.SetData(cloudevents.ApplicationJSON, payloadV2); err != nil {
			logger.Fatal(stacktrace.Propagate(err, "cannot set payload v2 data"))
		}

		if err := repository.Save(context.Background(), event); err != nil {
			logger.Fatal(stacktrace.Propagate(err, "cannot save event after updating"))
		}
	}
}

func migrateSentEvent(logger telemetry.Logger, repository repositories.EventRepository, event cloudevents.Event) {
	var payload events.MessageAPISentPayloadV1
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		logger.Fatal(stacktrace.Propagate(err, msg))
	}

	if payload.MessageID != uuid.Nil {
		payloadV2 := events.MessageAPISentPayload{
			MessageID:         payload.MessageID,
			UserID:            payload.UserID,
			Owner:             payload.Owner,
			MaxSendAttempts:   2,
			Contact:           payload.Contact,
			RequestReceivedAt: payload.RequestReceivedAt,
			Content:           payload.Content,
		}

		if err := event.SetData(cloudevents.ApplicationJSON, payloadV2); err != nil {
			logger.Fatal(stacktrace.Propagate(err, "cannot set payloadv2 data"))
		}

		if err := repository.Save(context.Background(), event); err != nil {
			logger.Fatal(stacktrace.Propagate(err, "cannot save event after updating"))
		}
	}
}
