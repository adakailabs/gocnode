package configtypes

import (
	"encoding/json"
	"fmt"
)

type MapBackendsCfg map[string]interface{}

type OptionsCfg struct {
	MapBackends map[string][]interface{} `json:"mapBackends"`
}

type RTView struct {
	DefaultBackends []string                  `json:"defaultBackends"`
	DefaultScribes  []DefaultScribe           `json:"defaultScribes"`
	ForwardDelay    interface{}               `json:"forwardDelay"`
	HasEKG          interface{}               `json:"hasEKG"`
	HasGUI          interface{}               `json:"hasGUI"`
	HasGreylog      interface{}               `json:"hasGraylog"`
	HasPrometheus   interface{}               `json:"hasPrometheus"`
	MinSeverity     string                    `json:"minSeverity"`
	Options         OptionsCfg                `json:"options"`
	Rotation        interface{}               `json:"rotation"`
	SetupBackends   []string                  `json:"setupBackends"`
	SetupScribes    []SetupScribeDescriptor   `json:"setupScribes"`
	TraceForwardTo  interface{}               `json:"traceForwardTo"`
	TraceAcceptAt   []TraceAcceptAtDescriptor `json:"traceAcceptAt"`
}

type RemoteAddrDescriptor struct {
	Contents []string `json:"contents"`
	Tag      string   `json:"tag"`
}

type TraceAcceptAtDescriptor struct {
	NodeName   string               `json:"nodeName"`
	RemoteAddr RemoteAddrDescriptor `json:"remoteAddr"`
}

func NewDefaultRTViewConfig() RTView {
	r := RTView{}
	r.DefaultBackends = []string{"KatipBK"}
	r.DefaultScribes = []DefaultScribe{
		[]string{"StdoutSK",
			"stdout"},
		[]string{"FileSK",
			"/home/lovelace/cardano-node/log/cardano-rt-view.log"},
	}
	r.MinSeverity = "Info"

	r.Options = OptionsCfg{}
	r.SetupBackends = []string{
		"KatipBK",
		"LogBufferBK",
		"TraceAcceptorBK",
	}

	r.SetupScribes = []SetupScribeDescriptor{
		{"ScText",
			"FileSK",
			"Emergency",
			"Debug",
			"/home/lovelace/cardano-node/log/cardano-rt-view.log",
			"ScPublic",
			&ScRotationDescriptor{
				10,
				50000,
				24,
			}},
		{
			"ScText",
			"StdoutSK",
			"Emergency",
			"Notice",
			"stdout",
			"ScPublic",
			nil,
		},
	}

	logBufferBk := LogBufferBkCfg{}
	logBufferBk.Kind = "UserDefinedBK"
	logBufferBk.Name = "ErrorBufferBK"

	r.Options.MapBackends = map[string][]interface{}{
		"cardano-rt-view.acceptor": {
			"LogBufferBK", logBufferBk}}

	r.TraceAcceptAt = make([]TraceAcceptAtDescriptor, 0, 10)

	b, _ := json.MarshalIndent(&r, "", "    ")

	fmt.Println(string(b))

	return r
}
