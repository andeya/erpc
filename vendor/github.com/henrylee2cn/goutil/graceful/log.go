// graceful package shutdown or reboot current process gracefully.
//
// Copyright 2016 HenryLee. All Rights Reserved.
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

package graceful

import (
	logPkg "log"
)

var log LoggerWithFlusher = loggerWithFlusher{new(logger)}

type (
	// Logger logger interface
	Logger interface {
		// Infof logs a message using INFO as log level.
		Infof(format string, v ...interface{})
		// Errorf logs a message using ERROR as log level.
		Errorf(format string, v ...interface{})
	}
	// LoggerWithFlusher logger interface with flusher
	LoggerWithFlusher interface {
		Logger
		// Flush writes any buffered log to the underlying io.Writer.
		Flush() error
	}
	logger            struct{}
	loggerWithFlusher struct{ Logger }
)

func (l *logger) Infof(format string, v ...interface{}) {
	logPkg.Printf("[I] "+format, v...)
}

func (l *logger) Errorf(format string, v ...interface{}) {
	logPkg.Printf("[E] "+format, v...)
}

func (l loggerWithFlusher) Flush() error {
	nl, ok := l.Logger.(LoggerWithFlusher)
	if ok {
		return nl.Flush()
	}
	return nil
}

// SetLog resets logger.
// NOTE:
//  the logger is best to implement LoggerWithFlusher interface
func SetLog(logger Logger) {
	l, ok := logger.(LoggerWithFlusher)
	if ok {
		log = l
	} else {
		log = loggerWithFlusher{logger}
	}
}
