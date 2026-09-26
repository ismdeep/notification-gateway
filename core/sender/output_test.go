package sender

import (
	"context"
	"testing"

	"github.com/ismdeep/notification-gateway/core/input"
)

func TestOutput_GetName(t *testing.T) {
	ctx := context.Background()
	output := &Output{Name: "output-sender"}
	if got := output.GetName(ctx); got != "output-sender" {
		t.Fatalf("GetName() = %q, want %q", got, "output-sender")
	}
}

func TestOutput_Send(t *testing.T) {
	ctx := context.Background()
	output := &Output{Name: "output-sender"}
	err := output.Send(ctx, input.Message{
		ClientMessageID: "client-1",
		Title:           "title",
		Content:         "content",
	})
	if err != nil {
		t.Fatalf("Send() returned error: %v", err)
	}
}
