include "./test.thrift";
include "./common/admin.thrift";
enum OtherEnum {
    OtherEnumUnknown = 0
    Unreviewed = 1
    Online = 2
    Rejected = 3
    Offline = 4
}
struct Config {
    1: i64 Id
    2: i32 Tag
    3: list<i32> TypeList
    4: bool Boolean
    5: admin.Status Status
    6: map<i64,string> FailMap
    7: double Fl
    8: double Db
    9: binary Bs
    10: test.TimeRange Nested
    11: list<test.TimeRange> NestedTypeList
    12: map<string,test.TimeRange> NestedTypeMap
}
service APIs {
    Config TestOther (1: Config Req)
}
