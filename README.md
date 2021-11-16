# protobuf-thrift
Little cli utility for lazy guy😉 ~ Transforming protobuf idl to thrift, and vice versa.

[![YYCoder](https://circleci.com/gh/YYCoder/protobuf-thrift.svg?style=svg)](https://app.circleci.com/pipelines/github/YYCoder/protobuf-thrift)
[![GoDoc](https://pkg.go.dev/badge/github.com/YYCoder/protobuf-thrift)](https://pkg.go.dev/github.com/YYCoder/protobuf-thrift)
[![goreportcard](https://goreportcard.com/badge/github.com/yycoder/protobuf-thrift)](https://goreportcard.com/report/github.com/yycoder/protobuf-thrift)

> [IDL](https://en.wikipedia.org/wiki/IDL)(Interface description language), which is a descriptive language used to define data types and interfaces in a way that is independent of the programming language or operating system/processor platform.

[中文文档](./docs/cn.md)


## Install
For folks don't have GO development environment, directly download corresponding platform binary from latest release is the best choice.

For Gophers, you can just `go install github.com/YYCoder/protobuf-thrift` it yourself.


## Usages

### Basic Usage
Basic thrift file to protobuf file transform:

```
protobuf-thrift -t thrift2proto -i ./path/to/idl.thrift -o ./idl.proto
```

Basic protobuf file to thrift file transform:

```
protobuf-thrift -t proto2thrift -i ./path/to/idl.thrift -o ./test.proto
```

### Interactive Usage
You can simply run like `protobuf-thrift -t thrift2proto` and then, paste your original idl file to the terminal and press ctrl+D.

![interactive.gif](./docs/2021-08-09%2021_54_20.gif)

> Note that interactive mode can not use **-r** option, as there is no files, only stdin.

### Case Converting
Protobuf-thrift provides complete case convert feature, thanks to [strcase](https://github.com/iancoleman/strcase), available options already listed in **--help** message.

### Recursive Transforming
Under some circumstances, you may want to transform a whole idl repo to another language, we provide you **-r** option to indicate protobuf-thrift to transform all imported files.

This option is off by default, so you have to specify it explicitly.

```
protobuf-thrift -t thrift2proto -i ./path/to/idl.thrift -o ./idl.proto -r 1
```


## Options

![](./docs/usage.jpeg)

## Notice

Since protobuf and thrift have many different syntaxes, we can only transform syntaxes that have same meaning, e.g. protobuf message => thrift struct, protobuf enum => thrift enum.

Here is a list of transformation rule, so we hope you don't have to worry about protobuf-thrift do sth unexpected.

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

### Nested Fields
protobuf support nested field within message, but thrift does not, so protobuf-thrift will prefix nested field name with outer message name to work around this. for example:

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

will transform to:

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

