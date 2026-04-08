package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mobilki/internal/vtp"
)

type ClickHouseConfig struct {
	Endpoint   string
	Database   string
	Table      string
	Username   string
	Password   string
	Timeout    time.Duration
	AutoCreate bool
}

type ClickHouseSink struct {
	cfg    ClickHouseConfig
	client *http.Client
}

func NewClickHouseSink(cfg ClickHouseConfig) (*ClickHouseSink, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, errors.New("clickhouse endpoint is empty")
	}
	if strings.TrimSpace(cfg.Database) == "" {
		return nil, errors.New("clickhouse database is empty")
	}
	if strings.TrimSpace(cfg.Table) == "" {
		return nil, errors.New("clickhouse table is empty")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if _, err := url.ParseRequestURI(cfg.Endpoint); err != nil {
		return nil, fmt.Errorf("invalid CLICKHOUSE_ENDPOINT: %w", err)
	}

	s := &ClickHouseSink{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
	if cfg.AutoCreate {
		if err := s.ensureTable(context.Background()); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (c *ClickHouseSink) Name() string {
	return "clickhouse"
}

func (c *ClickHouseSink) Publish(ctx context.Context, tick vtp.TickFrame) error {
	row, err := json.Marshal(struct {
		TsExchange   string `json:"ts_exchange"`
		InstrumentID uint64 `json:"instrument_id"`
		Price        int64  `json:"price"`
		Qty          uint64 `json:"qty"`
		Side         uint8  `json:"side"`
		SeqNo        uint64 `json:"seq_no"`
		TsUnixMs     uint64 `json:"ts_unix_ms"`
	}{
		TsExchange:   tick.ExchangeTimeUTC().Format("2006-01-02 15:04:05.000"),
		InstrumentID: tick.InstrumentID,
		Price:        tick.Price,
		Qty:          tick.Qty,
		Side:         tick.Side,
		SeqNo:        tick.SeqNo,
		TsUnixMs:     tick.TsUnixMs,
	})
	if err != nil {
		return err
	}

	query := fmt.Sprintf("INSERT INTO %s.%s FORMAT JSONEachRow", c.cfg.Database, c.cfg.Table)
	body := append(row, '\n')
	return c.exec(ctx, query, bytes.NewReader(body))
}

func (c *ClickHouseSink) Close() error {
	return nil
}

func (c *ClickHouseSink) ensureTable(ctx context.Context) error {
	query := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.%s (
	ts_exchange DateTime64(3, 'UTC'),
	instrument_id UInt64,
	price Int64,
	qty UInt64,
	side UInt8,
	seq_no UInt64,
	ts_unix_ms UInt64
) ENGINE = MergeTree
ORDER BY (instrument_id, ts_exchange, seq_no)
TTL toDateTime(ts_exchange) + INTERVAL 7 DAY
`, c.cfg.Database, c.cfg.Table)

	return c.exec(ctx, query, nil)
}

func (c *ClickHouseSink) exec(ctx context.Context, query string, body io.Reader) error {
	reqURL := c.cfg.Endpoint
	if !strings.Contains(reqURL, "?") {
		reqURL += "?query=" + url.QueryEscape(query)
	} else {
		reqURL += "&query=" + url.QueryEscape(query)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, body)
	if err != nil {
		return err
	}
	if c.cfg.Username != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		return fmt.Errorf("clickhouse status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}
