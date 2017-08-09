package main

import (
	"log"
	"net"
	"time"

	"github.com/henrylee2cn/teleport"
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
		go func(c teleport.Conn) {
			log.Printf("accept %s", c.Id())
			defer c.Close()
			for {
				// read request
				var (
					header *teleport.Header
					body   interface{}
				)
				n, err := c.ReadPacket(func(h *teleport.Header) interface{} {
					header = h
					return &body
				})
				if err != nil {
					log.Printf("[SVR] read request err: %v", err)
					return
				} else {
					log.Printf("[SVR] read request len: %d, header:%#v, body: %#v", n, header, body)
				}

				// write response
				header.StatusCode = 1
				header.Status = "test error"
				now := time.Now()
				n, err = c.WritePacket(header, now)
				if err != nil {
					log.Printf("[SVR] write response err: %v", err)
				}
				log.Printf("[SVR] write response len: %d, header: %#v, body: %#v", n, header, now)
			}
		}(teleport.WrapConn(conn))
	}
}
