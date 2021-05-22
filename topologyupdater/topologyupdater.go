package topologyupdater

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/k0kubun/pp"

	"github.com/adakailabs/gocnode/cardanocfg"
	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

const APIURL = "https://api.clio.one/htopology/v1"

type UpdaterGetNodes struct {
	Resultcode string            `json:"resultcode"`
	Datetime   string            `json:"datetime"`
	ClientIP   string            `json:"clientIp"`
	Iptype     uint              `json:"iptype"`
	Msg        string            `json:"msg"`
	Producers  []cardanocfg.Node `json:"producers"`
}

type TU struct {
	node     config.Node
	testMode bool
	log      *zap.SugaredLogger
	client   *resty.Client
}

func New(c *config.C, nodeID int) (tu *TU, err error) {
	tu = &TU{}
	if tu.log, err = l.NewLogConfig(c, "topologyupdater"); err != nil {
		return tu, err
	}
	tu.testMode = c.TestMode
	if tu.testMode {
		tu.log.Warnf("testmode is enabled")
	}
	tu.node = c.Relays[nodeID]
	tu.client = resty.New()
	return tu, err
}

func (t *TU) GetTopology() (UpdaterGetNodes, error) {
	url := fmt.Sprintf("%s/fetch/?max=%d&magic=%d&ipv=%d",
		APIURL,
		t.node.Peers,
		t.node.NetworkMagic,
		4)

	resp, err := t.client.R().
		EnableTrace().
		Get(url)
	if err != nil {
		return UpdaterGetNodes{}, err
	}

	toReturn := UpdaterGetNodes{}

	err = json.Unmarshal(resp.Body(), &toReturn)
	return toReturn, err
}

func (t *TU) GetCardanoBlock() (string, error) {
	prometheusHost := "http://prometheus:9090"
	if t.testMode {
		prometheusHost = "http://192.168.100.45:9090"
	}
	prometheusQueryURL := fmt.Sprintf("%s/api/v1/query?query=", prometheusHost)

	url := fmt.Sprintf("%s%s", prometheusQueryURL, "cardano_node_metrics_blockNum_int")
	resp, err := t.client.R().
		EnableTrace().
		Get(url)
	if err != nil {
		return "", err
	}
	p := PromResponse{}
	err = json.Unmarshal(resp.Body(), &p)

	for _, r := range p.Data.Result {
		if strings.Contains(r.Metric.Job, t.node.Name) {
			return r.Value[1].(string), err
		}
	}
	return "", fmt.Errorf("value not found")
}

func (t *TU) Ping() (int, error) {
	block, err := t.GetCardanoBlock()
	if err != nil {
		return 0, err
	}

	query := "?port=%d&blockNo=%s&valency=%d&magic=%d"
	query = fmt.Sprintf(query, t.node.Port, block, 1, t.node.NetworkMagic)

	t.log.Infof("network: %s name: %s port: %d", t.node.Network, t.node.Name, t.node.Port)

	url := fmt.Sprintf("%s/%s", APIURL, query)
	t.log.Info("url: ", url)
	resp, err := t.client.R().
		EnableTrace().
		Get(url)
	if err != nil {
		return 0, err
	}

	toReturn := UpdaterGetNodes{}

	err = json.Unmarshal(resp.Body(), &toReturn)
	if err != nil {
		return 0, err
	}

	t.log.Info(pp.Sprint(toReturn))
	t.log.Info("topology updater says: ", toReturn.Msg)
	code, err := strconv.Atoi(toReturn.Resultcode)

	if err != nil {
		return 0, err
	}

	return code, err
}
