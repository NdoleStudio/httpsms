package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"firebase.google.com/go/messaging"
	"github.com/joho/godotenv"
	"github.com/palantir/stacktrace"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	opt := option.WithCredentialsFile("serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, "cannot create firebase app"))
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, "cannot create messaging client"))
	}

	result, err := client.Send(context.Background(), &messaging.Message{
		Data: map[string]string{
			"hello": "world",
		},
		Token: os.Getenv("FCM_TOKEN"),
	})
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, "cannot send FCM event"))
	}

	log.Println(fmt.Sprintf("sent event with response [%s]", result))
}
