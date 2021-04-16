package rtview

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/adakailabs/gocnode/configtypes"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"go.uber.org/zap"
)

type Cfg struct {
	conf      *config.C
	log       *zap.SugaredLogger
	rtViewCfg configtypes.RTView
}

func New(c *config.C) (*Cfg, error) {
	var err error
	d := &Cfg{}
	d.conf = c
	if d.log, err = l.NewLogConfig(c, "config"); err != nil {
		return d, err
	}
	d.rtViewCfg = configtypes.NewDefaultRTViewConfig()
	return d, nil
}

func (c *Cfg) GetJSON() []byte {
	const remoteSocket = "RemoteSocket"
	const hostIP = "0.0.0.0"

	for i := range c.conf.Relays {
		tad := configtypes.TraceAcceptAtDescriptor{}
		tad.NodeName = c.conf.Relays[i].Name
		tad.RemoteAddr.Contents = []string{hostIP, fmt.Sprintf("%d", c.conf.Relays[i].RtViewPort)}
		tad.RemoteAddr.Tag = remoteSocket
		c.rtViewCfg.TraceAcceptAt = append(c.rtViewCfg.TraceAcceptAt, tad)
	}
	for i := range c.conf.Producers {
		tad := configtypes.TraceAcceptAtDescriptor{}
		tad.NodeName = c.conf.Producers[i].Name
		tad.RemoteAddr.Contents = []string{hostIP, fmt.Sprintf("%d", c.conf.Producers[i].RtViewPort)}
		tad.RemoteAddr.Tag = remoteSocket
		c.rtViewCfg.TraceAcceptAt = append(c.rtViewCfg.TraceAcceptAt, tad)
	}

	d, err := json.MarshalIndent(&c.rtViewCfg, "", "    ")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	return d
}

func (c *Cfg) CreateConfigFile() (file string, err error) {
	cfgBytes := c.GetJSON()
	file = "/home/lovelace/cardano-node/rt-view/cardano-rt-view.json"
	fileDir := filepath.Dir(file)
	if _, er := os.Stat(fileDir); er != nil {
		if os.IsNotExist(er) {
			if er = os.MkdirAll(fileDir, os.ModePerm); er != nil {
				return file, er
			}
		} else {
			return file, er
		}
	}

	if err := ioutil.WriteFile(file, cfgBytes, os.ModePerm); err != nil {
		return file, err
	}
	return file, err
}
