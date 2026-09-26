package sender

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/ismdeep/notification-gateway/core/input"
)

type Email struct {
	Name     string   `yaml:"name"`
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	SSL      bool     `yaml:"ssl"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	From     string   `yaml:"from"`
	Nickname string   `yaml:"nickname"`
	To       []string `yaml:"to"`
}

func (receiver *Email) GetName(ctx context.Context) string {
	return receiver.Name
}

func (receiver *Email) Send(ctx context.Context, msg input.Message) error {
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
	from.Name = receiver.Nickname
	if from.Name == "" {
		from.Name = from.Address
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
	if err := receiver.sendMail(address, auth, from.Address, recipients, body.Bytes()); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

// sendMail sends a message over SMTP. When SSL is enabled, the connection is
// established with TLS immediately (implicit TLS, as used by port 465).
func (receiver *Email) sendMail(address string, auth smtp.Auth, from string, recipients []string, body []byte) error {
	if !receiver.SSL {
		return smtp.SendMail(address, auth, from, recipients, body)
	}

	conn, err := tls.Dial("tcp", address, &tls.Config{ServerName: receiver.Host})
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, receiver.Host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = client.Close() }()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := io.Copy(writer, bytes.NewReader(body)); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
