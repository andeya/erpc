// Package xfer is transfer filter set.
//
// Copyright 2017 HenryLee. All Rights Reserved.
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
//
package xfer

import (
	"errors"
	"fmt"
	"math"
)

// XferFilter handles byte stream of message when transfer.
type XferFilter interface {
	// ID returns transfer filter id.
	ID() byte
	// Name returns transfer filter name.
	Name() string
	// OnPack performs filtering on packing.
	OnPack([]byte) ([]byte, error)
	// OnUnpack performs filtering on unpacking.
	OnUnpack([]byte) ([]byte, error)
}

var xferFilterMap = struct {
	idMap   map[byte]XferFilter
	nameMap map[string]XferFilter
}{
	idMap:   make(map[byte]XferFilter),
	nameMap: make(map[string]XferFilter),
}

// ErrXferPipeTooLong error
var ErrXferPipeTooLong = errors.New("The length of transfer pipe cannot be bigger than 255")

// Reg registers transfer filter.
func Reg(xferFilter XferFilter) {
	id := xferFilter.ID()
	name := xferFilter.Name()
	if _, ok := xferFilterMap.idMap[id]; ok {
		panic(fmt.Sprintf("multi-register transfer filter id: %d", xferFilter.ID()))
	}
	if _, ok := xferFilterMap.nameMap[name]; ok {
		panic("multi-register transfer filter name: " + xferFilter.Name())
	}
	xferFilterMap.idMap[id] = xferFilter
	xferFilterMap.nameMap[name] = xferFilter
}

// Get returns transfer filter by id.
func Get(id byte) (XferFilter, error) {
	xferFilter, ok := xferFilterMap.idMap[id]
	if !ok {
		return nil, fmt.Errorf("unsupported transfer filter id: %d", id)
	}
	return xferFilter, nil
}

// GetByName returns transfer filter by name.
func GetByName(name string) (XferFilter, error) {
	xferFilter, ok := xferFilterMap.nameMap[name]
	if !ok {
		return nil, fmt.Errorf("unsupported transfer filter name: %s", name)
	}
	return xferFilter, nil
}

// XferPipe transfer filter pipe, handlers from outer-most to inner-most.
// NOTE: the length can not be bigger than 255!
type XferPipe struct {
	filters []XferFilter
}

// NewXferPipe creates a new transfer filter pipe.
func NewXferPipe() *XferPipe {
	return new(XferPipe)
}

// Reset resets transfer filter pipe.
func (x *XferPipe) Reset() {
	x.filters = x.filters[:0]
}

// Append appends transfer filter by id.
func (x *XferPipe) Append(filterID ...byte) error {
	for _, id := range filterID {
		filter, err := Get(id)
		if err != nil {
			return err
		}
		x.filters = append(x.filters, filter)
	}
	return x.check()
}

// AppendFrom appends transfer filters from a *XferPipe.
func (x *XferPipe) AppendFrom(src *XferPipe) {
	for _, filter := range src.filters {
		x.filters = append(x.filters, filter)
	}
}

func (x *XferPipe) check() error {
	if x.Len() > math.MaxUint8 {
		return ErrXferPipeTooLong
	}
	return nil
}

// Len returns the length of transfer pipe.
func (x *XferPipe) Len() int {
	if x == nil {
		return 0
	}
	return len(x.filters)
}

// IDs returns the id list of transfer filters.
func (x *XferPipe) IDs() []byte {
	var ids = make([]byte, x.Len())
	if x.Len() == 0 {
		return ids
	}
	for i, filter := range x.filters {
		ids[i] = filter.ID()
	}
	return ids
}

// Names returns the name list of transfer filters.
func (x *XferPipe) Names() []string {
	var names = make([]string, x.Len())
	if x.Len() == 0 {
		return names
	}
	for i, filter := range x.filters {
		names[i] = filter.Name()
	}
	return names
}

// Range calls f sequentially for each XferFilter present in the XferPipe.
// If f returns false, range stops the iteration.
func (x *XferPipe) Range(callback func(idx int, filter XferFilter) bool) {
	for idx, filter := range x.filters {
		if !callback(idx, filter) {
			break
		}
	}
}

// OnPack packs transfer byte stream, from inner-most to outer-most.
func (x *XferPipe) OnPack(data []byte) ([]byte, error) {
	var err error
	for i := x.Len() - 1; i >= 0; i-- {
		if data, err = x.filters[i].OnPack(data); err != nil {
			return data, err
		}
	}
	return data, err
}

// OnUnpack unpacks transfer byte stream, from outer-most to inner-most.
func (x *XferPipe) OnUnpack(data []byte) ([]byte, error) {
	var err error
	var count = x.Len()
	for i := 0; i < count; i++ {
		if data, err = x.filters[i].OnUnpack(data); err != nil {
			return data, err
		}
	}
	return data, err
}
