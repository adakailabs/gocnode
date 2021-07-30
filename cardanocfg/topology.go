package cardanocfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	"gonum.org/v1/gonum/stat"

	"github.com/adakailabs/go-traceroute/traceroute"

	"github.com/juju/errors"

	"github.com/k0kubun/pp"

	"github.com/adakailabs/gocnode/fastping"

	"github.com/prometheus/common/log"
)

const regularRelay = "regular"

func (d *Downloader) DownloadAndSetTopologyFileRelay() (top Topology, err error) {
	d.log.Info("node is not producer")
	top, err = d.DownloadTopologyJSON(d.node.Network)
	if err != nil {
		return top, err
	}

	actualProducersdd := make([]Node, 0, 4)
	for _, p := range d.node.ExtProducer {
		aP := Node{}
		aP.Addr = p.Host
		aP.Port = p.Port
		aP.Atype = regularRelay
		aP.Valency = 1
		actualProducersdd = append(actualProducersdd, aP)
	}
	for i := range d.conf.Producers {
		if d.node.Pool != d.conf.Producers[i].Pool {
			continue
		}
		aP := Node{}
		aP.Addr = d.conf.Producers[i].Host
		aP.Port = d.conf.Producers[i].Port
		aP.Atype = regularRelay
		aP.Valency = 1
		actualProducersdd = append(actualProducersdd, aP)
	}
	top.Producers = append(top.Producers, actualProducersdd...)

	return top, err
}

func (d *Downloader) DownloadAndSetTopologyFileProducer() (top Topology, err error) {
	d.log.Info("node is producer")
	top = Topology{}
	top.Producers = make([]Node, len(d.node.Relays))

	for i, r := range d.node.Relays {
		top.Producers[i].Port = r.Port
		top.Producers[i].Valency = 1
		top.Producers[i].Addr = r.Host
		top.Producers[i].Atype = regularRelay
	}
	return top, err
}

