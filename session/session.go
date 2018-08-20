package session

import (
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	// 发送数据超时时长
	writeWait = 10 * time.Second
	// 等待接收ping帧的时长，超时则关闭连接
	pingWait = 60 * time.Second
)

// Session 会话
type Session struct {
	Ctx     context.Context
	Conn    *websocket.Conn
	send    chan []byte
	Logger  *log.Entry
	parser  Parser
	filters []Filter
}

// New 新建会话
func New(ctx context.Context, conn *websocket.Conn, logger *log.Entry) *Session {
	return &Session{
		Ctx:    ctx,
		Conn:   conn,
		send:   make(chan []byte, 256),
		Logger: logger,
		parser: LogParser{},
	}
}

// Close 关闭会话
func (s *Session) Close() {
	close(s.send)
	s.Conn = nil
	s.Logger.Debugf("websocket会话退出")
}

// Run 会话启动
func (s *Session) Run(h func() error) {
	s.Logger.Debugf("websocket会话启动")
	err := h()
	if err != nil {
		s.Logger.Errorf("websocket会话异常: %v", err)
		return
	}
}

// RegisterFilter 过滤器注册
func (s *Session) RegisterFilter(filter Filter) {
	s.filters = append(s.filters, filter)
}

// SendMessage 发送ws消息
func (s *Session) SendMessage(cancel context.CancelFunc) {
	defer cancel()

	for {
		select {
		case <-s.Ctx.Done():
			return
		case d := <-s.send:
			dSend, err := s.parser.Parse(d)
			if err != nil {
				s.Logger.Errorf("解析message[%X]异常: %v", d, err)
				continue
			}
			err = s.sendExec(dSend)
			if err != nil {
				s.Logger.Errorf("发送message[%X]异常: %v", dSend, err)
				return
			}
		}
	}
}

// Filter 执行过滤
func (s *Session) Filter(data []byte, strategy FilterStrategy) bool {
	if len(s.filters) < 1 {
		return true
	}

	for _, f := range s.filters {
		if ok, err := f.Filter(data, strategy); !ok || err != nil {
			return false
		}
	}

	return true
}

// ReadLoop 监听会话状态
func (s *Session) ReadLoop(cancel context.CancelFunc) {
	defer cancel()
	for {
		select {
		case <-s.Ctx.Done():
			return
		default:
		}

		if s.Conn == nil {
			s.Logger.Errorf("websocket已关闭")
			return
		}

		if _, _, err := s.Conn.NextReader(); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				s.Logger.Debugf("websocket被客户端关闭")
				return
			}
			s.Logger.Errorf("websocket异常: %v", err)
			return
		}
	}
}

// Send 发送数据
func (s *Session) Send() chan []byte {
	return s.send
}

func (s *Session) sendExec(d []byte) error {
	if s.Conn == nil {
		return fmt.Errorf("websocket已关闭")
	}
	s.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := s.Conn.WriteMessage(websocket.TextMessage, d)
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			return fmt.Errorf("websocket 关闭: %v", err)
		}
		return fmt.Errorf("%v", err)
	}
	s.Logger.Debugf("发送message[%v]", string(d))
	s.Conn.SetWriteDeadline(time.Time{})
	return nil
}
