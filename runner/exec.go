package runner

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/juju/errors"
)

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
