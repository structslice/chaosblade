package counter

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type TCPingTarget struct {
	Protocol string
	Host     string
	Port     int
	Counter  int
	Interval time.Duration
	Timeout  time.Duration
}
type TCPing struct {
	target *TCPingTarget
	done   chan struct{}
	result *TCPingResult
}
type TCPingResult struct {
	Counter        int
	SuccessCounter int
	Target         *TCPingTarget

	MinDuration   time.Duration
	MaxDuration   time.Duration
	TotalDuration time.Duration
}

func (result *TCPingResult) Avg() time.Duration {
	if result.SuccessCounter == 0 {
		return result.Target.Timeout
	}
	return result.TotalDuration / time.Duration(result.SuccessCounter)
}

// Failed return failed counter
func (result *TCPingResult) Failed() int {
	return result.Counter - result.SuccessCounter
}

// Start a tcping
func (tcping *TCPing) Start() <-chan struct{} {
	go func() {
		t := time.NewTicker(tcping.target.Interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if tcping.result.Counter >= tcping.target.Counter && tcping.target.Counter != 0 {
					tcping.Stop()
					return
				}
				//duration, remoteAddr, err := tcping.ping()
				duration, _, err := tcping.ping()
				tcping.result.Counter++

				if err != nil {
					logrus.Errorf("Ping %s - failed: %s\n", tcping.target, err)
				} else {
					//fmt.Printf("Ping %s(%s) - Connected - time=%s\n", tcping.target, remoteAddr, duration)
					if tcping.result.MinDuration == 0 {
						tcping.result.MinDuration = duration
					}
					if tcping.result.MaxDuration == 0 {
						tcping.result.MaxDuration = duration
					}
					tcping.result.SuccessCounter++
					if duration > tcping.result.MaxDuration {
						tcping.result.MaxDuration = duration
					} else if duration < tcping.result.MinDuration {
						tcping.result.MinDuration = duration
					}
					tcping.result.TotalDuration += duration
				}
			case <-tcping.done:
				return
			}
		}
	}()
	return tcping.done
}

// Stop the tcping
func (tcping *TCPing) Stop() {
	tcping.done <- struct{}{}
}

func (tcping *TCPing) ping() (time.Duration, net.Addr, error) {
	var remoteAddr net.Addr
	duration, errIfce := timeIt(func() interface{} {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", tcping.target.Host, tcping.target.Port), tcping.target.Timeout)
		if err != nil {
			return err
		}
		remoteAddr = conn.RemoteAddr()
		conn.Close()
		return nil
	})
	if errIfce != nil {
		err := errIfce.(error)
		return 0, remoteAddr, err
	}
	return time.Duration(duration), remoteAddr, nil
}
