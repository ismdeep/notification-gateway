package input

import (
	"encoding/json"
	"testing"
)

func TestMessageJSONTags(t *testing.T) {
	msg := Message{
		ClientMessageID: "client-1",
		Title:           "title",
		Content:         "content",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	want := `{"client_message_id":"client-1","title":"title","content":"content"}`
	if string(data) != want {
		t.Fatalf("marshal message = %s, want %s", data, want)
	}

	var decoded Message
	if err := json.Unmarshal([]byte(want), &decoded); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	if decoded != msg {
		t.Fatalf("decoded message = %#v, want %#v", decoded, msg)
	}
}
