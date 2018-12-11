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
	"reflect"
	"strconv"

	"github.com/henrylee2cn/goutil"
)

// plain text codec name and id
const (
	NAME_PLAIN = "plain"
	ID_PLAIN   = 's'
)

func init() {
	Reg(new(PlainCodec))
}

// PlainCodec plain text codec
type PlainCodec struct{}

// Name returns codec name.
func (PlainCodec) Name() string {
	return NAME_PLAIN
}

// ID returns codec id.
func (PlainCodec) ID() byte {
	return ID_PLAIN
}

// Marshal returns the string encoding of v.
func (PlainCodec) Marshal(v interface{}) ([]byte, error) {
	var b []byte
	switch vv := v.(type) {
	case nil:
	case string:
		b = goutil.StringToBytes(vv)
	case *string:
		b = goutil.StringToBytes(*vv)
	case []byte:
		b = vv
	case *[]byte:
		b = *vv
	default:
		s, ok := formatProperType(reflect.ValueOf(v))
		if !ok {
			return nil, fmt.Errorf("plain codec: %T can not be directly converted to []byte type", v)
		}
		b = goutil.StringToBytes(s)
	}
	return b, nil
}

func formatProperType(v reflect.Value) (string, bool) {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.String:
		return v.String(), true
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), true
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'f', -1, 32), true
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), true
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return goutil.BytesToString(v.Bytes()), true
		} else {
			return "", false
		}
	case reflect.Invalid:
		return "", true
	default:
		return "", false
	}
}

// Unmarshal parses the string-encoded data and stores the result
// in the value pointed to by v.
func (PlainCodec) Unmarshal(data []byte, v interface{}) error {
	switch s := v.(type) {
	case nil:
		return nil
	case *string:
		*s = string(data)
	case []byte:
		copy(s, data)
	case *[]byte:
		if length := len(data); cap(*s) < length {
			*s = make([]byte, length)
		} else {
			*s = (*s)[:length]
		}
		copy(*s, data)
	default:
		if !parseProperType(data, reflect.ValueOf(v)) {
			return fmt.Errorf("plain codec: []byte can not be directly converted to %T type", v)
		}
	}
	return nil
}

func parseProperType(data []byte, v reflect.Value) bool {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.CanSet() {
		return false
	}
	s := goutil.BytesToString(data)
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Bool:
		bol, err := strconv.ParseBool(s)
		if err != nil {
			return false
		}
		v.SetBool(bol)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		d, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return false
		}
		v.SetInt(d)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		d, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return false
		}
		v.SetUint(d)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return false
		}
		v.SetFloat(f)
	case reflect.Slice:
		if v.Type().Elem().Kind() != reflect.Uint8 {
			return false
		}
		v.SetBytes(data)
	case reflect.Invalid:
		return true
	default:
		return false
	}
	return true
}
