package configfiles_test

import (
	"os"
	"testing"

	"github.com/k0kubun/pp"

	"github.com/adakailabs/gocnode/configfiles"

	"github.com/stretchr/testify/assert"

	"github.com/adakailabs/gocnode/config"
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func TestConfig(t *testing.T) {
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "debug")
	a.Nil(err)

	d, err2 := configfiles.New(&c.Relays[2], c)
	a.Nil(err2)

	files := []string{
		configfiles.ConfigJSON,
		configfiles.TopologyJSON,
		configfiles.ByronGenesis,
		configfiles.ShelleyGenesis,
	}

	for _, file := range files {
		configF, er := d.GetConfigFile(file)
		a.Nil(er)

		if _, err := os.Stat(configF); err != nil {
			t.Errorf(err.Error())
			t.FailNow()
		}

		t.Logf("file: %s", configF)
	}
}

func TestConfigTopology(t *testing.T) {
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "debug")
	a.Nil(err)

	d, err2 := configfiles.New(&c.Relays[0], c)
	a.Nil(err2)

	files := []string{
		configfiles.TopologyJSON,
	}

	for _, file := range files {
		configF, er := d.GetConfigFile(file)
		a.Nil(er)

		if _, err := os.Stat(configF); err != nil {
			t.Errorf(er.Error())
			t.FailNow()
		}

		t.Logf("file: %s", configF)
	}
}

func TestConfigTopologyProducer(t *testing.T) {
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "debug")
	a.Nil(err)

	d, err2 := configfiles.New(&c.Producers[0], c)
	a.Nil(err2)

	files := []string{
		configfiles.TopologyJSON,
	}

	for _, file := range files {
		configF, er := d.GetConfigFile(file)
		a.Nil(er)

		if _, err := os.Stat(configF); err != nil {
			t.Errorf(er.Error())
			t.FailNow()
		}

		t.Logf("file: %s", configF)
	}
}

func TestConfigTestnetTopology(t *testing.T) {
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "debug")
	a.Nil(err)

	d, err2 := configfiles.New(&c.Relays[0], c)
	a.Nil(err2)

	relays, err3 := d.TestNetRelays()
	a.Nil(err3)

	pp.Print(relays)
}

func TestConfigMainnetTopology(t *testing.T) {
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "debug")
	a.Nil(err)

	d, err2 := configfiles.New(&c.Relays[0], c)
	a.Nil(err2)

	relays, err3 := d.MainNetRelays()
	a.Nil(err3)

	pp.Print(relays)
}
