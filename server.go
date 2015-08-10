package teleport

import (
	"encoding/json"
	"log"
	"net"
	"time"
)

// ***********************************************功能实现*************************************************** \\

// 以服务器模式启动
func (self *TP) server() {
	var err error
	self.listener, err = net.Listen("tcp", self.port)
	if err != nil {
		log.Printf("监听端口出错: %s", err.Error())
	}

	log.Println(" *     —— 已开启服务器监听 ——")
	for {
		// 等待下一个连接,如果没有连接,listener.Accept会阻塞
		conn, err := self.listener.Accept()
		if err != nil {
			continue
		}
		// log.Printf(" *     —— 客户端 %v 连接成功 ——", conn.RemoteAddr().String())

		// 开启该连接处理协程(读写两条协程)
		self.sGoConn(conn)
	}
}

// 为每个连接开启读写两个协程
func (self *TP) sGoConn(conn net.Conn) {
	remoteAddr, connect := NewConnect(conn, self.connBufferLen, self.connWChanCap)
	// 或者key为 strings.Split(conn.RemoteAddr().String(), ":")[0]
	self.connPool[remoteAddr] = connect
	// 初始化节点UID
	nodeuid := self.sInitConn(connect)
	if nodeuid == "" {
		return
	}
	log.Printf(" *     —— 客户端 %v (%v) 连接成功 ——", nodeuid, remoteAddr)

	// 开启读写双工协程
	go self.sReader(nodeuid)
	go self.sWriter(nodeuid)
}

// 连接初始化，绑定节点与连接，默认key为节点ip
func (self *TP) sInitConn(conn *Connect) (nodeuid string) {
	read_len, err := conn.Read(conn.Buffer)
	if err != nil || read_len == 0 {
		return ""
	}

	addr := conn.RemoteAddr().String()

	conn.TmpBuffer = append(conn.TmpBuffer, conn.Buffer[:read_len]...)
	// 解包
	dataSlice := make([][]byte, 10)
	dataSlice, conn.TmpBuffer = self.Unpack(conn.TmpBuffer)
	flag := true

	for _, data := range dataSlice {

		d := new(NetData)
		json.Unmarshal(data, d)

		if flag {
			if d.Operation != IDENTITY {
				conn.IsShort = true
			}
			// log.Println("收到身份信息：", data)
			if addr == d.From {
				// 或者key为 strings.Split(conn.RemoteAddr().String(), ":")[0]
				nodeuid = addr
			} else {
				nodeuid = d.From
				delete(self.connPool, addr)
				self.connPool[nodeuid] = conn
			}
			flag = false
		}

		// 添加到读取缓存
		self.apiReadChan <- d
	}

	defer func() {
		// 标记连接已经正式生效可用
		conn.UID = addr
	}()

	return
}

// 服务器读数据
func (self *TP) sReader(nodeuid string) {
	// 退出时关闭连接，删除连接池中的连接
	defer func() {
		self.closeConn(nodeuid)
	}()

	var conn = self.getConn(nodeuid)

	for {
		// 设置连接超时
		if !conn.IsShort {
			conn.SetReadDeadline(time.Now().Add(self.timeout))
		}
		// 等待读取数据
		if !self.read(conn) {
			break
		}
	}
}

// 服务器发送数据
func (self *TP) sWriter(nodeuid string) {
	// 退出时关闭连接，删除连接池中的连接
	defer func() {
		self.closeConn(nodeuid)
	}()

	var conn = self.getConn(nodeuid)

	for {
		data := <-conn.WriteChan
		self.send(data)
		if conn.IsShort {
			return
		}
	}
}
