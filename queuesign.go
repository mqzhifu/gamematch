package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
	"strconv"
	"strings"
	"sync"
)

type QueueSign struct {
	//计算匹配结果时，要加锁
	//1、阻塞住报名，2、阻塞其它匹配的协程
	Mutex 	sync.Mutex
	Rule 	Rule
}

func NewQueueSign(rule Rule)*QueueSign{
	queueSign := new(QueueSign)
	queueSign.Rule = rule
	return queueSign
}

//type QueueSignSignTimeoutElement struct {
//	PlayerId	int
//	SignTime		int
//	Flag 		int		//1超时 2状态变更
//	Memo 		string	//备注
//	id		int
//}

//报名类的整个：大前缀
func (queueSign *QueueSign) getRedisPrefixKey( )string{
	return redisPrefix + redisSeparation + "sign"
}
//不同的匹配池(规则)，要有不同的KEY
func (queueSign *QueueSign) getRedisCatePrefixKey(  )string{
	return queueSign.getRedisPrefixKey()+redisSeparation + queueSign.Rule.CategoryKey
}
//有序集合：组索引，所有报名的组，都在这个集合中，weight => id
func (queueSign *QueueSign) getRedisKeyWeight()string{
	return queueSign.getRedisCatePrefixKey() + redisSeparation + "group_weight"
}
//有序集合：组的人数索引，每个规则的池，允许N人成组，其中，每个组里有多少个人，就是这个索引
func (queueSign *QueueSign) getRedisKeyPersonIndexPrefix()string{
	return queueSign.getRedisCatePrefixKey() + redisSeparation + "group_person"
}
//组人数=>id
func (queueSign *QueueSign) getRedisKeyPersonIndex(personNum int)string{
	return queueSign.getRedisKeyPersonIndexPrefix() + redisSeparation + strconv.Itoa(personNum)
}
//最简单的string：一个组的详细信息
func (queueSign *QueueSign) getRedisKeyGroupElementPrefix(  )string{
	return queueSign.getRedisCatePrefixKey( ) + redisSeparation + "element"
}
func (queueSign *QueueSign) getRedisKeyGroupElement( id int)string{
	return queueSign.getRedisKeyGroupElementPrefix( ) + redisSeparation + strconv.Itoa(id)
}
//有序集合：一个小组，包含的所有玩家ID
func (queueSign *QueueSign) getRedisKeyGroupPlayer( )string{
	return queueSign.getRedisCatePrefixKey( ) + redisSeparation + "player"
}

//组的超时索引,有序集合
func (queueSign *QueueSign) getRedisKeyGroupSignTimeout( )string{
	return queueSign.getRedisCatePrefixKey( )  + redisSeparation + "timeout"
}
//===============================以上是 redis key 相关============================

