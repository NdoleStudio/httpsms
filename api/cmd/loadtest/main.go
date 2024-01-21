package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/joho/godotenv"

	"github.com/carlmjohnson/requests"
	"github.com/palantir/stacktrace"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	fmt.Printf("\n\n%s\n\n", decode("LTa1M0I0nDKNEOIQHFc0uEaRX3AnlZY="))
	sendSingle()
}

func bulkSend() {
	var to []string
	for i := 0; i < 100; i++ {
		to = append(to, os.Getenv("HTTPSMS_TO_BULK"))
	}

	var responsePayload string
	err := requests.
		URL("/v1/messages/bulk-send").
		Host("api.httpsms.com").
		Header("x-api-key", os.Getenv("HTTPSMS_KEY_BULK")).
		BodyJSON(&map[string]any{
			"content":    fmt.Sprintf("Bulk Load Test [%s]", time.Now().Format(time.RFC850)),
			"from":       os.Getenv("HTTPSMS_FROM_BULK"),
			"to":         to,
			"request_id": fmt.Sprintf("load-%s", uuid.NewString()),
		}).
		ToString(&responsePayload).
		Fetch(context.Background())
	if err != nil {
		log.Println(responsePayload)
		log.Fatal(stacktrace.Propagate(err, "cannot create request"))
	}
	log.Println(responsePayload)
}

func sendSingle() {
	for i := 0; i < 1; i++ {
		var responsePayload string
		err := requests.
			URL("/v1/messages/send").
			Host("leading-puma-internal.ngrok-free.app").
			Header("x-api-key", os.Getenv("HTTPSMS_KEY")).
			BodyJSON(&map[string]any{
				"content":    encrypt("This is a test text message"),
				"from":       os.Getenv("HTTPSMS_FROM"),
				"to":         os.Getenv("HTTPSMS_FROM"),
				"encrypted":  true,
				"request_id": fmt.Sprintf("load-%s-%d", uuid.NewString(), i),
			}).
			ToString(&responsePayload).
			Fetch(context.Background())
		if err != nil {
			log.Fatal(stacktrace.Propagate(err, "cannot create json payload"))
		}
		log.Println(responsePayload)
	}
}

func encrypt(value string) string {
	key := sha256.Sum256([]byte("Password123"))
	iv := make([]byte, 16)
	_, err := rand.Read(iv)
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, "cannot generate iv"))
	}
	c := ase256(value, key[:], iv)
	fmt.Println("iv", base64.StdEncoding.EncodeToString(iv))
	fmt.Println("cypher", base64.StdEncoding.EncodeToString(c))
	fmt.Println("cypher+iv", base64.StdEncoding.EncodeToString(append(iv, c...)))
	return base64.StdEncoding.EncodeToString(append(iv, c...))
}

func decode(value string) string {
	content, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		log.Fatal(err)
	}

	key := sha256.Sum256([]byte(os.Getenv("HTTPSMS_ENCRYPTION_KEY")))
	iv := content[:16]

	return ase256Decode(content[16:], key[:], iv)
}

func ase256(plaintext string, key []byte, bIV []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatal(err)
	}

	text := []byte(plaintext)

	stream := cipher.NewCFBEncrypter(block, bIV)
	cypher := make([]byte, len(text))
	stream.XORKeyStream(cypher, text)

	return cypher
}

func ase256Decode(cipherText []byte, key []byte, iv []byte) (decryptedString string) {
	// Create a new AES cipher with the key and encrypted message
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatal(err)
	}

	// Decrypt the message
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)
	return string(cipherText)
}
