package counter

import (
	"context"
	"fmt"
	"github.com/chaosblade-io/chaosblade-spec-go/channel"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"sync"
	"time"
)

func NetworkDelay(ctx context.Context, args map[string]string) (metricdatas []*MetricData) {
	var destination_ips []string
	var remote_ports []int
	var delay_time int
	if dst_ip, ok := args["--destination-ip"]; ok {
		destination_ips = strings.Split(dst_ip, ",")
	} else {
		logrus.Error("collect network delay metric faild, err: loss --destination-ip params")
		return
	}
	if delay, ok := args["--time"]; ok {
		var err error
		delay_time, err = strconv.Atoi(delay)
		if err != nil {
			logrus.Error("collect network delay metric faild, err: params --time is invaild")
			return
		}
	} else {
		logrus.Error("collect network delay metric faild, err: loss --time params")
		return
	}
	if rt_port, ok := args["--remote-port"]; ok {
		for _, port := range strings.Split(rt_port, ",") {
			port_range := strings.Split(port, "-")
			if len(port_range) == 1 {
				remote_port, err := strconv.Atoi(port_range[0])
				if err != nil {
					logrus.Errorf("collect network delay metric faild, err: --remote-port %s is invaild", port_range[0])
				} else {
					remote_ports = append(remote_ports, remote_port)
				}
			} else {
				var startN, endN int
				start, err := strconv.Atoi(port_range[0])
				if err != nil {
					logrus.Errorf("collect network delay metric faild, err: --remote-port  %s is invaild", port_range[0])
					continue
				}
				end, err := strconv.Atoi(port_range[1])
				if err != nil {
					logrus.Errorf("collect network delay metric faild, err: --remote-port  %s is invaild", port_range[1])
					continue
				}
				if start > end {
					startN = end
					endN = start
				} else {
					startN = start
					endN = end
				}
				for i := startN; i <= endN; i++ {
					remote_ports = append(remote_ports, i)
				}
			}
		}

	} else {
		logrus.Error("collect network delay metric faild, err: loss --remote-port params")
		return
	}
	ts := time.Now().Unix()
	wg := sync.WaitGroup{}
	results := make(chan *MetricData, len(remote_ports)*len(destination_ips))
	for _, dst_ip := range destination_ips {
		for _, remote_port := range remote_ports {
			wg.Add(1)
			go func(ip string, port int) {
				defer wg.Done()
				pinger := TCPing{
					done: make(chan struct{}),
				}
				if delay_time*2 > 5000 {
					delay_time = 5000
				}
				timeoutDuration := time.Duration(delay_time*2) * time.Millisecond
				intervalDuration, _ := time.ParseDuration("500ms")
				target := TCPingTarget{
					Timeout:  timeoutDuration,
					Interval: intervalDuration,
					Host:     ip,
					Port:     port,
					Counter:  2,
				}
				pinger.target = &target
				pinger.result = &TCPingResult{Target: &target}
				pingerDone := pinger.Start()
				select {
				case <-pingerDone:
					metricdata := MetricData{
						Metric:    "chaos_network_delay",
						Tags:      map[string]string{"dst_ip": ip, "remote-port": strconv.Itoa(port)},
						Value:     float64(pinger.result.Avg().Milliseconds()),
						Timestamp: ts,
					}
					results <- &metricdata
				case <-ctx.Done():
					return
				}
			}(dst_ip, remote_port)
		}
	}
	wg.Wait()
	close(results)
	for metricdata := range results {
		metricdatas = append(metricdatas, metricdata)
	}
	return metricdatas
}

const LossPKCMD = `-s -d qdisc show | grep netem -A 1 |grep Sent | awk '{print $4,$7}' | cut -d ',' -f 1`

