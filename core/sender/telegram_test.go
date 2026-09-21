package sender

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ismdeep/notification-gateway/core/input"
)

type telegramPayload struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type telegramRequest struct {
	Path        string
	ContentType string
	Payload     telegramPayload
}

func TestTelegram_GetName(t *testing.T) {
	telegram := &Telegram{Name: "telegram-sender", BotToken: "token", ChatID: "chat-id"}
	assert.Equal(t, "telegram-sender", telegram.GetName())
}

func newTelegramTestServer(t *testing.T) (*httptest.Server, chan telegramRequest) {
	t.Helper()

	requests := make(chan telegramRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}

		var payload telegramPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("unmarshal request body: %v", err)
			return
		}
		requests <- telegramRequest{
			Path:        request.URL.Path,
			ContentType: request.Header.Get("Content-Type"),
			Payload:     payload,
		}

		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	return server, requests
}

func TestTelegram_Send(t *testing.T) {
	server, requests := newTelegramTestServer(t)
	telegram := &Telegram{Name: "telegram", BotToken: "token", ChatID: "-1001234567890"}
	telegram.endpoint = server.URL + "/bottoken/sendMessage"

	err := telegram.Send(input.Message{
		ClientMessageID: "test-client-message-id-001",
		Title:           "Foo",
		Content:         "Bar",
	})
	assert.NoError(t, err)

	request := <-requests
	assert.Equal(t, "/bottoken/sendMessage", request.Path)
	assert.Equal(t, "application/json", request.ContentType)
	assert.Equal(t, "-1001234567890", request.Payload.ChatID)
	assert.Equal(t, "Foo\n\nBar", request.Payload.Text)
}

func TestTelegram_SendContent(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		content string
		want    string
	}{
		{
			name:    "title only",
			title:   "Foo",
			content: "",
			want:    "Foo",
		},
		{
			name:    "content only",
			title:   "",
			content: "Bar",
			want:    "Bar",
		},
		{
			name:    "title and content",
			title:   "Foo",
			content: "Bar",
			want:    "Foo\n\nBar",
		},
		{
			name:    "both empty",
			title:   "",
			content: "",
			want:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newTelegramTestServer(t)
			telegram := &Telegram{Name: "telegram", BotToken: "token", ChatID: "chat-id"}
			telegram.endpoint = server.URL + "/bottoken/sendMessage"

			err := telegram.Send(input.Message{Title: test.title, Content: test.content})
			assert.NoError(t, err)
			assert.Equal(t, test.want, (<-requests).Payload.Text)
		})
	}
}

func TestTelegram_SendValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		telegram *Telegram
		want     string
	}{
		{
			name:     "missing bot token",
			telegram: &Telegram{Name: "telegram", ChatID: "chat-id"},
			want:     "bot token is empty",
		},
		{
			name:     "missing chat id",
			telegram: &Telegram{Name: "telegram", BotToken: "token"},
			want:     "chat id is empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.telegram.Send(input.Message{Title: "Foo", Content: "Bar"})
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestTelegram_Send_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	server.Close()

	telegram := &Telegram{Name: "telegram", BotToken: "token", ChatID: "chat-id"}
	telegram.endpoint = server.URL + "/bottoken/sendMessage"

	err := telegram.Send(input.Message{Title: "Foo", Content: "Bar"})
	assert.ErrorContains(t, err, "send telegram request")
}

func TestTelegram_SendErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		want       string
	}{
		{
			name:       "non 2xx response",
			statusCode: http.StatusBadRequest,
			response:   "bad request",
			want:       "telegram returned status 400: bad request",
		},
		{
			name:       "invalid json response",
			statusCode: http.StatusOK,
			response:   "not-json",
			want:       "unmarshal telegram response",
		},
		{
			name:       "telegram api error",
			statusCode: http.StatusOK,
			response:   `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`,
			want:       "telegram send failed: error_code=400 description=Bad Request: chat not found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(test.statusCode)
				_, _ = writer.Write([]byte(test.response))
			}))
			t.Cleanup(server.Close)

			telegram := &Telegram{Name: "telegram", BotToken: "token", ChatID: "chat-id"}
			telegram.endpoint = server.URL + "/bottoken/sendMessage"
			err := telegram.Send(input.Message{Title: "Foo", Content: "Bar"})
			assert.ErrorContains(t, err, test.want)
		})
	}
}
