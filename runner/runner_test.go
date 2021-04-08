package runner_test

import (
	"os"
	"testing"

	"github.com/adakailabs/gocnode/config"
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func TestConfig(t *testing.T) {
	const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/gocnode/gocnode.yaml"
	_, _ = config.New(cfgFile, true, "debug")
}
