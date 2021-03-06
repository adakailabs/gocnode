package gen

import (
	"os/exec"

	"github.com/adakailabs/gocnode/runner/process"

	"github.com/adakailabs/gocnode/config"
	"go.uber.org/zap"
)

type R struct {
	C        *config.C
	NodeC    *config.Node
	NodeID   int
	Log      *zap.SugaredLogger
	Cmd1Args []string
	Cmd1Path string
	Cmd0Args []string
	Cmd0Path string
	Cmd0     *exec.Cmd
	Cmd1     *exec.Cmd
	P        process.P
}
