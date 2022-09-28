package net

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
	"websocket_server/util"
)

var CurrentServer *Server

type Server struct {
	users    *util.Array  // 用户列表
	rooms    *util.Array  // 房间列表
	matchs   *Matchs      // 匹配管理
	usersSQL *UserDataSQL // 用户数据库，管理已注册、登陆的用户基础信息
}

func (s *Server) InitServer() {
	CurrentServer = s
	s.matchs = &Matchs{
		matchUsers:      util.CreateArray(),
		matchUsersGroup: util.CreateArray(),
	}
	s.users = util.CreateArray()
	s.rooms = util.CreateArray()
	s.usersSQL = &UserDataSQL{
		users: map[string]*RegisterUserData{},
	}
}

// 开始侦听WebSocket服务器（ws）
func (s *Server) Listen(ip string, port int) {
	s.InitServer()
	fmt.Println("[WS]Server start:" + ip + ":" + fmt.Sprint(port))
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

// 开始侦听TLS协议WebSocket服务器（wss）
func (s *Server) ListenTLS(ip string, port int) {
	s.InitServer()
	c, e := tls.LoadX509KeyPair("tls.pem", "tls.key")
	if e != nil {
		panic(e)
	}
	config := &tls.Config{Certificates: []tls.Certificate{c}}
	fmt.Println("[WSS]Server start:" + ip + ":" + fmt.Sprint(port))
	n, e := tls.Listen("tcp", ip+":"+fmt.Sprint(port), config)
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
func (s *Server) CreateRoom(user *Client, option RoomConfigOption) *Room {
	if user.room != nil {
		return nil
	}

	create_uid := 1
	for _, v := range s.rooms.List {
		room := v.(*Room)
		if room.id >= create_uid {
			create_uid = room.id + 1
		}
	}

	interval := 1. / 30.
	interval = float64(time.Second) * interval

	// 如果房间没有定义最大人数，则默认为10个
	if option.maxCounts == 0 {
		option.maxCounts = 10
	} else if option.maxCounts > 100 {
		option.maxCounts = 100
	}

	room := Room{
		id:        create_uid,
		master:    user,
		interval:  time.Duration(interval),
		option:    &option,
		users:     util.CreateArray(),
		oldMsgs:   util.CreateArray(),
		userState: map[int]*ClientState{},
		roomState: &ClientState{
			Data: util.CreateMap(),
		},
		customData: util.CreateMap(),
	}
	s.rooms.Push(&room)
	room.JoinClient(user)
	return &room
}

// 加入房间
func (s *Server) JoinRoom(user *Client, roomid int, password string) (*Room, bool) {
	// 如果用户已经在房间中，则无法继续加入
	if user.room != nil {
		return nil, false
	}
	for _, v := range s.rooms.List {
		room := v.(*Room)
		if room.id == roomid {
			// 要处于非锁定、人数足够、密码验证通过才能加入
			if !room.lock && room.users.Length() < room.option.maxCounts && room.option.password == password {
				room.JoinClient(user)
			} else {
				return nil, false
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

// 匹配房间
func (s *Server) MatchRoom(c *Client) (*Room, bool) {
	if c.room == nil {
		for _, v := range s.rooms.List {
			r := v.(*Room)
			if r.matchOption != nil && r.matchOption.matchClient(c) {
				// 匹配房间不会去匹配带密码的房间
				r2, b2 := s.JoinRoom(c, r.id, "")
				if b2 {
					return r2, b2
				}
			}
		}
	}
	return nil, false
}
