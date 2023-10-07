package emails

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/matcornic/hermes/v2"
	"github.com/palantir/stacktrace"
)

type hermesNotificationEmailFactory struct {
	config    *HermesGeneratorConfig
	generator hermes.Hermes
}

// NewHermesNotificationEmailFactory creates a new instance of the UserEmailFactory
func NewHermesNotificationEmailFactory(config *HermesGeneratorConfig) NotificationEmailFactory {
	return &hermesNotificationEmailFactory{
		config:    config,
		generator: config.Generator(),
	}
}

func (factory *hermesNotificationEmailFactory) MessageExpired(user *entities.User, messageID uuid.UUID, owner string, contact string, content string) (*Email, error) {
	email := hermes.Email{
		Body: hermes.Body{
			Title: "Hello",
			Intros: []string{
				fmt.Sprintf("The SMS message which you sent to %s has expired at %s and you will need to resend this message.", owner, user.UserTimeString(time.Now())),
			},
			Dictionary: []hermes.Entry{
				{"ID", messageID.String()},
				{"From", owner},
				{"To", contact},
				{"Message", content},
			},
			Actions: []hermes.Action{
				{
					Instructions: "Messages expire because we couldn't connect with your mobile phone to send the outgoing SMS. You can fix this by making sure your phone is connected to the internet and also connect your phone to the charger all the time since Android may kill the httpSMS app if it has been active for a very long time so save phone battery.",
					Button: hermes.Button{
						Color:     "#329ef4",
						TextColor: "#FFFFFF",
						Text:      "View Messages",
						Link:      "https://httpsms.com/threads",
					},
				},
			},
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
		Subject: "🔔 Your SMS message has expired on httpSMS",
		HTML:    html,
		Text:    text,
	}, nil
}

func (factory *hermesNotificationEmailFactory) MessageFailed(user *entities.User, messageID uuid.UUID, owner, contact, content, reason string) (*Email, error) {
	email := hermes.Email{
		Body: hermes.Body{
			Title: "Hello",
			Intros: []string{
				fmt.Sprintf("The SMS message which you sent to %s has failed at %s and you will need to resend this message.", owner, user.UserTimeString(time.Now())),
			},
			Dictionary: []hermes.Entry{
				{"ID", messageID.String()},
				{"From", owner},
				{"To", contact},
				{"Message", content},
				{"Failure Reason", reason},
			},
			Actions: []hermes.Action{
				{
					Instructions: "Check the default SMS messaging app on your phone to find out the exact reason why the message failed. Usually messages fail because the httpSMS app phone has been un-installed or it is not active. Logout and login again on the mobile app on your Android phone and retry sending the SMS.",
					Button: hermes.Button{
						Color:     "#329ef4",
						TextColor: "#FFFFFF",
						Text:      "View Messages",
						Link:      "https://httpsms.com/threads",
					},
				},
			},
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
		Subject: "⚡ Your SMS message has failed on httpSMS",
		HTML:    html,
		Text:    text,
	}, nil
}
