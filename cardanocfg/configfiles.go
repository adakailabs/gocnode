package cardanocfg

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adakailabs/gocnode/downloader"

	"github.com/adakailabs/gocnode/topologyfile"

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
const AlonzoGenesis = "alonzo-genesis.json"
const TopologyJSON = "topology.json"

type Downloader struct {
	log              *zap.SugaredLogger
	conf             *config.C
	node             *config.Node
	relaysStream     chan topologyfile.Node
	relaysStreamDone chan interface{}
	Wg               *sync.WaitGroup
	ConfigJSON       string
	TopologyJSON     string
	ShelleyGenesis   string
	AlonzoGenesis    string
	ByronGenesis     string
	enableRtView     bool
}

func New(c *config.C, n *config.Node, enableRTView bool) (*Downloader, error) {
	var err error
	d := &Downloader{}
	d.Wg = &sync.WaitGroup{}
	d.conf = c
	d.node = n
	d.relaysStream = make(chan topologyfile.Node)
	d.relaysStreamDone = make(chan interface{})
	if d.log, err = l.NewLogConfig(c, "config"); err != nil {
		return d, err
	}
	d.enableRtView = enableRTView
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

func (d *Downloader) RelaysChan() chan topologyfile.Node {
	return d.relaysStream
}

func (d *Downloader) RelaysDone() chan interface{} {
	return d.relaysStreamDone
}

func (d *Downloader) GetURL(aType string) (url string, err error) {
	url = fmt.Sprintf("%s/%s-%s", URI, d.node.Network, aType)
	return url, err
}

func (d *Downloader) DownloadConfigFiles() (configJSON, topology, shelley, byron, alonzo string) {
	filesToGet := []string{
		ConfigJSON,
		TopologyJSON,
		ShelleyGenesis,
		AlonzoGenesis,
		ByronGenesis,
	}

	d.Wg.Add(len(filesToGet))

	for _, f := range filesToGet {
		go d.GetConfigFile(f)
	}

	d.Wg.Wait()

	d.log.Info("config file: ", d.ConfigJSON)
	d.log.Info("topology file: ", d.TopologyJSON)

	return d.ConfigJSON, d.TopologyJSON, d.ShelleyGenesis, d.ByronGenesis, d.AlonzoGenesis
}

func (d *Downloader) GetConfigFile(aType string) {
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
		d.Wg.Done()

	case ByronGenesis:
		if er := d.DownloadGenesis(recent, filePath, aType); err != nil {
			panic(er.Error())
		}
	case ShelleyGenesis:
		if er := d.DownloadGenesis(recent, filePath, aType); err != nil {
			d.log.Errorf(er.Error())
			panic(er.Error())
		}
	case AlonzoGenesis:
		if er := d.DownloadGenesis(recent, filePath, aType); err != nil {
			d.log.Errorf(er.Error())
			panic(er.Error())
		}
	case TopologyJSON:
		if er := d.DownloadAndSetTopologyFile(); er != nil {
			d.log.Errorf(er.Error())
			panic(er.Error())
		}
		d.TopologyJSON = filePath
		d.Wg.Done()

	default:
		panic(fmt.Errorf("bad type"))
	}
}

func (d *Downloader) DownloadGenesis(recent bool, filePath, aType string) (err error) {
	if !recent {
		var url string
		if url, err = d.GetURL(aType); err != nil {
			d.log.Errorf(err.Error())
			err = errors.Annotatef(err, "getting path for: %s", filePath)
			panic(err.Error())
		}

		const retries = 20

		for i := 0; i < retries; i++ {
			if er := downloader.DownloadFile(filePath, url); er != nil {
				if i == retries-1 {
					d.log.Errorf(er.Error())
					er = errors.Annotatef(er, "downloading file: %s", aType)
					return er
				}
				d.log.Errorf("error while downloading %s, %s", filePath, er.Error())
				time.Sleep(time.Second * 2)
				d.log.Warnf("re-attempting to download file %s", filePath)
			} else {
				break
			}
		}
	}

	jq := gojsonq.New().File(filePath)

	d.log.Infof("node %s network magic: %d", d.node.Name, d.node.NetworkMagic)
	if aType == ShelleyGenesis {
		d.node.NetworkMagic = uint64(jq.From("networkMagic").Get().(float64))
		d.ShelleyGenesis = filePath
	}
	if aType == ByronGenesis {
		d.ByronGenesis = filePath
	}
	if aType == AlonzoGenesis {
		// TODO: d.node.NetworkMagic = uint64(jq.From("networkMagic").Get().(float64))
		d.AlonzoGenesis = filePath
	}

	d.Wg.Done()

	return err
}
