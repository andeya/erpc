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
	"bytes"
	"unsafe"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/teleport/utils"
)

// Packet types
const (
	TypeUndefined byte = 0
	TypePull      byte = 1
	TypeReply     byte = 2 // reply to pull
	TypePush      byte = 3
	// TypeAuth      byte = 4
	// TypeHeartbeat byte = 5
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
	CodeDialFailed     = 105
	CodeConnClosed     = 102
	CodeWriteFailed    = 104
	CodeBadPacket      = 400
	CodeNotFound       = 404
	CodeNotImplemented = 501

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

// Internal Framework Rerror string.
var (
	rerror_dialFailed  = NewRerror(CodeDialFailed, "Dial Failed", "")
	rerror_connClosed  = NewRerror(CodeConnClosed, "Connection Closed", "")
	rerror_writeFailed = NewRerror(CodeWriteFailed, "Write Failed", "")
)

var (
	// methodNotAllowed_metaSetting = metaSetting(NewRerror(405, "Type Not Allowed", "").String())
	connClosed_metaSetting     = metaSetting(rerror_connClosed.String())
	notFound_metaSetting       = metaSetting(NewRerror(CodeNotFound, "Not Found", "").String())
	writeFailed_metaSetting    = metaSetting(rerror_writeFailed.String())
	notImplemented_metaSetting = metaSetting(NewRerror(CodeNotImplemented, "Not Implemented", "").String())
	badPacket_metaSetting      = metaSetting(NewRerror(CodeBadPacket, "Bad Packet", "").String())
)

type metaSetting string

func (m metaSetting) Inject(meta *utils.Args, detail ...string) {
	if len(detail) > 0 {
		m = m[:len(m)-2] + metaSetting(bytes.Replace(goutil.StringToBytes(detail[0]), re_d, re_e, -1)) + m[len(m)-2:]
	}
	meta.Set(MetaRerrorKey, *(*string)(unsafe.Pointer(&m)))
}
