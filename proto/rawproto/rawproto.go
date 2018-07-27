package rawproto

import (
	"github.com/henrylee2cn/teleport/socket"
)

// NewRawProtoFunc is creation function of fast socket protocol.
// NOTE:
//  it is the default protocol.
//  id:'r', name:"raw"
var NewRawProtoFunc = socket.NewRawProtoFunc
