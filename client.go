package teleport

import (
	"log"
	"net"
	"time"
)

// ***********************************************功能实现*************************************************** \\

// 以客户端模式启动
func (self *TP) client() {
	log.Println(" *     —— 正在连接服务器……")

RetryLabel:
	conn, err := net.Dial("tcp", self.serverAddr+self.port)
	if err != nil {
		time.Sleep(1e9)
		goto RetryLabel
	}
	// log.Printf(" *     —— 成功连接到服务器：%v ——", conn.RemoteAddr().String())

	// 开启该连接处理协程(读写两条协程)
	self.cGoConn(conn)

	// 当与服务器失连后，自动重新连接
	if !self.canClose {
		for self.CountNodes() > 0 {
			time.Sleep(1e9)
		}
		if !self.canClose {
			goto RetryLabel
		}
	}
}

// 为每个连接开启读写两个协程
func (self *TP) cGoConn(conn net.Conn) {
	remoteAddr, connect := NewConnect(conn, self.connBufferLen, self.connWChanCap)
	self.connPool["Server"] = connect
	// 绑定节点UID与conn
	if self.uid == "" {
		self.uid = conn.LocalAddr().String()
	}

	if !self.canClose {
		self.send(NewNetData(self.uid, "Server", IDENTITY, ""))
	}

	// 标记连接已经正式生效可用
	self.connPool["Server"].UID = remoteAddr

	log.Printf(" *     —— 成功连接到服务器：%v (%v)——", "Server", remoteAddr)
	// 开启读写双工协程
	go self.cReader("Server")
	go self.cWriter("Server")
}

// 客户端读数据
func (self *TP) cReader(nodeuid string) {
	// 退出时关闭连接，删除连接池中的连接
	defer func() {
		self.closeConn(nodeuid)
	}()

	var conn = self.getConn(nodeuid)

	for {
		if !self.read(conn) {
			break
		}
	}
}

// 客户端发送数据
func (self *TP) cWriter(nodeuid string) {
	// 退出时关闭连接，删除连接池中的连接
	defer func() {
		self.closeConn(nodeuid)
	}()

	var conn = self.getConn(nodeuid)

	for {
		if self.canClose {
			self.send(<-conn.WriteChan)
			continue
		}

		timing := time.After(self.timeout)
		data := new(NetData)
		select {
		case data = <-conn.WriteChan:
		case <-timing:
			// 保持心跳
			data = NewNetData(self.uid, nodeuid, HEARTBEAT, "")
		}

		self.send(data)
	}
}
