package sender

import "github.com/ismdeep/notification-gateway/core/input"

type Sender interface {
	GetName() string
	Send(msg input.Message) error
}

type Config struct {
	Type   string `yaml:"type"`
	Wecom  Wecom  `yaml:"wecom"`
	Email  Email  `yaml:"email"`
	Output Output `yaml:"output"`
}
