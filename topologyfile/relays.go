package topologyfile

import (
	"encoding/json"
	"io/ioutil"

	"github.com/adakailabs/gocnode/config"
	"github.com/adakailabs/gocnode/downloader"
	"github.com/juju/errors"
)

func (tpf *Tpf) GetTestNetRelays(conf *config.C) (tp T, newProduces []Node, err error) {
	tpf.log.Warn("downloading TESTNET relays")
	const URI = "https://explorer.cardano-testnet.iohkdev.io/relays/topology.json"
	const tmpPath = "/tmp/testnet.json"
	if er := downloader.DownloadFile(tmpPath, URI); er != nil {
		er = errors.Annotatef(er, "while attempting to download: %s", tmpPath)
		return tp, newProduces, er
	}

	tp = T{}
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

		for i := range conf.Relays {
			if conf.Relays[i].Host == p.Addr {
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

func (tpf *Tpf) GetMainNetRelays(conf *config.C) (tp T, newProduces []Node, err error) {
	tpf.log.Warn("downloading MAINNET relays")
	const URI = "https://a.adapools.org/topology?geo=us&limit=500"
	const tmpPath = "/tmp/testnet.json"
	if er := downloader.DownloadFile(tmpPath, URI); er != nil {
		er = errors.Annotatef(er, "while attempting to download: %s", tmpPath)
		return tp, newProduces, er
	}

	tp = T{}
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

		for i := range conf.Relays {
			if conf.Relays[i].Host == p.Addr {
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
