package cardanocfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/adakailabs/gocnode/optimizer"

	"github.com/adakailabs/gocnode/downloader"

	"github.com/adakailabs/gocnode/topologyfile"

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

func (d *Downloader) TestNetRelays() (tp topologyfile.T, err error) {
	opt, err := optimizer.NewOptimizer(d.conf, 0, false, true)

	tp.Producers, err = opt.GetRelays(15)

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
