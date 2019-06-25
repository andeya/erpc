namespace go thriftproto
namespace perl thriftproto
namespace py thriftproto
namespace cpp thriftproto
namespace rb thriftproto
namespace java thriftproto

struct payload {
    1: binary  meta,
    2: binary  xferPipe,
    3: i32     bodyCodec,
    4: binary  body
}

struct test {
    1: string  author,
}