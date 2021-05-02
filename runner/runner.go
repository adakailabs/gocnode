package runner

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/adakailabs/gocnode/topologyupdater"

	"github.com/adakailabs/gocnode/rtview"

	"github.com/adakailabs/gocnode/prometheuscfg"

	"github.com/k0kubun/pp"

	"github.com/adakailabs/gocnode/cardanocfg"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"go.uber.org/zap"
)

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

type R struct {
	c        *config.C
	nodeC    *config.Node
	nodeID   int
	log      *zap.SugaredLogger
	Cmd1Args []string
	Cmd1Path string
	Cmd0Args []string
	Cmd0Path string
	cmd0     *exec.Cmd
	cmd1     *exec.Cmd
}

func NewCardanoNodeRunner(conf *config.C, nodeID int, isProducer, passive bool) (r *R, err error) {
	r = &R{}
	r.c = conf
	if r.log, err = l.NewLogConfig(conf, "runner"); err != nil {
		return r, err
	}

	r.nodeID = nodeID

	if isProducer {
		r.log.Info("node is a producer")
		r.nodeC = &conf.Producers[nodeID]
		if passive {
			r.nodeC.PassiveMode = passive
		}
		if isProducer {
			r.nodeC.IsProducer = isProducer
		}
	} else {
		r.log.Infof("node is a relay, with ID: %d", nodeID)
		r.nodeC = &conf.Relays[nodeID]
	}

	r.Cmd0Path = "cardano-node"
	r.Cmd0Args = make([]string, 1, 10)
	r.Cmd0Args[0] = "run"

	r.Cmd1Path = "node_exporter"
	r.Cmd1Args = make([]string, 0, 10)

	return r, err
}

func (r *R) StartCnode() error {
	r.log.Info("starting gocnode")

	cnargs := cnodeArgs{}
	cnargs.DatabasePathS = "--database-path"
	cnargs.SocketPathS = "--socket-path"
	cnargs.NodePortS = "--port"
	cnargs.HostAddressS = "--host-addr"
	cnargs.NodeConfigS = "--config"
	cnargs.NodeTopologyS = "--topology"
	cnargs.KesKeyS = "--shelley-kes-key"
	cnargs.VrfKeyS = "--shelley-vrf-key"
	cnargs.OpCertS = "--shelley-operational-certificate"

	cnargs.DatabasePath = fmt.Sprintf("%s/db", r.nodeC.RootDir)

	if _, err := os.Stat(cnargs.DatabasePath); err != nil {
		if os.IsNotExist(err) {
			if er := os.MkdirAll(cnargs.DatabasePath, os.ModePerm); er != nil {
				return er
			}
		} else {
			return err
		}
	}

	cnargs.SocketPath = fmt.Sprintf("%s/node.socket", cnargs.DatabasePath)
	cnargs.NodePort = fmt.Sprintf("%d", r.nodeC.Port)
	cnargs.HostAddress = "0.0.0.0"

	cnargs.KesKey = fmt.Sprintf("%s/node_kes.key", r.c.SecretsPath)
	cnargs.VrfKey = fmt.Sprintf("%s/node_vrf.key", r.c.SecretsPath)
	cnargs.OpCert = fmt.Sprintf("%s/node.cert", r.c.SecretsPath)

	d, err := cardanocfg.New(r.nodeC, r.c)
	if err != nil {
		return err
	}

	cnargs.NodeConfig, err = d.GetConfigFile(cardanocfg.ConfigJSON)
	if err != nil {
		return err
	}

	cnargs.NodeTopology, err = d.GetConfigFile(cardanocfg.TopologyJSON)
	if err != nil {
		return err
	}

	r.Cmd0Args = append(r.Cmd0Args,
		cnargs.DatabasePathS,
		cnargs.DatabasePath,
		cnargs.SocketPathS,
		cnargs.SocketPath,
		cnargs.NodePortS,
		"3001",
		cnargs.HostAddressS,
		cnargs.HostAddress,
		cnargs.NodeTopologyS,
		cnargs.NodeTopology,
		cnargs.NodeConfigS,
		cnargs.NodeConfig,
	)

	if r.nodeC.IsProducer && !r.nodeC.PassiveMode {
		r.Cmd0Args = append(r.Cmd0Args,
			cnargs.KesKeyS,
			cnargs.KesKey,
			cnargs.VrfKeyS,
			cnargs.VrfKey,
			cnargs.OpCertS,
			cnargs.OpCert,
		)
	}

	_, err = d.GetConfigFile(cardanocfg.ShelleyGenesis)
	if err != nil {
		return err
	}

	_, err = d.GetConfigFile(cardanocfg.ByronGenesis)
	if err != nil {
		return err
	}

	r.log.Info(pp.Sprint(r.Cmd0Args))

	// node_exporter --web.listen-address=\":${PROMETHEUS_NODE_EXPORT_PORT}\" &"
	r.Cmd1Args = append(r.Cmd1Args,
		fmt.Sprintf("--web.listen-address=:%d", r.nodeC.PromeNExpPort))

	r.log.Info(pp.Sprint(r.Cmd1Args))

	cmdsErr := make(chan error)

	if !r.nodeC.TestMode {
		go func() {
			if er := r.Exec("node_exporter", r.Cmd1Path, r.Cmd1Args, r.cmd1); er != nil {
				cmdsErr <- er
			}
		}()

		go func() {
			time.Sleep(time.Second * 2)
			if er := r.Exec("cardano-node", r.Cmd0Path, r.Cmd0Args, r.cmd0); er != nil {
				cmdsErr <- er
			}
		}()

		ticker := time.NewTicker(time.Hour)

		go func() {
			if r.nodeC.IsProducer {
				return
			}
			tu, er := topologyupdater.New(r.c, r.nodeID)
			if er != nil {
				cmdsErr <- er
			}
			r.log.Info("ready")
			for range ticker.C {
				code, e := tu.Ping()
				if e != nil {
					r.log.Error(e.Error())
				}
				if code < 300 {
					r.log.Info("code is good:", code)
				} else {
					r.log.Error("return code:", code)
				}
			}
		}()
	}

	err = <-cmdsErr

	return err
}

