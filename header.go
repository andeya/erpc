package teleport

type Header struct {
	ID    string `json:"id"`
	URI   string `json:"uri"`
	Codec string `json:"codec,omitempty"`
	Gzip  int8   `json:"gzip,omitempty"` // gzip compression level(range [-2,9])
	Err   string `json:"err,omitempty"`  // only for response
}

// var (
// 	Magic = [5]byte{'h', 'e', 'n', 'r', 'y'}
// )
