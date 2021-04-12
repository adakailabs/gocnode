package configfiles

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

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
		top, err = d.DownloadTopologyJSON()
		if err != nil {
			return err
		}
		topOthers, er := d.MainNetRelays()
		if er != nil {
			return er
		}
		top.Producers = append(top.Producers, topOthers.Producers...)
	} else {
		d.log.Info("node is not producer")
		top = Topology{}
		top.Producers = make([]Producer, len(d.node.Relays))

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
	pp.Print(top)
	return nil
}

func (d *Downloader) DownloadTopologyJSON() (Topology, error) {
	filePathTmpTop, err := d.GetFilePath(TopologyJSON, true)
	if err != nil {
		return Topology{}, err
	}

	url := fmt.Sprintf("%s/%s-%s", URI, Mainnet, TopologyJSON)

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

	newProduces := make([]Producer, 0, len(topOthers.Producers))

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

	topOthers.Producers = newProduces[0:d.node.Peers]

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

	newProduces := make([]Producer, 0, len(topOthers.Producers))

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

	topOthers.Producers = newProduces[0:d.node.Peers]

	return topOthers, nil
}
