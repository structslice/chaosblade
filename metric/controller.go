package metric

import (
	"errors"
	"fmt"
	"github.com/chaosblade-io/chaosblade/metric/counter"
	"github.com/sirupsen/logrus"
	"net"
	"os"
	"strings"
	"sync"
)

var MetricCollector *MetricCollect

type MetricCollect struct {
	workers    map[string]*Collector
	indicatorC chan *counter.Indicator
	storageC   chan []*counter.MetricData
	lock       sync.Mutex
}

func (this *MetricCollect) Start() {
	this.workers = make(map[string]*Collector)
	this.indicatorC = make(chan *counter.Indicator)
	this.storageC = make(chan []*counter.MetricData)

	go this.report()
	this.collect()
}

func (this *MetricCollect) newcollector(indicator *counter.Indicator) *Collector {
	collector := &Collector{
		indicator: indicator,
		storageC:  this.storageC,
	}
	collector.tags = map[string]string{}
	hostname, _ := os.Hostname()
	ip, _ := getSelfIp([]string{"10.28", "10.27"})
	collector.tags["hostname"] = hostname
	collector.tags["ip"] = ip
	this.register(collector)
	return collector

}

func (this *MetricCollect) register(collector *Collector) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.workers[collector.indicator.Metric] = collector
}
func (this *MetricCollect) unregister(collector *Collector) {
	this.lock.Lock()
	defer this.lock.Unlock()
	if _, ok := this.workers[collector.indicator.Metric]; ok {
		delete(this.workers, collector.indicator.Metric)
	}
}

func (this *MetricCollect) StopCollector(indicator *counter.Indicator) error {
	if collector, ok := this.workers[indicator.Metric]; ok {
		go collector.stop()
		this.unregister(collector)
		logrus.Infof("stop collect %s metric success", indicator.Metric)
		return nil
	} else {
		return errors.New("not found indicator")
	}
}

func (this *MetricCollect) collect() {
	for indicator := range this.indicatorC {
		collector := this.newcollector(indicator)
		logrus.Infof("start collect %s metric ,args: %v ", indicator.Metric, indicator.Args)
		switch indicator.Metric {
		case "cpu":
			collector.run = counter.CPUCollect
		case "mem":
			collector.run = counter.MEMCollect
		case "disk_burn":
			collector.run = counter.DiskBurnCollect
		case "disk_fill":
			collector.run = counter.DiskFillCollect
		case "network_delay":
			collector.run = counter.NetworkDelay
		case "network_loss":
			collector.run = counter.NetworkLoss
		}
		collector.start()
	}

}

func (this *MetricCollect) Exists(indicator *counter.Indicator) bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	_, ok := this.workers[indicator.Metric]
	return ok
}

func (this *MetricCollect) CurrentWorker() map[string]string {
	this.lock.Lock()
	defer this.lock.Unlock()
	workers := map[string]string{}
	for _, collector := range this.workers {
		workers["name"] = collector.Name
		workers["start_time"] = collector.StartTime
	}
	return workers
}
func (this *MetricCollect) Send(indicator *counter.Indicator) {
	go func() {
		this.indicatorC <- indicator
	}()
}
func (this *MetricCollect) report() {
	for metricdata := range this.storageC {
		for _, i := range metricdata {
			fmt.Println(i.Metric, i.Value, i.Timestamp, i.Tags)
		}
	}
}

func getSelfIp(prefix []string) (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				for _, p := range prefix {
					if strings.Contains(ip, p) && strings.HasPrefix(ip, p) {
						return ip, nil
					}
				}
			}

		}
	}

	return "", errors.New("Can not find self ip address!")

}

func init() {
	MetricCollector = &MetricCollect{}
}
