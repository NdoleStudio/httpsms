package di

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/jinzhu/now"

	"github.com/uptrace/uptrace-go/uptrace"

	"github.com/NdoleStudio/httpsms/pkg/emails"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"

	"firebase.google.com/go/messaging"
	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/hirosassa/zerodriver"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/sdk/trace"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/NdoleStudio/httpsms/pkg/middlewares"
	"google.golang.org/api/option"

	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/listeners"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/gofiber/fiber/v2"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/swagger"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"

	"github.com/NdoleStudio/httpsms/pkg/handlers"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
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
	// Set location to UTC
	now.DefaultConfig = &now.Config{
		TimeLocation: time.UTC,
	}

	container = &Container{
		projectID: projectID,
		logger:    logger(3).WithService(fmt.Sprintf("%T", container)),
	}

	container.InitializeTraceProvider("0.0.1", os.Getenv("GCP_PROJECT_ID"))

	container.RegisterMessageListeners()
	container.RegisterMessageRoutes()

	container.RegisterMessageThreadRoutes()
	container.RegisterMessageThreadListeners()

	container.RegisterHeartbeatRoutes()
	container.RegisterHeartbeatListeners()

	container.RegisterUserRoutes()
	container.RegisterUserListeners()

	container.RegisterPhoneRoutes()

	container.RegisterEventRoutes()

	container.RegisterNotificationListeners()

	container.RegisterBillingRoutes()
	container.RegisterBillingListeners()

	container.RegisterWebhookRoutes()
	container.RegisterWebhookListeners()

	container.RegisterLemonsqueezyRoutes()

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

	app.Use(middlewares.OtelTraceContext(container.Tracer(), container.Logger(), "X-Cloud-Trace-Context", os.Getenv("GCP_PROJECT_ID")))

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

	config := &gorm.Config{}
	if isLocal() {
		config = &gorm.Config{Logger: container.GormLogger()}
	}

	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), config)
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

	if err = db.AutoMigrate(&entities.HeartbeatMonitor{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.HeartbeatMonitor{})))
	}

	if err = db.AutoMigrate(&entities.User{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.User{})))
	}

	if err = db.AutoMigrate(&entities.Phone{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.Phone{})))
	}

	if err = db.AutoMigrate(&entities.PhoneNotification{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.PhoneNotification{})))
	}

	if err = db.AutoMigrate(&entities.BillingUsage{}); err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot migrate %T", &entities.BillingUsage{})))
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

// CloudTasksClient creates a new instance of cloudtasks.Client
func (container *Container) CloudTasksClient() (client *cloudtasks.Client) {
	container.logger.Debug(fmt.Sprintf("creating %T", client))

	client, err := cloudtasks.NewClient(context.Background(), option.WithCredentialsJSON(container.FirebaseCredentials()))
	if err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, "cannot initialize cloud tasks client"))
	}

	return client
}

// EventsQueueConfiguration creates a new instance of services.PushQueueConfig
func (container *Container) EventsQueueConfiguration() (config services.PushQueueConfig) {
	container.logger.Debug(fmt.Sprintf("creating %T", config))

	return services.PushQueueConfig{
		UserAPIKey:       os.Getenv("EVENTS_QUEUE_USER_API_KEY"),
		Name:             os.Getenv("EVENTS_QUEUE_NAME"),
		UserID:           entities.UserID(os.Getenv("EVENTS_QUEUE_USER_ID")),
		ConsumerEndpoint: os.Getenv("EVENTS_QUEUE_ENDPOINT"),
	}
}

