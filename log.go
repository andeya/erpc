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
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil/graceful"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/teleport/utils"
	"github.com/henrylee2cn/teleport/utils/color"
)

type (
	// LoggerOutputter writes log.
	LoggerOutputter interface {
		// Output writes log, can append time, line and so on information.
		Output(calldepth int, msgBytes []byte, loggerLevel LoggerLevel)
		// Flush writes any buffered log to the underlying io.Writer.
		Flush() error
	}
	// LoggerLevel defines all available log levels for log messages.
	LoggerLevel int
	// Logger logger interface
	Logger interface {
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
)

// Logger levels.
const (
	OFF LoggerLevel = iota
	PRINT
	CRITICAL
	ERROR
	WARNING
	NOTICE
	INFO
	DEBUG
	TRACE
)

var loggerLevelMap = map[LoggerLevel]string{
	OFF:      "OFF",
	PRINT:    "PRINT",
	CRITICAL: "CRITICAL",
	ERROR:    "ERROR",
	WARNING:  "WARNING",
	NOTICE:   "NOTICE",
	INFO:     "INFO",
	DEBUG:    "DEBUG",
	TRACE:    "TRACE",
}

var loggerLevel = DEBUG

func (l LoggerLevel) String() string {
	s, ok := loggerLevelMap[l]
	if !ok {
		return "unknown"
	}
	return s
}

type easyLoggerOutputter struct {
	output func(calldepth int, msgBytes []byte, loggerLevel LoggerLevel)
	flush  func() error
}

// Output writes log.
func (e *easyLoggerOutputter) Output(calldepth int, msgBytes []byte, loggerLevel LoggerLevel) {
	e.output(calldepth, msgBytes, loggerLevel)
}

// Flush writes any buffered log to the underlying io.Writer.
func (e *easyLoggerOutputter) Flush() error {
	return e.flush()
}

var loggerOutputter = func() LoggerOutputter {
	type msg struct {
		logBytes    []byte
		loggerLevel LoggerLevel
	}
	var p = sync.Pool{
		New: func() interface{} {
			return new(msg)
		},
	}
	var loggerLevelTagMap = map[LoggerLevel]string{
		OFF:      color.Bold("OFF"),
		PRINT:    color.Bold("PRIN"),
		CRITICAL: color.Magenta(color.Bold("CRIT")),
		ERROR:    color.Red(color.Bold("ERRO")),
		WARNING:  color.Yellow(color.Bold("WARN")),
		NOTICE:   color.Green(color.Bold("NOTI")),
		INFO:     color.Green(color.Bold("INFO")),
		DEBUG:    color.Cyan(color.Bold("DEBU")),
		TRACE:    color.Cyan(color.Bold("TRAC")),
	}
	var c = make(chan *msg, 1024)
	go func() {
		for m := range c {
			if m == nil {
				continue
			}
			if m.loggerLevel > ERROR || m.loggerLevel == PRINT {
				color.Stderr.Write(m.logBytes)
			} else {
				color.Stdout.Write(m.logBytes)
			}
			p.Put(m)
		}
	}()
	return &easyLoggerOutputter{
		output: func(calldepth int, msgBytes []byte, loggerLevel LoggerLevel) {
			m := p.Get().(*msg)
			buf := utils.AcquireByteBuffer()
			buf.WriteString("[" + time.Now().Format("2006/01/02 15:04:05.000") + "]")
			buf.WriteString(" [" + loggerLevelTagMap[loggerLevel] + "] ")
			buf.Write(msgBytes)
			line := goutil.GetCallLine(calldepth + 1)
			if !strings.Contains(line, "github.com/henrylee2cn/teleport") &&
				!strings.Contains(line, "github.com/henrylee2cn/goutil/graceful") {
				buf.WriteString(" <" + line + ">\n")
			} else {
				buf.WriteByte('\n')
			}
			m.logBytes = goutil.StringToBytes(buf.String())
			m.loggerLevel = loggerLevel
			c <- m
		},
		flush: func() error {
			c <- nil
			for len(c) > 0 {
				runtime.Gosched()
			}
			return nil
		},
	}
}()

// FlushLogger writes any buffered log to the underlying io.Writer.
func FlushLogger() error {
	return loggerOutputter.Flush()
}

// SetLoggerOutputter sets logger outputter.
// NOTE: Concurrent is not safe!
func SetLoggerOutputter(outputter LoggerOutputter) (flusher func() error) {
	loggerOutputter = outputter
	return FlushLogger
}

// SetLoggerLevel sets the logger's level by string.
func SetLoggerLevel(level string) (flusher func() error) {
	for k, v := range loggerLevelMap {
		if v == level {
			loggerLevel = k
			return FlushLogger
		}
	}
	log.Printf("Unknown level string: %s", level)
	return FlushLogger
}

// SetLoggerLevel2 sets the logger's level by number.
func SetLoggerLevel2(level LoggerLevel) (flusher func() error) {
	_, ok := loggerLevelMap[level]
	if !ok {
		log.Printf("Unknown level number: %d", level)
		return FlushLogger
	}
	loggerLevel = level
	return FlushLogger
}

// GetLoggerLevel gets the logger's level.
func GetLoggerLevel() LoggerLevel {
	return loggerLevel
}

// EnableLoggerLevel returns if can print the level of log.
func EnableLoggerLevel(level LoggerLevel) bool {
	if level <= loggerLevel {
		return level != OFF
	}
	return false
}

// GetLogger returns the global logger object.
func GetLogger() Logger {
	return logger
}

func loggerOutput(loggerLevel LoggerLevel, format string, a ...interface{}) {
	if !EnableLoggerLevel(loggerLevel) {
		return
	}
	loggerOutputter.Output(3, goutil.StringToBytes(fmt.Sprintf(format, a...)), loggerLevel)
}

// ************ global logger functions ************

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
func Printf(format string, a ...interface{}) {
	loggerOutput(PRINT, format, a...)
}

// Fatalf is equivalent to l.Criticalf followed by a call to os.Exit(1).
func Fatalf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
	loggerOutputter.Flush()
	os.Exit(1)
}

