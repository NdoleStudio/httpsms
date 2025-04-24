package handlers

import (
	"os"

	"github.com/carlmjohnson/requests"
	_ "github.com/joho/godotenv/autoload" // import USER_API_KEY from .env file
)

func testClient() *requests.Builder {
	return requests.URL("http://localhost:8000").
		Header("x-api-key", os.Getenv("USER_API_KEY"))
}
