package main

import (
	"context"
	"fmt"
	"log"

	"github.com/NdoleStudio/http-sms-manager/pkg/di"
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
	dispatcher := container.EventDispatcher()

	events, err := eventRepo.FetchAll(context.Background())
	if err != nil {
		logger.Fatal(stacktrace.Propagate(err, "cannot fetch all events"))
	}
	logger.Info(fmt.Sprintf("fetched %d cloudevents", len(*events)))

	// create and start new bar
	bar := pb.StartNew(len(*events))
	for _, event := range *events {
		dispatcher.Publish(context.Background(), event)
		bar.Increment()
	}

	bar.Finish()
}
