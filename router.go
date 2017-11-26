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
	"path"
	"reflect"
	"strings"
	"sync"
	"unsafe"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/errors"
	"github.com/henrylee2cn/teleport/codec"
)

// Router the router of pull or push.
type (
	Router struct {
		handlers       map[string]*Handler
		unknownApiType **Handler
		// only for register router
		pathPrefix      string
		pluginContainer PluginContainer
		typ             string
		maker           HandlersMaker
	}
	// Handler pull or push handler type info
	Handler struct {
		name              string
		isUnknown         bool
		argElem           reflect.Type
		reply             reflect.Type // only for pull handler doc
		handleFunc        func(*readHandleCtx, reflect.Value)
		unknownHandleFunc func(*readHandleCtx)
		pluginContainer   PluginContainer
	}
	// HandlersMaker makes []*Handler
	HandlersMaker func(string, interface{}, PluginContainer) ([]*Handler, error)
)

// Group add handler group.
func (r *Router) Group(pathPrefix string, plugin ...Plugin) *Router {
	pluginContainer, err := r.pluginContainer.cloneAdd(plugin...)
	if err != nil {
		Fatalf("%v", err)
	}
	warnInvaildRouterHooks(plugin)
	return &Router{
		handlers:        r.handlers,
		unknownApiType:  r.unknownApiType,
		pathPrefix:      path.Join(r.pathPrefix, pathPrefix),
		pluginContainer: pluginContainer,
		maker:           r.maker,
	}
}

// Reg registers handler.
func (r *Router) Reg(ctrlStruct interface{}, plugin ...Plugin) {
	pluginContainer, err := r.pluginContainer.cloneAdd(plugin...)
	if err != nil {
		Fatalf("%v", err)
	}
	warnInvaildRouterHooks(plugin)
	handlers, err := r.maker(
		r.pathPrefix,
		ctrlStruct,
		pluginContainer,
	)
	if err != nil {
		Fatalf("%v", err)
	}
	for _, h := range handlers {
		if _, ok := r.handlers[h.name]; ok {
			Fatalf("There is a %s handler conflict: %s", r.typ, h.name)
		}
		pluginContainer.PostReg(h)
		r.handlers[h.name] = h
		Printf("register %s handler: %s", r.typ, h.name)
	}
}

// SetUnknown sets the default handler,
// which is called when no handler for pull or push is found.
func (r *Router) SetUnknown(unknownHandler interface{}, plugin ...Plugin) {
	pluginContainer, err := r.pluginContainer.cloneAdd(plugin...)
	if err != nil {
		Fatalf("%v", err)
	}
	warnInvaildRouterHooks(plugin)

	var h = &Handler{
		isUnknown:       true,
		argElem:         reflect.TypeOf([]byte{}),
		pluginContainer: pluginContainer,
	}

	switch r.typ {
	case "pull":
		h.name = "unknown_pull_handle"
		fn, ok := unknownHandler.(func(ctx UnknownPullCtx) (interface{}, *Rerror))
		if !ok {
			Fatalf("*Router.SetUnknown(): %s handler needs type:\n func(ctx UnknownPullCtx) (reply interface{}, rerr *Rerror)", h.name)
		}
		h.unknownHandleFunc = func(ctx *readHandleCtx) {
			body, rerr := fn(ctx)
			if rerr != nil {
				rerr.SetToMeta(ctx.output.Meta())
			} else if body != nil {
				ctx.output.SetBody(body)
				if ctx.output.BodyCodec() == codec.NilCodecId {
					ctx.output.SetBodyCodec(ctx.input.BodyCodec())
				}
			}
		}

	case "push":
		h.name = "unknown_push_handle"
		fn, ok := unknownHandler.(func(ctx UnknownPushCtx))
		if !ok {
			Fatalf("*Router.SetUnknown(): %s handler needs type:\n func(ctx UnknownPushCtx)", h.name)
		}
		h.unknownHandleFunc = func(ctx *readHandleCtx) {
			fn(ctx)
			if ctx.output.BodyCodec() == codec.NilCodecId {
				ctx.output.SetBodyCodec(ctx.input.BodyCodec())
			}
		}
	}

	if *r.unknownApiType == nil {
		Printf("set %s handler", h.name)
	} else {
		Warnf("covered %s handler", h.name)
	}
	r.unknownApiType = &h
}

func (r *Router) get(uriPath string) (*Handler, bool) {
	t, ok := r.handlers[uriPath]
	if ok {
		return t, true
	}
	if unknown := *r.unknownApiType; unknown != nil {
		return unknown, true
	}
	return nil, false
}

