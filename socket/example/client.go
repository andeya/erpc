package main

import (
	"log"
	"net"

	"github.com/henrylee2cn/teleport/socket"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatalf("[CLI] dial err: %v", err)
	}
	s := socket.Wrap(conn)
	defer s.Close()
	for i := 0; i < 10; i++ {
		// write request
		header := &socket.Header{
			Id:    "1",
			Uri:   "/a/b",
			Codec: "json",
			Gzip:  5,
		}
		var body interface{} = "ABC"
		n, err := s.WritePacket(header, body)
		if err != nil {
			log.Printf("[CLI] write request err: %v", err)
			continue
		}
		log.Printf("[CLI] write request len: %d, header: %#v body: %#v", n, header, body)

		// read response
		n, err = s.ReadPacket(func(h *socket.Header) interface{} {
			header = h
			return &body
		})
		if err != nil {
			log.Printf("[CLI] read response err: %v", err)
		} else {
			log.Printf("[CLI] read response len: %d, header:%#v, body: %#v", n, header, body)
		}
	}
}
