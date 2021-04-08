package runner

import (
	"bufio"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"time"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"github.com/juju/errors"
	"go.uber.org/zap"
)

type R struct {
	log                *zap.SugaredLogger
	exporterCmdArgs    []string
	exporterCmdPath    string
	cardanoNodeCmdArgs []string
	cardanoNodeCmdPath string
	cmd                *exec.Cmd
}

func New(conf *config.C) (r *R, err error) {
	r = &R{}
	if r.log, err = l.NewLogConfig(conf, "runner"); err != nil {
		return r, err
	}

	r.cardanoNodeCmdPath = "cardano-node"
	r.cardanoNodeCmdArgs = make([]string, 1, 10)
	r.cardanoNodeCmdArgs[0] = "run"

	r.exporterCmdPath = "nohup"
	r.exporterCmdArgs = make([]string, 1, 10)
	r.exporterCmdArgs[0] = "node_exporter" // node_exporter --web.listen-address=\":${PROMETHEUS_NODE_EXPORT_PORT}\" &"

	return r, err
}

func (r *R) Start() error {
	r.log.Info("starting gocnode")

	return nil
}

func (r *R) Exec(name string) (output string, err error) {
	var ioBufferStdOut io.ReadCloser

	var ioBufferStdErr io.ReadCloser
	r.log.Info("cmd args: ", r.cardanoNodeCmdArgs)
	r.cmd = exec.Command(r.cardanoNodeCmdPath, r.cardanoNodeCmdArgs...)

	// Get a pipe to read from standard out
	ioBufferStdOut, _ = r.cmd.StdoutPipe()
	ioBufferStdErr, _ = r.cmd.StderrPipe()

	r.log.Debugf("running cmd: %s ", r.cmd.String())

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	// Create a scanner which scans r InputC a line-by-line fashion
	scanner := bufio.NewScanner(ioBufferStdOut)

	// Use the scanner to scan the output line by line and log it
	// It's running InputC a goroutine so that it doesn't block
	var sb strings.Builder
	go func() {
		// Read line by line and process it
		for scanner.Scan() {
			newLine := scanner.Text() + "\n"
			sb.WriteString(newLine)
		}
		done <- struct{}{}
	}()

	// Start the command and check for errors
	err = r.startCommand()
	if err != nil {
		err = errors.Annotatef(err, "startCommand failed for %s",
			name)

		return output, err
	}
	// Wait for all output to be processed
	<-done

	stdErr, _ := ioutil.ReadAll(ioBufferStdErr)

	// Wait for the command to finish
	err = r.waitCommand()
	if err != nil {
		err = errors.Annotatef(err, "cardano-cli says: %s: %s",
			name,
			string(stdErr))
		r.log.Error(err.Error())
		return output, err
	}

	return sb.String(), err
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
