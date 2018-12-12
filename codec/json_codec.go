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
	"encoding/json"
)

// json codec name and id
const (
	NAME_JSON = "json"
	ID_JSON   = 'j'
)

func init() {
	Reg(new(JSONCodec))
}

// JSONCodec json codec
type JSONCodec struct{}

// Name returns codec name.
func (JSONCodec) Name() string {
	return NAME_JSON
}

// ID returns codec id.
func (JSONCodec) ID() byte {
	return ID_JSON
}

// Marshal returns the JSON encoding of v.
func (JSONCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal parses the JSON-encoded data and stores the result
// in the value pointed to by v.
func (JSONCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
