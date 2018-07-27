package main

import (
	// "net/http"
	// _ "net/http/pprof"
	"sync"
	"sync/atomic"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/socket/example/pb"
)

func main() {
	// go func() {
	// 	http.ListenAndServe("0.0.0.0:9092", nil)
	// }()
	tp.SetSocketNoDelay(false)
	tp.SetLoggerLevel("WARNING")
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var peer = tp.NewPeer(tp.PeerConfig{
		DefaultBodyCodec:   "protobuf",
		DefaultDialTimeout: time.Second * 5,
	})

	var sess, rerr = peer.Dial("127.0.0.1:9090")
	if rerr != nil {
		tp.Panicf("%v", rerr)
	}

	var count sync.WaitGroup
	t := time.Now()
	loop := 10
	group := 10000
	var failNum uint32
	defer func() {
		peer.Close()
		cost := time.Since(t)
		times := time.Duration(loop * group)
		tp.Printf("------------------- call times: %d ok: %d fail: %d | cost time: %v | QPS: %d -----------------", times, uint32(times)-failNum, failNum, cost, time.Second*times/cost)
	}()

	for j := 0; j < loop; j++ {
		count.Add(group)
		for i := 0; i < group; i++ {
			go func() {
				defer count.Done()
				var result = new(pb.PbTest)
				if rerr := sess.Call(
					"/group/home/test",
					&pb.PbTest{A: 10, B: 2},
					result,
				).Rerror(); rerr != nil {
					atomic.AddUint32(&failNum, 1)
					tp.Errorf("call error: %v", rerr)
				}
			}()
		}
		count.Wait()
	}
}
