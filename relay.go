package nrelay

import (
  "log"
  "context"
  "strings"
  "sync"

  "github.com/nats-io/nats.go"
  "github.com/rs/xid"
)

type RelayServer struct {
  mutex     *sync.Mutex
  priConn   *nats.Conn
  secConn   *nats.Conn
  opts      []nats.Option
  conf      RelayConfig
  subpubs   []*Subpub
  logger    *log.Logger
}

func NewRelayServer(ctx context.Context) *RelayServer {
  r := new(RelayServer)
  r.mutex = new(sync.Mutex)
  r.opts  = []nats.Option{
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
  r.opts = append(r.opts, nats.ErrorHandler(errorHandler))

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
  r.mutex.Lock()
  defer r.mutex.Unlock()

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

  if err := r.subscribeTopics(r.priConn, "primary"); err != nil {
    r.logger.Printf("error: primary subscription failure")
    r.closeAll()
    return err
  }

  if r.secConn != nil {
    if err := r.subscribeTopics(r.secConn, "secondary"); err != nil {
      r.logger.Printf("error: secondary subscription failure")
      r.closeAll()
      return err
    }
  }

  r.logger.Printf("info: relay server stared")
  return nil
}
func (r *RelayServer) Stop(sctx context.Context) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()
  r.logger.Printf("info: relay server stoping")

  if err := r.closeAll(); err != nil {
    return err
  }

  r.logger.Printf("info: relay server stoped")
  return nil
}
func (r *RelayServer) closeAll() error {
  r.logger.Printf("debug: relay subpub close")
  for _, subpub := range r.subpubs {
    subpub.Close()
  }

  r.logger.Printf("debug: relay secondary conn close")
  if r.secConn != nil {
    defer r.secConn.Close()
    // secondary disconn
    if err := r.secConn.Drain(); err != nil {
      return err
    }
  }

  r.logger.Printf("debug: relay primary conn close")
  if r.priConn != nil {
    defer r.priConn.Close()
    // primary disconn
    if err := r.priConn.Drain(); err != nil {
      return err
    }
  }

  return nil
}
func (r *RelayServer) Close() error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  return r.closeAll()
}
func (r *RelayServer) createRelayConn(id int) (*nats.Conn, error) {
  return nats.Connect(r.conf.NatsUrl, r.opts...)
}
func (r *RelayServer) subscribeTopics(src *nats.Conn, connType string) error {
  var lastErr error
  sp := make([]*Subpub, 0, len(r.conf.Topics))
  for topic, clientConf := range r.conf.Topics {
    guid           := xid.New()
    group          := r.makeGroupName(topic, guid)
    numWorker      := clientConf.WorkerNum
    numShard       := clientConf.ShardNum
    prefixSize     := clientConf.PrefixSize
    useLoadBalance := clientConf.UseLoadBalance

    // single instance subs many goroutines
    subpub := NewSubpub(connType, src, r.logger, r.createRelayConn)
    if err := subpub.Subscribe(topic, group, numWorker, numShard, prefixSize, useLoadBalance); err != nil {
      r.logger.Printf("error: subpub subscribe failure: %s", err.Error())
      lastErr = err
    }
    sp = append(sp, subpub)

    if lastErr == nil {
      log.Printf("info: subscribing %s to %d deque worker", topic, numWorker)
    }
  }
  src.Flush()

  r.subpubs = sp
  if lastErr != nil {
    r.closeAll()
  }
  return lastErr
}
func (r *RelayServer) makeGroupName(topic string, guid xid.ID) string {
  return strings.Join([]string{"group", guid.String()}, "-")
}
