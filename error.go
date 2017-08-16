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

package teleport

import (
	"encoding/json"

	"github.com/henrylee2cn/goutil"
)

// Error error for Handler.
type Error interface {
	// return error code
	Code() uint16
	// return error text
	Text() string
	// return json string, implement error interface
	Error() string
}

// NewError creates a new Error interface.
func NewError(code uint16, text string) Error {
	return &err{
		code: code,
		text: text,
	}
}

type err struct {
	code uint16
	text string
	json string
}

func (e *err) Code() uint16 {
	return e.code
}

func (e *err) Text() string {
	return e.text
}

func (e *err) Error() string {
	if len(e.json) == 0 {
		b, _ := json.Marshal(e)
		e.json = goutil.BytesToString(b)
	}
	return e.json
}
