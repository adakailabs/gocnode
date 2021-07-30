package cardanocfg_test

import (
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/adakailabs/go-traceroute/traceroute"

	"github.com/adakailabs/gocnode/fastping"
	"github.com/k0kubun/pp"

	"github.com/adakailabs/gocnode/cardanocfg"

	"github.com/stretchr/testify/assert"

	"github.com/adakailabs/gocnode/config"
)

const cfgFile = "/home/galuisal/Documents/cardano/adakailabs/cardano-docker/cardano-node/gocnode/gocnode.yaml"

//const cfgFile = "/home/galuisal/Documents/cardano/cardano-docker/cardano-node/gocnode/gocnode.yaml"

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

	for _, re := range relays.Producers {
		t.Logf("IP: %s --> %dms", re.Addr, re.GetLatency().Milliseconds())
	}
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

	_, relays, err4 := d.TestLatencyWithPing(relays)
	if !a.Nil(err4) {
		t.FailNow()
	}

	pp.Print(relays)

	for _, p := range relays {
		pp.Printf("relay: %s --> %s \n", p.Addr, p.GetLatency().Milliseconds())
	}
}

func TestConfigTestnetOptimzerRoute(t *testing.T) {
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

	relays, err4 := d.TestLatency(relays)
	if !a.Nil(err4) {
		t.FailNow()
	}

	pp.Print(relays)

	for _, p := range relays {
		t.Logf("relay: %v --> %v \n", p.Addr, p.GetLatency().Milliseconds())
	}
}

func TestValency(t *testing.T) {
	a := assert.New(t)

	c, err := config.New(cfgFile, true, "Debug")
	if !a.Nil(err) {
		t.FailNow()
	}

	d, err2 := cardanocfg.New(&c.Relays[0], c)
	if !a.Nil(err2) {
		t.FailNow()
	}

	tAddr := "relays-new.cardano-testnet.iohkdev.io"

	ips, err := net.LookupIP(tAddr)
	if !a.Nil(err) {
		t.FailNow()
	}

	if !a.Equal(8, len(ips)) {
		t.FailNow()
	}

	fmt.Println(ips)

	addr := net.ParseIP("relays-new.cardano-testnet.iohkdev.io")
	if !a.Nil(addr) {
		t.FailNow()
	}

	relays := make(cardanocfg.NodeList, 1)
	relays[0].Addr = tAddr

	for _, addr := range ips {
		addrIP := net.ParseIP(addr.String())
		if !a.NotNil(addrIP) {
			t.FailNow()
		}
	}

	relays, err = d.SetValency(relays)
	if err != nil {
		t.FailNow()
	}

	if !a.Equal(uint(8), relays[0].Valency) {
		t.FailNow()
	}
}

func TestPinger(t *testing.T) {
	a := assert.New(t)
	pTime, _, err := fastping.TestAddress("www.google.com")

	if !a.Nil(err) {
		t.FailNow()
	}

	t.Log("time: ", pTime)
}

func TestConfigDownloadAndSetTopology(t *testing.T) {
	a := assert.New(t)

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

func TestTraceRouteGoogle(t *testing.T) {
	// a := assert.New(tr)
	//hosts, _ := net.LookupIP("roci-master00.adakailabs.com")
	//hosts, _ := net.LookupIP("92.249.148.171")
	// hosts, _ := net.LookupIP("34.127.91.195")
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

// func TestTraceRoute(t *testing.T) {
//	//host := "relays-new.cardano-testnet.iohkdev.io"
//	a := assert.New(t)
//
//	c, err := config.New(cfgFile, true, "Debug")
//	if !a.Nil(err) {
//		t.FailNow()
//	}
//
//	d, err2 := cardanocfg.New(&c.Relays[0], c)
//	if !a.Nil(err2) {
//		t.FailNow()
//	}
//
//	_, relays, err3 := d.GetTestNetRelays()
//	if !a.Nil(err3) {
//		t.FailNow()
//	}
//
//	for _, relay := range relays {
//		host := relay.Addr
//		fmt.Println("testing relay: ", host)
//		hostIP, _ := net.LookupHost(host)
//		fmt.Println("hostIP: ", hostIP)
//		hops, err := traceroute.Trace(net.ParseIP(hostIP[0]))
//		if err != nil {
//			log.Fatal(err)
//		}
//		for _, h := range hops {
//			for _, n := range h.Nodes {
//				log.Printf("%d. %v %v", h.Distance, n.IP, n.RTT)
//			}
//		}
//
//		fmt.Println("hop count: ", len(hops))
//	}
//}

/*
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
*/
