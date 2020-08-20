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
    Port:         4220,
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
    nats.Name("master1"),
  )
  if err != nil {
    log.Printf("error: client conn %s", err.Error())
    return
  }
  defer nc.Close()

  // self publish
  tick1 := time.NewTicker(1000 * time.Millisecond)
  tick2 := time.NewTicker(1234 * time.Millisecond)
  tick3 := time.NewTicker(1321 * time.Millisecond)
  for{
    select{
    case <-tick1.C:
      val := fmt.Sprintf("master1 foo time=%d", time.Now().Unix())
      nc.Publish("foo.master1.t.o.p.i.c", []byte(val))
    case <-tick2.C:
      val := fmt.Sprintf("time=%d", time.Now().Unix())
      key := fmt.Sprintf("master1-%d", time.Now().Unix())
      nc.Publish("bar." + key, []byte(val))
    case <-tick3.C:
      val := fmt.Sprintf("master1 baz time=%d", time.Now().Unix())
      nc.Publish("baz.quux", []byte(val))
    }
  }
}
