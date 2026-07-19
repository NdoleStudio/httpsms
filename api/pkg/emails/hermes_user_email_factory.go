package emails

import (
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/stacktrace"
	"github.com/go-hermes/hermes/v2"
)

type hermesUserEmailFactory struct {
	factory
	config    *HermesGeneratorConfig
	generator hermes.Hermes
}

// formatBillingDate renders a date like "19 June 2026" in the user's timezone.
func formatBillingDate(t time.Time, location *time.Location) string {
	return t.In(location).Format("2 January 2006")
}

func (factory *hermesUserEmailFactory) APIKeyRotated(emailAddress string, timestamp time.Time, timezone string) (*Email, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}

	email := hermes.Email{
		Body: hermes.Body{
			Intros: []string{
				fmt.Sprintf("This is a confirmation email that your httpSMS API Key has been successfully rotated at %s.", timestamp.In(location).Format(time.RFC1123)),
			},
			Actions: []hermes.Action{
				{
					Instructions: "You can see your new API key in the httpSMS settings page.",
					Button: hermes.Button{
						Color:     "#329ef4",
						TextColor: "#FFFFFF",
						Text:      "httpSMS Settings",
						Link:      "https://httpsms.com/settings/",
					},
				},
			},
			Title:     "Hey,",
			Signature: "Cheers",
			Outros: []string{
				fmt.Sprintf("If you did not trigger this API key rotation please contact us immediately by replying to this email."),
			},
		},
	}

	html, err := factory.generator.GenerateHTML(email)
	if err != nil {
		return nil, stacktrace.Propagatef(err, "cannot generate html email")
	}

	text, err := factory.generator.GeneratePlainText(email)
	if err != nil {
		return nil, stacktrace.Propagatef(err, "cannot generate text email")
	}

	return &Email{
		ToEmail: emailAddress,
		Subject: "Your httpSMS API Key has been rotated successfully",
		HTML:    html,
		Text:    text,
	}, nil
}

// UsageLimitExceeded is the email sent when the plan limit is reached
func (factory *hermesUserEmailFactory) UsageLimitExceeded(user *entities.User, usage *entities.BillingUsage) (*Email, error) {
	email := hermes.Email{
		Body: hermes.Body{
			Intros: []string{
				fmt.Sprintf("You've reached your limit of %s messages on the %s plan, so new messages will not be processed until your usage resets.", factory.formatQuantity(user.SubscriptionName.Limit()), user.SubscriptionName),
				fmt.Sprintf("Between %s and %s you sent %s messages and received %s, for a total of %s.", formatBillingDate(usage.StartTimestamp, user.Location()), formatBillingDate(usage.EndTimestamp, user.Location()), factory.formatQuantity(usage.SentMessages), factory.formatQuantity(usage.ReceivedMessages), factory.formatQuantity(usage.TotalMessages())),
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
		return nil, stacktrace.Propagatef(err, "cannot generate html email")
	}

	text, err := factory.generator.GeneratePlainText(email)
	if err != nil {
		return nil, stacktrace.Propagatef(err, "cannot generate text email")
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
				fmt.Sprintf("This is a friendly heads-up that you've used %d%% of your monthly SMS limit on the %s plan.", percent, user.SubscriptionName),
				fmt.Sprintf("Between %s and %s you sent %s messages and received %s, for a total of %s out of your %s message limit.", formatBillingDate(usage.StartTimestamp, user.Location()), formatBillingDate(usage.EndTimestamp, user.Location()), factory.formatQuantity(usage.SentMessages), factory.formatQuantity(usage.ReceivedMessages), factory.formatQuantity(usage.TotalMessages()), factory.formatQuantity(user.SubscriptionName.Limit())),
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
		return nil, stacktrace.Propagatef(err, "cannot generate html email")
	}

	text, err := factory.generator.GeneratePlainText(email)
	if err != nil {
		return nil, stacktrace.Propagatef(err, "cannot generate text email")
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
		return nil, stacktrace.Propagatef(err, "cannot generate html email")
	}

	text, err := factory.generator.GeneratePlainText(email)
	if err != nil {
		return nil, stacktrace.Propagatef(err, "cannot generate text email")
	}

	return &Email{
		ToEmail: user.Email,
		Subject: fmt.Sprintf("⚠️ No heartbeat from android phone [%s]", factory.formatPhoneNumber(owner)),
		HTML:    html,
		Text:    text,
	}, nil
}
