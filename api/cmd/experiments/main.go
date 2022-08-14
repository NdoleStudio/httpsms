package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/di"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	container := di.NewContainer("http-sms")

	log.Println("finished creating container")

	err = container.UserService().SendPhoneDeadEmail(context.Background(), &services.UserSendPhoneDeadEmailParams{
		UserID:                 "XtABz6zdeFMoBLoltz6SREDvRSh2",
		PhoneID:                uuid.MustParse("fb78a476-ae9f-48e4-b644-0e2ee7d83b26"),
		Owner:                  os.Getenv("PHONE_NUMBER"),
		LastHeartbeatTimestamp: time.Now().UTC().Add(-2 * time.Minute),
	})

	if err != nil {
		log.Fatal(err)
	}
}
