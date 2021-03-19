package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
	"sync"
	"zlib"
)

type Result	 struct {
	Id 			int
	RuleId		int
	MatchCode	string
	ATime		int			//匹配成功的时间
	Timeout		int			//多少秒后无人来取，即超时，更新用户状态，删除数据
	Teams		[]int		//该结果，有几个 队伍，因为每个队伍是一个集合，要用来索引
	PlayerIds	[]int
	GroupIds	[]int
	PushId		int
	Groups		[]Group		//该结果下包含的两个组详细信息，属性挂载，用于push payload
	PushElement	PushElement	//该结果下推送的详细信息，属性挂载
}

type QueueSuccess struct {
	Mutex 	sync.Mutex
	Rule 	Rule
	Log *zlib.Log
	Gamematch	*Gamematch
}

func NewQueueSuccess(rule Rule , gamematch *Gamematch  )*QueueSuccess{
	queueSuccess := new(QueueSuccess)
	queueSuccess.Rule = rule
	queueSuccess.Gamematch = gamematch
	queueSuccess.Log = getRuleModuleLogInc(rule.CategoryKey,"success")
	return queueSuccess
}
func (queueSuccess *QueueSuccess)NewResult( )Result{
	result := Result{
		Id 		: queueSuccess.GetResultIncId( ),
		RuleId	: queueSuccess.Rule.Id,
		MatchCode: queueSuccess.Rule.CategoryKey,
		ATime	: zlib.GetNowTimeSecondToInt(),
		Timeout	: zlib.GetNowTimeSecondToInt() + queueSuccess.Rule.SuccessTimeout,
		Teams	: nil,
		GroupIds: nil,
		PlayerIds : nil,
		PushId	: 0 ,
		//Groups
		//PushElement
	}
	return result
}
//成功类的整个：大前缀
func (queueSuccess *QueueSuccess) getRedisPrefixKey( )string{
	return redisPrefix + redisSeparation + "success"
}
//不同的匹配池(规则)，要有不同的KEY
func (queueSuccess *QueueSuccess) getRedisCatePrefixKey(  )string{
	zlib.MyPrint(queueSuccess.Rule)
	return queueSuccess.getRedisPrefixKey()+redisSeparation + queueSuccess.Rule.CategoryKey
}
func (queueSuccess *QueueSuccess) getRedisKeyResultPrefix()string{
	return queueSuccess.getRedisCatePrefixKey() + redisSeparation + "result"
}

func (queueSuccess *QueueSuccess) getRedisKeyResult(successId int)string{
	return queueSuccess.getRedisKeyResultPrefix() + redisSeparation + strconv.Itoa(successId)
}

func (queueSuccess *QueueSuccess) getRedisKeyTimeout()string{
	return queueSuccess.getRedisCatePrefixKey() + redisSeparation + "timeout"
}


func  (queueSuccess *QueueSuccess)  getRedisResultIncKey()string{
	return queueSuccess.getRedisCatePrefixKey()  + redisSeparation + "inc_id"
}
//最简单的string：一个组的详细信息
func  (queueSuccess *QueueSuccess) getRedisKeyGroupPrefix( )string{
	return queueSuccess.getRedisCatePrefixKey() + redisSeparation + "group"
}
func  (queueSuccess *QueueSuccess) getRedisKeyGroup(groupId int)string{
	return queueSuccess.getRedisKeyGroupPrefix() + redisSeparation + strconv.Itoa(groupId)
}
//=================上面均是操作redis key==============

func (queueSuccess *QueueSuccess) GetResultById( id int ,isIncludeGroupInfo int ,isIncludePushInfo int ) (result Result ) {
	key := queueSuccess.getRedisKeyResult(id)
	res,_ := redis.String(myredis.RedisDo("get",key))
	if res == ""{
		return
	}

	result = queueSuccess.strToStruct(res)
	//fmt.Printf("%+v",result)
	//zlib.ExitPrint(result)
	if isIncludeGroupInfo == 1{
		var groups []Group
		for _,v :=range result.GroupIds{
			group := queueSuccess.getGroupElementById(v)
			groups = append(groups,group)
		}
		result.Groups = groups
	}

	if isIncludePushInfo == 1{
		//zlib.ExitPrint(result.RuleId)
		push := queueSuccess.Gamematch.getContainerPushByRuleId(result.RuleId)
		result.PushElement = push.getById(result.PushId)
	}
	//fmt.Printf("%+v",result)
	//zlib.ExitPrint(11)
	return result
}
func (queueSuccess *QueueSuccess) getGroupElementById(  id int ) (group  Group){
	key := queueSuccess.getRedisKeyGroup(  id)
	res,_ := redis.String(myredis.RedisDo("get",key))
	if res == ""{
		zlib.MyPrint(" getGroupElementById is empty!")
		return group
	}
	group = GroupStrToStruct (res)
	return group
}

