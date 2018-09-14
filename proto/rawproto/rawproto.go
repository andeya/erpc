package rawproto

import (
	"github.com/henrylee2cn/teleport/socket"
)

/*
# raw protocol format(Big Endian):

{4 bytes message length}
{1 byte protocol version}
{1 byte transfer pipe length}
{transfer pipe IDs}
# The following is handled data by transfer pipe
{2 bytes sequence length}
{sequence}
{1 byte message type} # e.g. CALL:1; REPLY:2; PUSH:3
{2 bytes URI length}
{URI}
{2 bytes metadata length}
{metadata(urlencoded)}
{1 byte body codec id}
{body}
*/

// NewRawProtoFunc is creation function of fast socket protocol.
// NOTE:
//  it is the default protocol.
//  id:'r', name:"raw"
var NewRawProtoFunc = socket.NewRawProtoFunc
