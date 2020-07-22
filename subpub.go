package nrelay

import(
  "log"
  "sync"
  "time"
  "strings"

  "github.com/nats-io/nats.go"
  "github.com/lafikl/consistent"
  "github.com/rs/xid"
  "github.com/octu0/chanque"
)

var(
  flushTO  = 5 * time.Millisecond
)

type connFactory func(id int) (*nats.Conn, error)

type syncWorker struct {
  mutex  *sync.Mutex
  worker chanque.Worker
}

type Subpub struct {
  mutex      *sync.Mutex
  srcConn    *nats.Conn
  connType   string
  logger     *log.Logger
  factory    connFactory
  sharding   *consistent.Consistent
  dstMap     *sync.Map
  sids       []string
  fallback   *nats.Conn

  dstConns   []*nats.Conn
  subs       []*nats.Subscription
  topic      string
}
func NewSubpub(connType string, src *nats.Conn, logger *log.Logger, factory connFactory) *Subpub {
  s         := new(Subpub)
  s.mutex    = new(sync.Mutex)
  s.srcConn  = src
  s.connType = connType
  s.logger   = logger
  s.factory  = factory
  s.sharding = consistent.New()
  s.dstMap   = new(sync.Map)
  return s
}
func (s *Subpub) makeDispatcher(fallback *nats.Conn, prefixSize int, useLoadBalance bool) nats.MsgHandler {
  if prefixSize < 1 {
    prefixSize = 0
  }
  fpub := func(msg *nats.Msg) {
    fallback.Publish(msg.Subject, msg.Data)
    fallback.FlushTimeout(flushTO)
  }
  subject := func(msg *nats.Msg) string {
    subj := msg.Subject
    if 0 < prefixSize && prefixSize < len(subj) {
      subj = subj[0 : prefixSize]
    }
    return subj
  }
  if useLoadBalance {
    return func(msg *nats.Msg){
      subj   := subject(msg)
      sid, err := s.sharding.GetLeast(subj)
      if err != nil {
        fpub(msg)
        return
      }
      val, ok := s.dstMap.Load(sid)
      if ok != true {
        fpub(msg)
        return
      }
      s.sharding.Inc(sid)
      defer s.sharding.Done(sid)

      sw := val.(*syncWorker)
      sw.mutex.Lock()
      defer sw.mutex.Unlock()
      sw.worker.Enqueue(msg)
    }
  }
  return func(msg *nats.Msg) {
    subj     := subject(msg)
    sid, err := s.sharding.Get(subj)
    if err != nil {
      fpub(msg)
      return
    }

    val, ok := s.dstMap.Load(sid)
    if ok != true {
      fpub(msg)
      return
    }

    sw := val.(*syncWorker)
    sw.mutex.Lock()
    defer sw.mutex.Unlock()
    sw.worker.Enqueue(msg)
  }
}
func (s *Subpub) createPublishHandler(conn *nats.Conn) chanque.WorkerHandler {
  return func(parameter interface{}) {
    msg := parameter.(*nats.Msg)
    conn.Publish(msg.Subject, msg.Data)
  }
}
func (s *Subpub) createFlushHandler(conn *nats.Conn) chanque.WorkerHook {
  return func(){
    conn.FlushTimeout(flushTO)
  }
}
func (s *Subpub) createWorkerPanicHandler() chanque.PanicHandler {
  return func(panicType chanque.PanicType, rcv interface{}) {
    s.logger.Printf("error: [recover] worker happen(on %s): %v", panicType, rcv)
  }
}
func (s *Subpub) Subscribe(topic, group string, numWorker, numShard int, prefixSize int, useLoadBalance bool, executor *chanque.Executor) error {
  s.mutex.Lock()
  defer s.mutex.Unlock()

  if numShard < 1 {
    numShard = 1
  }

  sids := make([]string, 0, numShard)
  for n := 0; n < numShard; n += 1 {
    dst, err := s.factory(n + 1)
    if err != nil {
      s.logger.Printf("error: relay nats(%d) connection failure: %s", n, err.Error())
      return err
    }

    for i := 0; i < numWorker; i += 1 {
      publishHandler := s.createPublishHandler(dst)
      flushHandler   := s.createFlushHandler(dst)
      panicHandler   := s.createWorkerPanicHandler()
      worker         := chanque.NewBufferWorker(publishHandler,
        chanque.WorkerPostHook(flushHandler),
        chanque.WorkerPanicHandler(panicHandler),
        chanque.WorkerExecutor(executor),
      )

      sid := xid.New().String()
      s.sharding.Add(sid)
      s.dstMap.Store(sid, &syncWorker{
        mutex:  new(sync.Mutex),
        worker: worker,
      })
      sids = append(sids, sid)
    }
  }
  s.sids = sids

  fallbackConn, err := s.factory(0)
  if err != nil {
    s.logger.Printf("error: relay fallback conn failure: %s", err.Error())
    return err
  }
  s.fallback = fallbackConn

  subs := make([]*nats.Subscription, 0)
  dispatcher := s.makeDispatcher(fallbackConn, prefixSize, useLoadBalance)
  sub, err := s.srcConn.QueueSubscribe(topic, group, dispatcher)
  if err != nil {
    s.logger.Printf("error: topic(%s) subscribe failure: %s", topic, group)
    return err
  }
  subs = append(subs, sub)
  s.srcConn.Flush()

  s.subs    = subs
  s.topic   = topic
  return nil
}
func (s *Subpub) Close() error {
  s.mutex.Lock()
  defer s.mutex.Unlock()

  for _, sub := range s.subs {
    if err := sub.Unsubscribe(); err != nil {
      return err
    }
  }
  s.srcConn.Drain()

  if 0 < len(s.sids) {
    for _, sid := range s.sids {
      if val, ok := s.dstMap.Load(sid); ok {
        sw     := val.(*syncWorker)
        sw.mutex.Lock()
        sw.worker.ShutdownAndWait()
        s.dstMap.Delete(sid)
        sw.mutex.Unlock()
      }
    }
  }
  if s.fallback != nil {
    if err := s.fallback.Drain(); err != nil {
      s.logger.Printf("warn: close fallback drain error: %s", err.Error())
    }
    s.fallback.Close()
  }

  return nil
}
func (s *Subpub) String() string {
  return strings.Join([]string{"subpub", s.connType, s.topic}, "/")
}
