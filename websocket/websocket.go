package websocket

import (
	"encoding/base64"
	"math/rand"
	"websocket_server/util"
)

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
