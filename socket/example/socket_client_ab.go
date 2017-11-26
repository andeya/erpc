package main

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/socket/example/pb"
)

func main() {
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
			go func(a uint64) {
				var packet = socket.GetPacket()
				defer func() {
					socket.PutPacket(packet)
					count.Done()
				}()
				// write request
				packet.Reset()
				packet.SetBodyCodec(codec.ID_PROTOBUF)
				packet.SetSeq(a)
				packet.SetUri("/a/b")
				packet.SetBody(&pb.PbTest{A: 10, B: 2})
				err = s.WritePacket(packet)
				if err != nil {
					atomic.AddUint32(&failNum, 1)
					log.Printf("[CLI] write request err: %v", err)
					return
				}

				// read response
				packet.Reset(socket.WithNewBody(func(seq uint64, ptype byte, uri string) interface{} {
					return new(pb.PbTest)
				}))
				err = s.ReadPacket(packet)
				if err != nil {
					atomic.AddUint32(&failNum, 1)
					log.Printf("[CLI] read response err: %v", err)
				}
			}(uint64(i * group))
		}
		count.Wait()
	}
}
