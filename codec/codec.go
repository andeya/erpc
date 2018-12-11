// Package codec is the body's codec set.
//
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
//
package codec

import (
	"fmt"
)

// Codec makes the body's Encoder and Decoder
type Codec interface {
	// ID returns codec id.
	ID() byte
	// Name returns codec name.
	Name() string
	// Marshal returns the encoding of v.
	Marshal(v interface{}) ([]byte, error)
	// Unmarshal parses the encoded data and stores the result
	// in the value pointed to by v.
	Unmarshal(data []byte, v interface{}) error
}

var codecMap = struct {
	idMap   map[byte]Codec
	nameMap map[string]Codec
}{
	idMap:   make(map[byte]Codec),
	nameMap: make(map[string]Codec),
}

const (
	// NilCodecID empty codec id.
	NilCodecID byte = 0
	// NilCodecName empty codec name.
	NilCodecName string = ""
)

// Reg registers Codec.
func Reg(codec Codec) {
	if codec.ID() == NilCodecID {
		panic(fmt.Sprintf("codec id can not be %d", NilCodecID))
	}
	if _, ok := codecMap.idMap[codec.ID()]; ok {
		panic(fmt.Sprintf("multi-register codec id: %d", codec.ID()))
	}
	if _, ok := codecMap.nameMap[codec.Name()]; ok {
		panic("multi-register codec name: " + codec.Name())
	}
	codecMap.idMap[codec.ID()] = codec
	codecMap.nameMap[codec.Name()] = codec
}

// Get returns Codec by id.
func Get(codecID byte) (Codec, error) {
	codec, ok := codecMap.idMap[codecID]
	if !ok {
		return nil, fmt.Errorf("unsupported codec id: %d", codecID)
	}
	return codec, nil
}

// GetByName returns Codec by name.
func GetByName(codecName string) (Codec, error) {
	codec, ok := codecMap.nameMap[codecName]
	if !ok {
		return nil, fmt.Errorf("unsupported codec name: %s", codecName)
	}
	return codec, nil
}

// Marshal returns the encoding of v.
func Marshal(codecID byte, v interface{}) ([]byte, error) {
	codec, err := Get(codecID)
	if err != nil {
		return nil, err
	}
	return codec.Marshal(v)
}

// Unmarshal parses the encoded data and stores the result
// in the value pointed to by v.
func Unmarshal(codecID byte, data []byte, v interface{}) error {
	codec, err := Get(codecID)
	if err != nil {
		return err
	}
	return codec.Unmarshal(data, v)
}

// MarshalByName returns the encoding of v.
func MarshalByName(codecName string, v interface{}) ([]byte, error) {
	codec, err := GetByName(codecName)
	if err != nil {
		return nil, err
	}
	return codec.Marshal(v)
}

// UnmarshalByName parses the encoded data and stores the result
// in the value pointed to by v.
func UnmarshalByName(codecName string, data []byte, v interface{}) error {
	codec, err := GetByName(codecName)
	if err != nil {
		return err
	}
	return codec.Unmarshal(data, v)
}
