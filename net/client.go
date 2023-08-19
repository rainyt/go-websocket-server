package net

import (
	"encoding/json"
	"net"
	"runtime"
	"sync"
	"time"
	"websocket_server/logs"
	"websocket_server/util"
	"websocket_server/websocket"
)

type ClientAction int

const (
	Error                   ClientAction = -1 // 通用错误，发生错误时，Data请传递`ClientError`结构体
	Message                 ClientAction = 0  // 普通消息
	CreateRoom              ClientAction = 1  // 创建房间
	JoinRoom                ClientAction = 2  // 加入房间
	ChangedRoom             ClientAction = 3  // 房间信息变更
	GetRoomData             ClientAction = 4  // 获取房间信息
	StartFrameSync          ClientAction = 5  // 开启帧同步
	StopFrameSync           ClientAction = 6  // 停止帧同步
	UploadFrame             ClientAction = 7  // 上传帧同步数据
	Login                   ClientAction = 8  // 登陆用户
	FData                   ClientAction = 9  // 帧数据
	RoomMessage             ClientAction = 10 // 发送房间消息
	JoinRoomClient          ClientAction = 11 // 加入房间的客户端信息
	ExitRoomClient          ClientAction = 12 // 退出房间的客户端信息
	OutOnlineRoomClient     ClientAction = 13 // 在房间中离线的客户端信息，请注意，只有开启了帧同步的情况下收到
	ExitRoom                ClientAction = 14 // 退出房间
	MatchUser               ClientAction = 15 // 匹配用户
	UpdateUserData          ClientAction = 16 // 更新用户数据
	GetRoomOldMessage       ClientAction = 17 // 获取房间的历史消息
	UpdateRoomCustomData    ClientAction = 18 // 更新自定义房间信息（房主操作）
	UpdateRoomOption        ClientAction = 19 // 更新房间的配置，如人数、密码等（房主操作）
	KickOut                 ClientAction = 20 // 踢出用户（房主操作）
	SelfKickOut             ClientAction = 21 // 自已被踢出房间
	GetFrameAt              ClientAction = 22 // 获取指定帧范围的帧事件
	SetRoomState            ClientAction = 23 // 设置房间状态数据
	RoomStateUpdate         ClientAction = 24 // 房间状态更新
	SetClientState          ClientAction = 25 // 设置用户状态
	ClientStateUpdate       ClientAction = 26 // 用户状态发生变化
	FrameSyncReady          ClientAction = 27 // 帧同步准备传输
	ResetRoom               ClientAction = 28 // 重置房间状态
	Matched                 ClientAction = 29 // 匹配成功，匹配成功后，可通过GetRoomData获取房间信息
	LockRoom                ClientAction = 30 // 锁定房间
	UnlockRoom              ClientAction = 31 // 取消锁定房间
	MatchRoom               ClientAction = 32 // 匹配房间
	SetRoomMatchOption      ClientAction = 33 // 设置房间的匹配参数
	UpdateRoomUserData      ClientAction = 34 // 更新房间用户中的数据
	GetRoomList             ClientAction = 35 // 获取房间列表
	SendServerMsg           ClientAction = 36 // 发送全服消息
	GetServerMsg            ClientAction = 37 // 接收到全服消息
	ListenerServerMsg       ClientAction = 38 // 侦听全服消息
	CannelListenerServerMsg ClientAction = 39 // 取消侦听全服消息
	GetUserDataByUID        ClientAction = 40 // 通过UID获取用户数据
	GetServerOldMsg         ClientAction = 41 // 获取全服历史消息
	ExtendsCall             ClientAction = 42 // 调用扩展方法
	CannelMatchUser         ClientAction = 43 // 取消匹配用户
	SendToUser              ClientAction = 44 // 发送消息给用户
	UserMessage             ClientAction = 45 // 接收到用户独立消息内容
)

type ClientMessage struct {
	Op   ClientAction `json:"op"`   // 客户端行为
	Data any          `json:"data"` // 客户端数据
}

type ClientError struct {
	Code ClientErrorCode `json:"code"` // 错误码
	Op   ClientAction    `json:"op"`   // 错误操作
	Msg  string          `json:"msg"`  // 错误信息
}

type ClientErrorCode int

const (
	CREATE_ROOM_ERROR      ClientErrorCode = 1001 // 创建房间信息错误
	GET_ROOM_ERROR         ClientErrorCode = 1002 // 获取房间信息错误
	START_FRAME_SYNC_ERROR ClientErrorCode = 1003 // 启动帧同步错误
	STOP_FRAME_SYNC_ERROR  ClientErrorCode = 1004 // 停止帧同步错误
	UPLOAD_FRAME_ERROR     ClientErrorCode = 1005 // 上传帧同步数据错误
	LOGIN_ERROR            ClientErrorCode = 1006 // 登陆失败
	LOGIN_OUT_ERROR        ClientErrorCode = 1007 // 在别处登陆事件
	OP_ERROR               ClientErrorCode = 1008 // 无效的操作指令
	SEND_ROOM_ERROR        ClientErrorCode = 1009 // 发送房间消息错误
	JOIN_ROOM_ERROR        ClientErrorCode = 1010 // 加入房间错误
	EXIT_ROOM_ERROR        ClientErrorCode = 1011 // 退出房间错误
	MATCH_ERROR            ClientErrorCode = 1012 // 匹配错误
	UPDATE_USER_ERROR      ClientErrorCode = 1013 // 更新用户数据错误
	ROOM_NOT_EXSIT         ClientErrorCode = 1014 // 房间不存在
	ROOM_PERMISSION_DENIED ClientErrorCode = 1015 // 房间权限不足
	DATA_ERROR             ClientErrorCode = 1016 // 数据结果错误
)

