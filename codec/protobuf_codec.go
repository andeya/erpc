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
	"fmt"

	"github.com/gogo/protobuf/proto"
)

//  protobuf codec name and id
const (
	NAME_PROTOBUF = "protobuf"
	ID_PROTOBUF   = 'p'
)

func init() {
	Reg(new(ProtoCodec))
}

// ProtoCodec protobuf codec
type ProtoCodec struct{}

// Name returns codec name.
func (ProtoCodec) Name() string {
	return NAME_PROTOBUF
}

// Id returns codec id.
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

var (
	// EmptyStruct empty struct for protobuf
	EmptyStruct = new(PbEmpty)
)

// ProtoMarshal returns the Protobuf encoding of v.
func ProtoMarshal(v interface{}) ([]byte, error) {
	if p, ok := v.(proto.Message); ok {
		return proto.Marshal(p)
	}
	switch v.(type) {
	case nil, *struct{}, struct{}:
		return proto.Marshal(EmptyStruct)
	}
	return nil, fmt.Errorf("protobuf codec: %T does not implement proto.Message", v)
}

// ProtoUnmarshal parses the Protobuf-encoded data and stores the result
// in the value pointed to by v.
func ProtoUnmarshal(data []byte, v interface{}) error {
	if p, ok := v.(proto.Message); ok {
		return proto.Unmarshal(data, p)
	}
	switch v.(type) {
	case nil, *struct{}, struct{}:
		return nil
	}
	return fmt.Errorf("protobuf codec: %T does not implement proto.Message", v)
}
