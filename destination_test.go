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

type testDestinationLogWriter struct {
	t *testing.T
}

func (w *testDestinationLogWriter) Write(p []byte) (int, error) {
	if testing.Verbose() {
		w.t.Logf("%s", p)
	}
	return len(p), nil
}

func TestSingleDestination(t *testing.T) {
	testPublish := func(tt *testing.T, num int) {
		msgCount := 1000
		e := chanque.NewExecutor(100, 100)
		tt.Cleanup(func() { e.Release() })

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
			tt.Skip("unable to start a NATS Server")
		}
		defer ns.Shutdown()

		url := fmt.Sprintf("nats://%s", ns.Addr().String())
		lg := log.New(&testDestinationLogWriter{tt}, tt.Name()+"@", log.LstdFlags)
		dest := NewSingleDestination(e, url, nil, lg)

		subjects := make(map[string]struct{}, num*msgCount)
		m := new(sync.Mutex)

		nc, err := nats.Connect(url)
		if err != nil {
			tt.Fatalf("check subscriber connect failed: %+v", err)
		}
		defer nc.Close()

		sub, err := nc.Subscribe("test.>", func(msg *nats.Msg) {
			m.Lock()
			defer m.Unlock()
			subjects[msg.Subject] = struct{}{}
		})
		if err != nil {
			tt.Fatalf("check subscriber sub failed: %+v", err)
		}
		defer sub.Unsubscribe()
		nc.Flush()

		if err := dest.Open(num); err != nil {
			tt.Errorf("must no error: %+v", err)
		}

		for i, w := range dest.Workers() {
			for j := 0; j < msgCount; j += 1 {
				w.Enqueue(&nats.Msg{Subject: fmt.Sprintf("test.worker.%d.msg%d", i, j), Data: []byte("")})
			}
		}

		if err := dest.Close(); err != nil {
			tt.Errorf("must no error: %+v", err)
		}

		if len(subjects) != (num * msgCount) {
			tt.Errorf("must %d msg published, actual:%d", num*msgCount, len(subjects))
		}

		for i := 0; i < num; i += 1 {
			for j := 0; j < msgCount; j += 1 {
				subj := fmt.Sprintf("test.worker.%d.msg%d", i, j)
				if _, ok := subjects[subj]; ok != true {
					tt.Errorf("not found %s", subj)
				}
			}
		}
	}

	t.Run("publish/worker1", func(tt *testing.T) {
		testPublish(tt, 1)
	})
	t.Run("publish/worker10", func(tt *testing.T) {
		testPublish(tt, 10)
	})
	t.Run("publish/worker100", func(tt *testing.T) {
		testPublish(tt, 100)
	})
}
