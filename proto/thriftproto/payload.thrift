namespace go payload
namespace perl payload
namespace py payload
namespace cpp payload
namespace rb payload
namespace java payload

// thrift-0.11.0

struct payload {
    1: binary  meta,
    2: binary  xferPipe,
    3: i32     bodyCodec,
    4: binary  body
}
