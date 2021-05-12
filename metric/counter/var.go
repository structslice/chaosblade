package counter

type MetricData struct {
	Metric    string            `json:"metric"`
	Tags      map[string]string `json:"tags"`
	Value     float64
	Timestamp int64
}

type Indicator struct {
	Metric   string            `json:"metric"`
	Args     map[string]string `json:"args"`
	Interval string            `json:"interval"`
}
