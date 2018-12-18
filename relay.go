package nrelay

import (
  "log"
  "context"
  "strings"

  "github.com/nats-io/go-nats"
)

type RelayServer struct {
  priConn   *nats.Conn
  secConn   *nats.Conn
  nsConn    *nats.Conn
  opts      []nats.Option
  conf      RelayConfig
  subpubs   []*Subpub
}

func NewRelayServer(ctx context.Context) *RelayServer {
  r := new(RelayServer)
  r.opts = []nats.Option{
    nats.NoEcho(),
    nats.Name(UA),
    nats.ErrorHandler(r.ErrorHandler),
  }
  r.conf = ctx.Value("relay-config").(RelayConfig)
  return r
}
func (r *RelayServer) ErrorHandler(nc *nats.Conn, sub *nats.Subscription, err error) {
  log.Printf("error: %s", err.Error())
}
func (r *RelayServer) Start(sctx context.Context) error {
  log.Printf("info: relay server starting")

  pnc, perr := nats.Connect(r.conf.PrimaryUrl, r.opts...)
  if perr != nil {
    log.Printf("error: primary(%s) connection failure", r.conf.PrimaryUrl)
    return perr
  }
  snc, serr := nats.Connect(r.conf.SecondaryUrl, r.opts...)
  if serr != nil {
    log.Printf("error: secondary(%s) connection failure", r.conf.SecondaryUrl)
    return serr
  }
  nnc, nerr := nats.Connect(r.conf.NatsUrl, r.opts...)
  if nerr != nil {
    log.Printf("error: nats(%s) connection failure", r.conf.NatsUrl)
    return nerr
  }

  r.priConn = pnc
  r.secConn = snc
  r.nsConn  = nnc

  if err := r.SubscribeTopics(r.priConn, r.conf.Topics); err != nil {
    log.Printf("error: primary subscription failure")
    r.Close()
    return err
  }
  if err := r.SubscribeTopics(r.secConn, r.conf.Topics); err != nil {
    log.Printf("error: secondary subscription failure")
    r.Close()
    return err
  }

  return nil
}
func (r *RelayServer) Stop(sctx context.Context) error {
  log.Printf("info: relay server stoping")
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

func (r *RelayServer) SubscribeTopics(src *nats.Conn, topics map[string]RelayClientConfig) error {
  var lastErr error
  sp := make([]*Subpub, 0)
  for topic, clientConf := range topics {
    group  := r.makeGroupName(topic)
    numWorker := clientConf.WorkerNum

    for i := 0; i < numWorker; i = i + 1 {
      // new conn?
      subpub := NewSubpub(src, r.nsConn)
      if err := subpub.Subscribe(topic, group); err != nil {
        log.Printf("error: subpub subscribe failure: %s", err.Error())
        lastErr = err
      }

      sp = append(sp, subpub)
    }
  }
  r.subpubs = sp
  if lastErr != nil {
    r.Close()
  }
  return lastErr
}
func (r *RelayServer) makeGroupName(topic string) string {
  return strings.Join([]string{topic, "group"}, "/")
}