//获取当前所有，已报名的，所有组，总数
func (queueSign *QueueSign) getAllGroupsWeightCnt(  )  int{
	return queueSign.getGroupsWeightCnt("-inf","+inf")
}
//获取当前所有，已报名的，所有组，总数
func (queueSign *QueueSign) getGroupsWeightCnt( rangeStart string,rangeEnd string  )  int{
	key := queueSign.getRedisKeyWeight()
	res,err := redis.Int(myredis.RedisDo("ZCOUNT",redis.Args{}.Add(key).Add(rangeStart).Add(rangeEnd)...))
	if err !=nil{
		zlib.ExitPrint("ZCOUNT err",err.Error())
	}
	return res
}
//获取当前所有，已报名的，组，总数
func (queueSign *QueueSign) getAllGroupPersonCnt(   ) map[int]int {
	groupPersonNum := make(map[int]int)
	for i:=1;i<=queueSign.Rule.GroupPersonMax;i++{
		groupPersonNum[i] = queueSign.getAllGroupPersonIndexCnt(i)
	}
	return groupPersonNum
}
func (queueSign *QueueSign) getAllGroupPersonIndexCnt(  personNum int )  int{
	return queueSign.getGroupPersonIndexCnt(personNum,"-inf","+inf")
}
//获取当前所有，已报名的，组，总数
func (queueSign *QueueSign) getGroupPersonIndexCnt(  personNum int ,rangeStart string,rangeEnd string)  int{
	key := queueSign.getRedisKeyPersonIndex( personNum)
	res,err := redis.Int(myredis.RedisDo("ZCOUNT",redis.Args{}.Add(key).Add(rangeStart).Add(rangeEnd)...))
	if err !=nil{
		zlib.ExitPrint("ZCOUNT err",err.Error())
	}
	return res
}
//获取当前所有，已报名的，玩家，总数
func (queueSign *QueueSign) getAllPlayersCnt(   )  int{
	return queueSign.getPlayersCnt("-inf","+inf")
}
//获取当前所有，已报名的，玩家，总数,这个是基于groupId
func (queueSign *QueueSign) getPlayersCnt(  rangeStart string,rangeEnd string )  int{
	key := queueSign.getRedisKeyGroupPlayer( )
	res,err := redis.Int(myredis.RedisDo("ZCOUNT",redis.Args{}.Add(key).Add(rangeStart).Add(rangeEnd)...))
	if err !=nil{
		zlib.ExitPrint("ZCOUNT err",err.Error())
	}
	return res
}

//获取当前所有，已报名的，玩家，总数,这个是基于 权重
func (queueSign *QueueSign) getPlayersCntTotalByWeight(  rangeStart string,rangeEnd string )  int{
	total := 0
	for i:=1;i<=queueSign.Rule.GroupPersonMax;i++{
		oneCnt := queueSign.getGroupPersonIndexCnt(i,rangeStart,rangeEnd)
		total += oneCnt * i

	}
	return total
}

func (queueSign *QueueSign) getPlayersCntByWeight(  rangeStart string,rangeEnd string )  map[int]int{
	groupPersonNum := make(map[int]int)
	for i:=1;i<=queueSign.Rule.GroupPersonMax;i++{
		groupPersonNum[i] = queueSign.getGroupPersonIndexCnt(i,rangeStart,rangeEnd)
	}
	return groupPersonNum
}

func (queueSign *QueueSign) cancel(  playerId int ) {
	queueSign.delOneByPlayerId(playerId)
}



//============================以上是 各种获取总数===============================


func (queueSign *QueueSign) getGroupElementById(  id int ) (group  Group){
	key := queueSign.getRedisKeyGroupElement(  id)
	res,_ := redis.String(myredis.RedisDo("get",key))
	if res == ""{
		mylog.Notice(" getGroupElementById is empty!")
		return group
	}
	group = GroupStrToStruct (res)
	return group
}

func (queueSign *QueueSign) getPlayerIdById( playerId int )   int {
	key := queueSign.getRedisKeyGroupPlayer( )
	memberIndex,_ := redis.Int(myredis.RedisDo("zrank",key,playerId))
	if memberIndex == 0{
		zlib.MyPrint(" getGroupElementById is empty!")
	}
	id,_ := redis.Int( myredis.RedisDo("zrange",redis.Args{}.Add(key).Add(memberIndex).Add(memberIndex).Add("WITHSCORES")...))
	//group := queueSign.strToStruct(res)
	return id
}
//============================以上都是根据ID，获取一条===============================
func (queueSign *QueueSign) getGroupPersonIndexList( personNum int, rangeStart string,rangeEnd string,limitOffset int ,limitCnt int,isDel bool)(ids map[int]int){
	key := queueSign.getRedisKeyPersonIndex(  personNum)
	//这里有个问题，person=>id,person是重复的，如果带着分值一并返回，MAP会有重复值的情况
	argc := redis.Args{}.Add(key).Add(rangeEnd).Add(rangeStart).Add("limit").Add(limitOffset).Add(limitCnt)
	res,err := redis.Ints(myredis.RedisDo("ZREVRANGEBYSCORE", argc...))
	if err != nil{
		zlib.MyPrint("getPersonRangeList err",err.Error())
	}
	if len(res) <= 0{
		return ids
	}

	rs := zlib.ArrCovertMap(res)

	if isDel{
		for _,v := range res{
			queueSign.delOneRuleOneGroupIndex(v)
		}
	}

	return rs
}

