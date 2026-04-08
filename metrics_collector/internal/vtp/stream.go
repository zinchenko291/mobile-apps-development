package vtp

import "bytes"

var magicPrefix = []byte{Magic0, Magic1}

// StreamParser accepts arbitrary byte chunks and extracts valid TickFrame v1 messages.
type StreamParser struct {
	verifyCRC bool
	buf       []byte
}

func NewStreamParser(verifyCRC bool) *StreamParser {
	return &StreamParser{
		verifyCRC: verifyCRC,
		buf:       make([]byte, 0, FrameSize*2),
	}
}

// Feed appends bytes and returns all successfully parsed frames plus count of dropped bytes.
func (p *StreamParser) Feed(chunk []byte) ([]TickFrame, int) {
	if len(chunk) == 0 {
		return nil, 0
	}

	p.buf = append(p.buf, chunk...)
	frames := make([]TickFrame, 0, 8)
	dropped := 0

	for {
		idx := bytes.Index(p.buf, magicPrefix)
		if idx < 0 {
			if len(p.buf) > 1 {
				dropped += len(p.buf) - 1
				p.buf = p.buf[len(p.buf)-1:]
			}
			break
		}
		if idx > 0 {
			dropped += idx
			p.buf = p.buf[idx:]
		}
		if len(p.buf) < FrameSize {
			break
		}

		frame, err := ParseTickFrame(p.buf[:FrameSize], p.verifyCRC)
		if err != nil {
			dropped++
			p.buf = p.buf[1:]
			continue
		}

		frames = append(frames, frame)
		p.buf = p.buf[FrameSize:]
	}

	return frames, dropped
}
