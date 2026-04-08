package vtp

import (
	"testing"
	"time"
)

func TestEncodeParseTickFrame(t *testing.T) {
	input := TickFrame{
		SeqNo:        1001,
		TsUnixMs:     uint64(time.Date(2026, 3, 12, 10, 11, 12, 345000000, time.UTC).UnixMilli()),
		InstrumentID: 123456789,
		Price:        289345,
		Qty:          10,
		Side:         1,
	}

	raw, err := EncodeTickFrame(input)
	if err != nil {
		t.Fatalf("EncodeTickFrame() error = %v", err)
	}

	out, err := ParseTickFrame(raw, true)
	if err != nil {
		t.Fatalf("ParseTickFrame() error = %v", err)
	}

	if out != input {
		t.Fatalf("decoded frame mismatch:\n got: %+v\nwant: %+v", out, input)
	}
}

func TestParseTickFrameCRCFails(t *testing.T) {
	input := TickFrame{
		SeqNo:        1,
		TsUnixMs:     uint64(time.Now().UnixMilli()),
		InstrumentID: 10,
		Price:        1,
		Qty:          1,
		Side:         2,
	}

	raw, err := EncodeTickFrame(input)
	if err != nil {
		t.Fatalf("EncodeTickFrame() error = %v", err)
	}
	raw[10] ^= 0xFF // corrupt payload, keep original CRC

	if _, err := ParseTickFrame(raw, true); err == nil {
		t.Fatalf("expected CRC error, got nil")
	}
}
