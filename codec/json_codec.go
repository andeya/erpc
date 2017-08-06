// Copyright 2016 HenryLee. All Rights Reserved.
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
	"io"
)

func init() {
	RegEncodeMaker("json", NewJsonEncoder)
	RegDecodeMaker("json", NewJsonDecoder)
}

// NewJsonEncoder returns a new json encoder that writes to w.
func NewJsonEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

// NewJsonDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may
// read data from r beyond the JSON values requested.
func NewJsonDecoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}
