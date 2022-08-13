package emails

import (
	"context"
	"fmt"
	"strings"
)

// Email represents an email message
type Email struct {
	ToName  string
	ToEmail string
	Subject string
	HTML    string
	Text    string
}

func (mail *Email) toAddress() string {
	if strings.TrimSpace(mail.ToName) != "" {
		return fmt.Sprintf("%s <%s>", mail.ToName, mail.ToEmail)
	}
	return mail.ToEmail
}

// Mailer is used for sending emails
type Mailer interface {
	// Send adds a message to the push queue
	Send(ctx context.Context, mail *Email) error
}
