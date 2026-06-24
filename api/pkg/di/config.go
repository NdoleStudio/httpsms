package di

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// LoadEnv will read your .env file(s) and load them into ENV for this process.
func LoadEnv(filenames ...string) {
	err := godotenv.Load(filenames...)
	if err != nil {
		log.Fatalf("Fatal: cannot load .env file: %v", err)
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func splitCommaEnv(key, defaultValue string) []string {
	value := getEnvWithDefault(key, defaultValue)
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