// Panicf is equivalent to l.Criticalf followed by a call to panic().
func Panicf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
	loggerOutputter.Flush()
	panic(fmt.Sprintf(format, a...))
}

// Criticalf logs a message using CRITICAL as log level.
func Criticalf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
}

// Errorf logs a message using ERROR as log level.
func Errorf(format string, a ...interface{}) {
	loggerOutput(ERROR, format, a...)
}

// Warnf logs a message using WARNING as log level.
func Warnf(format string, a ...interface{}) {
	loggerOutput(WARNING, format, a...)
}

// Noticef logs a message using NOTICE as log level.
func Noticef(format string, a ...interface{}) {
	loggerOutput(NOTICE, format, a...)
}

// Infof logs a message using INFO as log level.
func Infof(format string, a ...interface{}) {
	loggerOutput(INFO, format, a...)
}

// Debugf logs a message using DEBUG as log level.
func Debugf(format string, a ...interface{}) {
	loggerOutput(DEBUG, format, a...)
}

// Tracef logs a message using TRACE as log level.
func Tracef(format string, a ...interface{}) {
	loggerOutput(TRACE, format, a...)
}

// ************ globalLogger logger methods ************

type globalLogger struct{}

var (
	logger                            = new(globalLogger)
	_      Logger                     = logger
	_      graceful.LoggerWithFlusher = logger
)

func (globalLogger) Flush() error {
	return FlushLogger()
}

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
func (globalLogger) Printf(format string, a ...interface{}) {
	loggerOutput(PRINT, format, a...)
}

// Fatalf is equivalent to l.Criticalf followed by a call to os.Exit(1).
func (globalLogger) Fatalf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
	loggerOutputter.Flush()
	os.Exit(1)
}

// Panicf is equivalent to l.Criticalf followed by a call to panic().
func (globalLogger) Panicf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
	loggerOutputter.Flush()
	panic(fmt.Sprintf(format, a...))
}

// Criticalf logs a message using CRITICAL as log level.
func (globalLogger) Criticalf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
}

// Errorf logs a message using ERROR as log level.
func (globalLogger) Errorf(format string, a ...interface{}) {
	loggerOutput(ERROR, format, a...)
}

// Warnf logs a message using WARNING as log level.
func (globalLogger) Warnf(format string, a ...interface{}) {
	loggerOutput(WARNING, format, a...)
}

// Noticef logs a message using NOTICE as log level.
func (globalLogger) Noticef(format string, a ...interface{}) {
	loggerOutput(NOTICE, format, a...)
}

// Infof logs a message using INFO as log level.
func (globalLogger) Infof(format string, a ...interface{}) {
	loggerOutput(INFO, format, a...)
}

// Debugf logs a message using DEBUG as log level.
func (globalLogger) Debugf(format string, a ...interface{}) {
	loggerOutput(DEBUG, format, a...)
}

// Tracef logs a message using TRACE as log level.
func (globalLogger) Tracef(format string, a ...interface{}) {
	loggerOutput(TRACE, format, a...)
}

// ************ *session logger methods ************

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
func (s *session) Printf(format string, a ...interface{}) {
	loggerOutput(PRINT, format, a...)
}

// Fatalf is equivalent to l.Criticalf followed by a call to os.Exit(1).
func (s *session) Fatalf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
	loggerOutputter.Flush()
	os.Exit(1)
}

