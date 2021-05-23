package prometheuscfg

import (
	"fmt"

	"github.com/adakailabs/gocnode/runner/gen"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"github.com/k0kubun/pp"
)

type R struct {
	gen.R
}

func NewPrometheusRunner(conf *config.C, nodeID int, isProducer bool) (r *R, err error) {
	r = &R{}
	r.C = conf
	if r.Log, err = l.NewLogConfig(conf, "runner"); err != nil {
		return r, err
	}
	r.P.Log = r.Log
	r.Cmd0Path = "prometheus"
	r.Cmd0Args = make([]string, 0, 10)
	r.Cmd0Args = append(r.Cmd0Args,
		"--storage.tsdb.path=/prometheus",
		"--web.console.libraries=/usr/share/prometheus/console_libraries",
		"--web.console.templates=/usr/share/prometheus/consoles")

	return r, err
}

func (r *R) StartPrometheus() error {
	r.Log.Info("starting prometheus")

	d, err := New(r.C)
	if err != nil {
		return err
	}

	var file string
	if file, err = d.CreateConfigFile(); err != nil {
		return err
	}

	r.Cmd0Args = append(r.Cmd0Args, fmt.Sprintf("--config.file=%s", file))

	r.Log.Info(pp.Sprint(r.Cmd0Args))

	cmdsErr := make(chan error)

	go func() {
		if er := r.P.Exec("prometheus", r.Cmd0Path, r.Cmd0Args, r.Cmd0); er != nil {
			cmdsErr <- er
		}
	}()

	err = <-cmdsErr

	return err
}