//func (queueSign *QueueSign) getPersonRangeList( personNum int ,scoreMin int,scoreMax float32,limitOffset int ,limitCnt int)(ids map[int]int){
//	key := queueSign.getRedisKeyPersonIndex( personNum)
//	res,err := redis.IntMap(redisDo("ZREVRANGEBYSCORE", redis.Args{}.Add(key).Add(scoreMax).Add(scoreMin).Add("limit").Add(limitOffset).Add(limitCnt)...))
//	if err != nil{
//		zlib.MyPrint("getPersonRangeList err",err.Error())
//	}
//	if len(res) <= 0{
//		return ids
//	}
//
//	inc := 0
//	for _,v := range res{
//		ids[inc]  = v
//		inc++
//	}
//	return ids
//}

//============================以上都是范围性的取值===============================

//报名
func (queueSign *QueueSign) AddOne(group Group){
	mylog.Info(" add sign one group")
	group.SignTime = zlib.GetNowTimeSecondToInt()

	groupIndexKey := queueSign.getRedisKeyWeight( )
	res,err := myredis.RedisDo("zadd",redis.Args{}.Add(groupIndexKey).Add(group.Weight).Add(group.Id)...)
	mylog.Debug("add GroupWeightIndex rs : ",res,err)

	PersonIndexKey := queueSign.getRedisKeyPersonIndex(  group.Person)
	res,err = myredis.RedisDo("zadd",redis.Args{}.Add(PersonIndexKey).Add(group.Weight).Add(group.Id)...)
	mylog.Debug("add GroupPersonIndex ( ",group.Person," ) rs : ",res,err)

	groupSignTimeoutKey := queueSign.getRedisKeyGroupSignTimeout()
	res,err = myredis.RedisDo("zadd",redis.Args{}.Add(groupSignTimeoutKey).Add(group.SignTimeout).Add(group.Id)...)
	mylog.Debug("add GroupSignTimeout rs : ",res,err)

	groupElementRedisKey := queueSign.getRedisKeyGroupElement(group.Id)
	content := GroupStructToStr(group)

	groupPlayersKey := queueSign.getRedisKeyGroupPlayer()
	for _,v := range group.Players{
		res,err = myredis.RedisDo("zadd",redis.Args{}.Add(groupPlayersKey).Add(group.Id).Add(v.Id)...)
		zlib.MyPrint("add player rs : ",res,err)
	}

	res,err = myredis.RedisDo("set",redis.Args{}.Add(groupElementRedisKey).Add(content)...)
	mylog.Debug("add groupElement rs : ",res,err)

	mylog.Info("add sign one group finish")
}

//==============以下均是 删除操作======================================


//删除所有：池里的报名组、玩家、索引等-有点暴力，尽量不用
func  (queueSign *QueueSign)  delAll(){
	key := queueSign.getRedisPrefixKey()
	myredis.RedisDo("del",key)
}
//删除一条规则的所有匹配信息
func  (queueSign *QueueSign)  delOneRule(){
	mylog.Info(" delOneRule : ",queueSign.getRedisCatePrefixKey())
	queueSign.delOneRuleALLGroupElement()
	queueSign.delOneRuleALLPersonIndex()
	queueSign.delOneRuleAllGroupSignTimeout()
	queueSign.delOneRuleALLPlayers()
	queueSign.delOneRuleALLWeight()
}
//====================================================

