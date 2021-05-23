package node

import (
	"fmt"
	"os"
	"time"

	l "github.com/adakailabs/gocnode/logger"

	"github.com/adakailabs/gocnode/cardanocfg"

	"github.com/adakailabs/gocnode/config"
	"github.com/adakailabs/gocnode/runner/gen"
	"github.com/adakailabs/gocnode/topologyupdater"
	"github.com/k0kubun/pp"
)

type R struct {
	gen.R
	cnargs cnodeArgs
}

type cnodeArgs struct {
	DatabasePathS string
	DatabasePath  string

	SocketPathS string
	SocketPath  string

	NodePortS string
	NodePort  string

	HostAddressS string
	HostAddress  string

	NodeConfigS string
	NodeConfig  string

	NodeTopologyS string
	NodeTopology  string

	KesKeyS string
	KesKey  string

	VrfKeyS string
	VrfKey  string

	OpCertS string
	OpCert  string
}

func (r *R) Init(conf *config.C, nodeID int, isProducer, passive bool) (err error) {
	r.C = conf
	if r.Log, err = l.NewLogConfig(conf, "runner"); err != nil {
		return err
	}
	r.P.Log = r.Log
	r.NodeID = nodeID

	if isProducer {
		r.Log.Info("node is a producer")
		r.NodeC = &conf.Producers[nodeID]
		if passive {
			r.NodeC.PassiveMode = passive
		}
		if isProducer {
			r.NodeC.IsProducer = isProducer
		}
	} else {
		r.Log.Infof("node is a relay, with ID: %d", nodeID)
		r.NodeC = &conf.Relays[nodeID]
	}

	return err
}

func (r *R) newArgs() (cnodeArgs, error) {
	r.cnargs = cnodeArgs{}
	r.cnargs.DatabasePathS = "--database-path"
	r.cnargs.SocketPathS = "--socket-path"
	r.cnargs.NodePortS = "--port"
	r.cnargs.HostAddressS = "--host-addr"
	r.cnargs.NodeConfigS = "--config"
	r.cnargs.NodeTopologyS = "--topology"
	r.cnargs.KesKeyS = "--shelley-kes-key"
	r.cnargs.VrfKeyS = "--shelley-vrf-key"
	r.cnargs.OpCertS = "--shelley-operational-certificate"
	r.cnargs.NodePort = fmt.Sprintf("%d", r.NodeC.Port)
	r.cnargs.HostAddress = "0.0.0.0"

	r.cnargs.KesKey = fmt.Sprintf("%s/node_kes.key", r.C.SecretsPath)
	r.cnargs.VrfKey = fmt.Sprintf("%s/node_vrf.key", r.C.SecretsPath)
	r.cnargs.OpCert = fmt.Sprintf("%s/node.cert", r.C.SecretsPath)

	if er := r.setCheckDBPath(); er != nil {
		return r.cnargs, er
	}

	r.Log.Info("db path: ", r.cnargs.DatabasePath)

	d, err := cardanocfg.New(r.NodeC, r.C)
	if err != nil {
		return r.cnargs, err
	}
	r.cnargs.NodeConfig,
		r.cnargs.NodeTopology,
		_,
		_ = d.DownloadConfigFiles()

	return r.cnargs, nil
}

func (r *R) setCheckDBPath() error {
	r.cnargs.DatabasePath = fmt.Sprintf("%s/db", r.NodeC.RootDir)

	if _, err := os.Stat(r.cnargs.DatabasePath); err != nil {
		if os.IsNotExist(err) {
			if er := os.MkdirAll(r.cnargs.DatabasePath, os.ModePerm); er != nil {
				return er
			}
		} else {
			return err
		}
	}

	r.cnargs.SocketPath = fmt.Sprintf("%s/node.socket", r.cnargs.DatabasePath)

	return nil
}

func (r *R) setCMD0Args() {
	r.Cmd0Args = append(r.Cmd0Args,
		r.cnargs.DatabasePathS,
		r.cnargs.DatabasePath,
		r.cnargs.SocketPathS,
		r.cnargs.SocketPath,
		r.cnargs.NodePortS,
		"3001",
		r.cnargs.HostAddressS,
		r.cnargs.HostAddress,
		r.cnargs.NodeTopologyS,
		r.cnargs.NodeTopology,
		r.cnargs.NodeConfigS,
		r.cnargs.NodeConfig,
	)

	if r.NodeC.IsProducer && !r.NodeC.PassiveMode {
		r.Cmd0Args = append(r.Cmd0Args,
			r.cnargs.KesKeyS,
			r.cnargs.KesKey,
			r.cnargs.VrfKeyS,
			r.cnargs.VrfKey,
			r.cnargs.OpCertS,
			r.cnargs.OpCert,
		)
	}

	r.Log.Info(pp.Sprint(r.Cmd0Args))
}

func (r *R) setCMD1Args() {
	r.Cmd1Args = append(r.Cmd1Args,
		fmt.Sprintf("--web.listen-address=:%d", r.NodeC.PromeNExpPort))
	r.Log.Info(pp.Sprint(r.Cmd1Args))
}

func (r *R) runCNode(cer chan error) {
	time.Sleep(time.Second * 2)
	if er := r.P.Exec("cardano-node", r.Cmd0Path, r.Cmd0Args, r.Cmd0); er != nil {
		cer <- er
	}
}

func (r *R) runExporter(cer chan error) {
	if er := r.P.Exec("node_exporter", r.Cmd1Path, r.Cmd1Args, r.Cmd1); er != nil {
		cer <- er
	}
}

func (r *R) runTopologyUpdater(cer chan error) {
	ticker := time.NewTicker(time.Hour)
	if r.NodeC.IsProducer {
		return
	}
	tu, er := topologyupdater.New(r.C, r.NodeID)
	if er != nil {
		cer <- er
	}

	for range ticker.C {
		code, e := tu.Ping()
		if e != nil {
			r.Log.Error(e.Error())
		}
		if code < 300 {
			r.Log.Info("topology updater code is good:", code)
		} else {
			r.Log.Error("topology updater return code:", code)
		}
	}
}

func (r *R) StartCnode() (err error) {
	r.Log.Info("starting gocnode")

	r.cnargs, err = r.newArgs()
	if err != nil {
		return err
	}

	r.setCMD0Args()

	pp.Println(r.cnargs)

	r.setCMD1Args()

	cer := make(chan error)

	if !r.NodeC.TestMode {
		go r.runExporter(cer)
		go r.runCNode(cer)
		go r.runTopologyUpdater(cer)
	}

	err = <-cer

	return err
}

func NewCardanoNodeRunner(conf *config.C, nodeID int, isProducer, passive bool) (r *R, err error) {
	r = &R{}
	err = r.Init(conf, nodeID, isProducer, passive)
	if err != nil {
		return r, err
	}
	r.Cmd0Path = "cardano-node"
	r.Cmd0Args = make([]string, 1, 10)
	r.Cmd0Args[0] = "run"

	r.Cmd1Path = "node_exporter"
	r.Cmd1Args = make([]string, 0, 10)

	return r, err
}
