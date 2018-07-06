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
	"log"
	"os"
	"sync"

	"github.com/henrylee2cn/go-logging"
	"github.com/henrylee2cn/go-logging/color"
	"github.com/henrylee2cn/goutil/graceful"
)

// Logger interface
type Logger interface {
	// Level returns the logger's level.
	Level() string
	// SetLevel sets the logger's level.
	SetLevel(level string)
	// Printf formats according to a format specifier and writes to standard output.
	// It returns the number of bytes written and any write error encountered.
	Printf(format string, a ...interface{})
	// Fatalf is equivalent to Criticalf followed by a call to os.Exit(1).
	Fatalf(format string, a ...interface{})
	// Panicf is equivalent to Criticalf followed by a call to panic().
	Panicf(format string, a ...interface{})
	// Criticalf logs a message using CRITICAL as log level.
	Criticalf(format string, a ...interface{})
	// Errorf logs a message using ERROR as log level.
	Errorf(format string, a ...interface{})
	// Warnf logs a message using WARNING as log level.
	Warnf(format string, a ...interface{})
	// Noticef logs a message using NOTICE as log level.
	Noticef(format string, a ...interface{})
	// Infof logs a message using INFO as log level.
	Infof(format string, a ...interface{})
	// Debugf logs a message using DEBUG as log level.
	Debugf(format string, a ...interface{})
	// Tracef logs a message using TRACE as log level.
	Tracef(format string, a ...interface{})
}

var (
	// global logger
	globalLogger = func() Logger {
		logger := newDefaultlogger("TRACE")
		graceful.SetLog(logger)
		return logger
	}()
)

func newDefaultlogger(level string) Logger {
	l := &defaultLogger{
		level: level,
	}
	l.newSet()
	return l
}

type defaultLogger struct {
	*logging.Logger
	level string
	mu    sync.RWMutex
}

func (l *defaultLogger) newSet() {
	var consoleLogBackend = &logging.LogBackend{
		Logger:    log.New(color.NewColorableStdout(), "", 0),
		ErrLogger: log.New(color.NewColorableStderr(), "", 0),
		Color:     true,
	}
	consoleFormat := logging.MustStringFormatter("[%{time:2006/01/02 15:04:05.000}] [%{color:bold}%{level:.4s}%{color:reset}] %{message} <%{longfile}>")
	consoleBackendLevel := logging.AddModuleLevel(logging.NewBackendFormatter(consoleLogBackend, consoleFormat))
	level, err := logging.LogLevel(l.level)
	if err != nil {
		panic(err)
	}
	consoleBackendLevel.SetLevel(level, "")
	l.Logger = logging.NewLogger("teleport")
	l.Logger.SetBackend(consoleBackendLevel)
	l.Logger.ExtraCalldepth++
}

// Level returns the logger's level.
// Note: Concurrent is not safe!
func (l *defaultLogger) Level() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// SetLevel sets the logger's level.
// Note:
// Concurrent is not safe!
// the teleport default logger's level list: OFF PRINT CRITICAL ERROR WARNING NOTICE INFO DEBUG TRACE
func (l *defaultLogger) SetLevel(level string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
	l.newSet()
}

// GetLogger gets global logger.
func GetLogger() Logger {
	return globalLogger
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

// GetLoggerLevel gets the logger's level.
func GetLoggerLevel() string {
	return globalLogger.Level()
}

// SetLoggerLevel sets the logger's level.
func SetLoggerLevel(level string) {
	globalLogger.SetLevel(level)
}

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
func Printf(format string, a ...interface{}) {
	globalLogger.Printf(format, a...)
}

// Fatalf is equivalent to l.Criticalf followed by a call to os.Exit(1).
func Fatalf(format string, a ...interface{}) {
	globalLogger.Fatalf(format, a...)
	os.Exit(1)
}

// Panicf is equivalent to l.Criticalf followed by a call to panic().
func Panicf(format string, a ...interface{}) {
	globalLogger.Panicf(format, a...)
}

// Criticalf logs a message using CRITICAL as log level.
func Criticalf(format string, a ...interface{}) {
	globalLogger.Criticalf(format, a...)
}

// Errorf logs a message using ERROR as log level.
func Errorf(format string, a ...interface{}) {
	globalLogger.Errorf(format, a...)
}

// Warnf logs a message using WARNING as log level.
func Warnf(format string, a ...interface{}) {
	globalLogger.Warnf(format, a...)
}

// Noticef logs a message using NOTICE as log level.
func Noticef(format string, a ...interface{}) {
	globalLogger.Noticef(format, a...)
}

// Infof logs a message using INFO as log level.
func Infof(format string, a ...interface{}) {
	globalLogger.Infof(format, a...)
}

// Debugf logs a message using DEBUG as log level.
func Debugf(format string, a ...interface{}) {
	globalLogger.Debugf(format, a...)
}

// Tracef logs a message using TRACE as log level.
func Tracef(format string, a ...interface{}) {
	globalLogger.Tracef(format, a...)
}
