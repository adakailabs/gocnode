package promquery

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/k0kubun/pp"

	l "github.com/adakailabs/gocnode/logger"

	"github.com/adakailabs/gocnode/config"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type ResultMetric struct {
	Metric QueryResult   `json:"metric"`
	Value  []interface{} `json:"value"`
}

type QueryResult struct {
	Name     string `json:"__name__"`
	Instance string `json:"instance"`
	Job      string `json:"job"`
}

type PromeData struct {
	ResultType string         `json:"resultType"`
	Result     []ResultMetric `json:"result"`
}

type PromResponse struct {
	Status string    `json:"status"`
	Data   PromeData `json:"data"`
}

type TU struct {
	node     config.Node
	testMode bool
	log      *zap.SugaredLogger
	client   *resty.Client
}

func New(c *config.C, nodeID int) (tu *TU, err error) {
	tu = &TU{}
	if tu.log, err = l.NewLogConfig(c, "promquery"); err != nil {
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

func (t *TU) GetCardanoBlock() (string, error) {
	prometheusHost := "http://prometheus:9090"
	if t.testMode {
		prometheusHost = "http://192.168.100.46:9090"
	}

	t.log.Info("prometheus host: ", prometheusHost)

	prometheusQueryURL := fmt.Sprintf("%s/api/v1/query?query=", prometheusHost)

	url := fmt.Sprintf("%s%s", prometheusQueryURL, "cardano_node_metrics_blockNum_int")

	t.log.Info("prometheus query: ", url)

	resp, err := t.client.R().
		EnableTrace().
		Get(url)
	if err != nil {
		return "", err
	}

	p := PromResponse{}
	err = json.Unmarshal(resp.Body(), &p)

	for _, r := range p.Data.Result {
		t.log.Info(r.Metric.Job)
		t.log.Info(t.node.Name)
		if strings.Contains(r.Metric.Job, t.node.Name) {
			return r.Value[1].(string), err
		}
	}
	return "", fmt.Errorf("while quering prometheus for value %s: value not found", "cardano_node_metrics_blockNum_int")
}

func (t *TU) GetCardanoConnectedPeers() (string, error) {
	prometheusHost := "http://prometheus:9090"
	if t.testMode {
		prometheusHost = "http://192.168.100.46:9090"
	}

	t.log.Info("prometheus host: ", prometheusHost)

	prometheusQueryURL := fmt.Sprintf("%s/api/v1/query?query=", prometheusHost)

	url := fmt.Sprintf("%s%s", prometheusQueryURL, "cardano_node_metrics_connectedPeers_int")

	t.log.Info("prometheus query: ", url)

	resp, err := t.client.R().
		EnableTrace().
		Get(url)
	if err != nil {
		return "", err
	}

	p := PromResponse{}
	err = json.Unmarshal(resp.Body(), &p)

	t.log.Debug(pp.Sprint(p))

	for _, r := range p.Data.Result {
		t.log.Info(r.Metric.Job)
		t.log.Info(t.node.Name)
		if strings.Contains(r.Metric.Job, t.node.Name) {
			return r.Value[1].(string), err
		}
	}
	return "", fmt.Errorf("while quering prometheus for value %s: value not found", "cardano_node_metrics_blockNum_int")
}

func (t *TU) CheckConnectedPeers() (string, error) {
	t.log.Info("ext relay: ", len(t.node.ExtRelays))
	t.log.Info("relays: ", len(t.node.Relays))

	expConnected := t.node.Peers + uint(len(t.node.Producers)) + 1
	t.log.Info("expected connected: ", expConnected)

	return "", nil
}
