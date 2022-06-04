package di

import (
	"log"

	"github.com/joho/godotenv"
)

// LoadEnv will read your .env file(s) and load them into ENV for this process.
func LoadEnv(filenames ...string) {
	err := godotenv.Load(filenames...)
	if err != nil {
		log.Fatalf("Fatal: cannot load .env file: %v", err)
	}
}
