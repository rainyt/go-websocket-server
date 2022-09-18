package net

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"
	"time"
	"websocket_server/util"
)

type Opcode int

const (
	Continuation Opcode = 0x00
	Text         Opcode = 0x01
	Binary       Opcode = 0x02
	Close        Opcode = 0x08
	Ping         Opcode = 0x09
	Pong         Opcode = 0x0A
)

type State int

const (
	Handshake       State = iota // 握手状态
	Head                         // 读取Head
	HeadExtraLength              // 读取内容长度
	HeadExtraMask                // 读取掩码
	Body                         // 读取内容
)

type Client struct {
	net.Conn
	websocket     bool       // 是否使用webscoket协议
	handshakeData string     // 握手信息
	data          util.Bytes // 缓存数据
	isFinal       bool       // 是否最终包
	opcode        Opcode     // 操作符
	frameIsBinary bool       // 是否二进制数据
	partialLength int        // 内容长度
	isMasked      bool       // 是否存在掩码
	state         State      // 状态码
	length        int        // 长度
	mask          []byte     // 掩码数据
	lastPong      int64      // 上一次心跳时间
	room          *Room      // 房间（每个用户只会进入到一个房间中）
}

// 发送数据给所有人
func (c Client) SendToAllUser(data []byte) {
	for _, v := range CurrentServer.users.List {
		v.(Client).SendToUser(data)
	}
}

// 单独发送数据到当前用户
func (c Client) SendToUser(data []byte) {
	c.Write(data)
}

// 发送客户端数据到当前用户
func (c Client) SendToUserOp(data *ClientMessage) {
	v, err := json.Marshal(data)
	if err == nil {
		// 发送
		util.Log("发送内容：", string(v))
		bdata := prepareFrame(v, Text, true)
		c.SendToUser(bdata.Data)
	}
}

// 数据缓存处理
func (c *Client) onData(data []byte) {
	c.data.WriteBytes(data)
	util.Log("onData")
	if c.state == Handshake {
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
		data, ok := ReadWebSocketData(c)
		if ok {
			util.Log(string(data))
		}
	}
}

// 发送一个WebSocket包
func (c Client) WriteWebSocketData(data []byte, opcode Opcode) {
	var dataContent = prepareFrame(data, opcode, true).Data
	c.SendToUser(dataContent)
	util.Log("发送的长度", len(dataContent))
}

// 包装成一个WebSocket包
func prepareFrame(data []byte, opcode Opcode, isFinal bool) util.Bytes {
	newdata := util.Bytes{Data: []byte{}}
	var isMasked = false // All clientes messages must be masked: http://tools.ietf.org/html/rfc6455#section-5.1
	var mask = generateMask()
	var sizeMask = 0x00
	if isMasked {
		sizeMask = 0x80
	}
	var sizeFinal = 0x00
	if isFinal {
		sizeFinal = 0x80
	}
	newdata.Write(int(opcode) | sizeFinal)
	util.Log("newdata.length=", newdata.ByteLength())
	var byteLength = len(data)
	util.Log("byteLength=", byteLength)
	if byteLength < 126 {
		newdata.Write(byteLength | sizeMask)
	} else if byteLength < 65536 {
		newdata.Write(126 | sizeMask)
		newdata.WriteShort(byteLength)
	} else {
		newdata.Write(127 | sizeMask)
		newdata.Write(0)
		newdata.Write(byteLength)
	}
	util.Log("newdata.length=", newdata.ByteLength())
	if isMasked {
		for i := 0; i < 4; i++ {
			newdata.Data = append(newdata.Data, mask[i])
		}
		util.Log("newdata.length=", newdata.ByteLength())
		util.Log("mask=", mask)
		maskdata := applyMask(data, mask[:])
		newdata.WriteBytes(maskdata)
	} else {
		newdata.WriteBytes(data)
	}
	util.Log("newdata.length=", newdata.ByteLength())
	return newdata
}

func generateMask() [4]byte {
	var maskData = [4]byte{}
	maskData[0] = byte(rand.Intn(256))
	maskData[1] = byte(rand.Intn(256))
	maskData[2] = byte(rand.Intn(256))
	maskData[3] = byte(rand.Intn(256))
	return maskData
}

