package sink

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"mobilki/internal/vtp"
)

type ValkeyConfig struct {
	Address  string
	Channel  string
	Username string
	Password string
	DB       int
	Timeout  time.Duration
}

type ValkeySink struct {
	cfg ValkeyConfig

	mu   sync.Mutex
	conn net.Conn
	rw   *bufio.ReadWriter
}

func NewValkeySink(cfg ValkeyConfig) (*ValkeySink, error) {
	if strings.TrimSpace(cfg.Address) == "" {
		return nil, errors.New("valkey address is empty")
	}
	if strings.TrimSpace(cfg.Channel) == "" {
		return nil, errors.New("valkey channel is empty")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 3 * time.Second
	}

	v := &ValkeySink{cfg: cfg}
	if err := v.connect(); err != nil {
		return nil, err
	}
	return v, nil
}

func (v *ValkeySink) Name() string {
	return "valkey"
}

func (v *ValkeySink) connect() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.connectLocked()
}

func (v *ValkeySink) connectLocked() error {
	if v.conn != nil {
		return nil
	}

	conn, err := net.DialTimeout("tcp", v.cfg.Address, v.cfg.Timeout)
	if err != nil {
		return fmt.Errorf("dial valkey: %w", err)
	}

	v.conn = conn
	v.rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	if v.cfg.Password != "" {
		if v.cfg.Username != "" {
			if _, err := v.commandLocked("AUTH", v.cfg.Username, v.cfg.Password); err != nil {
				v.closeLocked()
				return fmt.Errorf("AUTH failed: %w", err)
			}
		} else {
			if _, err := v.commandLocked("AUTH", v.cfg.Password); err != nil {
				v.closeLocked()
				return fmt.Errorf("AUTH failed: %w", err)
			}
		}
	}

	if v.cfg.DB > 0 {
		if _, err := v.commandLocked("SELECT", strconv.Itoa(v.cfg.DB)); err != nil {
			v.closeLocked()
			return fmt.Errorf("SELECT failed: %w", err)
		}
	}

	return nil
}

func (v *ValkeySink) Publish(_ context.Context, tick vtp.TickFrame) error {
	payload, err := json.Marshal(struct {
		SeqNo        uint64 `json:"seq_no"`
		TsUnixMs     uint64 `json:"ts_unix_ms"`
		InstrumentID uint64 `json:"instrument_id"`
		Price        int64  `json:"price_kopecks"`
		Qty          uint64 `json:"qty_lots"`
		Side         uint8  `json:"side"`
	}{
		SeqNo:        tick.SeqNo,
		TsUnixMs:     tick.TsUnixMs,
		InstrumentID: tick.InstrumentID,
		Price:        tick.Price,
		Qty:          tick.Qty,
		Side:         tick.Side,
	})
	if err != nil {
		return err
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	if err := v.connectLocked(); err != nil {
		return err
	}

	reply, err := v.commandLocked("PUBLISH", v.cfg.Channel, string(payload))
	if err != nil {
		v.closeLocked()
		return err
	}

	if _, ok := reply.(int64); !ok {
		return fmt.Errorf("unexpected PUBLISH reply type: %T", reply)
	}
	return nil
}

func (v *ValkeySink) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.closeLocked()
	return nil
}

func (v *ValkeySink) closeLocked() {
	if v.conn != nil {
		_ = v.conn.Close()
	}
	v.conn = nil
	v.rw = nil
}

func (v *ValkeySink) commandLocked(args ...string) (any, error) {
	if v.rw == nil {
		return nil, errors.New("valkey connection is not initialized")
	}
	if err := writeRESP(v.rw.Writer, args...); err != nil {
		return nil, err
	}
	if err := v.rw.Flush(); err != nil {
		return nil, err
	}
	return readRESP(v.rw.Reader)
}

func writeRESP(w *bufio.Writer, args ...string) error {
	if _, err := fmt.Fprintf(w, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}
	return nil
}

func readRESP(r *bufio.Reader) (any, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch prefix {
	case '+':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return line, nil
	case '-':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(line)
	case ':':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		n, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer reply %q: %w", line, err)
		}
		return n, nil
	case '$':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, fmt.Errorf("invalid bulk length %q: %w", line, err)
		}
		if n < 0 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		return string(buf[:n]), nil
	case '*':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, fmt.Errorf("invalid array length %q: %w", line, err)
		}
		if n < 0 {
			return nil, nil
		}
		items := make([]any, 0, n)
		for i := 0; i < n; i++ {
			item, err := readRESP(r)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unknown RESP prefix %q", string(prefix))
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, nil
}
