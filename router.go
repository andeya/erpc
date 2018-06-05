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

package tp

import (
	"path"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/errors"
)

/**
 * Router the router of pull or push handlers.
 *
 * 1. Pull-Controller-Struct API template
 *
 *  type Aaa struct {
 *      tp.PullCtx
 *  }
 *  func (x *Aaa) XxZz(args *<T>) (<T>, *tp.Rerror) {
 *      ...
 *      return r, nil
 *  }
 *
 * - register it to root router:
 *
 *  // register the pull route: /aaa/xx_zz
 *  peer.RoutePull(new(Aaa))
 *
 *  // or register the pull route: /xx_zz
 *  peer.RoutePullFunc((*Aaa).XxZz)
 *
 * 2. Pull-Handler-Function API template
 *
 *  func XxZz(ctx tp.PullCtx, args *<T>) (<T>, *tp.Rerror) {
 *      ...
 *      return r, nil
 *  }
 *
 * - register it to root router:
 *
 *  // register the pull route: /xx_zz
 *  peer.RoutePullFunc(XxZz)
 *
 * 3. Push-Controller-Struct API template
 *
 *  type Bbb struct {
 *      tp.PushCtx
 *  }
 *  func (b *Bbb) YyZz(args *<T>) *tp.Rerror {
 *      ...
 *      return nil
 *  }
 *
 * - register it to root router:
 *
 *  // register the push route: /bbb/yy_zz
 *  peer.RoutePush(new(Bbb))
 *
 *  // or register the push route: /yy_zz
 *  peer.RoutePushFunc((*Bbb).YyZz)
 *
 * 4. Push-Handler-Function API template
 *
 *  // YyZz register the route: /yy_zz
 *  func YyZz(ctx tp.PushCtx, args *<T>) *tp.Rerror {
 *      ...
 *      return nil
 *  }
 *
 * - register it to root router:
 *
 *  // register the push route: /yy_zz
 *  peer.RoutePushFunc(YyZz)
 *
 * 5. Unknown-Pull-Handler-Function API template
 *
 *  func XxxUnknownPull (ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
 *      ...
 *      return r, nil
 *  }
 *
 * - register it to root router:
 *
 *  // register the unknown pull route: /*
 *  peer.SetUnknownPull(XxxUnknownPull)
 *
 * 6. Unknown-Push-Handler-Function API template
 *
 *  func XxxUnknownPush(ctx tp.UnknownPushCtx) *tp.Rerror {
 *      ...
 *      return nil
 *  }
 *
 * - register it to root router:
 *
 *  // register the unknown push route: /*
 *  peer.SetUnknownPush(XxxUnknownPush)
 *
 * 7. The mapping rule of struct(func) name to URI path:
 *
 * - `AaBb` -> `/aa_bb`
 * - `Aa_Bb` -> `/aa/bb`
 * - `aa_bb` -> `/aa/bb`
 * - `Aa__Bb` -> `/aa_bb`
 * - `aa__bb` -> `/aa_bb`
 * - `ABC_XYZ` -> `/abc/xyz`
 * - `ABcXYz` -> `/abc_xyz`
 * - `ABC__XYZ` -> `/abc_xyz`
 **/

type (
	// Router the router of pull or push handlers.
	Router struct {
		subRouter *SubRouter
	}
	// SubRouter without the SetUnknownPull and SetUnknownPush methods
	SubRouter struct {
		root         *Router
		pullHandlers map[string]*Handler
		pushHandlers map[string]*Handler
		unknownPull  **Handler
		unknownPush  **Handler
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
		handleFunc        func(*handlerCtx, reflect.Value)
		unknownHandleFunc func(*handlerCtx)
		pluginContainer   *PluginContainer
		routerTypeName    string
	}
	// HandlersMaker makes []*Handler
	HandlersMaker func(string, interface{}, *PluginContainer) ([]*Handler, error)
)

const (
	pnPush        = "PUSH"
	pnPull        = "PULL"
	pnUnknownPush = "UNKNOWN_PUSH"
	pnUnknownPull = "UNKNOWN_PULL"
)

