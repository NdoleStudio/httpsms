package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/NdoleStudio/httpsms/docs"
	"github.com/NdoleStudio/httpsms/pkg/di"
	"github.com/NdoleStudio/stacktrace"
	"github.com/gofiber/fiber/v3"
)

// Version is injected at runtime
var Version string

// @title       httpSMS API Reference
// @version     1.0
// @description Use your Android phone to send and receive SMS messages via a simple programmable API with end-to-end encryption.
//
// @contact.name  support@httpsms.com
// @contact.email support@httpsms.com
//
// @license.name AGPL-3.0
// @license.url  https://raw.githubusercontent.com/NdoleStudio/http-sms-manager/main/LICENSE
//
// @host     api.httpsms.com
// @schemes  https
// @BasePath /v1
//
// @securitydefinitions.apikey ApiKeyAuth
// @in header
// @name x-api-Key
func main() {
	if len(os.Args) == 1 {
		di.LoadEnv()
	}

	if host := strings.TrimSpace(os.Getenv("SWAGGER_HOST")); len(host) > 0 {
		docs.SwaggerInfo.Host = host
	}
	if len(Version) > 0 {
		docs.SwaggerInfo.Version = Version
	}

	container := di.NewContainer(os.Getenv("GCP_PROJECT_ID"), Version)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := container.App().Listen(
		fmt.Sprintf("%s:%s", os.Getenv("APP_HOST"), os.Getenv("APP_PORT")),
		fiber.ListenConfig{GracefulContext: ctx, ShutdownTimeout: 5 * time.Second},
	); err != nil {
		container.Logger().Fatal(stacktrace.Propagatef(err, "cannot listen for HTTP requests"))
	}
}