//获取并生成一个自增GROUP-ID
func (queueSuccess *QueueSuccess) GetResultIncId(  )  int{
	key := queueSuccess.getRedisResultIncKey()
	res,_ := redis.Int(myredis.RedisDo("INCR",key))
	return res
}
//添加一条匹配成功记录
func  (queueSuccess *QueueSuccess)addOne( redisConn redis.Conn, result Result,push *Push) {
	mymetrics.IncNode("matchingSuccess")

	queueSuccess.Log.Info("func : addOne")
	//zlib.ExitPrint(result)
	//添加元素超时信息
	key := queueSuccess.getRedisKeyTimeout()
	res,err := myredis.Send(redisConn,"zadd",redis.Args{}.Add(key).Add(result.Timeout).Add(result.Id)...)
	//res,err := myredis.RedisDo("zadd",redis.Args{}.Add(key).Add(result.Timeout).Add(result.Id)...)
	queueSuccess.Log.Info("add timeout index rs : ",res,err)
	//这里注意下：pushId = 0
	resultStr := queueSuccess.structToStr(result)
	payload := strings.Replace(resultStr,separation,PayloadSeparation,-1)
	pushId := push.addOnePush(redisConn,result.Id,PushCategorySuccess,payload)
	result.PushId = pushId
	queueSuccess.Log.Info("addOnePush , newId : ",pushId)
	//添加一条元素
	key = queueSuccess.getRedisKeyResult(result.Id)
	//这里还得重新再  to str 一下，因为pushid 已经可以拿到了
	str := queueSuccess.structToStr(result)
	res,err = myredis.Send(redisConn,"set",redis.Args{}.Add(key).Add(str) ...)
	queueSuccess.Log.Info("add successResult rs : ",res,err)

}
//一条匹配成功记录，要包括N条组信息，这是添加一个组的记录
func  (queueSuccess *QueueSuccess)addOneGroup( redisConn redis.Conn,group Group){
	key := queueSuccess.getRedisKeyGroup(group.Id)
	content := GroupStructToStr(group)
	res,err := myredis.Send(redisConn,"set",redis.Args{}.Add(key).Add(content)...)
	//res,err := myredis.RedisDo("set",redis.Args{}.Add(key).Add(content)...)
	queueSuccess.Log.Info("addOneGroup  success ",res,err)
}

func  (queueSuccess *QueueSuccess)delOneGroup( redisConn redis.Conn,groupId int){
	key := queueSuccess.getRedisKeyGroup(groupId)
	res,err := myredis.Send(redisConn,"del",redis.Args{}.Add(key).Add(key)...)
	//res,err := myredis.RedisDo("del",redis.Args{}.Add(key).Add(key)...)
	mylog.Info("success delOneGroup : ",res,err)
	queueSuccess.Log.Info("delOneGroup",groupId,res,err)
}

func (queueSuccess *QueueSuccess)  strToStruct(redisStr string )Result{
	strArr := strings.Split(redisStr,separation)
	//zlib.MyPrint(strArr)
	//fmt.Printf("%+v",strArr)
	//zlib.ExitPrint(1111122223333)
	Teams := strings.Split(strArr[5],IdsSeparation)
	PlayerIds := strings.Split(strArr[6],IdsSeparation)
	GroupIds := strings.Split(strArr[7],IdsSeparation)
	//zlib.ExitPrint(Teams, PlayerIds, GroupIds)
	result := Result{
		Id:		zlib.Atoi(strArr[0]),
		RuleId:	zlib.Atoi(strArr[1]),
		MatchCode: strArr[2],
		ATime:	zlib.Atoi(strArr[3]),
		Timeout:zlib.Atoi(strArr[4]),
		Teams : zlib.ArrStringCoverArrInt(Teams),
		PlayerIds: zlib.ArrStringCoverArrInt(PlayerIds),
		GroupIds:  zlib.ArrStringCoverArrInt(GroupIds),
		PushId: zlib.Atoi(strArr[8]),
	}
	//fmt.Printf("%+v",result)
	return result
}

