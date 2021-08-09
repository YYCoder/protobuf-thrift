namespace * test.test.test;

enum status {
	statusUnknown = 0
	statusUnreviewed = 1
	statusOnline = 2
	statusRejected = 3
	statusOffline = 4
}
struct reqOfTestPostApi {
	1: i64 a
	2: string b
}
struct respOfTestPostApi {
	1: i32 code
	2: string message
}
struct config {
	1: i64 id
	2: i32 tag
	3: list<i32> typeList
	4: bool boolean
	5: status status
	6: map<i64,string> failMap
	7: double fl
	8: double db
	9: binary bs
	10: timeRange nested
	11: list<timeRange> nestedTypeList
	12: map<string,timeRange> nestedTypeMap
}
struct timeRange {
	1: i64 start
	2: i64 end
}
struct reqOfTestGetApi {
	1: i64 a
	2: string b
}
struct respOfTestGetApi {
	1: i32 code
	2: string message
}

service aPIs {
	RespOfTestGetApi testGetApi (1: ReqOfTestGetApi req)
	RespOfTestPostApi testPostApi (1: ReqOfTestPostApi req)
}
