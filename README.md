# nats-relay

[![Apache License](https://img.shields.io/github/license/octu0/nats-relay)](https://github.com/octu0/nats-relay/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/octu0/nats-relay?status.svg)](https://godoc.org/github.com/octu0/nats-relay)
[![Go Report Card](https://goreportcard.com/badge/github.com/octu0/nats-relay)](https://goreportcard.com/report/github.com/octu0/nats-relay)
[![Releases](https://img.shields.io/github/v/release/octu0/nats-relay)](https://github.com/octu0/nats-relay/releases)

Simple low-latency [NATS](https://nats.io/) relay(replication) server.

Relay(replicate) one or any Topics in NATS server to another NATS server.

![nats-relay](https://user-images.githubusercontent.com/42143893/50095373-c3fc9a00-0258-11e9-9174-74775dfe9d5d.png)

## Configuration

Configuration is done using a YAML file:

```
primary: "nats://master1.example.com:4222/"
secondary: "nats://master2.example.com:4222/"
nats: "nats://localhost:4222/"
topic:
  "foo.>":
    worker: 2
  "bar.*":
    worker: 2
  "baz.quux":
    worker: 2
```

Specifiable wildcard('>' or '*') topicss are available

see more [examples](https://github.com/octu0/nats-relay/tree/master/example)

## Customize

### logger

need to implement [log.Logger](https://golang.org/pkg/log/#Logger)

```
type MyLogger struct { ... }
func (l *MyLogger) Fatalf(string, v ...interface{}) { ... }
:
func (l *MyLogger) Printf(string, v ...interface{}) { ... }

ctx  = context.WithValue(ctx, "relay.logger", new(MyLogger))
```

### nats.Option

set []nats.Option to context

```
// []nats.Option
ctx := context.WithValue(ctx, "relay.nats-options", []nats.Option{
  nats.PingInterval(800 * time.Millisecond),
  nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error){
    if err == nats.ErrSlowConsumer {
      pendingMsgs, _, e := sub.Pending()
      if e != nil {
        log.Printf("warn: does not get pending messages: %v", e)
        return
      }
      log.Printf("warn: falling behind with %d pending messages on subject %s", pendingMsgs, sub.Subject)
    } else {
      log.Printf("error: %v", err)
    }
  }),
})
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
   1.5.10

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
