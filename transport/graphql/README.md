# GraphQL

## 什么是GraphQL？

GraphQL 是一种用于应用编程接口（API）的查询语言和服务器端运行时，它可以使客户端准确地获得所需的数据，没有任何冗余。

GraphQL 由 Facebook 开发，并于 2012 年首次应用于移动应用。GraphQL 规范于 2015 年实现开源。现在，它受 GraphQL 基金会监管。

GraphQL 是一种针对 Graph（图状数据）进行查询特别有优势的 Query Language（查询语言），所以叫做 GraphQL。

它跟 SQL 的关系是共用 QL 后缀，就好像「汉语」和「英语」共用后缀一样，但他们本质上是不同的语言。

GraphQL 跟用作存储的 NoSQL 没有必然联系，虽然 GraphQL 背后的实际存储可以选择 NoSQL 类型的数据库，但也可以用 SQL 类型的数据库，或者任意其它存储方式（例如文本文件、存内存里等等）。

GraphQL 最大的优势是查询图状数据。

## GraphQL有什么用？

GraphQL 旨在让 API 变得快速、灵活并且为开发人员提供便利。它甚至可以部署在名为 GraphiQL 的集成开发环境（IDE）中。作为 REST 的替代方案，GraphQL 允许开发人员构建相应的请求，从而通过单个 API 调用从多个数据源中提取数据。

此外，GraphQL 还可让 API 维护人员灵活地添加或弃用字段，而不会影响现有查询。开发人员可以使用自己喜欢的方法来构建 API，并且 GraphQL 规范将确保它们以可预测的方式在客户端发挥作用。

## GraphQL 的优缺点

### GraphQL 的优点

* GraphQL 模式会在 GraphQL 应用中设置单一事实来源。它为企业提供了一种整合其整个 API 的方法。
* 一次往返通讯可以处理多个 GraphQL 调用。客户端可得到自己所请求的内容，不会超量。
* 严格定义的数据类型可减少客户端与服务器之间的通信错误。
* GraphQL 具有自检功能。客户端可以请求一个可用数据类型的列表。这非常适合文档的自动生成。
* GraphQL 允许应用 API 进行更新优化，而无需破坏现有查询。
* 许多开源 GraphQL 扩展可提供 REST API 所不具备的功能。
* GraphQL 不指定特定的应用架构。它能够以现有的 REST API 为基础，并与现有的 API 管理工具配合使用。

### GraphQL 的缺点

* 即便是熟悉 REST API 的开发人员，也需要一定时间才能掌握 GraphQL。
* GraphQL 将数据查询的大部分工作都转移到服务器端，由此增加了服务器开发人员工作的复杂度。
* 根据不同的实施方式，GraphQL 可能需要不同于 REST API 的 API 管理策略，尤其是在考虑速率限制和定价的情况下。
* 缓存机制比 REST 更加复杂。
* API 维护人员还会面临编写可维护 GraphQL 模式的额外任务。

## GraphQL支持的数据类型以及关键字

### 标量类型

* Int：带符号的32位整数，对应 JavaScript 的 Number
* Float：带符号的双精度浮点数，对应 JavaScript 的 Number
* String：UTF-8字符串，对应 JavaScript 的 String
* Boolean：布尔值，对应 JavaScript 的 Boolean
* ID：ID 值，是一个序列化后值唯一的字符串，可以视作对应 ES 2015 新增的 Symbol

### 高级类型

#### 接口类型

`Interface`是包含一组确定字段的集合的抽象类型，实现该接口的类型必须包含`interface`定义的所有字段。比如：

```graphql
interface Character {
  id: ID!
  name: String!
  friends: [Character]
  appearsIn: [Episode]!
}

type Human implements Character {
  id: ID!
  name: String!
  friends: [Character]
  appearsIn: [Episode]!
  starships: [Starship]
  totalCredits: Int
}

type Droid implements Character {
  id: ID!
  name: String!
  friends: [Character]
  appearsIn: [Episode]!
  primaryFunction: String
}
```

#### 联合类型

Union类型非常类似于interface，但是他们在类型之间不需要指定任何共同的字段。通常用于描述某个字段能够支持的所有返回类型以及具体请求真正的返回类型。比如定义：

```graphql
union SearchResult = Human | Droid | Starship
```

#### 枚举类型

又称Enums,这是一种特殊的标量类型，通过此类型，我们可以限制值为一组特殊的值。比如：

```graphql
enum Episode {
  NEWHOPE
  EMPIRE
  JEDI
}
```

#### 输入类型

input类型对mutations来说非常重要，在 GraphQL schema 语言中，它看起来和常规的对象类型非常类似，但是我们使用关键字input而非type,input类型按如下定义：

```graphql
input CommentInput {
    body: String!
}
```

为什么不直接使用Object Type呢？因为 Object 的字段可能存在循环引用，或者字段引用了不能作为查询输入对象的接口和联合类型。

#### 数组类型和非空类型

使用`[]`来表示**数组**，使用`!`来表示**非空**。`Non-Null`强制类型的值不能为null，并且在请求出错时一定会报错。可以用于必须保证值不能为null的字段

#### 对象类型

GraphQL schema最基本的类型就是Object Type。用于描述层级或者树形数据结构。比如：

```graphql
type Character {
  name: String!
  appearsIn: [Episode!]!
}
```

## GraphQL查询语法

