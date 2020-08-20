package nrelay

type Config struct {
	DebugMode   bool
	VerboseMode bool
	Procs       int
	LogDir      string
	RelayYaml   string
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
	PrimaryUrl   string                       `yaml:"primary"`
	SecondaryUrl string                       `yaml:"secondary"`
	NatsUrl      string                       `yaml:"nats"`
	Topics       map[string]RelayClientConfig `yaml:"topic"`
}

type RelayClientConfig struct {
	WorkerNum      int  `yaml:"worker"`
	PrefixSize     int  `yaml:"prefix"`
	UseLoadBalance bool `yaml:"loadbalance"`
}

func Topics(topics ...*topicNameOptionTuple) map[string]RelayClientConfig {
	conf := make(map[string]RelayClientConfig)
	for _, topic := range topics {
		conf[topic.name] = topic.option
	}
	return conf
}

type topicNameOptionTuple struct {
	name   string
	option RelayClientConfig
}

func defaultTopicOption() RelayClientConfig {
	return RelayClientConfig{
		WorkerNum:      1,
		PrefixSize:     0,
		UseLoadBalance: false,
	}
}

type topicOptionFunc func(*RelayClientConfig)

func Topic(name string, funcs ...topicOptionFunc) *topicNameOptionTuple {
	tuple := &topicNameOptionTuple{
		name:   name,
		option: defaultTopicOption(),
	}
	for _, fn := range funcs {
		fn(&tuple.option)
	}
	return tuple
}

func WorkerNum(num int) topicOptionFunc {
	return func(opt *RelayClientConfig) {
		opt.WorkerNum = num
	}
}

func PrefixSize(size int) topicOptionFunc {
	return func(opt *RelayClientConfig) {
		opt.PrefixSize = size
	}
}

func UseLoadBalance(use bool) topicOptionFunc {
	return func(opt *RelayClientConfig) {
		opt.UseLoadBalance = use
	}
}
