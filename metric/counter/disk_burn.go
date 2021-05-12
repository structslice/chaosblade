package counter

import (
	"context"
	"fmt"
	"github.com/chaosblade-io/chaosblade-spec-go/channel"
	"github.com/shirou/gopsutil/disk"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

func ioReadBytes(arr [2]*ioStat) uint64 {
	if arr[0] == nil {
		return 0
	}
	return arr[1].stat.ReadBytes - arr[0].stat.ReadBytes
}
func ioWriteBytes(arr [2]*ioStat) uint64 {
	if arr[0] == nil {
		return 0
	}
	return arr[1].stat.WriteBytes - arr[0].stat.WriteBytes
}

func ioUtil(arr [2]*ioStat) float64 {
	if arr[0] == nil {
		return 0
	}
	use := arr[1].stat.IoTime - arr[0].stat.IoTime
	duration := uint64(arr[1].ts.Sub(arr[0].ts).Nanoseconds() / 1000000)
	fmt.Println(use, duration)
	if duration == 0 {
		return 0
	}
	io_util := float64(use) * 100.0 / float64(duration)
	if io_util > 100.0 {
		io_util = 100.0
	}
	return io_util

}

type ioStat struct {
	stat disk.IOCountersStat
	ts   time.Time
}

func get_mount_disk(ctx context.Context, path string) (dev string) {
	cmd := fmt.Sprintf(`-h %s | awk 'NR!=1 {print $1","$NF}' | tr '\n' ' '`, path)
	response := channel.NewLocalChannel().Run(ctx, "df", cmd)
	if !response.Success {
		logrus.Errorf("collect disk io  metric faild, get_mount_disk_map exec dt %s faild %v", cmd, response.Err)
		return
	}
	disks := response.Result.(string)
	fields := strings.Fields(disks)
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		logrus.Errorf("collect disk io metric faild, get disk Partitions faild, %v", err)
		return
	}
	device := ""
	for _, df := range fields {
		if strings.HasPrefix(df, "/") {
			arr := strings.Split(df, ",")
			if len(arr) < 2 {
				continue
			}
			for _, partition := range partitions {
				if partition.Mountpoint == arr[1] {
					device = partition.Device
					break
				}
			}
			if device != "" {
				dev = strings.ReplaceAll(device, "/dev/", "")
				return
			}
		}
	}
	return ""
}

func DiskBurnCollect(ctx context.Context, args map[string]string) (metricdatas []*MetricData) {
	var Path string
	var isRead, isWrite bool
	if path, ok := args["--path"]; ok {
		Path = path
	}
	if _, ok := args["--read"]; ok {
		isRead = true
	}
	if _, ok := args["--write"]; ok {
		isWrite = true
	}
	device := get_mount_disk(ctx, Path)
	if device == "" {
		logrus.Errorf("collect disk io metric faild, get device from path %s faild", Path)
		return
	}
	ioStats := [2]*ioStat{}
	ts := time.Now().Unix()
	for i := 0; i < 2; i++ {
		iocounterMap, _ := disk.IOCountersWithContext(ctx)
		for name, iocounter := range iocounterMap {
			if strings.Contains(device, name) && strings.HasPrefix(device, name) {
				ioStats[i] = &ioStat{iocounter, time.Now()}
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	if isRead {
		io_read_bytes, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", float64(ioReadBytes(ioStats))/1024), 64)
		metricdata := MetricData{
			Metric:    "chaos_disk_io_read",
			Tags:      map[string]string{"device": device, "path": Path},
			Value:     io_read_bytes,
			Timestamp: ts,
		}
		metricdatas = append(metricdatas, &metricdata)
	}
	if isWrite {
		io_write_bytes, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", float64(ioWriteBytes(ioStats))/1024), 64)
		metricdata := MetricData{
			Metric:    "chaos_disk_io_write",
			Tags:      map[string]string{"device": device, "path": Path},
			Value:     io_write_bytes,
			Timestamp: ts,
		}
		metricdatas = append(metricdatas, &metricdata)
	}

	io_util, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", ioUtil(ioStats)), 64)
	metricdata := MetricData{
		Metric:    "chaos_disk_io_util",
		Tags:      map[string]string{"device": device, "path": Path},
		Value:     io_util,
		Timestamp: ts,
	}
	metricdatas = append(metricdatas, &metricdata)
	return metricdatas
}
