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
	// Time allowed to read a message from the peer.
	readWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	logTimeout = 30 * time.Minute

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

// Client ...
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
			err = c.sendExec(dSend)
			if err != nil {
				c.logger.Errorf("发送message[%v]异常: %v", string(dSend), err)
				return
			}
		}
	}
}

func (c *Client) filter(data []byte, filterMap map[string]interface{}) (bool, error) {
	var msg map[string]interface{}

	err := json.Unmarshal(data, &msg)
	if err != nil {
		return false, fmt.Errorf("过滤消息%v异常: %v", string(data), err)
	}

	if len(filterMap) < 1 {
		return true, nil
	}

	// c.logger.Debugf("data: %v", string(data))
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
	ctxx, cancel := context.WithDeadline(context.Background(), time.Now().Add(logTimeout))
	defer cancel()

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		ctx:    ctxx,
		logger: logger,
	}
	defer client.Close()

	go client.sendMsg(cancel)
	go client.ping(cancel)
	// go client.pong(cancel)

	client.logger.Debug("websocket 开启")
	client.run(kafkaOpt, filterMap)
}

func (c *Client) pong(cancel context.CancelFunc) {
	defer cancel()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debugf("pong 停止")
			return
		default:
		}
		message, err := c.receiveExec()
		if err != nil {
			c.logger.Error("接收pong异常: ", err)
			return
		}
		if bytes.Compare(message, []byte("pong")) == 0 {
			c.logger.Debugf("收到pong")
		} else if bytes.Compare(message, []byte("ping")) == 0 {
			c.logger.Debugf("收到ping")
			if err := c.sendExec([]byte("pong")); err != nil {
				c.logger.Error("发送pong异常: ", err)
				cancel()
				return
			}
			c.logger.Debugf("发送pong")
		}
	}
}

func (c *Client) ping(cancel context.CancelFunc) {
	ticker := time.NewTicker(pingPeriod)
	defer c.logger.Debugf("ping 停止")
	defer ticker.Stop()
	defer cancel()

	for {
		select {
		case <-ticker.C:
			if err := c.pingExec(cancel); err != nil {
				c.logger.Debugf("%v", err)
				return
			}
			_, err := c.receiveExec()
			if err != nil {
				c.logger.Error("接收pong异常: ", err)
				return
			}
			c.logger.Debugf("接收pong")
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) pingExec(cancel context.CancelFunc) error {
	err := c.sendExec([]byte("ping"))
	if err != nil {
		return fmt.Errorf("发送ping异常: %v", err)
	}
	c.logger.Debugf("发送ping")
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	return nil
}

func (c *Client) sendExec(d []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := c.conn.WriteMessage(websocket.TextMessage, d)
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			return fmt.Errorf("websocket 关闭: %v", err)
		}
		return fmt.Errorf("发送message[%v]异常: %v", string(d), err)
	}
	c.conn.SetWriteDeadline(time.Unix(0, 0))
	return nil
}

func (c *Client) receiveExec() ([]byte, error) {
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("接收message异常: %v", err)
	}

	c.conn.SetReadDeadline(time.Unix(0, 0))
	return message, nil
}

// Close ...
func (c *Client) Close() {
	close(c.send)
	c.logger.Debugf("websocket 关闭")
}
