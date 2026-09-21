package sender

import (
	"bytes"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/ismdeep/notification-gateway/core/input"
)

type Email struct {
	Name     string
	Host     string
	Port     int
	Username string
	Password string
	From     string
	To       []string
}

func (receiver *Email) GetName() string {
	return receiver.Name
}

func (receiver *Email) Send(msg input.Message) error {
	if receiver.Host == "" {
		return fmt.Errorf("email smtp host is empty")
	}
	if receiver.Port == 0 {
		return fmt.Errorf("email smtp port is empty")
	}
	if receiver.From == "" {
		return fmt.Errorf("email from address is empty")
	}
	if len(receiver.To) == 0 {
		return fmt.Errorf("email recipient is empty")
	}

	from, err := mail.ParseAddress(receiver.From)
	if err != nil {
		return fmt.Errorf("parse email from address: %w", err)
	}

	recipients := make([]string, 0, len(receiver.To))
	for _, recipient := range receiver.To {
		address, err := mail.ParseAddress(recipient)
		if err != nil {
			return fmt.Errorf("parse email recipient %q: %w", recipient, err)
		}
		recipients = append(recipients, address.Address)
	}

	var body bytes.Buffer
	_, _ = fmt.Fprintf(&body, "From: %s\r\n", from.String())
	_, _ = fmt.Fprintf(&body, "To: %s\r\n", strings.Join(recipients, ", "))
	_, _ = fmt.Fprintf(&body, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", msg.Title))
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	body.WriteString("\r\n")
	body.WriteString(msg.Content)

	var auth smtp.Auth
	if receiver.Username != "" {
		auth = smtp.PlainAuth("", receiver.Username, receiver.Password, receiver.Host)
	}

	address := net.JoinHostPort(receiver.Host, strconv.Itoa(receiver.Port))
	if err := smtp.SendMail(address, auth, from.Address, recipients, body.Bytes()); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}
