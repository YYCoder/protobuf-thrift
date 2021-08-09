include "./common/admin.thrift";
include "./test.thrift";
enum otherEnum {
	otherEnumUnknown = 0
	unreviewed = 1
	online = 2
	rejected = 3
	offline = 4
}
struct config {
	1: i64 id
	2: i32 tag
	3: list<i32> typeList
	4: bool boolean
	5: admin.status status
	6: map<i64,string> failMap
	7: double fl
	8: double db
	9: binary bs
	10: test.timeRange nested
	11: list<test.timeRange> nestedTypeList
	12: map<string,test.timeRange> nestedTypeMap
}

service aPIs {
	Config testOther (1: Config req)
}
