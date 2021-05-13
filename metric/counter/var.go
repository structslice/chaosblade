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
	Tags     map[string]string `json:"tags"`
}

func MergTags(tag1, tag2 map[string]string) map[string]string {
	new_tags := map[string]string{}
	for k, v := range tag1 {
		new_tags[k] = v
	}
	for k, v := range tag2 {
		new_tags[k] = v
	}
	return new_tags
}
