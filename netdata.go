//socket长连接， JSON 数据传输包。
package teleport

import (
// "net"
)

const (
	// 返回成功
	SUCCESS = 0
	// 返回失败
	FAILURE = -1
	// 返回非法请求
	LLLEGAL = -2
)

// 定义数据传输结构
type NetData struct {
	// 消息体
	Body interface{}
	// 操作代号
	Operation string
	// 唯一标识符
	UID string
	// 发信节点uid
	From string
	// 收信节点uid
	To string
	// 返回状态
	Status int
}

func NewNetData(from, to, operation string, body interface{}) *NetData {
	return &NetData{
		From:      from,
		To:        to,
		Body:      body,
		Operation: operation,
		Status:    SUCCESS,
	}
}
