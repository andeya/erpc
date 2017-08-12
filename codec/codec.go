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
	"io"
)

type (
	// Codec makes Encoder and Decoder
	Codec interface {
		NewEncoder(io.Writer) Encoder
		NewDecoder(io.Reader) Decoder
	}
	// Encoder encodes data
	Encoder interface {
		Encode(v interface{}) error
	}
	// Decoder decodes data
	Decoder interface {
		Decode(v interface{}) error
	}
)

var codecMap = make(map[string]Codec)

// Reg registers Codec
func Reg(name string, codec Codec) {
	if _, ok := codecMap[name]; ok {
		panic("multi-register codec: " + name)
	}
	codecMap[name] = codec
}

// Get returns Codec
func Get(name string) (Codec, error) {
	codec, ok := codecMap[name]
	if !ok {
		return nil, fmt.Errorf("unsupported codec: %s", name)
	}
	return codec, nil
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(name string, w io.Writer) (Encoder, error) {
	codec, err := Get(name)
	if err != nil {
		return nil, err
	}
	return codec.NewEncoder(w), nil
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(name string, r io.Reader) (Decoder, error) {
	codec, err := Get(name)
	if err != nil {
		return nil, err
	}
	return codec.NewDecoder(r), nil
}
