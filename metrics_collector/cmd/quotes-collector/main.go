package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mobilki/internal/collector"
	"mobilki/internal/config"
	"mobilki/internal/sink"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Fatalf("config error: %v", err)
	}

	sinks := make([]sink.Sink, 0, 3)

	if cfg.ValkeyEnabled {
		v, err := sink.NewValkeySink(sink.ValkeyConfig{
			Address:  cfg.ValkeyAddress,
			Channel:  cfg.ValkeyChannel,
			Username: cfg.ValkeyUsername,
			Password: cfg.ValkeyPassword,
			DB:       cfg.ValkeyDB,
			Timeout:  cfg.ValkeyTimeout,
		})
		if err != nil {
			if cfg.StrictSinks {
				logger.Fatalf("valkey init error: %v", err)
			}
			logger.Printf("valkey disabled (init error): %v", err)
		} else {
			sinks = append(sinks, v)
			logger.Printf("valkey sink enabled: %s channel=%s", cfg.ValkeyAddress, cfg.ValkeyChannel)
		}
	}

	if cfg.ClickHouseEnabled {
		ch, err := sink.NewClickHouseSink(sink.ClickHouseConfig{
			Endpoint:   cfg.ClickHouseEndpoint,
			Database:   cfg.ClickHouseDatabase,
			Table:      cfg.ClickHouseTable,
			Username:   cfg.ClickHouseUsername,
			Password:   cfg.ClickHousePassword,
			Timeout:    cfg.ClickHouseTimeout,
			AutoCreate: cfg.ClickHouseAutoCreate,
		})
		if err != nil {
			if cfg.StrictSinks {
				logger.Fatalf("clickhouse init error: %v", err)
			}
			logger.Printf("clickhouse disabled (init error): %v", err)
		} else {
			sinks = append(sinks, ch)
			logger.Printf("clickhouse sink enabled: %s %s.%s", cfg.ClickHouseEndpoint, cfg.ClickHouseDatabase, cfg.ClickHouseTable)
		}
	}

	if cfg.StdoutEnabled || len(sinks) == 0 {
		sinks = append(sinks, sink.NewStdoutSink(os.Stdout))
		logger.Printf("stdout sink enabled")
	}

	out := sink.NewMultiSink(sinks...)
	defer func() {
		if err := out.Close(); err != nil {
			logger.Printf("sink close error: %v", err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	app := collector.New(collector.Config{
		DevicePath:      cfg.DevicePath,
		DeviceOpenRetry: cfg.DeviceOpenRetry,
		InputMode:       cfg.InputMode,
		PublishDelay:    cfg.PublishDelay,
		ReadBufferBytes: cfg.ReadBufferBytes,
		VerifyCRC:       cfg.VerifyCRC,
		LogInterval:     cfg.LogInterval,
	}, out, logger)

	if err := app.Run(ctx); err != nil {
		logger.Fatalf("collector failed: %v", err)
	}
}
