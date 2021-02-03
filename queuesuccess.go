package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
	"strconv"
	"strings"
	"sync"
)

type Result	 struct {
	Id 			int
	ATime		int		//匹配成功的时间
	Timeout		int		//多少秒后无人来取，即超时，更新用户状态，删除数据
	Teams		[]int		//该结果，有几个 队伍，因为每个队伍是一个集合，要用来索引
	PlayerIds	[]int
	GroupIds	[]int
	PushId		int
	Groups		[]Group
	//Players 	[]Player
	PushElement	PushElement
}

type QueueSuccess struct {
	Mutex 	sync.Mutex
	Rule 	Rule
	Push *Push
}

func NewQueueSuccess(rule Rule ,push *Push)*QueueSuccess{
	queueSuccess := new(QueueSuccess)
	queueSuccess.Rule = rule
	queueSuccess.Push = push
	return queueSuccess
}
func (queueSuccess *QueueSuccess)NewResult( )Result{
	result := Result{
		Id 		: queueSuccess.GetResultIncId( ),
		ATime	: zlib.GetNowTimeSecondToInt(),
		Timeout	: zlib.GetNowTimeSecondToInt() + queueSuccess.Rule.SuccessTimeout,
		Teams	: nil,
		GroupIds: nil,
		PlayerIds : nil,
	}
	return result
}
//成功类的整个：大前缀
func (queueSuccess *QueueSuccess) getRedisPrefixKey( )string{
	return redisPrefix + redisSeparation + "success"
}
//不同的匹配池(规则)，要有不同的KEY
func (queueSuccess *QueueSuccess) getRedisCatePrefixKey(  )string{
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
	res,_ := redis.String(redisDo("get",key))
	if res == ""{
		return
	}

	result = queueSuccess.strToStruct(res)

	if isIncludeGroupInfo == 1{
		var groups []Group
		for _,v :=range result.GroupIds{
			group := queueSuccess.getGroupElementById(v)
			groups = append(groups,group)
		}
		result.Groups = groups
	}

	if isIncludePushInfo == 1{
		result.PushElement = queueSuccess.Push.getById(result.PushId)
	}
	//fmt.Printf("%+v",result)
	//zlib.ExitPrint(11)
	return result
}
func (queueSuccess *QueueSuccess) getGroupElementById(  id int ) (group  Group){
	key := queueSuccess.getRedisKeyGroup(  id)
	res,_ := redis.String(redisDo("get",key))
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
	res,_ := redis.Int(redisDo("INCR",key))
	return res
}
//添加一条匹配成功记录
func  (queueSuccess *QueueSuccess)addOne(result Result,push *Push) {
	//zlib.ExitPrint(result)
	//添加元素超时信息
	key := queueSuccess.getRedisKeyTimeout()
	res,err := redisDo("zadd",redis.Args{}.Add(key).Add(result.Timeout).Add(result.Id)...)
	zlib.MyPrint("add timeout index rs : ",res,err)

	resultStr := queueSuccess.structToStr(result)
	pushId := push.addOnePush(result.Id,PushCategorySuccess,resultStr)
	result.PushId = pushId
	//添加一条元素
	key = queueSuccess.getRedisKeyResult(result.Id)
	str := queueSuccess.structToStr(result)
	//zlib.ExitPrint("str",str)
	res,err = redisDo("set",redis.Args{}.Add(key).Add(str) ...)
	zlib.MyPrint("add timeout index rs : ",res,err)



}
//一条匹配成功记录，要包括N条组信息，这是添加一个组的记录
func  (queueSuccess *QueueSuccess)addOneGroup( group Group){
	key := queueSuccess.getRedisKeyGroup(group.Id)
	content := GroupStructToStr(group)
	res,err := redisDo("set",redis.Args{}.Add(key).Add(content)...)
	zlib.MyPrint("add timeout index rs : ",res,err)
}

func (queueSuccess *QueueSuccess)  strToStruct(redisStr string )Result{
	strArr := strings.Split(redisStr,separation)
	Teams := strings.Split(strArr[3],IdsSeparation)
	PlayerIds := strings.Split(strArr[4],IdsSeparation)
	GroupIds := strings.Split(strArr[5],IdsSeparation)

	result := Result{
		Id:zlib.Atoi(strArr[0]),
		Timeout 	:	zlib.Atoi(strArr[1]),
		ATime 	:	zlib.Atoi(strArr[2]),
		Teams : zlib.ArrStringCoverArrInt(Teams),
		PlayerIds : zlib.ArrStringCoverArrInt(PlayerIds),
		GroupIds :  zlib.ArrStringCoverArrInt(GroupIds),
		PushId: zlib.Atoi(strArr[6]),
	}
	return result
}

