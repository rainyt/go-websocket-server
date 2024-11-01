package net

import (
	"crypto/tls"
	"fmt"
	"net"
	"reflect"
	"time"
	"websocket_server/logs"
	"websocket_server/util"
)

var CurrentServer *Server

// 空数据
type EmptyData struct{}

// 扩展方法绑定
type CallFunc struct {
	Api    any
	Method reflect.Method
}

// 调用接口
func (c *CallFunc) Call(client *Client, message *ClientMessage, data any) {
	if data == nil {
		data = EmptyData{}
	}
	args := []reflect.Value{
		reflect.ValueOf(c.Api),
		reflect.ValueOf(client),
		reflect.ValueOf(data),
	}
	v := c.Method.Func.Call(args)
	r := v[0].Interface()
	e := v[1].Interface()
	if e != nil {
		client.SendError(OP_ERROR, message.Op, e.(error).Error())
	} else if r != nil {
		client.SendToUserOp(&ClientMessage{
			Op:   message.Op,
			Data: r,
		})
	}
}

// 服务器方法
type Server struct {
	apps             *util.Map            // 所有应用的管理网
	ConnectCounts    int                  // 当前连接数
	MaxConnectCounts int                  // 当前服务器最大连接数
	ExtendsApi       map[string]*CallFunc // 扩展方法
	OnClosedApi      map[string]*CallFunc // 客户端关闭扩展方法
}

// 扩展注册
func (s *Server) Register(extendsApi any) {
	// 反射类型值
	val := reflect.ValueOf(extendsApi)
	// 反射类型
	t := reflect.TypeOf(extendsApi)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		t = t.Elem()
	}
	// 获得类型名称
	tName := t.Name()
	// 遍历所有方法
	for i := 0; i < val.NumMethod(); i++ {
		method := t.Method(i)
		id := tName + "." + method.Name
		logs.InfoM("Register:", id)
		switch method.Name {
		case "OnClosed":
			// 特定接口
			s.OnClosedApi[tName] = &CallFunc{
				Api:    extendsApi,
				Method: method,
			}
		default:
			// 注册方法
			s.ExtendsApi[id] = &CallFunc{
				Api:    extendsApi,
				Method: method,
			}
		}

	}
}

// 初始化服务器
func (s *Server) InitServer() {
	CurrentServer = s
	s.ExtendsApi = map[string]*CallFunc{}
	s.OnClosedApi = map[string]*CallFunc{}
	s.apps = util.CreateMap()
}

// 开始侦听WebSocket服务器（ws）
func (s *Server) Listen(ip string, port int) {
	fmt.Println("[WS]Server start:" + ip + ":" + fmt.Sprint(port))
	n, e := net.Listen("tcp", ip+":"+fmt.Sprint(port))
	if e != nil {
		fmt.Println(e.Error())
	}
	for {
		c, e := n.Accept()
		if e == nil {
			// TODO 当连接人数大于服务器最大限制人数后，直接中断
			// if s.ConnectCounts >= s.MaxConnectCounts {
			// 	c.Close()
			// 	return
			// }
			// s.ConnectCounts++
			// 创建客户端
			CreateClient(c)
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
			CreateClient(c)
			// s.users.Push(CreateClient(c))
		}
	}
}

// 追加用户
func (s *Server) PushUser(c *Client) {
	c.getApp().users.Push(c)
}

// 获取App对象
func (s *Server) getApp(appid string) *App {
	app := s.apps.GetData(appid, nil)
	if app == nil {
		app = &App{}
		app.(*App).initApp()
		s.apps.Store(appid, app)
	}
	return app.(*App)
}

type App struct {
	users        *util.Array  // 用户列表
	rooms        *util.Array  // 房间列表
	matchs       *Matchs      // 匹配管理
	usersSQL     *UserDataSQL // 用户数据库，管理已注册、登陆的用户基础信息
	msglist      *util.Array  // 全服消息列表
	msgListeners *util.Array  // 全服消息侦听列表
}

// 初始化App
func (s *App) initApp() {
	s.matchs = &Matchs{
		matchUsers:      util.CreateArray(),
		matchUsersGroup: util.CreateArray(),
	}
	s.users = util.CreateArray()
	s.rooms = util.CreateArray()
	s.msglist = util.CreateArray()
	s.msgListeners = util.CreateArray()
	s.usersSQL = &UserDataSQL{
		users: map[string]*RegisterUserData{},
	}
}

