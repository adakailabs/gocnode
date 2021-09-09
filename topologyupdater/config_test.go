package topologyupdater_test

import (
	"os"
	"testing"

	"github.com/adakailabs/gocnode/cardanocfg"

	"github.com/adakailabs/gocnode/topologyupdater"

	"github.com/k0kubun/pp"

	"github.com/stretchr/testify/assert"

	"github.com/adakailabs/gocnode/config"
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func TestConfig(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "debug")
	a.Nil(err)

	tu, err := topologyupdater.New(c, 2)
	a.Nil(err)

	top, err := tu.GetTopology()
	a.Nil(err)

	a.Nil(err)
	pp.Println(top)
}

func TestPing(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"
	nodeID := 2
	c, err := config.New(cfgFile, true, "debug")
	a.Nil(err)

	d, err2 := cardanocfg.New(c, &c.Relays[nodeID], false)
	a.Nil(err2)

	_, _, _, _, _ = d.DownloadConfigFiles()
	a.Nil(err)

	tu, err := topologyupdater.New(c, nodeID)
	a.Nil(err)

	code, err := tu.Ping()
	a.Nil(err)

	t.Log("code: ", code)
}

func TestGetBlock(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "debug")
	a.Nil(err)

	tu, err := topologyupdater.New(c, 2)
	a.Nil(err)

	presp, err := tu.GetCardanoBlock()
	a.Nil(err)
	t.Log(presp)
}
