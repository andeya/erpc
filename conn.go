// Connect长连接结构体。
package teleport

import (
	"net"
)

// 长连接
type Connect struct {
	// 唯一标识符，Teleport确定为有效连接后才可赋值
	UID string
	// 专用写入数据缓存通道
	WriteChan chan *NetData
	// 从连接循环接收数据
	Buffer []byte
	// 临时缓冲区，用来存储被截断的数据
	TmpBuffer []byte
	// 标准包conn接口实例，继承该接口所有方法
	net.Conn

	// 是否为短链接模式
	IsShort bool
}

// 创建Connect实例
func NewConnect(conn net.Conn, bufferLen int, wChanCap int) (k string, v *Connect) {
	k = conn.RemoteAddr().String()

	v = &Connect{
		UID:       "",
		WriteChan: make(chan *NetData, wChanCap),
		Buffer:    make([]byte, bufferLen),
		TmpBuffer: make([]byte, 0),
		Conn:      conn,
	}
	return k, v
}

// 以UID判断连接是否完成准备好
func (self *Connect) IsReady() bool {
	if self.UID == "" {
		return false
	}
	return true
}
