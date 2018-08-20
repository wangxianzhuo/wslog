package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/wangxianzhuo/wslog/kafka"
	"github.com/wangxianzhuo/wslog/session"
)

const (
	// 发送数据超时时长
	writeWait = 10 * time.Second
	// 等待接收ping帧的时长，超时则关闭连接
	pingWait = 60 * time.Second
	// websocket连接最长允许时间
	logTimeout = 30 * time.Minute
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeWs ...
func ServeWs(w http.ResponseWriter, r *http.Request, logger *log.Entry, kafkaOpt KafkaOpt, filterMap map[string]interface{}) {
	defer logger.Infof("从[%v]的websocket连接关闭", r.RemoteAddr)
	logger.Debugf("过滤配置: %v", filterMap)

	// 创建ws连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("创建websocket连接异常: %v", err)
		return
	}
	defer conn.Close()

	// 设置ping帧处理handler
	conn.SetPingHandler(func(string) error {
		err := conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(writeWait))
		if err != nil {
			return err
		}
		conn.SetReadDeadline(time.Now().Add(pingWait))
		return nil
	})

	// 设置带超时的context
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(logTimeout))
	defer cancel()

	// 设置close帧的处理handler
	conn.SetCloseHandler(func(code int, text string) error { cancel(); return nil })

	// 初始化会话对象
	s := session.New(ctx, conn, logger)
	defer s.Close()

	// 注册过滤器
	s.RegisterFilter(session.LogFilter{
		FilterMap: filterMap,
	})

	// 启动ws数据发送线程
	go s.SendMessage(cancel)
	// 启动ws状态监听线程
	go s.ReadLoop(cancel)

	// 启动会话执行线程
	s.Run(func() error {
		dataChan := make(chan []byte)
		consumer, err := kafka.New(kafkaOpt.Brokers, kafkaOpt.Topic, dataChan)
		if err != nil {
			return fmt.Errorf("新建kafka消费者失败: %v", err)
		}
		defer consumer.Close(s.Logger)
		s.Logger.Infof("监听%v的topic[%v]", kafkaOpt.Brokers, kafkaOpt.Topic)

		go consumer.Start(s.Logger)

		for {
			select {
			case <-s.Ctx.Done():
				return nil
			case msg := <-dataChan:
				s.Logger.Debugf("准备发送消息: %v", string(msg))
				if s.Filter(msg, session.FilterStrategy{}) {
					s.Send() <- msg
				}
			}
		}
	})
}
