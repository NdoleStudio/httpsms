package emails

import (
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/matcornic/hermes/v2"
	"github.com/palantir/stacktrace"
)

type hermesUserEmailFactory struct {
	factory
	config    *HermesGeneratorConfig
	generator hermes.Hermes
}

// UsageLimitExceeded is the email sent when the plan limit is reached
func (factory *hermesUserEmailFactory) UsageLimitExceeded(user *entities.User) (*Email, error) {
	email := hermes.Email{
		Body: hermes.Body{
			Intros: []string{
				fmt.Sprintf("You have exceeded your limit of %d messages on your %s plan.", user.SubscriptionName.Limit(), user.SubscriptionName),
			},
			Actions: []hermes.Action{
				{
					Instructions: "Click the button below to upgrade your plan and continue sending more messages",
					Button: hermes.Button{
						Color:     "#329ef4",
						TextColor: "#FFFFFF",
						Text:      "UPGRADE YOUR PLAN",
						Link:      "https://httpsms.com/billing",
					},
				},
			},
			Title:     "Hey,",
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
		Subject: "⚠️ You have exceeded your plan limit",
		HTML:    html,
		Text:    text,
	}, nil
}

// UsageLimitAlert is the email sent when the plan limit is reached
func (factory *hermesUserEmailFactory) UsageLimitAlert(user *entities.User, usage *entities.BillingUsage) (*Email, error) {
	percent := (usage.TotalMessages() * 100) / user.SubscriptionName.Limit()
	email := hermes.Email{
		Body: hermes.Body{
			Intros: []string{
				fmt.Sprintf("This is a friendly notification that you have exceeded %d%% of your monthly SMS limit on the %s plan.", percent, user.SubscriptionName),
				fmt.Sprintf("You have sent %d messages and received %d messages using httpSMS this month.", usage.SentMessages, usage.ReceivedMessages),
			},
			Actions: []hermes.Action{
				{
					Instructions: "Click the button below to upgrade your plan so you can continue without any disruptions",
					Button: hermes.Button{
						Color:     "#329ef4",
						TextColor: "#FFFFFF",
						Text:      "UPGRADE YOUR PLAN",
						Link:      "https://httpsms.com/billing",
					},
				},
			},
			Title:     "Hey,",
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
		Subject: fmt.Sprintf("⚠️ %d%% Usage Limit Alert", percent),
		HTML:    html,
		Text:    text,
	}, nil
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
	location, err := time.LoadLocation(user.Timezone)
	if err != nil {
		location = time.UTC
	}

	email := hermes.Email{
		Body: hermes.Body{
			Intros: []string{
				fmt.Sprintf("We haven't received any heartbeat event from android  phone %s since %s.", factory.formatPhoneNumber(owner), lastHeartbeatTimestamp.In(location).Format(time.RFC1123)),
				fmt.Sprintf("Check if the mobile phone is powered on and if it has stable internet connection."),
			},
			Actions: []hermes.Action{
				{
					Instructions: "Check your heartbeat events on httpSMS",
					Button: hermes.Button{
						Color:     "#329ef4",
						TextColor: "#FFFFFF",
						Text:      "HEARTBEATS",
						Link:      fmt.Sprintf("https://httpsms.com/heartbeats/%s", owner),
					},
				},
			},
			Title:     "Hey,",
			Signature: "Cheers",
			Outros: []string{
				fmt.Sprintf("Don't hesitate to contact us by replying to this email. You can disable this email notification on https://httpsms.com/settings/#email-notifications"),
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
		Subject: fmt.Sprintf("⚠️ No heartbeat from android phone [%s]", factory.formatPhoneNumber(owner)),
		HTML:    html,
		Text:    text,
	}, nil
}
