package topologyfile

import (
	"time"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"go.uber.org/zap"
)

type Tpf struct {
	log *zap.SugaredLogger
}

func New(c *config.C) (*Tpf, error) {
	var err error
	d := &Tpf{}
	if d.log, err = l.NewLogConfig(c, "config"); err != nil {
		return d, err
	}

	return d, nil
}

type T struct {
	Producers []Node `json:"Producers"`
}

type NodeList []Node

// Len is part of sort.Interface.
func (ms NodeList) Len() int {
	return len(ms)
}

// Swap is part of sort.Interface.
func (ms NodeList) Swap(i, j int) {
	ms[i], ms[j] = ms[j], ms[i]
}

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that discriminates between
// the two items (one is less than the other). Note that it can call the
// less functions twice per call. We could change the functions to return
// -1, 0, 1 and reduce the number of calls for greater efficiency: an
// exercise for the reader.
func (ms NodeList) Less(i, j int) bool {
	return ms[i].latency < ms[j].latency
}

type Node struct {
	Atype   string `json:"type"`
	Addr    string `json:"addr"`
	Port    uint   `json:"port"`
	Valency uint   `json:"valency"`
	Debug   string `json:"debug"`
	latency time.Duration
}

func (p *Node) SetLatency(la time.Duration) {
	p.latency = la
}

func (p *Node) GetLatency() time.Duration {
	return p.latency
}
