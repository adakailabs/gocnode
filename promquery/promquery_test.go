package promquery_test

import (
	"os"
	"testing"

	"github.com/adakailabs/gocnode/promquery"

	"github.com/stretchr/testify/assert"

	"github.com/adakailabs/gocnode/config"
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func TestGetBlock(t *testing.T) {
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/cardano-docker/cardano-node/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "debug")
	if !a.Nil(err) {
		t.FailNow()
	}

	tu, err := promquery.New(c, 1)
	if !a.Nil(err) {
		t.FailNow()
	}

	presp, err := tu.GetCardanoBlock()
	if !a.Nil(err) {
		t.FailNow()
	}
	t.Log(presp)
}

func TestGetConnectedPeers(t *testing.T) {
	a := assert.New(t)
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/cardano-docker/cardano-node/gocnode/gocnode.yaml"

	c, err := config.New(cfgFile, true, "debug")
	if !a.Nil(err) {
		t.FailNow()
	}

	tu, err := promquery.New(c, 1)
	if !a.Nil(err) {
		t.FailNow()
	}

	presp, err := tu.GetCardanoConnectedPeers()
	if !a.Nil(err) {
		t.FailNow()
	}
	t.Log(presp)

	_, err2 := tu.CheckConnectedPeers()
	if !a.Nil(err2) {
		t.FailNow()
	}
}
