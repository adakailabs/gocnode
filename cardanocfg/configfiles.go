package cardanocfg

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/juju/errors"

	"github.com/thedevsaddam/gojsonq"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"go.uber.org/zap"
)

const Testnet = "testnet"
const Mainnet = "mainnet"
const URI = "https://hydra.iohk.io/job/Cardano/cardano-node/cardano-deployment/latest-finished/download/1"
const ConfigJSON = "config.json"
const ByronGenesis = "byron-genesis.json"
const ShelleyGenesis = "shelley-genesis.json"
const TopologyJSON = "topology.json"

type Downloader struct {
	log              *zap.SugaredLogger
	conf             *config.C
	node             *config.Node
	relaysStream     chan Node
	relaysStreamDone chan interface{}
	wg               *sync.WaitGroup
	ConfigJSON       string
	TopologyJSON     string
	ShelleyGenesis   string
	ByronGenesis     string
}

type Topology struct {
	Producers []Node `json:"Producers"`
}

type Node struct {
	Atype   string `json:"type"`
	Addr    string `json:"addr"`
	Port    uint   `json:"port"`
	Valency uint   `json:"valency"`
	Debug   string `json:"debug"`
}

func New(n *config.Node, c *config.C) (*Downloader, error) {
	var err error
	d := &Downloader{}
	d.wg = &sync.WaitGroup{}
	d.conf = c
	d.node = n
	d.relaysStream = make(chan Node)
	d.relaysStreamDone = make(chan interface{})
	if d.log, err = l.NewLogConfig(c, "config"); err != nil {
		return d, err
	}

	return d, nil
}

func (d *Downloader) GetFilePath(aType string, isTmp bool) (filePath string, err error) {
	filePath = fmt.Sprintf("%s/config/%s-%s",
		d.node.TmpDir,
		d.node.Network,
		aType)

	if !isTmp {
		filePath = fmt.Sprintf("%s/config/%s-%s",
			d.node.RootDir,
			d.node.Network,
			aType)
	}

	dir := filepath.Dir(filePath)
	if _, er := os.Stat(dir); er != nil {
		if os.IsNotExist(er) {
			if er := os.MkdirAll(dir, os.ModePerm); er != nil {
				return filePath, er
			}
		}
	}
	return filePath, err
}

func (d *Downloader) RelaysChan() chan Node {
	return d.relaysStream
}

func (d *Downloader) RelaysDone() chan interface{} {
	return d.relaysStreamDone
}

func (d *Downloader) GetURL(aType string) (url string, err error) {
	url = fmt.Sprintf("%s/%s-%s", URI, d.node.Network, aType)
	return url, err
}

func (d *Downloader) DownloadConfigFiles() (configJSON,
	topology,
	shelley,
	byron string) {
	d.wg.Add(4)
	go d.GetConfigFile(ConfigJSON)
	go d.GetConfigFile(TopologyJSON)
	go d.GetConfigFile(ShelleyGenesis)
	go d.GetConfigFile(ByronGenesis)
	d.wg.Wait()

	d.log.Info("config file: ", d.ConfigJSON)
	d.log.Info("topology file: ", d.TopologyJSON)

	return d.ConfigJSON, d.TopologyJSON, d.ShelleyGenesis, d.ByronGenesis
}

func (d *Downloader) GetConfigFile(aType string) {
	var url string
	var err error
	var filePath string
	var recent bool
	if filePath, err = d.GetFilePath(aType, false); err != nil {
		err = errors.Annotatef(err, "getting path for: %s", filePath)
		panic(err.Error())
	}

	statInfo, err := os.Stat(filePath)

	if err == nil {
		fTime := statInfo.ModTime()
		eDuration := time.Since(fTime).Hours()
		d.log.Info("Duration: ", eDuration)
		if eDuration < 24*5 {
			d.log.Info("file is recent: ", filePath)
			// recent = true
		}
	}

	recent = false //FIXME: overriding recent

	switch aType {
	case ConfigJSON:

		filePath, err = d.GetConfigJSON(aType)
		if err != nil {
			err = errors.Annotatef(err, "getting path for: %s", filePath)
			panic(err.Error())
		}
		d.ConfigJSON = filePath
		d.wg.Done()

	case ByronGenesis:
		if !recent {
			if url, err = d.GetURL(aType); err != nil {
				err = errors.Annotatef(err, "getting path for: %s", filePath)
				panic(err.Error())
			}
			err = d.DownloadFile(filePath, url)
			if err != nil {
				err = errors.Annotatef(err, "getting path for: %s", filePath)
				panic(err.Error())
			}
		}
		d.ByronGenesis = filePath
		d.wg.Done()
	case ShelleyGenesis:
		if !recent {
			if url, err = d.GetURL(aType); err != nil {
				err = errors.Annotatef(err, "getting path for: %s", filePath)
				panic(err.Error())
			}
			err = d.DownloadFile(filePath, url)
			if err != nil {
				err = errors.Annotatef(err, "getting path for: %s", filePath)
				panic(err.Error())
			}
		}
		jq := gojsonq.New().File(filePath)

		d.node.NetworkMagic = uint64(jq.From("networkMagic").Get().(float64))
		d.log.Infof("node %s network magic: %d", d.node.Name, d.node.NetworkMagic)
		d.ShelleyGenesis = filePath
		d.wg.Done()
	case TopologyJSON:
		err = d.DownloadAndSetTopologyFile()
		d.TopologyJSON = filePath
		d.wg.Done()
	}
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func (d *Downloader) DownloadFile(filePath, url string) error {
	d.log.Info("downloading from URL: ", url)
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(resp)
		fmt.Println(err.Error())
		return err
	}
	defer resp.Body.Close()

	// Create the file
	d.log.Info("creatind dir: ", filepath.Dir(filePath))
	if er := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); er != nil {
		return er
	}
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	d.log.Info("saved to file: ", filePath)

	return err
}
