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
	"encoding/xml"
)

// xml codec name and id
const (
	NAME_XML = "xml"
	ID_XML   = 'x'
)

func init() {
	Reg(new(XMLCodec))
}

// XMLCodec xml codec
type XMLCodec struct{}

// Name returns codec name.
func (XMLCodec) Name() string {
	return NAME_XML
}

// ID returns codec id.
func (XMLCodec) ID() byte {
	return ID_XML
}

// Marshal returns the XML encoding of v.
func (XMLCodec) Marshal(v interface{}) ([]byte, error) {
	return xml.Marshal(v)
}

// Unmarshal parses the XML-encoded data and stores the result
// in the value pointed to by v.
func (XMLCodec) Unmarshal(data []byte, v interface{}) error {
	return xml.Unmarshal(data, v)
}
