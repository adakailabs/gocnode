package nettest

import (
	"fmt"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/adakailabs/gocnode/nettest/fastping"

	"github.com/adakailabs/go-traceroute/traceroute"
	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"github.com/adakailabs/gocnode/topologyfile"
	"github.com/juju/errors"
	"github.com/k0kubun/pp"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/stat"
)

type Tpf struct {
	log *zap.SugaredLogger
}

func New(c *config.C) (*Tpf, error) {
	var err error
	d := &Tpf{}
	if d.log, err = l.NewLogConfig(c, "nettest"); err != nil {
		err = errors.Annotatef(err, "new nettest, creating log instance")
		return d, err
	}

	return d, nil
}

func (t *Tpf) TestTCPDial(relays []topologyfile.Node) (goods, bads topologyfile.NodeList, err error) {
	goods = make(topologyfile.NodeList, 0, 100)
	bads = make(topologyfile.NodeList, 0, 100)

	for _, relay := range relays {
		servAddr := fmt.Sprintf("%s:%d", relay.Addr, relay.Port)
		tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
		if err != nil {
			t.log.Errorf("eesolveTCPAddr failed: %s", err.Error())
			bads = append(bads, relay)
			continue
		}
		now := time.Now()
		d := net.Dialer{Timeout: time.Second}
		conn, err := d.Dial("tcp", servAddr)
		if err != nil {
			t.log.Errorf("addr: %s", tcpAddr)
			bads = append(bads, relay)
			continue
		}
		relay.SetLatency(time.Since(now) + time.Second*3)
		t.log.Infof("good: %s latency: %d", tcpAddr, relay.GetLatency().Milliseconds())
		goods = append(goods, relay)
		conn.Close()
	}
	return goods, bads, err
}

func (t *Tpf) TestLatencyWithPing(newProduces topologyfile.NodeList) (partialLost,
	allLostPackets, finalProducers topologyfile.NodeList, err error) {
	rand.Shuffle(len(newProduces),
		func(i, j int) {
			newProduces[i],
				newProduces[j] = newProduces[j],
				newProduces[i]
		})

	nodeChan := make(chan topologyfile.Node)
	defer close(nodeChan)
	mu := &sync.Mutex{}
	allLostPackets = make(topologyfile.NodeList, 0)
	partialLost = make(topologyfile.NodeList, 0)

	testNode := func(p topologyfile.Node) {
		var duration time.Duration
		var packetLoss float64

		duration, packetLoss, err = fastping.TestAddress(p.Addr)
		if err != nil {
			if packetLoss == 100 {
				mu.Lock()
				// t.log.Warn("all packets lost for: ", p.Addr)
				allLostPackets = append(allLostPackets, p)
				mu.Unlock()
				err = nil
			} else if packetLoss > 0 {
				// t.log.Warn("some packets losts for: ", p.Addr)
				mu.Lock()
				partialLost = append(partialLost, p)
				mu.Unlock()
				err = nil
			}
		} else {
			p.SetLatency(duration)
			t.log.Infof(" ping-tested:adding IP to list of good nodes: %s", p.Addr)
			nodeChan <- p
		}
	}

	for _, p := range newProduces {
		go testNode(p)
	}

	c := time.NewTimer(time.Second * 60)

	for {
		select {
		case <-c.C:
			if err != nil {
				err = errors.Annotatef(err, "test timeout && error: %s", err.Error())
				t.log.Errorf(err.Error())
				err = nil
			}
			sort.Sort(finalProducers)
			return partialLost, allLostPackets, finalProducers, err

		case p := <-nodeChan:
			finalProducers = append(finalProducers, p)
			if len(finalProducers) == len(newProduces) {
				sort.Sort(finalProducers)
				return partialLost, allLostPackets, finalProducers, err
			}
		}
	}
}

