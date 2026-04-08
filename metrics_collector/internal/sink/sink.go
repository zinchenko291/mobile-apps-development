package sink

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"mobilki/internal/vtp"
)

type Sink interface {
	Name() string
	Publish(ctx context.Context, tick vtp.TickFrame) error
	Close() error
}

type MultiSink struct {
	sinks []Sink
}

func NewMultiSink(sinks ...Sink) *MultiSink {
	return &MultiSink{sinks: sinks}
}

func (m *MultiSink) Name() string {
	return "multi"
}

func (m *MultiSink) Len() int {
	return len(m.sinks)
}

func (m *MultiSink) Publish(ctx context.Context, tick vtp.TickFrame) error {
	errs := make([]string, 0, len(m.sinks))
	for _, s := range m.sinks {
		if err := s.Publish(ctx, tick); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", s.Name(), err))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (m *MultiSink) Close() error {
	errs := make([]string, 0, len(m.sinks))
	for _, s := range m.sinks {
		if err := s.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", s.Name(), err))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
