package metric

import (
	"context"
	"github.com/shirou/gopsutil/cpu"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

func parsTimeUnit(t string) time.Duration {
	if len(t) == 0 {
		return time.Duration(5) * time.Second
	}
	num, err := strconv.Atoi(t[:-1])
	if err != nil {
		return time.Duration(5) * time.Second
	}
	unit := string(t[-1])
	switch unit {
	case "s":
		return time.Duration(num) * time.Second
	case "m":
		return time.Duration(num) * time.Minute
	case "h":
		return time.Duration(num) * time.Hour
	}
	return time.Duration(5) * time.Second
}

type Indicator struct {
	Metric   string
	Args     map[string]string
	Interval time.Duration
}

func cpu_collect(ctx context.Context, args map[string]string, interval time.Duration) (metricdatas []*MetricData) {
	var percpu bool
	if _, ok := args["cpu-count"]; ok {
		percpu = true
	}
	if _, ok := args["cpu-list"]; ok {
		percpu = true
	}
	ts := time.Now().Unix()
	percents, err := cpu.PercentWithContext(ctx, interval, percpu)
	if err != nil {
		logrus.Errorf("collect cpu percent metric faild, err: %v", err)
	}
	if percpu {
		for index, percent := range percents {
			core := strconv.Itoa(index)
			metricdata := MetricData{
				Metric:    "chaos_cpu_percent",
				Tags:      map[string]string{"core": core},
				Value:     percent,
				Timestamp: ts,
			}
			metricdatas = append(metricdatas, &metricdata)
		}
	} else {
		metricdata := MetricData{
			Metric:    "chaos_cpu_percent",
			Tags:      map[string]string{"core": "all"},
			Value:     percents[0],
			Timestamp: ts,
		}
		metricdatas = append(metricdatas, &metricdata)
	}
	return metricdatas
}
