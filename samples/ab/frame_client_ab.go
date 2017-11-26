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
	tp.SetRawlogLevel("error")
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var cfg = &tp.PeerConfig{
		DefaultBodyCodec: "protobuf",
	}

	var peer = tp.NewPeer(cfg)

	var sess, err = peer.Dial("127.0.0.1:9090")
	if err != nil {
		tp.Panicf("%v", err)
	}

	var count sync.WaitGroup
	t := time.Now()
	loop := 30
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
				var reply = new(pb.PbTest)
				var pullcmd = sess.Pull(
					"/group/home/test",
					&pb.PbTest{A: 10, B: 2},
					reply,
				)
				if pullcmd.Rerror() != nil {
					atomic.AddUint32(&failNum, 1)
					tp.Errorf("pull error: %v", pullcmd.Rerror())
				}
				count.Done()
			}()
		}
		count.Wait()
	}
}
