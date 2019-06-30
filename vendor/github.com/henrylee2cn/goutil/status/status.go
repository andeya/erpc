// Package status is a handling status with code, msg, cause and stack.
package status

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/henrylee2cn/goutil"
)

const (
	// OK status
	OK int32 = 0

	// UnknownError status
	UnknownError int32 = -1
)

// Status a handling status with code, msg, cause and stack.
type Status struct {
	code  int32
	msg   string
	cause error
	*stack
}

// New creates a handling status with code, msg and cause.
// NOTE:
//  code=0 means no error
func New(code int32, msg string, cause interface{}) *Status {
	s := &Status{
		code:  code,
		msg:   msg,
		cause: toErr(cause),
	}
	return s
}

// NewWithStack creates a handling status with code, msg and cause and stack.
// NOTE:
//  code=0 means no error
func NewWithStack(code int32, msg string, cause interface{}) *Status {
	return New(code, msg, cause).TagStack(1)
}

// FromJSON parses the JSON bytes to a status object.
func FromJSON(b []byte, tagStack bool) (*Status, error) {
	s := new(Status)
	err := s.UnmarshalJSON(b)
	if err != nil {
		return nil, err
	}
	if tagStack {
		s.stack = callers(3)
	}
	return s, nil
}

// FromQuery parses the query bytes to a status object.
func FromQuery(b []byte, tagStack bool) *Status {
	s := new(Status)
	s.DecodeQuery(b)
	if tagStack {
		s.stack = callers(3)
	}
	return s
}

// TagStack marks stack information.
func (s *Status) TagStack(skip ...int) *Status {
	depth := 3
	if len(skip) > 0 {
		depth += skip[0]
	}
	s.stack = callers(depth)
	return s
}

// Copy returns the copy of Status.
func (s *Status) Copy(newCause interface{}, newStackSkip ...int) *Status {
	if s == nil {
		return nil
	}
	if newCause == nil {
		newCause = s.cause
	}
	copy := New(s.code, s.msg, newCause)
	if len(newStackSkip) != 0 {
		copy.stack = callers(3 + newStackSkip[0])
	}
	return copy
}

// SetCode sets a new code to the status object.
func (s *Status) SetCode(code int32) *Status {
	if s != nil {
		s.code = code
	}
	return s
}

// SetMsg sets a new msg to the status object.
func (s *Status) SetMsg(msg string) *Status {
	if s != nil {
		s.msg = msg
	}
	return s
}

// SetCause sets a new cause to the status object.
func (s *Status) SetCause(cause interface{}) *Status {
	if s != nil {
		s.cause = toErr(cause)
	}
	return s
}

// Clear clears the status.
func (s *Status) Clear() {
	*s = Status{}
}

// Code returns the status code.
func (s *Status) Code() int32 {
	if s == nil {
		return OK
	}
	return s.code
}

// Msg returns the status msg displayed to the user (optional).
func (s *Status) Msg() string {
	if s == nil {
		return ""
	}
	return s.msg
}

var noCause = errors.New("")

// Cause returns the cause of the status for debugging (optional).
// NOTE:
//  If s.OK() is false, the return value is never nil.
func (s *Status) Cause() error {
	if s == nil {
		return nil
	}
	if s.cause == nil && s.code != OK {
		return noCause
	}
	return s.cause
}

// OK returns whether is OK status (code=0).
func (s *Status) OK() bool {
	return s.Code() == OK
}

// UnknownError returns whether is UnknownError status (code=-1).
func (s *Status) UnknownError() bool {
	return s.Code() == UnknownError
}

// StackTrace returns stack trace.
func (s *Status) StackTrace() StackTrace {
	if s == nil || s.stack == nil {
		return nil
	}
	return s.stack.StackTrace()
}

// QueryString returns query string for the status object.
func (s *Status) QueryString() string {
	return goutil.BytesToString(s.EncodeQuery())
}

// JSONString returns JSON string for the status object.
func (s *Status) JSONString() string {
	if s == nil {
		return "{}"
	}
	b, _ := s.MarshalJSON()
	return goutil.BytesToString(b)
}

// String prints status info.
func (s *Status) String() string {
	if s == nil {
		return "<nil>"
	}
	b, _ := s.MarshalJSON()
	return goutil.BytesToString(b)
}

