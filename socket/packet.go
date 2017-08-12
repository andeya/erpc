// Socket package provides a concise, powerful and high-performance TCP socket.
//
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
	"encoding/json"
	"sync"

	"github.com/henrylee2cn/goutil"
)

var packetStack = new(struct {
	freePacket *Packet
	mu         sync.Mutex
})

// GetPacket gets a *Packet form packet stack.
// Note:
//  getBodyFunc is only for writing packet;
//  getBodyFunc should be nil when reading packet.
func GetPacket(getBodyFunc func(*Header) interface{}) *Packet {
	packetStack.mu.Lock()
	p := packetStack.freePacket
	if p == nil {
		p = newPacket(getBodyFunc)
	} else {
		packetStack.freePacket = p.next
		p.reset(getBodyFunc)
	}
	packetStack.mu.Unlock()
	return p
}

// PutPacket puts a *Packet to packet stack.
func PutPacket(p *Packet) {
	packetStack.mu.Lock()
	p.Body = nil
	p.next = packetStack.freePacket
	packetStack.freePacket = p
	packetStack.mu.Unlock()
}

// Packet provides header and body's containers for receiving and sending packet.
type Packet struct {
	// header content
	Header *Header `json:"header"`
	// body content
	Body interface{} `json:"body"`
	// header length
	HeaderLength int64 `json:"header_length"`
	// body length
	BodyLength int64 `json:"body_length"`
	// HeaderLength + BodyLength
	Length int64 `json:"length"`
	// get body by header
	// Note:
	//  only for writing packet;
	//  should be nil when reading packet.
	getBodyByHeader func(*Header) interface{} `json:"-"`
	next            *Packet                   `json:"-"`
}

func newPacket(getBodyFunc func(*Header) interface{}) *Packet {
	var p = &Packet{
		Header:          new(Header),
		getBodyByHeader: getBodyFunc,
	}
	return p
}

func (p *Packet) reset(getBodyFunc func(*Header) interface{}) {
	p.Header.Reset()
	p.Body = nil
	p.getBodyByHeader = getBodyFunc
	p.HeaderLength = 0
	p.BodyLength = 0
	p.Length = 0
}

func (p *Packet) loadBody() interface{} {
	p.Body = p.getBodyByHeader(p.Header)
	return p.Body
}

func (p *Packet) String() string {
	b, _ := json.MarshalIndent(p, "", "  ")
	return goutil.BytesToString(b)
}
