package tcptrace_test

import (
	"os"
	"testing"

	"github.com/adakailabs/gocnode/tcptrace"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func TestBasic(t *testing.T) {
	a := assert.New(t)
	tc, err := tcptrace.New()
	a.Nil(err)
	a.NotNil(tc)

	const cmdPath = "/usr/bin/tcptraceroute"
	const cmdArgs = "www.google.com"

	cmdArgsSlice := []string{cmdArgs}

	if er := tc.Exec("cardano-node", cmdPath, cmdArgsSlice); er != nil {
		tc.Log.Error(er.Error())
	}
}
