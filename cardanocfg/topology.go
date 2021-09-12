package cardanocfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/k0kubun/pp"

	"github.com/adakailabs/gocnode/optimizer"

	"github.com/adakailabs/gocnode/downloader"

	"github.com/adakailabs/gocnode/topologyfile"
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
			topOthers, err = d.GetRelaysFromRedis(false, false)
		} else {
			topOthers, err = d.GetRelaysFromRedis(true, false)
		}
		d.log.Info(pp.Sprint("topOthers", topOthers))

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
		d.log.Info("filePath:", filePath)
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
	if err != nil {
		return topologyfile.T{}, err
	}

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

func (d *Downloader) GetRelaysFromRedis(isTestnet, testMode bool) (tp topologyfile.T, err error) {
	opt, err := optimizer.NewOptimizer(d.conf, 0, testMode, isTestnet)
	if err != nil {
		return tp, err
	}
	tp.Producers, err = opt.GetRelays(15)
	return tp, err
}
