// Copyright 2018 henrylee2cn Author. All Rights Reserved.
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

package httpbody

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"strings"
)

// NewFormBody returns form request content type and body reader.
// NOTE:
//  @values format: <fieldName,[value]>
//  @files format: <fieldName,[fileName]>
func NewFormBody(values, files url.Values) (contentType string, bodyReader io.Reader, err error) {
	if len(files) == 0 {
		return "application/x-www-form-urlencoded", strings.NewReader(values.Encode()), nil
	}
	var rw = bytes.NewBuffer(make([]byte, 32*1024*len(files)))
	var bodyWriter = multipart.NewWriter(rw)
	var buf = make([]byte, 32*1024)
	var fileWriter io.Writer
	var f *os.File
	for fieldName, postfiles := range files {
		for _, fileName := range postfiles {
			fileWriter, err = bodyWriter.CreateFormFile(fieldName, fileName)
			if err != nil {
				return
			}
			f, err = os.Open(fileName)
			if err != nil {
				return
			}
			_, err = io.CopyBuffer(fileWriter, f, buf)
			f.Close()
			if err != nil {
				return
			}
		}
	}
	for k, v := range values {
		for _, vv := range v {
			bodyWriter.WriteField(k, vv)
		}
	}
	bodyWriter.Close()
	return bodyWriter.FormDataContentType(), rw, nil
}

type (
	// Files maps a string key to a list of files.
	Files map[string][]File
	// File interface for form.
	File interface {
		// Name returns the name of the file as presented to Open.
		Name() string
		// Read reads up to len(b) bytes from the File.
		// It returns the number of bytes read and any error encountered.
		// At end of file, Read returns 0, io.EOF.
		Read(p []byte) (n int, err error)
	}
)

// NewFormBody2 returns form request content type and body reader.
// NOTE:
//  @values format: <fieldName,[value]>
//  @files format: <fieldName,[File]>
func NewFormBody2(values url.Values, files Files) (contentType string, bodyReader io.Reader) {
	if len(files) == 0 {
		return "application/x-www-form-urlencoded", strings.NewReader(values.Encode())
	}
	var pr, pw = io.Pipe()
	var bodyWriter = multipart.NewWriter(pw)
	var fileWriter io.Writer
	var buf = make([]byte, 32*1024)
	go func() {
		for fieldName, postfiles := range files {
			for _, file := range postfiles {
				fileWriter, _ = bodyWriter.CreateFormFile(fieldName, file.Name())
				io.CopyBuffer(fileWriter, file, buf)
			}
		}
		for k, v := range values {
			for _, vv := range v {
				bodyWriter.WriteField(k, vv)
			}
		}
		bodyWriter.Close()
		pw.Close()
	}()
	return bodyWriter.FormDataContentType(), pr
}

// NewFile creates a file for HTTP form.
func NewFile(name string, bodyReader io.Reader) File {
	return &fileReader{name, bodyReader}
}

// fileReader file name and bytes.
type fileReader struct {
	name       string
	bodyReader io.Reader
}

func (f *fileReader) Name() string {
	return f.name
}

func (f *fileReader) Read(p []byte) (int, error) {
	return f.bodyReader.Read(p)
}

// NewJSONBody returns JSON request content type and body reader.
func NewJSONBody(v interface{}) (contentType string, bodyReader io.Reader, err error) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	return "application/json;charset=utf-8", bytes.NewReader(b), nil
}

// NewXMLBody returns XML request content type and body reader.
func NewXMLBody(v interface{}) (contentType string, bodyReader io.Reader, err error) {
	b, err := xml.Marshal(v)
	if err != nil {
		return
	}
	return "application/xml;charset=utf-8", bytes.NewReader(b), nil
}
