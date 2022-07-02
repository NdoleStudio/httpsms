package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"

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

	err = createSmsMessage()
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, "cannot create sms message"))
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

func createSmsMessage() error {
	payload, err := json.Marshal(map[string]string{
		"from":    os.Getenv("PHONE_NUMBER"),
		"to":      os.Getenv("PHONE_NUMBER"),
		"content": fmt.Sprintf("[%s] random message [%d]", time.Now().String(), rand.Int()),
	})
	if err != nil {
		return stacktrace.Propagate(err, "cannot convert message to map")
	}

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPost, "https://api.httpsms.com/v1/messages/send", bytes.NewBuffer(payload))
	if err != nil {
		return stacktrace.Propagate(err, "cannot do http request")
	}

	req.Header = http.Header{
		"Content-Type": {"application/json"},
		"X-API-Key":    {os.Getenv("API_KEY")},
	}

	response, err := client.Do(req)
	if err != nil {
		return stacktrace.Propagate(err, "cannot do http request")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return stacktrace.Propagate(err, "cannot read response body")
	}

	if response.StatusCode >= 400 {
		return stacktrace.NewError(fmt.Sprintf("[%s] %s", response.StatusCode, string(body)))
	}

	spew.Dump(string(body))
	return nil
}
