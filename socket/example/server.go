package main

import (
	"log"
	"net"

	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/socket/example/pb"
)

func main() {
	socket.SetPacketSizeLimit(512)
	lis, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		log.Fatalf("[SVR] listen err: %v", err)
	}
	log.Printf("listen tcp 0.0.0.0:8000")
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Fatalf("[SVR] accept err: %v", err)
		}
		go func(s socket.Socket) {
			log.Printf("accept %s", s.Id())
			defer s.Close()
			var pbTest = new(pb.PbTest)
			for {
				// read request
				var packet = socket.GetPacket(socket.WithNewBody(
					func(seq uint64, ptype byte, uri string) interface{} {
						*pbTest = pb.PbTest{}
						return pbTest
					}),
				)
				err = s.ReadPacket(packet)
				if err != nil {
					log.Printf("[SVR] read request err: %v", err)
					return
				} else {
					// log.Printf("[SVR] read request: %v", packet)
				}

				// write response
				pbTest.A = pbTest.A + pbTest.B
				pbTest.B = pbTest.A - pbTest.B*2
				packet.SetBody(pbTest)

				err = s.WritePacket(packet)
				if err != nil {
					log.Printf("[SVR] write response err: %v", err)
				} else {
					// log.Printf("[SVR] write response: %v", packet)
				}
				socket.PutPacket(packet)
			}
		}(socket.GetSocket(conn))
	}
}