// newRouter creates root router.
func newRouter(rootGroup string, pluginContainer *PluginContainer) *Router {
	rootGroup = path.Join("/", rootGroup)
	root := &Router{
		subRouter: &SubRouter{
			pullHandlers:    make(map[string]*Handler),
			pushHandlers:    make(map[string]*Handler),
			unknownPull:     new(*Handler),
			unknownPush:     new(*Handler),
			pathPrefix:      rootGroup,
			pluginContainer: pluginContainer,
		},
	}
	root.subRouter.root = root
	return root
}

// Root returns the root router.
func (r *SubRouter) Root() *Router {
	return r.root
}

// ToRouter converts to the router which is added the SetUnknownPull and SetUnknownPush methods.
func (r *SubRouter) ToRouter() *Router {
	return &Router{subRouter: r}
}

// SubRoute adds handler group.
func (r *Router) SubRoute(pathPrefix string, plugin ...Plugin) *SubRouter {
	return r.subRouter.SubRoute(pathPrefix, plugin...)
}

// SubRoute adds handler group.
func (r *SubRouter) SubRoute(pathPrefix string, plugin ...Plugin) *SubRouter {
	pluginContainer := r.pluginContainer.cloneAndAppendMiddle(plugin...)
	warnInvaildHandlerHooks(plugin)
	return &SubRouter{
		root:            r.root,
		pullHandlers:    r.pullHandlers,
		pushHandlers:    r.pushHandlers,
		unknownPull:     r.unknownPull,
		unknownPush:     r.unknownPush,
		pathPrefix:      path.Join(r.pathPrefix, pathPrefix),
		pluginContainer: pluginContainer,
	}
}

// RoutePull registers PULL handlers, and returns the paths.
func (r *Router) RoutePull(pullCtrlStruct interface{}, plugin ...Plugin) []string {
	return r.subRouter.RoutePull(pullCtrlStruct, plugin...)
}

// RoutePull registers PULL handlers, and returns the paths.
func (r *SubRouter) RoutePull(pullCtrlStruct interface{}, plugin ...Plugin) []string {
	return r.reg(pnPull, makePullHandlersFromStruct, pullCtrlStruct, plugin)
}

// RoutePullFunc registers PULL handler, and returns the path.
func (r *Router) RoutePullFunc(pullHandleFunc interface{}, plugin ...Plugin) string {
	return r.subRouter.RoutePullFunc(pullHandleFunc, plugin...)
}

// RoutePullFunc registers PULL handler, and returns the path.
func (r *SubRouter) RoutePullFunc(pullHandleFunc interface{}, plugin ...Plugin) string {
	return r.reg(pnPull, makePullHandlersFromFunc, pullHandleFunc, plugin)[0]
}

// RoutePush registers PUSH handlers, and returns the paths.
func (r *Router) RoutePush(pushCtrlStruct interface{}, plugin ...Plugin) []string {
	return r.subRouter.RoutePush(pushCtrlStruct, plugin...)
}

// RoutePush registers PUSH handlers, and returns the paths.
func (r *SubRouter) RoutePush(pushCtrlStruct interface{}, plugin ...Plugin) []string {
	return r.reg(pnPush, makePushHandlersFromStruct, pushCtrlStruct, plugin)
}

// RoutePushFunc registers PUSH handler, and returns the path.
func (r *Router) RoutePushFunc(pushHandleFunc interface{}, plugin ...Plugin) string {
	return r.subRouter.RoutePushFunc(pushHandleFunc, plugin...)
}

// RoutePushFunc registers PUSH handler, and returns the path.
func (r *SubRouter) RoutePushFunc(pushHandleFunc interface{}, plugin ...Plugin) string {
	return r.reg(pnPush, makePushHandlersFromFunc, pushHandleFunc, plugin)[0]
}

func (r *SubRouter) reg(
	routerTypeName string,
	handlerMaker func(string, interface{}, *PluginContainer) ([]*Handler, error),
	ctrlStruct interface{},
	plugins []Plugin,
) []string {
	pluginContainer := r.pluginContainer.cloneAndAppendMiddle(plugins...)
	warnInvaildHandlerHooks(plugins)
	handlers, err := handlerMaker(
		r.pathPrefix,
		ctrlStruct,
		pluginContainer,
	)
	if err != nil {
		Fatalf("%v", err)
	}
	var names []string
	var hadHandlers map[string]*Handler
	if routerTypeName == pnPull {
		hadHandlers = r.pullHandlers
	} else {
		hadHandlers = r.pushHandlers
	}
	for _, h := range handlers {
		if _, ok := hadHandlers[h.name]; ok {
			Fatalf("there is a handler conflict: %s", h.name)
		}
		h.routerTypeName = routerTypeName
		hadHandlers[h.name] = h
		pluginContainer.postReg(h)
		Printf("register %s handler: %s", routerTypeName, h.name)
		names = append(names, h.name)
	}
	return names
}

