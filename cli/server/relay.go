package server

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/comail/colog"
	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/yaml.v2"

	"github.com/octu0/chanque"
	"github.com/octu0/nats-relay"
)

func relayServerAction(c *cli.Context) error {
	if c.GlobalBool("debug") {
		colog.SetMinLevel(colog.LDebug)
		if c.GlobalBool("verbose") {
			colog.SetMinLevel(colog.LTrace)
		}
	}
	path := c.String("yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return errors.WithStack(err)
	}

	relayConfig := nrelay.RelayConfig{}
	if err := yaml.Unmarshal(data, &relayConfig); err != nil {
		return errors.WithStack(err)
	}

	executor := chanque.NewExecutor(c.Int("pool-min"), c.Int("pool-max"))
	defer executor.Release()

	logger := log.New(os.Stdout, "nrelay ", log.Ldate|log.Ltime|log.Lshortfile)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svr := nrelay.NewDefaultServer(
		nrelay.ServerOptRelayConfig(relayConfig),
		nrelay.ServerOptExecutor(executor),
		nrelay.ServerOptLogger(logger),
	)
	return svr.Run(ctx)
}

func init() {
	addCommand(cli.Command{
		Name:   "relay",
		Usage:  "run relay server",
		Action: relayServerAction,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "yaml, c",
				Usage:  "relay configuration yaml file path",
				Value:  "./relay.yaml",
				EnvVar: "NRELAY_RELAY_YAML",
			},
			cli.IntFlag{
				Name:   "pool-min",
				Usage:  "goroutine pool min size",
				Value:  100,
				EnvVar: "NRELAY_POOL_MIN",
			},
			cli.IntFlag{
				Name:   "pool-max",
				Usage:  "goroutine pool min size",
				Value:  1000,
				EnvVar: "NRELAY_POOL_MAX",
			},
		},
	})
}
