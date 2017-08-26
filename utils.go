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
	"crypto/tls"
	"os"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil/pool"
)

var (
	maxGoroutinesAmount      int
	maxGoroutineIdleDuration time.Duration
	_gopool                  *pool.GoPool
	newGopoolOnce            sync.Once
)

// Go go func
func Go(fn func()) {
	newGopoolOnce.Do(func() {
		_gopool = pool.NewGoPool(maxGoroutinesAmount, maxGoroutineIdleDuration)
	})
	var err error
	for {
		if err = _gopool.Go(fn); err != nil {
			Warnf("%s", err.Error())
			time.Sleep(time.Millisecond * 20)
		} else {
			break
		}
	}
}

func init() {
	Printf("The current process PID: %d", os.Getpid())
}

func newTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	var tlsConfig *tls.Config
	if len(certFile) > 0 && len(keyFile) > 0 {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			// NextProtos:               []string{"http/1.1", "h2"},
			PreferServerCipherSuites: true,
		}
	}
	return tlsConfig, nil
}