// SetUnknownPull sets the default handler,
// which is called when no handler for PULL is found.
func (r *Router) SetUnknownPull(fn func(UnknownPullCtx) (interface{}, *Rerror), plugin ...Plugin) {
	pluginContainer := r.subRouter.pluginContainer.cloneAndAppendMiddle(plugin...)
	warnInvaildHandlerHooks(plugin)

	var h = &Handler{
		name:            pnUnknownPull,
		isUnknown:       true,
		argElem:         reflect.TypeOf([]byte{}),
		pluginContainer: pluginContainer,
		unknownHandleFunc: func(ctx *handlerCtx) {
			body, rerr := fn(ctx)
			if rerr != nil {
				ctx.handleErr = rerr
				rerr.SetToMeta(ctx.output.Meta())
			} else {
				ctx.output.SetBody(body)
			}
		},
	}

	if *r.subRouter.unknownPull == nil {
		Printf("set %s handler", h.name)
	} else {
		Warnf("covered %s handler", h.name)
	}
	r.subRouter.unknownPull = &h
}

// SetUnknownPush sets the default handler,
// which is called when no handler for PUSH is found.
func (r *Router) SetUnknownPush(fn func(UnknownPushCtx) *Rerror, plugin ...Plugin) {
	pluginContainer := r.subRouter.pluginContainer.cloneAndAppendMiddle(plugin...)
	warnInvaildHandlerHooks(plugin)

	var h = &Handler{
		name:            pnUnknownPush,
		isUnknown:       true,
		argElem:         reflect.TypeOf([]byte{}),
		pluginContainer: pluginContainer,
		unknownHandleFunc: func(ctx *handlerCtx) {
			ctx.handleErr = fn(ctx)
		},
	}

	if *r.subRouter.unknownPush == nil {
		Printf("set %s handler", h.name)
	} else {
		Warnf("covered %s handler", h.name)
	}
	r.subRouter.unknownPush = &h
}

func (r *SubRouter) getPull(uriPath string) (*Handler, bool) {
	t, ok := r.pullHandlers[uriPath]
	if ok {
		return t, true
	}
	if unknown := *r.unknownPull; unknown != nil {
		return unknown, true
	}
	return nil, false
}

func (r *SubRouter) getPush(uriPath string) (*Handler, bool) {
	t, ok := r.pushHandlers[uriPath]
	if ok {
		return t, true
	}
	if unknown := *r.unknownPush; unknown != nil {
		return unknown, true
	}
	return nil, false
}

func getStructNameFromStruct(ctrlStruct interface{})string{
	var structName string
	ctype := reflect.TypeOf(ctrlStruct)
	if method, ok := ctype.MethodByName("Name"); ok {
		result := method.Func.Call([]reflect.Value{reflect.ValueOf(ctrlStruct)})
		structName = result[0].Interface().(string)
	} else {
		structName = ToUriPath(ctrlStructName(ctype))
	}

	return structName
}