func (queueSuccess *QueueSuccess) structToStr(result Result)string{
	//fmt.Printf("%+v",result)
	TeamsStr := zlib.ArrCoverStr(result.Teams,IdsSeparation)
	PlayerIds := zlib.ArrCoverStr(result.PlayerIds,IdsSeparation)
	GroupIds :=  zlib.ArrCoverStr(result.GroupIds,IdsSeparation)

	content :=
		strconv.Itoa( result.Id) + separation +
		strconv.Itoa( result.ATime) + separation +
		strconv.Itoa( result.Timeout) + separation +
			TeamsStr + separation +  PlayerIds + separation + GroupIds + separation +
			strconv.Itoa( result.PushId)

	return content
}


////删除所有：池里的报名组、玩家、索引等-有点暴力，尽量不用
func  (queueSuccess *QueueSuccess)   delAll(){
	key := queueSuccess.getRedisPrefixKey()
	redisDo("del",key)

	queueSuccess.Push.delAll()
}

func  (queueSuccess *QueueSuccess)   delOneRule(){
	log.Info(" delOneRule ")
	queueSuccess.delALLResult()
	queueSuccess.delALLTimeout()
	queueSuccess.delALLGroup()
	queueSuccess.Push.delOneRule()
}

//====================================================

//删除一条规则的，所有分组详细信息
func (queueSuccess *QueueSuccess)  delALLResult( ){
	prefix := queueSuccess.getRedisKeyResultPrefix()
	res,_ := redis.Strings( redisDo("keys",prefix + "*"  ))
	if len(res) == 0{
		log.Notice(" delALLResult by keys(*) : is empty")
		return
	}
	//zlib.ExitPrint(res,-200)
	for _,v := range res{
		res,_ := redis.Int(redisDo("del",v))
		zlib.MyPrint("del success element v :",res)
	}
}

//删除一条规则的，所有分组详细信息
func (queueSuccess *QueueSuccess)  delALLGroup( ){
	prefix := queueSuccess.getRedisKeyGroupPrefix()
	res,_ := redis.Strings( redisDo("keys",prefix + "*"  ))
	if len(res) == 0{
		log.Notice(" delALLGroup by keys(*) : is empty")
		return
	}
	//zlib.ExitPrint(res,-200)
	for _,v := range res{
		res,_ := redis.Int(redisDo("del",v))
		zlib.MyPrint("del success group element v :",res)
	}

	queueSuccess.Push.delOneRule()
}

func (queueSuccess *QueueSuccess)  delOneResult( id int){
	key := queueSuccess.getRedisKeyResult(id)
	res,_ := redis.Strings( redisDo("del",key ))
	log.Debug(" delOneRuleOneResult res",res)

	queueSuccess.Push.delOnePush(id)
}

//删除一条规则的，所有分组详细信息
func (queueSuccess *QueueSuccess)  delALLTimeout( ){
	key := queueSuccess.getRedisKeyTimeout()
	res,_ := redisDo("del",key)
	log.Debug(" delALLTimeout res",res)
}

func (queueSuccess *QueueSuccess)  delOneTimeout( id int){
	key := queueSuccess.getRedisKeyTimeout()
	res,_ :=  redisDo("ZREM",redis.Args{}.Add(key).Add(id) )
	log.Debug(" delOneTimeout res",res)
}

func  (queueSuccess *QueueSuccess) CheckTimeout(push *Push){
	keys := queueSuccess.getRedisKeyTimeout()
	now := zlib.GetNowTimeSecondToInt()

	inc := 1
	for {
		log.Info("loop inc : ",inc )
		if inc >= 2147483647{
			inc = 0
		}
		inc++

		res,err := redis.IntMap(  redisDo("ZREVRANGEBYSCORE",redis.Args{}.Add(keys).Add(now).Add("-inf").Add("WITHSCORES")...))
		log.Info("sign timeout group element total : ",len(res))
		if err != nil{
			log.Error("redis keys err :",err.Error())
			return
		}

		if len(res) == 0{
			log.Notice(" empty , no need process")
			mySleepSecond(1)
			continue
		}
		for resultId,_ := range res{
			queueSuccess.delOneTimeout(zlib.Atoi(resultId))
			//zlib.ExitPrint(-199)
		}
		mySleepSecond(1)
	}
}