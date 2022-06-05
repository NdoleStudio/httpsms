package main

import (
	"os"

	_ "github.com/NdoleStudio/http-sms-manager/docs"
	"github.com/NdoleStudio/http-sms-manager/pkg/di"
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
// @schemes  https
// @BasePath /v1
func main() {
	if len(os.Args) == 1 {
		di.LoadEnv()
	}

	container := di.NewContainer("http-sms")
	container.Logger().Info(container.App().Listen(":8000").Error())
}
