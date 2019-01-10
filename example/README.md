# example

![output](https://user-images.githubusercontent.com/42143893/50947167-4edf5600-14e0-11e9-8e75-3a29c12be1c7.gif)

## run

```
# pub origin 1
$ go run example/master1.go 

# pub origin 2
$ go run example/master2.go 

# new pub origin
$ go run example/relay-master.go

# relay start
$ go run cmd/main.go --relay-yaml example/relay.yaml
```

## without yaml

import

```
import(
  "context"
  "github.com/octu0/nats-relay"
)

```

configuration and run

```
topics            := make(map[string]nrelay.RelayClientConfig)
topics["foo.>"]    = nrelay.RelayClientConfig{ WorkerNum: 2 }
topics["bar.*"]    = nrelay.RelayClientConfig{ WorkerNum: 2 }
topics["baz.quux"] = nrelay.RelayClientConfig{ WorkerNum: 2 }

conf := nrelay.RelayConfig{
  PrimaryUrl: "nats://192.168.0.1:4201",
  SecondaryUrl: "nats://192.168.0.2:4201",
  NatsUrl: "nats://127.0.0.1:4222",
  Topics: topics,
}

ctx := context.Background()
ctx  = context.WithValue(ctx, "relay.config", conf)

svr := nrelay.NewRelayServer(rctx)
svr.Start(context.TODO())
```
