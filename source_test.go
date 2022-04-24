package nrelay

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/octu0/chanque"
)

var (
	_ (chanque.Worker) = (*testSourceWorker)(nil)
)

type testSourceWorker struct {
	mutex    *sync.Mutex
	subjects map[string]struct{}
}

func (s *testSourceWorker) get() map[string]struct{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.subjects
}

func (s *testSourceWorker) Enqueue(param interface{}) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	msg := param.(*nats.Msg)
	s.subjects[msg.Subject] = struct{}{}
	return true
}

func (s *testSourceWorker) CloseEnqueue() bool {
	return false
}

func (s *testSourceWorker) Shutdown() {}

func (s *testSourceWorker) ShutdownAndWait() {}

func (s *testSourceWorker) ForceStop() {}

func testNatsServerStart() (*server.Server, bool) {
	ns := server.New(&server.Options{
		Host:         "127.0.0.1",
		Port:         -1,
		HTTPPort:     -1,
		Cluster:      server.ClusterOpts{Port: -1},
		NoLog:        true,
		NoSigs:       true,
		Debug:        false,
		Trace:        false,
		MaxPayload:   int32(1024),
		PingInterval: 5 * time.Millisecond,
		MaxPingsOut:  10,
	})
	go ns.Start()
	if ns.ReadyForConnections(10*time.Second) != true {
		return nil, false
	}
	return ns, true
}

func TestMultipleSource(t *testing.T) {
	testPublish := func(tt *testing.T, urls []string, workerNum int) {
		msgCount := 1000
		workers := make([]*testSourceWorker, workerNum)
		for i := 0; i < workerNum; i += 1 {
			workers[i] = &testSourceWorker{new(sync.Mutex), make(map[string]struct{})}
		}
		cw := make([]chanque.Worker, workerNum)
		for i := 0; i < workerNum; i += 1 {
			cw[i] = workers[i]
		}

		ncs := make([]*nats.Conn, len(urls))
		for i, url := range urls {
			nc, err := nats.Connect(url)
			if err != nil {
				tt.Errorf("must no error")
			}
			ncs[i] = nc
		}
		defer func() {
			for _, nc := range ncs {
				nc.Close()
			}
		}()

		topicPrefix := "test.source"
		lg := log.New(&testDestinationLogWriter{tt}, tt.Name()+"@", log.LstdFlags)
		src := NewMultipleSource(urls, nil, lg)
		if err := src.Open(); err != nil {
			tt.Errorf("must no error")
		}
		if err := src.Subscribe(topicPrefix+".>", len(topicPrefix)+2, cw); err != nil {
			tt.Errorf("must no error")
		}

		for i, nc := range ncs {
			for j := 0; j < msgCount; j += 1 {
				nc.Publish(fmt.Sprintf("%s.%d.%d", topicPrefix, i, j), []byte(""))
			}
			nc.Flush()
		}

		<-time.After(100 * time.Millisecond)

		if err := src.Close(); err != nil {
			tt.Errorf("must no error")
		}

		allSubjects := make(map[string]struct{}, len(urls)*msgCount)
		for _, w := range workers {
			for subj, _ := range w.get() {
				allSubjects[subj] = struct{}{}
			}
		}

		if len(allSubjects) != (len(urls) * msgCount) {
			tt.Errorf("must %d msg published, actual:%d", len(urls)*msgCount, len(allSubjects))
		}
		for i, _ := range ncs {
			for j := 0; j < msgCount; j += 1 {
				subj := fmt.Sprintf("%s.%d.%d", topicPrefix, i, j)
				if _, ok := allSubjects[subj]; ok != true {
					tt.Errorf("not found %s", subj)
				}
			}
		}
	}

	t.Run("source/1/worker/1", func(tt *testing.T) {
		ns, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}

		url := fmt.Sprintf("nats://%s", ns.Addr().String())
		testPublish(tt, []string{url}, 1)
	})
	t.Run("source/1/worker/10", func(tt *testing.T) {
		ns, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}

		url := fmt.Sprintf("nats://%s", ns.Addr().String())
		testPublish(tt, []string{url}, 10)
	})
	t.Run("source/2/worker/1", func(tt *testing.T) {
		ns1, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}
		ns2, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}

		url1 := fmt.Sprintf("nats://%s", ns1.Addr().String())
		url2 := fmt.Sprintf("nats://%s", ns2.Addr().String())
		testPublish(tt, []string{url1, url2}, 1)
	})
	t.Run("source/2/worker/10", func(tt *testing.T) {
		ns1, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}
		ns2, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}

		url1 := fmt.Sprintf("nats://%s", ns1.Addr().String())
		url2 := fmt.Sprintf("nats://%s", ns2.Addr().String())
		testPublish(tt, []string{url1, url2}, 10)
	})
	t.Run("source/3/worker/1", func(tt *testing.T) {
		ns1, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}
		ns2, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}
		ns3, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}

		url1 := fmt.Sprintf("nats://%s", ns1.Addr().String())
		url2 := fmt.Sprintf("nats://%s", ns2.Addr().String())
		url3 := fmt.Sprintf("nats://%s", ns3.Addr().String())
		testPublish(tt, []string{url1, url2, url3}, 1)
	})
	t.Run("source/3/worker/10", func(tt *testing.T) {
		ns1, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}
		ns2, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}
		ns3, ok := testNatsServerStart()
		if ok != true {
			tt.Skip("unable to start a NATS Server")
		}

		url1 := fmt.Sprintf("nats://%s", ns1.Addr().String())
		url2 := fmt.Sprintf("nats://%s", ns2.Addr().String())
		url3 := fmt.Sprintf("nats://%s", ns3.Addr().String())
		testPublish(tt, []string{url1, url2, url3}, 10)
	})
}