// newPullRouter creates pull packet router.
// Note: ctrlStruct needs to implement PullCtx interface.
func newPullRouter(pluginContainer PluginContainer) *Router {
	return &Router{
		handlers:        make(map[string]*Handler),
		unknownApiType:  new(*Handler),
		pathPrefix:      "/",
		pluginContainer: pluginContainer,
		maker:           pullHandlersMaker,
		typ:             "pull",
	}
}

// Note: ctrlStruct needs to implement PullCtx interface.
func pullHandlersMaker(pathPrefix string, ctrlStruct interface{}, pluginContainer PluginContainer) ([]*Handler, error) {
	var (
		ctype    = reflect.TypeOf(ctrlStruct)
		handlers = make([]*Handler, 0, 1)
	)

	if ctype.Kind() != reflect.Ptr {
		return nil, errors.Errorf("register pull handler: the type is not struct point: %s", ctype.String())
	}

	var ctypeElem = ctype.Elem()
	if ctypeElem.Kind() != reflect.Struct {
		return nil, errors.Errorf("register pull handler: the type is not struct point: %s", ctype.String())
	}

	if _, ok := ctrlStruct.(PullCtx); !ok {
		return nil, errors.Errorf("register pull handler: the type is not implemented PullCtx interface: %s", ctype.String())
	}

	iType, ok := ctypeElem.FieldByName("PullCtx")
	if !ok || !iType.Anonymous {
		return nil, errors.Errorf("register pull handler: the struct do not have anonymous field PullCtx: %s", ctype.String())
	}

	var pullCtxOffset = iType.Offset

	if pluginContainer == nil {
		pluginContainer = newPluginContainer()
	}

	type PullCtrlValue struct {
		ctrl   reflect.Value
		ctxPtr *PullCtx
	}
	var pool = &sync.Pool{
		New: func() interface{} {
			ctrl := reflect.New(ctypeElem)
			pullCtxPtr := ctrl.Pointer() + pullCtxOffset
			ctxPtr := (*PullCtx)(unsafe.Pointer(pullCtxPtr))
			return &PullCtrlValue{
				ctrl:   ctrl,
				ctxPtr: ctxPtr,
			}
		},
	}

	for m := 0; m < ctype.NumMethod(); m++ {
		method := ctype.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" || isPullCtxType(mname) {
			continue
		}
		// Method needs two ins: receiver, *args.
		if mtype.NumIn() != 2 {
			return nil, errors.Errorf("register pull handler: %s.%s needs one in argument, but have %d", ctype.String(), mname, mtype.NumIn())
		}
		// Receiver need be a struct pointer.
		structType := mtype.In(0)
		if structType.Kind() != reflect.Ptr || structType.Elem().Kind() != reflect.Struct {
			return nil, errors.Errorf("register pull handler: %s.%s receiver need be a struct pointer: %s", ctype.String(), mname, structType)
		}
		// First arg need be exported or builtin, and need be a pointer.
		argType := mtype.In(1)
		if !goutil.IsExportedOrBuiltinType(argType) {
			return nil, errors.Errorf("register pull handler: %s.%s args type not exported: %s", ctype.String(), mname, argType)
		}
		if argType.Kind() != reflect.Ptr {
			return nil, errors.Errorf("register pull handler: %s.%s args type need be a pointer: %s", ctype.String(), mname, argType)
		}
		// Method needs two outs: reply error.
		if mtype.NumOut() != 2 {
			return nil, errors.Errorf("register pull handler: %s.%s needs two out arguments, but have %d", ctype.String(), mname, mtype.NumOut())
		}
		// Reply type must be exported.
		replyType := mtype.Out(0)
		if !goutil.IsExportedOrBuiltinType(replyType) {
			return nil, errors.Errorf("register pull handler: %s.%s first reply type not exported: %s", ctype.String(), mname, replyType)
		}

		// The return type of the method must be Error.
		if returnType := mtype.Out(1); !isRerrorType(returnType.String()) {
			return nil, errors.Errorf("register pull handler: %s.%s second reply type %s not *tp.Rerror", ctype.String(), mname, returnType)
		}

		var methodFunc = method.Func
		var handleFunc = func(ctx *readHandleCtx, argValue reflect.Value) {
			obj := pool.Get().(*PullCtrlValue)
			*obj.ctxPtr = ctx
			rets := methodFunc.Call([]reflect.Value{obj.ctrl, argValue})
			ctx.output.SetBody(rets[0].Interface())
			rerr, _ := rets[1].Interface().(*Rerror)
			if rerr != nil {
				rerr.SetToMeta(ctx.output.Meta())

			} else if ctx.output.Body() != nil && ctx.output.BodyCodec() == codec.NilCodecId {
				ctx.output.SetBodyCodec(ctx.input.BodyCodec())
			}
			pool.Put(obj)
		}

		handlers = append(handlers, &Handler{
			name:            path.Join(pathPrefix, ctrlStructSnakeName(ctype), goutil.SnakeString(mname)),
			handleFunc:      handleFunc,
			argElem:         argType.Elem(),
			reply:           replyType,
			pluginContainer: pluginContainer,
		})
	}
	return handlers, nil
}