// Note: pullCtrlStruct needs to implement PullCtx interface.
func makePullHandlersFromStruct(pathPrefix string, pullCtrlStruct interface{}, pluginContainer *PluginContainer) ([]*Handler, error) {
	var (
		ctype    = reflect.TypeOf(pullCtrlStruct)
		handlers = make([]*Handler, 0, 1)
	)

	if ctype.Kind() != reflect.Ptr {
		return nil, errors.Errorf("pull-handler: the type is not struct point: %s", ctype.String())
	}

	var ctypeElem = ctype.Elem()
	if ctypeElem.Kind() != reflect.Struct {
		return nil, errors.Errorf("pull-handler: the type is not struct point: %s", ctype.String())
	}

	if _, ok := pullCtrlStruct.(PullCtx); !ok {
		return nil, errors.Errorf("pull-handler: the type is not implemented tp.PullCtx interface: %s", ctype.String())
	}

	iType, ok := ctypeElem.FieldByName("PullCtx")
	if !ok || !iType.Anonymous {
		return nil, errors.Errorf("pull-handler: the struct do not have anonymous field tp.PullCtx: %s", ctype.String())
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
		if method.PkgPath != "" || mname == "Name" || isBelongToPullCtx(mname) {
			continue
		}
		// Method needs two ins: receiver, *args.
		if mtype.NumIn() != 2 {
			return nil, errors.Errorf("pull-handler: %s.%s needs one in argument, but have %d", ctype.String(), mname, mtype.NumIn())
		}
		// Receiver need be a struct pointer.
		structType := mtype.In(0)
		if structType.Kind() != reflect.Ptr || structType.Elem().Kind() != reflect.Struct {
			return nil, errors.Errorf("pull-handler: %s.%s receiver need be a struct pointer: %s", ctype.String(), mname, structType)
		}
		// First arg need be exported or builtin, and need be a pointer.
		argType := mtype.In(1)
		if !goutil.IsExportedOrBuiltinType(argType) {
			return nil, errors.Errorf("pull-handler: %s.%s args type not exported: %s", ctype.String(), mname, argType)
		}
		if argType.Kind() != reflect.Ptr {
			return nil, errors.Errorf("pull-handler: %s.%s args type need be a pointer: %s", ctype.String(), mname, argType)
		}
		// Method needs two outs: reply, *Rerror.
		if mtype.NumOut() != 2 {
			return nil, errors.Errorf("pull-handler: %s.%s needs two out arguments, but have %d", ctype.String(), mname, mtype.NumOut())
		}
		// Reply type must be exported.
		replyType := mtype.Out(0)
		if !goutil.IsExportedOrBuiltinType(replyType) {
			return nil, errors.Errorf("pull-handler: %s.%s first reply type not exported: %s", ctype.String(), mname, replyType)
		}

		// The return type of the method must be *Rerror.
		if returnType := mtype.Out(1); !isRerrorType(returnType.String()) {
			return nil, errors.Errorf("pull-handler: %s.%s second out argument %s is not *tp.Rerror", ctype.String(), mname, returnType)
		}

		var methodFunc = method.Func
		var handleFunc = func(ctx *handlerCtx, argValue reflect.Value) {
			obj := pool.Get().(*PullCtrlValue)
			*obj.ctxPtr = ctx
			rets := methodFunc.Call([]reflect.Value{obj.ctrl, argValue})
			rerr, _ := rets[1].Interface().(*Rerror)
			if rerr != nil {
				ctx.handleErr = rerr
				rerr.SetToMeta(ctx.output.Meta())
			} else {
				ctx.output.SetBody(rets[0].Interface())
			}
			pool.Put(obj)
		}

		handlers = append(handlers, &Handler{
			name:            path.Join(pathPrefix, getStructNameFromStruct(pullCtrlStruct), ToUriPath(mname)),
			handleFunc:      handleFunc,
			argElem:         argType.Elem(),
			reply:           replyType,
			pluginContainer: pluginContainer,
		})
	}
	return handlers, nil
}

