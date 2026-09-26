package sender

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ismdeep/notification-gateway/core/input"
)

type wecomPayload struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

func TestWecom_GetName(t *testing.T) {
	ctx := context.Background()
	wecom := &Wecom{Name: "wecom-sender", Endpoint: "https://example.com/webhook"}
	assert.Equal(t, "wecom-sender", wecom.GetName(ctx))
}

func newWecomTestServer(t *testing.T) (*httptest.Server, chan wecomPayload) {
	t.Helper()

	payloads := make(chan wecomPayload, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}

		var payload wecomPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("unmarshal request body: %v", err)
			return
		}
		payloads <- payload

		_, _ = writer.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	t.Cleanup(server.Close)

	return server, payloads
}

func TestWecom_Send(t *testing.T) {
	ctx := context.Background()

	server, payloads := newWecomTestServer(t)
	wecom := &Wecom{Name: "wecom", Endpoint: server.URL}

	err := wecom.Send(ctx, input.Message{
		ClientMessageID: "test-client-message-id-001",
		Title:           "Foo",
		Content:         "Bar",
	})
	assert.NoError(t, err)

	payload := <-payloads
	assert.Equal(t, "text", payload.MsgType)
	assert.Equal(t, "Foo\n\nBar", payload.Text.Content)
}

func TestWecom_SendContent(t *testing.T) {
	ctx := context.Background()

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
			server, payloads := newWecomTestServer(t)
			wecom := &Wecom{Name: "wecom", Endpoint: server.URL}

			err := wecom.Send(ctx, input.Message{Title: test.title, Content: test.content})
			assert.NoError(t, err)
			assert.Equal(t, test.want, (<-payloads).Text.Content)
		})
	}
}

func TestWecom_Send_InvalidEndpoint(t *testing.T) {
	ctx := context.Background()

	wecom := &Wecom{Name: "wecom", Endpoint: "://bad-endpoint"}

	err := wecom.Send(ctx, input.Message{Title: "Foo", Content: "Bar"})
	assert.ErrorContains(t, err, "create wecom request")
}

func TestWecom_Send_NetworkError(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	server.Close()

	wecom := &Wecom{Name: "wecom", Endpoint: server.URL}

	err := wecom.Send(ctx, input.Message{Title: "Foo", Content: "Bar"})
	assert.ErrorContains(t, err, "send wecom request")
}

func TestWecom_SendErrorResponses(t *testing.T) {
	ctx := context.Background()

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
			want:       "wecom returned status 400: bad request",
		},
		{
			name:       "invalid json response",
			statusCode: http.StatusOK,
			response:   "not-json",
			want:       "unmarshal wecom response",
		},
		{
			name:       "wecom errcode",
			statusCode: http.StatusOK,
			response:   `{"errcode":93000,"errmsg":"invalid webhook url"}`,
			want:       "wecom send failed: errcode=93000 errmsg=invalid webhook url",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(test.statusCode)
				_, _ = writer.Write([]byte(test.response))
			}))
			t.Cleanup(server.Close)

			wecom := &Wecom{Name: "wecom", Endpoint: server.URL}
			err := wecom.Send(ctx, input.Message{Title: "Foo", Content: "Bar"})
			assert.ErrorContains(t, err, test.want)
		})
	}
}
