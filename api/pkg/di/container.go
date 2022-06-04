package di

import (
	"fmt"
	"os"

	"github.com/NdoleStudio/http-sms-manager/pkg/handlers"
	"github.com/NdoleStudio/http-sms-manager/pkg/services"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/NdoleStudio/http-sms-manager/pkg/validators"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// Container is used to resolve services at runtime
type Container struct {
	projectID string
	logger    telemetry.Logger
}

// NewContainer creates a new dependency injection container
func NewContainer(projectID string) (container *Container) {
	return &Container{
		projectID: projectID,
		logger:    logger().WithService(fmt.Sprintf("%T", container)),
	}
}

// Logger creates a new instance of telemetry.Logger
func (container Container) Logger() telemetry.Logger {
	container.logger.Debug("creating telemetry.Logger")
	return logger()
}

// Tracer creates a new instance of telemetry.Tracer
func (container Container) Tracer() (t telemetry.Tracer) {
	container.logger.Debug("creating telemetry.Tracer")
	return telemetry.NewOtelLogger(
		container.projectID,
		container.Logger(),
	)
}

// MessageHandlerValidator creates a new instance of validators.MessageHandlerValidator
func (container *Container) MessageHandlerValidator() (validator *validators.MessageHandlerValidator) {
	container.logger.Debug(fmt.Sprintf("creating %T", validator))
	return validators.NewMessageHandlerValidator(
		container.Logger(),
		container.Tracer(),
	)
}

// MessageService creates a new instance of services.MessageService
func (container *Container) MessageService() (service *services.MessageService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewMessageService(
		container.Logger(),
		container.Tracer(),
	)
}

// MessageHandler creates a new instance of handlers.MessageHandler
func (container *Container) MessageHandler() (handler *handlers.MessageHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", handler))
	return handlers.NewMessageHandler(
		container.Logger(),
		container.Tracer(),
		container.MessageHandlerValidator(),
		container.MessageService(),
	)
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
