package sender

import (
	"bufio"
	"context"
	"encoding/base64"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ismdeep/notification-gateway/core/input"
)

type smtpTestServer struct {
	listener net.Listener
	host     string
	port     int
	data     chan string
	auth     chan string
	done     chan struct{}
}

func TestEmail_GetName(t *testing.T) {
	ctx := context.Background()

	email := Email{
		Name:     "email-sender",
		Host:     "smtp.example.com",
		Port:     2525,
		Username: "",
		Password: "",
		From:     "sender@example.com",
		To:       []string{"receiver@example.com"},
	}
	assert.Equal(t, "email-sender", email.GetName(ctx))
}

func newSMTPTestServer(t *testing.T) *smtpTestServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	host, portString, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	server := &smtpTestServer{
		listener: listener,
		host:     host,
		port:     port,
		data:     make(chan string, 1),
		auth:     make(chan string, 1),
		done:     make(chan struct{}),
	}

	go server.serve()
	t.Cleanup(func() {
		_ = listener.Close()
		<-server.done
	})

	return server
}

func (server *smtpTestServer) serve() {
	defer close(server.done)

	conn, err := server.listener.Accept()
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	scanner := bufio.NewScanner(conn)
	writer := bufio.NewWriter(conn)
	writeLine := func(line string) {
		_, _ = writer.WriteString(line + "\r\n")
		_ = writer.Flush()
	}

	writeLine("220 localhost ESMTP")

	var dataLines []string
	inData := false
	defer func() {
		if len(dataLines) > 0 {
			server.data <- strings.Join(dataLines, "\n")
		}
	}()

	for scanner.Scan() {
		line := scanner.Text()
		if inData {
			if line == "." {
				inData = false
				writeLine("250 2.0.0 OK")
				continue
			}
			dataLines = append(dataLines, line)
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			writeLine("250-localhost")
			writeLine("250-AUTH PLAIN")
			writeLine("250 PIPELINING")
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			server.auth <- line
			writeLine("235 2.7.0 Authentication successful")
		case strings.HasPrefix(upper, "MAIL FROM"):
			writeLine("250 OK")
		case strings.HasPrefix(upper, "RCPT TO"):
			writeLine("250 OK")
		case strings.HasPrefix(upper, "DATA"):
			writeLine("354 End data with <CR><LF>.<CR><LF>")
			inData = true
		case strings.HasPrefix(upper, "QUIT"):
			writeLine("221 2.0.0 Bye")
			return
		default:
			writeLine("250 OK")
		}
	}
}

func TestEmail_Send(t *testing.T) {
	ctx := context.Background()

	server := newSMTPTestServer(t)
	email := Email{
		Name:     "email-sender",
		Host:     server.host,
		Port:     server.port,
		Username: "",
		Password: "",
		From:     "sender@example.com",
		To:       []string{"receiver@example.com"},
	}

	err := email.Send(ctx, input.Message{
		ClientMessageID: "test-client-message-id-001",
		Title:           "Hello",
		Content:         "World",
	})
	assert.NoError(t, err)

	data := <-server.data
	assert.Contains(t, data, `From: "sender@example.com" <sender@example.com>`)
	assert.Contains(t, data, "To: receiver@example.com")
	assert.Contains(t, data, "Subject: Hello")
	assert.Contains(t, data, "World")
}

func TestEmail_SendWithNickname(t *testing.T) {
	ctx := context.Background()

	server := newSMTPTestServer(t)
	email := Email{
		Name:     "email-sender",
		Host:     server.host,
		Port:     server.port,
		From:     "sender@example.com",
		Nickname: "Notification Gateway",
		To:       []string{"receiver@example.com"},
	}

	err := email.Send(ctx, input.Message{Title: "Hello", Content: "World"})
	assert.NoError(t, err)

	data := <-server.data
	assert.Contains(t, data, `From: "Notification Gateway" <sender@example.com>`)
}

func TestEmail_SendWithAuth(t *testing.T) {
	ctx := context.Background()
	server := newSMTPTestServer(t)
	email := Email{
		Name:     "email",
		Host:     server.host,
		Port:     server.port,
		Username: "user",
		Password: "password",
		From:     "sender@example.com",
		To:       []string{"receiver@example.com"},
	}

	err := email.Send(ctx, input.Message{
		ClientMessageID: "test-client-message-id-002",
		Title:           "Hello",
		Content:         "World",
	})
	assert.NoError(t, err)

	authLine := <-server.auth
	fields := strings.Fields(authLine)
	assert.Len(t, fields, 3)
	assert.Equal(t, "AUTH", strings.ToUpper(fields[0]))
	assert.Equal(t, "PLAIN", strings.ToUpper(fields[1]))

	credentials, err := base64.StdEncoding.DecodeString(fields[2])
	assert.NoError(t, err)
	assert.Equal(t, "\x00user\x00password", string(credentials))
}

func TestEmail_Send_SMTPError(t *testing.T) {
	ctx := context.Background()

	server := newSMTPTestServer(t)
	_ = server.listener.Close()

	email := Email{
		Name:     "email",
		Host:     server.host,
		Port:     server.port,
		Username: "",
		Password: "",
		From:     "sender@example.com",
		To:       []string{"receiver@example.com"},
	}

	err := email.Send(ctx, input.Message{Title: "Foo", Content: "Bar"})
	assert.ErrorContains(t, err, "send email")
}

func TestEmail_SendValidationErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		email *Email
		want  string
	}{
		{
			name:  "missing host",
			email: &Email{Name: "email", Port: 2525, From: "sender@example.com", To: []string{"receiver@example.com"}},
			want:  "smtp host is empty",
		},
		{
			name:  "missing port",
			email: &Email{Name: "email", Host: "smtp.example.com", From: "sender@example.com", To: []string{"receiver@example.com"}},
			want:  "smtp port is empty",
		},
		{
			name:  "missing from",
			email: &Email{Name: "email", Host: "smtp.example.com", Port: 2525, To: []string{"receiver@example.com"}},
			want:  "from address is empty",
		},
		{
			name:  "missing recipient",
			email: &Email{Name: "email", Host: "smtp.example.com", Port: 2525, From: "sender@example.com"},
			want:  "recipient is empty",
		},
		{
			name:  "invalid from",
			email: &Email{Name: "email", Host: "smtp.example.com", Port: 2525, From: "not-an-email", To: []string{"receiver@example.com"}},
			want:  "parse email from address",
		},
		{
			name:  "invalid recipient",
			email: &Email{Name: "email", Host: "smtp.example.com", Port: 2525, From: "sender@example.com", To: []string{"not-an-email"}},
			want:  "parse email recipient",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.email.Send(ctx, input.Message{Title: "Foo", Content: "Bar"})
			assert.ErrorContains(t, err, test.want)
		})
	}
}
