package main

import (
	"context"
	"fmt"
	"log"

	"github.com/carlmjohnson/requests"
	"github.com/palantir/stacktrace"
)

func main() {
	for i := 0; i < 50; i++ {
		var responsePayload string
		err := requests.
			URL("/v1/messages/send").
			Host("localhost:8000").
			Scheme("http").
			Header("x-api-key", "Uv38ByGCZU8WP18PmmIdcpVmx00QA3xNe7sEB9HixkmBhVrYaB0NhtHpHgAWeTnL").
			BodyJSON(&map[string]string{
				"content": fmt.Sprintf("testing http api sample: [%d]", i),
				"from":    "+37259139660",
				"to":      "+37253517181",
			}).
			ToString(&responsePayload).
			Fetch(context.Background())
		if err != nil {
			log.Fatal(stacktrace.Propagate(err, "cannot create json payload"))
		}

		log.Println(responsePayload)
	}
}
