package optimizer_test

import (
	"os"
	"testing"
	"time"

	"github.com/adakailabs/gocnode/optimizer"

	"github.com/adakailabs/gocnode/cardanocfg"

	"github.com/stretchr/testify/assert"

	"github.com/adakailabs/gocnode/config"
)

const cfgFile = "/home/galuisal/Documents/cardano/cardano-docker/cardano-node/gocnode/gocnode.yaml"

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func TestConfigTestnetTopologyOptim(t *testing.T) {
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.FailNow()
	}

	_, err2 := cardanocfg.New(c, &c.Relays[0], false)
	if !a.Nil(err2) {
		t.FailNow()
	}
}

func TestOptimRedisTestNet(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.Error(err.Error())
		t.FailNow()
	}

	opt, err := optimizer.NewOptimizer(c, 0, true, true)
	if !a.Nil(err) {
		t.FailNow()
	}

	go opt.Run()
	end := make(chan error)
	go func() {
		errEnd := <-opt.Wait()
		if errEnd != nil {
			end <- errEnd
		}
		close(end)
	}()
	time.Sleep(time.Minute * 10)
	opt.Stop()
	if errEnd := <-end; errEnd != nil {
		t.Error(err.Error())
		t.FailNow()
	}
}

func TestOptimRedisMainNet(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.Error(err.Error())
		t.FailNow()
	}

	opt, err := optimizer.NewOptimizer(c, 0, true, false)
	if !a.Nil(err) {
		t.FailNow()
	}

	go opt.Run()
	end := make(chan error)
	go func() {
		errEnd := <-opt.Wait()
		if errEnd != nil {
			end <- errEnd
		}
		close(end)
	}()
	time.Sleep(time.Minute * 10)
	opt.Stop()
	if errEnd := <-end; errEnd != nil {
		t.Error(err.Error())
		t.FailNow()
	}
}

func TestOptimRedisGet(t *testing.T) {
	const relaysNum = 15
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.Error(err.Error())
		t.FailNow()
	}

	opt, err := optimizer.NewOptimizer(c, 0, true, false)
	if !a.Nil(err) {
		t.FailNow()
	}

	relays, err := opt.GetRedisRelays()

	if err != nil {
		a.Nil(err)
		t.Error(err.Error())
		t.FailNow()
	}

	if len(relays) < relaysNum {
		t.Error("too few relays")
		t.FailNow()
	}

	bestRelays, randRelays, err := opt.GetBestAndRandom(5, 5)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	t.Log("best: ", bestRelays)
	t.Log("rand relays", randRelays)

	for _, relay := range bestRelays {
		t.Log(relay.GetLatency())
	}

	t.Log("------------------------------")

	for _, relay := range randRelays {
		t.Log(relay.GetLatency())
	}

	t.Log("------------------------------")

	relays, err = opt.GetRelays(11)
	if err != nil {
		t.FailNow()
	}

	for _, relay := range relays {
		t.Log("XX", relay.GetLatency())
	}
}
