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
	"fmt"

	"github.com/gogo/protobuf/proto"
)

//  protobuf codec id
const (
	NAME_PROTOBUF = "protobuf"
	ID_PROTOBUF   = 'p'
)

func init() {
	Reg(new(ProtoCodec))
}

// ProtoCodec protobuf codec
type ProtoCodec struct{}

// Name returns codec string
func (ProtoCodec) Name() string {
	return NAME_PROTOBUF
}

// Id returns codec id
func (ProtoCodec) Id() byte {
	return ID_PROTOBUF
}

// Marshal returns the Protobuf encoding of v.
func (ProtoCodec) Marshal(v interface{}) ([]byte, error) {
	return ProtoMarshal(v)
}

// Unmarshal parses the Protobuf-encoded data and stores the result
// in the value pointed to by v.
func (ProtoCodec) Unmarshal(data []byte, v interface{}) error {
	return ProtoUnmarshal(data, v)
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

func ProtoUnmarshal(data []byte, v interface{}) error {
	if p, ok := v.(proto.Message); ok {
		return proto.Unmarshal(data, p)
	}
	if v == nil || v == emptyStruct {
		return nil
	}
	return fmt.Errorf("%T does not implement proto.Message", v)
}
