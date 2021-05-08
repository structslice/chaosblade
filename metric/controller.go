package metric

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
)

type MetricData struct {
	Metric    string            `json:"metric"`
	Tags      map[string]string `json:"tags"`
	Value     float64
	Timestamp int64
}

type MetricCollect struct {
	workers    map[string]*Collector
	indicatorC chan *Indicator
	storageC   chan []*MetricData
	lock       sync.Mutex
}

func (this *MetricCollect) Start() {
	this.workers = make(map[string]*Collector)
	this.indicatorC = make(chan *Indicator)
	this.storageC = make(chan []*MetricData)
	go this.report()
	this.collect()
}

func (this *MetricCollect) newcollector(indicator *Indicator) *Collector {
	collector := &Collector{
		indicator: indicator,
		storageC:  this.storageC,
	}
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

func (this *MetricCollect) StopCollector(indicator *Indicator) error {
	if collector, ok := this.workers[indicator.Metric]; ok {
		collector.stop()
		this.unregister(collector)
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
			collector.run = cpu_collect
		}
		collector.start()
	}

}

func (this *MetricCollect) report() {
	for metricdata := range this.storageC {
		fmt.Println(metricdata)
	}
}
