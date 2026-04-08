package vtp

import (
	"bytes"
	"testing"
	"time"
)

func TestStreamParserWithNoiseAndSplitFrames(t *testing.T) {
	f1, err := EncodeTickFrame(TickFrame{
		SeqNo:        1,
		TsUnixMs:     uint64(time.Now().UnixMilli()),
		InstrumentID: 100,
		Price:        10100,
		Qty:          5,
		Side:         1,
	})
	if err != nil {
		t.Fatalf("EncodeTickFrame f1: %v", err)
	}

	f2, err := EncodeTickFrame(TickFrame{
		SeqNo:        2,
		TsUnixMs:     uint64(time.Now().Add(1 * time.Second).UnixMilli()),
		InstrumentID: 100,
		Price:        10110,
		Qty:          3,
		Side:         2,
	})
	if err != nil {
		t.Fatalf("EncodeTickFrame f2: %v", err)
	}

	stream := append([]byte{0x00, 0x01, 0x02, 0x03}, f1...)
	stream = append(stream, 0xAB, 0xCD)
	stream = append(stream, f2...)

	p := NewStreamParser(true)
	var parsed []TickFrame
	var dropped int

	parts := [][]byte{
		stream[:13],
		stream[13:17],
		stream[17:68],
		stream[68:],
	}
	for _, part := range parts {
		frames, d := p.Feed(part)
		parsed = append(parsed, frames...)
		dropped += d
	}

	if len(parsed) != 2 {
		t.Fatalf("parsed len = %d, want 2", len(parsed))
	}
	if parsed[0].SeqNo != 1 || parsed[1].SeqNo != 2 {
		t.Fatalf("unexpected seq numbers: %+v", parsed)
	}
	if dropped < 6 {
		t.Fatalf("dropped=%d, expected at least noise bytes", dropped)
	}
}

func TestStreamParserInvalidFrameIsSkipped(t *testing.T) {
	valid, err := EncodeTickFrame(TickFrame{
		SeqNo:        11,
		TsUnixMs:     uint64(time.Now().UnixMilli()),
		InstrumentID: 7,
		Price:        12345,
		Qty:          8,
		Side:         1,
	})
	if err != nil {
		t.Fatalf("EncodeTickFrame: %v", err)
	}

	bad := bytes.Clone(valid)
	bad[2] = 0xFF // wrong version

	stream := append(bad, valid...)
	p := NewStreamParser(true)
	frames, _ := p.Feed(stream)

	if len(frames) != 1 {
		t.Fatalf("frames=%d, want 1", len(frames))
	}
	if frames[0].SeqNo != 11 {
		t.Fatalf("seq=%d, want 11", frames[0].SeqNo)
	}
}
