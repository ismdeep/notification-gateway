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

func (receiver *Output) GetName(ctx context.Context) string {
	return receiver.Name
}

func (receiver *Output) Send(ctx context.Context, msg input.Message) error {
	log.WithContext(ctx).Info("send",
		zap.Any("name", receiver.Name),
		zap.Any("msg", msg))
	return nil
}
