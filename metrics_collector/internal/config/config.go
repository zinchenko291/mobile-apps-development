package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DevicePath      string
	DeviceOpenRetry time.Duration
	InputMode       string
	PublishDelay    time.Duration
	ReadBufferBytes int
	VerifyCRC       bool
	LogInterval     time.Duration
	StrictSinks     bool
	StdoutEnabled   bool

	ValkeyEnabled  bool
	ValkeyAddress  string
	ValkeyChannel  string
	ValkeyUsername string
	ValkeyPassword string
	ValkeyDB       int
	ValkeyTimeout  time.Duration

	ClickHouseEnabled    bool
	ClickHouseEndpoint   string
	ClickHouseDatabase   string
	ClickHouseTable      string
	ClickHouseUsername   string
	ClickHousePassword   string
	ClickHouseTimeout    time.Duration
	ClickHouseAutoCreate bool
}

func LoadFromEnv() (Config, error) {
	var cfg Config

	cfg.DevicePath = envString("COLLECTOR_DEVICE_PATH", "/dev/urandom")
	cfg.InputMode = envString("COLLECTOR_INPUT_MODE", "random")
	cfg.ValkeyPassword = envString("VALKEY_PASSWORD", "")
	cfg.ValkeyUsername = envString("VALKEY_USERNAME", "")
	cfg.ValkeyAddress = envString("VALKEY_ADDRESS", "127.0.0.1:6379")
	cfg.ValkeyChannel = envString("VALKEY_CHANNEL", "quotes.ticks")
	cfg.ClickHouseEndpoint = envString("CLICKHOUSE_ENDPOINT", "http://127.0.0.1:8123")
	cfg.ClickHouseDatabase = envString("CLICKHOUSE_DATABASE", "default")
	cfg.ClickHouseTable = envString("CLICKHOUSE_TABLE", "raw_ticks")
	cfg.ClickHouseUsername = envString("CLICKHOUSE_USERNAME", "")
	cfg.ClickHousePassword = envString("CLICKHOUSE_PASSWORD", "")

	switch cfg.InputMode {
	case "random", "vtp":
	default:
		return cfg, fmt.Errorf("COLLECTOR_INPUT_MODE must be 'random' or 'vtp'")
	}

	var err error
	if cfg.ReadBufferBytes, err = envInt("COLLECTOR_READ_BUFFER_BYTES", 4096); err != nil {
		return cfg, err
	}
	if cfg.ReadBufferBytes < 256 {
		return cfg, fmt.Errorf("COLLECTOR_READ_BUFFER_BYTES must be >= 256")
	}

	if cfg.VerifyCRC, err = envBool("COLLECTOR_VERIFY_CRC", true); err != nil {
		return cfg, err
	}
	if cfg.StrictSinks, err = envBool("COLLECTOR_STRICT_SINKS", false); err != nil {
		return cfg, err
	}
	if cfg.StdoutEnabled, err = envBool("COLLECTOR_STDOUT_ENABLED", false); err != nil {
		return cfg, err
	}
	if cfg.ValkeyEnabled, err = envBool("VALKEY_ENABLED", true); err != nil {
		return cfg, err
	}
	if cfg.ClickHouseEnabled, err = envBool("CLICKHOUSE_ENABLED", true); err != nil {
		return cfg, err
	}
	if cfg.ClickHouseAutoCreate, err = envBool("CLICKHOUSE_AUTO_CREATE_TABLE", true); err != nil {
		return cfg, err
	}

	if cfg.ValkeyDB, err = envInt("VALKEY_DB", 0); err != nil {
		return cfg, err
	}
	if cfg.ValkeyDB < 0 {
		return cfg, fmt.Errorf("VALKEY_DB cannot be negative")
	}

	if cfg.LogInterval, err = envDuration("COLLECTOR_LOG_INTERVAL", 10*time.Second); err != nil {
		return cfg, err
	}
	if cfg.DeviceOpenRetry, err = envDuration("COLLECTOR_DEVICE_OPEN_RETRY", 1*time.Second); err != nil {
		return cfg, err
	}
	if cfg.DeviceOpenRetry <= 0 {
		return cfg, fmt.Errorf("COLLECTOR_DEVICE_OPEN_RETRY must be > 0")
	}
	if cfg.PublishDelay, err = envDuration("COLLECTOR_PUBLISH_DELAY", 0); err != nil {
		return cfg, err
	}
	if cfg.PublishDelay < 0 {
		return cfg, fmt.Errorf("COLLECTOR_PUBLISH_DELAY must be >= 0")
	}
	if cfg.ValkeyTimeout, err = envDuration("VALKEY_TIMEOUT", 3*time.Second); err != nil {
		return cfg, err
	}
	if cfg.ClickHouseTimeout, err = envDuration("CLICKHOUSE_TIMEOUT", 5*time.Second); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func envString(name, def string) string {
	v, ok := os.LookupEnv(name)
	if !ok {
		return def
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	return v
}

func envBool(name string, def bool) (bool, error) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return def, nil
	}
	out, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return false, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

func envInt(name string, def int) (int, error) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return def, nil
	}
	out, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

func envDuration(name string, def time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return def, nil
	}
	out, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}
