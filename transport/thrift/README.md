# Thrift

## 什么是Thrift

Thrift是Facebook于2007年开发的跨语言的rpc服框架，提供多语言的编译功能，并提供多种服务器工作模式；用户通过Thrift的IDL（接口定义语言）来描述接口函数及数据类型，然后通过Thrift的编译环境生成各种语言类型的接口文件，用户可以根据自己的需要采用不同的语言开发客户端代码和服务器端代码。2007年由facebook贡献到apache基金，是apache下的顶级项目，具备如下特点：

* 支持多语言：C、C++ 、C# 、D 、Delphi 、Erlang 、Go 、Haxe 、Haskell 、Java 、JavaScript、node.js 、OCaml 、Perl 、PHP 、Python 、Ruby 、SmallTalk
* 消息定义文件支持注释，数据结构与传输表现的分离，支持多种消息格式
* 包含完整的客户端/服务端堆栈，可快速实现RPC，支持同步和异步通信

## Thrift的优缺点

### Thrift的优点

* One-stop shop，相对于protobuf，序列化和RPC支持一站式解决，如果是pb的话，还需要考虑选择RPC框架，现在Google是开源了gRpc，但是几年以前是没有第一方的标准解决方案的
* 特性丰富，idl层面支持map，protobuf应该是最近才支持的，map的key支持任意类型，avro只支持string，序列化支持自定义protocol, rpc支持thread pool, hsha, no blocking 多种形式，必有一款适合你，对于多语言的支持也非常丰富
* RPC和序列化性能都不错，这个到处都有benchmark，并不是性能最好的，但是基本上不会成为瓶颈或者短板
* 有很多开源项目的周边支持都是thrift的，hbase提供thrift服务，hive，spark sql，cassandra等一系列对外的标准服务接口都是thrift的以支持多语言。
* Column Storage的话，parquet支持直接通过thrift idl转换，如果在Hadoop集群上存储数据，elephant-bird 支持得很好，你可以很方便地针对thrift的数据通过pig写dsl，如果你希望在rpc服务外做一系列工作，可以用finagle包装一层。不过，这部分对于protobuf和avro支持一般也不错

### Thrift的缺点

* 基本没有官方文档
* RPC在 0.6.1 升级到 0.7.0 是不兼容的！这个对于早于 0.6.1 开始使用的用户来说是个大坑
* bug fix和更新不积极，好在序列化和RPC服务都不是太复杂的问题，需要考量的设计问题不多，自己维护patch的成本不高，如果我没有记错的话，0.6.1的java的ThreadPool Server是会有Thread死亡之后的Thread泄露问题的
* 不支持双向通道，如果要支持双向通道比较麻烦
* rpc方法非线程安全，这就是为何很多时候服务器会被挂死，是因为客户端的并发rpc调用导致的，只需要客户端对rpc的调用进行串行化即可。统一服务器应答的时候，也需要串行化，否则有可能会把对方给挂死。特别是在多线程情况下。

## 什么时候应该选择Thrift

* 需要在非常多的语言间进行数据交换
* 对CPU敏感
* 协议层、传输层有多种控制要求
* 需要稳定的版本
* 不需要良好的文档和示例

## Thrift 网络栈架构

### TTransport 层

* TSocket ：阻塞 Socket
* TFrameTransport ：以 frame 为单位进行传输， 非阻塞式服务中使用
* TFileTransport : 以文件形式进行传输

### TProtocol 层

代表 thrift 客户端和服务端之间的传输数据的协议，指的是客户端和服务端传输数据的格式，比如 Json, thrift 中有如下格式：

* TBinaryProtocol：二进制格式
* TCompactProtocol：压缩格式
* TJSONProtocol : Json格式
* TSimpleJsonProtocol：提供只写的 JSON 协议

Thrift 支持的 Server 模型

