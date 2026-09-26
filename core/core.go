package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/ismdeep/log"
	"go.uber.org/zap"

	"github.com/ismdeep/notification-gateway/core/database"
	"github.com/ismdeep/notification-gateway/core/input"
	"github.com/ismdeep/notification-gateway/core/model"
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
		senderName := s.GetName(ctx)
		alreadySent, err := core.database.MessageAlreadySent(senderName, msg.ClientMessageID)
		if err != nil {
			log.WithContext(ctx).Error("failed to run core.database.MessageAlreadySent",
				zap.Any("sender", senderName),
				zap.Any("client_message_id", msg.ClientMessageID),
				zap.Any("msg", msg),
				zap.Error(err))
			errs = append(errs, fmt.Errorf("failed to run core.database.MessageAlreadySent, sender_name: %v, client_message_id: %v, err: %w", senderName, msg.ClientMessageID, err))
			continue
		}
		if alreadySent {
			continue
		}

		sendStatus := model.MessageSendPending
		log.WithContext(ctx).Info("send message",
			zap.Any("sender", senderName),
			zap.Any("client_message_id", msg.ClientMessageID),
			zap.String("status", model.MessageSendStatusText(sendStatus)),
		)
		err = s.Send(ctx, msg)
		if err != nil {
			sendStatus = model.MessageSendFail
			errs = append(errs, fmt.Errorf("failed to send message to sender, sender_name: %v, client_message_id: %v, err: %w", senderName, msg.ClientMessageID, err))
		} else {
			sendStatus = model.MessageSendSuccess
		}
		log.WithContext(ctx).Info("send message",
			zap.Any("sender", senderName),
			zap.Any("client_message_id", msg.ClientMessageID),
			zap.String("status", model.MessageSendStatusText(sendStatus)),
			zap.Any("err", err))

		if err := core.database.MessageMarkSendStatus(senderName, msg, sendStatus); err != nil {
			log.WithContext(ctx).Error("failed to mark message as sent",
				zap.Any("sender", senderName),
				zap.Any("client_message_id", msg.ClientMessageID),
				zap.Any("msg", msg),
				zap.Error(err))
			errs = append(errs, fmt.Errorf("failed to mark send status, sender_name: %v, client_message_id: %v, err: %w", senderName, msg.ClientMessageID, err))
			continue
		}
	}
	return errors.Join(errs...)
}
