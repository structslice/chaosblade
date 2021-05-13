/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chaosblade-io/chaosblade/metric"
	"github.com/chaosblade-io/chaosblade/metric/counter"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/channel"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const startServerKey = "blade server start --nohup"
const APIPublicKey = "devops.autohome.cc.chaosblade.io"
const APIPublicIV = "autohome.com.cn."

type StartServerCommand struct {
	baseCommand
	ip    string
	port  string
	nohup bool
}

func (ssc *StartServerCommand) Init() {
	ssc.command = &cobra.Command{
		Use:     "start",
		Short:   "Start server mode, exposes web services",
		Long:    "Start server mode, exposes web services. Under the mode, you can send http request to trigger experiments",
		Aliases: []string{"s"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return ssc.run(cmd, args)
		},
		Example: startServerExample(),
	}
	ssc.command.Flags().StringVarP(&ssc.ip, "ip", "i", "", "service ip address, default value is *")
	ssc.command.Flags().StringVarP(&ssc.port, "port", "p", "9526", "service port")
	ssc.command.Flags().BoolVarP(&ssc.nohup, "nohup", "n", false, "used by internal")
}

func (ssc *StartServerCommand) run(cmd *cobra.Command, args []string) error {
	// check if the process named `blade server --start` exists or not
	pids, err := channel.NewLocalChannel().GetPidsByProcessName(startServerKey, context.TODO())
	if err != nil {
		return spec.ReturnFail(spec.Code[spec.ServerError], err.Error())
	}
	if len(pids) > 0 {
		return spec.ReturnFail(spec.Code[spec.DuplicateError], "the chaosblade has been started. If you want to stop it, you can execute blade server stop command")
	}
	if ssc.nohup {
		ssc.start0()
	}
	err = ssc.start()
	if err != nil {
		return err
	}
	cmd.Println(fmt.Sprintf("success, listening on %s:%s", ssc.ip, ssc.port))
	return nil
}

// start used nohup command and check the process
func (ssc *StartServerCommand) start() error {
	// use nohup to invoke blade server start command
	cl := channel.NewLocalChannel()
	bladeBin := path.Join(util.GetProgramPath(), "blade")
	args := fmt.Sprintf("%s server start --nohup --port %s", bladeBin, ssc.port)
	if ssc.ip != "" {
		args = fmt.Sprintf("%s --ip %s", args, ssc.ip)
	}
	fmt.Println(args)
	response := cl.Run(context.TODO(), "nohup", fmt.Sprintf("%s > /dev/null 2>&1 &", args))
	if !response.Success {
		return response
	}
	time.Sleep(time.Second)
	// check process
	pids, err := channel.NewLocalChannel().GetPidsByProcessName(startServerKey, context.TODO())
	if err != nil {
		return spec.ReturnFail(spec.Code[spec.ServerError], err.Error())
	}
	if len(pids) == 0 {
		// read logs
		logFile, err := util.GetLogFile(util.Blade)
		if err != nil {
			return spec.ReturnFail(spec.Code[spec.ServerError], "start blade server failed and can't get log file")
		}
		if !util.IsExist(logFile) {
			return spec.ReturnFail(spec.Code[spec.ServerError], "start blade server failed and log file does not exist")
		}
		response := cl.Run(context.TODO(), "tail", fmt.Sprintf("-1 %s", logFile))
		if !response.Success {
			return spec.ReturnFail(spec.Code[spec.ServerError], "start blade server failed and can't read log file")
		}
		return spec.ReturnFail(spec.Code[spec.ServerError], response.Result.(string))
	}
	logrus.Infof("start blade server success, listen on %s:%s", ssc.ip, ssc.port)
	//log.Info("start blade server success", "ip", ssc.ip, ssc.port)
	return nil
}

// start0 starts web service
func (ssc *StartServerCommand) start0() {
	go metric.MetricCollector.Start()
	go func() {
		err := http.ListenAndServe(ssc.ip+":"+ssc.port, nil)
		if err != nil {
			logrus.Errorf("start blade server error, %v", err)
			//log.Error(err, "start blade server error")
			os.Exit(1)
		}
	}()
	Register("/chaosblade")
	RegisterHealthRouter()
	RegisterMetricCollect()
	util.Hold()
}

