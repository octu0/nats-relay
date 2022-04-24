package nrelay

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/octu0/chanque"
	"github.com/pkg/errors"
)

const (
	testSourceNoError uint8 = iota
	testSourceErrorOpen
	testSourceErrorClose
	testSourceErrorSubscribe
	testSourceErrorUnsubscribe
	testDestinationNoError
	testDestinationErrorOpen
	testDestinationErrorClose
)

type testMultipleSourceSingleDestinationRelay_SourceWithError struct {
	errType uint8
}

func (s *testMultipleSourceSingleDestinationRelay_SourceWithError) Open() error {
	if s.errType == testSourceErrorOpen {
		return errors.New("errSrcOpen")
	}
	return nil
}

func (s *testMultipleSourceSingleDestinationRelay_SourceWithError) Close() error {
	if s.errType == testSourceErrorClose {
		return errors.New("errSrcClose")
	}
	return nil
}

func (s *testMultipleSourceSingleDestinationRelay_SourceWithError) Subscribe(t string, p int, ws []chanque.Worker) error {
	if s.errType == testSourceErrorSubscribe {
		return errors.New("errSrcSubscribe")
	}
	return nil
}

func (s *testMultipleSourceSingleDestinationRelay_SourceWithError) Unsubscribe() error {
	if s.errType == testSourceErrorUnsubscribe {
		return errors.New("errSrcUnsubscribe")
	}
	return nil
}

type testMultipleSourceSingleDestinationRelay_DestinationWithError struct {
	errType uint8
}

func (d *testMultipleSourceSingleDestinationRelay_DestinationWithError) Open(w int) error {
	if d.errType == testDestinationErrorOpen {
		return errors.New("errDstOpen")
	}
	return nil
}

func (d *testMultipleSourceSingleDestinationRelay_DestinationWithError) Close() error {
	if d.errType == testDestinationErrorClose {
		return errors.New("errDstClose")
	}
	return nil
}

func (d *testMultipleSourceSingleDestinationRelay_DestinationWithError) Workers() []chanque.Worker {
	return nil
}

type testRelayLogWriter struct {
	t *testing.T
}

func (w *testRelayLogWriter) Write(p []byte) (int, error) {
	if testing.Verbose() {
		w.t.Logf("%s", p)
	}
	return len(p), nil
}

func TestMultipleRelayError(t *testing.T) {
	testCases := []struct {
		name      string
		expectErr string
		src       Source
		dst       Destination
	}{
		{
			"src/open",
			"errSrcOpen",
			&testMultipleSourceSingleDestinationRelay_SourceWithError{testSourceErrorOpen},
			&testMultipleSourceSingleDestinationRelay_DestinationWithError{testDestinationNoError},
		},
		{
			"src/close",
			"errSrcClose",
			&testMultipleSourceSingleDestinationRelay_SourceWithError{testSourceErrorClose},
			&testMultipleSourceSingleDestinationRelay_DestinationWithError{testDestinationNoError},
		},
		{
			"src/subscribe",
			"errSrcSubscribe",
			&testMultipleSourceSingleDestinationRelay_SourceWithError{testSourceErrorSubscribe},
			&testMultipleSourceSingleDestinationRelay_DestinationWithError{testDestinationNoError},
		},
		{
			"src/Unsubscribe",
			"errSrcUnsubscribe",
			&testMultipleSourceSingleDestinationRelay_SourceWithError{testSourceErrorUnsubscribe},
			&testMultipleSourceSingleDestinationRelay_DestinationWithError{testDestinationNoError},
		},
		{
			"dst/open",
			"errDstOpen",
			&testMultipleSourceSingleDestinationRelay_SourceWithError{testSourceNoError},
			&testMultipleSourceSingleDestinationRelay_DestinationWithError{testDestinationErrorOpen},
		},
		{
			"dst/close",
			"errDstClose",
			&testMultipleSourceSingleDestinationRelay_SourceWithError{testSourceNoError},
			&testMultipleSourceSingleDestinationRelay_DestinationWithError{testDestinationErrorClose},
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(tt *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // no block

			lg := log.New(&testRelayLogWriter{tt}, c.name+"@", log.LstdFlags)
			r := NewMultipleSourceSingleDestinationRelay("test.topic."+c.name, c.src, c.dst, 0, 0, lg)
			err := r.Run(ctx)
			if err != nil {
				if err.Error() != c.expectErr {
					tt.Errorf("Run Error expect:%s actual:%s", c.expectErr, err.Error())
				}
			} else {
				tt.Errorf("mustError")
			}
		})
	}

	t.Run("noerr/contextCancel", func(tt *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		src := &testMultipleSourceSingleDestinationRelay_SourceWithError{testSourceNoError}
		dst := &testMultipleSourceSingleDestinationRelay_DestinationWithError{testDestinationNoError}
		lg := log.New(&testRelayLogWriter{tt}, tt.Name()+"@", log.LstdFlags)
		go func() {
			<-time.After(10 * time.Millisecond)
			cancel()
		}()
		r := NewMultipleSourceSingleDestinationRelay("test.topic."+tt.Name(), src, dst, 0, 0, lg)
		err := r.Run(ctx)
		if err != nil {
			tt.Errorf("context cancel no error")
		}
	})
}

func TestMultipleRelayRun(t *testing.T) {
	t.Run("blocking", func(tt *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		src := &testMultipleSourceSingleDestinationRelay_SourceWithError{testSourceNoError}
		dst := &testMultipleSourceSingleDestinationRelay_DestinationWithError{testDestinationNoError}
		lg := log.New(&testRelayLogWriter{tt}, tt.Name()+"@", log.LstdFlags)
		go func() {
			<-time.After(10 * time.Millisecond)
			cancel()
		}()
		start := time.Now()
		r := NewMultipleSourceSingleDestinationRelay("test.topic."+tt.Name(), src, dst, 0, 0, lg)
		r.Run(ctx)
		if time.Since(start) < (10 * time.Millisecond) {
			tt.Errorf("Run wait context.Done()")
		}
	})
}
