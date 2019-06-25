package goutil

import (
	"bytes"
	"strconv"
	"strings"
	"unicode/utf8"
)

// SnakeString converts the accepted string to a snake string (XxYy to xx_yy)
func SnakeString(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	for _, d := range StringToBytes(s) {
		if d >= 'A' && d <= 'Z' {
			if j {
				data = append(data, '_')
				j = false
			}
		} else if d != '_' {
			j = true
		}
		data = append(data, d)
	}
	return strings.ToLower(BytesToString(data))
}

// CamelString converts the accepted string to a camel string (xx_yy to XxYy)
func CamelString(s string) string {
	data := make([]byte, 0, len(s))
	j := false
	k := false
	num := len(s) - 1
	for i := 0; i <= num; i++ {
		d := s[i]
		if k == false && d >= 'A' && d <= 'Z' {
			k = true
		}
		if d >= 'a' && d <= 'z' && (j || k == false) {
			d = d - 32
			j = false
			k = true
		}
		if k && d == '_' && num > i && s[i+1] >= 'a' && s[i+1] <= 'z' {
			j = true
			continue
		}
		data = append(data, d)
	}
	return string(data[:])
}

var spaceReplacer = strings.NewReplacer(
	"  ", " ",
	"\n\n", "\n",
	"\r\r", "\r",
	"\t\t", "\t",
	"\r\n\r\n", "\r\n",
	" \n", "\n",
	"\t\n", "\n",
	" \t", "\t",
	"\t ", "\t",
	"\v\v", "\v",
	"\f\f", "\f",
	string(0x85)+string(0x85),
	string(0x85),
	string(0xA0)+string(0xA0),
	string(0xA0),
)

// SpaceInOne combines multiple consecutive space characters into one.
func SpaceInOne(s string) string {
	var old string
	for old != s {
		old = s
		s = spaceReplacer.Replace(s)
	}
	return s
}

// StringMarshalJSON converts the string to JSON byte stream.
func StringMarshalJSON(s string, escapeHTML bool) []byte {
	a := StringToBytes(s)
	var buf = bytes.NewBuffer(make([]byte, 0, 64))
	buf.WriteByte('"')
	start := 0
	for i := 0; i < len(a); {
		if b := a[i]; b < utf8.RuneSelf {
			if htmlSafeSet[b] || (!escapeHTML && safeSet[b]) {
				i++
				continue
			}
			if start < i {
				buf.Write(a[start:i])
			}
			switch b {
			case '\\', '"':
				buf.WriteByte('\\')
				buf.WriteByte(b)
			case '\n':
				buf.WriteByte('\\')
				buf.WriteByte('n')
			case '\r':
				buf.WriteByte('\\')
				buf.WriteByte('r')
			case '\t':
				buf.WriteByte('\\')
				buf.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				buf.WriteString(`\u00`)
				buf.WriteByte(hexSet[b>>4])
				buf.WriteByte(hexSet[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRune(a[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				buf.Write(a[start:i])
			}
			buf.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				buf.Write(a[start:i])
			}
			buf.WriteString(`\u202`)
			buf.WriteByte(hexSet[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(a) {
		buf.Write(a[start:])
	}
	buf.WriteByte('"')
	return buf.Bytes()
}

var hexSet = "0123456789abcdef"

// safeSet holds the value true if the ASCII character with the given array
// position can be represented inside a JSON string without any further
// escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), and the backslash character ("\").
var safeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}

// htmlSafeSet holds the value true if the ASCII character with the given
// array position can be safely represented inside a JSON string, embedded
// inside of HTML <script> tags, without any additional escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), the backslash character ("\"), HTML opening and closing
// tags ("<" and ">"), and the ampersand ("&").
var htmlSafeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      false,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      false,
	'=':      true,
	'>':      false,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}

// StringsToBools converts string slice to bool slice.
func StringsToBools(a []string) ([]bool, error) {
	r := make([]bool, len(a))
	for k, v := range a {
		i, err := strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}
		r[k] = i
	}
	return r, nil
}

// StringsToFloat32s converts string slice to float32 slice.
func StringsToFloat32s(a []string) ([]float32, error) {
	r := make([]float32, len(a))
	for k, v := range a {
		i, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, err
		}
		r[k] = float32(i)
	}
	return r, nil
}

// StringsToFloat64s converts string slice to float64 slice.
func StringsToFloat64s(a []string) ([]float64, error) {
	r := make([]float64, len(a))
	for k, v := range a {
		i, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		r[k] = i
	}
	return r, nil
}

// StringsToInts converts string slice to int slice.
func StringsToInts(a []string) ([]int, error) {
	r := make([]int, len(a))
	for k, v := range a {
		i, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		r[k] = i
	}
	return r, nil
}

// StringsToInt64s converts string slice to int64 slice.
func StringsToInt64s(a []string) ([]int64, error) {
	r := make([]int64, len(a))
	for k, v := range a {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		r[k] = i
	}
	return r, nil
}

// StringsToInt32s converts string slice to int32 slice.
func StringsToInt32s(a []string) ([]int32, error) {
	r := make([]int32, len(a))
	for k, v := range a {
		i, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, err
		}
		r[k] = int32(i)
	}
	return r, nil
}

// StringsToInt16s converts string slice to int16 slice.
func StringsToInt16s(a []string) ([]int16, error) {
	r := make([]int16, len(a))
	for k, v := range a {
		i, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return nil, err
		}
		r[k] = int16(i)
	}
	return r, nil
}

// StringsToInt8s converts string slice to int8 slice.
func StringsToInt8s(a []string) ([]int8, error) {
	r := make([]int8, len(a))
	for k, v := range a {
		i, err := strconv.ParseInt(v, 10, 8)
		if err != nil {
			return nil, err
		}
		r[k] = int8(i)
	}
	return r, nil
}

// StringsToUint8s converts string slice to uint8 slice.
func StringsToUint8s(a []string) ([]uint8, error) {
	r := make([]uint8, len(a))
	for k, v := range a {
		i, err := strconv.ParseUint(v, 10, 8)
		if err != nil {
			return nil, err
		}
		r[k] = uint8(i)
	}
	return r, nil
}

// StringsToUint16s converts string slice to uint16 slice.
func StringsToUint16s(a []string) ([]uint16, error) {
	r := make([]uint16, len(a))
	for k, v := range a {
		i, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return nil, err
		}
		r[k] = uint16(i)
	}
	return r, nil
}

// StringsToUint32s converts string slice to uint32 slice.
func StringsToUint32s(a []string) ([]uint32, error) {
	r := make([]uint32, len(a))
	for k, v := range a {
		i, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return nil, err
		}
		r[k] = uint32(i)
	}
	return r, nil
}

// StringsToUint64s converts string slice to uint64 slice.
func StringsToUint64s(a []string) ([]uint64, error) {
	r := make([]uint64, len(a))
	for k, v := range a {
		i, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, err
		}
		r[k] = uint64(i)
	}
	return r, nil
}

// StringsToUints converts string slice to uint slice.
func StringsToUints(a []string) ([]uint, error) {
	r := make([]uint, len(a))
	for k, v := range a {
		i, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, err
		}
		r[k] = uint(i)
	}
	return r, nil
}
