package di

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"firebase.google.com/go/messaging"
	"github.com/hirosassa/zerodriver"
	"github.com/rs/zerolog"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/NdoleStudio/http-sms-manager/pkg/middlewares"
	"google.golang.org/api/option"

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
		logger:    logger(3).WithService(fmt.Sprintf("%T", container)),
	}

	container.RegisterMessageListeners()
	container.RegisterMessageRoutes()

	container.RegisterMessageThreadRoutes()
	container.RegisterMessageThreadListeners()

	container.RegisterHeartbeatRoutes()
	container.RegisterHeartbeatListeners()

	container.RegisterUserRoutes()

	container.RegisterPhoneRoutes()

	container.RegisterNotificationListeners()

	// this has to be last since it registers the /* route
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

	app.Use(middlewares.BearerAuth(container.Logger(), container.Tracer(), container.FirebaseAuthClient()))
	app.Use(middlewares.APIKeyAuth(container.Logger(), container.Tracer(), container.UserRepository()))

	container.app = app
	return app
}

// AuthenticatedMiddleware creates a new instance of middlewares.Authenticated
func (container *Container) AuthenticatedMiddleware() fiber.Handler {
	container.logger.Debug("creating middlewares.Authenticated")
	return middlewares.Authenticated(container.Tracer())
}

// AuthRouter creates router for authenticated requests
func (container *Container) AuthRouter() fiber.Router {
	container.logger.Debug("creating authRouter")
	return container.App().Group("v1").Use(container.AuthenticatedMiddleware())
}

// Logger creates a new instance of telemetry.Logger
func (container *Container) Logger(skipFrameCount ...int) telemetry.Logger {
	container.logger.Debug("creating telemetry.Logger")
	if len(skipFrameCount) > 0 {
		return logger(skipFrameCount[0])
	}
	return logger(3)
}

// GormLogger creates a new instance of gormLogger.Interface
func (container *Container) GormLogger() gormLogger.Interface {
	container.logger.Debug("creating gormLogger.Interface")
	return telemetry.NewGormLogger(
		container.Tracer(),
		container.Logger(6),
	)
}

// DB creates an instance of gorm.DB if it has not been created already
func (container *Container) DB() (db *gorm.DB) {
	if container.db != nil {
		return container.db
	}

	container.logger.Debug(fmt.Sprintf("creating %T", db))

	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{Logger: container.GormLogger()})
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

	if err = db.AutoMigrate(&entities.Heartbeat{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.Heartbeat{})))
	}

	if err = db.AutoMigrate(&entities.User{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.User{})))
	}

	if err = db.AutoMigrate(&entities.Phone{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.Phone{})))
	}

	return container.db
}

// FirebaseApp creates a new instance of firebase.App
func (container *Container) FirebaseApp() (app *firebase.App) {
	container.logger.Debug(fmt.Sprintf("creating %T", app))
	app, err := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(container.FirebaseCredentials()))
	if err != nil {
		msg := "cannot initialize firebase application"
		container.logger.Fatal(stacktrace.Propagate(err, msg))
	}
	return app
}

// FirebaseAuthClient creates a new instance of auth.Client
func (container *Container) FirebaseAuthClient() (client *auth.Client) {
	container.logger.Debug(fmt.Sprintf("creating %T", client))
	authClient, err := container.FirebaseApp().Auth(context.Background())
	if err != nil {
		msg := "cannot initialize firebase auth client"
		container.logger.Fatal(stacktrace.Propagate(err, msg))
	}
	return authClient
}

// FirebaseMessagingClient creates a new instance of messaging.Client
func (container *Container) FirebaseMessagingClient() (client *messaging.Client) {
	container.logger.Debug(fmt.Sprintf("creating %T", client))
	messagingClient, err := container.FirebaseApp().Messaging(context.Background())
	if err != nil {
		msg := "cannot initialize firebase messaging client"
		container.logger.Fatal(stacktrace.Propagate(err, msg))
	}
	return messagingClient
}