//删除一条规则的，所有玩家索引
func  (queueSign *QueueSign)  delOneRuleALLPlayers( ){
	key := queueSign.getRedisKeyGroupPlayer()
	res,_ := redis.Int(myredis.RedisDo("del",key))
	mylog.Debug("delOneRuleALLPlayers : ",res)
}
//删除一条规则的，所有权重索引
func  (queueSign *QueueSign)  delOneRuleALLWeight( ){
	key := queueSign.getRedisKeyWeight()
	res,_ := redis.Int(myredis.RedisDo("del",key))
	mylog.Debug("delOneRuleALLPlayers : ",res)
}
//删除一条规则的，所有人数分组索引
func  (queueSign *QueueSign)  delOneRuleALLPersonIndex( ){
	for i:=1 ; i <= RuleGroupPersonMax;i++{
		queueSign.delOneRuleOnePersonIndex(i)
	}
}

//删除一条规则的，所有分组详细信息
func  (queueSign *QueueSign)  delOneRuleALLGroupElement( ){
	prefix := queueSign.getRedisKeyGroupElementPrefix()
	res,_ := redis.Strings( myredis.RedisDo("keys",prefix + "*"  ))
	if len(res) == 0{
		mylog.Notice(" GroupElement by keys(*) : is empty")
		return
	}
	//zlib.ExitPrint(res,-200)
	for _,v := range res{
		res,_ := redis.Int(myredis.RedisDo("del",v))
		mylog.Debug("del group element v :",res)
	}
}

//删除一条规则的，所有人组超时索引
func  (queueSign *QueueSign)  delOneRuleAllGroupSignTimeout( ){
	key := queueSign.getRedisKeyGroupSignTimeout()
	res,_ := redis.Int(myredis.RedisDo("del",key))
	mylog.Debug("delOneRuleALLPlayers : ",res)
}
//删除一条规则的，某一人数各类的，所有人数分组索引
func  (queueSign *QueueSign)  delOneRuleOnePersonIndex( personNum int){
	key := queueSign.getRedisKeyPersonIndex(personNum)
	res,_ := redis.Int(myredis.RedisDo("del",key))
	mylog.Debug("delOneRuleALLPlayers : ",res)
}
//====================================================

func  (queueSign *QueueSign)  delOneRuleOnePersonIndexById( personNum int,id int){
	key := queueSign.getRedisKeyPersonIndex(personNum)
	res, _ := myredis.RedisDo("ZREM",redis.Args{}.Add(key).Add(id)...)
	mylog.Debug("delOne PersonIndexById : ",res)
}

//删除一个组的所有玩家信息
func  (queueSign *QueueSign) delOneRuleOneGroupPlayers(  id int){
	key := queueSign.getRedisKeyGroupPlayer()
	res,_ := myredis.RedisDo("ZREMRANGEBYSCORE",redis.Args{}.Add(key).Add(id).Add(id)...)
	mylog.Debug("delOne GroupPlayers : ",res)
}
//删除一条规则的一个组的详细信息
func  (queueSign *QueueSign) delOneRuleOneGroupSignTimeout( id int){
	key := queueSign.getRedisKeyGroupSignTimeout()
	res, _ := myredis.RedisDo("ZREM",redis.Args{}.Add(key).Add(id)...)
	mylog.Debug("delOne GroupSignTimeoutById : ",res)
}
//删除一条规则的权限分组索引
func  (queueSign *QueueSign)  delOneRuleOneWeight( id int){
	key := queueSign.getRedisKeyWeight()
	res,_ := myredis.RedisDo("ZREM",redis.Args{}.Add(key).Add(id)...)
	mylog.Debug("delOneWeight : ",res)
}
//删除一条规则的一个组的详细信息
func  (queueSign *QueueSign) delOneRuleOneGroupElement( id int){
	key := queueSign.getRedisKeyGroupElement(id)
	res,_ := myredis.RedisDo("del",key)
	mylog.Debug("delOneGroupElement : ",res)
}
//删除一个玩家的报名
func (queueSign *QueueSign) delOneByPlayerId( playerId int){
	id := queueSign.getPlayerIdById(playerId)
	queueSign.delOneRuleOneGroup(id,1)
	//这里还要更新当前组所有用户的，状态信息
}

