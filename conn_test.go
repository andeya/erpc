package teleport

import (
	"net"
	"testing"
	"time"
)

func TestConn(t *testing.T) {
	// server
	go func() {
		lis, err := net.Listen("tcp", "0.0.0.0:8000")
		if err != nil {
			t.Fatalf("[SVR] listen err: %v", err)
		}
		for {
			conn, err := lis.Accept()
			if err != nil {
				t.Fatalf("[SVR] accept err: %v", err)
			}
			c := WrapConn(conn)
			t.Logf("[SVR] c.LocalAddr(): %s, c.RemoteAddr(): %s", c.LocalAddr(), c.RemoteAddr())
			// read request
			header, n, err := c.ReadHeader()
			if err != nil {
				t.Fatalf("[SVR] read request header err: %v", err)
			}
			t.Logf("[SVR] read request header len: %d, header: %#v", n, header)

			var body interface{}
			n, err = c.ReadBody(&body)
			if err != nil {
				t.Fatalf("[SVR] read request body err: %v", err)
			}
			t.Logf("[SVR] read request body len: %d, body: %#v", n, body)

			// write response
			header.Err = "test error"
			now := time.Now()
			n, err = c.WriteHeaderBody(header, now)
			if err != nil {
				t.Fatalf("[SVR] write response err: %v", err)
			}
			t.Logf("[SVR] write response len: %d, body: %#v", n, now)
		}
	}()

	time.Sleep(time.Second * 3)

	// client
	{
		conn, err := net.Dial("tcp", "127.0.0.1:8000")
		if err != nil {
			t.Fatalf("[CLI] dial err: %v", err)
		}
		c := WrapConn(conn)
		t.Logf("[CLI] c.LocalAddr(): %s, c.RemoteAddr(): %s", c.LocalAddr(), c.RemoteAddr())

		// write request
		header := &Header{
			ID:    "1",
			URI:   "/a/b",
			Codec: "json",
			Gzip:  2,
		}
		// body := map[string]string{"a": "A"}
		reqBody := "aA"
		n, err := c.WriteHeaderBody(header, reqBody)
		if err != nil {
			t.Fatalf("[CLI] write request err: %v", err)
		}
		t.Logf("[CLI] write request len: %d, body: %#v", n, reqBody)

		// read response
		header, n, err = c.ReadHeader()
		if err != nil {
			t.Fatalf("[CLI] read response header err: %v", err)
		}
		t.Logf("[CLI] read response header len: %d, header: %#v", n, header)

		var respBody interface{}
		n, err = c.ReadBody(&respBody)
		if err != nil {
			t.Fatalf("[CLI] read response body err: %v", err)
		}
		t.Logf("[CLI] read response body len: %d, body: %#v", n, respBody)
	}
}
