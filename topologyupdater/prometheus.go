package topologyupdater

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
