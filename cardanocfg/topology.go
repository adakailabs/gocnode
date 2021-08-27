package cardanocfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/adakailabs/gocnode/downloader"
	"github.com/adakailabs/gocnode/nettest"

	"github.com/adakailabs/gocnode/topologyfile"

	"gonum.org/v1/gonum/stat"

	"github.com/adakailabs/go-traceroute/traceroute"

	"github.com/k0kubun/pp"

	"github.com/prometheus/common/log"
)

const regularRelay = "regular"

func (d *Downloader) DownloadAndSetTopologyFileRelay() (top topologyfile.T, err error) {
	d.log.Info("node is not producer")
	top, err = d.DownloadTopologyJSON(d.node.Network)
	if err != nil {
		return top, err
	}

	actualProducersdd := make([]topologyfile.Node, 0, 4)
	for _, p := range d.node.ExtProducer {
		aP := topologyfile.Node{}
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
		aP := topologyfile.Node{}
		aP.Addr = d.conf.Producers[i].Host
		aP.Port = d.conf.Producers[i].Port
		aP.Atype = regularRelay
		aP.Valency = 1
		actualProducersdd = append(actualProducersdd, aP)
	}
	top.Producers = append(top.Producers, actualProducersdd...)

	return top, err
}

func (d *Downloader) DownloadAndSetTopologyFileProducer() (top topologyfile.T, err error) {
	d.log.Info("node is producer")
	top = topologyfile.T{}
	top.Producers = make([]topologyfile.Node, len(d.node.Relays))

	for i, r := range d.node.Relays {
		top.Producers[i].Port = r.Port
		top.Producers[i].Valency = 1
		top.Producers[i].Addr = r.Host
		top.Producers[i].Atype = regularRelay
	}
	return top, err
}

func (d *Downloader) DownloadAndSetTopologyFile() error {
	var top topologyfile.T
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
		var topOthers topologyfile.T
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

func (d *Downloader) DownloadTopologyJSON(aNet string) (topologyfile.T, error) {
	filePathTmpTop, err := d.GetFilePath(TopologyJSON, true)
	if err != nil {
		return topologyfile.T{}, err
	}

	url := fmt.Sprintf("%s/%s-%s", URI, aNet, TopologyJSON)

	err = downloader.DownloadFile(filePathTmpTop, url)

	top := topologyfile.T{}

	fBytes, err := ioutil.ReadFile(filePathTmpTop)
	if err != nil {
		return topologyfile.T{}, err
	}
	err = json.Unmarshal(fBytes, &top)
	if err != nil {
		return topologyfile.T{}, err
	}
	if er := os.Remove(filePathTmpTop); er != nil {
		return top, err
	}
	return top, nil
}

func (d *Downloader) GetTestNetRelays() (tp topologyfile.T, newProduces []topologyfile.Node, err error) {
	tpf, err := topologyfile.New(d.conf)
	if err != nil {
		return tp, newProduces, err
	}

	tp, newProduces, err = tpf.GetTestNetRelays(d.conf)

	return tp, newProduces, err
}

func (d *Downloader) TestNetRelays() (tp topologyfile.T, err error) {
	var pingRelays topologyfile.NodeList
	// var allLost NodeList
	var netRelays topologyfile.NodeList
	var conRelays topologyfile.NodeList

	relaysMap := make(map[string]bool)

	tp, netRelays, err = d.GetTestNetRelays()
	if err != nil {
		return
	}

	for i := range pingRelays {
		netRelays[i].Valency = 1
	}

	tn, err := nettest.New(d.conf)
	if err != nil {
		return tp, err
	}

	// allLost, pingRelays, err = d.TestLatencyWithPing(netRelays)
	// if err != nil {
	//	err = errors.Annotatef(err, "TestNetRelays:")
	//	return tp, err
	// }

	// for i := range pingRelays {
	// 	pingRelays[i].Valency = 1
	// }

	// for _, p := range pingRelays {
	//	key := fmt.Sprintf("%s:%d", p.Addr, p.Port)
	//	relaysMap[key] = true
	// }

	conRelays, err = tn.TestLatency(netRelays)

	if err != nil {
		return tp, err
	}

	//relays := pingRelays
	relays := make(topologyfile.NodeList, 0)

	for _, r := range conRelays {
		key := fmt.Sprintf("%s:%d", r.Addr, r.Port)
		_, ok := relaysMap[key]
		if !ok {
			d.log.Debugf("adding con relay: %s", r.Addr)
			r.Valency = 1
			relays = append(relays, r)
		}
	}

	relays, err = tn.SetValency(relays)
	if err != nil {
		d.log.Error(err.Error())
		return topologyfile.T{}, err
	}

	if len(relays) > int(d.node.Peers) {
		tp.Producers = relays[0:d.node.Peers]
	} else {
		tp.Producers = relays
	}

	return tp, err
}

func (d *Downloader) MainNetRelays() (topologyfile.T, error) {
	rand.Seed(time.Now().UnixNano()) // FIXME
	const URI = "https://a.adapools.org/topology?geo=us&limit=50"
	const tmpPath = "/tmp/testnet.json"
	if err := downloader.DownloadFile(tmpPath, URI); err != nil {
		return topologyfile.T{}, err
	}

	topOthers := topologyfile.T{}
	fBytesOthers, err := ioutil.ReadFile(tmpPath)
	if err != nil {
		return topologyfile.T{}, err
	}
	err = json.Unmarshal(fBytesOthers, &topOthers)
	if err != nil {
		return topologyfile.T{}, err
	}

	newProduces := make([]topologyfile.Node, 0, len(topOthers.Producers))
	finalProducers := make([]topologyfile.Node, 0, 30)
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

func (d *Downloader) MainnetDownloadNodes() ([]topologyfile.Node, error) {
	rand.Seed(time.Now().UnixNano()) // FIXME
	const URI = "https://a.adapools.org/topology?geo=us&limit=50"
	const tmpPath = "/tmp/testnet.json"
	if err := d.DownloadFile(tmpPath, URI); err != nil {
		return nil, err
	}

	topOthers := topologyfile.T{}
	fBytesOthers, err := ioutil.ReadFile(tmpPath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(fBytesOthers, &topOthers)
	if err != nil {
		return nil, err
	}

	newNodes := make([]topologyfile.Node, 0, len(topOthers.Producers))

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
