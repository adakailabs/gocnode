package cardanocfg_test

import (
	"os"
	"testing"
	"time"

	"github.com/adakailabs/gocnode/fastping"

	"github.com/k0kubun/pp"

	"github.com/adakailabs/gocnode/cardanocfg"

	"github.com/stretchr/testify/assert"

	"github.com/adakailabs/gocnode/config"
)

const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/cardano-docker/cardano-node/gocnode/gocnode.yaml"

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

	d, err2 := cardanocfg.New(&c.Relays[0], c)
	a.Nil(err2)

	d.DownloadConfigFiles()
}

func TestConfigTopology(t *testing.T) {
	a := assert.New(t)
	//const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/cardano-docker/cardano-node/stack/secrets/gocnode.yaml"

	c, err := config.New(cfgFile, true, "Debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(&c.Relays[0], c)
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
		d.GetConfigFile(file)
	}
}

func TestConfigTestnetTopology(t *testing.T) {
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.FailNow()
	}

	d, err2 := cardanocfg.New(&c.Relays[0], c)
	if !a.Nil(err2) {
		t.FailNow()
	}

	relays, err3 := d.TestNetRelays()
	if !a.Nil(err3) {
		t.FailNow()
	}

	pp.Print(relays)
}

func TestConfigTestnetOptimzer(t *testing.T) {
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.FailNow()
	}

	d, err2 := cardanocfg.New(&c.Relays[0], c)
	if !a.Nil(err2) {
		t.FailNow()
	}

	_, relays, err3 := d.GetTestNetRelays()
	if !a.Nil(err3) {
		t.FailNow()
	}

	relays, err4 := d.TestLatencyWithPing(relays)
	if !a.Nil(err4) {
		t.FailNow()
	}

	pp.Print(relays)

	for _, p := range relays {
		pp.Printf("relay: %s --> %s \n", p.Addr, p.Latency.Milliseconds())
	}
}

func TestPinger(t *testing.T) {
	a := assert.New(t)
	pTime, err := fastping.TestAddress("www.google.com")

	if !a.Nil(err) {
		t.FailNow()
	}

	t.Log("time: ", pTime)
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
