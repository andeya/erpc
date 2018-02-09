// Copyright 2015-2017 HenryLee. All Rights Reserved.
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

package tp

import (
	"github.com/henrylee2cn/teleport/socket"
)

// Packet types
const (
	TypeUndefined byte = 0
	TypePull      byte = 1
	TypeReply     byte = 2 // reply to pull
	TypePush      byte = 3
)

// TypeText returns the packet type text.
// If the type is undefined returns 'Undefined'.
func TypeText(typ byte) string {
	switch typ {
	case TypePull:
		return "PULL"
	case TypeReply:
		return "REPLY"
	case TypePush:
		return "PUSH"
	default:
		return "Undefined"
	}
}

// Internal Framework Rerror code.
// Note: Recommended custom code is greater than 1000.
const (
	CodeUnknownError    = -1
	CodeDialFailed      = 105
	CodeConnClosed      = 102
	CodeWriteFailed     = 104
	CodeBadPacket       = 400
	CodeNotFound        = 404
	CodePtypeNotAllowed = 405
	CodeHandleTimeout   = 408

	// CodeConflict                      = 409
	// CodeUnsupportedTx                 = 410
	// CodeUnsupportedCodecType          = 415
	// CodeUnauthorized                  = 401
	// CodeInternalServerError           = 500
	// CodeBadGateway                    = 502
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
	case CodeDialFailed:
		return "Dial Failed"
	case CodeConnClosed:
		return "Connection Closed"
	case CodeWriteFailed:
		return "Write Failed"
	case CodeBadPacket:
		return "Bad Packet"
	case CodeNotFound:
		return "Not Found"
	case CodeHandleTimeout:
		return "Handle Timeout"
	case CodePtypeNotAllowed:
		return "Packet Type Not Allowed"
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
	rerrBadPacket           = NewRerror(CodeBadPacket, CodeText(CodeBadPacket), "")
	rerrNotFound            = NewRerror(CodeNotFound, CodeText(CodeNotFound), "")
	rerrCodePtypeNotAllowed = NewRerror(CodePtypeNotAllowed, CodeText(CodePtypeNotAllowed), "")
	rerrHandleTimeout       = NewRerror(CodeHandleTimeout, CodeText(CodeHandleTimeout), "")
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

const (
	// MetaRerrorKey reply error metadata key
	MetaRerrorKey = "X-Reply-Error"
	// MetaRealId real ID metadata key
	MetaRealId = "X-Real-ID"
	// MetaRealIp real IP metadata key
	MetaRealIp = "X-Real-IP"
)

// WithRealIdMeta sets the real ID to metadata.
func WithRealId(id string) socket.PacketSetting {
	return socket.WithAddMeta(MetaRealId, id)
}

// WithRealIpMeta sets the real IP to metadata.
func WithRealIp(ip string) socket.PacketSetting {
	return socket.WithAddMeta(MetaRealIp, ip)
}
