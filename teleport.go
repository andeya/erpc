// Teleport是一款适用于分布式系统的高并发API框架，它采用socket长连接、全双工通信，实现S/C对等工作，内部数据传输格式为JSON。
// Version 0.3.2
package teleport

import (
	"encoding/json"
	"log"
	"net"
	// "strings"
	"time"
)

// mode
const (
	SERVER = iota + 1
	CLIENT
	// BOTH
)

// API中定义操作时必须保留的字段
const (
	// 身份登记
	IDENTITY = "+identity+"
	// 心跳操作符
	HEARTBEAT = "+heartbeat+"
)

type Teleport interface {
	// *以服务器模式运行
	Server(port string)
	// *以客户端模式运行
	Client(serverAddr string, port string)
	// *主动推送信息，不写nodeuid默认随机发送给一个节点
	Request(body interface{}, operation string, nodeuid ...string)
	// 指定自定义的应用程序API
	SetAPI(api API) Teleport

	// 设置本节点唯一标识符，默认为本节点ip:port
	SetUID(string) Teleport
	// 设置包头字符串，默认为henrylee2cn
	SetPackHeader(string) Teleport
	// 设置指定API处理的数据的接收缓存通道长度
	SetApiRChan(int) Teleport
	// 设置每个连接对象的发送缓存通道长度
	SetConnWChan(int) Teleport
	// 设置每个连接对象的接收缓冲区大小
	SetConnBuffer(int) Teleport
	// 设置连接超时(心跳频率)
	SetTimeout(time.Duration) Teleport

	// 返回运行模式
	GetMode() int
	// 返回当前连接节点数
	CountNodes() int
}

type TP struct {
	// 本节点唯一标识符
	uid string
	// 运行模式 1 SERVER  2 CLIENT (用于判断自身模式)
	mode int
	// 服务器端口号，格式如":9988"
	port string
	// 服务器地址（不含端口号），格式如"127.0.0.1"
	serverAddr string
	// 长连接池，key为host:port形式
	connPool map[string]*Connect
	// 动态绑定节点功能与conn，key节点UID，value为节点地址host:port
	nodesMap map[string]string
	// 连接时长，心跳时长的依据
	timeout time.Duration
	// 粘包处理
	*Protocol
	// 全局接收缓存通道
	apiReadChan chan *NetData
	// 每个连接对象的发送缓存通道长度
	connWChanCap int
	// 每个连接对象的接收缓冲区大小
	connBufferLen int
	// 应用程序API
	api API
}

// 每个API方法需判断stutas状态，并做相应处理
type API map[string]func(*NetData) *NetData

// 创建接口实例，0为默认设置
func New() Teleport {
	return &TP{
		connPool:      make(map[string]*Connect),
		nodesMap:      make(map[string]string),
		api:           API{},
		Protocol:      NewProtocol("henrylee2cn"),
		apiReadChan:   make(chan *NetData, 4096),
		connWChanCap:  2048,
		connBufferLen: 1024,
	}
}

// ***********************************************实现接口*************************************************** \\

// 指定应用程序API
func (self *TP) SetAPI(api API) Teleport {
	self.api = api
	return self
}

// 启动服务器模式
func (self *TP) Server(port string) {
	self.reserveAPI()
	self.mode = SERVER
	self.port = port
	if self.timeout == 0 {
		// 默认连接超时为5秒
		self.timeout = 5e9
	}
	go self.apiHandle()
	go self.server()
}

// 启动客户端模式
func (self *TP) Client(serverAddr string, port string) {
	self.reserveAPI()
	self.mode = CLIENT
	self.port = port
	self.serverAddr = serverAddr
	if self.timeout == 0 {
		// 默认心跳频率为3秒1次
		self.timeout = 3e9
	}

	go self.apiHandle()
	go self.client()
}

