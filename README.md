# protobuf-thrift
Little cli utility for lazy guy😉 ~ Transforming protobuf idl to thrift, and vice versa.

[![YYCoder](https://circleci.com/gh/YYCoder/protobuf-thrift.svg?style=svg)](https://app.circleci.com/pipelines/github/YYCoder/protobuf-thrift)
[![GoDoc](https://pkg.go.dev/badge/github.com/YYCoder/protobuf-thrift)](https://pkg.go.dev/github.com/YYCoder/protobuf-thrift)
[![goreportcard](https://goreportcard.com/badge/github.com/yycoder/protobuf-thrift)](https://goreportcard.com/report/github.com/yycoder/protobuf-thrift)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)

> [IDL](https://en.wikipedia.org/wiki/IDL)(Interface description language), which is a descriptive language used to define data types and interfaces in a way that is independent of the programming language or operating system/processor platform.

[中文文档](./docs/cn.md)

Feel free to try out our [web interface](https://pb-thrift.markeyyuan.monster/), and of course, both languages specification as below, if there are any questions, don't hesitate to open an issue, and PRs are welcome too.

* [thrift](https://thrift.apache.org/docs/idl.html)

* [protobuf](https://developers.google.com/protocol-buffers/docs/reference/proto3-spec)

## Install
### Use as a executable
1. first git clone this repo, `git clone github.com/YYCoder/protobuf-thrift`

2. make, the executable will be compiled to `./exe` folder

### Use as a library
1. go get this in your go module, `go get github.com/YYCoder/protobuf-thrift`

2. import package from `github.com/YYCoder/protobuf-thrift`


## Usages

### Basic Usage
Basic thrift file to protobuf file transform:

```
protobuf-thrift -t thrift2proto -i ./path/to/idl.thrift -o ./idl.proto
```

Basic protobuf file to thrift file transform:

```
protobuf-thrift -t proto2thrift -i ./path/to/idl.proto -o ./idl.thrift
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

We hope you don't have to worry about protobuf-thrift do sth unexpected, so we **strongly recommend you to read the following document** to get a grasp of what it will do for specific syntaxes.

### Basic Types
Here is a list of basic type conversion rules:

|[protobuf type](https://developers.google.com/protocol-buffers/docs/proto3#scalar)|[thrift type](https://thrift.apache.org/docs/types.html#base-types)|
|:--:|:--:|
|uint32|-|
|uint64|-|
|sint32|-|
|sint64|-|
|fixed32|-|
|fixed64|-|
|sfixed32|-|
|sfixed64|-|
|-|i16|
|int32|i32|
|int64|i64|
|float|double|
|double|double|
|bool|bool|
|string|string|
|bytes|-|
|-|byte|

### Enum
Protobuf and thrift both have `enum` declaration syntax and basically same grammar, only to note that:

> **Proto3 enum declaration's first element must be zero, so if thrift enum with non-zero first element transform to protobuf, protobut-thrift will automatically generate a zero element for you.**

for example, if thrift enum like this:

```thrift
enum Status {
    StatusUnreviewed = 1 // first non-zero element
    StatusOnline = 2
    StatusRejected = 3
    StatusOffline = 4
}
```

will be transformed to:

```protobuf
enum Status {
    Status_Unknown = 0;
    Status_Unreviewed = 1; // first non-zero element
    Status_Online = 2;
    Status_Rejected = 3;
    Status_Offline = 4;
}
```

### Service
Protobuf and thrift both have same `service` declaration syntax, but there are several differences:

1. **oneway**: only thrift support, which means function will not wait for response. so during thrift-to-pb transformation, this keyword will be ignored.

2. **throws**: only thrift support, which specified what kind of exceptions can be thrown by the function. this keyword will be ignored, too, in thrift-to-pb mode.

3. **arguments**: 
    * thrift supports multiple arguments for one function, but protobuf only supports one, so it will ignore all the arguments other than the first one in thrift-to-pb transformation.

    * thrift functions support `void` return type, but protobuf doesn't, so it will leave the return type blank in thrift-to-pb mode.
    
    * currently, only support basic type and identifier for function/rpc request and response type, might be implemented in the future.

### Options || Annotation
Both language support this feature, but they have different syntax to apply it, since the meaning for them are language-bound, we decide to ignore this between transformations.

### Message || Struct
Thrift `struct` and protobuf `message` are very similar, but still have some differences:

1. **set type**: only thrift support, it will be transformed to `repeated` field in protobuf just like thrift `list`.

2. **optional**: thrift and proto2 support, it will be ignored in thrift-to-pb mode if protobuf syntax is proto3

3. **required**: thrift and proto2 support, since it's highly not recommend to mark field as `required`, currently it will be ignored, if you have any questions about this, please open an issue.

4. **map type**: as protobuf [language-specification](https://developers.google.com/protocol-buffers/docs/reference/proto3-spec#map_field) mentioned, protobuf only support basic type as key type, but thrift support any [FieldType](https://thrift.apache.org/docs/idl.html) as map key type, for simplicity, currently only support basic type and identifier as map key and value


### Import || Include
As [language-specification](https://developers.google.com/protocol-buffers/docs/proto#importing_definitions) mentioned, protobuf import paths are relative to protoc command's working directory or using -I/--proto_path specified path, and can not include relative paths prefix, such as `./XXX.proto`, we are not able to detect the correct path for current file both in thrift-to-pb mode and pb-to-thrift mode, since it's dynamic.

So, you have to manually check whether the generated path is correct.

### Constant || Const
Currently not supported.

### Package || Namespace
Thrift `namespace` value will be used for protobuf `package`, the NamespaceScope will be ignored in thrift-to-pb mode.

In pb-to-thrift mode, generated `namespace` will use `*` as NamespaceScope.

### Nested Types
Protobuf supports [nested types](https://developers.google.com/protocol-buffers/docs/proto#nested) within message, but thrift does not, so protobuf-thrift will prefix nested field name with outer message name to work around this. for example:

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

