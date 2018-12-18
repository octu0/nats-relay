package main

import(
  "log"
  "fmt"
  "time"
  natsd "github.com/nats-io/gnatsd/server"
  "github.com/nats-io/go-nats"
)

func main(){
  opts := &natsd.Options{
    Host:         "127.0.0.1",
    Port:         4221,
    HTTPPort:     -1,
    Cluster:      natsd.ClusterOpts{Port: -1},
    NoLog:        true,
    NoSigs:       true,
    Debug:        true,
    Trace:        true,
  }
  ns, err := natsd.NewServer(opts)
  defer ns.Shutdown()

  go ns.Start()

  if ns.ReadyForConnections(10 * time.Second) != true {
    log.Printf("error: unable to start a NATS Server on %s:%d", opts.Host, opts.Port)
    return
  }

  natsUrl := fmt.Sprintf("nats://%s/", ns.Addr().String())
  log.Printf("info: server url %s", natsUrl)

  nc, err := nats.Connect(
    natsUrl,
    nats.NoEcho(),
    nats.Name("master2"),
  )
  if err != nil {
    log.Printf("error: client conn %s", err.Error())
    return
  }
  defer nc.Close()

  // self publish
  tick := time.NewTicker(1000 * time.Millisecond)
  for{
    select{
    case <-tick.C:
      val := fmt.Sprintf("master2 time=%d", time.Now().Unix())
      nc.Publish("foo.master2.t.o.p.i.c", []byte(val))
      nc.Publish("bar.master2", []byte(val))
      nc.Publish("baz.quux", []byte(val))
    }
  }
}