// *主动推送信息，直到有连接出现开始发送，不写nodeuid默认随机发送给一个节点
func (self *TP) Request(body interface{}, operation string, nodeuid ...string) {
	var conn *Connect
	var to string
	if len(nodeuid) == 0 {
		for {
			if len(self.nodesMap) > 0 {
				break
			}
			time.Sleep(5e8)
		}
		// 一个随机节点的信息
		for uid, connKey := range self.nodesMap {
			to = connKey
			nodeuid = append(nodeuid, uid)
			goto aLabel
		}
	} else {
		// 获取指定节点的地址
		to = self.nodesMap[nodeuid[0]]
	}
aLabel:
	// 等待并取得连接实例
	conn = self.getConnByUID(nodeuid[0])
	for conn == nil {
		conn = self.getConnByUID(nodeuid[0])
		time.Sleep(5e8)
	}
	conn.WriteChan <- NewNetData1(conn.LocalAddr().String(), to, operation, body)
	// log.Println("添加一条请求：", conn.RemoteAddr().String(), operation, body)
}

// 设置本节点唯一标识符，默认为本节点IP
func (self *TP) SetUID(nodeuid string) Teleport {
	self.uid = nodeuid
	return self
}

// 设置包头字符串，默认为henrylee2cn
func (self *TP) SetPackHeader(header string) Teleport {
	self.Protocol.ReSet(header)
	return self
}

// 设置全局接收缓存通道长度
func (self *TP) SetApiRChan(length int) Teleport {
	self.apiReadChan = make(chan *NetData, length)
	return self
}

// 设置每个连接对象的发送缓存通道长度
func (self *TP) SetConnWChan(length int) Teleport {
	self.connWChanCap = length
	return self
}

// 每个连接对象的接收缓冲区大小
func (self *TP) SetConnBuffer(length int) Teleport {
	self.connBufferLen = length
	return self
}

// 设置连接超长(心跳频率)
func (self *TP) SetTimeout(long time.Duration) Teleport {
	self.timeout = long
	return self
}

// 返回运行模式
func (self *TP) GetMode() int {
	return self.mode
}

// 返回当前连接节点数
func (self *TP) CountNodes() int {
	return len(self.nodesMap)
}

// ***********************************************功能实现*************************************************** \\

// 以服务器模式启动
func (self *TP) server() {
	listener, err := net.Listen("tcp", self.port)
	if err != nil {
		log.Printf("监听端口出错: %s", err.Error())
	}

	log.Println(" *     —— 已开启服务器监听 ——")
	for {
		// 等待下一个连接,如果没有连接,listener.Accept会阻塞
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		// log.Printf(" *     —— 客户端 %v 连接成功 ——", conn.RemoteAddr().String())

		// 开启该连接处理协程(读写两条协程)
		self.sGoConn(conn)
	}
}

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
	for len(self.connPool) != 0 {
		time.Sleep(1e9)
	}
	goto RetryLabel
}

// 为每个长连接开启读写两个协程
func (self *TP) sGoConn(conn net.Conn) {
	remoteAddr, connect := NewConnect(conn, self.connBufferLen, self.connWChanCap)
	self.connPool[remoteAddr] = connect
	// 登记节点UID
	nodeuid, nodeAddr := self.setNodesMap(connect)
	if nodeuid == "" || nodeAddr == "" {
		return
	}
	log.Printf(" *     —— 客户端 %v (%v) 连接成功 ——", nodeuid, nodeAddr)

	// 开启读写双工协程
	go self.sReader(connect)
	go self.sWriter(connect)
}

// 为每个长连接开启读写两个协程
func (self *TP) cGoConn(conn net.Conn) {
	remoteAddr, connect := NewConnect(conn, self.connBufferLen, self.connWChanCap)
	self.connPool[remoteAddr] = connect
	// 绑定节点UID与conn
	nodeuid, nodeAddr := self.setNodesMap(connect)
	if nodeuid == "" || nodeAddr == "" {
		return
	}
	log.Printf(" *     —— 成功连接到服务器：%v (%v)——", nodeuid, nodeAddr)
	// 开启读写双工协程
	go self.cReader(connect)
	go self.cWriter(connect)
}

// 服务器读数据
func (self *TP) sReader(conn *Connect) {
	// 退出时关闭连接，删除连接池中的连接
	connkey := conn.RemoteAddr().String()
	defer func() {
		self.closeConn(connkey)
	}()

	for {
		// 设置连接超时
		conn.SetReadDeadline(time.Now().Add(self.timeout))
		// 等待读取数据
		if !self.read(conn) {
			break
		}
	}
}

