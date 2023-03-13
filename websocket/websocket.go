package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"io"
	"math/rand"
	"net"
	"strings"
	"time"
	"websocket_server/runtime"
	"websocket_server/util"
)

type WebSocket struct {
	net.Conn                  // 请求Conn
	IsWebSocket   bool        // 是否使用webscoket协议
	HandshakeData string      // 握手信息
	Data          *util.Bytes // 缓存数据
	IsFinal       bool        // 是否最终包
	Opcode        Opcode      // 操作符
	FrameIsBinary bool        // 是否二进制数据
	PartialLength int         // 内容长度
	IsMasked      bool        // 是否存在掩码
	State         State       // 状态码
	Length        int         // 长度
	Mask          []byte      // 掩码数据
	LastPong      int64       // 上一次心跳时间
	Compress      bool        // 客户端是否启动压缩传输
	Connected     bool        // 是否建立了连接
	WriteChannel  chan []byte // 写入数据的发送渠道
}

type IWebSocket interface {
	OnMessage(data []byte)
	GetWebSocket() *WebSocket
	Close() error
	OnUserOut()
}

func (w *WebSocket) GetWebSocket() *WebSocket {
	return w
}

// 创建一个随机的SecWebSocketKey
func CreateSecWebSocketKey() string {
	str := "qwertyuioplkjghgfdsazxcvbnm"
	l := len(str)
	sign := ""
	for i := 0; i < 16; i++ {
		sign += string(str[rand.Intn(l)])
	}
	key := base64.RawStdEncoding.EncodeToString([]byte(sign))
	return key
}

// 包装成一个WebSocket包
func PrepareFrame(data []byte, opcode Opcode, isFinal bool, compress bool) util.Bytes {
	newdata := util.Bytes{Data: []byte{}}
	var isMasked = false // All clientes messages must be masked: http://tools.ietf.org/html/rfc6455#section-5.1
	var mask = GenerateMask()
	var sizeMask = 0x00
	if isMasked {
		sizeMask = 0x80
	}
	var sizeFinal = 0x00
	if isFinal {
		sizeFinal = 0x80
	}
	newdata.Write(int(opcode) | sizeFinal)
	var byteLength = len(data)
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
	if isMasked {
		for i := 0; i < 4; i++ {
			newdata.Data = append(newdata.Data, mask[i])
		}
		maskdata := ApplyMask(data, mask[:])
		newdata.WriteBytes(maskdata)
	} else {
		newdata.WriteBytes(data)
	}
	if compress {
		util.Log("压缩传输")
	}
	return newdata
}

func GenerateMask() [4]byte {
	var maskData = [4]byte{}
	maskData[0] = byte(rand.Intn(256))
	maskData[1] = byte(rand.Intn(256))
	maskData[2] = byte(rand.Intn(256))
	maskData[3] = byte(rand.Intn(256))
	return maskData
}

func ApplyMask(data []byte, mask []byte) []byte {
	var newdata = make([]byte, len(data))
	var makelen = len(mask)
	for i, v := range data {
		newdata[i] = v ^ mask[i%makelen]
	}
	return newdata
}

