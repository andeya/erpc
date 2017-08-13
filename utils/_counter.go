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

package utils

// Counter count size
type Counter struct {
	count int
}

// Write count size of writed.
func (c *Counter) Write(b []byte) (int, error) {
	cnt := len(b)
	c.count += cnt
	return cnt, nil
}

// Count returns count.
func (c *Counter) Count() int {
	return c.count
}

// Reset clear count.
func (c *Counter) Reset() {
	c.count = 0
}
