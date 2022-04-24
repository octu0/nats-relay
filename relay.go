package nrelay

import (
	"context"
	"log"

	"github.com/pkg/errors"
)

type Relay interface {
	Run(context.Context) error
}

// check interface
var (
	_ Relay = (*MultipleSourceSingleDestinationRelay)(nil)
)

type MultipleSourceSingleDestinationRelay struct {
	topic      string
	src        Source
	dst        Destination
	prefixSize int
	workerNum  int
	logger     *log.Logger
}

func (r *MultipleSourceSingleDestinationRelay) Run(ctx context.Context) error {
	r.logger.Printf("info: relay/forward stream start:%s", r.topic)
	if err := r.src.Open(); err != nil {
		return errors.WithStack(err)
	}
	if err := r.dst.Open(r.workerNum); err != nil {
		return errors.WithStack(err)
	}

	if err := r.src.Subscribe(r.topic, r.prefixSize, r.dst.Workers()); err != nil {
		return errors.WithStack(err)
	}

	<-ctx.Done()

	r.logger.Printf("info: relay/forward stream stop:%s", r.topic)
	if err := r.src.Unsubscribe(); err != nil {
		return errors.WithStack(err)
	}

	if err := r.src.Close(); err != nil {
		return errors.WithStack(err)
	}
	if err := r.dst.Close(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func NewMultipleSourceSingleDestinationRelay(topic string, src Source, dst Destination, prefix, num int, logger *log.Logger) *MultipleSourceSingleDestinationRelay {
	return &MultipleSourceSingleDestinationRelay{topic, src, dst, prefix, num, logger}
}