func (queueSuccess *QueueSuccess) structToStr(result Result)string{
	//fmt.Printf("%+v",result)
	//PushId		int
	//Groups		[]Group		//该结果下包含的两个组详细信息，属性挂载，用于push payload
	//PushElement	PushElement	//该结果下推送的详细信息，属性挂载

	TeamsStr := zlib.ArrCoverStr(result.Teams,IdsSeparation)
	PlayerIds := zlib.ArrCoverStr(result.PlayerIds,IdsSeparation)
	GroupIds :=  zlib.ArrCoverStr(result.GroupIds,IdsSeparation)

	content :=
		strconv.Itoa( result.Id) + separation +
		strconv.Itoa( result.RuleId)+separation +
		result.MatchCode + separation +
		strconv.Itoa( result.ATime) + separation +
		strconv.Itoa( result.Timeout) + separation +
		TeamsStr + separation +
		PlayerIds + separation +
		GroupIds + separation +
		strconv.Itoa( result.PushId) + separation
		//Groups		[]Group			这两个是挂载的，先不管
		//PushElement	PushElement		这两个是挂载的，先不管
	return content
}


//删除所有：池里的报名组、玩家、索引等-有点暴力，尽量不用
//func  (queueSuccess *QueueSuccess)   delAll(){
//	key := queueSuccess.getRedisPrefixKey()
//	myredis.RedisDo("del",key)
//
//	queueSuccess.Push.delAll()
//}
//
func  (queueSuccess *QueueSuccess)   delOneRule(){
	mylog.Info(" queueSuccess delOneRule ")
	keys := queueSuccess.getRedisCatePrefixKey() + "*"
	myredis.RedisDelAllByPrefix(keys)
	//queueSuccess.delALLResult()
	//queueSuccess.delALLTimeout()
	//queueSuccess.delALLGroup()
}

//====================================================

//删除一条规则的，所有分组详细信息
//func (queueSuccess *QueueSuccess)  delALLTimeout( ){
//	key := queueSuccess.getRedisKeyTimeout()
//	res,_ := myredis.RedisDo("del",key)
//	mylog.Debug(" delALLTimeout res",res)
//}
//删除一条规则的，所有分组详细信息
//func (queueSuccess *QueueSuccess)  delALLResult( ){
//	prefix := queueSuccess.getRedisKeyResultPrefix()
//	res,_ := redis.Strings( myredis.RedisDo("keys",prefix + "*"  ))
//	if len(res) == 0{
//		mylog.Notice(" delALLResult by keys(*) : is empty")
//		return
//	}
//	//zlib.ExitPrint(res,-200)
//	for _,v := range res{
//		res,_ := redis.Int(myredis.RedisDo("del",v))
//		zlib.MyPrint("del success element v :",res)
//	}
//}

////删除一条规则的，所有分组详细信息
//func (queueSuccess *QueueSuccess)  delALLGroup( ){
//	prefix := queueSuccess.getRedisKeyGroupPrefix()
//	res,_ := redis.Strings( myredis.RedisDo("keys",prefix + "*"  ))
//	if len(res) == 0{
//		mylog.Notice(" delALLGroup by keys(*) : is empty")
//		return
//	}
//	//zlib.ExitPrint(res,-200)
//	for _,v := range res{
//		res,_ := redis.Int(myredis.RedisDo("del",v))
//		zlib.MyPrint("del success group element v :",res)
//	}
//
//	queueSuccess.Push.delOneRule()
//}

