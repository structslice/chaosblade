package counter

import (
	"context"
	"github.com/shirou/gopsutil/cpu"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

func CPUCollect(ctx context.Context, args map[string]string) (metricdatas []*MetricData) {
	var percpu bool
	if _, ok := args["--cpu-count"]; ok {
		percpu = true
	}
	if _, ok := args["--cpu-list"]; ok {
		percpu = true
	}
	ts := time.Now().Unix()
	percents, err := cpu.PercentWithContext(ctx, time.Duration(3)*time.Second, percpu)
	if err != nil {
		logrus.Errorf("collect cpu percent metric faild, err: %v", err)
		return
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
