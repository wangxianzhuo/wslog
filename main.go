package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/wangxianzhuo/logrus-conf"

	log "github.com/sirupsen/logrus"
	"github.com/wangxianzhuo/wslog/server"
)

var addr string
var kafkaBrokers, topic string
var kafkaBrokerList []string

func init() {
	flag.StringVar(&addr, "server-address", ":9000", "服务监听地址")
	flag.StringVar(&kafkaBrokers, "kafka-brokers", "localhost:9092", "消息队列地址,例如 <addr1>, <addr2>,...,<addrn>")
	flag.StringVar(&topic, "kafka-topic", "log_msg", "kafka topic")
}

func main() {
	// 解析传入参数
	flag.Parse()
	parseKafkaBrokerList()
	logconf.Configure()
	logconf.ConfigureLocalFileHook()

	http.HandleFunc("/log", logServer)
	log.Infof("server %v start", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func logServer(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	filterMap := make(map[string]interface{})

	for k, v := range r.Form {
		if k == "" {
			continue
		}

		filterMap[k] = v
	}

	server.ServeWs(w, r, log.WithField("websocket from", r.RemoteAddr), server.KafkaOpt{
		Topic:   topic,
		Brokers: kafkaBrokerList,
	}, filterMap)
}

func parseKafkaBrokerList() {
	if kafkaBrokers == "" {
		fmt.Println("Error: --kafka-brokers参数不能为空")
		flag.PrintDefaults()
		os.Exit(1)
	}
	kafkaBrokerList = strings.Split(kafkaBrokers, ",")
	if len(kafkaBrokerList) == 0 {
		fmt.Println("Error: --kafka-brokers参数不能为空")
		flag.PrintDefaults()
		os.Exit(1)
	}
	for i, broker := range kafkaBrokerList {
		kafkaBrokerList[i] = strings.TrimSpace(broker)
	}
}
