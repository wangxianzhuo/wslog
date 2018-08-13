package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/wangxianzhuo/wslog/kafka"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn *websocket.Conn

	send chan []byte
}

func (c *Client) run(logger *log.Entry, kafkaOpt KafkaOpt, key, value string) {
	dataChan := make(chan []byte)
	consumer, err := kafka.New(kafkaOpt.Brokers, kafkaOpt.Topic, dataChan)
	if err != nil {
		logger.Errorf("新建kafka消费者失败: %v", err)
		return
	}
	defer consumer.Close(logger)
	logger.Infof("监听%v的topic[%v]", kafkaOpt.Brokers, kafkaOpt.Topic)

	go consumer.Start(logger)

	for {
		select {
		case msg := <-dataChan:
			if ok, err := c.filter(msg, key, value); ok && err == nil {
				c.send <- msg
				logger.Infof("%v", string(msg))
			} else if err != nil {
				logger.Warnf("处理消息%v异常: %v", string(msg), err)
			}
		}
	}
}

func (c *Client) sendMsg() {
	for {
		select {
		case d := <-c.send:
			c.conn.WriteMessage(websocket.TextMessage, d)
		}
	}
}

func (c *Client) filter(data []byte, key, value string) (bool, error) {
	var msg map[string]string

	err := json.Unmarshal(data, &msg)
	if err != nil {
		return false, fmt.Errorf("过滤消息%v异常: %v", string(data), err)
	}

	if key == "" {
		return true, nil
	}

	if v, ok := msg[key]; ok && (value == "" || v == value) {
		return true, nil
	}
	return false, nil
}

// ServeWs ...
func ServeWs(w http.ResponseWriter, r *http.Request, logger *log.Entry, kafkaOpt KafkaOpt, filterOpt FilterOpt) {
	defer logger.Infof("websocket connection from %v closed", r.RemoteAddr)
	if logger == nil {
		logger = log.New().WithField("websocket from", r.RemoteAddr)
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("websocket error: %v", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}

	client.run(logger, kafkaOpt, filterOpt.Key, filterOpt.Value)
}
