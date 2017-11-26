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
)

type (
	// Codec makes Encoder and Decoder
	Codec interface {
		// Id returns codec id.
		Id() byte
		// Name returns codec name.
		Name() string
		// Marshal returns the encoding of v.
		Marshal(v interface{}) ([]byte, error)
		// Unmarshal parses the encoded data and stores the result
		// in the value pointed to by v.
		Unmarshal(data []byte, v interface{}) error
	}
)

var codecMap = struct {
	nameMap map[string]Codec
	idMap   map[byte]Codec
}{
	nameMap: make(map[string]Codec),
	idMap:   make(map[byte]Codec),
}

const (
	NilCodecId   byte   = 0
	NilCodecName string = ""
)

// Reg registers Codec
func Reg(codec Codec) {
	if codec.Id() == NilCodecId {
		panic(fmt.Sprintf("codec id can not be %d", NilCodecId))
	}
	if _, ok := codecMap.nameMap[codec.Name()]; ok {
		panic("multi-register codec name: " + codec.Name())
	}
	if _, ok := codecMap.idMap[codec.Id()]; ok {
		panic(fmt.Sprintf("multi-register codec id: %d", codec.Id()))
	}
	codecMap.nameMap[codec.Name()] = codec
	codecMap.idMap[codec.Id()] = codec
}

// Get returns Codec
func Get(id byte) (Codec, error) {
	codec, ok := codecMap.idMap[id]
	if !ok {
		return nil, fmt.Errorf("unsupported codec id: %d", id)
	}
	return codec, nil
}

// GetByName returns Codec
func GetByName(name string) (Codec, error) {
	codec, ok := codecMap.nameMap[name]
	if !ok {
		return nil, fmt.Errorf("unsupported codec name: %s", name)
	}
	return codec, nil
}
