package socket

import (
	"log"
	"net"
	"testing"

	"github.com/andeya/erpc/v7/socket"
	"github.com/andeya/erpc/v7/socket/example/pb"
	"github.com/andeya/goutil"
)

//go:generate go test -v -c -o "${GOPACKAGE}_server" $GOFILE

func TestServer(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}
	socket.SetNoDelay(false)
	socket.SetMessageSizeLimit(512)
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
			log.Printf("accept %s", s.ID())
			defer s.Close()
			var pbTest = new(pb.PbTest)
			for {
				// read request
				var message = socket.GetMessage(socket.WithNewBody(
					func(header socket.Header) interface{} {
						*pbTest = pb.PbTest{}
						return pbTest
					}),
				)
				err = s.ReadMessage(message)
				if err != nil {
					log.Printf("[SVR] read request err: %v", err)
					return
					// } else {
					// log.Printf("[SVR] read request: %s", message.String())
				}

				// write response
				pbTest.A = pbTest.A + pbTest.B
				pbTest.B = pbTest.A - pbTest.B*2
				message.SetBody(pbTest)

				err = s.WriteMessage(message)
				if err != nil {
					log.Printf("[SVR] write response err: %v", err)
					// } else {
					// log.Printf("[SVR] write response: %s", message.String())
				}
				socket.PutMessage(message)
			}
		}(socket.GetSocket(conn))
	}
}
