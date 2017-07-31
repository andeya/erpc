package main

import (
	"log"
	"net"
	"time"

	"github.com/henrylee2cn/teleport/packet"
)

func main() {
	// server
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
		log.Printf("accept %s", conn.RemoteAddr().String())
		go func(c packet.Conn) {
			for {
				// read request
				header, n, err := c.ReadHeader()
				if err != nil {
					log.Printf("[SVR] read request header err: %v", err)
					return
				}
				log.Printf("[SVR] read request header len: %d, header: %#v", n, header)

				var body interface{}
				n, err = c.ReadBody(&body)
				if err != nil {
					log.Printf("[SVR] read request body err: %v", err)
					return
				}
				log.Printf("[SVR] read request body len: %d, body: %#v", n, body)

				// write response
				header.Err = "test error"
				now := time.Now()
				n, err = c.Write(header, now)
				if err != nil {
					log.Printf("[SVR] write response err: %v", err)
					return
				}
				log.Printf("[SVR] write response len: %d, body: %#v", n, now)
			}
		}(packet.WrapConn(conn))
	}
}