// EventsQueue creates a new instance of services.PushQueue
func (container *Container) EventsQueue() (queue services.PushQueue) {
	container.logger.Debug("creating events services.PushQueue")

	return services.NewGooglePushQueue(
		container.Logger(),
		container.Tracer(),
		container.CloudTasksClient(),
		container.EventsQueueConfiguration(),
	)
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
		container.PhoneService(),
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

// BillingHandler creates a new instance of handlers.BillingHandler
func (container *Container) BillingHandler() (h *handlers.BillingHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", h))
	return handlers.NewBillingHandler(
		container.Logger(),
		container.Tracer(),
		container.BillingHandlerValidator(),
		container.BillingService(),
	)
}

// WebhookHandler creates a new instance of handlers.WebhookHandler
func (container *Container) WebhookHandler() (h *handlers.WebhookHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", h))
	return handlers.NewWebhookHandler(
		container.Logger(),
		container.Tracer(),
		container.WebhookService(),
		container.WebhookHandlerValidator(),
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

// BillingHandlerValidator creates a new instance of validators.BillingHandlerValidator
func (container *Container) BillingHandlerValidator() (validator *validators.BillingHandlerValidator) {
	container.logger.Debug(fmt.Sprintf("creating %T", validator))
	return validators.NewBillingHandlerValidator(
		container.Logger(),
		container.Tracer(),
	)
}

// WebhookHandlerValidator creates a new instance of validators.WebhookHandlerValidator
func (container *Container) WebhookHandlerValidator() (validator *validators.WebhookHandlerValidator) {
	container.logger.Debug(fmt.Sprintf("creating %T", validator))
	return validators.NewWebhookHandlerValidator(
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
		container.EventsQueue(),
		container.EventsQueueConfiguration(),
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

// BillingUsageRepository creates a new instance of repositories.BillingUsageRepository
func (container *Container) BillingUsageRepository() (repository repositories.BillingUsageRepository) {
	container.logger.Debug("creating GORM repositories.BillingUsageRepository")
	return repositories.NewGormBillingUsageRepository(
		container.Logger(),
		container.Tracer(),
		container.DB(),
	)
}

// WebhookRepository creates a new instance of repositories.WebhookRepository
func (container *Container) WebhookRepository() (repository repositories.WebhookRepository) {
	container.logger.Debug("creating GORM repositories.WebhookRepository")
	return repositories.NewGormWebhookRepository(
		container.Logger(),
		container.Tracer(),
		container.DB(),
	)
}

// PhoneNotificationRepository creates a new instance of repositories.PhoneNotificationRepository
func (container *Container) PhoneNotificationRepository() (repository repositories.PhoneNotificationRepository) {
	container.logger.Debug("creating GORM repositories.PhoneNotificationRepository")
	return repositories.NewGormPhoneNotificationRepository(
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

// HeartbeatMonitorRepository creates a new instance of repositories.HeartbeatMonitorRepository
func (container *Container) HeartbeatMonitorRepository() (repository repositories.HeartbeatMonitorRepository) {
	container.logger.Debug("creating GORM repositories.HeartbeatMonitorRepository")
	return repositories.NewGormHeartbeatMonitorRepository(
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
		container.HeartbeatMonitorRepository(),
		container.EventDispatcher(),
	)
}

// BillingService creates a new instance of services.BillingService
func (container *Container) BillingService() (service *services.BillingService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewBillingService(
		container.Logger(),
		container.Tracer(),
		container.BillingUsageRepository(),
	)
}

// WebhookService creates a new instance of services.WebhookService
func (container *Container) WebhookService() (service *services.WebhookService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewWebhookService(
		container.Logger(),
		container.Tracer(),
		container.HTTPClient("webhook"),
		container.WebhookRepository(),
	)
}

// HTTPClient creates a new http.Client
func (container *Container) HTTPClient(name string) *http.Client {
	container.logger.Debug(fmt.Sprintf("creating %s %T", name, http.DefaultClient))
	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: container.RetryHTTPRoundTripper(),
	}
}

//func (container *Container) HTTPRoundTripper(name string) http.RoundTripper {
//	container.logger.Debug(fmt.Sprintf("Debug: initializing %s %T", name, http.DefaultTransport))
//	return otelroundtripper.New(
//		otelroundtripper.WithName(name),
//		otelroundtripper.WithParent(container.RetryHTTPRoundTripper()),
//		otelroundtripper.WithMeter(global.Meter(os.Getenv("NAMESPACE"))),
//		otelroundtripper.WithAttributes(initializers.InitializeOtelResources(container.Version, container.Namespace).Attributes()...),
//	)
//}

// RetryHTTPRoundTripper creates a retryable http.RoundTripper
func (container *Container) RetryHTTPRoundTripper() http.RoundTripper {
	container.logger.Debug(fmt.Sprintf("initializing retry %T", http.DefaultTransport))
	retryClient := retryablehttp.NewClient()
	retryClient.Logger = container.Logger()
	return retryClient.StandardClient().Transport
}

// PhoneService creates a new instance of services.PhoneService
func (container *Container) PhoneService() (service *services.PhoneService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewPhoneService(
		container.Logger(),
		container.Tracer(),
		container.PhoneRepository(),
		container.EventDispatcher(),
	)
}

// MarketingService creates a new instance of services.MarketingService
func (container *Container) MarketingService() (service *services.MarketingService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewMarketingService(
		container.Logger(),
		container.Tracer(),
		container.FirebaseAuthClient(),
		os.Getenv("SENDGRID_API_KEY"),
		os.Getenv("SENDGRID_LIST_ID"),
	)
}

// UserService creates a new instance of services.UserService
func (container *Container) UserService() (service *services.UserService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewUserService(
		container.Logger(),
		container.Tracer(),
		container.UserRepository(),
		container.Mailer(),
		container.UserEmailFactory(),
		container.MarketingService(),
		container.LemonsqueezyClient(),
	)
}

// Mailer creates a new instance of emails.Mailer
func (container *Container) Mailer() (mailer emails.Mailer) {
	container.logger.Debug("creating emails.Mailer")
	return emails.NewSMTPEmailService(
		container.Tracer(),
		emails.SMTPConfig{
			FromName:  os.Getenv("SMTP_FROM_NAME"),
			FromEmail: os.Getenv("SMTP_FROM_EMAIL"),
			Username:  os.Getenv("SMTP_USERNAME"),
			Password:  os.Getenv("SMTP_PASSWORD"),
			Hostname:  os.Getenv("SMTP_HOST"),
			Port:      os.Getenv("SMTP_PORT"),
		},
	)
}

// UserEmailFactory creates a new instance of emails.UserEmailFactory
func (container *Container) UserEmailFactory() (factory emails.UserEmailFactory) {
	container.logger.Debug("creating emails.UserEmailFactory")
	return emails.NewHermesUserEmailFactory(&emails.HermesGeneratorConfig{
		AppURL:     os.Getenv("APP_URL"),
		AppName:    os.Getenv("APP_NAME"),
		AppLogoURL: os.Getenv("APP_LOGO_URL"),
	})
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

// EventsHandler creates a new instance of handlers.EventsHandler
func (container *Container) EventsHandler() (handler *handlers.EventsHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", handler))

	return handlers.NewEventsHandler(
		container.Logger(),
		container.Tracer(),
		container.EventsQueueConfiguration(),
		container.EventDispatcher(),
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

// LemonsqueezyService creates a new instance of services.LemonsqueezyService
func (container *Container) LemonsqueezyService() (service *services.LemonsqueezyService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewLemonsqueezyService(
		container.Logger(),
		container.Tracer(),
		container.UserRepository(),
		container.EventDispatcher(),
	)
}

// LemonsqueezyHandler creates a new instance of handlers.LemonsqueezyHandler
func (container *Container) LemonsqueezyHandler() (handler *handlers.LemonsqueezyHandler) {
	container.logger.Debug(fmt.Sprintf("creating %T", handler))

	return handlers.NewLemonsqueezyHandler(
		container.Logger(),
		container.Tracer(),
		container.LemonsqueezyService(),
		container.LemonsqueezyHandlerValidator(),
	)
}

// LemonsqueezyHandlerValidator creates a new instance of validators.LemonsqueezyHandlerValidator
func (container *Container) LemonsqueezyHandlerValidator() (validator *validators.LemonsqueezyHandlerValidator) {
	container.logger.Debug(fmt.Sprintf("creating %T", validator))
	return validators.NewLemonsqueezyHandlerValidator(
		container.Logger(),
		container.Tracer(),
		container.LemonsqueezyClient(),
	)
}

// LemonsqueezyClient creates a new instance of lemonsqueezy.Client
func (container *Container) LemonsqueezyClient() (client *lemonsqueezy.Client) {
	container.logger.Debug(fmt.Sprintf("creating %T", client))
	return lemonsqueezy.New(
		lemonsqueezy.WithHTTPClient(container.HTTPClient("lemonsqueezy")),
		lemonsqueezy.WithAPIKey(os.Getenv("LEMONSQUEEZY_API_KEY")),
		lemonsqueezy.WithSigningSecret(os.Getenv("LEMONSQUEEZY_SIGNING_SECRET")),
	)
}

// RegisterLemonsqueezyRoutes registers routes for the /project-settings prefix
func (container *Container) RegisterLemonsqueezyRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.LemonsqueezyHandler{}))
	container.LemonsqueezyHandler().RegisterRoutes(container.App())
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

// RegisterNotificationListeners registers event listeners for listeners.PhoneNotificationListener
func (container *Container) RegisterNotificationListeners() {
	container.logger.Debug(fmt.Sprintf("registering listners for %T", listeners.PhoneNotificationListener{}))
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

// RegisterUserListeners registers event listeners for listeners.UserListener
func (container *Container) RegisterUserListeners() {
	container.logger.Debug(fmt.Sprintf("registering listners for %T", listeners.UserListener{}))
	_, routes := listeners.NewUserListener(
		container.Logger(),
		container.Tracer(),
		container.UserService(),
	)

	for event, handler := range routes {
		container.EventDispatcher().Subscribe(event, handler)
	}
}

// RegisterBillingListeners registers event listeners for listeners.BillingListener
func (container *Container) RegisterBillingListeners() {
	container.logger.Debug(fmt.Sprintf("registering listeners for %T", listeners.BillingListener{}))
	_, routes := listeners.NewBillingListener(
		container.Logger(),
		container.Tracer(),
		container.BillingService(),
	)

	for event, handler := range routes {
		container.EventDispatcher().Subscribe(event, handler)
	}
}

// RegisterWebhookListeners registers event listeners for listeners.WebhookListener
func (container *Container) RegisterWebhookListeners() {
	container.logger.Debug(fmt.Sprintf("registering listeners for %T", listeners.WebhookListener{}))
	_, routes := listeners.NewWebhookListener(
		container.Logger(),
		container.Tracer(),
		container.WebhookService(),
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
		container.PhoneService(),
	)
}

// NotificationService creates a new instance of services.PhoneNotificationService
func (container *Container) NotificationService() (service *services.PhoneNotificationService) {
	container.logger.Debug(fmt.Sprintf("creating %T", service))
	return services.NewNotificationService(
		container.Logger(),
		container.Tracer(),
		container.FirebaseMessagingClient(),
		container.PhoneRepository(),
		container.PhoneNotificationRepository(),
		container.EventDispatcher(),
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

// RegisterBillingRoutes registers routes for the /billing prefix
func (container *Container) RegisterBillingRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.BillingHandler{}))
	container.BillingHandler().RegisterRoutes(container.AuthRouter())
}

// RegisterWebhookRoutes registers routes for the /webhooks prefix
func (container *Container) RegisterWebhookRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.WebhookHandler{}))
	container.WebhookHandler().RegisterRoutes(container.App(), container.AuthenticatedMiddleware())
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

// RegisterEventRoutes registers routes for the /events prefix
func (container *Container) RegisterEventRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", &handlers.EventsHandler{}))
	container.EventsHandler().RegisterRoutes(container.AuthRouter())
}

// RegisterSwaggerRoutes registers routes for swagger
func (container *Container) RegisterSwaggerRoutes() {
	container.logger.Debug(fmt.Sprintf("registering %T routes", swagger.HandlerDefault))
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

// InitializeTraceProvider initializes the open telemetry trace provider
func (container *Container) InitializeTraceProvider(version string, namespace string) func() {
	if isLocal() {
		return container.initializeUptraceProvider(version, namespace)
	}
	return container.initializeGoogleTraceProvider(version, namespace)
}

func (container *Container) initializeGoogleTraceProvider(version string, namespace string) func() {
	container.logger.Debug("initializing google trace provider")

	exporter, err := cloudtrace.New(cloudtrace.WithProjectID(os.Getenv("GCP_PROJECT_ID")))
	if err != nil {
		container.logger.Fatal(stacktrace.Propagate(err, "cannot create cloud trace exporter"))
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(container.InitializeOtelResources(version, namespace)),
	)

	otel.SetTracerProvider(tp)

	return func() {
		_ = exporter.Shutdown(context.Background())
	}
}

func (container *Container) initializeUptraceProvider(version string, namespace string) (flush func()) {
	container.logger.Debug("initializing uptrace provider")
	// Configure OpenTelemetry with sensible defaults.
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(os.Getenv("UPTRACE_DSN")),
		uptrace.WithServiceName(namespace),
		uptrace.WithServiceVersion(version),
	)

	// Send buffered spans and free resources.
	return func() {
		err := uptrace.Shutdown(context.Background())
		if err != nil {
			container.logger.Error(err)
		}
	}
}

// InitializeOtelResources initializes open telemetry resources
func (container *Container) InitializeOtelResources(version string, namespace string) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(namespace),
		semconv.ServiceNamespaceKey.String(namespace),
		semconv.ServiceVersionKey.String(version),
		semconv.ServiceInstanceIDKey.String(hostName()),
		attribute.String("service.environment", os.Getenv("ENV")),
	)
}

func logger(skipFrameCount int) telemetry.Logger {
	fields := map[string]string{
		"pid":      strconv.Itoa(os.Getpid()),
		"hostname": hostName(),
	}

	return telemetry.NewZerologLogger(
		os.Getenv("GCP_PROJECT_ID"),
		fields,
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

func hostName() string {
	h, err := os.Hostname()
	if err != nil {
		h = strconv.Itoa(os.Getpid())
	}
	return h
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
