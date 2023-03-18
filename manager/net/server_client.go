package net

import (
	"fmt"
	"net"
	"strings"
	"time"
	"websocket_server/logs"
	"websocket_server/runtime"
	"websocket_server/util"
	"websocket_server/websocket"
)

// 与每个服务器建立联系的客户端
type ServerClient struct {
	net.Conn
	ip    string
	port  int
	state websocket.State
	data  *util.Bytes
}

// 服务器联系逻辑
func onServerClient(s *ServerClient) {
	defer runtime.GoRecover()
	logs.InfoM("ServerClient create:" + s.ip + ":" + fmt.Sprint(s.port))
	for {
		// 预判conn连接
		if s.Conn == nil {
			// 进行连接
			conn, err := net.Dial("tcp", s.ip+":"+fmt.Sprint(s.port))
			if err != nil {
				// 连接错误，1秒后重连
				time.Sleep(time.Second)
			} else {
				// 连接成功
				s.Conn = conn
				// Sec-WebSocket-Key是由16个随机字符得到的base64
				key := websocket.CreateSecWebSocketKey()
				// 发送握手信息
				content := []string{
					"GET / HTTP/1.1",
					"Host: manager://",
					"Upgrade: websocket",
					"Connection: Upgrade",
					"Sec-WebSocket-Key: " + key,
					"Sec-WebSocket-Version: 13",
				}
				// 发送握手事件
				contentData := strings.Join(content, "\r\n") + "\r\n\r\n"
				s.Conn.Write([]byte(contentData))
				logs.InfoM("listene server " + s.ip + ":" + fmt.Sprint(s.port))
			}
		} else {
			// 数据读取
			var bytes [126]byte
			n, err := s.Conn.Read(bytes[:])
			if err == nil {
				if n != 0 {
					// 将数据缓存，进行读取使用
					s.data.WriteBytes(bytes[0:n])
					s.onMessage()
				}
			} else {
				// 与服务器断开了连接，设置为nil，使服务侦听进行重连处理
				logs.InfoM("服务器已断开，准备重试重连")
				s.Conn.Close()
				s.data = util.CreateBytes()
				s.state = websocket.Handshake
				s.Conn = nil
			}
		}
	}
}

// 绑定指定的接口侦听
func bind(ip string, port int) *ServerClient {
	s := &ServerClient{
		state: websocket.Handshake,
		data:  util.CreateBytes(),
	}
	s.ip = ip
	s.port = port
	go onServerClient(s)
	return s
}