// 客户端读数据
func (self *TP) cReader(conn *Connect) {
	// 退出时关闭连接，删除连接池中的连接
	connkey := conn.RemoteAddr().String()
	defer func() {
		self.closeConn(connkey)
	}()

	for {
		if !self.read(conn) {
			break
		}
	}
}

func (self *TP) read(conn *Connect) bool {
	read_len, err := conn.Read(conn.Buffer)
	if err != nil {
		return false
	}
	if read_len == 0 {
		return false // connection already closed by client
	}
	conn.TmpBuffer = append(conn.TmpBuffer, conn.Buffer[:read_len]...)
	self.Save(conn)
	return true
}

// 服务器发送数据
func (self *TP) sWriter(conn *Connect) {
	// 退出时关闭连接，删除连接池中的连接
	connkey := conn.RemoteAddr().String()
	defer func() {
		self.closeConn(connkey)
	}()
	for {
		data := <-conn.WriteChan
		self.Send(data)
	}
}

// 客户端发送数据
func (self *TP) cWriter(conn *Connect) {
	// 退出时关闭连接，删除连接池中的连接
	connkey := conn.RemoteAddr().String()
	defer func() {
		self.closeConn(connkey)
	}()
	i := 0
	for {
		timing := time.After(self.timeout)
		data := new(NetData)
		select {
		case data = <-conn.WriteChan:
		case <-timing:
			// 保持心跳
			data = NewNetData2(conn, HEARTBEAT, i)
		}
		self.Send(data)
	}
}

// 根据地址获取连接对象
func (self *TP) getConnByAddr(connKey string) *Connect {
	conn, ok := self.connPool[connKey]
	if !ok {
		// log.Printf("已与节点 %v 失去连接，无法完成发送请求！", connKey)
		return nil
	}
	return conn
}

// 根据节点UID获取连接对象
func (self *TP) getConnByUID(nodeuid string) *Connect {
	addr, ok := "", false
	for {
		addr, ok = self.nodesMap[nodeuid]
		if ok {
			break
		}
		time.Sleep(5e7)
	}
	return self.getConnByAddr(addr)
}

// 连接初始化，绑定节点与连接，默认key为节点ip
func (self *TP) setNodesMap(conn *Connect) (nodeuid, nodeAddr string) {
	if self.uid == "" {
		self.uid = conn.LocalAddr().String()
	}
	self.Send(NewNetData2(conn, IDENTITY, self.uid))
	if !self.read(conn) {
		return
	}
	data := <-conn.WriteChan
	// log.Println("收到身份信息：", data)
	nodeAddr = conn.RemoteAddr().String()
	nodeuid = data.Body.(string)
	if data.Operation == IDENTITY && nodeuid == "" {
		// 或者key为 strings.Split(conn.RemoteAddr().String(), ":")[0]
		nodeuid = nodeAddr
	}
	self.nodesMap[nodeuid] = nodeAddr
	return
}

// 关闭连接，退出协程
func (self *TP) closeConn(connkey string) {
	self.connPool[connkey].Close()
	delete(self.connPool, connkey)
	for k, v := range self.nodesMap {
		if v == connkey {
			delete(self.nodesMap, k)
			log.Printf(" *     —— 与节点 %v (%v) 断开连接！——", k, v)
			break
		}
	}
}

// 通信数据编码与发送
func (self *TP) Send(data *NetData) {
	d, err := json.Marshal(*data)
	if err != nil {
		log.Println("编码出错了", err)
		return
	}
	conn := self.getConnByAddr(data.To)
	if conn == nil {
		return
	}
	// 封包
	end := self.Packet(d)
	// 发送
	conn.Write(end)
	// log.Println("成功发送一条信息：", data)
}

