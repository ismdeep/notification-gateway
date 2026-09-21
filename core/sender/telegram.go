package sender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ismdeep/notification-gateway/core/input"
)

type Telegram struct {
	Name     string `yaml:"name"`
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
	endpoint string
}

func (receiver *Telegram) GetName() string {
	return receiver.Name
}

func (receiver *Telegram) Send(msg input.Message) error {
	if receiver.BotToken == "" {
		return fmt.Errorf("telegram bot token is empty")
	}
	if receiver.ChatID == "" {
		return fmt.Errorf("telegram chat id is empty")
	}

	content := msg.Title
	if msg.Content != "" {
		if content != "" {
			content += "\n\n"
		}
		content += msg.Content
	}

	payload := struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}{
		ChatID: receiver.ChatID,
		Text:   content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	endpoint := receiver.endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", receiver.BotToken)
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read telegram response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("telegram returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("unmarshal telegram response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("telegram send failed: error_code=%d description=%s", result.ErrorCode, result.Description)
	}

	return nil
}
