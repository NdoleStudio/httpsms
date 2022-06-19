package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/palantir/stacktrace"
)

func main() {
	client := http.Client{}
	for i := 0; i < 100; i++ {
		payload, err := json.Marshal(map[string]string{
			"content": fmt.Sprintf("testing http api sample: [%d]", i),
			"from":    "+37259139660",
			"to":      "+37253517181",
		})
		if err != nil {
			log.Fatal(stacktrace.Propagate(err, "cannot create json payload"))
		}
		response, err := client.Post("https://api.httpsms.com/v1/messages/send", "application/json", bytes.NewBuffer(payload))
		if err != nil {
			log.Fatal(stacktrace.Propagate(err, "cannot perform http request"))
		}

		if response.StatusCode != http.StatusOK {
			log.Fatal(stacktrace.NewError(fmt.Sprintf("status code [%d] is different from expected [%d]", response.StatusCode, http.StatusOK)))
		}

		responsePayload, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(stacktrace.NewError(fmt.Sprintf("cannot read response")))
		}

		log.Println(string(responsePayload))
	}
}
