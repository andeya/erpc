// Package binder is Parameter Binding Verification Plugin for Struct Handler.
//
// Copyright 2018 HenryLee. All Rights Reserved.
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
package binder

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/utils"
	"github.com/henrylee2cn/goutil"
)

/**
Parameter Binding Verification Plugin for Struct Handler.

- Param-Tags

tag   |   key    | required |     value     |   desc
------|----------|----------|---------------|----------------------------------
param |   meta    | no |  (name e.g.`param:"<meta:id>"`)  | It indicates that the parameter is from the meta.
param |   swap    | no |   name (e.g.`param:"<swap:id>"`)  | It indicates that the parameter is from the context swap.
param |   desc   |      no      |     (e.g.`param:"<desc:id>"`)   | Parameter Description
param |   len    |      no      |   (e.g.`param:"<len:3:6>"`)  | Length range [a,b] of parameter's value
param |   range  |      no      |   (e.g.`param:"<range:0:10>"`)   | Numerical range [a,b] of parameter's value
param |  nonzero |      no      |    -    | Not allowed to zero
param |  regexp  |      no      |   (e.g.`param:"<regexp:^\\w+$>"`)  | Regular expression validation
param |   stat   |      no      |(e.g.`param:"<stat:100002:wrong password format>"`)| Custom error code and message

NOTES:
* `param:"-"` means ignore
* Encountered untagged exportable anonymous structure field, automatic recursive resolution
* Parameter name is the name of the structure field converted to snake format
* If the parameter is not from `meta` or `swap`, it is the default from the body

- Field-Types

base    |   slice    | special
--------|------------|------------
string  |  []string  | [][]byte
byte    |  []byte    | [][]uint8
uint8   |  []uint8   | struct
bool    |  []bool    |
int     |  []int     |
int8    |  []int8    |
int16   |  []int16   |
int32   |  []int32   |
int64   |  []int64   |
uint8   |  []uint8   |
uint16  |  []uint16  |
uint32  |  []uint32  |
uint64  |  []uint64  |
float32 |  []float32 |
float64 |  []float64 |
*/
type (
	// StructArgsBinder a plugin that binds and validates structure type parameters.
	StructArgsBinder struct {
		binders map[string]*Params
		errFunc ErrorFunc
	}
	// ErrorFunc creates an relational error.
	ErrorFunc func(handlerName, paramName, reason string) *erpc.Status
)

var (
	_ erpc.PostRegPlugin          = new(StructArgsBinder)
	_ erpc.PostReadCallBodyPlugin = new(StructArgsBinder)
)

// NewStructArgsBinder creates a plugin that binds and validates structure type parameters.
func NewStructArgsBinder(fn ErrorFunc) *StructArgsBinder {
	s := &StructArgsBinder{
		binders: make(map[string]*Params),
		errFunc: fn,
	}
	s.SetErrorFunc(fn)
	return s
}

var (
	_ erpc.PostRegPlugin          = new(StructArgsBinder)
	_ erpc.PostReadCallBodyPlugin = new(StructArgsBinder)
)

// SetErrorFunc sets the binding or balidating error function.
// NOTE: If fn=nil, set as default.
func (s *StructArgsBinder) SetErrorFunc(fn ErrorFunc) {
	if fn != nil {
		s.errFunc = fn
		return
	}
	s.errFunc = func(handlerName, paramName, reason string) *erpc.Status {
		return erpc.NewStatus(
			erpc.CodeBadMessage,
			"Invalid Parameter",
			fmt.Sprintf(`{"handler": %q, "param": %q, "reason": %q}`, handlerName, paramName, reason),
		)
	}
}

// Name returns the plugin name.
func (*StructArgsBinder) Name() string {
	return "StructArgsBinder"
}

// PostReg preprocessing struct handler.
func (s *StructArgsBinder) PostReg(h *erpc.Handler) error {
	if h.ArgElemType().Kind() != reflect.Struct {
		return nil
	}
	params := newParams(h.Name(), s)
	err := params.addFields([]int{}, h.ArgElemType(), h.NewArgValue().Elem())
	if err != nil {
		erpc.Fatalf("%v", err)
	}
	s.binders[h.Name()] = params
	return nil
}

// PostReadCallBody binds and validates the registered struct handler.
func (s *StructArgsBinder) PostReadCallBody(ctx erpc.ReadCtx) *erpc.Status {
	params, ok := s.binders[ctx.ServiceMethod()]
	if !ok {
		return nil
	}
	bodyValue := reflect.ValueOf(ctx.Input().Body())
	stat := params.bindAndValidate(bodyValue, ctx.CopyMeta(), ctx.Swap())
	if !stat.OK() {
		return stat
	}
	return nil
}

