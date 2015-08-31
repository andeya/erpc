package main

import (
	"github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/debug"
	"log"
	// "time"
)

// 有标识符UID的demo，保证了客户端链接唯一性
func main() {
	// 开启Teleport错误日志调试
	debug.Debug = true
	tp := teleport.New().SetUID("", "abc").SetAPI(teleport.API{
		"报到": new(报到),
	})
	tp.Client("127.0.0.1", ":20125")
	select {}
}

type 报到 struct{}

func (*报到) Process(receive *teleport.NetData) *teleport.NetData {
	if receive.Status == teleport.SUCCESS {
		log.Printf("%v", receive.Body)
	}
	return nil
}

// 有标识符UID的demo，保证了客户端链接唯一性
// func main() {
// 	tp := teleport.New().SetUID("C").SetAPI(teleport.API{
// 		"回复": func(receive *teleport.NetData) *teleport.NetData {
// 			log.Printf("回复：%v", receive.Body)
// 			return nil
// 		},
// 	})
// 	tp.Client("127.0.0.1", ":20125")
// 	tp.Request("我是客户端，我来报个到")
// 	select {}
// }

// 无标识符UID的demo
// func main() {
// 	tp := teleport.New().SetAPI(teleport.API{
// 		"报到回复": func(receive *teleport.NetData) *teleport.NetData {
// 			log.Printf("报到回复：%v", receive.Body)
// 			return nil
// 		},
// 	})
// 	tp.Client("127.0.0.1", ":20125")
// 	tp.Request("报到", "我是客户端，我来报个到")
// 	select {}
// }
