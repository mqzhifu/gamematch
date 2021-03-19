package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
	"sync"
	"zlib"
)

type QueueSign struct {
	//计算匹配结果时，要加锁
	//1、阻塞住报名，2、阻塞其它匹配的协程
	Mutex 	sync.Mutex
	Rule 	Rule
	Log 	*zlib.Log
	Gamematch	*Gamematch
}

func NewQueueSign(rule Rule , gamematch *Gamematch)*QueueSign{
	queueSign := new(QueueSign)
	queueSign.Rule = rule
	queueSign.Log = getRuleModuleLogInc(rule.CategoryKey,"sign")
	queueSign.Gamematch = gamematch
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

func (queueSign *QueueSign) cancelByPlayerId(  playerId int ) {
	//queueSign.delOneByPlayerId(playerId)
}

func (queueSign *QueueSign) cancelByGroupId(  groupId int ) error {
	group := queueSign.getGroupElementById(groupId)
	mylog.Debug(group)
	//这里是偷懒了，判断是否为空，按说应该返回2个参数，但这个方法调用 地方多，先这样
	if group.Id == 0 || group.Person == 0 || len(group.Players) == 0{
		msg := make(map[int]string)
		msg[0] = strconv.Itoa(groupId)
		return myerr.NewErrorCodeReplace(750,msg)
	}
	redisConn := myredis.GetNewConnFromPool()
	defer redisConn.Close()

	for _,player := range group.Players{
		playerElement,isEmpty := playerStatus.GetById(player.Id)
		if isEmpty == 0{
			if playerElement.Status != PlayerStatusSign{
				msg := make(map[int]string)
				msg[0] = strconv.Itoa(playerElement.Status )
				return myerr.NewErrorCodeReplace(623,msg)
			}
		}

	}

	queueSign.delOneRuleOneGroup(redisConn,groupId,1)
	return nil
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

func (queueSign *QueueSign) getGroupIdByPlayerId( playerId int )   int {
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
		redisConn := myredis.GetNewConnFromPool()
		defer redisConn.Close()
		for _,groupId := range res{
			queueSign.delOneRuleOneGroupIndex(redisConn,groupId,personNum)
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
func (queueSign *QueueSign) AddOne(group Group,connFd redis.Conn){
	queueSign.Log.Debug("start :  add sign one group")
	mylog.Info("")
	group.SignTime = zlib.GetNowTimeSecondToInt()

	groupIndexKey := queueSign.getRedisKeyWeight( )
	res,err := myredis.Send(connFd,"zadd",redis.Args{}.Add(groupIndexKey).Add(group.Weight).Add(group.Id)...)
	//res,err := myredis.RedisDo("zadd",redis.Args{}.Add(groupIndexKey).Add(group.Weight).Add(group.Id)...)

	queueSign.Log.Debug("add GroupWeightIndex rs : ",res,err)

	PersonIndexKey := queueSign.getRedisKeyPersonIndex(  group.Person)
	res,err = myredis.Send(connFd,"zadd",redis.Args{}.Add(PersonIndexKey).Add(group.Weight).Add(group.Id)...)
	//res,err = myredis.RedisDo("zadd",redis.Args{}.Add(PersonIndexKey).Add(group.Weight).Add(group.Id)...)
	queueSign.Log.Debug("add GroupPersonIndex ( ",group.Person," ) rs : ",res,err)

	groupSignTimeoutKey := queueSign.getRedisKeyGroupSignTimeout()
	res,err = myredis.Send(connFd,"zadd",redis.Args{}.Add(groupSignTimeoutKey).Add(group.SignTimeout).Add(group.Id)...)
	//res,err = myredis.RedisDo("zadd",redis.Args{}.Add(groupSignTimeoutKey).Add(group.SignTimeout).Add(group.Id)...)
	queueSign.Log.Debug("add GroupSignTimeout rs : ",res,err)

	groupElementRedisKey := queueSign.getRedisKeyGroupElement(group.Id)
	content := GroupStructToStr(group)

	//zlib.ExitPrint(1111)
	groupPlayersKey := queueSign.getRedisKeyGroupPlayer()
	for _,v := range group.Players{
		res,err = myredis.Send(connFd,"zadd",redis.Args{}.Add(groupPlayersKey).Add(group.Id).Add(v.Id)...)
		//res,err = myredis.RedisDo("zadd",redis.Args{}.Add(groupPlayersKey).Add(group.Id).Add(v.Id)...)
		queueSign.Log.Debug("add player rs : ",res,err)
	}

	res,err = myredis.Send(connFd,"set",redis.Args{}.Add(groupElementRedisKey).Add(content)...)
	//res,err = myredis.RedisDo("set",redis.Args{}.Add(groupElementRedisKey).Add(content)...)
	queueSign.Log.Debug("add groupElement rs : ",res,err)
	queueSign.Log.Debug("finish :  add sign one group")
}

//==============以下均是 删除操作======================================


////删除 所有Rule：池里的报名组、玩家、索引等-有点暴力，尽量不用
//func  (queueSign *QueueSign)  delAll(){
//	key := queueSign.getRedisPrefixKey()
//	myredis.RedisDo("del",key)
//}
//删除一条规则的所有匹配信息
func  (queueSign *QueueSign)  delOneRule(){
	mylog.Info(" queueSign delOneRule : ")
	keys := queueSign.getRedisCatePrefixKey() + "*"
	myredis.RedisDelAllByPrefix(keys)

	//queueSign.delOneRuleALLGroupElement()
	//queueSign.delOneRuleALLPersonIndex()
	//queueSign.delOneRuleAllGroupSignTimeout()
	//queueSign.delOneRuleALLPlayers()
	//queueSign.delOneRuleALLWeight()
}
//====================================================

////删除一条规则的，所有玩家索引
//func  (queueSign *QueueSign)  delOneRuleALLPlayers( ){
//	key := queueSign.getRedisKeyGroupPlayer()
//	res,_ := redis.Int(myredis.RedisDo("del",key))
//	mylog.Debug("delOneRuleALLPlayers : ",res)
//}
////删除一条规则的，所有权重索引
//func  (queueSign *QueueSign)  delOneRuleALLWeight( ){
//	key := queueSign.getRedisKeyWeight()
//	res,_ := redis.Int(myredis.RedisDo("del",key))
//	mylog.Debug("delOneRuleALLPlayers : ",res)
//}
//删除一条规则的，所有人数分组索引
//func  (queueSign *QueueSign)  delOneRuleALLPersonIndex( ){
//	for i:=1 ; i <= RuleGroupPersonMax;i++{
//		queueSign.delOneRuleOnePersonIndex(i)
//	}
//}
//
////删除一条规则的，所有分组详细信息
//func  (queueSign *QueueSign)  delOneRuleALLGroupElement( ){
//	prefix := queueSign.getRedisKeyGroupElementPrefix()
//	res,_ := redis.Strings( myredis.RedisDo("keys",prefix + "*"  ))
//	if len(res) == 0{
//		mylog.Notice(" GroupElement by keys(*) : is empty")
//		return
//	}
//	//zlib.ExitPrint(res,-200)
//	for _,v := range res{
//		res,_ := redis.Int(myredis.RedisDo("del",v))
//		mylog.Debug("del group element v :",res)
//	}
//}

////删除一条规则的，所有人组超时索引
//func  (queueSign *QueueSign)  delOneRuleAllGroupSignTimeout( ){
//	key := queueSign.getRedisKeyGroupSignTimeout()
//	res,_ := redis.Int(myredis.RedisDo("del",key))
//	mylog.Debug("delOneRuleALLPlayers : ",res)
//}

//删除一条规则的，某一人数各类的，所有人数分组索引
func  (queueSign *QueueSign)  delOneRuleOnePersonIndex( personNum int){
	key := queueSign.getRedisKeyPersonIndex(personNum)
	res,_ := redis.Int(myredis.RedisDo("del",key))
	mylog.Debug("delOneRuleALLPlayers : ",res)
}
//====================================================

func  (queueSign *QueueSign)  delOneRuleOnePersonIndexById( redisConn redis.Conn,personNum int,id int){
	key := queueSign.getRedisKeyPersonIndex(personNum)
	res, _ := myredis.Send(redisConn,"ZREM",redis.Args{}.Add(key).Add(id)...)
	//res, _ := myredis.RedisDo("ZREM",redis.Args{}.Add(key).Add(id)...)
	mylog.Debug("delOne PersonIndexById : ",res)
	queueSign.Log.Info("delOneRuleOnePersonIndexById",personNum,id)
}

//删除一个组的所有玩家信息
func  (queueSign *QueueSign) delOneRuleOneGroupPlayers( redisConn redis.Conn, id int){
	key := queueSign.getRedisKeyGroupPlayer()
	res,_ := myredis.Send(redisConn,"ZREMRANGEBYSCORE",redis.Args{}.Add(key).Add(id).Add(id)...)
	//res,_ := myredis.RedisDo("ZREMRANGEBYSCORE",redis.Args{}.Add(key).Add(id).Add(id)...)
	mylog.Debug("delOne GroupPlayers : ",res)
	queueSign.Log.Info("delOneRuleOneGroupPlayers:",id)
}
//删除一条规则的一个组的详细信息
func  (queueSign *QueueSign) delOneRuleOneGroupSignTimeout(redisConn redis.Conn, id int){
	key := queueSign.getRedisKeyGroupSignTimeout()
	res, _ := myredis.Send(redisConn,"ZREM",redis.Args{}.Add(key).Add(id)...)
	//res, _ := myredis.RedisDo("ZREM",redis.Args{}.Add(key).Add(id)...)
	mylog.Debug("delOneRuleOneGroupSignTimeout : ",res)
	queueSign.Log.Info("delOneRuleOneGroupSignTimeout ",id)
}
//删除一条规则的权限分组索引
func  (queueSign *QueueSign)  delOneRuleOneWeight( redisConn redis.Conn,id int){
	key := queueSign.getRedisKeyWeight()
	res,_ := myredis.Send(redisConn,"ZREM",redis.Args{}.Add(key).Add(id)...)
	//res,_ := myredis.RedisDo("ZREM",redis.Args{}.Add(key).Add(id)...)
	mylog.Debug("delOneWeight : ",res)
	queueSign.Log.Info("delOneWeight ",id)
}
//删除一条规则的一个组的详细信息
func  (queueSign *QueueSign) delOneRuleOneGroupElement(redisConn redis.Conn, id int){
	key := queueSign.getRedisKeyGroupElement(id)
	res,_ := myredis.Send(redisConn,"del",key)
	//res,_ := myredis.RedisDo("del",key)
	mylog.Debug("delOneGroupElement : ",res)
	queueSign.Log.Info("delOneGroupElement ",id)
}
////删除一个玩家的报名
//func (queueSign *QueueSign) delOneByPlayerId( playerId int){
//	id := queueSign.getGroupIdByPlayerId(playerId)
//	if id <=0 {
//		mylog.Error("getGroupIdByPlayerId is empty !!!")
//		return
//	}
//	queueSign.delOneRuleOneGroup(id,1)
//	//这里还要更新当前组所有用户的，状态信息
//}

//====================================================
//删除一个组
func (queueSign *QueueSign) delOneRuleOneGroup( redisConn redis.Conn,id int,isDelPlayerStatus int)error{
	rootAndSingToLogInfoMsg(queueSign,"action : delOneRuleOneGroup id:" , id)

	group := queueSign.getGroupElementById(id)
	mylog.Debug(group)
	//这里是偷懒了，判断是否为空，按说应该返回2个参数，但这个方法调用 地方多，先这样
	if group.Id == 0 || group.Person == 0 || len(group.Players) == 0{
		msg := make(map[int]string)
		msg[0] = strconv.Itoa(id)
		nowerr := myerr.NewErrorCodeReplace(750,msg)

		mylog.Error(750,nowerr.Error())
		queueSign.Log.Error(750,nowerr.Error())

		return nowerr
	}

	queueSign.delOneRuleOneGroupElement(redisConn,id)
	queueSign.delOneRuleOneWeight(redisConn,id)
	queueSign.delOneRuleOnePersonIndexById(redisConn,group.Person,id)
	queueSign.delOneRuleOneGroupPlayers(redisConn,id)
	queueSign.delOneRuleOneGroupSignTimeout(redisConn,id)

	if isDelPlayerStatus == 1{
		for _,v := range group.Players{
			queueSign.Log.Info("playerStatus delOneById : ",v.Id)
			//playerStatus.upInfo(v)
			playerStatus.delOneById(redisConn,v.Id)
		}
	}

	rootAndSingToLogInfoMsg(queueSign,"delOneRuleOneGroup finish, id:" , id)

	return nil
}
////删除一个组的，所有索引信息,反向看：除了小组基础信息外，其余均删除
//func  (queueSign *QueueSign)  delOneGroupIndex(groupId int){
//	zlib.MyPrint(" delOneGroupIndex : ",queueSign.getRedisCatePrefixKey())
//	group := queueSign.getGroupElementById(groupId)
//
//	queueSign.delOneRuleOneWeight(groupId)
//	queueSign.delOneRuleOnePersonIndexById(group.Person,groupId)
//	queueSign.delOneRuleOneGroupPlayers(groupId)
//	queueSign.delOneRuleOneGroupSignTimeout(groupId)
//}
//添加一个组的:<权限+人数>,<人数+权重>的索引
//-匹配时，会把小于的索引先删掉，避免重复，最终结果，可能有些小组还要再PUSH BACK 回来
func  (queueSign *QueueSign)  addOneGroupIndex(redisConn redis.Conn, groupId int,personNum int,weight float32){
	//zlib.MyPrint(" addOneGroupIndex : ",queueSign.getRedisCatePrefixKey())
	//group := queueSign.getGroupElementById(groupId)

	groupIndexKey := queueSign.getRedisKeyWeight( )
	res,err := myredis.RedisDo("zadd",redis.Args{}.Add(groupIndexKey).Add(weight).Add(groupId)...)
	mylog.Debug("add GroupWeightIndex rs : ",res,err)

	PersonIndexKey := queueSign.getRedisKeyPersonIndex(  personNum)
	res,err = myredis.RedisDo("zadd",redis.Args{}.Add(PersonIndexKey).Add(weight).Add(groupId)...)
	mylog.Debug("add GroupPersonIndex ( ",personNum," ) rs : ",res,err)
}

//func  (queueSign *QueueSign)  addOneGroupIndex(groupId int){
//	zlib.MyPrint(" addOneGroupIndex : ",queueSign.getRedisCatePrefixKey())
//	group := queueSign.getGroupElementById(groupId)
//
//	groupIndexKey := queueSign.getRedisKeyWeight( )
//	res,err := myredis.RedisDo("zadd",redis.Args{}.Add(groupIndexKey).Add(group.Weight).Add(group.Id)...)
//	mylog.Debug("add GroupWeightIndex rs : ",res,err)
//
//	PersonIndexKey := queueSign.getRedisKeyPersonIndex(  group.Person)
//	res,err = myredis.RedisDo("zadd",redis.Args{}.Add(PersonIndexKey).Add(group.Weight).Add(group.Id)...)
//	mylog.Debug("add GroupPersonIndex ( ",group.Person," ) rs : ",res,err)
//}

//删除一个组的:<权限+人数>,<人数+权重>的索引
//-匹配的时候，前期计算均是基于索引，在getList时，要把这个索引先删了，等到计算出最终结果，再决定是否全删
//-如果有些索引要放回去，直接恢复这两个索引即可
func (queueSign *QueueSign) delOneRuleOneGroupIndex(redisConn redis.Conn, groupId int,personNum int){
	mylog.Info("action : delOneRuleOneGroupIndex id:" , groupId)
	//group := queueSign.getGroupElementById(id)
	//mylog.Debug(group)

	queueSign.delOneRuleOneWeight(redisConn,groupId)
	queueSign.delOneRuleOnePersonIndexById(redisConn,personNum,groupId)
}

func  (queueSign *QueueSign) CheckTimeout( ){
	rootAndSingToLogInfoMsg(queueSign," one rule CheckSignTimeout , ruleId : ",queueSign.Rule.Id)
	//mylog.Info(" one rule CheckSignTimeout , ruleId : ",queueSign.Rule.Id)
	//queueSign.Log.Info(" one rule CheckSignTimeout , ruleId : ",queueSign.Rule.Id)

	push := queueSign.Gamematch.getContainerPushByRuleId(queueSign.Rule.Id)
	keys := queueSign.getRedisKeyGroupSignTimeout()
	//inc := 1
	//for {
	//	mylog.Info("loop inc : ",inc )
	//	queueSign.Log.Info("loop inc : ",inc)
	//	if inc >= 2147483647{
	//		inc = 0
	//	}
	//	inc++

		now := zlib.GetNowTimeSecondToInt()
		res,err := redis.IntMap(myredis.RedisDo("ZREVRANGEBYSCORE",redis.Args{}.Add(keys).Add(now).Add("-inf").Add("WITHSCORES")...))
		rootAndSingToLogInfoMsg(queueSign,"sign timeout group element total : ",len(res))
		//mylog.Info("sign timeout group element total : ",len(res))
		//queueSign.Log.Info("sign timeout group element total : ",len(res))
		if err != nil{
			mylog.Error("redis ZREVRANGEBYSCORE err :",err.Error())
			queueSign.Log.Error("redis ZREVRANGEBYSCORE err :",err.Error())
			return
		}
		//res , _:= redis.IntMap(doRes,err)
		//mylog.Info("sign timeout group element total : ",len(res))
		if len(res) == 0{
			mylog.Notice(" empty , no need process")
			queueSign.Log.Notice(" empty , no need process")
			//myGosched("sign CheckTimeout")
			//mySleepSecond(1, " sign CheckTimeout ")
			return
		}
		redisConn := myredis.GetNewConnFromPool()
		defer redisConn.Close()
		for groupIdStr,_ := range res{
			myredis.Multi(redisConn)

			groupId := zlib.Atoi(groupIdStr)
			group := queueSign.getGroupElementById(groupId)
			payload := GroupStructToStr (group)
			payload = strings.Replace(payload,separation,PayloadSeparation,-1)
			queueSign.Log.Info("addOnePush",groupId,PushCategorySignTimeout,payload)
			push.addOnePush(redisConn,groupId,PushCategorySignTimeout,payload)
			queueSign.Log.Info("delOneRuleOneGroup")
			queueSign.delOneRuleOneGroup(redisConn,groupId,1)
			//zlib.ExitPrint(-199)

			myredis.Exec(redisConn)
		}
		rootAndSingToLogInfoMsg(queueSign,"queueSign checkTimeOut finish in oneTime")
		//myGosched("sign CheckTimeout")
		//mySleepSecond(1," sign CheckTimeout ")
	//}
}