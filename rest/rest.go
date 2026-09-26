package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ismdeep/log"
	"go.uber.org/zap"

	"github.com/ismdeep/notification-gateway/core/input"
)

type Config struct {
	Bind          string `yaml:"bind"`
	Port          int    `yaml:"port"`
	Authorization string `yaml:"authorization"`
}

type Rest struct {
	cfg       Config
	inputChan chan input.Message
	eng       *gin.Engine
	server    *http.Server
}

func NewRest(ctx context.Context, config Config, inputChan chan input.Message) (*Rest, error) {
	r := &Rest{
		cfg:       config,
		inputChan: inputChan,
		eng:       nil,
	}
	if err := r.initRoute(ctx); err != nil {
		log.WithContext(ctx).Error("init route failed", zap.Error(err))
		return nil, err
	}
	r.server = &http.Server{
		Addr:    fmt.Sprintf("%v:%v", r.cfg.Bind, r.cfg.Port),
		Handler: r.eng,
	}
	return r, nil
}

func (r *Rest) initRoute(ctx context.Context) error {
	gin.SetMode(gin.ReleaseMode)
	r.eng = gin.New()
	if r.cfg.Authorization != "" {
		log.WithContext(ctx).Info("register authorization check middleware ...")
		r.eng.Use(func(c *gin.Context) {
			if c.GetHeader("Authorization") != r.cfg.Authorization {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"msg": ErrUnauthorized.Error()})
				return
			}
		})
	} else {
		log.WithContext(ctx).Warn("authorization is not set")
	}

	r.eng.POST("/api/v1/messages", r.PushMessage)
	return nil
}

func (r *Rest) Run(ctx context.Context) error {
	log.WithContext(ctx).Info("rest start", zap.String("bind", r.cfg.Bind), zap.Int("port", r.cfg.Port))
	if r.server == nil {
		r.server = &http.Server{
			Addr:    fmt.Sprintf("%v:%v", r.cfg.Bind, r.cfg.Port),
			Handler: r.eng,
		}
	}
	if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown stops accepting new requests and waits for in-flight requests to
// finish until ctx is cancelled.
func (r *Rest) Shutdown(ctx context.Context) error {
	if r.server == nil {
		return nil
	}
	return r.server.Shutdown(ctx)
}
