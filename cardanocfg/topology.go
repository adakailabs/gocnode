package cardanocfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/prometheus/common/log"

	"github.com/juju/errors"

	"github.com/k0kubun/pp"
)

func (d *Downloader) DownloadAndSetTopologyFile() error {
	var top Topology
	var err error
	var filePath string

	if filePath, err = d.GetFilePath(TopologyJSON, false); err != nil {
		return err
	}

	if !d.node.IsProducer {
		d.log.Info("node is not producer")
		top, err = d.DownloadTopologyJSON(d.node.Network)
		if err != nil {
			return err
		}

		actualProducersdd := make([]Node, 0, 4)
		for _, p := range d.node.ExtProducer {
			aP := Node{}
			aP.Addr = p.Host
			aP.Port = p.Port
			aP.Atype = "regular"
			aP.Valency = 1
			actualProducersdd = append(actualProducersdd, aP)
		}
		for _, p := range d.conf.Producers {
			if d.node.Pool != p.Pool {
				continue
			}
			aP := Node{}
			aP.Addr = p.Host
			aP.Port = p.Port
			aP.Atype = "regular"
			aP.Valency = 1
			actualProducersdd = append(actualProducersdd, aP)
		}
		top.Producers = append(top.Producers, actualProducersdd...)
	} else {
		d.log.Info("node is producer")
		top = Topology{}
		top.Producers = make([]Node, len(d.node.Relays))

		for i, r := range d.node.Relays {
			top.Producers[i].Port = r.Port
			top.Producers[i].Valency = 1
			top.Producers[i].Addr = r.Host
			top.Producers[i].Atype = "regular"
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
	log.Warn("first write")
	pp.Print(top)

	go func() {
		if !d.node.IsProducer {
			var topOthers Topology
			if d.node.Network == Mainnet {
				topOthers, err = d.MainNetRelays()
			} else {
				topOthers, err = d.TestNetRelays()
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
			log.Warn("second write")
			pp.Print(top)
		}
	}()

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

func (d *Downloader) TestNetRelays() (Topology, error) {
	rand.Seed(time.Now().UnixNano()) // FIXME
	const URI = "https://explorer.cardano-testnet.iohkdev.io/relays/topology.json"
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

		p.Valency = 1
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
			d.log.Warnf("%s: %s", p.Addr, err.Error())
		} else {
			duration := time.Since(now)
			conn.Close()

			if duration.Milliseconds() > 300 {
				d.log.Warnf("relay is bad: %s -- %d ms", p.Addr, duration.Milliseconds())
			} else {
				d.log.Infof("relay is good: %s -- %d ms", p.Addr, duration.Milliseconds())
				finalProducers = append(finalProducers, p)
			}
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
