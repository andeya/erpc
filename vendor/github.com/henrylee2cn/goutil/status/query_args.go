package status

type (
	argsScanner struct {
		b []byte
	}
	argsKV struct {
		key   []byte
		value []byte
	}
)

func (s *argsScanner) next(kv *argsKV) bool {
	if len(s.b) == 0 {
		return false
	}

	isKey := true
	k := 0
	for i, c := range s.b {
		switch c {
		case '=':
			if isKey {
				isKey = false
				kv.key = decodeArg(kv.key, s.b[:i], true)
				k = i + 1
			}
		case '&':
			if isKey {
				kv.key = decodeArg(kv.key, s.b[:i], true)
				kv.value = kv.value[:0]
			} else {
				kv.value = decodeArg(kv.value, s.b[k:i], true)
			}
			s.b = s.b[i+1:]
			return true
		}
	}

	if isKey {
		kv.key = decodeArg(kv.key, s.b, true)
		kv.value = kv.value[:0]
	} else {
		kv.value = decodeArg(kv.value, s.b[k:], true)
	}
	s.b = s.b[len(s.b):]
	return true
}

func decodeArg(dst, src []byte, decodePlus bool) []byte {
	return decodeArgAppend(dst[:0], src, decodePlus)
}

func decodeArgAppend(dst, src []byte, decodePlus bool) []byte {
	for i, n := 0, len(src); i < n; i++ {
		c := src[i]
		if c == '%' {
			if i+2 >= n {
				return append(dst, src[i:]...)
			}
			x1 := hexbyte2int(src[i+1])
			x2 := hexbyte2int(src[i+2])
			if x1 < 0 || x2 < 0 {
				dst = append(dst, c)
			} else {
				dst = append(dst, byte(x1<<4|x2))
				i += 2
			}
		} else if decodePlus && c == '+' {
			dst = append(dst, ' ')
		} else {
			dst = append(dst, c)
		}
	}
	return dst
}

// appendQuotedArg appends url-encoded src to dst and returns appended dst.
func appendQuotedArg(dst, src []byte) []byte {
	for _, c := range src {
		// See http://www.w3.org/TR/html5/forms.html#form-submission-algorithm
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			c == '*' || c == '-' || c == '.' || c == '_' {
			dst = append(dst, c)
		} else {
			dst = append(dst, '%', hexCharUpper(c>>4), hexCharUpper(c&15))
		}
	}
	return dst
}

var hex2intTable = func() []byte {
	b := make([]byte, 255)
	for i := byte(0); i < 255; i++ {
		c := byte(0)
		if i >= '0' && i <= '9' {
			c = 1 + i - '0'
		} else if i >= 'a' && i <= 'f' {
			c = 1 + i - 'a' + 10
		} else if i >= 'A' && i <= 'F' {
			c = 1 + i - 'A' + 10
		}
		b[i] = c
	}
	return b
}()

func hexbyte2int(c byte) int {
	return int(hex2intTable[c]) - 1
}

func hexCharUpper(c byte) byte {
	if c < 10 {
		return '0' + c
	}
	return c - 10 + 'A'
}
