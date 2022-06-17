package di

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/listeners"
	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
	"github.com/NdoleStudio/http-sms-manager/pkg/services"
	"github.com/gofiber/fiber/v2"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/swagger"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"

	"github.com/NdoleStudio/http-sms-manager/pkg/handlers"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/NdoleStudio/http-sms-manager/pkg/validators"
	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	gormLogger "gorm.io/gorm/logger"
)

// Container is used to resolve services at runtime
type Container struct {
	projectID       string
	db              *gorm.DB
	app             *fiber.App
	eventDispatcher *services.EventDispatcher
	logger          telemetry.Logger
}

// NewContainer creates a new dependency injection container
func NewContainer(projectID string) (container *Container) {
	container = &Container{
		projectID: projectID,
		logger:    logger().WithService(fmt.Sprintf("%T", container)),
	}

	container.RegisterMessageListeners()
	container.RegisterMessageRoutes()

	container.RegisterMessageThreadRoutes()
	container.RegisterMessageThreadListeners()

	container.RegisterSwaggerRoutes()

	return container
}

// App creates a new instance of fiber.App
func (container *Container) App() (app *fiber.App) {
	if container.app != nil {
		return container.app
	}

	container.logger.Debug(fmt.Sprintf("creating %T", app))

	app = fiber.New()

	if os.Getenv("APP_HTTP_LOGGER") == "true" {
		app.Use(fiberLogger.New())
	}

	// Default config
	app.Use(cors.New())

	container.app = app
	return app
}

// Logger creates a new instance of telemetry.Logger
func (container *Container) Logger() telemetry.Logger {
	container.logger.Debug("creating telemetry.Logger")
	return logger()
}

// DB creates an instance of gorm.DB if it has not been created already
func (container *Container) DB() (db *gorm.DB) {
	if container.db != nil {
		return container.db
	}

	gl := gormLogger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), gormLogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormLogger.Info,
		IgnoreRecordNotFoundError: false,
		Colorful:                  true,
	})

	container.logger.Debug(fmt.Sprintf("creating %T", db))
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{Logger: gl})
	if err != nil {
		container.logger.Fatal(err)
	}
	container.db = db

	container.logger.Debug(fmt.Sprintf("Running migrations for %T", db))

	if err = db.AutoMigrate(&entities.Message{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.Message{})))
	}

	if err = db.AutoMigrate(&repositories.GormEvent{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &repositories.GormEvent{})))
	}

	if err = db.AutoMigrate(&entities.EventListenerLog{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.EventListenerLog{})))
	}

	if err = db.AutoMigrate(&entities.MessageThread{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.MessageThread{})))
	}

	return container.db
}

// Tracer creates a new instance of telemetry.Tracer
func (container *Container) Tracer() (t telemetry.Tracer) {
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

// MessageThreadHandler creates a new instance of handlers.MessageThreadHandler
func (container *Container) MessageThreadHandler() (h *handlers.MessageThreadHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", h))
	return handlers.NewMessageThreadHandler(
		container.Logger(),
		container.Tracer(),
		container.MessageThreadHandlerValidator(),
		container.MessageThreadService(),
	)
}

// MessageThreadHandlerValidator creates a new instance of validators.MessageThreadHandlerValidator
func (container *Container) MessageThreadHandlerValidator() (validator *validators.MessageThreadHandlerValidator) {
	container.logger.Debug(fmt.Sprintf("creating %T", validator))
	return validators.NewMessageThreadHandlerValidator(
		container.Logger(),
		container.Tracer(),
	)
}

// EventDispatcher creates a new instance of services.EventDispatcher
func (container *Container) EventDispatcher() (dispatcher *services.EventDispatcher) {
	if container.eventDispatcher != nil {
		return container.eventDispatcher
	}

	container.logger.Debug(fmt.Sprintf("creating %T", dispatcher))
	dispatcher = services.NewEventDispatcher(
		container.Logger(),
		container.Tracer(),
		container.EventRepository(),
	)

	container.eventDispatcher = dispatcher
	return dispatcher
}

// MessageRepository creates a new instance of repositories.MessageRepository
func (container *Container) MessageRepository() (repository repositories.MessageRepository) {
	container.logger.Debug("creating GORM repositories.MessageRepository")
	return repositories.NewGormMessageRepository(
		container.Logger(),
		container.Tracer(),
		container.DB(),
	)
}

// MessageThreadRepository creates a new instance of repositories.MessageThreadRepository
func (container *Container) MessageThreadRepository() (repository repositories.MessageThreadRepository) {
	container.logger.Debug("creating GORM repositories.MessageThreadRepository")
	return repositories.NewGormMessageThreadRepository(
		container.Logger(),
		container.Tracer(),
		container.DB(),
	)
}

// EventRepository creates a new instance of repositories.EventRepository
func (container *Container) EventRepository() (repository repositories.EventRepository) {
	container.logger.Debug("creating GORM repositories.EventRepository")
	return repositories.NewGormEventRepository(
		container.Logger(),
		container.Tracer(),
		container.DB(),
	)
}

// EventListenerLogRepository creates a new instance of repositories.EventListenerLogRepository
func (container *Container) EventListenerLogRepository() (repository repositories.EventListenerLogRepository) {
	container.logger.Debug("creating GORM repositories.EventListenerLogRepository")
	return repositories.NewGormEventListenerLogRepository(
		container.Logger(),
		container.Tracer(),
		container.DB(),
	)
}

// MessageThreadService creates a new instance of services.MessageService
func (container *Container) MessageThreadService() (service *services.MessageThreadService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewMessageThreadService(
		container.Logger(),
		container.Tracer(),
		container.MessageThreadRepository(),
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

// RegisterMessageListeners registers event listeners for listeners.MessageListener
func (container *Container) RegisterMessageListeners() {
	container.logger.Debug(fmt.Sprintf("registering listners for %T", listeners.MessageListener{}))
	_, routes := listeners.NewMessageListener(
		container.Logger(),
		container.Tracer(),
		container.MessageService(),
		container.EventListenerLogRepository(),
	)

	for event, handler := range routes {
		container.EventDispatcher().Subscribe(event, handler)
	}
}

// RegisterMessageThreadListeners registers event listeners for listeners.MessageThreadListener
func (container *Container) RegisterMessageThreadListeners() {
	container.logger.Debug(fmt.Sprintf("registering listners for %T", listeners.MessageThreadListener{}))
	_, routes := listeners.NewMessageThreadListener(
		container.Logger(),
		container.Tracer(),
		container.MessageThreadService(),
		container.EventListenerLogRepository(),
	)

	for event, handler := range routes {
		container.EventDispatcher().Subscribe(event, handler)
	}
}

// MessageService creates a new instance of services.MessageService
func (container *Container) MessageService() (service *services.MessageService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewMessageService(
		container.Logger(),
		container.Tracer(),
		container.MessageRepository(),
		container.EventDispatcher(),
	)
}

// RegisterMessageRoutes registers routes for the /messages prefix
func (container *Container) RegisterMessageRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.MessageHandler{}))
	container.MessageHandler().RegisterRoutes(container.App().Group("v1"))
}

// RegisterMessageThreadRoutes registers routes for the /message-threads prefix
func (container *Container) RegisterMessageThreadRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.MessageThreadHandler{}))
	container.MessageThreadHandler().RegisterRoutes(container.App().Group("v1"))
}

// RegisterSwaggerRoutes registers routes for swagger
func (container *Container) RegisterSwaggerRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.MessageHandler{}))
	container.App().Get("/*", swagger.HandlerDefault)
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
