package websocket

type Opcode int

const (
	Continuation Opcode = 0x00
	Text         Opcode = 0x01
	Binary       Opcode = 0x02
	Close        Opcode = 0x08
	Ping         Opcode = 0x09
	Pong         Opcode = 0x0A
)
