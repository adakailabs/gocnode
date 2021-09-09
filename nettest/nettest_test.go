package nettest_test

import (
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	"github.com/adakailabs/gocnode/nettest/fastping"

	"github.com/adakailabs/gocnode/nettest"

	"github.com/adakailabs/gocnode/topologyfile"

	"github.com/adakailabs/go-traceroute/traceroute"

	"github.com/k0kubun/pp"

	"github.com/stretchr/testify/assert"

	"github.com/adakailabs/gocnode/config"
)

const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/cardano-docker/cardano-node/gocnode/gocnode.yaml"

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func TestConfigTestnetOptimzerRoute(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.FailNow()
	}

	tp, err := topologyfile.New(c)
	if !a.Nil(err) {
		t.FailNow()
	}

	_, nodeList, err := tp.GetOnlineRelays(c, true)
	if !a.Nil(err) {
		t.FailNow()
	}

	tn, err := nettest.New(c)
	if !a.Nil(err) {
		t.FailNow()
	}

	relays, err4 := tn.TestLatency(nodeList)
	if !a.Nil(err4) {
		t.FailNow()
	}

	pp.Print(relays)

	for _, p := range relays {
		t.Logf("relay: %v --> %v \n", p.Addr, p.GetLatency().Milliseconds())
	}
}

func TestConfigTestnetOptimzerPing(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.FailNow()
	}

	tp, err := topologyfile.New(c)
	if !a.Nil(err) {
		t.FailNow()
	}

	_, nodeList, err := tp.GetOnlineRelays(c, true)
	if !a.Nil(err) {
		t.FailNow()
	}

	tn, err := nettest.New(c)
	if !a.Nil(err) {
		t.FailNow()
	}

	_, allLost, newNodeList, err := tn.TestLatencyWithPing(nodeList)
	if !a.Nil(err) {
		t.FailNow()
	}

	pp.Println("allLost:", allLost)
	pp.Println("newList:", newNodeList)

	relays, err4 := tn.TestLatency(nodeList)
	if !a.Nil(err4) {
		t.FailNow()
	}

	pp.Print(relays)

	for _, p := range relays {
		t.Logf("relay: %v --> %v \n", p.Addr, p.GetLatency().Milliseconds())
	}
}

func TestConfigTestnetTestTcpDial(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.FailNow()
	}

	tp, err := topologyfile.New(c)
	if !a.Nil(err) {
		t.FailNow()
	}

	_, nodeList, err := tp.GetOnlineRelays(c, true)
	if !a.Nil(err) {
		t.FailNow()
	}

	tn, err := nettest.New(c)
	if !a.Nil(err) {
		t.FailNow()
	}

	goodRelays, badRelays, err := tn.TestTCPDial(nodeList)

	for _, p := range badRelays {
		t.Logf("bad  relay: %v --> %v \n", p.Addr, p.GetLatency().Milliseconds())
	}
	t.Log("###################################################3")
	for _, p := range goodRelays {
		t.Logf("good relay: %v --> %v \n", p.Addr, p.GetLatency().Milliseconds())
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

func TestTraceRouteGoogle(t *testing.T) {
	defer os.RemoveAll("/tmp/logs")
	hosts, _ := net.LookupIP("195.154.69.26")
	ip := hosts[0]
	hops, err := traceroute.Trace(ip)
	if err != nil {
		log.Fatal(err)
	}
	for _, h := range hops {
		for _, n := range h.Nodes {
			log.Printf("%d. %v %v", h.Distance, n.IP, n.RTT)
		}
	}

	list := hops[len(hops)-1].Nodes[len(hops[len(hops)-1].Nodes)-1]

	fmt.Println(list)
}
