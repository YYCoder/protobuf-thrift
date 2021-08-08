# protobuf-thrift
Little thing for lazy guy, transforming protobuf idl to thrift, and vice versa.

> IDL(Interface description language), reference [Wikipedia](https://en.wikipedia.org/wiki/IDL).

[cn](./docs/cn.md)

## Caveats

Since protobuf and thrift have many different grammars, so we can only transform grammars that have same meaning, e.g. protobuf message => thrift struct, protobuf enum => thrift enum.

Here is a list of transformation rule, so we hope you won't have to worry about protobuf-thrift 

* 