// Params struct handler information for binding and validation
type Params struct {
	handlerName string
	params      []*Param
	binder      *StructArgsBinder
}

// struct binder parameters'tag
const (
	TAG_PARAM        = "param"   // request param tag name
	TAG_IGNORE_PARAM = "-"       // ignore request param tag value
	KEY_META         = "meta"    // meta param(optional), value means parameter(optional)
	KEY_SWAP         = "swap"    // swap param from the context swap(ctx.Swap()) (optional), value means parameter(optional)
	KEY_DESC         = "desc"    // request param description
	KEY_LEN          = "len"     // length range of param's value
	KEY_RANGE        = "range"   // numerical range of param's value
	KEY_NONZERO      = "nonzero" // param`s value can not be zero
	KEY_REGEXP       = "regexp"  // verify the value of the param with a regular expression(param value can not be null)
	KEY_RERR         = "stat"    // the custom error code and message for binding or validating
)

func newParams(handlerName string, binder *StructArgsBinder) *Params {
	return &Params{
		handlerName: handlerName,
		params:      make([]*Param, 0),
		binder:      binder,
	}
}

func (p *Params) addFields(parentIndexPath []int, t reflect.Type, v reflect.Value) error {
	var err error
	var deep = len(parentIndexPath) + 1
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		indexPath := make([]int, deep)
		copy(indexPath, parentIndexPath)
		indexPath[deep-1] = i

		var field = t.Field(i)
		var value = v.Field(i)
		canSet := v.Field(i).CanSet()

		tag, ok := field.Tag.Lookup(TAG_PARAM)
		if !ok {
			if canSet && field.Anonymous {
				if field.Type.Kind() == reflect.Struct {
					if err = p.addFields(indexPath, field.Type, value); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("%s.%s anonymous field can only be struct type", t.String(), field.Name)
				}
			}
			continue
		}

		if tag == TAG_IGNORE_PARAM {
			continue
		}

		if !canSet {
			return fmt.Errorf("%s.%s can not be a non-settable field", t.String(), field.Name)
		}

		if field.Type.Kind() == reflect.Ptr {
			return fmt.Errorf("%s.%s can not be a pointer field", t.String(), field.Name)
		}

		var parsedTags = parseTags(tag)
		var paramTypeString = field.Type.String()
		var kind = field.Type.Kind()

		if _, ok := parsedTags[KEY_LEN]; ok {
			if kind != reflect.String && kind != reflect.Slice && kind != reflect.Map && kind != reflect.Array {
				return fmt.Errorf("%s.%s invalid `len` tag for field value", t.String(), field.Name)
			}
		}
		if _, ok := parsedTags[KEY_RANGE]; ok {
			switch paramTypeString {
			case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
			case "[]int", "[]int8", "[]int16", "[]int32", "[]int64", "[]uint", "[]uint8", "[]uint16", "[]uint32", "[]uint64", "[]float32", "[]float64":
			default:
				return fmt.Errorf("%s.%s invalid `range` tag for non-number field", t.String(), field.Name)
			}
		}
		if _, ok := parsedTags[KEY_REGEXP]; ok {
			if paramTypeString != "string" && paramTypeString != "[]string" {
				return fmt.Errorf("%s.%s invalid `regexp` tag for non-string field", t.String(), field.Name)
			}
		}

		fd := &Param{
			handlerName: p.handlerName,
			indexPath:   indexPath,
			tags:        parsedTags,
			rawTag:      field.Tag,
			rawValue:    value,
			binder:      p.binder,
		}
		statTag, ok := fd.tags[KEY_RERR]
		if ok {
			idx := strings.Index(statTag, ":")
			if idx != -1 {
				if codeStr := strings.TrimSpace(statTag[:idx]); len(codeStr) > 0 {
					statCode, err := strconv.Atoi(codeStr)
					if err == nil {
						fd.statCode = int32(statCode)
					} else {
						return fmt.Errorf("%s.%s invalid `stat` tag (correct example: `<stat: 100001: Invalid Parameter>`)", t.String(), field.Name)
					}
				}
				fd.statMsg = strings.TrimSpace(statTag[idx+1:])
			} else {
				return fmt.Errorf("%s.%s invalid `stat` tag (correct example: `<stat: 100001: Invalid Parameter>`)", t.String(), field.Name)
			}
		}

		if fd.name, ok = parsedTags[KEY_META]; ok {
			fd.position = KEY_META
		} else if fd.name, ok = parsedTags[KEY_SWAP]; ok {
			fd.position = KEY_SWAP
		}
		if fd.name == "" {
			fd.name = goutil.SnakeString(field.Name)
		}

		if err = fd.makeVerifyFuncs(); err != nil {
			return fmt.Errorf("%s.%s invalid validation failed: %s", t.String(), field.Name, err.Error())
		}

		p.params = append(p.params, fd)
	}

	return nil
}

