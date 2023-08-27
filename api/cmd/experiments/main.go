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

	"github.com/carlmjohnson/requests"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	container := di.NewLiteContainer()
	cache := container.RistrettoCache()
	logger := container.Logger()

	for i := 0; i < 100; i++ {
		result := cache.SetWithTTL(fmt.Sprintf("how are you %d", i), entities.AuthUser{
			ID:    "dasfdfds",
			Email: "arnoldewin@gmail.com",
		}, 1, 2*time.Hour)
		logger.Info(fmt.Sprintf("cached [%t]", result))
	}
}

func loadTest() {
	wg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(count int) {
			sendSMS(count)
			wg.Done()
		}(i)

	}
	wg.Wait()
}

func sendSMS(count int) {
	var response string
	err := requests.URL(os.Getenv("BASIC_URL")).
		BodyJSON(map[string]any{
			"content": fmt.Sprintf("Hello, World [%d]", count),
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
