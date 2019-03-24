// errors is improved errors package.
package errors

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/henrylee2cn/goutil"
)

// New returns an error that formats as the given text.
func New(text string) error {
	return &myerror{text}
}

// myerror is a trivial implementation of error.
type myerror struct {
	s string
}

func (e *myerror) Error() string {
	return e.s
}

// Errorf formats according to a format specifier and returns the string
// as a value that satisfies error.
func Errorf(format string, a ...interface{}) error {
	return New(fmt.Sprintf(format, a...))
}

// Merge merges multiple errors.
func Merge(errs ...error) error {
	return Append(nil, errs...)
}

// Append appends multiple errors to the error.
func Append(err error, errs ...error) error {
	count := len(errs)
	if count == 0 {
		return err
	}
	var merged []error
	if err != nil {
		if e, ok := err.(*multiError); ok {
			_count := len(e.errs)
			merged = make([]error, _count, count+_count)
			copy(merged, e.errs)
		} else {
			merged = make([]error, 1, count+1)
			merged[0] = err
		}
	}
	for _, err := range errs {
		switch e := err.(type) {
		case nil:
			continue
		case *multiError:
			merged = append(merged, e.errs...)
		default:
			merged = append(merged, e)
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return &multiError{
		errs: merged,
	}
}

// multiError merged multiple errors
type multiError struct {
	errs []error
	text string
}

// mergePrefix the multiple errors prefix
var mergePrefix = []byte("MultiError:\n")

// Error implement error interface.
func (m *multiError) Error() string {
	if len(m.text) > 0 {
		return m.text
	}
	var bText = make([]byte, len(mergePrefix), 56)
	copy(bText, mergePrefix)
	for i, err := range m.errs {
		bText = append(bText, strconv.Itoa(i+1)...)
		bText = append(bText, ". "...)
		bText = append(bText, bytes.Trim(goutil.StringToBytes(err.Error()), "\n")...)
		bText = append(bText, '\n')
	}
	m.text = goutil.BytesToString(bText)
	return m.text
}