func (p *Params) fieldsForBinding(structElem reflect.Value) []reflect.Value {
	count := len(p.params)
	fields := make([]reflect.Value, count)
	for i := 0; i < count; i++ {
		value := structElem
		param := p.params[i]
		for _, index := range param.indexPath {
			value = value.Field(index)
		}
		fields[i] = value
	}
	return fields
}

func (p *Params) bindAndValidate(structValue reflect.Value, meta *utils.Args, swap goutil.Map) (stat *erpc.Status) {
	defer func() {
		if r := recover(); r != nil {
			stat = p.binder.errFunc(p.handlerName, "", fmt.Sprint(r))
		}
	}()
	var (
		err    error
		fields = p.fieldsForBinding(reflect.Indirect(structValue))
	)
	for i, param := range p.params {
		value := fields[i]
		// bind meta or swap param
		switch param.position {
		case KEY_META:
			paramValues := meta.PeekMulti(param.name)
			if len(paramValues) > 0 {
				if err = convertAssign(value, toSliceString(paramValues)); err != nil {
					return param.fixStatus(p.binder.errFunc(param.handlerName, param.name, err.Error()))
				}
			}
		case KEY_SWAP:
			paramValue, ok := swap.Load(param.name)
			if ok {
				value = reflect.Indirect(value)
				canSet := value.CanSet()
				var srcValue reflect.Value
				if canSet {
					srcValue = reflect.Indirect(reflect.ValueOf(paramValue))
					destType := value.Type()
					srcType := srcValue.Type()
					canSet = srcType.AssignableTo(destType)
					if !canSet {
						if srcType.ConvertibleTo(destType) {
							srcValue = srcValue.Convert(destType)
							canSet = srcValue.Type().AssignableTo(destType)
						}
					}
				}
				if !canSet {
					return param.fixStatus(p.binder.errFunc(
						param.handlerName,
						param.name,
						value.Type().Name()+" can not be setted"),
					)
				}
				value.Set(srcValue)
			}
		}
		if stat = param.validate(value); !stat.OK() {
			return stat
		}
	}
	return
}

func toSliceString(b [][]byte) []string {
	if len(b) == 0 {
		return nil
	}
	a := make([]string, len(b))
	for k, v := range b {
		a[k] = goutil.BytesToString(v)
	}
	return a
}

// parseTags returns the key-value in the tag string.
// If the tag does not have the conventional format,
// the value returned by parseTags is unspecified.
func parseTags(tag string) map[string]string {
	var values = map[string]string{}

	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] != '<' {
			i++
		}
		if i >= len(tag) || tag[i] != '<' {
			break
		}
		i++

		// Skip the left Spaces
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		if i >= len(tag) {
			break
		}

		tag = tag[i:]
		if tag == "" {
			break
		}

		var name, value string
		var hadName bool
		i = 0
	PAIR:
		for i < len(tag) {
			switch tag[i] {
			case ':':
				if hadName {
					i++
					continue
				}
				name = strings.TrimRight(tag[:i], " ")
				tag = strings.TrimLeft(tag[i+1:], " ")
				hadName = true
				i = 0
			case '\\':
				i++
				// Fix the escape character of `\\<` or `\\>`
				if tag[i] == '<' || tag[i] == '>' {
					tag = tag[:i-1] + tag[i:]
				} else {
					i++
				}
			case '>':
				if !hadName {
					name = strings.TrimRight(tag[:i], " ")
				} else {
					value = strings.TrimRight(tag[:i], " ")
				}
				values[name] = value
				break PAIR
			default:
				i++
			}
		}
		if i >= len(tag) {
			break
		}
		tag = tag[i+1:]
	}
	return values
}

// Param use the struct field to define a request parameter model
type Param struct {
	handlerName string // handler name
	name        string // param name
	indexPath   []int
	position    string            // param position
	tags        map[string]string // struct tags for this param
	verifyFuncs []func(reflect.Value) error
	rawTag      reflect.StructTag // the raw tag
	rawValue    reflect.Value     // the raw tag value
	statCode    int32             // the custom error code for binding or validating
	statMsg     string            // the custom error message for binding or validating
	binder      *StructArgsBinder
}

