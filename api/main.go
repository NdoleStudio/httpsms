package main

import (
	_ "github.com/NdoleStudio/http-sms-manager/docs"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/swagger"
	"log"
)

// @title HTTP SMS API
// @version 1.0
// @description Sample API to send messages using android sms manager via HTTP
// @termsOfService http://swagger.io/terms/
// @contact.name HTTP SMS Support
// @contact.email supportd@httpsms.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host api.httpsms.com
// @BasePath /
func main() {
	app := fiber.New()

	// log requests/responses
	app.Use(logger.New())

	app.Get("/*", swagger.HandlerDefault)

	log.Println(app.Listen(":8000"))
}
