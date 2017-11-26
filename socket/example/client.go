package main

import (
	"log"
	"net"

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
	var packet = socket.GetPacket()
	defer socket.PutPacket(packet)
	for i := uint64(0); i < 1; i++ {
		// write request
		packet.Reset()
		packet.SetPtype(0)
		packet.SetBodyCodec(codec.ID_JSON)
		packet.SetSeq(i)
		packet.SetUri("/a/b")
		packet.SetBody(&pb.PbTest{A: 10, B: 2})
		err = s.WritePacket(packet)
		if err != nil {
			log.Printf("[CLI] write request err: %v", err)
			continue
		}
		log.Printf("[CLI] write request: %v", packet)

		// read response
		packet.Reset(socket.WithNewBody(
			func(seq uint64, ptype byte, uri string) interface{} {
				return new(pb.PbTest)
			}),
		)
		err = s.ReadPacket(packet)
		if err != nil {
			log.Printf("[CLI] read response err: %v", err)
		} else {
			log.Printf("[CLI] read response: %v", packet)
		}
	}
	// select {}
}
