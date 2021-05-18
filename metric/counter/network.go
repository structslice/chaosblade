package counter

import (
	"context"
	"github.com/sirupsen/logrus"
)

func NetworkDelay(ctx context.Context, args map[string]string) (metricdatas []*MetricData) {
	var destination_ip, remote_port string
	if dst_ip, ok := args["--destination-ip"]; ok {
		destination_ip = dst_ip
	} else {
		logrus.Error("collect network delay metric faild, err: loss --destination-ip params")
		return
	}
	if rt_port, ok := args["--remote-port"]; ok {
		remote_port = rt_port
	}

	return metricdatas
}
