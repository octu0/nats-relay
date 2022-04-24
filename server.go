package nrelay

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/octu0/chanque"
	"github.com/pkg/errors"
)

type Server interface {
	Run(context.Context) error
}

type ServerOptFunc func(*serverOpt)

type serverOpt struct {
	relayConf RelayConfig
	executor  *chanque.Executor
	logger    *log.Logger
	natsOpts  []nats.Option
}

func ServerOptRelayConfig(conf RelayConfig) ServerOptFunc {
	return func(opt *serverOpt) {
		opt.relayConf = conf
	}
}

func ServerOptExecutor(executor *chanque.Executor) ServerOptFunc {
	return func(opt *serverOpt) {
		opt.executor = executor
	}
}

func ServerOptLogger(logger *log.Logger) ServerOptFunc {
	return func(opt *serverOpt) {
		opt.logger = logger
	}
}

func ServerOptNatsOptions(natsOpts ...nats.Option) ServerOptFunc {
	return func(opt *serverOpt) {
		opt.natsOpts = natsOpts
	}
}

// check interface
var (
	_ (Server) = (*DefaultServer)(nil)
)

type DefaultServer struct {
	opt *serverOpt
}

func (s *DefaultServer) Run(ctx context.Context) error {
	sourceNatsUrls := make([]string, 0, 2)
	sourceNatsUrls = append(sourceNatsUrls, s.opt.relayConf.PrimaryUrl)
	if 0 < len(s.opt.relayConf.SecondaryUrl) {
		sourceNatsUrls = append(sourceNatsUrls, s.opt.relayConf.SecondaryUrl)
	}

	relays := make([]Relay, 0, len(s.opt.relayConf.Topics)*len(sourceNatsUrls))
	for topic, conf := range s.opt.relayConf.Topics {
		src := NewMultipleSource(sourceNatsUrls, s.opt.natsOpts, s.opt.logger)
		dst := NewSingleDestination(s.opt.executor, s.opt.relayConf.NatsUrl, s.opt.natsOpts, s.opt.logger)
		relay := NewMultipleSourceSingleDestinationRelay(topic, src, dst, conf.PrefixSize, conf.WorkerNum, s.opt.logger)
		relays = append(relays, relay)
	}

	return runRelays(ctx, s.opt.executor, s.opt.logger, relays)
}

func NewDefaultServer(funcs ...ServerOptFunc) *DefaultServer {
	opt := new(serverOpt)
	for _, fn := range funcs {
		fn(opt)
	}
	return &DefaultServer{opt}
}

func runRelays(ctx context.Context, executor *chanque.Executor, logger *log.Logger, relays []Relay) error {
	logger.Printf("info: relay server stared")
	sctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 0)
	subexec := executor.SubExecutor()
	for i, relay := range relays {
		subexec.Submit(func(idx int, r Relay) chanque.Job {
			return func() {
				logger.Printf("debug: relay[%d] started", idx)
				defer logger.Printf("debug: relay[%d] stoped", idx)

				if err := r.Run(sctx); err != nil {
					select {
					case errCh <- errors.WithStack(err):
					default:
						// already shutdown
					}
				}
			}
		}(i, relay))
	}

	select {
	case <-ctx.Done():
		logger.Printf("info: relay server shutdown, context done")
		cancel()

		subexec.Wait()
		close(errCh)
		if err := <-errCh; err != nil {
			return errors.WithStack(err)
		}
		return nil
	case err := <-errCh:
		logger.Printf("info: relay server shutdown, error happen")
		cancel()

		subexec.Wait()
		close(errCh)
		return errors.WithStack(err)
	}
	return nil
}
