package nrelay

import (
	"context"
	"log"
	"sync/atomic"
	"testing"

	"github.com/octu0/chanque"
	"github.com/pkg/errors"
)

type testServerRunRelays_RelayWithError struct {
	msg string
}

func (r *testServerRunRelays_RelayWithError) Run(context.Context) error {
	return errors.New(r.msg)
}

type testServerRunRelays_RelayWithNoError struct {
	c *int32
}

func (r *testServerRunRelays_RelayWithNoError) Run(context.Context) error {
	atomic.AddInt32(r.c, 1)
	return nil
}

type testServerLogWriter struct {
	t *testing.T
}

func (w *testServerLogWriter) Write(p []byte) (int, error) {
	if testing.Verbose() {
		w.t.Logf("%s", p)
	}
	return len(p), nil
}

func TestServerRunRelays(t *testing.T) {
	t.Run("runErr/noCtxDone", func(tt *testing.T) {
		e := chanque.NewExecutor(10, 10)
		tt.Cleanup(func() { e.Release() })

		lg := log.New(&testServerLogWriter{tt}, tt.Name()+"@", log.LstdFlags)
		relays := []Relay{
			&testServerRunRelays_RelayWithError{"err1"},
			&testServerRunRelays_RelayWithError{"err2"},
			&testServerRunRelays_RelayWithError{"err3"},
		}
		err := runRelays(context.TODO(), e, lg, relays)
		if err != nil {
			switch err.Error() {
			case "err1", "err2", "err3":
				// ok
			default:
				tt.Errorf("expect:['err1' or 'err2' or 'err3'] actual:%s", err.Error())
			}
		} else {
			tt.Errorf("must error")
		}
	})
	t.Run("runErrCombine/noCtxDone", func(tt *testing.T) {
		e := chanque.NewExecutor(10, 10)
		tt.Cleanup(func() { e.Release() })

		counter := int32(0)
		lg := log.New(&testServerLogWriter{tt}, tt.Name()+"@", log.LstdFlags)
		relays := []Relay{
			&testServerRunRelays_RelayWithNoError{&counter},
			&testServerRunRelays_RelayWithError{"err2"},
			&testServerRunRelays_RelayWithNoError{&counter},
		}
		err := runRelays(context.TODO(), e, lg, relays)
		if err != nil {
			if err.Error() != "err2" {
				tt.Errorf("expect:err2 actual:%s", err.Error())
			}
		} else {
			tt.Errorf("must error")
		}

		if atomic.LoadInt32(&counter) != 2 {
			tt.Errorf("Run() called twice: %d", counter)
		}
	})
	t.Run("runNoErr/ctxDone", func(tt *testing.T) {
		e := chanque.NewExecutor(10, 10)
		tt.Cleanup(func() { e.Release() })

		lg := log.New(&testServerLogWriter{tt}, tt.Name()+"@", log.LstdFlags)
		counter := int32(0)
		relays := []Relay{
			&testServerRunRelays_RelayWithNoError{&counter},
			&testServerRunRelays_RelayWithNoError{&counter},
			&testServerRunRelays_RelayWithNoError{&counter},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // already done

		err := runRelays(ctx, e, lg, relays)
		if err != nil {
			tt.Errorf("must no error %v", err)
		}
		if counter != 3 {
			tt.Errorf("Run() called 3 times")
		}
	})
}