//====================================================
//删除一个组的，所有索引信息,反向看：除了小组基础信息外，其余均删除
func  (queueSign *QueueSign)  delOneGroupIndex(groupId int){
	zlib.MyPrint(" delOneGroupIndex : ",queueSign.getRedisCatePrefixKey())
	group := queueSign.getGroupElementById(groupId)

	queueSign.delOneRuleOneWeight(groupId)
	queueSign.delOneRuleOnePersonIndexById(group.Person,groupId)
	queueSign.delOneRuleOneGroupPlayers(groupId)
	queueSign.delOneRuleOneGroupSignTimeout(groupId)
}
//添加一个组的，所有索引信息,反向看：除了小组基础信息外，其余均删除
func  (queueSign *QueueSign)  addOneGroupIndex(groupId int){
	zlib.MyPrint(" delOneGroupIndex : ",queueSign.getRedisCatePrefixKey())
	group := queueSign.getGroupElementById(groupId)

	groupIndexKey := queueSign.getRedisKeyWeight( )
	res,err := myredis.RedisDo("zadd",redis.Args{}.Add(groupIndexKey).Add(group.Weight).Add(group.Id)...)
	mylog.Debug("add GroupWeightIndex rs : ",res,err)

	PersonIndexKey := queueSign.getRedisKeyPersonIndex(  group.Person)
	res,err = myredis.RedisDo("zadd",redis.Args{}.Add(PersonIndexKey).Add(group.Weight).Add(group.Id)...)
	mylog.Debug("add GroupPersonIndex ( ",group.Person," ) rs : ",res,err)
}
//删除一个组
func (queueSign *QueueSign) delOneRuleOneGroup( id int,isDelPlayerStatus int){
	mylog.Info("action : delOneRuleOneGroup id:" , id)
	group := queueSign.getGroupElementById(id)
	mylog.Debug(group)

	queueSign.delOneRuleOneGroupElement(id)
	queueSign.delOneRuleOneWeight(id)
	queueSign.delOneRuleOnePersonIndexById(group.Person,id)
	queueSign.delOneRuleOneGroupPlayers(id)
	queueSign.delOneRuleOneGroupSignTimeout(id)

	if isDelPlayerStatus == 1{
		for _,v := range group.Players{
			//playerStatus.upInfo(v)
			playerStatus.delOneById(v.Id)
		}
	}

}

//临时，删除一个组的权限+人数的索引
func (queueSign *QueueSign) delOneRuleOneGroupIndex( id int){
	mylog.Info("action : delOneRuleOneGroupIndex id:" , id)
	group := queueSign.getGroupElementById(id)
	mylog.Debug(group)

	queueSign.delOneRuleOneWeight(id)
	queueSign.delOneRuleOnePersonIndexById(group.Person,id)
}


func  (queueSign *QueueSign) CheckTimeout( push *Push){
	mylog.Info(" one rule CheckSignTimeout , ruleId : ",queueSign.Rule.Id)

	keys := queueSign.getRedisKeyGroupSignTimeout()
	now := zlib.GetNowTimeSecondToInt()

	inc := 1
	for {
		mylog.Info("loop inc : ",inc )
		if inc >= 2147483647{
			inc = 0
		}
		inc++

		res,err := redis.IntMap(  myredis.RedisDo("ZREVRANGEBYSCORE",redis.Args{}.Add(keys).Add(now).Add("-inf").Add("WITHSCORES")...))
		mylog.Info("sign timeout group element total : ",len(res))
		if err != nil{
			mylog.Error("redis keys err :",err.Error())
			return
		}

		if len(res) == 0{
			mylog.Notice(" empty , no need process")
			mySleepSecond(1)
			continue
		}
		for groupIdStr,_ := range res{
			groupId := zlib.Atoi(groupIdStr)
			group := queueSign.getGroupElementById(groupId)
			payload := GroupStructToStr (group)
			payload = strings.Replace(payload,separation,PayloadSeparation,-1)
			push.addOnePush(groupId,PushCategorySignTimeout,payload)
			queueSign.delOneRuleOneGroup(groupId,1)

			zlib.ExitPrint(-199)
		}
		mySleepSecond(1)
	}
}