const (
	stringTypeString = "string"
	bytesTypeString  = "[]byte"
	bytes2TypeString = "[]uint8"
)

// Raw gets the param's original value
func (param *Param) Raw() interface{} {
	return param.rawValue.Interface()
}

// Name gets parameter field name
func (param *Param) Name() string {
	return param.name
}

// Description gets the description value for the param
func (param *Param) Description() string {
	return param.tags[KEY_DESC]
}

// validate tests if the param conforms to it's validation constraints specified
// int the KEY_REGEXP struct tag
func (param *Param) validate(value reflect.Value) (stat *erpc.Status) {
	defer func() {
		if r := recover(); r != nil {
			stat = param.fixStatus(param.binder.errFunc(param.handlerName, param.name, fmt.Sprint(r)))
		}
	}()
	var err error
	for _, fn := range param.verifyFuncs {
		if err = fn(value); err != nil {
			return param.fixStatus(param.binder.errFunc(param.handlerName, param.name, err.Error()))
		}
	}
	return nil
}

func (param *Param) makeVerifyFuncs() (err error) {
	defer func() {
		p := recover()
		if p != nil {
			err = fmt.Errorf("%v", p)
		}
	}()
	// length
	if tuple, ok := param.tags[KEY_LEN]; ok {
		if fn, err := validateLen(tuple); err == nil {
			param.verifyFuncs = append(param.verifyFuncs, fn)
		} else {
			return err
		}
	}
	// range
	if tuple, ok := param.tags[KEY_RANGE]; ok {
		if fn, err := validateRange(tuple); err == nil {
			param.verifyFuncs = append(param.verifyFuncs, fn)
		} else {
			return err
		}
	}
	// nonzero
	if _, ok := param.tags[KEY_NONZERO]; ok {
		if fn, err := validateNonZero(); err == nil {
			param.verifyFuncs = append(param.verifyFuncs, fn)
		} else {
			return err
		}
	}
	// regexp
	if reg, ok := param.tags[KEY_REGEXP]; ok {
		var isStrings = param.rawValue.Kind() == reflect.Slice
		if fn, err := validateRegexp(isStrings, reg); err == nil {
			param.verifyFuncs = append(param.verifyFuncs, fn)
		} else {
			return err
		}
	}
	return
}

func parseTuple(tuple string) (string, string) {
	c := strings.Split(tuple, ":")
	var a, b string
	switch len(c) {
	case 1:
		a = c[0]
		if len(a) > 0 {
			return a, a
		}
	case 2:
		a = c[0]
		b = c[1]
		if len(a) > 0 || len(b) > 0 {
			return a, b
		}
	}
	panic("invalid validation tuple")
}

func validateNonZero() (func(value reflect.Value) error, error) {
	return func(value reflect.Value) error {
		obj := value.Interface()
		if obj == reflect.Zero(value.Type()).Interface() {
			return errors.New("zero value")
		}
		return nil
	}, nil
}

func validateLen(tuple string) (func(value reflect.Value) error, error) {
	var a, b = parseTuple(tuple)
	var min, max int
	var err error
	if len(a) > 0 {
		min, err = strconv.Atoi(a)
		if err != nil {
			return nil, err
		}
	}
	if len(b) > 0 {
		max, err = strconv.Atoi(b)
		if err != nil {
			return nil, err
		}
	}
	return func(value reflect.Value) error {
		length := value.Len()
		if len(a) > 0 {
			if length < min {
				return fmt.Errorf("shorter than %s: %v", a, value.Interface())
			}
		}
		if len(b) > 0 {
			if length > max {
				return fmt.Errorf("longer than %s: %v", b, value.Interface())
			}
		}
		return nil
	}, nil
}

const accuracy = 0.0000001

func validateRange(tuple string) (func(value reflect.Value) error, error) {
	var a, b = parseTuple(tuple)
	var min, max float64
	var err error
	if len(a) > 0 {
		min, err = strconv.ParseFloat(a, 64)
		if err != nil {
			return nil, err
		}
	}
	if len(b) > 0 {
		max, err = strconv.ParseFloat(b, 64)
		if err != nil {
			return nil, err
		}
	}
	return func(value reflect.Value) error {
		var f64 float64
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			f64 = float64(value.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			f64 = float64(value.Uint())
		case reflect.Float32, reflect.Float64:
			f64 = value.Float()
		}
		if len(a) > 0 {
			if math.Min(f64, min) == f64 && math.Abs(f64-min) > accuracy {
				return fmt.Errorf("smaller than %s: %v", a, value.Interface())
			}
		}
		if len(b) > 0 {
			if math.Max(f64, max) == f64 && math.Abs(f64-max) > accuracy {
				return fmt.Errorf("bigger than %s: %v", b, value.Interface())
			}
		}
		return nil
	}, nil
}

