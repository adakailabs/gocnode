package prometheuscfg

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type PromJob struct {
	JobName        string        `yaml:"job_name"`
	ScrapeInterval time.Duration `yaml:"scrape_interval"`
	StaticConfigs  []SConfigs    `yaml:"static_configs"`
	ScrapeTimeout  time.Duration `yaml:"scrape_timeout"`
}

func NewPromJob(name string, targetHost string, interval, timeout time.Duration) PromJob {
	pr := PromJob{}
	pr.JobName = name
	pr.ScrapeInterval = interval
	pr.ScrapeTimeout = timeout
	pr.StaticConfigs = make([]SConfigs, 1, 1)
	pr.StaticConfigs[0] = SConfigs{[]string{targetHost}}
	return pr
}

type SConfigs struct {
	Targets []string `yaml:"targets"`
}

type PromConfig struct {
	Global        GlbConfig `yaml:"global"`
	ScrapeConfigs []PromJob `yaml:"scrape_configs"`
}

type GlbConfig struct {
	ScrapeInterval     time.Duration     `yaml:"scrape_interval"`
	ScrapeTimeout      time.Duration     `yaml:"scrape_timeout"`
	EvaluationInternal time.Duration     `yaml:"evaluation_interval"`
	ExternalLabels     map[string]string `yaml:"external_labels"`
}

type SrpConfigs struct {
	Jobs []PromJob
}

type Cfg struct {
	conf *config.C
	log  *zap.SugaredLogger
}

func New(c *config.C) (*Cfg, error) {
	var err error
	d := &Cfg{}
	d.conf = c
	if d.log, err = l.NewLogConfig(c, "config"); err != nil {
		return d, err
	}

	return d, nil
}

func (c *Cfg) GetYaml() []byte {
	pcfg := PromConfig{}

	g := GlbConfig{}
	g.EvaluationInternal = time.Second * 15
	g.ScrapeInterval = time.Second * 15
	g.ExternalLabels = map[string]string{"monitor": "adakailabs"}

	p1 := NewPromJob("master", "localhost:9100", time.Second*5, time.Second*5)
	pcfg.ScrapeConfigs = make([]PromJob, 1)
	pcfg.ScrapeConfigs[0] = p1

	for _, r := range c.conf.Relays {
		exporterName := fmt.Sprintf("%s-exporter", r.Name)
		cardanoName := fmt.Sprintf("%s-cardano", r.Name)
		pExpHost := fmt.Sprintf("%s:%d", r.Name, r.PromeNExpPort)
		pCardHost := fmt.Sprintf("%s:%s", r.Name, "12798")
		p1 := NewPromJob(exporterName, pExpHost, time.Second*5, time.Second*5)
		p2 := NewPromJob(cardanoName, pCardHost, time.Second*5, time.Second*5)
		pcfg.ScrapeConfigs = append(pcfg.ScrapeConfigs, p1, p2)
	}

	pcfg.Global = g

	d, err := yaml.Marshal(&pcfg)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Println(string(d))

	return d
}

func (c *Cfg) CreateConfigFile() (file string, err error) {
	cfgBytes := c.GetYaml()
	file = "/prometheus/prometheus.yaml"
	if err := ioutil.WriteFile(file, cfgBytes, os.ModePerm); err != nil {
		return file, err
	}
	return file, err
}
