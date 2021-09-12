package tcptrace

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/juju/errors"
)

type P struct {
	Log *zap.SugaredLogger
}

func (r *P) Exec(name, cmdPath string, cmdArgs []string) (err error) {
	var ioBufferStdOut io.ReadCloser
	var ioBufferStdErr io.ReadCloser

	r.Log.Info("Cmd0 args: ", cmdPath, cmdArgs)
	cmd := exec.Command(cmdPath, cmdArgs...)

	// Get a pipe to read from standard out
	ioBufferStdOut, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}
	ioBufferStdErr, err = cmd.StderrPipe()
	if err != nil {
		return err
	}

	r.Log.Debugf("running Cmd0: %s ", cmd.String())

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	// Create a scannerStdErr which scans r InputC a line-by-line fashion
	scannerStdErr := bufio.NewScanner(ioBufferStdErr)
	scannerStdErrBuf := make([]byte, 0, 5*1024*1024)
	scannerStdErr.Buffer(scannerStdErrBuf, 5*1024*1024)

	scannerStdOut := bufio.NewScanner(ioBufferStdOut)
	scannerStdOutBuf := make([]byte, 0, 5*1024*1024)
	scannerStdOut.Buffer(scannerStdOutBuf, 5*1024*1024)

	// Use the scannerStdErr to scan the output line by line and Log it
	// It's running InputC a goroutine so that it doesn't block
	go func() {
		// Read line by line and process it
		for scannerStdErr.Scan() {
			newLine := scannerStdErr.Text() + "\n"
			fmt.Println(newLine)
		}
		if scannerStdErr.Err() != nil {
			r.Log.Error("STDERR: ", scannerStdErr.Err())
		}
		r.Log.Info("stderr processing finished")
		done <- struct{}{}
	}()

	go func() {
		// Read line by line and process it
		for scannerStdOut.Scan() {
			newLine := scannerStdOut.Text()
			r.Log.Info(newLine)
			if strings.Contains(newLine, "open") {
				r.Log.Info("ip/port is open")
				aSlice := strings.Fields(newLine)
				r.Log.Info("slice: ", aSlice[4])
				r.Log.Info("slice: ", aSlice[6])
				r.Log.Info("slice: ", aSlice[8])
			}
		}
		if scannerStdOut.Err() != nil {
			r.Log.Error("STDOUT: ", scannerStdOut.Err())
		}
		r.Log.Info("stdout processing finished")

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
	r.Log.Info("all output processed, cmd: ", cmd.String())
	// Wait for the command to finish
	err = r.waitCommand(cmd)
	if err != nil {
		err = errors.Annotatef(err, "%s:",
			name)
		r.Log.Error(err.Error())
		return err
	}

	return err
}

func (r *P) processError(line string) {
	if !strings.Contains(line, "Email cannot be sent") {
		fmt.Println(line)
	}

	timeRe := regexp.MustCompile(`Application Exception: (\d+\.+\d+\.+\d+\.+\d+:\d+) ExceededTimeLimit`)
	if timeRe.Match([]byte(line)) {
		l := timeRe.FindStringSubmatch(line)
		r.Log.Errorf("time limit error: %s", l[1])
	}
}
func (r *P) processInfo(line string) {
	if !strings.Contains(line, "cardano.node.BlockFetchClient") &&
		!strings.Contains(line, "cardano.node.BlockFetchDecision") {
		fmt.Println(line)
	}
}

func (r *P) startCommand(cmd *exec.Cmd) error {
	lcCmd := cmd.String()
	err := cmd.Start()
	if err != nil {
		err = errors.Annotatef(err, "while attempting to start command %s",
			lcCmd)
		r.Log.Error(err.Error())
		time.Sleep(time.Millisecond * 500)
		return err
	}

	return nil
}

func (r *P) waitCommand(cmd *exec.Cmd) error {
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
