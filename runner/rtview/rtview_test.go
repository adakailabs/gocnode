package rtview_test

import (
	"os"
	"testing"

	"github.com/adakailabs/gocnode/runner/rtview"

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

	p, err2 := rtview.New(c)
	a.Nil(err2)

	_, err3 := p.CreateConfigFile()
	a.Nil(err3)
}
