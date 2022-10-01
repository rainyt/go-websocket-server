package net

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
	"websocket_server/util"
	"websocket_server/websocket"
)

type ClientAction int

const (
	Error                ClientAction = -1 // 通用错误，发生错误时，Data请传递`ClientError`结构体
	Message              ClientAction = 0  // 普通消息
	CreateRoom           ClientAction = 1  // 创建房间
	JoinRoom             ClientAction = 2  // 加入房间
	ChangedRoom          ClientAction = 3  // 房间信息变更
	GetRoomData          ClientAction = 4  // 获取房间信息
	StartFrameSync       ClientAction = 5  // 开启帧同步
	StopFrameSync        ClientAction = 6  // 停止帧同步
	UploadFrame          ClientAction = 7  // 上传帧同步数据
	Login                ClientAction = 8  // 登陆用户
	FData                ClientAction = 9  // 帧数据
	RoomMessage          ClientAction = 10 // 发送房间消息
	JoinRoomClient       ClientAction = 11 // 加入房间的客户端信息
	ExitRoomClient       ClientAction = 12 // 退出房间的客户端信息
	OutOnlineRoomClient  ClientAction = 13 // 在房间中离线的客户端信息，请注意，只有开启了帧同步的情况下收到
	ExitRoom             ClientAction = 14 // 退出房间
	MatchUser            ClientAction = 15 // 匹配用户
	UpdateUserData       ClientAction = 16 // 更新用户数据
	GetRoomOldMessage    ClientAction = 17 // 获取房间的历史消息
	UpdateRoomCustomData ClientAction = 18 // 更新自定义房间信息（房主操作）
	UpdateRoomOption     ClientAction = 19 // 更新房间的配置，如人数、密码等（房主操作）
	KickOut              ClientAction = 20 // 踢出用户（房主操作）
	SelfKickOut          ClientAction = 21 // 自已被踢出房间
	GetFrameAt           ClientAction = 22 // 获取指定帧范围的帧事件
	SetRoomState         ClientAction = 23 // 设置房间状态数据
	RoomStateUpdate      ClientAction = 24 // 房间状态更新
	SetClientState       ClientAction = 25 // 设置用户状态
	ClientStateUpdate    ClientAction = 26 // 用户状态发生变化
	FrameSyncReady       ClientAction = 27 // 帧同步准备传输
	ResetRoom            ClientAction = 28 // 重置房间状态
	Matched              ClientAction = 29 // 匹配成功，匹配成功后，可通过GetRoomData获取房间信息
	LockRoom             ClientAction = 30 // 锁定房间
	UnlockRoom           ClientAction = 31 // 取消锁定房间
	MatchRoom            ClientAction = 32 // 匹配房间
	SetRoomMatchOption   ClientAction = 33 // 设置房间的匹配参数
	UpdateRoomUserData   ClientAction = 34 // 更新房间用户中的数据
	GetRoomList          ClientAction = 35 // 获取房间列表
)

type ClientMessage struct {
	Op   ClientAction `json:"op"`
	Data any          `json:"data"`
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
	net.Conn
	websocket     bool             // 是否使用webscoket协议
	handshakeData string           // 握手信息
	data          *util.Bytes      // 缓存数据
	isFinal       bool             // 是否最终包
	opcode        websocket.Opcode // 操作符
	frameIsBinary bool             // 是否二进制数据
	partialLength int              // 内容长度
	isMasked      bool             // 是否存在掩码
	state         websocket.State  // 状态码
	length        int              // 长度
	mask          []byte           // 掩码数据
	lastPong      int64            // 上一次心跳时间
	room          *Room            // 房间（每个用户只会进入到一个房间中）
	userData      map[string]any   // 用户自定义数据
	frames        *util.Array      // 用户帧同步缓存操作
	uid           int              // 用户ID
	name          string           // 用户名称
	online        bool             // 是否在线
	matchOption   *MatchOption     // 房间匹配参数
	compress      bool             // 客户端是否启动压缩传输
}

// 发送数据给所有人
func (c *Client) SendToAllUser(data []byte) {
	for _, v := range CurrentServer.users.List {
		v.(*Client).SendToUser(data)
	}
}

// 单独发送数据到当前用户
func (c *Client) SendToUser(data []byte) {
	_, err := c.Write(data)
	if err != nil {
		fmt.Println(err.Error())
	}
}

