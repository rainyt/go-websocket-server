package net

import (
	"fmt"
	"net"
	"strings"
	"time"
	"websocket_server/util"
	"websocket_server/websocket"
)

// 与每个服务器建立联系的客户端
type ServerClient struct {
	conn net.Conn
	ip   string
	port int
}

// 服务器联系逻辑
func onServerClient(s *ServerClient) {
	util.Log("ServerClient create:" + s.ip + ":" + fmt.Sprint(s.port))
	for {
		// 预判conn连接
		if s.conn == nil {
			// 进行连接
			conn, err := net.Dial("tcp", s.ip+":"+fmt.Sprint(s.port))
			if err != nil {
				// 连接错误，1秒后重连
				time.Sleep(time.Second)
			} else {
				// 连接成功
				s.conn = conn
				// Sec-WebSocket-Key是由16个随机字符得到的base64
				key := websocket.CreateSecWebSocketKey()
				// 发送握手信息
				content := []string{
					"GET /chat HTTP/1.1",
					"Host: manager://",
					"Upgrade: websocket",
					"Connection: Upgrade",
					"Sec-WebSocket-Key: " + key,
					"Sec-WebSocket-Version: 13",
				}
				// 发送握手事件
				s.conn.Write([]byte(strings.Join(content, "\r\n") + "\r\n\r\n"))
				util.Log("ServerClient Handshake:" + key)
			}
		} else {
			// 数据读取
			var bytes []byte
			n, err := s.conn.Read(bytes)
			if err == nil {
				if n != 0 {
					// 将数据缓存，进行读取使用
				}
			} else {
				// 与服务器断开了连接，设置为nil，使服务侦听进行重连处理
				s.conn.Close()
				s.conn = nil
			}
		}
	}
}

// 绑定指定的接口侦听
func bind(ip string, port int) *ServerClient {
	s := &ServerClient{}
	s.ip = ip
	s.port = port
	go onServerClient(s)
	return s
}
