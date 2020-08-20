package nrelay

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/lafikl/consistent"
	"github.com/nats-io/nats.go"
	"github.com/octu0/chanque"
	"github.com/octu0/nats-pool"
	"github.com/rs/xid"
)

var (
	defaultFlushTimeout time.Duration = 5 * time.Second
)

type Subpub struct {
	mutex    *sync.Mutex
	connType string
	topic    string
	gid      string
	srcPool  *pool.ConnPool
	dstPool  *pool.ConnPool
	logger   *log.Logger
	executor *chanque.Executor
	conf     RelayClientConfig
	shards   *consistent.Consistent
	shardMap *sync.Map
	shardIds []string
	// Subscribe init
	sub      *nats.Subscription
	srcConn  *nats.Conn
	dstConns []*nats.Conn
	workers  []*pubWorker
}

func NewSubpub(
	connType string,
	topic string,
	srcPool *pool.ConnPool,
	dstPool *pool.ConnPool,
	logger *log.Logger,
	executor *chanque.Executor,
	conf RelayClientConfig,
) *Subpub {
	workerNum := conf.WorkerNum
	if workerNum < 1 {
		workerNum = 1
	}

	shards := consistent.New()
	shardIds := make([]string, workerNum)
	for i := 0; i < workerNum; i += 1 {
		sid := xid.New().String()
		shardIds[i] = sid
		shards.Add(sid)
	}

	return &Subpub{
		mutex:    new(sync.Mutex),
		connType: connType,
		topic:    topic,
		gid:      strings.Join([]string{topic, connType, xid.New().String()}, "-"),
		srcPool:  srcPool,
		dstPool:  dstPool,
		logger:   logger,
		executor: executor,
		conf:     conf,
		shards:   shards,
		shardMap: new(sync.Map),
		shardIds: shardIds,
		dstConns: make([]*nats.Conn, workerNum),
		workers:  make([]*pubWorker, workerNum),
	}
}
func (s *Subpub) Subscribe() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	prefixSize := s.conf.PrefixSize
	if prefixSize < 1 {
		prefixSize = 0
	}
	useLoadBalance := s.conf.UseLoadBalance

	srcConn, err := s.srcPool.Get()
	if err != nil {
		s.logger.Printf("error: [%s] relay nats src connection failure: %s", s.connType, err.Error())
		return err
	}

	for idx, _ := range s.shardIds {
		nc, err := s.dstPool.Get()
		if err != nil {
			s.logger.Printf("error: [%s] relay nats dst connection failure: %s", s.connType, err.Error())
			return err
		}
		s.dstConns[idx] = nc
	}

	for idx, _ := range s.shardIds {
		s.workers[idx] = createPubWorker(s.dstConns[idx], s.executor, s.logger)
	}

	fallbackPub := s.makeFallbackPublish(s.dstConns[0])

	var dispatcher nats.MsgHandler
	if useLoadBalance {
		dispatcher = s.makeLoadBalanceDispatcher(prefixSize, fallbackPub)
	} else {
		dispatcher = s.makeDispatcher(prefixSize, fallbackPub)
	}

	sub, err := srcConn.QueueSubscribe(s.topic, s.gid, dispatcher)
	if err != nil {
		s.logger.Printf("error: [%s] nats subscribe(topic:%s) failure: %s", s.connType, s.topic, err.Error())
		return err
	}
	srcConn.Flush()

	s.sub = sub
	s.srcConn = srcConn
	return nil
}

func (s *Subpub) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.sub != nil {
		if err := s.sub.Unsubscribe(); err != nil {
			return err
		}
	}
	for _, wrk := range s.workers {
		if wrk != nil {
			wrk.Stop()
		}
	}
	if s.srcConn != nil {
		s.srcConn.Drain()
		s.srcPool.Put(s.srcConn)
	}
	for _, dstConn := range s.dstConns {
		if dstConn != nil {
			dstConn.Drain()
			s.dstPool.Put(dstConn)
		}
	}
	return nil
}

func fullSubject(msg *nats.Msg) string {
	return msg.Subject
}

func prefixedSubject(size int) func(*nats.Msg) string {
	return func(msg *nats.Msg) string {
		if size < len(msg.Subject) {
			return msg.Subject[0:size]
		}
		return fullSubject(msg)
	}
}

func (s *Subpub) makeDispatcher(prefixSize int, fallback nats.MsgHandler) nats.MsgHandler {
	var subject func(*nats.Msg) string
	if 0 < prefixSize {
		subject = prefixedSubject(prefixSize)
	} else {
		subject = fullSubject
	}
	return func(msg *nats.Msg) {
		subj := subject(msg)
		sid, err := s.shards.GetLeast(subj)
		if err != nil {
			fallback(msg)
			return
		}
		val, ok := s.shardMap.Load(sid)
		if ok != true {
			fallback(msg)
			return
		}
		pubWorker := val.(*pubWorker)
		pubWorker.Enqueue(msg)
	}
}

func (s *Subpub) makeLoadBalanceDispatcher(prefixSize int, fallback nats.MsgHandler) nats.MsgHandler {
	var subject func(*nats.Msg) string
	if 0 < prefixSize {
		subject = prefixedSubject(prefixSize)
	} else {
		subject = fullSubject
	}
	return func(msg *nats.Msg) {
		subj := subject(msg)
		sid, err := s.shards.GetLeast(subj)
		if err != nil {
			fallback(msg)
			return
		}
		val, ok := s.shardMap.Load(sid)
		if ok != true {
			fallback(msg)
			return
		}
		s.shards.Inc(sid)
		defer s.shards.Done(sid)

		pubWorker := val.(*pubWorker)
		pubWorker.Enqueue(msg)
	}
}

func (s *Subpub) makeFallbackPublish(nc *nats.Conn) nats.MsgHandler {
	return func(msg *nats.Msg) {
		nc.Publish(msg.Subject, msg.Data)
		nc.FlushTimeout(defaultFlushTimeout)
	}
}

type pubWorker struct {
	worker chanque.Worker
}

func createPubWorker(dst *nats.Conn, executor *chanque.Executor, logger *log.Logger) *pubWorker {
	publishHandler := createPublishHandler(dst)
	flushHandler := createFlushHandler(dst)
	panicHandler := createWorkerPanicHandler(logger)
	worker := chanque.NewBufferWorker(publishHandler,
		chanque.WorkerPostHook(flushHandler),
		chanque.WorkerPanicHandler(panicHandler),
		chanque.WorkerExecutor(executor),
	)
	return &pubWorker{worker}
}
func (w *pubWorker) Enqueue(msg *nats.Msg) {
	w.worker.Enqueue(msg)
}
func (w *pubWorker) Stop() {
	w.worker.Shutdown()
}

func createPublishHandler(conn *nats.Conn) chanque.WorkerHandler {
	return func(parameter interface{}) {
		msg := parameter.(*nats.Msg)
		conn.Publish(msg.Subject, msg.Data)
	}
}
func createFlushHandler(conn *nats.Conn) chanque.WorkerHook {
	return func() {
		conn.FlushTimeout(defaultFlushTimeout)
	}
}
func createWorkerPanicHandler(logger *log.Logger) chanque.PanicHandler {
	return func(panicType chanque.PanicType, rcv interface{}) {
		logger.Printf("error: [recover] worker happen(on %s): %v", panicType, rcv)
	}
}
