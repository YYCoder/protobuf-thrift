namespace * test.test.test;

// comment enum
enum Status {
    StatusUnknown = 0
    StatusUnreviewed = 1 // comment enum
    StatusOnline = 2
    StatusRejected = 3
    StatusOffline = 4
}
/**
 * comments
 *comments 
 */
struct Config {
    1: i64 Id
    2: i32 Tag
    3: list<i32> TypeList
    4: bool Boolean // comment
    5: Status Status /* 1231231     asdasd  */
    6: map<i64, string> FailMap /* asdasdasdasdasdsad  */
    7: double Fl
    8: double Db
    9: binary Bs
    10: TimeRange Nested
    11: list<TimeRange> NestedTypeList
    12: map<string, TimeRange> NestedTypeMap
}
struct TimeRange {
    1: i64 Start
    2: i64 End
}
struct ReqOfTestGetApi {
    1: i64 A
    2: string B
}
struct RespOfTestGetApi {
    1: i32 Code
    2: string Message
}
struct ReqOfTestPostApi {
    1: i64 A
    2: string B
}
struct RespOfTestPostApi {
    1: i32 Code
    2: string Message
}
// service comment aaaa

service APIs {
    // rpc comment aaaa
    RespOfTestGetApi TestGetApi (1: ReqOfTestGetApi Req)
    /**
     * rpc comment bbbb 
     */
    RespOfTestPostApi TestPostApi (1: ReqOfTestPostApi Req)
}
