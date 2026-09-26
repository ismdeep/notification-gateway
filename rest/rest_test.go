package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ismdeep/notification-gateway/core/input"
)

func TestRestMessagesRequiresAuthorization(t *testing.T) {
	messages := make(chan input.Message, 1)
	r, err := NewRest(context.Background(), Config{Authorization: "secret"}, messages)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/messages", nil)
	resp := httptest.NewRecorder()
	r.eng.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
	if len(messages) != 0 {
		t.Fatal("unauthorized request was queued")
	}
}

func TestRestMessagesRejectsInvalidJSON(t *testing.T) {
	messages := make(chan input.Message, 1)
	r, err := NewRest(context.Background(), Config{Authorization: "secret"}, messages)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBufferString("not-json"))
	req.Header.Set("Authorization", "secret")
	resp := httptest.NewRecorder()
	r.eng.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
	if len(messages) != 0 {
		t.Fatal("invalid request was queued")
	}
}

func TestRestMessagesQueuesValidMessage(t *testing.T) {
	messages := make(chan input.Message, 1)
	r, err := NewRest(context.Background(), Config{}, messages)
	if err != nil {
		t.Fatal(err)
	}
	want := input.Message{ClientMessageID: "client-1", Title: "title", Content: "content"}
	body, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	r.eng.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	select {
	case got := <-messages:
		if got != want {
			t.Fatalf("queued message = %#v, want %#v", got, want)
		}
	default:
		t.Fatal("valid request was not queued")
	}
}
