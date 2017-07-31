package codec

import (
	"encoding/json"
	"io"
)

func init() {
	RegEncodeMaker("json", NewJsonEncoder)
	RegDecodeMaker("json", NewJsonDecoder)
}

// NewJsonEncoder returns a new json encoder that writes to w.
func NewJsonEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

// NewJsonDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may
// read data from r beyond the JSON values requested.
func NewJsonDecoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}