func NetworkLoss(ctx context.Context, args map[string]string) (metricdatas []*MetricData) {
	var destination_ips []string
	var remote_ports []int

	if dst_ip, ok := args["--destination-ip"]; ok {
		destination_ips = strings.Split(dst_ip, ",")
	} else {
		logrus.Error("collect network delay metric faild, err: loss --destination-ip params")
		return
	}
	if rt_port, ok := args["--remote-port"]; ok {
		for _, port := range strings.Split(rt_port, ",") {
			port_range := strings.Split(port, "-")
			if len(port_range) == 1 {
				remote_port, err := strconv.Atoi(port_range[0])
				if err != nil {
					logrus.Errorf("collect network delay metric faild, err: --remote-port %s is invaild", port_range[0])
				} else {
					remote_ports = append(remote_ports, remote_port)
				}
			} else {
				var startN, endN int
				start, err := strconv.Atoi(port_range[0])
				if err != nil {
					logrus.Errorf("collect network delay metric faild, err: --remote-port  %s is invaild", port_range[0])
					continue
				}
				end, err := strconv.Atoi(port_range[1])
				if err != nil {
					logrus.Errorf("collect network delay metric faild, err: --remote-port  %s is invaild", port_range[1])
					continue
				}
				if start > end {
					startN = end
					endN = start
				} else {
					startN = start
					endN = end
				}
				for i := startN; i <= endN; i++ {
					remote_ports = append(remote_ports, i)
				}
			}
		}

	} else {
		logrus.Error("collect network delay metric faild, err: loss --remote-port params")
		return
	}
	ts := time.Now().Unix()
	wg := sync.WaitGroup{}
	results := make(chan *MetricData, len(remote_ports)*len(destination_ips))
	for _, dst_ip := range destination_ips {
		for _, remote_port := range remote_ports {
			wg.Add(1)
			go func(ip string, port int) {
				defer wg.Done()
				sub_ctx, cancel := context.WithCancel(ctx)
				go func(ctx2 context.Context) {
					pinger := TCPing{
						done: make(chan struct{}),
					}
					timeoutDuration := time.Duration(1) * time.Second
					intervalDuration, _ := time.ParseDuration("50ms")
					target := TCPingTarget{
						Timeout:  timeoutDuration,
						Interval: intervalDuration,
						Host:     ip,
						Port:     port,
						Counter:  100,
					}
					pinger.target = &target
					pinger.result = &TCPingResult{Target: &target}
					pingerDone := pinger.Start()
					select {
					case <-pingerDone:
						return
					case <-ctx2.Done():
						return
					}
				}(sub_ctx)
				time.Sleep(1 * time.Second)
				var total_pkg_sum, drop_pkgs_sum int
				for i := 0; i < 3; i++ {
					response := channel.NewLocalChannel().Run(ctx, "tc", LossPKCMD)
					if !response.Success {
						logrus.Errorf("collect network loss metric faild, err: dst_ip %s remote-port %d exec LossPKCMD faild ,%v", ip, port, response.Err)
						return
					}
					data := response.Result.(string)
					data_split := strings.Split(data, " ")
					if len(data_split) != 2 {
						logrus.Errorf("collect network loss metric faild, err: dst_ip %s remote-port %d exec LossPKCMD response invaild  %s", ip, port, data)
						return
					}
					total_pkg, _ := strconv.Atoi(strings.Trim(data_split[0], "\n"))
					drop_pkg, _ := strconv.Atoi(strings.Trim(data_split[1], "\n"))
					total_pkg_sum = total_pkg_sum + total_pkg
					drop_pkgs_sum = drop_pkgs_sum + drop_pkg
					time.Sleep(100 * time.Millisecond)
				}
				cancel()
				var loss_pkg_percent float64
				if total_pkg_sum == 0 {
					loss_pkg_percent = 0
				} else {
					loss_pkg_percent, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", float64(drop_pkgs_sum)/float64(total_pkg_sum)*100), 64)
				}
				metricdata := MetricData{
					Metric:    "chaos_network_loss",
					Tags:      map[string]string{"dst_ip": ip, "remote-port": strconv.Itoa(port)},
					Value:     loss_pkg_percent,
					Timestamp: ts,
				}
				results <- &metricdata

			}(dst_ip, remote_port)
		}
	}
	wg.Wait()
	close(results)
	for metricdata := range results {
		metricdatas = append(metricdatas, metricdata)
	}
	return metricdatas
}
