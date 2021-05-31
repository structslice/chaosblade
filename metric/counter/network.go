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
	results := make(chan *TCPingResult, len(remote_ports)*len(destination_ips))
	for _, dst_ip := range destination_ips {
		for _, remote_port := range remote_ports {
			wg.Add(1)
			go func() {
				pinger := TCPing{
					done: make(chan struct{}),
				}
				timeoutDuration, _ := time.ParseDuration("2s")
				intervalDuration, _ := time.ParseDuration("500ms")
				target := TCPingTarget{
					Timeout:  timeoutDuration,
					Interval: intervalDuration,
					Host:     dst_ip,
					Port:     remote_port,
					Counter:  2,
				}
				pinger.target = &target
				pingerDone := pinger.Start()
				select {
				case <-pingerDone:
					results <- pinger.result
					wg.Done()
				case <-ctx.Done():
					wg.Done()
					return
				}
			}()
		}
		go func() {
			for {
				result, ok := <-results
				if !ok {
					break
				}
				metricdata := MetricData{
					Metric:    "chaos_network_delay",
					Tags:      map[string]string{"dst_ip": result.Target.Host, "remote-port": strconv.Itoa(result.Target.Port)},
					Value:     float64(result.Avg().Milliseconds()),
					Timestamp: ts,
				}
				metricdatas = append(metricdatas, &metricdata)
			}
		}()
	}
	wg.Wait()
	close(results)
	return metricdatas
}

const LossPKCMD = `tc -s -d qdisc show | grep netem -A 1 |grep Sent | awk '{print $4,$7}' | cut -d ',' -f 1`

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
			go func() {
				defer wg.Done()
				response := channel.NewLocalChannel().Run(ctx, LossPKCMD, "")
				if !response.Success {
					logrus.Error("collect network loss metric faild, err: dst_ip %s remote-port %d exec LossPKCMD faild ,%v", dst_ip, remote_port, response.Err)
					return
				}
				data := response.Result.(string)
				data_split := strings.Split(data, " ")
				if len(data_split) != 2 {
					logrus.Error("collect network loss metric faild, err: dst_ip %s remote-port %d exec LossPKCMD response invaild  %s", dst_ip, remote_port, data)
					return
				}
				total_pkg, _ := strconv.Atoi(data_split[0])
				drop_pkg, _ := strconv.Atoi(data_split[1])
				var loss_pkg_percent float64
				if total_pkg == 0 {
					loss_pkg_percent = 0
				} else {
					loss_pkg_percent, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", float64(drop_pkg)/float64(total_pkg)*100), 64)
				}
				metricdata := MetricData{
					Metric:    "chaos_network_loss",
					Tags:      map[string]string{"dst_ip": dst_ip, "remote-port": strconv.Itoa(remote_port)},
					Value:     loss_pkg_percent,
					Timestamp: ts,
				}
				results <- &metricdata

			}()
		}
	}

	go func() {
		for {
			result, ok := <-results
			if !ok {
				break
			}
			metricdatas = append(metricdatas, result)
		}
	}()
	wg.Wait()
	close(results)
	return metricdatas
}
