# Thrift

## 安装编译器

### Linux安装编译器

```bash
sudo apt install thrift-compiler
```

### Windows安装编译器

#### Scoop

```bash
scoop install main/thrift
```

#### Chocolatey

```bash
choco install thrift
```

#### 下载二进制

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
thrift -r --gen go:package_prefix=github.com/tx7do/kratos-transport/testing/api/thrift/gen-go/ shared.thrift
thrift -r --gen go:package_prefix=github.com/tx7do/kratos-transport/testing/api/thrift/gen-go/ tutorial.thrift
thrift -r --gen go:package_prefix=github.com/tx7do/kratos-transport/testing/api/thrift/gen-go/ echo.thrift
thrift -r --gen go:package_prefix=github.com/tx7do/kratos-transport/testing/api/thrift/gen-go/ hygrothermograph.thrift
```