// newPushRouter creates push packet router.
// Note: ctrlStruct needs to implement PushCtx interface.
func newPushRouter(pluginContainer PluginContainer) *Router {
	return &Router{
		handlers:        make(map[string]*Handler),
		unknownApiType:  new(*Handler),
		pathPrefix:      "/",
		pluginContainer: pluginContainer,
		maker:           pushHandlersMaker,
		typ:             "push",
	}
}

// Note: ctrlStruct needs to implement PushCtx interface.
func pushHandlersMaker(pathPrefix string, ctrlStruct interface{}, pluginContainer PluginContainer) ([]*Handler, error) {
	var (
		ctype    = reflect.TypeOf(ctrlStruct)
		handlers = make([]*Handler, 0, 1)
	)

	if ctype.Kind() != reflect.Ptr {
		return nil, errors.Errorf("register push handler: the type is not struct point: %s", ctype.String())
	}

	var ctypeElem = ctype.Elem()
	if ctypeElem.Kind() != reflect.Struct {
		return nil, errors.Errorf("register push handler: the type is not struct point: %s", ctype.String())
	}

	if _, ok := ctrlStruct.(PushCtx); !ok {
		return nil, errors.Errorf("register push handler: the type is not implemented PushCtx interface: %s", ctype.String())
	}

	iType, ok := ctypeElem.FieldByName("PushCtx")
	if !ok || !iType.Anonymous {
		return nil, errors.Errorf("register push handler: the struct do not have anonymous field PushCtx: %s", ctype.String())
	}

	var pushCtxOffset = iType.Offset

	if pluginContainer == nil {
		pluginContainer = newPluginContainer()
	}
	type PushCtrlValue struct {
		ctrl   reflect.Value
		ctxPtr *PushCtx
	}
	var pool = &sync.Pool{
		New: func() interface{} {
			ctrl := reflect.New(ctypeElem)
			pushCtxPtr := ctrl.Pointer() + pushCtxOffset
			ctxPtr := (*PushCtx)(unsafe.Pointer(pushCtxPtr))
			return &PushCtrlValue{
				ctrl:   ctrl,
				ctxPtr: ctxPtr,
			}
		},
	}

	for m := 0; m < ctype.NumMethod(); m++ {
		method := ctype.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" || isPushCtxType(mname) {
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

		var methodFunc = method.Func
		var handleFunc = func(ctx *readHandleCtx, argValue reflect.Value) {
			obj := pool.Get().(*PushCtrlValue)
			*obj.ctxPtr = ctx
			methodFunc.Call([]reflect.Value{obj.ctrl, argValue})
			pool.Put(obj)
		}
		handlers = append(handlers, &Handler{
			name:            path.Join(pathPrefix, ctrlStructSnakeName(ctype), goutil.SnakeString(mname)),
			handleFunc:      handleFunc,
			argElem:         argType.Elem(),
			pluginContainer: pluginContainer,
		})
	}
	return handlers, nil
}

func isPullCtxType(name string) bool {
	ctype := reflect.TypeOf(PullCtx(new(readHandleCtx)))
	for m := 0; m < ctype.NumMethod(); m++ {
		if name == ctype.Method(m).Name {
			return true
		}
	}
	return false
}

func isPushCtxType(name string) bool {
	ctype := reflect.TypeOf(PushCtx(new(readHandleCtx)))
	for m := 0; m < ctype.NumMethod(); m++ {
		if name == ctype.Method(m).Name {
			return true
		}
	}
	return false
}

func isRerrorType(s string) bool {
	return strings.HasPrefix(s, "*") && strings.HasSuffix(s, ".Rerror")
}

func ctrlStructSnakeName(ctype reflect.Type) string {
	split := strings.Split(ctype.String(), ".")
	tName := split[len(split)-1]
	return goutil.SnakeString(tName)
}

// Name returns the handler name.
func (h *Handler) Name() string {
	return h.name
}

// ArgElemType returns the handler arg elem type.
func (h *Handler) ArgElemType() reflect.Type {
	return h.argElem
}

// ReplyType returns the handler reply type
func (h *Handler) ReplyType() reflect.Type {
	return h.reply
}

// IsPush checks if it is push handler or not.
func (h *Handler) IsPush() bool {
	return h.reply == nil
}

// IsPull checks if it is pull handler or not.
func (h *Handler) IsPull() bool {
	return !h.IsPush()
}
