package main

import (
	"log"
	"net"

	"github.com/henrylee2cn/teleport/packet"
)

func main() {
	// client
	conn, err := net.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatalf("[CLI] dial err: %v", err)
	}
	c := packet.WrapConn(conn)
	defer c.Close()
	for i := 0; i < 10; i++ {
		// write request
		header := &packet.Header{
			ID:    "1",
			URI:   "/a/b",
			Codec: "json",
			Gzip:  5,
		}
		reqBody := map[string]string{"a": "A"}
		// reqBody := "aA"
		n, err := c.Write(header, reqBody)
		if err != nil {
			log.Fatalf("[CLI] write request err: %v", err)
		}
		log.Printf("[CLI] write request len: %d, body: %#v", n, reqBody)

		// read response
		header, n, err = c.ReadHeader()
		if err != nil {
			log.Fatalf("[CLI] read response header err: %v", err)
		}
		log.Printf("[CLI] read response header len: %d, header: %#v", n, header)

		var respBody interface{}
		n, err = c.ReadBody(&respBody)
		if err != nil {
			log.Fatalf("[CLI] read response body err: %v", err)
		}
		log.Printf("[CLI] read response body len: %d, body: %#v", n, respBody)
	}
}