* TSimpleServer ：用于简单的单线程模型，常用于测试
* TThreadPoolServer ：多线程模型，使用标准的阻塞 IO
* TNoBlockingServer: 多线程服务模型，使用非阻塞 IO，需要使用TFramedTransport 数据传输方式。
* THsHaServer : THsHa
  引入了线程池去处理，其模型读写任务放到线程池去处理，Half-sync/Half-async处理模式，Half-async是在处理IO事件上(
  accept/read/write io)，Half-sync用于handler对rpc的同步处理；

## Thrift支持的数据类型以及关键字

### 基本数据类型

| 类型  | 说明  |
|-----|-----|
|  byte   | 有符号字节    |
|   i16  |  16 位有符号整数   |
|   i32  |  32 位有符号整数   |
|   i64  |  64 位有符号整数   |
|   double  |   64 位浮点数  |
|   string  |  字符串   |

### 容器类型

| 类型  | 说明  |
|-----|-----|
|  list   | 一系列由 T 类型的数据组成的有序列表, 元素可以重复    |
|   set  |  一系列由 T 类型组成的无序集合，元素不可以重复   |
|   map  |  一个字典结构，Key 为 K 类型， Value 为 V 类型，和 Java 中的 HashMap 类似thrift 支持 struct 类型，可以将一些数据类型聚合到一块。   |

### Struct类型

Struct：类似于C的struct，是一系列相关数据的封装，在OOP语言中会转换为类(class)，struct的每个元素包括一个唯一的数字标识、一个数据类型、一个名称和一个可选的默认值。语法：

```thrift
struct People {
    1:string name;
    2:i32 age;
    3:string gender;
}
```

### Union类型

```thrift
union SomeUnion {
  2: string string_thing,
  3: i32 i32_thing
}
```

### Enum类型

Enum：Thrift枚举类型只支持单个32位int类型数据，第一个元素如果没有给值那么默认是0，之后的元素如果没有给值，则是在前一个元素基础上加1，语法：

```thrift
enum Gender {
    MALE,
    FEMALE
}
```

### 别名

Typedef：Thrift 支持C/C++风格的类型自定义，语法：

```thrift
// typedef 原类型 自定义类型

typedef i32 int
typedef i64 long
```

### 常量

Const：定义常量，Thrift允许使用JSON来定义复杂类型和struct类型，语法：

```thrift
// const 字段类型 名称标识 = 值 | 列表

const i32 MAX_RETRIES_TIME = 10;
const string MY_WEBSITE = "http://facebook.com";
```

### Exception类型

Exception：异常跟struct类似，会跟目标语言本地异常集成，语法：

```thrift
exception RequestException {
    1:i32 code;
    2:string reason;
}
```

### Service类型

Service：service是Thrift 服务器提供的一系列功能列表接口，在客户端就是调用这些接口来完成操作，语法：

```thrift
service HelloWorldService {
    // service中可以定义若干个服务，相当于Java Interface中定义的方法
    string doAction(1:string name, 2:i32 age);
}
```

### 命名空间

定义名称空间/包名/模块等等，可以使用编程语言名称规定某一特定语言的namespace，用*表示所有未匹配到的语言的namespace，语法

```thrift
// namespace [语言名称] 标识符

namespace cpp api
namespace go api
namespace d api
namespace dart api
namespace java api
namespace php api
namespace perl api
namespace haxe api
namespace netstd api

// 用*表示所有未匹配到的语言的namespace
namespace * api
```

### 注释

```thrift
# shell风格注释

/*
 * 多行注释
 */

// 单行注释
```

### 包含引用

Thrift Include：将所有声明包含的Thrift文档都包含到当前Thrift中来，语法：

```thrift
// 包含其他的thrift文件
// inlucde "文件名"
include "other.thrift"
```

C++ Include：将用户自己的C++头文件包含到当前Thrift中来，语法：

```thrift
// 将用户自己的C++头文件包含到当前Thrift中来
// cpp_include "头文件"
cpp_include "string"
```

## 安装编译器