func makePullHandlersFromFunc(pathPrefix string, pullHandleFunc interface{}, pluginContainer *PluginContainer) ([]*Handler, error) {
	var (
		ctype      = reflect.TypeOf(pullHandleFunc)
		cValue     = reflect.ValueOf(pullHandleFunc)
		typeString = objectName(cValue)
	)

	if ctype.Kind() != reflect.Func {
		return nil, errors.Errorf("pull-handler: the type is not function: %s", typeString)
	}

	// needs two outs: reply, *Rerror.
	if ctype.NumOut() != 2 {
		return nil, errors.Errorf("pull-handler: %s needs two out arguments, but have %d", typeString, ctype.NumOut())
	}

	// Reply type must be exported.
	replyType := ctype.Out(0)
	if !goutil.IsExportedOrBuiltinType(replyType) {
		return nil, errors.Errorf("pull-handler: %s first reply type not exported: %s", typeString, replyType)
	}

	// The return type of the method must be *Rerror.
	if returnType := ctype.Out(1); !isRerrorType(returnType.String()) {
		return nil, errors.Errorf("pull-handler: %s second out argument %s is not *tp.Rerror", typeString, returnType)
	}

	// needs two ins: PullCtx, *args.
	if ctype.NumIn() != 2 {
		return nil, errors.Errorf("pull-handler: %s needs two in argument, but have %d", typeString, ctype.NumIn())
	}

	// First arg need be exported or builtin, and need be a pointer.
	argType := ctype.In(1)
	if !goutil.IsExportedOrBuiltinType(argType) {
		return nil, errors.Errorf("pull-handler: %s args type not exported: %s", typeString, argType)
	}
	if argType.Kind() != reflect.Ptr {
		return nil, errors.Errorf("pull-handler: %s args type need be a pointer: %s", typeString, argType)
	}

	// first agr need be a PullCtx (struct pointer or PullCtx).
	ctxType := ctype.In(0)
	if !ctxType.Implements(reflect.TypeOf((*PullCtx)(nil)).Elem()) {

		return nil, errors.Errorf("pull-handler: %s's first arg need implement tp.PullCtx: %s", typeString, ctxType)
	}

	var handleFunc func(*handlerCtx, reflect.Value)

	switch ctxType.Kind() {
	default:
		return nil, errors.Errorf("pull-handler: %s's first arg must be tp.PullCtx type or struct pointer: %s", typeString, ctxType)

	case reflect.Interface:
		if !reflect.TypeOf((*PullCtx)(nil)).Elem().Implements(reflect.New(ctxType).Type().Elem()) {
			return nil, errors.Errorf("pull-handler: %s's first arg must be tp.PullCtx type or struct pointer: %s", typeString, ctxType)
		}

		handleFunc = func(ctx *handlerCtx, argValue reflect.Value) {
			rets := cValue.Call([]reflect.Value{reflect.ValueOf(ctx), argValue})
			rerr, _ := rets[1].Interface().(*Rerror)
			if rerr != nil {
				ctx.handleErr = rerr
				rerr.SetToMeta(ctx.output.Meta())
			} else {
				ctx.output.SetBody(rets[0].Interface())
			}
		}

	case reflect.Ptr:
		var ctxTypeElem = ctxType.Elem()
		if ctxTypeElem.Kind() != reflect.Struct {
			return nil, errors.Errorf("pull-handler: %s's first arg must be tp.PullCtx type or struct pointer: %s", typeString, ctxType)
		}

		iType, ok := ctxTypeElem.FieldByName("PullCtx")
		if !ok || !iType.Anonymous {
			return nil, errors.Errorf("pull-handler: %s's first arg do not have anonymous field tp.PullCtx: %s", typeString, ctxType)
		}

		type PullCtrlValue struct {
			ctrl   reflect.Value
			ctxPtr *PullCtx
		}
		var pullCtxOffset = iType.Offset
		var pool = &sync.Pool{
			New: func() interface{} {
				ctrl := reflect.New(ctxTypeElem)
				pullCtxPtr := ctrl.Pointer() + pullCtxOffset
				ctxPtr := (*PullCtx)(unsafe.Pointer(pullCtxPtr))
				return &PullCtrlValue{
					ctrl:   ctrl,
					ctxPtr: ctxPtr,
				}
			},
		}

		handleFunc = func(ctx *handlerCtx, argValue reflect.Value) {
			obj := pool.Get().(*PullCtrlValue)
			*obj.ctxPtr = ctx
			rets := cValue.Call([]reflect.Value{obj.ctrl, argValue})
			rerr, _ := rets[1].Interface().(*Rerror)
			if rerr != nil {
				ctx.handleErr = rerr
				rerr.SetToMeta(ctx.output.Meta())
			} else {
				ctx.output.SetBody(rets[0].Interface())
			}
			pool.Put(obj)
		}
	}

	if pluginContainer == nil {
		pluginContainer = newPluginContainer()
	}
	return []*Handler{&Handler{
		name:            path.Join(pathPrefix, ToUriPath(handlerFuncName(cValue))),
		handleFunc:      handleFunc,
		argElem:         argType.Elem(),
		reply:           replyType,
		pluginContainer: pluginContainer,
	}}, nil
}

