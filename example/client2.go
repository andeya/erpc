package main

import (
	"github.com/henrylee2cn/teleport"
	"log"
	// "time"
)

// 有标识符UID的demo，保证了客户端链接唯一性
func main() {
	tp := teleport.New().SetUID("C2").SetAPI(teleport.API{
		"报到": func(receive *teleport.NetData) *teleport.NetData {
			log.Printf("%v", receive.Body)
			return nil
		},
	})
	tp.Client("127.0.0.1", ":20125")
	select {}
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
