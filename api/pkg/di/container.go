package di

import (
	"fmt"
	"os"

	"github.com/NdoleStudio/http-sms-manager/pkg/handlers"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// Container is used to resolve services at runtime
type Container struct {
	logger         telemetry.Logger
	messageHandler *handlers.MessageHandler
}

// NewContainer creates a new dependency injection container
func NewContainer() (container *Container) {
	return &Container{
		logger: logger().WithService(fmt.Sprintf("%T", container)),
	}
}

// Logger creates a new instance of telemetry.Logger
func (container Container) Logger() (l telemetry.Logger) {
	container.logger.Debug(fmt.Sprintf("creating %T", l))
	return logger()
}

// MessageHandler creates a new instance of handlers.MessageHandler
func (container *Container) MessageHandler() (handler *handlers.MessageHandler) {
	if container.messageHandler != nil {
		return container.messageHandler
	}

	container.logger.Debug(fmt.Sprintf("creating %T", handler))

	return handlers.NewMessageHandler()
}

func logger() telemetry.Logger {
	hostname, _ := os.Hostname()
	fields := fiber.Map{
		"pid":      os.Getpid(),
		"hostname": hostname,
	}
	return telemetry.NewZerologLogger(
		zerolog.New(
			zerolog.ConsoleWriter{
				Out: os.Stderr,
			},
		).With().Fields(fields).Timestamp().CallerWithSkipFrameCount(3),
	)
}
