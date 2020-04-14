package nrelay

type Config struct {
  DebugMode      bool
  VerboseMode    bool
  Procs          int
  LogDir         string
  RelayYaml      string
}

//
// relay.yaml
// ----------
// primary: "nats://master1.example.com:4222/"
// secondary: "nats://master2.example.com:4222/"
// nats: "nats://localhost:4222/"
// topic:
//   "foo.>":
//     worker: 2
//   "bar.>":
//     worker: 2
//
type RelayConfig struct {
  PrimaryUrl     string                       `yaml:"primary"`
  SecondaryUrl   string                       `yaml:"secondary"`
  NatsUrl        string                       `yaml:"nats"`
  Topics         map[string]RelayClientConfig `yaml:"topic"`
}

type RelayClientConfig struct {
  WorkerNum      int    `yaml:"worker"`
  ShardNum       int    `yaml:"shard"`
  PrefixSize     int    `yaml:"prefix"`
}
