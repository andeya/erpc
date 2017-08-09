package main

import (
	"log"
	"net"

	"github.com/henrylee2cn/teleport"
)

func main() {
	// client
	conn, err := net.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatalf("[CLI] dial err: %v", err)
	}
	c := teleport.WrapConn(conn)
	defer c.Close()
	for i := 0; i < 10; i++ {
		// write request
		header := &teleport.Header{
			Id:    "1",
			Uri:   "/a/b",
			Codec: "json",
			Gzip:  5,
		}
		// body := map[string]string{"a": "A"}
		var body interface{} = "aA"
		n, err := c.WritePacket(header, body)
		if err != nil {
			log.Printf("[CLI] write request err: %v", err)
			continue
		}
		log.Printf("[CLI] write request len: %d, header: %#v body: %#v", n, header, body)

		// read response
		n, err = c.ReadPacket(func(h *teleport.Header) interface{} {
			header = h
			return &body
		})
		if err != nil {
			log.Printf("[CLI] read request err: %v", err)
		} else {
			log.Printf("[CLI] read request len: %d, header:%#v, body: %#v", n, header, body)
		}
	}
}
