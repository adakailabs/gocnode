package runner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/k0kubun/pp"

	"github.com/adakailabs/gocnode/configfiles"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"github.com/juju/errors"
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
	c                  *config.C
	nodeC              *config.Node
	nID                int
	log                *zap.SugaredLogger
	exporterCmdArgs    []string
	exporterCmdPath    string
	cardanoNodeCmdArgs []string
	cardanoNodeCmdPath string
	cmd                *exec.Cmd
}

func New(conf *config.C, nodeID int, isProducer bool) (r *R, err error) {
	r = &R{}
	r.c = conf
	if r.log, err = l.NewLogConfig(conf, "runner"); err != nil {
		return r, err
	}

	r.nID = nodeID

	if isProducer {
		r.log.Info("node is a producer")
		r.nodeC = &conf.Producers[nodeID]
	} else {
		r.log.Infof("node is a relay, with ID: %d", nodeID)
		r.nodeC = &conf.Relays[nodeID]
	}

	r.cardanoNodeCmdPath = "cardano-node"
	r.cardanoNodeCmdArgs = make([]string, 1, 10)
	r.cardanoNodeCmdArgs[0] = "run"

	r.exporterCmdPath = "node_exporter"
	r.exporterCmdArgs = make([]string, 0, 10)
	// r.exporterCmdArgs[0] = "node" // node_exporter --web.listen-address=\":${PROMETHEUS_NODE_EXPORT_PORT}\" &"

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

	d, err := configfiles.New(r.nodeC, r.c)
	if err != nil {
		return err
	}

	cnargs.NodeConfig, err = d.GetConfigFile(configfiles.ConfigJSON)
	if err != nil {
		return err
	}

	cnargs.NodeTopology, err = d.GetConfigFile(configfiles.TopologyJSON)
	if err != nil {
		return err
	}

	r.cardanoNodeCmdArgs = append(r.cardanoNodeCmdArgs,
		cnargs.DatabasePathS,
		cnargs.DatabasePath,
		cnargs.SocketPathS,
		cnargs.SocketPath,
		cnargs.NodePortS,
		cnargs.NodePort,
		cnargs.HostAddressS,
		cnargs.HostAddress,
		cnargs.NodeTopologyS,
		cnargs.NodeTopology,
		cnargs.NodeConfigS,
		cnargs.NodeConfig,
	)

	if r.nodeC.IsProducer {
		r.cardanoNodeCmdArgs = append(r.cardanoNodeCmdArgs,
			cnargs.KesKeyS,
			cnargs.KesKey,
			cnargs.VrfKeyS,
			cnargs.VrfKey,
			cnargs.OpCertS,
			cnargs.OpCert,
		)
	}

	_, err = d.GetConfigFile(configfiles.ShelleyGenesis)
	if err != nil {
		return err
	}

	_, err = d.GetConfigFile(configfiles.ByronGenesis)
	if err != nil {
		return err
	}

	r.log.Info(pp.Sprint(r.cardanoNodeCmdArgs))

	// node_exporter --web.listen-address=\":${PROMETHEUS_NODE_EXPORT_PORT}\" &"
	r.exporterCmdArgs = append(r.exporterCmdArgs,
		fmt.Sprintf("--web.listen-address=:%d", r.nodeC.PromeNExpPort))

	r.log.Info(pp.Sprint(r.exporterCmdArgs))

	cmdsErr := make(chan error)

	if !r.nodeC.TestMode {
		/*go func() {
			if err := r.Exec("node_exporter", r.exporterCmdPath, r.exporterCmdArgs); err != nil {
				cmdsErr <- err
			}
		}()
		*/
		go func() {
			if err := r.Exec("cardano-node", r.cardanoNodeCmdPath, r.cardanoNodeCmdArgs); err != nil {
				cmdsErr <- err
			}
		}()
	}

	err = <-cmdsErr

	return err
}

func (r *R) Exec(name string, cmdPath string, cmdArgs []string) (err error) {
	var ioBufferStdOut io.ReadCloser
	var ioBufferStdErr io.ReadCloser

	r.log.Info("cmd args: ", cmdPath, cmdArgs)
	r.cmd = exec.Command(cmdPath, cmdArgs...)

	// Get a pipe to read from standard out
	ioBufferStdOut, _ = r.cmd.StdoutPipe()
	ioBufferStdErr, _ = r.cmd.StderrPipe()

	r.log.Debugf("running cmd: %s ", r.cmd.String())

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	// Create a scannerStdErr which scans r InputC a line-by-line fashion
	scannerStdErr := bufio.NewScanner(ioBufferStdErr)
	scannerStdOut := bufio.NewScanner(ioBufferStdOut)

	// Use the scannerStdErr to scan the output line by line and log it
	// It's running InputC a goroutine so that it doesn't block
	go func() {
		// Read line by line and process it
		for scannerStdErr.Scan() {
			newLine := scannerStdErr.Text() + "\n"
			fmt.Print(newLine)
		}
		done <- struct{}{}
	}()

	go func() {
		// Read line by line and process it
		for scannerStdOut.Scan() {
			newLine := scannerStdOut.Text() + "\n"
			fmt.Print(newLine)
		}
		done <- struct{}{}
	}()

	// StartCnode the command and check for errors
	err = r.startCommand()
	if err != nil {
		err = errors.Annotatef(err, "startCommand failed for %s",
			name)

		return err
	}
	// Wait for all output to be processed
	<-done

	// Wait for the command to finish
	err = r.waitCommand()
	if err != nil {
		err = errors.Annotatef(err, "%s:",
			name)
		r.log.Error(err.Error())
		return err
	}

	return err
}

func (r *R) startCommand() error {
	lcCmd := r.cmd.String()
	err := r.cmd.Start()
	if err != nil {
		err = errors.Annotatef(err, "while attempting to start command %s",
			lcCmd)
		r.log.Error(err.Error())
		time.Sleep(time.Millisecond * 500)
		return err
	}

	return nil
}

func (r *R) waitCommand() error {
	lcCmd := r.cmd.String()
	err := r.cmd.Wait()
	if err != nil {
		err = errors.Annotatef(err, "waitCommand failed for %s",
			lcCmd)
		time.Sleep(time.Millisecond * 500)
		return err
	}
	return nil
}
