// Connect长连接结构体。
package teleport

import (
	"net"
)

// 长连接
type Connect struct {
	// 唯一标识符
	UID string
	// 专用写入数据缓存通道
	WriteChan chan *NetData
	// 从连接循环接收数据
	Buffer []byte
	// 临时缓冲区，用来存储被截断的数据
	TmpBuffer []byte
	// 标准包conn接口实例，继承该接口所有方法
	net.Conn
}

// 创建Connect实例，uid默认为conn的RemoteAddr
func NewConnect(conn net.Conn, bufferLen int, wChanCap int) (k string, v *Connect) {
	k = conn.RemoteAddr().String()

	v = &Connect{
		UID:       k,
		WriteChan: make(chan *NetData, wChanCap),
		Buffer:    make([]byte, bufferLen),
		TmpBuffer: make([]byte, 0),
		Conn:      conn,
	}
	return k, v
}