func (queueSuccess *QueueSuccess)  delOneResult(redisConn redis.Conn, id int,isIncludeGroupInfo int ,isIncludePushInfo int,isIncludeTimeout int,isIncludePlayerStatus int){
	mylog.Info("delOneResult id :",id,isIncludeGroupInfo,isIncludePushInfo,isIncludeTimeout)
	element := queueSuccess.GetResultById(id,isIncludeGroupInfo,isIncludePushInfo)
	key := queueSuccess.getRedisKeyResult(id)
	res,err :=   myredis.Send(redisConn,"del",redis.Args{}.Add(key)... )
	//res,err :=   myredis.RedisDo("del",redis.Args{}.Add(key)... )

	mylog.Debug(" delOneRuleOneResult res",res,err)
	queueSuccess.Log.Info(" delOneRuleOneResult res",res,err)

	if isIncludePushInfo == 1{
		push := queueSuccess.Gamematch.getContainerPushByRuleId(element.RuleId)
		queueSuccess.Log.Info("delOnePush",element.PushId)
		push.delOnePush(redisConn,element.PushId)
	}

	if isIncludeTimeout == 1{
		queueSuccess.Log.Info("delOneTimeout",id)
		queueSuccess.delOneTimeout(redisConn,id)
	}

	if isIncludeGroupInfo == 1{
		for _,groupId :=range element.GroupIds{
			queueSuccess.delOneGroup(redisConn,groupId)
		}
	}

	if isIncludeTimeout == 1{
		for _,playerId := range element.PlayerIds{
			queueSuccess.Log.Info("playerStatus.delOneById ",playerId)
			playerStatus.delOneById(redisConn,playerId)
		}
	}

}

func (queueSuccess *QueueSuccess)  delOneTimeout(redisConn  redis.Conn, id int){
	key := queueSuccess.getRedisKeyTimeout()
	res,_ :=  myredis.Send(redisConn,"ZREM",redis.Args{}.Add(key).Add(id)... )
	//res,_ :=  myredis.RedisDo("ZREM",redis.Args{}.Add(key).Add(id)... )
	mylog.Info(" success delOneTimeout res",res)
}
func  (queueSuccess *QueueSuccess) GetAllTimeoutCnt()int{
	key := queueSuccess.getRedisKeyTimeout()
	res,_ := redis.Int(  myredis.RedisDo("ZCOUNT",redis.Args{}.Add(key).Add("-inf").Add("+inf")...))
	return res
}
func  (queueSuccess *QueueSuccess) CheckTimeout(){
	keys := queueSuccess.getRedisKeyTimeout()
	push := queueSuccess.Gamematch.getContainerPushByRuleId(queueSuccess.Rule.Id)
	//inc := 1
	//for {
	//	mylog.Info("loop inc : ",inc )
	//	queueSuccess.Log.Info("loop inc : ",inc )
	//	if inc >= 2147483647{
	//		inc = 0
	//	}
	//	inc++
		redisConnFD := myredis.GetNewConnFromPool()
		defer redisConnFD.Close()

		now := zlib.GetNowTimeSecondToInt()
		res,err := redis.IntMap(  myredis.RedisDo("ZREVRANGEBYSCORE",redis.Args{}.Add(keys).Add(now).Add("-inf").Add("WITHSCORES")...))
		mylog.Info("queueSuccess timeout group element total : ",len(res))
		queueSuccess.Log.Info("queueSuccess timeout group element total : ",len(res))
		if err != nil{
			mylog.Error("redis keys err :",err.Error())
			return
		}
		//zlib.ExitPrint(123123)
		if len(res) == 0{
			mylog.Notice(" empty , no need process")
			queueSuccess.Log.Info(" empty , no need process")
			//myGosched("success CheckTimeout")
			//mySleepSecond(1," success CheckTimeout ")
			return
		}
		for resultId,_ := range res{
			myredis.Send(redisConnFD,"multi")

			resultIdInt := zlib.Atoi(resultId)
			element := queueSuccess.GetResultById(resultIdInt,0,0)
			queueSuccess.Log.Info("GetResultById",resultIdInt,element)
			//fmt.Printf("%+v",element)
			//zlib.ExitPrint(element)
			queueSuccess.delOneResult(redisConnFD,resultIdInt,1,1,1,1)
			payload := queueSuccess.structToStr(element)
			payload = strings.Replace(payload,separation,PayloadSeparation,-1)
			push.addOnePush(redisConnFD,resultIdInt,PushCategorySuccessTimeout, payload)

			myredis.ConnDo(redisConnFD,"exec")
			//zlib.ExitPrint(111111111)
		}
		//myGosched("success CheckTimeout")
	//	mySleepSecond(1," success CheckTimeout ")
	//}
}