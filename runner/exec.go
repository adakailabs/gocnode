package runner

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/juju/errors"
)

func (r *R) Exec(name, cmdPath string, cmdArgs []string, cmd *exec.Cmd) (err error) {
	var ioBufferStdOut io.ReadCloser
	var ioBufferStdErr io.ReadCloser

	r.log.Info("cmd0 args: ", cmdPath, cmdArgs)
	cmd = exec.Command(cmdPath, cmdArgs...)

	// Get a pipe to read from standard out
	ioBufferStdOut, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}
	ioBufferStdErr, err = cmd.StderrPipe()
	if err != nil {
		return err
	}

	r.log.Debugf("running cmd0: %s ", cmd.String())

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	// Create a scannerStdErr which scans r InputC a line-by-line fashion
	scannerStdErr := bufio.NewScanner(ioBufferStdErr)
	scannerStdErrBuf := make([]byte, 0, 5*1024*1024)
	scannerStdErr.Buffer(scannerStdErrBuf, 5*1024*1024)

	scannerStdOut := bufio.NewScanner(ioBufferStdOut)
	scannerStdOutBuf := make([]byte, 0, 5*1024*1024)
	scannerStdOut.Buffer(scannerStdOutBuf, 5*1024*1024)

	// Use the scannerStdErr to scan the output line by line and log it
	// It's running InputC a goroutine so that it doesn't block
	go func() {
		// Read line by line and process it
		for scannerStdErr.Scan() {
			newLine := scannerStdErr.Text() + "\n"
			if strings.Contains(newLine, "Error") {
				r.processError(newLine)
			} else {
				r.processInfo(newLine)
			}
		}
		if scannerStdErr.Err() != nil {
			r.log.Error("STDERR: ", scannerStdErr.Err())
		}
		r.log.Info("stderr processing finished")
		done <- struct{}{}
	}()

	go func() {
		// Read line by line and process it
		for scannerStdOut.Scan() {
			newLine := scannerStdOut.Text() + "\n"
			if strings.Contains(newLine, "Error") {
				r.processError(newLine)
			} else {
				r.processInfo(newLine)
			}
		}
		if scannerStdOut.Err() != nil {
			r.log.Error("STDOUT: ", scannerStdOut.Err())
		}
		r.log.Info("stdout processing finished")
		done <- struct{}{}
	}()

	// StartCnode the command and check for errors
	err = r.startCommand(cmd)
	if err != nil {
		err = errors.Annotatef(err, "startCommand failed for %s",
			name)

		return err
	}
	// Wait for all output to be processed

	<-done
	r.log.Info("all output processed, cmd: ", cmd.String())
	// Wait for the command to finish
	err = r.waitCommand(cmd)
	if err != nil {
		err = errors.Annotatef(err, "%s:",
			name)
		r.log.Error(err.Error())
		return err
	}

	return err
}

//cardano.node.BlockFetchClient
func (r *R) processError(line string) {
	// Application Exception: 76.255.14.156:3005 ExceededTimeLimit
	//fmt.Println(line)
	if !strings.Contains(line, "Email cannot be sent") {
		fmt.Println(line)
	}

	timeRe := regexp.MustCompile(`Application Exception: (\d+\.+\d+\.+\d+\.+\d+:\d+) ExceededTimeLimit`)
	if timeRe.Match([]byte(line)) {
		l := timeRe.FindStringSubmatch(line)
		r.log.Errorf("time limit error: %s", l[1])
	}
	//cardano_relay1.1.7ok8rxpj2x8d@raspberry00    | [34e768bb:cardano.node.DnsSubscription:Error:17976] [2021-05-10 18:57:35.21 UTC] Domain: "rocinante.mooo.com" Connection Attempt Exception, destination 186.32.161.134:5100 exception: Network.Socket.connect: <socket: 48>: does not exist (Connection refused)
}
func (r *R) processInfo(line string) {
	if !strings.Contains(line, "cardano.node.BlockFetchClient") &&
		!strings.Contains(line, "cardano.node.BlockFetchDecision") {
		fmt.Println(line)
	}
}

func (r *R) startCommand(cmd *exec.Cmd) error {
	lcCmd := cmd.String()
	err := cmd.Start()
	if err != nil {
		err = errors.Annotatef(err, "while attempting to start command %s",
			lcCmd)
		r.log.Error(err.Error())
		time.Sleep(time.Millisecond * 500)
		return err
	}

	return nil
}

func (r *R) waitCommand(cmd *exec.Cmd) error {
	lcCmd := cmd.String()
	err := cmd.Wait()
	if err != nil {
		err = errors.Annotatef(err, "waitCommand failed for %s",
			lcCmd)
		time.Sleep(time.Millisecond * 500)
		return err
	}
	return nil
}
