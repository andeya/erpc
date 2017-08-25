package main

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/teleport"
)

func main() {
	teleport.SetRawlogLevel("error")
	teleport.GraceSignal()
	teleport.SetShutdown(time.Second*20, nil, nil)
	var cfg = &teleport.PeerConfig{
		DefaultReadTimeout:  time.Minute * 1,
		DefaultWriteTimeout: time.Minute * 1,
		TlsCertFile:         "",
		TlsKeyFile:          "",
		SlowCometDuration:   time.Millisecond * 500,
		DefaultCodec:        "json",
		DefaultGzipLevel:    0,
		PrintBody:           false,
	}

	var peer = teleport.NewPeer(cfg)

	var sess, err = peer.Dial("127.0.0.1:9090", "simple_server:9090")
	if err != nil {
		teleport.Panicf("%v", err)
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
		teleport.Printf("------------------- call times: %d ok: %d fail: %d | cost time: %v | QPS: %d -----------------", times, uint32(times)-failNum, failNum, cost, time.Second*times/cost)
	}()

	for j := 0; j < loop; j++ {
		count.Add(group)
		for i := 0; i < group; i++ {
			go func() {
				var reply int
				var pullcmd = sess.Pull(
					"/group/home/test",
					[2]int{1, 2},
					&reply,
				)
				if pullcmd.Xerror != nil {
					atomic.AddUint32(&failNum, 1)
					teleport.Errorf("pull error: %v", pullcmd.Xerror.Error())
				}
				count.Done()
			}()
		}
		count.Wait()
	}
}
