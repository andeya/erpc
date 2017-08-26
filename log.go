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
	"log"

	"github.com/henrylee2cn/go-logging"
	"github.com/henrylee2cn/go-logging/color"
	"github.com/henrylee2cn/goutil/graceful"
)

// Logger interface
type Logger interface {
	// // Print formats using the default formats for its operands and writes to standard output.
	// // Spaces are added between operands when neither is a string.
	// // It returns the number of bytes written and any write error encountered.
	// Print(args ...interface{})

	// Printf formats according to a format specifier and writes to standard output.
	// It returns the number of bytes written and any write error encountered.
	Printf(format string, args ...interface{})

	// // Fatal is equivalent to Critica followed by a call to os.Exit(1).
	// Fatal(args ...interface{})

	// Fatalf is equivalent to Criticalf followed by a call to os.Exit(1).
	Fatalf(format string, args ...interface{})

	// // Panic is equivalent to Critical followed by a call to panic().
	// Panic(args ...interface{})

	// Panicf is equivalent to Criticalf followed by a call to panic().
	Panicf(format string, args ...interface{})

	// // Critical logs a message using CRITICAL as log level.
	// Critical(args ...interface{})

	// Criticalf logs a message using CRITICAL as log level.
	Criticalf(format string, args ...interface{})

	// // Error logs a message using ERROR as log level.
	// Error(args ...interface{})

	// Errorf logs a message using ERROR as log level.
	Errorf(format string, args ...interface{})

	// // Warn logs a message using WARNING as log level.
	// Warn(args ...interface{})

	// Warnf logs a message using WARNING as log level.
	Warnf(format string, args ...interface{})

	// // Notice logs a message using NOTICE as log level.
	// Notice(args ...interface{})

	// Noticef logs a message using NOTICE as log level.
	Noticef(format string, args ...interface{})

	// // Info logs a message using INFO as log level.
	// Info(args ...interface{})

	// Infof logs a message using INFO as log level.
	Infof(format string, args ...interface{})

	// // Debug logs a message using DEBUG as log level.
	// Debug(args ...interface{})

	// Debugf logs a message using DEBUG as log level.
	Debugf(format string, args ...interface{})

	// // Trace logs a message using TRACE as log level.
	// Trace(args ...interface{})

	// Tracef logs a message using TRACE as log level.
	Tracef(format string, args ...interface{})
}

var (
	// global logger
	globalLogger Logger
	// __loglevel__: PRINT CRITICAL ERROR WARNING NOTICE INFO DEBUG TRACE
	__loglevel__ = "TRACE"
)

func init() {
	setRawlogger()
}

func setRawlogger() {
	var consoleLogBackend = &logging.LogBackend{
		Logger: log.New(color.NewColorableStdout(), "", 0),
		Color:  true,
	}
	consoleFormat := logging.MustStringFormatter("[%{time:2006/01/02 15:04:05.000}] %{color:bold}[%{level:.4s}]%{color:reset} %{message}<%{longfile}>")
	consoleBackendLevel := logging.AddModuleLevel(logging.NewBackendFormatter(consoleLogBackend, consoleFormat))
	level, err := logging.LogLevel(__loglevel__)
	if err != nil {
		panic(err)
	}
	consoleBackendLevel.SetLevel(level, "")
	logger := logging.NewLogger("teleport")
	logger.SetBackend(consoleBackendLevel)
	logger.ExtraCalldepth++

	SetLogger(logger)
}

// SetRawlogLevel sets the default logger's level.
// Note: Concurrent is not safe!
func SetRawlogLevel(level string) {
	__loglevel__ = level
	setRawlogger()
}

// SetLogger sets global logger.
// Note: Concurrent is not safe!
func SetLogger(logger Logger) {
	if logger == nil {
		return
	}
	globalLogger = logger
	graceful.SetLog(logger)
}

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
func Printf(format string, args ...interface{}) {
	globalLogger.Printf(format, args...)
}

// Fatalf is equivalent to l.Criticalf followed by a call to os.Exit(1).
func Fatalf(format string, args ...interface{}) {
	globalLogger.Fatalf(format, args...)
}

// Panicf is equivalent to l.Criticalf followed by a call to panic().
func Panicf(format string, args ...interface{}) {
	globalLogger.Panicf(format, args...)
}

// Criticalf logs a message using CRITICAL as log level.
func Criticalf(format string, args ...interface{}) {
	globalLogger.Criticalf(format, args...)
}

// Errorf logs a message using ERROR as log level.
func Errorf(format string, args ...interface{}) {
	globalLogger.Errorf(format, args...)
}

// Warnf logs a message using WARNING as log level.
func Warnf(format string, args ...interface{}) {
	globalLogger.Warnf(format, args...)
}

// Noticef logs a message using NOTICE as log level.
func Noticef(format string, args ...interface{}) {
	globalLogger.Noticef(format, args...)
}

// Infof logs a message using INFO as log level.
func Infof(format string, args ...interface{}) {
	globalLogger.Infof(format, args...)
}

// Debugf logs a message using DEBUG as log level.
func Debugf(format string, args ...interface{}) {
	globalLogger.Debugf(format, args...)
}

// Tracef logs a message using TRACE as log level.
func Tracef(format string, args ...interface{}) {
	globalLogger.Tracef(format, args...)
}
