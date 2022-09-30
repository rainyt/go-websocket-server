package websocket

type State int

const (
	Handshake       State = iota // 握手状态
	Head                         // 读取Head
	HeadExtraLength              // 读取内容长度
	HeadExtraMask                // 读取掩码
	Body                         // 读取内容
)
