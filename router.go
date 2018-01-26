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

// Router the router of pull or push handlers.
//
// PullController Model Demo:
//  type XxxPullController struct {
//      tp.PullCtx
//  }
//  // XxZz register the route: /aaa/xx_zz
//  func (x *XxxPullController) XxZz(args *<T>) (<T>, *tp.Rerror) {
//      ...
//      return r, nil
//  }
//  // YyZz register the route: /aaa/yy_zz
//  func (x *XxxPullController) YyZz(args *<T>) (<T>, *tp.Rerror) {
//      ...
//      return r, nil
//  }
//
// PushController Model Demo:
//  type XxxPushController struct {
//      tp.PushCtx
//  }
//  // XxZz register the route: /bbb/yy_zz
//  func (b *XxxPushController) XxZz(args *<T>) *tp.Rerror {
//      ...
//      return r, nil
//  }
//  // YyZz register the route: /bbb/yy_zz
//  func (b *XxxPushController) YyZz(args *<T>) *tp.Rerror {
//      ...
//      return r, nil
//  }
//
// UnknownPullHandler Type Demo:
//  func XxxUnknownPullHandler (ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
//      ...
//      return r, nil
//  }
//
// UnknownPushHandler Type Demo:
//  func XxxUnknownPushHandler(ctx tp.UnknownPushCtx) *tp.Rerror {
//      ...
//      return nil
//  }
type (
	// Router the router of pull or push handlers.
	Router struct {
		*SubRouter
	}
	// SubRouter without the SetUnknownPull and SetUnknownPush methods
	SubRouter struct {
		root        *Router
		handlers    map[string]*Handler
		unknownPull **Handler
		unknownPush **Handler
		// only for register router
		pathPrefix      string
		pluginContainer *PluginContainer
	}
	// Handler pull or push handler type info
	Handler struct {
		name              string
		isUnknown         bool
		argElem           reflect.Type
		reply             reflect.Type // only for pull handler doc
		handleFunc        func(*readHandleCtx, reflect.Value)
		unknownHandleFunc func(*readHandleCtx)
		pluginContainer   *PluginContainer
		routerTypeName    string
	}
	// HandlersMaker makes []*Handler
	HandlersMaker func(string, interface{}, *PluginContainer) ([]*Handler, error)
)

const (
	pnPush        = "push"
	pnPull        = "pull"
	pnUnknownPush = "unknown_push"
	pnUnknownPull = "unknown_pull"
)

// newRouter creates root router.
func newRouter(rootGroup string, pluginContainer *PluginContainer) *Router {
	rootGroup = path.Join("/", rootGroup)
	root := &Router{
		SubRouter: &SubRouter{
			handlers:        make(map[string]*Handler),
			unknownPull:     new(*Handler),
			unknownPush:     new(*Handler),
			pathPrefix:      rootGroup,
			pluginContainer: pluginContainer,
		},
	}
	root.SubRouter.root = root
	return root
}

// Root returns the root router which is added the SetUnknownPull and SetUnknownPush methods.
func (r *SubRouter) Root() *Router {
	return r.root
}

// WithUnknown converts to the router which is added the SetUnknownPull and SetUnknownPush methods.
func (r *SubRouter) WithUnknown() *Router {
	return &Router{r}
}

// SubRoute adds handler group.
func (r *SubRouter) SubRoute(pathPrefix string, plugin ...Plugin) *SubRouter {
	pluginContainer := r.pluginContainer.cloneAppendRight(plugin...)
	warnInvaildHandlerHooks(plugin)
	return &SubRouter{
		root:            r.root,
		handlers:        r.handlers,
		unknownPull:     r.unknownPull,
		unknownPush:     r.unknownPush,
		pathPrefix:      path.Join(r.pathPrefix, pathPrefix),
		pluginContainer: pluginContainer,
	}
}

// RoutePull registers PULL handler.
func (r *SubRouter) RoutePull(ctrlStruct interface{}, plugin ...Plugin) {
	r.reg(pnPull, pullHandlersMaker, ctrlStruct, plugin)
}

// RoutePush registers PUSH handler.
func (r *SubRouter) RoutePush(ctrlStruct interface{}, plugin ...Plugin) {
	r.reg(pnPush, pushHandlersMaker, ctrlStruct, plugin)
}

func (r *SubRouter) reg(
	routerTypeName string,
	handlerMaker func(string, interface{}, *PluginContainer) ([]*Handler, error),
	ctrlStruct interface{},
	plugins []Plugin,
) {
	pluginContainer := r.pluginContainer.cloneAppendRight(plugins...)
	warnInvaildHandlerHooks(plugins)
	handlers, err := handlerMaker(
		r.pathPrefix,
		ctrlStruct,
		pluginContainer,
	)
	if err != nil {
		Fatalf("%v", err)
	}
	for _, h := range handlers {
		if _, ok := r.handlers[h.name]; ok {
			Fatalf("there is a handler conflict: %s", h.name)
		}
		h.routerTypeName = routerTypeName
		r.handlers[h.name] = h
		pluginContainer.PostReg(h)
		Printf("register %s handler: %s", routerTypeName, h.name)
	}
}

