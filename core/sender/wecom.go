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

type Wecom struct {
	Name     string
	Endpoint string
}

func (receiver *Wecom) GetName() string {
	return receiver.Name
}

func (receiver *Wecom) Send(msg input.Message) error {
	content := msg.Title
	if msg.Content != "" {
		if content != "" {
			content += "\n\n"
		}
		content += msg.Content
	}

	payload := struct {
		MsgType string `json:"msgtype"`
		Text    struct {
			Content string `json:"content"`
		} `json:"text"`
	}{
		MsgType: "text",
	}
	payload.Text.Content = content

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal wecom payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, receiver.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create wecom request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send wecom request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read wecom response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("wecom returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("unmarshal wecom response: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wecom send failed: errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
	}

	return nil
}
