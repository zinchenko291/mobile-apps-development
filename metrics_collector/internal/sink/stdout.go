package sink

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"mobilki/internal/vtp"
)

type StdoutSink struct {
	mu  sync.Mutex
	enc *json.Encoder
}

func NewStdoutSink(w io.Writer) *StdoutSink {
	return &StdoutSink{enc: json.NewEncoder(w)}
}

func (s *StdoutSink) Name() string {
	return "stdout"
}

func (s *StdoutSink) Publish(_ context.Context, tick vtp.TickFrame) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enc.Encode(struct {
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
}

func (s *StdoutSink) Close() error {
	return nil
}