// FirebaseCredentials returns firebase credentials as bytes.
func (container *Container) FirebaseCredentials() []byte {
	container.logger.Debug("creating firebase credentials")
	return []byte(os.Getenv("FIREBASE_CREDENTIALS"))
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

// HeartbeatHandler creates a new instance of handlers.HeartbeatHandler
func (container *Container) HeartbeatHandler() (h *handlers.HeartbeatHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", h))
	return handlers.NewHeartbeatHandler(
		container.Logger(),
		container.Tracer(),
		container.HeartbeatHandlerValidator(),
		container.HeartbeatService(),
	)
}

// HeartbeatHandlerValidator creates a new instance of validators.HeartbeatHandlerValidator
func (container *Container) HeartbeatHandlerValidator() (validator *validators.HeartbeatHandlerValidator) {
	container.logger.Debug(fmt.Sprintf("creating %T", validator))
	return validators.NewHeartbeatHandlerValidator(
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

// PhoneHandlerValidator creates a new instance of validators.PhoneHandlerValidator
func (container *Container) PhoneHandlerValidator() (validator *validators.PhoneHandlerValidator) {
	container.logger.Debug(fmt.Sprintf("creating %T", validator))
	return validators.NewPhoneHandlerValidator(
		container.Logger(),
		container.Tracer(),
	)
}

// UserHandlerValidator creates a new instance of validators.UserHandlerValidator
func (container *Container) UserHandlerValidator() (validator *validators.UserHandlerValidator) {
	container.logger.Debug(fmt.Sprintf("creating %T", validator))
	return validators.NewUserHandlerValidator(
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

// PhoneRepository creates a new instance of repositories.PhoneRepository
func (container *Container) PhoneRepository() (repository repositories.PhoneRepository) {
	container.logger.Debug("creating GORM repositories.PhoneRepository")
	return repositories.NewGormPhoneRepository(
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

// HeartbeatService creates a new instance of services.HeartbeatService
func (container *Container) HeartbeatService() (service *services.HeartbeatService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewHeartbeatService(
		container.Logger(),
		container.Tracer(),
		container.HeartbeatRepository(),
	)
}

// PhoneService creates a new instance of services.PhoneService
func (container *Container) PhoneService() (service *services.PhoneService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewPhoneService(
		container.Logger(),
		container.Tracer(),
		container.PhoneRepository(),
	)
}

// UserService creates a new instance of services.UserService
func (container *Container) UserService() (service *services.UserService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewUserService(
		container.Logger(),
		container.Tracer(),
		container.UserRepository(),
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

// UserHandler creates a new instance of handlers.MessageHandler
func (container *Container) UserHandler() (handler *handlers.UserHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", handler))
	return handlers.NewUserHandler(
		container.Logger(),
		container.Tracer(),
		container.UserHandlerValidator(),
		container.UserService(),
	)
}

// PhoneHandler creates a new instance of handlers.PhoneHandler
func (container *Container) PhoneHandler() (handler *handlers.PhoneHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", handler))
	return handlers.NewPhoneHandler(
		container.Logger(),
		container.Tracer(),
		container.PhoneService(),
		container.PhoneHandlerValidator(),
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

// RegisterNotificationListeners registers event listeners for listeners.NotificationListener
func (container *Container) RegisterNotificationListeners() {
	container.logger.Debug(fmt.Sprintf("registering listners for %T", listeners.NotificationListener{}))
	_, routes := listeners.NewNotificationListener(
		container.Logger(),
		container.Tracer(),
		container.NotificationService(),
	)

	for event, handler := range routes {
		container.EventDispatcher().Subscribe(event, handler)
	}
}

// RegisterHeartbeatListeners registers event listeners for listeners.HeartbeatListener
func (container *Container) RegisterHeartbeatListeners() {
	container.logger.Debug(fmt.Sprintf("registering listners for %T", listeners.HeartbeatListener{}))
	_, routes := listeners.NewHeartbeatListener(
		container.Logger(),
		container.Tracer(),
		container.HeartbeatService(),
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

// NotificationService creates a new instance of services.NotificationService
func (container *Container) NotificationService() (service *services.NotificationService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewNotificationService(
		container.Logger(),
		container.Tracer(),
		container.FirebaseMessagingClient(),
		container.PhoneRepository(),
	)
}

// RegisterMessageRoutes registers routes for the /messages prefix
func (container *Container) RegisterMessageRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.MessageHandler{}))
	container.MessageHandler().RegisterRoutes(container.AuthRouter())
}

// RegisterMessageThreadRoutes registers routes for the /message-threads prefix
func (container *Container) RegisterMessageThreadRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.MessageThreadHandler{}))
	container.MessageThreadHandler().RegisterRoutes(container.AuthRouter())
}

// RegisterHeartbeatRoutes registers routes for the /heartbeats prefix
func (container *Container) RegisterHeartbeatRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.HeartbeatHandler{}))
	container.HeartbeatHandler().RegisterRoutes(container.AuthRouter())
}

// RegisterPhoneRoutes registers routes for the /phone prefix
func (container *Container) RegisterPhoneRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.PhoneHandler{}))
	container.PhoneHandler().RegisterRoutes(container.AuthRouter())
}

// RegisterUserRoutes registers routes for the /users prefix
func (container *Container) RegisterUserRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.UserHandler{}))
	container.UserHandler().RegisterRoutes(container.AuthRouter())
}

