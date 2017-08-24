// Copyright 2015-2017 HenryLee. All Rights Reserved.
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
	"fmt"
	"io"

	"github.com/gogo/protobuf/proto"

	"github.com/henrylee2cn/teleport/utils"
)

func init() {
	Reg(new(ProtoCodec))
}

// ProtoCodec protobuf codec
type ProtoCodec struct{}

// Name returns codec name
func (p *ProtoCodec) Name() string {
	return "protobuf"
}

// Id returns codec id
func (p *ProtoCodec) Id() byte {
	return 'p'
}

// ProtoEncoder proto decoder
type ProtoEncoder struct {
	writer io.Writer
}

// NewEncoder returns a new protobuf encoder that writes to writer.
func (*ProtoCodec) NewEncoder(writer io.Writer) Encoder {
	return &ProtoEncoder{writer}
}

// Encode writes the Protobuf encoding of v to the writer.
func (p *ProtoEncoder) Encode(v interface{}) error {
	b, err := ProtoMarshal(v)
	if err != nil {
		return err
	}
	_, err = p.writer.Write(b)
	return err
}

// ProtoDecoder proto decoder
type ProtoDecoder struct {
	limitReader io.Reader
	buf         *bytes.Buffer
}

// NewDecoder returns a new protobuf decoder that reads from limit reader.
func (*ProtoCodec) NewDecoder(limitReader io.Reader) Decoder {
	return &ProtoDecoder{
		limitReader: limitReader,
		buf:         bytes.NewBuffer(make([]byte, 0, bytes.MinRead)),
	}
}

// Decode reads the next Protobuf-encoded value from its
// input and stores it in the value pointed to by v.
func (p *ProtoDecoder) Decode(v interface{}) error {
	p.buf.Reset()
	err := utils.ReadAll(p.limitReader, p.buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	return ProtoUnmarshal(p.buf.Bytes(), v)
}

var emptyStruct = struct{}{}

func ProtoMarshal(v interface{}) ([]byte, error) {
	if p, ok := v.(proto.Message); ok {
		return proto.Marshal(p)
	}
	if v == nil || v == emptyStruct {
		return proto.Marshal(Empty)
	}
	return nil, fmt.Errorf("%T does not implement proto.Message", v)
}

func ProtoUnmarshal(b []byte, v interface{}) error {
	if p, ok := v.(proto.Message); ok {
		return proto.Unmarshal(b, p)
	}
	if v == nil || v == emptyStruct {
		return nil
	}
	return fmt.Errorf("%T does not implement proto.Message", v)
}
