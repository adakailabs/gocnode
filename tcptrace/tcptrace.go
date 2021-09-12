package tcptrace

import (
	l "github.com/adakailabs/gocnode/logger"
	"go.uber.org/zap"
)

type Config struct {
	logLevel string
}

func (c *Config) LogLevel() string {
	return c.logLevel
}

type Trace struct {
	Log *zap.SugaredLogger
	p   *P
}

func New() (t *Trace, err error) {
	t = &Trace{}
	c := &Config{}
	c.logLevel = "info"
	if t.Log, err = l.NewLogConfig(c, "config"); err != nil {
		return t, err
	}

	t.p = &P{}
	t.p.Log = t.Log

	return t, err
}

func (tc *Trace) Exec(name, cmdPath string, cmdArgs []string) error {
	return tc.p.Exec(name, cmdPath, cmdArgs)
}
