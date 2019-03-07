package tp

import (
	"encoding/json"
	"strconv"
	"unsafe"

	"github.com/tidwall/gjson"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/teleport/utils"
)

// Internal Framework Rerror code.
// NOTE: Recommended custom code is greater than 1000.
//  unknown error code: -1.
//  sender peer error code range: [100,199].
//  message handling error code range: [400,499].
//  receiver peer error code range: [500,599].
const (
	CodeUnknownError        = -1
	CodeNoError             = 0 // nil error
	CodeConnClosed          = 102
	CodeWriteFailed         = 104
	CodeDialFailed          = 105
	CodeBadMessage          = 400
	CodeUnauthorized        = 401
	CodeNotFound            = 404
	CodeMtypeNotAllowed     = 405
	CodeHandleTimeout       = 408
	CodeInternalServerError = 500
	CodeBadGateway          = 502

	// CodeConflict                      = 409
	// CodeUnsupportedTx                 = 410
	// CodeUnsupportedCodecType          = 415
	// CodeServiceUnavailable            = 503
	// CodeGatewayTimeout                = 504
	// CodeVariantAlsoNegotiates         = 506
	// CodeInsufficientStorage           = 507
	// CodeLoopDetected                  = 508
	// CodeNotExtended                   = 510
	// CodeNetworkAuthenticationRequired = 511
)

// CodeText returns the reply error code text.
// If the type is undefined returns 'Unknown Error'.
func CodeText(rerrCode int32) string {
	switch rerrCode {
	case CodeBadMessage:
		return "Bad Message"
	case CodeUnauthorized:
		return "Unauthorized"
	case CodeDialFailed:
		return "Dial Failed"
	case CodeConnClosed:
		return "Connection Closed"
	case CodeWriteFailed:
		return "Write Failed"
	case CodeNotFound:
		return "Not Found"
	case CodeHandleTimeout:
		return "Handle Timeout"
	case CodeMtypeNotAllowed:
		return "Message Type Not Allowed"
	case CodeInternalServerError:
		return "Internal Server Error"
	case CodeBadGateway:
		return "Bad Gateway"
	case CodeUnknownError:
		fallthrough
	default:
		return "Unknown Error"
	}
}

// Internal Framework Rerror string.
var (
	rerrUnknownError        = NewRerror(CodeUnknownError, CodeText(CodeUnknownError), "")
	rerrDialFailed          = NewRerror(CodeDialFailed, CodeText(CodeDialFailed), "")
	rerrConnClosed          = NewRerror(CodeConnClosed, CodeText(CodeConnClosed), "")
	rerrWriteFailed         = NewRerror(CodeWriteFailed, CodeText(CodeWriteFailed), "")
	rerrBadMessage          = NewRerror(CodeBadMessage, CodeText(CodeBadMessage), "")
	rerrNotFound            = NewRerror(CodeNotFound, CodeText(CodeNotFound), "")
	rerrCodeMtypeNotAllowed = NewRerror(CodeMtypeNotAllowed, CodeText(CodeMtypeNotAllowed), "")
	rerrHandleTimeout       = NewRerror(CodeHandleTimeout, CodeText(CodeHandleTimeout), "")
	rerrInternalServerError = NewRerror(CodeInternalServerError, CodeText(CodeInternalServerError), "")
)

// IsConnRerror determines whether the error is a connection error
func IsConnRerror(rerr *Rerror) bool {
	if rerr == nil {
		return false
	}
	if rerr.Code == CodeDialFailed || rerr.Code == CodeConnClosed {
		return true
	}
	return false
}

type (
	// Rerror error only for reply message
	Rerror struct {
		// Code error code
		Code int32
		// Message the error message displayed to the user (optional)
		Message string
		// Reason the cause of the error for debugging (optional)
		Reason string
	}
)

var (
	_ json.Marshaler   = new(Rerror)
	_ json.Unmarshaler = new(Rerror)

	reA = []byte(`{"code":`)
	reB = []byte(`,"message":`)
	reC = []byte(`,"reason":`)
)

// NewRerror creates a *Rerror.
func NewRerror(code int32, message, reason string) *Rerror {
	return &Rerror{
		Code:    code,
		Message: message,
		Reason:  reason,
	}
}

// NewRerrorFromMeta creates a *Rerror from 'X-Reply-Error' metadata.
// Return nil if there is no 'X-Reply-Error' in metadata.
func NewRerrorFromMeta(meta *utils.Args) *Rerror {
	b := meta.Peek(MetaRerror)
	if len(b) == 0 {
		return nil
	}
	r := new(Rerror)
	r.UnmarshalJSON(b)
	return r
}

// SetToMeta sets self to 'X-Reply-Error' metadata.
func (r *Rerror) SetToMeta(meta *utils.Args) {
	b, _ := r.MarshalJSON()
	if len(b) == 0 {
		return
	}
	meta.Set(MetaRerror, goutil.BytesToString(b))
}

// Copy returns the copy of Rerror
func (r Rerror) Copy() *Rerror {
	return &r
}

// SetMessage sets the error message displayed to the user.
func (r *Rerror) SetMessage(message string) *Rerror {
	r.Message = message
	return r
}

// SetReason sets the cause of the error for debugging.
func (r *Rerror) SetReason(reason string) *Rerror {
	r.Reason = reason
	return r
}

// String prints error info.
func (r *Rerror) String() string {
	if r == nil {
		return "<nil>"
	}
	b, _ := r.MarshalJSON()
	return goutil.BytesToString(b)
}

// MarshalJSON marshals Rerror into JSON, implements json.Marshaler interface.
func (r *Rerror) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte{}, nil
	}
	var b = append(reA, strconv.FormatInt(int64(r.Code), 10)...)
	if len(r.Message) > 0 {
		b = append(b, reB...)
		b = append(b, utils.ToJSONStr(goutil.StringToBytes(r.Message), false)...)
	}
	if len(r.Reason) > 0 {
		b = append(b, reC...)
		b = append(b, utils.ToJSONStr(goutil.StringToBytes(r.Reason), false)...)
	}
	b = append(b, '}')
	return b, nil
}

// UnmarshalJSON unmarshals a JSON description of self.
func (r *Rerror) UnmarshalJSON(b []byte) error {
	if r == nil {
		return nil
	}
	s := goutil.BytesToString(b)
	r.Code = int32(gjson.Get(s, "code").Int())
	r.Message = gjson.Get(s, "message").String()
	r.Reason = gjson.Get(s, "reason").String()
	return nil
}

func hasRerror(meta *utils.Args) bool {
	return meta.Has(MetaRerror)
}

func getRerrorBytes(meta *utils.Args) []byte {
	return meta.Peek(MetaRerror)
}

// ToError converts to error
func (r *Rerror) ToError() error {
	if r == nil || r.Code == CodeNoError {
		return nil
	}
	return (*rerror)(unsafe.Pointer(r))
}

// ToRerror converts error to *Rerror
func ToRerror(err error) *Rerror {
	if err == nil {
		return nil
	}
	r, ok := err.(*rerror)
	if ok {
		return r.toRerror()
	}
	rerr := rerrUnknownError.Copy().SetReason(err.Error())
	return rerr
}

type rerror Rerror

func (r *rerror) Error() string {
	b, _ := r.toRerror().MarshalJSON()
	return goutil.BytesToString(b)
}

func (r *rerror) toRerror() *Rerror {
	return (*Rerror)(unsafe.Pointer(r))
}