func (t *Tpf) TestLatency(newProduces topologyfile.NodeList) (finalProducers topologyfile.NodeList, err error) {
	rand.Shuffle(len(newProduces),
		func(i, j int) {
			newProduces[i],
				newProduces[j] = newProduces[j],
				newProduces[i]
		})

	nodeChan := make(chan topologyfile.Node)
	done := false

	testNode := func(p topologyfile.Node) {
		p.SetLatency(time.Second)
		var conn net.Conn
		var er error
		// t.log.Info("testing relay: ", p.Addr)
		if conn, er = net.Dial("tcp", fmt.Sprintf("%s:%d", p.Addr, p.Port)); er != nil {
			t.log.Errorf("%s: %s", p.Addr, er.Error())
			return
		}
		conn.Close()

		duration, er := t.latencyBaseadOnRoute(p.Addr)
		if er != nil {
			t.log.Errorf(er.Error())
			return
		}
		p.SetLatency(duration)
		if !done {
			nodeChan <- p
		}
	}

	for _, p := range newProduces {
		go testNode(p)
	}

	c := time.NewTimer(time.Second * 40)

	for {
		select {
		case <-c.C:
			done = true
			t.log.Warn("node tests time count, number of nodes that meet the criteria is: ", len(finalProducers))
			sort.Sort(finalProducers)
			return finalProducers, err

		case p := <-nodeChan:
			finalProducers = append(finalProducers, p)
			if len(finalProducers) == len(newProduces) {
				done = true
				sort.Sort(finalProducers)
				return finalProducers, err
			}
		}
	}
}

func (t *Tpf) latencyBaseadOnRoute(addr string) (time.Duration, error) {
	delay := rand.Intn(15)
	time.Sleep(time.Second * time.Duration(delay))
	ip := net.ParseIP(addr)
	// t.log.Info("routing ip: ", ip, addr)
	if ip == nil {
		hosts, er := net.LookupIP(addr)
		if er != nil {
			t.log.Error(er.Error())
			return time.Second * 2, er
		}
		ip = hosts[0]
	}

	duration := time.Second * 2

	const tries = 3

	for i := 0; i < tries; i++ {
		hops, err := traceroute.Trace(ip)
		if err != nil {
			return duration, err
		}
		if len(hops) > 3 {
			nodes := hops[len(hops)-1].Nodes
			if len(nodes) > 0 {
				list := nodes[len(nodes)-1].RTT
				listFloat := make([]float64, len(list))
				for i, num := range list {
					listFloat[i] = float64(num)
				}
				duration = time.Duration(stat.Mean(listFloat, nil))
				stdDev := stat.StdDev(listFloat, nil)
				if stdDev > 50*float64(time.Millisecond) {
					duration = time.Second * 2
					return duration, err
				}
				if duration < time.Millisecond*50 {
					pp.Println(list)
					pp.Println(hops)
				}
				return duration, err
			}
			t.log.Warnf("route nodes for IP: %v is 0", addr)
		} else {
			time.Sleep(time.Second)
			// t.log.Warnf("hops for IP: %v is 0, try: %d", ip.String(), i)
		}
	}
	t.log.Errorf("hops for IP: %v is 0", addr)

	return duration, nil
}

func (t *Tpf) SetValency(relays topologyfile.NodeList) (topologyfile.NodeList, error) {
	const retry = 10
	for i := range relays {
		addr := net.ParseIP(relays[i].Addr)
		if addr == nil {
			if relays[i].Valency < 2 {
				for j := 0; j < retry; j++ {
					ipList, err := net.LookupIP(relays[i].Addr)
					if err != nil {
						relays[i].Valency = 0
						time.Sleep(time.Millisecond * 200)
					}
					relays[i].Valency = uint(len(ipList))
					break
				}
			}
		}
	}

	newRelays := make(topologyfile.NodeList, 0, len(relays))

	for _, relay := range relays {
		if relay.Valency > 0 {
			newRelays = append(newRelays, relay)
		}
	}

	return newRelays, nil
}
