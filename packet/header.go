package packet

type Header struct {
	ID    string `json:"id"`
	URI   string `json:"uri"`
	Codec string `json:"codec,omitempty"`
	Gzip  int8   `json:"gzip,omitempty"` // gzip compression level(range [-2,9])
	Err   string `json:"err,omitempty"`  // only for response
	Len   uint32 `json:"len"`
}