func (d *Downloader) DownloadAndSetTopologyFile() error {
	var top Topology
	var err error
	var filePath string

	if filePath, err = d.GetFilePath(TopologyJSON, false); err != nil {
		return err
	}

	if !d.node.IsProducer {
		top, err = d.DownloadAndSetTopologyFileRelay()
		if err != nil {
			return err
		}
	} else {
		top, err = d.DownloadAndSetTopologyFileProducer()
		if err != nil {
			return err
		}
	}

	newBytes, err := json.MarshalIndent(&top, "", "   ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filePath, newBytes, os.ModePerm)
	if err != nil {
		return err
	}

	if !d.node.IsProducer {
		var topOthers Topology
		if d.node.Network == Mainnet {
			topOthers, err = d.MainNetRelays()
		} else {
			topOthers, err = d.TestNetRelays()
			pp.Println("topOthers", topOthers)
		}
		if err != nil {
			d.log.Errorf(err.Error())
		}
		top.Producers = append(top.Producers, topOthers.Producers...)
		newBytes, err := json.MarshalIndent(&top, "", "   ")
		if err != nil {
			d.log.Errorf(err.Error())
		}
		err = ioutil.WriteFile(filePath, newBytes, os.ModePerm)
		if err != nil {
			d.log.Errorf(err.Error())
		}
		log.Info("filePath:", filePath)
	}

	return nil
}

func (d *Downloader) DownloadTopologyJSON(aNet string) (Topology, error) {
	filePathTmpTop, err := d.GetFilePath(TopologyJSON, true)
	if err != nil {
		return Topology{}, err
	}

	url := fmt.Sprintf("%s/%s-%s", URI, aNet, TopologyJSON)

	err = d.DownloadFile(filePathTmpTop, url)
	if err != nil {
		return Topology{}, err
	}

	top := Topology{}

	fBytes, err := ioutil.ReadFile(filePathTmpTop)
	if err != nil {
		return Topology{}, err
	}
	err = json.Unmarshal(fBytes, &top)
	if err != nil {
		return Topology{}, err
	}
	if er := os.Remove(filePathTmpTop); er != nil {
		return top, err
	}
	return top, nil
}

func (d *Downloader) TestLatency(newProduces NodeList) (finalProducers NodeList, err error) {
	rand.Shuffle(len(newProduces),
		func(i, j int) {
			newProduces[i],
				newProduces[j] = newProduces[j],
				newProduces[i]
		})

	nodeChan := make(chan Node)
	defer close(nodeChan)

	testNode := func(p Node) {
		p.SetLatency(time.Second)
		var conn net.Conn
		var er error
		d.log.Info("testing relay: ", p.Addr)

		if conn, er = net.Dial("tcp", fmt.Sprintf("%s:%d", p.Addr, p.Port)); er != nil {
			d.log.Errorf("%s: %s", p.Addr, er.Error())
			return
		} else {
			conn.Close()
			duration, er := d.latencyBaseadOnRoute(p.Addr)
			d.log.Infof("IP: %s --> %dms", p.Addr, duration.Milliseconds())
			if er != nil {
				d.log.Errorf(er.Error())
				return
			}
			p.SetLatency(duration)
			nodeChan <- p
			d.log.Debugf("relay %s latency: %v", p.Addr, duration)
		}
	}

	for _, p := range newProduces {
		go testNode(p)
	}

	c := time.NewTimer(time.Second * 40)

	for {
		select {
		case <-c.C:
			d.log.Warn("node tests time count, number of nodes that meet the criteria is: ", len(finalProducers))
			sort.Sort(finalProducers)
			return finalProducers, err

		case p := <-nodeChan:
			finalProducers = append(finalProducers, p)
			if len(finalProducers) == len(newProduces) {
				sort.Sort(finalProducers)
				return finalProducers, err
			}
		}
	}
}

func (d *Downloader) TestLatencyWithPing(newProduces NodeList) (allLostPackets, finalProducers NodeList, err error) {
	rand.Shuffle(len(newProduces),
		func(i, j int) {
			newProduces[i],
				newProduces[j] = newProduces[j],
				newProduces[i]
		})

	nodeChan := make(chan Node)
	defer close(nodeChan)
	mu := &sync.Mutex{}
	allLostPackets = make(NodeList, 0)

	testNode := func(p Node) {
		var duration time.Duration
		var packetLoss float64

		d.log.Info("testing relay: ", p.Addr)
		duration, packetLoss, err = fastping.TestAddress(p.Addr)
		if err != nil {
			if packetLoss == 100 {
				mu.Lock()
				allLostPackets = append(allLostPackets, p)
				mu.Unlock()
				err = nil
				d.log.Debugf("100% packets lost %s", p.Addr)
			} else if packetLoss > 0 {
				d.log.Warnf("droping relay %s due to ping test: %f lost ", p.Addr, packetLoss)
				err = nil
			} else if packetLoss == 0 {
				d.log.Error("ping error: ", err.Error())
			}
		} else {
			p.SetLatency(duration)
			nodeChan <- p
			d.log.Infof("relay %s latency: %v", p.Addr, duration)
		}
	}

	for _, p := range newProduces {
		go testNode(p)
	}

	c := time.NewTimer(time.Second * 60)

	for {
		select {
		case <-c.C:
			d.log.Warn("node tests time count, number of nodes that meet the criteria is: ", len(finalProducers))
			if err != nil {
				err = errors.Annotatef(err, "test timeout && error: ", err.Error())
			}
			sort.Sort(finalProducers)
			return allLostPackets, finalProducers, err

		case p := <-nodeChan:
			finalProducers = append(finalProducers, p)
			if len(finalProducers) == len(newProduces) {
				sort.Sort(finalProducers)
				return allLostPackets, finalProducers, err
			}
		}
	}
}

func (d *Downloader) TestLatencyWithTraceRoute(newProduces NodeList) (allLostPackets, finalProducers NodeList, err error) {
	rand.Shuffle(len(newProduces),
		func(i, j int) {
			newProduces[i],
				newProduces[j] = newProduces[j],
				newProduces[i]
		})

	nodeChan := make(chan Node)
	defer close(nodeChan)
	allLostPackets = make(NodeList, 0)

	testNode := func(p Node) {
		var duration time.Duration
		var packetLoss float64

		d.log.Info("testing relay: ", p.Addr)
		duration, packetLoss, err = fastping.TestAddress(p.Addr)
		if err != nil {
			d.log.Warnf("addresss %s did not pass latency test: %s", p.Addr, err.Error())
			if packetLoss == 100 {
				allLostPackets = append(allLostPackets, p)
			} else if packetLoss > 0 {
				d.log.Warnf("droping relay %s due to packet loss test", p.Addr)
			}
		} else {
			p.SetLatency(duration)
			nodeChan <- p
			d.log.Infof("relay %s latency: %v", p.Addr, duration)
		}
	}

	for _, p := range newProduces {
		go testNode(p)
	}

	c := time.NewTimer(time.Second * 60)

	for {
		select {
		case <-c.C:
			d.log.Warn("node tests time count, number of nodes that meet the criteria is: ", len(finalProducers))
			sort.Sort(finalProducers)
			return allLostPackets, finalProducers, err

		case p := <-nodeChan:
			finalProducers = append(finalProducers, p)
			if len(finalProducers) == len(newProduces) {
				sort.Sort(finalProducers)
				return allLostPackets, finalProducers, err
			}
		}
	}
	// return allLostPackets, finalProducers, fmt.Errorf("unexpected function end")
}

func (d *Downloader) GetTestNetRelays() (tp Topology, newProduces []Node, err error) {
	rand.Seed(time.Now().UnixNano()) // FIXME
	const URI = "https://explorer.cardano-testnet.iohkdev.io/relays/topology.json"
	const tmpPath = "/tmp/testnet.json"
	if er := d.DownloadFile(tmpPath, URI); er != nil {
		er = errors.Annotatef(er, "while attempting to download: %s", tmpPath)
		return tp, newProduces, er
	}

	tp = Topology{}
	fBytesOthers, err := ioutil.ReadFile(tmpPath)
	if err != nil {
		err = errors.Annotatef(err, "while opening %s: ", tmpPath)
		return tp, newProduces, err
	}
	err = json.Unmarshal(fBytesOthers, &tp)
	if err != nil {
		return tp, newProduces, err
	}

	newProduces = make([]Node, 0, len(tp.Producers))

	for _, p := range tp.Producers {
		found := false

		for i := range d.conf.Relays {
			if d.conf.Relays[i].Host == p.Addr {
				found = true
			}
		}

		if found {
			continue
		}

		p.Valency = 1
		newProduces = append(newProduces, p)
	}
	return tp, newProduces, err
}

func (d *Downloader) SetValency(relays NodeList) (NodeList, error) {
	for i := range relays {
		addr := net.ParseIP(relays[i].Addr)
		if addr == nil {
			if relays[i].Valency < 2 {
				ipList, err := net.LookupIP(relays[i].Addr)
				if err != nil {
					return relays, err
				}
				relays[i].Valency = uint(len(ipList))
			}
		}
	}

	return relays, nil
}

func (d *Downloader) TestNetRelays() (tp Topology, err error) {
	var pingRelays NodeList
	var allLost NodeList
	var netRelays NodeList
	var conRelays NodeList

	relaysMap := make(map[string]bool)

	tp, netRelays, err = d.GetTestNetRelays()
	if err != nil {
		return
	}

	allLost, pingRelays, err = d.TestLatencyWithPing(netRelays)
	if err != nil {
		err = errors.Annotatef(err, "TestNetRelays:")
		return tp, err
	}

	for i := range pingRelays {
		pingRelays[i].Valency = 1
	}

	for _, p := range pingRelays {
		key := fmt.Sprintf("%s:%d", p.Addr, p.Port)
		relaysMap[key] = true
	}

	conRelays, err = d.TestLatency(allLost)
	if err != nil {
		return tp, err
	}

	relays := pingRelays

	for _, r := range conRelays {
		key := fmt.Sprintf("%s:%d", r.Addr, r.Port)
		_, ok := relaysMap[key]
		if !ok {
			d.log.Debugf("adding con relay: %s", r.Addr)
			r.Valency = 1
			relays = append(relays, r)
		}
	}

	relays, err = d.SetValency(relays)
	if err != nil {
		d.log.Error(err.Error())
		return Topology{}, err
	}

	if len(relays) > int(d.node.Peers) {
		tp.Producers = relays[0:d.node.Peers]
	} else {
		tp.Producers = relays
	}

	return tp, err
}

func (d *Downloader) MainNetRelays() (Topology, error) {
	rand.Seed(time.Now().UnixNano()) // FIXME
	const URI = "https://a.adapools.org/topology?geo=us&limit=50"
	const tmpPath = "/tmp/testnet.json"
	if err := d.DownloadFile(tmpPath, URI); err != nil {
		return Topology{}, err
	}

	topOthers := Topology{}
	fBytesOthers, err := ioutil.ReadFile(tmpPath)
	if err != nil {
		return Topology{}, err
	}
	err = json.Unmarshal(fBytesOthers, &topOthers)
	if err != nil {
		return Topology{}, err
	}

	newProduces := make([]Node, 0, len(topOthers.Producers))
	finalProducers := make([]Node, 0, 30)
	for _, p := range topOthers.Producers {
		found := false

		for _, i := range d.node.Relays {
			if i.Host == p.Addr {
				found = true
			}
		}

		if found {
			continue
		}
		newProduces = append(newProduces, p)
	}

	rand.Shuffle(len(newProduces),
		func(i, j int) {
			newProduces[i],
				newProduces[j] = newProduces[j],
				newProduces[i]
		})

	producersTmp := newProduces[0 : d.node.Peers*3]
	for _, p := range producersTmp {
		now := time.Now()
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", p.Addr, p.Port))
		if err != nil {
			d.log.Errorf("%s: %s", p.Addr, err.Error())
		} else {
			duration := time.Since(now)
			conn.Close()
			d.log.Infof("relay is good: %s -- %d ms", p.Addr, duration.Milliseconds())
			finalProducers = append(finalProducers, p)
		}
	}
	if len(finalProducers) >= int(d.node.Peers) {
		topOthers.Producers = finalProducers[0:d.node.Peers]
	} else {
		topOthers.Producers = finalProducers
	}
	if len(finalProducers) == 0 {
		panic("no relays found")
	}

	return topOthers, nil
}

func (d *Downloader) MainnetDownloadNodes() ([]Node, error) {
	rand.Seed(time.Now().UnixNano()) // FIXME
	const URI = "https://a.adapools.org/topology?geo=us&limit=50"
	const tmpPath = "/tmp/testnet.json"
	if err := d.DownloadFile(tmpPath, URI); err != nil {
		return nil, err
	}

	topOthers := Topology{}
	fBytesOthers, err := ioutil.ReadFile(tmpPath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(fBytesOthers, &topOthers)
	if err != nil {
		return nil, err
	}

	newNodes := make([]Node, 0, len(topOthers.Producers))

	for _, p := range topOthers.Producers {
		found := false

		for _, i := range d.node.Relays {
			if i.Host == p.Addr {
				found = true
			}
		}

		if found {
			continue
		}
		newNodes = append(newNodes, p)
	}

	rand.Shuffle(len(newNodes),
		func(i, j int) {
			newNodes[i],
				newNodes[j] = newNodes[j],
				newNodes[i]
		})

	return newNodes, nil
}

func (d *Downloader) latencyBaseadOnRoute(addr string) (time.Duration, error) {
	delay := rand.Intn(15)
	time.Sleep(time.Second * time.Duration(delay))
	ip := net.ParseIP(addr)
	d.log.Info("routing ip: ", ip, addr)
	if ip == nil {
		hosts, er := net.LookupIP(addr)
		if er != nil {
			d.log.Error(er.Error())
			return time.Second * 2, er
		}
		ip = hosts[0]
	}

	d.log.Info("tracing ip: ", ip.String())

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
					d.log.Errorf("XXXX: %s %dms stdev: %f", addr, duration.Milliseconds(), stdDev)
					pp.Println(list)
					pp.Println(hops)
				}
				return duration, err
			} else {
				d.log.Warnf("route nodes for IP: %v is 0", addr)
			}
		} else {
			time.Sleep(time.Second)
			d.log.Warnf("hops for IP: %v is 0, try: %d", ip.String(), i)
		}
	}
	d.log.Errorf("hops for IP: %v is 0", addr)

	return duration, nil
}

/*
func (d *Downloader) MainNetGetNodes() error {
	newProduces, err := d.MainnetDownloadNodes()
	if err != nil {
		err = errors.Annotatef(err, "downloading nodes")
		return err
	}

	producersTmp := newProduces[0 : d.node.Peers*3]
	nCount := 0
	for _, p := range producersTmp {
		now := time.Now()
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", p.Addr, p.Port))
		if err != nil {
			d.log.Errorf("%s: %s", p.Addr, err.Error())
			if conn != nil {
				conn.Close()
			}
		} else {
			duration := time.Since(now)
			conn.Close()
			if duration.Milliseconds() < 300 {
				d.log.Infof("relay is good: %s -- %d ms", p.Addr, duration.Milliseconds())
				d.relaysStream <- p
				nCount++
				if nCount >= int(d.node.Peers) {
					d.log.Info("breaking: ", d.node.Peers)
					break
				}
			} else {
				d.log.Warnf("duration too long: %d", duration.Milliseconds())
			}
		}
	}

	if nCount == 0 {
		panic("no relays found")
	}
	d.log.Info("sending done")
	d.relaysStreamDone <- 0
	return nil
}
*/