// RegisterSwaggerRoutes registers routes for swagger
func (container *Container) RegisterSwaggerRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.MessageHandler{}))
	container.App().Get("/*", swagger.HandlerDefault)
}

// HeartbeatRepository registers a new instance of repositories.HeartbeatRepository
func (container *Container) HeartbeatRepository() repositories.HeartbeatRepository {
	container.logger.Debug("creating GORM repositories.HeartbeatRepository")
	return repositories.NewGormHeartbeatRepository(
		container.Logger(),
		container.Tracer(),
		container.DB(),
	)
}

// UserRepository registers a new instance of repositories.UserRepository
func (container *Container) UserRepository() repositories.UserRepository {
	container.logger.Debug("creating GORM repositories.UserRepository")
	return repositories.NewGormUserRepository(
		container.Logger(),
		container.Tracer(),
		container.DB(),
	)
}

func logger(skipFrameCount int) telemetry.Logger {
	hostname, _ := os.Hostname()
	fields := map[string]string{
		"pid":      strconv.Itoa(os.Getpid()),
		"hostname": hostname,
	}

	return telemetry.NewZerologLogger(
		os.Getenv("GCP_PROJECT_ID"),
		fields,
		3,
		logDriver(skipFrameCount),
		nil,
	)
}

func logDriver(skipFrameCount int) *zerodriver.Logger {
	if isLocal() {
		return consoleLogger(skipFrameCount)
	}
	return jsonLogger(skipFrameCount)
}

func jsonLogger(skipFrameCount int) *zerodriver.Logger {
	logLevel := zerolog.DebugLevel
	zerolog.SetGlobalLevel(logLevel)

	// See: https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#LogSeverity
	logLevelSeverity := map[zerolog.Level]string{
		zerolog.TraceLevel: "DEFAULT",
		zerolog.DebugLevel: "DEBUG",
		zerolog.InfoLevel:  "INFO",
		zerolog.WarnLevel:  "WARNING",
		zerolog.ErrorLevel: "ERROR",
		zerolog.PanicLevel: "CRITICAL",
		zerolog.FatalLevel: "CRITICAL",
	}

	zerolog.LevelFieldName = "severity"
	zerolog.LevelFieldMarshalFunc = func(l zerolog.Level) string {
		return logLevelSeverity[l]
	}
	zerolog.TimestampFieldName = "time"
	zerolog.TimeFieldFormat = time.RFC3339Nano

	zl := zerolog.New(os.Stderr).With().Timestamp().CallerWithSkipFrameCount(skipFrameCount).Logger()
	return &zerodriver.Logger{Logger: &zl}
}

func consoleLogger(skipFrameCount int) *zerodriver.Logger {
	l := zerolog.New(
		zerolog.ConsoleWriter{
			Out: os.Stderr,
		}).With().Timestamp().CallerWithSkipFrameCount(skipFrameCount).Logger()
	return &zerodriver.Logger{
		Logger: &l,
	}
}

func isLocal() bool {
	return os.Getenv("ENV") == "local"
}
