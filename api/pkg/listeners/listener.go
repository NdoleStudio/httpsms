package listeners

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type listener struct {
	repository repositories.EventListenerLogRepository
}

func (listener listener) handlerSignature(handler any, event cloudevents.Event) string {
	return fmt.Sprintf("%s.%T", event.Type(), handler)
}