// 解码收到的数据并存入缓存
func (self *TP) Save(conn *Connect) {
	// 解包
	dataSlice := make([][]byte, 10)
	dataSlice, conn.TmpBuffer = self.Unpack(conn.TmpBuffer)

	for _, data := range dataSlice {
		// js := map[string]interface{}{}
		// json.Unmarshal(data, &js)
		// log.Printf("接收信息为：%v", js)

		d := new(NetData)
		if err := json.Unmarshal(data, d); err == nil {
			// 修复缺失请求方地址的请求
			if d.From == "" {
				d.From = conn.RemoteAddr().String()
			}
			// 添加到读取缓存
			self.apiReadChan <- d
			// log.Printf("接收信息为：%v", d)
		}
	}
}

// 使用API并发处理请求
func (self *TP) apiHandle() {
	for {
		req := <-self.apiReadChan
		go func(req *NetData) {
			var conn *Connect

			operation, from, to := req.Operation, req.To, req.From
			fn, ok := self.api[operation]

			// 非法请求返回错误
			if !ok {
				self.autoErrorHandle(req, LLLEGAL, "您请求的API方法（"+req.Operation+"）不存在！", to)

				for k, v := range self.nodesMap {
					if v == to {
						log.Printf("非法请求：%v ，来自：%v (%v)", req.Operation, k, v)
					}
				}
				return
			}

			resp := fn(req)
			if resp == nil {
				return //continue
			}

			if resp.To == "" {
				resp.To = to
			} else if v, ok := self.nodesMap[resp.To]; ok {
				// 试探resp.To是否为nodeuid，并自动转换为连接地址
				resp.To = v
			}

			// 若指定节点连接不存在，则向原请求端返回错误
			if conn = self.getConnByAddr(resp.To); conn == nil {
				self.autoErrorHandle(req, FAILURE, "", to)
				return
			}

			// 默认指定与req相同的操作符
			if resp.Operation == "" {
				resp.Operation = operation
			}

			resp.From = from

			conn.WriteChan <- resp

		}(req)
	}
}

func (self *TP) autoErrorHandle(data *NetData, status int, msg string, reqFrom string) bool {
	oldConn := self.getConnByAddr(reqFrom)
	if oldConn == nil {
		return false
	}
	respErr := ReturnError(data, status, msg)
	respErr.To = reqFrom
	oldConn.WriteChan <- respErr
	return true
}

// 每隔一秒检查一次是否存在对端连接实例
// 计时90秒
// func (self *TP) waitReturn(addrStr string) (conn *Connect, ok bool) {
// 	tick := time.Tick(1e9)
// 	timeOut := time.After(9e10)
// 	for {
// 		conn = self.getConnByAddr(addrStr)
// 		if conn != nil {
// 			return conn, true
// 		}
// 		select {
// 		case <-tick:
// 		case <-timeOut:
// 			if conn == nil {
// 				return
// 			}
// 		}
// 	}
// }

// 强制设定系统保留的API
func (self *TP) reserveAPI() {
	// 添加保留规则——身份识别
	self.api[IDENTITY] = func(receive *NetData) *NetData {
		receive.From, receive.To = receive.To, receive.From
		return receive
	}
	// 添加保留规则——忽略心跳请求
	self.api[HEARTBEAT] = func(receive *NetData) *NetData { return nil }
}

// ***********************************************常用函数*************************************************** \\
// API中生成返回结果的方法
// operationAndNodeuid[0]参数为空时，系统将指定与对端相同的操作符
// operationAndNodeuid[1]参数为空时，系统将指定与对端为接收者
func ReturnData(body interface{}, operationAndNodeuid ...string) *NetData {
	data := &NetData{
		Status: SUCCESS,
		Body:   body,
	}
	if len(operationAndNodeuid) > 0 {
		data.Operation = operationAndNodeuid[0]
	}
	if len(operationAndNodeuid) > 1 {
		data.To = operationAndNodeuid[1]
	}
	return data
}

// 返回错误，receive建议为直接接收到的*NetData
func ReturnError(receive *NetData, status int, msg string, nodeuid ...string) *NetData {
	receive.Status = status
	receive.Body = msg
	if len(nodeuid) > 0 {
		receive.To = nodeuid[0]
	} else {
		receive.To = ""
	}
	return receive
}
