package database

import "github.com/ismdeep/notification-gateway/core/input"

type Database interface {
	MessageAlreadySent(senderName string, clientMessageID string) (bool, error)
	MessageMarkedAsSent(senderName string, msg input.Message) error
}
