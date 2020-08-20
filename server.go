package nrelay

import (
  "log"
  "io/ioutil"
  "sync"

  "github.com/nats-io/nats.go"
  "github.com/octu0/chanque"
  "github.com/octu0/nats-pool"
)

type ServerConfig struct {
  Logger    *log.Logger
  Executor  *chanque.Executor
}

func NewServerConfig(logger *log.Logger, executor *chanque.Executor) *ServerConfig {
  return &ServerConfig{
    Logger:   logger,
    Executor: executor,
  }
}

func DefaultServerConfig() *ServerConfig {
  return new(ServerConfig)
}

type Server struct {
  mutex         *sync.Mutex
  relayConf     RelayConfig
  logger        *log.Logger
  executor      *chanque.Executor
  primaryPool   *pool.ConnPool
  secondaryPool *pool.ConnPool
  localPool     *pool.ConnPool
  subpubs       []*Subpub
}

func NewServer(svrConf *ServerConfig, relayConf RelayConfig, options ...nats.Option) *Server {
  logger   := svrConf.Logger
  executor := svrConf.Executor

  topicNum     := 0
  maxWorkerNum := 0
  for _, conf := range relayConf.Topics {
    topicNum     += 1
    maxWorkerNum += conf.WorkerNum
  }
  if logger == nil {
    logger = log.New(ioutil.Discard, "", 0)
  }
  if executor == nil {
    executor = chanque.NewExecutor(0, maxWorkerNum)
  }

  natsOpts      := append(options, nats.NoEcho(), nats.Name(UA), nats.MaxReconnects(-1))

  var primaryPool,secondaryPool,localPool *pool.ConnPool
  localPool     = pool.New(maxWorkerNum, relayConf.NatsUrl, natsOpts...)
  primaryPool   = pool.New(maxWorkerNum, relayConf.PrimaryUrl, natsOpts...)
  if 0 < len(relayConf.SecondaryUrl) {
    secondaryPool = pool.New(maxWorkerNum, relayConf.SecondaryUrl, natsOpts...)
    topicNum      = topicNum * 2
  }

  return &Server{
    mutex:         new(sync.Mutex),
    relayConf:     relayConf,
    logger:        logger,
    executor:      executor,
    primaryPool:   primaryPool,
    secondaryPool: secondaryPool,
    localPool:     localPool,
    subpubs:       make([]*Subpub, 0, topicNum),
  }
}

func (r *Server) createSubpub(connType string, targetPool *pool.ConnPool) []*Subpub {
  subs := make([]*Subpub, 0, len(r.relayConf.Topics))
  for topicName, conf := range r.relayConf.Topics {
    s   := NewSubpub(connType, topicName, targetPool, r.localPool, r.logger, r.executor, conf)
    subs = append(subs, s)
  }
  return subs
}

func (r *Server) Start() error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  r.logger.Printf("info: relay server starting")

  r.subpubs = append(r.subpubs, r.createSubpub("primary", r.primaryPool)...)
  if r.secondaryPool != nil {
    r.subpubs = append(r.subpubs, r.createSubpub("secondary", r.secondaryPool)...)
  }

  for _, subpub := range r.subpubs {
    if err := subpub.Subscribe(); err != nil {
      return err
    }
  }

  r.logger.Printf("info: relay server stared")
  return nil
}
func (r *Server) Stop() error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  r.logger.Printf("info: relay server stoping")

  for _, subpub := range r.subpubs {
    if err := subpub.Stop(); err != nil {
      return err
    }
  }

  r.logger.Printf("info: relay server stoped")
  return nil
}
