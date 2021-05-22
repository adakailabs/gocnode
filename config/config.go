package config

import (
	"fmt"

	l "github.com/adakailabs/gocnode/logger"
	"github.com/juju/errors"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const PrometheusConfigPath = "/home/lovelace/prometheus/"

var RelaysHostsList map[string][]NodeShort
var ProducerHostsList map[string][]NodeShort

type NodeShort struct {
	Port uint   `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type Node struct {
	Name          string
	Host          string      `mapstructure:"host"`
	LHost         string      `mapstructure:"host"`
	IP            string      `mapstructure:"ip"`
	Network       string      `mapstructure:"network"`
	Port          uint        `mapstructure:"port"`
	Era           string      `mapstructure:"era"`
	Peers         uint        `mapstructure:"peers"`
	RtViewPort    uint        `mapstructure:"rtview_port"`
	PromeNExpPort uint        `mapstructure:"prom_node_port"`
	TestMode      bool        `mapstructure:"test_mode"`
	Pool          string      `mapstructure:"pool"`
	Producers     []NodeShort `mapstructure:"producer"`
	IsProducer    bool        `mapstructure:"is_producer"`
	RootDir       string      `mapstructure:"root_dir"`

	ExtRelays   []NodeShort `mapstructure:"ext_relays"`
	ExtProducer []NodeShort `mapstructure:"ext_producer"`

	NetworkMagic uint64
	TmpDir       string
	Relays       []NodeShort

	PassiveMode bool

	LogMinSeverity    string `mapstructure:"log_min_severity"`
	FilterMinSeverity string `mapstructure:"filter_min_severity"`
}

type Mapped struct {
	TestnetPortBase   uint `mapstructure:"testnet_port_base"`
	TestnetRTPortBase uint `mapstructure:"testnet_rt_port_base"`

	MainnetPortBase   uint `mapstructure:"mainnet_port_base"`
	MainnetRTPortBase uint `mapstructure:"mainnet_rt_port_base"`

	SecretsPath     string `mapstructure:"secrets_path"`
	Producers       []Node `mapstructure:"producers"`
	Relays          []Node `mapstructure:"relays"`
	RelaysHostsList map[string][]NodeShort

	PrometheusConfigPath string
}

type C struct {
	Mapped
	TestMode bool
	logLevel string
	log      *zap.SugaredLogger
}

func New(configFile string, testmode bool, logLevel string) (c *C, err error) {
	c = &C{}
	c.TestMode = testmode
	if c.log, err = l.NewLogConfig(c, "config"); err != nil {
		return c, err
	}

	c.log.Info("starting config")

	if err = c.configViper(configFile); err != nil {
		err = errors.Annotate(err, "configuring viper")
		return nil, err
	}

	c.log.Info("config file: ", configFile)

	m := Mapped{}
	err = viper.Unmarshal(&m)
	if err != nil {
		return nil, err
	}

	if m.TestnetPortBase == 0 {
		m.TestnetPortBase = 5000
	}

	if m.MainnetPortBase == 0 {
		m.MainnetPortBase = 3000
	}

	if m.TestnetRTPortBase == 0 {
		m.TestnetRTPortBase = 7000
	}

	if m.MainnetRTPortBase == 0 {
		m.MainnetRTPortBase = 6000
	}

	ProducerHostsList = make(map[string][]NodeShort)
	RelaysHostsList = make(map[string][]NodeShort)
	m.PrometheusConfigPath = PrometheusConfigPath

	c.Mapped = m

	c.configNodes()

	_ = c.log.Sync()

	return c, err
}

func (c *C) SetLogMinSeverity(logMinSeverity string, id int, isProducer bool) error {
	if isProducer {
		if id > len(c.Mapped.Producers) {
			return fmt.Errorf("incorrect producer id")
		}
		c.Mapped.Producers[id].LogMinSeverity = logMinSeverity
	} else {
		if id > len(c.Mapped.Relays) {
			return fmt.Errorf("incorrect relay id")
		}
		c.Mapped.Relays[id].LogMinSeverity = logMinSeverity
	}
	return nil
}

func (c *C) configViper(configFile string) error {
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with poolName ".ctool" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath("./")
		viper.AddConfigPath("../")
		viper.AddConfigPath("/run/secrets")
		viper.SetConfigName("gocnode")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		c.log.Info("Using config file:", viper.ConfigFileUsed())
	} else {
		err = errors.Annotate(err, "could not read in viper config")
		return err
	}

	return nil
}

func (c *C) LogLevel() string {
	return c.logLevel
}

func (c *C) configNodes() {
	for i := range c.Mapped.Producers {
		rtPortBase := c.Mapped.MainnetRTPortBase + 700
		portBase := c.Mapped.MainnetPortBase + 100
		if c.Mapped.Producers[i].Network == "testnet" {
			rtPortBase = c.Mapped.TestnetRTPortBase + 700
			portBase = c.Mapped.TestnetPortBase + 100
		}

		if c.Mapped.Producers[i].LogMinSeverity == "" {
			c.Mapped.Producers[i].LogMinSeverity = "Info"
		}

		if c.Mapped.Producers[i].FilterMinSeverity == "" {
			c.Mapped.Producers[i].FilterMinSeverity = "Info"
		}

		c.Mapped.Producers[i].Name = fmt.Sprintf("producer%d", i)

		if c.Mapped.Producers[i].Port == 0 {
			c.Mapped.Producers[i].Port = portBase + uint(i)
			c.log.Warnf("for node %s setting node port to: %d", c.Mapped.Producers[i].Name, c.Mapped.Producers[i].Port)
		}

		if c.Mapped.Producers[i].RtViewPort == 0 {
			c.Mapped.Producers[i].RtViewPort = rtPortBase + uint(i)
			c.log.Warnf("for node %d setting node rtview port to: %d", i, c.Mapped.Producers[i].RtViewPort)
		}

		if c.Mapped.Producers[i].PromeNExpPort == 0 {
			c.Mapped.Producers[i].PromeNExpPort = uint(9100)
			c.log.Warnf("for node %s setting prometheus node exporter port to: %d", c.Mapped.Producers[i].Name, c.Mapped.Producers[i].PromeNExpPort)
		}

		pool, ok := ProducerHostsList[c.Mapped.Producers[i].Pool]
		if !ok {
			pool = make([]NodeShort, 0, 5)
			ProducerHostsList[c.Mapped.Producers[i].Pool] = pool
		}
		pool = append(pool, NodeShort{c.Mapped.Producers[i].Port, c.Mapped.Producers[i].Host})
		ProducerHostsList[c.Mapped.Producers[i].Pool] = pool
	}
	for i := range c.Mapped.Relays {
		rtPortBase := c.Mapped.MainnetRTPortBase + 600
		portBase := c.Mapped.MainnetPortBase
		if c.Mapped.Relays[i].Network == "testnet" {
			rtPortBase = c.Mapped.TestnetRTPortBase + 600
			portBase = c.Mapped.TestnetPortBase
		}

		if c.Mapped.Relays[i].LogMinSeverity == "" {
			c.Mapped.Relays[i].LogMinSeverity = "Info"
		}

		c.Mapped.Relays[i].Name = fmt.Sprintf("relay%d", i)

		if c.Mapped.Relays[i].Port == 0 {
			c.Mapped.Relays[i].Port = portBase + uint(i)
			c.log.Warnf("for node %s setting node port to: %d", c.Mapped.Relays[i].Name, c.Mapped.Relays[i].Port)
		}
		if c.Mapped.Relays[i].RtViewPort == 0 {
			c.Mapped.Relays[i].RtViewPort = rtPortBase + uint(i)
			c.log.Warnf("for node %s setting node rtview port to: %d", c.Mapped.Relays[i].Name, c.Mapped.Relays[i].RtViewPort)
		}

		if c.Mapped.Relays[i].PromeNExpPort == 0 {
			c.Mapped.Relays[i].PromeNExpPort = uint(9100)
			c.log.Warnf("for node %s setting prometheus node exporter port to: %d", c.Mapped.Relays[i].Name, c.Mapped.Relays[i].PromeNExpPort)
		}

		pool, ok := RelaysHostsList[c.Mapped.Relays[i].Pool]
		if !ok {
			pool = make([]NodeShort, 0, 5)
			RelaysHostsList[c.Mapped.Relays[i].Pool] = pool
		}

		pool = append(pool, NodeShort{c.Mapped.Relays[i].Port, c.Mapped.Relays[i].Host})
		RelaysHostsList[c.Mapped.Relays[i].Pool] = pool
	}

	for i := range c.Mapped.Producers {
		var ok bool
		c.Mapped.Producers[i].Relays, ok = RelaysHostsList[c.Mapped.Producers[i].Pool]
		if !ok {
			c.log.Warn("producer %s does not have relays associated", c.Mapped.Producers[i].Name)
		}
		c.Mapped.Producers[i].Relays = append(c.Mapped.Producers[i].Relays, c.Mapped.Producers[i].ExtRelays...)
	}

	for i := range c.Mapped.Relays {
		var ok bool
		c.Mapped.Relays[i].Producers, ok = ProducerHostsList[c.Mapped.Relays[i].Pool]
		if !ok {
			c.log.Warnf("producer %s does not have producers associated", c.Mapped.Relays[i].Name)
		}
		c.Mapped.Relays[i].Producers = append(c.Mapped.Relays[i].Producers, c.Mapped.Relays[i].ExtProducer...)
	}

	for i := range c.Mapped.Producers {
		const rootDir = "/home/lovelace/cardano-node"
		if c.Mapped.Producers[i].RootDir == "" {
			c.Mapped.Producers[i].RootDir = fmt.Sprintf("%s/%s/%s/%s", rootDir,
				c.Mapped.Producers[i].Network,
				c.Mapped.Producers[i].Pool,
				c.Mapped.Producers[i].Name)
		}
		if c.Mapped.Producers[i].TmpDir == "" {
			c.Mapped.Producers[i].TmpDir = fmt.Sprintf("%s/%s/%s/%s", "/tmp/cardano-node",
				c.Mapped.Producers[i].Network,
				c.Mapped.Producers[i].Pool,
				c.Mapped.Producers[i].Name)
		}
	}

	for i := range c.Mapped.Relays {
		const rootDir = "/home/lovelace/cardano-node"
		if c.Mapped.Relays[i].RootDir == "" {
			c.Mapped.Relays[i].RootDir = fmt.Sprintf("%s/%s/%s/%s", rootDir,
				c.Mapped.Relays[i].Network,
				c.Mapped.Relays[i].Pool,
				c.Mapped.Relays[i].Name)
		}
		if c.Mapped.Relays[i].TmpDir == "" {
			c.Mapped.Relays[i].TmpDir = fmt.Sprintf("%s/%s/%s/%s", "/tmp/cardano-node",
				c.Mapped.Relays[i].Network,
				c.Mapped.Relays[i].Pool,
				c.Mapped.Relays[i].Name)
		}
	}
}
