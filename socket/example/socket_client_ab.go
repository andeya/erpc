package main

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/erpc/v6/codec"
	"github.com/henrylee2cn/erpc/v6/socket"
	"github.com/henrylee2cn/erpc/v6/socket/example/pb"
)

//go:generate go build $GOFILE

func main() {
	socket.SetNoDelay(false)
	conn, err := net.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatalf("[CLI] dial err: %v", err)
	}
	s := socket.GetSocket(conn)
	defer s.Close()

	var count sync.WaitGroup
	t := time.Now()
	loop := 30
	group := 10000
	var failNum uint32
	defer func() {
		cost := time.Since(t)
		times := time.Duration(loop * group)
		log.Printf("------------------- call times: %d ok: %d fail: %d | cost time: %v | QPS: %d -----------------", times, uint32(times)-failNum, failNum, cost, time.Second*times/cost)
	}()

	for j := 0; j < loop; j++ {
		count.Add(group)
		for i := 0; i < group; i++ {
			go func(a int) {
				var message = socket.GetMessage()
				defer func() {
					socket.PutMessage(message)
					count.Done()
				}()
				// write request
				message.Reset()
				message.SetBodyCodec(codec.ID_PROTOBUF)
				message.SetSeq(int32(a))
				message.SetServiceMethod("/a/b")
				message.SetBody(&pb.PbTest{A: 10, B: 2})
				err = s.WriteMessage(message)
				if err != nil {
					atomic.AddUint32(&failNum, 1)
					log.Printf("[CLI] write request err: %v", err)
					return
				}

				// read response
				message.Reset(socket.WithNewBody(func(header socket.Header) interface{} {
					return new(pb.PbTest)
				}))
				err = s.ReadMessage(message)
				if err != nil {
					atomic.AddUint32(&failNum, 1)
					log.Printf("[CLI] read response err: %v", err)
				}
			}(i * group)
		}
		count.Wait()
	}
}
