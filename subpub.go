package nrelay

import(
  "log"
  "strings"

  "github.com/nats-io/nats.go"
)

type SubpubDone chan struct{}
type MsgQueue   chan *nats.Msg
type Subpub struct {
  srcConn    *nats.Conn
  dstConn    *nats.Conn
  msgChan    MsgQueue
  logger     *log.Logger
  sub        *nats.Subscription
  connType   string
  topic      string
  dones      []SubpubDone
}
func NewSubpub(connType string, src, dst *nats.Conn, logger *log.Logger) *Subpub {
  subpub         := new(Subpub)
  subpub.srcConn  = src
  subpub.dstConn  = dst
  subpub.msgChan  = make(MsgQueue, 0)
  subpub.connType = connType
  subpub.logger   = logger
  return subpub
}
func (s *Subpub) Subscribe(topic, group string, numWorker int) error {
  sub, err := s.srcConn.ChanQueueSubscribe(topic, group, s.msgChan)
  if err != nil {
    s.logger.Printf("error: topic(%s) subscribe failure: %s", topic, group)
    return err
  }
  s.srcConn.Flush()
  s.sub     = sub
  s.topic   = topic
  s.dones   = make([]SubpubDone, numWorker)
  for i := 0; i < numWorker; i += 1 {
    s.dones[i] = make(SubpubDone)
  }
  for i := 0; i < numWorker; i += 1 {
    go s.exchangeLoop(s.dones[i])
  }
  return nil
}
func (s *Subpub) exchangeLoop(done SubpubDone) {
  for {
    select {
    case <-done:
      return
    case msg, ok := <-s.msgChan:
      if ok != true {
        return
      }

      // ignore reply
      s.dstConn.Publish(msg.Subject, msg.Data)
    }
  }
}
func (s *Subpub) Close() error {
  if err := s.sub.Unsubscribe(); err != nil {
    return err
  }
  for _, done := range s.dones {
    done <- struct{}{}
  }
  close(s.msgChan)

  return nil
}
func (s *Subpub) String() string {
  return strings.Join([]string{"subpub", s.connType, s.topic}, "/")
}
