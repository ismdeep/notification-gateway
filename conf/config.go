package conf

import (
	"github.com/ismdeep/notification-gateway/core/database"
	"github.com/ismdeep/notification-gateway/core/sender"
	"github.com/ismdeep/notification-gateway/rest"
)

type Config struct {
	Rest    rest.Config       `yaml:"rest"`
	DB      database.DBConfig `yaml:"db"`
	Senders []sender.Config   `yaml:"senders"`
}
