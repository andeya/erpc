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

package xfer

import (
	"errors"
	"fmt"
	"math"
)

type (
	// XferPipe transfer filter pipe, handlers from outer-most to inner-most.
	// Note: the length can not be bigger than 255!
	XferPipe struct {
		filters []XferFilter
	}
	// XferFilter handles byte stream of packet when transfer.
	XferFilter interface {
		Id() byte
		OnPack([]byte) ([]byte, error)
		OnUnpack([]byte) ([]byte, error)
	}
)

// Reset resets transfer filter pipe.
func (x *XferPipe) Reset() {
	x.filters = x.filters[:0]
}

// Append appends transfer filter by id.
func (x *XferPipe) Append(filterId ...byte) error {
	for _, id := range filterId {
		filter, err := Get(id)
		if err != nil {
			return err
		}
		x.filters = append(x.filters, filter)
	}
	return x.check()
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

// Ids returns the id list of transfer filters.
func (x *XferPipe) Ids() []byte {
	var ids = make([]byte, x.Len())
	if x.Len() == 0 {
		return ids
	}
	for i, filter := range x.filters {
		ids[i] = filter.Id()
	}
	return ids
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

var xferFilterMap = struct {
	idMap map[byte]XferFilter
}{
	idMap: make(map[byte]XferFilter),
}

// ErrXferPipeTooLong error
var ErrXferPipeTooLong = errors.New("The length of transfer pipe cannot be bigger than 255")

// Reg registers transfer filter.
func Reg(xferFilter XferFilter) {
	if _, ok := xferFilterMap.idMap[xferFilter.Id()]; ok {
		panic(fmt.Sprintf("multi-register transfer filter id: %d", xferFilter.Id()))
	}
	xferFilterMap.idMap[xferFilter.Id()] = xferFilter
}

// Get returns transfer filter.
func Get(id byte) (XferFilter, error) {
	xferFilter, ok := xferFilterMap.idMap[id]
	if !ok {
		return nil, fmt.Errorf("unsupported transfer filter id: %d", id)
	}
	return xferFilter, nil
}
