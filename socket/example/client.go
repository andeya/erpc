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
	s := socket.GetSocket(conn)
	defer s.Close()
	var packet = socket.GetPacket(nil)
	defer socket.PutPacket(packet)
	for i := 0; i < 10; i++ {
		// write request
		packet.Reset(
			nil,
			socket.WithHeaderCodec("json"),
			socket.WithBodyCodec("json"),
		)
		packet.Header.Id = "1"
		packet.Header.Uri = "/a/b"
		packet.Header.Gzip = 5
		packet.Body = map[string]string{"a": "A"}
		err = s.WritePacket(packet)
		if err != nil {
			log.Printf("[CLI] write request err: %v", err)
			continue
		}
		log.Printf("[CLI] write request: %v", packet)

		// read response
		packet.Reset(func(_ *socket.Header) interface{} {
			return new(string)
		})
		err = s.ReadPacket(packet)
		if err != nil {
			log.Printf("[CLI] read response err: %v", err)
		} else {
			log.Printf("[CLI] read response: %v", packet)
		}
	}
}
