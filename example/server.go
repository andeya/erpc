package main

import (
	"github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/debug"
	"log"
	// "time"
)

// 有标识符UID的demo
// func main() {
// 	tp := teleport.New().SetUID("S")
// 	tp.SetAPI(teleport.API{
// 		"报到": func(receive *teleport.NetData) *teleport.NetData {
// 			log.Printf("报到：%v", receive.Body)
// 			resp := &teleport.NetData{
// 				Operation: "报到回复",
// 				Body:      "我是服务器，我已经收到你的来信",
// 			}
// 			return resp
// 		},
// 	})

// 	tp.Server(":20125")
// 	select {}
// }

// 无标识符UID的demo
var tp = teleport.New()

func main() {
	// 开启Teleport错误日志调试
	debug.Debug = true
	tp.SetUID("abc").SetAPI(teleport.API{
		"报到": new(报到),

		// 短链接不可以直接转发请求
		"短链接报到": new(短链接报到),
	}).Server(":20125")
	// time.Sleep(30e9)
	// tp.Close()
	select {}
}

type 报到 struct{}

func (*报到) Process(receive *teleport.NetData) *teleport.NetData {
	log.Printf("报到：%v", receive.Body)
	return teleport.ReturnData("服务器："+receive.From+"客户端已经报到！", "报到", "C3")
}

type 短链接报到 struct{}

func (*短链接报到) Process(receive *teleport.NetData) *teleport.NetData {
	log.Printf("报到：%v", receive.Body)
	tp.Request("服务器："+receive.From+"客户端已经报到！", "报到", "C3")
	return nil
}

// func main() {
// 	tp := teleport.New()
// 	tp.SetAPI(teleport.API{
// 		"报到": func(receive *teleport.NetData) *teleport.NetData {
// 			log.Printf("报到：%v", receive.Body)
// 			return teleport.ReturnData("我是服务器，我已经收到你的来信", "回复")
// 		},
// 	})

// 	tp.Server("")
// 	select {}
// }
