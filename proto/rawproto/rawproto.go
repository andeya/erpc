package rawproto

import (
	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/socket"
)

/*
# raw protocol format(Big Endian):

{4 bytes message length}
{1 byte protocol version} # 6
{1 byte transfer pipe length}
{transfer pipe IDs}
# The following is handled data by transfer pipe
{1 bytes sequence length}
{sequence (HEX 36 string of int32)}
{1 byte message type} # e.g. CALL:1; REPLY:2; PUSH:3
{1 bytes service method length}
{service method}
{2 bytes status length}
{status(urlencoded)}
{2 bytes metadata length}
{metadata(urlencoded)}
{1 byte body codec id}
{body}
*/

// NewRawProtoFunc is creation function of fast socket protocol.
// NOTE:
//  it is the default protocol.
//  id:6, name:"raw"
func NewRawProtoFunc() erpc.ProtoFunc {
	return socket.RawProtoFunc
}
