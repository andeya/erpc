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
	"path"
	"reflect"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/errors"
)

// Router the router of request or push.
type (
	Router struct {
		handlers map[string]*Handler
		// only for register router
		pathPrefix string
		plugins    []Plugin
		typ        string
		fn         HandlersMaker
	}
	// Handler request or push handler type info
	Handler struct {
		name            string
		originStruct    reflect.Type
		method          reflect.Method
		arg             reflect.Type
		reply           reflect.Type // only for request handler doc
		pluginContainer PluginContainer
	}
	// HandlersMaker makes []*Handler
	HandlersMaker func(string, interface{}, PluginContainer) ([]*Handler, error)
)

// Group add handler group.
func (r *Router) Group(pathPrefix string, plugins ...Plugin) *Router {
	ps := make([]Plugin, len(r.plugins)+len(plugins))
	copy(ps, r.plugins)
	copy(ps[len(r.plugins):], plugins)
	return &Router{
		handlers:   r.handlers,
		pathPrefix: path.Join(r.pathPrefix, pathPrefix),
		plugins:    ps,
	}
}

// Reg registers handler.
func (r *Router) Reg(pathPrefix string, ctrlStruct interface{}, plugin ...Plugin) {
	handlers, err := r.fn(
		path.Join(r.pathPrefix, pathPrefix),
		ctrlStruct,
		&pluginContainer{append(r.plugins, plugin...)},
	)
	if err != nil {
		Fatalf("%v", err)
	}
	for _, requestHandler := range handlers {
		name := path.Join(pathPrefix, requestHandler.name)
		if _, ok := r.handlers[name]; ok {
			Fatalf("There is a %s handler conflict: %s", r.typ, name)
		}
		r.handlers[name] = requestHandler
		Printf("register %s handler: %s", r.typ, name)
	}
}

func (r *Router) get(uriPath string) (*Handler, bool) {
	t, ok := r.handlers[uriPath]
	return t, ok
}

// newRequestRouter creates request packet router.
// Note: ctrlStruct needs to implement RequestCtx interface.
func newRequestRouter() *Router {
	return &Router{
		handlers:   make(map[string]*Handler),
		pathPrefix: "/",
		plugins:    make([]Plugin, 0),
		fn:         requestHandlersMaker,
		typ:        "request",
	}
}

// Precompute the reflect type for Xerror interface. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfError = reflect.TypeOf((Xerror)(nil))

// Note: ctrlStruct needs to implement RequestCtx interface.
func requestHandlersMaker(pathPrefix string, ctrlStruct interface{}, pluginContainer PluginContainer) ([]*Handler, error) {
	var (
		ctype    = reflect.TypeOf(ctrlStruct)
		handlers = make([]*Handler, 0, 1)
	)
	if _, ok := ctrlStruct.(RequestCtx); !ok {
		return nil, errors.Errorf("register request handler: the type is not implemented RequestCtx interface: %s", ctype.String())
	}
	if pluginContainer == nil {
		pluginContainer = newPluginContainer()
	}
	for m := 0; m < ctype.NumMethod(); m++ {
		method := ctype.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs two ins: receiver, *args.
		if mtype.NumIn() != 2 {
			return nil, errors.Errorf("register request handler: %s.%s needs one in argument, but have %d", ctype.String(), mname, mtype.NumIn())
		}
		// Receiver need be a struct pointer.
		structType := mtype.In(0)
		if structType.Kind() != reflect.Ptr || structType.Elem().Kind() != reflect.Struct {
			return nil, errors.Errorf("register request handler: %s.%s receiver need be a struct pointer: %s", ctype.String(), mname, structType)
		}
		// First arg need be exported or builtin, and need be a pointer.
		argType := mtype.In(1)
		if !goutil.IsExportedOrBuiltinType(argType) {
			return nil, errors.Errorf("register request handler: %s.%s args type not exported: %s", ctype.String(), mname, argType)
		}
		if argType.Kind() != reflect.Ptr {
			return nil, errors.Errorf("register request handler: %s.%s args type need be a pointer: %s", ctype.String(), mname, argType)
		}
		// Method needs two outs: reply error.
		if mtype.NumOut() != 2 {
			return nil, errors.Errorf("register request handler: %s.%s needs two out arguments, but have %d", ctype.String(), mname, mtype.NumOut())
		}
		// Reply type must be exported.
		replyType := mtype.Out(0)
		if !goutil.IsExportedOrBuiltinType(replyType) {
			return nil, errors.Errorf("register request handler: %s.%s first reply type not exported: %s", ctype.String(), mname, replyType)
		}
		// The return type of the method must be Error.
		if returnType := mtype.Out(1); returnType != typeOfError {
			return nil, errors.Errorf("register request handler: %s.%s second reply type %s not *Error", ctype.String(), mname, returnType)
		}
		handlers = append(handlers, &Handler{
			name:            goutil.CamelString(ctype.Name() + "/" + mname),
			originStruct:    ctype,
			method:          method,
			arg:             argType,
			reply:           replyType,
			pluginContainer: pluginContainer,
		})
	}
	return handlers, nil
}

// newPushRouter creates push packet router.
// Note: ctrlStruct needs to implement PushCtx interface.
func newPushRouter() *Router {
	return &Router{
		handlers:   make(map[string]*Handler),
		pathPrefix: "/",
		plugins:    make([]Plugin, 0),
		fn:         pushHandlersMaker,
		typ:        "push",
	}
}

// Note: ctrlStruct needs to implement PushCtx interface.
func pushHandlersMaker(pathPrefix string, ctrlStruct interface{}, pluginContainer PluginContainer) ([]*Handler, error) {
	var (
		ctype    = reflect.TypeOf(ctrlStruct)
		handlers = make([]*Handler, 0, 1)
	)
	if _, ok := ctrlStruct.(PushCtx); !ok {
		return nil, errors.Errorf("register push handler: the type is not implemented PushCtx interface: %s", ctype.String())
	}
	if pluginContainer == nil {
		pluginContainer = newPluginContainer()
	}
	for m := 0; m < ctype.NumMethod(); m++ {
		method := ctype.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs two ins: receiver, *args.
		if mtype.NumIn() != 2 {
			return nil, errors.Errorf("register push handler: %s.%s needs one in argument, but have %d", ctype.String(), mname, mtype.NumIn())
		}
		// Receiver need be a struct pointer.
		structType := mtype.In(0)
		if structType.Kind() != reflect.Ptr || structType.Elem().Kind() != reflect.Struct {
			return nil, errors.Errorf("register push handler: %s.%s receiver need be a struct pointer: %s", ctype.String(), mname, structType)
		}
		// First arg need be exported or builtin, and need be a pointer.
		argType := mtype.In(1)
		if !goutil.IsExportedOrBuiltinType(argType) {
			return nil, errors.Errorf("register push handler: %s.%s args type not exported: %s", ctype.String(), mname, argType)
		}
		if argType.Kind() != reflect.Ptr {
			return nil, errors.Errorf("register push handler: %s.%s args type need be a pointer: %s", ctype.String(), mname, argType)
		}
		// Method does not need out argument.
		if mtype.NumOut() != 0 {
			return nil, errors.Errorf("register push handler: %s.%s does not need out argument, but have %d", ctype.String(), mname, mtype.NumOut())
		}
		handlers = append(handlers, &Handler{
			name:            goutil.CamelString(ctype.Name() + "/" + mname),
			originStruct:    ctype,
			method:          method,
			arg:             argType,
			pluginContainer: pluginContainer,
		})
	}
	return handlers, nil
}