// Panicf is equivalent to l.Criticalf followed by a call to panic().
func (s *session) Panicf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
	loggerOutputter.Flush()
	panic(fmt.Sprintf(format, a...))
}

// Criticalf logs a message using CRITICAL as log level.
func (s *session) Criticalf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
}

// Errorf logs a message using ERROR as log level.
func (s *session) Errorf(format string, a ...interface{}) {
	loggerOutput(ERROR, format, a...)
}

// Warnf logs a message using WARNING as log level.
func (s *session) Warnf(format string, a ...interface{}) {
	loggerOutput(WARNING, format, a...)
}

// Noticef logs a message using NOTICE as log level.
func (s *session) Noticef(format string, a ...interface{}) {
	loggerOutput(NOTICE, format, a...)
}

// Infof logs a message using INFO as log level.
func (s *session) Infof(format string, a ...interface{}) {
	loggerOutput(INFO, format, a...)
}

// Debugf logs a message using DEBUG as log level.
func (s *session) Debugf(format string, a ...interface{}) {
	loggerOutput(DEBUG, format, a...)
}

// Tracef logs a message using TRACE as log level.
func (s *session) Tracef(format string, a ...interface{}) {
	loggerOutput(TRACE, format, a...)
}

// ************ *handlerCtx Pure Logger Methods ************

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
func (c *handlerCtx) Printf(format string, a ...interface{}) {
	loggerOutput(PRINT, format, a...)
}

// Fatalf is equivalent to l.Criticalf followed by a call to os.Exit(1).
func (c *handlerCtx) Fatalf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
	loggerOutputter.Flush()
	os.Exit(1)
}

// Panicf is equivalent to l.Criticalf followed by a call to panic().
func (c *handlerCtx) Panicf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
	loggerOutputter.Flush()
	panic(fmt.Sprintf(format, a...))
}

// Criticalf logs a message using CRITICAL as log level.
func (c *handlerCtx) Criticalf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
}

// Errorf logs a message using ERROR as log level.
func (c *handlerCtx) Errorf(format string, a ...interface{}) {
	loggerOutput(ERROR, format, a...)
}

// Warnf logs a message using WARNING as log level.
func (c *handlerCtx) Warnf(format string, a ...interface{}) {
	loggerOutput(WARNING, format, a...)
}

// Noticef logs a message using NOTICE as log level.
func (c *handlerCtx) Noticef(format string, a ...interface{}) {
	loggerOutput(NOTICE, format, a...)
}

// Infof logs a message using INFO as log level.
func (c *handlerCtx) Infof(format string, a ...interface{}) {
	loggerOutput(INFO, format, a...)
}

// Debugf logs a message using DEBUG as log level.
func (c *handlerCtx) Debugf(format string, a ...interface{}) {
	loggerOutput(DEBUG, format, a...)
}

// Tracef logs a message using TRACE as log level.
func (c *handlerCtx) Tracef(format string, a ...interface{}) {
	loggerOutput(TRACE, format, a...)
}

// ************ *callCmd Pure Logger Methods ************

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
func (c *callCmd) Printf(format string, a ...interface{}) {
	loggerOutput(PRINT, format, a...)
}

// Fatalf is equivalent to l.Criticalf followed by a call to os.Exit(1).
func (c *callCmd) Fatalf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
	loggerOutputter.Flush()
	os.Exit(1)
}

// Panicf is equivalent to l.Criticalf followed by a call to panic().
func (c *callCmd) Panicf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
	loggerOutputter.Flush()
	panic(fmt.Sprintf(format, a...))
}

// Criticalf logs a message using CRITICAL as log level.
func (c *callCmd) Criticalf(format string, a ...interface{}) {
	loggerOutput(CRITICAL, format, a...)
}

// Errorf logs a message using ERROR as log level.
func (c *callCmd) Errorf(format string, a ...interface{}) {
	loggerOutput(ERROR, format, a...)
}

// Warnf logs a message using WARNING as log level.
func (c *callCmd) Warnf(format string, a ...interface{}) {
	loggerOutput(WARNING, format, a...)
}

// Noticef logs a message using NOTICE as log level.
func (c *callCmd) Noticef(format string, a ...interface{}) {
	loggerOutput(NOTICE, format, a...)
}

// Infof logs a message using INFO as log level.
func (c *callCmd) Infof(format string, a ...interface{}) {
	loggerOutput(INFO, format, a...)
}

// Debugf logs a message using DEBUG as log level.
func (c *callCmd) Debugf(format string, a ...interface{}) {
	loggerOutput(DEBUG, format, a...)
}

// Tracef logs a message using TRACE as log level.
func (c *callCmd) Tracef(format string, a ...interface{}) {
	loggerOutput(TRACE, format, a...)
}
