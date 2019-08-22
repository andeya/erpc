package tp

import (
	"github.com/henrylee2cn/goutil/status"
)

// Status a handling status with code, msg, cause and stack.
type Status = status.Status

var (
	// NewStatus creates a message status with code, msg and cause.
	// NOTE:
	//  code=0 means no error
	// TYPE:
	//  func NewStatus(code int32, msg string, cause interface{}) *Status
	NewStatus = status.New

	// NewStatusWithStack creates a message status with code, msg and cause and stack.
	// NOTE:
	//  code=0 means no error
	// TYPE:
	//  func NewStatusWithStack(code int32, msg string, cause interface{}) *Status
	NewStatusWithStack = status.NewWithStack

	// NewStatusFromQuery parses the query bytes to a status object.
	// TYPE:
	// func NewStatusFromQuery(b []byte, tagStack bool) *Status
	NewStatusFromQuery = status.FromQuery
)

// NewStatusByCodeText creates a message status with code, msg, cause or stack.
// NOTE:
//  The msg comes from the CodeText(code) value.
func NewStatusByCodeText(code int32, cause interface{}, tagStack bool) *Status {
	stat := NewStatus(code, CodeText(code), cause)
	if tagStack {
		stat.TagStack(1)
	}
	return stat
}

// Internal Framework Status code.
// NOTE: Recommended custom code is greater than 1000.
//  unknown error code: -1.
//  sender peer error code range: [100,199].
//  message handling error code range: [400,499].
//  receiver peer error code range: [500,599].
const (
	CodeUnknownError        int32 = -1
	CodeOK                  int32 = 0      // nil error (ok)
	CodeNoError             int32 = CodeOK // nil error (ok)
	CodeInvalidOp           int32 = 1
	CodeWrongConn           int32 = 100
	CodeConnClosed          int32 = 102
	CodeWriteFailed         int32 = 104
	CodeDialFailed          int32 = 105
	CodeBadMessage          int32 = 400
	CodeUnauthorized        int32 = 401
	CodeNotFound            int32 = 404
	CodeMtypeNotAllowed     int32 = 405
	CodeHandleTimeout       int32 = 408
	CodeInternalServerError int32 = 500
	CodeBadGateway          int32 = 502

	// CodeConflict                      int32 = 409
	// CodeUnsupportedTx                 int32 = 410
	// CodeUnsupportedCodecType          int32 = 415
	// CodeServiceUnavailable            int32 = 503
	// CodeGatewayTimeout                int32 = 504
	// CodeVariantAlsoNegotiates         int32 = 506
	// CodeInsufficientStorage           int32 = 507
	// CodeLoopDetected                  int32 = 508
	// CodeNotExtended                   int32 = 510
	// CodeNetworkAuthenticationRequired int32 = 511
)

// CodeText returns the reply error code text.
// If the type is undefined returns 'Unknown Error'.
func CodeText(statCode int32) string {
	switch statCode {
	case CodeNoError:
		return ""
	case CodeInvalidOp:
		return "Invalid Operation"
	case CodeBadMessage:
		return "Bad Message"
	case CodeUnauthorized:
		return "Unauthorized"
	case CodeDialFailed:
		return "Dial Failed"
	case CodeWrongConn:
		return "Wrong Connection"
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

// Internal Framework Status string.
var (
	statInvalidOpError      = NewStatus(CodeInvalidOp, CodeText(CodeInvalidOp), "")
	statUnknownError        = NewStatus(CodeUnknownError, CodeText(CodeUnknownError), "")
	statDialFailed          = NewStatus(CodeDialFailed, CodeText(CodeDialFailed), "")
	statConnClosed          = NewStatus(CodeConnClosed, CodeText(CodeConnClosed), "")
	statWriteFailed         = NewStatus(CodeWriteFailed, CodeText(CodeWriteFailed), "")
	statBadMessage          = NewStatus(CodeBadMessage, CodeText(CodeBadMessage), "")
	statNotFound            = NewStatus(CodeNotFound, CodeText(CodeNotFound), "")
	statCodeMtypeNotAllowed = NewStatus(CodeMtypeNotAllowed, CodeText(CodeMtypeNotAllowed), "")
	statHandleTimeout       = NewStatus(CodeHandleTimeout, CodeText(CodeHandleTimeout), "")
	statInternalServerError = NewStatus(CodeInternalServerError, CodeText(CodeInternalServerError), "")
)

// IsConnError determines whether the status is a connection error.
func IsConnError(stat *Status) bool {
	if stat == nil {
		return false
	}
	code := stat.Code()
	if code == CodeDialFailed || code == CodeConnClosed {
		return true
	}
	return false
}
