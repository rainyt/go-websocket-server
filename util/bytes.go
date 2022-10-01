package util

// 二进制数据解析
type Bytes struct {
	Data []byte
}

// 创建一个二进制数据
func CreateBytes() *Bytes {
	return &Bytes{
		Data: []byte{},
	}
}

// 写入一个int字节
func (b *Bytes) Write(i int) {
	b.Data = append(b.Data, byte(i))
}

// 写入二进制数组
func (b *Bytes) WriteBytes(data []byte) {
	b.Data = append(b.Data, data...)
}

// 当前可用的长度
func (b *Bytes) ByteLength() int {
	return len(b.Data)
}

// 读取字符串
func (b *Bytes) ReadUTFString(l int) string {
	data_tmp := b.Data[0:l]
	if l == len(b.Data) {
		b.Data = []byte{}
	} else {
		b.Data = b.Data[l+1:]
	}
	return string(data_tmp)
}

// 读取一个字节
func (b *Bytes) Read() byte {
	data_tmp := b.Data[0]
	b.Data = b.Data[1:]
	return data_tmp
}

// 读取一个整数字节
func (b *Bytes) ReadInt() int {
	return int(b.Read())
}

// 写入一个Short整数
func (b *Bytes) WriteShort(v int) {
	b.Write((v >> 8) & 0xFF)
	b.Write((v >> 0) & 0xFF)
}

func (b *Bytes) ReadUnsignedShort() int {
	var h = b.ReadInt()
	var l = b.ReadInt()
	return (h << 8) | (l << 0)
}

func (b *Bytes) ReadUnsignedInt() int {
	var v3 = b.ReadInt()
	var v2 = b.ReadInt()
	var v1 = b.ReadInt()
	var v0 = b.ReadInt()
	return (v3 << 24) | (v2 << 16) | (v1 << 8) | (v0 << 0)
}

func (b *Bytes) ReadBytes(l int) []byte {
	data_tmp := b.Data[0:l]
	b.Data = b.Data[l:]
	return data_tmp
}
