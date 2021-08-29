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

//const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/cardano-docker/cardano-node/gocnode/gocnode.yaml"

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

	_, err2 := cardanocfg.New(&c.Relays[0], c)
	if !a.Nil(err2) {
		t.FailNow()
	}
}

func TestOptimRedis(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.Error(err.Error())
		t.FailNow()
	}

	opt, err := optimizer.NewOptimizer(c, 0, true)
	if !a.Nil(err) {
		t.FailNow()
	}

	go opt.Run()
	end := make(chan bool)
	go func() {
		err := <-opt.Wait()
		if err != nil {
			t.Error(err.Error())
			t.FailNow()
		}
		close(end)
	}()
	time.Sleep(time.Minute * 10)
	opt.Stop()
	<-end
}

func TestOptimRedisGet(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.Error(err.Error())
		t.FailNow()
	}

	opt, err := optimizer.NewOptimizer(c, 0, true)
	if !a.Nil(err) {
		t.FailNow()
	}

	relays, err := opt.GetRedisRelays()

	if err != nil {
		a.Nil(err)
		t.FailNow()
	}

	if len(relays) < 10 {
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
		t.Log(relay.GetLatency())
	}
}
