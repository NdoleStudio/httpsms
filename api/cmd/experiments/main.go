package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/carlmjohnson/requests"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	loadTest()
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
