package randomticks

import (
	"encoding/binary"
	"time"

	"mobilki/internal/vtp"
)

const (
	// One synthetic tick is produced from each 24-byte chunk.
	chunkSize = 24
)

// StreamDecoder transforms arbitrary bytes into synthetic tick frames.
type StreamDecoder struct {
	buf []byte
	seq uint64
}

func NewStreamDecoder(startSeq uint64) *StreamDecoder {
	return &StreamDecoder{
		buf: make([]byte, 0, chunkSize*8),
		seq: startSeq,
	}
}

func (d *StreamDecoder) Feed(chunk []byte) ([]vtp.TickFrame, int) {
	if len(chunk) == 0 {
		return nil, 0
	}

	d.buf = append(d.buf, chunk...)
	frames := make([]vtp.TickFrame, 0, len(d.buf)/chunkSize)

	for len(d.buf) >= chunkSize {
		block := d.buf[:chunkSize]
		d.seq++

		instrumentID := uint64(1000 + (binary.LittleEndian.Uint32(block[0:4]) % 500))
		price := int64(10_000 + (binary.LittleEndian.Uint64(block[4:12]) % 900_000))
		qty := uint64(1 + (binary.LittleEndian.Uint32(block[12:16]) % 100))
		side := uint8(1 + (block[16] % 2))

		frames = append(frames, vtp.TickFrame{
			SeqNo:        d.seq,
			TsUnixMs:     uint64(time.Now().UnixMilli()),
			InstrumentID: instrumentID,
			Price:        price,
			Qty:          qty,
			Side:         side,
		})

		d.buf = d.buf[chunkSize:]
	}

	return frames, 0
}