// Note: pushCtrlStruct needs to implement PushCtx interface.
func makePushHandlersFromStruct(pathPrefix string, pushCtrlStruct interface{}, pluginContainer *PluginContainer) ([]*Handler, error) {
	var (
		ctype    = reflect.TypeOf(pushCtrlStruct)
		handlers = make([]*Handler, 0, 1)
	)

	if ctype.Kind() != reflect.Ptr {
		return nil, errors.Errorf("push-handler: the type is not struct point: %s", ctype.String())
	}

	var ctypeElem = ctype.Elem()
	if ctypeElem.Kind() != reflect.Struct {
		return nil, errors.Errorf("push-handler: the type is not struct point: %s", ctype.String())
	}

	if _, ok := pushCtrlStruct.(PushCtx); !ok {
		return nil, errors.Errorf("push-handler: the type is not implemented tp.PushCtx interface: %s", ctype.String())
	}

	iType, ok := ctypeElem.FieldByName("PushCtx")
	if !ok || !iType.Anonymous {
		return nil, errors.Errorf("push-handler: the struct do not have anonymous field tp.PushCtx: %s", ctype.String())
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
		if method.PkgPath != "" || mname == "Name" || isBelongToPushCtx(mname) {
			continue
		}
		// Method needs two ins: receiver, *args.
		if mtype.NumIn() != 2 {
			return nil, errors.Errorf("push-handler: %s.%s needs one in argument, but have %d", ctype.String(), mname, mtype.NumIn())
		}
		// Receiver need be a struct pointer.
		structType := mtype.In(0)
		if structType.Kind() != reflect.Ptr || structType.Elem().Kind() != reflect.Struct {
			return nil, errors.Errorf("push-handler: %s.%s receiver need be a struct pointer: %s", ctype.String(), mname, structType)
		}
		// First arg need be exported or builtin, and need be a pointer.
		argType := mtype.In(1)
		if !goutil.IsExportedOrBuiltinType(argType) {
			return nil, errors.Errorf("push-handler: %s.%s args type not exported: %s", ctype.String(), mname, argType)
		}
		if argType.Kind() != reflect.Ptr {
			return nil, errors.Errorf("push-handler: %s.%s args type need be a pointer: %s", ctype.String(), mname, argType)
		}

		// Method needs one out: *Rerror.
		if mtype.NumOut() != 1 {
			return nil, errors.Errorf("push-handler: %s.%s needs one out arguments, but have %d", ctype.String(), mname, mtype.NumOut())
		}

		// The return type of the method must be *Rerror.
		if returnType := mtype.Out(0); !isRerrorType(returnType.String()) {
			return nil, errors.Errorf("push-handler: %s.%s out argument %s is not *tp.Rerror", ctype.String(), mname, returnType)
		}

		var methodFunc = method.Func
		var handleFunc = func(ctx *handlerCtx, argValue reflect.Value) {
			obj := pool.Get().(*PushCtrlValue)
			*obj.ctxPtr = ctx
			rets := methodFunc.Call([]reflect.Value{obj.ctrl, argValue})
			ctx.handleErr, _ = rets[0].Interface().(*Rerror)
			pool.Put(obj)
		}

		handlers = append(handlers, &Handler{
			name:            path.Join(pathPrefix, getStructNameFromStruct(pushCtrlStruct), ToUriPath(mname)),
			handleFunc:      handleFunc,
			argElem:         argType.Elem(),
			pluginContainer: pluginContainer,
		})
	}
	return handlers, nil
}