// Format formats the status object according to the fmt.Formatter interface.
//
//    %s	lists source files for each Frame in the stack
//    %v	lists the source file and line number for each Frame in the stack
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//    %+v   Prints filename, function, and line number for each Frame in the stack.
func (s *Status) Format(state fmt.State, verb rune) {
	switch verb {
	case 'v':
		if state.Flag('+') {
			fmt.Fprintf(state, "%+v", s.String())
			if s.stack != nil {
				s.stack.Format(state, verb)
			}
			return
		}
		fallthrough
	case 's':
		io.WriteString(state, s.String())
	case 'q':
		fmt.Fprintf(state, "%q", s.String())
	}
}

type exportStatus struct {
	Code  int32  `json:"code"`
	Msg   string `json:"msg"`
	Cause string `json:"cause"`
}

var (
	reA  = []byte(`{"code":`)
	reB  = []byte(`,"msg":`)
	reC  = []byte(`,"cause":`)
	null = []byte("null")
)
var (
	_ json.Marshaler   = new(Status)
	_ json.Unmarshaler = new(Status)
)

// MarshalJSON marshals the status object into JSON, implements json.Marshaler interface.
func (s *Status) MarshalJSON() ([]byte, error) {
	if s == nil {
		return null, nil
	}
	b := append(reA, strconv.FormatInt(int64(s.code), 10)...)

	b = append(b, reB...)
	b = append(b, goutil.StringMarshalJSON(s.msg, false)...)

	var cause string
	if s.cause != nil {
		cause = s.cause.Error()
	}
	b = append(b, reC...)
	b = append(b, goutil.StringMarshalJSON(cause, false)...)

	b = append(b, '}')
	return b, nil
}

// UnmarshalJSON unmarshals a JSON description of self.
func (s *Status) UnmarshalJSON(b []byte) error {
	if s == nil {
		return nil
	}
	if len(b) == 0 {
		s.Clear()
		return nil
	}
	var v exportStatus
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	s.code = v.Code
	s.msg = v.Msg
	if v.Cause != "" {
		s.cause = errors.New(v.Cause)
	} else {
		s.cause = nil
	}
	return nil
}

var (
	keyCode  = []byte("code")
	keyMsg   = []byte("msg")
	keyCause = []byte("cause")
)

// EncodeQuery encodes the status object into query bytes.
func (s *Status) EncodeQuery() []byte {
	if s == nil {
		return nil
	}
	b := make([]byte, 0, 384)
	b = append(b, keyCode...)
	b = append(b, '=')
	b = append(b, strconv.FormatInt(int64(s.code), 10)...)
	if len(s.msg) > 0 {
		b = append(b, '&')
		b = append(b, keyMsg...)
		b = append(b, '=')
		b = appendQuotedArg(b, goutil.StringToBytes(s.msg))
	}
	if s.cause != nil {
		b = append(b, '&')
		b = append(b, keyCause...)
		b = append(b, '=')
		b = appendQuotedArg(b, goutil.StringToBytes(s.cause.Error()))
	}
	return b
}

// DecodeQuery parses the given b containing query args to the status object.
func (s *Status) DecodeQuery(b []byte) {
	if s == nil {
		return
	}
	s.Clear()
	if len(b) == 0 {
		return
	}
	var hadCode, hadMsg, hadCause bool
	kv := &argsKV{
		key:   make([]byte, 0, 8),
		value: make([]byte, 0, 128),
	}
	a := argsScanner{b: b}
	for a.next(kv) {
		if !hadCode && bytes.Equal(keyCode, kv.key) {
			i, _ := strconv.ParseInt(goutil.BytesToString(kv.value), 10, 32)
			s.code = int32(i)
			if hadMsg && hadCause {
				return
			}
			hadCode = true
		} else if !hadMsg && bytes.Equal(keyMsg, kv.key) {
			if hadCode && hadCause {
				s.msg = goutil.BytesToString(kv.value)
				return
			}
			s.msg = string(kv.value)
			hadMsg = true
		} else if !hadCause && bytes.Equal(keyCause, kv.key) {
			if hadCode && hadMsg {
				s.cause = errors.New(goutil.BytesToString(kv.value))
				return
			}
			s.cause = errors.New(string(kv.value))
			hadCause = true
		}
	}
}

func toErr(cause interface{}) error {
	switch v := cause.(type) {
	case nil:
		return nil
	case error:
		return v
	case string:
		return errors.New(v)
	case *Status:
		return v.cause
	case Status:
		return v.cause
	default:
		return fmt.Errorf("%v", v)
	}
}
