# nats-relay

[![Apache License](https://img.shields.io/github/license/octu0/nats-relay)](https://github.com/octu0/nats-relay/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/octu0/nats-relay?status.svg)](https://godoc.org/github.com/octu0/nats-relay)
[![Go Report Card](https://goreportcard.com/badge/github.com/octu0/nats-relay)](https://goreportcard.com/report/github.com/octu0/nats-relay)
[![Releases](https://img.shields.io/github/v/release/octu0/nats-relay)](https://github.com/octu0/nats-relay/releases)

Simple low-latency [NATS](https://nats.io/) relay(replication) server.

Relay(replicate) one or any Topics in NATS server to another NATS server.

![nats-relay usage](https://user-images.githubusercontent.com/42143893/50095373-c3fc9a00-0258-11e9-9174-74775dfe9d5d.png)

## Configuration

Configuration is done using a YAML file:

```yaml
primary:   "nats://primary-natsd.local:4222/"
secondary: "nats://secondary-natsd.local:4222/"
nats:      "nats://localhost:4222/"
topic:
  "foo.>":
    worker: 2
  "bar.*":
    worker: 2
  "baz.1.>":
    worker: 2
  "baz.2.>":
    worker: 2
```

Specifiable wildcard('>' or '*') topicss are available

see more [examples](https://github.com/octu0/nats-relay/tree/master/example)

## Embeding

```go
import (
	"log"
	"time"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/octu0/nats-relay"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	relayConf := nrelay.RelayConfig{
		PrimaryUrl:   "nats://primary-natsd.local:4222",
		SecondaryUrl: "nats://secondary-natsd.local:4222",
		NatsUrl:      "nats://localhost:4222",
		Topics: nrelay.Topics(
			nrelay.Topic("foo.>", nrelay.WorkerNum(2)),
			nrelay.Topic("bar.*", nrelay.WorkerNum(2)),
			nrelay.Topic("baz.1.>", nrelay.WorkerNum(2)),
			nrelay.Topic("baz.2.>", nrelay.WorkerNum(2)),
		),
	}
	executor := chanque.NewExecutor(10, 100)
	logger := log.New(os.Stdout, "nrelay ", log.Ldate|log.Ltime|log.Lshortfile)

	svr := nrelay.NewServer(
		nrelay.ServerOptRelayConfig(relayConfig),
		nrelay.ServerOptExecutor(executor),
		nrelay.ServerOptLogger(logger),
		nrelay.ServerOptNatsOptions(
			nats.PingInterval(500*time.Millisecond),
			nats.ReconnectBufSize(16*1024*1024),
			nats.CustomDialer(...),
		),
	)
	svr.Run(ctx)
	<-ctx.Done()
}
```

## Build

Build requires Go version 1.16+ installed.

```
$ go version
```

Run `make pkg` to Build and package for linux, darwin.

```
$ git clone https://github.com/octu0/nats-relay
$ make pkg
```

## Help

```
NAME:
   nats-relay

USAGE:
   nats-relay [global options] command [command options] [arguments...]

VERSION:
   1.7.0

COMMANDS:
     relay    run relay server
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug, -d    debug mode [$NRELAY_DEBUG]
   --verbose, -V  verbose. more message [$NRELAY_VERBOSE]
   --help, -h     show help
   --version, -v  print the version
```

### subcommand: relay

```
NAME:
   nats-relay relay - run relay server

USAGE:
   nats-relay relay [command options] [arguments...]

OPTIONS:
   --yaml value, -c value  relay configuration yaml file path (default: "./relay.yaml") [$NRELAY_RELAY_YAML]
   --pool-min value        goroutine pool min size (default: 100) [$NRELAY_POOL_MIN]
   --pool-max value        goroutine pool min size (default: 1000) [$NRELAY_POOL_MAX]
```

## License

Apache License 2.0, see LICENSE file for details.
