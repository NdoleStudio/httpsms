package main

import (
	"context"
	"fmt"
	"log"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/listeners"

	"github.com/NdoleStudio/httpsms/pkg/di"
	"github.com/cheggaaa/pb/v3"
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

	listener, _ := listeners.NewBillingListener(
		container.Logger(),
		container.Tracer(),
		container.BillingService(),
	)

	cloudEvents, err := eventRepo.FetchAll(context.Background())
	if err != nil {
		logger.Fatal(stacktrace.Propagate(err, "cannot fetch all events"))
	}
	logger.Info(fmt.Sprintf("fetched %d cloudevents", len(*cloudEvents)))

	// create and start new bar
	bar := pb.StartNew(len(*cloudEvents))
	for _, event := range *cloudEvents {
		if event.Type() == events.EventTypeMessageAPISent {
			if err := listener.OnMessageAPISent(context.Background(), event); err != nil {
				logger.Fatal(stacktrace.Propagate(err, "cannot register api sent event"))
			}
		}
		if event.Type() == events.EventTypeMessagePhoneReceived {
			if err := listener.OnMessagePhoneReceived(context.Background(), event); err != nil {
				logger.Fatal(stacktrace.Propagate(err, "cannot register api received event"))
			}
		}
		bar.Increment()
	}

	bar.Finish()
}
