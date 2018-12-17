# nats-relay

Simple [NATS](https://nats.io/) relay(replication) server.

Relay(replicate) one or any Topics in NATS server to another NATS server.

## Configuration

Configuration is done using a YAML file:

```
primary: "nats://master1.example.com:4222/"
secondary: "nats://master2.example.com:4222/"
nats: "nats://localhost:4222/"
topic:
  "foo.>":
    worker: 2
  "bar.>":
    worker: 2
```

Specifiable wildcard('>' or '*') topicss are available

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
   1.0.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --relay-yaml -c          relay.yaml
   --help, -h               show help
   --version, -v            print the version
```
