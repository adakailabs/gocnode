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

type NodeShort struct {
	Port uint   `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type Node struct {
	Name          string
	Host          string    `mapstructure:"host"`
	IP            string    `mapstructure:"ip"`
	Network       string    `mapstructure:"network"`
	Port          uint      `mapstructure:"port"`
	Era           string    `mapstructure:"era"`
	Peers         uint      `mapstructure:"peers"`
	RtViewPort    uint      `mapstructure:"rtview_port"`
	PromeNExpPort uint      `mapstructure:"prom_node_port"`
	TestMode      bool      `mapstructure:"test_mode"`
	Pool          string    `mapstructure:"pool"`
	Producer      NodeShort `mapstructure:"producer"`
	IsProducer    bool      `mapstructure:"is_producer"`
	RootDir       string    `mapstructure:"root_dir"`
	NetworkMagic  uint64
	TmpDir        string
	Relays        []NodeShort
}

type Mapped struct {
	SecretsPath       string `mapstructure:"secrets_path"`
	Producers         []Node `mapstructure:"producers"`
	Relays            []Node `mapstructure:"relays"`
	RelaysHostsList   map[string][]NodeShort
	ProducerHostsList map[string][]NodeShort
}

type C struct {
	Mapped

	logLevel string
	log      *zap.SugaredLogger
}

func New(configFile string, testmode bool, logLevel string) (c *C, err error) {
	c = &C{}

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

	m.ProducerHostsList = make(map[string][]NodeShort)
	m.RelaysHostsList = make(map[string][]NodeShort)

	c.Mapped = m

	c.configNodes()

	_ = c.log.Sync()

	return c, err
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
		c.Mapped.Producers[i].Name = fmt.Sprintf("producer%02d", i)

		if c.Mapped.Producers[i].Port == 0 {
			c.Mapped.Producers[i].Port = uint(3100 + i)
			c.log.Warnf("for node %s setting node port to: %d", c.Mapped.Producers[i].Name, c.Mapped.Producers[i].Port)
		}
		if c.Mapped.Producers[i].RtViewPort == 0 {
			c.Mapped.Producers[i].RtViewPort = uint(6700 + i)
			c.log.Warnf("for node %s setting node rtview port to: %d", c.Mapped.Relays[i].Name, c.Mapped.Relays[i].RtViewPort)
		}
		if c.Mapped.Producers[i].PromeNExpPort == 0 {
			c.Mapped.Producers[i].PromeNExpPort = uint(9100)
			c.log.Warnf("for node %s setting prometheus node exporter port to: %d", c.Mapped.Relays[i].Name, c.Mapped.Relays[i].PromeNExpPort)
		}

		pool, ok := c.ProducerHostsList[c.Mapped.Producers[i].Pool]
		if !ok {
			pool = make([]NodeShort, 0, 5)
			c.ProducerHostsList[c.Mapped.Producers[i].Pool] = pool
		}
		pool = append(pool, NodeShort{c.Mapped.Producers[i].Port, c.Mapped.Producers[i].Host})
		c.ProducerHostsList[c.Mapped.Producers[i].Pool] = pool
	}
	for i := range c.Mapped.Relays {
		c.Mapped.Relays[i].Name = fmt.Sprintf("relay%02d", i)

		if c.Mapped.Relays[i].Port == 0 {
			c.Mapped.Relays[i].Port = uint(3000 + i)
			c.log.Warnf("for node %s setting node port to: %d", c.Mapped.Relays[i].Name, c.Mapped.Relays[i].Port)
		}
		if c.Mapped.Relays[i].RtViewPort == 0 {
			c.Mapped.Relays[i].RtViewPort = uint(6600 + i)
			c.log.Warnf("for node %s setting node rtview port to: %d", c.Mapped.Relays[i].Name, c.Mapped.Relays[i].RtViewPort)
		}

		if c.Mapped.Relays[i].PromeNExpPort == 0 {
			c.Mapped.Relays[i].PromeNExpPort = uint(9100)
			c.log.Warnf("for node %s setting prometheus node exporter port to: %d", c.Mapped.Relays[i].Name, c.Mapped.Relays[i].PromeNExpPort)
		}

		pool, ok := c.RelaysHostsList[c.Mapped.Relays[i].Pool]
		if !ok {
			pool = make([]NodeShort, 0, 5)
			c.RelaysHostsList[c.Mapped.Relays[i].Pool] = pool
		}

		pool = append(pool, NodeShort{c.Mapped.Relays[i].Port, c.Mapped.Relays[i].Host})
		c.RelaysHostsList[c.Mapped.Relays[i].Pool] = pool
	}

	for i := range c.Mapped.Producers {
		var ok bool
		c.Mapped.Producers[i].Relays, ok = c.RelaysHostsList[c.Mapped.Producers[i].Pool]
		if !ok {
			c.log.Warn("producer %s does not have relays associated", c.Mapped.Producers[i].Name)
		}
	}

	for i := range c.Mapped.Relays {
		producerList, ok := c.ProducerHostsList[c.Mapped.Relays[i].Pool]
		if !ok {
			c.log.Warnf("producer %s does not have producers associated", c.Mapped.Relays[i].Name)
		} else {
			c.Mapped.Relays[i].Producer = producerList[0]
		}
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