// 发送客户端数据到当前用户
func (c *Client) SendToUserOp(data *ClientMessage) {
	v, err := json.Marshal(data)
	if err == nil {
		// 发送
		bdata := websocket.PrepareFrame(v, websocket.Text, true, c.compress)
		c.SendToUser(bdata.Data)
	}
}

// 数据缓存处理
func (c *Client) onData(data []byte) {
	c.data.WriteBytes(data)
	if c.state == websocket.Handshake {
		// 接收到结束符
		cdata := c.data.ReadUTFString(c.data.ByteLength())
		c.handshakeData += cdata
		index := strings.Index(c.handshakeData, "\r\n\r\n")
		if index != -1 {
			// 开始握手
			c.handshake(c.handshakeData)
		}
	} else {
		// todo 这里需要解析websocket的数据结构
		data, ok := readWebSocketData(c)
		if ok {
			util.Log(string(data))
		}
	}
}

// 发送一个WebSocket包
func (c *Client) WriteWebSocketData(data []byte, opcode websocket.Opcode) {
	var dataContent = websocket.PrepareFrame(data, opcode, true, c.compress).Data
	c.SendToUser(dataContent)
}

// 用户离线时触发
func (c *Client) onUserOut() {
	// 如果存在房间时，则需要退出房间
	if c.room != nil {
		util.Log("用户退出")
		c.online = false
		// 如果房间存在，而且房间没有锁定时，离线则可以直接退出房间
		if c.room != nil {
			if !c.room.lock {
				CurrentServer.ExitRoom(c)
			} else {
				// 离线状态
				c.room.SendToAllUserOp(&ClientMessage{
					Op:   OutOnlineRoomClient,
					Data: c.GetUserData(),
				}, c)
			}
		}
		// 从服务器列表中删除
	}
	CurrentServer.users.Remove(c)
	// 从服务器匹配列表中取消
	CurrentServer.matchs.cannelMatchUser(c)
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

func formatString(str string) string {
	str = strings.ReplaceAll(str, " ", "")
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\r", "")
	return str
}

// 同意WebSocket握手
func (c *Client) handshake(content string) {
	s := strings.Split(content, "\n")
	var secWebSocketKey string
	var extensions string
	for _, v := range s {
		keys := strings.Split(v, ":")
		switch keys[0] {
		case "Sec-WebSocket-Extensions":
			// 判断压缩是否开启
			extensions = formatString(keys[1])
			util.Log("extensions:" + extensions)
			// todo 待处理
			// index := strings.Index(extensions, "permessage-deflate")
			// if index == -1 {
			// 	index = strings.Index(extensions, "x-webkit-deflate-frame")
			// }
			// c.compress = index != -1
			c.compress = false
		case "Sec-WebSocket-Key":
			secWebSocketKey = formatString(keys[1])
		}
	}
	if secWebSocketKey != "" {
		// 同意握手时，返回信息
		base := secWebSocketKey + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
		util.Log("secWebSocketKey=" + base)
		t := sha1.New()
		io.WriteString(t, base)
		bs := t.Sum(nil)
		encoded := base64.StdEncoding.EncodeToString(bs)
		handdata := []string{
			"HTTP/1.1 101 Switching Protocols",
			"Upgrade: websocket",
			"Connection: Upgrade",
			"Sec-WebSocket-Accept: " + encoded,
			"Access-Control-Allow-Credentials: true",
			"Access-Control-Allow-Headers: content-type",
			// "Sec-WebSocket-Protocol: chat",
		}
		// TODO 启动压缩时返回
		// if c.compress {
		// handdata = append(handdata, "Sec-WebSocket-Extensions: "+extensions)
		// handdata = append(handdata, "Sec-WebSocket-Extensions: "+"permessage-deflate")
		// handdata = append(handdata, "Sec-WebSocket-Extensions: permessage-deflate; server_no_context_takeover; client_no_context_takeover")
		// }
		data := strings.Join(handdata, "\r\n") + "\r\n\r\n"
		c.SendToUser([]byte(data))
		// 标记握手成功
		c.state = websocket.Head
	} else {
		c.Close()
	}
}

// 客户端逻辑处理
func clientHandle(c *Client) {
	defer c.Close()
	defer c.onUserOut()
	for {
		// 每次客户端读取的数据长度
		var bytes [128]byte
		n, e := c.Read(bytes[:])
		if e != nil {
			break
		}
		if n == 0 {
			continue
		}
		// 缓存数据
		c.onData(bytes[:n])
	}
}

// 创建客户端对象
func CreateClient(c net.Conn) *Client {
	client := Client{
		Conn:      c,
		websocket: true,
		data:      &util.Bytes{Data: []byte{}},
		state:     websocket.Handshake,
		userData:  map[string]any{},
		online:    true,
		frames:    util.CreateArray(),
	}
	go clientHandle(&client)
	return &client
}

// 读取WebSocket的数据包
func readWebSocketData(c *Client) ([]byte, bool) {
	var byteLength = c.data.ByteLength()
	switch c.state {
	case websocket.Head:
		// 字节少于2的时候，意味着数据不足
		if byteLength < 2 {
			return nil, false
		}
		b0 := c.data.ReadInt()
		b1 := c.data.ReadInt()
		c.isFinal = ((b0 >> 7) & 1) != 0
		c.opcode = websocket.Opcode(((b0 >> 0) & 0xF))
		if c.opcode == websocket.Text {
			c.frameIsBinary = false
		} else if c.opcode == websocket.Binary {
			c.frameIsBinary = true
		}
		c.partialLength = ((b1 >> 0) & 0x7F)
		c.isMasked = ((b1 >> 7) & 1) != 0

		util.Log(b0, b1)
		// util.Log("c.isFinal=", c.isFinal)
		// util.Log("c.isMasked=", c.isMasked)
		// util.Log("c.opcode=", c.opcode)
		// util.Log("c.partialLength=", c.partialLength)
		c.state = websocket.HeadExtraLength
	case websocket.HeadExtraLength:
		if c.partialLength == 126 {
			if byteLength < 2 {
				return nil, false
			}
			c.length = c.data.ReadUnsignedShort()
		} else if c.partialLength == 127 {
			if byteLength < 8 {
				return nil, false
			}
			var tmp = c.data.ReadUnsignedInt()
			if tmp != 0 {
				return nil, false
			}
			c.length = c.data.ReadUnsignedInt()
		} else {
			c.length = c.partialLength
		}
		c.state = websocket.HeadExtraMask

		// util.Log("c.length=", c.length)
	case websocket.HeadExtraMask:
		if c.isMasked {
			if byteLength < 4 {
				return nil, false
			}
			c.mask = c.data.ReadBytes(4)
			// util.Log("c.mask=", c.mask)
		}
		c.state = websocket.Body
	case websocket.Body:
		// util.Log("len=", byteLength, c.length)
		if byteLength < c.length {
			return nil, false
		}
		data := c.data.ReadBytes(c.length)
		switch c.opcode {
		case websocket.Binary, websocket.Text, websocket.Continuation:
			fmt.Println("do c.opcode")

			// TODO 如果后续需要支持压缩支持，这里需要确认
			//根据刚才存有压缩内容的buffer获取flate Reader
			// buf := new(bytes.Buffer)
			// buf.Write(data)
			// flateReader := flate.NewReader(buf)
			// defer flateReader.Close()
			//copy flate Reader中的内容
			// deBuffer := new(bytes.Buffer)
			// _, err := io.Copy(deBuffer, flateReader)
			// if err != nil {
			// 	fmt.Println("解压错误:" + err.Error())
			// } else {
			// 	data2 := deBuffer.Bytes()
			// 	data2 = applyMask(data2, c.mask)
			// 	fmt.Println("解压成功", string(data2))
			// }

			if c.isFinal {
				if c.isMasked {
					data = websocket.ApplyMask(data, c.mask)
				}
			}
			util.Log(string(data))
			c.onMessage(data)
		case websocket.Ping:
			c.WriteWebSocketData(data, websocket.Pong)
		case websocket.Pong:
			c.lastPong = time.Now().Unix()
		case websocket.Close:
			data = websocket.ApplyMask(data, c.mask)
			util.Log("中断：", string(data))
			c.Close()
		}
		c.state = websocket.Head
	default:
		return nil, false
	}

	byteLength = c.data.ByteLength()
	if byteLength > 0 {
		return readWebSocketData(c)
	}
	return nil, false
}
