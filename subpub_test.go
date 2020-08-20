package nrelay

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"testing"
	"time"

	natsd "github.com/nats-io/nats-server/server"
	"github.com/nats-io/nats.go"
	"github.com/octu0/chanque"
)

func testStartNatsd(port int) (*natsd.Server, error) {
	opts := &natsd.Options{
		Host:            "127.0.0.1",
		Port:            port,
		ClientAdvertise: "127.0.0.1",
		HTTPPort:        -1,
		Cluster:         natsd.ClusterOpts{Port: -1},
		NoLog:           true,
		NoSigs:          true,
		Debug:           true,
		Trace:           false,
		MaxPayload:      512,
		PingInterval:    1 * time.Second,
		MaxPingsOut:     10,
		WriteDeadline:   2 * time.Second,
	}
	ns, err := natsd.NewServer(opts)
	if err != nil {
		return nil, err
	}
	go ns.Start()

	if ns.ReadyForConnections(100*time.Millisecond) != true {
		return nil, fmt.Errorf("natsd startup failure")
	}
	return ns, nil
}

func testWriter(url string, topic string, count int) (chan struct{}, error) {
	nc, err := nats.Connect(url,
		nats.DontRandomize(),
		nats.NoEcho(),
		nats.Name(topic+"/writer"),
	)
	if err != nil {
		return nil, err
	}

	latch := make(chan struct{})
	go func() {
		<-latch
		for i := 0; i < count; i += 1 {
			nc.Publish(topic, []byte(strconv.Itoa(i)))
			if (i % 3) == 0 {
				nc.FlushTimeout(5 * time.Millisecond)
			}
		}
	}()
	return latch, nil
}

func testReader(url string, topic string, count int, tt *testing.T) (*sync.WaitGroup, error) {
	nc, err := nats.Connect(url,
		nats.DontRandomize(),
		nats.NoEcho(),
		nats.Name(topic+"/reader"),
	)
	if err != nil {
		return nil, err
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func(parent *sync.WaitGroup) {
		defer parent.Done()
		n := 0
		done := make(chan struct{})
		values := make([]string, 0)
		go func(d chan struct{}) {
			nc.Subscribe(topic, func(msg *nats.Msg) {
				//println(msg.Subject, string(msg.Data))
				values = append(values, string(msg.Data))
				if count-1 <= n {
					d <- struct{}{}
				}
				n += 1
			})
		}(done)
		<-done

		for i := 0; i < len(values); i += 1 {
			ia := strconv.Itoa(i)
			if ia != values[i] {
				tt.Errorf("[%d] expect: %s != actual: %s", i, ia, values[i])
			}
		}
	}(wg)
	return wg, nil
}

type testLogWriter struct {
	t *testing.T
}

func (w *testLogWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return 0, nil
}

func TestConcurrentWrite(t *testing.T) {
	primary, err := testStartNatsd(4222)
	if err != nil {
		t.Errorf("%w", err)
	}
	relay, err := testStartNatsd(4223)
	if err != nil {
		t.Errorf("%w", err)
	}

	primaryUrl := fmt.Sprintf("nats://%s", primary.Addr().String())
	relayUrl := fmt.Sprintf("nats://%s", relay.Addr().String())

	concW := 16
	n := 100
	latchs := make([]chan struct{}, 0)
	topics := make([]string, 0)
	keys := []string{"bar", "bbr", "bcr"}
	for i := 0; i < concW; i += 1 {
		for _, key := range keys {
			topic := fmt.Sprintf("topic.foo.%d.%s", i, key)
			latch, err := testWriter(primaryUrl, topic, n)
			if err != nil {
				t.Errorf("%w", err)
			}
			latchs = append(latchs, latch)
			topics = append(topics, topic)
		}
	}
	wgs := make([]*sync.WaitGroup, 0, len(topics))
	for i := 0; i < 1; i += 1 {
		topic := topics[i]
		wg, err := testReader(relayUrl, topic, n, t)
		if err != nil {
			t.Errorf("%w", err)
		}
		wgs = append(wgs, wg)
	}

	l := log.New(&testLogWriter{t}, " nrlay", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	exec := chanque.NewExecutor(100, 1000)

	tps := make([]*topicNameOptionTuple, concW)
	for i := 0; i < concW; i += 1 {
		tps[i] = Topic(fmt.Sprintf("topic.foo.%d.>", i), WorkerNum(10), PrefixSize(15))
	}

	svrConf := NewServerConfig(l, exec)
	rlyConf := RelayConfig{
		PrimaryUrl:   primaryUrl,
		SecondaryUrl: "",
		NatsUrl:      relayUrl,
		Topics:       Topics(tps...),
	}
	svr := NewServer(svrConf, rlyConf,
		nats.NoEcho(),
	)
	svr.Start()
	defer svr.Stop()

	t.Log("start")
	for _, latch := range latchs {
		latch <- struct{}{}
	}
	t.Log("wait")
	for _, wg := range wgs {
		wg.Wait()
	}
}
