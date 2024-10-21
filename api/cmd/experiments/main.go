package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/palantir/stacktrace"

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
	logger := container.Logger()

	logger.Info("Starting experiments")
	deleteContacts(container)
}

func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}

func deleteContacts(container *di.Container) {
	sendgrid := container.MarketingService()
	logger := container.Logger()

	b, err := os.ReadFile("28462979_873d41e1-cd34-4782-8992-0762ed247667.csv") // just pass the file name
	if err != nil {
		logger.Fatal(stacktrace.Propagate(err, "cannot read file"))
	}

	lines := strings.Split(string(b), "\n")[1:]
	var contacts []string
	for _, line := range lines {
		if len(line) >= 17 {
			contacts = append(contacts, strings.ReplaceAll(strings.Split(line, ",")[17], "\"", ""))
		}
	}

	chunks := chunkBy(contacts, 100)
	for _, chunk := range chunks {
		err = sendgrid.DeleteContacts(context.Background(), chunk)
		if err != nil {
			logger.Fatal(err)
		}
	}
}

func text3CX() {
	container := di.NewLiteContainer()
	repo := container.Integration3CXRepository()

	err := repo.Save(context.Background(), &entities.Integration3CX{
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
			wg.Done()
		}(i)

	}
	wg.Wait()
}

func sendSingle() {
	payload, err := json.Marshal(map[string]any{
		"content":   fmt.Sprintf("This is a test text message"),
		"from":      os.Getenv("HTTPSMS_FROM"),
		"to":        os.Getenv("HTTPSMS_TO"),
		"encrypted": false,
	})
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, "cannot marshal payload"))
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.httpsms.com/v1/messages/send", bytes.NewBuffer(payload))
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, "cannot create request"))
	}

	req.Header.Add("x-api-key", os.Getenv("HTTPSMS_KEY"))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, "cannot send request"))
	}

	fmt.Printf("client: got response!\n")
	fmt.Printf("client: status code: %d\n", res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, "could not read response body"))
	}

	fmt.Printf("client: response body: %s\n", resBody)
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
