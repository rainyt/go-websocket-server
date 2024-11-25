package websocketv2

import (
	"log"
	"time"
	"websocket_server/logs"
	"websocket_server/runtime"
	"websocket_server/util"

	"github.com/gorilla/websocket"
)

// 消息数据
type MessageByte struct {
	callback chan int
	data     []byte
}

const (
	// 允许向对等方写入消息的时间。
	writeWait = 30 * time.Second

	// 读取来自对等方的下一条pong消息的时间。
	pongWait = 60 * time.Second

	// 使用此周期向对等点发送ping。必须小于pong等待。
	pingPeriod = (pongWait * 9) / 10

	// 对等方允许的最大消息大小。
	maxMessageSize = 0
)

// WebSocket包装器
type WebSocket struct {
	// WebSocket连接器
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan *MessageByte

	Connected bool

	FrameIsBinary bool

	// 是否已经关闭通道
	isClosed bool

	// 是否已经释放
	isReleased bool

	// 是否已经取消注册过程中
	isUnregister bool

	userData map[string]any

	frames *util.Array

	OnUserOutCallback func()

	OnWorkData func(data []byte)
}

// 发送二进制数据
func (client *WebSocket) SendBytes(data []byte) {
	client.send <- &MessageByte{
		callback: nil,
		data:     data,
	}
}

func CreateWebSocketClient(conn *websocket.Conn) *WebSocket {
	client := &WebSocket{conn: conn, send: make(chan *MessageByte, 256), Connected: true, isClosed: false,
		isReleased: false, isUnregister: false, FrameIsBinary: false,
		userData: map[string]any{}, frames: util.CreateArray()}
	go client.readMessage()
	go client.writeMessage()
	return client
}

func (c *WebSocket) readMessage() {
	defer runtime.GoRecover()
	defer func() {
		if !c.isClosed {
			c.isClosed = true
			data := &RegisterClient{
				Client: c,
			}
			SERVER_HUB.unregister <- data
			c.conn.Close()
		}
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		var data []byte
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				logs.ErrorF("error: %v", err)
			} else {
				logs.ErrorF("error: %v", err)
			}
			break
		}
		// 处理客户端数据
		c.onWork(data)
	}
}

func (c *WebSocket) writeMessage() {
	ticker := time.NewTicker(pingPeriod)
	defer runtime.GoRecover()
	defer func() {
		if !c.isClosed {
			c.isClosed = true
			ticker.Stop()
			data := &RegisterClient{
				Client: c,
			}
			SERVER_HUB.unregister <- data
			c.conn.Close()
		}
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				logs.InfoM("socket closed.")
				if message.callback != nil {
					message.callback <- 1
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				if message.callback != nil {
					message.callback <- 1
				}
				log.Panicln("write err", err)
				return
			}
			w.Write(message.data)

			if message.callback != nil {
				message.callback <- 0
			}

			// Add queued chat messages to the current websocket message.
			// n := len(c.send)
			// for i := 0; i < n; i++ {
			// 	w.Write(newline)
			// 	w.Write(<-c.send)
			// }

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WebSocket) onWork(data []byte) {
	c.OnWorkData(data)
}

func (c *WebSocket) Close() {
	// TODO 断线处理
	c.conn.Close()
}
