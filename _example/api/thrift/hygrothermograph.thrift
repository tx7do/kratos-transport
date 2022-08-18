// thrift -r -gen go hygrothermograph.thrift

namespace cpp api
namespace go api
namespace d api
namespace dart api
namespace java api
namespace php api
namespace perl api
namespace haxe api
namespace netstd api

struct Hygrothermograph {
  1: optional double Humidity,
  2: optional double Temperature,
}

service HygrothermographService {
  Hygrothermograph getHygrothermograph()
}
