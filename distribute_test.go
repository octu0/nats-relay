package nrelay

import (
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/octu0/chanque"
)

var (
	_ (chanque.Worker) = (*testDistributeWorker)(nil)
)

type testDistributeWorker struct {
	counter int32
}

func (d *testDistributeWorker) get() int32 {
	return atomic.LoadInt32(&d.counter)
}

func (d *testDistributeWorker) Enqueue(interface{}) bool {
	atomic.AddInt32(&d.counter, 1)
	return true
}

func (d *testDistributeWorker) CloseEnqueue() bool {
	return false
}

func (d *testDistributeWorker) Shutdown() {}

func (d *testDistributeWorker) ShutdownAndWait() {}

func (d *testDistributeWorker) ForceStop() {}

func TestDistribute(t *testing.T) {
	t.Run("distribute/consistent/one", func(tt *testing.T) {
		workers := []*testDistributeWorker{
			new(testDistributeWorker),
			new(testDistributeWorker),
			new(testDistributeWorker),
		}

		cw := make([]chanque.Worker, len(workers))
		for i, w := range workers {
			cw[i] = w
		}

		dist := newDistribute(cw)
		for i := 0; i < 10; i += 1 {
			dist.Publish("testA", &nats.Msg{})
		}

		switch {
		case workers[0].get() == 10:
			if (workers[1].get() == 0 && workers[2].get() == 0) != true {
				tt.Errorf("concentration of values on workers[0] by consistent key, workers[1] and workers[2] must 0")
			}
		case workers[1].get() == 10:
			if (workers[0].get() == 0 && workers[2].get() == 0) != true {
				tt.Errorf("concentration of values on workers[1] by consistent key, workers[0] and workers[2] must 0")
			}
		case workers[2].get() == 10:
			if (workers[0].get() == 0 && workers[1].get() == 0) != true {
				tt.Errorf("concentration of values on workers[2] by consistent key, workers[0] and workers[1] must 0")
			}
		default:
			tt.Errorf("dist consistent")
		}
	})
	t.Run("distribute/consistent/even", func(tt *testing.T) {
		workers := []*testDistributeWorker{
			new(testDistributeWorker),
			new(testDistributeWorker),
			new(testDistributeWorker),
		}

		cw := make([]chanque.Worker, len(workers))
		for i, w := range workers {
			cw[i] = w
		}

		dist := newDistribute(cw)
		for i := 0; i < 100; i += 1 {
			dist.Publish(strconv.Itoa(i), &nats.Msg{})
		}

		w0 := workers[0].get()
		w1 := workers[1].get()
		w2 := workers[2].get()

		if (w0 + w1 + w2) != 100 {
			tt.Errorf("without omission")
		}
		if w0 < 1 {
			tt.Errorf("must be key assigned: workers[0]")
		}
		if w1 < 1 {
			tt.Errorf("must be key assigned: workers[1]")
		}
		if w2 < 1 {
			tt.Errorf("must be key assigned: workers[2]")
		}
	})
}
