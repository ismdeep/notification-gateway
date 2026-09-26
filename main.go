package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ismdeep/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/ismdeep/notification-gateway/conf"
	"github.com/ismdeep/notification-gateway/core"
	"github.com/ismdeep/notification-gateway/core/database"
	"github.com/ismdeep/notification-gateway/core/input"
	"github.com/ismdeep/notification-gateway/core/sender"
	"github.com/ismdeep/notification-gateway/rest"
	"github.com/ismdeep/notification-gateway/version"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Init("console://[stdout]?level=debug&time_encoder=rfc3339&trace_level=fatal")

	log.WithContext(ctx).Info("notification-gateway", zap.String("version", version.Version))

	// 解析配置
	log.WithContext(ctx).Info("loading config.yaml ...")
	raw, err := os.ReadFile("config.yaml")
	if err != nil {
		panic(fmt.Errorf("failed to read config.yaml: %w", err))
	}
	var cfg conf.Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		panic(fmt.Errorf("failed to unmarshal config.yaml: %w", err))
	}

	// 准备管道
	inputChan := make(chan input.Message, 1024)

	// 连接数据库
	log.WithContext(ctx).Info("connecting to database ...")
	db, err := database.NewDB(cfg.DB)
	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}

	// 组装 senders
	var senders []sender.Sender
	for _, sc := range cfg.Senders {
		switch sc.Type {
		case "wecom":
			if sc.Wecom.Name == "" {
				panic("wecom name is required")
			}
			senders = append(senders, &sc.Wecom)
		case "email":
			if sc.Email.Name == "" {
				panic("email name is required")
			}
			senders = append(senders, &sc.Email)
		case "telegram":
			if sc.Telegram.Name == "" {
				panic("telegram name is required")
			}
			senders = append(senders, &sc.Telegram)
		case "output":
			if sc.Output.Name == "" {
				panic("output name is required")
			}
			senders = append(senders, &sc.Output)
		default:
			panic(fmt.Errorf("unknown sender type: %s", sc.Type))
		}
	}

	// 创建并启动 core
	log.WithContext(ctx).Info("starting core ...")
	c := core.NewCore(inputChan, senders, db)
	coreDone := make(chan struct{})
	go func() {
		defer close(coreDone)
		if err := c.Run(ctx); err != nil {
			log.WithContext(ctx).Error("core stopped with error", zap.Error(err))
		}
	}()

	// 创建 rest 服务
	r, err := rest.NewRest(ctx, cfg.Rest, inputChan)
	if err != nil {
		panic(fmt.Errorf("failed to create rest server: %w", err))
	}

	// 启动 rest 服务
	log.WithContext(ctx).Info("starting rest ...")
	restErr := make(chan error, 1)
	go func() { restErr <- r.Run(ctx) }()

	select {
	case err := <-restErr:
		if err != nil {
			log.WithContext(ctx).Error("rest stopped with error", zap.Error(err))
			log.WithContext(ctx).Info("close inputChan ...")
			close(inputChan)
			<-coreDone
			log.WithContext(ctx).Info("inputChan processing completed")
			panic(fmt.Errorf("failed to start rest server: %w", err))
		}
	case <-ctx.Done():
		log.WithContext(ctx).Info("shutting down ...")
		// Wait for all in-flight HTTP handlers to finish before closing the
		// input channel; otherwise a handler could send to a closed channel.
		if err := r.Shutdown(context.Background()); err != nil {
			log.WithContext(ctx).Warn("failed to shutdown rest server", zap.Error(err))
		}
		log.WithContext(ctx).Info("close inputChan ...")
		close(inputChan)
		<-coreDone
		log.WithContext(ctx).Info("inputChan processing completed")
		if err := <-restErr; err != nil {
			log.WithContext(ctx).Warn("rest server stopped with error", zap.Error(err))
		}
	}
}
