package main

import (
	"log"

	_ "github.com/NdoleStudio/http-sms-manager/docs"
	"github.com/NdoleStudio/http-sms-manager/pkg/di"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/swagger"
)

// @title       HTTP SMS API
// @version     1.0
// @description API to send SMS messages using android [SmsManager](https://developer.android.com/reference/android/telephony/SmsManager) via HTTP
//
// @contact.name  HTTP SMS Support
// @contact.email supportd@httpsms.com
//
// @license.name MIT
// @license.url  https://raw.githubusercontent.com/NdoleStudio/http-sms-manager/main/LICENSE
//
// @host     api.httpsms.com
// @BasePath /v1
func main() {
	di.LoadEnv()

	app := fiber.New()

	app.Use(logger.New())

	container := di.NewContainer("http-sms")

	apiV1 := app.Group("v1")
	apiV1.Post("/messages/send", container.MessageHandler().Send)

	app.Get("/*", swagger.HandlerDefault)

	log.Println(app.Listen(":8000"))
}
