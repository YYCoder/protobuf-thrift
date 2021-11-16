# protobuf-thrift
为懒人准备的 protobuf 与 thrift 互转的小工具😉。

[![YYCoder](https://circleci.com/gh/YYCoder/protobuf-thrift.svg?style=svg)](https://app.circleci.com/pipelines/github/YYCoder/protobuf-thrift)
[![GoDoc](https://pkg.go.dev/badge/github.com/YYCoder/protobuf-thrift)](https://pkg.go.dev/github.com/YYCoder/protobuf-thrift)
[![goreportcard](https://goreportcard.com/badge/github.com/yycoder/protobuf-thrift)](https://goreportcard.com/report/github.com/yycoder/protobuf-thrift)

> [IDL](https://en.wikipedia.org/wiki/IDL)(Interface description language)。是指一种用于定义数据类型以及接口的描述性语言，与编程语言以及平台无关，常用在微服务架构中。

## 安装
如果没有 go 开发环境，可以直接从 release 中下载最新版的对应平台的二进制文件。

对于 Gophers，可以直接使用 `go install github.com/YYCoder/protobuf-thrift`。

## 使用示例

### 基本用法
将 thrift 文件转成 protobuf 文件：

```
protobuf-thrift -t thrift2proto -i ./path/to/idl.thrift -o ./idl.proto
```

将 protobuf 文件转成 thrift 文件：

```
protobuf-thrift -t proto2thrift -i ./path/to/idl.thrift -o ./test.proto
```

### 交互式用法
直接使用 `protobuf-thrift -t thrift2proto` 命令会进入交互模式，直接粘贴你的源 idl 源码到终端，并按下 ctrl+D 即可。

![interactive.gif](./2021-08-09%2021_54_20.gif)

### 大小写转换
得益于 [strcase](https://github.com/iancoleman/strcase)，Protobuf-thrift 提供了完整的变量大小写转换能力，可用的选项已经在 **--help** 提示信息中了，请自行查阅。

### 递归转换
某些场景下，我们可能需要将整个 idl 仓库转成另一种语言，此时我们就可以使用 **-r** 选项来递归地将 import 的文件全部转换。

该选项默认是禁用的，要使用它时需要显式指定。


```
protobuf-thrift -t thrift2proto -i ./path/to/idl.thrift -o ./idl.proto -r 1
```


## 可用选项

![](./usage.jpeg)

## 注意事项
由于 protobuf 与 thrift 有很多语法上的不同，我们不可能完全将一种 idl 转换成另一种，protobuf-thrift 也只是一个帮助我们摆脱复制粘贴的小工具，它所提供的功能能够满足 80% 的场景就足够了。因此，我们只会尽可能将有相同语义的语法进行转换，如 protobuf message => thrift struct，protobuf enum => thrift enum。

为了确保你能够明确的知道 protobuf-thrift 会如何转换，如下是目前的转换规则：

|protobuf type|thrift type|field type|notice|
|:--:|:--:|:--:|:--:|
|message|struct|optional => optional; repeated T => list\<T\>|only protobuf 2 have optional field|
|map<T1,T2>|map<T1,T2>||T1 only support int32/int64/string/float/double, due to thrift syntax|
|enum|enum|||
|int32|i32|||
|int64|i64|||
|float|double|||
|double|double|||
|bool|bool|||
|string|string|||
|bytes|binary|||
|service|service|rpc => methods||
|constant|const||not support currently|
|package|namespace|||
|import|include|||
|syntax|||only supported in protobuf, so thrift will omit it|
|option|||only supported in protobuf, so thrift will omit it|
|extend|||only supported in protobuf, so thrift will omit it|
|extension|||only supported in protobuf, so thrift will omit it|

### 嵌套字段
protobuf 支持在 message 结构体中嵌套字段（如 enum/message），但在 thrift 中不支持，因此 protobuf-thrift 会通过给嵌套字段的标识符使用外部 message 名称作为前缀的方式来实现相同命名空间的效果。如下例：

```protobuf
message GroupMsgTaskQueryExpress {
    enum QueryOp {
        Unknown = 0;
        GT = 1;
    }
    message TimeRange {
        int32 range_start = 1;
        int32 range_end = 2;
    }
    QueryOp express_op = 1;
    int32 op_int = 2;
    TimeRange time_op = 3;
    int32 next_op_int = 4;
}
```

会被转换成：

```thrift
struct GroupMsgTaskQueryExpress {
    1: GroupMsgTaskQueryExpressQueryOp ExpressOp
    2: i32 OpInt
    3: GroupMsgTaskQueryExpressTimeRange TimeOp
    4: i32 NextOpInt
}
enum GroupMsgTaskQueryExpressQueryOp {
    Unknown = 0
    GT = 1
}
struct GroupMsgTaskQueryExpressTimeRange {
    1: i32 RangeStart
    2: i32 RangeEnd
}
```



