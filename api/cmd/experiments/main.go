package main

import (
	"context"
	"log"

	"github.com/NdoleStudio/httpsms/pkg/di"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/joho/godotenv"
	"github.com/palantir/stacktrace"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	container := di.NewLiteContainer()
	mailer := container.Mailer()

	factory := container.UserEmailFactory()

	user := &entities.User{
		Email:            "arnoldewin@gmail.com",
		SubscriptionName: entities.SubscriptionNameUltraMonthly,
	}

	mail, err := factory.UsageLimitExceeded(user)
	if err != nil {
		container.Logger().Fatal(stacktrace.Propagate(err, "cannot create email"))
	}

	if err = mailer.Send(context.Background(), mail); err != nil {
		container.Logger().Fatal(stacktrace.Propagate(err, "cannot send email"))
	}

	container.Logger().Info("email sent")
}