type Client struct {
	websocket.WebSocket                // WebSocket基础类
	room                *Room          // 房间（每个用户只会进入到一个房间中）
	userData            map[string]any // 用户自定义数据
	frames              *util.Array    // 用户帧同步缓存操作
	uid                 int            // 用户ID
	name                string         // 用户名称
	matchOption         *MatchOption   // 房间匹配参数
	appid               string         // 绑定的AppId
	sendLock            sync.Mutex     // 发送消息锁定
}

// 发送数据给所有人
func (c *Client) SendToAllUser(data []byte) {
	for _, v := range c.getApp().users.List {
		v.(*Client).SendToUser(data)
	}
}

// 获取App
func (c *Client) getApp() *App {
	return CurrentServer.getApp(c.appid)
}

// 单独发送数据到当前用户
func (c *Client) SendToUser(data []byte) {
	// 使用线程发送
	if c.Connected {
		if len(c.WriteChannel) == cap(c.WriteChannel) {
			logs.InfoM("用户缓存已超出最大值，中断处理")
			c.Connected = false
			c.Close()
			return
		}
		select {
		case c.WriteChannel <- data:
		case <-time.After(5 * time.Second):
			// 阻塞超时
			c.Connected = false
			c.Close()
		default:
			logs.InfoM("发送数据渠道已关闭")
		}
	}
}

// 发送客户端数据到当前用户
func (c *Client) SendToUserOp(data *ClientMessage) {
	if data != nil {
		c.sendLock.Lock()
		// 指针会有nil丢失的情况，保护数据
		var value ClientMessage = ClientMessage{
			Op:   data.Op,
			Data: data.Data,
		}
		v, err := json.Marshal(value)
		if err == nil {
			bdata := websocket.PrepareFrame(v, websocket.Text, true, c.Compress)
			c.SendToUser(bdata.Data)
		}
		c.sendLock.Unlock()
	}
}

// 用户离线时触发
func (c *Client) OnUserOut() {
	// 如果存在房间时，则需要退出房间
	c.Connected = false
	if c.room != nil {
		logs.InfoM("用户" + c.name + "退出房间")
		// 如果房间存在，而且房间没有锁定时，离线则可以直接退出房间
		if c.room.isInvalidRoom() {
			// 如果是已经无效的房间，则全部移除
			for _, v := range c.room.users.List {
				c.getApp().ExitRoom(v.(*Client))
			}
		} else if !c.room.lock {
			c.getApp().ExitRoom(c)
		} else {
			// 离线状态
			c.room.SendToAllUserOp(&ClientMessage{
				Op:   OutOnlineRoomClient,
				Data: c.GetUserData(),
			}, c)
		}
		// 从服务器列表中删除
	}
	c.getApp().users.Remove(c)
	// 从服务器消息侦听中删除
	c.getApp().CannelListenerServerMsg(c)
	// 从服务器匹配列表中取消
	c.getApp().matchs.cannelMatchUser(c)
	// 关闭缓存区
	close(c.WriteChannel)
	// 触发扩展关闭接口
	for _, cf := range CurrentServer.OnClosedApi {
		cf.Call(c, &ClientMessage{}, nil)
	}
}

func (c *Client) SendError(errCode ClientErrorCode, op ClientAction, data string) {
	c.SendToUserOp(&ClientMessage{
		Op: Error,
		Data: ClientError{
			Code: errCode,
			Op:   op,
			Msg:  data,
		},
	})
}

// 获取用户数据
func (c *Client) GetUserData() any {
	data := map[string]any{}
	data["uid"] = c.uid
	data["name"] = c.name
	data["data"] = c.userData
	return data
}

// 获取注册数据
func (c *Client) GetRegisterUserData() *RegisterUserData {
	return c.getApp().usersSQL.GetUserDataByUid(c.uid)
}

// 创建客户端对象
func CreateClient(c net.Conn) *Client {
	client := &Client{
		userData: map[string]any{},
		frames:   util.CreateArray(),
	}
	client.Connected = true
	client.Conn = c
	client.IsWebSocket = true
	client.Data = &util.Bytes{Data: []byte{}}
	client.State = websocket.Handshake
	client.WriteChannel = make(chan []byte, 1024)
	// 创建Handle绑定
	logs.InfoM("线程数量：", runtime.NumGoroutine())
	go websocket.CreateClientHandle(client)
	go websocket.CreateClientWriteHandle(client)
	return client
}
