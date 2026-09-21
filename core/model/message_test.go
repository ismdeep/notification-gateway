package model

import (
	"testing"
	"time"
)

func TestMessage_TableName(t *testing.T) {
	type fields struct {
		ID              int64
		SenderName      string
		ClientMessageID string
		Title           string
		Content         string
		CreatedAt       time.Time
		UpdatedAt       time.Time
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "",
			fields: fields{},
			want:   "messages",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := &Message{
				ID:              tt.fields.ID,
				SenderName:      tt.fields.SenderName,
				ClientMessageID: tt.fields.ClientMessageID,
				Title:           tt.fields.Title,
				Content:         tt.fields.Content,
				CreatedAt:       tt.fields.CreatedAt,
				UpdatedAt:       tt.fields.UpdatedAt,
			}
			if got := receiver.TableName(); got != tt.want {
				t.Errorf("TableName() = %v, want %v", got, tt.want)
			}
		})
	}
}
