package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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

	ctx context.Context

	logger *log.Entry
}

func (c *Client) run(kafkaOpt KafkaOpt, filterMap map[string]interface{}) {
	dataChan := make(chan []byte)
	consumer, err := kafka.New(kafkaOpt.Brokers, kafkaOpt.Topic, dataChan)
	if err != nil {
		c.logger.Errorf("新建kafka消费者失败: %v", err)
		return
	}
	defer consumer.Close(c.logger)
	c.logger.Infof("监听%v的topic[%v]", kafkaOpt.Brokers, kafkaOpt.Topic)

	go consumer.Start(c.logger)

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-dataChan:
			if ok, err := c.filter(msg, filterMap); ok && err == nil {
				c.send <- msg
				c.logger.Debugf("获取消息: %v", string(msg))
			} else if err != nil {
				c.logger.Warnf("处理消息%v异常: %v", string(msg), err)
			}
		}
	}
}

func (c *Client) sendMsg(cancel context.CancelFunc) {
	defer cancel()

	for {
		select {
		case <-c.ctx.Done():
			return
		case d := <-c.send:
			dSend, err := c.parseLog(d)
			if err != nil {
				c.logger.Errorf("解析message[%v]异常: %v", string(d), err)
				continue
			}
			// c.logger.Debugf("发送日志: {%v}", strings.TrimSuffix(string(dSend), "\n"))
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err = c.conn.WriteMessage(websocket.TextMessage, dSend)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					c.logger.Errorf("websocket 关闭: %v", err)
					return
				}
				c.logger.Errorf("发送message[%v]异常: %v", string(dSend), err)
				continue
			}
		}
	}
}

func (c *Client) filter(data []byte, filterMap map[string]interface{}) (bool, error) {
	// func (c *Client) filter(data []byte, key string, value interface{}) (bool, error) {
	var msg map[string]interface{}

	err := json.Unmarshal(data, &msg)
	if err != nil {
		return false, fmt.Errorf("过滤消息%v异常: %v", string(data), err)
	}

	if len(filterMap) < 1 {
		return false, nil
	}

	for k, v := range filterMap {
		if value, ok := msg[k]; ok && (value == nil || v == value) {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) parseLog(data []byte) ([]byte, error) {
	var e map[string]interface{}
	json.Unmarshal(data, &e)
	// c.logger.Debugf("获得logrus的entry%v", e)
	var host, l, msg, logTime string
	if v, ok := e["host"]; ok {
		host = v.(string)
		delete(e, "host")
	}
	if _, ok := e["topic"]; ok {
		delete(e, "topic")
	}
	if v, ok := e["level"]; ok {
		l = v.(string)
		delete(e, "level")
	}
	if v, ok := e["msg"]; ok {
		msg = v.(string)
		delete(e, "msg")
	}
	if v, ok := e["time"]; ok {
		logTime = v.(string)
		delete(e, "time")
	}

	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buffer := bytes.Buffer{}
	for _, k := range keys {
		buffer.WriteString(k)
		buffer.WriteString("=")

		switch v := e[k].(type) {
		case string:
			buffer.WriteString(v)
		case error:
			buffer.WriteString(v.Error())
		default:
			fmt.Fprint(&buffer, v)
		}
		buffer.WriteString(" ")
	}
	var level string
	switch l {
	case "info":
		level = "INFO"
	case "warning":
		level = "WARN"
	case "debug":
		level = "DEBU"
	case "error":
		level = "ERRO"
	case "panic":
		level = "PANI"
	case "fatal":
		level = "FATA"
	default:
		level = "UNKNOWN"
	}
	line := fmt.Sprintf("%s\t[%v]\t[%s]\t%-80s\t%s\n", level, logTime, host, msg, buffer.String())

	return []byte(line), nil
}

// ServeWs ...
func ServeWs(w http.ResponseWriter, r *http.Request, logger *log.Entry, kafkaOpt KafkaOpt, filterMap map[string]interface{}) {
	defer logger.Infof("websocket connection from %v closed", r.RemoteAddr)
	logger.Debugf("过滤配置: %v", filterMap)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("websocket error: %v", err)
		return
	}
	defer conn.Close()
	ctxx, cancel := context.WithCancel(context.Background())

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		ctx:    ctxx,
		logger: logger,
	}
	defer client.Close()

	go client.sendMsg(cancel)
	go client.ping(cancel)
	go client.pong(cancel)

	client.logger.Debug("websocket 开启")
	client.run(kafkaOpt, filterMap)
}

func (c *Client) pong(cancel context.CancelFunc) {
	defer cancel()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.logger.Error("pong error: ", err)
			return
		}
		if bytes.Compare(message, []byte("pong")) == 0 {
			c.conn.SetReadDeadline(time.Now().Add(pongWait))
			c.logger.Debugf("pong")
		}
	}
}

func (c *Client) ping(cancel context.CancelFunc) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer cancel()
	for {
		select {
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			// if err := c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
			// 	c.logger.Error("ping: ", err)
			// 	cancel()
			// 	return
			// }
			if err := c.conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				c.logger.Error("ping error: ", err)
				cancel()
				return
			}
		case <-c.ctx.Done():
			c.logger.Debugf("ping 停止")
			return
		}
	}
}

// Close ...
func (c *Client) Close() {
	close(c.send)
	c.logger.Debugf("websocket 关闭")
}
