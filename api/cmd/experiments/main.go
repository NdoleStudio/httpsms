package main

import (
	"log"

	"github.com/NdoleStudio/httpsms/pkg/di"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	_ = di.NewContainer("http-sms", "")
}
