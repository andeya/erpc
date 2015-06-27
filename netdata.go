//socket长连接， JSON 数据传输包。
package teleport

import (
// "net"
)

// 定义数据传输结构
type NetData struct {
	// 消息体
	Body interface{}
	// 操作代号
	Operation string
	// 唯一标识符
	UID string
	// 发信方代号
	From string
	// 收信方代号
	To string
}

func NewNetData1(from string, to string, operation string, body interface{}) *NetData {
	return &NetData{
		Body:      body,
		Operation: operation,
		From:      from,
		To:        to,
	}
}

func NewNetData2(conn *Connect, operation string, body interface{}) *NetData {
	return NewNetData1(conn.LocalAddr().String(), conn.RemoteAddr().String(), operation, body)
}
