package cardanocfg_test

import (
	"os"
	"testing"
	"time"

	"github.com/k0kubun/pp"

	"github.com/adakailabs/gocnode/cardanocfg"

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

	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(&c.Relays[2], c)
	a.Nil(err2)

	files := []string{
		cardanocfg.ConfigJSON,
		cardanocfg.TopologyJSON,
		cardanocfg.ByronGenesis,
		cardanocfg.ShelleyGenesis,
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
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/cardano-docker/cardano-node/stack/secrets/gocnode.yaml"

	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(&c.Relays[0], c)
	a.Nil(err2)

	files := []string{
		cardanocfg.TopologyJSON,
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

	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(&c.Producers[0], c)
	a.Nil(err2)

	files := []string{
		cardanocfg.TopologyJSON,
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

	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(&c.Relays[0], c)
	a.Nil(err2)

	relays, err3 := d.TestNetRelays()
	a.Nil(err3)

	pp.Print(relays)
}

func TestConfigDownloadAndSetTopology(t *testing.T) {
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(&c.Relays[0], c)
	a.Nil(err2)

	err3 := d.DownloadAndSetTopologyFile()
	a.Nil(err3)

	time.Sleep(time.Second * 30)
}

func TestConfigMainnetTopology(t *testing.T) {
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(&c.Relays[0], c)
	a.Nil(err2)

	relays, err3 := d.MainNetRelays()
	a.Nil(err3)

	pp.Print(relays)
}

func TestConfigMainnetTopologyStream(t *testing.T) {
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(&c.Relays[0], c)
	a.Nil(err2)

	go d.MainNetGetNodes()
	nodeCount := 0
	processNodes := true
	for processNodes {
		select {
		case newRelay := <-d.RelaysChan():
			{
				t.Log("new rel")
				nodeCount++
				pp.Print(newRelay)
			}
		case <-d.RelaysDone():
			{
				t.Log("break")
				processNodes = false
			}
		}
	}

	a.Equal(c.Relays[0].Peers, uint(nodeCount))

	t.Log("end")
}
