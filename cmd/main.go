package main

import (
	"log"
	"os"

	"github.com/comail/colog"
	"gopkg.in/urfave/cli.v1"

	"github.com/octu0/nats-relay"
	"github.com/octu0/nats-relay/cli/server"
)

func mergeCommand(values ...[]cli.Command) []cli.Command {
	merged := make([]cli.Command, 0, 0xff)
	for _, commands := range values {
		merged = append(merged, commands...)
	}
	return merged
}

func main() {
	colog.SetDefaultLevel(colog.LDebug)
	colog.SetMinLevel(colog.LInfo)

	colog.SetFormatter(&colog.StdFormatter{
		Flag: log.Ldate | log.Ltime | log.Lshortfile,
	})
	colog.Register()

	app := cli.NewApp()
	app.Version = nrelay.Version
	app.Name = nrelay.AppName
	app.Author = ""
	app.Email = ""
	app.Usage = ""
	app.Commands = mergeCommand(
		server.Command(),
	)
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug, d",
			Usage:  "debug mode",
			EnvVar: "NRELAY_DEBUG",
		},
		cli.BoolFlag{
			Name:   "verbose, V",
			Usage:  "verbose. more message",
			EnvVar: "NRELAY_VERBOSE",
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Printf("error: %+v", err)
		cli.OsExiter(1)
	}
}
