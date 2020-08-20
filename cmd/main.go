package main

import(
  "log"
  "runtime"
  "os"
  "os/signal"
  "syscall"
  "io/ioutil"

  "github.com/comail/colog"
  "gopkg.in/urfave/cli.v1"
  "gopkg.in/yaml.v2"

  "github.com/octu0/nats-relay"
)

var (
  Commands = make([]cli.Command, 0)
)
func AddCommand(cmd cli.Command){
  Commands = append(Commands, cmd)
}

func action(c *cli.Context) error {
  config := nrelay.Config{
    DebugMode:     c.Bool("debug"),
    VerboseMode:   c.Bool("verbose"),
    Procs:         c.Int("procs"),
    LogDir:        c.String("log-dir"),
    RelayYaml:     c.String("relay-yaml"),
  }
  if config.Procs < 1 {
    config.Procs = 1
  }

  if config.DebugMode {
    colog.SetMinLevel(colog.LDebug)
    if config.VerboseMode {
      colog.SetMinLevel(colog.LTrace)
    }
  }

  logger := nrelay.NewGeneralLogger(config)
  colog.SetOutput(logger)

  data, err := ioutil.ReadFile(config.RelayYaml)
  if err != nil {
    log.Printf("error: yaml '%s' read failure: %s", config.RelayYaml, err.Error())
    return err
  }

  relayConfig := nrelay.RelayConfig{}
  if err := yaml.Unmarshal(data, &relayConfig); err != nil {
    log.Printf("error: yaml '%s' unmarshal error: %s", config.RelayYaml, err.Error())
    return err
  }

  svrConfig   := nrelay.NewServerConfig(logger.Logger(), nil)
  relayServer := nrelay.NewServer(svrConfig, relayConfig)
  error_chan  := make(chan error, 0)
  stopService := func() error {
    if err := relayServer.Stop(); err != nil {
      log.Printf("error: %s", err.Error())
      return err
    }

    return nil
  }

  go func(){
    if err := relayServer.Start(); err != nil {
      error_chan <- err
    }
  }()

  signal_chan := make(chan os.Signal, 10)
  signal.Notify(signal_chan, syscall.SIGTERM)
  signal.Notify(signal_chan, syscall.SIGHUP)
  signal.Notify(signal_chan, syscall.SIGQUIT)
  signal.Notify(signal_chan, syscall.SIGINT)
  running := true
  var lastErr error
  for running {
    select {
    case err := <-error_chan:
      log.Printf("error: error has occurred: %s", err.Error())
      lastErr = err
      if e := stopService(); e != nil {
        lastErr = e
      }
      running = false
    case sig := <-signal_chan:
      log.Printf("info: signal trap(%s)", sig.String())
      if err := stopService(); err != nil {
        lastErr = err
      }
      running = false
    }
  }
  if lastErr == nil {
    log.Printf("info: shutdown successful")
    return nil
  }
  return lastErr
}

func main(){
  colog.SetDefaultLevel(colog.LDebug)
  colog.SetMinLevel(colog.LInfo)

  colog.SetFormatter(&colog.StdFormatter{
    Flag: log.Ldate | log.Ltime | log.Lshortfile,
  })
  colog.Register()

  app         := cli.NewApp()
  app.Version  = nrelay.Version
  app.Name     = nrelay.AppName
  app.Author   = ""
  app.Email    = ""
  app.Usage    = ""
  app.Action   = action
  app.Commands = Commands
  app.Flags    = []cli.Flag{
    cli.StringFlag{
      Name: "relay-yaml, c",
      Usage: "relay configuration yaml file path",
      Value: "./relay.yaml",
      EnvVar: "NRELAY_RELAY_YAML",
    },
    cli.StringFlag{
      Name: "log-dir",
      Usage: "/path/to/log directory",
      Value: nrelay.DEFAULT_LOG_DIR,
      EnvVar: "NRELAY_LOG_DIR",
    },
    cli.IntFlag{
      Name: "procs, P",
      Usage: "attach cpu(s)",
      Value: runtime.NumCPU(),
      EnvVar: "NRELAY_PROCS",
    },
    cli.BoolFlag{
      Name: "debug, d",
      Usage: "debug mode",
      EnvVar: "NRELAY_DEBUG",
    },
    cli.BoolFlag{
      Name: "verbose, V",
      Usage: "verbose. more message",
      EnvVar: "NRELAY_VERBOSE",
    },
  }
  if err := app.Run(os.Args); err != nil {
    log.Printf("error: %s", err.Error())
    cli.OsExiter(1)
  }
}
