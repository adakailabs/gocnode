package logger

import (
	"encoding/json"

	"github.com/juju/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type config interface {
	LogLevel() string
}

func NewLogConfig(c config, name string) (loggerout *zap.SugaredLogger, err error) {
	var logger *zap.Logger
	rawJSON := []byte(`{
	  "level": "debug",
	  "encoding": "console",
	  "outputPaths": ["stdout", "/tmp/logs"],
	  "errorOutputPaths": ["stderr"],
	  "initialFields": {"component": "config"},
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase"
	  }
	}`)

	var cfg zap.Config
	if err = json.Unmarshal(rawJSON, &cfg); err != nil {
		err = errors.Annotate(err, "loading zap configuration")

		return nil, err
	}

	// set the logger level
	aLevel := zap.NewAtomicLevel()
	if e := aLevel.UnmarshalText([]byte(c.LogLevel())); e != nil {
		return nil, e
	}
	cfg.Level = aLevel

	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.InitialFields = map[string]interface{}{
		"component": name,
	}
	logger, err = cfg.Build()
	if err != nil {
		err = errors.Annotate(err, "building zap configuration")
		return nil, err
	}
	return logger.Sugar(), err
}
