package main

import (
	"log"
	"net"
	"time"

	"github.com/henrylee2cn/teleport/socket"
)

func main() {
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
			for {
				// read request
				var packet = socket.GetPacket(func(_ *socket.Header) interface{} {
					return new(map[string]string)
				})
				err = s.ReadPacket(packet)
				if err != nil {
					log.Printf("[SVR] read request err: %v", err)
					return
				} else {
					log.Printf("[SVR] read request: %v", packet)
				}

				// write response
				packet.HeaderCodec = "json"
				packet.BodyCodec = "json"
				packet.Header.StatusCode = 200
				packet.Header.Status = "ok"
				packet.Body = time.Now()
				err = s.WritePacket(packet)
				if err != nil {
					log.Printf("[SVR] write response err: %v", err)
				}
				log.Printf("[SVR] write response: %v", packet)
				socket.PutPacket(packet)
			}
		}(socket.GetSocket(conn))
	}
}
