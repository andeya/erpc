package rawproto

import (
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/socket"
)

/*
# raw protocol format(Big Endian):

{4 bytes message length}
{1 byte protocol version}
{1 byte transfer pipe length}
{transfer pipe IDs}
# The following is handled data by transfer pipe
{1 bytes sequence length}
{sequence}
{1 byte message type} # e.g. CALL:1; REPLY:2; PUSH:3
{1 bytes service method length}
{service method}
{2 bytes metadata length}
{metadata(urlencoded)}
{1 byte body codec id}
{body}
*/

// NewRawProtoFunc is creation function of fast socket protocol.
// NOTE:
//  it is the default protocol.
//  id:'r', name:"raw"
func NewRawProtoFunc() tp.ProtoFunc {
	return socket.RawProtoFunc
}
