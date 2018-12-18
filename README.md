# nats-relay

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

see more [examples](https://github.com/octu0/nats-relay/tree/master/cmd)

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
   1.1.0

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
