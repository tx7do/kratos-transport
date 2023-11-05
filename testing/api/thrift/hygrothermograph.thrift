// thrift -r -gen go hygrothermograph.thrift

namespace cpp hygrothermograph
namespace go hygrothermograph
namespace d hygrothermograph
namespace dart hygrothermograph
namespace java hygrothermograph
namespace php hygrothermograph
namespace perl hygrothermograph
namespace haxe hygrothermograph
namespace netstd hygrothermograph

struct Hygrothermograph {
  1: optional double Humidity,
  2: optional double Temperature,
}

service HygrothermographService {
  Hygrothermograph getHygrothermograph()
}
