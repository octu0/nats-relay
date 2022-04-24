package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/nats-server/v2/server"
)

func main() {
	opts := &server.Options{
		Host:     "127.0.0.1",
		Port:     4222,
		HTTPPort: -1,
		Cluster:  server.ClusterOpts{Port: -1},
		NoLog:    true,
		NoSigs:   true,
		Debug:    true,
		Trace:    true,
	}
	ns, err := server.NewServer(opts)
	defer ns.Shutdown()

	go ns.Start()

	if ns.ReadyForConnections(10*time.Second) != true {
		log.Printf("error: unable to start a NATS Server on %s:%d", opts.Host, opts.Port)
		return
	}

	natsUrl := fmt.Sprintf("nats://%s/", ns.Addr().String())
	log.Printf("info: server url %s", natsUrl)

	nc, err := nats.Connect(
		natsUrl,
		nats.NoEcho(),
		nats.Name("relay-master"),
	)
	if err != nil {
		log.Printf("error: client conn %s", err.Error())
		return
	}
	defer nc.Close()

	nc.Subscribe(">", func(m *nats.Msg) {
		log.Printf("Received %s message: %s", m.Subject, string(m.Data))
	})

	signal_chan := make(chan os.Signal, 10)
	signal.Notify(signal_chan, syscall.SIGTERM)
	signal.Notify(signal_chan, syscall.SIGHUP)
	signal.Notify(signal_chan, syscall.SIGQUIT)
	signal.Notify(signal_chan, syscall.SIGINT)
	for {
		select {
		case sig := <-signal_chan:
			log.Printf("info: signal trap(%s)", sig.String())
			return
		}
	}
}
