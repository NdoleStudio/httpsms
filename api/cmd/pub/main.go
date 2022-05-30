package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(1)
	}()

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Password: os.Getenv("REDIS_PASSWORD"),
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	})

	for i := 0; ; i++ {
		fmt.Printf("publishing [%d]\n", i)
		if err = rdb.Publish(context.Background(), "httpsms", i).Err(); err != nil {
			log.Fatal(err)
		}
		time.Sleep(5 * time.Second)
	}
}
