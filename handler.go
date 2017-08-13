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
	"reflect"
	"sync"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/errors"
)

type (
	Controller struct {
		name     string
		typ      reflect.Type  // type of the receiver
		val      reflect.Value // receiver of methods for the service
		handlers map[string]*Handler
		PluginContainer
	}

	Handler struct {
		structType   reflect.Type
		method       reflect.Method
		argType      reflect.Type
		replyType    reflect.Type
		defaultBytes []byte
		numCalls     uint64
		sync.Mutex   // protects counters
		PluginContainer
	}
)

func newController(ctrlStruct interface{}, pluginContainer PluginContainer) *Controller {
	if pluginContainer == nil {
		pluginContainer = NewPluginContainer()
	}
	c := &Controller{}
	return c
}

func (c *Controller) makeHandlers() error {
	c.handlers = make(map[string]*Handler)
	for m := 0; m < c.typ.NumMethod(); m++ {
		method := c.typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs two ins: receiver, *args.
		if mtype.NumIn() != 2 {
			return errors.Errorf("Handler: %s.%s needs one in argument, but have %d", c.typ.String(), mname, mtype.NumIn())
		}
		// Receiver need be a struct pointer.
		structType := mtype.In(0)
		if structType.Kind() != reflect.Ptr || structType.Elem().Kind() != reflect.Struct {
			return errors.Errorf("Handler: %s.%s receiver need be a struct pointer: %s", c.typ.String(), mname, structType)
		}
		// First arg need be exported or builtin, and need be a pointer.
		argType := mtype.In(1)
		if !goutil.IsExportedOrBuiltinType(argType) {
			return errors.Errorf("Handler: %s.%s args type not exported: %s", c.typ.String(), mname, argType)
		}
		if argType.Kind() != reflect.Ptr {
			return errors.Errorf("Handler: %s.%s args type need be a pointer: %s", c.typ.String(), mname, argType)
		}
		// Method needs two outs: reply error.
		if mtype.NumOut() != 2 {
			return errors.Errorf("Handler: %s.%s needs two out arguments, but have %d", c.typ.String(), mname, mtype.NumOut())
		}
		// First arg must be a pointer.
		replyType := mtype.Out(0)
		// Reply type must be exported.
		if !goutil.IsExportedOrBuiltinType(replyType) {
			return errors.Errorf("Handler: %s.%s first reply type not exported: %s", c.typ.String(), mname, replyType)
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(1); returnType != typeOfError {
			return errors.Errorf("Handler: %s.%s second reply type %s not *Error", c.typ.String(), mname, returnType)
		}
		c.handlers[mname] = &Handler{
			method:    method,
			argType:   argType,
			replyType: replyType,
		}
	}
	return nil
}

func (h *Handler) NewArg() reflect.Value {
	return reflect.New(h.argType.Elem())
}

// Precompute the reflect type for error. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfError = reflect.TypeOf((*Error)(nil)).Elem()

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