// SetUnknownPull sets the default handler,
// which is called when no handler for PULL is found.
func (r *Router) SetUnknownPull(fn func(UnknownPullCtx) (interface{}, *Rerror), plugin ...Plugin) {
	pluginContainer := r.pluginContainer.cloneAppendRight(plugin...)
	warnInvaildHandlerHooks(plugin)

	var h = &Handler{
		name:            pnUnknownPush,
		isUnknown:       true,
		argElem:         reflect.TypeOf([]byte{}),
		pluginContainer: pluginContainer,
		unknownHandleFunc: func(ctx *readHandleCtx) {
			body, rerr := fn(ctx)
			if rerr != nil {
				ctx.handleErr = rerr
				rerr.SetToMeta(ctx.output.Meta())
			} else if body != nil {
				ctx.output.SetBody(body)
				if ctx.output.BodyCodec() == codec.NilCodecId {
					ctx.output.SetBodyCodec(ctx.input.BodyCodec())
				}
			}
		},
	}

	if *r.unknownPull == nil {
		Printf("set %s handler", h.name)
	} else {
		Warnf("covered %s handler", h.name)
	}
	r.unknownPull = &h
}

// SetUnknownPush sets the default handler,
// which is called when no handler for PUSH is found.
func (r *Router) SetUnknownPush(fn func(UnknownPushCtx) *Rerror, plugin ...Plugin) {
	pluginContainer := r.pluginContainer.cloneAppendRight(plugin...)
	warnInvaildHandlerHooks(plugin)

	var h = &Handler{
		name:            pnUnknownPush,
		isUnknown:       true,
		argElem:         reflect.TypeOf([]byte{}),
		pluginContainer: pluginContainer,
		unknownHandleFunc: func(ctx *readHandleCtx) {
			ctx.handleErr = fn(ctx)
		},
	}

	if *r.unknownPush == nil {
		Printf("set %s handler", h.name)
	} else {
		Warnf("covered %s handler", h.name)
	}
	r.unknownPush = &h
}

func (r *SubRouter) getPull(uriPath string) (*Handler, bool) {
	t, ok := r.handlers[uriPath]
	if ok {
		return t, true
	}
	if unknown := *r.unknownPull; unknown != nil {
		return unknown, true
	}
	return nil, false
}

func (r *SubRouter) getPush(uriPath string) (*Handler, bool) {
	t, ok := r.handlers[uriPath]
	if ok {
		return t, true
	}
	if unknown := *r.unknownPush; unknown != nil {
		return unknown, true
	}
	return nil, false
}

// Note: ctrlStruct needs to implement PullCtx interface.
func pullHandlersMaker(pathPrefix string, ctrlStruct interface{}, pluginContainer *PluginContainer) ([]*Handler, error) {
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
				ctx.handleErr = rerr
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

// Note: ctrlStruct needs to implement PushCtx interface.
func pushHandlersMaker(pathPrefix string, ctrlStruct interface{}, pluginContainer *PluginContainer) ([]*Handler, error) {
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
		if mtype.NumOut() != 1 {
			return nil, errors.Errorf("register push handler: %s.%s needs one out arguments, but have %d", ctype.String(), mname, mtype.NumOut())
		}

		// The return type of the method must be Error.
		if returnType := mtype.Out(0); !isRerrorType(returnType.String()) {
			return nil, errors.Errorf("register push handler: %s.%s reply type %s not *tp.Rerror", ctype.String(), mname, returnType)
		}

		var methodFunc = method.Func
		var handleFunc = func(ctx *readHandleCtx, argValue reflect.Value) {
			obj := pool.Get().(*PushCtrlValue)
			*obj.ctxPtr = ctx
			rets := methodFunc.Call([]reflect.Value{obj.ctrl, argValue})
			ctx.handleErr, _ = rets[0].Interface().(*Rerror)
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

// NewArgValue creates a new arg elem value.
func (h *Handler) NewArgValue() reflect.Value {
	return reflect.New(h.argElem)
}

// ReplyType returns the handler reply type
func (h *Handler) ReplyType() reflect.Type {
	return h.reply
}

// IsPull checks if it is pull handler or not.
func (h *Handler) IsPull() bool {
	return h.routerTypeName == pnPull || h.routerTypeName == pnUnknownPull
}

// IsPush checks if it is push handler or not.
func (h *Handler) IsPush() bool {
	return h.routerTypeName == pnPush || h.routerTypeName == pnUnknownPush
}

// IsUnknown checks if it is unknown handler(pull/push) or not.
func (h *Handler) IsUnknown() bool {
	return h.isUnknown
}

// RouterTypeName returns the router type name.
func (h *Handler) RouterTypeName() string {
	return h.routerTypeName
}
