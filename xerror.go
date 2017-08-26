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

package tp

import (
	"encoding/json"

	"github.com/henrylee2cn/goutil"
)

// Xerror error for Handler.
type Xerror interface {
	// return error code
	Code() int32
	// return error text
	Text() string
	// return json string, implement error interface
	Error() string
}

// NewXerror creates a new Error interface.
func NewXerror(code int32, text string) Xerror {
	return &xerr{
		Cod: code,
		Txt: text,
	}
}

type xerr struct {
	Cod  int32  `json:"code"`
	Txt  string `json:"text"`
	json string `json:"-"`
}

func (e *xerr) Code() int32 {
	return e.Cod
}

func (e *xerr) Text() string {
	return e.Txt
}

func (e *xerr) Error() string {
	if len(e.json) == 0 {
		b, _ := json.Marshal(e)
		e.json = goutil.BytesToString(b)
	}
	return e.json
}
