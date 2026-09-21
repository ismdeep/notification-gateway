package sender

import (
	"context"

	"github.com/ismdeep/log"
	"go.uber.org/zap"

	"github.com/ismdeep/notification-gateway/core/input"
)

type Output struct {
	Name string `json:"name" yaml:"name"`
}

func (receiver *Output) GetName() string {
	return receiver.Name
}

func (receiver *Output) Send(msg input.Message) error {
	log.WithContext(context.Background()).Info("send",
		zap.Any("name", receiver.Name),
		zap.Any("msg", msg))
	return nil
}
