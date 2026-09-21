package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"

	"github.com/ismdeep/notification-gateway/core/input"
	"github.com/ismdeep/notification-gateway/core/model"
)

type DBConfig struct {
	Dialect string `yaml:"dialect"`
	DSN     string `yaml:"dsn"`
}

type DB struct {
	db *gorm.DB
}

func (receiver *DB) SupportedDialects() []string {
	return []string{
		"mysql",
		"postgres",
		"sqlite",
	}
}

func NewDB(dbConfig DBConfig) (*DB, error) {
	dialect := dbConfig.Dialect
	dsn := dbConfig.DSN

	var db *gorm.DB
	var err error

	switch dialect {
	case "mysql":
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
	case "postgres":
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
	case "sqlite":
		db, err = gorm.Open(sqlite.New(sqlite.Config{
			DriverName: "sqlite",
			DSN:        dsn,
		}), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
	default:
		return nil, fmt.Errorf("dialect %s not supported, supported dialects: %v", dialect, strings.Join((&DB{}).SupportedDialects(), ", "))
	}

	if err := db.AutoMigrate(&model.Message{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate database: %w", err)
	}

	return &DB{db: db}, nil
}

func (receiver *DB) MessageAlreadySent(senderName string, clientMessageID string) (bool, error) {
	var cnt int64
	if err := receiver.db.Model(&model.Message{}).
		Where("sender_name = ? AND client_message_id = ?", senderName, clientMessageID).
		Limit(1).
		Count(&cnt).Error; err != nil {
		return false, fmt.Errorf("failed to query sender message: %w", err)
	}
	return cnt > 0, nil
}

func (receiver *DB) MessageMarkSendStatus(senderName string, msg input.Message, status int) error {
	now := time.Now()
	m := model.Message{
		ID:              0,
		SenderName:      senderName,
		ClientMessageID: msg.ClientMessageID,
		Title:           msg.Title,
		Content:         msg.Content,
		SendStatus:      status,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := receiver.db.Create(&m).Error; err != nil {
		return fmt.Errorf("failed to mark message as sent: %w", err)
	}
	return nil
}
