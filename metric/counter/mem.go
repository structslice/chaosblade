package counter

import (
	"context"
	"fmt"
	"github.com/shirou/gopsutil/mem"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

func MEMCollect(ctx context.Context, args map[string]string) (metricdatas []*MetricData) {
	var burnMemMode string
	var includeBufferCache bool
	if mode, ok := args["--mode"]; ok {
		burnMemMode = mode
	} else {
		burnMemMode = "cache"
	}
	if _, ok := args["--include-buffer-cache"]; ok {
		includeBufferCache = true
	}
	ts := time.Now().Unix()
	virtualMemory, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		logrus.Errorf("collect mem used percent metric faild, err: %v", err)
		return
	}
	var total, available float64
	total = float64(virtualMemory.Total)
	available = float64(virtualMemory.Free)
	if burnMemMode == "ram" && !includeBufferCache {
		available = available + float64(virtualMemory.Buffers+virtualMemory.Cached)
	}
	used_percent, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", (1-available/total)*100), 64)

	metricdata := MetricData{
		Metric:    "chaos_mem_used_percent",
		Tags:      map[string]string{},
		Value:     used_percent,
		Timestamp: ts,
	}
	metricdatas = append(metricdatas, &metricdata)
	return metricdatas
}
