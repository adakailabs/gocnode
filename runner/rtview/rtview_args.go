package rtview

import (
	"fmt"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"github.com/adakailabs/gocnode/runner/gen"
	"github.com/k0kubun/pp"
)

type R struct {
	gen.R
}

func NewRtViewRunner(conf *config.C) (r *R, err error) {
	r = &R{}
	r.C = conf
	if r.Log, err = l.NewLogConfig(conf, "runner"); err != nil {
		return r, err
	}
	r.P.Log = r.Log
	r.Cmd0Path = "/usr/local/rt-view/cardano-rt-view"
	r.Cmd0Args = make([]string, 0, 10)
	r.Cmd0Args = append(r.Cmd0Args,
		"--static",
		"/usr/local/rt-view/static")

	return r, err
}

func (r *R) StartRtView() error {
	r.Log.Info("starting rtview")

	d, err := New(r.C)
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

	r.Log.Info(pp.Sprint(r.Cmd0Args))

	cmdsErr := make(chan error)

	go func() {
		if er := r.P.Exec("rtview", r.Cmd0Path, r.Cmd0Args, r.Cmd0); er != nil {
			cmdsErr <- er
		}
	}()

	err = <-cmdsErr

	return err
}
