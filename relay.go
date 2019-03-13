package nrelay

import (
  "log"
  "context"
  "strings"

  "github.com/nats-io/go-nats"
  "github.com/rs/xid"
)

type RelayServer struct {
  priConn   *nats.Conn
  secConn   *nats.Conn
  nsConn    *nats.Conn
  opts      []nats.Option
  conf      RelayConfig
  subpubs   []*Subpub
  logger    *log.Logger
}

func NewRelayServer(ctx context.Context) *RelayServer {
  r := new(RelayServer)
  r.opts = []nats.Option{
    nats.NoEcho(),
    nats.Name(UA),
  }
  opts := ctx.Value("relay.nats-options")
  if opts != nil {
    r.opts = append(r.opts, opts.([]nats.Option)...)
  }

  r.conf = ctx.Value("relay.config").(RelayConfig)

  var errorHandler nats.ErrHandler
  var ctxErrHandler interface{}
  ctxErrHandler = ctx.Value("relay.error-handler")
  if ctxErrHandler != nil {
    errorHandler = ctxErrHandler.(nats.ErrHandler)
  } else {
    errorHandler = r.ErrorHandler
  }
  r.opts   = append(r.opts, nats.ErrorHandler(errorHandler))

  var ctxLogger interface{}
  ctxLogger = ctx.Value("relay.logger")
  if ctxLogger != nil {
    r.logger = ctxLogger.(*log.Logger)
  } else {
    r.logger = ctx.Value("logger.general").(*GeneralLogger).Logger()
  }

  return r
}
func (r *RelayServer) ErrorHandler(nc *nats.Conn, sub *nats.Subscription, err error) {
  r.logger.Printf("error: %s", err.Error())
}
func (r *RelayServer) Start(sctx context.Context) error {
  r.logger.Printf("info: relay server starting")

  pnc, perr := nats.Connect(r.conf.PrimaryUrl, r.opts...)
  if perr != nil {
    r.logger.Printf("error: primary(%s) connection failure", r.conf.PrimaryUrl)
    return perr
  }
  r.priConn = pnc

  if 0 < len(r.conf.SecondaryUrl) {
    // secondary nats are optional
    snc, serr := nats.Connect(r.conf.SecondaryUrl, r.opts...)
    if serr != nil {
      r.logger.Printf("error: secondary(%s) connection failure", r.conf.SecondaryUrl)
      return serr
    }
    r.secConn = snc
  }

  nnc, nerr := nats.Connect(r.conf.NatsUrl, r.opts...)
  if nerr != nil {
    r.logger.Printf("error: nats(%s) connection failure", r.conf.NatsUrl)
    return nerr
  }
  r.nsConn  = nnc

  if err := r.SubscribeTopics(r.priConn, "primary"); err != nil {
    r.logger.Printf("error: primary subscription failure")
    r.Close()
    return err
  }

  if r.secConn != nil {
    if err := r.SubscribeTopics(r.secConn, "secondary"); err != nil {
      r.logger.Printf("error: secondary subscription failure")
      r.Close()
      return err
    }
  }

  return nil
}
func (r *RelayServer) Stop(sctx context.Context) error {
  r.logger.Printf("info: relay server stoping")
  return nil
}
func (r *RelayServer) Close() error {
  for _, subpub := range r.subpubs {
    subpub.Close()
  }

  if r.secConn != nil {
    // secondary disconn
    if err := r.secConn.Drain(); err != nil {
      return err
    }
    r.secConn.Close()
    r.secConn = nil
  }

  if r.priConn != nil {
    // primary disconn
    if err := r.priConn.Drain(); err != nil {
      return err
    }
    r.priConn.Close()
    r.priConn = nil
  }

  if r.nsConn != nil {
    // todo: check buffered len before flush
    if err := r.nsConn.Flush(); err != nil {
      return err
    }
    r.nsConn.Close()
    r.nsConn = nil
  }
  return nil
}

func (r *RelayServer) SubscribeTopics(src *nats.Conn, connType string) error {
  var lastErr error
  sp := make([]*Subpub, 0)
  for topic, clientConf := range r.conf.Topics {
    guid   := xid.New()
    group  := r.makeGroupName(topic, guid)
    numWorker := clientConf.WorkerNum

    // single instance subs many goroutines
    subpub := NewSubpub(connType, src, r.nsConn, r.logger)
    for i := 0; i < numWorker; i = i + 1 {
      if err := subpub.Subscribe(topic, group); err != nil {
        r.logger.Printf("error: subpub subscribe failure: %s", err.Error())
        lastErr = err
      }
    }
    sp = append(sp, subpub)

    if lastErr == nil {
      log.Printf("info: subscribing %s to %d deque worker", topic, numWorker)
    }
  }
  src.Flush()

  r.subpubs = sp
  if lastErr != nil {
    r.Close()
  }
  return lastErr
}
func (r *RelayServer) makeGroupName(topic string, guid xid.ID) string {
  return strings.Join([]string{"group", guid.String()}, "-")
}