### Linux安装编译器

```bash
sudo apt install thrift-compiler
```

### Windows安装编译器

先去官网下载编译器：<https://thrift.apache.org/download>

然后把编译器放在一个全局可以运行的目录下面，比如：c:/Windows。

### Mac安装编译器

```bash
brew install thrift
```

## 编译Thrift文件

```bash
# 
thrift --gen <language> <Thrift filename>

# 如果有thrift文件中有包含其他thrift，可以使用递归生成命令
thrift -r --gen <language> <Thrift filename>

# 示例
thrift -r -gen go tutorial.thrift
```

## 开始在Kratos微服务框架下使用Thrift

我封装了一个Thrift服务，可以在Kratos微服务框架下直接使用：<https://github.com/tx7do/kratos-transport/tree/main/transport/thrift>
。

实例程序的目标是从服务器获取温湿度信息，然后将温湿度信息发送给客户端。示例代码可以在单元测试里面找到。

### 创建Thrift文件

```thrift
namespace go api

struct Hygrothermograph {
  1: optional double Humidity,
  2: optional double Temperature,
}

service HygrothermographService {
  Hygrothermograph getHygrothermograph()
}
```

然后生成代码。生成的代码在thrift文件所在目录下的：gen-go/{namespace}之下。

### 开发服务器

#### 首先，实现Handler

```go
package server

type HygrothermographHandler struct {
}

func NewHygrothermographHandler() *HygrothermographHandler {
    return &HygrothermographHandler{}
}

func (p *HygrothermographHandler) GetHygrothermograph(ctx context.Context) (_r *api.Hygrothermograph, _err error) {
  var Humidity = float64(rand.Intn(100))
  var Temperature = float64(rand.Intn(100))
  _r = &api.Hygrothermograph{
    Humidity:    &Humidity,
    Temperature: &Temperature,
  }
  fmt.Println("Humidity:", Humidity, "Temperature:", Temperature)
  return
}
```

Handler在Kratos里面的使用的语义是Service，实际应用的时候，将之实现为Service。

#### 实现服务端

```go
package server

ctx := context.Background()

srv := NewServer(
  WithAddress(":7700"),
  WithProcessor(api.NewHygrothermographServiceProcessor(NewHygrothermographHandler())),
)

if err := srv.Start(ctx); err != nil {
    panic(err)
}

defer func () {
  if err := srv.Stop(ctx); err != nil {
    t.Errorf("expected nil got %v", err)
  }
}()
```

使用`WithProcessor`方法我们把之前的`Handler`注册成`Processor`到服务器。

### 开发客户端

```go
package client

conn, err := Dial(
    WithEndpoint("localhost:7700"),
)
if err != nil {
    t.Fatal(err)
}
defer conn.Close()

client := api.NewHygrothermographServiceClient(conn.Client)

reply, err := client.GetHygrothermograph(context.Background())
//t.Log(err)
if err != nil {
  t.Errorf("failed to call: %v", err)
}
t.Log(*reply.Humidity, *reply.Temperature)
```

Thrift客户端跟gRpc的客户端是一样的，使用`Dial`方法创建一个连接，然后直接调用RPC方法。

## 参考文档

* [Thrift官方网站](https://thrift.apache.org/)
* [Thrift各平台安装方法](https://thrift.apache.org/docs/install/)
* [thrift 原理浅析](https://developer.aliyun.com/article/937914?spm=a2c6h.12873639.article-detail.42.513b14e8U6LbgI)
* [Thrift入门基础知识-thrift文件(IDL)说明和生成目标语言源代码](https://www.jianshu.com/p/6913543a5770)
* [哪个互联网公司使用 facebook thrift 做底层架构，实现高性能、可扩展的web应用？引入thrift之后的优缺点是什么？](https://www.zhihu.com/question/20189791)
* [一文搞懂gRPC和Thrift的基本原理和区别](https://juejin.cn/post/6984961473378271245)
