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

Start with `NewServer` and `*ServerConfig` for server configuration.  
Logger interface, specify [log.Logger](https://golang.org/pkg/log/#Logger) and use [chanque](https://github.com/octu0/chanque) to management goroutine,
which will be set automatically by specifying nil

```go
import (
	"github.com/nats-io/nats.go"
	"github.com/octu0/nats-relay"
	"time"
)

func main() {
	serverConf := nrelay.DefaultServerConfig()
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
	relayd := nrelay.NewServer(serverConf, relayConf,
		nats.PingInterval(500*time.Millisecond),
		nats.ReconnectBufSize(16*1024*1024),
		nats.CustomDialer(...),
	)
	relayd.Start()
}
```

## Build

Build requires Go version 1.11+ installed.

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
   1.6.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --relay-yaml value, -c value  relay configuration yaml file path (default: "./relay.yaml") [$NRELAY_RELAY_YAML]
   --log-dir value               /path/to/log directory (default: "/tmp") [$NRELAY_LOG_DIR]
   --procs value, -P value       attach cpu(s) (default: 8) [$NRELAY_PROCS]
   --debug, -d                   debug mode [$NRELAY_DEBUG]
   --verbose, -V                 verbose. more message [$NRELAY_VERBOSE]
   --help, -h                    show help
   --version, -v                 print the version
```

## License

Apache License 2.0, see LICENSE file for details.
