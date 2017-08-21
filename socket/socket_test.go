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
			s := GetSocket(conn)
			defer s.Close()
			t.Logf("[SVR] s.LocalAddr(): %s, s.RemoteAddr(): %s", s.LocalAddr(), s.RemoteAddr())

			// read request
			var packet = GetPacket(func(_ *Header) interface{} {
				return new(map[string]string)
			})
			err = s.ReadPacket(packet)
			if err != nil {
				t.Fatalf("[SVR] read request err: %v", err)
			} else {
				t.Logf("[SVR] read request: %v", packet)
			}

			// write response
			packet.Header.StatusCode = 200
			packet.Header.Status = "ok"
			packet.Body = time.Now()
			err = s.WritePacket(packet)
			if err != nil {
				t.Fatalf("[SVR] write response err: %v", err)
			}
			t.Logf("[SVR] write response: %v", packet)
			PutPacket(packet)
		}
	}()

	time.Sleep(time.Second * 2)

	// client
	{
		conn, err := net.Dial("tcp", "127.0.0.1:8000")
		if err != nil {
			t.Fatalf("[CLI] dial err: %v", err)
		}
		s := GetSocket(conn)
		t.Logf("[CLI] s.LocalAddr(): %s, s.RemoteAddr(): %s", s.LocalAddr(), s.RemoteAddr())

		var packet = GetPacket(nil)

		// write request
		packet.HeaderCodec = "json"
		packet.BodyCodec = "json"
		packet.Header.Seq = 1
		packet.Header.Uri = "/a/b"
		packet.Header.Gzip = 5
		packet.Body = map[string]string{"a": "A"}
		err = s.WritePacket(packet)
		if err != nil {
			t.Fatalf("[CLI] write request err: %v", err)
		}
		t.Logf("[CLI] write request: %v", packet)

		// read response
		packet.Reset(func(_ *Header) interface{} {
			return new(string)
		})
		err = s.ReadPacket(packet)
		if err != nil {
			t.Fatalf("[CLI] read response err: %v", err)
		} else {
			t.Logf("[CLI] read response: %v", packet)
		}

		PutPacket(packet)
	}
}