func NewPrometheusRunner(conf *config.C, nodeID int, isProducer bool) (r *R, err error) {
	r = &R{}
	r.c = conf
	if r.log, err = l.NewLogConfig(conf, "runner"); err != nil {
		return r, err
	}

	r.Cmd0Path = "prometheus"
	r.Cmd0Args = make([]string, 0, 10)
	r.Cmd0Args = append(r.Cmd0Args,
		"--storage.tsdb.path=/prometheus",
		"--web.console.libraries=/usr/share/prometheus/console_libraries",
		"--web.console.templates=/usr/share/prometheus/consoles")

	// prometheus --config.file=$CONFIG_FILE_LOCAL --storage.tsdb.path=/prometheus
	// --web.console.libraries=/usr/share/prometheus/console_libraries --web.console.templates=/usr/share/prometheus/consoles
	return r, err
}

func (r *R) StartPrometheus() error {
	r.log.Info("starting prometheus")

	d, err := prometheuscfg.New(r.c)
	if err != nil {
		return err
	}

	var file string
	if file, err = d.CreateConfigFile(); err != nil {
		return err
	}

	r.Cmd0Args = append(r.Cmd0Args, fmt.Sprintf("--config.file=%s", file))

	r.log.Info(pp.Sprint(r.Cmd0Args))

	cmdsErr := make(chan error)

	go func() {
		if er := r.Exec("prometheus", r.Cmd0Path, r.Cmd0Args, r.cmd0); er != nil {
			cmdsErr <- er
		}
	}()

	err = <-cmdsErr

	return err
}

func NewRtViewRunner(conf *config.C) (r *R, err error) {
	r = &R{}
	r.c = conf
	if r.log, err = l.NewLogConfig(conf, "runner"); err != nil {
		return r, err
	}

	r.Cmd0Path = "/usr/local/rt-view/cardano-rt-view"
	r.Cmd0Args = make([]string, 0, 10)
	r.Cmd0Args = append(r.Cmd0Args,
		"--static",
		"/usr/local/rt-view/static")

	return r, err
}

func (r *R) StartRtView() error {
	r.log.Info("starting rtview")

	d, err := rtview.New(r.c)
	if err != nil {
		return err
	}

	var file string
	if file, err = d.CreateConfigFile(); err != nil {
		return err
	}
	// --port 8666 --config $CONFIG_FILE_LOCAL
	r.Cmd0Args = append(r.Cmd0Args,
		"--port",
		fmt.Sprintf("%d", 8666),
		"--config",
		file)

	r.log.Info(pp.Sprint(r.Cmd0Args))

	cmdsErr := make(chan error)

	go func() {
		if er := r.Exec("rtview", r.Cmd0Path, r.Cmd0Args, r.cmd0); er != nil {
			cmdsErr <- er
		}
	}()

	err = <-cmdsErr

	return err
}
