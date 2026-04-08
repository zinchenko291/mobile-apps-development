package collector

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync/atomic"
	"time"

	"mobilki/internal/randomticks"
	"mobilki/internal/sink"
	"mobilki/internal/vtp"
)

type Config struct {
	DevicePath      string
	DeviceOpenRetry time.Duration
	InputMode       string
	PublishDelay    time.Duration
	ReadBufferBytes int
	VerifyCRC       bool
	LogInterval     time.Duration
}

type Collector struct {
	cfg    Config
	logger *log.Logger
	sink   sink.Sink

	vtpParser    *vtp.StreamParser
	randomParser *randomticks.StreamDecoder

	bytesRead    uint64
	framesParsed uint64
	framesSent   uint64
	droppedBytes uint64
	sinkErrors   uint64
}

func New(cfg Config, out sink.Sink, logger *log.Logger) *Collector {
	if logger == nil {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}
	if cfg.ReadBufferBytes <= 0 {
		cfg.ReadBufferBytes = 4096
	}
	if cfg.InputMode == "" {
		cfg.InputMode = "random"
	}
	if cfg.DeviceOpenRetry <= 0 {
		cfg.DeviceOpenRetry = 1 * time.Second
	}
	if cfg.LogInterval <= 0 {
		cfg.LogInterval = 10 * time.Second
	}
	if cfg.PublishDelay < 0 {
		cfg.PublishDelay = 0
	}

	c := &Collector{
		cfg:    cfg,
		logger: logger,
		sink:   out,
	}
	if cfg.InputMode == "vtp" {
		c.vtpParser = vtp.NewStreamParser(cfg.VerifyCRC)
	} else {
		c.randomParser = randomticks.NewStreamDecoder(0)
	}
	return c
}

func (c *Collector) Run(ctx context.Context) error {
	var (
		dev *os.File
		err error
	)
	for {
		dev, err = os.Open(c.cfg.DevicePath)
		if err == nil {
			break
		}
		c.logger.Printf(
			"device is not ready yet (%s): %v; retry in %s",
			c.cfg.DevicePath,
			err,
			c.cfg.DeviceOpenRetry,
		)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(c.cfg.DeviceOpenRetry):
		}
	}
	defer dev.Close()

	c.logger.Printf(
		"collector started: device=%s input_mode=%s verify_crc=%t publish_delay=%s",
		c.cfg.DevicePath,
		c.cfg.InputMode,
		c.cfg.VerifyCRC,
		c.cfg.PublishDelay,
	)

	// Unblock read() on shutdown.
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = dev.Close()
		case <-done:
		}
	}()
	defer close(done)

	logTicker := time.NewTicker(c.cfg.LogInterval)
	defer logTicker.Stop()

	buf := make([]byte, c.cfg.ReadBufferBytes)

	for {
		n, readErr := dev.Read(buf)
		if n > 0 {
			atomic.AddUint64(&c.bytesRead, uint64(n))
			var (
				frames  []vtp.TickFrame
				dropped int
			)
			if c.cfg.InputMode == "vtp" {
				frames, dropped = c.vtpParser.Feed(buf[:n])
			} else {
				frames, dropped = c.randomParser.Feed(buf[:n])
			}
			if dropped > 0 {
				atomic.AddUint64(&c.droppedBytes, uint64(dropped))
			}

			for _, frame := range frames {
				atomic.AddUint64(&c.framesParsed, 1)
				if err := c.sink.Publish(ctx, frame); err != nil {
					atomic.AddUint64(&c.sinkErrors, 1)
					c.logger.Printf("publish error: %v", err)
					continue
				}
				atomic.AddUint64(&c.framesSent, 1)
				if c.cfg.PublishDelay > 0 {
					select {
					case <-ctx.Done():
						c.logger.Printf("collector stopped: %v", ctx.Err())
						return nil
					case <-time.After(c.cfg.PublishDelay):
					}
				}
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				if ctx.Err() != nil {
					c.logger.Printf("collector stopped: %v", ctx.Err())
					return nil
				}
				time.Sleep(50 * time.Millisecond)
				continue
			}
			if errors.Is(readErr, os.ErrClosed) && ctx.Err() != nil {
				c.logger.Printf("collector stopped: %v", ctx.Err())
				return nil
			}
			return fmt.Errorf("read device: %w", readErr)
		}

		select {
		case <-ctx.Done():
			c.logger.Printf("collector stopped: %v", ctx.Err())
			return nil
		case <-logTicker.C:
			c.logStats()
		default:
		}
	}
}

func (c *Collector) logStats() {
	c.logger.Printf(
		"stats bytes=%d frames_parsed=%d frames_sent=%d dropped_bytes=%d sink_errors=%d",
		atomic.LoadUint64(&c.bytesRead),
		atomic.LoadUint64(&c.framesParsed),
		atomic.LoadUint64(&c.framesSent),
		atomic.LoadUint64(&c.droppedBytes),
		atomic.LoadUint64(&c.sinkErrors),
	)
}