func makePushHandlersFromFunc(pathPrefix string, pushHandleFunc interface{}, pluginContainer *PluginContainer) ([]*Handler, error) {
	var (
		ctype      = reflect.TypeOf(pushHandleFunc)
		cValue     = reflect.ValueOf(pushHandleFunc)
		typeString = objectName(cValue)
	)

	if ctype.Kind() != reflect.Func {
		return nil, errors.Errorf("push-handler: the type is not function: %s", typeString)
	}

	// needs one out: *Rerror.
	if ctype.NumOut() != 1 {
		return nil, errors.Errorf("push-handler: %s needs one out arguments, but have %d", typeString, ctype.NumOut())
	}

	// The return type of the method must be *Rerror.
	if returnType := ctype.Out(0); !isRerrorType(returnType.String()) {
		return nil, errors.Errorf("push-handler: %s out argument %s is not *tp.Rerror", typeString, returnType)
	}

	// needs two ins: PushCtx, *args.
	if ctype.NumIn() != 2 {
		return nil, errors.Errorf("push-handler: %s needs two in argument, but have %d", typeString, ctype.NumIn())
	}

	// First arg need be exported or builtin, and need be a pointer.
	argType := ctype.In(1)
	if !goutil.IsExportedOrBuiltinType(argType) {
		return nil, errors.Errorf("push-handler: %s args type not exported: %s", typeString, argType)
	}
	if argType.Kind() != reflect.Ptr {
		return nil, errors.Errorf("push-handler: %s args type need be a pointer: %s", typeString, argType)
	}

	// first agr need be a PushCtx (struct pointer or PushCtx).
	ctxType := ctype.In(0)
	if !ctxType.Implements(reflect.TypeOf((*PushCtx)(nil)).Elem()) {

		return nil, errors.Errorf("push-handler: %s's first arg need implement tp.PushCtx: %s", typeString, ctxType)
	}

	var handleFunc func(*handlerCtx, reflect.Value)

	switch ctxType.Kind() {
	default:
		return nil, errors.Errorf("push-handler: %s's first arg must be tp.PushCtx type or struct pointer: %s", typeString, ctxType)

	case reflect.Interface:
		if !reflect.TypeOf((*PushCtx)(nil)).Elem().Implements(reflect.New(ctxType).Type().Elem()) {
			return nil, errors.Errorf("push-handler: %s's first arg must be tp.PushCtx type or struct pointer: %s", typeString, ctxType)
		}

		handleFunc = func(ctx *handlerCtx, argValue reflect.Value) {
			rets := cValue.Call([]reflect.Value{reflect.ValueOf(ctx), argValue})
			ctx.handleErr, _ = rets[0].Interface().(*Rerror)
		}

	case reflect.Ptr:
		var ctxTypeElem = ctxType.Elem()
		if ctxTypeElem.Kind() != reflect.Struct {
			return nil, errors.Errorf("push-handler: %s's first arg must be tp.PushCtx type or struct pointer: %s", typeString, ctxType)
		}

		iType, ok := ctxTypeElem.FieldByName("PushCtx")
		if !ok || !iType.Anonymous {
			return nil, errors.Errorf("push-handler: %s's first arg do not have anonymous field tp.PushCtx: %s", typeString, ctxType)
		}

		type PushCtrlValue struct {
			ctrl   reflect.Value
			ctxPtr *PushCtx
		}
		var pushCtxOffset = iType.Offset
		var pool = &sync.Pool{
			New: func() interface{} {
				ctrl := reflect.New(ctxTypeElem)
				pushCtxPtr := ctrl.Pointer() + pushCtxOffset
				ctxPtr := (*PushCtx)(unsafe.Pointer(pushCtxPtr))
				return &PushCtrlValue{
					ctrl:   ctrl,
					ctxPtr: ctxPtr,
				}
			},
		}

		handleFunc = func(ctx *handlerCtx, argValue reflect.Value) {
			obj := pool.Get().(*PushCtrlValue)
			*obj.ctxPtr = ctx
			rets := cValue.Call([]reflect.Value{obj.ctrl, argValue})
			ctx.handleErr, _ = rets[0].Interface().(*Rerror)
			pool.Put(obj)
		}
	}

	if pluginContainer == nil {
		pluginContainer = newPluginContainer()
	}
	return []*Handler{&Handler{
		name:            path.Join(pathPrefix, ToUriPath(handlerFuncName(cValue))),
		handleFunc:      handleFunc,
		argElem:         argType.Elem(),
		pluginContainer: pluginContainer,
	}}, nil
}

func isBelongToPullCtx(name string) bool {
	ctype := reflect.TypeOf(PullCtx(new(handlerCtx)))
	for m := 0; m < ctype.NumMethod(); m++ {
		if name == ctype.Method(m).Name {
			return true
		}
	}
	return false
}

func isBelongToPushCtx(name string) bool {
	ctype := reflect.TypeOf(PushCtx(new(handlerCtx)))
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

func ctrlStructName(ctype reflect.Type) string {
	split := strings.Split(ctype.String(), ".")
	return split[len(split)-1]
}

func handlerFuncName(v reflect.Value) string {
	str := objectName(v)
	split := strings.Split(str, ".")
	return split[len(split)-1]
}

func objectName(v reflect.Value) string {
	t := v.Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(v.Pointer()).Name()
	}
	return t.String()
}

// ToUriPath maps struct(func) name to URI path.
func ToUriPath(name string) string {
	p := strings.Replace(name, "__", ".", -1)
	a := strings.Split(p, "_")
	for k, v := range a {
		a[k] = goutil.SnakeString(v)
	}
	p = path.Join(a...)
	p = path.Join("/", p)
	return strings.Replace(p, ".", "_", -1)
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
