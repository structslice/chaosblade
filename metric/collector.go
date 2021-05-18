package metric

import (
	"context"
	"github.com/chaosblade-io/chaosblade/metric/counter"
	"time"
)

const (
	//metricDelayStop = 15 * time.Minute
	metricDelayStop = 15 * time.Second
)

func ParsTimeUnit(t string) time.Duration {
	duration, err := time.ParseDuration(t)
	if err != nil {
		return time.Duration(5) * time.Second
	}
	return duration
}

type Collector struct {
	run       func(ctx context.Context, args map[string]string) []*counter.MetricData
	indicator *counter.Indicator
	storageC  chan []*counter.MetricData
	cancel    func()
	tags      map[string]string
	Name      string
	StartTime string
}

func (this *Collector) start() {
	this.Name = this.indicator.Metric
	this.StartTime = time.Now().Format("2006-01-02 15:04:05")
	args := this.indicator.Args
	interval := ParsTimeUnit(this.indicator.Interval)
	ctx, cancel := context.WithCancel(context.Background())
	this.cancel = cancel
	go func() {
		sleep := time.NewTicker(interval)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				data := this.run(ctx, args)
				if len(data) > 0 {
					this.storageC <- data
				}
				<-sleep.C
			}
		}
	}()
}

func (this *Collector) stop() {
	timer := time.NewTimer(metricDelayStop)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			this.cancel()
			return
		default:
			time.Sleep(1 * time.Second)
		}
	}
}
