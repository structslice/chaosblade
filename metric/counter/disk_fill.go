package counter

import (
	"context"
	"fmt"
	"github.com/shirou/gopsutil/disk"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

func DiskFillCollect(ctx context.Context, args map[string]string) (metricdatas []*MetricData) {
	if path, ok := args["--path"]; ok {
		if disk_usage, err := disk.UsageWithContext(ctx, path); err == nil {
			used_percent, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", disk_usage.UsedPercent), 64)
			ts := time.Now().Unix()
			metricdata := MetricData{
				Metric:    "chaos_disk_used_percent",
				Tags:      map[string]string{"path": path},
				Value:     used_percent,
				Timestamp: ts,
			}
			metricdatas = append(metricdatas, &metricdata)
		} else {
			logrus.Errorf("collect disk usage metric faild, get Usage from path %s faild", path)
		}
	} else {
		logrus.Errorf("collect disk usage metric faild, --path params is not exists")
	}

	return metricdatas
}
