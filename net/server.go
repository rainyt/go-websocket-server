package net

import (
	"fmt"
	"net"
	"time"
	"websocket_server/util"
)

var CurrentServer *Server

type Server struct {
	users      util.Array  // 用户列表
	rooms      util.Array  // 房间列表
	create_uid int         // 房间的创建ID
	usersSQL   UserDataSQL // 用户数据库，管理已注册、登陆的用户基础信息
}

// 开始侦听服务器
func (s *Server) Listen(ip string, port int) {
	CurrentServer = s
	fmt.Println("Server start:" + ip + ":" + fmt.Sprint(port))
	n, e := net.Listen("tcp", ip+":"+fmt.Sprint(port))
	if e != nil {
		fmt.Println(e.Error())
	}
	for {
		c, e := n.Accept()
		if e == nil {
			// 将用户写入到用户列表中
			s.users.Push(CreateClient(c))
		}
	}
}

// 创建房间
func (s *Server) CreateRoom(user *Client) *Room {
	if user.room != nil {
		return nil
	}
	s.create_uid++
	interval := 1. / 30.
	interval = float64(time.Second) * interval
	room := Room{
		id:        s.create_uid,
		master:    user,
		interval:  time.Duration(interval),
		maxCounts: 10,
	}
	s.rooms.Push(&room)
	room.JoinClient(user)
	return &room
}

// 加入房间
func (s *Server) JoinRoom(user *Client, roomid int) (*Room, bool) {
	// 如果用户已经在房间中，则无法继续加入
	if user.room != nil {
		return nil, false
	}
	for _, v := range s.rooms.List {
		room := v.(*Room)
		if room.id == roomid {
			if room.users.Length() < room.maxCounts {
				room.JoinClient(user)
			}
			return room, true
		}
	}
	return nil, false
}

// 退出房间
func (s *Server) ExitRoom(c *Client) {
	if c.room != nil {
		room := c.room
		c.room.ExitClient(c)
		if room.users.Length() == 0 {
			s.rooms.Remove(room)
		}
	}
}