// 读取WebSocket的数据包
func ReadWebSocketData(c *Client) ([]byte, bool) {
	var byteLength = c.data.ByteLength()
	switch c.state {
	case Head:
		// 字节少于2的时候，意味着数据不足
		if byteLength < 2 {
			return nil, false
		}
		b0 := c.data.ReadInt()
		b1 := c.data.ReadInt()
		c.isFinal = ((b0 >> 7) & 1) != 0
		c.opcode = Opcode(((b0 >> 0) & 0xF))
		if c.opcode == Text {
			c.frameIsBinary = false
		} else if c.opcode == Binary {
			c.frameIsBinary = true
		}
		c.partialLength = ((b1 >> 0) & 0x7F)
		c.isMasked = ((b1 >> 7) & 1) != 0

		util.Log(b0, b1)
		util.Log("c.isFinal=", c.isFinal)
		util.Log("c.isMasked=", c.isMasked)
		util.Log("c.opcode=", c.opcode)
		util.Log("c.partialLength=", c.partialLength)
		c.state = HeadExtraLength
	case HeadExtraLength:
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
		c.state = HeadExtraMask

		util.Log("c.length=", c.length)
	case HeadExtraMask:
		if c.isMasked {
			if byteLength < 4 {
				return nil, false
			}
			c.mask = c.data.ReadBytes(4)
			util.Log("c.mask=", c.mask)
		}
		c.state = Body
	case Body:
		util.Log("len=", byteLength, c.length)
		if byteLength < c.length {
			return nil, false
		}
		data := c.data.ReadBytes(c.length)
		switch c.opcode {
		case Binary, Text, Continuation:
			util.Log("do c.opcode")
			if c.isFinal {
				if c.isMasked {
					data = applyMask(data, c.mask)
				}
			}
			util.Log(string(data))
			// 回复一句话
			// c.WriteWebSocketData([]byte("我是来自服务器的消息"), Text)
			c.onMessage(data)
		case Ping:
			c.WriteWebSocketData(data, Pong)
		case Pong:
			c.lastPong = time.Now().Unix()
		case Close:
			data = applyMask(data, c.mask)
			util.Log("中断：", string(data))
			c.Close()
		}
		c.state = Head
	default:
		return nil, false
	}

	byteLength = c.data.ByteLength()
	if byteLength > 0 {
		return ReadWebSocketData(c)
	}
	return nil, false
}

func applyMask(data []byte, mask []byte) []byte {
	var newdata = make([]byte, len(data))
	var makelen = len(mask)
	for i, v := range data {
		newdata[i] = v ^ mask[i%makelen]
	}
	return newdata
}

type ClientAction int

const (
	Error      ClientAction = -1
	Message    ClientAction = 0
	CreateRoom ClientAction = 1
	JoinRoom   ClientAction = 2
)

type ClientMessage struct {
	Op   ClientAction
	Data any
}

type ClientError struct {
	Code ClientErrorCode
	Msg  string
}

type ClientErrorCode int

const (
	CREATE_ROOM_ERROR ClientErrorCode = 1001
)

// 消息处理
func (c *Client) onMessage(data []byte) {
	if !c.frameIsBinary {
		// 解析API操作
		message := ClientMessage{}
		err := json.Unmarshal(data, &message)
		if err == nil {
			fmt.Println("处理命令", message)
			switch message.Op {
			case Message:
				// 接收到消息
				fmt.Println("服务器接收到消息：", message.Data)
			case CreateRoom:
				// 创建一个房间
				room := CurrentServer.CreateRoom(c)
				util.Log("开始创建房间", room)
				if room != nil {
					// 创建成功
					c.SendToUserOp(&ClientMessage{
						Op: CreateRoom,
						Data: map[string]any{
							"id": room.id,
						},
					})
				} else {
					// 创建失败
					c.SendToUserOp(&ClientMessage{
						Op: Error,
						Data: ClientError{
							Code: CREATE_ROOM_ERROR,
							Msg:  "创建房间失败",
						},
					})
				}
			}
		} else {
			fmt.Println("处理命令失败", string(data))
		}
	}
}

// 同意WebSocket握手
func (c *Client) handshake(content string) {
	util.Log("handshake")
	s := strings.Split(content, "\n")
	var secWebSocketKey string
	for _, v := range s {
		keys := strings.Split(v, ":")
		switch keys[0] {
		case "Sec-WebSocket-Key":
			secWebSocketKey = keys[1]
			secWebSocketKey = strings.ReplaceAll(secWebSocketKey, " ", "")
			secWebSocketKey = strings.ReplaceAll(secWebSocketKey, "\n", "")
			secWebSocketKey = strings.ReplaceAll(secWebSocketKey, "\r", "")
		}
	}
	if secWebSocketKey != "" {
		// 同意握手时，返回信息
		base := secWebSocketKey + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
		t := sha1.New()
		io.WriteString(t, base)
		bs := t.Sum(nil)
		encoded := base64.StdEncoding.EncodeToString(bs)
		handdata := []string{
			"HTTP/1.1 101 Switching Protocols",
			"Upgrade: websocket",
			"Connection: Upgrade",
			"Sec-WebSocket-Accept: " + encoded,
		}
		data := strings.Join(handdata, "\n") + "\r\n\r\n"
		c.SendToUser([]byte(data))
		// 标记握手成功
		c.state = Head
		util.Log("handshake end")
	} else {
		c.Close()
	}
}

// 客户端逻辑处理
func clientHandle(c Client) {
	defer c.Close()
	defer util.Log("Out user", c.RemoteAddr().String())
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
func CreateClient(c net.Conn) Client {
	util.Log("Join user", c.RemoteAddr().String())
	client := Client{
		Conn:      c,
		websocket: true,
		data:      util.Bytes{Data: []byte{}},
		state:     Handshake,
	}
	go clientHandle(client)
	return client
}
