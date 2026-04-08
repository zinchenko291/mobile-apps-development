package vtp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"time"
)

const (
	Magic0          = byte(0x4D) // 'M'
	Magic1          = byte(0x58) // 'X'
	Version1        = byte(0x01)
	MsgTypeTick     = byte(0x01)
	FrameSize       = 49
	frameWithoutCRC = FrameSize - 4
)

var (
	ErrFrameSize    = errors.New("invalid frame size")
	ErrMagic        = errors.New("invalid magic")
	ErrVersion      = errors.New("invalid version")
	ErrMsgType      = errors.New("invalid message type")
	ErrSide         = errors.New("invalid side")
	ErrCRC          = errors.New("crc mismatch")
	ErrTimestamp    = errors.New("invalid timestamp")
	ErrInstrumentID = errors.New("invalid instrument id")
)

// TickFrame is the normalized VTP-1 tick payload.
type TickFrame struct {
	SeqNo        uint64
	TsUnixMs     uint64
	InstrumentID uint64
	Price        int64
	Qty          uint64
	Side         uint8
}

func (t TickFrame) Validate() error {
	if t.TsUnixMs == 0 {
		return ErrTimestamp
	}
	if t.InstrumentID == 0 {
		return ErrInstrumentID
	}
	if t.Side > 2 {
		return ErrSide
	}
	return nil
}

func (t TickFrame) ExchangeTimeUTC() time.Time {
	return time.UnixMilli(int64(t.TsUnixMs)).UTC()
}

// EncodeTickFrame packs one TickFrame into a binary TickFrame v1.
// Encoding uses big-endian byte order.
func EncodeTickFrame(t TickFrame) ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	buf := make([]byte, FrameSize)
	buf[0] = Magic0
	buf[1] = Magic1
	buf[2] = Version1
	buf[3] = MsgTypeTick
	binary.BigEndian.PutUint64(buf[4:12], t.SeqNo)
	binary.BigEndian.PutUint64(buf[12:20], t.TsUnixMs)
	binary.BigEndian.PutUint64(buf[20:28], t.InstrumentID)
	binary.BigEndian.PutUint64(buf[28:36], uint64(t.Price))
	binary.BigEndian.PutUint64(buf[36:44], t.Qty)
	buf[44] = t.Side
	crc := crc32.ChecksumIEEE(buf[:frameWithoutCRC])
	binary.BigEndian.PutUint32(buf[45:49], crc)
	return buf, nil
}

// ParseTickFrame decodes a single binary TickFrame v1.
func ParseTickFrame(frame []byte, verifyCRC bool) (TickFrame, error) {
	if len(frame) != FrameSize {
		return TickFrame{}, fmt.Errorf("%w: got=%d want=%d", ErrFrameSize, len(frame), FrameSize)
	}
	if frame[0] != Magic0 || frame[1] != Magic1 {
		return TickFrame{}, ErrMagic
	}
	if frame[2] != Version1 {
		return TickFrame{}, ErrVersion
	}
	if frame[3] != MsgTypeTick {
		return TickFrame{}, ErrMsgType
	}

	crcExpected := binary.BigEndian.Uint32(frame[45:49])
	if verifyCRC {
		crcActual := crc32.ChecksumIEEE(frame[:frameWithoutCRC])
		if crcActual != crcExpected {
			return TickFrame{}, fmt.Errorf("%w: got=%08x want=%08x", ErrCRC, crcActual, crcExpected)
		}
	}

	out := TickFrame{
		SeqNo:        binary.BigEndian.Uint64(frame[4:12]),
		TsUnixMs:     binary.BigEndian.Uint64(frame[12:20]),
		InstrumentID: binary.BigEndian.Uint64(frame[20:28]),
		Price:        int64(binary.BigEndian.Uint64(frame[28:36])),
		Qty:          binary.BigEndian.Uint64(frame[36:44]),
		Side:         frame[44],
	}
	if err := out.Validate(); err != nil {
		return TickFrame{}, err
	}
	return out, nil
}
