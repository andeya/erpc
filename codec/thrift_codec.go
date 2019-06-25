// Copyright 2015-2018 HenryLee. All Rights Reserved.
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

package codec

import (
	"bytes"
	"context"
	"fmt"

	"git.apache.org/thrift.git/lib/go/thrift"
)

//  thrift codec name and id
const (
	NAME_THRIFT = "thrift"
	ID_THRIFT   = 't'
)

func init() {
	Reg(new(ThriftCodec))
}

// ThriftCodec thrift codec
type ThriftCodec struct{}

// Name returns codec name.
func (ThriftCodec) Name() string {
	return NAME_THRIFT
}

// ID returns codec id.
func (ThriftCodec) ID() byte {
	return ID_THRIFT
}

// Marshal returns the Thriftbuf encoding of v.
func (ThriftCodec) Marshal(v interface{}) ([]byte, error) {
	return ThriftMarshal(v)
}

// Unmarshal parses the Thriftbuf-encoded data and stores the result
// in the value pointed to by v.
func (ThriftCodec) Unmarshal(data []byte, v interface{}) error {
	return ThriftUnmarshal(data, v)
}

var (
	// ThriftEmptyStruct empty struct for thrift
	ThriftEmptyStruct = new(ThriftEmpty)
)

// ThriftMarshal returns the Thriftbuf encoding of v.
func ThriftMarshal(v interface{}) ([]byte, error) {
	trans := &tMemoryBuffer{bytes.NewBuffer(make([]byte, 0, 256))}
	p := thrift.NewTBinaryProtocol(trans, false, false)
	switch s := v.(type) {
	case thrift.TStruct:
		err := s.Write(p)
		return trans.Buffer.Bytes(), err
	case nil, *struct{}, struct{}:
		err := ThriftEmptyStruct.Write(p)
		return trans.Buffer.Bytes(), err
	}
	return nil, fmt.Errorf("thrift codec: %T does not implement thrift.TStruct", v)
}

// ThriftUnmarshal parses the Thriftbuf-encoded data and stores the result
// in the value pointed to by v.
func ThriftUnmarshal(data []byte, v interface{}) error {
	switch s := v.(type) {
	case thrift.TStruct:
		trans := &tMemoryBuffer{bytes.NewBuffer(data)}
		p := thrift.NewTBinaryProtocol(trans, false, false)
		return s.Read(p)
	case nil, *struct{}, struct{}:
		return nil
	}
	return fmt.Errorf("thrift codec: %T does not implement thrift.TStruct", v)
}

// tMemoryBuffer buffer-based implementation of the TTransport interface.
type tMemoryBuffer struct {
	*bytes.Buffer
}

func (*tMemoryBuffer) IsOpen() bool {
	return true
}

func (*tMemoryBuffer) Open() error {
	return nil
}

func (t *tMemoryBuffer) Close() error {
	t.Buffer.Reset()
	return nil
}

// Flushing a memory buffer is a no-op
func (*tMemoryBuffer) Flush(context.Context) error {
	return nil
}

func (t *tMemoryBuffer) RemainingBytes() uint64 {
	return uint64(t.Buffer.Len())
}
