package main

import (
	"context"
	"log"
	"os"
	"time"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/httpsms/pkg/di"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	container := di.NewContainer(os.Getenv("GCP_PROJECT_ID"), "")
	client := container.FirebaseMessagingClient()

	result, err := client.Send(context.Background(), &messaging.Message{
		Data: map[string]string{
			"KEY_HEARTBEAT_ID": time.Now().UTC().Format(time.RFC3339),
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
		Token: os.Getenv("FIREBASE_TOKEN"),
	})
	if err != nil {
		container.Logger().Fatal(err)
	}

	container.Logger().Info(result)
}
