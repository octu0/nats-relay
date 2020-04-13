package nrelay

import(
  "log"
  "sync"
  "strings"

  "github.com/nats-io/nats.go"
)

type Subpub struct {
  mutex      *sync.Mutex
  srcConn    *nats.Conn
  dstConn    *nats.Conn
  logger     *log.Logger
  subs       []*nats.Subscription
  connType   string
  topic      string
}
func NewSubpub(connType string, src, dst *nats.Conn, logger *log.Logger) *Subpub {
  s         := new(Subpub)
  s.mutex    = new(sync.Mutex)
  s.srcConn  = src
  s.dstConn  = dst
  s.connType = connType
  s.logger   = logger
  return s
}
func (s *Subpub) pub(msg *nats.Msg) {
  s.dstConn.Publish(msg.Subject, msg.Data)
}
func (s *Subpub) Subscribe(topic, group string, numWorker int) error {
  s.mutex.Lock()
  defer s.mutex.Unlock()

  subs := make([]*nats.Subscription, numWorker)
  for i := 0; i < numWorker; i += 1 {
    sub, err := s.srcConn.QueueSubscribe(topic, group, s.pub)
    if err != nil {
      s.logger.Printf("error: topic(%s) subscribe failure: %s", topic, group)
      return err
    }
    subs[i] = sub
  }
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

  return nil
}
func (s *Subpub) String() string {
  return strings.Join([]string{"subpub", s.connType, s.topic}, "/")
}
