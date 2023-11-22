package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/joho/godotenv"

	"github.com/carlmjohnson/requests"
	"github.com/palantir/stacktrace"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	bulkSend()
}

func bulkSend() {
	var to []string
	for i := 0; i < 100; i++ {
		to = append(to, os.Getenv("HTTPSMS_TO"))
	}

	var responsePayload string
	err := requests.
		URL("/v1/messages/send").
		Host("api.httpsms.com").
		// Host("localhost:8000").
		// Scheme("http").
		Header("x-api-key", os.Getenv("HTTPSMS_KEY")).
		BodyJSON(&map[string]any{
			"content":    fmt.Sprintf("Bulk Load Test [%s]", time.Now().Format(time.RFC850)),
			"from":       os.Getenv("HTTPSMS_FROM_BULK"),
			"to":         to,
			"request_id": fmt.Sprintf("load-%s", uuid.NewString()),
		}).
		ToString(&responsePayload).
		Fetch(context.Background())
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, "cannot create json payload"))
	}
	log.Println(responsePayload)
}

func sendSingle() {
	for i := 0; i < 100; i++ {
		var responsePayload string
		err := requests.
			URL("/v1/messages/send").
			Host("api.httpsms.com").
			// Host("localhost:8000").
			// Scheme("http").
			Header("x-api-key", os.Getenv("HTTPSMS_KEY")).
			BodyJSON(&map[string]string{
				"content":    fmt.Sprintf("Load Test[%d]", i),
				"from":       os.Getenv("HTTPSMS_FROM"),
				"to":         os.Getenv("HTTPSMS_TO"),
				"request_id": fmt.Sprintf("load-%s-%d", uuid.NewString(), i),
			}).
			ToString(&responsePayload).
			Fetch(context.Background())
		if err != nil {
			log.Fatal(stacktrace.Propagate(err, "cannot create json payload"))
		}
		log.Println(responsePayload)
	}
}
