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
	"io/ioutil"

	"github.com/henrylee2cn/goutil"
)

func init() {
	Reg(new(StringCodec))
}

// StringCodec string codec
type StringCodec struct{}

// Name returns codec name
func (p *StringCodec) Name() string {
	return "string"
}

// Id returns codec id
func (p *StringCodec) Id() byte {
	return 's'
}

// StringEncoder string decoder
type StringEncoder struct {
	writer io.Writer
}

// NewEncoder returns a new string encoder that writes to writer.
func (*StringCodec) NewEncoder(writer io.Writer) Encoder {
	return &StringEncoder{writer}
}

// Encode writes the string encoding of v to the writer.
func (p *StringEncoder) Encode(v interface{}) error {
	var b []byte
	switch s := v.(type) {
	case nil:
	case string:
		b = goutil.StringToBytes(s)
	case *string:
		b = goutil.StringToBytes(*s)
	case []byte:
		b = s
	case *[]byte:
		b = *s
	default:
		return fmt.Errorf("%T can not be directly converted to []byte type", v)
	}
	_, err := p.writer.Write(b)
	return err
}

// StringDecoder string decoder
type StringDecoder struct {
	limitReader io.Reader
}

// NewDecoder returns a new string decoder that reads from limit reader.
func (*StringCodec) NewDecoder(limitReader io.Reader) Decoder {
	return &StringDecoder{
		limitReader: limitReader,
	}
}

// Decode reads the next string-encoded value from its
// input and stores it in the value pointed to by v.
func (p *StringDecoder) Decode(v interface{}) error {
	b, err := ioutil.ReadAll(p.limitReader)
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	switch s := v.(type) {
	case nil:
		return nil
	case *string:
		*s = goutil.BytesToString(b)
	case []byte:
		copy(s, b)
	case *[]byte:
		*s = b
	default:
		return fmt.Errorf("[]byte can not be directly converted to %T type", v)
	}
	return nil
}