// 发送全服消息
func (s *App) SendServerMsg(user *Client, message *ClientMessage) {
	// 动态类型使用Object引用
	serMsg := map[string]any{
		"uid":  user.uid,
		"data": message.Data,
	}
	s.msglist.Push(util.CreateObject(serMsg))
	// 全服消息仅保留100条消息
	if s.msglist.Length() > 100 {
		s.msglist.Remove(s.msglist.List[0])
	}
	for _, v := range s.msgListeners.List {
		u := v.(*Client)
		if u != user {
			u.SendToUserOp(&ClientMessage{
				Op:   GetServerMsg,
				Data: serMsg,
			})
		}
	}
}

// 侦听全服消息
func (s *App) ListenerServerMsg(user *Client) {
	if s.msgListeners.IndexOf(user) == -1 {
		s.msgListeners.Push(user)
	}
}

// 取消侦听全服消息
func (s *App) CannelListenerServerMsg(user *Client) {
	s.msgListeners.Remove(user)
}

// 获取全部服消息列表
func (s *App) GetServerMsg(user *Client, newCounts int) {
	if newCounts > s.msglist.Length() {
		newCounts = s.msglist.Length()
	}
	logs.InfoM("newCounts=", newCounts)
	if newCounts == 0 {
		// 返回空消息
		return
	}
	startIndex := s.msglist.Length() - newCounts
	for i := startIndex; i < s.msglist.Length(); i++ {
		o := s.msglist.List[i].(*util.Object)
		user.SendToUserOp(&ClientMessage{
			Op:   GetServerMsg,
			Data: o.Data,
		})
	}

}

// 创建房间
func (s *App) CreateRoom(user *Client, option RoomConfigOption) *Room {
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
		frameDatas: util.CreateArray(),
	}
	s.rooms.Push(&room)
	room.JoinClient(user)
	return &room
}

// 加入房间
func (s *App) JoinRoom(user *Client, roomid int, password string) (*Room, error) {
	// 如果用户已经在房间中，则无法继续加入
	if user.room != nil {
		return nil, fmt.Errorf("已存在房间")
	}
	for _, v := range s.rooms.List {
		room := v.(*Room)
		if room.id == roomid {
			// 要处于非锁定、人数足够、密码验证通过才能加入
			if !room.lock && room.users.Length() < room.option.maxCounts && room.option.password == password {
				room.JoinClient(user)
			} else {
				return nil, fmt.Errorf("房间不匹配，无法进入")
			}
			return room, nil
		}
	}
	return nil, fmt.Errorf("无法找到" + fmt.Sprint(roomid) + "房间")
}

// 退出房间
func (s *App) ExitRoom(c *Client) {
	if c.room != nil {
		room := c.room
		c.room.ExitClient(c)
		if room.users.Length() == 0 {
			s.rooms.Remove(room)
		}
	}
}

// 匹配房间
func (s *App) MatchRoom(c *Client) (*Room, error) {
	if c.room == nil {
		for _, v := range s.rooms.List {
			r := v.(*Room)
			if r.matchOption != nil && r.matchOption.matchClient(c) {
				// 匹配房间不会去匹配带密码的房间
				r2, err := s.JoinRoom(c, r.id, "")
				if err == nil {
					return r2, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("已在房间内")
}

// 房间的基础信息
type RoomInfo struct {
	Id        int    `json:"id"`        // 房间id
	Counts    int    `json:"counts"`    // 当前人数
	MaxCounts int    `json:"maxCounts"` // 最大人数
	Password  bool   `json:"password"`  // 是否存在密码
	Master    string `json:"master"`    // 房主名称
	Lock      bool   `json:"lock"`      // 房间是否已锁定
}

// 获取指定范围的房间列表状态（仅返回房间当前人数、房间ID、是否有密码等基础信息）
func (s *App) GetRoomList(page int, counts int) any {
	len := s.rooms.Length()
	allpage := int(len/counts) + 1
	roomLen := s.rooms.Length()
	if page > 0 && page <= allpage && roomLen > 0 {
		// 开始截取的位置
		startIndex := (page - 1) * counts
		// 最后截取的位置
		endIndex := startIndex + counts
		if endIndex > roomLen {
			endIndex = roomLen
		}
		list := s.rooms.List[startIndex:endIndex]
		if list != nil {
			arr := []RoomInfo{}
			for _, v := range list {
				r := v.(*Room)
				arr = append(arr, RoomInfo{
					Id:        r.id,
					Counts:    r.users.Length(),
					MaxCounts: r.option.maxCounts,
					Password:  r.option.password != "",
					Master:    r.master.name,
					Lock:      r.lock,
				})
			}
			return arr
		}
	}
	return nil
}
