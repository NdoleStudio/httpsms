package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/di"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"

	"github.com/carlmjohnson/requests"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	container := di.NewLiteContainer()
	repo := container.Integration3CXRepository()

	err = repo.Save(context.Background(), &entities.Integration3CX{
		ID:         uuid.MustParse("b0b1acdc-69dd-4aee-8277-ba4adc5d2e90"),
		UserID:     "XtABz6zdeFMoBLoltz6SREDvRSh2",
		WebhookURL: "https://lagomtest.3cx.com.au/sms/generic/155e125bf7874f8fae5adbcaac49fdf8",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err != nil {
		container.Logger().Fatal(err)
	}
}

func loadTest() {
	wg := sync.WaitGroup{}
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func(count int) {
			sendSMS(fmt.Sprintf("[%d] In the quiet of the night, the stars above whisper secrets of the universe. We, mere stardust, seek meaning in their cosmic dance, yearning to unlock the mysteries of existence that stretch far beyond our earthly bounds.", count))
			sendSMS(fmt.Sprintf("[%d] Hello, World", count))
			wg.Done()
		}(i)

	}
	wg.Wait()
}

func sendSMS(content string) {
	var response string
	err := requests.URL(os.Getenv("BASIC_URL")).
		BodyJSON(map[string]any{
			"content": content,
			"from":    os.Getenv("BASIC_FROM"),
			"to":      os.Getenv("BASIC_TO"),
		}).
		BasicAuth(os.Getenv("BASIC_USERNAME"), os.Getenv("BASIC_PASSWORD")).
		ToString(&response).
		Fetch(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s\n", response)
}
