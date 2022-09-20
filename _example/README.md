# example

![output](https://user-images.githubusercontent.com/42143893/50947167-4edf5600-14e0-11e9-8e75-3a29c12be1c7.gif)

## run with relay yaml

```shell
# pub origin 1
$ go run example/master1.go 

# pub origin 2
$ go run example/master2.go 

# new pub origin
$ go run example/relay-master.go

# relay start
$ go run cmd/main.go --relay-yaml example/relay.yaml
```

## or run programmable

```go
import(
	"context"
	"log"
  
	"github.com/octu0/chanque"
	"github.com/octu0/nats-relay"
)
```

configuration and run

```go
func runRelay(ctx context.Context) {
	conf := nrelay.RelayConfig{
		PrimaryUrl: "nats://192.168.0.1:4201",
		SecondaryUrl: "nats://192.168.0.2:4201",
		NatsUrl: "nats://127.0.0.1:4222",
		Topics: nrelay.Topics(
				nrelay.Topic("foo.>", nrelay.WorkerNum(2)),
				nrelay.Topic("bar.*", nrelay.WorkerNum(2)),
				nrelay.Topic("baz.quux", nrelay.WorkerNum(2)),
			),
	}
	executor := chanque.NewExecutor(10, 100)
	logger := log.New(os.Stdout, "nrelay ", log.Ldate|log.Ltime|log.Lshortfile)

	svr := nrelay.NewDefaultServer(
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
```