GraphQL的一次操作请求被称为一份文档（document），即GraphQL服务能够解析验证并执行的一串请求字符串(Source Text)。完整的一次操作由操作（Operation）和片段（Fragments）组成。一次请求可以包含多个操作和片段。只有包含操作的请求才会被GraphQL服务执行。

只包含一个操作的请求可以不带OperationName，如果是operationType是query的话，可以全部省略掉，即：

```graphql
{
  getMessage {
    # query也可以拥有注释，注释以#开头
    content
    author
  }
}
```

当query包含多个操作时，所有操作都必须带上名称。

GraphQL中，我们会有这样一个约定，Query和与之对应的Resolver是同名的，这样在GraphQL才能把它们对应起来。

### Query

Query用做读操作，也就是从服务器获取数据。以上图的请求为例，其返回结果如下，可以看出一一对应，精准返回数据

```json
{
  "data": {
    "single": [
      {
        "content": "test content 1",
        "author": "pp1"
      }
    ],
    "all": [
      {
        "content": "test content",
        "author": "pp",
        "id": "0"
      },
      {
        "content": "test content 1",
        "author": "pp1",
        "id": "1"
      }
    ]
  }
}
```

### Field

Field是我们想从服务器获取的对象的基本组成部分。`query`是数据结构的顶层，其下属的`all`和`single`都属于它的字段。

字段格式应该是这样的：`alias:name(argument:value)`

其中 alias 是字段的别名，即结果中显示的字段名称。

name 为字段名称，对应 schema 中定义的 fields 字段名。

argument 为参数名称，对应 schema 中定义的 fields 字段的参数名称。

value 为参数值，值的类型对应标量类型的值。

### Argument

和普通的函数一样，query可以拥有参数，参数是可选的或必须的。参数使用方法如上图所示。

**需要注意的是，GraphQL中的字符串需要包装在双引号中。**

### Variables

除了参数，`query`还允许你使用变量来让参数可动态变化，变量以`$`开头书写，使用方式如上图所示

变量还可以拥有默认值：

```graphql
query gm($id: ID = 2) {
  # 查询数据
  single: getMessage(id: $id) {
    ...entity
  }
  all: getMessage {
    ...entity
    id
  }
}
```

### Allases

别名，比如说，我们想分别获取全部消息和ID为1的消息，我们可以用下面的方法：

```graphql
query gm($id: ID = 2) {
  # 查询数据
  getMessage(id: $id) {
    ...entity
  }
  getMessage {
    ...entity
    id
  }
}
```

由于存在相同的name，上述代码会报错，要解决这个问题就要用到别名了Allases。

```graphql
query gm($id: ID = 2) {
  # 查询数据
  single: getMessage(id: $id) {
    ...entity
  }
  all: getMessage {
    ...entity
    id
  }
}
```

### Fragments

Fragments是一套在queries中可复用的fields。比如说我们想获取Message，在没有使用fragment之前是这样的：

```graphql
query gm($id: ID = 2) {
  # 查询数据
  single: getMessage(id: $id) {
    content
    author
  }
  all: getMessage {
    content
    author
    id
  }
}
```

但是如果fields过多，就会显得重复和冗余。Fragments在此时就可以起作用了。使用了Fragment之后的语法就如上图所示，简单清晰。

Fragment支持多层级地继承。

### Directives

`Directives`提供了一种动态使用变量改变我们的`queries`的方法。如本例，我们会用到以下两个`directive`:

```graphql
query gm($id: ID = 2, $isNotShowId: Boolean!, $showAuthor: Boolean!) {
  # 查询数据
  single: getMessage(id: $id) {
    ...entity
  }
  all: getMessage {
    ...entity
    id @skip(if: $isNotShowId)
  }
}

fragment entity on Message {
  content
  author @include(if: $showAuthor)
}

### 入参是：
{
  "id": 1,
  "isNotShowId": true,
  "showAuthor": false
}
```

@include: 只有当if中的参数为true时，才会包含对应fragment或field；

@skip:当if中的参数为true时,会跳过对应fragment或field；

结果如下：

```json
{
  "data": {
    "single": [
      {
        "content": "test content 1"
      }
    ],
    "all": [
      {
        "content": "test content"
      },
      {
        "content": "test content 1"
      }
    ]
  }
}
```

### Mutation

传统的API使用场景中，我们会有需要修改服务器上数据的场景，mutations就是应这种场景而生。mutations被用以执行写操作，通过mutations我们会给服务器发送请求来修改和更新数据，并且会接收到包含更新数据的反馈。mutations和queries具有类似的语法，仅有些许的差别。

1. operationType为mutation
2. 为了保证数据的完整性mutations是串形执行，而queries可以并行执行。

### Subscription

Subscription是GraphQL最后一个操作类型，它被称为订阅。当另外两个由客户端通过HTTP请求发送，订阅是服务器在某个事件发生时将数据本身推送给感兴趣的客户端的一种方式。这就是GraphQL 处理实时通信的方式。

## 编译GraphQL文件

在graphql同级目录下创建一个配置文件，命名为：gqlgen.yml

```yaml
schema:
  - "*.graphql"
```

或者是指定导出models项：

```yaml
models:
  Todo:
    model: github.com/Laisky/laisky-blog-graphql.Todo
```

然后在graphql文件同级目录下，使用命令行执行以下命令，即可生成go代码：

```shell
# 直接运行
go run github.com/99designs/gqlgen

# 安装
go get github.com/99designs/gqlgen
go install github.com/99designs/gqlgen
# 然后运行
gqlgen
```
