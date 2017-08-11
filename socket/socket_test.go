// Copyright 2017 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package socket

import (
	"net"
	"testing"
	"time"
)

func TestSocket(t *testing.T) {
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
			s := Wrap(conn)
			defer s.Close()
			t.Logf("[SVR] s.LocalAddr(): %s, s.RemoteAddr(): %s", s.LocalAddr(), s.RemoteAddr())

			// read request
			var (
				header *Header
				body   interface{}
			)
			n, err := s.ReadPacket(func(h *Header) interface{} {
				header = h
				return &body
			})
			if err != nil {
				t.Fatalf("[SVR] read request err: %v", err)
			} else {
				t.Logf("[SVR] read request len: %d, header:%#v, body: %#v", n, header, body)
			}

			// write response
			header.StatusCode = 1
			header.Status = "test error"
			now := time.Now()
			n, err = s.WritePacket(header, now)
			if err != nil {
				t.Fatalf("[SVR] write response err: %v", err)
			}
			t.Logf("[SVR] write response len: %d, header: %#v, body: %#v", n, header, now)
		}
	}()

	time.Sleep(time.Second * 2)

	// client
	{
		socket, err := net.Dial("tcp", "127.0.0.1:8000")
		if err != nil {
			t.Fatalf("[CLI] dial err: %v", err)
		}
		s := Wrap(socket)
		t.Logf("[CLI] s.LocalAddr(): %s, s.RemoteAddr(): %s", s.LocalAddr(), s.RemoteAddr())

		// write request
		var header = &Header{
			Id:    "1",
			Uri:   "/a/b",
			Codec: "json",
			Gzip:  2,
		}
		// body := map[string]string{"a": "A"}
		var body interface{} = "aA"
		n, err := s.WritePacket(header, body)
		if err != nil {
			t.Fatalf("[CLI] write request err: %v", err)
		}
		t.Logf("[CLI] write request len: %d, header: %#v body: %#v", n, header, body)

		// read response
		n, err = s.ReadPacket(func(h *Header) interface{} {
			header = h
			return &body
		})
		if err != nil {
			t.Fatalf("[CLI] read response err: %v", err)
		} else {
			t.Logf("[CLI] read response len: %d, header:%#v, body: %#v", n, header, body)
		}
	}
}
