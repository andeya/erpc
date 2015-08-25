// Teleport是一款适用于分布式系统的高并发API框架，它采用socket全双工通信，实现S/C对等工作，支持长、短两种连接模式，支持断开后自动连接与手动断开连接，内部数据传输格式为JSON。
package teleport

import (
	"encoding/json"
	"log"
	"time"
)

// mode
const (
	SERVER = iota + 1
	CLIENT
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
	Client(serverAddr string, port string, isShort ...bool)
	// *主动推送信息，不写nodeuid默认随机发送给一个节点
	Request(body interface{}, operation string, nodeuid ...string)
	// 指定自定义的应用程序API
	SetAPI(api API) Teleport
	// 断开连接，参数为空则断开所有连接，服务器模式下还将停止监听
	Close(nodeuid ...string)

	// 设置客户端唯一标识符，默认为本节点ip:port，对服务器模式无效，服务器模式的UID强制为“Server”
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
	// 返回当前有效连接节点数
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
	// 长连接池，刚一连接时key为host:port形式，随后通过身份验证替换为UID
	connPool map[string]*Connect
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
	// 服务器模式专有成员
	*tpServer
	// 客户端模式专有成员
	*tpClient
}

// 每个API方法需判断stutas状态，并做相应处理
type API map[string]Handle

// 请求处理接口
type Handle interface {
	Process(*NetData) *NetData
}

// 创建接口实例，0为默认设置
func New() Teleport {
	return &TP{
		connPool:      make(map[string]*Connect),
		api:           API{},
		Protocol:      NewProtocol("henrylee2cn"),
		apiReadChan:   make(chan *NetData, 4096),
		connWChanCap:  2048,
		connBufferLen: 1024,
		tpServer:      new(tpServer),
		tpClient:      new(tpClient),
	}
}

// ***********************************************实现接口*************************************************** \\

// 指定应用程序API
func (self *TP) SetAPI(api API) Teleport {
	self.api = api
	return self
}

// *主动推送信息，直到有连接出现开始发送，不写nodeuid默认随机发送给一个节点
func (self *TP) Request(body interface{}, operation string, nodeuid ...string) {
	var conn *Connect
	var uid string
	if len(nodeuid) == 0 {
		for {
			if self.CountNodes() > 0 {
				break
			}
			time.Sleep(5e8)
		}
		// 一个随机节点的信息
		for uid, conn = range self.connPool {
			if conn.IsReady() {
				nodeuid = append(nodeuid, uid)
				break
			}
		}
	}
	// 等待并取得连接实例
	conn = self.getConn(nodeuid[0])
	for conn == nil || !conn.IsReady() {
		conn = self.getConn(nodeuid[0])
		time.Sleep(5e8)
	}
	conn.WriteChan <- NewNetData(self.uid, nodeuid[0], operation, body)
	// log.Println("添加一条请求：", conn.RemoteAddr().String(), operation, body)
}

