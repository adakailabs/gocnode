package cardanocfg_test

import (
	"os"
	"testing"
	"time"

	"github.com/adakailabs/gocnode/nettest/fastping"

	"github.com/k0kubun/pp"

	"github.com/adakailabs/gocnode/cardanocfg"

	"github.com/stretchr/testify/assert"

	"github.com/adakailabs/gocnode/config"
)

// const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/cardano-docker/cardano-node/gocnode/gocnode.yaml"

const cfgFile = "/home/galuisal/Documents/cardano/cardano-docker/cardano-node/gocnode/gocnode.yaml"

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func TestConfig(t *testing.T) {
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if err != nil {
		t.Log(err.Error())
	}
	a.Nil(err)
	a.NotNil(c)
	a.NotNil(&c.Relays[0])

	d, err2 := cardanocfg.New(c, &c.Relays[0], false)
	a.Nil(err2)

	d.DownloadConfigFiles()
}

func TestConfigTopology(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(c, &c.Relays[0], false)
	a.Nil(err2)

	files := []string{
		cardanocfg.TopologyJSON,
	}
	d.Wg.Add(len(files))

	for _, file := range files {
		d.GetConfigFile(file)
	}
}

func TestConfigTopologyProducer(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(c, &c.Producers[0], false)
	a.Nil(err2)

	files := []string{
		cardanocfg.TopologyJSON,
	}
	d.Wg.Add(1)
	for _, file := range files {
		d.GetConfigFile(file)
	}
}

func TestConfigTestnetTopology(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.FailNow()
	}

	d, err2 := cardanocfg.New(c, &c.Relays[0], false)
	if !a.Nil(err2) {
		t.FailNow()
	}

	relays, err3 := d.GetRelaysFromRedis(true, true)
	if !a.Nil(err3) {
		t.FailNow()
	}

	pp.Print(relays)

	for _, re := range relays.Producers {
		t.Logf("IP: %s --> %dms", re.Addr, re.GetLatency().Milliseconds())
	}
}

func TestPinger(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)
	pTime, _, err := fastping.TestAddress("www.google.com")

	if !a.Nil(err) {
		t.FailNow()
	}

	t.Log("time: ", pTime)
}

func TestConfigDownloadAndSetTopology(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, false, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(c, &c.Relays[0], false)
	a.Nil(err2)

	err3 := d.DownloadAndSetTopologyFile()
	a.Nil(err3)

	time.Sleep(time.Second * 30)
}

func TestConfigMainnetTopology(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)
	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(c, &c.Relays[0], false)
	a.Nil(err2)

	relays, err3 := d.GetRelaysFromRedis(false, true)
	a.Nil(err3)

	pp.Print(relays)
}
