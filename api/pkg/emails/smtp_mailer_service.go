package emails

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	mail "github.com/jordan-wright/email"
	"github.com/palantir/stacktrace"
)

// SMTPConfig is the config for setting up the smtpMailer
type SMTPConfig struct {
	FromName  string
	FromEmail string
	Username  string
	Password  string
	Hostname  string
	Port      string
}

type smtpMailer struct {
	address string
	from    string
	tracer  telemetry.Tracer
	auth    smtp.Auth
}

// NewSMTPEmailService creates a new instance of the smtpMailer
func NewSMTPEmailService(tracer telemetry.Tracer, config SMTPConfig) Mailer {
	return &smtpMailer{
		tracer:  tracer,
		auth:    smtp.PlainAuth("", config.Username, config.Password, config.Hostname),
		address: fmt.Sprintf("%s:%s", config.Hostname, config.Port),
		from:    fmt.Sprintf("%s <%s>", config.FromName, config.FromEmail),
	}
}

// Send a new email
func (mailer *smtpMailer) Send(ctx context.Context, email *Email) (err error) {
	ctx, span := mailer.tracer.Start(ctx)
	defer span.End()

	e := mail.NewEmail()
	e.From = mailer.from
	e.To = []string{email.toAddress()}
	e.Subject = email.Subject
	e.Text = []byte(email.Text)
	e.HTML = []byte(email.HTML)

	err = e.Send(mailer.address, mailer.auth)
	if err != nil {
		return stacktrace.Propagate(err, "cannot send email")
	}

	return nil
}
