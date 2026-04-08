package randomticks

import "testing"

func TestFeedProducesTicks(t *testing.T) {
	d := NewStreamDecoder(0)

	// 2 full chunks + 3 bytes tail
	raw := make([]byte, chunkSize*2+3)
	for i := range raw {
		raw[i] = byte(i + 1)
	}

	frames, dropped := d.Feed(raw)
	if dropped != 0 {
		t.Fatalf("dropped=%d, want 0", dropped)
	}
	if len(frames) != 2 {
		t.Fatalf("frames=%d, want 2", len(frames))
	}
	if frames[0].SeqNo != 1 || frames[1].SeqNo != 2 {
		t.Fatalf("unexpected seq numbers: %d %d", frames[0].SeqNo, frames[1].SeqNo)
	}
	if frames[0].InstrumentID == 0 || frames[1].InstrumentID == 0 {
		t.Fatalf("instrument id should not be zero")
	}
	if frames[0].Qty == 0 || frames[1].Qty == 0 {
		t.Fatalf("qty should not be zero")
	}
	if frames[0].Side < 1 || frames[0].Side > 2 {
		t.Fatalf("invalid side=%d", frames[0].Side)
	}
}

func TestFeedKeepsTailBetweenCalls(t *testing.T) {
	d := NewStreamDecoder(0)

	part1 := make([]byte, chunkSize-5)
	part2 := make([]byte, 5)

	frames, _ := d.Feed(part1)
	if len(frames) != 0 {
		t.Fatalf("frames from first feed=%d, want 0", len(frames))
	}

	frames, _ = d.Feed(part2)
	if len(frames) != 1 {
		t.Fatalf("frames from second feed=%d, want 1", len(frames))
	}
}