// 读取WebSocket的数据包
func ReadWebSocketData(iweb IWebSocket) ([]byte, bool) {
	c := iweb.GetWebSocket()
	var byteLength = c.Data.ByteLength()
	switch c.State {
	case Head:
		// 字节少于2的时候，意味着数据不足
		if byteLength < 2 {
			return nil, false
		}
		b0 := c.Data.ReadInt()
		b1 := c.Data.ReadInt()
		c.IsFinal = ((b0 >> 7) & 1) != 0
		c.Opcode = Opcode(((b0 >> 0) & 0xF))
		if c.Opcode == Text {
			c.FrameIsBinary = false
		} else if c.Opcode == Binary {
			c.FrameIsBinary = true
		}
		c.PartialLength = ((b1 >> 0) & 0x7F)
		c.IsMasked = ((b1 >> 7) & 1) != 0

		// util.Log("c.IsFinal=", c.IsFinal)
		// util.Log("c.IsMasked=", c.IsMasked)
		// util.Log("c.Opcode=", c.Opcode)
		// util.Log("c.PartialLength=", c.PartialLength)
		c.State = HeadExtraLength
	case HeadExtraLength:
		if c.PartialLength == 126 {
			if byteLength < 2 {
				return nil, false
			}
			c.Length = c.Data.ReadUnsignedShort()
		} else if c.PartialLength == 127 {
			if byteLength < 8 {
				return nil, false
			}
			var tmp = c.Data.ReadUnsignedInt()
			if tmp != 0 {
				return nil, false
			}
			c.Length = c.Data.ReadUnsignedInt()
		} else {
			c.Length = c.PartialLength
		}
		c.State = HeadExtraMask

		// util.Log("c.Length=", c.Length)
	case HeadExtraMask:
		if c.IsMasked {
			if byteLength < 4 {
				return nil, false
			}
			c.Mask = c.Data.ReadBytes(4)
			// util.Log("c.mask=", c.mask)
		}
		c.State = Body
	case Body:
		// util.Log("len=", byteLength, c.Length)
		if byteLength < c.Length {
			return nil, false
		}
		data := c.Data.ReadBytes(c.Length)
		switch c.Opcode {
		case Binary, Text, Continuation:
			// fmt.Println("do c.Opcode")
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

			if c.IsFinal {
				if c.IsMasked {
					data = ApplyMask(data, c.Mask)
				}
			}
			iweb.OnMessage(data)
		case Ping:
			c.WriteWebSocketData(data, Pong)
		case Pong:
			c.LastPong = time.Now().Unix()
		case Close:
			data = ApplyMask(data, c.Mask)
			util.Log("中断：", string(data))
			c.Close()
		}
		c.State = Head
	default:
		return nil, false
	}

	byteLength = c.Data.ByteLength()
	if byteLength > 0 {
		return ReadWebSocketData(iweb)
	}
	return nil, false
}

// 客户端写入数据逻辑处理
func CreateClientWriteHandle(client IWebSocket) {
	defer runtime.GoRecover()
	for {
		web := client.GetWebSocket()
		writeData, b := <-web.WriteChannel
		if !b {
			return
		}
		if !web.Connected {
			return
		}
		_, err := web.Write(writeData)
		if err != nil {
			util.Log("Error:" + err.Error())
			web.Connected = false
			web.Close()
		}
	}
}

// 客户端读取数据逻辑处理
func CreateClientHandle(iweb IWebSocket) {
	defer runtime.GoRecover()
	c := iweb.GetWebSocket()
	defer c.Close()
	defer iweb.OnUserOut()
	for {
		// 每次客户端读取的数据长度
		if !c.Connected {
			util.Log("已断开链接")
			break
		}
		var bytes [128]byte
		// if c.State == Handshake {
		// 	// 握手时，需要超时15秒
		// 	c.Conn.SetReadDeadline(time.Now().Add(15 * time.Second))
		// }
		n, e := c.Read(bytes[:])
		if e != nil {
			break
		}
		if n == 0 {
			continue
		}
		// 缓存数据
		OnData(iweb, bytes[:n])
	}
}

// 发送一个WebSocket包
func (c *WebSocket) WriteWebSocketData(data []byte, opcode Opcode) {
	var dataContent = PrepareFrame(data, opcode, true, c.Compress).Data
	c.Write(dataContent)
}

// 读取一个字节包
func (c *WebSocket) Read(b []byte) (int, error) {
	// 读取时，暂不做超时处理
	// c.Conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	return c.Conn.Read(b)
}

// 写入一个字节包
func (c *WebSocket) Write(data []byte) (int, error) {
	c.Conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	return c.Conn.Write(data)
}

// 数据缓存处理
func OnData(iweb IWebSocket, data []byte) {
	c := iweb.GetWebSocket()
	c.Data.WriteBytes(data)
	if c.State == Handshake {
		// 接收到结束符
		cdata := c.Data.ReadUTFString(c.Data.ByteLength())
		c.HandshakeData += cdata
		index := strings.Index(c.HandshakeData, "\r\n\r\n")
		if index != -1 {
			// 开始握手
			handshake(iweb, c.HandshakeData)
		}
	} else {
		// todo 这里需要解析websocket的数据结构
		data, ok := ReadWebSocketData(iweb)
		if ok {
			util.Log(string(data))
		}
	}
}

// 同意WebSocket握手
func handshake(iweb IWebSocket, content string) {
	c := iweb.GetWebSocket()
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
			c.Compress = false
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
		// 15秒超时
		c.Write([]byte(data))
		// 标记握手成功
		c.State = Head
	} else {
		c.Close()
	}
}

func formatString(str string) string {
	str = strings.ReplaceAll(str, " ", "")
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\r", "")
	return str
}