func validateRegexp(isStrings bool, reg string) (func(value reflect.Value) error, error) {
	re, err := regexp.Compile(reg)
	if err != nil {
		return nil, err
	}
	if !isStrings {
		return func(value reflect.Value) error {
			s := value.String()
			if !re.MatchString(s) {
				return fmt.Errorf("not match %s: %s", reg, s)
			}
			return nil
		}, nil
	} else {
		return func(value reflect.Value) error {
			for _, s := range value.Interface().([]string) {
				if !re.MatchString(s) {
					return fmt.Errorf("not match %s: %s", reg, s)
				}
			}
			return nil
		}, nil
	}
}

func (param *Param) fixStatus(stat *erpc.Status) *erpc.Status {
	if param.statMsg != "" {
		stat.SetMsg(param.statMsg)
	}
	if param.statCode != 0 {
		stat.SetCode(param.statCode)
	}
	return stat
}

func convertAssign(dest reflect.Value, src []string) (err error) {
	if len(src) == 0 {
		return nil
	}

	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("%s can not be setted", dest.Type().Name())
	}

	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v", p)
		}
	}()

	switch dest.Interface().(type) {
	case string:
		dest.Set(reflect.ValueOf(src[0]))
		return nil

	case []string:
		dest.Set(reflect.ValueOf(src))
		return nil

	case []byte:
		dest.Set(reflect.ValueOf([]byte(src[0])))
		return nil

	case [][]byte:
		b := make([][]byte, 0, len(src))
		for _, s := range src {
			b = append(b, []byte(s))
		}
		dest.Set(reflect.ValueOf(b))
		return nil

	case bool:
		dest.Set(reflect.ValueOf(parseBool(src[0])))
		return nil

	case []bool:
		b := make([]bool, 0, len(src))
		for _, s := range src {
			b = append(b, parseBool(s))
		}
		dest.Set(reflect.ValueOf(b))
		return nil
	}

	switch dest.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i64, err := strconv.ParseInt(src[0], 10, dest.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting type %T (%q) to a %s: %v", src, src[0], dest.Kind(), err)
		}
		dest.SetInt(i64)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u64, err := strconv.ParseUint(src[0], 10, dest.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting type %T (%q) to a %s: %v", src, src[0], dest.Kind(), err)
		}
		dest.SetUint(u64)
		return nil

	case reflect.Float32, reflect.Float64:
		f64, err := strconv.ParseFloat(src[0], dest.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting type %T (%q) to a %s: %v", src, src[0], dest.Kind(), err)
		}
		dest.SetFloat(f64)
		return nil

	case reflect.Slice:
		member := dest.Type().Elem()
		switch member.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			for _, s := range src {
				i64, err := strconv.ParseInt(s, 10, member.Bits())
				if err != nil {
					err = strconvErr(err)
					return fmt.Errorf("converting type %T (%q) to a %s: %v", src, s, dest.Kind(), err)
				}
				dest.Set(reflect.Append(dest, reflect.ValueOf(i64).Convert(member)))
			}
			return nil

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			for _, s := range src {
				u64, err := strconv.ParseUint(s, 10, member.Bits())
				if err != nil {
					err = strconvErr(err)
					return fmt.Errorf("converting type %T (%q) to a %s: %v", src, s, dest.Kind(), err)
				}
				dest.Set(reflect.Append(dest, reflect.ValueOf(u64).Convert(member)))
			}
			return nil

		case reflect.Float32, reflect.Float64:
			for _, s := range src {
				f64, err := strconv.ParseFloat(s, member.Bits())
				if err != nil {
					err = strconvErr(err)
					return fmt.Errorf("converting type %T (%q) to a %s: %v", src, s, dest.Kind(), err)
				}
				dest.Set(reflect.Append(dest, reflect.ValueOf(f64).Convert(member)))
			}
			return nil
		}
	}

	return fmt.Errorf("unsupported storing type %T into type %s", src, dest.Kind())
}

func parseBool(val string) bool {
	switch strings.TrimSpace(strings.ToLower(val)) {
	case "true", "on", "1":
		return true
	}
	return false
}

func strconvErr(err error) error {
	if ne, ok := err.(*strconv.NumError); ok {
		return ne.Err
	}
	return err
}
