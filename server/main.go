package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/wangxianzhuo/logrus-conf"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/wangxianzhuo/wslog/server/server"
)

var addr string
var kafkaBrokers, topic, kafkaClientID string
var kafkaBrokerList []string

func init() {
	flag.StringVar(&addr, "server-address", ":9000", "服务监听地址")
	flag.StringVar(&kafkaBrokers, "kafka-brokers", "localhost:9092", "消息队列地址,例如 <addr1>, <addr2>,...,<addrn>")
	flag.StringVar(&kafkaClientID, "kafka-client-id", "wslog_server", "kafka消费者客户端编码")
}

func main() {
	// 解析传入参数
	flag.Parse()
	parseKafkaBrokerList()
	logconf.Configure()
	logconf.ConfigureLocalFileHook()

	r := mux.NewRouter()
	r.HandleFunc("/ws/log/{topic}", logServer).Methods("GET")
	r.HandleFunc("/log/topics", getAllLogTopics).Methods("GET")
	http.Handle("/", r)
	log.Infof("wslog server %v start", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func logServer(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	filterMap := make(map[string]interface{})

	for k, v := range r.Form {
		if k == "" {
			continue
		}

		if len(v) > 0 {
			filterMap[k] = v[0]
		}
	}

	vars := mux.Vars(r)
	t, ok := vars["topic"]
	if !ok {
		log.Debugf("no topic, use default[%v]", topic)
		t = topic
	}

	if !checkTopic(t) {
		fmt.Fprintf(w, "%v", errorMsg("传入非法topic: "+t))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	server.ServeWs(w, r, log.WithField("websocket from", r.RemoteAddr), server.KafkaOpt{
		Topic:    t,
		Brokers:  kafkaBrokerList,
		ClientID: kafkaClientID,
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

func getAllLogTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := server.GetTopics(kafkaBrokerList)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", errorMsg("获取topics出错"))
		log.Errorf("获取topics异常: %v", err)
		return
	}
	checkedTopics := make([]string, 0)
	for _, t := range topics {
		if checkTopic(t) {
			checkedTopics = append(checkedTopics, t)
		}
	}
	t, err := json.Marshal(checkedTopics)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", errorMsg("打包topics出错"))
		log.Errorf("打包topics[%v]异常: %v", topics, err)
		return
	}
	fmt.Fprintf(w, "%v", string(t))
	w.WriteHeader(http.StatusOK)
}

func errorMsg(msg string) string {
	m, _ := json.Marshal(map[string]string{"msg": msg})
	return string(m)
}

func checkTopic(topic string) bool {
	if len(topic) >= 3 && topic[:len("log")] == "log" {
		return true
	}
	return false
}
