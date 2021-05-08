package metric

import (
	"context"
	"time"
)

const (
	metricDelayStop = 15 * time.Minute
)

type Collector struct {
	run       func(ctx context.Context, args map[string]string, interval time.Duration) []*MetricData
	indicator *Indicator
	storageC  chan []*MetricData
	cancel    func()
}

func (this *Collector) start() {
	args := this.indicator.Args
	interval := this.indicator.Interval
	ctx, cancel := context.WithCancel(context.Background())
	this.cancel = cancel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				data := this.run(ctx, args, interval)
				if len(data) > 0 {
					this.storageC <- data
				}
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
