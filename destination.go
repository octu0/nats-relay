package nrelay

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/octu0/chanque"
	"github.com/pkg/errors"
)

const (
	defaultWorkerCapacity   int           = 1024
	defaultWorkerMaxMsgSize int           = 32
	defaultFlushTimeout     time.Duration = 50 * time.Millisecond
)

type Destination interface {
	Open(num int) error
	Close() error
	Workers() []chanque.Worker
}

// check interface
var (
	_ Destination = (*SingleDestination)(nil)
)

type SingleDestination struct {
	executor *chanque.Executor
	url      string
	natsOpts []nats.Option
	logger   *log.Logger
	conns    []*nats.Conn
	workers  []chanque.Worker
}

func (d *SingleDestination) Open(num int) error {
	conns := make([]*nats.Conn, 0, num)
	workers := make([]chanque.Worker, 0, num)

	for i := 0; i < num; i += 1 {
		conn, err := nats.Connect(d.url, d.natsOpts...)
		if err != nil {
			return errors.WithStack(err)
		}
		d.logger.Printf("debug: nats destination connect %s", d.url)

		conns = append(conns, conn)
		workers = append(workers, d.createWorker(conn))
	}
	d.conns = conns
	d.workers = workers
	return nil
}

func (d *SingleDestination) Close() error {
	for _, worker := range d.workers {
		worker.CloseEnqueue()
	}
	for _, worker := range d.workers {
		worker.ShutdownAndWait()
	}
	for _, conn := range d.conns {
		conn.Flush()
		conn.Drain()
	}
	return nil
}

func (d *SingleDestination) Workers() []chanque.Worker {
	return d.workers
}

func (d *SingleDestination) createWorker(conn *nats.Conn) chanque.Worker {
	return chanque.NewDefaultWorker(
		d.createWorkerHandler(conn),
		chanque.WorkerExecutor(d.executor),
		chanque.WorkerCapacity(defaultWorkerCapacity),
		chanque.WorkerMaxDequeueSize(defaultWorkerMaxMsgSize),
		chanque.WorkerPostHook(d.createWorkerPostHook(conn)),
		chanque.WorkerAbortQueueHandler(func(param interface{}) {
			d.logger.Printf("error: destination queue aborted: %v", param)
		}),
	)
}

func (d *SingleDestination) createWorkerHandler(conn *nats.Conn) chanque.WorkerHandler {
	return func(param interface{}) {
		msg := param.(*nats.Msg)
		if err := conn.Publish(msg.Subject, msg.Data); err != nil {
			d.logger.Printf("warn: failed to publish subj:%s err:%+v", msg.Subject, err)
		}
	}
}

func (d *SingleDestination) createWorkerPostHook(conn *nats.Conn) chanque.WorkerHook {
	return func() {
		conn.FlushTimeout(defaultFlushTimeout)
	}
}

func NewSingleDestination(executor *chanque.Executor, url string, natsOpts []nats.Option, logger *log.Logger) *SingleDestination {
	return &SingleDestination{executor, url, natsOpts, logger, nil, nil}
}