func Register(requestPath string) {
	http.HandleFunc(requestPath, func(writer http.ResponseWriter, request *http.Request) {
		err := request.ParseForm()
		if err != nil {
			fmt.Fprintf(writer,
				spec.ReturnFail(spec.Code[spec.IllegalParameters], err.Error()).Print())
			return
		}
		ts := request.Header.Get("ts")
		token := request.Header.Get("token")
		if len(ts) == 0 || len(token) == 0 {
			fmt.Fprintf(writer,
				spec.ReturnFail(spec.Code[spec.IllegalParameters], "illegal ts or token parameter").Print())
			return
		}
		if !Auth(ts, token) {
			fmt.Fprintf(writer,
				spec.ReturnFail(spec.Code[spec.Forbidden], "authentication faild").Print())
			return
		}

		cmds := request.Form["cmd"]
		if len(cmds) != 1 {
			fmt.Fprintf(writer,
				spec.ReturnFail(spec.Code[spec.IllegalParameters], "illegal cmd parameter").Print())
			return
		}
		ctx := context.WithValue(context.Background(), "mode", "server")
		response := channel.NewLocalChannel().Run(ctx, path.Join(util.GetProgramPath(), "blade"), cmds[0])
		if response.Success {
			fmt.Fprintf(writer, response.Result.(string))
		} else {
			fmt.Fprintf(writer, response.Err)
		}
	})
}
func RegisterHealthRouter() {
	http.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprint(writer, "ok")
		return
	})
}

func Auth(ts, token string) bool {
	//dt_ts, err := strconv.ParseInt(ts, 10, 64)
	//if err != nil {
	//	return false
	//}
	//if time.Now().Unix()-dt_ts > 60 {
	//	return false
	//}
	AesKey := []byte(APIPublicKey)
	AesIV := []byte(APIPublicIV)
	origin, err := AesDecrypt([]byte(token), AesKey, AesIV)
	if err == nil && string(origin) == ts {
		return true
	}
	return false
}

func RegisterMetricCollect() {
	http.HandleFunc("/metric-collect", func(writer http.ResponseWriter, request *http.Request) {
		ts := request.Header.Get("ts")
		token := request.Header.Get("token")
		if len(ts) == 0 || len(token) == 0 {
			fmt.Fprintf(writer,
				spec.ReturnFail(spec.Code[spec.IllegalParameters], "illegal ts or token parameter").Print())
			return
		}
		if !Auth(ts, token) {
			fmt.Fprintf(writer,
				spec.ReturnFail(spec.Code[spec.Forbidden], "authentication faild").Print())
			return
		}
		// 新建采集
		if request.Method == "GET" {
			fmt.Fprintf(writer,
				spec.ReturnSuccess(metric.MetricCollector.CurrentWorker()).Print())
			return

		} else {
			err := request.ParseForm()
			if err != nil {
				fmt.Fprintf(writer,
					spec.ReturnFail(spec.Code[spec.IllegalParameters], err.Error()).Print())
				return
			}
			json_body, _ := ioutil.ReadAll(request.Body)
			fmt.Println(string(json_body))
			var body counter.Indicator
			if err := json.Unmarshal(json_body, &body); err != nil {
				fmt.Fprintf(writer,
					spec.ReturnFail(spec.Code[spec.IllegalParameters], "request body is invaild, json Unmarshal faild "+err.Error()).Print())
				return
			}
			if len(body.Metric) == 0 {
				fmt.Fprintf(writer,
					spec.ReturnFail(spec.Code[spec.IllegalParameters], "request body loss metric ").Print())
				return
			}
			//if len(body.Args) == 0 {
			//	fmt.Fprintf(writer,
			//		spec.ReturnFail(spec.Code[spec.IllegalParameters], "request body loss args ").Print())
			//	return
			//
			//}
			if len(body.Interval) == 0 {
				body.Interval = "5s"
			}
			if request.Method == "POST" {
				if metric.MetricCollector.Exists(&body) {
					fmt.Fprintf(writer,
						spec.ReturnFail(spec.Code[spec.DuplicateError], "collect worker is already exists").Print())
					return
				} else {
					metric.MetricCollector.Send(&body)
				}

				fmt.Fprintf(writer,
					spec.ReturnSuccess("start collect success").Print())
			} else if request.Method == "DELETE" {
				err := metric.MetricCollector.StopCollector(&body)
				if err != nil {
					fmt.Fprintf(writer,
						spec.ReturnFail(spec.Code[spec.IllegalCommand], "collect worker not found").Print())
					return
				}
				fmt.Fprintf(writer,
					spec.ReturnSuccess("stop collect success").Print())
			} else {
				fmt.Fprintf(writer,
					spec.ReturnFail(spec.Code[spec.ServerError], "405 not support method").Print())
			}
		}

		return
	})
}

func startServerExample() string {
	return `blade server start --port 8000`
}
