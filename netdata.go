//socket长连接， JSON 数据传输包。
package teleport

import (
// "net"
)

const (
	// 返回成功
	SUCCESS = iota
	// 返回失败
	FAILURE
	// 返回非法请求
	LLLEGAL
)

// 定义数据传输结构
type NetData struct {
	// 消息体
	Body interface{}
	// 操作代号
	Operation string
	// 唯一标识符
	UID string
	// 发信方ip:port
	From string
	// 收信方ip:port
	To string
	// 返回状态
	Status int
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
