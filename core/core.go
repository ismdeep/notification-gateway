package core

import (
	"context"
	"errors"

	"github.com/ismdeep/log"
	"go.uber.org/zap"

	"github.com/ismdeep/notification-gateway/core/database"
	"github.com/ismdeep/notification-gateway/core/input"
	"github.com/ismdeep/notification-gateway/core/sender"
)

type Core struct {
	inputChan chan input.Message
	senders   []sender.Sender
	database  database.Database
}

func NewCore(inputChan chan input.Message, senders []sender.Sender, database database.Database) *Core {
	return &Core{
		inputChan: inputChan,
		senders:   senders,
		database:  database,
	}
}

func (core *Core) Run(ctx context.Context) error {
	for msg := range core.inputChan {
		if err := core.processInputMessage(ctx, msg); err != nil {
			log.WithContext(ctx).Warn("failed to process message", zap.Any("msg", msg), zap.Error(err))
		}
	}
	return nil
}

func (core *Core) processInputMessage(ctx context.Context, msg input.Message) error {
	var errs []error
	for _, s := range core.senders {
		senderName := s.GetName()
		alreadySent, err := core.database.MessageAlreadySent(senderName, msg.ClientMessageID)
		if err != nil {
			log.WithContext(ctx).Error("failed to run core.database.MessageAlreadySent",
				zap.Any("sender", senderName),
				zap.Any("msg", msg),
				zap.Error(err))
			errs = append(errs, err)
			continue
		}
		if alreadySent {
			continue
		}

		if err := s.Send(msg); err != nil {
			log.WithContext(ctx).Error("failed to send message to sender",
				zap.Any("sender", senderName),
				zap.Any("msg", msg),
				zap.Error(err))
			errs = append(errs, err)
		}

		if err := core.database.MessageMarkedAsSent(senderName, msg); err != nil {
			log.WithContext(ctx).Error("failed to mark message as sent",
				zap.Any("sender", senderName),
				zap.Any("msg", msg),
				zap.Error(err))
			errs = append(errs, err)
			continue
		}
	}
	return errors.Join(errs...)
}