// 断开连接，参数为空则断开所有连接，服务器模式下停止监听
func (self *TP) Close(nodeuid ...string) {
	if self.mode == CLIENT {
		self.tpClient.mustClose = true

	} else if self.mode == SERVER && self.tpServer.listener != nil {
		self.tpServer.listener.Close()
		log.Printf(" *     —— 服务器已终止监听 %v！ ——", self.port)
	}

	if len(nodeuid) == 0 {
		// 断开全部连接
		for uid, conn := range self.connPool {
			log.Printf(" *     —— 与节点 %v (%v) 断开连接！——", uid, conn.UID)
			delete(self.connPool, uid)
			conn.Close()
		}
		return
	}

	for _, uid := range nodeuid {
		conn := self.connPool[uid]
		delete(self.connPool, uid)
		conn.Close()
	}
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

// 返回当前有效连接节点数
func (self *TP) CountNodes() int {
	count := 0
	for _, conn := range self.connPool {
		if conn != nil && conn.IsReady() {
			count++
		}
	}
	return count
}

// ***********************************************公用方法*************************************************** \\

func (self *TP) read(conn *Connect) bool {
	read_len, err := conn.Read(conn.Buffer)
	if err != nil {
		return false
	}
	if read_len == 0 {
		return false // connection already closed by client
	}
	conn.TmpBuffer = append(conn.TmpBuffer, conn.Buffer[:read_len]...)
	self.save(conn)
	return true
}

// 根据uid获取连接对象
func (self *TP) getConn(nodeuid string) *Connect {
	return self.connPool[nodeuid]
}

// 根据uid获取连接对象地址
func (self *TP) getConnAddr(nodeuid string) string {
	conn := self.getConn(nodeuid)
	if conn == nil {
		// log.Printf("已与节点 %v 失去连接，无法完成发送请求！", nodeuid)
		return ""
	}
	return conn.RemoteAddr().String()
}

// 关闭连接，退出协程
func (self *TP) closeConn(nodeuid string, reconnect bool) {
	conn, ok := self.connPool[nodeuid]
	if !ok {
		return
	}

	// 是否允许自动重连
	if reconnect {
		self.connPool[nodeuid] = nil
	} else {
		delete(self.connPool, nodeuid)
	}

	if conn == nil {
		return
	}

	conn.Close()

	log.Printf(" *     —— 与节点 %v (%v) 断开连接！——", nodeuid, conn.RemoteAddr().String())
}

// 通信数据编码与发送
func (self *TP) send(data *NetData) {
	if data.From == "" {
		data.From = self.uid
	}

	d, err := json.Marshal(*data)
	if err != nil {
		log.Println("编码出错了", err)
		return
	}
	conn := self.getConn(data.To)
	if conn == nil {
		// log.Println("发送信息失败：", data)
		return
	}
	// 封包
	end := self.Packet(d)
	// 发送
	conn.Write(end)
	// log.Println("成功发送一条信息：", data)
}

// 解码收到的数据并存入缓存
func (self *TP) save(conn *Connect) {
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
				d.From = conn.UID
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
			handle, ok := self.api[operation]

			// 非法请求返回错误
			if !ok {
				self.autoErrorHandle(req, LLLEGAL, "您请求的API方法（"+req.Operation+"）不存在！", to)
				log.Printf("非法请求：%v ，来自：%v (%v)", req.Operation, to, self.getConnAddr(to))
				return
			}

			resp := handle.Process(req)
			if resp == nil {
				if conn = self.getConn(to); conn != nil && self.getConn(to).Short {
					self.closeConn(to, false)
				}
				return //continue
			}

			if resp.To == "" {
				resp.To = to
			}

			// 若指定节点连接不存在，则向原请求端返回错误
			if conn = self.getConn(resp.To); conn == nil {
				self.autoErrorHandle(req, FAILURE, "", to)
				return
			}

			// 默认指定与req相同的操作符
			if resp.Operation == "" {
				resp.Operation = operation
			}

			if resp.From == "" {
				resp.From = from
			}

			conn.WriteChan <- resp

		}(req)
	}
}

func (self *TP) autoErrorHandle(data *NetData, status int, msg string, reqFrom string) bool {
	oldConn := self.getConn(reqFrom)
	if oldConn == nil {
		return false
	}
	respErr := ReturnError(data, status, msg)
	respErr.From = self.uid
	respErr.To = reqFrom
	oldConn.WriteChan <- respErr
	return true
}

// 强制设定系统保留的API
func (self *TP) reserveAPI() {
	// 添加保留规则——身份识别
	self.api[IDENTITY] = identi
	// 添加保留规则——忽略心跳请求
	self.api[HEARTBEAT] = beat
}

var identi, beat = new(identity), new(heartbeat)

type identity struct{}

func (*identity) Process(receive *NetData) *NetData {
	receive.From, receive.To = receive.To, receive.From
	return receive
}

type heartbeat struct{}

func (*heartbeat) Process(receive *NetData) *NetData {
	return nil
}
