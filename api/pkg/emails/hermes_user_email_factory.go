package emails

import (
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/matcornic/hermes/v2"
	"github.com/palantir/stacktrace"
)

type hermesUserEmailFactory struct {
	config    *HermesGeneratorConfig
	generator hermes.Hermes
}

// NewHermesUserEmailFactory creates a new instance of the UserEmailFactory
func NewHermesUserEmailFactory(config *HermesGeneratorConfig) UserEmailFactory {
	return &hermesUserEmailFactory{
		config:    config,
		generator: config.Generator(),
	}
}

// PhoneDead is the email sent to a user when their phone is dead
func (factory *hermesUserEmailFactory) PhoneDead(user *entities.User, lastHeartbeatTimestamp time.Time, owner string) (*Email, error) {
	email := hermes.Email{
		Body: hermes.Body{
			Intros: []string{
				fmt.Sprintf("We haven't received any heartbeat event from your mobile phone <b>%s</b> since %s.", owner, lastHeartbeatTimestamp.Format(time.RFC822)),
				fmt.Sprintf("Check if the mobile phone is powered on and if it has stable internet connection."),
			},
			Title:     "Hello 🤚",
			Signature: "Cheers",
			Outros: []string{
				fmt.Sprintf("Don't hesitate to contact us by replying to this email."),
			},
		},
	}

	html, err := factory.generator.GenerateHTML(email)
	if err != nil {
		return nil, stacktrace.Propagate(err, "cannot generate html email")
	}

	text, err := factory.generator.GeneratePlainText(email)
	if err != nil {
		return nil, stacktrace.Propagate(err, "cannot generate text email")
	}

	return &Email{
		ToEmail: user.Email,
		Subject: fmt.Sprintf("⚠ No heartbeat from android phone [%s]", owner),
		HTML:    html,
		Text:    text,
	}, nil
}
